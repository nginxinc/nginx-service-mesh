package pod_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/nginxinc/nginx-service-mesh/pkg/apis/mesh"
	"github.com/nginxinc/nginx-service-mesh/pkg/pod"
)

var _ = Describe("Pod", func() {
	It("determines if a pod is injected", func() {
		Expect(pod.IsInjected(&v1.Pod{})).To(BeFalse())

		podObj := &v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					mesh.InjectedAnnotation: "invalid",
				},
			},
		}
		Expect(pod.IsInjected(podObj)).To(BeFalse())

		podObj.Annotations[mesh.InjectedAnnotation] = mesh.Injected
		Expect(pod.IsInjected(podObj)).To(BeTrue())
	})

	Context("gets a pod owner", func() {
		trueVal := true
		It("has a replicaset-based owner", func() {
			podObj := &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
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
					Name: "test-replicaset",
					OwnerReferences: []metav1.OwnerReference{
						{
							Kind:       "Deployment",
							Name:       "test-deployment",
							Controller: &trueVal,
						},
					},
				},
			}
			k8sClient := fake.NewClientBuilder().WithRuntimeObjects(podObj, replicaset).Build()
			owner, name, err := pod.GetOwner(context.TODO(), k8sClient, podObj)
			Expect(err).ToNot(HaveOccurred())
			Expect(owner).To(Equal("deployment"))
			Expect(name).To(Equal("test-deployment"))
		})

		It("has a non-replicaset-based owner", func() {
			podObj := &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					OwnerReferences: []metav1.OwnerReference{
						{
							Kind:       "DaemonSet",
							Name:       "test-daemonset",
							Controller: &trueVal,
						},
					},
				},
			}
			k8sClient := fake.NewClientBuilder().WithRuntimeObjects(podObj).Build()
			owner, name, err := pod.GetOwner(context.TODO(), k8sClient, podObj)
			Expect(err).ToNot(HaveOccurred())
			Expect(owner).To(Equal("daemonset"))
			Expect(name).To(Equal("test-daemonset"))
		})

		It("has no owner", func() {
			podObj := &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-pod",
				},
			}
			k8sClient := fake.NewClientBuilder().WithRuntimeObjects(podObj).Build()
			owner, name, err := pod.GetOwner(context.TODO(), k8sClient, podObj)
			Expect(err).ToNot(HaveOccurred())
			Expect(owner).To(Equal("pod"))
			Expect(name).To(Equal("test-pod"))
		})
	})

	It("gets a replicaset owner", func() {
		ctlr := true
		replicas := &appsv1.ReplicaSet{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: v1.NamespaceDefault,
				Name:      "test",
				OwnerReferences: []metav1.OwnerReference{
					{
						Kind:       "Deployment",
						Controller: &ctlr,
						Name:       "test-deployment",
					},
				},
			},
		}

		k8sClient := fake.NewClientBuilder().WithRuntimeObjects(replicas).Build()
		ownerType, ownerName, err := pod.GetReplicaSetOwner(context.TODO(), k8sClient, v1.NamespaceDefault, "test")
		Expect(err).ToNot(HaveOccurred())
		Expect(ownerType).To(Equal("deployment"))
		Expect(ownerName).To(Equal("test-deployment"))
		// remove owner
		replicas.OwnerReferences = []metav1.OwnerReference{}
		Expect(k8sClient.Update(context.TODO(), replicas)).To(Succeed())

		ownerType, ownerName, err = pod.GetReplicaSetOwner(context.TODO(), k8sClient, v1.NamespaceDefault, "test")
		Expect(err).ToNot(HaveOccurred())
		Expect(ownerType).To(Equal("replicaset"))
		Expect(ownerName).To(Equal("test"))
	})
})
