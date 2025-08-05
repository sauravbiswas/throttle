package redis

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/throttle/core"
)

// setupTestRedis creates a test Redis client
func setupTestRedis(t *testing.T) *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       15, // Use DB 15 for testing to avoid conflicts
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		t.Skipf("Redis not available: %v", err)
	}

	// Clean up test database
	client.FlushDB(ctx)

	return client
}

// setupBenchmarkRedis creates a Redis client for benchmarks
func setupBenchmarkRedis() *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       15,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil // Benchmarks will skip if Redis is not available
	}

	// Clean up test database
	client.FlushDB(ctx)

	return client
}

func TestNewBackend(t *testing.T) {
	client := setupTestRedis(t)
	defer client.Close()

	backend := NewBackend(client, "test-prefix")
	assert.NotNil(t, backend)
	assert.Equal(t, "test-prefix", backend.prefix)
}

func TestNewBackendFromURL(t *testing.T) {
	// Test with valid URL
	backend, err := NewBackendFromURL("redis://localhost:6379/15", "test-prefix")
	if err != nil {
		t.Skipf("Redis not available: %v", err)
	}
	defer backend.Close()

	assert.NotNil(t, backend)
	assert.Equal(t, "test-prefix", backend.prefix)

	// Test with invalid URL
	_, err = NewBackendFromURL("invalid-url", "test-prefix")
	assert.Error(t, err)
}

func TestBackend_Get_NonExistentKey(t *testing.T) {
	client := setupTestRedis(t)
	defer client.Close()

	backend := NewBackend(client, "test")
	ctx := context.Background()

	state, err := backend.Get(ctx, "non-existent")
	assert.NoError(t, err)
	assert.NotNil(t, state)
	assert.Equal(t, float64(0), state.Tokens)
	assert.WithinDuration(t, time.Now(), state.Created, 2*time.Second)
	assert.WithinDuration(t, time.Now(), state.LastUpdate, 2*time.Second)
}

func TestBackend_SetAndGet(t *testing.T) {
	client := setupTestRedis(t)
	defer client.Close()

	backend := NewBackend(client, "test")
	ctx := context.Background()

	// Create a test state
	now := time.Now()
	state := &core.State{
		Tokens:     5.5,
		LastUpdate: now,
		Created:    now,
	}

	// Set the state
	err := backend.Set(ctx, "test-key", state)
	assert.NoError(t, err)

	// Get the state back
	retrievedState, err := backend.Get(ctx, "test-key")
	assert.NoError(t, err)
	assert.NotNil(t, retrievedState)
	assert.Equal(t, state.Tokens, retrievedState.Tokens)
	assert.Equal(t, state.LastUpdate, retrievedState.LastUpdate)
	assert.Equal(t, state.Created, retrievedState.Created)
}

func TestBackend_Delete(t *testing.T) {
	client := setupTestRedis(t)
	defer client.Close()

	backend := NewBackend(client, "test")
	ctx := context.Background()

	// Set a state
	state := &core.State{
		Tokens:     10.0,
		LastUpdate: time.Now(),
		Created:    time.Now(),
	}
	err := backend.Set(ctx, "test-key", state)
	assert.NoError(t, err)

	// Verify it exists
	retrievedState, err := backend.Get(ctx, "test-key")
	assert.NoError(t, err)
	assert.Equal(t, state.Tokens, retrievedState.Tokens)

	// Delete it
	err = backend.Delete(ctx, "test-key")
	assert.NoError(t, err)

	// Verify it's gone (should return empty state)
	retrievedState, err = backend.Get(ctx, "test-key")
	assert.NoError(t, err)
	assert.Equal(t, float64(0), retrievedState.Tokens)
}

func TestBackend_KeyPrefixing(t *testing.T) {
	client := setupTestRedis(t)
	defer client.Close()

	backend1 := NewBackend(client, "prefix1")
	backend2 := NewBackend(client, "prefix2")
	ctx := context.Background()

	// Set states with different prefixes
	state1 := &core.State{Tokens: 1.0, LastUpdate: time.Now(), Created: time.Now()}
	state2 := &core.State{Tokens: 2.0, LastUpdate: time.Now(), Created: time.Now()}

	err := backend1.Set(ctx, "same-key", state1)
	assert.NoError(t, err)
	err = backend2.Set(ctx, "same-key", state2)
	assert.NoError(t, err)

	// Retrieve and verify they're different
	retrieved1, err := backend1.Get(ctx, "same-key")
	assert.NoError(t, err)
	assert.Equal(t, state1.Tokens, retrieved1.Tokens)

	retrieved2, err := backend2.Get(ctx, "same-key")
	assert.NoError(t, err)
	assert.Equal(t, state2.Tokens, retrieved2.Tokens)
}

func TestBackend_ConcurrentAccess(t *testing.T) {
	client := setupTestRedis(t)
	defer client.Close()

	backend := NewBackend(client, "test")
	ctx := context.Background()

	// Test concurrent writes
	const numGoroutines = 10
	const numOperations = 100

	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer func() { done <- true }()

			for j := 0; j < numOperations; j++ {
				key := fmt.Sprintf("concurrent-key-%d", id)
				state := &core.State{
					Tokens:     float64(j),
					LastUpdate: time.Now(),
					Created:    time.Now(),
				}

				err := backend.Set(ctx, key, state)
				assert.NoError(t, err)

				retrievedState, err := backend.Get(ctx, key)
				assert.NoError(t, err)
				assert.Equal(t, state.Tokens, retrievedState.Tokens)
			}
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}
}

func TestBackend_GetStats(t *testing.T) {
	client := setupTestRedis(t)
	defer client.Close()

	backend := NewBackend(client, "test")

	stats := backend.GetStats()
	assert.NotNil(t, stats)
	assert.Contains(t, stats, "total_connections")
	assert.Contains(t, stats, "idle_connections")
	assert.Contains(t, stats, "stale_connections")
}

func TestBackend_Close(t *testing.T) {
	client := setupTestRedis(t)
	defer client.Close()

	backend := NewBackend(client, "test")

	// Close should not error
	err := backend.Close()
	assert.NoError(t, err)
}

func BenchmarkBackend_Set(b *testing.B) {
	client := setupBenchmarkRedis()
	if client == nil {
		b.Skip("Redis not available")
	}
	defer client.Close()

	backend := NewBackend(client, "benchmark")
	ctx := context.Background()

	state := &core.State{
		Tokens:     10.0,
		LastUpdate: time.Now(),
		Created:    time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("benchmark-key-%d", i)
		err := backend.Set(ctx, key, state)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkBackend_Get(b *testing.B) {
	client := setupBenchmarkRedis()
	if client == nil {
		b.Skip("Redis not available")
	}
	defer client.Close()

	backend := NewBackend(client, "benchmark")
	ctx := context.Background()

	// Pre-populate with data
	state := &core.State{
		Tokens:     10.0,
		LastUpdate: time.Now(),
		Created:    time.Now(),
	}

	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("benchmark-key-%d", i)
		backend.Set(ctx, key, state)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("benchmark-key-%d", i%1000)
		_, err := backend.Get(ctx, key)
		if err != nil {
			b.Fatal(err)
		}
	}
}
