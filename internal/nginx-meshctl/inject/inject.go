// Package inject provides utility functions for sidecar injection.
package inject

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/nginxinc/nginx-service-mesh/pkg/apis/mesh"
)

var errFileDoesNotExist = errors.New("files does not exist")

// ReadFileOrURL returns the body from a local or remote file.
func ReadFileOrURL(filename string) ([]byte, error) {
	switch {
	case filename != "" && !strings.HasPrefix(filename, "http"):
		// Local file
		if exists := fileExists(filename); !exists {
			return nil, fmt.Errorf("%w: %s", errFileDoesNotExist, filename)
		}
		body, err := os.ReadFile(filepath.Clean(filename))
		if err != nil {
			return nil, fmt.Errorf("error reading file: %w", err)
		}

		return body, nil
	case strings.HasPrefix(filename, "http"):
		// Remote file
		fileClient := &http.Client{Timeout: time.Minute}
		req, err := http.NewRequestWithContext(context.TODO(), http.MethodGet, filename, http.NoBody)
		if err != nil {
			return nil, fmt.Errorf("error creating http request: %w", err)
		}
		resp, err := fileClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("error getting file: %w", err)
		}
		// this get around the errcheck lint when we really do not care
		// 'Error return value of `resp.Body.Close` is not checked (errcheck)'
		defer func() {
			_ = resp.Body.Close()
		}()

		err = checkResponse(resp)
		if err != nil {
			return nil, err
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("error reading file contents: %w", err)
		}

		return body, nil
	}

	return nil, nil
}

// CreateFileFromSTDIN creates a temporary file from the data contained in stdin and returns the data and file pointer.
func CreateFileFromSTDIN() ([]byte, *os.File, error) {
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return nil, nil, fmt.Errorf("error reading file contents: %w", err)
	}

	f, err := os.CreateTemp("", "nginx-mesh-api-temp-file.txt")
	if err != nil {
		return nil, nil, fmt.Errorf("error creating temporary file: %w", err)
	}

	return data, f, nil
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}

	return !info.IsDir()
}

var errResponse = errors.New("http request failed")

func checkResponse(resp *http.Response) error {
	if c := resp.StatusCode; c >= 200 && c <= 299 {
		return nil
	}
	req := resp.Request

	return fmt.Errorf("%w: %v %v failed: %v", errResponse, req.Method, req.URL.String(), resp.Status)
}

// BuildMultipartRequestBody builds a multipart/form-data request to be used in the inject request.
func BuildMultipartRequestBody(values map[string][]int, body []byte, filename string) (io.Reader, string, error) {
	buffer := new(bytes.Buffer)
	writer := multipart.NewWriter(buffer)
	fieldWriter, err := writer.CreateFormFile(mesh.FileField, filename)
	if err != nil {
		return nil, "", fmt.Errorf("could not create field writer: %w", err)
	}
	_, _ = fieldWriter.Write(body)
	err = addFormFields(values, writer)
	if err != nil {
		return nil, "", fmt.Errorf("could not add form fields: %w", err)
	}
	contentType := writer.FormDataContentType()
	err = writer.Close()
	if err != nil {
		return nil, "", fmt.Errorf("could not close multipart writer: %w", err)
	}

	return buffer, contentType, nil
}

func addFormFields(values map[string][]int, w *multipart.Writer) error {
	for key, items := range values {
		for _, port := range items {
			err := createTextPlainField(key, strconv.Itoa(port), w)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func createTextPlainField(key, value string, w *multipart.Writer) error {
	h := make(textproto.MIMEHeader)
	h.Set("Content-Type", "text/plain")
	h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"`, key))
	pw, err := w.CreatePart(h)
	if err != nil {
		return fmt.Errorf("could not create part: %w", err)
	}
	_, err = pw.Write([]byte(value))
	if err != nil {
		return fmt.Errorf("could not write value to part: %w", err)
	}

	return nil
}
