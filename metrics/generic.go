package metrics

import (
	"sync"
	"time"
)

// GenericReporter implements MetricsReporter with actual metric collection
type GenericReporter struct {
	collector MetricsCollector
	mu        sync.RWMutex
}

// NewGenericReporter creates a new generic metrics reporter
func NewGenericReporter() *GenericReporter {
	return &GenericReporter{
		collector: NewCollector(),
	}
}

// RecordGrant records a grant decision
func (g *GenericReporter) RecordGrant(key string, allowed bool, remaining int64) {
	g.mu.Lock()
	defer g.mu.Unlock()

	now := time.Now()

	// Record total grant requests
	g.collector.AddMetric(Metric{
		Name:      "throttle_grant_total",
		Type:      Counter,
		Value:     1.0,
		Labels:    map[string]string{"key": key},
		Timestamp: now,
		Help:      "Total number of grant requests",
	})

	// Record allowed/denied requests
	decision := "denied"
	if allowed {
		decision = "allowed"
		g.collector.AddMetric(Metric{
			Name:      "throttle_grant_allowed_total",
			Type:      Counter,
			Value:     1.0,
			Labels:    map[string]string{"key": key},
			Timestamp: now,
			Help:      "Total number of allowed grant requests",
		})
	} else {
		g.collector.AddMetric(Metric{
			Name:      "throttle_grant_denied_total",
			Type:      Counter,
			Value:     1.0,
			Labels:    map[string]string{"key": key},
			Timestamp: now,
			Help:      "Total number of denied grant requests",
		})
	}

	// Record remaining tokens
	g.collector.AddMetric(Metric{
		Name:      "throttle_remaining_tokens",
		Type:      Gauge,
		Value:     float64(remaining),
		Labels:    map[string]string{"key": key, "decision": decision},
		Timestamp: now,
		Help:      "Current number of remaining tokens",
	})
}

// RecordPreview records a preview operation
func (g *GenericReporter) RecordPreview(key string, remaining int64) {
	g.mu.Lock()
	defer g.mu.Unlock()

	now := time.Now()

	// Record preview requests
	g.collector.AddMetric(Metric{
		Name:      "throttle_preview_total",
		Type:      Counter,
		Value:     1.0,
		Labels:    map[string]string{"key": key},
		Timestamp: now,
		Help:      "Total number of preview requests",
	})

	// Record remaining tokens
	g.collector.AddMetric(Metric{
		Name:      "throttle_remaining_tokens",
		Type:      Gauge,
		Value:     float64(remaining),
		Labels:    map[string]string{"key": key, "operation": "preview"},
		Timestamp: now,
		Help:      "Current number of remaining tokens",
	})
}

// RecordClear records a clear operation
func (g *GenericReporter) RecordClear(key string) {
	g.mu.Lock()
	defer g.mu.Unlock()

	now := time.Now()

	// Record clear operations
	g.collector.AddMetric(Metric{
		Name:      "throttle_clear_total",
		Type:      Counter,
		Value:     1.0,
		Labels:    map[string]string{"key": key},
		Timestamp: now,
		Help:      "Total number of clear operations",
	})

	// Record remaining tokens as 0 after clear
	g.collector.AddMetric(Metric{
		Name:      "throttle_remaining_tokens",
		Type:      Gauge,
		Value:     0.0,
		Labels:    map[string]string{"key": key, "operation": "clear"},
		Timestamp: now,
		Help:      "Current number of remaining tokens",
	})
}

// GetCollector returns the metrics collector
func (g *GenericReporter) GetCollector() MetricsCollector {
	return g.collector
}
