package redis

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// CacheHandler is the base cache service. It wraps a *redis.Client and exposes
// typed, nil-safe methods that all infrastructure services can use uniformly.
// When the underlying client is nil (Redis unavailable) every operation is a
// no-op / returns a sentinel "miss" so callers degrade gracefully.
type CacheHandler struct {
	client *redis.Client
}

// NewCacheHandler creates a CacheHandler. client may be nil; all methods are
// safe to call in that case and will behave as cache misses / no-ops.
func NewCacheHandler(client *redis.Client) *CacheHandler {
	return &CacheHandler{client: client}
}

// Available reports whether the underlying Redis client is connected.
func (h *CacheHandler) Available() bool {
	return h.client != nil
}

// Set stores value at key with the given TTL. A zero TTL means no expiry.
func (h *CacheHandler) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	if h.client == nil {
		return nil
	}
	return h.client.Set(ctx, key, value, ttl).Err()
}

// Get retrieves the string value at key.
// Returns ("", false, nil) when the key does not exist.
// Returns ("", false, err) on a real Redis error.
func (h *CacheHandler) Get(ctx context.Context, key string) (string, bool, error) {
	if h.client == nil {
		return "", false, nil
	}
	val, err := h.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return val, true, nil
}

// Del removes one or more keys. It is a no-op when Redis is unavailable.
func (h *CacheHandler) Del(ctx context.Context, keys ...string) error {
	if h.client == nil {
		return nil
	}
	return h.client.Del(ctx, keys...).Err()
}

// Exists returns true if all of the given keys exist in Redis.
func (h *CacheHandler) Exists(ctx context.Context, keys ...string) (bool, error) {
	if h.client == nil {
		return false, nil
	}
	n, err := h.client.Exists(ctx, keys...).Result()
	if err != nil {
		return false, err
	}
	return n == int64(len(keys)), nil
}

// SetNX sets key to value only if it does not already exist.
// Returns true if the key was set, false if it already existed.
func (h *CacheHandler) SetNX(ctx context.Context, key, value string, ttl time.Duration) (bool, error) {
	if h.client == nil {
		return false, nil
	}
	return h.client.SetNX(ctx, key, value, ttl).Result()
}

// Expire updates the TTL on an existing key. Returns false if the key does not
// exist; true if the TTL was set successfully.
func (h *CacheHandler) Expire(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	if h.client == nil {
		return false, nil
	}
	return h.client.Expire(ctx, key, ttl).Result()
}

// TTL returns the remaining time-to-live for key.
// Returns -2 if the key does not exist, -1 if it has no expiry.
func (h *CacheHandler) TTL(ctx context.Context, key string) (time.Duration, error) {
	if h.client == nil {
		return -2, nil
	}
	return h.client.TTL(ctx, key).Result()
}
