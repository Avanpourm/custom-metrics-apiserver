// Copyright 2018 The Meitu Authors.
//
// Author JamesBryce
// Author Email zmp@meitu.com
//
// Date 2018-1-12
// Last Modified by JamesBryce

package nginxstats

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"testing"
	"time"

	"github.com/kubernetes-incubator/custom-metrics-apiserver/pkg/kubecache"
	"github.com/kubernetes-incubator/custom-metrics-apiserver/pkg/sources"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	// . "github.com/kubernetes-incubator/custom-metrics-apiserver/pkg/sources/fake"
	"github.com/kubernetes-incubator/custom-metrics-apiserver/pkg/sources/nodes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type fakeClient struct {
	delay    time.Duration
	metrics  []*Stats
	lastHost string
	err      error
}

func (c *fakeClient) GetStats(ctx context.Context, host string) (datas []*Stats, err error) {
	if c.err != nil {
		return datas, c.err
	}
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("timed out")
	case <-time.After(c.delay):
	}

	c.lastHost = host

	return c.metrics, nil
}
func TestNginxStatsSource(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Nginx Stats Source Test Suite")
}

var _ = Describe("Nginx Stats Collect", func() {
	var (
		namespace = "nginxstats-test"
		src       sources.MetricSource
		client    *fakeClient
		// scrapeTime time.Time  = time.Now()
		nodeInfo nodes.Info = nodes.Info{
			ConnectAddress: "10.0.1.2",
			Name:           "node1",
		}
		index *kubecache.Cache
	)
	BeforeEach(func() {
		index = kubecache.NewCache(nil)
		resultMetrics := []*Stats{}
		for i := 0; i < 100; i++ {
			f, err := strconv.ParseFloat(fmt.Sprintf("200%d.2", i), 10)
			Expect(err).NotTo(HaveOccurred())
			podIP := fmt.Sprintf("192.168.1.%d", i)
			podPort := fmt.Sprintf("1920%d", i)
			index.AddPod(&v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("TestPod-kjhasdf-%d", i),
					Namespace: namespace,
				},
				Status: v1.PodStatus{
					PodIP: podIP,
				},
			})

			resultMetrics = append(resultMetrics, &Stats{
				ip:       podIP,
				port:     podPort,
				http_2xx: &f,
			})
		}
		client = &fakeClient{
			metrics: resultMetrics,
		}

		src = NewNginxStatsMetricsSource(nodeInfo, client, index)

	})
	It("should pass the provided context to the nginx stats client to time out requests", func() {
		By("setting up a context with a 1 second timeout")
		ctx, workDone := context.WithTimeout(context.Background(), 1*time.Second)

		By("collecting the batch with a 4 second delay")
		start := time.Now()
		client.delay = 4 * time.Second
		_, err := src.Collect(ctx)
		workDone()

		By("ensuring it timed out with an error after 1 second")
		Expect(time.Now().Sub(start)).To(BeNumerically("~", 1*time.Second, 10*time.Millisecond))
		Expect(err).To(HaveOccurred())
	})

	It("should get a error and nil result when client return error", func() {
		By("setting up a client error")
		client.err = fmt.Errorf("not found")

		By("collecting the batch")
		batch, err := src.Collect(context.Background())
		Expect(err).To(HaveOccurred())
		Expect(batch).Should(BeNil())

	})

	It("Errors are made when data is missing, but other normal data can still be obtained", func() {
		By("setting up a data miss")
		client.metrics[0].http_2xx = nil
		client.metrics[1] = nil

		By("collecting the batch")
		batch, err := src.Collect(context.Background())
		Expect(err).To(HaveOccurred())
		Expect(len(batch.Pods[namespace])).Should(Equal(len(client.metrics) - 2))

	})

	It("Data acquired when a corresponding pod does not exist needs to be discarded", func() {
		By("setting up a data miss")

		pod := &v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("TestPod-kjhasdf-%d", 1),
				Namespace: namespace,
			},
			Status: v1.PodStatus{
				PodIP: "192.168.1.1",
			},
		}
		index.RemovePod(pod)

		By("collecting the batch")
		batch, err := src.Collect(context.Background())
		Expect(err).To(HaveOccurred())
		Expect(len(batch.Pods[namespace])).Should(Equal(len(client.metrics) - 1))

	})

	It("Data larger than MaxInt64", func() {
		By("setting up a large data")
		l := float64(math.MaxInt64 / 10)
		n := l + 1.0
		client.metrics[0].http_2xx = &n

		podName := "TestPod-kjhasdf-0"
		By("collecting the batch")
		batch, err := src.Collect(context.Background())
		Expect(err).NotTo(HaveOccurred())
		expectRequest := resource.NewScaledQuantity(int64(l/10), resource.Scale(1))
		Expect(*(batch.Pods.Get(namespace, podName).Usage.Requests())).Should(Equal(*expectRequest))
	})

	It("should fetch by connection address", func() {
		By("collecting the batch")
		_, err := src.Collect(context.Background())
		Expect(err).NotTo(HaveOccurred())
		By("verifying that it submitted the right host to the client")
		Expect(client.lastHost).To(Equal(nodeInfo.ConnectAddress))
	})

	It("should return the working set and nginx stats for all pods on the node", func() {
		By("collecting the batch")
		batch, err := src.Collect(context.Background())
		Expect(err).NotTo(HaveOccurred())

		By("verifying that the batch contains the right node data")
		verifyPods(client.metrics, batch, index)
		Expect(src.Name()).Should(Equal(fmt.Sprintf("nginx_stats_:%s", nodeInfo.Name)))
	})

})

func verifyPods(datas []*Stats, batch *sources.CustomMetricsBatch, index kubecache.PodsIndexInterface) {
	expectNSMetrics := sources.NamespaceMetrics{}
	timestamp := time.Now()
	for _, stat := range datas {
		podPoint := sources.CustomMetricsPoint{
			Usage: sources.ResourceList{},
		}
		if stat == nil || stat.ip == "" {
			continue
		}
		requestTotal := resource.NewScaledQuantity(int64(*stat.http_2xx*1000+0.5), resource.Scale(-3))
		pod := index.GetPodByIP(stat.ip)
		if pod == nil {
			continue
		}
		podPoint.Timestamp = timestamp
		podPoint.Usage[sources.ResourceRequests] = *requestTotal
		singleNSMetrics, ok := expectNSMetrics[pod.Namespace]
		if !ok {
			singleNSMetrics = sources.ObjectMetrics{}
			expectNSMetrics[pod.Namespace] = singleNSMetrics
		}
		singleNSMetrics[pod.Name] = &podPoint
	}

	for namespace, singleNSPods := range batch.Pods {
		for name, actuallyPoint := range singleNSPods {
			expectnspods, ok := expectNSMetrics[namespace]
			Expect(ok).To(BeTrue())
			expectPoint, ok := expectnspods[name]
			Expect(ok).To(BeTrue())
			ok = actuallyPoint.Equal(expectPoint)
			if !ok {
				fmt.Printf("actuallyPoint:\n%+v\nexpectPoint:\n%+v", actuallyPoint, expectPoint)
			}
			Expect(ok).To(BeTrue())
		}
	}
	for namespace, singleNSPods := range expectNSMetrics {
		for name, expectPoint := range singleNSPods {
			actuallynspods, ok := batch.Pods[namespace]
			Expect(ok).To(BeTrue())
			actuallyPoint, ok := actuallynspods[name]
			Expect(ok).To(BeTrue())
			ok = actuallyPoint.Equal(expectPoint)
			Expect(ok).To(BeTrue())
		}
	}

}
