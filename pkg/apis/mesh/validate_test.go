package mesh_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/nginxinc/nginx-service-mesh/pkg/apis/mesh"
	specs "github.com/nginxinc/nginx-service-mesh/pkg/apis/specs/v1alpha1"
)

var _ = Describe("Validate", func() {
	Context("load balancing method", func() {
		var fakeClientBuilder *fake.ClientBuilder
		scheme := runtime.NewScheme()
		Expect(specs.AddToScheme(scheme)).To(Succeed())
		cb := &specs.CircuitBreaker{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-cb",
				Namespace: v1.NamespaceDefault,
			},
		}

		BeforeEach(func() {
			fakeClientBuilder = fake.NewClientBuilder().WithScheme(scheme)
		})

		It("is valid when no circuit breakers exist", func() {
			Expect(mesh.ValidateLBMethod(fakeClientBuilder.Build(), mesh.Random)).To(Succeed())
		})

		It("is valid when not set to random", func() {
			client := fakeClientBuilder.WithRuntimeObjects(cb).Build()
			Expect(mesh.ValidateLBMethod(client, mesh.LeastConn)).To(Succeed())
		})

		It("is invalid when set to random and circuit breakers exist'", func() {
			client := fakeClientBuilder.WithRuntimeObjects(cb).Build()
			Expect(mesh.ValidateLBMethod(client, mesh.Random)).ToNot(Succeed())
		})
	})
})
