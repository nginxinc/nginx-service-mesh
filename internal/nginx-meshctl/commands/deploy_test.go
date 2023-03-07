package commands

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"helm.sh/helm/v3/pkg/chart/loader"
	v1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/nginxinc/nginx-service-mesh/pkg/apis/mesh"
	"github.com/nginxinc/nginx-service-mesh/pkg/helm"
	"github.com/nginxinc/nginx-service-mesh/pkg/k8s/fake"
)

var _ = Describe("Deploy", func() {
	shouldSkipRelease := false
	It("checks for image pull errors", func() {
		namespace := "nginx-mesh"
		k8sClient := fake.NewFakeK8s(namespace, shouldSkipRelease)
		// pod that is not ready
		pod := &v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: namespace,
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
				Namespace:         namespace,
				CreationTimestamp: metav1.Time{Time: time.Now()},
			},
			Type:    v1.EventTypeWarning,
			Message: "Failed to pull image",
		}
		_, err := k8sClient.ClientSet().CoreV1().Pods(namespace).Create(context.TODO(), pod, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		_, err = k8sClient.ClientSet().CoreV1().Events(namespace).Create(context.TODO(), event, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		Expect(checkImagePullErrors(k8sClient)).ToNot(Succeed())

		// pod has recovered
		pod.Status.Conditions[0].Status = v1.ConditionTrue
		_, err = k8sClient.ClientSet().CoreV1().Pods(namespace).Update(context.TODO(), pod, metav1.UpdateOptions{})
		Expect(err).ToNot(HaveOccurred())

		Expect(checkImagePullErrors(k8sClient)).To(Succeed())
	})

	It("gets helm files and values", func() {
		files, values, err := helm.GetBufferedFilesAndValues()
		Expect(err).ToNot(HaveOccurred())

		// verify that some values are set
		Expect(values.Environment).To(Equal(string(mesh.Kubernetes)))
		Expect(values.NGINXLBMethod).To(Equal(mesh.LeastTime))
		Expect(values.MTLS.Mode).To(Equal(mesh.MtlsModePermissive))
		Expect(values.MTLS.CAKeyType).To(Equal("ec-p256"))

		Expect(len(files)).To(BeNumerically(">", 0))
	})

	It("sets persistent storage", func() {
		namespace := v1.NamespaceDefault
		k8sClient := fake.NewFakeK8s(namespace, shouldSkipRelease)
		values := &helm.Values{
			MTLS: helm.MTLS{
				PersistentStorage: "auto",
			},
		}

		// no storage class
		Expect(setPersistentStorage(k8sClient.ClientSet(), values)).To(Succeed())
		Expect(values.MTLS.PersistentStorage).To(Equal("off"))

		// persistentStorage on but no storage class
		values.MTLS.PersistentStorage = "on"
		Expect(setPersistentStorage(k8sClient.ClientSet(), values)).ToNot(Succeed())

		// storage class exists
		storageGroup := &storagev1.StorageClass{
			ObjectMeta: metav1.ObjectMeta{
				Name: "sg",
				Annotations: map[string]string{
					"storageclass.kubernetes.io/is-default-class": "true",
				},
			},
		}
		_, err := k8sClient.ClientSet().StorageV1().StorageClasses().Create(context.TODO(), storageGroup, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(setPersistentStorage(k8sClient.ClientSet(), values)).To(Succeed())
	})

	It("substitutes development images", func() {
		images := customImages{
			mesh.MeshController: {
				file:  "templates/nginx-mesh-controller.yaml",
				value: "mesh-controller-image",
			},
			mesh.MeshSidecar: {
				file:  "configs/meshconfig.conf",
				value: "sidecar-image",
			},
		}
		files := []*loader.BufferedFile{
			{
				Name: "templates/nginx-mesh-controller.yaml",
				Data: []byte("{{ .Values.registry.server }}/nginx-mesh-controller:{{ .Values.registry.imageTag }}"),
			},
			{
				Name: "configs/meshconfig.conf",
				Data: []byte("{{ printf \"%s/nginx-mesh-sidecar:%s\" .Values.registry.server .Values.registry.imageTag | quote }}"),
			},
			{
				Name: "configs/nginx-mesh-metrics.yaml",
				Data: []byte("{{ .Values.registry.server }}/nginx-mesh-metrics:{{ .Values.registry.imageTag }}"),
			},
		}

		subImages(images, files)
<<<<<<< HEAD
		Expect(string(files[0].Data)).To(Equal("\"mesh-api-image\""))
		Expect(string(files[1].Data)).To(Equal("\"sidecar-image\""))
=======
		Expect(string(files[0].Data)).To(Equal("mesh-controller-image"))
		Expect(string(files[1].Data)).To(Equal("sidecar-image"))
>>>>>>> 9f0d812 (rename mesh api -> mesh controller)
		Expect(string(files[2].Data)).To(Equal("{{ .Values.registry.server }}/nginx-mesh-metrics:{{ .Values.registry.imageTag }}"))
	})
	It("can validate an exporter config", func() {
		missingType := map[string]string{"host": "some-host", "port": "4000"}
		err := validateExporterConfig(missingType)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("missing type"))

		unsupportedType := map[string]string{"host": "some-host", "port": "4000", "type": "unsupported"}
		err = validateExporterConfig(unsupportedType)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("unsupported type"))

		missingHost := map[string]string{"port": "4000", "type": "otlp"}
		err = validateExporterConfig(missingHost)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("missing host"))

		missingPort := map[string]string{"host": "some-host", "type": "otlp"}
		err = validateExporterConfig(missingPort)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("missing port"))

		valid := map[string]string{"host": "some-host", "port": "4000", "type": "otlp"}
		Expect(validateExporterConfig(valid)).To(Succeed())

		// extra fields are ignored
		extraFields := map[string]string{"host": "some-host", "port": "4000", "type": "otlp", "extra": "extra"}
		Expect(validateExporterConfig(extraFields)).To(Succeed())
	})
	It("can convert exporter config into map", func() {
		badFormat := "not,a,key,val,pair"
		_, err := exporterStringToMap(badFormat)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("must be formatted as key=value"))

		// duplicate keys are not allowed
		duplicateKeys := "key1=val1,key1=val2"
		_, err = exporterStringToMap(duplicateKeys)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("duplicate key"))

		// valid k/v pairs
		valid := "type=otlp,host=host,port=4000"
		output, err := exporterStringToMap(valid)
		Expect(err).ToNot(HaveOccurred())
		Expect(output).To(HaveKeyWithValue("type", "otlp"))
		Expect(output).To(HaveKeyWithValue("host", "host"))
		Expect(output).To(HaveKeyWithValue("port", "4000"))
	})
	It("can convert telemetry options to helm values", func() {
		values := &helm.Values{}
		valid := map[string]string{"type": "otlp", "host": "host", "port": "4000"}
		sampleRatio := float32(0.5)
		err := convertTelemetryOpsToHelmValues(valid, sampleRatio, values)
		Expect(err).ToNot(HaveOccurred())
		Expect(values.Telemetry).ToNot(BeNil())
		Expect(values.Telemetry.SamplerRatio).To(Equal(sampleRatio))
		Expect(values.Telemetry.Exporters).ToNot(BeNil())
		Expect(values.Telemetry.Exporters.OTLP).ToNot(BeNil())
		Expect(values.Telemetry.Exporters.OTLP.Host).To(Equal("host"))
		Expect(values.Telemetry.Exporters.OTLP.Port).To(Equal(4000))

		invalidPort := map[string]string{"type": "otlp", "host": "host", "port": "not-a-port"}
		err = convertTelemetryOpsToHelmValues(invalidPort, sampleRatio, values)
		Expect(err).To(HaveOccurred())
	})
	Context("set tracing and telemetry values", func() {
		It("sets telemetry fields correctly", func() {
			tel := telemetryConfig{
				exporters:    []string{"type=otlp,host=host,port=4000"},
				samplerRatio: 0.1,
			}
			values := &helm.Values{}
			Expect(setTelemetryValues(tel, values)).To(Succeed())
			Expect(values.Telemetry).ToNot(BeNil())
			Expect(values.Telemetry.SamplerRatio).To(Equal(tel.samplerRatio))
			Expect(values.Telemetry.Exporters).ToNot(BeNil())
			Expect(values.Telemetry.Exporters.OTLP).ToNot(BeNil())
			Expect(values.Telemetry.Exporters.OTLP.Host).To(Equal("host"))
			Expect(values.Telemetry.Exporters.OTLP.Port).To(Equal(4000))
		})
		It("returns an error if telemetry exporters list is greater than 1", func() {
			tel := telemetryConfig{
				exporters:    []string{"type=otlp,host=host,port=4000", "some-other-config"},
				samplerRatio: 0.1,
			}
			err := setTelemetryValues(tel, &helm.Values{})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("only one telemetry exporter may be configured"))
		})
	})
	Context("can validate input", func() {
		var (
			values   *helm.Values
			user     = "user"
			password = "pw"
		)
		JustBeforeEach(func() {
			values = &helm.Values{}
		})

		runTest := func(v *helm.Values, fileData string, errSubString string) {
			err := validateInput(v, fileData)
			if errSubString != "" {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(errSubString))
			} else {
				Expect(err).ToNot(HaveOccurred())
			}
		}

		It("makes sure registry password is set if username is provided", func() {
			values.Registry.Username = user
			runTest(values, "", "both --registry-username and --registry-password must be set")
		})
		It("makes sure registry username is set if password is provided", func() {
			values.Registry.Password = password
			runTest(values, "", "both --registry-username and --registry-password must be set")
		})
		It("makes sure registry username/password is not set if registry key file is provided", func() {
			values.Registry.Username = user
			values.Registry.Password = password
			runTest(values, "key-file", "cannot set both --registry-key and --registry-username/--registry-password")
		})
		It("passes if everything is valid", func() {
			values.Registry.Username = user
			values.Registry.Password = password
			runTest(values, "", "")
		})
	})

	It("determines if a pod is ready", func() {
		notReady := v1.Pod{
			Status: v1.PodStatus{
				Conditions: []v1.PodCondition{
					{
						Type:   v1.PodInitialized,
						Status: v1.ConditionFalse,
					},
				},
			},
		}
		ready := v1.Pod{
			Status: v1.PodStatus{
				Conditions: []v1.PodCondition{
					{
						Type:   v1.PodReady,
						Status: v1.ConditionTrue,
					},
				},
			},
		}

		Expect(isPodReady(notReady)).To(BeFalse())
		Expect(isPodReady(ready)).To(BeTrue())
	})
})
