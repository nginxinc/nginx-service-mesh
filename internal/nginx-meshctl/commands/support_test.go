package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	fakeSupport "github.com/nginxinc/nginx-service-mesh/internal/nginx-meshctl/support/fake"
	"github.com/nginxinc/nginx-service-mesh/pkg/k8s/fake"
)

var _ = Describe("Support Command", func() {
	version := "0.0.0"
	collectSidecarLogs := true
	output, err := os.Getwd()
	Expect(err).ToNot(HaveOccurred())

	It("fails if mesh cannot be found", func() {
		shouldSkipRelease := true
		// mesh doesn't exist at all
		k8sClient := fake.NewFakeK8s("nginx-mesh", shouldSkipRelease)
		err = generateBundle(k8sClient, &fakeSupport.FakeFileWriter{}, output, version, collectSidecarLogs)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("could not verify mesh: namespace 'nginx-mesh' not found"))

		// namespace exists, but mesh does not
		namespace := &v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "nginx-mesh",
			},
		}
		_, err = k8sClient.ClientSet().CoreV1().Namespaces().Create(context.TODO(), namespace, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		err = generateBundle(k8sClient, &fakeSupport.FakeFileWriter{}, output, version, collectSidecarLogs)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("NGINX Service Mesh installation not found"))
	})

	It("calls expected writer function", func() {
		shouldSkipRelease := false
		k8sClient := fake.NewFakeK8s("nginx-mesh", shouldSkipRelease)
		namespace := &v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "nginx-mesh",
				Labels: map[string]string{
					"app.kubernetes.io/part-of": "nginx-service-mesh",
				},
			},
		}
		_, err := k8sClient.ClientSet().CoreV1().Namespaces().Create(context.TODO(), namespace, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		mw := &fakeSupport.FakeFileWriter{}
		err = generateBundle(k8sClient, mw, output, version, collectSidecarLogs)
		Expect(err).ToNot(HaveOccurred())
		Expect(mw.WriteTarFileCallCount()).To(Equal(1))
	})

	It("can write the mesh version", func() {
		shouldSkipRelease := false
		k8sClient := fake.NewFakeK8s("nginx-mesh", shouldSkipRelease)
		server := ghttp.NewServer()
		defer server.Close()

		k8sClient.Config().Host = server.URL()

		var buf bytes.Buffer
		enc := json.NewEncoder(&buf)
		Expect(enc.Encode(map[string][]string{"app1": {"1.0.0"}})).To(Succeed())
		server.AppendHandlers(
			ghttp.RespondWith(http.StatusOK, buf.Bytes()),
		)

		writer := &fakeSupport.FakeFileWriter{}
		Expect(writeMeshVersion(k8sClient, writer, "", version)).To(Succeed())
		_, str := writer.WriteArgsForCall(0)
		Expect(str).To(Equal("nginx-meshctl - v0.0.0\napp1 - v1.0.0\n"))
	})
})
