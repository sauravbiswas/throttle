package leakybucket

import (
	"context"
	"time"

	"github.com/throttle/core"
)

// Strategy implements the leaky bucket rate limiting algorithm
type Strategy struct {
	config core.Config
}

// NewStrategy creates a new leaky bucket strategy
func NewStrategy(config core.Config) *Strategy {
	return &Strategy{
		config: config,
	}
}

// Calculate determines if a request should be allowed and updates state
func (s *Strategy) Calculate(ctx context.Context, state *core.State, now time.Time) (core.Decision, error) {
	// Calculate time elapsed since last update
	elapsed := now.Sub(state.LastUpdate)

	// Calculate how much water has leaked out
	leakRate := float64(s.config.Limit) / float64(s.config.Interval) // tokens per nanosecond
	leakedTokens := leakRate * float64(elapsed)

	// Update the bucket level (water level)
	newLevel := state.Tokens - leakedTokens
	if newLevel < 0 {
		newLevel = 0
	}

	// Check if we can add one more drop (request)
	// In leaky bucket, we can only add if adding one more won't exceed burst
	allowed := (newLevel + 1.0) <= float64(s.config.Burst)

	var remaining int64
	var retryAfter time.Duration

	if allowed {
		// Add the request to the bucket
		remaining = int64(float64(s.config.Burst) - (newLevel + 1.0))
		state.Tokens = newLevel + 1.0
	} else {
		// Calculate when the bucket will have space for another request
		spaceNeeded := 1.0
		timeToLeak := time.Duration(spaceNeeded / leakRate)
		retryAfter = timeToLeak
		remaining = int64(float64(s.config.Burst) - newLevel)
	}

	// Update the last update time
	state.LastUpdate = now

	// Calculate reset time (when bucket will be empty)
	resetTime := now.Add(time.Duration(newLevel / leakRate))

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

	// Calculate how much water has leaked out
	leakRate := float64(s.config.Limit) / float64(s.config.Interval) // tokens per nanosecond
	leakedTokens := leakRate * float64(elapsed)

	// Update the bucket level (water level)
	newLevel := state.Tokens - leakedTokens
	if newLevel < 0 {
		newLevel = 0
	}

	// Check if we can add one more drop (request)
	// In leaky bucket, we can only add if adding one more won't exceed burst
	allowed := (newLevel + 1.0) <= float64(s.config.Burst)

	var remaining int64
	var retryAfter time.Duration

	if !allowed {
		// Calculate when the bucket will have space for another request
		spaceNeeded := 1.0
		timeToLeak := time.Duration(spaceNeeded / leakRate)
		retryAfter = timeToLeak
	}

	remaining = int64(float64(s.config.Burst) - newLevel)

	// Calculate reset time (when bucket will be empty)
	resetTime := now.Add(time.Duration(newLevel / leakRate))

	return core.Decision{
		Allowed:    allowed,
		Remaining:  remaining,
		ResetTime:  resetTime,
		RetryAfter: retryAfter,
	}, nil
}
