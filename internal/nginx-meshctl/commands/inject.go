// Package commands contains all of the cli commands
package commands // import "github.com/nginxinc/nginx-service-mesh/internal/nginx-meshctl/commands"

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/nginxinc/nginx-service-mesh/pkg/apis/mesh"
	"github.com/nginxinc/nginx-service-mesh/pkg/inject"
)

const (
	longInject = `Inject the NGINX Service Mesh sidecar into Kubernetes resources.
- Accepts JSON and YAML formats.
- Outputs JSON or YAML resources with injected sidecars to stdout.`

	exampleInject = `
  - Inject the resources in my-app.yaml and create in Kubernetes:

      nginx-meshctl inject -f ./my-app.yaml | kubectl apply -f -

  - Inject the resources passed into stdin and write the changes to the same file:

      nginx-meshctl inject < ./my-app.json > ./my-injected-app.json

  - Inject the resources in my-app.yaml and configure proxies to ignore ports 1433 and 1434 for outgoing traffic:

      nginx-meshctl inject --ignore-outgoing-ports 1433,1434 -f ./my-app.yaml

  - Inject the resources passed into stdin and configure proxies to ignore port 1433 for incoming traffic:

      nginx-meshctl inject --ignore-incoming-ports 1433 < ./my-app.json 
`
	genericInjectErrorInfo = "Cannot inject NGINX Service Mesh sidecar."
)

// Inject injects the sidecar proxy containers into a deployment yaml.
func Inject() *cobra.Command {
	var filename string
	var ignoreIncoming []int
	var ignoreOutgoing []int
	cmd := &cobra.Command{
		Use:     "inject",
		Short:   "Inject the NGINX Service Mesh sidecars into Kubernetes resources",
		Long:    longInject,
		Example: exampleInject,
	}
	cmd.Flags().StringVarP(
		&filename,
		"file",
		"f",
		"",
		`the filename that contains the resources you want to inject
		If no filename is provided, input will be taken from stdin`)
	cmd.Flags().IntSliceVar(
		&ignoreIncoming,
		"ignore-incoming-ports",
		[]int{},
		`ports to ignore for incoming traffic`)
	cmd.Flags().IntSliceVar(
		&ignoreOutgoing,
		"ignore-outgoing-ports",
		[]int{},
		`ports to ignore for outgoing traffic`)

	cmd.PersistentPreRunE = defaultPreRunFunc()
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		var input []byte
		var err error
		if filename != "" {
			input, err = readFileOrURL(filename)
			if err != nil {
				_, _ = fmt.Fprintln(os.Stderr, genericInjectErrorInfo)

				return fmt.Errorf("error reading input file: %w", err)
			}
		} else {
			var tmpFile *os.File
			input, tmpFile, err = createFileFromSTDIN()
			if err != nil {
				_, _ = fmt.Fprintln(os.Stderr, genericInjectErrorInfo)

				return fmt.Errorf("error reading input from stdin: %w", err)
			}
			filename = tmpFile.Name()
			defer func() {
				closeErr := tmpFile.Close()
				if closeErr != nil {
					fmt.Println(closeErr)
				}
			}()
		}

		ignPorts := inject.IgnorePorts{
			Incoming: ignoreIncoming,
			Outgoing: ignoreOutgoing,
		}
		if portErr := ignPorts.Validate(); portErr != nil {
			return fmt.Errorf("invalid ignore ports: %w", portErr)
		}

		injectConfig := inject.Inject{
			Resources:   input,
			IgnorePorts: ignPorts,
		}

		meshClient, err := mesh.NewMeshClient(initK8sClient.Config(), meshTimeout)
		if err != nil {
			return fmt.Errorf("failed to get mesh client: %w", err)
		}
		meshConfig, err := GetMeshConfig(meshClient)
		if err != nil {
			return fmt.Errorf("unable to get mesh config: %w", err)
		}

		res, err := inject.IntoFile(injectConfig, *meshConfig)
		if err != nil {
			return fmt.Errorf("error injecting sidecar: %w", err)
		}
		fmt.Print(res)

		return nil
	}

	return cmd
}

var errFileDoesNotExist = errors.New("files does not exist")

// readFileOrURL returns the body from a local or remote file.
func readFileOrURL(filename string) ([]byte, error) {
	switch {
	case filename != "" && !strings.HasPrefix(filename, "http"):
		// Local file
		if exists := fileExists(filename); !exists {
			return nil, fmt.Errorf("%w: %s", errFileDoesNotExist, filename)
		}
		body, err := os.ReadFile(filepath.Clean(filename))
		if err != nil {
			return nil, fmt.Errorf("error reading file: %w", err)
		}

		return body, nil
	case strings.HasPrefix(filename, "http"):
		// Remote file
		fileClient := &http.Client{Timeout: time.Minute}
		req, err := http.NewRequestWithContext(context.TODO(), http.MethodGet, filename, http.NoBody)
		if err != nil {
			return nil, fmt.Errorf("error creating http request: %w", err)
		}
		resp, err := fileClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("error getting file: %w", err)
		}
		// this get around the errcheck lint when we really do not care
		// 'Error return value of `resp.Body.Close` is not checked (errcheck)'
		defer func() {
			_ = resp.Body.Close()
		}()

		err = checkResponse(resp)
		if err != nil {
			return nil, err
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("error reading file contents: %w", err)
		}

		return body, nil
	}

	return nil, nil
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}

	return !info.IsDir()
}

var errResponse = errors.New("http request failed")

func checkResponse(resp *http.Response) error {
	if c := resp.StatusCode; c >= 200 && c <= 299 {
		return nil
	}
	req := resp.Request

	return fmt.Errorf("%w: %v %v failed: %v", errResponse, req.Method, req.URL.String(), resp.Status)
}

// createFileFromSTDIN creates a temporary file from the data contained in stdin and returns the data and file pointer.
func createFileFromSTDIN() ([]byte, *os.File, error) {
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return nil, nil, fmt.Errorf("error reading file contents: %w", err)
	}

	f, err := os.CreateTemp("", "nginx-mesh-api-temp-file.txt")
	if err != nil {
		return nil, nil, fmt.Errorf("error creating temporary file: %w", err)
	}

	return data, f, nil
}
