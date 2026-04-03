// Package main demonstrates concurrent OData queries using goroutines and WaitGroups.
//
// This example covers concurrency patterns:
//   - Parallel query execution across multiple entities
//   - Goroutine management with sync.WaitGroup
//   - Channel-based result collection
//   - Connection pooling and HTTP keep-alive
//   - Rate limiting and backpressure handling
//   - Error handling in concurrent contexts
//
// Use concurrent queries to improve throughput when querying independent data.
// The traverse client is safe for concurrent use with proper HTTP connection pooling.
// See: https://golang.org/pkg/net/http/#Client
package main

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/jhonsferg/traverse"
)

// main demonstrates concurrent query execution using goroutines and channels
// to efficiently query multiple entity sets in parallel while managing
// resource constraints through synchronization primitives.
func main() {
	client, err := traverse.New(
		traverse.WithBaseURL("http://services.odata.org/V2/Northwind/Northwind.svc"),
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	fmt.Println("=== Concurrent Queries ===\n")

	var wg sync.WaitGroup
	// Buffered channel prevents goroutines from blocking if main thread delays
	results := make(chan interface{}, 10)

	// List of entity sets to query concurrently
	entities := []string{"Products", "Customers", "Orders", "Employees"}

	// Launch a goroutine for each entity
	wg.Add(len(entities))
	for _, entity := range entities {
		go func(e string) {
			defer wg.Done()

			// Each goroutine executes an independent query
			qb := client.From(e).Top(10)
			res, err := qb.Find(ctx)
			if err != nil {
				log.Printf("Query for %s failed: %v", e, err)
				return
			}

			// Send result to channel for main thread collection
			results <- map[string]interface{}{
				"Entity": e,
				"Count":  len(res),
			}
		}(entity)
	}

	// Close channel after all goroutines complete
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect and process results from all concurrent queries
	for res := range results {
		fmt.Printf("Query result: %v\n", res)
	}
}
