package benchmarks

import (
	"context"
	"testing"
	"time"

	"github.com/jhonsferg/traverse"
)

// BenchmarkCollectionMethods compares different collection approaches
func BenchmarkCollectionMethods(b *testing.B) {
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

	b.Run("CollectAll", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, _ = client.From("Products").Collect(context.Background())
		}
	})

	b.Run("CollectWithFilter", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, _ = client.From("Products").
				Filter("Price gt 10").
				Collect(context.Background())
		}
	})

	b.Run("CollectWithSelect", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, _ = client.From("Products").
				Select("Name", "Price", "Status").
				Collect(context.Background())
		}
	})

	b.Run("CollectWithTopSkip", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, _ = client.From("Products").
				Top(100).
				Skip(50).
				Collect(context.Background())
		}
	})

	b.Run("CollectFirst", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, _ = client.From("Products").
				Top(1).
				Collect(context.Background())
		}
	})
}

// BenchmarkCollectionVsFirst compares different "first record" strategies
func BenchmarkCollectionVsFirst(b *testing.B) {
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

	// Get first record by collecting and indexing
	b.Run("CollectThenIndex", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			results, _ := client.From("Products").Top(1).Collect(context.Background())
			if len(results) > 0 {
				_ = results[0]
			}
		}
	})

	// Get first via streaming
	b.Run("StreamFirst", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			for result := range client.From("Products").Top(1).Stream(context.Background()) {
				if result.Err == nil {
					_ = result.Value
					break
				}
			}
		}
	})
}

// BenchmarkPaginatedCollection tests paginated collection approach
func BenchmarkPaginatedCollection(b *testing.B) {
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

	b.Run("SinglePageSmall", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, _ = client.From("Products").
				Top(100).
				Collect(context.Background())
		}
	})

	b.Run("MultiPageSequential", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			for page := 0; page < 10; page++ {
				_, _ = client.From("Products").
					Top(100).
					Skip(page * 100).
					Collect(context.Background())
			}
		}
	})

	b.Run("CountThenPagedCollection", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			count, _ := client.From("Products").Count(context.Background())
			pageCount := (count + 99) / 100

			for page := int64(0); page < pageCount && page < 10; page++ {
				_, _ = client.From("Products").
					Top(100).
					Skip(int(page * 100)).
					Collect(context.Background())
			}
		}
	})
}

// BenchmarkErrorInCollection tests error handling during collection
func BenchmarkErrorInCollection(b *testing.B) {
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

	b.Run("SuccessfulCollection", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, _ = client.From("Products").Collect(context.Background())
		}
	})

	b.Run("CollectionWithInvalidFilter", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, _ = client.From("Products").
				Filter("InvalidFilter}}{").
				Collect(context.Background())
		}
	})
}
