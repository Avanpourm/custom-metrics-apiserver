// Copyright 2018 The Meitu Authors.
// Author JamesBryce
// Author Email zmp@meitu.com
// Date 2018-1-12
// Last Modified by JamesBryce

package nginxstats

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/kubernetes-incubator/custom-metrics-apiserver/pkg/kubecache"

	"github.com/golang/glog"
	"github.com/kubernetes-incubator/custom-metrics-apiserver/pkg/sources"
	"github.com/kubernetes-incubator/custom-metrics-apiserver/pkg/sources/nodes"
	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/apimachinery/pkg/api/resource"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
)

var (
	statsRequestLatency = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "metrics_server",
			Subsystem: "nginx_stats",
			Name:      "request_duration_seconds",
			Help:      "The Nginx Stats request latencies in seconds.",
			// TODO(directxman12): it would be nice to calculate these buckets off of scrape duration,
			// like we do elsewhere, but we're not passed the scrape duration at this level.
			Buckets: prometheus.DefBuckets,
		},
		[]string{"node"},
	)
	scrapeTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "metrics_server",
			Subsystem: "nginx_stats",
			Name:      "scrapes_total",
			Help:      "Total number of attempted Nginx Stats API scrapes done by Metrics Server",
		},
		[]string{"success"},
	)
)

func init() {
	prometheus.MustRegister(statsRequestLatency)
	prometheus.MustRegister(scrapeTotal)
}

func NewNginxStatsMetricsSource(node nodes.Info, client NginxStatsInterface, index kubecache.PodsIndexInterface) (sources sources.MetricSource) {
	return &nginxStatsMetricsSource{
		node:      node,
		client:    client,
		podsIndex: index,
	}
}

// nginxStats-provided metrics for qps of pod.
type nginxStatsMetricsSource struct {
	node      nodes.Info
	client    NginxStatsInterface
	podsIndex kubecache.PodsIndexInterface
}

func (src *nginxStatsMetricsSource) Name() string {
	return src.String()
}

func (src *nginxStatsMetricsSource) String() string {
	return fmt.Sprintf("nginx_stats_:%s", src.node.Name)
}

func (src *nginxStatsMetricsSource) Collect(ctx context.Context) (rets *sources.CustomMetricsBatch, err error) {
	datas, err := func() ([]*Stats, error) {
		startTime := time.Now()
		defer statsRequestLatency.WithLabelValues(src.node.Name).Observe(float64(time.Since(startTime)) / float64(time.Second))
		return src.client.GetStats(ctx, src.node.ConnectAddress)
	}()

	if err != nil {
		scrapeTotal.WithLabelValues("false").Inc()
		return nil, fmt.Errorf("unable to fetch metrics from Nginx Stats %s (%s): %v", src.node.Name, src.node.ConnectAddress, err)
	}
	scrapeTotal.WithLabelValues("true").Inc()
	res := sources.NewCustomMetricsBatch()
	// resEndpoint :=  make([]sources.CustomMetricsPoint, len(datas))

	var errsTotal []error

	for _, endpoint := range datas {
		resEndpoint := &sources.CustomMetricsPoint{}
		hasMetrics, namespace, podName, podErrs := src.decodeEndpointStats(endpoint, resEndpoint)
		errsTotal = append(errsTotal, podErrs)
		// 当存在指标时，就才上报，不存的在指标不上报。不存在的指标不能上报0值。因为HPA在进行处理时
		// 对于不存在的指标会做特殊处理的。需要将这个不存在状态反馈给HPA
		if !hasMetrics {
			continue
		}
		res.Pods.Set(namespace, podName, resEndpoint)
	}

	return res, utilerrors.NewAggregate(errsTotal)

}

func (src *nginxStatsMetricsSource) decodeEndpointStats(data *Stats, target *sources.CustomMetricsPoint) (hasMetrics bool, namespace, podName string, err error) {
	hasMetrics = false
	// completely overwrite data in the target
	*target = sources.CustomMetricsPoint{
		Usage:     sources.ResourceList{},
		Timestamp: time.Now(),
	}
	response2xx := resource.Quantity{}

	if err := decodeResponse(&response2xx, data); err != nil {
		err = fmt.Errorf("unable to get Response Stats for node %q, discarding data: %v", src.node.ConnectAddress, err)
		return hasMetrics, namespace, podName, err
	} else {
		hasMetrics = true
		target.Usage[sources.ResourceRequests] = response2xx
	}
	namespace, podName, err = src.getPod(data.ip)
	return hasMetrics, namespace, podName, err

}
func (src *nginxStatsMetricsSource) getPod(podIP string) (namespace, podName string, err error) {
	pod := src.podsIndex.GetPodByIP(podIP)
	if pod == nil {
		err = fmt.Errorf("the pod ip is not found. pod IP:%+v", podIP)
		return
	}
	return pod.Namespace, pod.Name, nil
}

func decodeResponse(target *resource.Quantity, data *Stats) error {
	if data == nil || data.http_2xx == nil {
		return fmt.Errorf("missing response Stats metric")
	}
	*target = *float64Quantity(*data.http_2xx, 0)
	// target.Format = resource.BinarySI
	return nil
}

// float64Quantity converts a float64 into a Quantity, which only has constructors
func float64Quantity(val float64, scale resource.Scale) *resource.Quantity {
	// easy path -- we can safely fit val into an MaxFloat64
	if val*1000 <= math.MaxInt64 {
		return resource.NewScaledQuantity(int64(val*1000+0.5), resource.Scale(-3)+scale)
	}

	glog.V(1).Infof("unexpectedly large resource value %v, loosing precision to fit in scaled resource.Quantity", val)

	// otherwise, lose an decimal order-of-magnitude precision,
	// so we can fit into a scaled quantity
	return resource.NewScaledQuantity(int64(val/10), resource.Scale(1)+scale)
}
