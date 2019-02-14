// Copyright 2018 The Meitu Authors.
// Author JamesBryce
// Author Email zmp@meitu.com
// Date 2018-1-12
// Last Modified by JamesBryce

package sink

import (
	"fmt"
	"sync"
	"time"

	"github.com/kubernetes-incubator/custom-metrics-apiserver/pkg/sources"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

// kubernetesCadvisorWindow is the max window used by cAdvisor for calculating
// CPU usage rate.  While it can vary, it's no more than this number, but may be
// as low as half this number (when working with no backoff).  It would be really
// nice if the kubelet told us this in the summary API...
var kubernetesCadvisorWindow = 10 * time.Second

// sinkMetricsProvider is a provider.MetricsProvider that also acts as a sink.MetricSink
type sinkMetricsProvider struct {
	mu sync.RWMutex
	// NamespaceMetrics map[sinkMetricsInfo]sources.CustomMetricsPoint
	Sources *sources.CustomMetricsBatch
}

// NewSinkProvider returns a MetricSink that feeds into a MetricsProvider.
func NewSinkProvider() (MetricSink, MetricRise) {
	prov := &sinkMetricsProvider{}
	return prov, prov
}

func (p *sinkMetricsProvider) Receive(batch *sources.CustomMetricsBatch) error {
	// reca
	p.mu.Lock()
	defer p.mu.Unlock()
	p.Sources = batch

	return nil
}

func (p *sinkMetricsProvider) GetValue(GroupResource, namespace, object, metrics string) (timestamp time.Time, window time.Duration, value resource.Quantity, err error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	var point *sources.CustomMetricsPoint
	switch GroupResource {
	case "pods":
		point = p.Sources.Pods.Get(namespace, object)
	case "nodes":
		point = p.Sources.Nodes.Get(namespace, object)
	case "endpoints":
		point = p.Sources.EndpointQPSs.Get(namespace, object)
	default:
		err = fmt.Errorf("GroupResource:%v not found", GroupResource)
		return
	}
	if point == nil || point.Usage == nil {
		err = fmt.Errorf("GroupResource: %v ,namespace: %v, name: %v, metrics:%v, not found", GroupResource, namespace, object, metrics)
		return
	}
	timestamp = point.Timestamp
	window = point.WindowTime
	value, ok := point.Usage[corev1.ResourceName(metrics)]
	if !ok {
		err = fmt.Errorf("GroupResource: %v ,namespace: %v, name: %v, metrics: %v not found", GroupResource, namespace, object, metrics)
	}

	return timestamp, window, value, err
}

func (p *sinkMetricsProvider) Foreach(do func(GroupResource, namespace, object, metrics string,
	timestamp time.Time, window time.Duration, value resource.Quantity)) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	forEach("nodes", p.Sources.Nodes, do)
	forEach("pods", p.Sources.Pods, do)
	forEach("endpoints", p.Sources.EndpointQPSs, do)

}
func (p *sinkMetricsProvider) Count() int64 {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.Sources.Total
}

func forEach(GroupResource string, nm sources.NamespaceMetrics, do func(GroupResource, namespace, object, metrics string,
	timestamp time.Time, window time.Duration, value resource.Quantity)) {
	for namespace, objs := range nm {
		for objName, point := range objs {
			for metric, value := range point.Usage {
				do(GroupResource, namespace, objName, metric.String(), point.Timestamp, point.WindowTime, value)
			}
		}
	}
}
