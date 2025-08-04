package memory

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/throttle/core"
)

func TestBackend_GetSet(t *testing.T) {
	backend := NewBackend()
	ctx := context.Background()

	// Test setting and getting a state
	state := &core.State{
		Tokens:     5.0,
		LastUpdate: time.Now(),
		Created:    time.Now(),
	}

	err := backend.Set(ctx, "test-key", state)
	assert.NoError(t, err)

	retrieved, err := backend.Get(ctx, "test-key")
	assert.NoError(t, err)
	assert.NotNil(t, retrieved)
	assert.Equal(t, state.Tokens, retrieved.Tokens)
	assert.Equal(t, state.LastUpdate, retrieved.LastUpdate)
	assert.Equal(t, state.Created, retrieved.Created)
}

func TestBackend_GetNonExistent(t *testing.T) {
	backend := NewBackend()
	ctx := context.Background()

	// Test getting a non-existent key
	retrieved, err := backend.Get(ctx, "non-existent")
	assert.NoError(t, err)
	assert.Nil(t, retrieved)
}

func TestBackend_Delete(t *testing.T) {
	backend := NewBackend()
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
	retrieved, err := backend.Get(ctx, "test-key")
	assert.NoError(t, err)
	assert.NotNil(t, retrieved)

	// Delete it
	err = backend.Delete(ctx, "test-key")
	assert.NoError(t, err)

	// Verify it's gone
	retrieved, err = backend.Get(ctx, "test-key")
	assert.NoError(t, err)
	assert.Nil(t, retrieved)
}

func TestBackend_Concurrency(t *testing.T) {
	backend := NewBackend()
	ctx := context.Background()

	// Test concurrent access
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			key := fmt.Sprintf("key-%d", id)
			state := &core.State{
				Tokens:     float64(id),
				LastUpdate: time.Now(),
				Created:    time.Now(),
			}

			err := backend.Set(ctx, key, state)
			assert.NoError(t, err)

			retrieved, err := backend.Get(ctx, key)
			assert.NoError(t, err)
			assert.NotNil(t, retrieved)

			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestBackend_Stats(t *testing.T) {
	backend := NewBackend()
	ctx := context.Background()

	// Add some states
	for i := 0; i < 5; i++ {
		key := fmt.Sprintf("key-%d", i)
		state := &core.State{
			Tokens:     float64(i),
			LastUpdate: time.Now(),
			Created:    time.Now(),
		}
		backend.Set(ctx, key, state)
	}

	stats := backend.Stats()
	assert.Equal(t, 5, stats["keys_count"])
}

func TestBackend_Close(t *testing.T) {
	backend := NewBackend()
	ctx := context.Background()

	// Add a state
	state := &core.State{
		Tokens:     5.0,
		LastUpdate: time.Now(),
		Created:    time.Now(),
	}
	backend.Set(ctx, "test-key", state)

	// Close the backend
	err := backend.Close()
	assert.NoError(t, err)

	// Verify the store is cleared
	stats := backend.Stats()
	assert.Equal(t, 0, stats["keys_count"])
}
