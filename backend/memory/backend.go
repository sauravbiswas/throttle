package memory

import (
	"context"
	"sync"

	"github.com/throttle/core"
)

// Backend implements an in-memory storage backend for rate limiting
type Backend struct {
	store map[string]*core.State
	mu    sync.RWMutex
}

// NewBackend creates a new in-memory backend
func NewBackend() *Backend {
	return &Backend{
		store: make(map[string]*core.State),
	}
}

// Get retrieves the current state for a key
func (b *Backend) Get(ctx context.Context, key string) (*core.State, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	state, exists := b.store[key]
	if !exists {
		return nil, nil
	}

	// Return a copy to prevent external modifications
	return &core.State{
		Tokens:     state.Tokens,
		LastUpdate: state.LastUpdate,
		Created:    state.Created,
	}, nil
}

// Set stores the state for a key
func (b *Backend) Set(ctx context.Context, key string, state *core.State) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Store a copy to prevent external modifications
	b.store[key] = &core.State{
		Tokens:     state.Tokens,
		LastUpdate: state.LastUpdate,
		Created:    state.Created,
	}

	return nil
}

// Delete removes the state for a key
func (b *Backend) Delete(ctx context.Context, key string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	delete(b.store, key)
	return nil
}

// Close performs any necessary cleanup
func (b *Backend) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Clear the store
	b.store = make(map[string]*core.State)
	return nil
}

// Stats returns statistics about the backend
func (b *Backend) Stats() map[string]interface{} {
	b.mu.RLock()
	defer b.mu.RUnlock()

	return map[string]interface{}{
		"keys_count": len(b.store),
	}
}
