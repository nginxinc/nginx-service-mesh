// Package commands contains all of the cli commands
package commands // import "github.com/nginxinc/nginx-service-mesh/internal/nginx-meshctl/commands"

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	v1 "k8s.io/api/core/v1"
	aggregator "k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset"

	"github.com/nginxinc/nginx-service-mesh/internal/nginx-meshctl/top"
)

const (
	longTop = `Display traffic statistics.
Top provides information about the incoming and outgoing requests to and from a resource type or name.
Supported resource types are: Pods, Deployments, StatefulSets, DaemonSets, and Namespaces.`
	exampleTop = `
  - Display traffic statistics for all Deployments:
		
      nginx-meshctl top
  
  - Display traffic statistics for all Pods:
		
      nginx-meshctl top pods
  
  - Display traffic statistics for Deployment "my-app":
	 
      nginx-meshctl top deployments/my-app`

	supportedResources = `
- pods, pod, po
- deployments, deployment, deploy
- statefulsets, statefulset, ss
- daemonsets, daemonset, ds
- namespaces, namespace, ns
`
	genericTopErrorInfo = "Cannot build traffic statistics."
)

// Top builds top command for service-mesh-cli.
func Top() *cobra.Command {
	var namespace string
	cmd := &cobra.Command{
		Use:     "top [resource-type/resource]",
		Short:   "Display traffic statistics",
		Long:    longTop,
		Example: exampleTop,
		Args:    cobra.MaximumNArgs(1),
	}

	cmd.PersistentPreRunE = defaultPreRunFunc()
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		var err error
		tabWriter := TabWriterWithOpts()
		client, err := aggregator.NewForConfig(initK8sClient.Config())
		if err != nil {
			fmt.Println(genericTopErrorInfo)

			return fmt.Errorf("failed to get API Service client: %w", err)
		}
		ref := &top.MetricsMeta{
			Client: client,
			ObjectReference: v1.ObjectReference{
				Namespace: namespace,
			},
		}

		deployStr := "deployments"
		podStr := "pods"
		namespaceStr := "namespaces"
		statefulStr := "statefulsets"
		daemonStr := "daemonsets"

		if len(args) == 1 {
			argSlice := strings.Split(args[0], "/")
			if len(argSlice) == 1 {
				ref.Name = ""
			} else {
				ref.Name = argSlice[1]
			}
			switch argSlice[0] {
			case "deploy", "deployment", deployStr:
				ref.Kind = deployStr
				ref.DisplayName = "Deployment"
			case "po", "pod", podStr:
				ref.Kind = podStr
				ref.DisplayName = "Pod"
			case "ns", "namespace", namespaceStr:
				ref.Kind = namespaceStr
				ref.DisplayName = "Namespace"
			case "ss", "statefulset", statefulStr:
				ref.Kind = statefulStr
				ref.DisplayName = "StatefulSet"
			case "ds", "daemonset", daemonStr:
				ref.Kind = daemonStr
				ref.DisplayName = "DaemonSet"
			default:
				fmt.Println(genericTopErrorInfo)
				fmt.Print("You must provide one of the supported resource type names or aliases: ")
				fmt.Print(supportedResources)

				return fmt.Errorf("failed to build resource from name: %v", argSlice[0]) //nolint:goerr113 // one-off error
			}

			if err = top.BuildTopMetrics(tabWriter, ref); err != nil {
				fmt.Println(genericTopErrorInfo)

				return err
			}
		} else {
			var errCount int
			resources := make(map[string]string)
			resources[deployStr] = "Deployment"
			resources[statefulStr] = "StatefulSet"
			resources[daemonStr] = "DaemonSet"
			for k, v := range resources {
				ref.Kind = k
				ref.DisplayName = v
				if err = top.BuildTopMetrics(tabWriter, ref); err != nil {
					errCount++
				}
			}
			if errCount == len(resources) {
				fmt.Println(genericTopErrorInfo)

				return err
			}
		}

		return tabWriter.Flush()
	}

	cmd.Flags().StringVarP(&namespace, "namespace", "n", "default", "namespace where the resource(s) resides")

	return cmd
}
