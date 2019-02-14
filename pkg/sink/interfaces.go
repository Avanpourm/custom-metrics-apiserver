// Copyright 2018 The Meitu Authors.
// Author JamesBryce
// Author Email zmp@meitu.com
// Date 2018-1-12
// Last Modified by JamesBryce

package sink

import (
	"time"

	"github.com/kubernetes-incubator/custom-metrics-apiserver/pkg/sources"
	"k8s.io/apimachinery/pkg/api/resource"
)

// MetricSink knows how to receive metrics batches from a source.
type MetricSink interface {
	// Receive ingests a new batch of metrics.
	Receive(*sources.CustomMetricsBatch) error
}

// MetricRise knows how to Spit out metrics.
type MetricRise interface {
	// Receive ingests a new batch of metrics.

	GetValue(GroupResource, namespace, object, metrics string) (time.Time, time.Duration, resource.Quantity, error)
	Foreach(func(GroupResource, namespace, object, metrics string,
		timestamp time.Time, window time.Duration, value resource.Quantity))
	Count() (totalMetric int64)
}
