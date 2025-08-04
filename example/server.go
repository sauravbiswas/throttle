package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/throttle/backend/memory"
	"github.com/throttle/core"
	"github.com/throttle/metrics"
	"github.com/throttle/strategy/tokenbucket"
)

type Response struct {
	Allowed    bool      `json:"allowed"`
	Remaining  int64     `json:"remaining"`
	ResetTime  time.Time `json:"reset_time"`
	RetryAfter string    `json:"retry_after,omitempty"`
	Message    string    `json:"message"`
}

func main() {
	// Create rate limiter components
	backend := memory.NewBackend()
	strategy := tokenbucket.NewStrategy(core.Config{
		Limit:    10,          // 10 requests
		Interval: time.Minute, // per minute
		Burst:    15,          // with burst capacity of 15
	})
	metrics := metrics.NewNoOpReporter() // Use no-op metrics for simplicity

	// Create the rate limiter
	limiter := core.NewLimiter(backend, strategy, core.Config{
		Limit:    10,
		Interval: time.Minute,
		Burst:    15,
	}, metrics)

	// HTTP handler
	http.HandleFunc("/api/resource", func(w http.ResponseWriter, r *http.Request) {
		// Extract client identifier (IP address in this example)
		clientIP := r.RemoteAddr
		if forwardedFor := r.Header.Get("X-Forwarded-For"); forwardedFor != "" {
			clientIP = forwardedFor
		}

		// Check rate limit
		ctx := context.Background()
		decision, err := limiter.Grant(ctx, clientIP)
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		// Set rate limit headers
		w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", decision.Remaining))
		w.Header().Set("X-RateLimit-Reset", decision.ResetTime.Format(time.RFC3339))

		if !decision.Allowed {
			w.Header().Set("X-RateLimit-RetryAfter", decision.RetryAfter.String())
			w.Header().Set("Retry-After", decision.RetryAfter.String())
		}

		// Prepare response
		response := Response{
			Allowed:   decision.Allowed,
			Remaining: decision.Remaining,
			ResetTime: decision.ResetTime,
		}

		if !decision.Allowed {
			response.RetryAfter = decision.RetryAfter.String()
			response.Message = "Rate limit exceeded"
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
		} else {
			response.Message = "Request processed successfully"
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
		}

		// Send response
		json.NewEncoder(w).Encode(response)
	})

	// Preview endpoint to check current state without consuming tokens
	http.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		clientIP := r.RemoteAddr
		if forwardedFor := r.Header.Get("X-Forwarded-For"); forwardedFor != "" {
			clientIP = forwardedFor
		}

		ctx := context.Background()
		decision, err := limiter.Preview(ctx, clientIP)
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		response := Response{
			Allowed:   decision.Allowed,
			Remaining: decision.Remaining,
			ResetTime: decision.ResetTime,
			Message:   "Current rate limit status",
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	})

	// Clear endpoint to reset rate limit for a client
	http.HandleFunc("/api/clear", func(w http.ResponseWriter, r *http.Request) {
		clientIP := r.RemoteAddr
		if forwardedFor := r.Header.Get("X-Forwarded-For"); forwardedFor != "" {
			clientIP = forwardedFor
		}

		ctx := context.Background()
		err := limiter.Clear(ctx, clientIP)
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		response := Response{
			Allowed:   true,
			Remaining: 15, // Reset to burst capacity
			ResetTime: time.Now(),
			Message:   "Rate limit cleared",
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	})

	fmt.Println("Starting rate limiter example server on :8080")
	fmt.Println("Endpoints:")
	fmt.Println("  GET /api/resource - Make a request (consumes tokens)")
	fmt.Println("  GET /api/status   - Check current state (no consumption)")
	fmt.Println("  GET /api/clear    - Reset rate limit for your IP")

	log.Fatal(http.ListenAndServe(":8080", nil))
}
