package support

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	accessv1alpha2 "github.com/servicemeshinterface/smi-controller-sdk/apis/access/v1alpha2"
	specsv1alpha3 "github.com/servicemeshinterface/smi-controller-sdk/apis/specs/v1alpha3"
	splitv1alpha3 "github.com/servicemeshinterface/smi-controller-sdk/apis/split/v1alpha3"
	admv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apiextensionv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/fake"
	apiregistrationv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"
	metrics "k8s.io/metrics/pkg/apis/metrics/v1beta1"
	fakeMetrics "k8s.io/metrics/pkg/client/clientset/versioned/fake"
	"sigs.k8s.io/yaml"

	"github.com/nginxinc/nginx-service-mesh/pkg/apis/mesh"
	nsmspecsv1alpha1 "github.com/nginxinc/nginx-service-mesh/pkg/apis/specs/v1alpha1"
	nsmspecsv1alpha2 "github.com/nginxinc/nginx-service-mesh/pkg/apis/specs/v1alpha2"
	fakeK8s "github.com/nginxinc/nginx-service-mesh/pkg/k8s/fake"
)

// linter bug can't tell that we are actually using this fakeMetrics library, so this satisfies it.
var _ = fakeMetrics.Clientset{}

var _ = Describe("Support Data", func() {
	shouldSkipRelease := false
	var buf bytes.Buffer
	collectSidecarLogs := true
	fileWriter := NewWriter()
	namespace := v1.NamespaceDefault

	BeforeEach(func() {
		log.SetOutput(&buf)
	})
	AfterEach(func() {
		deleteTmpDirContents()
		buf.Reset()
	})

	It("writes control plane information", func() {
		pod1 := &v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: namespace,
				Name:      "control1",
			},
			Spec: v1.PodSpec{
				Containers: []v1.Container{
					{
						Name: "c1",
					},
				},
			},
		}
		pod1Description := `Name:         control1
Namespace:    default
Node:         <none>
Labels:       <none>
Annotations:  <none>
Status:       
IP:           
IPs:          <none>
Containers:
  c1:
    Image:        
    Port:         <none>
    Host Port:    <none>
    Environment:  <none>
    Mounts:       <none>
Volumes:          <none>
QoS Class:        BestEffort
Node-Selectors:   <none>
Tolerations:      <none>
Events:
  Type  Reason   Age        From  Message
  ----  ------   ----       ----  -------
        testing  <unknown>        fake event
`
		pod2 := &v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: namespace,
				Name:      "control2",
			},
			Spec: v1.PodSpec{
				Containers: []v1.Container{
					{
						Name: "c2",
					},
				},
			},
		}
		pod2Description := `Name:         control2
Namespace:    default
Node:         <none>
Labels:       <none>
Annotations:  <none>
Status:       
IP:           
IPs:          <none>
Containers:
  c2:
    Image:        
    Port:         <none>
    Host Port:    <none>
    Environment:  <none>
    Mounts:       <none>
Volumes:          <none>
QoS Class:        BestEffort
Node-Selectors:   <none>
Tolerations:      <none>
Events:
  Type  Reason   Age        From  Message
  ----  ------   ----       ----  -------
        testing  <unknown>        fake event
`
		podList := `NAME       READY   STATUS   RESTARTS   AGE         IP       NODE     NOMINATED NODE   READINESS GATES
control1   0/1              0          <unknown>   <none>   <none>   <none>           <none>
control2   0/1              0          <unknown>   <none>   <none>   <none>           <none>
`
		deployment := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "deployment-control",
				Namespace: namespace,
			},
		}
		statefulSet := &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "statefulset-control",
				Namespace: namespace,
			},
		}
		daemonSet := &appsv1.DaemonSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "daemonset-control",
				Namespace: namespace,
			},
		}
		service := &v1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "service-control",
				Namespace: namespace,
			},
		}
		configMap := &v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "configmap-control",
				Namespace: namespace,
			},
			Data: map[string]string{
				"foo": "bar",
			},
		}
		serviceAccount := &v1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "serviceaccount-control",
				Namespace: namespace,
			},
		}
		secret := &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "secret-control",
				Namespace: namespace,
			},
			Data: map[string][]byte{
				"foo": []byte("bar"),
			},
		}
		event := &v1.Event{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "event-control",
				Namespace: namespace,
			},
			Message: "fake event",
			Reason:  "testing",
		}
		eventList := `LAST SEEN   TYPE   REASON    OBJECT   SUBOBJECT   SOURCE   MESSAGE      FIRST SEEN   COUNT   NAME
<unknown>          testing                                 fake event   <unknown>    1       event-control
`
		validatingWebhookConfiguration := &admv1.ValidatingWebhookConfiguration{
			ObjectMeta: metav1.ObjectMeta{
				Name: "validatingwebhookconfiguration-control",
				Labels: map[string]string{
					meshLabelSelectorKey: meshLabelSelectorValue,
				},
			},
			Webhooks: []admv1.ValidatingWebhook{
				{
					Name: "validatingwebhook-control",
					ClientConfig: admv1.WebhookClientConfig{
						CABundle: []byte("Validating Webhook CA Bundle"),
						Service: &admv1.ServiceReference{
							Name:      service.Name,
							Namespace: namespace,
						},
					},
				},
			},
		}
		mutatingWebhookConfiguration := &admv1.MutatingWebhookConfiguration{
			ObjectMeta: metav1.ObjectMeta{
				Name: "mutatingwebhookconfiguration-control",
				Labels: map[string]string{
					meshLabelSelectorKey: meshLabelSelectorValue,
				},
			},
			Webhooks: []admv1.MutatingWebhook{
				{
					Name: "mutatingwebhook-control",
					ClientConfig: admv1.WebhookClientConfig{
						CABundle: []byte("Mutating Webhook CA Bundle"),
						Service: &admv1.ServiceReference{
							Name:      service.Name,
							Namespace: namespace,
						},
					},
				},
			},
		}
		clusterRole := &rbacv1.ClusterRole{
			ObjectMeta: metav1.ObjectMeta{
				Name: "clusterrole-control",
				Labels: map[string]string{
					meshLabelSelectorKey: meshLabelSelectorValue,
				},
			},
		}
		clusterRoleBinding := &rbacv1.ClusterRoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name: "clusterrolebinding-control",
				Labels: map[string]string{
					meshLabelSelectorKey: meshLabelSelectorValue,
				},
			},
			RoleRef: rbacv1.RoleRef{
				Kind: "ClusterRole",
				Name: clusterRole.Name,
			},
			Subjects: []rbacv1.Subject{
				{
					Kind:      "ServiceAccount",
					Name:      serviceAccount.Name,
					Namespace: namespace,
				},
			},
		}
		crd := &apiextensionv1.CustomResourceDefinition{
			ObjectMeta: metav1.ObjectMeta{
				Name: "crd-control",
				Labels: map[string]string{
					meshLabelSelectorKey: meshLabelSelectorValue,
				},
			},
		}
		apiService := &apiregistrationv1.APIService{
			ObjectMeta: metav1.ObjectMeta{
				Name: "apiservice-control",
				Labels: map[string]string{
					meshLabelSelectorKey: meshLabelSelectorValue,
				},
			},
			Spec: apiregistrationv1.APIServiceSpec{
				CABundle: []byte("API Service CA Bundle"),
			},
		}

		// create K8s objects

		k8sConfig := fakeK8s.NewFakeK8s(namespace, shouldSkipRelease, pod1, pod2, deployment, statefulSet, daemonSet, service, configMap,
			serviceAccount, secret, validatingWebhookConfiguration, mutatingWebhookConfiguration, clusterRole, clusterRoleBinding, event)
		_, err := k8sConfig.APIRegistrationClientSet().ApiregistrationV1().APIServices().Create(
			context.TODO(), apiService, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		_, err = k8sConfig.APIExtensionClientSet().ApiextensionsV1().CustomResourceDefinitions().Create(
			context.TODO(), crd, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		// write everything to disk
		dataFetcher := NewDataFetcher(k8sConfig, fileWriter, "", tmpDir, collectSidecarLogs)
		dataFetcher.writeControlPlaneInformation()
		Expect(buf.String()).ToNot(ContainSubstring("could not"))

		// create yamls to compare against
		pod1Yaml, err := yaml.Marshal(pod1)
		Expect(err).ToNot(HaveOccurred())
		pod2Yaml, err := yaml.Marshal(pod2)
		Expect(err).ToNot(HaveOccurred())
		deploymentYaml, err := yaml.Marshal(deployment)
		Expect(err).ToNot(HaveOccurred())
		statefulSetYaml, err := yaml.Marshal(statefulSet)
		Expect(err).ToNot(HaveOccurred())
		daemonSetYaml, err := yaml.Marshal(daemonSet)
		Expect(err).ToNot(HaveOccurred())
		serviceYaml, err := yaml.Marshal(service)
		Expect(err).ToNot(HaveOccurred())
		configMapYaml, err := yaml.Marshal(configMap)
		Expect(err).ToNot(HaveOccurred())
		serviceAccountYaml, err := yaml.Marshal(serviceAccount)
		Expect(err).ToNot(HaveOccurred())
		secretYaml, err := yaml.Marshal(secret)
		Expect(err).ToNot(HaveOccurred())
		validatingWebhookConfigurationYaml, err := yaml.Marshal(validatingWebhookConfiguration)
		Expect(err).ToNot(HaveOccurred())
		mutatingWebhookConfigurationYaml, err := yaml.Marshal(mutatingWebhookConfiguration)
		Expect(err).ToNot(HaveOccurred())
		clusterRoleYaml, err := yaml.Marshal(clusterRole)
		Expect(err).ToNot(HaveOccurred())
		clusterRoleBindingYaml, err := yaml.Marshal(clusterRoleBinding)
		Expect(err).ToNot(HaveOccurred())
		crdYaml, err := yaml.Marshal(crd)
		Expect(err).ToNot(HaveOccurred())
		apiServiceYaml, err := yaml.Marshal(apiService)
		Expect(err).ToNot(HaveOccurred())

		// verify files exist and contain expected contents
		dir := filepath.Join(tmpDir, namespace)
		files := []struct {
			name     string
			expected string
		}{
			{
				name:     filepath.Join(dir, pod1.Name, fmt.Sprintf("%s-logs.txt", pod1.Spec.Containers[0].Name)),
				expected: "fake logs",
			},
			{
				name:     filepath.Join(dir, pod1.Name, fmt.Sprintf("%s-previous-logs.txt", pod1.Spec.Containers[0].Name)),
				expected: "fake logs",
			},
			{
				name:     filepath.Join(dir, pod1.Name, podYamlFile),
				expected: string(pod1Yaml),
			},
			{
				name:     filepath.Join(dir, pod1.Name, podDescFile),
				expected: pod1Description,
			},
			{
				name:     filepath.Join(dir, pod2.Name, fmt.Sprintf("%s-logs.txt", pod2.Spec.Containers[0].Name)),
				expected: "fake logs",
			},
			{
				name:     filepath.Join(dir, pod2.Name, fmt.Sprintf("%s-previous-logs.txt", pod2.Spec.Containers[0].Name)),
				expected: "fake logs",
			},
			{
				name:     filepath.Join(dir, pod2.Name, podYamlFile),
				expected: string(pod2Yaml),
			},
			{
				name:     filepath.Join(dir, pod2.Name, podDescFile),
				expected: pod2Description,
			},
			{
				name:     filepath.Join(dir, podListFile),
				expected: podList,
			},
			{
				name:     filepath.Join(dir, deploymentsFile),
				expected: withHeader(deployment.Name, string(deploymentYaml)),
			},
			{
				name:     filepath.Join(dir, statefulSetsFile),
				expected: withHeader(statefulSet.Name, string(statefulSetYaml)),
			},
			{
				name:     filepath.Join(dir, daemonSetsFile),
				expected: withHeader(daemonSet.Name, string(daemonSetYaml)),
			},
			{
				name:     filepath.Join(dir, servicesFile),
				expected: withHeader(service.Name, string(serviceYaml)),
			},
			{
				name:     filepath.Join(dir, configMapsFile),
				expected: withHeader(configMap.Name, string(configMapYaml)),
			},
			{
				name:     filepath.Join(dir, serviceAccountsFile),
				expected: withHeader(serviceAccount.Name, string(serviceAccountYaml)),
			},
			{
				name:     filepath.Join(dir, secretsFile),
				expected: withHeader(secret.Name, string(secretYaml)),
			},
			{
				name:     filepath.Join(dir, eventListFile),
				expected: eventList,
			},
			{
				name:     filepath.Join(tmpDir, validatingWebhookConfigurationsFile),
				expected: withHeader(validatingWebhookConfiguration.Name, string(validatingWebhookConfigurationYaml)),
			},
			{
				name:     filepath.Join(tmpDir, mutatingWebhookConfigurationsFile),
				expected: withHeader(mutatingWebhookConfiguration.Name, string(mutatingWebhookConfigurationYaml)),
			},
			{
				name:     filepath.Join(tmpDir, clusterRolesFile),
				expected: withHeader(clusterRole.Name, string(clusterRoleYaml)),
			},
			{
				name:     filepath.Join(tmpDir, clusterRoleBindingsFile),
				expected: withHeader(clusterRoleBinding.Name, string(clusterRoleBindingYaml)),
			},
			{
				name:     filepath.Join(tmpDir, crdsFile),
				expected: withHeader(crd.Name, string(crdYaml)),
			},
			{
				name:     filepath.Join(tmpDir, apiServicesFile),
				expected: withHeader(apiService.Name, string(apiServiceYaml)),
			},
		}

		for _, file := range files {
			blob, err := os.ReadFile(filepath.Clean(file.name))
			Expect(err).ToNot(HaveOccurred())
			Expect(string(blob)).To(Equal(file.expected))
		}
	})

	It("writes sidecar information", func() {
		nameObj := &v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: v1.NamespaceDefault,
			},
		}

		// create injected pod
		injectedPod := &v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: namespace,
				Name:      "good-pod",
				Annotations: map[string]string{
					mesh.InjectedAnnotation: mesh.Injected,
				},
			},
			Spec: v1.PodSpec{
				Containers: []v1.Container{
					{
						Name: "c3",
					},
					{
						Name: mesh.MeshSidecar,
					},
				},
				InitContainers: []v1.Container{
					{
						Name: mesh.MeshSidecarInit,
					},
				},
			},
		}

		// create pod in ignored namespace
		ignoredNSPod := &v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "kube-system",
				Name:      "ignoredNS",
				Annotations: map[string]string{
					mesh.InjectedAnnotation: mesh.Injected,
				},
			},
			Spec: v1.PodSpec{
				Containers: []v1.Container{
					{
						Name: "c1",
					},
				},
			},
		}

		// create non-injected pod
		nonInjectedPod := &v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: namespace,
				Name:      "not-injected",
			},
			Spec: v1.PodSpec{
				Containers: []v1.Container{
					{
						Name: "c2",
					},
				},
			},
		}

		k8sConfig := fakeK8s.NewFakeK8s("nginx-mesh", shouldSkipRelease, nameObj, ignoredNSPod, nonInjectedPod, injectedPod)

		dataFetcher := NewDataFetcher(k8sConfig, fileWriter, "", tmpDir, collectSidecarLogs)
		dataFetcher.writeSidecarInformation()

		// verify no errors were logged
		Expect(buf.String()).ToNot(ContainSubstring("could not"))

		expInjectedStr, err := yaml.Marshal(injectedPod)
		Expect(err).ToNot(HaveOccurred())
		injectedPodDir := filepath.Join(tmpDir, nameObj.Name, injectedPod.Name)

		// check for valid pod files
		files := []struct {
			name     string
			expected string
		}{
			{
				name:     filepath.Join(filepath.Clean(injectedPodDir), podYamlFile),
				expected: string(expInjectedStr),
			},
			{
				name:     filepath.Join(injectedPodDir, fmt.Sprintf("%s-logs.txt", mesh.MeshSidecar)),
				expected: "fake logs",
			},
			{
				name:     filepath.Join(injectedPodDir, fmt.Sprintf("%s-previous-logs.txt", mesh.MeshSidecar)),
				expected: "fake logs",
			},
			{
				name:     filepath.Join(injectedPodDir, fmt.Sprintf("%s-logs.txt", mesh.MeshSidecarInit)),
				expected: "fake logs",
			},
			{
				name:     filepath.Join(injectedPodDir, fmt.Sprintf("%s-previous-logs.txt", mesh.MeshSidecarInit)),
				expected: "fake logs",
			},
		}

		for _, file := range files {
			blob, readErr := os.ReadFile(file.name)
			Expect(readErr).ToNot(HaveOccurred())
			Expect(string(blob)).To(Equal(file.expected))
		}

		// logs for non-sidecar containers should not exist
		_, err = os.ReadFile(filepath.Join(filepath.Clean(injectedPodDir), "c3-logs.txt"))
		Expect(err).To(HaveOccurred())

		_, err = os.ReadFile(filepath.Join(filepath.Clean(injectedPodDir), "c3-previous-logs.txt"))
		Expect(err).To(HaveOccurred())

		// files for invalid pods should not exist
		_, err = os.ReadDir(filepath.Join(tmpDir, "kube-system", ignoredNSPod.Name))
		Expect(err).To(HaveOccurred())

		_, err = os.ReadDir(filepath.Join(tmpDir, nameObj.Name, nonInjectedPod.Name))
		Expect(err).To(HaveOccurred())
	})

	It("writes the mesh configuration", func() {
		configMap := &v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      mesh.MeshConfigMap,
				Namespace: namespace,
			},
			BinaryData: map[string][]byte{
				mesh.MeshConfigFileName: []byte("{\"field\": \"value\"}"),
			},
		}

		k8sConfig := fakeK8s.NewFakeK8s(namespace, shouldSkipRelease, configMap)

		dataFetcher := NewDataFetcher(k8sConfig, fileWriter, "", tmpDir, collectSidecarLogs)
		dataFetcher.writeMeshConfig()

		// verify no errors were logged
		Expect(buf.String()).ToNot(ContainSubstring("could not"))

		blob, err := os.ReadFile(filepath.Join(tmpDir, mesh.MeshConfigFileName))
		Expect(err).ToNot(HaveOccurred())
		Expect(string(blob)).To(Equal("{\n\t\"field\": \"value\"\n}"))

		// delete CM, should get error log
		Expect(k8sConfig.ClientSet().CoreV1().ConfigMaps(namespace).Delete(
			context.TODO(), mesh.MeshConfigMap, metav1.DeleteOptions{})).To(Succeed())

		dataFetcher.writeMeshConfig()
		Expect(buf.String()).To(ContainSubstring("could not get ConfigMap"))
	})

	It("writes the deployment configuration", func() {
		k8sConfig := fakeK8s.NewFakeK8s(namespace, shouldSkipRelease)

		dataFetcher := NewDataFetcher(k8sConfig, fileWriter, "", tmpDir, collectSidecarLogs)
		dataFetcher.writeDeployConfig()

		// verify no errors were logged
		Expect(buf.String()).ToNot(ContainSubstring("could not"))

		blob, err := os.ReadFile(filepath.Join(tmpDir, "deploy-config.json"))
		Expect(err).ToNot(HaveOccurred())
		// verify some default values
		Expect(string(blob)).To(ContainSubstring("\"accessControlMode\": \"allow\""))
		Expect(string(blob)).To(ContainSubstring("\"nginxErrorLogLevel\": \"warn\""))
	})

	It("writes k8s metrics information", func() {
		podMetrics := &metrics.PodMetrics{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: namespace,
				Name:      "pod",
			},
			Containers: []metrics.ContainerMetrics{
				{
					Name: "c1",
					Usage: v1.ResourceList{
						v1.ResourceCPU:    resource.MustParse("100m"),
						v1.ResourceMemory: resource.MustParse("3456789"),
					},
				},
			},
		}
		pod2Metrics := &metrics.PodMetrics{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: namespace,
				Name:      "pod2",
			},
			Containers: []metrics.ContainerMetrics{
				{
					Name: "c1",
					Usage: v1.ResourceList{
						v1.ResourceCPU:    resource.MustParse("200m"),
						v1.ResourceMemory: resource.MustParse("567890"),
					},
				},
				{
					Name: "c2",
					Usage: v1.ResourceList{
						v1.ResourceCPU:    resource.MustParse("300m"),
						v1.ResourceMemory: resource.MustParse("0123456"),
					},
				},
			},
		}
		gvr := schema.GroupVersionResource{Group: "metrics.k8s.io", Version: "v1beta1", Resource: "pods"}
		k8sConfig := fakeK8s.NewFakeK8s(namespace, shouldSkipRelease)
		mClient, ok := k8sConfig.MetricsClientSet().(*fakeMetrics.Clientset)
		Expect(ok).To(BeTrue())
		Expect(mClient.Tracker().Create(gvr, podMetrics, namespace)).To(Succeed())
		Expect(mClient.Tracker().Create(gvr, pod2Metrics, namespace)).To(Succeed())

		dataFetcher := NewDataFetcher(k8sConfig, fileWriter, "", tmpDir, collectSidecarLogs)
		Expect(dataFetcher.createControlPlaneDirectory()).To(Succeed())
		dataFetcher.writeK8sMetrics()
		// verify no errors were logged
		Expect(buf.String()).ToNot(ContainSubstring("could not"))

		file := fmt.Sprintf("%s/%s/metrics.txt", tmpDir, namespace)
		blob, err := os.ReadFile(filepath.Clean(file))
		Expect(err).ToNot(HaveOccurred())
		expStr := "Pod: pod\n- Container: c1; CPU: 0.100; Memory: 4M\nPod: pod2\n" +
			"- Container: c1; CPU: 0.200; Memory: 1M\n- Container: c2; CPU: 0.300; Memory: 1M\n"
		Expect(string(blob)).To(Equal(expStr))
	})

	It("writes container logs", func() {
		pod := &v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: namespace,
				Name:      "pod1",
			},
			Spec: v1.PodSpec{
				Containers: []v1.Container{
					{
						Name: "c1",
					},
					{
						Name: "c2",
					},
				},
			},
		}

		dataFetcher := NewDataFetcher(fakeK8s.NewFakeK8s(namespace, shouldSkipRelease), fileWriter, "", tmpDir, collectSidecarLogs)

		namespaceDir := filepath.Join(tmpDir, namespace)
		podDir := filepath.Join(namespaceDir, pod.Name)
		Expect(fileWriter.MkdirAll(podDir)).To(Succeed())

		dataFetcher.writeContainerLogs(pod, namespaceDir, nil)
		// verify no errors were logged
		Expect(buf.String()).ToNot(ContainSubstring("could not"))

		validateFiles := func(files []string) {
			for _, file := range files {
				blob, err := os.ReadFile(filepath.Clean(file))
				Expect(err).ToNot(HaveOccurred())
				Expect(string(blob)).To(Equal("fake logs"))
			}
		}

		// verify files exist and contain expected contents
		logs1File := filepath.Join(namespaceDir, pod.Name, fmt.Sprintf("%s-logs.txt", pod.Spec.Containers[0].Name))
		prevLogs1File := filepath.Join(namespaceDir, pod.Name, fmt.Sprintf("%s-previous-logs.txt", pod.Spec.Containers[0].Name))
		logs2File := filepath.Join(namespaceDir, pod.Name, fmt.Sprintf("%s-logs.txt", pod.Spec.Containers[1].Name))
		prevLogs2File := filepath.Join(namespaceDir, pod.Name, fmt.Sprintf("%s-previous-logs.txt", pod.Spec.Containers[1].Name))

		validateFiles([]string{logs1File, prevLogs1File, logs2File, prevLogs2File})

		// reset directory
		Expect(os.RemoveAll(podDir)).To(Succeed())
		Expect(fileWriter.MkdirAll(podDir)).To(Succeed())

		// now test that we only write logs for the specified container
		dataFetcher.writeContainerLogs(pod, namespaceDir, map[string]struct{}{"c1": {}})
		// verify no errors were logged
		Expect(buf.String()).ToNot(ContainSubstring("could not"))

		validateFiles([]string{logs1File, prevLogs1File})

		for _, file := range []string{logs2File, prevLogs2File} {
			_, err := os.ReadFile(filepath.Clean(file))
			Expect(err).To(HaveOccurred())
		}
	})

	It("gets container logs", func() {
		pod := &v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: namespace,
				Name:      "pod",
			},
			Spec: v1.PodSpec{
				Containers: []v1.Container{
					{
						Name: "c1",
					},
				},
			},
		}
		client := fake.NewSimpleClientset(pod)

		current, previous := getContainerLogs(client, namespace, pod.Name, pod.Spec.Containers[0].Name)

		logs := make([]byte, 9)
		_, err := current.Read(logs)
		Expect(err).ToNot(HaveOccurred())
		Expect(string(logs)).To(Equal("fake logs"))

		_, err = previous.Read(logs)
		Expect(err).ToNot(HaveOccurred())
		Expect(string(logs)).To(Equal("fake logs"))

		Expect(buf.String()).To(ContainSubstring("Getting logs for container 'c1' in Pod 'default/pod'."))
		Expect(buf.String()).To(ContainSubstring("Getting previous logs for container 'c1' in Pod 'default/pod'."))
		Expect(buf.String()).ToNot(ContainSubstring("could not"))
	})

	It("writes a readme file", func() {
		dataFetcher := NewDataFetcher(fakeK8s.NewFakeK8s("nginx-mesh", shouldSkipRelease), fileWriter, "", tmpDir, collectSidecarLogs)

		dataFetcher.writeReadme()
		// verify no errors were logged
		Expect(buf.String()).ToNot(ContainSubstring("could not"))

		file := filepath.Clean(filepath.Join(tmpDir, "README.md"))
		blob, err := os.ReadFile(file)
		Expect(err).ToNot(HaveOccurred())
		Expect(string(blob)).To(ContainSubstring("- nginx-mesh/"))
		Expect(string(blob)).To(ContainSubstring(`- \<user-namespace\>/`))

		Expect(os.Remove(file)).To(Succeed())

		// disable sidecar logs
		dataFetcher.collectSidecarLogs = false
		dataFetcher.writeReadme()
		// verify no errors were logged
		Expect(buf.String()).ToNot(ContainSubstring("could not"))

		blob, err = os.ReadFile(file)
		Expect(err).ToNot(HaveOccurred())
		Expect(string(blob)).ToNot(ContainSubstring(`- \<user-namespace\>/`))
	})

	It("writes traffic policies", func() {
		trafficSplit := &splitv1alpha3.TrafficSplit{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: namespace,
				Name:      "traffic-split",
			},
			Spec: splitv1alpha3.TrafficSplitSpec{
				Service: "foo",
			},
		}
		trafficTarget := &accessv1alpha2.TrafficTarget{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: namespace,
				Name:      "traffic-target",
			},
			Spec: accessv1alpha2.TrafficTargetSpec{
				Destination: accessv1alpha2.IdentityBindingSubject{
					Name:      "foo",
					Kind:      "bar",
					Namespace: namespace,
				},
			},
		}
		httpRouteGroup := &specsv1alpha3.HTTPRouteGroup{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: namespace,
				Name:      "http-route-group",
			},
			Spec: specsv1alpha3.HTTPRouteGroupSpec{
				Matches: []specsv1alpha3.HTTPMatch{
					{
						Name:      "foo",
						PathRegex: "*",
					},
				},
			},
		}
		tcpRoute := &specsv1alpha3.TCPRoute{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: namespace,
				Name:      "tcp-route",
			},
		}
		rateLimit := &nsmspecsv1alpha2.RateLimit{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: namespace,
				Name:      "rate-limit",
			},
			Spec: nsmspecsv1alpha2.RateLimitSpec{
				Name:  "foo",
				Rate:  "10rs",
				Burst: 30,
			},
		}
		circuitBreaker := &nsmspecsv1alpha1.CircuitBreaker{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: namespace,
				Name:      "circuit-breaker",
			},
			Spec: nsmspecsv1alpha1.CircuitBreakerSpec{
				Errors:         5,
				TimeoutSeconds: 30,
			},
		}

		resources := []struct {
			obj runtime.Object
			gvr schema.GroupVersionResource
		}{
			{
				gvr: schema.GroupVersionResource{
					Group:    splitv1alpha3.GroupVersion.Group,
					Version:  splitv1alpha3.GroupVersion.Version,
					Resource: "trafficsplits",
				},
				obj: trafficSplit,
			},
			{
				gvr: schema.GroupVersionResource{
					Group:    accessv1alpha2.GroupVersion.Group,
					Version:  accessv1alpha2.GroupVersion.Version,
					Resource: "traffictargets",
				},
				obj: trafficTarget,
			},
			{
				gvr: schema.GroupVersionResource{
					Group:    specsv1alpha3.GroupVersion.Group,
					Version:  specsv1alpha3.GroupVersion.Version,
					Resource: "httproutegroups",
				},
				obj: httpRouteGroup,
			},
			{
				gvr: schema.GroupVersionResource{
					Group:    specsv1alpha3.GroupVersion.Group,
					Version:  specsv1alpha3.GroupVersion.Version,
					Resource: "tcproutes",
				},
				obj: tcpRoute,
			},
			{
				gvr: schema.GroupVersionResource{
					Group:    nsmspecsv1alpha1.SchemeGroupVersion.Group,
					Version:  nsmspecsv1alpha1.SchemeGroupVersion.Version,
					Resource: "circuitbreakers",
				},
				obj: circuitBreaker,
			},
			{
				gvr: schema.GroupVersionResource{
					Group:    nsmspecsv1alpha2.SchemeGroupVersion.Group,
					Version:  nsmspecsv1alpha2.SchemeGroupVersion.Version,
					Resource: "ratelimits",
				},
				obj: rateLimit,
			},
		}
		k8sConfig := fakeK8s.NewFakeK8s(namespace, shouldSkipRelease)

		for _, resource := range resources {
			var unstrObj unstructured.Unstructured
			var err error
			unstrObj.Object, err = runtime.DefaultUnstructuredConverter.ToUnstructured(resource.obj)
			Expect(err).ToNot(HaveOccurred())

			_, err = k8sConfig.DynamicClientSet().Resource(resource.gvr).
				Namespace(unstrObj.GetNamespace()).Create(context.TODO(), &unstrObj, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
		}

		// write everything to disk
		dataFetcher := NewDataFetcher(k8sConfig, fileWriter, "", tmpDir, collectSidecarLogs)
		dataFetcher.writeTrafficPolicies()

		// create yamls to compare against
		trafficSplitYaml, err := yaml.Marshal(trafficSplit)
		Expect(err).ToNot(HaveOccurred())
		trafficTargetYaml, err := yaml.Marshal(trafficTarget)
		Expect(err).ToNot(HaveOccurred())
		httpRouteGroupYaml, err := yaml.Marshal(httpRouteGroup)
		Expect(err).ToNot(HaveOccurred())
		tcpRouteYaml, err := yaml.Marshal(tcpRoute)
		Expect(err).ToNot(HaveOccurred())
		rateLimitYaml, err := yaml.Marshal(rateLimit)
		Expect(err).ToNot(HaveOccurred())
		circuitBreakerYaml, err := yaml.Marshal(circuitBreaker)
		Expect(err).ToNot(HaveOccurred())

		// verify files exist and contain expected contents
		files := []struct {
			name     string
			expected string
		}{
			{
				name:     filepath.Join(tmpDir, trafficSplitsFile),
				expected: withHeader(trafficSplit.Name, string(trafficSplitYaml)),
			},
			{
				name:     filepath.Join(tmpDir, trafficTargetsFile),
				expected: withHeader(trafficTarget.Name, string(trafficTargetYaml)),
			},
			{
				name:     filepath.Join(tmpDir, httpRouteGroupsFile),
				expected: withHeader(httpRouteGroup.Name, string(httpRouteGroupYaml)),
			},
			{
				name:     filepath.Join(tmpDir, tcpRoutesFile),
				expected: withHeader(tcpRoute.Name, string(tcpRouteYaml)),
			},
			{
				name:     filepath.Join(tmpDir, rateLimitsFile),
				expected: withHeader(rateLimit.Name, string(rateLimitYaml)),
			},
			{
				name:     filepath.Join(tmpDir, circuitBreakersFile),
				expected: withHeader(circuitBreaker.Name, string(circuitBreakerYaml)),
			},
		}

		for _, file := range files {
			blob, err := os.ReadFile(filepath.Clean(file.name))
			Expect(err).ToNot(HaveOccurred())
			Expect(string(blob)).To(Equal(file.expected))
		}
	})
})
