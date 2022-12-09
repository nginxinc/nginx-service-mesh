package upstreamauthority

import "github.com/nginxinc/nginx-service-mesh/pkg/helm"

type diskConfig struct {
	CertFilePath   string `json:"cert_file_path"`
	KeyFilePath    string `json:"key_file_path"`
	BundleFilePath string `json:"bundle_file_path"`
}

func (dc diskConfig) configure() (*helm.UpstreamAuthority, error) {
	upstreamAuthority := &helm.UpstreamAuthority{
		Disk: &helm.Disk{},
	}
	cert, key, bundle, err := readCertFiles(dc.CertFilePath, dc.KeyFilePath, dc.BundleFilePath)
	if err != nil {
		return nil, err
	}

	upstreamAuthority.Disk = &helm.Disk{
		Cert:   cert,
		Key:    key,
		Bundle: bundle,
	}

	return upstreamAuthority, nil
}
