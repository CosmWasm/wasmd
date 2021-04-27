package keeper

import (
	wasmvmtypes "github.com/CosmWasm/wasmvm/types"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	labelPinned = "pinned"
	labelMemory = "memory"
	labelFs     = "memory"
)

// metricSource source of wasmvm metrics
type metricSource interface {
	GetMetrics() wasmvmtypes.Metrics
}

var _ prometheus.Collector = (*WasmVMMetricsCollector)(nil)

// WasmVMMetricsCollector custom metrics collector to be used with Prometheus
type WasmVMMetricsCollector struct {
	source             metricSource
	CacheHitsDescr     *prometheus.Desc
	CacheMissesDescr   *prometheus.Desc
	CacheElementsDescr *prometheus.Desc
	CacheSizeDescr     *prometheus.Desc
}

//NewWasmVMMetricsCollector constructor
func NewWasmVMMetricsCollector(s metricSource) *WasmVMMetricsCollector {
	return &WasmVMMetricsCollector{
		source:             s,
		CacheHitsDescr:     prometheus.NewDesc("wasmvm_cache_hits_total", "Total number of cache hits", nil, nil),
		CacheMissesDescr:   prometheus.NewDesc("wasmvm_cache_misses_total", "Total number of cache misses", nil, nil),
		CacheElementsDescr: prometheus.NewDesc("wasmvm_cache_elements_total", "Total number of elements in the cache", nil, nil),
		CacheSizeDescr:     prometheus.NewDesc("wasmvm_cache_size_bytes", "Total number of elements in the cache", nil, nil),
	}
}

// Register registers all metrics
func (p *WasmVMMetricsCollector) Register(r prometheus.Registerer) {
	r.Register(p)
}

// Describe sends the super-set of all possible descriptors of metrics
func (p *WasmVMMetricsCollector) Describe(descs chan<- *prometheus.Desc) {
	descs <- p.CacheHitsDescr
	descs <- p.CacheMissesDescr
	descs <- p.CacheElementsDescr
	descs <- p.CacheSizeDescr
}

// Collect is called by the Prometheus registry when collecting metrics.
func (p *WasmVMMetricsCollector) Collect(c chan<- prometheus.Metric) {
	m := p.source.GetMetrics()
	c <- prometheus.MustNewConstMetric(p.CacheHitsDescr, prometheus.CounterValue, float64(m.HitsPinnedMemoryCache), labelPinned)
	c <- prometheus.MustNewConstMetric(p.CacheHitsDescr, prometheus.CounterValue, float64(m.HitsMemoryCache), labelMemory)
	c <- prometheus.MustNewConstMetric(p.CacheHitsDescr, prometheus.CounterValue, float64(m.HitsFsCache), labelFs)
	c <- prometheus.MustNewConstMetric(p.CacheMissesDescr, prometheus.CounterValue, float64(m.Misses))
	c <- prometheus.MustNewConstMetric(p.CacheElementsDescr, prometheus.GaugeValue, float64(m.ElementsPinnedMemoryCache), labelPinned)
	c <- prometheus.MustNewConstMetric(p.CacheElementsDescr, prometheus.GaugeValue, float64(m.ElementsMemoryCache), labelMemory)
	c <- prometheus.MustNewConstMetric(p.CacheSizeDescr, prometheus.GaugeValue, float64(m.SizeMemoryCache), labelMemory)
	c <- prometheus.MustNewConstMetric(p.CacheSizeDescr, prometheus.GaugeValue, float64(m.SizePinnedMemoryCache), labelPinned)
}
