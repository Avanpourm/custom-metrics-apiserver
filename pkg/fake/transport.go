// Copyright 2018 The Meitu Authors.
//
// Author JamesBryce
// Author Email zmp@meitu.com
//
// Date 2018-1-12
// Last Modified by JamesBryce

package fake

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type Transport struct {
	ExpectRequest  *http.Request
	ExpectResponse *http.Response
}

func (t *Transport) RoundTrip(req *http.Request) (rep *http.Response, err error) {

	if req.URL.String() != t.ExpectRequest.URL.String() {
		return nil, fmt.Errorf("Expect:%+v\n actually:%+v\n", t.ExpectRequest.URL, req.URL)
	}

	return t.ExpectResponse, nil
}

func BuildRequest(scheme, host, path string, port int, body []byte) (req *http.Request, err error) {
	url := url.URL{
		Scheme: scheme,
		Host:   net.JoinHostPort(host, strconv.Itoa(port)),
		Path:   path,
	}
	req, err = http.NewRequest("GET", url.String(), bufio.NewReader(strings.NewReader(string(body))))
	if err != nil {
		return nil, err
	}
	return req, nil
}

type mbufer struct {
	rd io.Reader
}

func (m *mbufer) Read(p []byte) (n int, err error) {
	return m.rd.Read(p)
}
func (m *mbufer) Close() error {
	return nil
}

func BuildResponse(req *http.Request, data []byte) (resp *http.Response, err error) {
	// data, err := json.Marshal(obj)
	// if err != nil {
	// 	return nil, err
	// }

	buf := mbufer{
		rd: bufio.NewReaderSize(strings.NewReader(string(data)), 8000000),
	}

	resp = &http.Response{
		Status:        "200 OK",
		StatusCode:    200,
		Proto:         "HTTP/1.1",
		Header:        http.Header{},
		Body:          &buf,
		ContentLength: int64(len(data)),
		Close:         true,
		Request:       req,
	}

	return resp, nil
}
