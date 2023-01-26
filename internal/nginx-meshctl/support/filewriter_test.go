package support

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Support FileWriter", func() {
	AfterEach(func() {
		deleteTmpDirContents()
	})

	It("makes directories", func() {
		fwriter := NewWriter()
		dir1 := tmpDir + "/dir1"
		Expect(fwriter.Mkdir(dir1)).To(Succeed())
		_, err := os.Stat(dir1)
		Expect(err).ToNot(HaveOccurred())

		nested := tmpDir + "/dir2/dir3"
		Expect(fwriter.MkdirAll(nested)).To(Succeed())
		_, err = os.Stat(nested)
		Expect(err).ToNot(HaveOccurred())
	})

	It("writes a file", func() {
		fwriter := NewWriter()
		filename := tmpDir + "/testfile"
		contents := "contents of testfile"
		Expect(fwriter.Write(filename, contents)).To(Succeed())

		body, err := os.ReadFile(filename)
		Expect(err).ToNot(HaveOccurred())
		Expect(string(body)).To(Equal(contents))
	})

	It("writes a file from a reader", func() {
		fwriter := NewWriter()
		filename := tmpDir + "/testfile"
		contents := "contents of testfile"
		reader := io.NopCloser(bytes.NewReader([]byte(contents)))

		Expect(fwriter.WriteFromReader(filename, reader)).To(Succeed())

		body, err := os.ReadFile(filename)
		Expect(err).ToNot(HaveOccurred())
		Expect(string(body)).To(Equal(contents))
	})

	It("writes a tar file", func() {
		fwriter := NewWriter()
		tarName := tmpDir + "/test.tar.gz"

		// write some files into the tmpDir
		Expect(fwriter.Write(tmpDir+"/file1", "file1")).To(Succeed())
		Expect(fwriter.Write(tmpDir+"/file2", "file2")).To(Succeed())

		Expect(fwriter.WriteTarFile(tmpDir, tarName)).To(Succeed())

		// delete the original files
		Expect(os.Remove(tmpDir + "/file1")).To(Succeed())
		Expect(os.Remove(tmpDir + "/file2")).To(Succeed())

		// validate contents of tarball
		file, err := os.Open(filepath.Clean(tarName))
		Expect(err).ToNot(HaveOccurred())
		archive, err := gzip.NewReader(file)
		Expect(err).ToNot(HaveOccurred())
		tr := tar.NewReader(archive)

		for _, expStr := range []string{"file1", "file2"} {
			f, err := tr.Next()
			Expect(err).ToNot(HaveOccurred())
			Expect(f.Name).To(Equal(expStr))
		}
	})
})
