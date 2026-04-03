package traverse

import (
	"sync"
)

// StringInterning provides optimized string deduplication to reduce memory allocations
// when processing entity and property names that are repeated across many records.
//
// When processing millions of records from an OData service, entity names and property
// names are repeated in every record's JSON response. StringInterning deduplicates these
// strings, ensuring that identical strings reference the same memory location.
//
// For example, if every record has a "Name" property, StringInterning returns the same
// string reference for all 1 million records instead of allocating 1 million string copies.
//
// Example:
//
//	si := traverse.NewStringInterning()
//	propName1 := si.Intern("Name")      // allocates
//	propName2 := si.Intern("Name")      // returns cached reference
//	propName1 == propName2              // true, same memory location
type StringInterning struct {
	mu    sync.RWMutex
	cache map[string]string
}

// NewStringInterning creates a new string interning cache.
//
// NewStringInterning returns a ready-to-use *StringInterning with an initial
// capacity of 256 entries. It is safe for concurrent use.
func NewStringInterning() *StringInterning {
	return &StringInterning{
		cache: make(map[string]string, 256),
	}
}

// Intern returns a deduplicated reference to the string.
//
// If the string already exists in the cache, Intern returns the cached reference.
// Otherwise, it adds the string to the cache and returns it.
//
// Intern uses a fast-path read lock for cache hits and a slow-path write lock for
// misses with double-check pattern to avoid race conditions.
//
// Empty strings are returned as-is without caching.
func (si *StringInterning) Intern(s string) string {
	if s == "" {
		return s
	}

	// Fast path: read lock for most cases
	si.mu.RLock()
	if existing, ok := si.cache[s]; ok {
		si.mu.RUnlock()
		return existing
	}
	si.mu.RUnlock()

	// Slow path: acquire write lock if not found
	si.mu.Lock()
	// Double-check pattern to avoid race conditions
	if existing, ok := si.cache[s]; ok {
		si.mu.Unlock()
		return existing
	}
	// Add to cache
	si.cache[s] = s
	si.mu.Unlock()

	return s
}

// InternBatch interns multiple strings efficiently with a single lock acquisition.
//
// InternBatch is more efficient than calling [Intern] multiple times when processing
// many strings at once. It acquires one write lock to check/insert all provided strings,
// avoiding repeated lock acquisitions per string.
//
// Returns a new slice with deduplicated string references.
func (si *StringInterning) InternBatch(strings ...string) []string {
	if len(strings) == 0 {
		return strings
	}

	result := make([]string, len(strings))

	si.mu.Lock()
	defer si.mu.Unlock()

	for i, s := range strings {
		if s == "" {
			result[i] = s
			continue
		}

		if existing, ok := si.cache[s]; ok {
			result[i] = existing
		} else {
			si.cache[s] = s
			result[i] = s
		}
	}

	return result
}

// Clear removes all strings from the cache.
//
// Clear is useful for testing or resetting memory state. It reinitializes the cache
// with the default capacity of 256 entries.
func (si *StringInterning) Clear() {
	si.mu.Lock()
	defer si.mu.Unlock()
	si.cache = make(map[string]string, 256)
}

// CacheSize returns the current number of strings in the cache.
//
// CacheSize is useful for monitoring cache growth during bulk operations.
func (si *StringInterning) CacheSize() int {
	si.mu.RLock()
	defer si.mu.RUnlock()
	return len(si.cache)
}

// Global string interning instance for entity and property names
var globalInterning = NewStringInterning()

// InternString interns a string using the global interning cache.
//
// InternString should be used for frequently repeated strings like entity names
// and property names that appear across millions of records.
//
// The global cache is automatically initialized and can be managed with
// [ClearGlobalCache] and [GlobalCacheSize].
//
// Example:
//
//	propName := traverse.InternString("CustomerName")  // first call allocates
//	propName = traverse.InternString("CustomerName")   // second call returns cached ref
func InternString(s string) string {
	return globalInterning.Intern(s)
}

// ClearGlobalCache clears the global string interning cache.
//
// ClearGlobalCache removes all cached strings and reinitializes the cache.
// Useful for testing or when you want to reset memory state in long-running processes.
func ClearGlobalCache() {
	globalInterning.Clear()
}

// GlobalCacheSize returns the current size of the global string interning cache.
//
// GlobalCacheSize returns the number of unique interned strings in the global cache.
// Use this to monitor cache growth and detect potential memory leaks.
func GlobalCacheSize() int {
	return globalInterning.CacheSize()
}
