// Copyright 2018 The Meitu Authors.
//
// Author JamesBryce
// Author Email zmp@meitu.com
//
// Date 2018-02-13
// Last Modified by JamesBryce

package summary

import (
	"fmt"

	"k8s.io/client-go/rest"
)

// GetKubeletConfig fetches connection config for connecting to the Kubelet.
func GetKubeletConfig(baseKubeConfig *rest.Config, port int, insecureTLS bool, completelyInsecure bool) *KubeletClientConfig {
	cfg := rest.CopyConfig(baseKubeConfig)
	if completelyInsecure {
		cfg = rest.AnonymousClientConfig(cfg)        // don't use auth to avoid leaking auth details to insecure endpoints
		cfg.TLSClientConfig = rest.TLSClientConfig{} // empty TLS config --> no TLS
	} else if insecureTLS {
		cfg.TLSClientConfig.Insecure = true
		cfg.TLSClientConfig.CAData = nil
		cfg.TLSClientConfig.CAFile = ""
	}
	kubeletConfig := &KubeletClientConfig{
		Port:                         port,
		RESTConfig:                   cfg,
		DeprecatedCompletelyInsecure: completelyInsecure,
	}

	return kubeletConfig
}

// KubeletClientConfig represents configuration for connecting to Kubelets.
type KubeletClientConfig struct {
	Port                         int
	RESTConfig                   *rest.Config
	DeprecatedCompletelyInsecure bool
}

// KubeletClientFor constructs a new KubeletInterface for the given configuration.
func KubeletClientFor(config *KubeletClientConfig) (KubeletInterface, error) {
	transport, err := rest.TransportFor(config.RESTConfig)
	if err != nil {
		return nil, fmt.Errorf("unable to construct transport: %v", err)
	}

	return NewKubeletClient(transport, config.Port, config.DeprecatedCompletelyInsecure)
}
