// Copyright: meitu.com
// Author: JamesBryce
// Author Email: zmp@meitu.com
// Date: 2018-1-12
// Last Modified by:   JamesBryce

package sources

import (
	"github.com/golang/glog"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

const (
	// // CPU, in cores. (500m = .5 cores)
	// ResourceCPU ResourceName = "cpu"
	// // Memory, in bytes. (500Gi = 500GiB = 500 * 1024 * 1024 * 1024)
	// ResourceMemory ResourceName = "memory"
	// // Volume size, in bytes (e,g. 5Gi = 5GiB = 5 * 1024 * 1024 * 1024)
	// ResourceStorage ResourceName = "storage"
	// // Local ephemeral storage, in bytes. (500Gi = 500GiB = 500 * 1024 * 1024 * 1024)
	// // The resource name for ResourceEphemeralStorage is alpha and it can change across releases.
	// ResourceEphemeralStorage ResourceName = "ephemeral-storage"

	ResourceNetworkRxBytes corev1.ResourceName = "network-rxbytes"
	ResourceNetworkTxBytes corev1.ResourceName = "network-txbytes"
	ResourceRequests       corev1.ResourceName = "requests"
)

// type Namespace string
// type ObjectName string
type ResourceList corev1.ResourceList

// Cpu Returns the CPU usage if specified.
func (r *ResourceList) Cpu() *resource.Quantity {
	if val, ok := (*r)[corev1.ResourceCPU]; ok {
		return &val
	}
	//Zero is different from nil. HPA has special handling logic for nil, so it needs to be separated from the zero value
	return nil
}

// Memory Returns the Memory usage if specified.
func (r *ResourceList) Memory() *resource.Quantity {
	if val, ok := (*r)[corev1.ResourceMemory]; ok {
		return &val
	}
	//Zero is different from nil. HPA has special handling logic for nil, so it needs to be separated from the zero value
	return nil //&resource.Quantity{Format: resource.BinarySI}
}

// NetworkRxBytes Returns the Network RxBytes usage if specified.
func (r *ResourceList) NetworkRxBytes() *resource.Quantity {
	if val, ok := (*r)[ResourceNetworkRxBytes]; ok {
		return &val
	}
	//Zero is different from nil. HPA has special handling logic for nil, so it needs to be separated from the zero value
	return nil
}

// NetworkTxBytes Returns the Network TxBytes usage if specified.
func (r *ResourceList) NetworkTxBytes() *resource.Quantity {
	if val, ok := (*r)[ResourceNetworkTxBytes]; ok {
		return &val
	}
	//Zero is different from nil. HPA has special handling logic for nil, so it needs to be separated from the zero value
	return nil
}

// Requests Returns the LB qps if specified.
func (r *ResourceList) Requests() *resource.Quantity {
	if val, ok := (*r)[ResourceRequests]; ok {
		return &val
	}
	return nil
}

// Rate it is a long story.
// Network bandwidth, QPS collection is a historical accumulation of the value,
// it needs to be converted into the rate of increase per second,
// surface memory and CPU is the occupation of the value of resources, do not need to be converted.
// NB:Zero is different from nil. HPA has special handling logic for nil, so it needs to be separated from the zero value
func (r *ResourceList) Rate(early *ResourceList, duration int64) (new ResourceList, metricsNum int64) {
	newNetworkRx := quantityRate(r.NetworkRxBytes(), early.NetworkRxBytes(), duration)
	newNetworkTx := quantityRate(r.NetworkTxBytes(), early.NetworkTxBytes(), duration)
	request := quantityRate(r.Requests(), early.Requests(), duration)
	new = ResourceList{}
	if newNetworkRx != nil {
		metricsNum++
		new[ResourceNetworkRxBytes] = *newNetworkRx
	}
	if newNetworkTx != nil {
		metricsNum++
		new[ResourceNetworkTxBytes] = *newNetworkTx
	}
	if request != nil {
		metricsNum++
		new[ResourceRequests] = *request
	}
	if r.Cpu() != nil {
		metricsNum++
		new[corev1.ResourceCPU] = *r.Cpu()
	}
	if r.Memory() != nil {
		metricsNum++
		new[corev1.ResourceMemory] = *r.Memory()
	}

	return new, metricsNum
}

// Merge it is a long story too.
func (r *ResourceList) Merge(new *ResourceList) (metricsNum int64) {
	newNetworkRx := quantityMerge(r.NetworkRxBytes(), new.NetworkRxBytes())
	newNetworkTx := quantityMerge(r.NetworkTxBytes(), new.NetworkTxBytes())
	request := quantityMerge(r.Requests(), new.Requests())

	if newNetworkRx != nil {
		metricsNum++
		(*r)[ResourceNetworkRxBytes] = *newNetworkRx
	}
	if newNetworkTx != nil {
		metricsNum++
		(*r)[ResourceNetworkTxBytes] = *newNetworkTx
	}
	if request != nil {
		metricsNum++
		(*r)[ResourceRequests] = *request
	}
	if new.Cpu() != nil {
		metricsNum++
		(*r)[corev1.ResourceCPU] = *new.Cpu()
	}
	if new.Memory() != nil {
		metricsNum++
		(*r)[corev1.ResourceMemory] = *new.Memory()
	}
	return
}
func quantityMerge(a, b *resource.Quantity) (rst *resource.Quantity) {
	if b == nil {
		return a
	}
	if a == nil {
		return b
	}
	a.Add(*b)
	return a
}

func quantityRate(a, b *resource.Quantity, seconds int64) (rst *resource.Quantity) {
	if a == nil || b == nil {
		return nil
	}

	aI, ok := a.AsInt64()
	if !ok {
		glog.Errorf("parse resource.Quantity failed:%+v", a)
	}
	bI, ok := b.AsInt64()
	if !ok {
		glog.Errorf("parse resource.Quantity failed:%+v", b)
	}
	val := (aI - bI) / seconds
	rst = resource.NewScaledQuantity(int64(val), 0)
	rst.Format = a.Format
	return rst
}
