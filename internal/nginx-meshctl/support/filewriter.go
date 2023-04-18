package support

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
)

//go:generate counterfeiter -o fake/fake_filewriter.gen.go ./. FileWriter

// FileWriter is an interface for writing regular files and tar files.
// It also wraps many os functions to provide easy mocking for unit tests.
type FileWriter interface {
	// Write writes a file with a name and contents.
	Write(filename, contents string) error
	// WriteFromReader writes a file with a name and contents from a ReadCloser.
	WriteFromReader(filename string, contents io.ReadCloser) error
	// WriteTarFile tars a directory and gives it a name.
	WriteTarFile(directory, filename string) error
	// Mkdir wraps os.Mkdir.
	Mkdir(name string) error
	// MkdirAll wraps os.MkdirAll.
	MkdirAll(name string) error
	// TempDir wraps ioutil.TempDir.
	TempDir(name string) (string, error)
	// OpenFile wraps os.OpenFile
	OpenFile(name string) (*os.File, error)
	// Close wraps os.File.Close.
	Close(file *os.File) error
	// RemoveAll wraps os.RemoveAll.
	RemoveAll(name string) error
}

// Writer is the base implementation of a FileWriter.
type Writer struct{}

// NewWriter returns an instantiated Writer.
func NewWriter() *Writer {
	return &Writer{}
}

// Mkdir wraps os.Mkdir.
func (w *Writer) Mkdir(name string) error {
	return os.Mkdir(name, os.ModePerm)
}

// MkdirAll wraps os.MkdirAll.
func (w *Writer) MkdirAll(name string) error {
	return os.MkdirAll(name, os.ModePerm)
}

// TempDir wraps ioutil.TempDir.
func (w *Writer) TempDir(name string) (string, error) {
	return os.MkdirTemp("", name)
}

// OpenFile wraps os.OpenFile.
func (w *Writer) OpenFile(name string) (*os.File, error) {
	return os.OpenFile(filepath.Clean(name), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
}

// Close wraps os.File.Close.
func (w *Writer) Close(file *os.File) error {
	return file.Close()
}

// RemoveAll wraps os.RemoveAll.
func (w *Writer) RemoveAll(name string) error {
	return os.RemoveAll(name)
}

// Write writes a file with a name and contents.
func (w *Writer) Write(filename, contents string) error {
	file, err := os.OpenFile(filepath.Clean(filename), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("could not create file '%s': %w", filename, err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			log.Fatalf("could not close file: %v", closeErr)
		}
	}()

	if _, err := file.WriteString(contents); err != nil {
		return fmt.Errorf("could not write contents to file '%s': %w", filename, err)
	}

	return nil
}

// WriteFromReader writes a file with a name and contents from a ReadCloser.
func (w *Writer) WriteFromReader(filename string, contents io.ReadCloser) error {
	f, err := os.Create(filename) //nolint:gosec,varnamelen // filepaths generated statically at writeContainerLogs()
	if err != nil {
		return fmt.Errorf("could not create file '%s': %w", filename, err)
	}
	defer func() {
		if closeErr := f.Close(); closeErr != nil {
			log.Fatalf("could not close file: %v", closeErr)
		}
	}()

	if _, err := io.Copy(f, contents); err != nil {
		return fmt.Errorf("could not write contents to file '%s': %w", filename, err)
	}

	return nil
}

// WriteTarFile tars a directory and gives it a name.
func (w *Writer) WriteTarFile(directory, filename string) error {
	filename = filepath.Clean(filename)
	tarFile, err := os.Create(filename) //nolint:gosec // passed in via user input from cluster
	// written to temp dir
	if err != nil {
		return fmt.Errorf("could not create tar file: %w", err)
	}
	defer func() {
		if closeErr := tarFile.Close(); closeErr != nil {
			log.Fatalf("could not close tar file: %v", closeErr)
		}
	}()

	gzipWriter := gzip.NewWriter(tarFile)
	defer func() {
		if closeErr := gzipWriter.Close(); closeErr != nil {
			log.Fatalf("could not close gzip writer: %v", closeErr)
		}
	}()

	tarWriter := tar.NewWriter(gzipWriter)
	defer func() {
		if closeErr := tarWriter.Close(); closeErr != nil {
			log.Fatalf("could not close tar writer: %v", closeErr)
		}
	}()

	// loop through files and archive them
	err = filepath.Walk(directory, func(path string, info os.FileInfo, _ error) error {
		if !info.IsDir() {
			if addErr := addFileToArchive(tarWriter, directory, path); addErr != nil {
				log.Printf("\tcould not add file '%s' to support package: %v\n", path, addErr)
			}
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("could not walk directory tree: %w", err)
	}

	return nil
}

// addFileToArchive reads a file and adds it to the tarball.
func addFileToArchive(tarWriter *tar.Writer, directory, filePath string) error {
	file, err := os.Open(filepath.Clean(filePath))
	if err != nil {
		return fmt.Errorf("could not open file: %w", err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			log.Fatalf("could not close file: %v", closeErr)
		}
	}()

	// get file information
	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("could not get information about file: %w", err)
	}

	// create a tar header from the file information
	header, err := tar.FileInfoHeader(info, info.Name())
	if err != nil {
		return fmt.Errorf("could not create tar header for file: %w", err)
	}

	relPath, err := filepath.Rel(directory, filePath)
	if err != nil {
		return fmt.Errorf("could not get relative path for file: %w", err)
	}
	header.Name = relPath

	if err := tarWriter.WriteHeader(header); err != nil {
		return fmt.Errorf("could not write tar file header: %w", err)
	}

	if _, err := io.Copy(tarWriter, file); err != nil {
		return fmt.Errorf("could not copy file: %w", err)
	}

	return nil
}
