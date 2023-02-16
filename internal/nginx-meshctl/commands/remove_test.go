package commands

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fakeClient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/nginxinc/nginx-service-mesh/pkg/apis/mesh"
	"github.com/nginxinc/nginx-service-mesh/pkg/k8s"
	"github.com/nginxinc/nginx-service-mesh/pkg/k8s/fake"
)

var _ = Describe("Remove", func() {
	var fakeK8s k8s.Client
	shouldSkipRelease := false

	BeforeEach(func() {
		fakeK8s = fake.NewFakeK8s("nginx-mesh", shouldSkipRelease)
	})

	It("gets proxied resources", func() {
		customNS := "custom-ns"
		trueVal := true
		expRes := proxiedResources{
			v1.NamespaceDefault: {
				"deployment": {
					"test-deployment",
				},
			},
			customNS: {
				"daemonset": {
					"test-daemonset",
				},
				"deployment": {
					"test-deployment",
				},
			},
		}

		podObj := &v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-pod",
				Namespace: v1.NamespaceDefault,
				Annotations: map[string]string{
					mesh.InjectedAnnotation: mesh.Injected,
				},
				OwnerReferences: []metav1.OwnerReference{
					{
						Kind:       "ReplicaSet",
						Name:       "test-replicaset",
						Controller: &trueVal,
					},
				},
			},
		}
		replicaset := &appsv1.ReplicaSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-replicaset",
				Namespace: v1.NamespaceDefault,
				OwnerReferences: []metav1.OwnerReference{
					{
						Kind:       "Deployment",
						Name:       "test-deployment",
						Controller: &trueVal,
					},
				},
			},
		}
		podObj2 := &v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-pod",
				Namespace: customNS,
				Annotations: map[string]string{
					mesh.InjectedAnnotation: mesh.Injected,
				},
				OwnerReferences: []metav1.OwnerReference{
					{
						Kind:       "ReplicaSet",
						Name:       "test-replicaset",
						Controller: &trueVal,
					},
				},
			},
		}
		replicaset2 := &appsv1.ReplicaSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-replicaset",
				Namespace: customNS,
				OwnerReferences: []metav1.OwnerReference{
					{
						Kind:       "Deployment",
						Name:       "test-deployment",
						Controller: &trueVal,
					},
				},
			},
		}
		podObj3 := &v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-pod2",
				Namespace: customNS,
				Annotations: map[string]string{
					mesh.InjectedAnnotation: mesh.Injected,
				},
				OwnerReferences: []metav1.OwnerReference{
					{
						Kind:       "DaemonSet",
						Name:       "test-daemonset",
						Controller: &trueVal,
					},
				},
			},
		}

		client := fakeClient.NewClientBuilder().WithRuntimeObjects(podObj, podObj2, podObj3, replicaset, replicaset2).Build()

		res, err := getProxiedResources(client)
		Expect(err).ToNot(HaveOccurred())
		Expect(res).To(Equal(expRes))
	})

	It("removes the mesh", func() {
		namespace := &v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "nginx-mesh",
			},
		}
		nsClient := fakeK8s.ClientSet().CoreV1().Namespaces()

		_, err := nsClient.Create(context.TODO(), namespace, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		deleteNamespace := true
		remover := newRemover(fakeK8s)
		Expect(remover.remove("nginx-service-mesh", deleteNamespace)).To(Succeed())

		_, err = nsClient.Get(context.TODO(), namespace.Name, metav1.GetOptions{})
		Expect(err).To(HaveOccurred())
	})

	It("removes CRDs", func() {
		// create CRD
		crd := &apiextv1.CustomResourceDefinition{
			ObjectMeta: metav1.ObjectMeta{
				Name: "crd-one",
				Labels: map[string]string{
					"app.kubernetes.io/part-of": "nginx-service-mesh",
				},
			},
		}

		client := fakeK8s.APIExtensionClientSet().ApiextensionsV1().CustomResourceDefinitions()
		_, err := client.Create(context.TODO(), crd, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		_, err = client.Get(context.TODO(), "crd-one", metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())

		// remove CRD
		removed, err := removeCRDs(context.TODO(), fakeK8s)
		Expect(err).ToNot(HaveOccurred())
		Expect(removed).To(BeTrue())
		_, err = client.Get(context.TODO(), "crd-one", metav1.GetOptions{})
		Expect(err).To(HaveOccurred())
	})
})
