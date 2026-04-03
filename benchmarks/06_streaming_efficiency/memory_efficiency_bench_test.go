package benchmarks

import (
	"context"
	"testing"
	"time"

	"github.com/jhonsferg/traverse"
)

// BenchmarkMemoryAllocationPattern analyzes memory allocation patterns
func BenchmarkMemoryAllocationPattern(b *testing.B) {
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

	b.Run("SingleSmallQuery", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, _ = client.From("Products").Top(10).Collect(context.Background())
		}
	})

	b.Run("SingleMediumQuery", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, _ = client.From("Products").Top(1000).Collect(context.Background())
		}
	})

	b.Run("SingleLargeQuery", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, _ = client.From("Products").Collect(context.Background())
		}
	})

	b.Run("StreamSmall", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			count := 0
			for result := range client.From("Products").Stream(context.Background()) {
				if result.Err != nil {
					break
				}
				count++
				if count >= 100 {
					break
				}
			}
		}
	})

	b.Run("StreamLarge", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			for result := range client.From("Products").Stream(context.Background()) {
				if result.Err != nil {
					break
				}
			}
		}
	})
}

// BenchmarkMemoryByDataSize tests memory usage scaling with dataset size
func BenchmarkMemoryByDataSize(b *testing.B) {
	sizes := []int{100, 1000, 10000, 100000}

	for _, maxRecs := range sizes {
		b.Run(string(rune(maxRecs/1000)), func(b *testing.B) {
			server := NewMockODataServer(ServerConfig{
				Latency:         5 * time.Millisecond,
				DefaultPageSize: 100,
				MaxRecords:      maxRecs,
				RecordSize:      RecordSizeSmall,
			})
			defer server.Close()

			client, _ := traverse.New(
				traverse.WithBaseURL(server.URL),
				traverse.WithODataVersion(traverse.ODataV4),
			)

			b.Run("Collect", func(b *testing.B) {
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					_, _ = client.From("Products").Collect(context.Background())
				}
			})

			b.Run("Stream", func(b *testing.B) {
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					for result := range client.From("Products").Stream(context.Background()) {
						if result.Err != nil {
							break
						}
					}
				}
			})
		})
	}
}

// BenchmarkMemoryPayloadSize tests allocation impact of payload size
func BenchmarkMemoryPayloadSize(b *testing.B) {
	payloadSizes := map[string]RecordSize{
		"Small":  RecordSizeSmall,
		"Medium": RecordSizeMedium,
		"Large":  RecordSizeLarge,
	}

	for name, size := range payloadSizes {
		b.Run(name, func(b *testing.B) {
			server := NewMockODataServer(ServerConfig{
				Latency:         5 * time.Millisecond,
				DefaultPageSize: 100,
				MaxRecords:      10000,
				RecordSize:      size,
			})
			defer server.Close()

			client, _ := traverse.New(
				traverse.WithBaseURL(server.URL),
				traverse.WithODataVersion(traverse.ODataV4),
			)

			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_, _ = client.From("Products").Collect(context.Background())
			}
		})
	}
}

// BenchmarkGCPressure measures garbage collection impact
func BenchmarkGCPressure(b *testing.B) {
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

	// Cache queries with many small allocations
	b.Run("ManySmallAllocations", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			for j := 0; j < 100; j++ {
				_, _ = client.From("Products").
					Top(10).
					Collect(context.Background())
			}
		}
	})

	// Few large allocations
	b.Run("FewLargeAllocations", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, _ = client.From("Products").
				Top(100000).
				Collect(context.Background())
		}
	})

	// Streaming with constant memory
	b.Run("StreamingConstantMemory", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			count := 0
			for result := range client.From("Products").Stream(context.Background()) {
				if result.Err != nil {
					break
				}
				count++
				// Process one at a time without accumulating
				_ = result.Value
			}
		}
	})
}
