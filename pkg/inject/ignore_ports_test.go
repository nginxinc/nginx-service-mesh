package inject_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/nginxinc/nginx-service-mesh/pkg/apis/mesh"
	"github.com/nginxinc/nginx-service-mesh/pkg/inject"
)

var _ = Describe("Ignore Ports", func() {
	Context("Validate ports", func() {
		When("a port is not in range", func() {
			It("returns the correct error", func() {
				ignPorts := inject.IgnorePorts{Incoming: []int{999999}}
				err := ignPorts.Validate()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("not a valid port"))
			})
		})
		When("not all incoming ports are valid", func() {
			It("returns the correct error", func() {
				ignPorts := inject.IgnorePorts{
					Incoming: []int{80, 999999},
				}
				err := ignPorts.Validate()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("incoming ignore ports are not valid"))
			})
		})
		When("not all outgoing ports are valid", func() {
			It("returns the correct error", func() {
				ignPorts := inject.IgnorePorts{
					Incoming: nil,
					Outgoing: []int{80, 999999},
				}
				err := ignPorts.Validate()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("outgoing ignore ports are not valid"))
			})
		})
		When("ignore ports is empty", func() {
			It("does not return an error", func() {
				Expect(inject.IgnorePorts{}.Validate()).To(Succeed(), "an empty ignore ports object should be valid")
			})
		})
		When("ports are valid", func() {
			It("does not return an error", func() {
				ignPorts := inject.IgnorePorts{
					Incoming: []int{1, 5672},
					Outgoing: []int{6300, 9000},
				}
				Expect(ignPorts.Validate()).To(Succeed(), "ports should be valid")
			})
		})
	})

	Context("Get Ignore Ports", func() {
		When("only annotations exist", func() {
			It("returns ignore ports from annotations", func() {
				annotations := map[string]string{
					mesh.IgnoreIncomingPortsAnnotation: "90",
					mesh.IgnoreOutgoingPortsAnnotation: "80",
				}
				ports, err := inject.GetIgnorePorts(annotations, inject.IgnorePorts{})
				Expect(err).ToNot(HaveOccurred())
				Expect(ports.Outgoing).To(ConsistOf(80))
				Expect(ports.Incoming).To(ConsistOf(90))
			})
		})
		When("no annotations exist", func() {
			It("returns ignore ports", func() {
				ignPorts := inject.IgnorePorts{
					Incoming: []int{80},
					Outgoing: []int{81},
				}
				ports, err := inject.GetIgnorePorts(nil, ignPorts)
				Expect(err).ToNot(HaveOccurred())
				Expect(ports).To(Equal(ignPorts))
			})
			It("returns an empty IgnorePorts object if no ports are provided", func() {
				ports, err := inject.GetIgnorePorts(nil, inject.IgnorePorts{})
				Expect(err).ToNot(HaveOccurred())
				Expect(ports.IsEmpty()).To(BeTrue())
			})
		})
		When("both annotations and ports are given", func() {
			It("returns an error if they don't match", func() {
				annotations := map[string]string{
					mesh.IgnoreIncomingPortsAnnotation: "90",
					mesh.IgnoreOutgoingPortsAnnotation: "80",
				}
				ignPorts := inject.IgnorePorts{
					Incoming: []int{80},
					Outgoing: []int{81},
				}
				ports, err := inject.GetIgnorePorts(annotations, ignPorts)
				Expect(err).To(HaveOccurred())
				Expect(ports.IsEmpty()).To(BeTrue())
			})
			It("returns the correct ignore ports object if they match", func() {
				annotations := map[string]string{
					mesh.IgnoreIncomingPortsAnnotation: "90",
					mesh.IgnoreOutgoingPortsAnnotation: "80",
				}
				ignPorts := inject.IgnorePorts{
					Incoming: []int{90},
					Outgoing: []int{80},
				}
				ports, err := inject.GetIgnorePorts(annotations, ignPorts)
				Expect(err).ToNot(HaveOccurred())
				Expect(ports).To(Equal(ignPorts))
			})
		})
	})
})
