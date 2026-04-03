// Package main demonstrates batch OData operations for efficient multi-request processing.
//
// This example covers batch operations:
//   - Grouping multiple operations into a single HTTP request (\)
//   - Reduced network latency and improved throughput
//   - Atomic execution of related operations
//   - Handling multiple CRUD operations in one roundtrip
//   - Error handling at operation level
//
// Use batch operations when you have multiple independent operations to execute.
// Batch requests significantly reduce latency for bulk operations.
// Limits: Most OData services restrict batch size (typically 100-1000 operations).
// See: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_BatchRequests
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/jhonsferg/traverse"
)

// main demonstrates batch request execution using the traverse library.
// Batch requests allow multiple Create, Update, and Delete operations
// to be sent in a single HTTP request, improving performance.
func main() {
	client, err := traverse.New(
		traverse.WithBaseURL("http://services.odata.org/V2/Northwind/Northwind.svc"),
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	fmt.Println("=== Batch Operations ===\n")

	ctx := context.Background()

	// Create a new batch request container
	batch := client.NewBatch()

	// Add first Create operation to the batch
	batch.Create("Products", map[string]interface{}{
		"ProductName": "Product 1",
		"UnitPrice":   10.0,
	})

	// Add second Create operation to the batch
	batch.Create("Products", map[string]interface{}{
		"ProductName": "Product 2",
		"UnitPrice":   20.0,
	})

	// Add an Update operation to the batch
	// All operations are queued and sent together
	batch.Update("Products", "'1'", map[string]interface{}{
		"UnitPrice": 15.0,
	})

	// Execute all batch operations in a single HTTP request
	// Results are returned in the same order as operations were added
	results, err := batch.Execute(ctx)
	if err != nil {
		log.Fatalf("Batch execution failed: %v", err)
	}

	fmt.Printf("Executed %d operations\n", len(results))
	for i, result := range results {
		fmt.Printf("Operation %d: %v\n", i+1, result)
	}
}
