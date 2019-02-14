// Copyright 2018 The Meitu Authors.
//
// Author JamesBryce
// Author Email zmp@meitu.com
//
// Date 2018-1-12
// Last Modified by JamesBryce

package fake

import (
	"context"

	"github.com/kubernetes-incubator/custom-metrics-apiserver/pkg/sources"
)

// StaticSourceProvider is a fake sources.MetricSourceProvider that returns
// metrics from a static list.
type StaticSourceProvider []sources.MetricSource

func (p StaticSourceProvider) GetMetricSources() ([]sources.MetricSource, error) { return p, nil }

// FunctionSource is a sources.MetricSource that calls a function to
// return the given data points
type FunctionSource struct {
	SourceName    string
	GenerateBatch CollectFunc
}

func (f *FunctionSource) Name() string {
	return f.SourceName
}

func (f *FunctionSource) Collect(ctx context.Context) (*sources.CustomMetricsBatch, error) {
	return f.GenerateBatch(ctx)
}

// CollectFunc is the function signature of FunctionSource#GenerateBatch,
// and knows how to generate a CustomMetricsBatch.
type CollectFunc func(context.Context) (*sources.CustomMetricsBatch, error)
