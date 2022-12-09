package top

import (
	"bytes"
	"fmt"
	"os"
	"text/tabwriter"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	tm "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/metrics/v1alpha1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Top", func() {
	var (
		fakeObj   *FakeMetricsMeta
		output    *bytes.Buffer
		tabWriter *tabwriter.Writer
	)
	BeforeEach(func() {
		fakeObj = &FakeMetricsMeta{
			returnList: &tm.TrafficMetricsList{
				Items: []*tm.TrafficMetrics{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "test-src",
						},
						Edge: &tm.Edge{
							Direction: tm.To,
							Resource: &v1.ObjectReference{
								Name: "test-dest",
							},
						},
						Metrics: []*tm.Metric{
							{
								Name:  "p99_response_latency",
								Value: resource.NewScaledQuantity(2000, -3),
							},
							{
								Name:  "p90_response_latency",
								Value: resource.NewScaledQuantity(2000, -3),
							},
							{
								Name:  "p50_response_latency",
								Value: resource.NewScaledQuantity(2000, -3),
							},
							{
								Name:  "success_count",
								Value: resource.NewScaledQuantity(2000, -3),
							},
							{
								Name:  "failure_count",
								Value: resource.NewScaledQuantity(2000, -3),
							},
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "test-src",
						},
						Edge: &tm.Edge{
							Direction: tm.From,
							Resource: &v1.ObjectReference{
								Name: "test-dest",
							},
						},
						Metrics: []*tm.Metric{
							{
								Name:  "p99_response_latency",
								Value: resource.NewScaledQuantity(2000, -3),
							},
							{
								Name:  "p90_response_latency",
								Value: resource.NewScaledQuantity(2000, -3),
							},
							{
								Name:  "p50_response_latency",
								Value: resource.NewScaledQuantity(2000, -3),
							},
							{
								Name:  "success_count",
								Value: resource.NewScaledQuantity(2000, -3),
							},
							{
								Name:  "failure_count",
								Value: resource.NewScaledQuantity(2000, -3),
							},
						},
					},
				},
			},
			MetricsMeta: MetricsMeta{
				DisplayName: "Test",
			},
		}
		output = new(bytes.Buffer)
		tabWriter = tabWriterWithOpts()
	})
	When("not provided an object name", func() {
		It("formats metrics", func() {
			var expOutput bytes.Buffer
			testWriter := tabWriterWithOpts()
			fmt.Fprintf(testWriter, "Test\tIncoming Success\tOutgoing Success\tNumRequests\n")
			fmt.Fprintf(testWriter, "test-src\t50.00%%\t50.00%%\t8\n\n") //nolint:dupword // allow for this test
			err := testWriter.Flush()
			Expect(err).ToNot(HaveOccurred())

			err = BuildTopMetrics(tabWriter, fakeObj)
			Expect(err).ToNot(HaveOccurred())
			err = tabWriter.Flush()
			Expect(err).ToNot(HaveOccurred())
			Expect(output.String()).To(Equal(expOutput.String()))
		})
	})
	When("provided an object name", func() {
		It("formats metrics", func() {
			var expOutput bytes.Buffer
			testWriter := tabWriterWithOpts()
			fmt.Fprintf(testWriter, "Test\tDirection\tResource\tSuccess Rate\tP99\tP90\tP50\tNumRequests\n")
			fmt.Fprintf(testWriter, "test\t\t\t\t\t\t\t\t\t\t\t\t\t\t\n")
			fmt.Fprintf(testWriter, "\tTo\ttest-dest\t50.00%%\t2ms\t2ms\t2ms\t4\n")   //nolint:dupword // allow for this test
			fmt.Fprintf(testWriter, "\tFrom\ttest-dest\t50.00%%\t2ms\t2ms\t2ms\t4\n") //nolint:dupword // allow for this test
			err := testWriter.Flush()
			Expect(err).ToNot(HaveOccurred())

			fakeObj.ObjectReference.Name = "test"
			err = BuildTopMetrics(tabWriter, fakeObj)
			Expect(err).ToNot(HaveOccurred())
			err = tabWriter.Flush()
			Expect(err).ToNot(HaveOccurred())
			Expect(output.String()).To(Equal(expOutput.String()))
		})
	})
})

func tabWriterWithOpts() *tabwriter.Writer {
	return tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
}
