package mesh_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/nginxinc/nginx-service-mesh/pkg/apis/mesh"
)

var _ = Describe("Config Manager", func() {
	var mgr *mesh.ConfigManager

	BeforeEach(func() {
		mgr = mesh.NewConfigManager(mesh.FullMeshConfig{})
		Expect(mgr).ToNot(BeNil())
	})

	It("manages agent versions", func() {
		mgr.RecordAgentVersion("agent1", "version1")
		mgr.RecordAgentVersion("agent2", "version1")
		mgr.RecordAgentVersion("agent2", "version1")
		mgr.RecordAgentVersion("agent3", "version2")

		Expect(mgr.AgentVersions).To(HaveLen(2))
		Expect(mgr.AgentVersions["version1"]).To(HaveLen(2))
		Expect(mgr.AgentVersions["version2"]).To(HaveLen(1))

		versions := mgr.GetAgentVersions()
		Expect(versions).To(HaveLen(2))
		Expect(versions).To(ContainElements("version1", "version2"))

		mgr.DismissAgentVersion("agent2")
		Expect(mgr.GetAgentVersions()).To(HaveLen(2))

		mgr.DismissAgentVersion("agent3")
		Expect(mgr.GetAgentVersions()).To(HaveLen(1))
	})

	It("gets the mesh config", func() {
		cm := &v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      mesh.MeshConfigMap,
				Namespace: "nginx-mesh",
			},
			Data: map[string]string{
				mesh.MeshConfigFileName: `{"accessControlMode": "deny"}`,
			},
		}
		fakeClient := fake.NewClientBuilder().WithRuntimeObjects(cm).Build()

		config, err := mesh.GetMeshConfig(context.TODO(), fakeClient, cm.Namespace)
		Expect(err).ToNot(HaveOccurred())
		Expect(config.AccessControlMode).To(Equal(mesh.AccessControlModeDeny))
	})
})
