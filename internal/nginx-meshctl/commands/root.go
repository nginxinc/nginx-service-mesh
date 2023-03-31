// Package commands contains all of the cli commands
package commands // import "github.com/nginxinc/nginx-service-mesh/internal/nginx-meshctl/commands"

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/rest"

	"github.com/nginxinc/nginx-service-mesh/pkg/apis/mesh"
	"github.com/nginxinc/nginx-service-mesh/pkg/health"
	"github.com/nginxinc/nginx-service-mesh/pkg/k8s"
)

// usageTemplate is from https://github.com/spf13/cobra/blob/b04b5bfc50cbb6c036d2115ed55ea1bccdaf82a9/command.go#L456
// but modified slightly to include different wording for the Use line.
const usageTemplate = `Usage:{{if .Runnable}}
  {{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}
  {{.CommandPath}} [command]{{end}}{{if gt (len .Aliases) 0}}

Aliases:
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

Examples:
{{.Example}}{{end}}{{if .HasAvailableSubCommands}}

Available Commands:{{range .Commands}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

Global Flags:
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasHelpSubCommands}}

Additional help topics:{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
  {{rpad .CommandPath .CommandPathPadding}} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableSubCommands}}

Use "{{.CommandPath}} [command] --help"  or "{{.CommandPath}} help [command]" for more information about a command.{{end}}
`

var (
	initK8sClient k8s.Client
	debug         bool
	kubeconfig    = k8s.GetKubeConfig()
	meshNamespace = "nginx-mesh"
	meshTimeout   = 5 * time.Second
)

func getK8sClient(kubeconfig, namespace string) error {
	var err error
	initK8sClient, err = k8s.NewK8SClient(kubeconfig, namespace)
	if err != nil {
		return fmt.Errorf("unable to connect to Kubernetes, please validate your kubeconfig: %w", err)
	}

	return nil
}

func defaultPreRunFunc() func(*cobra.Command, []string) error {
	return func(c *cobra.Command, args []string) error {
		return getK8sClient(kubeconfig, meshNamespace)
	}
}

// Root command prints usage information.
func Root(cmdName string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   cmdName,
		Short: "nginx-meshctl is the CLI utility for the NGINX Service Mesh control plane",
		Long: `nginx-meshctl is the CLI utility for the NGINX Service Mesh control plane. 
Requires a connection to a Kubernetes cluster via a kubeconfig.`,
	}

	cmd.RunE = func(c *cobra.Command, args []string) error {
		return c.Help()
	}

	cmd.PersistentFlags().StringVarP(
		&kubeconfig,
		"kubeconfig",
		"k",
		kubeconfig, "path to kubectl config file")

	cmd.PersistentFlags().StringVarP(
		&meshNamespace,
		"namespace",
		"n",
		meshNamespace, "NGINX Service Mesh control plane namespace")

	cmd.PersistentFlags().DurationVarP(
		&meshTimeout,
		"timeout",
		"t",
		meshTimeout, "timeout when communicating with NGINX Service Mesh")

	cmd.PersistentFlags().BoolVarP(
		&debug,
		"debug",
		"",
		debug, "debug flag")

	cmd.SetUsageTemplate(usageTemplate)
	cmd.SetHelpCommand(Help())

	// hide debug flag from user
	err := cmd.PersistentFlags().MarkHidden("debug")
	if err != nil {
		fmt.Println("Failed to mark debug flag as hidden, error: ", err)
	}

	return cmd
}

// Help defines the output of nginx-meshctl help and nginx-meshctl [command] help.
// Overrides the default help command in order to keep consistent naming.
// The Run function is taken straight from the default help command in cobra:
// https://github.com/spf13/cobra/blob/b04b5bfc50cbb6c036d2115ed55ea1bccdaf82a9/command.go#L1015
func Help() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "help [command]",
		Short: "Help for nginx-meshctl or any command",
		Long: `Help provides help for any command in the application.
Simply type nginx-meshctl help [path to command] for full details.`,
		DisableFlagsInUseLine: true,
		RunE: func(c *cobra.Command, args []string) error {
			var err error
			cmd, _, e := c.Root().Find(args)
			if cmd == nil || e != nil {
				c.Printf("Unknown help topic %#q\n", args)
				err = c.Root().Usage()
			} else {
				cmd.InitDefaultHelpFlag() // make possible 'help' flag to be shown
				err = cmd.Help()
			}

			return fmt.Errorf("error running Help command: %w", err)
		},
	}

	return cmd
}

// NewStatusCmd creates a status command to connect to the mesh.
func NewStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Check connection to NGINX Service Mesh",
		Long:  `Check connection to NGINX Service Mesh.`,
	}

	cmd.PersistentPreRunE = defaultPreRunFunc()
	cmd.RunE = func(c *cobra.Command, args []string) error {
		fmt.Println("Checking NGINX Service Mesh setup....")
		err := health.TestMeshControllerConnection(initK8sClient.Client(), initK8sClient.Namespace(), 1)
		if err == nil {
			fmt.Println("Connection to NGINX Service Mesh was successful.")
		}

		return err
	}

	return cmd
}

// NewVersionCmd create a new cobra to handle version.
func NewVersionCmd(cmdName, version, commit string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Display NGINX Service Mesh version",
		Long: `Display NGINX Service Mesh version.
Will contact the mesh for version and timeout if unable to connect.`,
	}

	cmd.Run = func(c *cobra.Command, args []string) {
		if !strings.HasPrefix(version, "v") {
			version = "v" + version
		}
		// print CLI version
		fmt.Printf("%s - %s", cmdName, version)
		if debug {
			fmt.Printf("+%s", commit)
		}
		fmt.Println()

		printHelp := func(err error) {
			fmt.Println("Unable to get versions for remaining components, make sure:")
			fmt.Printf("- NGINX Service Mesh is installed in \"%s\" namespace\n", meshNamespace)
			fmt.Printf("- Your kubectl config file \"%s\" is valid\n", kubeconfig)
			fmt.Println("- Your Kubernetes context is set to a valid and running cluster, see \"kubectl config get-contexts\"")
			fmt.Println(fmt.Errorf("Error message: %w", err).Error())
		}

		if err := getK8sClient(kubeconfig, meshNamespace); err != nil {
			printHelp(err)

			return
		}

		// Print remaining version if an error wasn't encountered
		versions, err := getComponentVersions(initK8sClient.Config(), meshNamespace, meshTimeout)
		if err != nil {
			printHelp(err)

			return
		}
		fmt.Println(versions)
	}

	return cmd
}

// contact mesh controller to get component versions.
func getComponentVersions(config *rest.Config, namespace string, timeout time.Duration) (string, error) {
	client, server, err := newVersionClient(config, namespace, timeout)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, server, http.NoBody)
	if err != nil {
		return "", fmt.Errorf("error building http request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer func() {
		closeErr := resp.Body.Close()
		if closeErr != nil {
			fmt.Println(closeErr)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("bad status code: %v", resp.StatusCode)
	}

	versions := make(map[string][]string)
	err = json.NewDecoder(resp.Body).Decode(&versions)
	if err != nil {
		return "", fmt.Errorf("error decoding API response: %w", err)
	}

	var str string
	for name, v := range versions {
		str += fmt.Sprintf("%s - v%s\n", name, strings.Join(v, ", v"))
	}

	return str, nil
}

func newVersionClient(config *rest.Config, namespace string, timeout time.Duration) (*http.Client, string, error) {
	gv := schema.GroupVersion{Group: "", Version: "v1"}
	config.GroupVersion = &gv
	config.APIPath = fmt.Sprintf(
		"/api/v1/namespaces/%s/services/nginx-mesh-controller:%d/proxy/version",
		namespace,
		mesh.ControllerVersionPort,
	)
	config.Timeout = timeout
	config.NegotiatedSerializer = serializer.NewCodecFactory(runtime.NewScheme())

	restClient, err := rest.RESTClientFor(config)
	if err != nil {
		return nil, "", fmt.Errorf("error creating RESTClient: %w", err)
	}

	httpClient := &http.Client{
		Timeout: timeout,
	}

	if restClient.Client != nil {
		httpClient = restClient.Client
	}

	server := fmt.Sprintf("%s%s", config.Host, config.APIPath)

	return httpClient, server, nil
}
