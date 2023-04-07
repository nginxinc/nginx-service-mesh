// Package sidecar contains the configuration sent to the agent for nginx.
package sidecar

import (
	split "github.com/servicemeshinterface/smi-controller-sdk/apis/split/v1alpha3"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/nginxinc/nginx-service-mesh/pkg/apis/mesh"
	"github.com/nginxinc/nginx-service-mesh/pkg/apis/mesh/v1alpha1"
	specs "github.com/nginxinc/nginx-service-mesh/pkg/apis/specs/v1alpha1"
)

// Config contains all the configs consumed by the sidecar agent.
type Config struct {
	Pods                map[string]Pod
	ServiceAddresses    AgentKeyval
	HTTPPlaceholders    AgentKeyval
	StreamPlaceholders  AgentKeyval
	HTTPSvcNames        AgentKeyval
	StreamSvcNames      AgentKeyval
	Redirects           AgentKeyval
	HTTPLBMethods       map[string]string
	StreamLBMethods     map[string]string
	HTTPUpstreams       map[string][]UpstreamServer
	StreamUpstreams     map[string][]UpstreamServer
	HTTPEgressUpstream  *EgressEndpoint
	TrafficSplits       map[string]AgentTrafficSplit
	RateLimits          AgentLimit
	CircuitBreakers     AgentBreaker
	RetryTimeouts       map[string]AgentKeyval
	HTTPAccessControl   map[string]AgentKeyval
	StreamAccessControl map[string]AgentKeyval
	MeshConfig          mesh.FullMeshConfig
}

// ports that the NGINX sidecar proxy listens on.
const (
	// MetricsPort is the Prometheus metrics port.
	MetricsPort = 8887
	// IncomingPort is the incoming HTTP port.
	IncomingPort = 8888
	// OutgoingPort is the outgoing HTTP port.
	OutgoingPort = 8889
	// IncomingPermissivePort is the incoming HTTP port when mTLS mode is permissive.
	IncomingPermissivePort = 8890
	// IncomingGrpcPort is the incoming gRPC port.
	IncomingGrpcPort = 8891
	// OutgoingGrpcPort is the outgoing gRPC port.
	OutgoingGrpcPort = 8892
	// IncomingGrpcPermissivePort is the incoming gRPC port when mTLS mode is permissive.
	IncomingGrpcPermissivePort = 8893
	// OutgoingDefaultEgressPort is the outgoing port for egress traffic when NGINX Ingress Controller is deployed as an egress controller.
	OutgoingDefaultEgressPort = 8894
	// OutgoingRedirectPort is the port that redirects requests to another port in the proxy based on the request protocol.
	OutgoingRedirectPort = 8900
	// IncomingRedirectPort is the port that redirects requests to another port in the proxy based on the request protocol.
	IncomingRedirectPort = 8901
	// OutgoingNotInKeyvalPort is the outgoing port for destinations that are not a part of the mesh.
	OutgoingNotInKeyvalPort = 8902
	// IncomingNotInKeyvalPort is the incoming port that handles requests from Services not in the mesh.
	IncomingNotInKeyvalPort = 8903
	// IncomingTCPPort is the incoming TCP port.
	IncomingTCPPort = 8904
	// IncomingTCPDenyPort denies TCP traffic if it is not part of the mesh or if access to the sidecar is not allowed.
	IncomingTCPDenyPort = 8905
	// OutgoingTCPPort is the outgoing TCP port.
	OutgoingTCPPort = 8906
	// IncomingTCPPermissivePort is the incoming TCP port when mTLS mode is permissive.
	IncomingTCPPermissivePort = 8907
	// OutgoingUDPPort is the outgoing UDP port.
	OutgoingUDPPort = 8908
	// IncomingUDPPort is the incoming UDP port.
	IncomingUDPPort = 8909
	// PlusAPIPort is the NGINX Plus API port. This is not accessible outside of the sidecar proxy.
	PlusAPIPort = 8886
	// RedirectHealthPort is the port that redirects HTTP health probes to the application container.
	RedirectHealthPort = 8895
	// RedirectHealthHTTPSPort is the port that redirects HTTPS health probes to the application container.
	RedirectHealthHTTPSPort = 8896
)

// AgentKeyval holds the data for configuring a single keyval in the agent.
type AgentKeyval map[string]string

// Upstream should correspond to a service DNS name.
type Upstream struct {
	Name            string
	UpstreamServers []UpstreamServer
	Block           Block
}

// UpstreamServer defines an upstream address and port.
type UpstreamServer struct {
	Address string `json:"address"`
	Port    int32  `json:"port"`
}

// LBMethod represents a load balancing method for an nginx block.
type LBMethod struct {
	Method string
	Block  Block
}

// String returns the string representation of an LBMethod.
func (lb LBMethod) String() string {
	var lbMethod string
	if lb.Method != mesh.RoundRobin {
		lbMethod = lb.Method
		switch lbMethod {
		case string(mesh.LeastTime):
			if lb.Block == HTTP {
				lbMethod += " header"
			} else {
				lbMethod += " first_byte"
			}
		case string(mesh.RandomTwoLeastTime):
			if lb.Block == HTTP {
				lbMethod += "=header"
			} else {
				lbMethod += "=first_byte"
			}
		}
		lbMethod += ";"
	}

	return lbMethod
}

// AgentTrafficSplit mirrors a split.TrafficSplitSpec, but uses
// a map of specs.HTTPMatch json strings instead of v1.TypedLocalObjectReference,
// for easier handling by the agent.
type AgentTrafficSplit struct {
	// Service represents the apex service.
	Service string `json:"service"`

	// Matches is a string representation of a list of specs.HTTPMatch that should be applied to the traffic split.
	Matches string `json:"matches,omitempty"`

	// Backends defines a list of Kubernetes services
	// used as the traffic split destination.
	Backends []split.TrafficSplitBackend `json:"backends"`
}

// Equals returns whether or not two AgentTrafficSplits are equal.
func (a *AgentTrafficSplit) Equals(tsplit AgentTrafficSplit) bool {
	if a.Service != tsplit.Service {
		return false
	}
	if len(a.Backends) != len(tsplit.Backends) || len(a.Matches) != len(tsplit.Matches) {
		return false
	}

	for _, backend := range a.Backends {
		if !TrafficSplitBackendExists(backend, tsplit.Backends) {
			return false
		}
	}

	return a.Matches == tsplit.Matches
}

// TrafficSplitBackendExists returns whether or not a TrafficSplitBackend exists in the list.
func TrafficSplitBackendExists(backend split.TrafficSplitBackend, backends []split.TrafficSplitBackend) bool {
	for _, n := range backends {
		if n == backend {
			return true
		}
	}

	return false
}

// NginxDynSplitBackend is the expected backend struct of ngx_http_dyn_split_module.
type NginxDynSplitBackend struct {
	Service string `json:"name"`
	Weight  int    `json:"weight"`
}

// AgentRateLimit is a wrapper around the RateLimitSpec that contains a string of
// specs.HTTPMatch instead of the rules field.
type AgentRateLimit struct {
	// Delay sets the amount a request will be delayed (in seconds) when the rate
	// limit is hit. Set to "nodelay" to send back an 503 status immediately
	// see: https://www.nginx.com/blog/rate-limiting-nginx/#bursts
	// +optional
	Delay *intstr.IntOrString `json:"delay,omitempty"`

	// Name of Rate Limit, i.e. 10rs
	Name string `json:"name"`

	// Maximum allowed rate of traffic
	Rate string `json:"rate"`

	// Matches is a string representation of a list of specs.HTTPMatch that should be applied to the rate limit.
	Matches string `json:"matches,omitempty"`

	// Sources defines from where traffic should be limited
	// +optional
	Sources []v1.ObjectReference `json:"sources,omitempty"`

	// Burst sets the maximum number of requests a client can make in excess of rate
	// see: https://www.nginx.com/blog/rate-limiting-nginx/#bursts
	// +optional
	Burst int `json:"burst,omitempty"`
}

// AgentLimit holds a one-to-one mapping of how the agent will configure
// rate limiting.
type AgentLimit map[string][]AgentRateLimit

// NewAgentLimit returns an initialized map from dest string to array of sources.
func NewAgentLimit() AgentLimit {
	return make(map[string][]AgentRateLimit)
}

// AgentBreaker is a map of destination names to their associated circuit breaker specs.
type AgentBreaker map[string]specs.CircuitBreakerSpec

// AgentRetryTimeout is a map of destination names to their associated retry/timeout specs.
type AgentRetryTimeout map[string]v1alpha1.RetryTimeoutConfigSpec

// Egress ports.
const (
	EgressSSLPort = 443
	EgressPort    = 80
)

// EgressEndpoint contains the DNS name and the upstream servers for egress.
type EgressEndpoint struct {
	DNSName   string
	Upstreams []UpstreamServer
}
