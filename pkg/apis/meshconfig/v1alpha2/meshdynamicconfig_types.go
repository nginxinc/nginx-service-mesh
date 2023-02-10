package v1alpha2

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// MeshDynamicConfig defines the top level CustomResource for holding the dynamic mesh configuration.
// This configuration can be updated by a user at runtime to change the global mesh settings.
type MeshDynamicConfig struct { //nolint:govet // fieldalignment not desired
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the desired dynamic configuration for NGINX Service Mesh.
	Spec MeshDynamicConfigSpec `json:"spec"`

	// Status defines the dynamic configuration status for NGINX Service Mesh.
	// +optional
	Status MeshDynamicConfigStatus `json:"status"`
}

// MeshDynamicConfigSpec defines the desired dynamic configuration for NGINX Service Mesh.
type MeshDynamicConfigSpec struct { //nolint:govet // fieldalignment not desired
	// Mtls is the configuration for mutual TLS.
	// +optional
	Mtls *MtlsDynamicSpec `json:"mtls,omitempty"`

	// MeshConfigClassName is the name of the associated MeshConfigClass
	MeshConfigClassName string `json:"meshConfigClassName"`

	// AccessControlMode for service-to-service communication.
	// +optional
	AccessControlMode *string `json:"accessControlMode,omitempty"`

	// ClientMaxBodySize is NGINX client max body size.
	// +optional
	ClientMaxBodySize *string `json:"clientMaxBodySize,omitempty"`

	// NGINXErrorLogLevel is the NGINX error log level.
	// +optional
	NGINXErrorLogLevel *string `json:"nginxErrorLogLevel,omitempty"`

	// NGINXLBMethod is the NGINX load balancing method.
	// +optional
	NGINXLBMethod *string `json:"nginxLBMethod,omitempty"`

	// NGINXLogFormat is the NGINX log format.
	// +optional
	NGINXLogFormat *string `json:"nginxLogFormat,omitempty"`

	// PrometheusAddress is the address of a Prometheus server deployed in your Kubernetes cluster.
	// +optional
	PrometheusAddress *string `json:"prometheusAddress,omitempty"`

	// EnabledNamespaces is the list of namespaces where automatic sidecar injection is enabled.
	// +optional
	EnabledNamespaces *[]string `json:"enabledNamespaces,omitempty"`

	// Telemetry is the configuration for telemetry.
	// +optional
	Telemetry *TelemetryDynamicSpec `json:"telemetry,omitempty"`

	// DisableAutoInjection disables automatic sidecar injection globally.
	// +optional
	DisableAutoInjection *bool `json:"disableAutoInjection,omitempty"`
}

// MtlsDynamicSpec defines the mTLS configuration.
type MtlsDynamicSpec struct {
	// Mode for pod-to-pod communication.
	// +optional
	Mode *string `json:"mode,omitempty"`

	// CaKeyType is the key type used for the SPIRE Server CA.
	// +optional
	CaKeyType *string `json:"caKeyType,omitempty"`

	// CaTTL is the CA/signing key TTL in hours(h). Min value 24h. Max value 999999h.
	// +optional
	CaTTL *string `json:"caTTL,omitempty"`

	// SvidTTL is the TTL of certificates issued to workloads in hours(h) or minutes(m). Max value is 999999.
	// +optional
	SvidTTL *string `json:"svidTTL,omitempty"`
}

// TelemetryDynamicSpec defines the OpenTelemetry configuration.
type TelemetryDynamicSpec struct {
	// Exporters is the exporters configuration for telemetry.
	// +optional
	Exporters *ExportersDynamicSpec `json:"exporters,omitempty"`

	// SamplerRatio is the percentage of traces that are processed and exported to the telemetry backend.
	// +optional
	SamplerRatio *float32 `json:"samplerRatio,omitempty"`
}

// ExportersDynamicSpec defines the telemetry exporters configuration.
type ExportersDynamicSpec struct {
	// Otlp is the configuration for an OTLP gRPC exporter.
	Otlp OtlpDynamicSpec `json:"otlp"`
}

// OtlpDynamicSpec defines the OTLP exporter configuration.
type OtlpDynamicSpec struct {
	// Host of the OpenTelemetry gRPC exporter to connect to.
	// +optional
	Host *string `json:"host,omitempty"`

	// Port of the OpenTelemetry gRPC exporter to connect to.
	// +optional
	Port *int32 `json:"port,omitempty"`
}

// MeshDynamiconfigStatus defines the dynamic configuration status for NGINX Service Mesh.
type MeshDynamicConfigStatus struct{}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// MeshDynamicConfigList is a list of MeshDynamicConfig resources.
type MeshDynamicConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []MeshDynamicConfig `json:"items"`
}
