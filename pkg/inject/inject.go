// Package inject contains code for sidecar injection.
package inject

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"strings"

	"github.com/golang/glog"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sJson "k8s.io/apimachinery/pkg/runtime/serializer/json"
	k8sYaml "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/kubectl/pkg/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/nginxinc/nginx-service-mesh/pkg/apis/mesh"
)

const (
	injectTemplateYAML = `{{ range $index, $element := .Inputs }}---
{{writeResource $.Serializer $element.Encode $element.Object $element.Doc}}{{end}}`

	//nolint:lll // JSON string
	injectTemplateJSON = `{{$length := len .Inputs}}{{if eq $length 1}}{{writeResource .Serializer (index .Inputs 0).Encode (index .Inputs 0).Object (index .Inputs 0).Doc}}{{else}}{{if gt $length 1}}{
  "apiVersion": "v1",
	"kind": "List",
	"items": [
{{ range $index, $element := .Inputs }} {{if $index}},
{{end}}{{writeResource $.Serializer $element.Encode $element.Object $element.Doc}}{{end}}
  ]
}{{end}}{{end}}
`
	livenessProbe  = "livenessProbe"
	readinessProbe = "readinessProbe"
)

type (
	// InjectionConfig is the proxy sidecar configuration to be injected.
	InjectionConfig struct {
		Annotations      map[string]string
		RegistryKey      *v1.Secret
		Labels           map[string]string
		Volumes          []v1.Volume
		ImagePullSecrets []v1.LocalObjectReference
		Probes           []Probe
		InitContainers   []v1.Container
		Containers       []v1.Container
	}

	injectInput struct {
		Object runtime.Object
		Doc    []byte
		Encode bool
	}

	injectTemplateArgs struct {
		Serializer runtime.Encoder
		Inputs     []injectInput
		IsJSON     bool
	}

	// Inject holds the config for a manual sidecar injection.
	Inject struct {
		Resources   []byte
		IgnorePorts IgnorePorts
	}
)

// IntoFile takes a yaml or json resource file and adds the sidecar containers.
func IntoFile(
	injectConfig Inject,
	meshConfig mesh.FullMeshConfig,
) (string, error) {
	var docs [][]byte
	serializer := k8sJson.NewSerializerWithOptions(k8sJson.DefaultMetaFactory, nil, nil, k8sJson.SerializerOptions{Pretty: true})
	isJSON, _, _ := serializer.RecognizesData(injectConfig.Resources)

	if isJSON {
		var resList metav1.List
		err := json.Unmarshal(injectConfig.Resources, &resList)
		if err != nil {
			return "", fmt.Errorf("could not parse JSON document(s): %w", err)
		}

		if resList.Kind != "List" {
			docs = append(docs, injectConfig.Resources)
		} else {
			for _, d := range resList.Items {
				docs = append(docs, d.Raw)
			}
		}
	} else {
		yamlReader := bytes.NewReader(injectConfig.Resources)
		decoder := k8sYaml.NewDocumentDecoder(io.NopCloser(yamlReader))
		defer decoder.Close()

		// Read all docs in the yaml file (separated by ---)
		for {
			readArray := make([]byte, 10000)
			length, err := decoder.Read(readArray)
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				return "", fmt.Errorf("error reading documents: %w", err)
			}
			truncatedReadArray := readArray[0:length]
			docs = append(docs, truncatedReadArray)
		}
		serializer = k8sJson.NewSerializerWithOptions(k8sJson.DefaultMetaFactory, nil, nil, k8sJson.SerializerOptions{Yaml: true})
	}

	tmplArgs := injectTemplateArgs{
		IsJSON:     isJSON,
		Serializer: serializer,
		Inputs:     make([]injectInput, 0, len(docs)),
	}
	registryKeySeen := make(map[string]struct{})
	for _, doc := range docs {
		obj, _, err := scheme.Codecs.UniversalDeserializer().Decode(doc, nil, nil)
		if err != nil {
			return "", fmt.Errorf("error decoding file into k8s object: %w", err)
		}

		var registryKey *v1.Secret
		switch o := obj.(type) {
		case *appsv1.Deployment:
			registryKey, err = updateResource(
				meshConfig, injectConfig.IgnorePorts, &o.Spec.Template.ObjectMeta,
				&o.Spec.Template.Spec, "deployment", o.Name, o.Namespace, registryKeySeen)
			if err != nil {
				return "", err
			}
			tmplArgs.Inputs = append(tmplArgs.Inputs, injectInput{o, doc, true})
		case *appsv1.DaemonSet:
			registryKey, err = updateResource(
				meshConfig, injectConfig.IgnorePorts, &o.Spec.Template.ObjectMeta,
				&o.Spec.Template.Spec, "daemonset", o.Name, o.Namespace, registryKeySeen)
			if err != nil {
				return "", err
			}
			tmplArgs.Inputs = append(tmplArgs.Inputs, injectInput{o, doc, true})
		case *appsv1.StatefulSet:
			registryKey, err = updateResource(
				meshConfig, injectConfig.IgnorePorts, &o.Spec.Template.ObjectMeta,
				&o.Spec.Template.Spec, "statefulset", o.Name, o.Namespace, registryKeySeen)
			if err != nil {
				return "", err
			}
			tmplArgs.Inputs = append(tmplArgs.Inputs, injectInput{o, doc, true})
		case *appsv1.ReplicaSet:
			registryKey, err = updateResource(
				meshConfig, injectConfig.IgnorePorts, &o.Spec.Template.ObjectMeta,
				&o.Spec.Template.Spec, "replicaset", o.Name, o.Namespace, registryKeySeen)
			if err != nil {
				return "", err
			}
			tmplArgs.Inputs = append(tmplArgs.Inputs, injectInput{o, doc, true})
		case *batchv1.Job:
			registryKey, err = updateResource(
				meshConfig, injectConfig.IgnorePorts, &o.Spec.Template.ObjectMeta,
				&o.Spec.Template.Spec, "job", o.Name, o.Namespace, registryKeySeen)
			if err != nil {
				return "", err
			}
			tmplArgs.Inputs = append(tmplArgs.Inputs, injectInput{o, doc, true})
		case *v1.ReplicationController:
			registryKey, err = updateResource(
				meshConfig, injectConfig.IgnorePorts, &o.Spec.Template.ObjectMeta,
				&o.Spec.Template.Spec, "replicationcontroller", o.Name, o.Namespace, registryKeySeen)
			if err != nil {
				return "", err
			}
			tmplArgs.Inputs = append(tmplArgs.Inputs, injectInput{o, doc, true})
		case *v1.Pod:
			registryKey, err = updateResource(
				meshConfig, injectConfig.IgnorePorts, &o.ObjectMeta, &o.Spec,
				"pod", o.Name, o.Namespace, registryKeySeen)
			if err != nil {
				return "", err
			}
			tmplArgs.Inputs = append(tmplArgs.Inputs, injectInput{o, doc, true})
		default:
			tmplArgs.Inputs = append(tmplArgs.Inputs, injectInput{o, doc, false})
		}

		if registryKey != nil {
			tmplArgs.Inputs = append(tmplArgs.Inputs, injectInput{registryKey, doc, true})
		}
	}

	return constructOutput(tmplArgs)
}

// Injects the sidecar into a PodSpec.
func updateResource(
	meshConfig mesh.FullMeshConfig,
	ignorePorts IgnorePorts,
	meta *metav1.ObjectMeta,
	spec *v1.PodSpec,
	parentType,
	name,
	namespace string,
	registryKeySeen map[string]struct{},
) (*v1.Secret, error) {
	ip, err := GetIgnorePorts(meta.Annotations, ignorePorts)
	if err != nil {
		return nil, err
	}
	cfg, err := CreateInjectionConfig(
		meshConfig, ip, spec.Containers, name, namespace, parentType, meta.Annotations)
	if err != nil {
		return nil, fmt.Errorf("creating injection config for \"%s\": %w", name, err)
	}
	var prb Probe
	for _, prb = range cfg.Probes {
		switch prb.ProbeType {
		case livenessProbe:
			spec.Containers[prb.ContainerIdx].LivenessProbe.HTTPGet = &prb.HTTPGet
		case readinessProbe:
			spec.Containers[prb.ContainerIdx].ReadinessProbe.HTTPGet = &prb.HTTPGet
		default:
			spec.Containers[prb.ContainerIdx].StartupProbe.HTTPGet = &prb.HTTPGet
		}
	}
	spec.Containers = append(spec.Containers, cfg.Containers...)
	spec.InitContainers = append(spec.InitContainers, cfg.InitContainers...)
	spec.Volumes = append(spec.Volumes, cfg.Volumes...)
	spec.ImagePullSecrets = append(spec.ImagePullSecrets, cfg.ImagePullSecrets...)
	if meta.Annotations == nil {
		meta.Annotations = make(map[string]string)
	}
	if meta.Labels == nil {
		meta.Labels = make(map[string]string)
	}
	for k, v := range cfg.Annotations {
		meta.Annotations[k] = v
	}
	for k, v := range cfg.Labels {
		meta.Labels[k] = v
	}

	if _, ok := registryKeySeen[namespace]; !ok {
		registryKeySeen[namespace] = struct{}{}

		return cfg.RegistryKey, nil
	}

	return nil, nil
}

// Creates the resource string for writing back to a file.
func writeResource(
	serializer runtime.Encoder,
	encode bool,
	obj runtime.Object,
	doc []byte,
) string {
	var output string
	var buff bytes.Buffer

	if encode {
		err := serializer.Encode(obj, &buff)
		if err != nil {
			glog.Errorf("skipping document. error encoding resource: %v", err)
			output += string(doc)
		} else {
			output += buff.String()
		}
	} else {
		output = string(doc)
		if output[len(output)-1:] != "\n" {
			output += "\n"
		}
	}

	return output
}

func constructOutput(args injectTemplateArgs) (string, error) {
	funcs := template.FuncMap{
		"writeResource": writeResource,
	}

	var templateText string
	switch {
	case args.IsJSON:
		templateText = injectTemplateJSON
	default:
		templateText = injectTemplateYAML
	}

	tmpl, err := template.New("inject").Funcs(funcs).Parse(templateText)
	if err != nil {
		return "", fmt.Errorf("failed parsing inject template: %w", err)
	}
	var strBuilder strings.Builder
	err = tmpl.Execute(&strBuilder, args)
	if err != nil {
		return "", fmt.Errorf("failed executing inject template: %w", err)
	}

	formatted := strings.ReplaceAll(strBuilder.String(), "&#34;", "\"")
	formatted = strings.ReplaceAll(formatted, "&#39;", "'")
	if args.IsJSON {
		var out bytes.Buffer
		err = json.Indent(&out, []byte(formatted), "", "  ")

		return out.String(), err
	}

	return formatted, nil
}

// IsNamespaceInjectable determines if namespace is injectable.
func IsNamespaceInjectable(ctx context.Context, k8sClient client.Client, namespace string) (bool, error) {
	// Never inject ignored namespaces.
	for ignoredNS := range mesh.IgnoredNamespaces {
		if namespace == ignoredNS {
			return false, nil
		}
	}

	eventNamespace := &v1.Namespace{}
	key := client.ObjectKey{
		Namespace: "",
		Name:      namespace,
	}
	err := k8sClient.Get(ctx, key, eventNamespace)
	if err != nil {
		return false, err
	}

	namespaceLabels := eventNamespace.GetLabels()

	if val, ok := namespaceLabels[mesh.AutoInjectLabel]; ok && val == mesh.Enabled {
		return true, nil
	}

	return false, nil
}
