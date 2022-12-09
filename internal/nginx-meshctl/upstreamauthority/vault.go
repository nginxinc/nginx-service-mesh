package upstreamauthority

import "github.com/nginxinc/nginx-service-mesh/pkg/helm"

type vaultConfig struct {
	TokenAuth          *tokenAuthConfig   `json:"token_auth"`
	AppRoleAuth        *appRoleAuthConfig `json:"approle_auth"`
	PKIMountPath       *string            `json:"pki_mount_path"`
	InsecureSkipVerify *bool              `json:"insecure_skip_verify"`
	CertAuth           *certAuthConfig    `json:"cert_auth"`
	CACertPath         string             `json:"ca_cert_path"`
	VaultAddr          string             `json:"vault_addr"`
	Namespace          string             `json:"namespace"`
}

type certAuthConfig struct {
	CertAuthRoleName   *string `json:"cert_auth_role_name"`
	CertAuthMountPoint *string `json:"cert_auth_mount_point"`
	ClientCertPath     string  `json:"client_cert_path"`
	ClientKeyPath      string  `json:"client_key_path"`
}

type tokenAuthConfig struct {
	Token string `json:"token"`
}

type appRoleAuthConfig struct {
	AppRoleAuthMountPoint *string `json:"approle_auth_mount_point"`
	AppRoleID             string  `json:"approle_id"`
	AppRoleSecretID       string  `json:"approle_secret_id"`
}

func (v vaultConfig) configure() (*helm.UpstreamAuthority, error) {
	upstreamAuthority := &helm.UpstreamAuthority{
		Vault: &helm.Vault{},
	}
	caCert, _, _, err := readCertFiles(v.CACertPath, "", "")
	if err != nil {
		return nil, err
	}

	var clientCert, key string
	if v.CertAuth != nil {
		clientCert, key, _, err = readCertFiles(v.CertAuth.ClientCertPath, v.CertAuth.ClientKeyPath, "")
		if err != nil {
			return nil, err
		}
	}

	upstreamAuthority.Vault = &helm.Vault{
		VaultAddr:          v.VaultAddr,
		Namespace:          v.Namespace,
		CACert:             caCert,
		PKIMountPoint:      dereferenceString(v.PKIMountPath),
		InsecureSkipVerify: dereferenceBool(v.InsecureSkipVerify),
	}
	if v.CertAuth != nil {
		upstreamAuthority.Vault.CertAuth = &helm.CertAuth{
			ClientCert:         clientCert,
			ClientKey:          key,
			CertAuthMountPoint: dereferenceString(v.CertAuth.CertAuthMountPoint),
			CertAuthRoleName:   dereferenceString(v.CertAuth.CertAuthRoleName),
		}
	}

	if v.TokenAuth != nil {
		upstreamAuthority.Vault.TokenAuth = &helm.TokenAuth{
			Token: v.TokenAuth.Token,
		}
	}

	if v.AppRoleAuth != nil {
		upstreamAuthority.Vault.ApproleAuth = &helm.ApproleAuth{
			ApproleID:             v.AppRoleAuth.AppRoleID,
			ApproleSecretID:       v.AppRoleAuth.AppRoleSecretID,
			ApproleAuthMountPoint: dereferenceString(v.AppRoleAuth.AppRoleAuthMountPoint),
		}
	}

	return upstreamAuthority, nil
}
