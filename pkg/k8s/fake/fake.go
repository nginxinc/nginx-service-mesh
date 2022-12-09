// Package fake provides a fake k8s client implementation.
package fake

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	rt "runtime"

	accessv1alpha2 "github.com/servicemeshinterface/smi-controller-sdk/apis/access/v1alpha2"
	specsv1alpha3 "github.com/servicemeshinterface/smi-controller-sdk/apis/specs/v1alpha3"
	splitv1alpha3 "github.com/servicemeshinterface/smi-controller-sdk/apis/split/v1alpha3"
	"helm.sh/helm/v3/pkg/action"
	apiextension "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	fakeApiExt "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	fakeDynamic "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
	aggregator "k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset"
	fakeAggregator "k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset/fake"
	metrics "k8s.io/metrics/pkg/client/clientset/versioned"
	fakeMetrics "k8s.io/metrics/pkg/client/clientset/versioned/fake"
	"sigs.k8s.io/yaml"

	nsmspecsv1alpha1 "github.com/nginxinc/nginx-service-mesh/pkg/apis/specs/v1alpha1"
	nsmspecsv1alpha2 "github.com/nginxinc/nginx-service-mesh/pkg/apis/specs/v1alpha2"
	"github.com/nginxinc/nginx-service-mesh/pkg/k8s"
)

// Client implements the k8s.Client interface.
type Client struct {
	metricsClientset         metrics.Interface
	HelmInvoker              k8s.HelmWrapper
	clientset                kubernetes.Interface
	apiExtensionClientset    apiextension.Interface
	apiRegistrationClientset aggregator.Interface
	dynamicClientset         dynamic.Interface
	config                   *rest.Config
	namespace                string
	skipRelease              bool
}

// NewFakeK8s creates a fake k8s client for unit tests.
func NewFakeK8s(namespace string, skipRelease bool, objects ...runtime.Object) *Client {
	scheme := runtime.NewScheme()
	_ = splitv1alpha3.AddToScheme(scheme)
	_ = accessv1alpha2.AddToScheme(scheme)
	_ = specsv1alpha3.AddToScheme(scheme)
	_ = nsmspecsv1alpha1.AddToScheme(scheme)
	_ = nsmspecsv1alpha2.AddToScheme(scheme)

	fakeClientSet := fake.NewSimpleClientset(objects...)

	return &Client{
		config:                   &rest.Config{},
		namespace:                namespace,
		clientset:                fakeClientSet,
		apiExtensionClientset:    fakeApiExt.NewSimpleClientset(),
		apiRegistrationClientset: fakeAggregator.NewSimpleClientset(),
		metricsClientset:         fakeMetrics.NewSimpleClientset(),
		dynamicClientset:         fakeDynamic.NewSimpleDynamicClient(scheme),
		skipRelease:              skipRelease,
		HelmInvoker:              NewFakeHelmInvoker(namespace, skipRelease, fakeClientSet),
	}
}

// Config return a k8s rest.Config object.
func (k *Client) Config() *rest.Config {
	return k.config
}

// Namespace return the namespace we are using for k8s.
func (k *Client) Namespace() string {
	return k.namespace
}

// ClientSet return the clientset interface.
func (k *Client) ClientSet() kubernetes.Interface {
	return k.clientset
}

// APIExtensionClientSet return the clients interface.
func (k *Client) APIExtensionClientSet() apiextension.Interface {
	return k.apiExtensionClientset
}

// APIRegistrationClientSet return the APIRegistrationClientSet interface.
func (k *Client) APIRegistrationClientSet() aggregator.Interface {
	return k.apiRegistrationClientset
}

// MetricsClientSet return the MetricsClientset interface.
func (k *Client) MetricsClientSet() metrics.Interface {
	return k.metricsClientset
}

// DynamicClientSet return the DynamicClientset interface.
func (k *Client) DynamicClientSet() dynamic.Interface {
	return k.dynamicClientset
}

// MeshExists returns if the mesh exists.
func (k *Client) MeshExists() (bool, error) {
	return false, nil
}

type fakeCachedDiscoveryClient struct {
	discovery.DiscoveryInterface
}

func (d *fakeCachedDiscoveryClient) Fresh() bool {
	return true
}

func (d *fakeCachedDiscoveryClient) Invalidate() {}

func (d *fakeCachedDiscoveryClient) ServerGroupsAndResources() ([]*metav1.APIGroup, []*metav1.APIResourceList, error) {
	return []*metav1.APIGroup{}, []*metav1.APIResourceList{}, nil
}

func (d *fakeCachedDiscoveryClient) ServerVersion() (*version.Info, error) {
	return &version.Info{GitVersion: "1.20", Major: "", Minor: ""}, nil
}

// HelmAction returns a helm action with a fake release.
func (k *Client) HelmAction(string) (*action.Configuration, error) {
	return k.HelmInvoker.HelmAction("nginx-mesh")
}

var errGettingCaller = errors.New("could not get function caller")

func testValues() (map[string]interface{}, error) {
	_, b, _, ok := rt.Caller(0)
	if !ok {
		return nil, errGettingCaller
	}
	valuesYaml, err := os.ReadFile(filepath.Dir(b) + "/../../../helm-chart/values.yaml")
	if err != nil {
		return nil, err
	}

	jsonBytes, err := yaml.YAMLToJSON(valuesYaml)
	if err != nil {
		return nil, fmt.Errorf("error converting values to json: %w", err)
	}

	var valuesMap map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &valuesMap); err != nil {
		return nil, fmt.Errorf("error unmarshaling values: %w", err)
	}

	return valuesMap, nil
}
