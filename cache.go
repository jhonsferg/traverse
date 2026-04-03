package traverse

import "sync"

// CacheStore is an interface for caching OData metadata.
//
// CacheStore provides a contract for caching [Metadata] retrieved from OData services.
// Different implementations can use in-memory storage, Redis, file-based caching, or other strategies.
//
// This interface allows the client to optimize metadata fetches by caching the service schema
// across requests, avoiding repeated $metadata calls for the same service.
//
// Example custom implementation:
//
//	type RedisCache struct {
//		client *redis.Client
//	}
//	func (c *RedisCache) Get(key string) (*Metadata, bool) { ... }
//	func (c *RedisCache) Set(key string, metadata *Metadata) { ... }
//	func (c *RedisCache) Clear() { ... }
type CacheStore interface {
	// Get retrieves cached metadata by key.
	// Returns nil and false if key is not found.
	Get(key string) (*Metadata, bool)

	// Set stores metadata in the cache with the given key.
	Set(key string, metadata *Metadata)

	// Clear removes all entries from the cache.
	Clear()
}

// NoOpCache is a no-op cache implementation that doesn't cache anything.
//
// NoOpCache is useful for development, testing, or when caching is not desired.
// It always returns cache misses and ignores Set/Clear operations.
type NoOpCache struct{}

// Get returns nil, false (cache miss).
func (c *NoOpCache) Get(key string) (*Metadata, bool) {
	return nil, false
}

// Set is a no-op.
func (c *NoOpCache) Set(key string, metadata *Metadata) {
}

// Clear is a no-op.
func (c *NoOpCache) Clear() {
}

// MemoryCache is a simple in-memory cache for OData metadata.
//
// MemoryCache is safe for concurrent use using lock-free sync.Map.
// It stores [Metadata] objects keyed by service endpoint or custom identifier.
//
// Metadata is cached for the lifetime of the MemoryCache instance; it is not
// automatically expired. For long-running processes, consider periodic cache
// invalidation or custom expiration logic in a wrapper.
//
// Example:
//
//	cache := traverse.NewMemoryCache()
//	client, _ := traverse.New(
//		traverse.WithBaseURL("https://odata.service/"),
//		traverse.WithCache(cache),
//	)
//	// First call fetches and caches metadata
//	metadata, _ := client.Metadata(ctx)
//	// Subsequent calls hit the cache
//	metadata, _ := client.Metadata(ctx)
type MemoryCache struct {
	cache sync.Map // lock-free map[string]*Metadata
}

// NewMemoryCache creates a new in-memory metadata cache.
//
// NewMemoryCache returns a ready-to-use *MemoryCache that is safe for
// concurrent access from multiple goroutines.
func NewMemoryCache() *MemoryCache {
	return &MemoryCache{}
}

// Get retrieves cached metadata by key.
//
// Get returns the [Metadata] stored under the given key, or (nil, false) if not cached.
func (c *MemoryCache) Get(key string) (*Metadata, bool) {
	val, found := c.cache.Load(key)
	if !found {
		return nil, false
	}
	metadata, ok := val.(*Metadata)
	return metadata, ok
}

// Set stores metadata in the cache with the given key.
//
// Set overwrites any existing entry for the key. The operation is atomic and safe
// for concurrent use.
func (c *MemoryCache) Set(key string, metadata *Metadata) {
	c.cache.Store(key, metadata)
}

// Clear removes all entries from the cache.
//
// Clear iterates over all cached entries and deletes them, leaving the cache empty.
// Useful for cache invalidation or cleanup.
func (c *MemoryCache) Clear() {
	c.cache.Range(func(key, value interface{}) bool {
		c.cache.Delete(key)
		return true
	})
}
