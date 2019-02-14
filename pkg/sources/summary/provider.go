// Copyright 2018 The Meitu Authors.
//
// Author JamesBryce
// Author Email zmp@meitu.com
//
// Date 2018-02-13
// Last Modified by JamesBryce

package summary

import (
	"fmt"

	"github.com/kubernetes-incubator/custom-metrics-apiserver/pkg/sources"
	"github.com/kubernetes-incubator/custom-metrics-apiserver/pkg/sources/nodes"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	v1listers "k8s.io/client-go/listers/core/v1"
	// "k8s.io/apimachinery/pkg/types"
)

type summaryProvider struct {
	nodeLister    v1listers.NodeLister
	kubeletClient KubeletInterface
	addrResolver  nodes.AddressResolver
}

func (p *summaryProvider) GetMetricSources() ([]sources.MetricSource, error) {
	sources := []sources.MetricSource{}
	nodes, err := p.nodeLister.List(labels.Everything())
	if err != nil {
		return nil, fmt.Errorf("unable to list nodes: %v", err)
	}

	var errs []error
	for _, node := range nodes {
		info, err := p.getNodeInfo(node)
		if err != nil {
			errs = append(errs, fmt.Errorf("unable to extract connection information for node %q: %v", node.Name, err))
			continue
		}
		sources = append(sources, NewSummaryMetricsSource(info, p.kubeletClient))
	}
	return sources, utilerrors.NewAggregate(errs)
}

func (p *summaryProvider) getNodeInfo(node *corev1.Node) (nodes.Info, error) {
	addr, err := p.addrResolver.Address(node)
	if err != nil {
		return nodes.Info{}, err
	}
	info := nodes.Info{
		Name:           node.Name,
		ConnectAddress: addr,
	}

	return info, nil
}

func NewSummaryProvider(nodeLister v1listers.NodeLister, kubeletClient KubeletInterface, addrResolver nodes.AddressResolver) sources.MetricSourceProvider {
	return &summaryProvider{
		nodeLister:    nodeLister,
		kubeletClient: kubeletClient,
		addrResolver:  addrResolver,
	}
}
