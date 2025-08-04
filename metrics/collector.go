package metrics

import (
	"sync"
)

// Collector implements MetricsCollector with thread-safe metric storage
type Collector struct {
	mu      sync.RWMutex
	metrics map[string][]Metric
}

// NewCollector creates a new metrics collector
func NewCollector() *Collector {
	return &Collector{
		metrics: make(map[string][]Metric),
	}
}

// AddMetric adds a metric to the collector
func (c *Collector) AddMetric(metric Metric) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.metrics[metric.Name] == nil {
		c.metrics[metric.Name] = make([]Metric, 0)
	}
	c.metrics[metric.Name] = append(c.metrics[metric.Name], metric)
}

// Collect returns all current metrics
func (c *Collector) Collect() []Metric {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var allMetrics []Metric
	for _, metrics := range c.metrics {
		allMetrics = append(allMetrics, metrics...)
	}
	return allMetrics
}

// GetMetrics returns metrics for a specific name
func (c *Collector) GetMetrics(name string) []Metric {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if metrics, exists := c.metrics[name]; exists {
		// Return a copy to prevent external modifications
		result := make([]Metric, len(metrics))
		copy(result, metrics)
		return result
	}
	return nil
}

// Reset clears all metrics
func (c *Collector) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.metrics = make(map[string][]Metric)
}

// GetMetricsByType returns all metrics of a specific type
func (c *Collector) GetMetricsByType(metricType MetricType) []Metric {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var result []Metric
	for _, metrics := range c.metrics {
		for _, metric := range metrics {
			if metric.Type == metricType {
				result = append(result, metric)
			}
		}
	}
	return result
}

// GetMetricsSummary returns a summary of metrics by name and type
func (c *Collector) GetMetricsSummary() map[string]map[MetricType]int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	summary := make(map[string]map[MetricType]int)

	for name, metrics := range c.metrics {
		summary[name] = make(map[MetricType]int)
		for _, metric := range metrics {
			summary[name][metric.Type]++
		}
	}

	return summary
}
