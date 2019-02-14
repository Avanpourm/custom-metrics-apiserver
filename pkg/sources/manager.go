// Copyright: meitu.com
// Author: JamesBryce
// Author Email: zmp@meitu.com
// Date: 2018-1-12
// Last Modified by:   JamesBryce

package sources

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"

	utilmetrics "github.com/kubernetes-incubator/custom-metrics-apiserver/pkg/metrics"
)

const (
	maxDelayMs       = 4 * 1000
	delayPerSourceMs = 8
)

var (
	lastScrapeTimestamp = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "metrics_server",
			Subsystem: "scraper",
			Name:      "last_time_seconds",
			Help:      "Last time custom-metrics-apiserver performed a scrape since unix epoch in seconds.",
		},
		[]string{"source"},
	)

	// initialized below to an actual value by a call to RegisterScraperDuration
	// (acts as a no-op by default), but we can't just register it in the constructor,
	// since it could be called multiple times during setup.
	scraperDuration *prometheus.HistogramVec = prometheus.NewHistogramVec(prometheus.HistogramOpts{}, []string{"source"})
)

func init() {
	prometheus.MustRegister(lastScrapeTimestamp)
}

// RegisterScraperDuration creates and registers a histogram metric for
// scrape duration, suitable for use in the source manager.
func RegisterDurationMetrics(scrapeTimeout time.Duration) {
	scraperDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "metrics_server",
			Subsystem: "scraper",
			Name:      "duration_seconds",
			Help:      "Time spent scraping sources in seconds.",
			Buckets:   utilmetrics.BucketsForScrapeDuration(scrapeTimeout),
		},
		[]string{"source"},
	)
	prometheus.MustRegister(scraperDuration)
}

func NewSourceManager(scrapeTimeout time.Duration, srcProvs ...MetricSourceProvider) MetricSource {
	return &sourceManager{
		srcProvs:      srcProvs,
		scrapeTimeout: scrapeTimeout,
		history:       NewCustomMetricsBatch(),
	}
}

type sourceManager struct {
	srcProvs      []MetricSourceProvider
	scrapeTimeout time.Duration
	history       *CustomMetricsBatch
}

func (m *sourceManager) Name() string {
	return "source_manager"
}

func (m *sourceManager) Collect(baseCtx context.Context) (*CustomMetricsBatch, error) {
	var errs []error
	sources := []MetricSource{}
	for _, srcProv := range m.srcProvs {
		source, err := srcProv.GetMetricSources()
		if err != nil {
			// save the error, and continue on in case of partial results
			errs = append(errs, err)
		}
		sources = append(sources, source...)
	}

	glog.V(1).Infof("Scraping metrics from %v sources", len(sources))

	responseChannel := make(chan *CustomMetricsBatch, len(sources))
	errChannel := make(chan error, len(sources))
	defer close(responseChannel)
	defer close(errChannel)

	startTime := time.Now()

	// TODO: re-evaluate this code -- do we really need to stagger fetches like this?
	delayMs := delayPerSourceMs * len(sources)
	if delayMs > maxDelayMs {
		delayMs = maxDelayMs
	}

	for _, source := range sources {
		go func(source MetricSource) {
			// Prevents network congestion.
			sleepDuration := time.Duration(rand.Intn(delayMs)) * time.Millisecond
			time.Sleep(sleepDuration)
			// make the timeout a bit shorter to account for staggering, so we still preserve
			// the overall timeout
			ctx, cancelTimeout := context.WithTimeout(baseCtx, m.scrapeTimeout-sleepDuration)
			defer cancelTimeout()

			glog.V(2).Infof("Querying source: %s", source)
			metrics, err := scrapeWithMetrics(ctx, source)
			if err != nil {
				errChannel <- fmt.Errorf("unable to fully scrape metrics from source %s: %v", source.Name(), err)
				responseChannel <- metrics
				return
			}
			responseChannel <- metrics
			errChannel <- nil
		}(source)
	}

	res := NewCustomMetricsBatch()
	raw := NewCustomMetricsBatch()
	// var nodeNum, PodNum, EndpointQPSsNum int64
	for range sources {
		err := <-errChannel
		srcBatch := <-responseChannel
		if err != nil {
			errs = append(errs, err)
			// NB: partial node results are still worth saving, so
			// don't skip storing results if we got an error
		}
		if srcBatch == nil {
			continue
		}
		raw.Merge(srcBatch)
	}
	res.Filter(m.history, raw)
	m.history = raw
	glog.V(1).Infof("ScrapeMetrics: time: %s, nodes: %v, pods: %v, Endpoint: %v",
		time.Since(startTime), res.CountNodeMetrics, res.CountPodMetrics, res.CountEndpointMetrics)
	return res, utilerrors.NewAggregate(errs)
}

func scrapeWithMetrics(ctx context.Context, s MetricSource) (*CustomMetricsBatch, error) {
	sourceName := s.Name()
	startTime := time.Now()
	defer lastScrapeTimestamp.
		WithLabelValues(sourceName).
		Set(float64(time.Now().Unix()))
	defer scraperDuration.
		WithLabelValues(sourceName).
		Observe(float64(time.Since(startTime)) / float64(time.Second))

	return s.Collect(ctx)
}
