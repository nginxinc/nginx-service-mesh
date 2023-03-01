// Package commands contains all of the cli commands
package commands // import "github.com/nginxinc/nginx-service-mesh/internal/nginx-meshctl/commands"

import (
	"context"
	"fmt"
	"strings"

	v1 "k8s.io/api/core/v1"
	discoveryv1beta1 "k8s.io/api/discovery/v1beta1"
	"k8s.io/utils/strings/slices"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/spf13/cobra"

	"github.com/nginxinc/nginx-service-mesh/pkg/apis/mesh"
)

const longServices = `List the Services registered with NGINX Service Mesh.
- Outputs the Services and their upstream addresses and ports.
- The list contains only those Services whose Pods contain the NGINX Service Mesh sidecar.
`

type serviceDetails struct {
	name      string
	namespace string
	ports     []v1.ServicePort
	addresses []string
}

// GetServices prints the list of services registered with NGINX Service Mesh.
func GetServices() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "services",
		Short: "List the Services registered with NGINX Service Mesh",
		Long:  longServices,
	}

	cmd.PersistentPreRunE = defaultPreRunFunc()
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		var meshServices []serviceDetails
		k8sClient, err := client.New(initK8sClient.Config(), client.Options{})
		if err != nil {
			fmt.Printf("failed to initialize k8s client: %v\n", err)
			return err
		}
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		services := &v1.ServiceList{}
		if err := k8sClient.List(ctx, services); err != nil {
			fmt.Printf("error getting list of services: %v\n", err)
			return err
		}

		for _, serviceObj := range services.Items {
			serviceObj := serviceObj
			// if we consider a service injectable then we consider it registered with our mesh
			if injectable, err := isNamespaceInjectionEnabled(ctx, k8sClient, serviceObj.Namespace); err == nil && injectable {
				upstreams, epErr := getEndpoints(ctx, k8sClient, serviceObj)
				if err != nil {
					return epErr
				}
				newDetails := serviceDetails{
					name:      serviceObj.Name,
					namespace: serviceObj.Namespace,
					ports:     serviceObj.Spec.Ports,
					addresses: upstreams,
				}
				meshServices = append(meshServices, newDetails)
			} else if err != nil {
				return err
			}
		}

		tabWriter := TabWriterWithOpts()
		fmt.Fprintln(tabWriter, "Service\tUpstream\tPort")
		for _, svc := range meshServices {
			serviceLine := fmt.Sprintf("%s/%s\t", svc.namespace, svc.name)

			// create list of ports
			ports := make([]string, 0, len(svc.ports))
			for _, port := range svc.ports {
				ports = append(ports, fmt.Sprintf("%v", port.Port))
			}
			if len(svc.ports) == 0 {
				ports = append(ports, "<none>")
			}

			// create list of address and port combos
			for _, addr := range svc.addresses {
				serviceLine += fmt.Sprintf("%v\t%s\n\t", addr, strings.Join(ports, ","))
			}
			if len(svc.addresses) == 0 {
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

// getEndpoints returns a slice of upstream addresses for a service.
func getEndpoints(ctx context.Context, k8sClient client.Client, svc v1.Service) ([]string, error) {
	endpointSlices := &discoveryv1beta1.EndpointSliceList{}
	var upstreamAdresses []string
	opt := client.MatchingLabels{"kubernetes.io/service-name": svc.Name}
	if err := k8sClient.List(ctx, endpointSlices, opt); err != nil {
		fmt.Printf("error getting list of endpoint slices for service: %v\n", err)
		return nil, err
	}
	for _, epSlice := range endpointSlices.Items {
		epSlice := epSlice
		if epSlice.Namespace == svc.Namespace {
			for _, endpoint := range epSlice.Endpoints {
				upstreamAdresses = append(upstreamAdresses, endpoint.Addresses...)
			}
			return upstreamAdresses, nil
		}
	}
	// in the case that a service has no upstreams yet but is in a namespace where injection would be enabled
	// return nothing rather than an error since that service is still 'part' of the mesh.
	return nil, nil
}

// isNamespaceInjectionEnabled returns whether a given namespace has injection enabled.
func isNamespaceInjectionEnabled(ctx context.Context, k8sClient client.Client, ns string) (bool, error) {
	meshClient, err := mesh.NewMeshClient(initK8sClient.Config(), meshTimeout)
	if err != nil {
		return false, fmt.Errorf("failed to get mesh client: %w", err)
	}
	meshConfig, err := GetMeshConfig(meshClient)
	if err != nil {
		return false, fmt.Errorf("unable to get mesh config: %w", err)
	}
	nsObj := &v1.Namespace{}
	if err := k8sClient.Get(ctx, client.ObjectKey{
		Namespace: "",
		Name:      ns,
	}, nsObj); err != nil {
		fmt.Printf("error getting namespace: %v\n", err)
		return false, err
	}
	return slices.Contains(*meshConfig.EnabledNamespaces, ns) ||
		nsObj.GetLabels()[mesh.AutoInjectLabel] == mesh.AutoInjectionEnabled ||
		(*meshConfig.IsAutoInjectEnabled && !(mesh.IgnoredNamespaces[ns])), nil
}
