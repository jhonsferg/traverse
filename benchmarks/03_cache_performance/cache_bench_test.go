package benchmarks

import (
	"context"
	"testing"
	"time"

	"github.com/jhonsferg/traverse"
	"github.com/jhonsferg/traverse/ext/cache/memory"
)

// BenchmarkCacheMemoryGetHit - Memory cache hit performance
func BenchmarkCacheMemoryGetHit(b *testing.B) {
	cache := memory.New(10 * time.Minute)

	// Pre-populate cache
	testMetadata := &traverse.Metadata{
		Version:   "1.0",
		Namespace: "test",
		EntityTypes: []traverse.EntityType{
			{Name: "Product", Key: []traverse.PropertyRef{{Name: "ID"}}},
		},
	}

	for i := 0; i < 1000; i++ {
		cache.Set("metadata_"+string(rune(i%100)), testMetadata)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := "metadata_" + string(rune(i%100))
		_, _ = cache.Get(key)
	}
}

// BenchmarkCacheMemoryGetMiss - Memory cache miss performance
func BenchmarkCacheMemoryGetMiss(b *testing.B) {
	cache := memory.New(10 * time.Minute)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := "nonexistent_" + string(rune(i))
		_, _ = cache.Get(key)
	}
}

// BenchmarkCacheMemorySet - Memory cache write performance
func BenchmarkCacheMemorySet(b *testing.B) {
	cache := memory.New(10 * time.Minute)

	testMetadata := &traverse.Metadata{
		Version:   "1.0",
		Namespace: "test",
		EntityTypes: []traverse.EntityType{
			{Name: "Product", Key: []traverse.PropertyRef{{Name: "ID"}}},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := "metadata_" + string(rune(i))
		cache.Set(key, testMetadata)
	}
}

// BenchmarkCacheMemoryConcurrentRead - Concurrent cache reads
func BenchmarkCacheMemoryConcurrentRead(b *testing.B) {
	cache := memory.New(10 * time.Minute)

	testMetadata := &traverse.Metadata{
		Version:   "1.0",
		Namespace: "test",
		EntityTypes: []traverse.EntityType{
			{Name: "Product", Key: []traverse.PropertyRef{{Name: "ID"}}},
		},
	}

	for i := 0; i < 100; i++ {
		key := "metadata_" + string(rune(i))
		cache.Set(key, testMetadata)
	}

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := "metadata_" + string(rune(i%100))
			_, _ = cache.Get(key)
			i++
		}
	})
}

// BenchmarkCacheMemoryConcurrentWrite - Concurrent cache writes
func BenchmarkCacheMemoryConcurrentWrite(b *testing.B) {
	cache := memory.New(10 * time.Minute)

	testMetadata := &traverse.Metadata{
		Version:   "1.0",
		Namespace: "test",
		EntityTypes: []traverse.EntityType{
			{Name: "Product", Key: []traverse.PropertyRef{{Name: "ID"}}},
		},
	}

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := "metadata_" + string(rune(i))
			cache.Set(key, testMetadata)
			i++
		}
	})
}

// BenchmarkCacheMemoryConcurrentMixed - Concurrent mixed read/write
func BenchmarkCacheMemoryConcurrentMixed(b *testing.B) {
	cache := memory.New(10 * time.Minute)

	testMetadata := &traverse.Metadata{
		Version:   "1.0",
		Namespace: "test",
		EntityTypes: []traverse.EntityType{
			{Name: "Product", Key: []traverse.PropertyRef{{Name: "ID"}}},
		},
	}

	// Pre-populate
	for i := 0; i < 100; i++ {
		key := "metadata_" + string(rune(i))
		cache.Set(key, testMetadata)
	}

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			if i%2 == 0 {
				// Read
				key := "metadata_" + string(rune(i%100))
				_, _ = cache.Get(key)
			} else {
				// Write
				key := "metadata_" + string(rune(i))
				cache.Set(key, testMetadata)
			}
			i++
		}
	})
}

// BenchmarkCacheMemoryTTLExpiration - TTL expiration performance
func BenchmarkCacheMemoryTTLExpiration(b *testing.B) {
	cache := memory.New(1 * time.Millisecond)

	testMetadata := &traverse.Metadata{
		Version:   "1.0",
		Namespace: "test",
		EntityTypes: []traverse.EntityType{
			{Name: "Product", Key: []traverse.PropertyRef{{Name: "ID"}}},
		},
	}

	for i := 0; i < 100; i++ {
		key := "metadata_" + string(rune(i))
		cache.Set(key, testMetadata)
	}

	time.Sleep(10 * time.Millisecond)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := "metadata_" + string(rune(i%100))
		_, exists := cache.Get(key)
		if exists {
			b.Fatal("Expected expired entry to not exist")
		}
	}
}

// BenchmarkCacheMemoryClear - Cache clear performance
func BenchmarkCacheMemoryClear(b *testing.B) {
	testMetadata := &traverse.Metadata{
		Version:   "1.0",
		Namespace: "test",
		EntityTypes: []traverse.EntityType{
			{Name: "Product", Key: []traverse.PropertyRef{{Name: "ID"}}},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache := memory.New(10 * time.Minute)

		for j := 0; j < 1000; j++ {
			key := "metadata_" + string(rune(j))
			cache.Set(key, testMetadata)
		}

		cache.Clear()
	}
}

// BenchmarkMetadataCaching - Real metadata caching scenario
func BenchmarkMetadataCaching(b *testing.B) {
	server := NewMockODataServer(ServerConfig{
		Latency:         5 * time.Millisecond,
		DefaultPageSize: 100,
		MaxRecords:      1000,
		RecordSize:      RecordSizeSmall,
	})
	defer server.Close()

	cache := memory.New(10 * time.Minute)

	client, _ := traverse.New(
		traverse.WithBaseURL(server.URL),
		traverse.WithODataVersion(traverse.ODataV4),
		traverse.WithMetadataCache(cache),
	)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = client.Metadata(context.Background())
	}
}

// BenchmarkQueryWithCache - Query execution with/without cache
func BenchmarkQueryWithCache(b *testing.B) {
	server := NewMockODataServer(ServerConfig{
		Latency:         5 * time.Millisecond,
		DefaultPageSize: 100,
		MaxRecords:      1000,
		RecordSize:      RecordSizeSmall,
	})
	defer server.Close()

	cache := memory.New(10 * time.Minute)

	clientWithCache, _ := traverse.New(
		traverse.WithBaseURL(server.URL),
		traverse.WithODataVersion(traverse.ODataV4),
		traverse.WithMetadataCache(cache),
	)

	clientWithoutCache, _ := traverse.New(
		traverse.WithBaseURL(server.URL),
		traverse.WithODataVersion(traverse.ODataV4),
	)

	b.Run("WithCache", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = clientWithCache.From("Products").Top(10).Collect(context.Background())
		}
	})

	b.Run("WithoutCache", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = clientWithoutCache.From("Products").Top(10).Collect(context.Background())
		}
	})
}

// BenchmarkCacheSize - Performance with different cache sizes
func BenchmarkCacheSize(b *testing.B) {
	testMetadata := &traverse.Metadata{
		Version:   "1.0",
		Namespace: "test",
		EntityTypes: []traverse.EntityType{
			{Name: "Product", Key: []traverse.PropertyRef{{Name: "ID"}}},
		},
	}

	sizes := []int{100, 1000, 10000}

	for _, size := range sizes {
		b.Run(string(rune('a'+byte(len(sizes)))), func(b *testing.B) {
			cache := memory.New(10 * time.Minute)

			// Pre-populate
			for i := 0; i < size; i++ {
				key := "metadata_" + string(rune(i))
				cache.Set(key, testMetadata)
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				key := "metadata_" + string(rune(i%size))
				_, _ = cache.Get(key)
			}
		})
	}
}
