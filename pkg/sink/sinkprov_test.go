// Copyright 2018 The Meitu Authors.
// Author JamesBryce
// Author Email zmp@meitu.com
// Date 2018-1-12
// Last Modified by JamesBryce

package sink_test

import (
	"fmt"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/kubernetes-incubator/custom-metrics-apiserver/pkg/sink"
	. "github.com/kubernetes-incubator/custom-metrics-apiserver/pkg/sink"
	"github.com/kubernetes-incubator/custom-metrics-apiserver/pkg/sources"
)

var defaultWindow = 30 * time.Second

func TestSourceManager(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Provider/Sink Suite")
}

// func newMilliPoint(ts time.Time, cpu, memory int64) sources.MetricsPoint {
// 	return sources.MetricsPoint{
// 		Timestamp:   ts,
// 		CpuUsage:    *resource.NewMilliQuantity(cpu, resource.DecimalSI),
// 		MemoryUsage: *resource.NewMilliQuantity(memory, resource.BinarySI),
// 	}
// }

func buildBatch(ts time.Time, nodeInd, podStartInd, endpointStartInd, num int) (*sources.CustomMetricsBatch, error) {
	batch := sources.NewCustomMetricsBatch()
	total := 4
	pointer := sources.CustomMetricsPoint{
		Timestamp:  ts,
		WindowTime: time.Second * 10,
		Usage: sources.ResourceList{
			corev1.ResourceCPU:             *resource.NewQuantity(100+10*int64(nodeInd), resource.DecimalSI),
			corev1.ResourceMemory:          *resource.NewQuantity(200+10*int64(nodeInd), resource.BinarySI),
			sources.ResourceNetworkRxBytes: *resource.NewQuantity(300+10*int64(nodeInd), resource.BinarySI),
			sources.ResourceNetworkTxBytes: *resource.NewQuantity(400+10*int64(nodeInd), resource.BinarySI),
		},
	}
	batch.Nodes.Set("", fmt.Sprintf("node%d", nodeInd), &pointer)
	for i := 0; i < num; i++ {
		podInd := podStartInd + i
		name := fmt.Sprintf("pod%d", podInd)
		namespace := fmt.Sprintf("ns%d", podInd)
		pointer := sources.CustomMetricsPoint{
			Timestamp:  ts,
			WindowTime: time.Second * 10,
			Usage: sources.ResourceList{
				corev1.ResourceCPU:             *resource.NewQuantity(300+10*int64(podInd), resource.DecimalSI),
				corev1.ResourceMemory:          *resource.NewQuantity(400+10*int64(podInd), resource.BinarySI),
				sources.ResourceNetworkRxBytes: *resource.NewQuantity(300+10*int64(podInd), resource.BinarySI),
				sources.ResourceNetworkTxBytes: *resource.NewQuantity(400+10*int64(podInd), resource.BinarySI),
			},
		}
		batch.Pods.Set(namespace, name, &pointer)
		total += 4
	}
	for i := 0; i < num; i++ {
		Ind := endpointStartInd + i
		name := fmt.Sprintf("pod%d", Ind)
		namespace := fmt.Sprintf("ns%d", Ind)
		pointer := sources.CustomMetricsPoint{
			Timestamp:  ts,
			WindowTime: time.Second * 10,
			Usage: sources.ResourceList{
				sources.ResourceRequests: *resource.NewQuantity(500+10*int64(Ind), resource.BinarySI),
			},
		}
		batch.EndpointQPSs.Set(namespace, name, &pointer)
		total += 1
	}
	batch.Total = int64(total)
	return batch, nil
}

var _ = Describe("In-memory Sink Provider", func() {
	var (
		batch    *sources.CustomMetricsBatch
		provRise sink.MetricRise
		provSink sink.MetricSink
		now      time.Time
	)

	BeforeEach(func() {
		now = time.Now()
		var err error
		batch, err = buildBatch(now, 1, 1, 2, 2)
		Expect(err).NotTo(HaveOccurred())
		provSink, provRise = NewSinkProvider()
	})

	It("should receive batches of metrics", func() {
		By("sending the batch to the sink")
		Expect(provSink.Receive(batch)).To(Succeed())

		By("making sure that the provider contains all nodes received")
		for _, objs := range batch.Nodes {
			for name := range objs {
				_, _, _, err := provRise.GetValue("nodes", "", name, "cpu")
				Expect(err).NotTo(HaveOccurred())
			}

		}

		By("making sure that the provider contains all pods received")
		for namespace, pod := range batch.Pods {
			for name, point := range pod {
				ts, du, value, err := provRise.GetValue("pods", namespace, name, "cpu")
				Expect(err).NotTo(HaveOccurred())
				Expect(ts).To(Equal(point.Timestamp))
				Expect(du).To(Equal(point.WindowTime))
				Expect(value).To(Equal(point.Usage["cpu"]))
			}
		}
		for namespace, ep := range batch.EndpointQPSs {
			for name, point := range ep {
				ts, du, value, err := provRise.GetValue("endpoints", namespace, name, sources.ResourceRequests.String())
				Expect(err).NotTo(HaveOccurred())
				Expect(ts).To(Equal(point.Timestamp))
				Expect(du).To(Equal(point.WindowTime))
				Expect(value).To(Equal(point.Usage["requests"]))
			}
		}
	})

	It("should return nil metrics for missing pods", func() {
		By("sending the batch to the sink")
		Expect(provSink.Receive(batch)).To(Succeed())

		By("fetching the a present pod and a missing pod")
		_, _, _, err := provRise.GetValue("pods", "ns1", "pod1i65", "cpu")
		Expect(err).To(HaveOccurred())

		By("fetching the a present pod and a missing metrics")
		_, _, _, err = provRise.GetValue("pods", "ns1", "pod1", "cpuxxxx")
		Expect(err).To(HaveOccurred())

		By("fetching the a metrics lost groupresource")
		_, _, _, err = provRise.GetValue("podnotfound", "ns1", "pod1", "cpu")
		Expect(err).To(HaveOccurred())
	})

	It("should get the count", func() {
		By("sending the batch to the sink")
		Expect(provSink.Receive(batch)).To(Succeed())

		By("fetching the count")
		total := provRise.Count()
		Expect(total).To(Equal(batch.Total))
	})

	It("should  goes through all the indices", func() {
		By("sending the batch to the sink")
		Expect(provSink.Receive(batch)).To(Succeed())

		By("fetching all metrics")
		total := int64(0)
		provRise.Foreach(func(GroupResource, namespace, object, metrics string,
			timestamp time.Time, window time.Duration, value resource.Quantity) {
			total += 1
			var point *sources.CustomMetricsPoint
			var ok bool
			switch GroupResource {
			case "pods":
				point, ok = batch.Pods[namespace][object]
			case "nodes":
				point, ok = batch.Nodes[namespace][object]
			case "endpoints":
				point, ok = batch.EndpointQPSs[namespace][object]
			}
			if !ok {
				fmt.Printf("\nNot found the :%+v, %v, %v\n", GroupResource, namespace, object)
			}
			Expect(ok).To(BeTrue())
			Expect(timestamp).To(Equal(point.Timestamp))
			Expect(window).To(Equal(point.WindowTime))
			Expect(value).To(Equal(point.Usage[corev1.ResourceName(metrics)]))
		})
		Expect(total).To(Equal(batch.Total))

	})

})
