package mesh

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/rest"

	meshErrors "github.com/nginxinc/nginx-service-mesh/pkg/errors"
)

const apiPath = "/apis/nsm.nginx.com/v1alpha1"

const checkIfServiceMeshIsRunningMsg = "Run 'nginx-meshctl status' to make sure that the NGINX Service Mesh is running.\n" +
	"Also, ensure that the correct namespace was specified (default 'nginx-mesh') and that you are authorized " +
	"to access services in your Kubernetes cluster."

// NewMeshClient returns a new Client object for communicating with the mesh API.
func NewMeshClient(config *rest.Config, timeout time.Duration) (*Client, error) {
	gv := schema.GroupVersion{Group: "", Version: "v1"}
	config.GroupVersion = &gv
	config.APIPath = apiPath
	config.Timeout = timeout
	config.NegotiatedSerializer = serializer.NewCodecFactory(runtime.NewScheme())

	restClient, err := rest.RESTClientFor(config)
	if err != nil {
		return nil, fmt.Errorf("error creating RESTClient: %w", err)
	}

	httpClient := &http.Client{
		Timeout: timeout,
	}

	if restClient.Client != nil {
		httpClient = restClient.Client
	}

	server := fmt.Sprintf("%s%s", config.Host, config.APIPath)

	mc, err := NewClient(server, WithHTTPClient(httpClient))
	if err != nil {
		return nil, fmt.Errorf("error building mesh client: %w", err)
	}

	return mc, nil
}

// ParseAPIError parses the error message from an HTTP response.
func ParseAPIError(res *http.Response) error {
	// handle 404 case before attempting to unmarshal
	if res.StatusCode == http.StatusNotFound {
		fmt.Println(checkIfServiceMeshIsRunningMsg)

		return meshErrors.ErrNotFound
	}

	bodyBytes, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("error reading response body: %w", err)
	}
	//  check for nsm server errors
	var errModel ErrorModel
	if err = json.Unmarshal(bodyBytes, &errModel); err == nil {
		if res.StatusCode == http.StatusBadRequest {
			return meshErrors.InvalidRequestError{Msg: errModel.Message}
		}
		if res.StatusCode == http.StatusInternalServerError {
			return meshErrors.InternalServiceError{Msg: errModel.Message}
		}

		return fmt.Errorf("%w: %s", meshErrors.ErrMeshControllerResponse, errModel.Message)
	}
	// then check for Kubernetes API errors
	var status metav1.Status
	if err = json.Unmarshal(bodyBytes, &status); err == nil {
		return fmt.Errorf("%w: %s", meshErrors.ErrK8sAPIResponse, status.Message)
	}

	// otherwise error is unexpected & unknown
	return meshErrors.UnexpectedStatusError{Code: res.StatusCode}
}
