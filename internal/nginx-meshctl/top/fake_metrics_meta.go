package top

import (
	tm "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/metrics/v1alpha1"
)

// FakeMetricsMeta stubs the call to the apiservice, but otherwise operates the same
// as object.
type FakeMetricsMeta struct {
	returnList *tm.TrafficMetricsList
	MetricsMeta
}

// GetName gets the name referenced in the Object.
func (f *FakeMetricsMeta) GetName() string {
	return f.MetricsMeta.GetName()
}

// GetDisplayName gets the display name referenced in the Object.
func (f *FakeMetricsMeta) GetDisplayName() string {
	return f.MetricsMeta.GetDisplayName()
}

// GetMetricsList returns the trafficMetricsList defined in the FakeMetricsMeta.
func (f *FakeMetricsMeta) GetMetricsList() (*tm.TrafficMetricsList, error) {
	return f.returnList, nil
}
