package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	redisclient "github.com/redis/go-redis/v9"
	"github.com/throttle/backend/redis"
	"github.com/throttle/core"
	"github.com/throttle/metrics"
	"github.com/throttle/strategy/tokenbucket"
)

func main() {
	fmt.Println("🚀 Throttle Redis Backend Server")
	fmt.Println("================================")
	fmt.Println()

	// Create Redis client
	rdb := redisclient.NewClient(&redisclient.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	// Test Redis connection
	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	fmt.Println("✅ Connected to Redis")

	// Create Redis backend
	backend, err := redis.NewBackendFromURL("redis://localhost:6379/0", "throttle-server")
	if err != nil {
		log.Fatalf("Failed to create Redis backend: %v", err)
	}
	defer backend.Close()

	// Create rate limiter configuration
	config := core.Config{
		Limit:    10,          // 10 requests per minute
		Interval: time.Minute, // 1 minute window
		Burst:    15,          // 15 burst capacity
	}

	// Create strategy and metrics
	strategy := tokenbucket.NewStrategy(config)
	reporter := metrics.NewGenericReporter()

	// Create limiter with Redis backend
	limiter := core.NewLimiter(backend, strategy, config, reporter)

	// Create HTTP server with rate limiting middleware
	http.HandleFunc("/api/resource", rateLimitMiddleware(limiter, handleResource))
	http.HandleFunc("/api/status", rateLimitMiddleware(limiter, handleStatus))
	http.HandleFunc("/api/clear", handleClear(limiter))

	fmt.Println("🌐 Starting HTTP server on :8080")
	fmt.Println("Endpoints:")
	fmt.Println("  GET /api/resource - Make a request (rate limited)")
	fmt.Println("  GET /api/status   - Check current state")
	fmt.Println("  GET /api/clear    - Reset rate limit for your IP")
	fmt.Println()

	log.Fatal(http.ListenAndServe(":8080", nil))
}

// rateLimitMiddleware applies rate limiting to HTTP handlers
func rateLimitMiddleware(limiter *core.Limiter, handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Use client IP as the rate limit key
		clientIP := getClientIP(r)

		// Check rate limit
		decision, err := limiter.Grant(r.Context(), clientIP)
		if err != nil {
			http.Error(w, "Rate limit error", http.StatusInternalServerError)
			return
		}

		// Set rate limit headers
		w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", limiter.Config().Limit))
		w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", decision.Remaining))
		w.Header().Set("X-RateLimit-Reset", decision.ResetTime.Format(time.RFC3339))

		if !decision.Allowed {
			w.Header().Set("Retry-After", fmt.Sprintf("%.0f", decision.RetryAfter.Seconds()))
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		// Call the actual handler
		handler(w, r)
	}
}

// handleResource handles the main resource endpoint
func handleResource(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]interface{}{
		"message": "Resource accessed successfully",
		"time":    time.Now().Format(time.RFC3339),
		"ip":      getClientIP(r),
	}

	fmt.Fprintf(w, `{"message":"%s","time":"%s","ip":"%s"}`,
		response["message"], response["time"], response["ip"])
}

// handleStatus shows current rate limit status
func handleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]interface{}{
		"status": "Rate limiting active",
		"time":   time.Now().Format(time.RFC3339),
		"ip":     getClientIP(r),
	}

	fmt.Fprintf(w, `{"status":"%s","time":"%s","ip":"%s"}`,
		response["status"], response["time"], response["ip"])
}

// handleClear resets rate limit for the client IP
func handleClear(limiter *core.Limiter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		clientIP := getClientIP(r)

		// Clear the rate limit for this IP
		err := limiter.Clear(r.Context(), clientIP)
		if err != nil {
			http.Error(w, "Failed to clear rate limit", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		response := map[string]interface{}{
			"message": "Rate limit cleared successfully",
			"time":    time.Now().Format(time.RFC3339),
			"ip":      clientIP,
		}

		fmt.Fprintf(w, `{"message":"%s","time":"%s","ip":"%s"}`,
			response["message"], response["time"], response["ip"])
	}
}

// getClientIP extracts the client IP from the request
func getClientIP(r *http.Request) string {
	// Check for forwarded headers first
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		return ip
	}
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}

	// Fall back to remote address
	return r.RemoteAddr
}
