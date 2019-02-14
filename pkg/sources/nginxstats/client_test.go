// Copyright 2018 The Meitu Authors.
//
// Author JamesBryce
// Author Email zmp@meitu.com
//
// Date 2018-1-12
// Last Modified by JamesBryce

package nginxstats

import (
	"context"
	"fmt"
	"net/http"

	. "github.com/kubernetes-incubator/custom-metrics-apiserver/pkg/fake"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Nginx Stats Client", func() {
	var (
		transport      *Transport
		port           int
		ConnectAddress string
		rstDatas       []*Stats
		// scrapeTime     time.Time = time.Now()
	)
	BeforeEach(func() {

		port = 10240
		request := 9000.2
		ConnectAddress = "192.168.1.1"
		rstDatas = []*Stats{
			{
				ip:       "192.168.1.200",
				port:     "8001",
				http_2xx: &request,
			},
			{
				ip:       "192.168.1.202",
				port:     "8001",
				http_2xx: &request,
			},
		}

		req, err := BuildRequest("http", ConnectAddress, "/us", port, []byte("bbg"))
		Expect(err).NotTo(HaveOccurred())
		data := statsToStrings(rstDatas)
		response, err := BuildResponse(req, []byte(data))
		Expect(err).NotTo(HaveOccurred())
		transport = &Transport{
			ExpectRequest:  req,
			ExpectResponse: response,
		}
	})
	It("should get a nginx Stats succussfully", func() {
		By("setting up a nginx Stats client")
		Client, err := NewNginxStatsClient(transport, port, "/us", true)
		Expect(err).NotTo(HaveOccurred())

		By("GetStats from remote nginx Stats")
		desdata, err := Client.GetStats(context.Background(), ConnectAddress)
		Expect(err).NotTo(HaveOccurred())
		eq := equalStats(desdata, rstDatas)
		Expect(eq).To(BeTrue())

	})
	It("should get a nginx Stats failed", func() {
		By("setting up a nginx Stats client")
		transport.ExpectRequest.URL.Path = "notfound"

		Client, err := NewNginxStatsClient(transport, port, "/us", true)
		Expect(err).NotTo(HaveOccurred())

		By("GetStats from remote nginx Stats")
		_, err = Client.GetStats(context.Background(), ConnectAddress)
		Expect(err).To(HaveOccurred())

	})

	It("should get a nginx Stats failed, data notfound", func() {
		By("setting up a nginx Stats client return 404")
		transport.ExpectResponse.Status = "404 Notfound"
		transport.ExpectResponse.StatusCode = http.StatusNotFound
		Client, err := NewNginxStatsClient(transport, port, "/us", true)
		Expect(err).NotTo(HaveOccurred())

		By("GetStats from remote nginx Stats")
		_, err = Client.GetStats(context.Background(), ConnectAddress)
		Expect(err).To(HaveOccurred())
		Expect(IsNotFoundError(err)).Should(BeTrue())
		fmt.Print(err.Error())

		By("setting up a nginx Stats client return 500")
		transport.ExpectResponse.Status = "500 "
		transport.ExpectResponse.StatusCode = http.StatusInternalServerError
		Client, err = NewNginxStatsClient(transport, port, "/us", true)
		Expect(err).NotTo(HaveOccurred())

		By("GetStats from remote nginx Stats")
		_, err = Client.GetStats(context.Background(), ConnectAddress)
		Expect(err).To(HaveOccurred())

	})
})
