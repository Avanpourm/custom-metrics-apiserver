// Copyright: meitu.com
// Author: JamesBryce
// Author Email: zmp@meitu.com
// Date: 2018-1-12
// Last Modified by:   JamesBryce

package sources

import "context"

// MetricSource knows how to collect pod, container, and node metrics from some location.
// It is expected that the batch returned contains unique values (i.e. it does not return
// the same node, pod, or container as any other source).
type MetricSource interface {
	// Collect fetches a batch of metrics.  It may return both a partial result and an error,
	// and non-nil results thus must be well-formed and meaningful even when accompanied by
	// and error.
	Collect(context.Context) (*CustomMetricsBatch, error)
	// Name names the metrics source for identification purposes
	Name() string
}

// MetricSourceProvider provides metric sources to collect from.
type MetricSourceProvider interface {
	// GetMetricSources fetches all sources known to this metrics provider.
	// It may return both partial results and an error.
	GetMetricSources() ([]MetricSource, error)
}
