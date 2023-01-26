package upstreamauthority

import (
	"bytes"
	"encoding/pem"
	"testing"
	"text/template"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestUpstreamAuthority(t *testing.T) {
	t.Parallel()
	RegisterFailHandler(Fail)
	RunSpecs(t, "Upstream Authority Suite")
}

func createCertFile(path string) string {
	buf := &bytes.Buffer{}
	Expect(pem.Encode(buf, &pem.Block{Type: "Test"})).To(Succeed())
	tmp, err := createFile(path, buf.String())
	Expect(err).ToNot(HaveOccurred())

	return tmp
}

// writeUAFile creates a fake UA file for testing.
func writeUAFile(tmpl string, config interface{}) string {
	t := template.Must(template.New("").Parse(tmpl))
	buf := &bytes.Buffer{}
	Expect(t.Execute(buf, config)).To(Succeed())

	ua, err := createFile("test_ua", buf.String())
	Expect(err).ToNot(HaveOccurred())

	return ua
}
