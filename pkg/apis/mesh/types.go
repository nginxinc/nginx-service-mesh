// Package mesh provides the types and functions for interacting with
// the NGINX Service Mesh API and configuration.
package mesh

// injector annotations and labels.
const (
	// InjectedAnnotation tells us if a pod has been injected.
	InjectedAnnotation = "injector.nsm.nginx.com/status"
	// AutoInjectLabel tells whether a pod should be injected with the sidecar.
	AutoInjectLabel = "injector.nsm.nginx.com/auto-inject"
)

// AutoInjectorPort is the port that the automatic injection webhook binds to.
const AutoInjectorPort = 9443

// ControllerVersionPort is the port that the controller runs the /version endpoint on.
const ControllerVersionPort = 8082

// Injected is used as the value in the InjectedAnnotation.
const Injected = "injected"

// Enabled is used as the value in the AutoInjectLabel.
const Enabled = "enabled"

// IgnoredNamespaces is a map of the namespaces that the service mesh will ignore.
var IgnoredNamespaces = map[string]bool{
	"kube-system": true,
}

// DeployLabel is the label key for deployment type of the resource.
const DeployLabel = "nsm.nginx.com/"

// SpiffeIDLabel is the label to tell SPIRE to issue certs.
const SpiffeIDLabel = "spiffe.io/spiffeid"

// proxy config annotations.
const (
	// IgnoreIncomingPortsAnnotation tells us which ports to ignore for incoming traffic.
	IgnoreIncomingPortsAnnotation = "config.nsm.nginx.com/ignore-incoming-ports"
	// IgnoreOutgoingPortsAnnotation tells us which ports to ignore for outgoing traffic.
	IgnoreOutgoingPortsAnnotation = "config.nsm.nginx.com/ignore-outgoing-ports"
	// MTLSModeAnnotation tells us the mtls-mode of the pod.
	MTLSModeAnnotation = "config.nsm.nginx.com/mtls-mode"
	// LoadBalancingAnnotation tells us the load balancing method for the service.
	LoadBalancingAnnotation = "config.nsm.nginx.com/lb-method"
	// DefaultEgressRouteAllowedAnnotation tells us if a pod is allowed to send egress traffic to the egress endpoint.
	DefaultEgressRouteAllowedAnnotation = "config.nsm.nginx.com/default-egress-allowed"
	// ClientMaxBodySizeAnnotation tells us the client-max-body-size of the pod.
	ClientMaxBodySizeAnnotation = "config.nsm.nginx.com/client-max-body-size"
)

// NATS channel names.
const (
	// NatsAgentConfigChannel sends the mesh config from mesh-api to agent.
	NatsAgentConfigChannel = "nginx.nsm.agent.config"
	// NatsAgentSubChannel sends a subscription and version notice from agent to mesh-api.
	NatsAgentSubChannel = "nginx.nsm.agent.subscription"
	// NatsAPIPingChannel sends a ping from mesh-api to agent on restart.
	NatsAPIPingChannel = "nginx.nsm.api.ping"
)

// k8s static resource names.
const (
	// MeshConfigMap is the name of the config map that holds the mesh config.
	MeshConfigMap = "meshconfig"
	// MeshConfigFileName is the name of the file where the mesh config is stored.
	MeshConfigFileName = "meshconfig.json"
	// NatsServer is the name of the nats-server service.
	NatsServer = "nats-server"
	// MeshAPI is the name of the mesh api.
	MeshAPI = "nginx-mesh-api"
	// MeshCertReloader is the name of the mesh cert reloader image.
	MeshCertReloader = "nginx-mesh-cert-reloader"
	// MeshSidecar is the name of the mesh sidecar.
	MeshSidecar = "nginx-mesh-sidecar"
	// MeshSidecarInit is the name of the mesh init container.
	MeshSidecarInit = "nginx-mesh-init"
	// MetricsService is the name of the traffic metrics service.
	MetricsService = "nginx-mesh-metrics-svc"
	// MetricsServiceAccount is the name of the service account of traffic metrics.
	MetricsServiceAccount = "nginx-mesh-metrics"
	// MetricsDeployment is the name of the traffic metrics deployment.
	MetricsDeployment = MetricsServiceAccount
	// HTTPRouteGroupKind is the kind for HTTPRouteGroups.
	HTTPRouteGroupKind = "HTTPRouteGroup"
	// TCPRouteKind is the kind of TcpRoutes.
	TCPRouteKind = "TCPRoute"
)

// NGINX Ingress Controller labels.
const (
	// EnableIngressLabel tells us if the pod is the NGINX Ingress Controller and if ingress traffic is enabled.
	EnableIngressLabel = "nsm.nginx.com/enable-ingress"
	// EnableEgressLabel tells us if the pod is the NGINX Ingress Controller and if egress traffic is enabled.
	EnableEgressLabel = "nsm.nginx.com/enable-egress"
)

// MTLS modes.
const (
	MtlsModeOff        = "off"
	MtlsModePermissive = "permissive"
	MtlsModeStrict     = "strict"
)

// MtlsModes are the supported mtls modes.
var MtlsModes = map[string]struct{}{
	MtlsModeOff:        {},
	MtlsModePermissive: {},
	MtlsModeStrict:     {},
}

// Access Control Modes.
const (
	AccessControlModeDeny  = "deny"
	AccessControlModeAllow = "allow"
)

// AccessControlModes are the supported access control modes.
var AccessControlModes = map[string]struct{}{
	AccessControlModeAllow: {},
	AccessControlModeDeny:  {},
}

// NGINX error log levels.
const (
	NginxErrorLogLevelDebug  = "debug"
	NginxErrorLogLevelInfo   = "info"
	NginxErrorLogLevelNotice = "notice"
	NginxErrorLogLevelWarn   = "warn"
	NginxErrorLogLevelError  = "error"
	NginxErrorLogLevelCrit   = "crit"
	NginxErrorLogLevelAlert  = "alert"
	NginxErrorLogLevelEmerg  = "emerg"
)

// NGINX log formats.
const (
	NginxLogFormatDefault = "default"
	NginxLogFormatJSON    = "json"
)

// NGINXLogFormats are the supported NGINX log formats.
var NGINXLogFormats = map[string]struct{}{
	NginxLogFormatDefault: {},
	NginxLogFormatJSON:    {},
}

// NGINX load balancing methods.
const (
	RoundRobin                 = "round_robin"
	LeastConn                  = "least_conn"
	LeastTime                  = "least_time"
	LeastTimeLastByte          = "least_time last_byte"
	LeastTimeLastByteInflight  = "least_time last_byte inflight"
	Random                     = "random"
	RandomTwo                  = "random two"
	RandomTwoLeastConn         = "random two least_conn"
	RandomTwoLeastTime         = "random two least_time"
	RandomTwoLeastTimeLastByte = "random two least_time=last_byte"
)

// LoadBalancingMethods are the available NGINX load balancing methods.
var LoadBalancingMethods = map[string]struct{}{
	RoundRobin:                 {},
	LeastConn:                  {},
	LeastTime:                  {},
	LeastTimeLastByte:          {},
	LeastTimeLastByteInflight:  {},
	Random:                     {},
	RandomTwo:                  {},
	RandomTwoLeastConn:         {},
	RandomTwoLeastTime:         {},
	RandomTwoLeastTimeLastByte: {},
}

// Kubernetes environments.
const (
	Kubernetes = "kubernetes"
	Openshift  = "openshift"
)

// Environments are the supported kubernetes environments.
var Environments = map[string]struct{}{
	Kubernetes: {},
	Openshift:  {},
}

// Svid cert and key names.
const (
	SvidFileName       = "tls.crt"
	SvidKeyFileName    = "tls.key"
	SvidBundleFileName = "rootca.pem"
)

// DefaultSamplerRatio is the default sampler ratio for telemetry.
const DefaultSamplerRatio = float32(0.01)

// MetricsConfig holds the data that may be dynamically updated at runtime for the nginx-mesh-metrics component.
type MetricsConfig struct {
	PromAddr *string `json:"PrometheusAddress,omitempty"`
}

// ExternalServiceAnnotation tells us if an endpoint is for an external service.
const ExternalServiceAnnotation = "service.nsm.nginx.com/external"
