package commands

import (
	"context"
	"encoding/json"
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"helm.sh/helm/v3/pkg/chart/loader"
	v1 "k8s.io/api/core/v1"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/nginxinc/nginx-service-mesh/pkg/apis/mesh"
	"github.com/nginxinc/nginx-service-mesh/pkg/k8s"
	"github.com/nginxinc/nginx-service-mesh/pkg/k8s/fake"
)

func createMeshConfigMap(client kubernetes.Interface, namespace string) {
	cfgMap := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: mesh.MeshConfigMap},
		BinaryData: map[string][]byte{
			mesh.MeshConfigFileName: []byte(`{
	"mtls": {
		"caKeyType": "ec-p256",
		"caTTL": "720h",
		"mode": "permissive",
		"svidTTL": "1h"
	},
	"telemetry": {
		"samplerRatio": 0.1,
		"exporters": {
			"otlp": {
				"host": "host",
				"port": 4317
			}
		}
	},
	"sidecarImage": {
		"name": "should-not-change",
		"image": "nginx-mesh-sidecar:old"
	},
	"sidecarInitImage": {
		"name": "should-not-change",
		"image": "nginx-mesh-init:old"
	},
	"isAutoInjectEnabled": true,
	"enabledNamespaces": [],
	"loadBalancingMethod": "round_robin",
	"accessControlMode": "deny",
	"nginxErrorLogLevel": "warn",
	"nginxLogFormat": "default",
	"clientMaxBodySize": "1m"
}`),
		},
	}
	_, err := client.CoreV1().ConfigMaps(namespace).Create(context.TODO(), cfgMap, metav1.CreateOptions{})
	Expect(err).ToNot(HaveOccurred())
}

var _ = Describe("Upgrade", func() {
	var upg *upgrader
	var fakeK8s k8s.Client
	shouldSkipRelease := false

	BeforeEach(func() {
		var err error
		fakeK8s = fake.NewFakeK8s(v1.NamespaceDefault, shouldSkipRelease)
		upg, err = newUpgrader(fakeK8s, true)
		Expect(err).ToNot(HaveOccurred())
	})

	Context("upgrades the mesh", func() {
		It("upgrades the mesh using telemetry config", func() {
			createMeshConfigMap(fakeK8s.ClientSet(), fakeK8s.Namespace())

			Expect(upg.upgrade("x.x.x")).To(Succeed())
			// check some values
			Expect(upg.values.Registry.ImageTag).To(Equal("x.x.x"))
			Expect(upg.values.AccessControlMode).To(Equal("deny"))
			Expect(upg.values.Telemetry.SamplerRatio).To(Equal(float32(0.1)))
			Expect(upg.values.Telemetry.Exporters.OTLP.Host).To(Equal("host"))
		})
	})

	It("upgrades CRDs", func() {
		// create CRD
		crd := &apiextv1.CustomResourceDefinition{
			ObjectMeta: metav1.ObjectMeta{
				Name: "crd-one",
				Labels: map[string]string{
					"labelKey": "labelValue",
				},
			},
		}

		client := fakeK8s.APIExtensionClientSet().ApiextensionsV1().CustomResourceDefinitions()
		createdCRD, err := client.Create(context.TODO(), crd, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(createdCRD.Labels).To(HaveKey("labelKey"))
		Expect(createdCRD.Labels["labelKey"]).To(Equal("labelValue"))

		// update CRD
		crd.Labels["labelKey"] = "newValue"
		crdBytes, err := json.Marshal(crd)
		Expect(err).ToNot(HaveOccurred())

		// add updated CRD to helm file
		upg.files = append(upg.files, &loader.BufferedFile{
			Name: "crds/crd.yaml",
			Data: crdBytes,
		})
		Expect(upg.upgradeCRDs(context.TODO())).To(Succeed())

		updatedCRD, err := client.Get(context.TODO(), "crd-one", metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(updatedCRD.Labels).To(HaveKey("labelKey"))
		Expect(updatedCRD.Labels["labelKey"]).To(Equal("newValue"))
	})

	It("checks for image pull errors", func() {
		stdout := os.Stdout
		defer func() { os.Stdout = stdout }()

		out, err := os.CreateTemp("", "")
		Expect(err).ToNot(HaveOccurred())
		os.Stdout = out

		// create an error
		pod := &v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: fakeK8s.Namespace(),
			},
			Status: v1.PodStatus{
				Conditions: []v1.PodCondition{
					{
						Type:   v1.PodReady,
						Status: v1.ConditionFalse,
					},
				},
			},
		}
		// image pull error event
		event := &v1.Event{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:         fakeK8s.Namespace(),
				CreationTimestamp: metav1.Time{Time: time.Now()},
			},
			Type:    v1.EventTypeWarning,
			Message: "Failed to pull image",
		}

		_, err = fakeK8s.ClientSet().CoreV1().Pods(fakeK8s.Namespace()).Create(context.TODO(), pod, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		_, err = fakeK8s.ClientSet().CoreV1().Events(fakeK8s.Namespace()).Create(context.TODO(), event, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		done := make(chan struct{})
		defer close(done)

		go loopImageErrorCheck(fakeK8s, done)

		Eventually(func() string {
			b, err := os.ReadFile(out.Name())
			Expect(err).ToNot(HaveOccurred())

			return string(b)
		}).Should(ContainSubstring("could not pull images"))

		Expect(os.Remove(out.Name())).To(Succeed())
	})
})
