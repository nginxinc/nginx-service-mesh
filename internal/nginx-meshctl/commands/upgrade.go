package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"

	"github.com/nginxinc/nginx-service-mesh/pkg/helm"
	"github.com/nginxinc/nginx-service-mesh/pkg/k8s"
)

const longUpgradeMsg = `Upgrade NGINX Service Mesh to the latest version.
This command removes the existing NGINX Service Mesh while preserving user configuration data.
The latest version of NGINX Service Mesh is then deployed using that data.
`

// FIXME(@jbyers19): Remove this warning and its references after v2.0.0 is released.
const NSMv2UpgradeWarning = `
Warning!
  In NGINX Service Mesh v2, sidecars are no longer auto-injected by default and the '--enabled-namespaces' flag has been removed.
  To enable automatic sidecar injection, add the auto-inject label to each namespace where auto-injection is desired:
	'kubectl label namespaces <namespace> injector.nsm.nginx.com/auto-inject=enabled'
  The label must be added BEFORE upgrading the sidecar to prevent it from being removed.`

var upgradeTimeout = 5 * time.Minute

// Upgrade handles a version upgrade of NGINX Service Mesh.
func Upgrade(version string) *cobra.Command {
	var (
		tagOverride    string
		serverOverride string
		dryRun         bool
	)

	cmd := &cobra.Command{
		Use:   "upgrade",
		Short: "Upgrade NGINX Service Mesh",
		Long:  longUpgradeMsg,
	}

	cmd.PersistentPreRunE = defaultPreRunFunc()
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		namespace := initK8sClient.Namespace()
		// Verify mesh install exists
		if _, err := verifyMeshInstall(initK8sClient); err != nil {
			return err
		}
		if !yes {
			msg := fmt.Sprintf("Preparing to upgrade NGINX Service Mesh in namespace \"%s\".\n%s\n", namespace, NSMv2UpgradeWarning)
			if err := ReadYes(msg); err != nil {
				return err
			}
		}
		fmt.Printf("Upgrading NGINX Service Mesh in namespace \"%s\".\n", namespace)

		if tagOverride != "" {
			version = tagOverride
		}

		upgrader, err := newUpgrader(initK8sClient, dryRun)
		if err != nil {
			return fmt.Errorf("error initializing upgrader: %w", err)
		}
		fmt.Printf("Waiting up to %s for components to be ready...", upgradeTimeout.String())

		// start checking for ImagePullErrors in the background; return when we find an error or successfully upgrade
		done := make(chan struct{})
		defer close(done)

		go loopImageErrorCheck(initK8sClient, done)

		if upgradeErr := upgrader.upgrade(version, serverOverride); upgradeErr != nil {
			fmt.Println() // newline to append to the "waiting" statement above

			return upgradeErr
		}

		fmt.Println("Upgrade complete.")
		fmt.Println("To upgrade sidecars, re-roll resources using 'kubectl rollout restart <resource>/<name>'.")
		if yes {
			fmt.Println(NSMv2UpgradeWarning)
		}

		return nil
	}
	cmd.Flags().BoolVarP(
		&yes,
		"yes",
		"y",
		false,
		"answer yes for confirmation of upgrade",
	)
	cmd.PersistentFlags().DurationVarP(
		&upgradeTimeout,
		"timeout",
		"t",
		upgradeTimeout,
		"timeout when waiting for an upgrade to finish",
	)
	cmd.Flags().BoolVar(
		&dryRun,
		"dry-run",
		false,
		`render the upgrade manifest and print to stdout
		Doesn't perform the upgrade`,
	)
	cmd.Flags().StringVar(
		&serverOverride,
		"registry-server",
		serverOverride, `hostname:port (if needed) for registry and path to images
		Affects: `+formatValues(registryServerImages),
	)
	cmd.Flags().StringVar(
		&tagOverride,
		"image-tag",
		tagOverride, `tag used for pulling images from registry
		Affects: `+formatValues(registryServerImages),
	)
	if err := cmd.Flags().MarkHidden("dry-run"); err != nil {
		fmt.Println("error marking flag as hidden: ", err)
	}

	cmd.SetUsageTemplate(upgradeTemplate)

	return cmd
}

func loopImageErrorCheck(k8sClient k8s.Client, done chan struct{}) {
	for {
		select {
		case <-done:
			return
		default:
			// sleep briefly to prevent tight loop
			time.Sleep(100 * time.Millisecond) //nolint:gomnd // not worth adding another global for
			if err := checkImagePullErrors(k8sClient); err != nil {
				fmt.Println()
				fmt.Println(err.Error())

				return
			}
		}
	}
}

// Custom template for Upgrade to fix the timeout usage statement (default template shows parent usage).
var upgradeTemplate = fmt.Sprintf(`Usage:{{if .Runnable}}
  {{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}
  {{.CommandPath}} [command]{{end}}{{if gt (len .Aliases) 0}}

Aliases:
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

Examples:
{{.Example}}{{end}}{{if .HasAvailableSubCommands}}

Available Commands:{{range .Commands}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}
  -t, --timeout duration   timeout when waiting for an upgrade to finish (default 5m0s)

Global Flags:
  -k, --kubeconfig string   path to kubectl config file (default "%s")
  -n, --namespace string    NGINX Service Mesh control plane namespace (default "%s"){{if .HasHelpSubCommands}}

Additional help topics:{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
  {{rpad .CommandPath .CommandPathPadding}} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableSubCommands}}

Use "{{.CommandPath}} [command] --help"  or "{{.CommandPath}} help [command]" for more information about a command.{{end}}
`, k8s.GetKubeConfig(), meshNamespace)

type upgrader struct {
	k8sClient k8s.Client
	values    *helm.Values
	files     []*loader.BufferedFile
	dryRun    bool
}

// newUpgrader returns a new upgrader object.
func newUpgrader(k8sClient k8s.Client, dryRun bool) (*upgrader, error) {
	files, defaultValues, err := helm.GetBufferedFilesAndValues()
	if err != nil {
		return nil, fmt.Errorf("error getting helm files and values: %w", err)
	}
	return &upgrader{
		files:     files,
		values:    defaultValues,
		k8sClient: k8sClient,
		dryRun:    dryRun,
	}, nil
}

// upgrade the mesh by calling "helm upgrade".
func (u *upgrader) upgrade(version string, registry string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := u.upgradeCRDs(ctx); err != nil {
		return fmt.Errorf("error upgrading CRDs: %w", err)
	}

	// initialize values and chart for new release
	vals, err := u.buildValues(ctx, version, registry)
	if err != nil {
		return err
	}

	chart, err := loader.LoadFiles(u.files)
	if err != nil {
		return fmt.Errorf("error loading helm files: %w", err)
	}

	actionConfig, err := u.k8sClient.HelmAction(u.k8sClient.Namespace())
	if err != nil {
		return fmt.Errorf("error initializing helm action: %w", err)
	}

	upgrade := action.NewUpgrade(actionConfig)
	upgrade.Namespace = u.k8sClient.Namespace()
	upgrade.Timeout = upgradeTimeout
	upgrade.Atomic = true
	upgrade.DryRun = u.dryRun

	rel, err := upgrade.RunWithContext(ctx, "nginx-service-mesh", chart, vals)
	if err != nil {
		return fmt.Errorf("error upgrading NGINX Service Mesh: %w", err)
	}

	if u.dryRun {
		fmt.Println(rel.Manifest)

		return nil
	}

	return nil
}

// build the new values as follows:
// - get previous release's deploy-time configuration (helm values)
// - copy on top of the new release's deploy-time configuration
// - get previous release's run-time configuration (mesh-config ConfigMap)
// - copy on top of the new release's deploy-time configuration
// - set new version
// - set new registry server if needed.
func (u *upgrader) buildValues(ctx context.Context, version, registry string) (map[string]interface{}, error) {
	// get the previous deployment configuration
	_, oldValueBytes, err := helm.GetDeployValues(u.k8sClient, "nginx-service-mesh")
	if err != nil {
		return nil, fmt.Errorf("error getting old helm values: %w", err)
	}

	// copy previous values on top of the new default values
	if jsonErr := json.Unmarshal(oldValueBytes, &u.values); jsonErr != nil {
		return nil, fmt.Errorf("error unmarshaling old values into new values: %w", jsonErr)
	}

	// get and save the old runtime mesh config
	client := u.k8sClient.ClientSet().CoreV1().ConfigMaps(u.k8sClient.Namespace())
	meshConfigMap, err := client.Get(ctx, "mesh-config", metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("error getting previous mesh configuration: %w", err)
	}

	var meshConfig previousMeshConfig
	if jsonErr := json.Unmarshal(meshConfigMap.BinaryData["mesh-config.json"], &meshConfig); jsonErr != nil {
		return nil, fmt.Errorf("error unmarshaling previous mesh configuration: %w", jsonErr)
	}

	u.savePreviousConfig(meshConfig)

	// update to new version
	u.values.Registry.ImageTag = version
	if registry != "" {
		u.values.Registry.Server = registry
	}

	vals, err := u.values.ConvertToMap()
	if err != nil {
		return nil, fmt.Errorf("error converting helm values to map: %w", err)
	}

	return vals, nil
}

type getCRDError struct {
	name string
}

func (e *getCRDError) Error() string {
	return fmt.Sprintf("error getting current CRD '%s'", e.name)
}

// updates all custom CRDs.
func (u *upgrader) upgradeCRDs(ctx context.Context) error {
	client := u.k8sClient.APIExtensionClientSet().ApiextensionsV1().CustomResourceDefinitions()
	for _, file := range u.files {
		if !strings.HasPrefix(file.Name, "crds/") {
			continue
		}

		var crd apiextv1.CustomResourceDefinition
		jsonBytes, err := yaml.YAMLToJSON(file.Data)
		if err != nil {
			return fmt.Errorf("error converting yaml to json: %w", err)
		}

		if jsonErr := json.Unmarshal(jsonBytes, &crd); jsonErr != nil {
			return fmt.Errorf("could not unmarshal CRD '%s': %w", file.Name, jsonErr)
		}

		// get current resource version since update requires one
		currentCRD, err := client.Get(ctx, crd.Name, metav1.GetOptions{})
		if err != nil {
			if k8sErrors.IsNotFound(err) {
				// if not found, create it
				if _, err := client.Create(ctx, &crd, metav1.CreateOptions{}); err != nil {
					return fmt.Errorf("error creating CRD '%s': %w", crd.Name, err)
				}

				continue
			}

			return &getCRDError{name: crd.Name}
		}
		crd.ResourceVersion = currentCRD.ResourceVersion

		if _, err := client.Update(ctx, &crd, metav1.UpdateOptions{}); err != nil {
			return fmt.Errorf("error updating CRD '%s': %w", crd.Name, err)
		}
	}

	return nil
}

// Update the new values with the previous runtime mesh configuration.
// This has to be done manually right now because we can't quite unmarshal types.MeshConfig into Values struct.
// FIXME: NSM-3616 should remedy this.
func (u *upgrader) savePreviousConfig(meshConfig previousMeshConfig) {
	u.values.AccessControlMode = meshConfig.AccessControlMode
	u.values.NGINXLBMethod = meshConfig.LoadBalancingMethod
	u.values.NGINXErrorLogLevel = meshConfig.NginxErrorLogLevel
	u.values.NGINXLogFormat = meshConfig.NginxLogFormat
	u.values.PrometheusAddress = meshConfig.PrometheusAddress
	u.values.MTLS.CAKeyType = *meshConfig.Mtls.CaKeyType
	u.values.MTLS.CATTL = *meshConfig.Mtls.CaTTL
	u.values.MTLS.SVIDTTL = *meshConfig.Mtls.SvidTTL
	u.values.MTLS.Mode = *meshConfig.Mtls.Mode
	u.values.ClientMaxBodySize = meshConfig.ClientMaxBodySize

	if meshConfig.Telemetry != (previousTelemetry{}) {
		if u.values.Telemetry == nil {
			u.values.Telemetry = &helm.Telemetry{
				Exporters: &helm.Exporter{
					OTLP: &helm.OTLP{},
				},
			}
		} else if u.values.Telemetry.Exporters == nil {
			u.values.Telemetry.Exporters = &helm.Exporter{OTLP: &helm.OTLP{}}
		}
		u.values.Telemetry.SamplerRatio = *meshConfig.Telemetry.SamplerRatio
		u.values.Telemetry.Exporters.OTLP.Host = meshConfig.Telemetry.Exporters.Otlp.Host
		u.values.Telemetry.Exporters.OTLP.Port = meshConfig.Telemetry.Exporters.Otlp.Port
	}
}

// represents the old mesh config structure from the old configmap.
type previousMeshConfig struct {
	Mtls                previousMtls      `json:"mtls"`
	Telemetry           previousTelemetry `json:"telemetry"`
	AccessControlMode   string            `json:"accessControlMode"`
	ClientMaxBodySize   string            `json:"clientMaxBodySize"`
	LoadBalancingMethod string            `json:"loadBalancingMethod"`
	NginxErrorLogLevel  string            `json:"nginxErrorLogLevel"`
	NginxLogFormat      string            `json:"nginxLogFormat"`
	PrometheusAddress   string            `json:"prometheusAddress"`
}

type previousMtls struct {
	CaKeyType *string `json:"caKeyType,omitempty"`
	CaTTL     *string `json:"caTTL,omitempty"`
	Mode      *string `json:"mode,omitempty"`
	SvidTTL   *string `json:"svidTTL,omitempty"`
}

type previousTelemetry struct {
	Exporters    *exporters `json:"exporters,omitempty"`
	SamplerRatio *float32   `json:"samplerRatio,omitempty"`
}

type exporters struct {
	Otlp *otlp `json:"otlp,omitempty"`
}

type otlp struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}
