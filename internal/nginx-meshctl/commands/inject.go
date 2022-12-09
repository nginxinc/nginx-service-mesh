// Package commands contains all of the cli commands
package commands // import "github.com/nginxinc/nginx-service-mesh/internal/nginx-meshctl/commands"

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/nginxinc/nginx-service-mesh/internal/nginx-meshctl/inject"
	"github.com/nginxinc/nginx-service-mesh/pkg/apis/mesh"
	"github.com/nginxinc/nginx-service-mesh/pkg/k8s"
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
			input, err = inject.ReadFileOrURL(filename)
			if err != nil {
				_, _ = fmt.Fprintln(os.Stderr, genericInjectErrorInfo)

				return fmt.Errorf("error reading input file: %w", err)
			}
		} else {
			var tmpFile *os.File
			input, tmpFile, err = inject.CreateFileFromSTDIN()
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
		err = validatePorts(ignoreIncoming)
		if err != nil {
			_, _ = fmt.Fprintln(os.Stderr, genericInjectErrorInfo)

			return fmt.Errorf("ignore incoming ports value is not valid: %w", err)
		}
		err = validatePorts(ignoreOutgoing)
		if err != nil {
			_, _ = fmt.Fprintln(os.Stderr, genericInjectErrorInfo)

			return fmt.Errorf("ignore outgoing ports value is not valid: %w", err)
		}

		err = injectProxy(initK8sClient, input, filename, valueMap(ignoreIncoming, ignoreOutgoing))
		if err != nil && strings.Contains(err.Error(), "Client.Timeout") {
			_, _ = fmt.Fprintln(os.Stderr, "Try increasing the timeout using the '--timeout' flag.")
		}

		return err
	}

	return cmd
}

func valueMap(incoming, outgoing []int) map[string][]int {
	return map[string][]int{
		mesh.IgnoreIncomingPortsField: incoming,
		mesh.IgnoreOutgoingPortsField: outgoing,
	}
}

var errPortNotValid = errors.New("port is not valid")

// validatePort checks that the port is within the range 1-65535.
func validatePorts(ports []int) error {
	for _, p := range ports {
		if p <= 0 || p > 65535 {
			return fmt.Errorf("%w: %d", errPortNotValid, p)
		}
	}

	return nil
}

func injectProxy(k8sClient k8s.Client, input []byte, filename string, values map[string][]int) error {
	meshClient, err := mesh.NewMeshClient(k8sClient.Config(), meshTimeout)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, genericInjectErrorInfo)

		return err
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	body, contentType, err := inject.BuildMultipartRequestBody(values, input, filename)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, genericInjectErrorInfo)

		return err
	}
	res, err := meshClient.InjectSidecarProxyWithBody(ctx, contentType, body)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, genericInjectErrorInfo)

		return fmt.Errorf("error injecting sidecar proxy with body: %w", err)
	}
	defer func() {
		closeErr := res.Body.Close()
		if closeErr != nil {
			fmt.Println(closeErr)
		}
	}()

	if res.StatusCode != http.StatusOK {
		_, _ = fmt.Fprintln(os.Stderr, genericInjectErrorInfo)

		return mesh.ParseAPIError(res)
	}

	data, err := mesh.ParseInjectSidecarProxyResponse(res)
	if err != nil {
		return fmt.Errorf("error parsing response body from Service Mesh API Server: %w", err)
	}
	fmt.Print(string(data.Body))

	return nil
}
