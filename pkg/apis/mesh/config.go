package mesh

// FullMeshConfig defines the entire static configuration for NGINX Service Mesh.
type FullMeshConfig struct { //nolint:govet // fieldalignment not desired
	// Mtls is the configuration for mutual TLS.
	Mtls Mtls `yaml:"mtls" json:"mtls"`

	// AccessControlMode for service-to-service communication.
	AccessControlMode string `yaml:"accessControlMode" json:"accessControlMode"`

	// ClientMaxBodySize is NGINX client max body size.
	ClientMaxBodySize string `yaml:"clientMaxBodySize" json:"clientMaxBodySize"`

	// Environment to deploy the mesh into.
	Environment string `yaml:"environment" json:"environment"`

	// Namespace that the NGINX Service Mesh control plane belongs to.
	Namespace string `yaml:"namespace" json:"namespace"`

	// NGINXErrorLogLevel is the NGINX error log level.
	NGINXErrorLogLevel string `yaml:"nginxErrorLogLevel" json:"nginxErrorLogLevel"`

	// NGINXLBMethod is the NGINX load balancing method.
	NGINXLBMethod string `yaml:"nginxLBMethod" json:"nginxLBMethod"`

	// NGINXLogFormat is the NGINX log format.
	NGINXLogFormat string `yaml:"nginxLogFormat" json:"nginxLogFormat"`

	// PrometheusAddress is the address of a Prometheus server deployed in your Kubernetes cluster.
	PrometheusAddress string `yaml:"prometheusAddress" json:"prometheusAddress"`

	// Registry contains the NGINX Service Mesh image registry settings.
	Registry Registry `yaml:"registry" json:"registry"`

	// Telemetry is the configuration for telemetry.
	Telemetry *Telemetry `yaml:"telemetry,omitempty" json:"telemetry,omitempty"`

	// EnableUDP traffic proxying (beta).
	EnableUDP bool `yaml:"enableUDP" json:"enableUDP"`

	// Transparent is set when the mesh is removed to turn all sidecar proxies transparent.
	Transparent bool `yaml:"transparent" json:"transparent"`
}

// Mtls defines the mTLS configuration.
type Mtls struct {
	// Mode for pod-to-pod communication.
	Mode string `yaml:"mode" json:"mode"`

	// CaKeyType is the key type used for the SPIRE Server CA.
	CaKeyType string `yaml:"caKeyType" json:"caKeyType"`

	// CaTTL is the CA/signing key TTL in hours(h). Min value 24h. Max value 999999h.
	CaTTL string `yaml:"caTTL" json:"caTTL"`

	// SvidTTL is the TTL of certificates issued to workloads in hours(h) or minutes(m). Max value is 999999.
	SvidTTL string `yaml:"svidTTL" json:"svidTTL"`

	// TrustDomain of the mesh.
	TrustDomain string `yaml:"trustDomain" json:"trustDomain"`
}

// Telemetry defines the OpenTelemetry configuration.
type Telemetry struct {
	// Exporters is the exporters configuration for telemetry.
	Exporters *Exporters `yaml:"exporters,omitempty" json:"exporters,omitempty"`

	// SamplerRatio is the percentage of traces that are processed and exported to the telemetry backend.
	SamplerRatio *float32 `yaml:"samplerRatio,omitempty" json:"samplerRatio,omitempty"`
}

// Exporters defines the telemetry exporters configuration.
type Exporters struct {
	// Otlp is the configuration for an OTLP gRPC exporter.
	Otlp Otlp `yaml:"otlp,omitempty" json:"otlp,omitempty"`
}

// Otlp defines the OTLP exporter configuration.
type Otlp struct {
	// Host of the OpenTelemetry gRPC exporter to connect to.
	Host string `yaml:"host" json:"host"`

	// Port of the OpenTelemetry gRPC exporter to connect to.
	Port int32 `yaml:"port" json:"port"`
}

// Registry contains the NGINX Service Mesh image registry settings.
type Registry struct { //nolint:govet // fieldalignment not desired
	// Server is the hostname:port for registry and path to images.
	Server string `yaml:"server" json:"server"`

	// ImageTag used for pulling images from registry.
	ImageTag string `yaml:"imageTag" json:"imageTag"`

	// ImagePullPolicy for NGINX Service Mesh images.
	ImagePullPolicy string `yaml:"imagePullPolicy" json:"imagePullPolicy"`

	// SidecarImage is the image to be used for the sidecar.
	SidecarImage string `yaml:"sidecarImage" json:"sidecarImage"`

	// SidecarInitImage is the image to be used for the sidecar init container.
	SidecarInitImage string `yaml:"sidecarInitImage" json:"sidecarInitImage"`

	// RegistryKeyName is the name of the registry key for pulling images.
	RegistryKeyName string `yaml:"registryKeyName" json:"registryKeyName"`

	// DisablePublicImages disables the pulling of third party images from public repositories.
	DisablePublicImages bool `yaml:"disablePublicImages" json:"disablePublicImages"`
}

// DeepCopyInto performs a deepcopy of the FullMeshConfig.
func (in *FullMeshConfig) DeepCopyInto(out *FullMeshConfig) {
	*out = *in
	if in.Telemetry != nil {
		in, out := &in.Telemetry, &out.Telemetry
		*out = new(Telemetry)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopyInto performs a deepcopy of the Telemetry config.
func (in *Telemetry) DeepCopyInto(out *Telemetry) {
	*out = *in
	if in.Exporters != nil {
		in, out := &in.Exporters, &out.Exporters
		*out = new(Exporters)
		(*in).DeepCopyInto(*out)
	}
	if in.SamplerRatio != nil {
		in, out := &in.SamplerRatio, &out.SamplerRatio
		*out = new(float32)
		**out = **in
	}
}

// DeepCopyInto performs a deepcopy of the Exporters config.
func (in *Exporters) DeepCopyInto(out *Exporters) {
	*out = *in
	out.Otlp = in.Otlp
}
