// Copyright 2018 The Meitu Authors.
// Author JamesBryce
// Author Email zmp@meitu.com
// Date 2018-1-12
// Last Modified by JamesBryce

package nginxstats

import (
	"net"
	"strconv"
	"strings"
	"unsafe"
)

func ByteToString(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

type Stats struct {
	ip       string
	port     string
	http_2xx *float64
}

func parseStats(value []byte) (datas []*Stats) {
	str := ByteToString(value)
	lines := strings.Split(str, "\n")
	datas = make([]*Stats, 0, len(lines))

	for _, v := range lines {
		details := strings.Split(v, ",")

		if len(details) <= 11 {
			continue
		}
		IPPort := strings.Split(details[0], ":")
		if net.ParseIP(IPPort[0]) == nil {
			continue
		}

		reqtotal, err := strconv.ParseFloat(details[5], 64)
		if err != nil {
			continue
		}
		s := Stats{
			ip:       IPPort[0],
			http_2xx: &reqtotal,
		}
		if len(IPPort) > 1 {
			s.port = IPPort[1]
		}
		datas = append(datas, &s)
	}
	datas = datas[:]
	return datas
}
