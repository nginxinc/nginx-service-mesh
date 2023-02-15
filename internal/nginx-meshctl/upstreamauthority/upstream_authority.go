// Package upstreamauthority is responsible for converting upstream authority
// configuration to Helm values configuration
package upstreamauthority

import (
	"bytes"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"os"

	"github.com/xeipuuv/gojsonschema"
	convert "sigs.k8s.io/yaml"

	"github.com/nginxinc/nginx-service-mesh/pkg/helm"
)

const (
	certificate = "certificate"
	key         = "key"
)

type versions map[string]struct{}

var supportedVersions versions = map[string]struct{}{"v1": {}}

// config holds the meta and upstream authority configuration as specified
// by the user.
type config struct {
	uaUnmarshaler uaUnmarshaler
	meta
}

// meta holds the meta config for the upstream authority.
type meta struct {
	APIVersion string `json:"apiVersion"`
	Name       string `json:"upstreamAuthority"`
}

// uaUnmarshaler is a wrapper for upstream authority to allow "config" field
// to be unmarshalled.
type uaUnmarshaler struct {
	UpstreamAuthority upstreamAuthority `json:"config"`
}

type upstreamAuthority interface {
	// write upstream authority spire config
	configure() (*helm.UpstreamAuthority, error)
}

var (
	errInput       = errors.New("could not validate upstream authority input")
	errVersion     = errors.New("unsupported upstream authority version")
	errBadChain    = errors.New("chain not allowed")
	errPemEncoding = errors.New("not pem encoded")
)

// GetUpstreamAuthorityValues reads an upstream authority file and converts it to what Helm expects.
func GetUpstreamAuthorityValues(file string) (*helm.UpstreamAuthority, error) {
	b, err := os.ReadFile(file) //nolint:gosec,varnamelen // uses mTLS filenames from deployment config
	// see Deploy() for more information
	if err != nil {
		return nil, err
	}
	b, err = convert.YAMLToJSON(b)
	if err != nil {
		return nil, fmt.Errorf("could not convert upstream authority yaml: %w", err)
	}

	schemaLoader := gojsonschema.NewStringLoader(uaSchema)
	jsonLoader := gojsonschema.NewBytesLoader(b)
	res, err := gojsonschema.Validate(schemaLoader, jsonLoader)
	if err != nil {
		return nil, fmt.Errorf("could not validate upstream authority input: %w", err)
	}
	if !res.Valid() {
		var errStr string
		for _, desc := range res.Errors() {
			errStr += fmt.Sprintf("\t%v\n", desc)
		}

		return nil, fmt.Errorf("%w: %s", errInput, errStr)
	}

	var upstreamAuthority config
	err = json.Unmarshal(b, &upstreamAuthority)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal upstream authority config: %w", err)
	}

	if _, ok := supportedVersions[upstreamAuthority.APIVersion]; !ok {
		return nil, fmt.Errorf("%w: %v", errVersion, upstreamAuthority.APIVersion)
	}

	return upstreamAuthority.uaUnmarshaler.UpstreamAuthority.configure()
}

func (c *config) UnmarshalJSON(blob []byte) error {
	if err := json.Unmarshal(blob, &c.meta); err != nil {
		return err
	}

	upstreamAuthority := &c.uaUnmarshaler.UpstreamAuthority
	switch c.Name {
	case "disk":
		*upstreamAuthority = &diskConfig{}
	case "aws_pca":
		*upstreamAuthority = &awsPCAConfig{}
	case "awssecret":
		*upstreamAuthority = &awsSecretConfig{}
	case "vault":
		*upstreamAuthority = &vaultConfig{}
	case "cert-manager":
		*upstreamAuthority = &certManagerConfig{}
	}

	return json.Unmarshal(blob, &c.uaUnmarshaler)
}

// read the cert, key, and bundle files and return their contents.
func readCertFiles(
	certFile,
	keyFile,
	bundleFile string,
) (string, string, string, error) {
	var upstreamCert, upstreamKey, upstreamBundle string
	if certFile != "" {
		upstreamCertBytes, err := os.ReadFile(certFile) //nolint:gosec // file comes from cli flag
		if err != nil {
			return "", "", "", fmt.Errorf("error reading upstream certificate file: %w", err)
		}

		err = isPem(upstreamCertBytes, false, certificate)
		if err != nil {
			return "", "", "", fmt.Errorf("invalid certificate: %w", err)
		}
		upstreamCert = string(upstreamCertBytes)
	}

	if keyFile != "" {
		upstreamKeyBytes, err := os.ReadFile(keyFile) //nolint
		if err != nil {
			return "", "", "", fmt.Errorf("error reading upstream certificate key file: %w", err)
		}

		err = isPem(upstreamKeyBytes, false, key)
		if err != nil {
			return "", "", "", fmt.Errorf("invalid private key: %w", err)
		}
		upstreamKey = string(upstreamKeyBytes)
	}

	if bundleFile != "" {
		upstreamBundleBytes, err := os.ReadFile(bundleFile) //nolint
		if err != nil {
			return "", "", "", fmt.Errorf("error reading upstream certificate bundle file: %w", err)
		}

		err = isPem(upstreamBundleBytes, true, certificate)
		if err != nil {
			return "", "", "", fmt.Errorf("invalid bundle: %w", err)
		}
		upstreamBundle = string(upstreamBundleBytes)
	}

	return upstreamCert, upstreamKey, upstreamBundle, nil
}

func isPem(contents []byte, chain bool, resourceType string) error {
	firstPass := true
	var block *pem.Block
	for {
		if !firstPass && !chain {
			return fmt.Errorf("%w: %v", errBadChain, resourceType)
		}
		block, contents = pem.Decode(bytes.TrimSpace(contents))
		if block == nil {
			return fmt.Errorf("%w: %v", errPemEncoding, resourceType)
		}
		if len(contents) == 0 {
			return nil
		}
		firstPass = false
	}
}

func dereferenceString(str *string) string {
	if str != nil {
		return *str
	}

	return ""
}

func dereferenceBool(b *bool) bool {
	if b != nil {
		return *b
	}

	return false
}
