package sidecar_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/nginxinc/nginx-service-mesh/pkg/apis/mesh"
	"github.com/nginxinc/nginx-service-mesh/pkg/sidecar"
)

var _ = Describe("Config", func() {
	It("can build an LBMethod string", func() {
		lbMethod := sidecar.LBMethod{Method: mesh.RoundRobin}
		Expect(lbMethod.String()).To(Equal(""))

		lbMethod = sidecar.LBMethod{Block: sidecar.HTTP, Method: mesh.LeastTime}
		Expect(lbMethod.String()).To(Equal("least_time header;"))

		lbMethod = sidecar.LBMethod{Block: sidecar.Stream, Method: mesh.LeastTime}
		Expect(lbMethod.String()).To(Equal("least_time first_byte;"))

		lbMethod = sidecar.LBMethod{Block: sidecar.HTTP, Method: mesh.RandomTwoLeastTime}
		Expect(lbMethod.String()).To(Equal("random two least_time=header;"))

		lbMethod = sidecar.LBMethod{Block: sidecar.Stream, Method: mesh.RandomTwoLeastTime}
		Expect(lbMethod.String()).To(Equal("random two least_time=first_byte;"))

		lbMethod = sidecar.LBMethod{Block: sidecar.HTTP, Method: mesh.LeastConn}
		Expect(lbMethod.String()).To(Equal("least_conn;"))
	})
})
