package memory

import (
	"testing"
	"time"

	"github.com/jhonsferg/traverse"
	"github.com/stretchr/testify/assert"
)

func TestCacheGetSet(t *testing.T) {
	cache := New(time.Hour)

	metadata := &traverse.Metadata{
		Version:   "1.0",
		Namespace: "test",
		EntityTypes: []traverse.EntityType{
			{Name: "Product"},
		},
	}

	// Test Set and Get
	cache.Set("test_key", metadata)
	retrieved, exists := cache.Get("test_key")

	assert.True(t, exists)
	assert.Equal(t, metadata, retrieved)
}

func TestCacheGetMissing(t *testing.T) {
	cache := New(time.Hour)

	retrieved, exists := cache.Get("nonexistent_key")

	assert.False(t, exists)
	assert.Nil(t, retrieved)
}

func TestCacheTTLExpiration(t *testing.T) {
	cache := New(100 * time.Millisecond)

	metadata := &traverse.Metadata{
		Version:   "1.0",
		Namespace: "test",
		EntityTypes: []traverse.EntityType{
			{Name: "Product"},
		},
	}

	cache.Set("test_key", metadata)

	// Immediately should exist
	retrieved, exists := cache.Get("test_key")
	assert.True(t, exists)
	assert.NotNil(t, retrieved)

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Should be expired
	retrieved, exists = cache.Get("test_key")
	assert.False(t, exists)
	assert.Nil(t, retrieved)
}

func TestCacheNoTTL(t *testing.T) {
	cache := New(0) // No TTL

	metadata := &traverse.Metadata{
		Version:   "1.0",
		Namespace: "test",
		EntityTypes: []traverse.EntityType{
			{Name: "Product"},
		},
	}

	cache.Set("test_key", metadata)

	// Wait a bit and should still exist
	time.Sleep(200 * time.Millisecond)

	retrieved, exists := cache.Get("test_key")
	assert.True(t, exists)
	assert.Equal(t, metadata, retrieved)
}

func TestCacheClear(t *testing.T) {
	cache := New(time.Hour)

	metadata := &traverse.Metadata{
		Version:   "1.0",
		Namespace: "test",
		EntityTypes: []traverse.EntityType{
			{Name: "Product"},
		},
	}

	cache.Set("key1", metadata)
	cache.Set("key2", metadata)

	assert.Equal(t, 2, cache.Size())

	cache.Clear()

	assert.Equal(t, 0, cache.Size())
	_, exists := cache.Get("key1")
	assert.False(t, exists)
}

func TestCacheMultipleKeys(t *testing.T) {
	cache := New(time.Hour)

	meta1 := &traverse.Metadata{
		Version:   "1.0",
		Namespace: "test",
		EntityTypes: []traverse.EntityType{
			{Name: "E1"},
		},
	}
	meta2 := &traverse.Metadata{
		Version:   "1.0",
		Namespace: "test",
		EntityTypes: []traverse.EntityType{
			{Name: "E2"},
		},
	}

	cache.Set("key1", meta1)
	cache.Set("key2", meta2)

	retrieved1, exists1 := cache.Get("key1")
	retrieved2, exists2 := cache.Get("key2")

	assert.True(t, exists1)
	assert.True(t, exists2)
	assert.Equal(t, meta1, retrieved1)
	assert.Equal(t, meta2, retrieved2)
}

func TestCacheConcurrentAccess(t *testing.T) {
	cache := New(time.Hour)
	done := make(chan bool, 100)

	metadata := &traverse.Metadata{
		Version:   "1.0",
		Namespace: "test",
		EntityTypes: []traverse.EntityType{
			{Name: "Product"},
		},
	}

	// Concurrent writes
	for i := 0; i < 50; i++ {
		go func(idx int) {
			cache.Set("concurrent_key", metadata)
			done <- true
		}(i)
	}

	// Concurrent reads
	for i := 0; i < 50; i++ {
		go func(idx int) {
			cache.Get("concurrent_key")
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 100; i++ {
		<-done
	}

	retrieved, exists := cache.Get("concurrent_key")
	assert.True(t, exists)
	assert.Equal(t, metadata, retrieved)
}

func TestCacheCleanupExpired(t *testing.T) {
	cache := New(100 * time.Millisecond)

	meta := &traverse.Metadata{
		Version:   "1.0",
		Namespace: "test",
		EntityTypes: []traverse.EntityType{
			{Name: "E"},
		},
	}

	for i := 0; i < 10; i++ {
		cache.Set("key"+string(rune(i+'0')), meta)
	}

	assert.Equal(t, 10, cache.Size())

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Cleanup
	removed := cache.CleanupExpired()
	assert.Equal(t, 10, removed)
	assert.Equal(t, 0, cache.Size())
}

func TestCacheCleanupWithNoTTL(t *testing.T) {
	cache := New(0) // No TTL

	meta := &traverse.Metadata{
		Version:   "1.0",
		Namespace: "test",
		EntityTypes: []traverse.EntityType{
			{Name: "E"},
		},
	}

	cache.Set("key", meta)
	removed := cache.CleanupExpired()

	// Should not remove anything when TTL is 0
	assert.Equal(t, 0, removed)
	assert.Equal(t, 1, cache.Size())
}

func TestCacheImplementsInterface(t *testing.T) {
	cache := New(time.Hour)
	var _ traverse.CacheStore = cache
}

func BenchmarkCacheGet(b *testing.B) {
	cache := New(time.Hour)
	metadata := &traverse.Metadata{
		Version:   "1.0",
		Namespace: "test",
		EntityTypes: []traverse.EntityType{
			{Name: "Product"},
		},
	}
	cache.Set("bench_key", metadata)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Get("bench_key")
	}
}

func BenchmarkCacheSet(b *testing.B) {
	cache := New(time.Hour)
	metadata := &traverse.Metadata{
		Version:   "1.0",
		Namespace: "test",
		EntityTypes: []traverse.EntityType{
			{Name: "Product"},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Set("bench_key", metadata)
	}
}

func BenchmarkCacheConcurrentGetSet(b *testing.B) {
	cache := New(time.Hour)
	metadata := &traverse.Metadata{
		Version:   "1.0",
		Namespace: "test",
		EntityTypes: []traverse.EntityType{
			{Name: "Product"},
		},
	}
	cache.Set("bench_key", metadata)

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			if i%2 == 0 {
				cache.Get("bench_key")
			} else {
				cache.Set("bench_key", metadata)
			}
			i++
		}
	})
}
