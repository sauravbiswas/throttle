package main

import (
	"context"
	"fmt"
	"time"

	"github.com/throttle/backend/memory"
	"github.com/throttle/core"
	"github.com/throttle/metrics"
	"github.com/throttle/strategy/leakybucket"
	"github.com/throttle/strategy/tokenbucket"
)

func main() {
	fmt.Println("🔄 Throttle Strategy Comparison")
	fmt.Println("===============================")
	fmt.Println()

	// Configuration for both strategies
	config := core.Config{
		Limit:    10,          // 10 requests per minute
		Interval: time.Minute, // 1 minute window
		Burst:    15,          // 15 burst capacity
	}

	// Create backends and metrics
	backend1 := memory.NewBackend()
	backend2 := memory.NewBackend()
	metrics1 := metrics.NewGenericReporter()
	metrics2 := metrics.NewGenericReporter()

	// Create strategies
	tokenBucketStrategy := tokenbucket.NewStrategy(config)
	leakyBucketStrategy := leakybucket.NewStrategy(config)

	// Create limiters
	tokenBucketLimiter := core.NewLimiter(backend1, tokenBucketStrategy, config, metrics1)
	leakyBucketLimiter := core.NewLimiter(backend2, leakyBucketStrategy, config, metrics2)

	ctx := context.Background()

	fmt.Println("📊 Comparing Token Bucket vs Leaky Bucket")
	fmt.Println("=========================================")
	fmt.Println()

	// Test 1: Initial burst
	fmt.Println("🧪 Test 1: Initial Burst (20 rapid requests)")
	fmt.Println("---------------------------------------------")

	results := make([]string, 20)
	for i := 0; i < 20; i++ {
		key := fmt.Sprintf("user-%d", i%5) // Use 5 different users

		// Test token bucket
		tbDecision, _ := tokenBucketLimiter.Grant(ctx, key)
		tbResult := "✅"
		if !tbDecision.Allowed {
			tbResult = "❌"
		}

		// Test leaky bucket
		lbDecision, _ := leakyBucketLimiter.Grant(ctx, key)
		lbResult := "✅"
		if !lbDecision.Allowed {
			lbResult = "❌"
		}

		results[i] = fmt.Sprintf("Request %2d: Token Bucket %s, Leaky Bucket %s", i+1, tbResult, lbResult)
	}

	// Print results in columns
	for i := 0; i < 20; i += 2 {
		if i+1 < len(results) {
			fmt.Printf("%-35s | %s\n", results[i], results[i+1])
		} else {
			fmt.Printf("%s\n", results[i])
		}
	}

	fmt.Println()
	fmt.Println("📈 Test 2: Time-based Behavior")
	fmt.Println("===============================")

	// Reset for time-based test
	backend1.Close()
	backend2.Close()
	backend1 = memory.NewBackend()
	backend2 = memory.NewBackend()

	tokenBucketLimiter = core.NewLimiter(backend1, tokenBucketStrategy, config, metrics1)
	leakyBucketLimiter = core.NewLimiter(backend2, leakyBucketStrategy, config, metrics2)

	// Test over time
	fmt.Println("Testing behavior over 30 seconds...")
	fmt.Println("Time    | Token Bucket | Leaky Bucket")
	fmt.Println("--------|-------------|-------------")

	for i := 0; i < 30; i++ {
		time.Sleep(1 * time.Second)

		// Try to make a request
		tbDecision, _ := tokenBucketLimiter.Grant(ctx, "test-user")
		lbDecision, _ := leakyBucketLimiter.Grant(ctx, "test-user")

		tbStatus := "✅"
		if !tbDecision.Allowed {
			tbStatus = "❌"
		}

		lbStatus := "✅"
		if !lbDecision.Allowed {
			lbStatus = "❌"
		}

		fmt.Printf("%6ds | %s (%-2d)     | %s (%-2d)\n",
			i+1, tbStatus, tbDecision.Remaining, lbStatus, lbDecision.Remaining)
	}

	fmt.Println()
	fmt.Println("🔍 Key Differences:")
	fmt.Println("===================")
	fmt.Println("• Token Bucket: Allows burst up to capacity, then refills over time")
	fmt.Println("• Leaky Bucket: Processes requests at a steady rate, no burst")
	fmt.Println("• Token Bucket: Better for handling traffic spikes")
	fmt.Println("• Leaky Bucket: Better for smoothing traffic flow")
	fmt.Println()

	// Show metrics comparison
	fmt.Println("📊 Metrics Comparison:")
	fmt.Println("======================")

	tbCollector := metrics1.GetCollector()
	lbCollector := metrics2.GetCollector()

	tbMetrics := tbCollector.Collect()
	lbMetrics := lbCollector.Collect()

	fmt.Printf("Token Bucket - Total Requests: %d\n", len(tbMetrics))
	fmt.Printf("Leaky Bucket - Total Requests: %d\n", len(lbMetrics))

	// Count allowed vs denied
	tbAllowed := 0
	tbDenied := 0
	lbAllowed := 0
	lbDenied := 0

	for _, metric := range tbMetrics {
		if metric.Name == "throttle_grant_allowed_total" {
			tbAllowed++
		} else if metric.Name == "throttle_grant_denied_total" {
			tbDenied++
		}
	}

	for _, metric := range lbMetrics {
		if metric.Name == "throttle_grant_allowed_total" {
			lbAllowed++
		} else if metric.Name == "throttle_grant_denied_total" {
			lbDenied++
		}
	}

	fmt.Printf("Token Bucket - Allowed: %d, Denied: %d\n", tbAllowed, tbDenied)
	fmt.Printf("Leaky Bucket - Allowed: %d, Denied: %d\n", lbAllowed, lbDenied)

	fmt.Println()
	fmt.Println("💡 Use Cases:")
	fmt.Println("=============")
	fmt.Println("Token Bucket: API rate limiting, user quotas, burst handling")
	fmt.Println("Leaky Bucket: Network traffic shaping, queue processing, smooth output")
}
