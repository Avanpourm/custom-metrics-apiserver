// Copyright: meitu.com
// Author: JamesBryce
// Author Email: zmp@meitu.com
// Date: 2018-1-12
// Last Modified by:   JamesBryce

package sources_test

import (
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"fmt"

	. "github.com/kubernetes-incubator/custom-metrics-apiserver/pkg/sources"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func TestMetricsBatch(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "metricsbatch Manager Suite")
}

func buildBatch(ts time.Time, nodeInd, podStartInd, addInd, numPods int) (*CustomMetricsBatch, error) {
	batch := NewCustomMetricsBatch()
	pointer := CustomMetricsPoint{
		Timestamp: ts,
		Usage: ResourceList{
			corev1.ResourceCPU:     *resource.NewQuantity(100+10*int64(addInd), resource.DecimalSI),
			corev1.ResourceMemory:  *resource.NewQuantity(200+10*int64(addInd), resource.BinarySI),
			ResourceNetworkRxBytes: *resource.NewQuantity(300+10*int64(addInd), resource.BinarySI),
			ResourceNetworkTxBytes: *resource.NewQuantity(400+10*int64(addInd), resource.BinarySI),
			ResourceRequests:       *resource.NewQuantity(500+10*int64(addInd), resource.BinarySI),
		},
	}
	batch.Nodes.Set("", fmt.Sprintf("node%d", nodeInd), &pointer)
	for i := 0; i < numPods; i++ {
		podInd := int64(podStartInd + i + addInd)
		name := fmt.Sprintf("pod%d", podStartInd+i)
		namespace := fmt.Sprintf("ns%d", podStartInd+i)
		pointer := CustomMetricsPoint{
			Timestamp: ts,
			Usage: ResourceList{
				corev1.ResourceCPU:     *resource.NewQuantity(300+10*podInd, resource.DecimalSI),
				corev1.ResourceMemory:  *resource.NewQuantity(400+10*podInd, resource.BinarySI),
				ResourceNetworkRxBytes: *resource.NewQuantity(300+10*int64(podInd), resource.BinarySI),
				ResourceNetworkTxBytes: *resource.NewQuantity(400+10*int64(podInd), resource.BinarySI),
				ResourceRequests:       *resource.NewQuantity(500+10*int64(podInd), resource.BinarySI),
			},
		}
		batch.Pods.Set(namespace, name, &pointer)
	}
	return batch, nil
}

var _ = Describe("metricsbatch Manager", func() {
	Context("when filter a new CustomMetricsPoint", func() {
		It("should return the rate of resource", func() {
			By("setting up the CustomMetricsPoint")
			start := time.Now()
			scrapttime := start
			history := CustomMetricsPoint{
				Timestamp: scrapttime,
				Usage: ResourceList{
					corev1.ResourceCPU:     *resource.NewQuantity(300, resource.DecimalSI),
					corev1.ResourceMemory:  *resource.NewQuantity(400, resource.BinarySI),
					ResourceNetworkRxBytes: *resource.NewQuantity(500, resource.BinarySI),
					ResourceNetworkTxBytes: *resource.NewQuantity(600, resource.BinarySI),
					ResourceRequests:       *resource.NewQuantity(700, resource.BinarySI),
				},
			}
			secondwait := 2
			scrapttime = start.Add(time.Second * time.Duration(secondwait))
			raw := CustomMetricsPoint{
				Timestamp: scrapttime,
				Usage: ResourceList{
					corev1.ResourceCPU:     *resource.NewQuantity(200, resource.DecimalSI),
					corev1.ResourceMemory:  *resource.NewQuantity(300, resource.BinarySI),
					ResourceNetworkRxBytes: *resource.NewQuantity(510, resource.BinarySI),
					ResourceNetworkTxBytes: *resource.NewQuantity(620, resource.BinarySI),
					ResourceRequests:       *resource.NewQuantity(730, resource.BinarySI),
				},
			}
			expectRst := CustomMetricsPoint{
				Timestamp:  scrapttime,
				WindowTime: time.Second * time.Duration(secondwait),
				Usage: ResourceList{
					corev1.ResourceCPU:     *resource.NewQuantity(200, resource.DecimalSI),
					corev1.ResourceMemory:  *resource.NewQuantity(300, resource.BinarySI),
					ResourceNetworkRxBytes: *resource.NewQuantity(int64(10/secondwait), resource.BinarySI),
					ResourceNetworkTxBytes: *resource.NewQuantity(int64(20/secondwait), resource.BinarySI),
					ResourceRequests:       *resource.NewQuantity(int64(30/secondwait), resource.BinarySI),
				},
			}

			By("running the CustomMetricsFilter to the the reources rate in the new custom metrics")
			actrullyRst, num := raw.RateOfSome(&history)
			ok := actrullyRst.Equal(&expectRst)
			if !ok {
				fmt.Printf("metricsbatch:actrully:\n%+v\nexpect:\n%+v\n", actrullyRst, expectRst)
			}
			Expect(ok).To(BeTrue())
			Expect(num).To(Equal(int64(5)))
		})

		It("Returns the required data, ignoring data that does not exist", func() {
			By("setting up the CustomMetricsPoint")
			start := time.Now()
			scrapttime := start
			history := CustomMetricsPoint{
				Timestamp: scrapttime,
				Usage: ResourceList{
					corev1.ResourceCPU:     *resource.NewQuantity(300, resource.DecimalSI),
					ResourceNetworkRxBytes: *resource.NewQuantity(500, resource.BinarySI),
					ResourceRequests:       *resource.NewQuantity(700, resource.BinarySI),
				},
			}
			secondwait := 2
			scrapttime = start.Add(time.Second * time.Duration(secondwait))
			raw := CustomMetricsPoint{
				Timestamp: scrapttime,
				Usage: ResourceList{
					corev1.ResourceCPU:     *resource.NewQuantity(200, resource.DecimalSI),
					ResourceNetworkRxBytes: *resource.NewQuantity(510, resource.BinarySI),
					ResourceRequests:       *resource.NewQuantity(730, resource.BinarySI),
				},
			}
			expectRst := CustomMetricsPoint{
				Timestamp:  scrapttime,
				WindowTime: time.Second * time.Duration(secondwait),
				Usage: ResourceList{
					corev1.ResourceCPU:     *resource.NewQuantity(200, resource.DecimalSI),
					ResourceNetworkRxBytes: *resource.NewQuantity(int64(10/secondwait), resource.BinarySI),
					ResourceRequests:       *resource.NewQuantity(int64(30/secondwait), resource.BinarySI),
				},
			}

			By("running the CustomMetricsFilter to the the reources rate in the new custom metrics")
			actrullyRst, num := raw.RateOfSome(&history)
			ok := actrullyRst.Equal(&expectRst)
			if !ok {
				fmt.Printf("metricsbatch:actrully:\n%+v\nexpect:\n%+v\n", actrullyRst, expectRst)
			}
			Expect(ok).To(BeTrue())
			Expect(num).To(Equal(int64(3)))
		})
	})

	Context("when compare the two CustomMetricsPoint", func() {
		It("should returns true or false for both data ", func() {
			By("setting up the CustomMetricsPoint")
			start := time.Now()
			scrapttime := start
			point1 := CustomMetricsPoint{
				Timestamp: scrapttime,
				Usage: ResourceList{
					corev1.ResourceCPU:     *resource.NewQuantity(300, resource.DecimalSI),
					corev1.ResourceMemory:  *resource.NewQuantity(400, resource.BinarySI),
					ResourceNetworkRxBytes: *resource.NewQuantity(500, resource.BinarySI),
					ResourceNetworkTxBytes: *resource.NewQuantity(600, resource.BinarySI),
					ResourceRequests:       *resource.NewQuantity(700, resource.BinarySI),
				},
			}
			secondwait := 2
			scrapttime = start.Add(time.Second * time.Duration(secondwait))
			point2 := CustomMetricsPoint{
				Timestamp:  scrapttime,
				WindowTime: time.Second * time.Duration(secondwait),
				Usage: ResourceList{
					corev1.ResourceCPU:     *resource.NewQuantity(200, resource.DecimalSI),
					corev1.ResourceMemory:  *resource.NewQuantity(300, resource.BinarySI),
					ResourceNetworkRxBytes: *resource.NewQuantity(510, resource.BinarySI),
					ResourceNetworkTxBytes: *resource.NewQuantity(620, resource.BinarySI),
					ResourceRequests:       *resource.NewQuantity(730, resource.BinarySI),
				},
			}
			point3 := CustomMetricsPoint{
				Timestamp:  scrapttime,
				WindowTime: time.Second * time.Duration(secondwait),
				Usage: ResourceList{
					corev1.ResourceCPU:     *resource.NewQuantity(200, resource.DecimalSI),
					corev1.ResourceMemory:  *resource.NewQuantity(300, resource.BinarySI),
					ResourceNetworkRxBytes: *resource.NewQuantity(510, resource.BinarySI),
					ResourceNetworkTxBytes: *resource.NewQuantity(620, resource.BinarySI),
					ResourceRequests:       *resource.NewQuantity(730, resource.BinarySI),
				},
			}
			point4 := CustomMetricsPoint{
				Timestamp:  scrapttime,
				WindowTime: time.Second * time.Duration(secondwait),
				Usage: ResourceList{
					corev1.ResourceCPU:     *resource.NewQuantity(200, resource.DecimalSI),
					corev1.ResourceMemory:  *resource.NewQuantity(300, resource.BinarySI),
					ResourceNetworkRxBytes: *resource.NewQuantity(510, resource.BinarySI),
					ResourceRequests:       *resource.NewQuantity(730, resource.BinarySI),
				},
			}

			By("compare the data")
			Expect(point1.Equal(&point2)).To(BeFalse())
			Expect(point2.Equal(&point3)).To(BeTrue())
			Expect(point3.Equal(&point4)).To(BeFalse())
			Expect(point1.Equal(&point4)).To(BeFalse())

		})

	})

	Context("when filter a new NamespaceMetrics", func() {
		It("should return the rate of resource", func() {
			start := time.Now()
			secondwait := int64(2)
			batch, err := buildBatch(start, 1, 1, 1, 2)
			Expect(err).NotTo(HaveOccurred())
			scrapttime := start.Add(time.Second * time.Duration(secondwait))
			batch2, err := buildBatch(scrapttime, 1, 1, 10, 2)
			Expect(err).NotTo(HaveOccurred())
			rst := NewCustomMetricsBatch()
			rst.Filter(batch, batch2)

			expectBatch := CustomMetricsBatch{
				Nodes:        make(NamespaceMetrics),
				Pods:         make(NamespaceMetrics),
				EndpointQPSs: make(NamespaceMetrics),
			}
			expectBatch.Nodes.Set("", "node1", &CustomMetricsPoint{
				Timestamp:  scrapttime,
				WindowTime: time.Second * time.Duration(secondwait),
				Usage: ResourceList{
					corev1.ResourceCPU:     *resource.NewQuantity(100+10*int64(10), resource.DecimalSI),
					corev1.ResourceMemory:  *resource.NewQuantity(200+10*int64(10), resource.BinarySI),
					ResourceNetworkRxBytes: *resource.NewQuantity(int64(90)/secondwait, resource.BinarySI),
					ResourceNetworkTxBytes: *resource.NewQuantity(int64(90)/secondwait, resource.BinarySI),
					ResourceRequests:       *resource.NewQuantity(int64(90)/secondwait, resource.BinarySI),
				},
			})
			expectBatch.Pods.Set("ns1", "pod1", &CustomMetricsPoint{ //11*10 - 2*10 =110-20=90
				Timestamp:  scrapttime,
				WindowTime: time.Second * time.Duration(secondwait),
				Usage: ResourceList{
					corev1.ResourceCPU:     *resource.NewQuantity(300+10*11, resource.DecimalSI),
					corev1.ResourceMemory:  *resource.NewQuantity(400+10*11, resource.BinarySI),
					ResourceNetworkRxBytes: *resource.NewQuantity(int64(90)/secondwait, resource.BinarySI),
					ResourceNetworkTxBytes: *resource.NewQuantity(int64(90)/secondwait, resource.BinarySI),
					ResourceRequests:       *resource.NewQuantity(int64(90)/secondwait, resource.BinarySI),
				},
			})
			expectBatch.Pods.Set("ns2", "pod2", &CustomMetricsPoint{ //12*10 - 3*10 =120-30=90
				Timestamp:  scrapttime,
				WindowTime: time.Second * time.Duration(secondwait),
				Usage: ResourceList{
					corev1.ResourceCPU:     *resource.NewQuantity(300+10*12, resource.DecimalSI),
					corev1.ResourceMemory:  *resource.NewQuantity(400+10*12, resource.BinarySI),
					ResourceNetworkRxBytes: *resource.NewQuantity(int64(90)/secondwait, resource.BinarySI),
					ResourceNetworkTxBytes: *resource.NewQuantity(int64(90)/secondwait, resource.BinarySI),
					ResourceRequests:       *resource.NewQuantity(int64(90)/secondwait, resource.BinarySI),
				},
			})
			// fmt.Printf("\n%+v\n%+v\n", rst.Nodes[NullNamespace]["node1"], expectBatch.Nodes[NullNamespace]["node1"])
			// fmt.Printf("\n%+v\n%+v\n", rst.Pods["ns1"]["pod1"], expectBatch.Pods["ns1"]["pod1"])
			Expect(rst.Nodes.Equal(expectBatch.Nodes)).To(BeTrue())
			Expect(rst.Pods.Equal(expectBatch.Pods)).To(BeTrue())
		})
	})
})
