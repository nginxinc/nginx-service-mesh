package v1alpha1

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CircuitBreaker creates a breaking point at which it will deliver a static
// response rather than continue sending traffic to a backend.
type CircuitBreaker struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the circuit breaker spec for a destination's traffic
	Spec CircuitBreakerSpec `json:"spec"`
}

// CircuitBreakerSpec defines the circuit breaker spec that restricts connections
// to the destination.
type CircuitBreakerSpec struct {
	// Destination defines which destination to include in the circuit breaker
	Destination v1.ObjectReference `json:"destination"`

	// Defines a fallback service that should be routed to in the event that
	// the circuit breaker trips, rather than returning an error.
	// +optional
	Fallback FallbackSpec `json:"fallback,omitempty"`

	// Errors sets the number of errors before the circuit breaker trips
	Errors int `json:"errors"`

	// TimeoutSeconds sets the timeout the errors must fall within to trip the circuit
	// breaker. Also defines how long the circuit breaker will be tripped before
	// allowing connections to the destination again.
	TimeoutSeconds int `json:"timeoutSeconds"`
}

// FallbackSpec defines the fallback service spec to redirect traffic to when
// a circuit trips.
type FallbackSpec struct {
	// Service is the name of the Kubernetes Service to send traffic to.
	// Should be of the form <namespace>/<name>. If namespace is not specified,
	// defaults to the 'default' namespace.
	// +optional
	Service string `json:"service,omitempty"`

	// Port is the port on the Service to send traffic to. Defaults to 80.
	// +optional
	Port int `json:"port,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CircuitBreakerList satisfies K8s code gen requirements.
type CircuitBreakerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []CircuitBreaker `json:"items"`
}
