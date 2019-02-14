// Copyright 2018 The Meitu Authors.
//
// Author JamesBryce
// Author Email zmp@meitu.com
//
// Date 2018-1-12
// Last Modified by JamesBryce

package kubecache

import (
	"fmt"
	"testing"

	"github.com/kubernetes-incubator/custom-metrics-apiserver/pkg/fake"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientgocache "k8s.io/client-go/tools/cache"
)

func TestSourceManager(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Provider/Sink Suite")
}

var _ = Describe("Kube POD Cache", func() {
	var (
		namespace = "kube-cache-test"
		informer  *fake.SharedIndexInformer
		cache     *Cache
		rawObjs   []*v1.Pod
	)
	BeforeEach(func() {
		for i := 0; i < 100; i++ {
			podIP := fmt.Sprintf("192.168.1.%d", i)
			rawObjs = append(rawObjs, &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("TestPod-kjhasdf-%d", i),
					Namespace: namespace,
				},
				Spec: v1.PodSpec{
					NodeName: fmt.Sprintf("zmp-node-%d", i),
				},
				Status: v1.PodStatus{
					PodIP: podIP,
					Phase: v1.PodRunning,
				},
			})
		}

		informer = &fake.SharedIndexInformer{}
		cache = NewCache(informer)
	})
	It("should add pod succussfully", func() {
		By("add a pod to the cache")
		informer.Handler.OnAdd(rawObjs[0])
		By("get a pod from the cache")
		pod := cache.GetPodByIP(rawObjs[0].Status.PodIP)
		Expect(*pod).Should(Equal(*rawObjs[0]))
		By("add a obj but type is not pod")
		dd := 123
		cache.AddPod(&dd)
	})
	It("should update a pod succussfully", func() {
		By("add pods to the cache")
		informer.Handler.OnAdd(rawObjs[0])
		informer.Handler.OnAdd(rawObjs[1])
		informer.Handler.OnAdd(rawObjs[2])

		By("update a pod from the cache")
		newPod := *rawObjs[0]
		newPod.Status.Reason = "change"
		informer.Handler.OnUpdate(rawObjs[0], &newPod)
		pod := cache.GetPodByIP(rawObjs[0].Status.PodIP)
		Expect(*pod).Should(Equal(newPod))
		By("update obj but type is not pod")
		dd := 123
		cache.UpdatePod(rawObjs[0], &dd)
		cache.UpdatePod(&dd, &newPod)

	})
	It("should delete a pod succussfully", func() {
		By("add pods to the cache")
		informer.Handler.OnAdd(rawObjs[0])
		informer.Handler.OnAdd(rawObjs[1])
		informer.Handler.OnAdd(rawObjs[2])

		By("delete a pod from the cache")
		delPod := *rawObjs[0]
		informer.Handler.OnDelete(&delPod)
		pod := cache.GetPodByIP(delPod.Status.PodIP)
		Expect(pod).Should(BeNil())
		By("delete obj but type is not pod")
		dd := 123
		cache.RemovePod(&dd)
		informer.Handler.OnDelete(&dd)
	})
	It("should delete a FinalStateUnknown pod succussfully", func() {
		By("add pods to the cache")
		informer.Handler.OnAdd(rawObjs[0])
		informer.Handler.OnAdd(rawObjs[1])
		informer.Handler.OnAdd(rawObjs[2])

		By("delete a pod from the cache")
		delPod := clientgocache.DeletedFinalStateUnknown{
			Key: "xxxxx",
			Obj: rawObjs[0],
		}
		informer.Handler.OnDelete(delPod)
		pod := cache.GetPodByIP(rawObjs[0].Status.PodIP)
		Expect(pod).Should(BeNil())
		By("delete obj but type is not pod")

		dd := 123
		delPod = clientgocache.DeletedFinalStateUnknown{
			Key: "xxxxx",
			Obj: dd,
		}
		cache.RemovePod(delPod)
		informer.Handler.OnDelete(delPod)
	})
	It("should register with not informer or unqualified pod", func() {
		By("new cache with not informer")
		NewCache(nil)
		By("unqualified pod")
		rawObjs[0].Spec.NodeName = ""
		rawObjs[1].Status.Phase = v1.PodSucceeded
		rawObjs[2].Status.Phase = v1.PodFailed
		By("add pods to the cache")
		informer.Handler.OnAdd(rawObjs[0])
		informer.Handler.OnAdd(rawObjs[1])
		informer.Handler.OnAdd(rawObjs[2])
		pod := cache.GetPodByIP(rawObjs[0].Status.PodIP)
		Expect(pod).Should(BeNil())
		pod = cache.GetPodByIP(rawObjs[1].Status.PodIP)
		Expect(pod).Should(BeNil())
		pod = cache.GetPodByIP(rawObjs[2].Status.PodIP)
		Expect(pod).Should(BeNil())
	})
})
