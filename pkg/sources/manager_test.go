// Copyright: meitu.com
// Author: JamesBryce
// Author Email: zmp@meitu.com
// Date: 2018-1-12
// Last Modified by:   JamesBryce

package sources_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	fakesrc "github.com/kubernetes-incubator/custom-metrics-apiserver/pkg/fake"
	. "github.com/kubernetes-incubator/custom-metrics-apiserver/pkg/sources"
)

func TestSourceManager(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Source Manager Suite")
}

// sleepySource returns a MetricSource that takes some amount of time (respecting
// context timeouts) to collect a MetricsBatch with a single node's data point.
func sleepySource(delay time.Duration, nodeName string, point *CustomMetricsPoint) MetricSource {
	return &fakesrc.FunctionSource{
		SourceName: "sleepy_source:" + nodeName,
		GenerateBatch: func(ctx context.Context) (*CustomMetricsBatch, error) {
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return nil, fmt.Errorf("timed out")
			}
			batch := NewCustomMetricsBatch()
			batch.Nodes.Set("", nodeName, point)
			return batch, nil
		},
	}
}

func fullSource(ts time.Time, nodeInd, podStartInd, numPods int) MetricSource {
	return &fakesrc.FunctionSource{
		SourceName: fmt.Sprintf("static_source:node%d", nodeInd),
		GenerateBatch: func(_ context.Context) (*CustomMetricsBatch, error) {
			batch := NewCustomMetricsBatch()
			pointer := CustomMetricsPoint{
				Timestamp: ts,
				Usage: ResourceList{
					corev1.ResourceCPU:    *resource.NewQuantity(100+10*int64(nodeInd), resource.DecimalSI),
					corev1.ResourceMemory: *resource.NewQuantity(200+10*int64(nodeInd), resource.DecimalSI),
				},
			}
			batch.Nodes.Set("", fmt.Sprintf("node%d", nodeInd), &pointer)
			for i := 0; i < numPods; i++ {
				podInd := int64(podStartInd + i)
				name := fmt.Sprintf("pod%d", podInd)
				namespace := fmt.Sprintf("ns%d", nodeInd)
				pointer := CustomMetricsPoint{
					Timestamp: ts,
					Usage: ResourceList{
						corev1.ResourceCPU:    *resource.NewQuantity(300+10*podInd, resource.DecimalSI),
						corev1.ResourceMemory: *resource.NewQuantity(400+10*podInd, resource.DecimalSI),
					},
				}
				batch.Pods.Set(namespace, name, &pointer)
			}
			return batch, nil
		},
	}
}

var _ = Describe("Source Manager", func() {
	var (
		scrapeTime    = time.Now()
		nodeDataPoint = CustomMetricsPoint{
			Timestamp: scrapeTime,
			Usage: ResourceList{
				corev1.ResourceCPU:    *resource.NewQuantity(100, resource.DecimalSI),
				corev1.ResourceMemory: *resource.NewQuantity(200, resource.DecimalSI),
			},
		}
	)

	Context("when all sources return in time", func() {
		It("should return the results of all sources, both pods and nodes", func() {
			By("setting up sources that take 1 second to complete")
			metricsSourceProvider := fakesrc.StaticSourceProvider{
				sleepySource(500*time.Millisecond, "node1", &nodeDataPoint),
				sleepySource(500*time.Millisecond, "node2", &nodeDataPoint),
			}

			By("running the source manager with a scrape and context timeout of 1*seconds")
			start := time.Now()
			manager := NewSourceManager(1*time.Second, metricsSourceProvider)
			timeoutCtx, doneWithWork := context.WithTimeout(context.Background(), 1*time.Second)
			dataBatch, errs := manager.Collect(timeoutCtx)
			doneWithWork()
			Expect(errs).NotTo(HaveOccurred())

			By("ensuring that the full time took at most 1 seconds")
			Expect(time.Now().Sub(start)).To(BeNumerically("<=", 1*time.Second))

			// You have to do it twice. Because the first collection point is ignored
			timeoutCtx, doneWithWork = context.WithTimeout(context.Background(), 1*time.Second)
			dataBatch, errs = manager.Collect(timeoutCtx)
			doneWithWork()
			Expect(errs).NotTo(HaveOccurred())

			By("ensuring that all the nodes are listed")
			// fmt.Printf("\n%+v\n", dataBatch.Nodes)
			Expect(*dataBatch.Nodes.Get(NullNamespace, "node1")).To(Equal(nodeDataPoint))
			Expect(*dataBatch.Nodes.Get(NullNamespace, "node2")).To(Equal(nodeDataPoint))
		})

		It("should return the results of all sources' nodes and pods", func() {
			By("setting up multiple sources")
			metricsSourceProvider := fakesrc.StaticSourceProvider{
				fullSource(scrapeTime, 1, 0, 4),
				fullSource(scrapeTime, 2, 4, 2),
				fullSource(scrapeTime, 3, 6, 1),
			}

			By("running the source manager")
			manager := NewSourceManager(1*time.Second, metricsSourceProvider)
			dataBatch, errs := manager.Collect(context.Background())
			Expect(errs).NotTo(HaveOccurred())

			dataBatch, errs = manager.Collect(context.Background())
			Expect(errs).NotTo(HaveOccurred())

			By("figuring out the expected node and pod points")
			// var expectedNodePoints []interface{}
			// var expectedPodPoints []interface{}
			expectBatch := NewCustomMetricsBatch()
			for _, src := range metricsSourceProvider {
				res, err := src.Collect(context.Background())
				Expect(err).NotTo(HaveOccurred())
				expectBatch.Merge(res)
			}

			By("ensuring that all nodes are present")
			ok := dataBatch.Nodes.Equal(expectBatch.Nodes)
			Expect(ok).To(BeTrue())

			By("ensuring that all pods are present")
			ok = dataBatch.Pods.Equal(expectBatch.Pods)
			Expect(ok).To(BeTrue())
		})
	})

	Context("when some sources take too long", func() {
		It("should pass the scrape timeout to the source context, so that sources can time out", func() {
			By("setting up one source to take 4 seconds, and another to take 2")
			metricsSourceProvider := fakesrc.StaticSourceProvider{
				sleepySource(2*time.Second, "node1", &nodeDataPoint),
				sleepySource(500*time.Millisecond, "node2", &nodeDataPoint),
			}

			By("running the source manager with a scrape timeout of 3 seconds")
			start := time.Now()
			manager := NewSourceManager(1*time.Second, metricsSourceProvider)
			dataBatch, errs := manager.Collect(context.Background())

			By("ensuring that scraping took around 3 seconds")
			Expect(time.Now().Sub(start)).To(BeNumerically("~", 1*time.Second, 10*time.Millisecond))
			dataBatch, errs = manager.Collect(context.Background())

			By("ensuring that an error and partial results (data from source 2) were returned")
			Expect(errs).To(HaveOccurred())
			Expect(*dataBatch.Nodes.Get(NullNamespace, "node2")).To(Equal(nodeDataPoint))

		})

		It("should respect the parent context's general timeout, even with a longer scrape timeout", func() {
			By("setting up some sources with 4 second delays")
			metricsSourceProvider := fakesrc.StaticSourceProvider{
				sleepySource(4*time.Second, "node1", &nodeDataPoint),
				sleepySource(4*time.Second, "node2", &nodeDataPoint),
			}

			By("running the source manager with a scrape timeout of 5 seconds, but a context timeout of 1 second")
			start := time.Now()
			manager := NewSourceManager(5*time.Second, metricsSourceProvider)
			timeoutCtx, doneWithWork := context.WithTimeout(context.Background(), 1*time.Second)
			dataBatch, errs := manager.Collect(timeoutCtx)
			doneWithWork()

			By("ensuring that it times out after 1 second with errors and no data")
			Expect(time.Now().Sub(start)).To(BeNumerically("~", 1*time.Second, 10*time.Millisecond))
			timeoutCtx, doneWithWork = context.WithTimeout(context.Background(), 1*time.Second)
			dataBatch, errs = manager.Collect(timeoutCtx)
			doneWithWork()

			Expect(errs).To(HaveOccurred())
			Expect(dataBatch.Nodes).To(BeEmpty())
		})
	})
})
