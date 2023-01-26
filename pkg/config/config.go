// Package config contains the configuration sent to the agent for nginx.
package config

import (
	split "github.com/servicemeshinterface/smi-controller-sdk/apis/split/v1alpha3"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/nginxinc/nginx-service-mesh/pkg/apis/mesh"
	specs "github.com/nginxinc/nginx-service-mesh/pkg/apis/specs/v1alpha1"
)

// CombinedConfig contains all the configs consumed by the sidecar agent.
// For use by agent when unmarshaling config.
type CombinedConfig struct {
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
	HTTPAccessControl   map[string]AgentKeyval
	StreamAccessControl map[string]AgentKeyval
	MeshConfig          mesh.MeshConfig
}

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
	Method mesh.MeshConfigLoadBalancingMethod
	Block  Block
}

// String returns the string representation of an LBMethod.
func (lb LBMethod) String() string {
	var lbMethod string
	if lb.Method != mesh.MeshConfigLoadBalancingMethodRoundRobin {
		lbMethod = string(lb.Method)
		switch lbMethod {
		case string(mesh.MeshConfigLoadBalancingMethodLeastTime):
			if lb.Block == HTTP {
				lbMethod += " header"
			} else {
				lbMethod += " first_byte"
			}
		case string(mesh.MeshConfigLoadBalancingMethodRandomTwoLeastTime):
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
