package mesh

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"sigs.k8s.io/controller-runtime/pkg/client"

	specs "github.com/nginxinc/nginx-service-mesh/pkg/apis/specs/v1alpha1"
)

// CheckForInvalidConfig returns an error if config is not valid
// Invalid configs:
//   - AutoInjection is disabled but there are disabled namespaces
//   - AutoInjection is enabled but there are enabled namespaces
//   - LoadBalancingMethod is "random" when CircuitBreakers exist
//   - both tracing and telemetry are enabled
//
//nolint:goerr113 // can convert to constants at some point if desired
func (config *MeshConfig) CheckForInvalidConfig(k8sClient client.Client) error {
	if *config.IsAutoInjectEnabled && len(*config.EnabledNamespaces) > 0 {
		return errors.New("invalid configuration: enabled namespaces should not be set " +
			"when auto injection is enabled")
	}
	if !*config.IsAutoInjectEnabled && len(*config.Injection.DisabledNamespaces) > 0 {
		return errors.New("invalid configuration: disabled namespaces should not be set " +
			"when auto injection is disabled")
	}

	if k8sClient != nil && strings.Contains(string(config.LoadBalancingMethod), string(MeshConfigLoadBalancingMethodRandom)) {
		cbs := &specs.CircuitBreakerList{}
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		if err := k8sClient.List(ctx, cbs); err != nil {
			return fmt.Errorf("could not list CircuitBreakers while verifying load balancing method: %w", err)
		}

		if len(cbs.Items) > 0 {
			return errors.New("invalid configuration: load balancing method cannot be random when " +
				"CircuitBreaker resources exist")
		}
	}

	if config.Tracing.BackendAddress != nil && *config.Tracing.BackendAddress != "" {
		if config.Tracing.Backend == nil || *config.Tracing.Backend == "" {
			return errors.New("invalid configuration: if tracing.backendAddress is specified " +
				"tracing.backend must also be specified")
		}
		if config.Telemetry.Exporters != nil {
			return errors.New("invalid configuration: tracing and telemetry cannot both be enabled. " +
				"Either set 'tracing.isEnabled' to 'false', set 'tracing' to an empty object, or set 'telemetry' " +
				"to an empty object")
		}
	} else if config.Tracing.Backend != nil && *config.Tracing.Backend != "" {
		return errors.New("invalid configuration: if tracing.backend is specified " +
			"tracing.backendAddress must also be specified")
	}

	return nil
}
