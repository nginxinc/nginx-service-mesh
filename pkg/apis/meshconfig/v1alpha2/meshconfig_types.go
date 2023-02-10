package v1alpha2

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// MeshConfig defines the top level CustomResource for holding the global static mesh configuration.
// This configuration is set on installation of the mesh and becomes immutable.
// To update the runtime configuration, see the MeshDynamicConfig CustomResource.
type MeshConfig struct { //nolint:govet // fieldalignment not desired
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the desired installation configuration for NGINX Service Mesh.
	Spec MeshConfigSpec `json:"spec"`

	// Status defines the configuration status for NGINX Service Mesh.
	// +optional
	Status MeshConfigStatus `json:"status"`
}

// MeshConfigSpec defines the desired installation configuration for NGINX Service Mesh.
type MeshConfigSpec struct { //nolint:govet // fieldalignment not desired
	// Mtls is the configuration for mutual TLS.
	Mtls MtlsSpec `json:"mtls"`

	// MeshConfigClassName is the name of the associated MeshConfigClass
	MeshConfigClassName string `json:"meshConfigClassName"`

	// AccessControlMode for service-to-service communication.
	AccessControlMode string `json:"accessControlMode"`

	// ClientMaxBodySize is NGINX client max body size.
	ClientMaxBodySize string `json:"clientMaxBodySize"`

	// Environment to deploy the mesh into.
	Environment string `json:"environment"`

	// Namespace that the NGINX Service Mesh control plane belongs to.
	Namespace string `json:"namespace"`

	// NGINXErrorLogLevel is the NGINX error log level.
	NGINXErrorLogLevel string `json:"nginxErrorLogLevel"`

	// NGINXLBMethod is the NGINX load balancing method.
	NGINXLBMethod string `json:"nginxLBMethod"`

	// NGINXLogFormat is the NGINX log format.
	NGINXLogFormat string `json:"nginxLogFormat"`

	// PrometheusAddress is the address of a Prometheus server deployed in your Kubernetes cluster.
	// +optional
	PrometheusAddress *string `json:"prometheusAddress,omitempty"`

	// Registry contains the NGINX Service Mesh image registry settings.
	Registry RegistrySpec `json:"registry"`

	// EnabledNamespaces is the list of namespaces where automatic sidecar injection is enabled.
	EnabledNamespaces []string `json:"enabledNamespaces"`

	// Telemetry is the configuration for telemetry.
	// +optional
	Telemetry *TelemetrySpec `json:"telemetry,omitempty"`

	// DisableAutoInjection disables automatic sidecar injection globally.
	DisableAutoInjection bool `json:"disableAutoInjection"`

	// EnableUDP traffic proxying (beta).
	EnableUDP bool `json:"enableUDP"`

	// Transparent is set when the mesh is removed to turn all sidecar proxies transparent.
	Transparent bool `json:"transparent"`
}

// MtlsSpec defines the mTLS configuration.
type MtlsSpec struct {
	// Mode for pod-to-pod communication.
	Mode string `json:"mode"`

	// CaKeyType is the key type used for the SPIRE Server CA.
	CaKeyType string `json:"caKeyType"`

	// CaTTL is the CA/signing key TTL in hours(h). Min value 24h. Max value 999999h.
	CaTTL string `json:"caTTL"`

	// SvidTTL is the TTL of certificates issued to workloads in hours(h) or minutes(m). Max value is 999999.
	SvidTTL string `json:"svidTTL"`

	// TrustDomain of the mesh.
	TrustDomain string `json:"trustDomain"`
}

// TelemetrySpec defines the OpenTelemetry configuration.
type TelemetrySpec struct {
	// Exporters is the exporters configuration for telemetry.
	Exporters ExportersSpec `json:"exporters"`

	// SamplerRatio is the percentage of traces that are processed and exported to the telemetry backend.
	SamplerRatio float32 `json:"samplerRatio"`
}

// ExportersSpec defines the telemetry exporters configuration.
type ExportersSpec struct {
	// Otlp is the configuration for an OTLP gRPC exporter.
	Otlp OtlpSpec `json:"otlp"`
}

// OtlpSpec defines the OTLP exporter configuration.
type OtlpSpec struct {
	// Host of the OpenTelemetry gRPC exporter to connect to.
	Host string `json:"host"`

	// Port of the OpenTelemetry gRPC exporter to connect to.
	Port int32 `json:"port"`
}

// RegistrySpec contains the NGINX Service Mesh image registry settings.
type RegistrySpec struct { //nolint:govet // fieldalignment not desired
	// Server is the hostname:port for registry and path to images.
	Server string `json:"server"`

	// ImageTag used for pulling images from registry.
	ImageTag string `json:"imageTag"`

	// ImagePullPolicy for NGINX Service Mesh images.
	ImagePullPolicy string `json:"imagePullPolicy"`

	// SidecarImage is the image to be used for the sidecar.
	SidecarImage string `json:"sidecarImage"`

	// SidecarInitImage is the image to be used for the sidecar init container.
	SidecarInitImage string `json:"sidecarInitImage"`

	// RegistryKeyName is the name of the registry key for pulling images.
	// +optional
	RegistryKeyName *string `json:"registryKeyName,omitempty"`

	// DisablePublicImages disables the pulling of third party images from public repositories.
	DisablePublicImages bool `json:"disablePublicImages"`
}

// MeshConfigStatus defines the configuration status for NGINX Service Mesh.
type MeshConfigStatus struct {
	// Transparent status is updated once the mesh controller
	// has successfully turned all sidecar proxies transparent.
	Transparent bool `json:"transparent"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// MeshConfigList is a list of MeshConfig resources.
type MeshConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []MeshConfig `json:"items"`
}
