package v1alpha1

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RetryTimeoutConfig creates a timeout and/or retry policy for a target.
type RetryTimeoutConfig struct {
	// Spec defines the retry and timeout spec for requests to a target.
	Spec RetryTimeoutConfigSpec `json:"spec"`

	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`
}

// RetryTimeoutConfigSpec defines the retry and timeout configuration for a target.
type RetryTimeoutConfigSpec struct {
	// TargetRef defines which target the retry and timeout applies to.
	TargetRef TargetRefSpec `json:"targetRef"`

	// Default is the default policy configuration to apply to the targetRef.
	// Currently this isn't useful, but adding it now to minimize changes
	// once override values come into play. That's also why "omitempty" is there.
	Default RetryTimeoutDefaultSpec `json:"default,omitempty"`
}

// TargetRef defines the resource a policy is applied to.
// TargetRef embeds v1.ObjectReference and adds the "Port" field.
type TargetRefSpec struct {
	// Port is the port of a resource a policy is applied to if the
	// resource has multiple ports defined.
	// +optional
	Port *int `json:"port,omitempty"`

	v1.ObjectReference
}

// RetryTimeoutDefaultSpec contains the retry and timeout specifications.
type RetryTimeoutDefaultSpec struct {
	// Timeout is the time to wait for requests to the targetRef to finish before
	// aborting the request and returning an error.
	// Valid timeouts are integers followed immediately by "m", "s", or "ms" to
	// specify if the timeouts are minutes, seconds or milliseconds. Example: "500ms"
	// +optional
	Timeout string `json:"timeout,omitempty"`

	// Retry is the configuration for retrying failed requests to the targetRef.
	// +optional
	Retry RetrySpec `json:"retry,omitempty"`
}

// RetrySpec is the configuration for retrying failed requests to the targetRef.
type RetrySpec struct {
	// Count is the number of times to retry a failed request before giving up.
	// Defaults to 500ms delay between retries if backoff is not specified.
	// +optional
	Count *int `json:"count,omitempty"`

	// Backoff is the configuration for exponentionally backing off between retries.
	// +optional
	Backoff BackoffSpec `json:"backoff,omitempty"`
}

// BackoffSpec is the configuration for exponentionally backing off between retries.
type BackoffSpec struct {
	// InitialInterval is the time between the original request and the first retry.
	// Subsequent retries will be delayed exponentially based on the initial interval.
	// Example: with InitialInterval=1s, first retry is 1s, second, is 2s, third is 4s, etc.
	// +optional
	InitialInterval string `json:"initialInterval,omitempty"`

	// MaxInterval is the maximum delay between retries.
	// Once the max interval is reached by exponentionally backing off
	// all subsequent retries delay for the max interval.
	// +optional
	MaxInterval string `json:"maxInterval,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RetryTimeoutConfigList satisfies K8s code gen requirements.
type RetryTimeoutConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []RetryTimeoutConfig `json:"items"`
}
