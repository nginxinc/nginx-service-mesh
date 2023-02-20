package inject_test

import (
	"encoding/json"
	"os"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/yaml"

	"github.com/nginxinc/nginx-service-mesh/pkg/apis/mesh"
	"github.com/nginxinc/nginx-service-mesh/pkg/inject"
)

var _ = Describe("Inject", func() {
	var (
		meshConfig   mesh.FullMeshConfig
		injectConfig inject.Inject

		// cheap gosec remediation
		readFile func(string) ([]byte, error)
	)
	BeforeEach(func() {
		meshConfig = mesh.FullMeshConfig{
			Registry: mesh.Registry{
				SidecarImage:     "docker-registry/nginx-mesh-sidecar:latest",
				SidecarInitImage: "docker-registry/nginx-mesh-init:latest",
			},
			Mtls: mesh.Mtls{
				Mode: mesh.MtlsModePermissive,
			},
		}
		injectConfig = inject.Inject{}

		readFile = os.ReadFile
	})

	It("injects all resources", func() {
		resList := []string{
			"testdata/resources.yaml",
			"testdata/resources.json",
			"testdata/resource.json",
			"testdata/resource.yaml",
		}
		for _, resFile := range resList {
			resources, err := readFile(resFile)
			Expect(err).ToNot(HaveOccurred())
			injectConfig.Resources = resources
			allInjected, err := inject.IntoFile(injectConfig, meshConfig)
			Expect(err).ToNot(HaveOccurred())

			// The number of "containers" strings will tell us how many resources
			// we should have expected to be injected
			expCount := strings.Count(allInjected, "containers")
			Expect(expCount).ToNot(BeZero())
			// Check for the existence of our proxy image
			proxyCount := strings.Count(allInjected, "docker-registry/nginx-mesh-sidecar:latest")
			Expect(proxyCount).To(Equal(expCount))
			// Check for resource label (only for list object, since this covers everything)
			if strings.Contains(resFile, "resources") {
				var list v1.List
				jsonBytes := []byte(allInjected)
				if strings.Contains(resFile, "yaml") {
					jsonBytes, err = yaml.YAMLToJSON([]byte(allInjected))
					Expect(err).ToNot(HaveOccurred())
				}
				Expect(json.Unmarshal(jsonBytes, &list)).To(Succeed())
				for _, item := range list.Items {
					raw := string(item.Raw)
					split := strings.Split(raw, "\n")
					var kindLine string
					for _, line := range split {
						if strings.Contains(line, "kind") {
							kindLine = line

							break
						}
					}
					trimmed := strings.TrimPrefix(strings.TrimSpace(kindLine), "\"kind\": \"")
					kind := strings.ToLower(strings.TrimSuffix(trimmed, "\","))
					if kind == "service" {
						continue
					}

					Expect(raw).To(ContainSubstring(mesh.DeployLabel + kind))
				}
			}
		}
	})
	It("errors with non-yaml config", func() {
		notYaml := `blarg
	this is not yaml:
	:
`
		injectConfig.Resources = []byte(notYaml)
		_, err := inject.IntoFile(injectConfig, meshConfig)
		Expect(err).To(HaveOccurred(), "should have gotten error with non-yaml config")
	})
	It("errors with non-k8s config", func() {
		notK8s := `
	this: yaml
	butNot: deployment
`
		injectConfig.Resources = []byte(notK8s)
		_, err := inject.IntoFile(injectConfig, meshConfig)
		Expect(err).To(HaveOccurred(), "should have gotten error with non-k8s config")
	})
	It("errors when mtls annotation conflicts with strict mode", func() {
		meshConfig.Mtls.Mode = mesh.MtlsModeStrict
		invalid := `{"apiVersion": "apps/v1",
"kind": "Deployment",
"metadata": {
  "name": "foo"
},
"spec": {
"template": {
	"metadata": {
		"annotations": {
			"config.nsm.nginx.com/mtls-mode": "off"
		}
	}
}
}}`
		injectConfig.Resources = []byte(invalid)
		_, err := inject.IntoFile(injectConfig, meshConfig)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("global mtls mode is 'strict'"))
	})
	It("injects valid config", func() {
		valid := `apiVersion: v1
kind: ConfigMap
metadata:
name: mm
---
apiVersion: apps/v1
kind: Deployment
metadata:
name: foo
spec:
replicas: 3
`
		injectConfig.Resources = []byte(valid)
		cfg, err := inject.IntoFile(injectConfig, meshConfig)
		Expect(err).ToNot(HaveOccurred())
		Expect(cfg).To(ContainSubstring("ConfigMap"),
			"even documents with non-deployments should be added to mutate output")

		validCount := strings.Count(cfg, "docker-registry")
		Expect(validCount).To(Equal(2))
	})
	It("errors when container ports are invalid", func() {
		resources, err := os.ReadFile("testdata/unsupportedSCTPContainerPort.yaml")
		Expect(err).ToNot(HaveOccurred())
		injectConfig.Resources = resources
		_, err = inject.IntoFile(injectConfig, meshConfig)
		Expect(err).To(HaveOccurred())

		resources, err = os.ReadFile("testdata/duplicateContainerPorts.yaml")
		Expect(err).ToNot(HaveOccurred())
		injectConfig.Resources = resources
		_, err = inject.IntoFile(injectConfig, meshConfig)
		Expect(err).To(HaveOccurred())
	})
})
