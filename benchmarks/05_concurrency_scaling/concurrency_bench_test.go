package benchmarks

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jhonsferg/traverse"
)

// BenchmarkConcurrentSameQuery - Multiple goroutines executing same query
func BenchmarkConcurrentSameQuery(b *testing.B) {
	server := NewMockODataServer(ServerConfig{
		Latency:         5 * time.Millisecond,
		DefaultPageSize: 100,
		MaxRecords:      1000,
		RecordSize:      RecordSizeSmall,
	})
	defer server.Close()

	client, _ := traverse.New(
		traverse.WithBaseURL(server.URL),
		traverse.WithODataVersion(traverse.ODataV4),
	)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = client.From("Products").Top(10).Collect(context.Background())
		}
	})
}

// BenchmarkConcurrentDifferentQueries - Multiple goroutines with different queries
func BenchmarkConcurrentDifferentQueries(b *testing.B) {
	server := NewMockODataServer(ServerConfig{
		Latency:         5 * time.Millisecond,
		DefaultPageSize: 100,
		MaxRecords:      1000,
		RecordSize:      RecordSizeSmall,
	})
	defer server.Close()

	client, _ := traverse.New(
		traverse.WithBaseURL(server.URL),
		traverse.WithODataVersion(traverse.ODataV4),
	)

	b.ResetTimer()
	counter := int64(0)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			idx := atomic.AddInt64(&counter, 1)
			skip := int((idx % 100) * 10)

			_, _ = client.From("Products").
				Top(10).
				Skip(skip).
				Collect(context.Background())
		}
	})
}

// BenchmarkConcurrentStreamAndCollect - Mix of streaming and collecting in parallel
func BenchmarkConcurrentStreamAndCollect(b *testing.B) {
	server := NewMockODataServer(ServerConfig{
		Latency:         5 * time.Millisecond,
		DefaultPageSize: 100,
		MaxRecords:      1000,
		RecordSize:      RecordSizeSmall,
	})
	defer server.Close()

	client, _ := traverse.New(
		traverse.WithBaseURL(server.URL),
		traverse.WithODataVersion(traverse.ODataV4),
	)

	b.ResetTimer()
	counter := int64(0)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			idx := atomic.AddInt64(&counter, 1)

			if idx%2 == 0 {
				// Stream
				for range client.From("Products").Stream(context.Background()) {
					// Consume stream
				}
			} else {
				// Collect
				_, _ = client.From("Products").Top(10).Collect(context.Background())
			}
		}
	})
}

// BenchmarkConcurrentMetadataAccess - Concurrent metadata access pattern
func BenchmarkConcurrentMetadataAccess(b *testing.B) {
	server := NewMockODataServer(ServerConfig{
		Latency:         5 * time.Millisecond,
		DefaultPageSize: 100,
		MaxRecords:      1000,
		RecordSize:      RecordSizeSmall,
	})
	defer server.Close()

	cache := &traverse.NoOpCache{}
	client, _ := traverse.New(
		traverse.WithBaseURL(server.URL),
		traverse.WithODataVersion(traverse.ODataV4),
		traverse.WithMetadataCache(cache),
	)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = client.Metadata(context.Background())
		}
	})
}

// BenchmarkConcurrentMixedOperations - Real-world mix: metadata, count, query, stream
func BenchmarkConcurrentMixedOperations(b *testing.B) {
	server := NewMockODataServer(ServerConfig{
		Latency:         5 * time.Millisecond,
		DefaultPageSize: 100,
		MaxRecords:      5000,
		RecordSize:      RecordSizeSmall,
	})
	defer server.Close()

	cache := &traverse.NoOpCache{}
	client, _ := traverse.New(
		traverse.WithBaseURL(server.URL),
		traverse.WithODataVersion(traverse.ODataV4),
		traverse.WithMetadataCache(cache),
	)

	b.ResetTimer()
	counter := int64(0)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			idx := atomic.AddInt64(&counter, 1) % 4

			switch idx {
			case 0:
				// Metadata
				_, _ = client.Metadata(context.Background())
			case 1:
				// Count
				_, _ = client.From("Products").Count(context.Background())
			case 2:
				// Query
				_, _ = client.From("Products").Top(50).Collect(context.Background())
			case 3:
				// Stream (consume first item only)
				for result := range client.From("Products").Stream(context.Background()) {
					if result.Err != nil {
						break
					}
					break // Just consume one item
				}
			}
		}
	})
}

// BenchmarkConcurrentCount - Concurrent count operations
func BenchmarkConcurrentCount(b *testing.B) {
	server := NewMockODataServer(ServerConfig{
		Latency:         5 * time.Millisecond,
		DefaultPageSize: 100,
		MaxRecords:      10000,
		RecordSize:      RecordSizeSmall,
	})
	defer server.Close()

	client, _ := traverse.New(
		traverse.WithBaseURL(server.URL),
		traverse.WithODataVersion(traverse.ODataV4),
	)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = client.From("Products").Count(context.Background())
		}
	})
}

// BenchmarkConcurrentFiltered - Concurrent queries with filters
func BenchmarkConcurrentFiltered(b *testing.B) {
	server := NewMockODataServer(ServerConfig{
		Latency:         5 * time.Millisecond,
		DefaultPageSize: 100,
		MaxRecords:      10000,
		RecordSize:      RecordSizeSmall,
	})
	defer server.Close()

	client, _ := traverse.New(
		traverse.WithBaseURL(server.URL),
		traverse.WithODataVersion(traverse.ODataV4),
	)

	b.ResetTimer()
	counter := int64(0)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			idx := atomic.AddInt64(&counter, 1)
			filter := ""

			switch idx % 3 {
			case 0:
				filter = "Price gt 10"
			case 1:
				filter = "Rating eq 4"
			case 2:
				filter = "Name ne 'Discontinued'"
			}

			_, _ = client.From("Products").
				Filter(filter).
				Top(10).
				Collect(context.Background())
		}
	})
}

// BenchmarkContextCancellation - Goroutines with context cancellation
func BenchmarkContextCancellation(b *testing.B) {
	server := NewMockODataServer(ServerConfig{
		Latency:         5 * time.Millisecond,
		DefaultPageSize: 100,
		MaxRecords:      100000,
		RecordSize:      RecordSizeSmall,
	})
	defer server.Close()

	client, _ := traverse.New(
		traverse.WithBaseURL(server.URL),
		traverse.WithODataVersion(traverse.ODataV4),
	)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)

		for range client.From("Products").Stream(ctx) {
			// Just consume until cancelled
		}

		cancel()
	}
}

// BenchmarkConcurrentRateLimited - Goroutines with rate limiting
func BenchmarkConcurrentRateLimited(b *testing.B) {
	server := NewMockODataServer(ServerConfig{
		Latency:         5 * time.Millisecond,
		DefaultPageSize: 100,
		MaxRecords:      1000,
		RecordSize:      RecordSizeSmall,
	})
	defer server.Close()

	client, _ := traverse.New(
		traverse.WithBaseURL(server.URL),
		traverse.WithODataVersion(traverse.ODataV4),
	)

	// Rate limiter: 10 concurrent operations max
	semaphore := make(chan struct{}, 10)

	b.ResetTimer()
	var wg sync.WaitGroup
	for i := 0; i < b.N; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			_, _ = client.From("Products").Top(10).Collect(context.Background())
		}()
	}
	wg.Wait()
}

// BenchmarkGoroutineScaling - Test scaling with increasing goroutine counts
func BenchmarkGoroutineScaling(b *testing.B) {
	server := NewMockODataServer(ServerConfig{
		Latency:         5 * time.Millisecond,
		DefaultPageSize: 100,
		MaxRecords:      1000,
		RecordSize:      RecordSizeSmall,
	})
	defer server.Close()

	client, _ := traverse.New(
		traverse.WithBaseURL(server.URL),
		traverse.WithODataVersion(traverse.ODataV4),
	)

	goroutineCounts := []int{1, 2, 4, 8, 16}

	for _, count := range goroutineCounts {
		b.Run(fmt.Sprintf("concurrency_%d", count), func(b *testing.B) {
			var wg sync.WaitGroup

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				for j := 0; j < count; j++ {
					wg.Add(1)
					go func() {
						defer wg.Done()
						_, _ = client.From("Products").Top(10).Collect(context.Background())
					}()
				}
				wg.Wait()
			}
		})
	}
}
