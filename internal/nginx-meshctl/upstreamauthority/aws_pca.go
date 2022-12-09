package upstreamauthority

import (
	"fmt"

	"github.com/nginxinc/nginx-service-mesh/pkg/helm"
)

type awsPCAConfig struct {
	Region                  string  `json:"region"`
	CertificateAuthorityARN string  `json:"certificate_authority_arn"`
	CASigningTemplateARN    *string `json:"ca_signing_template_arn"`
	SigningAlgorithm        *string `json:"signing_algorithm"`
	AssumeRoleArn           *string `json:"assume_role_arn"`
	Endpoint                *string `json:"endpoint"`
	SupplementalBundlePath  *string `json:"supplemental_bundle_path"`
	// Credentials
	AccessKeyID     string `json:"aws_access_key_id"`
	SecretAccessKey string `json:"aws_secret_access_key"`
}

func (pca awsPCAConfig) configure() (*helm.UpstreamAuthority, error) {
	upstreamAuthority := &helm.UpstreamAuthority{
		AWSPCA: &helm.AWSPCA{},
	}
	var bundle string
	if pca.SupplementalBundlePath != nil {
		var err error
		_, _, bundle, err = readCertFiles("", "", *pca.SupplementalBundlePath)
		if err != nil {
			return nil, fmt.Errorf("error with supplemental bundle: %w", err)
		}
	}
	upstreamAuthority.AWSPCA = &helm.AWSPCA{
		Region:                  pca.Region,
		CertificateAuthorityArn: pca.CertificateAuthorityARN,
		AWSAccessKeyID:          pca.AccessKeyID,
		AWSSecretAccessKey:      pca.SecretAccessKey,
		CASigningTemplateArn:    dereferenceString(pca.CASigningTemplateARN),
		SigningAlgorithm:        dereferenceString(pca.SigningAlgorithm),
		AssumeRoleArn:           dereferenceString(pca.AssumeRoleArn),
		Endpoint:                dereferenceString(pca.Endpoint),
		SupplementalBundle:      bundle,
	}

	return upstreamAuthority, nil
}
