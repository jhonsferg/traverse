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

// BenchmarkThroughputBaseline measures baseline throughput with various configurations
func BenchmarkThroughputBaseline(b *testing.B) {
	configs := map[string]ServerConfig{
		"LowLatency_SmallPayload": {
			Latency:         1 * time.Millisecond,
			DefaultPageSize: 100,
			MaxRecords:      10000,
			RecordSize:      RecordSizeSmall,
		},
		"MediumLatency_MediumPayload": {
			Latency:         5 * time.Millisecond,
			DefaultPageSize: 100,
			MaxRecords:      10000,
			RecordSize:      RecordSizeMedium,
		},
		"HighLatency_LargePayload": {
			Latency:         50 * time.Millisecond,
			DefaultPageSize: 100,
			MaxRecords:      10000,
			RecordSize:      RecordSizeLarge,
		},
		"NoLatency_MediumPayload": {
			Latency:         0 * time.Millisecond,
			DefaultPageSize: 100,
			MaxRecords:      10000,
			RecordSize:      RecordSizeMedium,
		},
	}

	for name, config := range configs {
		b.Run(name, func(b *testing.B) {
			server := NewMockODataServer(config)
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

// BenchmarkThroughputByPageSize measures throughput with different page sizes
func BenchmarkThroughputByPageSize(b *testing.B) {
	pageSizes := map[string]int{
		"Page_50":    50,
		"Page_100":   100,
		"Page_500":   500,
		"Page_1000":  1000,
		"Page_5000":  5000,
		"Page_10000": 10000,
	}

	for name, pageSize := range pageSizes {
		b.Run(name, func(b *testing.B) {
			server := NewMockODataServer(ServerConfig{
				Latency:         5 * time.Millisecond,
				DefaultPageSize: pageSize,
				MaxRecords:      100000,
				RecordSize:      RecordSizeMedium,
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

// BenchmarkThroughputSustained measures sustained throughput over time
func BenchmarkThroughputSustained(b *testing.B) {
	durations := map[string]time.Duration{
		"1s":  1 * time.Second,
		"5s":  5 * time.Second,
		"10s": 10 * time.Second,
	}

	for name, duration := range durations {
		b.Run(name, func(b *testing.B) {
			server := NewMockODataServer(ServerConfig{
				Latency:         5 * time.Millisecond,
				DefaultPageSize: 100,
				MaxRecords:      10000,
				RecordSize:      RecordSizeMedium,
			})
			defer server.Close()

			client, _ := traverse.New(
				traverse.WithBaseURL(server.URL),
				traverse.WithODataVersion(traverse.ODataV4),
			)

			b.ResetTimer()
			b.ReportAllocs()

			timeout := time.Now().Add(duration)
			ops := 0
			for time.Now().Before(timeout) {
				_, _ = client.From("Products").Top(100).Collect(context.Background())
				ops++
			}

			b.ReportMetric(float64(ops), "ops")
		})
	}
}

// BenchmarkThroughputPeak measures peak throughput achievable
func BenchmarkThroughputPeak(b *testing.B) {
	concurrencies := []int{1, 2, 4, 8, 16, 32, 64}

	for _, concurrency := range concurrencies {
		b.Run(fmt.Sprintf("Concurrency_%d", concurrency), func(b *testing.B) {
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

						_, _ = client.From("Products").Top(100).Collect(context.Background())
					}()
				}
				wg.Wait()
			}
		})
	}
}

// BenchmarkThroughputDistribution measures throughput across different entity types
func BenchmarkThroughputDistribution(b *testing.B) {
	entities := map[string]string{
		"Products": "Products",
		"Orders":   "Orders",
		"Customers": "Customers",
	}

	for name, entity := range entities {
		b.Run(name, func(b *testing.B) {
			server := NewMockODataServer(ServerConfig{
				Latency:         5 * time.Millisecond,
				DefaultPageSize: 100,
				MaxRecords:      10000,
				RecordSize:      RecordSizeMedium,
			})
			defer server.Close()

			client, _ := traverse.New(
				traverse.WithBaseURL(server.URL),
				traverse.WithODataVersion(traverse.ODataV4),
			)

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				_, _ = client.From(entity).Collect(context.Background())
			}
		})
	}
}

// BenchmarkThroughputPayloadSize measures impact of payload size on throughput
func BenchmarkThroughputPayloadSize(b *testing.B) {
	payloads := map[string]RecordSize{
		"Small":  RecordSizeSmall,
		"Medium": RecordSizeMedium,
		"Large":  RecordSizeLarge,
	}

	for name, recordSize := range payloads {
		b.Run(name, func(b *testing.B) {
			server := NewMockODataServer(ServerConfig{
				Latency:         5 * time.Millisecond,
				DefaultPageSize: 100,
				MaxRecords:      10000,
				RecordSize:      recordSize,
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

// BenchmarkThroughputWithFilters measures throughput impact of filters
func BenchmarkThroughputWithFilters(b *testing.B) {
	filters := map[string]string{
		"NoFilter":       "",
		"SimpleFilter":   "Price gt 100",
		"ComplexFilter":  "Price gt 100 and Status eq 'Active' and Category eq 'Electronics'",
		"MultipleOr":     "Status eq 'Active' or Status eq 'Pending' or Status eq 'Processing'",
	}

	for name, filter := range filters {
		b.Run(name, func(b *testing.B) {
			server := NewMockODataServer(ServerConfig{
				Latency:         5 * time.Millisecond,
				DefaultPageSize: 100,
				MaxRecords:      10000,
				RecordSize:      RecordSizeMedium,
			})
			defer server.Close()

			client, _ := traverse.New(
				traverse.WithBaseURL(server.URL),
				traverse.WithODataVersion(traverse.ODataV4),
			)

			b.ResetTimer()
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

// BenchmarkThroughputSaturation measures throughput as system approaches saturation
func BenchmarkThroughputSaturation(b *testing.B) {
	loads := []int{1, 5, 10, 20, 50, 100}

	for _, load := range loads {
		b.Run(fmt.Sprintf("Concurrent_%d", load), func(b *testing.B) {
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

			var counter atomic.Int64
			var wg sync.WaitGroup

			for i := 0; i < load; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					for j := 0; j < b.N/load; j++ {
						_, _ = client.From("Products").Top(10).Collect(context.Background())
						counter.Add(1)
					}
				}()
			}

			wg.Wait()
			b.ReportMetric(float64(counter.Load()), "completed_ops")
		})
	}
}

// BenchmarkThroughputVariability measures variance in throughput
func BenchmarkThroughputVariability(b *testing.B) {
	scenarios := map[string]struct {
		latency time.Duration
		payload RecordSize
	}{
		"Stable_LowLatency":     {1 * time.Millisecond, RecordSizeSmall},
		"Variable_HighLatency":  {50 * time.Millisecond, RecordSizeLarge},
		"Balanced_MediumLatency": {5 * time.Millisecond, RecordSizeMedium},
	}

	for name, scenario := range scenarios {
		b.Run(name, func(b *testing.B) {
			server := NewMockODataServer(ServerConfig{
				Latency:         scenario.latency,
				DefaultPageSize: 100,
				MaxRecords:      10000,
				RecordSize:      scenario.payload,
			})
			defer server.Close()

			client, _ := traverse.New(
				traverse.WithBaseURL(server.URL),
				traverse.WithODataVersion(traverse.ODataV4),
			)

			timings := make([]time.Duration, 0)
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				start := time.Now()
				_, _ = client.From("Products").Top(100).Collect(context.Background())
				timings = append(timings, time.Since(start))
			}

			// Calculate variance
			if len(timings) > 1 {
				var sum time.Duration
				for _, t := range timings {
					sum += t
				}
				avg := sum / time.Duration(len(timings))

				var variance time.Duration
				for _, t := range timings {
					diff := t - avg
					if diff < 0 {
						diff = -diff
					}
					variance += diff
				}
				variance /= time.Duration(len(timings))

				b.ReportMetric(float64(variance.Microseconds()), "variance_us")
			}
		})
	}
}
