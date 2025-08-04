package tokenbucket

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

	// Test with initial state (no existing state)
	now := time.Now()
	state := &core.State{
		Tokens:     15.0, // Start with burst capacity
		LastUpdate: now,
		Created:    now,
	}

	decision, err := strategy.Calculate(ctx, state, now)
	assert.NoError(t, err)
	assert.True(t, decision.Allowed)
	assert.Equal(t, int64(14), decision.Remaining)
}

func TestStrategy_Calculate_NoTokens(t *testing.T) {
	config := core.Config{
		Limit:    10,
		Interval: time.Minute,
		Burst:    15,
	}
	strategy := NewStrategy(config)
	ctx := context.Background()

	// Test with no tokens available
	now := time.Now()
	state := &core.State{
		Tokens:     0.0,
		LastUpdate: now,
		Created:    now,
	}

	decision, err := strategy.Calculate(ctx, state, now)
	assert.NoError(t, err)
	assert.False(t, decision.Allowed)
	assert.Equal(t, int64(0), decision.Remaining)
	assert.Greater(t, decision.RetryAfter, time.Duration(0))
}

func TestStrategy_Calculate_TokenRefill(t *testing.T) {
	config := core.Config{
		Limit:    10,
		Interval: time.Minute,
		Burst:    15,
	}
	strategy := NewStrategy(config)
	ctx := context.Background()

	// Test token refill over time
	baseTime := time.Now()
	state := &core.State{
		Tokens:     0.0,
		LastUpdate: baseTime,
		Created:    baseTime,
	}

	// Advance time by 30 seconds (should add 5 tokens: 30s/60s * 10)
	now := baseTime.Add(30 * time.Second)
	decision, err := strategy.Calculate(ctx, state, now)
	assert.NoError(t, err)
	assert.True(t, decision.Allowed)
	assert.Equal(t, int64(4), decision.Remaining) // 5 tokens - 1 consumed = 4 remaining
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
	assert.Equal(t, int64(5), decision.Remaining)
}

func TestStrategy_BurstLimit(t *testing.T) {
	config := core.Config{
		Limit:    10,
		Interval: time.Minute,
		Burst:    5, // Small burst limit
	}
	strategy := NewStrategy(config)
	ctx := context.Background()

	// Test that tokens don't exceed burst limit
	baseTime := time.Now()
	state := &core.State{
		Tokens:     0.0,
		LastUpdate: baseTime,
		Created:    baseTime,
	}

	// Advance time by 2 minutes (should add 20 tokens, but burst is 5)
	now := baseTime.Add(2 * time.Minute)
	decision, err := strategy.Calculate(ctx, state, now)
	assert.NoError(t, err)
	assert.True(t, decision.Allowed)
	assert.Equal(t, int64(4), decision.Remaining) // 5 burst - 1 consumed = 4 remaining
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
