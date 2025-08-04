#!/bin/bash

# Throttle Demo Script
# This script demonstrates the rate limiter's effectiveness with visual examples

set -e

echo "🎬 Throttle Demo"
echo "================"
echo

# Build tools
echo "Building demo tools..."
go build -o bin/loadtest cmd/loadtest/main.go
go build -o bin/interactive cmd/interactive/main.go
echo "✅ Build complete"
echo

echo "📊 Demo 1: Basic Rate Limiting"
echo "-------------------------------"
echo "Testing: 5 requests per minute limit with burst capacity of 8"
echo "Sending 20 rapid requests to see rate limiting in action..."
echo

./bin/loadtest -duration=5s -concurrency=1 -rate=4 -limit=5 -interval=1m -burst=8 -keys=1

echo
echo "📊 Demo 2: Burst Capacity Test"
echo "-------------------------------"
echo "Testing: 3 requests per minute limit with burst capacity of 10"
echo "This should allow initial burst, then start limiting..."
echo

./bin/loadtest -duration=8s -concurrency=1 -rate=2 -limit=3 -interval=1m -burst=10 -keys=1

echo
echo "📊 Demo 3: Multiple Clients"
echo "---------------------------"
echo "Testing: 10 requests per minute limit across 5 different clients"
echo "Each client should be limited independently..."
echo

./bin/loadtest -duration=10s -concurrency=5 -rate=3 -limit=10 -interval=1m -burst=15 -keys=5

echo
echo "📊 Demo 4: High Concurrency Stress Test"
echo "----------------------------------------"
echo "Testing: 50 requests per minute limit with 20 concurrent workers"
echo "This demonstrates the rate limiter under high load..."
echo

./bin/loadtest -duration=15s -concurrency=20 -rate=10 -limit=50 -interval=1m -burst=75 -keys=10

echo
echo "🎉 Demo Complete!"
echo
echo "Key Observations:"
echo "1. Rate limiting effectively controls traffic flow"
echo "2. Burst capacity allows initial spikes"
echo "3. Multiple clients are limited independently"
echo "4. High concurrency is handled safely"
echo "5. Latency remains low even under load"
echo
echo "Try the interactive tester for hands-on experimentation:"
echo "  ./bin/interactive"
echo
echo "Or run the full load test suite:"
echo "  ./scripts/loadtest.sh" 