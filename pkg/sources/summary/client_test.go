// Copyright 2018 The Meitu Authors.
//
// Author JamesBryce
// Author Email zmp@meitu.com
//
// Date 2018-02-13
// Last Modified by JamesBryce
package summary_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	. "github.com/kubernetes-incubator/custom-metrics-apiserver/pkg/fake"
	. "github.com/kubernetes-incubator/custom-metrics-apiserver/pkg/sources/summary"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	stats "k8s.io/kubernetes/pkg/kubelet/apis/stats/v1alpha1"
)

func equalSummary(a, b *stats.Summary) bool {
	if *a.Node.CPU.UsageNanoCores != *b.Node.CPU.UsageNanoCores {
		fmt.Printf("node cpu not equal. \na:%+v,\n b:%+v", *a.Node.CPU.UsageNanoCores, *b.Node.CPU.UsageNanoCores)
		return false
	}
	if *a.Node.Memory.WorkingSetBytes != *b.Node.Memory.WorkingSetBytes {
		fmt.Printf("node memory not equal. \na:%+v,\n b:%+v", a.Node.CPU.UsageNanoCores, b.Node.CPU.UsageNanoCores)
		return false
	}
	if len(a.Node.Network.Interfaces) != len(b.Node.Network.Interfaces) ||
		*a.Node.Network.InterfaceStats.TxBytes != *b.Node.Network.InterfaceStats.TxBytes ||
		*a.Node.Network.InterfaceStats.RxBytes != *b.Node.Network.InterfaceStats.RxBytes {
		fmt.Printf("node network not equal. \na:%+v,\n b:%+v", a, b)
		return false
	}
	for _, poda := range a.Pods {
		found := false
		for _, podb := range b.Pods {
			if podb.PodRef.Name == poda.PodRef.Name && podb.PodRef.Namespace == poda.PodRef.Namespace {
				found = true
				if len(podb.Containers) != len(poda.Containers) {
					fmt.Printf("pod container not equal. a:%+v, b:%+v", a, b)
					return false
				}
				for i := range poda.Containers {
					if *poda.Containers[i].CPU.UsageNanoCores != *podb.Containers[i].CPU.UsageNanoCores ||
						*poda.Containers[i].Memory.WorkingSetBytes != *podb.Containers[i].Memory.WorkingSetBytes {
						fmt.Printf("pod container not equal2. a:%+v, b:%+v", a, b)
						return false
					}
				}
				if len(poda.Network.Interfaces) != len(podb.Network.Interfaces) ||
					*poda.Network.InterfaceStats.RxBytes != *podb.Network.InterfaceStats.RxBytes ||
					*poda.Network.InterfaceStats.TxBytes != *podb.Network.InterfaceStats.TxBytes {
					fmt.Printf("pod network not equal. a:%+v, b:%+v", a, b)
					return false
				}

			}
		}
		if !found {
			fmt.Printf("pod not equal. a:%+v, b:%+v", a, b)
			return false
		}
	}
	return true
}

var _ = Describe("Summary Client", func() {
	var (
		transport      *Transport
		port           int
		ConnectAddress string
		summary        *stats.Summary
		scrapeTime     time.Time = time.Now()
	)
	BeforeEach(func() {

		port = 10240
		ConnectAddress = "192.168.1.1"
		summary = &stats.Summary{
			Node: stats.NodeStats{
				NodeName: "node1",
				CPU:      cpuStats(100, scrapeTime.Add(1000*time.Millisecond)),
				Memory:   memStats(200, scrapeTime.Add(2000*time.Millisecond)),
				Network: networkStats(
					[]stats.InterfaceStats{
						interfaceStats("eth0", 300, 400),
						interfaceStats("eth1", 300, 400),
					},
					scrapeTime.Add(200*time.Millisecond)),
			},
			Pods: []stats.PodStats{
				podStats("ns1", "pod1",
					networkStats(
						[]stats.InterfaceStats{
							interfaceStats("eth0", 300, 400),
							interfaceStats("eth1", 300, 400),
						},
						scrapeTime.Add(200*time.Millisecond)),
					containerStats("container1", 300, 400, scrapeTime.Add(1000*time.Millisecond)),
					containerStats("container2", 500, 600, scrapeTime.Add(2000*time.Millisecond))),
				podStats("ns1", "pod2",
					networkStats(
						[]stats.InterfaceStats{
							interfaceStats("eth0", 300, 400),
						},
						scrapeTime.Add(200*time.Millisecond)),
					containerStats("container1", 700, 800, scrapeTime.Add(3000*time.Millisecond))),
				podStats("ns2", "pod1",
					networkStats(
						[]stats.InterfaceStats{
							interfaceStats("eth0", 10, 400),
						},
						scrapeTime.Add(200*time.Millisecond)),
					containerStats("container1", 900, 1000, scrapeTime.Add(4000*time.Millisecond))),
				podStats("ns3", "pod1",
					networkStats(
						[]stats.InterfaceStats{
							interfaceStats("eth0", 300, 400),
						},
						scrapeTime.Add(200*time.Millisecond)),
					containerStats("container1", 1100, 1200, scrapeTime.Add(5000*time.Millisecond))),
			},
		}
		req, err := BuildRequest("https", ConnectAddress, "/stats/summary/", port, []byte("bbg"))
		Expect(err).NotTo(HaveOccurred())
		data, err := json.Marshal(summary)
		Expect(err).NotTo(HaveOccurred())

		response, err := BuildResponse(req, data)
		Expect(err).NotTo(HaveOccurred())
		transport = &Transport{
			ExpectRequest:  req,
			ExpectResponse: response,
		}
	})
	It("should get a summary succussfully", func() {
		By("setting up a kubelet client")
		kubeletClient, err := NewKubeletClient(transport, port, false)
		Expect(err).NotTo(HaveOccurred())

		By("GetSummary from remote kubelet")
		desdata, err := kubeletClient.GetSummary(context.Background(), ConnectAddress)
		Expect(err).NotTo(HaveOccurred())
		eq := equalSummary(desdata, summary)
		Expect(eq).To(BeTrue())

	})

	It("should get a summary failed, data notfound", func() {
		By("setting up a summary client return 404")
		transport.ExpectResponse.Status = "404 Notfound"
		transport.ExpectResponse.StatusCode = http.StatusNotFound
		Client, err := NewKubeletClient(transport, port, false)
		Expect(err).NotTo(HaveOccurred())

		By("GetStats from remote summary")
		_, err = Client.GetSummary(context.Background(), ConnectAddress)
		Expect(err).To(HaveOccurred())
		Expect(IsNotFoundError(err)).Should(BeTrue())
		fmt.Print(err.Error())

		By("setting up a summary client return 500")
		transport.ExpectResponse.Status = "500 "
		transport.ExpectResponse.StatusCode = http.StatusInternalServerError
		Client, err = NewKubeletClient(transport, port, false)
		Expect(err).NotTo(HaveOccurred())

		By("GetStats from remote summary")
		_, err = Client.GetSummary(context.Background(), ConnectAddress)
		Expect(err).To(HaveOccurred())
	})

	It("should get a summary succussfully", func() {
		By("setting up a kubelet client")
		transport.ExpectRequest.URL.Path = "notfound"

		kubeletClient, err := NewKubeletClient(transport, port, false)
		Expect(err).NotTo(HaveOccurred())

		By("GetSummary from remote kubelet")
		_, err = kubeletClient.GetSummary(context.Background(), ConnectAddress)
		Expect(err).To(HaveOccurred())

	})
})
