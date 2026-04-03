package benchmarks

import (
	"context"
	"testing"
	"time"

	"github.com/jhonsferg/traverse"
)

// BenchmarkSimpleQuery - Basic query
func BenchmarkSimpleQuery(b *testing.B) {
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
	for i := 0; i < b.N; i++ {
		_, _ = client.From("Products").Top(10).Collect(context.Background())
	}
}

// BenchmarkFilteredQuery - Single filter on Name
func BenchmarkFilteredQuery(b *testing.B) {
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
	for i := 0; i < b.N; i++ {
		_, _ = client.From("Products").
			Filter("Name eq 'Product 1'").
			Top(10).
			Collect(context.Background())
	}
}

// BenchmarkComplexFilteredQuery - Multiple AND/OR filters
func BenchmarkComplexFilteredQuery(b *testing.B) {
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
	for i := 0; i < b.N; i++ {
		_, _ = client.From("Products").
			Filter("(Price gt 10 and Price lt 100) or Rating eq 5").
			Top(10).
			Collect(context.Background())
	}
}

// BenchmarkOrderByQuery - Order by single field
func BenchmarkOrderByQuery(b *testing.B) {
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
	for i := 0; i < b.N; i++ {
		_, _ = client.From("Products").
			OrderBy("Name").
			Top(10).
			Collect(context.Background())
	}
}

// BenchmarkMultiFieldOrderBy - Order by multiple fields
func BenchmarkMultiFieldOrderBy(b *testing.B) {
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
	for i := 0; i < b.N; i++ {
		_, _ = client.From("Products").
			OrderByDesc("Rating").
			OrderBy("Price").
			OrderBy("Name").
			Top(10).
			Collect(context.Background())
	}
}

// BenchmarkSelectQuery - Select subset of fields
func BenchmarkSelectQuery(b *testing.B) {
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
	for i := 0; i < b.N; i++ {
		_, _ = client.From("Products").
			Select("ID", "Name", "Price").
			Top(10).
			Collect(context.Background())
	}
}

// BenchmarkFullyQualifiedQuery - Complex query with all options
func BenchmarkFullyQualifiedQuery(b *testing.B) {
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
	for i := 0; i < b.N; i++ {
		_, _ = client.From("Products").
			Filter("Price gt 10 and Price lt 100").
			Select("ID", "Name", "Price", "Rating").
			OrderByDesc("Price").
			Top(50).
			Skip(0).
			Collect(context.Background())
	}
}

// BenchmarkCountVsFullQuery - Comparison of count vs full query
func BenchmarkCountVsFullQuery(b *testing.B) {
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

	b.Run("Count", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = client.From("Products").Count(context.Background())
		}
	})

	b.Run("FullQuery", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = client.From("Products").Collect(context.Background())
		}
	})
}

// BenchmarkPaginationOverhead - Overhead of pagination operations
func BenchmarkPaginationOverhead(b *testing.B) {
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

	b.Run("FirstPage", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = client.From("Products").
				Top(100).
				Skip(0).
				Collect(context.Background())
		}
	})

	b.Run("MiddlePage", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = client.From("Products").
				Top(100).
				Skip(5000).
				Collect(context.Background())
		}
	})

	b.Run("LastPage", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = client.From("Products").
				Top(100).
				Skip(9900).
				Collect(context.Background())
		}
	})
}

// BenchmarkResponseParsing - Time spent parsing different response sizes
func BenchmarkResponseParsing(b *testing.B) {
	b.Run("Small", func(b *testing.B) {
		server := NewMockODataServer(ServerConfig{
			Latency:         1 * time.Millisecond,
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
		for i := 0; i < b.N; i++ {
			_, _ = client.From("Products").Top(100).Collect(context.Background())
		}
	})

	b.Run("Medium", func(b *testing.B) {
		server := NewMockODataServer(ServerConfig{
			Latency:         1 * time.Millisecond,
			DefaultPageSize: 100,
			MaxRecords:      1000,
			RecordSize:      RecordSizeMedium,
		})
		defer server.Close()

		client, _ := traverse.New(
			traverse.WithBaseURL(server.URL),
			traverse.WithODataVersion(traverse.ODataV4),
		)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = client.From("Products").Top(100).Collect(context.Background())
		}
	})

	b.Run("Large", func(b *testing.B) {
		server := NewMockODataServer(ServerConfig{
			Latency:         1 * time.Millisecond,
			DefaultPageSize: 100,
			MaxRecords:      1000,
			RecordSize:      RecordSizeLarge,
		})
		defer server.Close()

		client, _ := traverse.New(
			traverse.WithBaseURL(server.URL),
			traverse.WithODataVersion(traverse.ODataV4),
		)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = client.From("Products").Top(100).Collect(context.Background())
		}
	})
}

// BenchmarkSkipWithPagination - Different skip values
func BenchmarkSkipWithPagination(b *testing.B) {
	server := NewMockODataServer(ServerConfig{
		Latency:         5 * time.Millisecond,
		DefaultPageSize: 100,
		MaxRecords:      50000,
		RecordSize:      RecordSizeSmall,
	})
	defer server.Close()

	client, _ := traverse.New(
		traverse.WithBaseURL(server.URL),
		traverse.WithODataVersion(traverse.ODataV4),
	)

	b.Run("Skip0", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = client.From("Products").Skip(0).Top(100).Collect(context.Background())
		}
	})

	b.Run("Skip5000", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = client.From("Products").Skip(5000).Top(100).Collect(context.Background())
		}
	})

	b.Run("Skip20000", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = client.From("Products").Skip(20000).Top(100).Collect(context.Background())
		}
	})
}
