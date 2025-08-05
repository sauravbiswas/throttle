package leakybucket

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/throttle/core"
)

func TestStrategy_Calculate_InitialState(t *testing.T) {
	config := core.Config{
		Limit:    10,
		Interval: time.Minute,
		Burst:    15,
	}
	strategy := NewStrategy(config)
	ctx := context.Background()

	// Test with initial state (empty bucket)
	now := time.Now()
	state := &core.State{
		Tokens:     0.0, // Start with empty bucket
		LastUpdate: now,
		Created:    now,
	}

	decision, err := strategy.Calculate(ctx, state, now)
	assert.NoError(t, err)
	assert.True(t, decision.Allowed)
	assert.Equal(t, int64(14), decision.Remaining) // 15 burst - 1 consumed = 14
}

func TestStrategy_Calculate_FullBucket(t *testing.T) {
	config := core.Config{
		Limit:    10,
		Interval: time.Minute,
		Burst:    5, // Small burst for testing
	}
	strategy := NewStrategy(config)
	ctx := context.Background()

	// Test with full bucket
	now := time.Now()
	state := &core.State{
		Tokens:     5.0, // Full bucket
		LastUpdate: now,
		Created:    now,
	}

	decision, err := strategy.Calculate(ctx, state, now)
	assert.NoError(t, err)
	assert.False(t, decision.Allowed)
	assert.Equal(t, int64(0), decision.Remaining)
	assert.Greater(t, decision.RetryAfter, time.Duration(0))
}

func TestStrategy_Calculate_LeakOverTime(t *testing.T) {
	config := core.Config{
		Limit:    60, // 60 requests per minute = 1 per second
		Interval: time.Minute,
		Burst:    10,
	}
	strategy := NewStrategy(config)
	ctx := context.Background()

	// Start with a full bucket
	baseTime := time.Now()
	state := &core.State{
		Tokens:     10.0, // Full bucket
		LastUpdate: baseTime,
		Created:    baseTime,
	}

	// After 5 seconds, 5 tokens should have leaked out
	now := baseTime.Add(5 * time.Second)
	decision, err := strategy.Calculate(ctx, state, now)
	assert.NoError(t, err)
	assert.True(t, decision.Allowed)              // Should be able to add one more
	assert.Equal(t, int64(4), decision.Remaining) // 10 - 5 leaked - 1 consumed = 4
}

func TestStrategy_Calculate_ContinuousLeak(t *testing.T) {
	config := core.Config{
		Limit:    120, // 120 requests per minute = 2 per second
		Interval: time.Minute,
		Burst:    10,
	}
	strategy := NewStrategy(config)
	ctx := context.Background()

	// Start with a full bucket
	baseTime := time.Now()
	state := &core.State{
		Tokens:     10.0, // Full bucket
		LastUpdate: baseTime,
		Created:    baseTime,
	}

	// Make multiple requests over time
	for i := 0; i < 5; i++ {
		now := baseTime.Add(time.Duration(i) * time.Second)
		decision, err := strategy.Calculate(ctx, state, now)
		assert.NoError(t, err)

		// In leaky bucket with full bucket:
		// - First request (t=0s): denied (bucket full, adding 1 would exceed burst)
		// - Later requests: allowed (tokens have leaked out)
		if i == 0 {
			assert.False(t, decision.Allowed)
		} else {
			assert.True(t, decision.Allowed)
		}
	}
}

func TestStrategy_Preview_NoStateChange(t *testing.T) {
	config := core.Config{
		Limit:    10,
		Interval: time.Minute,
		Burst:    15,
	}
	strategy := NewStrategy(config)
	ctx := context.Background()

	now := time.Now()
	state := &core.State{
		Tokens:     5.0,
		LastUpdate: now,
		Created:    now,
	}

	// Store original state
	originalTokens := state.Tokens
	originalLastUpdate := state.LastUpdate

	decision, err := strategy.Preview(ctx, state, now)
	assert.NoError(t, err)

	// Verify state wasn't modified
	assert.Equal(t, originalTokens, state.Tokens)
	assert.Equal(t, originalLastUpdate, state.LastUpdate)
	assert.True(t, decision.Allowed)
	assert.Equal(t, int64(10), decision.Remaining) // 15 - 5 = 10 (no consumption in preview)
}

func TestStrategy_Calculate_ResetTime(t *testing.T) {
	config := core.Config{
		Limit:    60, // 1 request per second
		Interval: time.Minute,
		Burst:    10,
	}
	strategy := NewStrategy(config)
	ctx := context.Background()

	now := time.Now()
	state := &core.State{
		Tokens:     5.0, // Half full bucket
		LastUpdate: now,
		Created:    now,
	}

	decision, err := strategy.Calculate(ctx, state, now)
	assert.NoError(t, err)
	assert.True(t, decision.Allowed)

	// Reset time should be approximately 5 seconds (5 tokens at 1 per second)
	expectedResetTime := now.Add(5 * time.Second)
	assert.WithinDuration(t, expectedResetTime, decision.ResetTime, 100*time.Millisecond)
}

func TestStrategy_Calculate_EdgeCases(t *testing.T) {
	config := core.Config{
		Limit:    10,
		Interval: time.Minute,
		Burst:    5,
	}
	strategy := NewStrategy(config)
	ctx := context.Background()

	// Test with negative tokens (should be clamped to 0)
	now := time.Now()
	state := &core.State{
		Tokens:     -1.0, // Invalid state
		LastUpdate: now,
		Created:    now,
	}

	decision, err := strategy.Calculate(ctx, state, now)
	assert.NoError(t, err)
	assert.True(t, decision.Allowed)              // Should be able to add to empty bucket
	assert.Equal(t, int64(4), decision.Remaining) // 5 - 0 - 1 = 4
}

func TestStrategy_Calculate_ZeroBurst(t *testing.T) {
	config := core.Config{
		Limit:    10,
		Interval: time.Minute,
		Burst:    0, // No burst capacity
	}
	strategy := NewStrategy(config)
	ctx := context.Background()

	now := time.Now()
	state := &core.State{
		Tokens:     0.0,
		LastUpdate: now,
		Created:    now,
	}

	decision, err := strategy.Calculate(ctx, state, now)
	assert.NoError(t, err)
	assert.False(t, decision.Allowed) // No burst capacity
	assert.Equal(t, int64(0), decision.Remaining)
}

func BenchmarkStrategy_Calculate(b *testing.B) {
	config := core.Config{
		Limit:    1000,
		Interval: time.Minute,
		Burst:    1500,
	}
	strategy := NewStrategy(config)
	ctx := context.Background()

	now := time.Now()
	state := &core.State{
		Tokens:     100.0,
		LastUpdate: now,
		Created:    now,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := strategy.Calculate(ctx, state, now)
		assert.NoError(b, err)
	}
}
