package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/throttle/core"
)

// Backend implements the core.Backend interface using Redis
type Backend struct {
	client *redis.Client
	prefix string
}

// NewBackend creates a new Redis backend
func NewBackend(client *redis.Client, prefix string) *Backend {
	if prefix == "" {
		prefix = "throttle"
	}
	return &Backend{
		client: client,
		prefix: prefix,
	}
}

// NewBackendFromURL creates a new Redis backend from a connection URL
func NewBackendFromURL(url, prefix string) (*Backend, error) {
	opts, err := redis.ParseURL(url)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %w", err)
	}

	client := redis.NewClient(opts)

	// Test the connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return NewBackend(client, prefix), nil
}

// Get retrieves the state for a key from Redis
func (b *Backend) Get(ctx context.Context, key string) (*core.State, error) {
	redisKey := b.makeKey(key)

	data, err := b.client.Get(ctx, redisKey).Bytes()
	if err != nil {
		if err == redis.Nil {
			// Key doesn't exist, return empty state
			now := time.Now()
			return &core.State{
				Tokens:     0,
				LastUpdate: now,
				Created:    now,
			}, nil
		}
		return nil, fmt.Errorf("failed to get key %s from Redis: %w", key, err)
	}

	var state core.State
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to unmarshal state for key %s: %w", key, err)
	}

	return &state, nil
}

// Set stores the state for a key in Redis
func (b *Backend) Set(ctx context.Context, key string, state *core.State) error {
	redisKey := b.makeKey(key)

	data, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("failed to marshal state for key %s: %w", key, err)
	}

	// Store with expiration to prevent memory leaks
	// Use a reasonable TTL based on the rate limit window
	ttl := 24 * time.Hour // Default TTL, can be made configurable

	if err := b.client.Set(ctx, redisKey, data, ttl).Err(); err != nil {
		return fmt.Errorf("failed to set key %s in Redis: %w", key, err)
	}

	return nil
}

// Delete removes the state for a key from Redis
func (b *Backend) Delete(ctx context.Context, key string) error {
	redisKey := b.makeKey(key)

	if err := b.client.Del(ctx, redisKey).Err(); err != nil {
		return fmt.Errorf("failed to delete key %s from Redis: %w", key, err)
	}

	return nil
}

// Close closes the Redis connection
func (b *Backend) Close() error {
	return b.client.Close()
}

// makeKey creates a Redis key with the configured prefix
func (b *Backend) makeKey(key string) string {
	return fmt.Sprintf("%s:%s", b.prefix, key)
}

// GetStats returns Redis connection statistics
func (b *Backend) GetStats() map[string]interface{} {
	stats := b.client.PoolStats()
	return map[string]interface{}{
		"total_connections": stats.TotalConns,
		"idle_connections":  stats.IdleConns,
		"stale_connections": stats.StaleConns,
	}
}
