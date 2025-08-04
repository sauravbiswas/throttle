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

type BenchmarkConfig struct {
	Duration    time.Duration
	Concurrency int
	KeyCount    int
	Limit       int64
	Interval    time.Duration
	Burst       int64
}

type BenchmarkResult struct {
	TotalRequests   int64
	AllowedRequests int64
	DeniedRequests  int64
	TotalLatency    time.Duration
	MinLatency      time.Duration
	MaxLatency      time.Duration
	AverageLatency  time.Duration
	Throughput      float64 // requests per second
}

func main() {
	var config BenchmarkConfig

	flag.DurationVar(&config.Duration, "duration", 10*time.Second, "Benchmark duration")
	flag.IntVar(&config.Concurrency, "concurrency", 10, "Number of concurrent workers")
	flag.IntVar(&config.KeyCount, "keys", 100, "Number of unique keys")
	flag.Int64Var(&config.Limit, "limit", 1000, "Rate limit")
	flag.DurationVar(&config.Interval, "interval", time.Minute, "Rate limit interval")
	flag.Int64Var(&config.Burst, "burst", 1500, "Burst capacity")
	flag.Parse()

	fmt.Printf("🚀 Throttle Benchmark\n")
	fmt.Printf("=====================\n")
	fmt.Printf("Duration: %v\n", config.Duration)
	fmt.Printf("Concurrency: %d workers\n", config.Concurrency)
	fmt.Printf("Keys: %d unique keys\n", config.KeyCount)
	fmt.Printf("Rate Limit: %d requests per %v\n", config.Limit, config.Interval)
	fmt.Printf("Burst: %d\n", config.Burst)
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

	// Run benchmarks
	fmt.Println("Running benchmarks...")
	fmt.Println()

	// Benchmark 1: Single key, high frequency
	fmt.Println("📊 Benchmark 1: Single Key, High Frequency")
	result1 := runBenchmark(limiter, config, 1, true)
	printBenchmarkResult("Single Key", result1)

	// Benchmark 2: Multiple keys, distributed load
	fmt.Println("📊 Benchmark 2: Multiple Keys, Distributed Load")
	result2 := runBenchmark(limiter, config, config.KeyCount, false)
	printBenchmarkResult("Multiple Keys", result2)

	// Benchmark 3: Mixed workload
	fmt.Println("📊 Benchmark 3: Mixed Workload (Grant + Preview)")
	result3 := runMixedBenchmark(limiter, config)
	printBenchmarkResult("Mixed Workload", result3)

	fmt.Println("🎉 Benchmark complete!")
}

func runBenchmark(limiter core.RateLimiter, config BenchmarkConfig, keyCount int, singleKey bool) BenchmarkResult {
	var (
		totalRequests   int64
		allowedRequests int64
		deniedRequests  int64
		totalLatency    int64
		minLatency      int64 = 1<<63 - 1
		maxLatency      int64
	)

	ctx, cancel := context.WithTimeout(context.Background(), config.Duration)
	defer cancel()

	var wg sync.WaitGroup
	startTime := time.Now()

	// Start workers
	for i := 0; i < config.Concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for {
				select {
				case <-ctx.Done():
					return
				default:
					// Choose key
					var key string
					if singleKey {
						key = "single-key"
					} else {
						key = fmt.Sprintf("key-%d", workerID%keyCount)
					}

					// Make request
					requestStart := time.Now()
					decision, err := limiter.Grant(ctx, key)
					latency := time.Since(requestStart)

					atomic.AddInt64(&totalRequests, 1)

					if err != nil {
						continue
					}

					if decision.Allowed {
						atomic.AddInt64(&allowedRequests, 1)
					} else {
						atomic.AddInt64(&deniedRequests, 1)
					}

					// Update latency stats
					latencyNs := latency.Nanoseconds()
					atomic.AddInt64(&totalLatency, latencyNs)

					// Update min/max (simplified - not perfectly accurate but good enough for benchmark)
					for {
						currentMin := atomic.LoadInt64(&minLatency)
						if latencyNs >= currentMin {
							break
						}
						if atomic.CompareAndSwapInt64(&minLatency, currentMin, latencyNs) {
							break
						}
					}

					for {
						currentMax := atomic.LoadInt64(&maxLatency)
						if latencyNs <= currentMax {
							break
						}
						if atomic.CompareAndSwapInt64(&maxLatency, currentMax, latencyNs) {
							break
						}
					}
				}
			}
		}(i)
	}

	wg.Wait()
	duration := time.Since(startTime)

	return BenchmarkResult{
		TotalRequests:   atomic.LoadInt64(&totalRequests),
		AllowedRequests: atomic.LoadInt64(&allowedRequests),
		DeniedRequests:  atomic.LoadInt64(&deniedRequests),
		TotalLatency:    time.Duration(atomic.LoadInt64(&totalLatency)),
		MinLatency:      time.Duration(atomic.LoadInt64(&minLatency)),
		MaxLatency:      time.Duration(atomic.LoadInt64(&maxLatency)),
		AverageLatency:  time.Duration(atomic.LoadInt64(&totalLatency)) / time.Duration(atomic.LoadInt64(&totalRequests)),
		Throughput:      float64(atomic.LoadInt64(&totalRequests)) / duration.Seconds(),
	}
}

func runMixedBenchmark(limiter core.RateLimiter, config BenchmarkConfig) BenchmarkResult {
	var (
		totalRequests   int64
		allowedRequests int64
		deniedRequests  int64
		totalLatency    int64
		minLatency      int64 = 1<<63 - 1
		maxLatency      int64
	)

	ctx, cancel := context.WithTimeout(context.Background(), config.Duration)
	defer cancel()

	var wg sync.WaitGroup
	startTime := time.Now()

	// Start workers
	for i := 0; i < config.Concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for {
				select {
				case <-ctx.Done():
					return
				default:
					key := fmt.Sprintf("mixed-key-%d", workerID%config.KeyCount)

					// Alternate between Grant and Preview
					var decision core.Decision
					var err error
					requestStart := time.Now()

					if workerID%2 == 0 {
						decision, err = limiter.Grant(ctx, key)
					} else {
						decision, err = limiter.Preview(ctx, key)
					}

					latency := time.Since(requestStart)
					atomic.AddInt64(&totalRequests, 1)

					if err != nil {
						continue
					}

					if decision.Allowed {
						atomic.AddInt64(&allowedRequests, 1)
					} else {
						atomic.AddInt64(&deniedRequests, 1)
					}

					// Update latency stats
					latencyNs := latency.Nanoseconds()
					atomic.AddInt64(&totalLatency, latencyNs)

					// Update min/max
					for {
						currentMin := atomic.LoadInt64(&minLatency)
						if latencyNs >= currentMin {
							break
						}
						if atomic.CompareAndSwapInt64(&minLatency, currentMin, latencyNs) {
							break
						}
					}

					for {
						currentMax := atomic.LoadInt64(&maxLatency)
						if latencyNs <= currentMax {
							break
						}
						if atomic.CompareAndSwapInt64(&maxLatency, currentMax, latencyNs) {
							break
						}
					}
				}
			}
		}(i)
	}

	wg.Wait()
	duration := time.Since(startTime)

	return BenchmarkResult{
		TotalRequests:   atomic.LoadInt64(&totalRequests),
		AllowedRequests: atomic.LoadInt64(&allowedRequests),
		DeniedRequests:  atomic.LoadInt64(&deniedRequests),
		TotalLatency:    time.Duration(atomic.LoadInt64(&totalLatency)),
		MinLatency:      time.Duration(atomic.LoadInt64(&minLatency)),
		MaxLatency:      time.Duration(atomic.LoadInt64(&maxLatency)),
		AverageLatency:  time.Duration(atomic.LoadInt64(&totalLatency)) / time.Duration(atomic.LoadInt64(&totalRequests)),
		Throughput:      float64(atomic.LoadInt64(&totalRequests)) / duration.Seconds(),
	}
}

func printBenchmarkResult(name string, result BenchmarkResult) {
	fmt.Printf("Results for %s:\n", name)
	fmt.Printf("  Total Requests: %d\n", result.TotalRequests)
	fmt.Printf("  Allowed: %d (%.1f%%)\n", result.AllowedRequests,
		float64(result.AllowedRequests)/float64(result.TotalRequests)*100)
	fmt.Printf("  Denied: %d (%.1f%%)\n", result.DeniedRequests,
		float64(result.DeniedRequests)/float64(result.TotalRequests)*100)
	fmt.Printf("  Throughput: %.2f req/sec\n", result.Throughput)
	fmt.Printf("  Average Latency: %v\n", result.AverageLatency)
	fmt.Printf("  Min Latency: %v\n", result.MinLatency)
	fmt.Printf("  Max Latency: %v\n", result.MaxLatency)
	fmt.Println()
}
