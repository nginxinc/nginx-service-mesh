package commands

import (
	"context"
	"fmt"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"helm.sh/helm/v3/pkg/action"
	helmRelease "helm.sh/helm/v3/pkg/release"
	v1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/nginxinc/nginx-service-mesh/pkg/apis/mesh"
	meshErrors "github.com/nginxinc/nginx-service-mesh/pkg/errors"
	"github.com/nginxinc/nginx-service-mesh/pkg/k8s"
	"github.com/nginxinc/nginx-service-mesh/pkg/pod"
)

const longRemove = `Remove the NGINX Service Mesh from your Kubernetes cluster.
- Removes the resources created by the deploy command from the Service Mesh namespace (default: "nginx-mesh").
- You will need to clean up all resources containing injected sidecar proxies manually.
`

const exampleRemove = `
  - Remove the NGINX Service Mesh from the default namespace ('nginx-mesh'):
		
      nginx-meshctl remove
	
  - Remove the NGINX Service Mesh from namespace 'my-namespace':
		
      nginx-meshctl remove --namespace my-namespace
		
  - Remove the NGINX Service Mesh without prompting the user to confirm removal:
	
      nginx-meshctl remove -y
`

var yes bool

// proxiedResources is a map of {"namespace": {"resource type": ["resource names"]}},
// used by the CLI to print out proxied resources when removing the mesh.
type proxiedResources map[string]map[string][]string

// Remove removes the service mesh control plane.
func Remove() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "remove",
		Short:   "Remove NGINX Service Mesh from your Kubernetes cluster",
		Long:    longRemove,
		Example: exampleRemove,
	}
	cmd.Flags().BoolVarP(
		&yes,
		"yes",
		"y",
		false,
		"answer yes for confirmation of removal",
	)
	var environment string
	cmd.Flags().StringVar(
		&environment,
		"environment",
		"",
		`environment the mesh is deployed in
		Valid values: `+formatValues(mesh.Environments),
	)
	err := cmd.Flags().MarkDeprecated("environment", "and will be removed in a future release.")
	if err != nil {
		fmt.Println("error marking flag as deprecated: ", err)
	}

	cmd.PersistentPreRunE = defaultPreRunFunc()
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		namespace := initK8sClient.Namespace()

		// Verify mesh exists
		releaseName, verifyErr := verifyMeshInstall(initK8sClient)
		if verifyErr != nil {
			if strings.Contains(verifyErr.Error(), "not found") {
				// namespace or release weren't found, but some resources may still remain
				msg := fmt.Sprintf("%v, however some mesh resources such as CustomResourceDefinitions may exist. "+
					"These will be deleted with your permission.", verifyErr)
				if err := ReadYes(msg); err != nil {
					return err
				}
				deleteNamespace := true
				var removed bool
				var err error
				if removed, err = postHelmCleanup(initK8sClient, deleteNamespace); err != nil {
					fmt.Printf("error cleaning up lingering resources: %v\n", err)
				}

				if removed {
					fmt.Println("Cleaned up lingering NGINX Service Mesh CRDs from previous installation.")
				}
			}

			return verifyErr
		}

		if !yes {
			msg := fmt.Sprintf("Preparing to remove NGINX Service Mesh from namespace \"%s\". "+
				"This will make all sidecar proxies transparent.", namespace)
			if err := ReadYes(msg); err != nil {
				return err
			}
		}

		var resources proxiedResources
		// The package functions that are called expect the controller-runtime client.
		// Ideally we can update our entire CLI to use this client instead of the current mismatched clients.
		k8sClient, err := client.New(initK8sClient.Config(), client.Options{})
		if err != nil {
			fmt.Printf("failed to initialize k8s client: %v\n", err)
		} else {
			resources, err = getProxiedResources(k8sClient)
			if err != nil {
				fmt.Println(err.Error())
				fmt.Println("To ensure minimal traffic disruption, re-roll resources using 'kubectl rollout restart <resource>/<name>'.")
			}
		}

		fmt.Printf("Removing NGINX Service Mesh from namespace \"%s\"...\n", namespace)

		csiDriverRunning := csiDriverRunning(initK8sClient, namespace)
		deleteNamespace := !(csiDriverRunning && len(resources) > 0)

		err = newRemover(initK8sClient).remove(releaseName, deleteNamespace)
		if err == nil {
			tabWriter := TabWriterWithOpts()
			if !deleteNamespace {
				fmt.Fprintln(tabWriter, "NGINX Service Mesh partially removed.")
			} else {
				fmt.Fprintln(tabWriter, "NGINX Service Mesh removed successfully.")
			}
			err = printResources(tabWriter, resources)
		}

		if !deleteNamespace {
			fmt.Println("\nSome NGINX Service Mesh components have been left around to ensure proper " +
				"cleanup of injected Pods.")
			fmt.Printf("Once injected Pods are either re-rolled or deleted, all remaining NGINX Service Mesh "+
				"components will be cleaned up automatically by the 'csi-driver-sentinel' Job in the %s namespace.\n\n", namespace)
			fmt.Printf("Once complete, the %[1]s namespace will need to be manually cleaned up by running:\n\n\t"+
				"kubectl delete namespace %[1]s\n", namespace)
			fmt.Println("\nFor more details, or if the 'csi-driver-sentinel' failed to deploy, " +
				"see https://docs.nginx.com/nginx-service-mesh/get-started/openshift-platform/considerations/#remove")
			fmt.Println()
		}

		return err
	}

	return cmd
}

func csiDriverRunning(k8sClient k8s.Client, namespace string) bool {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_, err := k8sClient.ClientSet().AppsV1().DaemonSets(namespace).Get(ctx, "spiffe-csi-driver", metav1.GetOptions{})

	return err == nil
}

type remover struct {
	k8sClient k8s.Client
}

// newRemover returns a new remover object.
func newRemover(k8sClient k8s.Client) *remover {
	return &remover{
		k8sClient: k8sClient,
	}
}

// remove initializes a helm action and calls helm uninstall, followed by further cleanup.
func (r *remover) remove(releaseName string, deleteNamespace bool) error {
	actionConfig, err := r.k8sClient.HelmAction(r.k8sClient.Namespace())
	if err != nil {
		return fmt.Errorf("error initializing helm action: %w", err)
	}

	remove := action.NewUninstall(actionConfig)
	remove.Wait = true
	if _, err := remove.Run(releaseName); err != nil {
		return fmt.Errorf("error removing NGINX Service Mesh: %w", err)
	}

	if _, err := postHelmCleanup(r.k8sClient, deleteNamespace); err != nil {
		return err
	}

	return nil
}

// postHelmCleanup performs the steps that helm doesn't do:
// - removes all mesh CRDs
// - removes the mesh namespace.
func postHelmCleanup(k8sClient k8s.Client, deleteNamespace bool) (bool, error) {
	namespace := k8sClient.Namespace()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var removed bool
	var err error
	if removed, err = removeCRDs(ctx, k8sClient); err != nil {
		return false, fmt.Errorf("error removing CRDs: %w", err)
	}

	if deleteNamespace {
		if err := k8sClient.ClientSet().CoreV1().Namespaces().
			Delete(ctx, namespace, metav1.DeleteOptions{}); err != nil && !k8sErrors.IsNotFound(err) {
			return removed, fmt.Errorf("error removing %s namespace: %w", namespace, err)
		}
	}

	return removed, nil
}

// removes all custom CRDs.
func removeCRDs(ctx context.Context, k8sClient k8s.Client) (bool, error) {
	crdClient := k8sClient.APIExtensionClientSet().ApiextensionsV1().CustomResourceDefinitions()
	crds, err := crdClient.List(ctx, metav1.ListOptions{LabelSelector: "app.kubernetes.io/part-of=nginx-service-mesh"})
	if err != nil {
		return false, fmt.Errorf("error listing NGINX Service Mesh CRDs: %w", err)
	}

	for _, crd := range crds.Items {
		if err := crdClient.Delete(ctx, crd.Name, metav1.DeleteOptions{}); err != nil {
			return false, fmt.Errorf("error deleting CRD '%s': %w", crd.Name, err)
		}
	}

	return len(crds.Items) > 0, nil
}

// getProxiedResources gets a list of resources that are proxied.
func getProxiedResources(k8sClient client.Client) (proxiedResources, error) {
	resources := make(proxiedResources)
	seen := make(map[string]struct{})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pods := &v1.PodList{}
	if err := k8sClient.List(ctx, pods); err != nil {
		return nil, fmt.Errorf("error getting list of pods: %w", err)
	}

	for _, podObj := range pods.Items {
		podObj := podObj
		if !pod.IsInjected(&podObj) {
			continue
		}

		ownerType, ownerName, err := pod.GetOwner(ctx, k8sClient, &podObj)
		if err != nil {
			fmt.Printf("error getting pod owner: %v\n", err)

			continue
		}

		if _, ok := seen[podObj.Namespace+ownerType+ownerName]; !ok {
			if resources[podObj.Namespace] == nil {
				resources[podObj.Namespace] = make(map[string][]string)
			}
			resources[podObj.Namespace][ownerType] = append(resources[podObj.Namespace][ownerType], ownerName)
			seen[podObj.Namespace+ownerType+ownerName] = struct{}{}
		}
	}

	return resources, nil
}

func printResources(writer *tabwriter.Writer, resources proxiedResources) error {
	if len(resources) > 0 {
		fmt.Fprintln(writer, "NOTE: The following resources still contain the sidecar proxy:")
		fmt.Fprintf(writer, "\n\tNAMESPACE\tRESOURCE\n")
		for ns, values := range resources {
			for rsType, names := range values {
				for _, name := range names {
					fmt.Fprintf(writer, "\t%s\t%s/%s\n", ns, rsType, name)
				}
			}
		}
		fmt.Fprintln(writer, "\nIf your resource supports rolling updates, re-roll using 'kubectl rollout restart <resource>/<name>'. "+
			"Otherwise, the Pods need to be deleted and re-created.")
	}

	return writer.Flush()
}

// verifyMeshInstall verifies that the install namespace exists and that a helm release of NSM currently exists.
func verifyMeshInstall(k8sClient k8s.Client) (string, error) {
	namespace := k8sClient.Namespace()
	_, err := k8sClient.ClientSet().CoreV1().Namespaces().Get(context.TODO(), namespace, metav1.GetOptions{})
	if err != nil && k8sErrors.IsNotFound(err) {
		return "", meshErrors.NamespaceNotFoundError{Namespace: namespace}
	}

	actionConfig, err := k8sClient.HelmAction(namespace)
	if err != nil {
		return "", fmt.Errorf("initializing helm action config failed: %w", err)
	}

	lister := action.NewList(actionConfig)
	releases, err := lister.Run()
	if err != nil {
		return "", fmt.Errorf("failed to list currently installed releases: %w", err)
	}
	var foundRelease *helmRelease.Release

	for _, release := range releases {
		if release.Chart != nil {
			if strings.Contains(release.Chart.Name(), "nginx-service-mesh") {
				foundRelease = release

				break
			}
		}
	}
	if err = checkReleaseStatus(foundRelease, namespace); err != nil {
		return "", err
	}

	return foundRelease.Name, nil
}

// checkReleaseStatus checks a passed release's status and errors if it doesn't support removal, or the passed release is nil.
func checkReleaseStatus(release *helmRelease.Release, namespace string) error {
	switch {
	case release == nil:
		//nolint:goerr113 // no reason to make this a package level static error as it won't be reused
		return fmt.Errorf("NGINX Service Mesh installation not found in namespace '%s'", namespace)
	case release.Info.Status == "uninstalled" || release.Info.Status == "failed" || release.Info.Status == "uninstalling":
		//nolint:goerr113 // no reason to make this a package level static error as it won't be reused
		return fmt.Errorf("the current status %s of the release %s in the namespace %s does not support removal",
			release.Info.Status, release.Name, release.Namespace)
	}

	return nil
}
