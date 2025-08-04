package metrics

import "time"

// MetricType represents the type of metric
type MetricType string

const (
	Counter   MetricType = "counter"
	Gauge     MetricType = "gauge"
	Histogram MetricType = "histogram"
)

// Metric represents a generic metric that can be consumed by any monitoring system
type Metric struct {
	Name      string            `json:"name"`
	Type      MetricType        `json:"type"`
	Value     float64           `json:"value"`
	Labels    map[string]string `json:"labels,omitempty"`
	Timestamp time.Time         `json:"timestamp"`
	Help      string            `json:"help,omitempty"`
}

// MetricsCollector defines the interface for collecting metrics
type MetricsCollector interface {
	// AddMetric adds a metric to the collector
	AddMetric(metric Metric)

	// Collect returns all current metrics
	Collect() []Metric

	// GetMetrics returns metrics for a specific name
	GetMetrics(name string) []Metric

	// Reset clears all metrics
	Reset()

	// GetMetricsSummary returns a summary of metrics by name and type
	GetMetricsSummary() map[string]map[MetricType]int
}

// MetricsReporter defines the interface for reporting metrics
type MetricsReporter interface {
	// RecordGrant records a grant decision
	RecordGrant(key string, allowed bool, remaining int64)

	// RecordPreview records a preview operation
	RecordPreview(key string, remaining int64)

	// RecordClear records a clear operation
	RecordClear(key string)

	// GetCollector returns the metrics collector
	GetCollector() MetricsCollector
}
