package core

import (
	"context"
	"time"
)

// Decision represents the result of a rate limiting decision
type Decision struct {
	Allowed    bool          // Whether the request is allowed
	Remaining  int64         // Remaining tokens/requests
	ResetTime  time.Time     // When the limit will reset
	RetryAfter time.Duration // How long to wait before retrying (if not allowed)
}

// RateLimiter defines the main interface for rate limiting operations
type RateLimiter interface {
	// Grant determines whether a request should be allowed now
	Grant(ctx context.Context, key string) (Decision, error)

	// Preview returns the current usage state without modifying anything
	Preview(ctx context.Context, key string) (Decision, error)

	// Clear resets internal counters for the key
	Clear(ctx context.Context, key string) error
}

// Backend defines the storage interface for rate limiting data
type Backend interface {
	// Get retrieves the current state for a key
	Get(ctx context.Context, key string) (*State, error)

	// Set stores the state for a key
	Set(ctx context.Context, key string, state *State) error

	// Delete removes the state for a key
	Delete(ctx context.Context, key string) error

	// Close performs any necessary cleanup
	Close() error
}

// State represents the internal state of a rate limiter for a key
type State struct {
	Tokens     float64   // Current number of tokens
	LastUpdate time.Time // Last time the state was updated
	Created    time.Time // When this state was first created
}

// Strategy defines the rate limiting algorithm interface
type Strategy interface {
	// Calculate determines if a request should be allowed and updates state
	Calculate(ctx context.Context, state *State, now time.Time) (Decision, error)

	// Preview calculates the decision without modifying state
	Preview(ctx context.Context, state *State, now time.Time) (Decision, error)
}

// Config holds configuration for rate limiting strategies
type Config struct {
	Limit    int64         // Maximum number of requests/tokens
	Interval time.Duration // Time window for the limit
	Burst    int64         // Maximum burst capacity (for token bucket)
}

// MetricsReporter defines the interface for reporting metrics
type MetricsReporter interface {
	// RecordGrant records a grant decision
	RecordGrant(key string, allowed bool, remaining int64)

	// RecordPreview records a preview operation
	RecordPreview(key string, remaining int64)

	// RecordClear records a clear operation
	RecordClear(key string)
}
