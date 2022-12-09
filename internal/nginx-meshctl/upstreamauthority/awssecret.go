package upstreamauthority

import "github.com/nginxinc/nginx-service-mesh/pkg/helm"

type awsSecretConfig struct {
	AssumeRoleArn   *string `json:"assume_role_arn"`
	AccessKeyID     *string `json:"aws_access_key_id"`
	SecretAccessKey *string `json:"aws_secret_access_key"`
	SecretToken     *string `json:"aws_secret_token"`
	CertFileArn     string  `json:"cert_file_arn"`
	KeyFileArn      string  `json:"key_file_arn"`
	Region          string  `json:"region"`
}

func (secret awsSecretConfig) configure() (*helm.UpstreamAuthority, error) {
	upstreamAuthority := &helm.UpstreamAuthority{
		AWSSecret: &helm.AWSSecret{},
	}
	upstreamAuthority.AWSSecret = &helm.AWSSecret{
		Region:             secret.Region,
		CertFileArn:        secret.CertFileArn,
		KeyFileArn:         secret.KeyFileArn,
		AWSAccessKeyID:     dereferenceString(secret.AccessKeyID),
		AWSSecretAccessKey: dereferenceString(secret.SecretAccessKey),
		AWSSecretToken:     dereferenceString(secret.SecretToken),
		AssumeRoleArn:      dereferenceString(secret.AssumeRoleArn),
	}

	return upstreamAuthority, nil
}
