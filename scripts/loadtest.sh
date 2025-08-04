#!/bin/bash

# Load testing script for Throttle
# This script runs various load test scenarios to demonstrate rate limiting effectiveness

set -e

echo "🚀 GoLimiter Load Testing Suite"
echo "================================"
echo

# Build the load test tool
echo "Building load test tool..."
go build -o bin/loadtest cmd/loadtest/main.go
echo "✅ Build complete"
echo

# Function to run a test scenario
run_scenario() {
    local name="$1"
    local args="$2"
    
    echo "🧪 Running: $name"
    echo "Command: ./bin/loadtest $args"
    echo "---"
    ./bin/loadtest $args
    echo "---"
    echo
}

# Scenario 1: Basic rate limiting test
echo "📊 Scenario 1: Basic Rate Limiting"
run_scenario "Basic Test" "-duration=30s -concurrency=5 -rate=20 -limit=10 -interval=1m -burst=15"

# Scenario 2: High concurrency test
echo "📊 Scenario 2: High Concurrency"
run_scenario "High Concurrency" "-duration=20s -concurrency=20 -rate=50 -limit=100 -interval=1m -burst=150"

# Scenario 3: Burst test
echo "📊 Scenario 3: Burst Capacity Test"
run_scenario "Burst Test" "-duration=10s -concurrency=10 -rate=100 -limit=5 -interval=1m -burst=20"

# Scenario 4: Multiple keys test
echo "📊 Scenario 4: Multiple Keys Test"
run_scenario "Multiple Keys" "-duration=15s -concurrency=8 -rate=30 -keys=10 -limit=20 -interval=1m -burst=25"

# Scenario 5: Short interval test
echo "📊 Scenario 5: Short Interval Test"
run_scenario "Short Interval" "-duration=20s -concurrency=5 -rate=40 -limit=5 -interval=10s -burst=10"

# Scenario 6: Stress test
echo "📊 Scenario 6: Stress Test"
run_scenario "Stress Test" "-duration=30s -concurrency=50 -rate=200 -limit=50 -interval=1m -burst=100"

echo "🎉 All load tests completed!"
echo
echo "Summary of scenarios tested:"
echo "1. Basic rate limiting with moderate load"
echo "2. High concurrency with higher limits"
echo "3. Burst capacity testing"
echo "4. Multiple client keys simulation"
echo "5. Short time intervals"
echo "6. High stress load testing"
echo
echo "Check the results above to see how effectively the rate limiter"
echo "controls traffic under different conditions." 