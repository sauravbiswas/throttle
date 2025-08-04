package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/throttle/backend/memory"
	"github.com/throttle/core"
	"github.com/throttle/metrics"
	"github.com/throttle/strategy/tokenbucket"
)

func main() {
	fmt.Println("🎯 Throttle Metrics Example")
	fmt.Println("===========================")
	fmt.Println()

	// Create rate limiter with generic metrics
	backend := memory.NewBackend()
	strategy := tokenbucket.NewStrategy(core.Config{
		Limit:    10,
		Interval: time.Minute,
		Burst:    15,
	})
	reporter := metrics.NewGenericReporter()

	limiter := core.NewLimiter(backend, strategy, core.Config{
		Limit:    10,
		Interval: time.Minute,
		Burst:    15,
	}, reporter)

	ctx := context.Background()

	// Simulate some requests
	fmt.Println("Making some requests to generate metrics...")

	keys := []string{"user-1", "user-2", "user-1", "user-3", "user-1"}

	for i, key := range keys {
		fmt.Printf("Request %d for %s: ", i+1, key)

		decision, err := limiter.Grant(ctx, key)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}

		if decision.Allowed {
			fmt.Printf("✅ ALLOWED (remaining: %d)\n", decision.Remaining)
		} else {
			fmt.Printf("🚫 DENIED (retry after: %v)\n", decision.RetryAfter)
		}

		time.Sleep(100 * time.Millisecond)
	}

	fmt.Println()
	fmt.Println("📊 Collected Metrics:")
	fmt.Println("====================")

	// Get the metrics collector
	collector := reporter.GetCollector()

	// Get all metrics
	allMetrics := collector.Collect()

	// Print metrics in a readable format
	for _, metric := range allMetrics {
		fmt.Printf("• %s (%s): %g\n", metric.Name, metric.Type, metric.Value)
		fmt.Printf("  Labels: %v\n", metric.Labels)
		fmt.Printf("  Help: %s\n", metric.Help)
		fmt.Println()
	}

	// Get metrics summary
	fmt.Println("📈 Metrics Summary:")
	fmt.Println("==================")
	summary := collector.GetMetricsSummary()

	for metricName, typeCounts := range summary {
		fmt.Printf("• %s:\n", metricName)
		for metricType, count := range typeCounts {
			fmt.Printf("  - %s: %d\n", metricType, count)
		}
	}

	// Get specific metrics
	fmt.Println()
	fmt.Println("🔍 Specific Metrics:")
	fmt.Println("===================")

	grantMetrics := collector.GetMetrics("throttle_grant_total")
	fmt.Printf("Grant requests: %d\n", len(grantMetrics))

	remainingMetrics := collector.GetMetrics("throttle_remaining_tokens")
	fmt.Printf("Remaining tokens metrics: %d\n", len(remainingMetrics))

	// Export as JSON (for monitoring systems)
	fmt.Println()
	fmt.Println("📤 JSON Export (for Datadog, etc.):")
	fmt.Println("===================================")

	jsonData := map[string]interface{}{
		"timestamp": time.Now().Format(time.RFC3339),
		"metrics":   allMetrics,
		"summary":   summary,
	}

	jsonBytes, _ := json.MarshalIndent(jsonData, "", "  ")
	fmt.Println(string(jsonBytes))

	// Reset metrics
	fmt.Println()
	fmt.Println("🧹 Resetting metrics...")
	collector.Reset()

	remainingAfterReset := collector.Collect()
	fmt.Printf("Metrics after reset: %d\n", len(remainingAfterReset))
}
