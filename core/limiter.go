package core

import (
	"context"
	"sync"
	"time"
)

// Limiter implements the RateLimiter interface
type Limiter struct {
	backend  Backend
	strategy Strategy
	config   Config
	metrics  MetricsReporter
	mu       sync.RWMutex
}

// NewLimiter creates a new rate limiter with the given components
func NewLimiter(backend Backend, strategy Strategy, config Config, metrics MetricsReporter) *Limiter {
	return &Limiter{
		backend:  backend,
		strategy: strategy,
		config:   config,
		metrics:  metrics,
	}
}

// Grant determines whether a request should be allowed now
func (l *Limiter) Grant(ctx context.Context, key string) (Decision, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Get current state
	state, err := l.backend.Get(ctx, key)
	if err != nil {
		return Decision{}, err
	}

	// If no state exists, create a new one
	if state == nil {
		now := time.Now()
		state = &State{
			Tokens:     float64(l.config.Burst),
			LastUpdate: now,
			Created:    now,
		}
	}

	// Calculate decision
	now := time.Now()
	decision, err := l.strategy.Calculate(ctx, state, now)
	if err != nil {
		return Decision{}, err
	}

	// Update state in backend
	if err := l.backend.Set(ctx, key, state); err != nil {
		return Decision{}, err
	}

	// Record metrics if available
	if l.metrics != nil {
		l.metrics.RecordGrant(key, decision.Allowed, decision.Remaining)
	}

	return decision, nil
}

// Preview returns the current usage state without modifying anything
func (l *Limiter) Preview(ctx context.Context, key string) (Decision, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	// Get current state
	state, err := l.backend.Get(ctx, key)
	if err != nil {
		return Decision{}, err
	}

	// If no state exists, return default state
	if state == nil {
		now := time.Now()
		state = &State{
			Tokens:     float64(l.config.Burst),
			LastUpdate: now,
			Created:    now,
		}
	}

	// Calculate preview decision
	now := time.Now()
	decision, err := l.strategy.Preview(ctx, state, now)
	if err != nil {
		return Decision{}, err
	}

	// Record metrics if available
	if l.metrics != nil {
		l.metrics.RecordPreview(key, decision.Remaining)
	}

	return decision, nil
}

// Clear resets internal counters for the key
func (l *Limiter) Clear(ctx context.Context, key string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	err := l.backend.Delete(ctx, key)
	if err != nil {
		return err
	}

	// Record metrics if available
	if l.metrics != nil {
		l.metrics.RecordClear(key)
	}

	return nil
}

// Config returns the current configuration
func (l *Limiter) Config() Config {
	return l.config
}
