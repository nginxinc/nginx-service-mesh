// Package v1alpha1 defines and implements Nginx Service Mesh APIs
package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/nginxinc/nginx-service-mesh/pkg/apis/specs"
)

// SchemeGroupVersion is the identifier for the API which includes
// the name of the group and the version of the API.
var SchemeGroupVersion = schema.GroupVersion{
	Group:   specs.GroupName,
	Version: "v1alpha1",
}

// Kind takes an unqualified kind and returns back a Group qualified GroupKind.
func Kind(kind string) schema.GroupKind {
	return SchemeGroupVersion.WithKind(kind).GroupKind()
}

// Resource takes an unqualified resource and returns a Group qualified GroupResource.
func Resource(resource string) schema.GroupResource {
	return SchemeGroupVersion.WithResource(resource).GroupResource()
}

var (
	// SchemeBuilder collects functions that add things to a scheme. It's to allow
	// code to compile without explicitly referencing generated types. You should
	// declare one in each package that will have generated deep copy or conversion
	// functions.
	SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)

	// AddToScheme applies all the stored functions to the scheme. A non-nil error
	// indicates that one function failed and the attempt was abandoned.
	AddToScheme = SchemeBuilder.AddToScheme
)

// Adds the list of known types to Scheme.
func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(SchemeGroupVersion,
		&CircuitBreaker{},
		&CircuitBreakerList{},
		&RateLimit{},
		&RateLimitList{},
	)
	metav1.AddToGroupVersion(scheme, SchemeGroupVersion)

	return nil
}
