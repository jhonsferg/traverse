package benchmarks

import (
	"context"
	"testing"
	"time"

	"github.com/jhonsferg/traverse"
)

// BenchmarkStreamingSmall - Stream 10K records
func BenchmarkStreamingSmall(b *testing.B) {
	server := NewMockODataServer(ServerConfig{
		Latency:         1 * time.Millisecond,
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
	for i := 0; i < b.N; i++ {
		count := 0
		for result := range client.From("Products").Stream(context.Background()) {
			if result.Err != nil {
				b.Fatalf("Stream error: %v", result.Err)
			}
			count++
		}

		if count != 10000 {
			b.Fatalf("Expected 10000 records, got %d", count)
		}
	}
}

// BenchmarkStreamingMedium - Stream 100K records
func BenchmarkStreamingMedium(b *testing.B) {
	server := NewMockODataServer(ServerConfig{
		Latency:         1 * time.Millisecond,
		DefaultPageSize: 100,
		MaxRecords:      100000,
		RecordSize:      RecordSizeMedium,
	})
	defer server.Close()

	client, _ := traverse.New(
		traverse.WithBaseURL(server.URL),
		traverse.WithODataVersion(traverse.ODataV4),
	)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		count := 0
		for result := range client.From("Products").Stream(context.Background()) {
			if result.Err != nil {
				b.Fatalf("Stream error: %v", result.Err)
			}
			count++
		}

		if count != 100000 {
			b.Fatalf("Expected 100000 records, got %d", count)
		}
	}
}

// BenchmarkCollectSmall - Collect 10K records into memory
func BenchmarkCollectSmall(b *testing.B) {
	server := NewMockODataServer(ServerConfig{
		Latency:         1 * time.Millisecond,
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
	for i := 0; i < b.N; i++ {
		records, err := client.From("Products").Collect(context.Background())

		if err != nil {
			b.Fatalf("Collect failed: %v", err)
		}

		if len(records) != 10000 {
			b.Fatalf("Expected 10000 records, got %d", len(records))
		}
	}
}

// BenchmarkCollectMedium - Collect 100K records into memory
func BenchmarkCollectMedium(b *testing.B) {
	server := NewMockODataServer(ServerConfig{
		Latency:         1 * time.Millisecond,
		DefaultPageSize: 100,
		MaxRecords:      100000,
		RecordSize:      RecordSizeMedium,
	})
	defer server.Close()

	client, _ := traverse.New(
		traverse.WithBaseURL(server.URL),
		traverse.WithODataVersion(traverse.ODataV4),
	)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		records, err := client.From("Products").Collect(context.Background())

		if err != nil {
			b.Fatalf("Collect failed: %v", err)
		}

		if len(records) != 100000 {
			b.Fatalf("Expected 100000 records, got %d", len(records))
		}
	}
}

// BenchmarkCountQuery - Count operation efficiency
func BenchmarkCountQuery(b *testing.B) {
	server := NewMockODataServer(ServerConfig{
		Latency:         1 * time.Millisecond,
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
		count, err := client.From("Products").Count(context.Background())
		if err != nil {
			b.Fatalf("Count failed: %v", err)
		}

		if count != 100000 {
			b.Fatalf("Expected 100000, got %d", count)
		}
	}
}

// BenchmarkPaginationSmallPages - Small page size (10 records per page)
func BenchmarkPaginationSmallPages(b *testing.B) {
	server := NewMockODataServer(ServerConfig{
		Latency:         1 * time.Millisecond,
		DefaultPageSize: 10,
		MaxRecords:      100,
		RecordSize:      RecordSizeSmall,
	})
	defer server.Close()

	client, _ := traverse.New(
		traverse.WithBaseURL(server.URL),
		traverse.WithODataVersion(traverse.ODataV4),
	)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		records, err := client.From("Products").Top(10).Collect(context.Background())
		if err != nil {
			b.Fatalf("Collect failed: %v", err)
		}

		if len(records) != 10 {
			b.Fatalf("Expected 10 records, got %d", len(records))
		}
	}
}

// BenchmarkPaginationLargePages - Large page size (1000 records per page)
func BenchmarkPaginationLargePages(b *testing.B) {
	server := NewMockODataServer(ServerConfig{
		Latency:         1 * time.Millisecond,
		DefaultPageSize: 1000,
		MaxRecords:      5000,
		RecordSize:      RecordSizeSmall,
	})
	defer server.Close()

	client, _ := traverse.New(
		traverse.WithBaseURL(server.URL),
		traverse.WithODataVersion(traverse.ODataV4),
	)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		records, err := client.From("Products").Top(1000).Collect(context.Background())
		if err != nil {
			b.Fatalf("Collect failed: %v", err)
		}

		if len(records) != 1000 {
			b.Fatalf("Expected 1000 records, got %d", len(records))
		}
	}
}

// BenchmarkStreamVsCollect - Direct comparison of Stream vs Collect efficiency
func BenchmarkStreamVsCollect(b *testing.B) {
	server := NewMockODataServer(ServerConfig{
		Latency:         1 * time.Millisecond,
		DefaultPageSize: 100,
		MaxRecords:      50000,
		RecordSize:      RecordSizeMedium,
	})
	defer server.Close()

	client, _ := traverse.New(
		traverse.WithBaseURL(server.URL),
		traverse.WithODataVersion(traverse.ODataV4),
	)

	b.Run("Stream", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			count := 0
			for result := range client.From("Products").Stream(context.Background()) {
				if result.Err != nil {
					b.Fatalf("Stream error: %v", result.Err)
				}
				count++
			}
		}
	})

	b.Run("Collect", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := client.From("Products").Collect(context.Background())
			if err != nil {
				b.Fatalf("Collect error: %v", err)
			}
		}
	})
}

// BenchmarkFirstRecord - Get first record efficiency
func BenchmarkFirstRecord(b *testing.B) {
	server := NewMockODataServer(ServerConfig{
		Latency:         1 * time.Millisecond,
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
	for i := 0; i < b.N; i++ {
		record, err := client.From("Products").First(context.Background())
		if err != nil {
			b.Fatalf("First failed: %v", err)
		}

		if record == nil {
			b.Fatal("Expected a record")
		}
	}
}

// BenchmarkPageOperation - Fetch single page
func BenchmarkPageOperation(b *testing.B) {
	server := NewMockODataServer(ServerConfig{
		Latency:         1 * time.Millisecond,
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
	for i := 0; i < b.N; i++ {
		page, err := client.From("Products").Top(100).Page(context.Background())
		if err != nil {
			b.Fatalf("Page failed: %v", err)
		}

		if page == nil || len(page.Value) == 0 {
			b.Fatal("Expected page with records")
		}
	}
}
