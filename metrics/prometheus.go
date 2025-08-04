package metrics

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// PrometheusReporter implements MetricsReporter with Prometheus metrics
type PrometheusReporter struct {
	mu sync.RWMutex

	// Metrics
	grantTotal     *prometheus.CounterVec
	grantAllowed   *prometheus.CounterVec
	grantDenied    *prometheus.CounterVec
	previewTotal   *prometheus.CounterVec
	clearTotal     *prometheus.CounterVec
	remainingGauge *prometheus.GaugeVec
}

// NewPrometheusReporter creates a new Prometheus metrics reporter
func NewPrometheusReporter() *PrometheusReporter {
	return &PrometheusReporter{
		grantTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "throttle_grant_total",
				Help: "Total number of grant requests",
			},
			[]string{"key"},
		),
		grantAllowed: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "throttle_grant_allowed_total",
				Help: "Total number of allowed grant requests",
			},
			[]string{"key"},
		),
		grantDenied: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "throttle_grant_denied_total",
				Help: "Total number of denied grant requests",
			},
			[]string{"key"},
		),
		previewTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "throttle_preview_total",
				Help: "Total number of preview requests",
			},
			[]string{"key"},
		),
		clearTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "throttle_clear_total",
				Help: "Total number of clear operations",
			},
			[]string{"key"},
		),
		remainingGauge: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "throttle_remaining_tokens",
				Help: "Current number of remaining tokens",
			},
			[]string{"key"},
		),
	}
}

// RecordGrant records a grant decision
func (p *PrometheusReporter) RecordGrant(key string, allowed bool, remaining int64) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.grantTotal.WithLabelValues(key).Inc()

	if allowed {
		p.grantAllowed.WithLabelValues(key).Inc()
	} else {
		p.grantDenied.WithLabelValues(key).Inc()
	}

	p.remainingGauge.WithLabelValues(key).Set(float64(remaining))
}

// RecordPreview records a preview operation
func (p *PrometheusReporter) RecordPreview(key string, remaining int64) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.previewTotal.WithLabelValues(key).Inc()
	p.remainingGauge.WithLabelValues(key).Set(float64(remaining))
}

// RecordClear records a clear operation
func (p *PrometheusReporter) RecordClear(key string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.clearTotal.WithLabelValues(key).Inc()
	p.remainingGauge.WithLabelValues(key).Set(0)
}
