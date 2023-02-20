package v1alpha2

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// MeshConfigClass defines a class of MeshConfigs to the user for creating MeshConfig resources.
type MeshConfigClass struct { //nolint:govet // fieldalignment not desired
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the desired state of MeshConfigClass.
	Spec MeshConfigClassSpec `json:"spec"`

	// Status defines the current state of MeshConfigClass.
	// +optional
	Status MeshConfigClassStatus `json:"status"`
}

// MeshConfigClassSpec defines the desired state of MeshConfigClass.
type MeshConfigClassSpec struct {
	// ControllerName is the name of the controller that is managing MeshConfigs of this class.
	ControllerName string `json:"controllerName"`
}

// MeshConfigClassStatus defines the current state of MeshConfigClass.
type MeshConfigClassStatus struct{}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// MeshConfigClassList is a list of MeshConfigClass resources.
type MeshConfigClassList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []MeshConfigClass `json:"items"`
}
