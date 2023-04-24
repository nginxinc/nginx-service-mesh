// Package support contains the methods for building a support package.
package support

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"path/filepath"
	"text/template"

	access "github.com/servicemeshinterface/smi-controller-sdk/apis/access/v1alpha2"
	specs "github.com/servicemeshinterface/smi-controller-sdk/apis/specs/v1alpha3"
	split "github.com/servicemeshinterface/smi-controller-sdk/apis/split/v1alpha3"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/printers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/kubectl/pkg/describe"
	"sigs.k8s.io/yaml"

	"github.com/nginxinc/nginx-service-mesh/pkg/apis/mesh"
	nsmspecsv1alpha1 "github.com/nginxinc/nginx-service-mesh/pkg/apis/specs/v1alpha1"
	nsmspecsv1alpha2 "github.com/nginxinc/nginx-service-mesh/pkg/apis/specs/v1alpha2"
	"github.com/nginxinc/nginx-service-mesh/pkg/helm"
	"github.com/nginxinc/nginx-service-mesh/pkg/k8s"
	podHelper "github.com/nginxinc/nginx-service-mesh/pkg/pod"
)

const (
	meshLabelSelectorKey                = "app.kubernetes.io/part-of"
	meshLabelSelectorValue              = "nginx-service-mesh"
	podDescFile                         = "desc.txt"
	podYamlFile                         = "pod.yaml"
	podListFile                         = "pods.txt"
	deploymentsFile                     = "deployments.yaml"
	statefulSetsFile                    = "statefulsets.yaml"
	daemonSetsFile                      = "daemonsets.yaml"
	servicesFile                        = "services.yaml"
	configMapsFile                      = "configmaps.yaml"
	serviceAccountsFile                 = "serviceaccounts.yaml"
	secretsFile                         = "secrets.yaml"
	validatingWebhookConfigurationsFile = "validatingwebhookconfigurations.yaml"
	mutatingWebhookConfigurationsFile   = "mutatingwebhookconfigurations.yaml"
	clusterRolesFile                    = "clusterroles.yaml"
	clusterRoleBindingsFile             = "clusterrolebindings.yaml"
	eventListFile                       = "events.txt"
	crdsFile                            = "crds.yaml"
	apiServicesFile                     = "apiservices.yaml"
	trafficSplitsFile                   = "trafficsplits.yaml"
	trafficTargetsFile                  = "traffictargets.yaml"
	httpRouteGroupsFile                 = "httproutegroups.yaml"
	tcpRoutesFile                       = "tcproutes.yaml"
	rateLimitsFile                      = "ratelimits.yaml"
	circuitBreakersFile                 = "circuitbreakers.yaml"
)

// DataFetcher gets all data for the support package and writes it to corresponding files.
type DataFetcher struct {
	writer             FileWriter
	k8sClient          k8s.Client
	kubeconfig         string
	directory          string
	meshNamespace      string
	controlPlaneDir    string
	collectSidecarLogs bool
}

// NewDataFetcher returns a new DataFetcher.
func NewDataFetcher(
	k8sClient k8s.Client,
	writer FileWriter,
	kubeconfig,
	directory string,
	collectSidecarLogs bool,
) *DataFetcher {
	return &DataFetcher{
		writer:             writer,
		k8sClient:          k8sClient,
		kubeconfig:         kubeconfig,
		directory:          directory,
		meshNamespace:      k8sClient.Namespace(),
		collectSidecarLogs: collectSidecarLogs,
	}
}

// GatherAndWriteData pulls together all support package data and writes it to corresponding files.
func (df *DataFetcher) GatherAndWriteData() {
	df.writeControlPlaneInformation()
	df.writeMeshConfig()
	df.writeDeployConfig()
	if df.collectSidecarLogs {
		df.writeSidecarInformation()
	}
	df.writeTrafficPolicies()
	df.writeReadme()
}

// writeControlPlaneInformation retrieves and writes information about the control plane resources.
func (df *DataFetcher) writeControlPlaneInformation() {
	if err := df.createControlPlaneDirectory(); err != nil {
		log.Printf("- could not create proper directory structure: %v", err)

		return
	}
	df.writeControlPlanePods()
	df.writeControlPlaneDeployments()
	df.writeControlPlaneStatefulSets()
	df.writeControlPlaneDaemonSets()
	df.writeControlPlaneServices()
	df.writeControlPlaneConfigMaps()
	df.writeControlPlaneServiceAccounts()
	df.writeControlPlaneSecrets()
	df.writeControlPlaneEvents()
	df.writeControlPlaneValidatingWebhookConfigurations()
	df.writeControlPlaneMutatingWebhookConfigurations()
	df.writeControlPlaneClusterRoles()
	df.writeControlPlaneClusterRoleBindings()
	df.writeControlPlaneCRDs()
	df.writeControlPlaneAPIServices()
	df.writeK8sMetrics()
}

// createControlPlaneDirectory creates the sub-directory for the nginx-mesh namespace.
func (df *DataFetcher) createControlPlaneDirectory() error {
	df.controlPlaneDir = filepath.Join(df.directory, df.meshNamespace)

	return df.writer.MkdirAll(df.controlPlaneDir)
}

// writeControlPlanePods retrieves and writes information about control plane pods.
func (df *DataFetcher) writeControlPlanePods() {
	log.Println("Getting control plane Pods.")
	pods, err := df.k8sClient.ClientSet().CoreV1().Pods(df.meshNamespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		log.Printf("- could not list Pods: %v\n", err)

		return
	}

	podDescriber := describe.PodDescriber{Interface: df.k8sClient.ClientSet()}

	df.writeControlPlanePodList(pods)

	for iter, pod := range pods.Items {
		// create directory for Pod
		if err := df.writer.Mkdir(filepath.Join(df.controlPlaneDir, pod.Name)); err != nil {
			log.Printf("- could not create control plane Pod directory: %v\n", err)

			continue
		}

		writeFile := func(filename, contents string) {
			if contents != "" {
				if err := df.writer.Write(filename, contents); err != nil {
					log.Printf("- could not write %s for Pod '%s/%s': %v",
						filename, df.meshNamespace, pod.Name, err)
				}
			}
		}

		// write yaml
		podYaml, err := yaml.Marshal(pod)
		if err != nil {
			log.Printf("- could not create yaml for Pod '%s/%s': %v\n", df.meshNamespace, pod.Name, err)
		} else {
			writeFile(filepath.Join(df.controlPlaneDir, pod.Name, podYamlFile), string(podYaml))
		}

		// write description
		description, err := podDescriber.Describe(df.meshNamespace, pod.Name, describe.DescriberSettings{ShowEvents: true})
		if err != nil {
			log.Printf("- could not get description for Pod '%s/%s': %v\n", df.meshNamespace, pod.Name, err)
		} else {
			writeFile(filepath.Join(df.controlPlaneDir, pod.Name, podDescFile), description)
		}

		// write logs
		df.writeContainerLogs(&pods.Items[iter], df.controlPlaneDir, nil)
	}
}

// writeControlPlanePods is the equivalent of `kubectl get pods -o wide`.
func (df *DataFetcher) writeControlPlanePodList(podList *v1.PodList) {
	podTable := &metav1.Table{
		ColumnDefinitions: []metav1.TableColumnDefinition{
			{Name: "Name", Type: "string", Format: "name"},
			{Name: "Ready", Type: "string", Format: ""},
			{Name: "Status", Type: "string", Format: ""},
			{Name: "Restarts", Type: "integer", Format: ""},
			{Name: "Age", Type: "string", Format: ""},
			{Name: "IP", Type: "string", Format: "", Priority: 1},
			{Name: "Node", Type: "string", Format: "", Priority: 1},
			{Name: "Nominated Node", Type: "string", Format: "", Priority: 1},
			{Name: "Readiness Gates", Type: "string", Format: "", Priority: 1},
		},
	}
	rows, err := printPodList(podList)
	if err != nil {
		log.Printf("- could not create Pod Table: %v\n", err)

		return
	}
	podTable.Rows = rows
	printer := printers.NewTablePrinter(printers.PrintOptions{
		Wide: true,
	})
	out := bytes.NewBuffer([]byte{})
	if err = printer.PrintObj(podTable, out); err != nil {
		log.Printf("- could not print Pod Table: %v\n", err)

		return
	}
	if err = df.writer.Write(filepath.Join(df.controlPlaneDir, podListFile), out.String()); err != nil {
		log.Printf("- could not write PodList: %v", err)
	}
}

// writeControlPlaneDeployments retrieves and writes information about control plane deployments.
func (df *DataFetcher) writeControlPlaneDeployments() {
	log.Println("Getting control plane Deployments.")
	deployments, err := df.k8sClient.ClientSet().AppsV1().Deployments(df.meshNamespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		log.Printf("- could not list Deployments: %v\n", err)

		return
	}

	if err = df.createYamlFromList(deployments, filepath.Join(df.controlPlaneDir, deploymentsFile)); err != nil {
		log.Printf("- could not write Deployments to file: %v", err)
	}
}

// writeControlPlaneStatefulSets retrieves and writes information about the control plane statefulsets.
func (df *DataFetcher) writeControlPlaneStatefulSets() {
	log.Println("Getting control plane StatefulSets.")
	statefulSets, err := df.k8sClient.ClientSet().AppsV1().StatefulSets(df.meshNamespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		log.Printf("- could not list StatefulSets: %v\n", err)

		return
	}

	if err = df.createYamlFromList(statefulSets, filepath.Join(df.controlPlaneDir, statefulSetsFile)); err != nil {
		log.Printf("- could not write StatefulSets to file: %v", err)
	}
}

// writeControlPlaneDaemonSets retrieves and writes information about control plane daemonsets.
func (df *DataFetcher) writeControlPlaneDaemonSets() {
	log.Println("Getting control plane DaemonSets.")
	daemonSets, err := df.k8sClient.ClientSet().AppsV1().DaemonSets(df.meshNamespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		log.Printf("- could not list DaemonSets: %v\n", err)

		return
	}

	if err = df.createYamlFromList(daemonSets, filepath.Join(df.controlPlaneDir, daemonSetsFile)); err != nil {
		log.Printf("- could not write DaemonSets to file: %v", err)
	}
}

// writeControlPlaneServices retrieves and writes information about control plane services.
func (df *DataFetcher) writeControlPlaneServices() {
	log.Println("Getting control plane Services.")
	services, err := df.k8sClient.ClientSet().CoreV1().Services(df.meshNamespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		log.Printf("- could not list Services: %v\n", err)

		return
	}

	if err = df.createYamlFromList(services, filepath.Join(df.controlPlaneDir, servicesFile)); err != nil {
		log.Printf("- could not write Services to file: %v", err)
	}
}

// writeControlPlaneConfigMaps retrieves and writes information about control plane config maps.
func (df *DataFetcher) writeControlPlaneConfigMaps() {
	log.Println("Getting control plane ConfigMaps.")
	configMaps, err := df.k8sClient.ClientSet().CoreV1().ConfigMaps(df.meshNamespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		log.Printf("- could not list ConfigMaps: %v\n", err)

		return
	}

	if err = df.createYamlFromList(configMaps, filepath.Join(df.controlPlaneDir, configMapsFile)); err != nil {
		log.Printf("- could not write ConfigMaps to file: %v", err)
	}
}

// writeControlPlaneServiceAccounts retrieves and writes information about control plane service accounts.
func (df *DataFetcher) writeControlPlaneServiceAccounts() {
	log.Println("Getting control plane Service Accounts.")
	serviceAccounts, err := df.k8sClient.ClientSet().CoreV1().ServiceAccounts(df.meshNamespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		log.Printf("- could not list ServiceAccounts: %v\n", err)

		return
	}

	if err = df.createYamlFromList(serviceAccounts, filepath.Join(df.controlPlaneDir, serviceAccountsFile)); err != nil {
		log.Printf("- could not write ServiceAccounts to file: %v", err)
	}
}

// writeControlPlaneSecrets retrieves and writes information about control plane secrets.
func (df *DataFetcher) writeControlPlaneSecrets() {
	log.Println("Getting control plane Secrets.")
	secrets, err := df.k8sClient.ClientSet().CoreV1().Secrets(df.meshNamespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		log.Printf("- could not list Secrets: %v\n", err)

		return
	}

	if err = df.createYamlFromList(secrets, filepath.Join(df.controlPlaneDir, secretsFile)); err != nil {
		log.Printf("- could not write Secrets to file: %v", err)
	}
}

// writeControlPlaneEvents is the equivalent of `kubectl get events -o wide`.
func (df *DataFetcher) writeControlPlaneEvents() {
	log.Println("Getting control plane Events.")
	events, err := df.k8sClient.ClientSet().CoreV1().Events(df.meshNamespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		log.Printf("- could not list Events: %v\n", err)
	}

	eventTable := &metav1.Table{
		ColumnDefinitions: []metav1.TableColumnDefinition{
			{Name: "Last Seen", Type: "string"},
			{Name: "Type", Type: "string"},
			{Name: "Reason", Type: "string"},
			{Name: "Object", Type: "string"},
			{Name: "Subobject", Type: "string", Priority: 1},
			{Name: "Source", Type: "string", Priority: 1},
			{Name: "Message", Type: "string"},
			{Name: "First Seen", Type: "string", Priority: 1},
			{Name: "Count", Type: "string", Priority: 1},
			{Name: "Name", Type: "string", Priority: 1},
		},
	}
	rows, err := printEventList(events)
	if err != nil {
		log.Printf("- could not create Event Table: %v\n", err)

		return
	}
	eventTable.Rows = rows
	printer := printers.NewTablePrinter(printers.PrintOptions{
		Wide: true,
	})
	out := bytes.NewBuffer([]byte{})
	if err = printer.PrintObj(eventTable, out); err != nil {
		log.Printf("- could not print Event Table: %v\n", err)

		return
	}
	if err := df.writer.Write(filepath.Join(df.controlPlaneDir, eventListFile), out.String()); err != nil {
		log.Printf("- could not write EventList: %v", err)
	}
}

// writeControlPlaneValidatingWebhookConfigurations retrieves and writes information about validating webhook configurations.
func (df *DataFetcher) writeControlPlaneValidatingWebhookConfigurations() {
	log.Println("Getting ValidatingWebhookConfigurations.")
	validatingWebhookConfigurations, err := df.k8sClient.ClientSet().AdmissionregistrationV1().ValidatingWebhookConfigurations().List(
		context.TODO(),
		metav1.ListOptions{
			LabelSelector: meshLabelSelectorKey + "=" + meshLabelSelectorValue,
		},
	)
	if err != nil {
		log.Printf("- could not list ValidatingWebhookConfigurations: %v\n", err)

		return
	}

	if err = df.createYamlFromList(
		validatingWebhookConfigurations, filepath.Join(df.directory, validatingWebhookConfigurationsFile)); err != nil {
		log.Printf("- could not write ValidatingWebhookConfigurations to file: %v", err)
	}
}

// writeControlPlaneMutatingWebhookConfigurations retrieves and writes information about mutating webhook configurations.
func (df *DataFetcher) writeControlPlaneMutatingWebhookConfigurations() {
	log.Println("Getting MutatingWebhookConfigurations.")
	mutatingWebhookConfigurations, err := df.k8sClient.ClientSet().AdmissionregistrationV1().MutatingWebhookConfigurations().List(
		context.TODO(),
		metav1.ListOptions{
			LabelSelector: meshLabelSelectorKey + "=" + meshLabelSelectorValue,
		},
	)
	if err != nil {
		log.Printf("- could not list MutatingWebhookConfigurations: %v\n", err)

		return
	}

	if err = df.createYamlFromList(
		mutatingWebhookConfigurations, filepath.Join(df.directory, mutatingWebhookConfigurationsFile)); err != nil {
		log.Printf("- could not write MutatingWebhookConfigurations to file: %v", err)
	}
}

// writeControlPlaneClusterRoles retrieves and writes information about cluster roles.
func (df *DataFetcher) writeControlPlaneClusterRoles() {
	log.Println("Getting ClusterRoles.")
	clusterRoles, err := df.k8sClient.ClientSet().RbacV1().ClusterRoles().List(context.TODO(), metav1.ListOptions{
		LabelSelector: meshLabelSelectorKey + "=" + meshLabelSelectorValue,
	})
	if err != nil {
		log.Printf("- could not list ClusterRoles: %v\n", err)

		return
	}

	if err = df.createYamlFromList(clusterRoles, filepath.Join(df.directory, clusterRolesFile)); err != nil {
		log.Printf("- could not write ClusterRoles to file: %v", err)
	}
}

// writeControlPlaneClusterRoleBindings retrieves and writes information about cluster role bindings.
func (df *DataFetcher) writeControlPlaneClusterRoleBindings() {
	log.Println("Getting ClusterRoleBindings.")
	clusterRoleBindings, err := df.k8sClient.ClientSet().RbacV1().ClusterRoleBindings().List(context.TODO(), metav1.ListOptions{
		LabelSelector: meshLabelSelectorKey + "=" + meshLabelSelectorValue,
	})
	if err != nil {
		log.Printf("- could not list ClusterRoleBindings: %v\n", err)

		return
	}

	if err = df.createYamlFromList(clusterRoleBindings, filepath.Join(df.directory, clusterRoleBindingsFile)); err != nil {
		log.Printf("- could not write ClusterRoleBindings to file: %v", err)
	}
}

// writeControlPlaneCRDs retrieves and writes information about control plane custom resource definitions (CRDs).
func (df *DataFetcher) writeControlPlaneCRDs() {
	log.Println("Getting control plane CRDs.")
	crds, err := df.k8sClient.APIExtensionClientSet().ApiextensionsV1().CustomResourceDefinitions().List(context.TODO(), metav1.ListOptions{
		LabelSelector: meshLabelSelectorKey + "=" + meshLabelSelectorValue,
	})
	if err != nil {
		log.Printf("- could not list CRDs: %v\n", err)

		return
	}

	if err = df.createYamlFromList(crds, filepath.Join(df.directory, crdsFile)); err != nil {
		log.Printf("- could not write CRDs to file: %v", err)
	}
}

// writeControlPlaneAPIServices retrieves and writes information about control plane api services.
func (df *DataFetcher) writeControlPlaneAPIServices() {
	log.Println("Getting control plane APIServices.")
	apiServices, err := df.k8sClient.APIRegistrationClientSet().ApiregistrationV1().APIServices().List(context.TODO(), metav1.ListOptions{
		LabelSelector: meshLabelSelectorKey + "=" + meshLabelSelectorValue,
	})
	if err != nil {
		log.Printf("- could not list APIServices: %v\n", err)

		return
	}

	if err = df.createYamlFromList(apiServices, filepath.Join(df.directory, apiServicesFile)); err != nil {
		log.Printf("- could not write APIServices to file: %v", err)
	}
}

// writeMeshConfig retrieves and writes the mesh config.
func (df *DataFetcher) writeMeshConfig() {
	log.Println("Getting mesh configuration.")
	meshCfg, err := df.k8sClient.ClientSet().CoreV1().
		ConfigMaps(df.meshNamespace).Get(context.TODO(), mesh.MeshConfigMap, metav1.GetOptions{})
	if err != nil {
		log.Printf("- could not get ConfigMap '%s/%s': %v", df.meshNamespace, mesh.MeshConfigMap, err)

		return
	}

	if err := df.writer.Write(
		filepath.Join(df.directory, mesh.MeshConfigFileName), meshCfg.Data[mesh.MeshConfigFileName]); err != nil {
		log.Printf("- could not write mesh configuration to file: %v", err)
	}
}

// writeDeployConfig retrieves and writes the deployment config.
func (df *DataFetcher) writeDeployConfig() {
	log.Println("Getting deployment configuration.")
	values, _, err := helm.GetDeployValues(df.k8sClient, "nginx-service-mesh")
	if err != nil {
		log.Printf("- could not get deployment configuration: %v", err)

		return
	}

	// redact registry credentials if set
	redacted := "REDACTED"
	if values.Registry.Key != "" {
		values.Registry.Key = redacted
	}
	if values.Registry.Username != "" {
		values.Registry.Username = redacted
	}
	if values.Registry.Password != "" {
		values.Registry.Password = redacted
	}

	cfg, err := json.MarshalIndent(values, "", "\t")
	if err != nil {
		log.Printf("- could not format deployment configuration: %v", err)

		return
	}

	if err := df.writer.Write(filepath.Join(df.directory, "deploy-config.json"), string(cfg)); err != nil {
		log.Printf("- could not write deployment configuration to file: %v", err)
	}
}

// writeK8sMetrics writes the CPU and memory usage info for the control plane components.
func (df *DataFetcher) writeK8sMetrics() {
	log.Println("Getting control plane CPU and memory.")
	pods, err := df.k8sClient.MetricsClientSet().MetricsV1beta1().PodMetricses(df.meshNamespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		log.Printf("- could not get control plane Pod metrics: %v", err)

		return
	}

	var str string
	for _, pod := range pods.Items {
		str += fmt.Sprintf("Pod: %s\n", pod.Name)
		for _, container := range pod.Containers {
			cpu := container.Usage[v1.ResourceCPU]
			memory := container.Usage[v1.ResourceMemory]

			scaledMem := memory.ScaledValue(resource.Mega)
			str += fmt.Sprintf("- Container: %s; CPU: %v; Memory: %vM\n", container.Name, cpu.AsDec(), scaledMem)
		}
	}

	if err := df.writer.Write(filepath.Join(df.controlPlaneDir, "metrics.txt"), str); err != nil {
		log.Printf("- could not write metrics to file: %v", err)
	}
}

// writeSidecarInformation retrieves and writes information about the sidecar resources.
func (df *DataFetcher) writeSidecarInformation() {
	namespaces, err := df.k8sClient.ClientSet().CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		log.Printf("- could not list Namespaces: %v\n", err)

		return
	}

	for _, namespace := range namespaces.Items {
		// skip ignored namespaces
		if ok := mesh.IgnoredNamespaces[namespace.Name]; ok || namespace.Name == df.meshNamespace {
			continue
		}

		namespaceDir := filepath.Join(df.directory, namespace.Name)
		if err := df.writer.MkdirAll(namespaceDir); err != nil {
			log.Printf("- could not create proper directory structure: %v", err)

			continue
		}

		pods, err := df.k8sClient.ClientSet().CoreV1().Pods(namespace.Name).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			log.Printf("- could not list Pods: %v\n", err)

			continue
		}

		for iter, pod := range pods.Items {
			// only process this pod if it contains the sidecar
			if !podHelper.IsInjected(&pods.Items[iter]) {
				continue
			}

			// create directory for Pod
			podDir := filepath.Join(namespaceDir, pod.Name)
			if err := df.writer.Mkdir(podDir); err != nil {
				log.Printf("- could not create Pod directory: %v\n", err)

				continue
			}

			// logs
			containers := map[string]struct{}{
				mesh.MeshSidecar:     {},
				mesh.MeshSidecarInit: {},
			}
			df.writeContainerLogs(&pods.Items[iter], namespaceDir, containers)

			// pod yaml
			podYaml, err := yaml.Marshal(pod)
			if err != nil {
				log.Printf("- could not marshal yaml for Pod '%s/%s': %v\n", namespace.Name, pod.Name, err)

				continue
			}
			if err := df.writer.Write(filepath.Join(podDir, podYamlFile), string(podYaml)); err != nil {
				log.Printf("- could not write yaml for Pod '%s/%s': %v\n", namespace.Name, pod.Name, err)
			}
		}
	}
}

// writeContainerLogs retrieves and writes the current and previous logs for all containers in a pod.
// If containerNames are specified, only get logs for those containers.
func (df *DataFetcher) writeContainerLogs(pod *v1.Pod, directory string, containerNames map[string]struct{}) {
	containers := pod.Spec.Containers
	containers = append(containers, pod.Spec.InitContainers...)

	for _, container := range containers {
		if _, ok := containerNames[container.Name]; containerNames != nil && !ok {
			continue
		}
		current, previous := getContainerLogs(df.k8sClient.ClientSet(), pod.Namespace, pod.Name, container.Name)

		writeLogsFile := func(filename string, contents io.ReadCloser) {
			if contents != nil {
				if err := df.writer.WriteFromReader(filename, contents); err != nil {
					log.Printf("- could not write log file for container '%s' in Pod '%s/%s': %v",
						container.Name, pod.Namespace, pod.Name, err)
				}
			}
		}
		writeLogsFile(filepath.Join(directory, pod.Name, fmt.Sprintf("%s-logs.txt", container.Name)), current)
		writeLogsFile(filepath.Join(directory, pod.Name, fmt.Sprintf("%s-previous-logs.txt", container.Name)), previous)
	}
}

var policies = []struct {
	file     string
	resource schema.GroupVersionResource
}{
	{
		file: trafficSplitsFile,
		resource: schema.GroupVersionResource{
			Group:    split.GroupVersion.Group,
			Version:  split.GroupVersion.Version,
			Resource: "trafficsplits",
		},
	},
	{
		file: trafficTargetsFile,
		resource: schema.GroupVersionResource{
			Group:    access.GroupVersion.Group,
			Version:  access.GroupVersion.Version,
			Resource: "traffictargets",
		},
	},
	{
		file: httpRouteGroupsFile,
		resource: schema.GroupVersionResource{
			Group:    specs.GroupVersion.Group,
			Version:  specs.GroupVersion.Version,
			Resource: "httproutegroups",
		},
	},
	{
		file: tcpRoutesFile,
		resource: schema.GroupVersionResource{
			Group:    specs.GroupVersion.Group,
			Version:  specs.GroupVersion.Version,
			Resource: "tcproutes",
		},
	},
	{
		file: rateLimitsFile,
		resource: schema.GroupVersionResource{
			Group:    nsmspecsv1alpha2.SchemeGroupVersion.Group,
			Version:  nsmspecsv1alpha2.SchemeGroupVersion.Version,
			Resource: "ratelimits",
		},
	},
	{
		file: circuitBreakersFile,
		resource: schema.GroupVersionResource{
			Group:    nsmspecsv1alpha1.SchemeGroupVersion.Group,
			Version:  nsmspecsv1alpha1.SchemeGroupVersion.Version,
			Resource: "circuitbreakers",
		},
	},
}

// writeTrafficPolicies calls individual functions to write out TrafficSplits, TrafficTargets, etc.
func (df *DataFetcher) writeTrafficPolicies() {
	for _, policy := range policies {
		log.Printf("Getting %s\n", policy.resource.Resource)
		list, err := df.k8sClient.DynamicClientSet().Resource(policy.resource).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			log.Printf("- could not list %s: %v\n", policy.resource.Resource, err)

			return
		}

		if err = df.createYamlFromList(list, filepath.Join(df.directory, policy.file)); err != nil {
			log.Printf("- could not write %s to file: %v", policy.resource.Resource, err)
		}
	}
}

// createYamlFromList takes in a list of k8s objects and outputs a single yaml with all them.
func (df *DataFetcher) createYamlFromList(list runtime.Object, file string) error {
	listItems, err := meta.ExtractList(list)
	if err != nil {
		return err
	}

	for _, item := range listItems {
		itemYaml, err := yaml.Marshal(item)
		if err != nil {
			return err
		}
		itemMeta, err := meta.Accessor(item)
		if err != nil {
			return err
		}
		if err = df.writer.Write(file, withHeader(itemMeta.GetName(), string(itemYaml))); err != nil {
			return err
		}
	}

	return nil
}

// getContainerLogs returns the current and previous logs for a container.
func getContainerLogs(
	clientset kubernetes.Interface,
	namespace,
	name,
	container string,
) (io.ReadCloser, io.ReadCloser) {
	var current, previous io.ReadCloser
	getLogs := func(previous bool) io.ReadCloser {
		req := clientset.CoreV1().Pods(namespace).GetLogs(name, &v1.PodLogOptions{Previous: previous, Container: container})
		logs, streamErr := req.Stream(context.TODO())
		if streamErr != nil {
			log.Printf("- could not read log stream: %v", streamErr)

			return nil
		}

		return logs
	}

	// current logs
	log.Printf("Getting logs for container '%s' in Pod '%s/%s'.\n", container, namespace, name)
	current = getLogs(false)

	// previous logs
	log.Printf("Getting previous logs for container '%s' in Pod '%s/%s'.\n", container, namespace, name)
	previous = getLogs(true)

	return current, previous
}

// writeReadme writes a basic README describing the contents of the support package.
func (df *DataFetcher) writeReadme() {
	readme := `
# Contents of support package

## {{.Namespace}}/

Directory containing NGINX Service Mesh control plane information.

- {{.Namespace}}/
	- \<pod-name\>/ - Directory containing Pod-specific information.
		- \<container-name\>-logs.txt: Logs for the container.
		- \<container-name\>-previous-logs.txt: Previous logs for the container (if applicable).
		- desc.txt: Description of the Pod.
		- pod.yaml: Configuration of the Pod.
	- configmaps.yaml: All the ConfigMap configurations in the {{.Namespace}} namespace.
	- daemonsets.yaml: All the DaemonSet configurations in the {{.Namespace}} namespace.
	- deployments.yaml: All the Deployment configurations in the {{.Namespace}} namespace.
	- events.txt: All the Event configurations in the {{.Namespace}} namespace.
	- metrics.txt: CPU and memory usage of each Pod.
	- pods.txt: Output of "kubectl -n {{.Namespace}} get pods -o wide".
	- secrets.yaml: All the Secret configurations in the {{.Namespace}} namespace.
	- serviceaccounts.yaml: All the ServiceAccount configurations in the {{.Namespace}} namespace.
	- services.yaml: All the Service configurations in the {{.Namespace}} namespace.
	- statefulsets.yaml: All the StatefulSet configurations in the {{.Namespace}} namespace.

{{- if .CollectSidecarLogs }}
## \<user-namespace\>/

Directory containing sidecar information.

- \<user-namespace\>/
	- \<pod-name\>/ - Directory containing Pod-specific information.
		- nginx-mesh-init-logs.txt: Logs of the nginx-mesh-init container.
		- nginx-mesh-sidecar-logs.txt: Logs of the nginx-mesh-sidecar container.
		- nginx-mesh-init-previous-logs.txt: Previous logs of the nginx-mesh-init container (if applicable).
		- nginx-mesh-sidecar-previous-logs.txt: Previous logs of the nginx-mesh-sidecar container (if applicable).
		- pod.yaml: Configuration of the Pod.
{{- end }}

## Configuration files

- apiservices.yaml: All the NGINX Service Mesh APIService configurations.
- circuitbreakers.yaml: All the CircuitBreaker configurations.
- clusterrolebindings.yaml: All the NGINX Service Mesh ClusterRoleBinding configurations.
- clusterroles.yaml: All the NGINX Service Mesh ClusterRole configurations.
- crds.yaml: All the NGINX Service Mesh Custom Resource Definition (CRD) configurations.
- deploy-config.json: Deploy-time configuration of NGINX Service Mesh.
- httproutegroups.yaml: All the HTTPRouteGroup configurations.
- mesh-config.json: Output of "nginx-meshctl config".
- mutatingwebhookconfigurations.yaml: All the NGINX Service Mesh MutatingWebhookConfiguration configurations.
- ratelimits.yaml: All the RateLimit configurations.
- supportpkg-creation-logs.txt: Logs that occurred while the support package was being created.
- tcproutes.yaml: All the TCPRoute configurations.
- trafficsplits.yaml: All the TrafficSplit configurations.
- traffictargets.yaml: All the TrafficTarget configurations.
- validatingwebhookconfigurations.yaml: All the NGINX Service Mesh ValidatingWebhookConfiguration configurations.
- version.txt: Output of "nginx-meshctl version".
`

	data := struct {
		Namespace          string
		CollectSidecarLogs bool
	}{
		Namespace:          df.meshNamespace,
		CollectSidecarLogs: df.collectSidecarLogs,
	}
	tmpl, err := template.New("readme").Parse(readme)
	if err != nil {
		log.Printf("- could not parse README template: %v\n", err)

		return
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		log.Printf("- could not render README: %v\n", err)

		return
	}

	if err := df.writer.Write(filepath.Join(df.directory, "README.md"), buf.String()); err != nil {
		log.Printf("- could not write README: %v\n", err)
	}
}

// withHeader adds a banner to separate multiple yamls in the same file.
func withHeader(name, yamlString string) string {
	return fmt.Sprintf("---\n"+
		"##################################################################################################\n"+
		"# %s\n"+
		"##################################################################################################\n"+
		"%s", name, yamlString)
}
