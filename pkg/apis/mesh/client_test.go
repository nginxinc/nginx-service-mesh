package mesh_test

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/nginxinc/nginx-service-mesh/pkg/apis/mesh"
	"github.com/nginxinc/nginx-service-mesh/pkg/k8s/fake"
)

var _ = Describe("Mesh", func() {
	It("initializes a mesh client", func() {
		shouldSkipRelease := false
		config := fake.NewFakeK8s("nginx-mesh", shouldSkipRelease).Config()
		client, err := mesh.NewMeshClient(config, 5*time.Second)
		Expect(err).ToNot(HaveOccurred())
		Expect(client.Server).To(Equal("/apis/nsm.nginx.com/v1alpha1/"))
	})
})
