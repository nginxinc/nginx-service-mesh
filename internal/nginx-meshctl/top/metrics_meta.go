package top

import (
	"context"
	"encoding/json"
	"fmt"

	tm "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/metrics/v1alpha1"
	v1 "k8s.io/api/core/v1"
	aggregator "k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset"
)

// MetricsMetaInterface provides an abstraction over Object, so that it may be faked.
type MetricsMetaInterface interface {
	GetName() string
	GetDisplayName() string
	GetMetricsList() (*tm.TrafficMetricsList, error)
}

// MetricsMeta holds information on what resource should be queried and displayed to
// the user.
type MetricsMeta struct {
	Client      *aggregator.Clientset
	DisplayName string
	v1.ObjectReference
}

// GetName returns the name of the object.
func (m *MetricsMeta) GetName() string {
	return m.Name
}

// GetDisplayName returns the display name of the object.
func (m *MetricsMeta) GetDisplayName() string {
	return m.DisplayName
}

// GetMetricsList queries the apiservice for the trafficmetricslist associated
// with the given object.
func (m *MetricsMeta) GetMetricsList() (*tm.TrafficMetricsList, error) {
	path := basePath
	if m.Kind != "namespaces" {
		path += fmt.Sprintf("namespaces/%v/", m.Namespace)
	}
	path += fmt.Sprintf("%v/", m.Kind)
	if m.Name != "" {
		path += fmt.Sprintf("%v/edges", m.Name)
	}
	data, err := m.Client.RESTClient().Get().AbsPath(path).DoRaw(context.TODO())
	if err != nil {
		return nil, fmt.Errorf("failed to get metrics from the Metrics API Server: %w", err)
	}

	var listDeploy tm.TrafficMetricsList
	if err = json.Unmarshal(data, &listDeploy); err != nil {
		return nil, fmt.Errorf("failed to unmarshal deployments list: %w", err)
	}

	return &listDeploy, nil
}
