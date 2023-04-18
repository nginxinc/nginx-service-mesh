package inject

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/golang/glog"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	"github.com/nginxinc/nginx-service-mesh/pkg/apis/mesh"
	"github.com/nginxinc/nginx-service-mesh/pkg/pod"
	"github.com/nginxinc/nginx-service-mesh/pkg/sidecar"
)

const (
	RedirectPath         = "/health-redirects/"
	startupProbe         = "startupProbe"
	sidecarContainerPort = 8887
	runAsUser            = int64(2102)
)

type (
	PortSet map[string]struct{}
	Probe   struct {
		ProbeType    string
		HTTPGet      v1.HTTPGetAction
		OrigHTTPGet  v1.HTTPGetAction
		ContainerIdx int
	}
)

// CreateInjectionConfig builds the config for injecting the sidecar proxy.
func CreateInjectionConfig(
	meshConfig mesh.FullMeshConfig,
	ignorePorts IgnorePorts,
	containers []v1.Container,
	parentName,
	parentNamespace,
	parentType string,
	podAnnotations map[string]string,
) (*InjectionConfig, error) {
	priv := false
	runAsRoot := int64(0)
	pullPolicy := v1.PullPolicy(meshConfig.Registry.ImagePullPolicy)

	// Build init container spec
	initContainer := v1.Container{
		Name:            mesh.MeshSidecarInit,
		Image:           meshConfig.Registry.SidecarInitImage,
		ImagePullPolicy: pullPolicy,
		SecurityContext: &v1.SecurityContext{
			RunAsUser: &runAsRoot,
			Capabilities: &v1.Capabilities{
				Add: []v1.Capability{
					"NET_ADMIN",
					"NET_RAW",
					"SYS_RESOURCE",
					"SYS_ADMIN",
				},
			},
			Privileged: &priv,
		},
	}
	if meshConfig.EnableUDP {
		initContainer.Args = append(initContainer.Args, "--enable-udp")
	}

	if meshConfig.Environment == mesh.Openshift {
		initContainer.SecurityContext.SELinuxOptions = &v1.SELinuxOptions{
			Type: "spc_t",
		}
	}

	// Build sidecar container spec
	user := runAsUser
	proxySidecar := v1.Container{
		Name:            mesh.MeshSidecar,
		Image:           meshConfig.Registry.SidecarImage,
		ImagePullPolicy: pullPolicy,
		Ports:           []v1.ContainerPort{{ContainerPort: sidecarContainerPort}},
		Env: []v1.EnvVar{
			{
				Name:  "MY_DEPLOY_NAME",
				Value: parentName,
			},
			{
				Name: "MY_NAMESPACE",
				ValueFrom: &v1.EnvVarSource{
					FieldRef: &v1.ObjectFieldSelector{
						FieldPath: "metadata.namespace",
					},
				},
			},
			{
				Name: "MY_POD_NAME",
				ValueFrom: &v1.EnvVarSource{
					FieldRef: &v1.ObjectFieldSelector{
						FieldPath: "metadata.name",
					},
				},
			},
			{
				Name: "MY_POD_IP",
				ValueFrom: &v1.EnvVarSource{
					FieldRef: &v1.ObjectFieldSelector{
						FieldPath: "status.podIP",
					},
				},
			},
			{
				Name: "MY_SERVICE_ACCOUNT",
				ValueFrom: &v1.EnvVarSource{
					FieldRef: &v1.ObjectFieldSelector{
						FieldPath: "spec.serviceAccountName",
					},
				},
			},
		},
		SecurityContext: &v1.SecurityContext{
			RunAsUser:                &user,
			Privileged:               &priv,
			AllowPrivilegeEscalation: &priv,
		},
	}

	// set port arguments
	if err := setPortArgs(containers, ignorePorts, &initContainer, &proxySidecar); err != nil {
		return nil, err
	}

	proxySidecar.Args = append(proxySidecar.Args, "-n", parentName, "--namespace", meshConfig.Namespace)

	redirectHealthPort := sidecar.RedirectHealthPort
	redirectHealthPortHTTPS := sidecar.RedirectHealthHTTPSPort
	probes := GetProbes(containers, redirectHealthPort, redirectHealthPortHTTPS)
	// Add original probes to the args for the agent to redirect to
	origProbes := make(map[string]v1.HTTPGetAction)
	for _, p := range probes {
		origProbes[p.HTTPGet.Path] = p.OrigHTTPGet
	}
	if len(origProbes) > 0 {
		b, jsonErr := json.Marshal(origProbes)
		if jsonErr != nil {
			return nil, fmt.Errorf("could not marshal the health probes: %w", jsonErr)
		}
		proxySidecar.Args = append(proxySidecar.Args, "--health-redirects", string(b))
		initContainer.Args = append(initContainer.Args, "--ignore-incoming-ports", strconv.Itoa(redirectHealthPort))
		initContainer.Args = append(initContainer.Args, "--ignore-incoming-ports", strconv.Itoa(redirectHealthPortHTTPS))
	}

	mtlsModeAnnotation, err := pod.GetMTLSModeAnnotation(podAnnotations)
	if err != nil {
		glog.Warning(err.Error())
	}
	clientMaxBodySizeAnnotation, err := pod.GetClientMaxBodySizeAnnotation(podAnnotations)
	if err != nil {
		glog.Warning(err.Error())
	}

	// if not set, agent will get it from NATS
	if mtlsModeAnnotation != "" {
		if err := ValidateMTLSAnnotation(mtlsModeAnnotation, meshConfig.Mtls.Mode); err != nil {
			return nil, fmt.Errorf("%w; mtls annotation for '%s' cannot conflict", err, parentName)
		}

		proxySidecar.Args = append(proxySidecar.Args, "--mtls-annotation", mtlsModeAnnotation)
	}

	// If clientMaxBodySize is supplied via annotation, add an arg. Otherwise, agent will get it from NATS
	if clientMaxBodySizeAnnotation != "" {
		proxySidecar.Args = append(proxySidecar.Args, "--client-max-body-size", clientMaxBodySizeAnnotation)
	}

	// set volumes
	var volumes []v1.Volume
	proxySidecar.VolumeMounts = append(proxySidecar.VolumeMounts, v1.VolumeMount{
		Name:      "spire-agent-socket",
		MountPath: "/run/spire/sockets",
		ReadOnly:  true,
	})
	spireSocketVolume := v1.Volume{
		Name: "spire-agent-socket",
	}
	if meshConfig.Environment == mesh.Openshift {
		readOnly := true
		spireSocketVolume.VolumeSource = v1.VolumeSource{
			CSI: &v1.CSIVolumeSource{
				Driver:   "csi.spiffe.io",
				ReadOnly: &readOnly,
			},
		}
	} else {
		hostPathFileSocket := v1.HostPathDirectoryOrCreate
		spireSocketVolume.VolumeSource = v1.VolumeSource{
			HostPath: &v1.HostPathVolumeSource{
				Path: "/run/spire/sockets",
				Type: &hostPathFileSocket,
			},
		}
	}
	volumes = append(volumes, spireSocketVolume)

	// set metadata
	labels := make(map[string]string)
	labels[mesh.DeployLabel+parentType] = parentName
	labels[mesh.SpiffeIDLabel] = "true"

	annotations := make(map[string]string)
	annotations[mesh.InjectedAnnotation] = mesh.Injected

	// set imagePullSecrets
	imagePullSecrets := make([]v1.LocalObjectReference, 0)
	var registryKey *v1.Secret
	if meshConfig.Registry.RegistryKeyName != "" {
		var err error
		registryKey, err = createRegistryKey(meshConfig, parentNamespace)
		if err != nil {
			return nil, fmt.Errorf("creating registry key for \"%s\" namespace: %w", parentNamespace, err)
		}
		imagePullSecrets = append(imagePullSecrets, v1.LocalObjectReference{
			Name: meshConfig.Registry.RegistryKeyName,
		})
	}

	cfg := &InjectionConfig{
		InitContainers:   []v1.Container{initContainer},
		Containers:       []v1.Container{proxySidecar},
		Volumes:          volumes,
		Probes:           probes,
		Labels:           labels,
		Annotations:      annotations,
		ImagePullSecrets: imagePullSecrets,
		RegistryKey:      registryKey,
	}

	return cfg, nil
}

// GetProbes builds and returns the redirected health probes for an application with a sidecar.
func GetProbes(containers []v1.Container, httpPort, httpsPort int) []Probe {
	var probes []Probe
	for idx, container := range containers {
		agentPath := RedirectPath + container.Name
		if container.LivenessProbe != nil && container.LivenessProbe.HTTPGet != nil {
			probes = append(probes, createProbe(
				container.LivenessProbe.HTTPGet, idx, httpPort, httpsPort, livenessProbe, agentPath, "/liveness", container.Ports))
		}
		if container.ReadinessProbe != nil && container.ReadinessProbe.HTTPGet != nil {
			probes = append(probes, createProbe(
				container.ReadinessProbe.HTTPGet, idx, httpPort, httpsPort, readinessProbe, agentPath, "/readiness", container.Ports))
		}
		if container.StartupProbe != nil && container.StartupProbe.HTTPGet != nil {
			probes = append(probes, createProbe(
				container.StartupProbe.HTTPGet, idx, httpPort, httpsPort, startupProbe, agentPath, "/startup", container.Ports))
		}
	}

	return probes
}

func createProbe(
	httpGet *v1.HTTPGetAction,
	idx,
	httpPort,
	httpsPort int,
	probeType,
	agentPath,
	ext string,
	ports []v1.ContainerPort,
) Probe {
	prb := Probe{
		HTTPGet:      *httpGet,
		OrigHTTPGet:  *httpGet,
		ProbeType:    probeType,
		ContainerIdx: idx,
	}
	if httpGet.Port.Type == intstr.String {
		for _, port := range ports {
			if port.Name == httpGet.Port.StrVal {
				prb.OrigHTTPGet.Port = intstr.FromInt(int(port.ContainerPort))

				break
			}
		}
	}
	prb.HTTPGet.Path = agentPath + ext
	// set scheme if omitted from spec
	if httpGet.Scheme == "" {
		prb.OrigHTTPGet.Scheme = v1.URISchemeHTTP
	}
	if httpGet.Scheme == v1.URISchemeHTTPS {
		prb.HTTPGet.Port = intstr.FromInt(httpsPort)
	} else {
		prb.HTTPGet.Port = intstr.FromInt(httpPort)
	}

	return prb
}

func createRegistryKey(meshConfig mesh.FullMeshConfig, podNamespace string) (*v1.Secret, error) {
	// Create K8S clientset from in-cluster config
	k8sConfig, err := config.GetConfig()
	if err != nil {
		return nil, err
	}
	k8sClient, err := client.New(k8sConfig, client.Options{})
	if err != nil {
		return nil, err
	}

	// Retrieve existing registry key for the NGINX Mesh namespace
	secret := v1.Secret{}
	key := client.ObjectKey{Namespace: meshConfig.Namespace, Name: meshConfig.Registry.RegistryKeyName}
	if err = k8sClient.Get(context.TODO(), key, &secret); err != nil {
		return nil, fmt.Errorf("error retrieving registry key from \"%s\" namespace: %w", meshConfig.Namespace, err)
	}

	secret.ObjectMeta.Namespace = podNamespace
	secret.ObjectMeta.ResourceVersion = ""
	secret.ObjectMeta.UID = ""
	secret.APIVersion = "v1"
	secret.Kind = "Secret"

	return &secret, nil
}

// setPortArgs sets the service port and ignore port arguments on the init/sidecar containers.
func setPortArgs(
	containers []v1.Container,
	ignorePorts IgnorePorts,
	initContainer,
	proxySidecar *v1.Container,
) error {
	ports, err := ValidatePorts(containers)
	if err != nil {
		return err
	}

	for port := range ports {
		proxySidecar.Args = append(proxySidecar.Args, "-s", port)
	}
	initContainer.Args = append(initContainer.Args, "--ignore-incoming-ports", strconv.Itoa(sidecar.MetricsPort))
	if ignorePorts.Incoming != nil {
		for _, port := range ignorePorts.Incoming {
			if port != sidecar.MetricsPort {
				initContainer.Args = append(initContainer.Args, "--ignore-incoming-ports", strconv.Itoa(port))
			}
		}
	}
	if ignorePorts.Outgoing != nil {
		for _, port := range ignorePorts.Outgoing {
			initContainer.Args = append(initContainer.Args, "--ignore-outgoing-ports", strconv.Itoa(port))
		}
	}

	return nil
}

// ValidatePorts parses and formats ports
// returns an error if port protocol is not supported
// or if port is a duplicate.
func ValidatePorts(containers []v1.Container) (PortSet, error) {
	pSet := make(PortSet)
	for _, container := range containers {
		for _, port := range container.Ports {
			if !supportedProtocol(port.Protocol) {
				return nil, fmt.Errorf("unsupported protocol %s; container protocol must be TCP", port.Protocol)
			}
			if _, ok := pSet[strconv.Itoa(int(port.ContainerPort))]; ok {
				return nil, fmt.Errorf("%d container port number already defined; duplicate ports are not supported", port.ContainerPort)
			}
			pSet[strconv.Itoa(int(port.ContainerPort))] = struct{}{}
		}
	}

	return pSet, nil
}

func supportedProtocol(protocol v1.Protocol) bool {
	return protocol == "" || protocol == v1.ProtocolTCP || protocol == v1.ProtocolUDP
}

func ValidateMTLSAnnotation(podAnnotation, globalMode string) error {
	if podAnnotation != "" {
		// if global mtls is strict but annotation is not, return an error
		if globalMode == mesh.MtlsModeStrict && podAnnotation != mesh.MtlsModeStrict {
			return errors.New("global mtls mode is 'strict'")
		}
	}

	return nil
}
