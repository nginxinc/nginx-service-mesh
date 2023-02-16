// Package pod contains information about a pod resource.
package pod

import (
	"context"
	"fmt"
	"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/nginxinc/nginx-service-mesh/pkg/apis/mesh"
)

const replicaset = "replicaset"

// IsInjected checks if a pod is injected.
func IsInjected(pod *v1.Pod) bool {
	val, ok := pod.Annotations[mesh.InjectedAnnotation]

	return ok && strings.ToLower(val) == mesh.Injected
}

// GetOwner gets a pod's owner type and name.
func GetOwner(ctx context.Context, k8sClient client.Client, pod *v1.Pod) (string, string, error) {
	ownerName := pod.Name
	for _, owner := range pod.GetOwnerReferences() {
		if owner.Controller == nil || !*owner.Controller {
			continue
		}
		ownerType := strings.ToLower(owner.Kind)
		ownerName = owner.Name
		if ownerType == replicaset {
			var err error
			// Get the owner replicaset. Retry for 10 seconds if the Get fails.
			for i := 0; i < 10; i++ {
				ownerType, ownerName, err = GetReplicaSetOwner(ctx, k8sClient, pod.Namespace, owner.Name)
				if err != nil {
					time.Sleep(1 * time.Second)

					continue
				}

				break
			}
			if err != nil {
				return ownerType, ownerName, fmt.Errorf("unable to determine top-level owner for pod: %w", err)
			}
		}

		return ownerType, ownerName, nil
	}

	return "pod", ownerName, nil
}

// GetReplicaSetOwner returns a ReplicaSet's owner if it exists, otherwise just return "replicaset".
func GetReplicaSetOwner(
	ctx context.Context,
	k8sClient client.Client,
	namespace,
	name string,
) (string, string, error) {
	var replicas appsv1.ReplicaSet
	key := client.ObjectKey{Namespace: namespace, Name: name}
	err := k8sClient.Get(ctx, key, &replicas)
	if err != nil {
		return replicaset, name, fmt.Errorf("error getting replicaset: %w", err)
	}

	for _, owner := range replicas.GetOwnerReferences() {
		if owner.Controller != nil && *owner.Controller {
			return strings.ToLower(owner.Kind), owner.Name, nil
		}
	}

	return replicaset, name, nil
}
