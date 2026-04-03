package benchmarks

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jhonsferg/traverse"
)

// BenchmarkMemoryAllocation tests memory allocation patterns during query execution
func BenchmarkMemoryAllocation(b *testing.B) {
	scenarios := map[string]struct {
		querySize int
		pageSize  int
	}{
		"Small_SinglePage":    {100, 100},
		"Medium_MultiPage":    {5000, 100},
		"Large_MultiPage":     {50000, 100},
		"XLarge_MultiPage":    {100000, 100},
		"Small_LargePageSize": {1000, 1000},
	}

	for name, scenario := range scenarios {
		b.Run(name, func(b *testing.B) {
			server := NewMockODataServer(ServerConfig{
				Latency:         5 * time.Millisecond,
				DefaultPageSize: scenario.pageSize,
				MaxRecords:      scenario.querySize,
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

// BenchmarkMemoryGrowth tests memory growth with increasing data sizes
func BenchmarkMemoryGrowth(b *testing.B) {
	sizes := map[string]struct {
		maxRecords int
		recordSize RecordSize
	}{
		"Small_SmallRecords":    {1000, RecordSizeSmall},
		"Small_MediumRecords":   {1000, RecordSizeMedium},
		"Small_LargeRecords":    {1000, RecordSizeLarge},
		"Medium_SmallRecords":   {10000, RecordSizeSmall},
		"Medium_MediumRecords":  {10000, RecordSizeMedium},
		"Medium_LargeRecords":   {10000, RecordSizeLarge},
		"Large_SmallRecords":    {50000, RecordSizeSmall},
		"Large_MediumRecords":   {50000, RecordSizeMedium},
	}

	for name, size := range sizes {
		b.Run(name, func(b *testing.B) {
			server := NewMockODataServer(ServerConfig{
				Latency:         5 * time.Millisecond,
				DefaultPageSize: 100,
				MaxRecords:      size.maxRecords,
				RecordSize:      size.recordSize,
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

// BenchmarkMemoryPressure tests behavior under memory-intensive scenarios
func BenchmarkMemoryPressure(b *testing.B) {
	tests := map[string]struct {
		name   string
		setup  func() (*MockODataServer, *traverse.Client)
		query  func(*traverse.Client) error
	}{
		"Multiple_Concurrent_Queries": {
			name: "Multiple_Concurrent_Queries",
			setup: func() (*MockODataServer, *traverse.Client) {
				server := NewMockODataServer(ServerConfig{
					Latency:         5 * time.Millisecond,
					DefaultPageSize: 100,
					MaxRecords:      10000,
					RecordSize:      RecordSizeLarge,
				})
				client, _ := traverse.New(
					traverse.WithBaseURL(server.URL),
					traverse.WithODataVersion(traverse.ODataV4),
				)
				return server, client
			},
			query: func(c *traverse.Client) error {
				_, err := c.From("Products").Top(1000).Collect(context.Background())
				return err
			},
		},
		"Large_Filter_Expression": {
			name: "Large_Filter_Expression",
			setup: func() (*MockODataServer, *traverse.Client) {
				server := NewMockODataServer(ServerConfig{
					Latency:         5 * time.Millisecond,
					DefaultPageSize: 100,
					MaxRecords:      50000,
					RecordSize:      RecordSizeMedium,
				})
				client, _ := traverse.New(
					traverse.WithBaseURL(server.URL),
					traverse.WithODataVersion(traverse.ODataV4),
				)
				return server, client
			},
			query: func(c *traverse.Client) error {
				_, err := c.From("Products").
					Filter("Price gt 100 and Status eq 'Active'").
					Collect(context.Background())
				return err
			},
		},
		"Select_Many_Fields": {
			name: "Select_Many_Fields",
			setup: func() (*MockODataServer, *traverse.Client) {
				server := NewMockODataServer(ServerConfig{
					Latency:         5 * time.Millisecond,
					DefaultPageSize: 100,
					MaxRecords:      10000,
					RecordSize:      RecordSizeLarge,
				})
				client, _ := traverse.New(
					traverse.WithBaseURL(server.URL),
					traverse.WithODataVersion(traverse.ODataV4),
				)
				return server, client
			},
			query: func(c *traverse.Client) error {
				fields := make([]string, 20)
				for i := 0; i < 20; i++ {
					fields[i] = fmt.Sprintf("Field%d", i)
				}
				_, err := c.From("Products").
					Select(fields...).
					Collect(context.Background())
				return err
			},
		},
	}

	for name, test := range tests {
		b.Run(name, func(b *testing.B) {
			server, client := test.setup()
			defer server.Close()
			defer client.Close()

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				_ = test.query(client)
			}
		})
	}
}

// BenchmarkHeapFragmentation tests heap behavior with different allocation patterns
func BenchmarkHeapFragmentation(b *testing.B) {
	patterns := map[string]func(*traverse.Client) error{
		"Sequential_Queries": func(c *traverse.Client) error {
			for i := 0; i < 100; i++ {
				_, err := c.From("Products").Top(10).Collect(context.Background())
				if err != nil {
					return err
				}
			}
			return nil
		},
		"Batch_Queries": func(c *traverse.Client) error {
			_, err := c.From("Products").Top(1000).Collect(context.Background())
			return err
		},
		"Mixed_PageSizes": func(c *traverse.Client) error {
			for i := 1; i <= 5; i++ {
				top := i * 100
				_, err := c.From("Products").Top(top).Collect(context.Background())
				if err != nil {
					return err
				}
			}
			return nil
		},
	}

	for name, pattern := range patterns {
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
				_ = pattern(client)
			}
		})
	}
}

// BenchmarkMemoryLeakDetection tests for potential memory leaks in edge cases
func BenchmarkMemoryLeakDetection(b *testing.B) {
	scenarios := []struct {
		name  string
		setup func(int) (*MockODataServer, *traverse.Client)
		test  func(*traverse.Client, int)
	}{
		{
			name: "ClientReuse",
			setup: func(_ int) (*MockODataServer, *traverse.Client) {
				server := NewMockODataServer(ServerConfig{
					Latency:         5 * time.Millisecond,
					DefaultPageSize: 100,
					MaxRecords:      5000,
					RecordSize:      RecordSizeSmall,
				})
				client, _ := traverse.New(
					traverse.WithBaseURL(server.URL),
					traverse.WithODataVersion(traverse.ODataV4),
				)
				return server, client
			},
			test: func(c *traverse.Client, n int) {
				for i := 0; i < n; i++ {
					_, _ = c.From("Products").Collect(context.Background())
				}
			},
		},
		{
			name: "ClientCreation",
			setup: func(_ int) (*MockODataServer, *traverse.Client) {
				server := NewMockODataServer(ServerConfig{
					Latency:         5 * time.Millisecond,
					DefaultPageSize: 100,
					MaxRecords:      1000,
					RecordSize:      RecordSizeSmall,
				})
				return server, nil
			},
			test: func(_ *traverse.Client, n int) {
				server := NewMockODataServer(ServerConfig{
					Latency:         5 * time.Millisecond,
					DefaultPageSize: 100,
					MaxRecords:      1000,
					RecordSize:      RecordSizeSmall,
				})
				defer server.Close()

				for i := 0; i < n; i++ {
					client, _ := traverse.New(
						traverse.WithBaseURL(server.URL),
						traverse.WithODataVersion(traverse.ODataV4),
					)
					_, _ = client.From("Products").Collect(context.Background())
					client.Close()
				}
			},
		},
	}

	for _, scenario := range scenarios {
		b.Run(scenario.name, func(b *testing.B) {
			server, client := scenario.setup(b.N)
			defer server.Close()
			if client != nil {
				defer client.Close()
			}

			b.ResetTimer()
			b.ReportAllocs()

			scenario.test(client, b.N)
		})
	}
}
