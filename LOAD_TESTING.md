# Throttle Load Testing Guide

This document provides a comprehensive guide to testing and validating the GoLimiter rate limiting library under various load conditions.

## Overview

The Throttle library includes several load testing tools designed to:

1. **Validate Rate Limiting Effectiveness** - Ensure the rate limiter correctly enforces limits
2. **Measure Performance** - Benchmark latency and throughput under load
3. **Test Concurrency Safety** - Verify thread-safe operation
4. **Demonstrate Real-World Usage** - Show how the library performs in realistic scenarios

## Available Tools

### 1. Command-Line Load Tester (`cmd/loadtest/main.go`)

A comprehensive load testing tool with configurable parameters.

**Usage:**
```bash
go run cmd/loadtest/main.go [flags]
```

**Parameters:**
- `-duration`: Test duration (e.g., 30s, 2m, 1h)
- `-concurrency`: Number of concurrent workers
- `-rate`: Requests per second per worker
- `-keys`: Number of unique keys to test
- `-limit`: Rate limit (requests per interval)
- `-interval`: Rate limit interval (e.g., 1m, 30s, 1h)
- `-burst`: Burst capacity
- `-progress`: Show real-time progress (default: true)

**Example:**
```bash
# Test 10 requests per minute limit with 5 concurrent workers
go run cmd/loadtest/main.go -duration=30s -concurrency=5 -rate=20 -limit=10 -interval=1m -burst=15
```

### 2. Interactive Load Tester (`cmd/interactive/main.go`)

A real-time interactive tool for hands-on testing and experimentation.

**Usage:**
```bash
go run cmd/interactive/main.go
```

**Commands:**
- `test <key>` - Test rate limiting for a specific key
- `burst <key>` - Rapid-fire requests to test burst capacity
- `status <key>` - Check current status without consuming tokens
- `clear <key>` - Reset rate limit for a key
- `load <workers>` - Run a quick load test with N workers
- `quit` - Exit

### 3. Benchmark Tool (`cmd/benchmark/main.go`)

A performance benchmarking tool that measures throughput and latency.

**Usage:**
```bash
go run cmd/benchmark/main.go [flags]
```

**Parameters:**
- `-duration`: Benchmark duration
- `-concurrency`: Number of concurrent workers
- `-keys`: Number of unique keys
- `-limit`: Rate limit
- `-interval`: Rate limit interval
- `-burst`: Burst capacity

**Benchmarks:**
1. **Single Key, High Frequency** - Tests performance with concentrated load
2. **Multiple Keys, Distributed Load** - Tests performance with distributed load
3. **Mixed Workload** - Tests both Grant and Preview operations

### 4. Automated Test Suite (`scripts/loadtest.sh`)

A script that runs 6 predefined test scenarios.

**Usage:**
```bash
./scripts/loadtest.sh
```

**Scenarios:**
1. Basic rate limiting with moderate load
2. High concurrency with higher limits
3. Burst capacity testing
4. Multiple client keys simulation
5. Short time intervals
6. High stress load testing

### 5. Demo Script (`scripts/demo.sh`)

A demonstration script showing the rate limiter in action.

**Usage:**
```bash
./scripts/demo.sh
```

## Test Scenarios

### Scenario 1: Basic Rate Limiting
```bash
go run cmd/loadtest/main.go -duration=30s -concurrency=5 -rate=20 -limit=10 -interval=1m -burst=15
```
**Purpose:** Test basic rate limiting functionality with moderate load.

**Expected Results:**
- Most requests should be denied after initial burst
- Allowed requests should be close to the rate limit
- Low latency (microseconds)

### Scenario 2: Burst Capacity Testing
```bash
go run cmd/loadtest/main.go -duration=10s -concurrency=10 -rate=100 -limit=5 -interval=1m -burst=20
```
**Purpose:** Test burst capacity and token refill behavior.

**Expected Results:**
- Initial burst should allow up to 20 requests
- Subsequent requests should be limited to 5 per minute
- Token refill should be visible over time

### Scenario 3: High Concurrency
```bash
go run cmd/loadtest/main.go -duration=20s -concurrency=20 -rate=50 -limit=100 -interval=1m -burst=150
```
**Purpose:** Test performance under high concurrency.

**Expected Results:**
- High throughput (thousands of req/sec)
- Consistent rate limiting across all workers
- Thread-safe operation

### Scenario 4: Multiple Keys
```bash
go run cmd/loadtest/main.go -duration=15s -concurrency=8 -rate=30 -keys=10 -limit=20 -interval=1m -burst=25
```
**Purpose:** Test independent rate limiting per key.

**Expected Results:**
- Each key should be limited independently
- Total allowed requests should be approximately keys × limit
- Fair distribution across keys

## Performance Expectations

### Latency
- **Average Latency:** < 10 microseconds
- **P95 Latency:** < 50 microseconds
- **P99 Latency:** < 100 microseconds

### Throughput
- **Single Key:** > 1,000,000 req/sec
- **Multiple Keys:** > 500,000 req/sec
- **Mixed Workload:** > 200,000 req/sec

### Rate Limiting Accuracy
- **Effectiveness:** > 90% of requests should be properly limited
- **Burst Handling:** Should respect burst capacity
- **Token Refill:** Should refill tokens at correct rate

## Interpreting Results

### Key Metrics

1. **Allowed vs Denied Requests**
   - High denial rate indicates effective rate limiting
   - Allowed requests should match rate limit configuration

2. **Latency Statistics**
   - Low latency indicates good performance
   - Consistent latency shows stable operation

3. **Throughput**
   - High throughput shows efficient implementation
   - Throughput should scale with concurrency

4. **Rate Limiting Effectiveness**
   - Percentage of denied requests should be high under load
   - Should match theoretical expectations

### Example Output Analysis

```
=== Load Test Results ===
Total Requests: 992
Allowed Requests: 75 (7.56%)
Denied Requests: 917 (92.44%)
Actual Rate: 99.20 req/sec

Latency Statistics:
  Average: 12.676µs
  Min: 316ns
  Max: 2.418423ms

Rate Limiting Analysis:
  Expected allowed (theoretical): ~15
  Actual allowed: 75
  Effectiveness: 92.44%
```

**Analysis:**
- 92.44% effectiveness is excellent
- Average latency of 12.676µs is very good
- Actual allowed (75) vs expected (15) shows burst capacity working
- 99.20 req/sec actual rate shows high throughput

## Troubleshooting

### Common Issues

1. **High Latency**
   - Check system resources (CPU, memory)
   - Reduce concurrency if needed
   - Verify no other processes consuming resources

2. **Low Effectiveness**
   - Verify rate limit configuration
   - Check if burst capacity is too high
   - Ensure proper key distribution

3. **Inconsistent Results**
   - Run tests multiple times
   - Check for system load variations
   - Verify test parameters

### Debug Mode

For detailed debugging, use the interactive tester:
```bash
go run cmd/interactive/main.go
```

This allows step-by-step testing and inspection of rate limiter state.

## Best Practices

1. **Start Small** - Begin with low concurrency and increase gradually
2. **Test Multiple Scenarios** - Use different configurations to validate behavior
3. **Monitor System Resources** - Ensure tests don't overwhelm the system
4. **Run Multiple Iterations** - Average results over multiple runs
5. **Document Results** - Keep records of performance baselines

## Conclusion

The Throttle load testing tools provide comprehensive validation of rate limiting functionality and performance. Use these tools to:

- Validate rate limiting effectiveness
- Measure performance characteristics
- Test edge cases and error conditions
- Establish performance baselines
- Demonstrate library capabilities

For production use, consider running these tests regularly to ensure consistent performance and behavior. 