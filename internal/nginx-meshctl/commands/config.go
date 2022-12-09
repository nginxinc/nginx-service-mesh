// Package commands contains all of the cli commands
package commands // import "github.com/nginxinc/nginx-service-mesh/internal/nginx-meshctl/commands"

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/spf13/cobra"

	"github.com/nginxinc/nginx-service-mesh/pkg/apis/mesh"
)

// GetMeshConfig attempts to get the mesh config from mesh-api.
func GetMeshConfig(meshClient *mesh.Client) (*mesh.MeshConfig, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	res, err := meshClient.GetConfig(ctx)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, mesh.ParseAPIError(res)
	}

	data, err := mesh.ParseGetConfigResponse(res)
	if err != nil {
		return nil, fmt.Errorf("error parsing the response body from the Service Mesh API Server: %w", err)
	}

	return data.JSON200, nil
}

// GetConfig displays mesh config.
func GetConfig() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Display the NGINX Service Mesh configuration",
		Long:  `Display the NGINX Service Mesh configuration.`,
	}

	cmd.PersistentPreRunE = defaultPreRunFunc()
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		meshClient, err := mesh.NewMeshClient(initK8sClient.Config(), meshTimeout)
		if err != nil {
			return fmt.Errorf("failed to get mesh client: %w", err)
		}
		meshConfig, err := GetMeshConfig(meshClient)
		if err != nil {
			return fmt.Errorf("unable to get mesh config: %w", err)
		}

		output, err := json.MarshalIndent(meshConfig, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to format config output: %w", err)
		}

		fmt.Println(string(output))

		return nil
	}

	return cmd
}
