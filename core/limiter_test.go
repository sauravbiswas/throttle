package core

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// MockBackend implements Backend for testing
type MockBackend struct {
	store map[string]*State
}

func NewMockBackend() *MockBackend {
	return &MockBackend{
		store: make(map[string]*State),
	}
}

func (m *MockBackend) Get(ctx context.Context, key string) (*State, error) {
	if state, exists := m.store[key]; exists {
		return &State{
			Tokens:     state.Tokens,
			LastUpdate: state.LastUpdate,
			Created:    state.Created,
		}, nil
	}
	return nil, nil
}

func (m *MockBackend) Set(ctx context.Context, key string, state *State) error {
	m.store[key] = &State{
		Tokens:     state.Tokens,
		LastUpdate: state.LastUpdate,
		Created:    state.Created,
	}
	return nil
}

func (m *MockBackend) Delete(ctx context.Context, key string) error {
	delete(m.store, key)
	return nil
}

func (m *MockBackend) Close() error {
	return nil
}

// MockStrategy implements Strategy for testing
type MockStrategy struct {
	shouldAllow bool
	remaining   int64
	resetTime   time.Time
	retryAfter  time.Duration
}

func NewMockStrategy(shouldAllow bool, remaining int64) *MockStrategy {
	return &MockStrategy{
		shouldAllow: shouldAllow,
		remaining:   remaining,
		resetTime:   time.Now().Add(time.Minute),
		retryAfter:  time.Second * 30,
	}
}

func (m *MockStrategy) Calculate(ctx context.Context, state *State, now time.Time) (Decision, error) {
	return Decision{
		Allowed:    m.shouldAllow,
		Remaining:  m.remaining,
		ResetTime:  m.resetTime,
		RetryAfter: m.retryAfter,
	}, nil
}

func (m *MockStrategy) Preview(ctx context.Context, state *State, now time.Time) (Decision, error) {
	return Decision{
		Allowed:    m.shouldAllow,
		Remaining:  m.remaining,
		ResetTime:  m.resetTime,
		RetryAfter: m.retryAfter,
	}, nil
}

// MockMetricsReporter implements MetricsReporter for testing
type MockMetricsReporter struct {
	grantCalls   int
	previewCalls int
	clearCalls   int
}

func NewMockMetricsReporter() *MockMetricsReporter {
	return &MockMetricsReporter{}
}

func (m *MockMetricsReporter) RecordGrant(key string, allowed bool, remaining int64) {
	m.grantCalls++
}

func (m *MockMetricsReporter) RecordPreview(key string, remaining int64) {
	m.previewCalls++
}

func (m *MockMetricsReporter) RecordClear(key string) {
	m.clearCalls++
}

func TestLimiter_Grant(t *testing.T) {
	backend := NewMockBackend()
	strategy := NewMockStrategy(true, 5)
	metrics := NewMockMetricsReporter()
	config := Config{Limit: 10, Interval: time.Minute, Burst: 15}

	limiter := NewLimiter(backend, strategy, config, metrics)

	ctx := context.Background()
	decision, err := limiter.Grant(ctx, "test-key")

	assert.NoError(t, err)
	assert.True(t, decision.Allowed)
	assert.Equal(t, int64(5), decision.Remaining)
	assert.Equal(t, 1, metrics.grantCalls)
}

func TestLimiter_Preview(t *testing.T) {
	backend := NewMockBackend()
	strategy := NewMockStrategy(false, 0)
	metrics := NewMockMetricsReporter()
	config := Config{Limit: 10, Interval: time.Minute, Burst: 15}

	limiter := NewLimiter(backend, strategy, config, metrics)

	ctx := context.Background()
	decision, err := limiter.Preview(ctx, "test-key")

	assert.NoError(t, err)
	assert.False(t, decision.Allowed)
	assert.Equal(t, int64(0), decision.Remaining)
	assert.Equal(t, 1, metrics.previewCalls)
}

func TestLimiter_Clear(t *testing.T) {
	backend := NewMockBackend()
	strategy := NewMockStrategy(true, 5)
	metrics := NewMockMetricsReporter()
	config := Config{Limit: 10, Interval: time.Minute, Burst: 15}

	limiter := NewLimiter(backend, strategy, config, metrics)

	ctx := context.Background()
	err := limiter.Clear(ctx, "test-key")

	assert.NoError(t, err)
	assert.Equal(t, 1, metrics.clearCalls)
}
