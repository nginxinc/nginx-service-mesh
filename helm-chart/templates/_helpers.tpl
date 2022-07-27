{{- define "nats.image-server" -}}
{{- if not .Values.registry.disablePublicImages }}{{ else }}{{ .Values.registry.server }}/{{ end }}
{{- end }}

{{- define "spire.image-server" -}}
{{- if not .Values.registry.disablePublicImages }}gcr.io/spiffe-io{{ else }}{{ .Values.registry.server }}{{ end }}
{{- end }}

{{- define "node-driver.image-server" -}}
{{- if not .Values.registry.disablePublicImages }}k8s.gcr.io/sig-storage{{ else }}{{ .Values.registry.server }}{{ end }}
{{- end }}

{{- define "hook.image-server" -}}
{{- if not .Values.registry.disablePublicImages }}bitnami{{ else }}{{ .Values.registry.server }}{{ end }}
{{- end }}

{{- define "ubuntu.image-server" -}}
{{- if not .Values.registry.disablePublicImages }}{{ else }}{{ .Values.registry.server }}/{{ end }}
{{- end }}

{{- define "registry-key-name" -}}
nginx-mesh-registry-key
{{- end }}

{{- define "docker-config-json" -}}
{{- if (and (.Values.registry.username) (.Values.registry.password)) }}
{
    "auths": {
        {{ quote .Values.registry.server }}: {
            "username": {{ quote .Values.registry.username }},
            "password": {{ quote .Values.registry.password }},
            "auth": {{ printf "%s:%s" .Values.registry.username .Values.registry.password | b64enc | quote }}
        }
    }
}
{{- else if (.Values.registry.key) }}
{
    "auths": {
        {{ quote .Values.registry.server }}: {
            "username": "_json_key",
            "password": {{ quote .Values.registry.key }}
        }
    }
}
{{- end }}
{{- end }}

{{/*
Define the name of the key where the Upstream Authority secret data is stored.
*/}}
{{- define "ua-secret-name" -}}
{{- if .Values.mtls.upstreamAuthority.awsPCA -}} {{- if .Values.mtls.upstreamAuthority.awsPCA.awsAccessKeyID -}}
credentials {{- end }}
{{- else if .Values.mtls.upstreamAuthority.disk -}}
upstreamCA.key
{{- else if .Values.mtls.upstreamAuthority.vault }}{{ if .Values.mtls.upstreamAuthority.vault.certAuth -}}
upstreamClient.key{{ end }}
{{- else if .Values.mtls.upstreamAuthority.certManager }}{{ if .Values.mtls.upstreamAuthority.certManager.kubeConfig -}}
cert-manager-kubeconfig{{ end }}
{{- end }}
{{- end }}

{{/*
Define the name of the mount path where the Upstream Authority secret data is stored.
*/}}
{{- define "ua-secret-mountpath" -}}
{{- if and .Values.mtls.upstreamAuthority.awsPCA -}} {{- if .Values.mtls.upstreamAuthority.awsPCA.awsAccessKeyID -}}
/root/.aws {{- end }}
{{- else if .Values.mtls.upstreamAuthority.disk -}}
/run/spire/secrets
{{- else if .Values.mtls.upstreamAuthority.vault }}{{ if .Values.mtls.upstreamAuthority.vault.certAuth -}}
/run/spire/secrets{{ end }}
{{- else if .Values.mtls.upstreamAuthority.certManager }}{{ if .Values.mtls.upstreamAuthority.certManager.kubeConfig -}}
/run/spire/secrets{{ end }}
{{- end }}
{{- end }}

{{/*
Define the upstream certificate to be used for the Upstream Authority.
*/}}
{{- define "ua-upstream-cert" -}}
{{- if .Values.mtls.upstreamAuthority.disk -}}
upstreamCA.crt: {{ quote .Values.mtls.upstreamAuthority.disk.cert }}
{{- else if .Values.mtls.upstreamAuthority.vault -}}
upstreamCA.crt: {{ quote .Values.mtls.upstreamAuthority.vault.caCert }}
{{- end }}
{{- end }}

{{/*
Define the upstream bundle to be used for the Upstream Authority.
*/}}
{{- define "ua-upstream-bundle" -}}
{{- if .Values.mtls.upstreamAuthority.disk }}{{ if .Values.mtls.upstreamAuthority.disk.bundle -}}
upstreamBundle.crt: {{ quote .Values.mtls.upstreamAuthority.disk.bundle }}{{ end }}
{{- else if .Values.mtls.upstreamAuthority.awsPCA }}{{ if .Values.mtls.upstreamAuthority.awsPCA.supplementalBundle -}}
upstreamBundle.crt: {{ quote .Values.mtls.upstreamAuthority.awsPCA.supplementalBundle }}{{ end }}
{{- end }}
{{- end }}

{{/*
Define the Upstream Authority value to be stored in the Secret.
*/}}
{{- define "ua-secret-value" -}}
{{- if .Values.mtls.upstreamAuthority.awsPCA -}}
{{ tpl (.Files.Get "configs/upstreamAuthority/aws-credentials.conf") . | b64enc }}
{{- else if .Values.mtls.upstreamAuthority.disk -}}
{{ .Values.mtls.upstreamAuthority.disk.key | b64enc }}
{{- else if .Values.mtls.upstreamAuthority.vault }}{{ if .Values.mtls.upstreamAuthority.vault.certAuth -}}
{{ .Values.mtls.upstreamAuthority.vault.certAuth.clientKey | b64enc }}{{ end }}
{{- else if .Values.mtls.upstreamAuthority.certManager }}{{ if .Values.mtls.upstreamAuthority.certManager.kubeConfig -}}
{{ .Values.mtls.upstreamAuthority.certManager.kubeConfig | b64enc }}{{ end }}
{{- end }}
{{- end }}

{{/*
Define variables associated with the Vault Upstream Authority.
*/}}

{{- define "ua-vault-env-name" -}}
{{- if .Values.mtls.upstreamAuthority.vault -}}
{{- if .Values.mtls.upstreamAuthority.vault.tokenAuth -}}
VAULT_TOKEN
{{- else if .Values.mtls.upstreamAuthority.vault.approleAuth -}}
VAULT_APPROLE_SECRET_ID
{{- end }}
{{- end }}
{{- end }}

{{- define "ua-vault-env-value" -}}
{{- if .Values.mtls.upstreamAuthority.vault -}}
{{- if .Values.mtls.upstreamAuthority.vault.tokenAuth -}}
{{ b64enc .Values.mtls.upstreamAuthority.vault.tokenAuth.token }}
{{- else if .Values.mtls.upstreamAuthority.vault.approleAuth -}}
{{ b64enc .Values.mtls.upstreamAuthority.vault.approleAuth.approleSecretID }}
{{- end }}
{{- end }}
{{- end }}

{{- define "ua-upstream-client-cert" -}}
{{- if .Values.mtls.upstreamAuthority.vault -}}
{{- if .Values.mtls.upstreamAuthority.vault.certAuth -}}
upstreamClient.crt: {{ quote .Values.mtls.upstreamAuthority.vault.certAuth.clientCert }}
{{- end }}
{{- end }}
{{- end }}
