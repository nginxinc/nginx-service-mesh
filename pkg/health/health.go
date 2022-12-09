// Package health verifies the health of the mesh.
package health

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"k8s.io/client-go/rest"

	"github.com/nginxinc/nginx-service-mesh/pkg/apis/mesh"
	meshErrors "github.com/nginxinc/nginx-service-mesh/pkg/errors"
)

// TestMeshAPIConnection attempts to connect to the control plane to ensure
// that the mesh-api is available and ready for use.
func TestMeshAPIConnection(config *rest.Config, retries int, timeout time.Duration) error {
	client, err := mesh.NewMeshClient(config, timeout)
	if err != nil {
		return err
	}

	server := strings.TrimSuffix(client.Server, "/")
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, server, http.NoBody)
	if err != nil {
		return fmt.Errorf("error building http request: %w", err)
	}

	for i := 0; i < retries; i++ {
		if i > 0 {
			time.Sleep(time.Second * 5) //nolint:gomnd // not worth another var
		}

		res, requestErr := client.Client.Do(req)
		if requestErr != nil || res.StatusCode != http.StatusOK {
			err = meshErrors.ErrMeshStatus

			continue
		}

		defer func() {
			closeErr := res.Body.Close()
			if closeErr != nil {
				fmt.Println(closeErr)
			}
		}()

		return nil
	}

	return err
}
