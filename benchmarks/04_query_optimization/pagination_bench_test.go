package benchmarks

import (
	"context"
	"testing"
	"time"

	"github.com/jhonsferg/traverse"
)

// BenchmarkPaginationStrategies compares different pagination approaches
func BenchmarkPaginationStrategies(b *testing.B) {
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

	scenarios := map[string]struct {
		top  int
		skip int
	}{
		"Top10Skip0":      {10, 0},
		"Top50Skip0":      {50, 0},
		"Top100Skip0":     {100, 0},
		"Top1000Skip0":    {1000, 0},
		"Top100Skip100":   {100, 100},
		"Top100Skip1000":  {100, 1000},
		"Top100Skip5000":  {100, 5000},
		"Top100Skip10000": {100, 10000},
		"Top100Skip50000": {100, 50000},
	}

	for name, params := range scenarios {
		b.Run(name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_, _ = client.From("Products").
					Top(params.top).
					Skip(params.skip).
					Collect(context.Background())
			}
		})
	}
}

// BenchmarkPageSizeOptimization tests different page sizes for efficiency
func BenchmarkPageSizeOptimization(b *testing.B) {
	pageSizes := []int{10, 50, 100, 500, 1000, 5000}

	for _, pageSize := range pageSizes {
		b.Run(string(rune(pageSize)), func(b *testing.B) {
			server := NewMockODataServer(ServerConfig{
				Latency:         5 * time.Millisecond,
				DefaultPageSize: pageSize,
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
				_, _ = client.From("Products").Collect(context.Background())
			}
		})
	}
}

// BenchmarkSkipPerformance isolates skip/top overhead
func BenchmarkSkipPerformance(b *testing.B) {
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

	b.Run("NoSkip", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, _ = client.From("Products").Top(100).Collect(context.Background())
		}
	})

	b.Run("Skip100", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, _ = client.From("Products").Top(100).Skip(100).Collect(context.Background())
		}
	})

	b.Run("Skip1000", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, _ = client.From("Products").Top(100).Skip(1000).Collect(context.Background())
		}
	})

	b.Run("Skip10000", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, _ = client.From("Products").Top(100).Skip(10000).Collect(context.Background())
		}
	})
}

// BenchmarkSelectFieldOptimization tests field selection efficiency
func BenchmarkSelectFieldOptimization(b *testing.B) {
	server := NewMockODataServer(ServerConfig{
		Latency:         5 * time.Millisecond,
		DefaultPageSize: 100,
		MaxRecords:      1000,
		RecordSize:      RecordSizeLarge, // Large payloads benefit more from Select
	})
	defer server.Close()

	client, _ := traverse.New(
		traverse.WithBaseURL(server.URL),
		traverse.WithODataVersion(traverse.ODataV4),
	)

	fieldSets := map[string][]string{
		"All":     {},
		"1Field":  {"Name"},
		"3Fields": {"Name", "Price", "Status"},
		"5Fields": {"Name", "Price", "Status", "Rating", "InStock"},
		"8Fields": {"Name", "Price", "Status", "Rating", "InStock", "CreatedAt", "UpdatedAt", "Category"},
	}

	for name, fields := range fieldSets {
		b.Run(name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				query := client.From("Products")
				if len(fields) > 0 {
					query = query.Select(fields...)
				}
				_, _ = query.Collect(context.Background())
			}
		})
	}
}

// BenchmarkCountQueryVsFullQuery compares count vs full data retrieval
func BenchmarkCountQueryVsFullQuery(b *testing.B) {
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

	b.Run("CountOnly", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, _ = client.From("Products").Count(context.Background())
		}
	})

	b.Run("FullQuery", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, _ = client.From("Products").Collect(context.Background())
		}
	})

	b.Run("CountFirst", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			count, _ := client.From("Products").Count(context.Background())
			if count > 0 {
				_, _ = client.From("Products").Top(100).Collect(context.Background())
			}
		}
	})
}
