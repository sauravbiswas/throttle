package main

import (
	"context"
	"flag"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/throttle/backend/memory"
	"github.com/throttle/core"
	"github.com/throttle/metrics"
	"github.com/throttle/strategy/tokenbucket"
)

type LoadTestConfig struct {
	Duration     time.Duration
	Concurrency  int
	RatePerSec   int
	KeyCount     int
	Limit        int64
	Interval     time.Duration
	Burst        int64
	ShowProgress bool
}

type LoadTestResult struct {
	TotalRequests   int64
	AllowedRequests int64
	DeniedRequests  int64
	AverageLatency  time.Duration
	MinLatency      time.Duration
	MaxLatency      time.Duration
	P95Latency      time.Duration
	P99Latency      time.Duration
	Errors          int64
}

func main() {
	var config LoadTestConfig

	flag.DurationVar(&config.Duration, "duration", 30*time.Second, "Test duration")
	flag.IntVar(&config.Concurrency, "concurrency", 10, "Number of concurrent workers")
	flag.IntVar(&config.RatePerSec, "rate", 100, "Requests per second per worker")
	flag.IntVar(&config.KeyCount, "keys", 5, "Number of unique keys to test")
	flag.Int64Var(&config.Limit, "limit", 10, "Rate limit (requests per interval)")
	flag.DurationVar(&config.Interval, "interval", time.Minute, "Rate limit interval")
	flag.Int64Var(&config.Burst, "burst", 15, "Burst capacity")
	flag.BoolVar(&config.ShowProgress, "progress", true, "Show progress during test")
	flag.Parse()

	fmt.Printf("Starting load test with configuration:\n")
	fmt.Printf("  Duration: %v\n", config.Duration)
	fmt.Printf("  Concurrency: %d workers\n", config.Concurrency)
	fmt.Printf("  Rate: %d req/sec per worker\n", config.RatePerSec)
	fmt.Printf("  Keys: %d unique keys\n", config.KeyCount)
	fmt.Printf("  Rate Limit: %d requests per %v\n", config.Limit, config.Interval)
	fmt.Printf("  Burst: %d\n", config.Burst)
	fmt.Printf("  Total expected rate: %d req/sec\n", config.Concurrency*config.RatePerSec)
	fmt.Println()

	// Create rate limiter
	backend := memory.NewBackend()
	strategy := tokenbucket.NewStrategy(core.Config{
		Limit:    config.Limit,
		Interval: config.Interval,
		Burst:    config.Burst,
	})
	metrics := metrics.NewNoOpReporter()

	limiter := core.NewLimiter(backend, strategy, core.Config{
		Limit:    config.Limit,
		Interval: config.Interval,
		Burst:    config.Burst,
	}, metrics)

	// Run load test
	result := runLoadTest(limiter, config)

	// Print results
	printResults(result, config)
}

func runLoadTest(limiter core.RateLimiter, config LoadTestConfig) LoadTestResult {
	var (
		totalRequests   int64
		allowedRequests int64
		deniedRequests  int64
		errors          int64
		latencies       []time.Duration
		latencyMutex    sync.Mutex
	)

	ctx, cancel := context.WithTimeout(context.Background(), config.Duration)
	defer cancel()

	// Create worker pool
	var wg sync.WaitGroup
	startTime := time.Now()

	// Progress reporting
	if config.ShowProgress {
		go func() {
			ticker := time.NewTicker(time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					total := atomic.LoadInt64(&totalRequests)
					allowed := atomic.LoadInt64(&allowedRequests)
					denied := atomic.LoadInt64(&deniedRequests)
					elapsed := time.Since(startTime).Seconds()
					rate := float64(total) / elapsed
					fmt.Printf("\rProgress: %d requests (%.1f req/sec) - Allowed: %d, Denied: %d",
						total, rate, allowed, denied)
				}
			}
		}()
	}

	// Start workers
	for i := 0; i < config.Concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			// Calculate delay between requests to achieve desired rate
			delay := time.Second / time.Duration(config.RatePerSec)
			ticker := time.NewTicker(delay)
			defer ticker.Stop()

			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					// Use different keys to simulate multiple clients
					key := fmt.Sprintf("client-%d", workerID%config.KeyCount)

					requestStart := time.Now()
					decision, err := limiter.Grant(ctx, key)
					latency := time.Since(requestStart)

					atomic.AddInt64(&totalRequests, 1)

					if err != nil {
						atomic.AddInt64(&errors, 1)
						continue
					}

					if decision.Allowed {
						atomic.AddInt64(&allowedRequests, 1)
					} else {
						atomic.AddInt64(&deniedRequests, 1)
					}

					// Record latency
					latencyMutex.Lock()
					latencies = append(latencies, latency)
					latencyMutex.Unlock()
				}
			}
		}(i)
	}

	// Wait for all workers to complete
	wg.Wait()

	if config.ShowProgress {
		fmt.Println() // New line after progress
	}

	// Calculate latency statistics
	latencyMutex.Lock()
	defer latencyMutex.Unlock()

	result := LoadTestResult{
		TotalRequests:   atomic.LoadInt64(&totalRequests),
		AllowedRequests: atomic.LoadInt64(&allowedRequests),
		DeniedRequests:  atomic.LoadInt64(&deniedRequests),
		Errors:          atomic.LoadInt64(&errors),
	}

	if len(latencies) > 0 {
		result.MinLatency = latencies[0]
		result.MaxLatency = latencies[0]
		var total time.Duration

		for _, lat := range latencies {
			total += lat
			if lat < result.MinLatency {
				result.MinLatency = lat
			}
			if lat > result.MaxLatency {
				result.MaxLatency = lat
			}
		}

		result.AverageLatency = total / time.Duration(len(latencies))

		// Calculate percentiles (simplified)
		if len(latencies) >= 100 {
			p95Index := int(float64(len(latencies)) * 0.95)
			p99Index := int(float64(len(latencies)) * 0.99)
			result.P95Latency = latencies[p95Index]
			result.P99Latency = latencies[p99Index]
		}
	}

	return result
}

func printResults(result LoadTestResult, config LoadTestConfig) {
	fmt.Printf("\n=== Load Test Results ===\n")
	fmt.Printf("Test Duration: %v\n", config.Duration)
	fmt.Printf("Concurrency: %d workers\n", config.Concurrency)
	fmt.Printf("Target Rate: %d req/sec per worker (%d total)\n", config.RatePerSec, config.Concurrency*config.RatePerSec)
	fmt.Printf("Rate Limit: %d requests per %v (burst: %d)\n", config.Limit, config.Interval, config.Burst)
	fmt.Printf("Unique Keys: %d\n", config.KeyCount)
	fmt.Printf("\n")

	fmt.Printf("Total Requests: %d\n", result.TotalRequests)
	fmt.Printf("Allowed Requests: %d (%.2f%%)\n", result.AllowedRequests,
		float64(result.AllowedRequests)/float64(result.TotalRequests)*100)
	fmt.Printf("Denied Requests: %d (%.2f%%)\n", result.DeniedRequests,
		float64(result.DeniedRequests)/float64(result.TotalRequests)*100)
	fmt.Printf("Errors: %d\n", result.Errors)

	actualRate := float64(result.TotalRequests) / config.Duration.Seconds()
	fmt.Printf("Actual Rate: %.2f req/sec\n", actualRate)

	fmt.Printf("\nLatency Statistics:\n")
	fmt.Printf("  Average: %v\n", result.AverageLatency)
	fmt.Printf("  Min: %v\n", result.MinLatency)
	fmt.Printf("  Max: %v\n", result.MaxLatency)
	if result.P95Latency > 0 {
		fmt.Printf("  P95: %v\n", result.P95Latency)
	}
	if result.P99Latency > 0 {
		fmt.Printf("  P99: %v\n", result.P99Latency)
	}

	// Rate limiting effectiveness analysis
	expectedAllowed := int64(config.Limit) * int64(config.Duration/config.Interval)
	if config.Duration < config.Interval {
		expectedAllowed = int64(config.Limit)*int64(config.Duration/config.Interval) + config.Burst
	}

	fmt.Printf("\nRate Limiting Analysis:\n")
	fmt.Printf("  Expected allowed (theoretical): ~%d\n", expectedAllowed)
	fmt.Printf("  Actual allowed: %d\n", result.AllowedRequests)
	fmt.Printf("  Effectiveness: %.2f%%\n",
		float64(result.DeniedRequests)/float64(result.TotalRequests)*100)
}
