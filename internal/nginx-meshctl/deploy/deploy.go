// Package deploy is responsible for deploying NGINX Service Mesh
package deploy

import (
	"fmt"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"

	meshErrors "github.com/nginxinc/nginx-service-mesh/pkg/errors"
	"github.com/nginxinc/nginx-service-mesh/pkg/helm"
	"github.com/nginxinc/nginx-service-mesh/pkg/k8s"
)

// Deployer deploys the mesh using helm charts.
type Deployer struct {
	k8sClient k8s.Client
	Values    *helm.Values
	files     []*loader.BufferedFile
	DryRun    bool
}

// NewDeployer returns a new Deployer object.
func NewDeployer(
	files []*loader.BufferedFile,
	values *helm.Values,
	k8sClient k8s.Client,
	dryRun bool,
) *Deployer {
	return &Deployer{
		files:     files,
		Values:    values,
		k8sClient: k8sClient,
		DryRun:    dryRun,
	}
}

// Deploy the mesh using Helm.
func (d *Deployer) Deploy() (string, error) {
	valuesMap, err := d.Values.ConvertToMap()
	if err != nil {
		return "", err
	}
	chart, err := loader.LoadFiles(d.files)
	if err != nil {
		return "", fmt.Errorf("error loading helm files: %w", err)
	}

	// check if mesh exists
	exists, existsErr := d.k8sClient.MeshExists()
	if existsErr != nil {
		if exists {
			return "", fmt.Errorf("%w. To remove the existing mesh, run \"nginx-meshctl remove\"", existsErr)
		}

		return "", fmt.Errorf("%v: %w", meshErrors.ErrCheckingExistence, existsErr)
	}
	fmt.Println("Deploying NGINX Service Mesh...")

	actionConfig, err := d.k8sClient.HelmAction(d.k8sClient.Namespace())
	if err != nil {
		return "", fmt.Errorf("error initializing helm action: %w", err)
	}

	installer := action.NewInstall(actionConfig)
	installer.Namespace = d.k8sClient.Namespace()
	installer.CreateNamespace = true
	installer.ReleaseName = "nginx-service-mesh"
	installer.DryRun = d.DryRun

	rel, err := installer.Run(chart, valuesMap)
	if err != nil {
		return "", fmt.Errorf("error installing NGINX Service Mesh: %w", err)
	}

	if d.DryRun {
		fmt.Println(rel.Manifest)

		return rel.Manifest, nil
	}

	return rel.Manifest, nil
}
