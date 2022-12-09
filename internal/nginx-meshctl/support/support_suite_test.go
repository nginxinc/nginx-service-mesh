package support

import (
	"os"
	"path"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var tmpDir string

func TestSupport(t *testing.T) {
	t.Parallel()
	tmpDir = t.TempDir()
	RegisterFailHandler(Fail)
	RunSpecs(t, "Support Suite")
}

func deleteTmpDirContents() {
	dir, err := os.ReadDir(tmpDir)
	Expect(err).ToNot(HaveOccurred())
	for _, d := range dir {
		Expect(os.RemoveAll(path.Join([]string{tmpDir, d.Name()}...))).To(Succeed())
	}
}
