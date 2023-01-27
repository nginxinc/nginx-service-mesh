package k8s_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	apiextensionv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	fakeApiExt "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/nginxinc/nginx-service-mesh/pkg/k8s"
	fakek8s "github.com/nginxinc/nginx-service-mesh/pkg/k8s/fake"
)

var _ = Describe("k8sclient", func() {
	It("determines if the mesh exists when it hasn't been installed", func() {
		client := &k8s.ClientImpl{
			Clientset:             fake.NewSimpleClientset(),
			APIExtensionClientset: fakeApiExt.NewSimpleClientset(),
		}
		client.HelmInvoker = fakek8s.NewFakeHelmInvoker("nginx-mesh", true, client.Clientset)

		// mesh CRD exists, but not for our mesh
		crd := &apiextensionv1.CustomResourceDefinition{
			ObjectMeta: metav1.ObjectMeta{
				Name: "trafficsplits.split.smi-spec.io",
			},
		}
		_, err := client.APIExtensionClientSet().ApiextensionsV1().
			CustomResourceDefinitions().Create(context.TODO(), crd, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		exists, err := client.MeshExists()
		Expect(exists).To(BeFalse())
		Expect(err).To(HaveOccurred())

		// mesh CRD exists for our mesh
		crd.Labels = map[string]string{
			"app.kubernetes.io/part-of": "nginx-service-mesh",
		}
		_, err = client.APIExtensionClientSet().ApiextensionsV1().
			CustomResourceDefinitions().Update(context.TODO(), crd, metav1.UpdateOptions{})
		Expect(err).ToNot(HaveOccurred())

		exists, err = client.MeshExists()
		Expect(err).To(HaveOccurred())
		Expect(exists).To(BeTrue())
		Expect(err.Error()).To(ContainSubstring("trafficsplits.split.smi-spec.io"))

		// mesh install exists (nullify crd to prevent false positive)
		crd.Labels = nil
		err = client.APIExtensionClientSet().ApiextensionsV1().
			CustomResourceDefinitions().Delete(context.TODO(), crd.Name, metav1.DeleteOptions{})
		Expect(err).ToNot(HaveOccurred())
	})
	It("determines if the mesh exists when it has been installed", func() {
		client := &k8s.ClientImpl{
			Clientset:             fake.NewSimpleClientset(),
			APIExtensionClientset: fakeApiExt.NewSimpleClientset(),
		}
		client.HelmInvoker = fakek8s.NewFakeHelmInvoker("nginx-mesh", false, client.Clientset)

		exists, err := client.MeshExists()
		Expect(exists).To(BeTrue())
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("found existing release"))
	})
})
