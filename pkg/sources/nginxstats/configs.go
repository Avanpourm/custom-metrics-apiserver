// Copyright 2018 The Meitu Authors.
// Author JamesBryce
// Author Email zmp@meitu.com
// Date 2018-1-12
// Last Modified by JamesBryce

package nginxstats

import (
	"fmt"

	"k8s.io/client-go/rest"
)

// NginxStatsClientConfig fetches connection config for connecting to the nginx stats.
func GetConfig(baseKubeConfig *rest.Config, port int, path string, insecureTLS bool, completelyInsecure bool) *NginxStatsClientConfig {
	cfg := rest.CopyConfig(baseKubeConfig)
	if completelyInsecure {
		cfg = rest.AnonymousClientConfig(cfg)        // don't use auth to avoid leaking auth details to insecure endpoints
		cfg.TLSClientConfig = rest.TLSClientConfig{} // empty TLS config --> no TLS
	} else if insecureTLS {
		cfg.TLSClientConfig.Insecure = true
		cfg.TLSClientConfig.CAData = nil
		cfg.TLSClientConfig.CAFile = ""
	}
	nginxstatConfig := &NginxStatsClientConfig{
		Port:                         port,
		RESTConfig:                   cfg,
		DeprecatedCompletelyInsecure: completelyInsecure,
	}

	return nginxstatConfig
}

// NginxStatsClientConfig represents configuration for connecting to nginx stats.
type NginxStatsClientConfig struct {
	Port                         int
	Path                         string
	RESTConfig                   *rest.Config
	DeprecatedCompletelyInsecure bool
}

// ClientFor constructs a new NginxStatsInterface for the given configuration.
func ClientFor(config *NginxStatsClientConfig) (NginxStatsInterface, error) {
	transport, err := rest.TransportFor(config.RESTConfig)
	if err != nil {
		return nil, fmt.Errorf("unable to construct transport: %v", err)
	}

	return NewNginxStatsClient(transport, config.Port, config.Path, config.DeprecatedCompletelyInsecure)
}
