package k8s

import (
	"fmt"

	"helm.sh/helm/v3/pkg/action"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/tools/clientcmd"
)

// HelmWrapper wraps the HelmAction function as an interface.
type HelmWrapper interface {
	HelmAction(namespace string) (*action.Configuration, error)
}

// HelmInvoker is a struct that satisfies HelmWrapper.
type HelmInvoker struct {
	deferredClientConfig clientcmd.ClientConfig
	namespace            string
	kubeconfig           string
}

// NewHelmInvoker returns a HelmInvoker satisfying the HelmWrapper Interface.
func NewHelmInvoker(kubeconfig, namespace string, deferredClientConfig clientcmd.ClientConfig) HelmWrapper {
	return &HelmInvoker{
		namespace:            namespace,
		kubeconfig:           kubeconfig,
		deferredClientConfig: deferredClientConfig,
	}
}

// HelmAction builds an action configuration for helm requests.
func (hi *HelmInvoker) HelmAction(namespace string) (*action.Configuration, error) {
	configFlags := genericclioptions.NewConfigFlags(true)
	configFlags.Namespace = &namespace
	configFlags.KubeConfig = &hi.kubeconfig

	cfg, err := hi.deferredClientConfig.RawConfig()
	if err != nil {
		return nil, fmt.Errorf("error getting k8s raw config: %w", err)
	}
	configFlags.Context = &cfg.CurrentContext

	actionConfig := new(action.Configuration)
	if initErr := actionConfig.Init(configFlags, namespace, "secret", func(_ string, _ ...interface{}) {}); initErr != nil {
		return nil, fmt.Errorf("error initializing helm action: %w", err)
	}

	return actionConfig, nil
}
