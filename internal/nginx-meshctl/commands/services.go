// Package commands contains all of the cli commands
package commands // import "github.com/nginxinc/nginx-service-mesh/internal/nginx-meshctl/commands"

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/spf13/cobra"

	"github.com/nginxinc/nginx-service-mesh/pkg/apis/mesh"
)

const longServices = `List the Services registered with NGINX Service Mesh.
- Outputs the Services and their upstream addresses and ports.
- The list contains only those Services whose Pods contain the NGINX Service Mesh sidecar.
`

const genericGetServicesErrorInfo = "Cannot get Services from NGINX Service Mesh API Server."

// GetServices prints the service list fetched from the Control Plane.
func GetServices() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "services",
		Short: "List the Services registered with NGINX Service Mesh",
		Long:  longServices,
	}

	cmd.PersistentPreRunE = defaultPreRunFunc()
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		meshClient, err := mesh.NewMeshClient(initK8sClient.Config(), meshTimeout)
		if err != nil {
			fmt.Println(genericGetServicesErrorInfo)

			return err
		}
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		res, err := meshClient.GetServices(ctx)
		if err != nil {
			fmt.Println(genericGetServicesErrorInfo)

			return err
		}
		defer func() {
			closeErr := res.Body.Close()
			if closeErr != nil {
				fmt.Println(closeErr)
			}
		}()

		if res.StatusCode != http.StatusOK {
			fmt.Println(genericGetServicesErrorInfo)

			return mesh.ParseAPIError(res)
		}

		data, err := mesh.ParseGetServicesResponse(res)
		if err != nil {
			fmt.Println(genericGetServicesErrorInfo)

			return fmt.Errorf("error parsing response body from NGINX Service Mesh API Server: %w", err)
		}

		tabWriter := TabWriterWithOpts()
		fmt.Fprintln(tabWriter, "Service\tUpstream\tPort")
		for _, svc := range *data.JSON200 {
			serviceLine := fmt.Sprintf("%s/%s\t", *svc.Namespace, svc.Name)

			// create list of ports
			ports := make([]string, 0, len(svc.Ports))
			for _, port := range svc.Ports {
				ports = append(ports, fmt.Sprintf("%v", port.Port))
			}
			if len(svc.Ports) == 0 {
				ports = append(ports, "<none>")
			}

			// create list of address and port combos
			for _, addr := range svc.Addresses {
				serviceLine += fmt.Sprintf("%v\t%s\n\t", addr, strings.Join(ports, ","))
			}
			if len(svc.Addresses) == 0 {
				serviceLine += fmt.Sprintf("<none>\t%s\n\t", strings.Join(ports, ","))
			}

			// print everything out
			fmt.Fprint(tabWriter, serviceLine)
			fmt.Fprint(tabWriter, "\t\t\n")
		}

		return tabWriter.Flush()
	}

	return cmd
}
