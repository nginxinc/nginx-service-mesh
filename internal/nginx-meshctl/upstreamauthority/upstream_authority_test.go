package upstreamauthority

import (
	"os"

	"github.com/nginxinc/nginx-service-mesh/pkg/helm"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func createFile(filename, contents string) (string, error) {
	file, err := os.CreateTemp("", filename)
	if err != nil {
		return "", err
	}
	_, err = file.WriteString(contents)

	return file.Name(), err
}

var awsPCAUA = `
apiVersion: v1
upstreamAuthority: aws_pca
config:
    region: "test_region"
    certificate_authority_arn: "arn:aws:acm-pca::123456789012:certificate-authority/test"
    aws_access_key_id: "test_access"
    aws_secret_access_key: "test_secret"
`

//nolint:gosec // Test data
var awsSecretUA = `
apiVersion: v1
upstreamAuthority: awssecret
config:
    region: "test_region"
    cert_file_arn: "arn:aws:acm-pca::123456789012:certificate-authority/test"
    key_file_arn: "arn:aws:acm-pca::123456789012:certificate-authority/testkey"
    aws_access_key_id: "test_access"
    aws_secret_access_key: "test_secret"
`

var diskUAConfigTemplate = `
apiVersion: v1
upstreamAuthority: "disk"
config:
    {{- if .CertFilePath}}
    cert_file_path: {{.CertFilePath}}{{end}}
    {{- if .KeyFilePath}}
    key_file_path: {{.KeyFilePath}}{{end}}
    {{- if .BundleFilePath}}
    bundle_file_path: {{.BundleFilePath}}{{end}}
`

var vaultUAConfigTemplate = `
apiVersion: v1
upstreamAuthority: vault
config:
  vault_addr: "https://vault:8200"
  namespace: "default"
  {{- if .CACertPath}}
  ca_cert_path: "{{.CACertPath}}"{{end}}
  cert_auth:
        {{- if .CertAuth.ClientCertPath}}
        client_cert_path: "{{.CertAuth.ClientCertPath}}"{{end}}
        {{- if .CertAuth.ClientKeyPath}}
        client_key_path: "{{.CertAuth.ClientKeyPath}}"{{end}}
`

var certManagerConfigTemplate = `
apiVersion: v1
upstreamAuthority: cert-manager
config:
  namespace: "default"
  issuer_name: "spire-ca"
  issuer_kind: "Issuer"
  issuer_group: "cert-manager.io"
  kube_config_file: {{.}}
`

var invalidUA = `
apiVersion: v1
upstreamAuthority: invalid
config:
    region: "test"
`

var invalidVersion = `
apiVersion: v2
upstreamAuthority: aws_pca
config:
    region: "test_region"
    certificate_authority_arn: "arn:aws:acm-pca::123456789012:certificate-authority/test"
    aws_access_key_id: "test_access"
    aws_secret_access_key: "test_secret"
`

type testDiskConfig struct {
	CertFile   bool
	KeyFile    bool
	BundleFile bool
}

func createDiskUA(config testDiskConfig) *diskConfig {
	var diskUAConfig diskConfig
	if config.CertFile {
		diskUAConfig.CertFilePath = createCertFile("cert_path")
	}
	if config.KeyFile {
		diskUAConfig.KeyFilePath = createCertFile("key_path")
	}
	if config.BundleFile {
		diskUAConfig.BundleFilePath = createCertFile("key_path")
	}

	return &diskUAConfig
}

func deleteDiskConfigFiles(uaFile string, config *diskConfig) error {
	if uaFile != "" {
		err := os.Remove(uaFile)
		if err != nil {
			return err
		}
	}
	if config.CertFilePath != "" {
		err := os.Remove(config.CertFilePath)
		if err != nil {
			return err
		}
	}
	if config.KeyFilePath != "" {
		err := os.Remove(config.KeyFilePath)
		if err != nil {
			return err
		}
	}
	if config.BundleFilePath != "" {
		err := os.Remove(config.BundleFilePath)
		if err != nil {
			return err
		}
	}

	return nil
}

func createVaultUA() *vaultConfig {
	return &vaultConfig{
		VaultAddr:  "vault:8080",
		Namespace:  "default",
		CACertPath: createCertFile("ca_cert_path"),
	}
}

func createVaultCertAuthUA() *vaultConfig {
	vaultUAConfig := createVaultUA()
	vaultUAConfig.CertAuth = &certAuthConfig{
		ClientCertPath: createCertFile("cert_path"),
		ClientKeyPath:  createCertFile("key_path"),
	}

	return vaultUAConfig
}

func deleteVaultConfigFiles(uaFile string, config *vaultConfig) error {
	if uaFile != "" {
		err := os.Remove(uaFile)
		if err != nil {
			return err
		}
	}
	if config.CACertPath != "" {
		err := os.Remove(config.CACertPath)
		if err != nil {
			return err
		}
	}
	if config.CertAuth.ClientCertPath != "" {
		err := os.Remove(config.CertAuth.ClientCertPath)
		if err != nil {
			return err
		}
	}
	if config.CertAuth.ClientKeyPath != "" {
		err := os.Remove(config.CertAuth.ClientKeyPath)
		if err != nil {
			return err
		}
	}

	return nil
}

var _ = Describe("Upstream Authority", func() {
	Context("Gets UA values", func() {
		It("can extract disk UA", func() {
			diskUA := createDiskUA(testDiskConfig{CertFile: true, KeyFile: true})
			uaFile := writeUAFile(diskUAConfigTemplate, diskUA)
			defer func() { Expect(deleteDiskConfigFiles(uaFile, diskUA)).To(Succeed()) }()
			vals, err := GetUpstreamAuthorityValues(uaFile)
			Expect(err).ToNot(HaveOccurred())
			expVals := helm.UpstreamAuthority{
				Disk: &helm.Disk{
					Cert: "-----BEGIN Test-----\n-----END Test-----\n",
					Key:  "-----BEGIN Test-----\n-----END Test-----\n",
				},
			}
			Expect(*vals).To(Equal(expVals))
		})
		It("can extract aws_pca UA", func() {
			ua, err := createFile("aws_pca", awsPCAUA)
			defer func() { Expect(os.Remove(ua)).To(Succeed()) }()
			Expect(err).ToNot(HaveOccurred())
			vals, err := GetUpstreamAuthorityValues(ua)
			Expect(err).ToNot(HaveOccurred())
			expVals := helm.UpstreamAuthority{
				AWSPCA: &helm.AWSPCA{
					Region:                  "test_region",
					CertificateAuthorityArn: "arn:aws:acm-pca::123456789012:certificate-authority/test",
					AWSAccessKeyID:          "test_access",
					AWSSecretAccessKey:      "test_secret",
				},
			}
			Expect(*vals).To(Equal(expVals))
		})
		It("can extract awssecret UA", func() {
			ua, err := createFile("awssecret", awsSecretUA)
			defer func() { Expect(os.Remove(ua)).To(Succeed()) }()
			Expect(err).ToNot(HaveOccurred())
			vals, err := GetUpstreamAuthorityValues(ua)
			Expect(err).ToNot(HaveOccurred())
			expVals := helm.UpstreamAuthority{
				AWSSecret: &helm.AWSSecret{
					Region:             "test_region",
					CertFileArn:        "arn:aws:acm-pca::123456789012:certificate-authority/test",
					KeyFileArn:         "arn:aws:acm-pca::123456789012:certificate-authority/testkey",
					AWSAccessKeyID:     "test_access",
					AWSSecretAccessKey: "test_secret",
				},
			}
			Expect(*vals).To(Equal(expVals))
		})
		It("can extract vault UA", func() {
			ua := createVaultCertAuthUA()
			uaFile := writeUAFile(vaultUAConfigTemplate, ua)
			defer func() { Expect(deleteVaultConfigFiles(uaFile, ua)).To(Succeed()) }()
			vals, err := GetUpstreamAuthorityValues(uaFile)
			Expect(err).ToNot(HaveOccurred())
			expVals := helm.UpstreamAuthority{
				Vault: &helm.Vault{
					VaultAddr:          "https://vault:8200",
					Namespace:          "default",
					CACert:             "-----BEGIN Test-----\n-----END Test-----\n",
					PKIMountPoint:      "",
					InsecureSkipVerify: false,
					CertAuth: &helm.CertAuth{
						ClientCert:         "-----BEGIN Test-----\n-----END Test-----\n",
						ClientKey:          "-----BEGIN Test-----\n-----END Test-----\n",
						CertAuthMountPoint: "",
						CertAuthRoleName:   "",
					},
				},
			}
			Expect(*vals).To(Equal(expVals))
		})
		It("can extract cert-manager UA", func() {
			kubeConfig, err := createFile("kubeconfig", "kube-config-contents")
			Expect(err).ToNot(HaveOccurred())

			uaFile := writeUAFile(certManagerConfigTemplate, kubeConfig)
			defer func() {
				Expect(os.Remove(uaFile)).To(Succeed())
				Expect(os.Remove(kubeConfig)).To(Succeed())
			}()
			vals, err := GetUpstreamAuthorityValues(uaFile)
			Expect(err).ToNot(HaveOccurred())
			expVals := helm.UpstreamAuthority{
				CertManager: &helm.CertManager{
					Namespace:   "default",
					IssuerName:  "spire-ca",
					IssuerKind:  "Issuer",
					IssuerGroup: "cert-manager.io",
					KubeConfig:  "kube-config-contents",
				},
			}
			Expect(*vals).To(Equal(expVals))
		})
		It("extracts no other UA", func() {
			ua, err := createFile("invalid", invalidUA)
			defer func() { Expect(os.Remove(ua)).To(Succeed()) }()
			Expect(err).ToNot(HaveOccurred())
			_, err = GetUpstreamAuthorityValues(ua)
			Expect(err).To(HaveOccurred())
		})
	})
	It("rejects invalid version", func() {
		ua, err := createFile("invalid", invalidVersion)
		defer func() { Expect(os.Remove(ua)).To(Succeed()) }()
		Expect(err).ToNot(HaveOccurred())
		_, err = GetUpstreamAuthorityValues(ua)
		Expect(err).To(HaveOccurred())
	})
})
