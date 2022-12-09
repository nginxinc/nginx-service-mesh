package deploy_test

import (
	"os"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const helmPath = "../../../helm-chart/"

var (
	valuesYaml   []byte
	valuesSchema []byte
	chart        []byte
	_            = BeforeSuite(func() {
		var err error
		valuesYaml, err = os.ReadFile(helmPath + "values.yaml")
		Expect(err).ToNot(HaveOccurred())

		valuesSchema, err = os.ReadFile(helmPath + "values.schema.json")
		Expect(err).ToNot(HaveOccurred())

		chart, err = os.ReadFile(helmPath + "Chart.yaml")
		Expect(err).ToNot(HaveOccurred())
	})
)

func TestDeploy(t *testing.T) {
	t.Parallel()
	RegisterFailHandler(Fail)
	RunSpecs(t, "Deploy Suite")
}
