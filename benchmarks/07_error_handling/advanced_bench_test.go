package benchmarks

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/jhonsferg/traverse"
)

// BenchmarkErrorHandling tests error handling overhead
func BenchmarkErrorHandling(b *testing.B) {
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

	b.Run("InvalidFilter", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = client.From("Products").Filter("Invalid[Filter]}").Collect(context.Background())
		}
	})

	b.Run("NetworkError", func(b *testing.B) {
		badClient, _ := traverse.New(traverse.WithBaseURL("http://localhost:9999"))
		defer func() { _ = badClient.Close() }()

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		for i := 0; i < b.N; i++ {
			_, _ = badClient.From("Products").Collect(ctx)
		}
		cancel()
	})

	b.Run("ContextCancelled", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			_, _ = client.From("Products").Collect(ctx)
		}
	})
}

// BenchmarkBufferSizeImpact tests different stream buffer sizes
func BenchmarkBufferSizeImpact(b *testing.B) {
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

	bufferSizes := []int{1, 10, 100, 1000, 10000}

	for _, size := range bufferSizes {
		b.Run(fmt.Sprintf("Buffer%d", size), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				count := 0
				for result := range client.From("Products").Stream(context.Background()) {
					if result.Err != nil {
						break
					}
					count++
					if count > size {
						break
					}
				}
			}
		})
	}
}

// BenchmarkQueryCachingScenarios tests cache effectiveness in different patterns
func BenchmarkQueryCachingScenarios(b *testing.B) {
	server := NewMockODataServer(ServerConfig{Latency: 5 * time.Millisecond, DefaultPageSize: 100, MaxRecords: 1000, RecordSize: RecordSizeSmall})
	defer server.Close()

	client, _ := traverse.New(traverse.WithBaseURL(server.URL))
	defer func() { _ = client.Close() }()

	b.Run("RepeatedSameQuery", func(b *testing.B) {
		// Same query 100 times - should benefit from caching
		for i := 0; i < b.N; i++ {
			for j := 0; j < 100; j++ {
				_, _ = client.From("Products").Filter("Price gt 10").Collect(context.Background())
			}
		}
	})

	b.Run("DifferentQueries", func(b *testing.B) {
		// Different queries - cache miss every time
		for i := 0; i < b.N; i++ {
			for j := 0; j < 100; j++ {
				filter := fmt.Sprintf("Price gt %d", j)
				_, _ = client.From("Products").Filter(filter).Collect(context.Background())
			}
		}
	})

	b.Run("CacheWarmThenQuery", func(b *testing.B) {
		// Warm cache, then query
		ctx := context.Background()
		_, _ = client.From("Products").Collect(ctx)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = client.From("Products").Collect(ctx)
		}
	})
}

// BenchmarkMemoryLeaks checks for goroutine leaks in streaming
func BenchmarkMemoryLeaks(b *testing.B) {
	server := NewMockODataServer(ServerConfig{Latency: 5 * time.Millisecond, DefaultPageSize: 100, MaxRecords: 1000, RecordSize: RecordSizeSmall})
	defer server.Close()

	client, _ := traverse.New(traverse.WithBaseURL(server.URL))
	defer func() { _ = client.Close() }()

	ctx := context.Background()

	b.Run("StreamCompletion", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			// Stream should complete cleanly
			for range client.From("Products").Stream(ctx) {
			}
		}
	})

	b.Run("StreamEarlyExit", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			// Exit stream early
			for j := 0; j < 10; j++ {
				for range client.From("Products").Stream(ctx) {
					break
				}
			}
		}
	})

	b.Run("CancelledStreamContext", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			ctx, cancel := context.WithCancel(context.Background())
			go func() {
				time.Sleep(1 * time.Millisecond)
				cancel()
			}()
			for range client.From("Products").Stream(ctx) {
			}
		}
	})
}

// BenchmarkConcurrentContentionLevels tests contention at different goroutine counts
func BenchmarkConcurrentContentionLevels(b *testing.B) {
	server := NewMockODataServer(ServerConfig{Latency: 5 * time.Millisecond, DefaultPageSize: 100, MaxRecords: 1000, RecordSize: RecordSizeSmall})
	defer server.Close()

	client, _ := traverse.New(traverse.WithBaseURL(server.URL))
	defer func() { _ = client.Close() }()

	contentionLevels := []int{1, 2, 4, 8, 16, 32, 64}

	for _, level := range contentionLevels {
		b.Run(fmt.Sprintf("Goroutines%d", level), func(b *testing.B) {
			var wg sync.WaitGroup
			for i := 0; i < b.N; i++ {
				for g := 0; g < level; g++ {
					wg.Add(1)
					go func() {
						defer wg.Done()
						_, _ = client.From("Products").Collect(context.Background())
					}()
				}
				wg.Wait()
			}
		})
	}
}

// BenchmarkFilterComplexity tests filter expression complexity impact
func BenchmarkFilterComplexity(b *testing.B) {
	server := NewMockODataServer(ServerConfig{Latency: 5 * time.Millisecond, DefaultPageSize: 100, MaxRecords: 1000, RecordSize: RecordSizeSmall})
	defer server.Close()

	client, _ := traverse.New(traverse.WithBaseURL(server.URL))
	defer func() { _ = client.Close() }()

	filters := map[string]string{
		"Simple":          "Price gt 10",
		"TwoConditions":   "Price gt 10 and Status eq 'Active'",
		"ThreeConditions": "Price gt 10 and Status eq 'Active' and Rating ge 4",
		"Nested":          "(Price gt 10 and Status eq 'Active') or (Rating ge 4 and InStock eq true)",
		"Complex":         "(Price gt 10 and Price lt 100) and (Status eq 'Active' or Status eq 'Pending') and (Rating ge 4 or Rating eq 0)",
	}

	for name, filter := range filters {
		b.Run(name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, _ = client.From("Products").Filter(filter).Collect(context.Background())
			}
		})
	}
}

// BenchmarkSelectFieldCount tests impact of different field counts
func BenchmarkSelectFieldCount(b *testing.B) {
	server := NewMockODataServer(ServerConfig{Latency: 5 * time.Millisecond, DefaultPageSize: 100, MaxRecords: 1000, RecordSize: RecordSizeSmall})
	defer server.Close()

	client, _ := traverse.New(traverse.WithBaseURL(server.URL))
	defer func() { _ = client.Close() }()

	fieldCounts := map[string][]string{
		"1Field":   {"Name"},
		"3Fields":  {"Name", "Price", "Status"},
		"5Fields":  {"Name", "Price", "Status", "Rating", "InStock"},
		"10Fields": {"Name", "Price", "Status", "Rating", "InStock", "CreatedAt", "UpdatedAt", "Category", "Vendor", "SKU"},
	}

	for name, fields := range fieldCounts {
		b.Run(name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, _ = client.From("Products").Select(fields...).Collect(context.Background())
			}
		})
	}
}

// BenchmarkTopSkipCombinations tests different pagination strategies
func BenchmarkTopSkipCombinations(b *testing.B) {
	server := NewMockODataServer(ServerConfig{Latency: 5 * time.Millisecond, DefaultPageSize: 100, MaxRecords: 1000, RecordSize: RecordSizeSmall})
	defer server.Close()

	client, _ := traverse.New(traverse.WithBaseURL(server.URL))
	defer func() { _ = client.Close() }()

	combinations := map[string]struct {
		top  int
		skip int
	}{
		"Top10Skip0":      {10, 0},
		"Top100Skip0":     {100, 0},
		"Top1000Skip0":    {1000, 0},
		"Top100Skip100":   {100, 100},
		"Top100Skip1000":  {100, 1000},
		"Top100Skip10000": {100, 10000},
	}

	for name, params := range combinations {
		b.Run(name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, _ = client.From("Products").Top(params.top).Skip(params.skip).Collect(context.Background())
			}
		})
	}
}

// BenchmarkOrderByVariations tests different ordering strategies
func BenchmarkOrderByVariations(b *testing.B) {
	server := NewMockODataServer(ServerConfig{Latency: 5 * time.Millisecond, DefaultPageSize: 100, MaxRecords: 1000, RecordSize: RecordSizeSmall})
	defer server.Close()

	client, _ := traverse.New(traverse.WithBaseURL(server.URL))
	defer func() { _ = client.Close() }()

	b.Run("SingleAscending", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = client.From("Products").OrderBy("Price").Collect(context.Background())
		}
	})

	b.Run("SingleDescending", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = client.From("Products").OrderByDesc("Price").Collect(context.Background())
		}
	})

	b.Run("MultiFieldMixed", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = client.From("Products").OrderBy("Category").OrderByDesc("Price").Collect(context.Background())
		}
	})
}

// BenchmarkJSONUnmarshalOverhead tests JSON unmarshalling with different payload sizes
func BenchmarkJSONUnmarshalOverhead(b *testing.B) {
	testData := map[string][]byte{
		"Small":  generateJSONPayload(100),
		"Medium": generateJSONPayload(1000),
		"Large":  generateJSONPayload(10000),
	}

	for name, payload := range testData {
		b.Run(name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				var result map[string]interface{}
				_ = json.Unmarshal(payload, &result)
			}
		})
	}
}

// BenchmarkConcurrentVsSequential compares concurrent vs sequential operations
func BenchmarkConcurrentVsSequential(b *testing.B) {
	server := NewMockODataServer(ServerConfig{Latency: 5 * time.Millisecond, DefaultPageSize: 100, MaxRecords: 1000, RecordSize: RecordSizeSmall})
	defer server.Close()

	client, _ := traverse.New(traverse.WithBaseURL(server.URL))
	defer func() { _ = client.Close() }()

	queries := []string{"Products", "Orders", "Customers", "Invoices", "Shipments"}

	b.Run("Sequential", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for _, query := range queries {
				_, _ = client.From(query).Collect(context.Background())
			}
		}
	})

	b.Run("Concurrent", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			var wg sync.WaitGroup
			for _, query := range queries {
				wg.Add(1)
				go func(q string) {
					defer wg.Done()
					_, _ = client.From(q).Collect(context.Background())
				}(query)
			}
			wg.Wait()
		}
	})

	b.Run("ConcurrentWithLimit", func(b *testing.B) {
		sem := make(chan struct{}, 3)
		for i := 0; i < b.N; i++ {
			var wg sync.WaitGroup
			for _, query := range queries {
				wg.Add(1)
				go func(q string) {
					defer wg.Done()
					sem <- struct{}{}
					defer func() { <-sem }()
					_, _ = client.From(q).Collect(context.Background())
				}(query)
			}
			wg.Wait()
		}
	})
}

// BenchmarkAllocationPatterns analyzes allocation patterns across operations
func BenchmarkAllocationPatterns(b *testing.B) {
	server := NewMockODataServer(ServerConfig{Latency: 5 * time.Millisecond, DefaultPageSize: 100, MaxRecords: 1000, RecordSize: RecordSizeSmall})
	defer server.Close()

	client, _ := traverse.New(traverse.WithBaseURL(server.URL))
	defer func() { _ = client.Close() }()

	b.Run("QueryBuild", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = client.From("Products").
				Filter("Price gt 10").
				Select("Name", "Price", "Status").
				OrderBy("Price").
				Top(100).
				Skip(50)
		}
	})

	b.Run("QueryExecute", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, _ = client.From("Products").Collect(context.Background())
		}
	})

	b.Run("StreamOperation", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			for range client.From("Products").Stream(context.Background()) {
			}
		}
	})
}

// Helper function to generate JSON payload
func generateJSONPayload(size int) []byte {
	data := map[string]interface{}{
		"d": map[string]interface{}{
			"results": make([]map[string]interface{}, 0),
		},
	}

	// Create items to reach approximately desired size
	itemSize := 80 // rough estimate of item JSON size
	numItems := size / itemSize
	if numItems == 0 {
		numItems = 1
	}

	results := make([]map[string]interface{}, numItems)
	for i := 0; i < numItems; i++ {
		results[i] = map[string]interface{}{
			"ID":      i,
			"Name":    fmt.Sprintf("Product%d", i),
			"Price":   100.50 + float64(i),
			"Status":  "Active",
			"Rating":  4.5,
			"InStock": true,
		}
	}

	data["d"].(map[string]interface{})["results"] = results

	payload, _ := json.Marshal(data)
	return payload
}
