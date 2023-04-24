package mesh

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"sigs.k8s.io/controller-runtime/pkg/client"

	specs "github.com/nginxinc/nginx-service-mesh/pkg/apis/specs/v1alpha1"
)

// ValidateLBMethod ensures the load balancing method is not set to "random" when circuit breakers exist.
func ValidateLBMethod(k8sClient client.Client, lbMethod string) error {
	if strings.Contains(lbMethod, Random) {
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

	return nil
}
