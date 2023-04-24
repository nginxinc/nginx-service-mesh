package inject_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/nginxinc/nginx-service-mesh/pkg/apis/mesh"
	"github.com/nginxinc/nginx-service-mesh/pkg/inject"
	"github.com/nginxinc/nginx-service-mesh/pkg/pod"
)

var _ = Describe("Config", func() {
	It("validates container ports", func() {
		type testCase struct {
			Desc       string
			Containers []v1.Container
			ExpPorts   []string
			ExpErr     bool
		}
		testCases := []testCase{
			{
				Desc: "one container and all valid ports",
				Containers: []v1.Container{
					{
						Ports: []v1.ContainerPort{
							{Protocol: v1.ProtocolTCP, ContainerPort: 80},
							{Protocol: v1.ProtocolTCP, ContainerPort: 443},
							{Protocol: v1.ProtocolUDP, ContainerPort: 53},
						},
					},
				},
				ExpPorts: []string{"80", "443"},
				ExpErr:   false,
			},
			{
				Desc: "multiple containers and all valid ports",
				Containers: []v1.Container{
					{
						Ports: []v1.ContainerPort{
							{Protocol: v1.ProtocolTCP, ContainerPort: 80},
							{Protocol: v1.ProtocolTCP, ContainerPort: 8080},
							{Protocol: v1.ProtocolUDP, ContainerPort: 53},
						},
					},
					{
						Ports: []v1.ContainerPort{
							{Protocol: v1.ProtocolTCP, ContainerPort: 443},
						},
					},
				},
				ExpPorts: []string{"80", "8080", "443"},
				ExpErr:   false,
			},
			{
				Desc: "one container and protocol is invalid",
				Containers: []v1.Container{
					{
						Ports: []v1.ContainerPort{
							{Protocol: v1.ProtocolTCP, ContainerPort: 80},
							{Protocol: v1.ProtocolSCTP, ContainerPort: 132},
						},
					},
				},
				ExpPorts: nil,
				ExpErr:   true,
			},
			{
				Desc: "multiple containers and protocol is invalid",
				Containers: []v1.Container{
					{
						Ports: []v1.ContainerPort{
							{Protocol: v1.ProtocolTCP, ContainerPort: 80},
						},
					},
					{
						Ports: []v1.ContainerPort{
							{Protocol: v1.ProtocolSCTP, ContainerPort: 132},
						},
					},
				},
				ExpPorts: nil,
				ExpErr:   true,
			},
			{
				Desc: "one containers and duplicate ports",
				Containers: []v1.Container{
					{
						Ports: []v1.ContainerPort{
							{ContainerPort: 80},
							{ContainerPort: 80},
						},
					},
				},
				ExpPorts: nil,
				ExpErr:   true,
			},
			{
				Desc: "multiple containers and duplicate ports",
				Containers: []v1.Container{
					{
						Ports: []v1.ContainerPort{
							{ContainerPort: 80},
						},
					},
					{
						Ports: []v1.ContainerPort{
							{ContainerPort: 80},
						},
					},
				},
				ExpPorts: nil,
				ExpErr:   true,
			},
		}
		for _, tcase := range testCases {
			ports, err := inject.ValidatePorts(tcase.Containers)
			if tcase.ExpErr {
				Expect(err).To(HaveOccurred(), "TestCase: %s", tcase.Desc)
			} else {
				Expect(err).ToNot(HaveOccurred(), "TestCase: %s", tcase.Desc)
				for _, key := range tcase.ExpPorts {
					Expect(ports).To(HaveKey(key), "TestCase: %s", tcase.Desc)
				}
			}
		}
	})

	Context("Get probes", func() {
		It("returns updated probe objects", func() {
			containers := []v1.Container{
				{
					Name: "container1",
					Ports: []v1.ContainerPort{
						{Name: "port1", ContainerPort: 8080},
					},
					LivenessProbe: &v1.Probe{
						ProbeHandler: v1.ProbeHandler{
							HTTPGet: &v1.HTTPGetAction{
								Path: "/live",
								Port: intstr.FromString("port1"),
							},
						},
					},
					ReadinessProbe: &v1.Probe{
						ProbeHandler: v1.ProbeHandler{
							HTTPGet: &v1.HTTPGetAction{
								Scheme: v1.URISchemeHTTPS,
								Path:   "/ready",
								Port:   intstr.FromInt(8081),
							},
						},
					},
					StartupProbe: &v1.Probe{
						ProbeHandler: v1.ProbeHandler{
							HTTPGet: &v1.HTTPGetAction{
								Scheme: v1.URISchemeHTTP,
								Path:   "/start",
								Port:   intstr.FromInt(8082),
							},
						},
					},
				},
			}
			probes := inject.GetProbes(containers, 8895, 8896)
			Expect(probes).To(HaveLen(3))
			Expect(probes[0].OrigHTTPGet.Port).To(Equal(intstr.FromInt(8080)))
			Expect(probes[0].OrigHTTPGet.Scheme).To(Equal(v1.URISchemeHTTP))
			Expect(int(probes[0].HTTPGet.Port.IntVal)).To(Equal(8895))
			Expect(probes[1].OrigHTTPGet.Scheme).To(Equal(v1.URISchemeHTTPS))
			Expect(int(probes[1].HTTPGet.Port.IntVal)).To(Equal(8896))
			Expect(probes[2].OrigHTTPGet.Scheme).To(Equal(v1.URISchemeHTTP))
			Expect(int(probes[2].HTTPGet.Port.IntVal)).To(Equal(8895))

			for _, p := range probes {
				Expect(p.HTTPGet.Path).To(ContainSubstring(inject.RedirectPath + "container1"))
				Expect(p.OrigHTTPGet.Path).ToNot(ContainSubstring(inject.RedirectPath + "container1"))
				Expect(int(p.OrigHTTPGet.Port.IntVal)).ToNot(Equal(8895))
				Expect(int(p.OrigHTTPGet.Port.IntVal)).ToNot(Equal(8896))
			}
		})
	})

	Context("validates the mtls annotation", func() {
		Specify("if annotation is set to off and globally set to permissive", func() {
			annotations := map[string]string{mesh.MTLSModeAnnotation: mesh.MtlsModeOff}
			val, err := pod.GetMTLSModeAnnotation(annotations)
			Expect(err).ToNot(HaveOccurred())
			Expect(val).To(Equal(mesh.MtlsModeOff))
			Expect(inject.ValidateMTLSAnnotation(val, mesh.MtlsModePermissive)).To(Succeed())
		})
		Specify("if annotation is set to strict but globally set to permissive", func() {
			annotations := map[string]string{mesh.MTLSModeAnnotation: mesh.MtlsModeStrict}
			val, err := pod.GetMTLSModeAnnotation(annotations)
			Expect(err).ToNot(HaveOccurred())
			Expect(val).To(Equal(mesh.MtlsModeStrict))
			Expect(inject.ValidateMTLSAnnotation(val, mesh.MtlsModePermissive)).To(Succeed())
		})
		Specify("if annotation is set to permissive but globally off", func() {
			annotations := map[string]string{mesh.MTLSModeAnnotation: mesh.MtlsModePermissive}
			val, err := pod.GetMTLSModeAnnotation(annotations)
			Expect(err).ToNot(HaveOccurred())
			Expect(val).To(Equal(mesh.MtlsModePermissive))
			Expect(inject.ValidateMTLSAnnotation(val, mesh.MtlsModeOff)).To(Succeed())
		})
		Specify("if annotation is set to strict but globally off", func() {
			annotations := map[string]string{mesh.MTLSModeAnnotation: mesh.MtlsModeStrict}
			val, err := pod.GetMTLSModeAnnotation(annotations)
			Expect(err).ToNot(HaveOccurred())
			Expect(val).To(Equal(mesh.MtlsModeStrict))
			Expect(inject.ValidateMTLSAnnotation(val, mesh.MtlsModeOff)).To(Succeed())
		})
	})
})
