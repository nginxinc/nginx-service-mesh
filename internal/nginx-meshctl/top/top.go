// Package top things related to metrics the service mesh
package top

import (
	"errors"
	"fmt"
	"io"
	"math"

	tm "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/metrics/v1alpha1"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type successMetric struct {
	outgoing    string
	incoming    string
	numRequests int64
}

var (
	basePath     = "/apis/metrics.smi-spec.io/v1alpha1/"
	errNoMetrics = errors.New("no metrics populated - make sure traffic is flowing")
)

// BuildTopMetrics build metrics by querying the traffic metrics endpoint.
func BuildTopMetrics(writer io.Writer, obj MetricsMetaInterface) error {
	list, err := obj.GetMetricsList()
	if err != nil {
		return err
	}
	if list == nil || len(list.Items) == 0 {
		return errNoMetrics
	}

	if obj.GetName() != "" {
		fmt.Fprintf(writer, "%v\tDirection\tResource\tSuccess Rate\tP99\tP90\tP50\tNumRequests\n",
			obj.GetDisplayName())
		fmt.Fprintf(writer, "%v\t\t\t\t\t\t\t\t\t\t\t\t\t\t\n", obj.GetName())
	} else {
		fmt.Fprintf(writer, "%v\tIncoming Success\tOutgoing Success\tNumRequests\n",
			obj.GetDisplayName())
	}

	if obj.GetName() != "" {
		for _, item := range list.Items {
			if item.Metrics[0].Name != "p99_response_latency" ||
				item.Metrics[1].Name != "p90_response_latency" ||
				item.Metrics[2].Name != "p50_response_latency" ||
				item.Metrics[3].Name != "success_count" ||
				item.Metrics[4].Name != "failure_count" {
				return fmt.Errorf("error formatting item %v. Metric query did not return expected format: %w", item.Name, err)
			}
			successVal := item.Metrics[3].Value
			failVal := item.Metrics[4].Value
			success := float32(successVal.Value()) / float32(successVal.Value()+failVal.Value())
			fmt.Fprintf(writer, "\t%v\t%v\t%.2f%%\t%vms\t%vms\t%vms\t%v\n",
				cases.Title(language.English).String(string(item.Edge.Direction)),
				item.Edge.Resource.Name,
				success*100, //nolint:gomnd // success is the important number
				item.Metrics[0].Value.Value(),
				item.Metrics[1].Value.Value(),
				item.Metrics[2].Value.Value(),
				successVal.Value()+failVal.Value())
		}
	} else {
		successMap := make(map[string]*successMetric)
		for _, item := range list.Items {
			if successMap[item.Name] == nil {
				successMap[item.Name] = &successMetric{}
			}
			if item.Metrics[3].Name != "success_count" ||
				item.Metrics[4].Name != "failure_count" {
				return fmt.Errorf("error formatting item %v. Metric query did not return expected format: %w", item.Name, err)
			}
			successVal := item.Metrics[3].Value
			failVal := item.Metrics[4].Value
			success := float32(successVal.Value()) / float32(successVal.Value()+failVal.Value())
			if item.Edge.Direction == tm.To {
				if math.IsNaN(float64(success)) {
					successMap[item.Name].outgoing = "<nil>"
				} else {
					successMap[item.Name].outgoing = fmt.Sprintf(
						"%.2f%%", success*100) //nolint:gomnd // success is the important number
				}
			} else {
				if math.IsNaN(float64(success)) {
					successMap[item.Name].incoming = "<nil>"
				} else {
					successMap[item.Name].incoming = fmt.Sprintf(
						"%.2f%%", success*100) //nolint:gomnd // success is the important number
				}
			}
			successMap[item.Name].numRequests += successVal.Value() + failVal.Value()
		}
		for key, success := range successMap {
			fmt.Fprintf(writer, "%v\t%v\t%v\t%v\n", key, success.incoming, success.outgoing, success.numRequests)
		}
		fmt.Fprint(writer, "\n")
	}

	return nil
}
