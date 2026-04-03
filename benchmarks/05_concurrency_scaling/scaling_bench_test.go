package benchmarks

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/jhonsferg/traverse"
)

// BenchmarkScalingHorizontal tests horizontal scaling (multiple instances simulated)
func BenchmarkScalingHorizontal(b *testing.B) {
	instances := []int{1, 2, 4, 8}

	for _, numInstances := range instances {
		b.Run(fmt.Sprintf("Instances%d", numInstances), func(b *testing.B) {
			servers := make([]*MockODataServer, numInstances)
			clients := make([]*traverse.Client, numInstances)

			// Create instances
			for i := 0; i < numInstances; i++ {
				servers[i] = NewMockODataServer(ServerConfig{
					Latency:         5 * time.Millisecond,
					DefaultPageSize: 100,
					MaxRecords:      10000,
					RecordSize:      RecordSizeSmall,
				})
				clients[i], _ = traverse.New(
					traverse.WithBaseURL(servers[i].URL),
					traverse.WithODataVersion(traverse.ODataV4),
				)
			}

			b.ResetTimer()
			b.ReportAllocs()

			// Distribute queries across instances
			for i := 0; i < b.N; i++ {
				var wg sync.WaitGroup
				for j := 0; j < 10; j++ {
					wg.Add(1)
					go func(instanceIdx, queryIdx int) {
						defer wg.Done()
						_, _ = clients[instanceIdx%numInstances].
							From("Products").
							Top(100).
							Collect(context.Background())
					}(i, j)
				}
				wg.Wait()
			}

			// Cleanup
			for i := 0; i < numInstances; i++ {
				servers[i].Close()
				clients[i].Close()
			}
		})
	}
}

// BenchmarkScalingDataset tests performance with different data sizes
func BenchmarkScalingDataset(b *testing.B) {
	sizes := map[string]int{
		"Small":  1000,
		"Medium": 10000,
		"Large":  100000,
	}

	for name, maxRecords := range sizes {
		b.Run(name, func(b *testing.B) {
			server := NewMockODataServer(ServerConfig{
				Latency:         5 * time.Millisecond,
				DefaultPageSize: 100,
				MaxRecords:      maxRecords,
				RecordSize:      RecordSizeSmall,
			})
			defer server.Close()

			client, _ := traverse.New(
				traverse.WithBaseURL(server.URL),
				traverse.WithODataVersion(traverse.ODataV4),
			)

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				_, _ = client.From("Products").Collect(context.Background())
			}
		})
	}
}

// BenchmarkScalingByField tests scaling with different field counts
func BenchmarkScalingByField(b *testing.B) {
	fieldCounts := map[string]int{
		"Few":    3,
		"Medium": 10,
		"Many":   50,
		"All":    100,
	}

	for name, fieldCount := range fieldCounts {
		b.Run(name, func(b *testing.B) {
			server := NewMockODataServer(ServerConfig{
				Latency:         5 * time.Millisecond,
				DefaultPageSize: 100,
				MaxRecords:      1000,
				RecordSize:      RecordSizeLarge, // Large payloads with many fields
			})
			defer server.Close()

			client, _ := traverse.New(
				traverse.WithBaseURL(server.URL),
				traverse.WithODataVersion(traverse.ODataV4),
			)

			// Generate field names
			fields := make([]string, fieldCount)
			for i := 0; i < fieldCount; i++ {
				fields[i] = fmt.Sprintf("Field%d", i)
			}

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				if fieldCount <= 10 {
					_, _ = client.From("Products").
						Select(fields...).
						Collect(context.Background())
				} else {
					// For many fields, don't select (simulate full fetch)
					_, _ = client.From("Products").Collect(context.Background())
				}
			}
		})
	}
}

// BenchmarkScalingByConcurrency tests different concurrency levels
func BenchmarkScalingByConcurrency(b *testing.B) {
	concurrencyLevels := []int{1, 2, 4, 8, 16, 32}

	for _, concurrency := range concurrencyLevels {
		b.Run(fmt.Sprintf("Concurrency%d", concurrency), func(b *testing.B) {
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
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				var wg sync.WaitGroup
				sem := make(chan struct{}, concurrency)

				for g := 0; g < concurrency; g++ {
					wg.Add(1)
					go func() {
						defer wg.Done()
						sem <- struct{}{}
						defer func() { <-sem }()

						_, _ = client.From("Products").Collect(context.Background())
					}()
				}
				wg.Wait()
			}
		})
	}
}
