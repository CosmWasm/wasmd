package keeper

import (
	"github.com/go-kit/kit/metrics"
	"github.com/go-kit/kit/metrics/discard"
	go_prometheus "github.com/go-kit/kit/metrics/prometheus"
	"github.com/prometheus/client_golang/prometheus"

	wasmvmtypes "github.com/Finschia/wasmvm/types"
)

const (
	labelPinned      = "pinned"
	labelMemory      = "memory"
	labelFs          = "fs"
	MetricsSubsystem = "wasm"
)

type Metrics struct {
	InstantiateElapsedTimes metrics.Histogram
	ExecuteElapsedTimes     metrics.Histogram
	MigrateElapsedTimes     metrics.Histogram
	SudoElapsedTimes        metrics.Histogram
	QuerySmartElapsedTimes  metrics.Histogram
	QueryRawElapsedTimes    metrics.Histogram
}

func PrometheusMetrics(namespace string, labelsAndValues ...string) *Metrics {
	return &Metrics{
		InstantiateElapsedTimes: go_prometheus.NewSummaryFrom(prometheus.SummaryOpts{
			Namespace: namespace,
			Subsystem: MetricsSubsystem,
			Name:      "instantiate",
			Help:      "elapsed time of Instantiate the wasm contract",
		}, nil),
		ExecuteElapsedTimes: go_prometheus.NewSummaryFrom(prometheus.SummaryOpts{
			Namespace: namespace,
			Subsystem: MetricsSubsystem,
			Name:      "execute",
			Help:      "elapsed time of Execute the wasm contract",
		}, nil),
		MigrateElapsedTimes: go_prometheus.NewSummaryFrom(prometheus.SummaryOpts{
			Namespace: namespace,
			Subsystem: MetricsSubsystem,
			Name:      "migrate",
			Help:      "elapsed time of Migrate the wasm contract",
		}, nil),
		SudoElapsedTimes: go_prometheus.NewSummaryFrom(prometheus.SummaryOpts{
			Namespace: namespace,
			Subsystem: MetricsSubsystem,
			Name:      "sudo",
			Help:      "elapsed time of Sudo the wasm contract",
		}, nil),
		QuerySmartElapsedTimes: go_prometheus.NewSummaryFrom(prometheus.SummaryOpts{
			Namespace: namespace,
			Subsystem: MetricsSubsystem,
			Name:      "query_smart",
			Help:      "elapsed time of QuerySmart the wasm contract",
		}, nil),
		QueryRawElapsedTimes: go_prometheus.NewSummaryFrom(prometheus.SummaryOpts{
			Namespace: namespace,
			Subsystem: MetricsSubsystem,
			Name:      "query_raw",
			Help:      "elapsed time of QueryRaw the wasm contract",
		}, nil),
	}
}

// NopMetrics returns no-op Metrics.
func NopMetrics() *Metrics {
	return &Metrics{
		InstantiateElapsedTimes: discard.NewHistogram(),
		ExecuteElapsedTimes:     discard.NewHistogram(),
		MigrateElapsedTimes:     discard.NewHistogram(),
		SudoElapsedTimes:        discard.NewHistogram(),
		QuerySmartElapsedTimes:  discard.NewHistogram(),
		QueryRawElapsedTimes:    discard.NewHistogram(),
	}
}

type MetricsProvider func() *Metrics

// PrometheusMetricsProvider returns PrometheusMetrics for each store
func PrometheusMetricsProvider(namespace string, labelsAndValues ...string) func() *Metrics {
	return func() *Metrics {
		return PrometheusMetrics(namespace, labelsAndValues...)
	}
}

// NopMetricsProvider returns NopMetrics for each store
func NopMetricsProvider() func() *Metrics {
	//nolint:gocritic
	return func() *Metrics {
		return NopMetrics()
	}
}

// metricSource source of wasmvm metrics
type metricSource interface {
	GetMetrics() (*wasmvmtypes.Metrics, error)
}

var _ prometheus.Collector = (*WasmVMCacheMetricsCollector)(nil)

// WasmVMCacheMetricsCollector custom metrics collector to be used with Prometheus
type WasmVMCacheMetricsCollector struct {
	source             metricSource
	CacheHitsDescr     *prometheus.Desc
	CacheMissesDescr   *prometheus.Desc
	CacheElementsDescr *prometheus.Desc
	CacheSizeDescr     *prometheus.Desc
}

// NewWasmVMCacheMetricsCollector constructor
func NewWasmVMCacheMetricsCollector(s metricSource) *WasmVMCacheMetricsCollector {
	return &WasmVMCacheMetricsCollector{
		source:             s,
		CacheHitsDescr:     prometheus.NewDesc("wasmvm_cache_hits_total", "Total number of cache hits", []string{"type"}, nil),
		CacheMissesDescr:   prometheus.NewDesc("wasmvm_cache_misses_total", "Total number of cache misses", nil, nil),
		CacheElementsDescr: prometheus.NewDesc("wasmvm_cache_elements_total", "Total number of elements in the cache", []string{"type"}, nil),
		CacheSizeDescr:     prometheus.NewDesc("wasmvm_cache_size_bytes", "Total number of elements in the cache", []string{"type"}, nil),
	}
}

// Register registers all metrics
func (p *WasmVMCacheMetricsCollector) Register(r prometheus.Registerer) {
	r.MustRegister(p)
}

// Describe sends the super-set of all possible descriptors of metrics
func (p *WasmVMCacheMetricsCollector) Describe(descs chan<- *prometheus.Desc) {
	descs <- p.CacheHitsDescr
	descs <- p.CacheMissesDescr
	descs <- p.CacheElementsDescr
	descs <- p.CacheSizeDescr
}

// Collect is called by the Prometheus registry when collecting metrics.
func (p *WasmVMCacheMetricsCollector) Collect(c chan<- prometheus.Metric) {
	m, err := p.source.GetMetrics()
	if err != nil {
		return
	}
	c <- prometheus.MustNewConstMetric(p.CacheHitsDescr, prometheus.CounterValue, float64(m.HitsPinnedMemoryCache), labelPinned)
	c <- prometheus.MustNewConstMetric(p.CacheHitsDescr, prometheus.CounterValue, float64(m.HitsMemoryCache), labelMemory)
	c <- prometheus.MustNewConstMetric(p.CacheHitsDescr, prometheus.CounterValue, float64(m.HitsFsCache), labelFs)
	c <- prometheus.MustNewConstMetric(p.CacheMissesDescr, prometheus.CounterValue, float64(m.Misses))
	c <- prometheus.MustNewConstMetric(p.CacheElementsDescr, prometheus.GaugeValue, float64(m.ElementsPinnedMemoryCache), labelPinned)
	c <- prometheus.MustNewConstMetric(p.CacheElementsDescr, prometheus.GaugeValue, float64(m.ElementsMemoryCache), labelMemory)
	c <- prometheus.MustNewConstMetric(p.CacheSizeDescr, prometheus.GaugeValue, float64(m.SizeMemoryCache), labelMemory)
	c <- prometheus.MustNewConstMetric(p.CacheSizeDescr, prometheus.GaugeValue, float64(m.SizePinnedMemoryCache), labelPinned)
	// Node about fs metrics:
	// The number of elements and the size of elements in the file system cache cannot easily be obtained.
	// We had to either scan the whole directory of potentially thousands of files or track the values when files are added or removed.
	// Such a tracking would need to be on disk such that the values are not cleared when the node is restarted.
}
