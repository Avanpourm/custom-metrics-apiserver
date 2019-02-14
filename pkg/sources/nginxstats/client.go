// Copyright 2018 The Meitu Authors.
// Author JamesBryce
// Author Email zmp@meitu.com
// Date 2018-1-12
// Last Modified by JamesBryce

package nginxstats

import (
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strconv"

	"github.com/golang/glog"
)

func NewNginxStatsClient(transport http.RoundTripper, port int, path string, deprecatedNoTLS bool) (NginxStatsInterface, error) {
	c := &http.Client{
		Transport: transport,
	}
	return &NginxStatsClient{
		port:            port,
		client:          c,
		path:            path,
		deprecatedNoTLS: deprecatedNoTLS,
	}, nil
}

// NginxStatsInterface knows how to fetch metrics from the Nginx
type NginxStatsInterface interface {
	// GetStats fetches Stats metrics from the given Nginx
	GetStats(ctx context.Context, host string) (datas []*Stats, err error)
}

type NginxStatsClient struct {
	port            int
	deprecatedNoTLS bool
	path            string
	client          *http.Client
}

type ErrNotFound struct {
	endpoint string
}

func (err *ErrNotFound) Error() string {
	return fmt.Sprintf("%q not found", err.endpoint)
}

func IsNotFoundError(err error) bool {
	_, isNotFound := err.(*ErrNotFound)
	return isNotFound
}

func (nc *NginxStatsClient) makeRequestAndGetValue(client *http.Client, req *http.Request) (datas []*Stats, err error) {
	response, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body - %v", err)
	}
	if response.StatusCode == http.StatusNotFound {
		return nil, &ErrNotFound{req.URL.String()}
	} else if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request failed - %q, response: %q", response.Status, string(body))
	}

	nginxAddr := "[unknown]"
	if req.URL != nil {
		nginxAddr = req.URL.Host
	}
	glog.V(10).Infof("Raw response from Nginx Stats at %s: %s", nginxAddr, string(body))
	datas = parseStats(body)
	return datas, nil
}

func (nc *NginxStatsClient) GetStats(ctx context.Context, host string) (datas []*Stats, err error) {
	scheme := "https"
	if nc.deprecatedNoTLS {
		scheme = "http"
	}
	path := "/us"

	if nc.path != "" {
		path = nc.path
	}
	url := url.URL{
		Scheme: scheme,
		Host:   net.JoinHostPort(host, strconv.Itoa(nc.port)),
		Path:   path,
	}

	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, err
	}
	client := nc.client
	if client == nil {
		client = http.DefaultClient
	}
	datas, err = nc.makeRequestAndGetValue(client, req.WithContext(ctx))
	return datas, err
}
