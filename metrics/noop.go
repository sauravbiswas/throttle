package metrics

// NoOpReporter implements MetricsReporter with no-op operations
type NoOpReporter struct {
	collector MetricsCollector
}

// NewNoOpReporter creates a new no-op metrics reporter
func NewNoOpReporter() *NoOpReporter {
	return &NoOpReporter{
		collector: NewCollector(),
	}
}

// RecordGrant records a grant decision (no-op)
func (n *NoOpReporter) RecordGrant(key string, allowed bool, remaining int64) {
	// No-op implementation
}

// RecordPreview records a preview operation (no-op)
func (n *NoOpReporter) RecordPreview(key string, remaining int64) {
	// No-op implementation
}

// RecordClear records a clear operation (no-op)
func (n *NoOpReporter) RecordClear(key string) {
	// No-op implementation
}

// GetCollector returns the metrics collector
func (n *NoOpReporter) GetCollector() MetricsCollector {
	return n.collector
}
