package v1alpha2

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RateLimit creates a maximum flow rate between resources. This can include:
//
//	Deployments
//	Services
type RateLimit struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the limit set on traffic between the specified resources
	Spec RateLimitSpec `json:"spec"`
}

// RateLimitSpec defines the limit spec that restricts the flow of traffic.
type RateLimitSpec struct {
	// Delay sets the amount a request will be delayed (in seconds) when the rate
	// limit is hit. Set to "nodelay" to send back an 503 status immediately
	Delay *intstr.IntOrString `json:"delay,omitempty"`

	// Destination is the resource to which traffic should be rate limited
	Destination v1.ObjectReference `json:"destination"`

	// Name of Rate Limit, i.e. 10rs
	Name string `json:"name"`

	// Maximum allowed rate of traffic
	Rate string `json:"rate"`

	// Sources defines from where traffic should be limited
	// +optional
	Sources []v1.ObjectReference `json:"sources,omitempty"`

	// Rules allows defining a list of HTTP Route Groups that this rate limit
	// object should match.
	// +optional
	Rules []RateLimitRule `json:"rules,omitempty"`

	// Burst sets the maximum number of requests a client can make in excess of rate
	// see: https://www.nginx.com/blog/rate-limiting-nginx/#bursts
	// +optional
	Burst int `json:"burst,omitempty"`
}

// RateLimitRule is the TrafficSpec that applies to a Rate Limit.
type RateLimitRule struct {
	// Kind is the kind of TrafficSpec to allow.
	Kind string `json:"kind"`
	// Name of the TrafficSpec to use.
	Name string `json:"name"`
	// Matches is a list of TrafficSpec routes that are applied to the Rate Limit object.
	// +optional
	Matches []string `json:"matches,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RateLimitList satisfies K8s code gen requirements.
type RateLimitList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []RateLimit `json:"items"`
}
