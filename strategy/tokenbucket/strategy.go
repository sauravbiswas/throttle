package tokenbucket

import (
	"context"
	"math"
	"time"

	"github.com/throttle/core"
)

// Strategy implements the token bucket rate limiting algorithm
type Strategy struct {
	config core.Config
}

// NewStrategy creates a new token bucket strategy
func NewStrategy(config core.Config) *Strategy {
	return &Strategy{
		config: config,
	}
}

// Calculate determines if a request should be allowed and updates state
func (s *Strategy) Calculate(ctx context.Context, state *core.State, now time.Time) (core.Decision, error) {
	// Calculate time elapsed since last update
	elapsed := now.Sub(state.LastUpdate)

	// Calculate tokens to add based on elapsed time
	tokensToAdd := float64(elapsed) / float64(s.config.Interval) * float64(s.config.Limit)

	// Add tokens to bucket, but don't exceed burst capacity
	newTokens := math.Min(state.Tokens+tokensToAdd, float64(s.config.Burst))

	// Check if we have enough tokens for this request
	allowed := newTokens >= 1.0

	var remaining int64
	var retryAfter time.Duration

	if allowed {
		// Consume one token
		remaining = int64(newTokens - 1.0)
		state.Tokens = newTokens - 1.0
	} else {
		// Calculate when the next token will be available
		tokensNeeded := 1.0 - newTokens
		timeNeeded := time.Duration(tokensNeeded / float64(s.config.Limit) * float64(s.config.Interval))
		retryAfter = timeNeeded
		remaining = int64(newTokens)
	}

	// Update the last update time
	state.LastUpdate = now

	// Calculate reset time
	resetTime := now.Add(time.Duration((float64(s.config.Burst) - newTokens) / float64(s.config.Limit) * float64(s.config.Interval)))

	return core.Decision{
		Allowed:    allowed,
		Remaining:  remaining,
		ResetTime:  resetTime,
		RetryAfter: retryAfter,
	}, nil
}

// Preview calculates the decision without modifying state
func (s *Strategy) Preview(ctx context.Context, state *core.State, now time.Time) (core.Decision, error) {
	// Calculate time elapsed since last update
	elapsed := now.Sub(state.LastUpdate)

	// Calculate tokens to add based on elapsed time
	tokensToAdd := float64(elapsed) / float64(s.config.Interval) * float64(s.config.Limit)

	// Add tokens to bucket, but don't exceed burst capacity
	newTokens := math.Min(state.Tokens+tokensToAdd, float64(s.config.Burst))

	// Check if we have enough tokens for this request
	allowed := newTokens >= 1.0

	var remaining int64
	var retryAfter time.Duration

	if !allowed {
		// Calculate when the next token will be available
		tokensNeeded := 1.0 - newTokens
		timeNeeded := time.Duration(tokensNeeded / float64(s.config.Limit) * float64(s.config.Interval))
		retryAfter = timeNeeded
	}

	remaining = int64(newTokens)

	// Calculate reset time
	resetTime := now.Add(time.Duration((float64(s.config.Burst) - newTokens) / float64(s.config.Limit) * float64(s.config.Interval)))

	return core.Decision{
		Allowed:    allowed,
		Remaining:  remaining,
		ResetTime:  resetTime,
		RetryAfter: retryAfter,
	}, nil
}
