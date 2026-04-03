package benchmarks

import (
	"context"
	"testing"
	"time"

	"github.com/jhonsferg/traverse"
)

// BenchmarkErrorPathPerformance tests performance of error handling paths
func BenchmarkErrorPathPerformance(b *testing.B) {
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

	b.Run("SuccessfulQuery", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, _ = client.From("Products").Collect(context.Background())
		}
	})

	b.Run("MalformedFilter", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, _ = client.From("Products").
				Filter("NOT[A]{VALID}FILTER").
				Collect(context.Background())
		}
	})

	b.Run("NonexistentEntity", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, _ = client.From("NonexistentEntity123").
				Collect(context.Background())
		}
	})

	b.Run("TimeoutContext", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
			_, _ = client.From("Products").Collect(ctx)
			cancel()
		}
	})

	b.Run("CancelledContext", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			_, _ = client.From("Products").Collect(ctx)
		}
	})
}

// BenchmarkErrorRecovery tests error recovery mechanisms
func BenchmarkErrorRecovery(b *testing.B) {
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

	b.Run("RetryOnError", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			var result interface{}
			var err error

			// Retry up to 3 times
			for attempt := 0; attempt < 3; attempt++ {
				result, err = client.From("Products").Collect(context.Background())
				if err == nil {
					_ = result
					break
				}
			}
		}
	})

	b.Run("FallbackStrategy", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			// Try main query
			result, err := client.From("Products").
				Filter("Price gt 10").
				Collect(context.Background())

			if err != nil {
				// Fall back to simpler query
				result, _ = client.From("Products").
					Top(100).
					Collect(context.Background())
			}
			_ = result
		}
	})
}

// BenchmarkEdgeCases tests performance with edge case data
func BenchmarkEdgeCases(b *testing.B) {
	server := NewMockODataServer(ServerConfig{
		Latency:         5 * time.Millisecond,
		DefaultPageSize: 100,
		MaxRecords:      0, // Empty dataset
		RecordSize:      RecordSizeSmall,
	})
	defer server.Close()

	client, _ := traverse.New(
		traverse.WithBaseURL(server.URL),
		traverse.WithODataVersion(traverse.ODataV4),
	)

	b.Run("EmptyDataset", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, _ = client.From("Products").Collect(context.Background())
		}
	})

	b.Run("CountEmptyDataset", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, _ = client.From("Products").Count(context.Background())
		}
	})

	b.Run("StreamEmptyDataset", func(b *testing.B) {
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

// BenchmarkPanicRecovery tests panic handling (if applicable)
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

	b.Run("ImmediateCancellation", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			_, _ = client.From("Products").Collect(ctx)
		}
	})

	b.Run("VeryShortTimeout", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
			_, _ = client.From("Products").Collect(ctx)
			cancel()
		}
	})

	b.Run("LongTimeout", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			_, _ = client.From("Products").Top(10).Collect(ctx)
			cancel()
		}
	})
}
