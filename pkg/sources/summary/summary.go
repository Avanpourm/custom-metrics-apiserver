// Copyright 2018 The Meitu Authors.
//
// Author JamesBryce
// Author Email zmp@meitu.com
//
// Date 2018-02-13
// Last Modified by JamesBryce

package summary

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/golang/glog"
	"github.com/kubernetes-incubator/custom-metrics-apiserver/pkg/sources"
	"github.com/kubernetes-incubator/custom-metrics-apiserver/pkg/sources/nodes"
	"github.com/prometheus/client_golang/prometheus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	stats "k8s.io/kubernetes/pkg/kubelet/apis/stats/v1alpha1"
	// "k8s.io/apimachinery/pkg/types"
)

var (
	summaryRequestLatency = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "metrics_server",
			Subsystem: "kubelet_summary",
			Name:      "request_duration_seconds",
			Help:      "The Kubelet summary request latencies in seconds.",
			// TODO(directxman12): it would be nice to calculate these buckets off of scrape duration,
			// like we do elsewhere, but we're not passed the scrape duration at this level.
			Buckets: prometheus.DefBuckets,
		},
		[]string{"node"},
	)
	scrapeTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "metrics_server",
			Subsystem: "kubelet_summary",
			Name:      "scrapes_total",
			Help:      "Total number of attempted Summary API scrapes done by Metrics Server",
		},
		[]string{"success"},
	)
)

func init() {
	prometheus.MustRegister(summaryRequestLatency)
	prometheus.MustRegister(scrapeTotal)
}

// Kubelet-provided metrics for pod and system container.
type summaryMetricsSource struct {
	node          nodes.Info
	kubeletClient KubeletInterface
}

func NewSummaryMetricsSource(node nodes.Info, client KubeletInterface) sources.MetricSource {
	return &summaryMetricsSource{
		node:          node,
		kubeletClient: client,
	}
}

func (src *summaryMetricsSource) Name() string {
	return src.String()
}

func (src *summaryMetricsSource) String() string {
	return fmt.Sprintf("kubelet_summary:%s", src.node.Name)
}

func (src *summaryMetricsSource) Collect(ctx context.Context) (*sources.CustomMetricsBatch, error) {
	summary, err := func() (*stats.Summary, error) {
		startTime := time.Now()
		defer summaryRequestLatency.WithLabelValues(src.node.Name).Observe(float64(time.Since(startTime)) / float64(time.Second))
		return src.kubeletClient.GetSummary(ctx, src.node.ConnectAddress)
	}()

	if err != nil {
		scrapeTotal.WithLabelValues("false").Inc()
		return nil, fmt.Errorf("unable to fetch metrics from Kubelet %s (%s): %v", src.node.Name, src.node.ConnectAddress, err)
	}

	scrapeTotal.WithLabelValues("true").Inc()
	res := sources.NewCustomMetricsBatch()
	resNode := &sources.CustomMetricsPoint{}
	// resPod := make([]sources.CustomMetricsPoint, len(summary.Pods))

	var errsTotal []error
	hasMetrics, errs := src.decodeNodeStats(&summary.Node, resNode)
	errsTotal = append(errsTotal, errs...)
	// if len(errs) != 0 {
	if !hasMetrics {
		// if we had errors providing node metrics, discard the data point
		// so that we don't incorrectly report metric values as zero.
		resNode = nil

	} else {
		res.Nodes.Set("", src.node.Name, resNode)
	}

	for _, pod := range summary.Pods {
		resPod := &sources.CustomMetricsPoint{}
		hasMetrics, podErrs := src.decodePodStats(&pod, resPod)
		errsTotal = append(errsTotal, podErrs...)
		// 当存在指标时，就可以上报，不存的在指标不上报。不存在的指标不能上报0值。因为HPA在进行处理时
		// 对于不存在的指标会做特殊处理的。需要将这个不存在状态反馈给HPA
		if !hasMetrics {
			// NB: we explicitly want to discard pods with partial results, since
			// the horizontal pod autoscaler takes special action when a pod is missing
			// metrics (and zero CPU or memory does not count as "missing metrics")

			// we don't care if we reuse slots in the result array,
			// because they get completely overwritten in decodePodStats
			continue
		}

		res.Pods.Set(pod.PodRef.Namespace, pod.PodRef.Name, resPod)
	}

	return res, utilerrors.NewAggregate(errsTotal)
}

func (src *summaryMetricsSource) decodeNodeStats(nodeStats *stats.NodeStats, target *sources.CustomMetricsPoint) (hasMetrics bool, errs []error) {
	hasMetrics = false
	timestamp, err := getScrapeTime(nodeStats.CPU, nodeStats.Memory, nodeStats.Network)
	if err != nil {
		// if we can't get a timestamp, assume bad data in general
		return hasMetrics, []error{fmt.Errorf("unable to get valid timestamp for metric point for node %q, discarding data: %v", src.node.ConnectAddress, err)}
	}
	*target = sources.CustomMetricsPoint{
		// Name:      src.node.Name,
		Timestamp: timestamp,
		Usage:     sources.ResourceList{}, // make(map[string]resource.Quantity),
	}
	cpuUsage := resource.Quantity{}
	memUsage := resource.Quantity{}
	NetworkRxBytesUsage := resource.Quantity{}
	NetworkTxBytesUsage := resource.Quantity{}

	if err := decodeCPU(&cpuUsage, nodeStats.CPU); err != nil {
		errs = append(errs, fmt.Errorf("unable to get CPU for node %q, discarding data: %v", src.node.ConnectAddress, err))
	} else {
		hasMetrics = true
		target.Usage[corev1.ResourceCPU] = cpuUsage
	}

	if err := decodeMemory(&memUsage, nodeStats.Memory); err != nil {
		errs = append(errs, fmt.Errorf("unable to get memory for node %q, discarding data: %v", src.node.ConnectAddress, err))
	} else {
		hasMetrics = true
		target.Usage[corev1.ResourceMemory] = memUsage
	}
	if err := decodeNetwork(&NetworkRxBytesUsage, &NetworkTxBytesUsage, nodeStats.Network); err != nil {
		errs = append(errs, fmt.Errorf("unable to get network for node %q, discarding data: %v", src.node.ConnectAddress, err))
	} else {
		hasMetrics = true
		target.Usage[sources.ResourceNetworkRxBytes] = NetworkRxBytesUsage
		target.Usage[sources.ResourceNetworkTxBytes] = NetworkTxBytesUsage
	}

	return hasMetrics, errs
}

func (src *summaryMetricsSource) decodePodStats(podStats *stats.PodStats, target *sources.CustomMetricsPoint) (hasMetrics bool, errs []error) {
	hasMetrics = false
	missCPUMetrics := false
	missMemoryMetrics := false
	// completely overwrite data in the target
	*target = sources.CustomMetricsPoint{
		// Name:  src.node.Name,
		Usage: sources.ResourceList{},
	}
	cpuUsage := resource.Quantity{}
	memUsage := resource.Quantity{}
	NetworkRxBytesUsage := resource.Quantity{}
	NetworkTxBytesUsage := resource.Quantity{}

	var earliest *time.Time

	// var errCPU, errMem []error

	for _, container := range podStats.Containers {
		timestamp, err := getScrapeTime(container.CPU, container.Memory, podStats.Network)
		if err != nil {
			// if we can't get a timestamp, assume bad data in general
			errs = append(errs, fmt.Errorf("unable to get a valid timestamp for metric point for container %q in pod %s/%s on node %q, discarding data: %v",
				container.Name, podStats.PodRef.Namespace, podStats.PodRef.Name, src.node.ConnectAddress, err))
			continue
		}
		if earliest == nil || earliest.After(timestamp) {
			earliest = &timestamp
		}
		cpuUsageTmp := resource.Quantity{}
		memUsageTmp := resource.Quantity{}

		if err := decodeCPU(&cpuUsageTmp, container.CPU); err != nil {
			errs = append(errs, fmt.Errorf("unable to get CPU for container %q in pod %s/%s on node %q, discarding data: %v",
				container.Name, podStats.PodRef.Namespace, podStats.PodRef.Name, src.node.ConnectAddress, err))
			// errs = append(errs, errCpu)
			missCPUMetrics = true
		} else {
			cpuUsage.Add(cpuUsageTmp)
		}

		if err := decodeMemory(&memUsageTmp, container.Memory); err != nil {
			errs = append(errs, fmt.Errorf("unable to get memory for container %q in pod %s/%s on node %q: %v, discarding data",
				container.Name, podStats.PodRef.Namespace, podStats.PodRef.Name, src.node.ConnectAddress, err))
			missMemoryMetrics = true
		} else {
			memUsage.Add(memUsageTmp)
		}
	}

	if !missCPUMetrics && earliest != nil {
		hasMetrics = true
		target.Usage[corev1.ResourceCPU] = cpuUsage
	}

	if !missMemoryMetrics && earliest != nil {
		hasMetrics = true
		target.Usage[corev1.ResourceMemory] = memUsage
	}

	if err := decodeNetwork(&NetworkRxBytesUsage, &NetworkTxBytesUsage, podStats.Network); err != nil {
		errs = append(errs, fmt.Errorf("unable to get Network for Pod %q in pod %s/%s on node %q: %v, discarding data",
			podStats.PodRef.Name, podStats.PodRef.Namespace, podStats.PodRef.Name, src.node.ConnectAddress, err))
	} else {
		hasMetrics = true
		target.Usage[sources.ResourceNetworkRxBytes] = NetworkRxBytesUsage
		target.Usage[sources.ResourceNetworkTxBytes] = NetworkTxBytesUsage
	}

	if earliest != nil {
		target.Timestamp = *earliest
	}
	return hasMetrics, errs
}

func decodeCPU(target *resource.Quantity, cpuStats *stats.CPUStats) error {
	if cpuStats == nil || cpuStats.UsageNanoCores == nil {
		return fmt.Errorf("missing cpu usage metric")
	}

	*target = *uint64Quantity(*cpuStats.UsageNanoCores, -9)
	return nil
}

func decodeMemory(target *resource.Quantity, memStats *stats.MemoryStats) error {
	if memStats == nil || memStats.WorkingSetBytes == nil {
		return fmt.Errorf("missing memory usage metric")
	}

	*target = *uint64Quantity(*memStats.WorkingSetBytes, 0)
	target.Format = resource.BinarySI

	return nil
}
func decodeNetwork(targetRx, targetTx *resource.Quantity, networkStats *stats.NetworkStats) error {

	if networkStats == nil {
		return fmt.Errorf("missing network usage metric, nofound network interface")
	}

	if networkStats.TxBytes != nil && networkStats.RxBytes != nil {
		*targetTx = *uint64Quantity(*networkStats.TxBytes, 0)
		targetTx.Format = resource.BinarySI
		*targetRx = *uint64Quantity(*networkStats.RxBytes, 0)
		targetRx.Format = resource.BinarySI
		return nil
	}

	for _, i := range networkStats.Interfaces {
		if i.TxBytes != nil {
			targetTx.Add(*uint64Quantity(*i.TxBytes, 0))
			targetTx.Format = resource.BinarySI
		} else {
			return fmt.Errorf("missing network usage metric:%v.TxBytes is nil", i.Name)
		}
		if i.RxBytes != nil {
			targetRx.Add(*uint64Quantity(*i.RxBytes, 0))
			targetTx.Format = resource.BinarySI
		} else {
			return fmt.Errorf("missing network usage metric:%v.RxBytes is nil", i.Name)
		}
	}

	return nil
}

func getScrapeTime(cpu *stats.CPUStats, memory *stats.MemoryStats, network *stats.NetworkStats) (time.Time, error) {
	// Ensure we get the earlier timestamp so that we can tell if a given data
	// point was tainted by pod initialization.

	var earliest *time.Time
	if cpu != nil && !cpu.Time.IsZero() && (earliest == nil || earliest.After(cpu.Time.Time)) {
		earliest = &cpu.Time.Time
	}

	if memory != nil && !memory.Time.IsZero() && (earliest == nil || earliest.After(memory.Time.Time)) {
		earliest = &memory.Time.Time
	}

	if network != nil && !network.Time.IsZero() && (earliest == nil || earliest.After(network.Time.Time)) {
		earliest = &network.Time.Time
	}
	if earliest == nil {
		return time.Time{}, fmt.Errorf("no non-zero timestamp on either CPU or memory")
	}

	return *earliest, nil
}

// uint64Quantity converts a uint64 into a Quantity, which only has constructors
// that work with int64 (except for parse, which requires costly round-trips to string).
// We lose precision until we fit in an int64 if greater than the max int64 value.
func uint64Quantity(val uint64, scale resource.Scale) *resource.Quantity {
	// easy path -- we can safely fit val into an int64
	if val <= math.MaxInt64 {
		return resource.NewScaledQuantity(int64(val), scale)
	}

	glog.V(1).Infof("unexpectedly large resource value %v, loosing precision to fit in scaled resource.Quantity", val)

	// otherwise, lose an decimal order-of-magnitude precision,
	// so we can fit into a scaled quantity
	return resource.NewScaledQuantity(int64(val/10), resource.Scale(1)+scale)
}
