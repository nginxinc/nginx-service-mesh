// Package errors contains error messages and error types for the Service Mesh
package errors

import (
	"errors"
	"fmt"
	"strings"

	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
)

// ErrInput is returned when CLI arguments are invalid.
var ErrInput = errors.New("error validating input")

// ErrCheckingExistence is returned when the mesh already exists.
var ErrCheckingExistence = errors.New("error looking for existing mesh")

// ErrMeshStatus is returned when the mesh controller is not ready.
var ErrMeshStatus = errors.New("mesh controller unavailable")

// ErrNotFound is returned when a 404 is returned by mesh API or mesh API is not running.
var ErrNotFound = errors.New("not found")

// ErrMeshControllerResponse is returned when the mesh controller returns an error.
var ErrMeshControllerResponse = errors.New("mesh Controller returned error")

// ErrK8sAPIResponse is returned when the k8s API returns an error.
var ErrK8sAPIResponse = errors.New("kubernetes API returned error")

// ErrDeployingMesh is returned when the mesh fails to deploy.
var ErrDeployingMesh = errors.New("failed to deploy NGINX Service Mesh")

// AlreadyExistsError is returned when the mesh already exists.
type AlreadyExistsError struct {
	Msg string
}

func (e AlreadyExistsError) Error() string {
	if e.Msg != "" {
		return fmt.Sprintf("mesh already exists: %s", e.Msg)
	}

	return "mesh already exists"
}

// TimedOutDeletingError error for when at least one resource fails to delete within a time window.
type TimedOutDeletingError struct{}

func (TimedOutDeletingError) Error() string {
	return "timed out while waiting for resources to delete"
}

// DeleteFailedError error for when at least one resource could not be deleted, e.g. delete call returns an error.
type DeleteFailedError struct{}

func (DeleteFailedError) Error() string {
	return "failed to delete all resources"
}

// UnexpectedStatusError is for when the NIGNX Service Mesh API returns an unexpected status code.
type UnexpectedStatusError struct {
	Code int
}

func (e UnexpectedStatusError) Error() string {
	return fmt.Sprintf("NGINX Service Mesh returned an unexpected status code: %d", e.Code)
}

// InternalServiceError is for when the NIGNX Service Mesh API returns an internal error.
type InternalServiceError struct {
	Msg string
}

func (e InternalServiceError) Error() string {
	return fmt.Sprintf("NGINX Service Mesh returned an internal server error: %s", e.Msg)
}

// InvalidRequestError is for when the NGINX Service Mesh API returns a 400.
type InvalidRequestError struct {
	Msg string
}

func (e InvalidRequestError) Error() string {
	return fmt.Sprintf("invalid request: %s", e.Msg)
}

// NamespaceNotFoundError is for when the NGINX Service Mesh namespace is not found.
type NamespaceNotFoundError struct {
	Namespace string
}

func (e NamespaceNotFoundError) Error() string {
	return fmt.Sprintf("namespace '%s' not found", e.Namespace)
}

// NamespaceExistsError returns instructions if the error is due to the namespace already existing.
func NamespaceExistsError(err error) string {
	if strings.Contains(err.Error(), "unable to create namespace") && k8sErrors.IsAlreadyExists(err) {
		return "NGINX Service Mesh needs to create its own dedicated namespace. Delete the namespace, or specify " +
			"a different namespace via the '--namespace' argument."
	}

	return ""
}

// ImagePullError is returned when k8s fails to pull images.
type ImagePullError struct {
	Msg string
}

func (e ImagePullError) Error() string {
	return fmt.Sprintf("could not pull images: %s", e.Msg)
}

// CheckForK8sFatalError checks if k8s error is fatal and returns instruction message.
// Fatal in this case means that the error will keep happening until the user addresses the issue.
func CheckForK8sFatalError(err error) (bool, string) {
	var isFatal bool
	var instructions string
	var statusError k8sErrors.APIStatus
	if errors.As(err, &statusError) {
		if k8sErrors.IsUnauthorized(err) {
			isFatal = true
			instructions = "Check credentials and retry command."
		}
		if k8sErrors.IsForbidden(err) {
			instructions = "Check configuration of your Kubernetes API Server."
		}
		if k8sErrors.IsServerTimeout(err) {
			isFatal = true
			instructions = "Retry command."
		}
		if k8sErrors.IsAlreadyExists(err) {
			isFatal = true
			instructions = "Run remove command to delete any lingering NGINX Service Mesh resources, then deploy again."
		}
		if k8sErrors.IsTooManyRequests(err) {
			isFatal = true
			instructions = "Retry command at a later time."
		}
		if k8sErrors.IsInternalError(err) {
			instructions = "Internal Kubernetes error."
		}
		if k8sErrors.IsServiceUnavailable(err) {
			isFatal = true
			instructions = "Retry command."
		}
		if k8sErrors.ReasonForError(err) == "" {
			isFatal = true
			instructions = "Make sure that your cluster is healthy and running before retrying command."
		}
	} else {
		isFatal = false
		instructions = ""
	}

	return isFatal, instructions
}
