package captain_test

import (
	"k8s.io/apimachinery/pkg/selection"

	"fmt"
	"testing"
	"time"

	. "github.com/kubernetes-incubator/custom-metrics-apiserver/pkg/captain"
	"github.com/kubernetes-incubator/custom-metrics-apiserver/pkg/dynamicmapper"
	"github.com/kubernetes-incubator/custom-metrics-apiserver/pkg/provider"
	"github.com/kubernetes-incubator/custom-metrics-apiserver/pkg/sink"
	"github.com/kubernetes-incubator/custom-metrics-apiserver/pkg/sources"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	core "k8s.io/client-go/testing"

	"sort"

	corev1 "k8s.io/api/core/v1"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/metrics/pkg/apis/custom_metrics"
)

const testingMapperRefreshInterval = 1 * time.Second

type fakeDynamicClient struct {
	Rs map[schema.GroupVersionResource]fakeNamespaceableResource
}

func (f *fakeDynamicClient) Resource(resource schema.GroupVersionResource) dynamic.NamespaceableResourceInterface {
	// dres, ok := f.Rs[resource]
	// if !ok {
	// 	return nil
	// }
	// return dres
	return f.Rs[resource]
}

type fakeNamespaceableResource struct {
	fakeResource
	nsResource map[string]fakeResource
}

func (f fakeNamespaceableResource) Namespace(name string) dynamic.ResourceInterface {
	// if nsRes, ok := f.nsResource[name]; ok {
	// 	return nsRes
	// }
	// return nil
	// 这里对于空值会自动装箱，不会出现panic
	return f.nsResource[name]
}

type fakeResource []struct {
	set  labels.Set
	list *unstructured.UnstructuredList
}

func (f fakeResource) List(opts metav1.ListOptions) (rst *unstructured.UnstructuredList, err error) {
	rst = &unstructured.UnstructuredList{}
	selector, err := labels.Parse(opts.LabelSelector)
	if err != nil {
		return nil, err
	}
	for _, kv := range f {
		if selector.Matches(kv.set) {
			if rst.Object == nil {
				rst.Object = make(map[string]interface{})
			}
			for k, v := range kv.list.Object {
				rst.Object[k] = v
			}
			rst.Items = append(kv.list.Items)
		}
	}
	if len(rst.Items) == 0 {
		return nil, fmt.Errorf("not found listOptions:%+v", opts)
	}
	return rst, nil
}
func (f fakeResource) Create(obj *unstructured.Unstructured, options metav1.CreateOptions, subresources ...string) (rst *unstructured.Unstructured, err error) {
	return
}
func (f fakeResource) Update(obj *unstructured.Unstructured, options metav1.UpdateOptions, subresources ...string) (rst *unstructured.Unstructured, err error) {
	return
}
func (f fakeResource) UpdateStatus(obj *unstructured.Unstructured, options metav1.UpdateOptions) (rst *unstructured.Unstructured, err error) {
	return
}
func (f fakeResource) Delete(name string, options *metav1.DeleteOptions, subresources ...string) (err error) {
	return
}
func (f fakeResource) DeleteCollection(options *metav1.DeleteOptions, listOptions metav1.ListOptions) (err error) {
	return
}
func (f fakeResource) Get(name string, options metav1.GetOptions, subresources ...string) (rst *unstructured.Unstructured, err error) {
	return
}
func (f fakeResource) Watch(opts metav1.ListOptions) (wa watch.Interface, err error) { return }
func (f fakeResource) Patch(name string, pt types.PatchType, data []byte, options metav1.UpdateOptions, subresources ...string) (rst *unstructured.Unstructured, err error) {
	return
}

type fakeMetricsRiseEntity struct {
	GroupResource string
	namespace     string
	objec         string
	metrics       string
	timestamp     time.Time
	window        time.Duration
	value         resource.Quantity
	err           error
}

type fakeMetricsRise struct {
	data []fakeMetricsRiseEntity
}

func (f *fakeMetricsRise) GetValue(GroupResource, namespace, object, metrics string) (timestamp time.Time, window time.Duration, value resource.Quantity, err error) {
	for _, v := range f.data {
		if GroupResource == v.GroupResource &&
			namespace == v.namespace &&
			object == v.objec &&
			metrics == v.metrics {
			return v.timestamp, v.window, v.value, v.err
		}
	}
	err = fmt.Errorf("Not Found")
	return
}
func (f *fakeMetricsRise) Foreach(do func(GroupResource, namespace, object, metrics string,
	timestamp time.Time, window time.Duration, value resource.Quantity)) {
	for _, v := range f.data {
		do(v.GroupResource, v.namespace, v.objec, v.metrics, v.timestamp, v.window, v.value)
	}
	return
}
func (f *fakeMetricsRise) Count() (totalMetric int64) {
	return int64(len(f.data))
}

func setupMapper(stopChan <-chan struct{}) (mapper *dynamicmapper.RegeneratingDiscoveryRESTMapper, discovery *dynamicmapper.FakeDiscovery) {
	fakeDiscovery := &dynamicmapper.FakeDiscovery{Fake: &core.Fake{}}
	fakeDiscovery.Resources = []*metav1.APIResourceList{
		{
			GroupVersion: "v1",
			APIResources: []metav1.APIResource{
				{Name: "pods", Namespaced: true, Kind: "Pod"},
				{Name: "nodes", Namespaced: false, Kind: "Node"},
				{Name: "endpoints", Namespaced: true, Kind: "Endpoints"},
			},
		},
	}

	mapper, err := dynamicmapper.NewRESTMapper(fakeDiscovery, testingMapperRefreshInterval)
	if err != nil {
		return
	}

	if stopChan != nil {
		mapper.RunUntil(stopChan)
	}
	// err = mapper.RegenerateMappings()
	// if err != nil {
	// 	return
	// }

	return mapper, fakeDiscovery
}
func setupDynamic() dynamic.Interface {
	dClient := fakeDynamicClient{
		Rs: map[schema.GroupVersionResource]fakeNamespaceableResource{
			{
				Resource: "pods",
				Version:  "v1",
			}: {
				nsResource: map[string]fakeResource{
					"ns1": {
						{
							set: labels.Set{
								"app":     "zmp",
								"app-st2": "bbg",
							},
							list: &unstructured.UnstructuredList{
								Object: map[string]interface{}{
									// "kind": "Pod",
									// "describedObject": &custom_metrics.ObjectReference{
									// 	Kind:            "Pod",
									// 	Namespace:       "ns1",
									// 	Name:            "pod1",
									// 	UID:             "bbg1",
									// 	APIVersion:      "/v1",
									// 	ResourceVersion: "",
									// 	FieldPath:       "",
									// },
								},
								Items: []unstructured.Unstructured{
									{
										Object: map[string]interface{}{
											"metadata": map[string]interface{}{
												"name":      "pod1",
												"namespace": "ns1",
												"uid":       "uidzmp1",
												"labels":    map[string]interface{}{"app": "zmp"},
											},
											"kind":       "Pod",
											"apiVersion": "v1",
										},
									},
									{
										Object: map[string]interface{}{
											"metadata": map[string]interface{}{
												"name":      "pod2",
												"namespace": "ns1",
												"uid":       "uidzmp1",
												"labels":    map[string]interface{}{"app": "value-notfound"},
											},
											"kind":       "Pod",
											"apiVersion": "v1",
										},
									},
								},
							},
						},
					},
				},
			},
			{
				Resource: "pods-nomapper",
				Version:  "v1",
			}: {
				nsResource: map[string]fakeResource{
					"ns1": {
						{
							set: labels.Set{
								"app":     "zmp",
								"app-st2": "bbg",
							},
							list: &unstructured.UnstructuredList{
								Items: []unstructured.Unstructured{
									{
										Object: map[string]interface{}{
											"metadata": map[string]interface{}{
												"name":      "pod1",
												"namespace": "ns1",
												"uid":       "uidzmp1",
												"labels":    map[string]interface{}{"app": "zmp"},
											},
											"kind":       "Pod",
											"apiVersion": "v1",
										},
									},
								},
							},
						},
					},
				},
			},
			{
				Resource: "nodes",
				Version:  "v1",
			}: {
				fakeResource: fakeResource{
					{
						set: labels.Set{
							"nodename": "nodezmp",
						},
						list: &unstructured.UnstructuredList{
							Object: map[string]interface{}{
								"kind": "Node",
							},
							Items: []unstructured.Unstructured{
								{
									Object: map[string]interface{}{
										"kind": "Node",
									},
								},
							},
						},
					},
				},
			},
			{
				Resource: "endpoinds",
				Version:  "v1",
			}: {
				nsResource: map[string]fakeResource{
					"ns1": {
						{
							set: labels.Set{
								"app":     "zmp2",
								"app-st2": "bbg2",
							},
							list: &unstructured.UnstructuredList{
								Object: map[string]interface{}{
									"kind": "Endpoinds",
								},
								Items: []unstructured.Unstructured{
									{
										Object: map[string]interface{}{
											"kind": "Endpoinds",
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	return &dClient
}
func setupRise(ts time.Time, du time.Duration) (rise sink.MetricRise) {

	rise = &fakeMetricsRise{
		data: []fakeMetricsRiseEntity{
			{
				GroupResource: "pods",
				namespace:     "ns1",
				objec:         "pod1",
				metrics:       "cpu",
				timestamp:     ts,
				window:        du,
				value:         *resource.NewQuantity(300, resource.DecimalSI),
				err:           nil,
			},
			{
				GroupResource: "pods-nomapper",
				namespace:     "ns1",
				objec:         "pod1",
				metrics:       "cpu",
				timestamp:     ts,
				window:        du,
				value:         *resource.NewQuantity(300, resource.DecimalSI),
				err:           nil,
			},
			{
				GroupResource: "pods-null-ns",
				namespace:     sources.NullNamespace,
				objec:         "pod1",
				metrics:       "cpu",
				timestamp:     ts,
				window:        du,
				value:         *resource.NewQuantity(300, resource.DecimalSI),
				err:           nil,
			},
			{
				GroupResource: "pods-null-ns2",
				namespace:     "",
				objec:         "pod1",
				metrics:       "cpu",
				timestamp:     ts,
				window:        du,
				value:         *resource.NewQuantity(300, resource.DecimalSI),
				err:           nil,
			},
		},
	}

	return
}

func TestSourceManager(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "captain Suite")
}

var _ = Describe("Get Custom Metrics", func() {
	var (
		client        dynamic.Interface
		mapper        apimeta.RESTMapper
		rise          sink.MetricRise
		prov          provider.MetricsProvider
		now           time.Time
		du            time.Duration
		windowSeconds int64
	)

	BeforeEach(func() {
		now = time.Now()
		windowSeconds = int64(2)
		du = time.Second * time.Duration(windowSeconds)
		client = setupDynamic()
		mapper, _ = setupMapper(nil)
		rise = setupRise(now, du)
		prov = NewMetricsProvider(client, mapper, rise)
	})

	It("should get a metrics by name", func() {
		By("set the custom metric info")
		name := types.NamespacedName{
			Name:      "pod1",
			Namespace: "ns1",
		}
		info := provider.CustomMetricInfo{
			GroupResource: corev1.Resource("pods"),
			Namespaced:    true,
			Metric:        corev1.ResourceCPU.String(),
		}
		By("GetMetricByName successfully")
		cmv, err := prov.GetMetricByName(name, info)
		Expect(err).NotTo(HaveOccurred())
		ExpectValue := custom_metrics.MetricValue{
			DescribedObject: custom_metrics.ObjectReference{
				Kind:            "Pod",
				Namespace:       "ns1",
				Name:            "pod1",
				UID:             "",
				APIVersion:      "/v1",
				ResourceVersion: "",
				FieldPath:       "",
			},
			Metric: custom_metrics.MetricIdentifier{
				Name: corev1.ResourceCPU.String(),
			},
			Timestamp:     metav1.Time{now},
			WindowSeconds: &windowSeconds,
			Value:         *resource.NewQuantity(300, resource.DecimalSI),
		}
		Expect(ExpectValue).To(Equal(*cmv))

		By("set the custom metric info")
		name = types.NamespacedName{
			Name:      "pod1-nodefound",
			Namespace: "ns1",
		}
		info = provider.CustomMetricInfo{
			GroupResource: corev1.Resource("pods"),
			Namespaced:    true,
			Metric:        corev1.ResourceCPU.String(),
		}
		By("GetMetricByName failed, no found value")
		_, err = prov.GetMetricByName(name, info)
		Expect(err).To(HaveOccurred())

		By("set the custom metric info")
		name = types.NamespacedName{
			Name:      "pod1",
			Namespace: "ns1",
		}
		info = provider.CustomMetricInfo{
			GroupResource: corev1.Resource("pods-nomapper"),
			Namespaced:    true,
			Metric:        corev1.ResourceCPU.String(),
		}
		By("GetMetricByName failed, no mapper")
		_, err = prov.GetMetricByName(name, info)
		Expect(err).To(HaveOccurred())

	})

	It("should get a metrics by selector", func() {
		By("set the custom metric info")
		namespace := "ns1"

		info := provider.CustomMetricInfo{
			GroupResource: corev1.Resource("pods"),
			Namespaced:    true,
			Metric:        corev1.ResourceCPU.String(),
		}
		selector := labels.NewSelector()
		req, err := labels.NewRequirement("app", selection.Equals, []string{"zmp"})
		Expect(err).NotTo(HaveOccurred())
		selector = selector.Add(*req)
		labelSelector, err := metav1.ParseToLabelSelector(selector.String())
		Expect(err).NotTo(HaveOccurred())

		By("GetMetrisBySelector successfully")
		cmvs, err := prov.GetMetricBySelector(namespace, selector, info)
		Expect(err).NotTo(HaveOccurred())
		Value := custom_metrics.MetricValue{
			DescribedObject: custom_metrics.ObjectReference{
				Kind:            "Pod",
				Namespace:       "ns1",
				Name:            "pod1",
				UID:             "",
				APIVersion:      "/v1",
				ResourceVersion: "",
				FieldPath:       "",
			},
			Metric: custom_metrics.MetricIdentifier{
				Name:     corev1.ResourceCPU.String(),
				Selector: labelSelector,
			},
			Timestamp:     metav1.Time{now},
			WindowSeconds: &windowSeconds,
			Value:         *resource.NewQuantity(300, resource.DecimalSI),
		}
		ExpectList := custom_metrics.MetricValueList{
			Items: []custom_metrics.MetricValue{
				Value,
			},
		}
		Expect(ExpectList).To(Equal(*cmvs))

		By("GetMetrisBySelector faild. selector not found")
		selectorNofound := labels.NewSelector()
		req, err = labels.NewRequirement("app", selection.Equals, []string{"zmpnoutfound"})
		Expect(err).NotTo(HaveOccurred())
		selectorNofound = selectorNofound.Add(*req)
		_, err = prov.GetMetricBySelector(namespace, selectorNofound, info)
		Expect(err).To(HaveOccurred())

		By("GetMetrisBySelector faild. value not found")

		selectorNoValue := labels.NewSelector()
		req, err = labels.NewRequirement("app", selection.Equals, []string{"value-notfound"})
		Expect(err).NotTo(HaveOccurred())
		selectorNoValue = selectorNoValue.Add(*req)

		_, err = prov.GetMetricBySelector(namespace, selectorNoValue, info)
		Expect(err).To(HaveOccurred())

		By("GetMetrisBySelector faild. value not mapper")
		info = provider.CustomMetricInfo{
			GroupResource: corev1.Resource("pods-nomapper"),
			Namespaced:    true,
			Metric:        corev1.ResourceCPU.String(),
		}
		_, err = prov.GetMetricBySelector(namespace, selector, info)
		Expect(err).To(HaveOccurred())
	})

	It("should get all metrics by ListAllMetrics", func() {
		By("set the custom metric info")

		cms := prov.ListAllMetrics()
		ExpectInfo := []provider.CustomMetricInfo{
			{
				GroupResource: corev1.Resource("pods"),
				Namespaced:    true,
				Metric:        "cpu",
			},
			{
				GroupResource: corev1.Resource("pods-nomapper"),
				Namespaced:    true,
				Metric:        "cpu",
			},
			{
				GroupResource: corev1.Resource("pods-null-ns"),
				Namespaced:    false,
				Metric:        "cpu",
			},
			{
				GroupResource: corev1.Resource("pods-null-ns2"),
				Namespaced:    false,
				Metric:        "cpu",
			},
		}
		sort.Slice(ExpectInfo, func(i, j int) bool {
			return ExpectInfo[i].GroupResource.String() > ExpectInfo[j].GroupResource.String()
		})
		sort.Slice(cms, func(i, j int) bool {
			return cms[i].GroupResource.String() > cms[j].GroupResource.String()
		})

		Expect(ExpectInfo).To(Equal(cms))
	})

})

var _ = Describe("Get External Metrics", func() {
	var (
		client        dynamic.Interface
		mapper        apimeta.RESTMapper
		rise          sink.MetricRise
		prov          provider.MetricsProvider
		now           time.Time
		du            time.Duration
		windowSeconds int64
	)

	BeforeEach(func() {
		now = time.Now()
		windowSeconds = int64(2)
		du = time.Second * time.Duration(windowSeconds)
		client = setupDynamic()
		mapper, _ = setupMapper(nil)
		rise = setupRise(now, du)
		prov = NewMetricsProvider(client, mapper, rise)
	})

	It("should get a External by name", func() {
		prov.GetExternalMetric("", nil, provider.ExternalMetricInfo{})
		prov.ListAllExternalMetrics()
	})

})
