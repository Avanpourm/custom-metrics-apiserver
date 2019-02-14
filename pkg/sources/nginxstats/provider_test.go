// Copyright 2018 The Meitu Authors.
//
// Author JamesBryce
// Author Email zmp@meitu.com
//
// Date 2018-02-13
// Last Modified by JamesBryce

package nginxstats

import (
	"fmt"

	"github.com/kubernetes-incubator/custom-metrics-apiserver/pkg/kubecache"
	"github.com/kubernetes-incubator/custom-metrics-apiserver/pkg/sources"
	"github.com/kubernetes-incubator/custom-metrics-apiserver/pkg/sources/nodes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"

	v1listers "k8s.io/client-go/listers/core/v1"
)

type fackeNodeLister struct {
	labels labels.Labels
	nodes  []*v1.Node
	err    error
}

func (fn *fackeNodeLister) List(selector labels.Selector) (ret []*v1.Node, err error) {
	if fn.err != nil {
		return ret, fn.err
	}
	if selector.Matches(fn.labels) {
		return fn.nodes, nil
	}
	return nil, fmt.Errorf("not found %v", selector.String())
}
func (fn *fackeNodeLister) Get(name string) (node *v1.Node, err error) { return }
func (fn *fackeNodeLister) ListWithPredicate(predicate v1listers.NodeConditionPredicate) (node []*v1.Node, err error) {
	return
}

var _ = Describe("Nginx Stats Provider", func() {
	var (
		provider     sources.MetricSourceProvider
		nodeLister   fackeNodeLister
		addrResolver nodes.AddressResolver
		index        *kubecache.Cache
		selector     labels.Selector
		client       *NginxStatsClient
	)
	BeforeEach(func() {
		nodeLister = fackeNodeLister{
			labels: labels.Set{"function": "nginxstats"},
			nodes: []*v1.Node{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "node1.szdc.inc.com",
					},
					Status: v1.NodeStatus{
						Addresses: []v1.NodeAddress{
							{
								Type:    v1.NodeInternalIP,
								Address: "192.168.1.1",
							},
							{
								Type:    v1.NodeHostName,
								Address: "node1.szdc.inc.com",
							},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "node2.szdc.inc.com",
					},
					Status: v1.NodeStatus{
						Addresses: []v1.NodeAddress{
							{
								Type:    v1.NodeInternalIP,
								Address: "192.168.1.2",
							},
							{
								Type:    v1.NodeHostName,
								Address: "node2.szdc.inc.com",
							},
						},
					},
				},
			},
		}
		client = &NginxStatsClient{}
		addrResolver = nodes.NewPriorityAddressResolver([]v1.NodeAddressType{v1.NodeInternalIP})
		index = kubecache.NewCache(nil)
		selector = labels.NewSelector()
		rq, _ := labels.NewRequirement("function", selection.Equals, []string{"nginxstats"})
		selector.Add(*rq)
		provider = NewNginxstatsProvider(&nodeLister, client, addrResolver, index, selector)
	})
	It("should get a source succussfully", func() {
		By("get a sources")
		sources, err := provider.GetMetricSources()
		Expect(err).NotTo(HaveOccurred())
		Expect(len(sources)).Should(Equal(2))

	})
	It("should get a error when nodelister failed", func() {
		By("get a sources")
		nodeLister.err = fmt.Errorf("not found")
		_, err := provider.GetMetricSources()
		Expect(err).To(HaveOccurred())

	})

	It("should get a address resolver error", func() {
		By("get a sources")
		addrResolver = nodes.NewPriorityAddressResolver([]v1.NodeAddressType{v1.NodeInternalDNS})
		provider = NewNginxstatsProvider(&nodeLister, client, addrResolver, index, selector)
		_, err := provider.GetMetricSources()
		Expect(err).To(HaveOccurred())
	})

})
