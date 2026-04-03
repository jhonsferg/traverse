package memory

import (
	"sync"
	"time"

	"github.com/jhonsferg/traverse"
)

// Cache is a thread-safe in-memory cache implementation with TTL support.
// It implements traverse.CacheStore interface.
type Cache struct {
	mu      sync.RWMutex
	entries map[string]*cacheEntry
	ttl     time.Duration
}

// cacheEntry holds cached data with expiration information.
type cacheEntry struct {
	metadata  *traverse.Metadata
	expiresAt time.Time
}

// New creates a new in-memory cache with the given TTL.
// If ttl is 0 or negative, entries never expire (except on manual Clear).
func New(ttl time.Duration) *Cache {
	return &Cache{
		entries: make(map[string]*cacheEntry),
		ttl:     ttl,
	}
}

// Get retrieves cached metadata by key.
// Returns nil and false if key is not found or has expired.
func (c *Cache) Get(key string) (*traverse.Metadata, bool) {
	c.mu.RLock()
	entry, exists := c.entries[key]
	c.mu.RUnlock()

	if !exists {
		return nil, false
	}

	// Check expiration
	if c.ttl > 0 && time.Now().After(entry.expiresAt) {
		// Expired, remove it
		c.mu.Lock()
		delete(c.entries, key)
		c.mu.Unlock()
		return nil, false
	}

	return entry.metadata, true
}

// Set stores metadata in the cache with the configured TTL.
func (c *Cache) Set(key string, metadata *traverse.Metadata) {
	c.mu.Lock()
	defer c.mu.Unlock()

	expiresAt := time.Time{} // No expiration by default
	if c.ttl > 0 {
		expiresAt = time.Now().Add(c.ttl)
	}

	c.entries[key] = &cacheEntry{
		metadata:  metadata,
		expiresAt: expiresAt,
	}
}

// Clear removes all entries from the cache.
func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries = make(map[string]*cacheEntry)
}

// Size returns the number of cached entries.
// Note: This includes expired entries that haven't been cleaned up yet.
func (c *Cache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.entries)
}

// CleanupExpired removes all expired entries from the cache.
// This is useful for periodic maintenance but not required for correctness
// since Get() will lazily clean up expired entries.
func (c *Cache) CleanupExpired() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.ttl <= 0 {
		return 0
	}

	now := time.Now()
	removed := 0
	for key, entry := range c.entries {
		if now.After(entry.expiresAt) {
			delete(c.entries, key)
			removed++
		}
	}
	return removed
}
