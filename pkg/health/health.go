// Package health verifies the health of the mesh.
package health

import (
	"context"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/nginxinc/nginx-service-mesh/pkg/apis/mesh"
	meshErrors "github.com/nginxinc/nginx-service-mesh/pkg/errors"
)

// TestMeshControllerConnection checks that the controller is available and ready for use.
func TestMeshControllerConnection(k8sClient client.Client, namespace string, retries int) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	var err error
	for i := 0; i < retries; i++ {
		if i > 0 {
			time.Sleep(5 * time.Second)
		}

		var ctlr appsv1.Deployment
		if err = k8sClient.Get(ctx, client.ObjectKey{Namespace: namespace, Name: mesh.MeshAPI}, &ctlr); err != nil {
			continue
		}

		if ctlr.Status.ReadyReplicas != 1 {
			err = meshErrors.ErrMeshStatus

			continue
		}

		return nil
	}

	return err
}
