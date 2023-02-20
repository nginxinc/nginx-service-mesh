// Package commands contains all of the cli commands
package commands // import "github.com/nginxinc/nginx-service-mesh/internal/nginx-meshctl/commands"

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/nginxinc/nginx-service-mesh/pkg/apis/mesh"
)

// GetConfig displays mesh config.
func GetConfig() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Display the NGINX Service Mesh configuration",
		Long:  `Display the NGINX Service Mesh configuration.`,
	}

	cmd.PersistentPreRunE = defaultPreRunFunc()
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx, cancel := context.WithTimeout(context.Background(), meshTimeout)
		defer cancel()

		meshConfig, err := mesh.GetMeshConfig(ctx, initK8sClient.Client(), initK8sClient.Namespace())
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
