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

type MetricsServer struct {
	limiter  core.RateLimiter
	reporter *metrics.GenericReporter
}

func main() {
	// Create rate limiter with generic metrics
	backend := memory.NewBackend()
	strategy := tokenbucket.NewStrategy(core.Config{
		Limit:    100,
		Interval: time.Minute,
		Burst:    150,
	})
	reporter := metrics.NewGenericReporter()

	limiter := core.NewLimiter(backend, strategy, core.Config{
		Limit:    100,
		Interval: time.Minute,
		Burst:    150,
	}, reporter)

	server := &MetricsServer{
		limiter:  limiter,
		reporter: reporter,
	}

	// Set up HTTP routes
	http.HandleFunc("/api/resource", server.handleResource)
	http.HandleFunc("/metrics", server.handleMetrics)
	http.HandleFunc("/metrics/json", server.handleMetricsJSON)
	http.HandleFunc("/health", server.handleHealth)

	fmt.Println("🚀 Throttle Metrics Server")
	fmt.Println("==========================")
	fmt.Println("Endpoints:")
	fmt.Println("  GET /api/resource    - Make a request (generates metrics)")
	fmt.Println("  GET /metrics         - Prometheus-style metrics")
	fmt.Println("  GET /metrics/json    - JSON metrics (for Datadog, etc.)")
	fmt.Println("  GET /health          - Health check")
	fmt.Println()
	fmt.Println("Starting server on :8080...")

	log.Fatal(http.ListenAndServe(":8080", nil))
}

func (s *MetricsServer) handleResource(w http.ResponseWriter, r *http.Request) {
	// Extract client identifier
	clientIP := r.RemoteAddr
	if forwardedFor := r.Header.Get("X-Forwarded-For"); forwardedFor != "" {
		clientIP = forwardedFor
	}

	// Check rate limit
	ctx := context.Background()
	decision, err := s.limiter.Grant(ctx, clientIP)
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
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte("Rate limit exceeded"))
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Request processed successfully"))
}

func (s *MetricsServer) handleMetrics(w http.ResponseWriter, r *http.Request) {
	collector := s.reporter.GetCollector()
	metrics := collector.Collect()

	w.Header().Set("Content-Type", "text/plain")

	for _, metric := range metrics {
		// Format as Prometheus-style metrics
		labels := ""
		if len(metric.Labels) > 0 {
			var labelPairs []string
			for k, v := range metric.Labels {
				labelPairs = append(labelPairs, fmt.Sprintf(`%s="%s"`, k, v))
			}
			labels = "{" + fmt.Sprintf("%s", labelPairs) + "}"
		}

		fmt.Fprintf(w, "# HELP %s %s\n", metric.Name, metric.Help)
		fmt.Fprintf(w, "# TYPE %s %s\n", metric.Name, metric.Type)
		fmt.Fprintf(w, "%s%s %g\n", metric.Name, labels, metric.Value)
	}
}

func (s *MetricsServer) handleMetricsJSON(w http.ResponseWriter, r *http.Request) {
	collector := s.reporter.GetCollector()
	metrics := collector.Collect()

	w.Header().Set("Content-Type", "application/json")

	response := map[string]interface{}{
		"timestamp": time.Now().Format(time.RFC3339),
		"metrics":   metrics,
		"summary":   collector.GetMetricsSummary(),
	}

	json.NewEncoder(w).Encode(response)
}

func (s *MetricsServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "healthy",
		"time":   time.Now().Format(time.RFC3339),
	})
}
