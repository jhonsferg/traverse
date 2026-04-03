package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jhonsferg/traverse"
	"github.com/redis/go-redis/v9"
)

// Cache is a Redis-backed cache implementation for OData metadata.
// It implements traverse.CacheStore interface.
type Cache struct {
	client    *redis.Client
	keyPrefix string
	ttl       time.Duration
}

// Config holds configuration for Redis cache.
type Config struct {
	Addr      string        // Redis address (default: "localhost:6379")
	Password  string        // Redis password (optional)
	DB        int           // Database number (default: 0)
	KeyPrefix string        // Prefix for all cache keys (default: "traverse:")
	TTL       time.Duration // Cache TTL (default: 1 hour)
}

// New creates a new Redis-backed cache.
func New(cfg *Config) (*Cache, error) {
	if cfg == nil {
		cfg = &Config{}
	}

	// Set defaults
	if cfg.Addr == "" {
		cfg.Addr = "localhost:6379"
	}
	if cfg.KeyPrefix == "" {
		cfg.KeyPrefix = "traverse:"
	}
	if cfg.TTL == 0 {
		cfg.TTL = time.Hour
	}

	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := client.Ping(ctx).Err()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &Cache{
		client:    client,
		keyPrefix: cfg.KeyPrefix,
		ttl:       cfg.TTL,
	}, nil
}

// Get retrieves cached metadata by key.
// Returns nil and false if key is not found.
func (c *Cache) Get(key string) (*traverse.Metadata, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	redisKey := c.keyPrefix + key
	data, err := c.client.Get(ctx, redisKey).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, false
		}
		// Log error but don't fail - return cache miss
		return nil, false
	}

	var metadata *traverse.Metadata
	if err := json.Unmarshal([]byte(data), &metadata); err != nil {
		// Corrupted data - log and return cache miss
		return nil, false
	}

	return metadata, true
}

// Set stores metadata in the cache with the configured TTL.
func (c *Cache) Set(key string, metadata *traverse.Metadata) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	data, err := json.Marshal(metadata)
	if err != nil {
		// Failed to serialize - silently skip caching
		return
	}

	redisKey := c.keyPrefix + key
	c.client.Set(ctx, redisKey, data, c.ttl)
}

// Clear removes all entries from the cache with the configured prefix.
func (c *Cache) Clear() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Scan for keys with the prefix and delete them
	iter := c.client.Scan(ctx, 0, c.keyPrefix+"*", 100).Iterator()

	var keys []string
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}

	if len(keys) > 0 {
		c.client.Del(ctx, keys...)
	}
}

// Close closes the Redis connection.
func (c *Cache) Close() error {
	return c.client.Close()
}

// Delete removes a specific key from the cache.
func (c *Cache) Delete(key string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	redisKey := c.keyPrefix + key
	c.client.Del(ctx, redisKey)
}

// Exists checks if a key exists in the cache.
func (c *Cache) Exists(key string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	redisKey := c.keyPrefix + key
	exists, err := c.client.Exists(ctx, redisKey).Result()
	if err != nil {
		return false
	}
	return exists > 0
}

// Size returns the number of cached entries with the configured prefix.
func (c *Cache) Size() int64 {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	iter := c.client.Scan(ctx, 0, c.keyPrefix+"*", 100).Iterator()

	count := int64(0)
	for iter.Next(ctx) {
		count++
	}
	return count
}

// SetTTL updates the TTL for an existing key.
func (c *Cache) SetTTL(key string, ttl time.Duration) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	redisKey := c.keyPrefix + key
	ok, err := c.client.Expire(ctx, redisKey, ttl).Result()
	if err != nil {
		return false
	}
	return ok
}
