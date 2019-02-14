// Copyright 2018 The Meitu Authors.
// Author JamesBryce
// Author Email zmp@meitu.com
// Date 2018-1-12
// Last Modified by JamesBryce

package kubecache

import (
	"sync"

	"github.com/golang/glog"
	"k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"
)

type PodsIndexInterface interface {
	GetPodByIP(ip string) (pod *v1.Pod)
}

func NewCache(informer cache.SharedIndexInformer) (c *Cache) {
	c = &Cache{
		pods: make(map[string]*v1.Pod),
	}
	c.RegisterEventHandler(informer)
	return c
}

type Cache struct {
	// This mutex guards all fields within this cache struct.
	mu   sync.RWMutex
	pods map[string]*v1.Pod
}

func (c *Cache) GetPodByIP(ip string) (pod *v1.Pod) {
	c.mu.RLock()
	pod, _ = c.pods[ip]
	c.mu.RUnlock()
	return pod
}
func (c *Cache) AddPod(obj interface{}) {
	pod, ok := obj.(*v1.Pod)
	if !ok {
		glog.Errorf("cannot convert to *v1.Pod: %v", obj)
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.pods[pod.Status.PodIP] = pod
	glog.V(7).Infof("add pod into kubecache *v1.Pod: %+v", pod)
	return
}

func (c *Cache) UpdatePod(oldObj, newObj interface{}) {
	oldPod, ok := oldObj.(*v1.Pod)
	if !ok {
		glog.Errorf("cannot convert oldObj to *v1.Pod: %v", oldObj)
		return
	}
	newPod, ok := newObj.(*v1.Pod)
	if !ok {
		glog.Errorf("cannot convert newObj to *v1.Pod: %v", newObj)
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.pods, oldPod.Status.PodIP)
	c.pods[newPod.Status.PodIP] = newPod
	glog.V(7).Infof("update kubecache *v1.Pod, old: %+v, new:%+v", oldPod, newPod)
	return
}

func (c *Cache) RemovePod(obj interface{}) {
	var pod *v1.Pod
	switch t := obj.(type) {
	case *v1.Pod:
		pod = t
	case cache.DeletedFinalStateUnknown:
		var ok bool
		pod, ok = t.Obj.(*v1.Pod)
		if !ok {
			glog.Errorf("cannot convert to *v1.Pod: %v", t.Obj)
			return
		}
	default:
		glog.Errorf("cannot convert to *v1.Pod: %v", t)
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.pods, pod.Status.PodIP)
	glog.V(7).Infof("remove pod from kubecache *v1.Pod: %+v", pod)
	return
}

func (c *Cache) RegisterEventHandler(informer cache.SharedIndexInformer) {
	if informer == nil {
		return
	}
	informer.AddEventHandler(
		cache.FilteringResourceEventHandler{
			FilterFunc: func(obj interface{}) bool {
				switch t := obj.(type) {
				case *v1.Pod:
					return assignedNonTerminatedPod(t)
				case cache.DeletedFinalStateUnknown:
					if pod, ok := t.Obj.(*v1.Pod); ok {
						return assignedNonTerminatedPod(pod)
					}
					glog.Errorf("unable to convert object %T to *v1.Pod in %T", obj, c)
					return false
				default:
					glog.Errorf("unable to handle object in %T: %T", c, obj)
					return false
				}
			},
			Handler: cache.ResourceEventHandlerFuncs{
				AddFunc:    c.AddPod,
				UpdateFunc: c.UpdatePod,
				DeleteFunc: c.RemovePod,
			},
		},
	)
	return
}

// assignedNonTerminatedPod selects pods that are assigned and non-terminal (scheduled and running).
func assignedNonTerminatedPod(pod *v1.Pod) bool {
	if len(pod.Spec.NodeName) == 0 {
		return false
	}
	if pod.Status.Phase == v1.PodSucceeded || pod.Status.Phase == v1.PodFailed {
		return false
	}
	return true
}
