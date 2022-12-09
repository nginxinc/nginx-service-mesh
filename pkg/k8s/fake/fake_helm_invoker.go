package fake

import (
	"errors"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	helmKube "helm.sh/helm/v3/pkg/kube/fake"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/storage"
	"helm.sh/helm/v3/pkg/storage/driver"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"

	"github.com/nginxinc/nginx-service-mesh/pkg/k8s"
)

type fakeHelmInvoker struct {
	clientset   kubernetes.Interface
	namespace   string
	skipRelease bool
}

// NewFakeHelmInvoker returns a fake helm invoker for use in testing.
func NewFakeHelmInvoker(namespace string, skipRelease bool, clientset kubernetes.Interface) k8s.HelmWrapper {
	return &fakeHelmInvoker{
		namespace:   namespace,
		skipRelease: skipRelease,
		clientset:   clientset,
	}
}

// HelmAction returns a mock of HelmAction for use in testing.
func (hi *fakeHelmInvoker) HelmAction(string) (*action.Configuration, error) {
	cfg := &action.Configuration{
		KubeClient:       &helmKube.FailingKubeClient{},
		Releases:         storage.Init(driver.NewSecrets(hi.clientset.CoreV1().Secrets(hi.namespace))),
		RESTClientGetter: genericclioptions.NewTestConfigFlags().WithDiscoveryClient(&fakeCachedDiscoveryClient{}),
		Log:              func(_ string, _ ...interface{}) {},
	}

	values, err := testValues()
	if err != nil {
		return nil, err
	}

	if !hi.skipRelease {
		rel := &release.Release{
			Name:      "nginx-service-mesh",
			Namespace: "nginx-mesh",
			Info: &release.Info{
				Status: release.StatusDeployed,
			},
			Config: values,
			Chart: &chart.Chart{
				Metadata: &chart.Metadata{
					Name: "nginx-service-mesh-0.5.0",
				},
			},
		}
		if err := cfg.Releases.Create(rel); err != nil && !errors.Is(err, driver.ErrReleaseExists) {
			return nil, err
		}
	}

	return cfg, nil
}
