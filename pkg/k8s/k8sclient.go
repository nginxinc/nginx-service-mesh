// Package k8s contains a client for accessing the kubernetes API.
package k8s

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/user"
	"path"
	"strings"

	"helm.sh/helm/v3/pkg/action"
	apiextension "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	aggregator "k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset"
	metrics "k8s.io/metrics/pkg/client/clientset/versioned"
	utilsPath "k8s.io/utils/path"
	k8sClient "sigs.k8s.io/controller-runtime/pkg/client"

	meshErrors "github.com/nginxinc/nginx-service-mesh/pkg/errors"
)

//go:generate counterfeiter -o fake/fake_k8s_client.gen.go ./. Client

// Client is a k8s client wrapper for interacting with mesh objects.
type Client interface {
	Config() *rest.Config
	Namespace() string
	Client() k8sClient.Client
	ClientSet() kubernetes.Interface
	APIExtensionClientSet() apiextension.Interface
	APIRegistrationClientSet() aggregator.Interface
	MetricsClientSet() metrics.Interface
	DynamicClientSet() dynamic.Interface
	MeshExists() (bool, error)
	HelmAction(namespace string) (*action.Configuration, error)
}

// ClientImpl contains the configuration and clients for a Kubernetes cluster.
type ClientImpl struct {
	client                   k8sClient.Client
	apiRegistrationClientset aggregator.Interface
	deferredClientConfig     clientcmd.ClientConfig
	metricsClientset         metrics.Interface
	Clientset                kubernetes.Interface
	HelmInvoker              HelmWrapper
	APIExtensionClientset    apiextension.Interface
	dynamicClientset         dynamic.Interface
	config                   *rest.Config
	namespace                string
}

// GetKubeConfig returns the kubeconfig.
func GetKubeConfig() string {
	var homeDir string

	usr, err := user.Current()
	if err != nil {
		homeDir = "/root"
	} else {
		homeDir = usr.HomeDir
	}
	kubeCfg := path.Join(homeDir, ".kube/config")
	if envCfg := os.Getenv("KUBECONFIG"); envCfg != "" {
		kubeCfg = envCfg
	}

	return kubeCfg
}

// NewK8SClient creates a mesh K8s client from a kubeconfig and control plane namespace.
func NewK8SClient(kubeconfig, namespace string) (*ClientImpl, error) {
	var err error
	var exists bool

	k8s := &ClientImpl{}

	configRules := clientcmd.NewDefaultClientConfigLoadingRules()
	exists, _ = utilsPath.Exists(utilsPath.CheckFollowSymlink, kubeconfig)
	if kubeconfig != "" && exists {
		configRules.ExplicitPath = kubeconfig
	}
	overrides := &clientcmd.ConfigOverrides{}
	if namespace != "" {
		overrides.Context.Namespace = namespace
	}
	k8s.deferredClientConfig = clientcmd.NewNonInteractiveDeferredLoadingClientConfig(configRules, overrides)
	k8s.config, err = k8s.deferredClientConfig.ClientConfig()
	if err != nil {
		return k8s, err
	}
	k8s.HelmInvoker = NewHelmInvoker(kubeconfig, namespace, k8s.deferredClientConfig)

	k8s.namespace, _, err = k8s.deferredClientConfig.Namespace()
	if err != nil {
		return k8s, err
	}

	k8s.Clientset, err = kubernetes.NewForConfig(k8s.config)
	if err != nil {
		return k8s, err
	}

	k8s.APIExtensionClientset, err = apiextension.NewForConfig(k8s.config)
	if err != nil {
		return k8s, err
	}

	k8s.apiRegistrationClientset, err = aggregator.NewForConfig(k8s.config)
	if err != nil {
		return k8s, err
	}

	k8s.dynamicClientset, err = dynamic.NewForConfig(k8s.config)
	if err != nil {
		return k8s, err
	}

	k8s.metricsClientset, err = metrics.NewForConfig(k8s.config)
	if err != nil {
		return k8s, err
	}

	k8s.client, err = k8sClient.New(k8s.config, k8sClient.Options{})
	if err != nil {
		return k8s, err
	}

	return k8s, nil
}

// Config returns a k8s rest.Config object.
func (k *ClientImpl) Config() *rest.Config {
	return k.config
}

// Namespace returns the control plane namespace.
func (k *ClientImpl) Namespace() string {
	return k.namespace
}

// Client returns the controller-runtime client.
func (k *ClientImpl) Client() k8sClient.Client {
	return k.client
}

// ClientSet returns the ClientSet interface.
func (k *ClientImpl) ClientSet() kubernetes.Interface {
	return k.Clientset
}

// APIExtensionClientSet returns the APIExtensionClientSet interface.
func (k *ClientImpl) APIExtensionClientSet() apiextension.Interface {
	return k.APIExtensionClientset
}

// APIRegistrationClientSet returns the APIRegistrationClientSet interface.
func (k *ClientImpl) APIRegistrationClientSet() aggregator.Interface {
	return k.apiRegistrationClientset
}

// MetricsClientSet returns the MetricsClientset interface.
func (k *ClientImpl) MetricsClientSet() metrics.Interface {
	return k.metricsClientset
}

// DynamicClientSet returns the DynamicClientset interface.
func (k *ClientImpl) DynamicClientSet() dynamic.Interface {
	return k.dynamicClientset
}

var meshCRDs = map[string]struct{}{
	"spiffeids.spiffeid.spiffe.io":        {},
	"trafficsplits.split.smi-spec.io":     {},
	"traffictargets.access.smi-spec.io":   {},
	"httproutegroups.specs.smi-spec.io":   {},
	"tcproutes.specs.smi-spec.io":         {},
	"ratelimits.specs.smi.nginx.com":      {},
	"circuitbreakers.specs.smi.nginx.com": {},
	"meshconfigclasses.nsm.nginx.com":     {},
	"meshconfigs.nsm.nginx.com":           {},
	"retrytimeoutconfig.nsm.nginx.com":    {},
}

var errCRDAlreadyExists = errors.New("CRD already exists")

// MeshExists returns if the mesh exists or not.
func (k *ClientImpl) MeshExists() (bool, error) {
	// check if a helm release exists under the name nginx-service-mesh in any namespace
	actionConfig, err := k.HelmAction("")
	if err != nil {
		return false, fmt.Errorf("initializing helm action config failed: %w", err)
	}

	lister := action.NewList(actionConfig)
	lister.AllNamespaces = true
	lister.All = true
	releases, err := lister.Run()
	if err != nil {
		return false, fmt.Errorf("failed to list currently installed releases: %w", err)
	}

	for _, release := range releases {
		if release.Chart != nil {
			if strings.Contains(release.Chart.Name(), "nginx-service-mesh") {
				return true, meshErrors.AlreadyExistsError{
					Msg: fmt.Sprintf("found existing release '%s' of NGINX Service Mesh in namespace '%s' with status '%s'",
						release.Name, release.Namespace, release.Info.Status),
				}
			}
		}
	}

	// check if mesh CRDs exist
	crds, err := k.APIExtensionClientSet().ApiextensionsV1().CustomResourceDefinitions().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return false, fmt.Errorf("unable to verify that NGINX Service Mesh is running; error checking CRDs: %w", err)
	}

	var foundCRDs []string
	labelKey := "app.kubernetes.io/part-of"
	labelVal := "nginx-service-mesh"

	for _, crd := range crds.Items {
		if _, ok := meshCRDs[crd.Name]; ok {
			if val, ok := crd.Labels[labelKey]; ok && val == labelVal {
				foundCRDs = append(foundCRDs, crd.Name)

				continue
			}
			// CRD is deployed, but not a part of our mesh
			return false, fmt.Errorf("NGINX Service Mesh not detected, but '%s' %w. Please remove "+
				"before deploying NGINX Service Mesh", crd.Name, errCRDAlreadyExists)
		}
	}
	if len(foundCRDs) > 0 {
		return true, meshErrors.AlreadyExistsError{
			Msg: fmt.Sprintf("NGINX Service Mesh CRDs detected: '%s'. Delete before deploying the mesh", strings.Join(foundCRDs, ", ")),
		}
	}

	return false, nil
}

// HelmAction builds an action configuration for helm requests.
func (k *ClientImpl) HelmAction(namespace string) (*action.Configuration, error) {
	return k.HelmInvoker.HelmAction(namespace)
}
