// Copyright 2018 The Meitu Authors.
// Author JamesBryce
// Author Email zmp@meitu.com
// Date 2018-1-12
// Last Modified by JamesBryce

package nginxstats

import (
	"fmt"

	"github.com/kubernetes-incubator/custom-metrics-apiserver/pkg/kubecache"
	"github.com/kubernetes-incubator/custom-metrics-apiserver/pkg/sources"
	"github.com/kubernetes-incubator/custom-metrics-apiserver/pkg/sources/nodes"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	v1listers "k8s.io/client-go/listers/core/v1"
)

type nginxStatsProvider struct {
	sourceSelector labels.Selector
	nodeLister     v1listers.NodeLister
	client         NginxStatsInterface
	addrResolver   nodes.AddressResolver
	podsIndex      kubecache.PodsIndexInterface
}

func (n *nginxStatsProvider) GetMetricSources() ([]sources.MetricSource, error) {
	sources := []sources.MetricSource{}
	nodes, err := n.nodeLister.List(n.sourceSelector)
	if err != nil {
		return nil, fmt.Errorf("unable to list nodes, selector:%+v, err:%v", n.sourceSelector, err)
	}

	var errs []error
	for _, node := range nodes {
		info, err := n.getNodeInfo(node)
		if err != nil {
			errs = append(errs, fmt.Errorf("unable to extract connection information for node %q: %v", node.Name, err))
			continue
		}
		sources = append(sources, NewNginxStatsMetricsSource(info, n.client, n.podsIndex))
	}
	return sources, utilerrors.NewAggregate(errs)
}

func (n *nginxStatsProvider) getNodeInfo(node *corev1.Node) (nodes.Info, error) {
	addr, err := n.addrResolver.Address(node)
	if err != nil {
		return nodes.Info{}, err
	}
	info := nodes.Info{
		Name:           node.Name,
		ConnectAddress: addr,
	}

	return info, nil
}

func NewNginxstatsProvider(nodeLister v1listers.NodeLister, client NginxStatsInterface,
	addrResolver nodes.AddressResolver, index kubecache.PodsIndexInterface, sourceSelector labels.Selector) sources.MetricSourceProvider {
	return &nginxStatsProvider{
		sourceSelector: sourceSelector,
		nodeLister:     nodeLister,
		client:         client,
		addrResolver:   addrResolver,
		podsIndex:      index,
	}
}
