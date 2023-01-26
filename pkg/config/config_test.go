package config_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/nginxinc/nginx-service-mesh/pkg/apis/mesh"
	"github.com/nginxinc/nginx-service-mesh/pkg/config"
)

var _ = Describe("Config", func() {
	It("can build an LBMethod string", func() {
		lbMethod := config.LBMethod{Method: mesh.MeshConfigLoadBalancingMethodRoundRobin}
		Expect(lbMethod.String()).To(Equal(""))

		lbMethod = config.LBMethod{Block: config.HTTP, Method: mesh.MeshConfigLoadBalancingMethodLeastTime}
		Expect(lbMethod.String()).To(Equal("least_time header;"))

		lbMethod = config.LBMethod{Block: config.Stream, Method: mesh.MeshConfigLoadBalancingMethodLeastTime}
		Expect(lbMethod.String()).To(Equal("least_time first_byte;"))

		lbMethod = config.LBMethod{Block: config.HTTP, Method: mesh.MeshConfigLoadBalancingMethodRandomTwoLeastTime}
		Expect(lbMethod.String()).To(Equal("random two least_time=header;"))

		lbMethod = config.LBMethod{Block: config.Stream, Method: mesh.MeshConfigLoadBalancingMethodRandomTwoLeastTime}
		Expect(lbMethod.String()).To(Equal("random two least_time=first_byte;"))

		lbMethod = config.LBMethod{Block: config.HTTP, Method: mesh.MeshConfigLoadBalancingMethodLeastConn}
		Expect(lbMethod.String()).To(Equal("least_conn;"))
	})
})
