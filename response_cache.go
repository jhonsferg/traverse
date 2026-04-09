package traverse

import (
	"strings"
	"sync"
	"time"
)

// ResponseCacheEntry holds a cached HTTP response body alongside metadata
// needed for conditional revalidation (ETag, Last-Modified) and TTL tracking.
type ResponseCacheEntry struct {
	// Body is the raw JSON response body bytes.
	Body []byte
	// ETag is the value of the ETag response header, used for If-None-Match.
	ETag string
	// LastModified is the value of the Last-Modified response header,
	// used for If-Modified-Since when ETag is absent.
	LastModified string
	// ExpiresAt is the absolute time after which this entry must be revalidated.
	// A zero value means the entry never expires automatically.
	ExpiresAt time.Time
}

// isExpired reports whether the entry has passed its ExpiresAt deadline.
// Entries with a zero ExpiresAt are considered non-expiring.
func (e *ResponseCacheEntry) isExpired() bool {
	return !e.ExpiresAt.IsZero() && time.Now().After(e.ExpiresAt)
}

// ResponseCache is the storage backend for HTTP-level response caching in traverse.
//
// Implement this interface to plug in a custom cache backend such as Redis,
// Memcached, or a specialised in-memory store. All methods must be safe for
// concurrent use.
//
// The default implementation is returned by [NewInMemoryResponseCache].
//
// Example:
//
//	client, _ := traverse.New(
//	    traverse.WithBaseURL("https://odata.example.com/v4"),
//	    traverse.WithResponseCache(traverse.NewInMemoryResponseCache()),
//	)
type ResponseCache interface {
	// Get returns the cached entry for key.
	// Expired entries are returned (callers must check isExpired) so that
	// their ETag or Last-Modified can be used for conditional revalidation.
	// Returns nil and false if the key is absent entirely.
	Get(key string) (*ResponseCacheEntry, bool)

	// Set stores entry for key. When ttl > 0, ExpiresAt is set to now+ttl.
	// A zero or negative ttl stores the entry without automatic expiry.
	Set(key string, entry *ResponseCacheEntry, ttl time.Duration)

	// Delete removes the entry for key. No-op if key is absent.
	Delete(key string)

	// Invalidate removes all entries whose key begins with prefix.
	// Used to purge all cached pages for an entity set after mutations.
	Invalidate(prefix string)

	// Clear removes all entries from the cache.
	Clear()
}

// inMemoryResponseCache is a sync.Map-based ResponseCache with per-entry TTL.
type inMemoryResponseCache struct {
	m sync.Map
}

// NewInMemoryResponseCache returns a new in-memory ResponseCache backed by
// sync.Map. It is safe for concurrent use and requires no external dependencies.
//
// Entries are stored until they are explicitly removed via Delete, Invalidate,
// or Clear, or until they are lazily cleaned up on access after expiry.
func NewInMemoryResponseCache() ResponseCache {
	return &inMemoryResponseCache{}
}

// Get returns the entry for key if it exists, even if expired.
// Returns nil and false only when the key is entirely absent.
func (c *inMemoryResponseCache) Get(key string) (*ResponseCacheEntry, bool) {
	val, ok := c.m.Load(key)
	if !ok {
		return nil, false
	}
	entry, ok := val.(*ResponseCacheEntry)
	if !ok {
		return nil, false
	}
	return entry, true
}

// Set stores entry for key. When ttl > 0, entry.ExpiresAt is set to now+ttl.
// The stored entry is a copy to prevent mutation of the caller's value.
func (c *inMemoryResponseCache) Set(key string, entry *ResponseCacheEntry, ttl time.Duration) {
	stored := &ResponseCacheEntry{
		Body:         entry.Body,
		ETag:         entry.ETag,
		LastModified: entry.LastModified,
	}
	if ttl > 0 {
		stored.ExpiresAt = time.Now().Add(ttl)
	}
	c.m.Store(key, stored)
}

// Delete removes the entry for key.
func (c *inMemoryResponseCache) Delete(key string) {
	c.m.Delete(key)
}

// Invalidate removes all entries whose key begins with prefix.
func (c *inMemoryResponseCache) Invalidate(prefix string) {
	c.m.Range(func(k, _ interface{}) bool {
		if key, ok := k.(string); ok && strings.HasPrefix(key, prefix) {
			c.m.Delete(key)
		}
		return true
	})
}

// Clear removes all entries from the cache.
func (c *inMemoryResponseCache) Clear() {
	c.m.Range(func(k, _ interface{}) bool {
		c.m.Delete(k)
		return true
	})
}
