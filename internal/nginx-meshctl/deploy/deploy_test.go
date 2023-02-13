package deploy_test

import (
	"os"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gopkg.in/yaml.v2"
	"helm.sh/helm/v3/pkg/chart/loader"
	v1 "k8s.io/api/core/v1"

	"github.com/nginxinc/nginx-service-mesh/internal/nginx-meshctl/deploy"
	"github.com/nginxinc/nginx-service-mesh/pkg/helm"
	"github.com/nginxinc/nginx-service-mesh/pkg/k8s"
	"github.com/nginxinc/nginx-service-mesh/pkg/k8s/fake"
)

var _ = Describe("Deploy", func() {
	var files []*loader.BufferedFile
	var deployer *deploy.Deployer
	var fakeK8s k8s.Client

	BeforeEach(func() {
		var err error
		files, _, err = helm.GetBufferedFilesAndValues()
		Expect(err).ToNot(HaveOccurred())
		shouldSkipRelease := false

		fakeK8s = fake.NewFakeK8s(v1.NamespaceDefault, shouldSkipRelease)
		deployer = deploy.NewDeployer(files, nil, fakeK8s, true)
	})

	It("deploys without error", func() {
		// mute stdout for this test since it will print the entire manifest
		stdout := os.Stdout
		defer func() { os.Stdout = stdout }()
		os.Stdout = os.NewFile(0, os.DevNull)

		var values helm.Values
		Expect(yaml.Unmarshal(valuesYaml, &values)).To(Succeed())
		deployer.Values = &values
		// default deploy values are not valid
		// we need to set either tracing/telemetry to nil
		values.Telemetry = nil
		values.Tracing = nil
		_, err := deployer.Deploy()
		Expect(err).ToNot(HaveOccurred())
	})

	It("deploys air-gapped mesh", func() {
		// mute stdout for this test since it will print the entire manifest
		stdout := os.Stdout
		defer func() { os.Stdout = stdout }()
		os.Stdout = os.NewFile(0, os.DevNull)

		var values helm.Values
		Expect(yaml.Unmarshal(valuesYaml, &values)).To(Succeed())
		deployer.Values = &values

		// default deploy values are not valid
		// we need to set tracing and telemetry to nil
		values.Telemetry = nil
		values.Tracing = nil

		// Set up an air-gapped deployment.
		values.Registry.Server = "test-registry"
		values.Registry.DisablePublicImages = true
		manifest, err := deployer.Deploy()
		Expect(err).ToNot(HaveOccurred())

		valid := isImageSourceValid(manifest, values.Registry.Server)
		Expect(valid).To(BeTrue())

		// Set up a non air-gapped deployment.
		values.Registry.DisablePublicImages = false
		manifest, err = deployer.Deploy()
		Expect(err).ToNot(HaveOccurred())

		valid = isImageSourceValid(manifest, values.Registry.Server)
		Expect(valid).To(BeFalse())
	})

	Context("input validation", func() {
		It("validates mtls input", func() {
			ttlValues := []string{"invalid", "00m", "01h", "2.3m", "4", "56", "7s", "1234567m", "h", "m", "mm", "hh"}
			for i := range ttlValues {
				values := &helm.Values{
					MTLS: helm.MTLS{
						Mode:              "invalid",
						CATTL:             ttlValues[i],
						SVIDTTL:           ttlValues[i],
						CAKeyType:         "invalid",
						PersistentStorage: "invalid",
					},
				}
				deployer.Values = values
				_, err := deployer.Deploy()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("mtls.mode must be one of the following"))
				Expect(err.Error()).To(ContainSubstring("mtls.caTTL: Does not match pattern"))
				Expect(err.Error()).To(ContainSubstring("mtls.svidTTL: Does not match pattern"))
				Expect(err.Error()).To(ContainSubstring("mtls.persistentStorage must be one of the following"))
				Expect(err.Error()).To(ContainSubstring("mtls.caKeyType must be one of the following"))
			}
		})

		It("validates nginx fields", func() {
			values := &helm.Values{
				NGINXErrorLogLevel: "invalid",
				NGINXLogFormat:     "invalid",
				NGINXLBMethod:      "invalid",
			}
			deployer.Values = values
			_, err := deployer.Deploy()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("nginxErrorLogLevel must be one of the following"))
			Expect(err.Error()).To(ContainSubstring("nginxLogFormat must be one of the following"))
			Expect(err.Error()).To(ContainSubstring("nginxLBMethod must be one of the following"))
		})

		It("validates environment field", func() {
			values := &helm.Values{
				Environment: "invalid",
			}
			deployer.Values = values
			_, err := deployer.Deploy()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("environment must be one of the following"))
		})

		It("validates accessControlMode field", func() {
			values := &helm.Values{
				AccessControlMode: "invalid",
			}
			deployer.Values = values
			_, err := deployer.Deploy()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("accessControlMode must be one of the following"))
		})

		It("validates telemetry fields", func() {
			values := &helm.Values{
				DisableAutoInjection: true,
				Telemetry: &helm.Telemetry{
					SamplerRatio: 5.0,
				},
				Tracing: nil,
			}
			values.DisableAutoInjection = true
			values.EnabledNamespaces = []string{"default"}
			deployer.Values = values
			_, err := deployer.Deploy()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("telemetry.samplerRatio: Must be less than or equal to 1"))
		})
	})
})

func isImageSourceValid(manifest, registry string) bool {
	for _, line := range strings.Split(manifest, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "image:") {
			imageTarget := strings.Split(line, ": ")[1]
			if !strings.Contains(imageTarget, registry) {
				return false
			}
		}
	}

	return true
}
