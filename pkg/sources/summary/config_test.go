// Copyright 2018 The Meitu Authors.
//
// Author JamesBryce
// Author Email zmp@meitu.com
//
// Date 2018-02-13
// Last Modified by JamesBryce

package summary_test

import (
	. "github.com/kubernetes-incubator/custom-metrics-apiserver/pkg/sources/summary"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/rest"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

var _ = Describe("Nginx Stats  config", func() {
	var (
		baseKubeConfig     *rest.Config
		port               int
		insecureTLS        bool
		completelyInsecure bool
	)
	BeforeEach(func() {
		baseKubeConfig = &rest.Config{}
		port = 1001
		insecureTLS = true
		completelyInsecure = true

	})
	It("should get config succussfully", func() {
		config := GetKubeletConfig(baseKubeConfig, port, insecureTLS, completelyInsecure)
		ngingStat, err := KubeletClientFor(config)
		Expect(err).NotTo(HaveOccurred())
		Expect(ngingStat).NotTo(BeNil())
	})
	It("should get config failed when configure error occured ", func() {
		insecureTLS = true
		completelyInsecure = false
		config := GetKubeletConfig(baseKubeConfig, port, insecureTLS, completelyInsecure)
		config.RESTConfig.ExecProvider = &clientcmdapi.ExecConfig{}
		ngingStat, err := KubeletClientFor(config)
		Expect(err).To(HaveOccurred())
		Expect(ngingStat).To(BeNil())
	})

})
