package benchmarks

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jhonsferg/traverse"
)

// BenchmarkCacheContention tests cache lock contention under various conditions
func BenchmarkCacheContention(b *testing.B) {
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

	// All goroutines access same metadata (high contention)
	b.Run("SameMetadataHighContention", func(b *testing.B) {
		var wg sync.WaitGroup
		var operations int64

		for i := 0; i < b.N; i++ {
			for g := 0; g < 32; g++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					_, _ = client.Metadata(context.Background())
					atomic.AddInt64(&operations, 1)
				}()
			}
			wg.Wait()
		}
		b.ReportMetric(float64(atomic.LoadInt64(&operations)), "ops")
	})

	// Goroutines access different queries (low contention)
	b.Run("DifferentQueriesLowContention", func(b *testing.B) {
		var wg sync.WaitGroup
		var operations int64

		for i := 0; i < b.N; i++ {
			for g := 0; g < 32; g++ {
				wg.Add(1)
				go func(idx int) {
					defer wg.Done()
					entity := "Entity" + string(rune(idx))
					_, _ = client.From(entity).Collect(context.Background())
					atomic.AddInt64(&operations, 1)
				}(g)
			}
			wg.Wait()
		}
		b.ReportMetric(float64(atomic.LoadInt64(&operations)), "ops")
	})

	// Mixed: reads and writes (medium contention)
	b.Run("MixedReadWriteMediumContention", func(b *testing.B) {
		var wg sync.WaitGroup
		var operations int64

		for i := 0; i < b.N; i++ {
			for g := 0; g < 32; g++ {
				wg.Add(1)
				go func(idx int) {
					defer wg.Done()
					if idx%3 == 0 {
						_, _ = client.Metadata(context.Background())
					} else {
						_, _ = client.From("Products").Collect(context.Background())
					}
					atomic.AddInt64(&operations, 1)
				}(g)
			}
			wg.Wait()
		}
		b.ReportMetric(float64(atomic.LoadInt64(&operations)), "ops")
	})
}

// BenchmarkMutexContention tests mutex behavior directly
func BenchmarkMutexContention(b *testing.B) {
	var mu sync.RWMutex
	data := make(map[string]interface{})

	b.Run("SequentialReads", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			mu.RLock()
			_ = data["key"]
			mu.RUnlock()
		}
	})

	b.Run("ConcurrentReads-4", func(b *testing.B) {
		var wg sync.WaitGroup
		for i := 0; i < b.N; i++ {
			for g := 0; g < 4; g++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					mu.RLock()
					_ = data["key"]
					mu.RUnlock()
				}()
			}
			wg.Wait()
		}
	})

	b.Run("ConcurrentReads-16", func(b *testing.B) {
		var wg sync.WaitGroup
		for i := 0; i < b.N; i++ {
			for g := 0; g < 16; g++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					mu.RLock()
					_ = data["key"]
					mu.RUnlock()
				}()
			}
			wg.Wait()
		}
	})

	b.Run("ConcurrentReads-64", func(b *testing.B) {
		var wg sync.WaitGroup
		for i := 0; i < b.N; i++ {
			for g := 0; g < 64; g++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					mu.RLock()
					_ = data["key"]
					mu.RUnlock()
				}()
			}
			wg.Wait()
		}
	})

	b.Run("SequentialWrites", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			mu.Lock()
			data["key"] = i
			mu.Unlock()
		}
	})
}

// BenchmarkLockContentionVsChannels compares locks vs channels
func BenchmarkLockContentionVsChannels(b *testing.B) {
	b.Run("RWMutex-Read", func(b *testing.B) {
		var mu sync.RWMutex
		data := make(map[string]interface{})

		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			mu.RLock()
			_ = data["key"]
			mu.RUnlock()
		}
	})

	b.Run("Channel-Send", func(b *testing.B) {
		ch := make(chan interface{})

		go func() {
			for range ch {
			}
		}()

		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			ch <- i
		}
	})

	b.Run("Atomic-Load", func(b *testing.B) {
		var counter int64

		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = atomic.LoadInt64(&counter)
		}
	})
}
