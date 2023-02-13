// Package helm contains the helm values and functions used for deploying the mesh.
package helm

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"strings"

	"gopkg.in/yaml.v2"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"

	chart "github.com/nginxinc/nginx-service-mesh/helm-chart"
	"github.com/nginxinc/nginx-service-mesh/pkg/k8s"
)

// Values is the top level representation of the Helm values.yaml.
type Values struct {
	Tracing              *Tracing   `yaml:"tracing" json:"tracing"`
	Telemetry            *Telemetry `yaml:"telemetry" json:"telemetry"`
	MTLS                 MTLS       `yaml:"mtls" json:"mtls"`
	ClientMaxBodySize    string     `yaml:"clientMaxBodySize" json:"clientMaxBodySize"`
	PrometheusAddress    string     `yaml:"prometheusAddress" json:"prometheusAddress"`
	Environment          string     `yaml:"environment" json:"environment"`
	AccessControlMode    string     `yaml:"accessControlMode" json:"accessControlMode"`
	NGINXErrorLogLevel   string     `yaml:"nginxErrorLogLevel" json:"nginxErrorLogLevel"`
	NGINXLBMethod        string     `yaml:"nginxLBMethod" json:"nginxLBMethod"`
	NGINXLogFormat       string     `yaml:"nginxLogFormat" json:"nginxLogFormat"`
	Registry             Registry   `yaml:"registry" json:"registry"`
	EnabledNamespaces    []string   `yaml:"enabledNamespaces" json:"enabledNamespaces"`
	EnableUDP            bool       `yaml:"enableUDP" json:"enableUDP"`
	DisableAutoInjection bool       `yaml:"disableAutoInjection" json:"disableAutoInjection"`
}

// Registry is the registry struct within Values.
type Registry struct {
	Server              string `yaml:"server" json:"server"`
	ImageTag            string `yaml:"imageTag" json:"imageTag"`
	Key                 string `yaml:"key" json:"key"`
	Username            string `yaml:"username" json:"username"`
	Password            string `yaml:"password" json:"password"`
	ImagePullPolicy     string `yaml:"imagePullPolicy" json:"imagePullPolicy"`
	DisablePublicImages bool   `yaml:"disablePublicImages" json:"disablePublicImages"`
}

// Tracing is the tracing struct within Values.
type Tracing struct {
	Address    string  `yaml:"address" json:"address"`
	Backend    string  `yaml:"backend" json:"backend"`
	SampleRate float32 `yaml:"sampleRate" json:"sampleRate"`
}

// Telemetry is the telemetry struct within Values.
type Telemetry struct {
	Exporters    *Exporter `yaml:"exporters,omitempty" json:"exporters,omitempty"`
	SamplerRatio float32   `yaml:"samplerRatio" json:"samplerRatio"`
}

// Exporter is the telemetry exporter struct within Values.
type Exporter struct {
	OTLP *OTLP `yaml:"otlp,omitempty" json:"otlp,omitempty"`
}

// OTLP is the telemetry OTLP struct within Values.
type OTLP struct {
	Host string `yaml:"host" json:"host"`
	Port int    `yaml:"port" json:"port"`
}

// MTLS is the mTLS struct within Values.
type MTLS struct {
	UpstreamAuthority     UpstreamAuthority `yaml:"upstreamAuthority,omitempty" json:"upstreamAuthority,omitempty"`
	Mode                  string            `yaml:"mode" json:"mode"`
	SVIDTTL               string            `yaml:"svidTTL" json:"svidTTL"`
	TrustDomain           string            `yaml:"trustDomain" json:"trustDomain"`
	PersistentStorage     string            `yaml:"persistentStorage" json:"persistentStorage"`
	SpireServerKeyManager string            `yaml:"spireServerKeyManager" json:"spireServerKeyManager"`
	CAKeyType             string            `yaml:"caKeyType" json:"caKeyType"`
	CATTL                 string            `yaml:"caTTL" json:"caTTL"`
}

// UpstreamAuthority is the upstreamAuthority struct within mTLS.
type UpstreamAuthority struct {
	Disk        *Disk        `yaml:"disk,omitempty" json:"disk,omitempty"`
	AWSPCA      *AWSPCA      `yaml:"awsPCA,omitempty" json:"awsPCA,omitempty"`
	AWSSecret   *AWSSecret   `yaml:"awsSecret,omitempty" json:"awsSecret,omitempty"`
	Vault       *Vault       `yaml:"vault,omitempty" json:"vault,omitempty"`
	CertManager *CertManager `yaml:"certManager,omitempty" json:"certManager,omitempty"`
}

// Disk is the disk struct within upstreamAuthority.
type Disk struct {
	Cert   string `yaml:"cert" json:"cert"`
	Key    string `yaml:"key" json:"key"`
	Bundle string `yaml:"bundle,omitempty" json:"bundle,omitempty"`
}

// AWSPCA is the awsPCA struct within upstreamAuthority.
type AWSPCA struct {
	Region                  string `yaml:"region" json:"region"`
	CertificateAuthorityArn string `yaml:"certificateAuthorityArn" json:"certificateAuthorityArn"`
	AWSAccessKeyID          string `yaml:"awsAccessKeyID" json:"awsAccessKeyID"`
	AWSSecretAccessKey      string `yaml:"awsSecretAccessKey" json:"awsSecretAccessKey"`
	CASigningTemplateArn    string `yaml:"caSigningTemplateArn,omitempty" json:"caSigningTemplateArn,omitempty"`
	SigningAlgorithm        string `yaml:"signingAlgorithm,omitempty" json:"signingAlgorithm,omitempty"`
	AssumeRoleArn           string `yaml:"assumeRoleArn,omitempty" json:"assumeRoleArn,omitempty"`
	Endpoint                string `yaml:"endpoint,omitempty" json:"endpoint,omitempty"`
	SupplementalBundle      string `yaml:"supplementalBundle,omitempty" json:"supplementalBundle,omitempty"`
}

// AWSSecret is the awsSecret struct within upstreamAuthority.
type AWSSecret struct {
	Region             string `yaml:"region" json:"region"`
	CertFileArn        string `yaml:"certFileArn" json:"certFileArn"`
	KeyFileArn         string `yaml:"keyFileArn" json:"keyFileArn"`
	AWSAccessKeyID     string `yaml:"awsAccessKeyID,omitempty" json:"awsAccessKeyID,omitempty"`
	AWSSecretAccessKey string `yaml:"awsSecretAccessKey,omitempty" json:"awsSecretAccessKey,omitempty"`
	AWSSecretToken     string `yaml:"awsSecretToken,omitempty" json:"awsSecretToken,omitempty"`
	AssumeRoleArn      string `yaml:"assumeRoleArn,omitempty" json:"assumeRoleArn,omitempty"`
}

// Vault is the vault struct within upstreamAuthority.
type Vault struct {
	TokenAuth          *TokenAuth   `yaml:"tokenAuth,omitempty" json:"tokenAuth,omitempty"`
	ApproleAuth        *ApproleAuth `yaml:"approleAuth,omitempty" json:"approleAuth,omitempty"`
	CertAuth           *CertAuth    `yaml:"certAuth,omitempty" json:"certAuth,omitempty"`
	CACert             string       `yaml:"caCert" json:"caCert"`
	PKIMountPoint      string       `yaml:"pkiMountPoint,omitempty" json:"pkiMountPoint,omitempty"`
	VaultAddr          string       `yaml:"vaultAddr" json:"vaultAddr"`
	Namespace          string       `yaml:"namespace" json:"namespace"`
	InsecureSkipVerify bool         `yaml:"insecureSkipVerify,omitempty" json:"insecureSkipVerify,omitempty"`
}

// CertAuth is the certAuth struct within vault.
type CertAuth struct {
	ClientCert         string `yaml:"clientCert" json:"clientCert"`
	ClientKey          string `yaml:"clientKey" json:"clientKey"`
	CertAuthMountPoint string `yaml:"certAuthMountPoint,omitempty" json:"certAuthMountPoint,omitempty"`
	CertAuthRoleName   string `yaml:"certAuthRoleName,omitempty" json:"certAuthRoleName,omitempty"`
}

// TokenAuth is the tokenAuth struct within vault.
type TokenAuth struct {
	Token string `yaml:"token" json:"token"`
}

// ApproleAuth is the approleAuth struct within vault.
type ApproleAuth struct {
	ApproleID             string `yaml:"approleID" json:"approleID"`
	ApproleSecretID       string `yaml:"approleSecretID" json:"approleSecretID"`
	ApproleAuthMountPoint string `yaml:"approleAuthMountPoint,omitempty" json:"approleAuthMountPoint,omitempty"`
}

// CertManager is the certManager struct within upstreamAuthority.
type CertManager struct {
	Namespace   string `yaml:"namespace" json:"namespace"`
	IssuerName  string `yaml:"issuerName" json:"issuerName"`
	IssuerKind  string `yaml:"issuerKind,omitempty" json:"issuerKind,omitempty"`
	IssuerGroup string `yaml:"issuerGroup,omitempty" json:"issuerGroup,omitempty"`
	KubeConfig  string `yaml:"kubeConfig,omitempty" json:"kubeConfig,omitempty"`
}

// GetBufferedFilesAndValues loads helm files and values.
func GetBufferedFilesAndValues() ([]*loader.BufferedFile, *Values, error) {
	root := "."
	var vals Values
	var files []*loader.BufferedFile
	err := fs.WalkDir(chart.HelmFiles(), root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		blob, err := fs.ReadFile(chart.HelmFiles(), path)
		if err != nil {
			return fmt.Errorf("error reading helm file: %w", err)
		}

		file := &loader.BufferedFile{
			Name: strings.TrimPrefix(path, root+"/"),
			Data: blob,
		}

		if file.Name == "values.yaml" {
			if err := yaml.Unmarshal(blob, &vals); err != nil {
				return fmt.Errorf("error unmarshaling default values: %w", err)
			}
		}

		files = append(files, file)

		return nil
	})

	return files, &vals, err
}

// GetDeployValues gets the values used to deploy the mesh.
// Returns both the struct and raw data.
func GetDeployValues(client k8s.Client, releaseName string) (*Values, []byte, error) {
	actionConfig, err := client.HelmAction(client.Namespace())
	if err != nil {
		return nil, nil, fmt.Errorf("error initializing helm action: %w", err)
	}

	getVals := action.NewGetValues(actionConfig)
	vals, err := getVals.Run(releaseName)
	if err != nil {
		return nil, nil, fmt.Errorf("could not get deploy configuration: %w", err)
	}

	jsonBytes, err := json.Marshal(vals)
	if err != nil {
		return nil, nil, fmt.Errorf("error marshaling values: %w", err)
	}

	var values Values
	if err := json.Unmarshal(jsonBytes, &values); err != nil {
		return nil, nil, fmt.Errorf("error unmarshaling values: %w", err)
	}

	return &values, jsonBytes, nil
}

// ConvertToMap converts a Values struct to a map[string]interface{}.
func (v *Values) ConvertToMap() (map[string]interface{}, error) {
	var valuesMap map[string]interface{}
	b, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("error marshaling values: %w", err)
	}
	if err := json.Unmarshal(b, &valuesMap); err != nil {
		return nil, fmt.Errorf("error unmarshaling values: %w", err)
	}

	return valuesMap, nil
}
