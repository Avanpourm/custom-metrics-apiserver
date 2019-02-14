// Copyright 2018 The Meitu Authors.
//
// Author JamesBryce
// Author Email zmp@meitu.com
//
// Date 2018-1-12
// Last Modified by JamesBryce

package nginxstats

import (
	"fmt"
	"strconv"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Nginx Stats struct", func() {
	var (
		data     string
		rstDatas []*Stats
	)
	BeforeEach(func() {
		for i := 0; i < 100; i++ {
			data += fmt.Sprintf("192.168.1.%d:1920%d,0,0,0,0,200%d.2,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0\n", i, i, i)
			f, err := strconv.ParseFloat(fmt.Sprintf("200%d.2", i), 10)
			Expect(err).NotTo(HaveOccurred())
			rstDatas = append(rstDatas, &Stats{
				ip:       fmt.Sprintf("192.168.1.%d", i),
				port:     fmt.Sprintf("1920%d", i),
				http_2xx: &f,
			})
		}
		data += "192.168.1.9999:1920,0,0,0,0,200.2,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0\n"
		data += "192.168.1.9:1920,0,0,0,0,200fb.2,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0\n"

	})
	It("should parse Stats succussfully", func() {
		By("setting up a nginx Stats client")
		data := parseStats([]byte(data))
		eq := equalStats(data, rstDatas)
		Expect(eq).To(BeTrue())
	})

})

func equalStats(as []*Stats, bs []*Stats) bool {
	if len(as) != len(bs) {
		return false
	}
	for i := range as {
		if as[i].ip != bs[i].ip || as[i].port != bs[i].port || *as[i].http_2xx != *bs[i].http_2xx {
			return false
		}
	}
	return true
}

func statsToStrings(a []*Stats) (data string) {
	for i := range a {
		data = fmt.Sprintf("%s\n%s:%s,0,0,0,0,%f,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0", data, a[i].ip, a[i].port, *a[i].http_2xx)
	}
	return data
}
