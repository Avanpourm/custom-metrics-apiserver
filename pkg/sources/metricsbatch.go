// Copyright: meitu.com
// Author: JamesBryce
// Author Email: zmp@meitu.com
// Date: 2018-1-12
// Last Modified by:   JamesBryce

package sources

// The root API path will look like /apis/custom-metrics/v1alpha1. For brevity, this will be left off below.
// /{object-type}/{object-name}/{metric-name...}: retrieve the given metric for the given non-namespaced object (e.g. Node, PersistentVolume)
// /{object-type}/*/{metric-name...}: retrieve the given metric for all non-namespaced objects of the given type
// /{object-type}/*/{metric-name...}?labelSelector=foo: retrieve the given metric for all non-namespaced objects of the given type matching the given label selector
// /namespaces/{namespace-name}/{object-type}/{object-name}/{metric-name...}: retrieve the given metric for the given namespaced object
// /namespaces/{namespace-name}/{object-type}/*/{metric-name...}: retrieve the given metric for all namespaced objects of the given type
// /namespaces/{namespace-name}/{object-type}/*/{metric-name...}?labelSelector=foo: retrieve the given metric for all namespaced objects of the given type matching the given label selector
// /namespaces/{namespace-name}/metrics/{metric-name}: retrieve the given metric which describes the given namespace.

// type ResourceName corev1.ResourceName

const (
	NullNamespace = "Null"
	// AllNamespace  Namespace = Namespace("All")
)

var (
	EmptyObjectMetrics      = ObjectMetrics{}
	EmptyCustomMetricsPoint = CustomMetricsPoint{}
)

// NewCustomMetricsBatch return a new NewCustomMetricsBatch
func NewCustomMetricsBatch() *CustomMetricsBatch {
	return &CustomMetricsBatch{
		Nodes:        make(NamespaceMetrics),
		Pods:         make(NamespaceMetrics),
		EndpointQPSs: make(NamespaceMetrics),
	}
}

// CustomMetricsBatch is a collection store object for all metrics.
type CustomMetricsBatch struct {
	// Metrics              map[schema.GroupResource]NamespaceMetrics
	Total                int64
	CountNodeMetrics     int64
	CountPodMetrics      int64
	CountEndpointMetrics int64
	Nodes                NamespaceMetrics
	Pods                 NamespaceMetrics
	EndpointQPSs         NamespaceMetrics
}

// Filter carry on the preliminary transition and the revision to the collection data.
func (c *CustomMetricsBatch) Filter(history, newsrc *CustomMetricsBatch) {
	c.CountNodeMetrics += c.Nodes.Filter(history.Nodes, newsrc.Nodes)
	c.CountPodMetrics += c.Pods.Filter(history.Pods, newsrc.Pods)
	c.CountEndpointMetrics += c.EndpointQPSs.Filter(history.EndpointQPSs, newsrc.EndpointQPSs)
	c.Total = c.CountNodeMetrics + c.CountPodMetrics + c.CountEndpointMetrics
	return
}

// Merge a piece of data to a data set
func (c *CustomMetricsBatch) Merge(src *CustomMetricsBatch) {
	c.CountNodeMetrics += c.Nodes.Merge(src.Nodes)
	c.CountPodMetrics += c.Pods.Merge(src.Pods)
	c.CountEndpointMetrics += c.EndpointQPSs.Merge(src.EndpointQPSs)
	c.Total = c.CountNodeMetrics + c.CountPodMetrics + c.CountEndpointMetrics
	return
}

type ObjectMetrics map[string]*CustomMetricsPoint
type NamespaceMetrics map[string]ObjectMetrics

// Equal checks whether the current object is Equal to the content of the target object
func (n NamespaceMetrics) Equal(d NamespaceMetrics) bool {
	for namespace, objs := range n {
		for name, obj := range objs {
			objs2, ok := d[namespace]
			if !ok {
				return false
			}
			obj2, ok := objs2[name]
			if !ok {
				return false
			}
			if !obj.Equal(obj2) {
				return false
			}
		}
	}
	for namespace, objs := range d {
		for name, obj := range objs {
			objs2, ok := n[namespace]
			if !ok {
				return false
			}
			obj2, ok := objs2[name]
			if !ok {
				return false
			}
			if !obj.Equal(obj2) {
				return false
			}
		}
	}
	return true
}

// Filter processes and filters the newly collected data of a single resource
func (n NamespaceMetrics) Filter(history, src NamespaceMetrics) (metricsNum int64) {
	for namespace, objects := range src {
		for objName, Pointer := range objects {
			historyPoint := history.Get(namespace, objName)
			newpoint, num := Pointer.RateOfSome(historyPoint)
			if newpoint == nil {
				continue
			}
			metricsNum += num
			n.Set(namespace, objName, newpoint)
		}
	}
	return
}

// Merge copy a metrics set into the dataset
func (n NamespaceMetrics) Merge(src NamespaceMetrics) (num int64) {
	for namespace, objects := range src {
		for objName, Pointer := range objects {
			num += n.MergeMetrics(namespace, objName, Pointer)
		}
	}
	return
}

// MergeMetrics a new metric into the dataset.
func (n NamespaceMetrics) MergeMetrics(namespace, object string, p *CustomMetricsPoint) (num int64) {
	if p == nil {
		return
	}
	if namespace == "" {
		namespace = NullNamespace
	}
	objs, ok := n[namespace]
	if !ok {
		objs = make(ObjectMetrics)
	}
	pointer, ok := objs[object]
	if !ok {
		objs[object] = p
	} else {
		num = pointer.Merge(p)
		objs[object] = pointer
	}

	n[namespace] = objs
	return num
}

// Set a new metric into the dataset.
func (n NamespaceMetrics) Set(namespace, object string, p *CustomMetricsPoint) {
	if p == nil {
		return
	}
	if namespace == "" {
		namespace = NullNamespace
	}
	objs, ok := n[namespace]
	if !ok {
		objs = make(ObjectMetrics)
	}
	objs[object] = p
	n[namespace] = objs
}

// Get some metric from the data set
func (n NamespaceMetrics) Get(namespace, object string) *CustomMetricsPoint {
	if namespace == "" {
		namespace = NullNamespace
	}
	obj, ok := n[namespace]
	if !ok || obj == nil {
		return nil
	}
	p, ok := obj[object]
	return p
}
