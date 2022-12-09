package helm_test

import (
	"encoding/json"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/nginxinc/nginx-service-mesh/pkg/helm"
	"github.com/nginxinc/nginx-service-mesh/pkg/k8s/fake"
)

var _ = Describe("Helm", func() {
	It("gets deploy values", func() {
		shouldSkipRelease := false
		client := fake.NewFakeK8s("nginx-mesh", shouldSkipRelease)

		values, valueBytes, err := helm.GetDeployValues(client, "nginx-service-mesh")
		Expect(err).ToNot(HaveOccurred())
		// verify some default values
		Expect(values.AccessControlMode).To(Equal("allow"))
		Expect(values.NGINXErrorLogLevel).To(Equal("warn"))
		Expect(values.MTLS.TrustDomain).To(Equal("example.org"))

		var expVals helm.Values
		Expect(json.Unmarshal(valueBytes, &expVals)).To(Succeed())
		Expect(expVals).To(Equal(*values))
	})
})
