// Package commands contains all of the cli commands
package commands // import "github.com/nginxinc/nginx-service-mesh/internal/nginx-meshctl/commands"

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"helm.sh/helm/v3/pkg/chart/loader"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/nginxinc/nginx-service-mesh/internal/nginx-meshctl/deploy"
	"github.com/nginxinc/nginx-service-mesh/internal/nginx-meshctl/upstreamauthority"
	"github.com/nginxinc/nginx-service-mesh/pkg/apis/mesh"
	meshErrors "github.com/nginxinc/nginx-service-mesh/pkg/errors"
	"github.com/nginxinc/nginx-service-mesh/pkg/health"
	"github.com/nginxinc/nginx-service-mesh/pkg/helm"
	"github.com/nginxinc/nginx-service-mesh/pkg/k8s"
)

const (
	meshAPIConnectionFailedInstructions = `Connection to NGINX Service Mesh API Server failed.
	Check the logs of the nginx-mesh-api container in namespace %s for more details.`

	deployMeshAPIRetries = 60

	longDeploy = `Deploy NGINX Service Mesh into your Kubernetes cluster.
This command installs the following resources into your Kubernetes cluster by default:
- NGINX Mesh API: The Control Plane for the Service Mesh.
- NGINX Metrics API: SMI-formatted metrics.
- SPIRE: mTLS service-to-service communication.
- NATS: Message bus.

`
	exampleDeploy = `
  Most of the examples below are abbreviated for convenience. The '...' in these 
  examples represents the image references. Be sure to include the image references 
  when running the deploy command.  
	
  - Deploy the latest version of NGINX Service Mesh, using default values, from your container registry:
		
      nginx-meshctl deploy --registry-server "registry:5000"
	
  - Deploy the Service Mesh in namespace "my-namespace":

      nginx-meshctl deploy ... --namespace my-namespace

  - Deploy the Service Mesh with mTLS and automatic injection turned off:

      nginx-meshctl deploy ... --mtls-mode off --disable-auto-inject

  - Deploy the Service Mesh and only allow automatic injection in namespace "my-namespace":

      nginx-meshctl deploy ... --disable-auto-inject --enabled-namespaces="my-namespace"

  - Deploy the Service Mesh and disallow automatic injection in namespaces "my-namespace-1" and "my-namespace-2" (deprecated)

      nginx-meshctl deploy ... --disabled-namespaces="my-namespace-1,my-namespace-2"

  - Deploy the Service Mesh and enable telemetry traces to be exported to your OTLP gRPC collector running in your Kubernetes cluster:
      
      nginx-meshctl deploy ... --telemetry-exporters "type=otlp,host=otel-collector.my-namespace.svc.cluster.local,port=4317"

  - Deploy the Service Mesh with a tracing server in your Kubernetes cluster (deprecated):

      nginx-meshctl deploy ... --tracing-backend="jaeger" --tracing-address="my-jaeger-server.my-namespace.svc.cluster.local:6831"

  - Deploy the Service Mesh with upstream certificates and keys for mTLS:

      nginx-meshctl deploy ... --mtls-upstream-ca-conf="disk.yaml"
`
)

const meshMetrics = "nginx-mesh-metrics"

// maps image names to their associated helm files for substitution if specified.
type customImages map[string]struct {
	file  string
	value string
}

type telemetryConfig struct {
	exporters    []string
	samplerRatio float32
}

// Deploy deploys the mesh control plane.
func Deploy() *cobra.Command {
	var (
		pullPolicy            bool
		cleanupOnError        bool
		dryRun                bool
		registryKeyFile       string
		mtlsUpstreamFile      string
		imageMeshAPI          string
		imageMetricsAPI       string
		imageMeshCertReloader string
		imageSidecar          string
		imageSidecarInit      string
		tracing               helm.Tracing
		telemetry             telemetryConfig
	)

	files, defaultValues, err := helm.GetBufferedFilesAndValues()
	if err != nil {
		log.Fatal(err)
	}
	values := &helm.Values{}

	cmd := &cobra.Command{
		Use:     "deploy",
		Short:   "Deploys NGINX Service Mesh into your Kubernetes cluster",
		Long:    longDeploy,
		Example: exampleDeploy,
	}

	cmd.PersistentPreRunE = defaultPreRunFunc()
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		if err = setTracingAndTelemetryValues(telemetry, tracing, values); err != nil {
			return err
		}
		// custom input validation for complex fields
		if err = validateInput(values, registryKeyFile); err != nil {
			return err
		}
		// Set pullPolicy to Always if specified
		values.Registry.ImagePullPolicy = defaultValues.Registry.ImagePullPolicy
		if pullPolicy {
			values.Registry.ImagePullPolicy = "Always"
		}

		if err = setPersistentStorage(initK8sClient.ClientSet(), values); err != nil {
			return err
		}

		if registryKeyFile != "" {
			var registryKey []byte
			registryKey, err = os.ReadFile(registryKeyFile) //nolint:gosec // path comes from CLI input
			if err != nil {
				return fmt.Errorf("error reading registry key file: %w", err)
			}
			values.Registry.Key = string(registryKey)
		}

		if mtlsUpstreamFile != "" {
			var upstreamAuthority *helm.UpstreamAuthority
			upstreamAuthority, err = upstreamauthority.GetUpstreamAuthorityValues(mtlsUpstreamFile)
			if err != nil {
				return fmt.Errorf("error building upstream authority config: %w", err)
			}
			values.MTLS.UpstreamAuthority = *upstreamAuthority
		}

		// substitute custom images if specified (dev mode)
		images := customImages{
			mesh.MeshAPI: {
				file:  "templates/nginx-mesh-api.yaml",
				value: imageMeshAPI,
			},
			meshMetrics: {
				file:  "templates/nginx-mesh-metrics.yaml",
				value: imageMetricsAPI,
			},
			mesh.MeshCertReloader: {
				file:  "templates/nats.yaml",
				value: imageMeshCertReloader,
			},
			mesh.MeshSidecar: {
				file:  "configs/mesh-config.conf",
				value: imageSidecar,
			},
			mesh.MeshSidecarInit: {
				file:  "configs/mesh-config.conf",
				value: imageSidecarInit,
			},
		}
		subImages(images, files)
		deployer := deploy.NewDeployer(files, values, initK8sClient, dryRun)

		return startDeploy(initK8sClient, deployer, cleanupOnError)
	}
	cmd.Flags().BoolVarP(
		&pullPolicy,
		"pull-always",
		"p",
		false,
		"set imagePullPolicy to be 'always'",
	)
	err = cmd.Flags().MarkHidden("pull-always")
	if err != nil {
		fmt.Println("error marking flag as hidden: ", err)
	}
	cmd.Flags().BoolVar(
		&dryRun,
		"dry-run",
		false,
		`render the manifest and print to stdout
		Doesn't deploy anything`,
	)
	err = cmd.Flags().MarkHidden("dry-run")
	if err != nil {
		fmt.Println("error marking flag as hidden: ", err)
	}
	cmd.Flags().BoolVar(
		&values.DisableAutoInjection,
		"disable-auto-inject",
		defaultValues.DisableAutoInjection,
		`disable automatic sidecar injection upon resource creation
		Use the --enabled-namespaces flag to enable automatic injection in select namespaces`)
	cmd.Flags().StringSliceVar(
		&values.AutoInjection.DisabledNamespaces,
		"disabled-namespaces",
		defaultValues.AutoInjection.DisabledNamespaces,
		`disable automatic sidecar injection for specific namespaces
		Cannot be used with --disable-auto-inject`,
	)
	err = cmd.Flags().MarkDeprecated("disabled-namespaces",
		"and will be removed in a future release. Allow listing patterns are recommended (enabled-namespaces or namespace labeling) "+
			"is the preferred way to configure injection.")
	if err != nil {
		fmt.Println("error marking flag as deprecated: ", err)
	}
	cmd.Flags().StringSliceVar(
		&values.EnabledNamespaces,
		"enabled-namespaces",
		defaultValues.EnabledNamespaces,
		`enable automatic sidecar injection for specific namespaces
		Must be used with --disable-auto-inject`,
	)
	cmd.Flags().Float32Var(
		&tracing.SampleRate,
		"sample-rate",
		defaultValues.Tracing.SampleRate,
		`the sample rate to use for tracing
		Float between 0 and 1`,
	)
	err = cmd.Flags().MarkDeprecated("sample-rate",
		"and will be removed in a future release. OpenTelemetry (--sampler-ratio and --telemetry-exporters) "+
			"is the preferred way to configure tracing.")
	if err != nil {
		fmt.Println("error marking flag as deprecated: ", err)
	}

	cmd.Flags().StringVar(
		&tracing.Backend,
		"tracing-backend",
		defaultValues.Tracing.Backend,
		`the tracing backend that you want to use
		Valid values: `+formatValues(mesh.TracingBackends),
	)
	err = cmd.Flags().MarkDeprecated("tracing-backend",
		"and will be removed in a future release. OpenTelemetry (--telemetry-exporters) "+
			"is the preferred way to configure tracing.")
	if err != nil {
		fmt.Println("error marking flag as deprecated: ", err)
	}

	cmd.Flags().StringVar(
		&tracing.Address,
		"tracing-address",
		defaultValues.Tracing.Address,
		`the address of a tracing server deployed in your Kubernetes cluster
		Address should be in the format <service-name>.<namespace>:<service_port>. Cannot be used with --telemetry-exporters.`,
	)
	err = cmd.Flags().MarkDeprecated("tracing-address",
		"and will be removed in a future release. OpenTelemetry (--telemetry-exporters) "+
			"is the preferred way to configure tracing.")
	if err != nil {
		fmt.Println("error marking flag as deprecated: ", err)
	}
	cmd.Flags().StringVar(
		&values.PrometheusAddress,
		"prometheus-address",
		defaultValues.PrometheusAddress,
		`the address of a Prometheus server deployed in your Kubernetes cluster
		Address should be in the format <service-name>.<namespace>:<service-port>`,
	)
	cmd.Flags().StringVar(
		&imageMeshAPI,
		"nginx-mesh-api-image",
		imageMeshAPI, "NGINX Service Mesh API image URI")
	err = cmd.Flags().MarkHidden("nginx-mesh-api-image")
	if err != nil {
		fmt.Println("error marking flag as hidden: ", err)
	}
	cmd.Flags().StringVar(
		&imageMetricsAPI,
		"nginx-mesh-metrics-image",
		imageMetricsAPI, "NGINX Service Mesh metrics API image URI")
	err = cmd.Flags().MarkHidden("nginx-mesh-metrics-image")
	if err != nil {
		fmt.Println("error marking flag as hidden: ", err)
	}
	cmd.Flags().StringVar(
		&imageSidecar,
		"nginx-mesh-sidecar-image",
		imageSidecar, "NGINX Service Mesh sidecar image URI")
	err = cmd.Flags().MarkHidden("nginx-mesh-sidecar-image")
	if err != nil {
		fmt.Println("error marking flag as hidden: ", err)
	}
	cmd.Flags().StringVar(
		&imageSidecarInit,
		"nginx-mesh-init-image",
		imageSidecarInit, "NGINX Service Mesh init image URI")
	err = cmd.Flags().MarkHidden("nginx-mesh-init-image")
	if err != nil {
		fmt.Println("error marking flag as hidden: ", err)
	}
	cmd.Flags().StringVar(
		&imageMeshCertReloader,
		"cert-reloader-image",
		imageMeshCertReloader, "NGINX Service Mesh cert reloader image URI")
	err = cmd.Flags().MarkHidden("cert-reloader-image")
	if err != nil {
		fmt.Println("error marking flag as hidden: ", err)
	}
	cmd.Flags().StringVar(
		&values.Registry.ImageTag,
		"image-tag",
		defaultValues.Registry.ImageTag, `tag used for pulling images from registry
		Affects: `+formatValues(registryServerImages),
	)
	cmd.Flags().StringVar(
		&values.AccessControlMode,
		"access-control-mode",
		defaultValues.AccessControlMode,
		`default access control mode for service-to-service communication
		Valid values: `+string(mesh.MeshConfigAccessControlModeAllow)+", "+string(mesh.MeshConfigAccessControlModeDeny),
	)
	cmd.Flags().StringVar(
		&values.MTLS.Mode,
		"mtls-mode",
		defaultValues.MTLS.Mode,
		`mTLS mode for pod-to-pod communication
		Valid values: `+formatValues(mesh.MtlsModes),
	)
	cmd.Flags().StringVar(
		&values.MTLS.CATTL,
		"mtls-ca-ttl",
		defaultValues.MTLS.CATTL,
		"the CA/signing key TTL in hours(h). Min value 24h. Max value 999999h.")
	cmd.Flags().StringVar(
		&values.MTLS.SVIDTTL,
		"mtls-svid-ttl",
		defaultValues.MTLS.SVIDTTL,
		"the TTL of certificates issued to workloads in hours(h) or minutes(m). Max value is 999999.")
	cmd.Flags().StringVar(
		&values.MTLS.TrustDomain,
		"mtls-trust-domain",
		defaultValues.MTLS.TrustDomain,
		"the trust domain of the NGINX Service Mesh")
	cmd.Flags().StringVar(
		&values.MTLS.SpireServerKeyManager,
		"spire-server-key-manager",
		defaultValues.MTLS.SpireServerKeyManager,
		`storage logic for SPIRE Server's private keys
		Valid values: `+formatValues(spireServerKeyManagerOptions),
	)
	cmd.Flags().StringVar(
		&values.MTLS.CAKeyType,
		"mtls-ca-key-type",
		defaultValues.MTLS.CAKeyType,
		`the key type used for the SPIRE Server CA
		Valid values: `+formatValues(caKeyTypeOptions),
	)
	cmd.Flags().StringVar(
		&mtlsUpstreamFile,
		"mtls-upstream-ca-conf",
		mtlsUpstreamFile,
		"the upstream certificate authority configuration file")
	cmd.Flags().StringVar(
		&values.Registry.Server,
		"registry-server",
		defaultValues.Registry.Server, `hostname:port (if needed) for registry and path to images
		Affects: `+formatValues(registryServerImages),
	)
	cmd.Flags().StringVar(
		&registryKeyFile,
		"registry-key",
		registryKeyFile, `path to JSON Key file for accessing private GKE registry
		Cannot be used with --registry-username or --registry-password`,
	)
	cmd.Flags().StringVar(
		&values.Registry.Username,
		"registry-username",
		defaultValues.Registry.Username, `username for accessing private registry
		Requires --registry-password to be set. Cannot be used with --registry-key`,
	)
	cmd.Flags().StringVar(
		&values.Registry.Password,
		"registry-password",
		defaultValues.Registry.Password, `password for accessing private registry
		Requires --registry-username to be set. Cannot be used with --registry-key`,
	)
	cmd.Flags().StringVar(
		&values.Environment,
		"environment",
		defaultValues.Environment,
		`environment to deploy the mesh into
		Valid values: `+formatValues(mesh.Environments),
	)
	cmd.Flags().BoolVar(
		&values.EnableUDP,
		"enable-udp",
		defaultValues.EnableUDP,
		`enable UDP traffic proxying (beta); Linux kernel 4.18 or greater is required`,
	)
	cmd.Flags().StringVar(
		&values.MTLS.PersistentStorage,
		"persistent-storage",
		"auto",
		`use persistent storage. "auto" will enable persistent storage if a default StorageClass exists
		Valid values: `+formatValues(persistentStorageOptions),
	)
	cmd.Flags().BoolVar(
		&cleanupOnError,
		"cleanup-on-error",
		true,
		`automatically remove Service Mesh pods if an error occurs during deployment
		This is a hidden flag for development purposes`,
	)
	err = cmd.Flags().MarkHidden("cleanup-on-error")
	if err != nil {
		fmt.Println("error marking flag as hidden: ", err)
	}
	cmd.Flags().StringVar(
		&values.NGINXLBMethod,
		"nginx-lb-method",
		defaultValues.NGINXLBMethod,
		`NGINX load balancing method
		Valid values: `+formatValues(mesh.LoadBalancingMethods),
	)
	cmd.Flags().StringVar(
		&values.NGINXErrorLogLevel,
		"nginx-error-log-level",
		defaultValues.NGINXErrorLogLevel,
		`NGINX error log level
		Valid values: `+formatLogLevels(),
	)
	cmd.Flags().StringVar(
		&values.NGINXLogFormat,
		"nginx-log-format",
		defaultValues.NGINXLogFormat,
		`NGINX log format
		Valid values: `+formatValues(mesh.NGINXLogFormats),
	)
	cmd.Flags().StringVar(
		&values.ClientMaxBodySize,
		"client-max-body-size",
		defaultValues.ClientMaxBodySize,
		`NGINX client max body size`,
	)
	cmd.Flags().BoolVar(
		&values.Registry.DisablePublicImages,
		"disable-public-images",
		defaultValues.Registry.DisablePublicImages,
		`don't pull third party images from public repositories`,
	)
	cmd.Flags().StringArrayVar(
		&telemetry.exporters,
		"telemetry-exporters",
		nil,
		`list of telemetry exporter key-value configurations
		Format: "type=<exporter_type>,host=<exporter_host>,port=<exporter_port>".
		Type, host, and port are required. Only type "otlp" exporter is supported. Cannot be used with --tracing-address.`,
	)
	cmd.Flags().Float32Var(
		&telemetry.samplerRatio,
		"telemetry-sampler-ratio",
		0.01, //nolint:gomnd // ignore default value
		`the percentage of traces that are processed and exported to the telemetry backend.
		Float between 0 and 1`,
	)

	return cmd
}

var (
	errTracingBackend = errors.New("unknown tracing server")
	errInvalidConfig  = errors.New("invalid configuration")
)

func setTracingAndTelemetryValues(telemetry telemetryConfig, tracing helm.Tracing, values *helm.Values) error {
	if len(telemetry.exporters) > 0 {
		if tracing.Address != "" {
			return fmt.Errorf("%w: cannot set both --tracing-address and --telemetry-exporters", errInvalidConfig)
		}
		if len(telemetry.exporters) != 1 {
			return fmt.Errorf("%w: only one telemetry exporter may be configured", errInvalidConfig)
		}
		exp := telemetry.exporters[0]
		exporterMap, err := exporterStringToMap(exp)
		if err != nil {
			return err
		}
		if err := validateExporterConfig(exporterMap); err != nil {
			return fmt.Errorf("invalid telemetry-exporters input %q: %w", exp, err)
		}

		if err := convertTelemetryOpsToHelmValues(exporterMap, telemetry.samplerRatio, values); err != nil {
			return err
		}
	} else {
		// set validate tracing backend and set values
		invalidTracingMessage := "both --tracing-address and --tracing-backend must be set"
		if tracing.Address != "" {
			if tracing.Backend == "" {
				return fmt.Errorf("%w: %s", errInvalidConfig, invalidTracingMessage)
			}
		} else if tracing.Backend != "" {
			return fmt.Errorf("%w: %s", errInvalidConfig, invalidTracingMessage)
		}

		if tracing.Address != "" && tracing.Backend != "" {
			if _, ok := mesh.TracingBackends[tracing.Backend]; !ok {
				return fmt.Errorf("%w \"%s\". Valid values: %v", errTracingBackend, tracing.Backend, formatValues(mesh.TracingBackends))
			}

			values.Tracing = &helm.Tracing{
				Address:    tracing.Address,
				Backend:    tracing.Backend,
				SampleRate: tracing.SampleRate,
			}
		}
	}

	return nil
}

func convertTelemetryOpsToHelmValues(exporterMap map[string]string, sampleRatio float32, values *helm.Values) error {
	port, err := strconv.Atoi(exporterMap["port"])
	if err != nil {
		return err
	}
	values.Telemetry = &helm.Telemetry{
		Exporters: &helm.Exporter{
			OTLP: &helm.OTLP{
				Host: exporterMap["host"],
				Port: port,
			},
		},
		SamplerRatio: sampleRatio,
	}

	return nil
}

func readAsCSV(s string) ([]string, error) {
	r := csv.NewReader(strings.NewReader(s))

	return r.Read()
}

//nolint:goerr113 // errorf is fine
func exporterStringToMap(exporter string) (map[string]string, error) {
	eList, err := readAsCSV(exporter)
	if err != nil {
		return nil, fmt.Errorf("failed to parse telemetry-exporters input: %q; expected key value pairs separated by commas", exporter)
	}
	eMap, err := sliceToMap(eList)
	if err != nil {
		return nil, fmt.Errorf("failed to parse telemetry-exporters input %q: %w", exporter, err)
	}

	return eMap, nil
}

var errInvalidExporterConfig = errors.New("invalid exporter config")

func validateExporterConfig(config map[string]string) error {
	t, ok := config["type"]
	if !ok {
		return fmt.Errorf("%w: missing type", errInvalidExporterConfig)
	}
	if t != "otlp" {
		return fmt.Errorf("%w: unsupported type: must be \"otlp\"", errInvalidExporterConfig)
	}
	if _, ok := config["host"]; !ok {
		return fmt.Errorf("%w: missing host", errInvalidExporterConfig)
	}
	if _, ok := config["port"]; !ok {
		return fmt.Errorf("%w: missing port", errInvalidExporterConfig)
	}

	return nil
}

//nolint:goerr113 // errorf is fine
func sliceToMap(ss []string) (map[string]string, error) {
	kvMap := make(map[string]string, len(ss))
	for _, pair := range ss {
		keyval := strings.SplitN(pair, "=", 2) //nolint:gomnd // it says pair right there
		if len(keyval) != 2 {                  //nolint:gomnd // it says pair right there
			return nil, fmt.Errorf("%s must be formatted as key=value", pair)
		}
		// check for duplicates
		if _, ok := kvMap[keyval[0]]; ok {
			return nil, fmt.Errorf("duplicate key: %s", keyval[0])
		}
		kvMap[keyval[0]] = keyval[1]
	}

	return kvMap, nil
}

// Custom input validation for complex values. Helm's error messages are not clear for these fields.
func validateInput(values *helm.Values, registryKeyFile string) error {
	if (values.Registry.Username == "") != (values.Registry.Password == "") {
		return fmt.Errorf("%w: both --registry-username and --registry-password must be set", errInvalidConfig)
	}

	if registryKeyFile != "" && values.Registry.Username != "" {
		return fmt.Errorf("%w: cannot set both --registry-key and --registry-username/--registry-password", errInvalidConfig)
	}

	if !values.DisableAutoInjection && len(values.EnabledNamespaces) > 0 {
		return fmt.Errorf("%w: enabled namespaces should not be set when auto injection is enabled", errInvalidConfig)
	}

	if values.DisableAutoInjection && len(values.AutoInjection.DisabledNamespaces) > 0 {
		return fmt.Errorf("%w: disabled namespaces should not be set when auto injection is disabled", errInvalidConfig)
	}

	return nil
}

func startDeploy(k8sClient k8s.Client, deployer *deploy.Deployer, cleanupOnError bool) error {
	// Catch and cleanup if necessary
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	signalHandler := newDeploySignalHandle(k8sClient, cleanOnSignal, os.Stdout, deployer.Values.Environment)
	signalHandler.Watch(ctx, syscall.SIGINT, syscall.SIGHUP, syscall.SIGTERM)

	// start checking for ImagePullErrors; return when we find an error or successfully connect
	done := make(chan struct{})
	go func() {
		for {
			select {
			case <-done:
				return
			default:
				// sleep briefly to prevent tight loop
				time.Sleep(100 * time.Millisecond) //nolint:gomnd // not worth another global
				err := checkImagePullErrors(k8sClient)
				if err != nil {
					fmt.Println(err.Error())
					if cleanupOnError {
						cleanup(k8sClient)
					}

					os.Exit(1)
				}
			}
		}
	}()

	_, err := deployer.Deploy()
	if err != nil {
		var alreadyExists meshErrors.AlreadyExistsError
		if errors.Is(err, meshErrors.ErrInput) ||
			errors.As(err, &alreadyExists) ||
			errors.Is(err, meshErrors.ErrCheckingExistence) {
			return err
		}

		signalHandler.Check()
		fmt.Printf("Failed to deploy: %v\n", err)
		if instructions := meshErrors.NamespaceExistsError(err); instructions != "" {
			fmt.Println(instructions)

			return meshErrors.ErrDeployingMesh
		}

		if _, instructions := meshErrors.CheckForK8sFatalError(err); instructions != "" {
			fmt.Println(instructions)
		}
		if cleanupOnError {
			cleanup(k8sClient)
		}

		return meshErrors.ErrDeployingMesh
	}
	if deployer.DryRun {
		return nil
	}
	signalHandler.Check()

	fmt.Println("All resources created. Testing the connection to the Service Mesh API Server...")
	// test connection
	err = health.TestMeshAPIConnection(k8sClient.Config(), deployMeshAPIRetries, meshTimeout)
	if err != nil {
		return formatConnectionError(k8sClient, err)
	}
	done <- struct{}{}
	fmt.Println("Connected to the NGINX Service Mesh API successfully.")
	fmt.Println("NGINX Service Mesh is running.")

	return nil
}

func formatConnectionError(k8sClient k8s.Client, err error) error {
	ns := k8sClient.Namespace()
	baseMsg := fmt.Sprintf(meshAPIConnectionFailedInstructions, ns)
	baseErr := fmt.Errorf("%s: %w", baseMsg, err)
	events, err := k8sClient.ClientSet().CoreV1().Events(ns).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("%w; unable to get events for namespace '%s'", baseErr, ns)
	}

	var msg string
	for _, e := range events.Items {
		if e.Type == v1.EventTypeWarning {
			msg += fmt.Sprintf("\n\t- %s", e.Message)
		}
	}
	if len(msg) > 0 {
		return fmt.Errorf("%w; the following events occurred:%s", baseErr, msg)
	}

	return baseErr
}

func setPersistentStorage(clientset kubernetes.Interface, values *helm.Values) error {
	const off = "off"
	persistentStorage := false
	if values.MTLS.PersistentStorage != off {
		storageClassList, err := clientset.StorageV1().StorageClasses().List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			return fmt.Errorf("error getting list of StorageClasses: %w", err)
		}
		for _, storageClass := range storageClassList.Items {
			if storageClass.ObjectMeta.Annotations["storageclass.kubernetes.io/is-default-class"] == "true" ||
				storageClass.ObjectMeta.Annotations["storageclass.beta.kubernetes.io/is-default-class"] == "true" {
				persistentStorage = true

				break
			}
		}
		if values.MTLS.PersistentStorage == "on" && !persistentStorage {
			return errors.New("unable to find default StorageClass, ensure one is configured") //nolint:goerr113 // one-off error
		}
	}

	if persistentStorage {
		values.MTLS.PersistentStorage = "on"
	} else {
		values.MTLS.PersistentStorage = off
		fmt.Println("Warning: Deploying without persistent storage, not suitable for production environments.")
		fmt.Println("         For production environments ensure a default StorageClass is set.")
	}

	return nil
}

func cleanup(k8sClient k8s.Client) {
	fmt.Println("\nCleaning up NGINX Service Mesh resources...")
	deleteNamespace := true
	err := newRemover(k8sClient).remove("nginx-service-mesh", deleteNamespace)
	if err == nil {
		fmt.Println("All NGINX Service Mesh resources have been deleted.")
	}
}

func checkImagePullErrors(k8sClient k8s.Client) error {
	var msg string
	events, err := k8sClient.ClientSet().CoreV1().Events(k8sClient.Namespace()).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		fmt.Printf("unable to get events for namespace '%s'\n", k8sClient.Namespace())

		return nil //nolint:nilerr // we don't want to return the error in this case
	}
	for _, e := range events.Items {
		// only get errors that are recent (if older than a minute they are likely from a previous deployment)
		if e.Type == v1.EventTypeWarning && time.Since(e.CreationTimestamp.Time) < 1*time.Minute {
			if strings.Contains(e.Message, "Failed to pull image") {
				msg += fmt.Sprintf("\n\t- %s", e.Message)
			}
		}
	}
	if len(msg) > 0 {
		// check if control plane pods are failing or if they've recovered
		pods, err := k8sClient.ClientSet().CoreV1().Pods(k8sClient.Namespace()).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			return meshErrors.ImagePullError{Msg: msg}
		}
		for _, pod := range pods.Items {
			if !isPodReady(pod) {
				return meshErrors.ImagePullError{Msg: msg}
			}
		}
	}

	return nil
}

// isPodReady returns whether a Pod is in the Ready state.
func isPodReady(pod v1.Pod) bool {
	for _, cond := range pod.Status.Conditions {
		if cond.Type == v1.PodReady && cond.Status == v1.ConditionTrue {
			return true
		}
	}

	return false
}

func subImages(images customImages, files []*loader.BufferedFile) {
	for component, cfg := range images {
		if cfg.value != "" {
			var oldName string
			// init and sidecar have slightly different format
			if component == mesh.MeshSidecar || component == mesh.MeshSidecarInit {
				oldName = fmt.Sprintf("{{ printf \"%%s/%s:%%s\" .Values.registry.server .Values.registry.imageTag | quote }}", component)
			} else {
				oldName = fmt.Sprintf("{{ .Values.registry.server }}/%s:{{ .Values.registry.imageTag }}", component)
			}
			for _, file := range files {
				if file.Name == cfg.file {
					str := string(file.Data)
					str = strings.ReplaceAll(str, oldName, cfg.value)
					file.Data = []byte(str)

					break
				}
			}
		}
	}
}

func formatLogLevels() string {
	return string(mesh.MeshConfigNginxErrorLogLevelDebug) + ", " +
		string(mesh.MeshConfigNginxErrorLogLevelInfo) + ", " +
		string(mesh.MeshConfigNginxErrorLogLevelNotice) + ", " +
		string(mesh.MeshConfigNginxErrorLogLevelWarn) + ", " +
		string(mesh.MeshConfigNginxErrorLogLevelError) + ", " +
		string(mesh.MeshConfigNginxErrorLogLevelCrit) + ", " +
		string(mesh.MeshConfigNginxErrorLogLevelAlert) + ", " +
		string(mesh.MeshConfigNginxErrorLogLevelEmerg)
}

func formatValues(values map[string]struct{}) string {
	keys := make([]string, 0, len(values))
	for k := range values {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	return strings.Join(keys, ", ")
}

var persistentStorageOptions = map[string]struct{}{
	"auto": {},
	"off":  {},
	"on":   {},
}

var spireServerKeyManagerOptions = map[string]struct{}{
	"disk":   {},
	"memory": {},
}

var registryServerImages = map[string]struct{}{
	"nginx-mesh-api":           {},
	"nginx-mesh-cert-reloader": {},
	"nginx-mesh-init":          {},
	"nginx-mesh-metrics":       {},
	"nginx-mesh-sidecar":       {},
}

var caKeyTypeOptions = map[string]struct{}{
	"ec-p256":  {},
	"ec-p384":  {},
	"rsa-2048": {},
	"rsa-4096": {},
}
