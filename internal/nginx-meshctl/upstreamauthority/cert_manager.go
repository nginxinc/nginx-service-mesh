package upstreamauthority

import (
	"fmt"
	"os"

	"github.com/nginxinc/nginx-service-mesh/pkg/helm"
)

type certManagerConfig struct {
	IssuerKind     *string `json:"issuer_kind"`
	IssuerGroup    *string `json:"issuer_group"`
	KubeConfigFile *string `json:"kube_config_file"`
	Namespace      string  `json:"namespace"`
	IssuerName     string  `json:"issuer_name"`
}

func (cm certManagerConfig) configure() (*helm.UpstreamAuthority, error) {
	var kubeConfig string
	if cm.KubeConfigFile != nil {
		f, err := os.ReadFile(*cm.KubeConfigFile)
		if err != nil {
			return nil, fmt.Errorf("error reading kube_config_file: %w", err)
		}

		kubeConfig = string(f)
	}

	upstreamAuthority := &helm.UpstreamAuthority{
		CertManager: &helm.CertManager{
			Namespace:   cm.Namespace,
			IssuerName:  cm.IssuerName,
			IssuerKind:  dereferenceString(cm.IssuerKind),
			IssuerGroup: dereferenceString(cm.IssuerGroup),
			KubeConfig:  kubeConfig,
		},
	}

	return upstreamAuthority, nil
}
