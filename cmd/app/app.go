package app

import (
	"fmt"
	"io"
	"time"

	"github.com/kubernetes-incubator/custom-metrics-apiserver/pkg/captain"
	basecmd "github.com/kubernetes-incubator/custom-metrics-apiserver/pkg/cmd"
	"github.com/kubernetes-incubator/custom-metrics-apiserver/pkg/dynamicmapper"
	"github.com/kubernetes-incubator/custom-metrics-apiserver/pkg/kubecache"
	"github.com/kubernetes-incubator/custom-metrics-apiserver/pkg/manager"
	"github.com/kubernetes-incubator/custom-metrics-apiserver/pkg/sink"
	"github.com/kubernetes-incubator/custom-metrics-apiserver/pkg/sources"
	"github.com/kubernetes-incubator/custom-metrics-apiserver/pkg/sources/nginxstats"
	"github.com/kubernetes-incubator/custom-metrics-apiserver/pkg/sources/nodes"
	"github.com/kubernetes-incubator/custom-metrics-apiserver/pkg/sources/summary"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	corev1 "k8s.io/api/core/v1"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apiserver/pkg/server/healthz"
)

// NewCommandStartCustomMetricsServer provides a CLI handler for the custom metrics server entrypoint
func NewCommandStartCustomMetricsServer(out, errOut io.Writer, stopCh <-chan struct{}) *cobra.Command {
	o := NewCustomMetricsServerOptions()

	cmd := &cobra.Command{
		Short: "Launch custom-metrics-apiserver",
		Long:  "Launch custom-metrics-apiserver",
		RunE: func(c *cobra.Command, args []string) error {
			if err := o.RunServer(stopCh); err != nil {
				return err
			}
			return nil
		},
	}

	flags := cmd.Flags()
	o.AddFlags(flags)
	return cmd
}

// CustomMetricsServerOptions  provides the main parameters of the custom-metric,
// the actual adapterBase main function, the initialization of the parameters,
// the initialization of the service, the startup of the service,
// which is the main entry class of the program
type CustomMetricsServerOptions struct {
	basecmd.AdapterBase
	// Only to be used to for testing
	DisableAuthForTesting bool
	restMapper            apimeta.RESTMapper
	MetricResolution      time.Duration

	KubeletPort int

	NginxStatsPort       int
	NginxStatsPath       string
	NginxNodeLabel       string
	NginxNodeSelectorKey string

	InsecureKubeletTLS                  bool
	KubeletPreferredAddressTypes        []string
	DeprecatedCompletelyInsecureKubelet bool
}

// NewCustomMetricsServerOptions constructs a new set of default options for custom-metrics-server.
func NewCustomMetricsServerOptions() *CustomMetricsServerOptions {
	o := &CustomMetricsServerOptions{
		AdapterBase: basecmd.AdapterBase{
			Name:              "custom-metrics-apiserver",
			DiscoveryInterval: 60 * time.Second,
		},
		MetricResolution:             60 * time.Second,
		KubeletPort:                  10250,
		NginxStatsPath:               "/us",
		NginxStatsPort:               80,
		NginxNodeLabel:               "nginxstats",
		NginxNodeSelectorKey:         "nodefunc",
		KubeletPreferredAddressTypes: make([]string, len(nodes.DefaultAddressTypePriority)),
	}

	for i, addrType := range nodes.DefaultAddressTypePriority {
		o.KubeletPreferredAddressTypes[i] = string(addrType)
	}

	return o
}

// AddFlags sets the set of custom parameters and introduces default parameters.
func (o *CustomMetricsServerOptions) AddFlags(flags *pflag.FlagSet) {

	o.FlagSet = flags
	o.Flags()
	flags.DurationVar(&o.MetricResolution, "metric-resolution", o.MetricResolution, "The resolution at which custom-metrics-apiserver will retain metrics.")
	flags.BoolVar(&o.InsecureKubeletTLS, "kubelet-insecure-tls", o.InsecureKubeletTLS, "Do not verify CA of serving certificates presented by Kubelets.  For testing purposes only.")
	flags.BoolVar(&o.DeprecatedCompletelyInsecureKubelet, "deprecated-kubelet-completely-insecure", o.DeprecatedCompletelyInsecureKubelet, "Do not use any encryption, authorization, or authentication when communicating with the Kubelet.")
	flags.IntVar(&o.KubeletPort, "kubelet-port", o.KubeletPort, "The port to use to connect to Kubelets.")
	flags.StringSliceVar(&o.KubeletPreferredAddressTypes, "kubelet-preferred-address-types", o.KubeletPreferredAddressTypes, "The priority of node address types to use when determining which address to use to connect to a particular node")
	flags.MarkDeprecated("deprecated-kubelet-completely-insecure", "This is rarely the right option, since it leaves kubelet communication completely insecure.  If you encounter auth errors, make sure you've enabled token webhook auth on the Kubelet, and if you're in a test cluster with self-signed Kubelet certificates, consider using kubelet-insecure-tls instead.")
	flags.IntVar(&o.NginxStatsPort, "nginx-stats-port", o.NginxStatsPort, "The port to use to connect to nginx stats.")
	flags.StringVar(&o.NginxStatsPath, "nginx-stats-path", o.NginxStatsPath, "The path to use to connect to nginx stats.")
	flags.StringVar(&o.NginxNodeLabel, "nginx-node-label", o.NginxStatsPath, "The label to use to list the nginx stats nodes.")
	flags.StringVar(&o.NginxNodeSelectorKey, "nginx-node-selector-key", o.NginxStatsPath, "The selector key of the nginx stats nodes.")

}

// RunServer init the param, start the main loop, start the http server.
func (o *CustomMetricsServerOptions) RunServer(stopCh <-chan struct{}) error {
	// change the default config
	// grab the config for the API server
	config, err := o.Config()
	if err != nil {
		return err
	}
	config.GenericConfig.EnableMetrics = true
	config.GenericConfig.MaxRequestsInFlight = 400
	config.GenericConfig.MaxMutatingRequestsInFlight = 200
	config.GenericConfig.RequestTimeout = time.Duration(2) * time.Second
	config.GenericConfig.MinRequestTimeout = 1800

	// set up the client config
	clientConfig, err := o.ClientConfig()
	if err != nil {
		return err
	}

	// set up the informers
	// we should never need to resync, since we're not worried about missing events,
	// and resync is actually for regular interval-based reconciliation these days,
	// so set the default resync interval to 0
	informerFactory, err := o.Informers()
	if err != nil {
		return err
	}

	// set up the source manager
	kubeletConfig := summary.GetKubeletConfig(clientConfig, o.KubeletPort, o.InsecureKubeletTLS, o.DeprecatedCompletelyInsecureKubelet)
	kubeletClient, err := summary.KubeletClientFor(kubeletConfig)
	if err != nil {
		return fmt.Errorf("unable to construct a client to connect to the kubelets: %v", err)
	}

	dclient, err := o.DynamicClient()
	if err != nil {
		return fmt.Errorf("unable to construct dynamic client: %v", err)
	}

	mapper, err := o.RESTMapper(stopCh)
	if err != nil {
		return fmt.Errorf("unable to construct discovery REST mapper: %v", err)
	}

	// set up the in-memory sink and provider
	metricSink, metricsRise := sink.NewSinkProvider()

	provider := captain.NewMetricsProvider(dclient, mapper, metricsRise)

	// set the sources scraper
	// set up an address resolver according to the user's priorities
	addrPriority := make([]corev1.NodeAddressType, len(o.KubeletPreferredAddressTypes))
	for i, addrType := range o.KubeletPreferredAddressTypes {
		addrPriority[i] = corev1.NodeAddressType(addrType)
	}
	addrResolver := nodes.NewPriorityAddressResolver(addrPriority)
	// set the source Porvider
	sumarySourceProvider := summary.NewSummaryProvider(informerFactory.Core().V1().Nodes().Lister(), kubeletClient, addrResolver)

	// new a pod cache
	podsIndex := kubecache.NewCache(informerFactory.Core().V1().Pods().Informer())
	// set up the source manager
	nginxConfig := nginxstats.GetConfig(clientConfig, o.NginxStatsPort, o.NginxStatsPath, o.InsecureKubeletTLS, o.DeprecatedCompletelyInsecureKubelet)
	nginxstatsClient, err := nginxstats.ClientFor(nginxConfig)
	if err != nil {
		return fmt.Errorf("unable to construct a client to connect to the nginx stats: %v", err)
	}
	selector := labels.NewSelector()
	rq, _ := labels.NewRequirement("nodefunc", selection.Equals, []string{o.NginxNodeLabel})
	selector.Add(*rq)
	// new a nginx stats source provider
	nginxstatsSourceProvider := nginxstats.NewNginxstatsProvider(informerFactory.Core().V1().Nodes().Lister(),
		nginxstatsClient, addrResolver, podsIndex, selector)

	scrapeTimeout := time.Duration(float64(o.MetricResolution) * 0.90) // scrape timeout is 90% of the scrape interval
	sources.RegisterDurationMetrics(scrapeTimeout)
	sourceManager := sources.NewSourceManager(scrapeTimeout, sumarySourceProvider, nginxstatsSourceProvider)

	// set up the general manager
	manager.RegisterDurationMetrics(o.MetricResolution)
	mgr := manager.NewManager(sourceManager, metricSink, o.MetricResolution)

	// inject the providers into the config
	o.WithCustomMetrics(provider)
	o.WithExternalMetrics(provider)

	// complete the config to get an API server
	server, err := o.Server()
	if err != nil {
		return err
	}

	// add health checks
	server.GenericAPIServer.AddHealthzChecks(healthz.NamedCheck("healthz", mgr.CheckHealth))

	// run everything (the apiserver runs the shared informer factory for us)
	mgr.RunUntil(stopCh)
	return o.Run(stopCh)
}

// RESTMapper returns a RESTMapper dynamically populated with discovery information.
// The discovery information will be periodically repopulated according to DiscoveryInterval.
func (o *CustomMetricsServerOptions) RESTMapper(stopChan <-chan struct{}) (apimeta.RESTMapper, error) {
	if o.restMapper == nil {
		discoveryClient, err := o.DiscoveryClient()
		if err != nil {
			return nil, err
		}
		// NB: since we never actually look at the contents of
		// the objects we fetch (beyond ObjectMeta), unstructured should be fine
		dynamicMapper, err := dynamicmapper.NewRESTMapper(discoveryClient, o.DiscoveryInterval)
		if err != nil {
			return nil, fmt.Errorf("unable to construct dynamic discovery mapper: %v", err)
		}
		if stopChan != nil {
			dynamicMapper.RunUntil(stopChan)
		}

		o.restMapper = dynamicMapper
	}
	return o.restMapper, nil
}
