package benchmarks

import (
	"context"
	"testing"
	"time"

	"github.com/jhonsferg/traverse"
)

// BenchmarkFilterComplexityImpact tests how filter expression complexity affects performance
func BenchmarkFilterComplexityImpact(b *testing.B) {
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

	filters := map[string]string{
		"Empty":          "",
		"Single":         "Price gt 10",
		"TwoAnd":         "Price gt 10 and Status eq 'Active'",
		"TwoOr":          "Price gt 10 or Rating ge 4",
		"ThreeAnd":       "Price gt 10 and Status eq 'Active' and InStock eq true",
		"ThreeOrMixed":   "Price gt 10 and Status eq 'Active' or Rating ge 4",
		"NestedParens":   "(Price gt 10 and Status eq 'Active') or (Rating ge 4 and InStock eq true)",
		"ComplexNested":  "((Price gt 10 or Price lt 100) and (Status eq 'Active' or Status eq 'Pending')) and (Rating ge 4 or Rating eq 0)",
		"VeryComplex":    "(A eq 1 and B eq 2) or (C eq 3 and D eq 4) or (E eq 5 and F eq 6) and (G eq 7 or H eq 8)",
		"MultiCondition": "A eq 1 and B eq 2 and C eq 3 and D eq 4 and E eq 5 and F eq 6 and G eq 7 and H eq 8",
	}

	for name, filter := range filters {
		b.Run(name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				query := client.From("Products")
				if filter != "" {
					query = query.Filter(filter)
				}
				_, _ = query.Collect(context.Background())
			}
		})
	}
}

// BenchmarkFilterVsNoFilter isolates filter overhead
func BenchmarkFilterVsNoFilter(b *testing.B) {
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

	b.Run("NoFilter", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, _ = client.From("Products").Collect(context.Background())
		}
	})

	b.Run("WithFilter", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, _ = client.From("Products").Filter("Price gt 10").Collect(context.Background())
		}
	})
}

// BenchmarkFilterBuilding isolates filter expression building time
func BenchmarkFilterBuilding(b *testing.B) {
	filters := []string{
		"Price gt 10",
		"Price gt 10 and Status eq 'Active'",
		"(Price gt 10 and Status eq 'Active') or (Rating ge 4 and InStock eq true)",
		"((A eq 1 and B eq 2) or (C eq 3 and D eq 4)) and (E eq 5 or F eq 6)",
	}

	for i, filter := range filters {
		b.Run(string(rune(i)), func(b *testing.B) {
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
			for j := 0; j < b.N; j++ {
				_ = client.From("Products").Filter(filter)
			}
		})
	}
}
