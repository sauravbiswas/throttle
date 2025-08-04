# Throttle

A modern, high-performance rate limiting library for Go with a clean, idiomatic, and modular design.

## Features

- **Clean Architecture**: Modular design with clear separation of concerns
- **High Performance**: In-memory backend with proper concurrency safety
- **Pluggable Strategies**: Support for different rate limiting algorithms
- **Metrics Integration**: Optional Prometheus-compatible metrics reporting
- **Per-Key Limiting**: Support for IP addresses, user IDs, or any string key
- **Preview Mode**: Check current state without consuming tokens
- **Thread-Safe**: Proper locking for concurrent access

## Quick Start

```go
package main

import (
    "context"
    "time"
    
    "github.com/throttle/backend/memory"
    "github.com/throttle/core"
    "github.com/throttle/metrics"
    "github.com/throttle/strategy/tokenbucket"
)

func main() {
    // Create components
    backend := memory.NewBackend()
    strategy := tokenbucket.NewStrategy(core.Config{
        Limit:   10,           // 10 requests
        Interval: time.Minute,  // per minute
        Burst:   15,           // with burst capacity of 15
    })
    metrics := metrics.NewNoOpReporter()

    // Create rate limiter
    limiter := core.NewLimiter(backend, strategy, core.Config{
        Limit:   10,
        Interval: time.Minute,
        Burst:   15,
    }, metrics)

    // Use the rate limiter
    ctx := context.Background()
    decision, err := limiter.Grant(ctx, "user-123")
    if err != nil {
        panic(err)
    }

    if decision.Allowed {
        // Process the request
        println("Request allowed, remaining:", decision.Remaining)
    } else {
        // Rate limited
        println("Rate limited, retry after:", decision.RetryAfter)
    }
}
```

## Architecture

### Core Components

#### RateLimiter Interface
```go
type RateLimiter interface {
    Grant(ctx context.Context, key string) (Decision, error)
    Preview(ctx context.Context, key string) (Decision, error)
    Clear(ctx context.Context, key string) error
}
```

#### Backend Interface
```go
type Backend interface {
    Get(ctx context.Context, key string) (*State, error)
    Set(ctx context.Context, key string, state *State) error
    Delete(ctx context.Context, key string) error
    Close() error
}
```

#### Strategy Interface
```go
type Strategy interface {
    Calculate(ctx context.Context, state *State, now time.Time) (Decision, error)
    Preview(ctx context.Context, state *State, now time.Time) (Decision, error)
}
```

### Available Components

#### Backends
- **Memory Backend** (`backend/memory`): High-performance in-memory storage with proper locking

#### Strategies
- **Token Bucket** (`strategy/tokenbucket`): Configurable token bucket algorithm with burst support

#### Metrics
- **NoOp Reporter** (`metrics/noop`): No-op implementation for when metrics aren't needed
- **Generic Reporter** (`metrics/generic`): Generic metrics collection for any monitoring system
- **Generic Reporter** (`metrics/generic`): Generic metrics collection for any monitoring system

## Configuration

### Token Bucket Configuration

```go
config := core.Config{
    Limit:   100,          // Number of tokens per interval
    Interval: time.Minute,  // Time window for the limit
    Burst:   150,          // Maximum burst capacity
}
```

### Metrics Configuration

```go
// For generic metrics (works with any monitoring system)
metrics := metrics.NewGenericReporter()

// For no metrics
metrics := metrics.NewNoOpReporter()

// For generic metrics (works with any monitoring system)
metrics := metrics.NewGenericReporter()
```

### Accessing Metrics

```go
// Get the metrics collector
collector := metrics.GetCollector()

// Get all metrics
allMetrics := collector.Collect()

// Get metrics by name
grantMetrics := collector.GetMetrics("throttle_grant_total")

// Get metrics summary
summary := collector.GetMetricsSummary()
```

## HTTP Server Example

Run the example server:

```bash
go run example/server.go
```

The server provides these endpoints:

- `GET /api/resource` - Make a request (consumes tokens)
- `GET /api/status` - Check current state (no consumption)
- `GET /api/clear` - Reset rate limit for your IP

## Metrics Server

Run the metrics server to expose metrics for monitoring systems:

```bash
go run cmd/metrics-server/main.go
```

The metrics server provides these endpoints:

- `GET /api/resource` - Make a request (generates metrics)
- `GET /metrics` - Prometheus-style metrics
- `GET /metrics/json` - JSON metrics (for Datadog, etc.)
- `GET /health` - Health check

### Example JSON Metrics Output

```json
{
  "timestamp": "2024-01-15T10:30:00Z",
  "metrics": [
    {
      "name": "throttle_grant_total",
      "type": "counter",
      "value": 1.0,
      "labels": {"key": "192.168.1.1"},
      "timestamp": "2024-01-15T10:30:00Z",
      "help": "Total number of grant requests"
    },
    {
      "name": "throttle_remaining_tokens",
      "type": "gauge",
      "value": 99.0,
      "labels": {"key": "192.168.1.1", "decision": "allowed"},
      "timestamp": "2024-01-15T10:30:00Z",
      "help": "Current number of remaining tokens"
    }
  ],
  "summary": {
    "throttle_grant_total": {"counter": 150},
    "throttle_remaining_tokens": {"gauge": 150}
  }
}
```

## Load Testing

The library includes comprehensive load testing tools to verify rate limiting effectiveness:

### Automated Load Test Suite

Run the complete load test suite:

```bash
./scripts/loadtest.sh
```

This runs 6 different scenarios:
1. Basic rate limiting with moderate load
2. High concurrency with higher limits
3. Burst capacity testing
4. Multiple client keys simulation
5. Short time intervals
6. High stress load testing

### Command-Line Load Tester

Run individual load tests with custom parameters:

```bash
go run cmd/loadtest/main.go -duration=30s -concurrency=10 -rate=50 -limit=20 -interval=1m -burst=30
```

Parameters:
- `-duration`: Test duration (e.g., 30s, 2m)
- `-concurrency`: Number of concurrent workers
- `-rate`: Requests per second per worker
- `-keys`: Number of unique keys to test
- `-limit`: Rate limit (requests per interval)
- `-interval`: Rate limit interval (e.g., 1m, 30s)
- `-burst`: Burst capacity
- `-progress`: Show real-time progress (default: true)

### Interactive Load Tester

For real-time testing and experimentation:

```bash
go run cmd/interactive/main.go
```

Interactive commands:
- `test <key>` - Test rate limiting for a specific key
- `burst <key>` - Rapid-fire requests to test burst capacity
- `status <key>` - Check current status without consuming tokens
- `clear <key>` - Reset rate limit for a key
- `load <workers>` - Run a quick load test with N workers
- `quit` - Exit

## Testing

Run all tests:

```bash
go test ./...
```

Run benchmarks:

```bash
go test -bench=. ./...
```

## Performance

The library is designed for high performance:

- **In-memory storage** with O(1) operations
- **Proper locking** for concurrent access
- **Efficient token bucket algorithm** with minimal allocations
- **Optional metrics** that can be disabled for maximum performance

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests for new functionality
5. Run the test suite
6. Submit a pull request

## License

MIT License - see LICENSE file for details. 
