package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/throttle/backend/memory"
	"github.com/throttle/core"
	"github.com/throttle/metrics"
	"github.com/throttle/strategy/tokenbucket"
)

func main() {
	fmt.Println("🎯 Throttle Interactive Load Tester")
	fmt.Println("====================================")
	fmt.Println()

	// Get configuration from user
	config := getConfigFromUser()

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

	fmt.Printf("\n✅ Rate limiter configured: %d requests per %v (burst: %d)\n",
		config.Limit, config.Interval, config.Burst)
	fmt.Println()

	// Start interactive mode
	runInteractiveMode(limiter, config)
}

type InteractiveConfig struct {
	Limit    int64
	Interval time.Duration
	Burst    int64
}

func getConfigFromUser() InteractiveConfig {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("Configure your rate limiter:")

	// Get limit
	fmt.Print("Rate limit (requests per interval): ")
	limitStr, _ := reader.ReadString('\n')
	limit, _ := strconv.ParseInt(strings.TrimSpace(limitStr), 10, 64)
	if limit <= 0 {
		limit = 10
		fmt.Printf("Using default limit: %d\n", limit)
	}

	// Get interval
	fmt.Print("Interval (e.g., 1m, 30s, 1h): ")
	intervalStr, _ := reader.ReadString('\n')
	interval, err := time.ParseDuration(strings.TrimSpace(intervalStr))
	if err != nil || interval <= 0 {
		interval = time.Minute
		fmt.Printf("Using default interval: %v\n", interval)
	}

	// Get burst
	fmt.Print("Burst capacity: ")
	burstStr, _ := reader.ReadString('\n')
	burst, _ := strconv.ParseInt(strings.TrimSpace(burstStr), 10, 64)
	if burst <= 0 {
		burst = limit + 5
		fmt.Printf("Using default burst: %d\n", burst)
	}

	return InteractiveConfig{
		Limit:    limit,
		Interval: interval,
		Burst:    burst,
	}
}

func runInteractiveMode(limiter core.RateLimiter, config InteractiveConfig) {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("Interactive Commands:")
	fmt.Println("  'test <key>'     - Test rate limiting for a specific key")
	fmt.Println("  'burst <key>'    - Rapid-fire requests to test burst capacity")
	fmt.Println("  'status <key>'   - Check current status without consuming tokens")
	fmt.Println("  'clear <key>'    - Reset rate limit for a key")
	fmt.Println("  'load <workers>' - Run a quick load test with N workers")
	fmt.Println("  'quit'           - Exit")
	fmt.Println()

	for {
		fmt.Print("> ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		if input == "quit" {
			fmt.Println("Goodbye!")
			break
		}

		parts := strings.Fields(input)
		if len(parts) == 0 {
			continue
		}

		command := parts[0]

		switch command {
		case "test":
			if len(parts) < 2 {
				fmt.Println("Usage: test <key>")
				continue
			}
			testKey(limiter, parts[1])

		case "burst":
			if len(parts) < 2 {
				fmt.Println("Usage: burst <key>")
				continue
			}
			burstTest(limiter, parts[1])

		case "status":
			if len(parts) < 2 {
				fmt.Println("Usage: status <key>")
				continue
			}
			checkStatus(limiter, parts[1])

		case "clear":
			if len(parts) < 2 {
				fmt.Println("Usage: clear <key>")
				continue
			}
			clearKey(limiter, parts[1])

		case "load":
			if len(parts) < 2 {
				fmt.Println("Usage: load <workers>")
				continue
			}
			workers, _ := strconv.Atoi(parts[1])
			if workers <= 0 {
				workers = 5
			}
			runQuickLoadTest(limiter, workers)

		default:
			fmt.Printf("Unknown command: %s\n", command)
		}
	}
}

func testKey(limiter core.RateLimiter, key string) {
	ctx := context.Background()
	start := time.Now()
	decision, err := limiter.Grant(ctx, key)
	latency := time.Since(start)

	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		return
	}

	if decision.Allowed {
		fmt.Printf("✅ ALLOWED - Key: %s, Remaining: %d, Latency: %v\n",
			key, decision.Remaining, latency)
	} else {
		fmt.Printf("🚫 DENIED  - Key: %s, Retry after: %v, Latency: %v\n",
			key, decision.RetryAfter, latency)
	}
}

func burstTest(limiter core.RateLimiter, key string) {
	fmt.Printf("🔥 Running burst test for key: %s\n", key)

	var allowed, denied int64
	start := time.Now()

	// Send 20 rapid requests
	for i := 0; i < 20; i++ {
		ctx := context.Background()
		decision, err := limiter.Grant(ctx, key)

		if err != nil {
			fmt.Printf("❌ Error on request %d: %v\n", i+1, err)
			continue
		}

		if decision.Allowed {
			atomic.AddInt64(&allowed, 1)
		} else {
			atomic.AddInt64(&denied, 1)
		}

		// Small delay to see the progression
		time.Sleep(10 * time.Millisecond)
	}

	duration := time.Since(start)
	fmt.Printf("📊 Burst Results: Allowed: %d, Denied: %d, Duration: %v\n",
		allowed, denied, duration)
}

func checkStatus(limiter core.RateLimiter, key string) {
	ctx := context.Background()
	decision, err := limiter.Preview(ctx, key)

	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		return
	}

	fmt.Printf("📊 Status for key '%s':\n", key)
	fmt.Printf("  Remaining tokens: %d\n", decision.Remaining)
	fmt.Printf("  Reset time: %v\n", decision.ResetTime)
	fmt.Printf("  Would be allowed: %t\n", decision.Allowed)
}

func clearKey(limiter core.RateLimiter, key string) {
	ctx := context.Background()
	err := limiter.Clear(ctx, key)

	if err != nil {
		fmt.Printf("❌ Error clearing key '%s': %v\n", key, err)
		return
	}

	fmt.Printf("🧹 Cleared rate limit for key: %s\n", key)
}

func runQuickLoadTest(limiter core.RateLimiter, workers int) {
	fmt.Printf("🚀 Running quick load test with %d workers for 10 seconds...\n", workers)

	var total, allowed, denied int64
	var wg sync.WaitGroup

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	start := time.Now()

	// Start workers
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			ticker := time.NewTicker(100 * time.Millisecond) // 10 req/sec per worker
			defer ticker.Stop()

			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					key := fmt.Sprintf("worker-%d", workerID)
					decision, err := limiter.Grant(ctx, key)

					atomic.AddInt64(&total, 1)

					if err != nil {
						continue
					}

					if decision.Allowed {
						atomic.AddInt64(&allowed, 1)
					} else {
						atomic.AddInt64(&denied, 1)
					}
				}
			}
		}(i)
	}

	wg.Wait()
	duration := time.Since(start)

	fmt.Printf("📊 Quick Load Test Results:\n")
	fmt.Printf("  Duration: %v\n", duration)
	fmt.Printf("  Total requests: %d\n", total)
	fmt.Printf("  Allowed: %d (%.1f%%)\n", allowed, float64(allowed)/float64(total)*100)
	fmt.Printf("  Denied: %d (%.1f%%)\n", denied, float64(denied)/float64(total)*100)
	fmt.Printf("  Rate: %.1f req/sec\n", float64(total)/duration.Seconds())
}
