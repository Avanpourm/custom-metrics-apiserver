package captain

import (
	"github.com/kubernetes-incubator/custom-metrics-apiserver/pkg/provider"
	"github.com/kubernetes-incubator/custom-metrics-apiserver/pkg/sink"
	"github.com/kubernetes-incubator/custom-metrics-apiserver/pkg/sources"
	"k8s.io/client-go/dynamic"

	"time"

	"github.com/golang/glog"
	"github.com/kubernetes-incubator/custom-metrics-apiserver/pkg/provider/helpers"
	corev1 "k8s.io/api/core/v1"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/metrics/pkg/apis/custom_metrics"
	"k8s.io/metrics/pkg/apis/external_metrics"
)

// Captain is a sample implementation of provider.MetricsProvider which stores a map of fake metrics
type Captain struct {
	client dynamic.Interface
	mapper apimeta.RESTMapper
	rise   sink.MetricRise
}

// NewMetricsProvider returns an instance of captain
func NewMetricsProvider(client dynamic.Interface, mapper apimeta.RESTMapper, rise sink.MetricRise) provider.MetricsProvider {
	provider := &Captain{
		client: client,
		mapper: mapper,
		rise:   rise,
	}
	return provider
}

// metricsFor is a wrapper used by GetMetricBySelector to format several metrics which match a resource selector
func (c *Captain) metricsFor(namespace string, selector labels.Selector, info provider.CustomMetricInfo) (*custom_metrics.MetricValueList, error) {
	names, err := helpers.ListObjectNames(c.mapper, c.client, namespace, selector, info)
	if err != nil {
		return nil, err
	}

	res := make([]custom_metrics.MetricValue, 0, len(names))
	errs := []error{}
	for _, name := range names {
		namespacedName := types.NamespacedName{Name: name, Namespace: namespace}
		timestamp, window, value, err := c.rise.GetValue(info.GroupResource.String(),
			namespacedName.Namespace, namespacedName.Name, info.Metric)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		metric, err := c.metricFor(timestamp, int64(window.Seconds()), value, namespacedName, selector, info)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		res = append(res, *metric)
	}

	return &custom_metrics.MetricValueList{
		Items: res,
	}, nil
}

// metricFor is a helper function which formats a value, metric, and object info into a MetricValue which can be returned by the metrics API
func (c *Captain) metricFor(timestamp time.Time, window int64, value resource.Quantity,
	name types.NamespacedName, selector labels.Selector, info provider.CustomMetricInfo) (*custom_metrics.MetricValue, error) {
	objRef, err := helpers.ReferenceFor(c.mapper, name, info)
	if err != nil {
		return nil, err
	}

	metric := &custom_metrics.MetricValue{
		DescribedObject: objRef,
		Metric: custom_metrics.MetricIdentifier{
			Name: info.Metric,
		},
		Timestamp:     metav1.Time{timestamp},
		WindowSeconds: &window,
		Value:         value,
	}

	if len(selector.String()) > 0 {
		labelSelector, err := metav1.ParseToLabelSelector(selector.String())
		if err != nil {
			return nil, err
		}
		metric.Metric.Selector = labelSelector
	}

	return metric, nil
}

func (c *Captain) GetMetricByName(name types.NamespacedName, info provider.CustomMetricInfo) (cmv *custom_metrics.MetricValue, err error) {
	timestamp, window, value, err := c.rise.GetValue(info.GroupResource.String(), name.Namespace, name.Name, info.Metric)
	if err != nil {
		glog.V(4).Infof("GetMetricByName GetValue faild. name:%+v, info:%+v, err:%+v", name, info, err)
		return nil, err
	}
	cmv, err = c.metricFor(timestamp, int64(window.Seconds()), value, name, labels.Everything(), info)
	glog.V(6).Infof("GetMetricByName metricFor faild. value:%+v, name:%+v, labels:%+v, info:%+v, cmv:%+v, err:%+v",
		value, name, labels.Everything(), info, cmv, err)
	return cmv, err

}

func (c *Captain) GetMetricBySelector(namespace string, selector labels.Selector, info provider.CustomMetricInfo) (cmvs *custom_metrics.MetricValueList, err error) {
	cmvs, err = c.metricsFor(namespace, selector, info)
	glog.V(6).Infof("GetMetricBySelector( namespece:%+v, selector:%+v,info:%+v ): %+v, err:%+v", namespace, selector, info, cmvs, err)
	return cmvs, err
}

func (c *Captain) ListAllMetrics() (cms []provider.CustomMetricInfo) {
	// Get unique CustomMetricInfos from wrapper CustomMetricResources
	infos := make(map[provider.CustomMetricInfo]struct{})
	c.rise.Foreach(func(GroupResource, namespace, object, metrics string,
		timestamp time.Time, window time.Duration, value resource.Quantity) {
		namespaced := true
		if namespace == "" || namespace == sources.NullNamespace {
			namespaced = false
		}
		info := provider.CustomMetricInfo{
			GroupResource: corev1.Resource(GroupResource),
			Namespaced:    namespaced,
			Metric:        metrics,
		}
		infos[info] = struct{}{}
	})

	// Build slice of CustomMetricInfos to be returns
	datas := make([]provider.CustomMetricInfo, 0, len(infos))
	for info := range infos {
		datas = append(datas, info)
	}
	glog.V(6).Infof("ListAllMetrics: %+v", datas)
	return datas
}

func (c *Captain) GetExternalMetric(namespace string, metricSelector labels.Selector, info provider.ExternalMetricInfo) (emv *external_metrics.ExternalMetricValueList, err error) {
	return
}

func (c *Captain) ListAllExternalMetrics() (ems []provider.ExternalMetricInfo) {
	return
}
