// Package main demonstrates delta query patterns for incremental data synchronization.
//
// This example covers delta sync concepts:
//   - Delta tokens (deltatoken) for incremental queries
//   - Tracking additions, modifications, and deletions
//   - Minimizing data transfer with incremental updates
//   - Synchronization state management
//   - Common use cases: mobile apps, caching, offline sync
//
// Use delta queries when you need to track changes over time and fetch only
// modifications since the last synchronization. This significantly reduces
// bandwidth and processing for large datasets with infrequent changes.
// See: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_DeltaQueries
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/jhonsferg/traverse"
)

// main demonstrates delta query operations for tracking incremental changes
// in OData services, enabling efficient synchronization of large datasets.
func main() {
	client, err := traverse.New(
		traverse.WithBaseURL("http://services.odata.org/V2/Northwind/Northwind.svc"),
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	fmt.Println("=== Delta Sync (Incremental Updates) ===\n")

	// First sync: perform a full query to establish baseline state
	// All records are retrieved from the beginning of the dataset
	fmt.Println("Initial sync (get all records):")
	qb := client.From("Products").Select("ProductID", "ProductName")
	results, err := qb.Find(ctx)
	if err != nil {
		log.Fatalf("Query failed: %v", err)
	}
	fmt.Printf("Retrieved %d products\n", len(results))

	// In a real scenario, extract and save the deltaToken from the response
	// The delta token represents the synchronization state at this point
	// Future queries use this token to fetch only subsequent changes
	fmt.Println("\nIn subsequent syncs, use delta tokens to fetch only changes:")
	fmt.Println("- Tracks additions (new entities inserted)")
	fmt.Println("- Tracks modifications (existing entities updated)")
	fmt.Println("- Tracks deletions (removed entities)")
	fmt.Println("- Minimizes data transfer (only changed data)")
	fmt.Println()
	fmt.Println("Delta query workflow:")
	fmt.Println("1. Execute initial query and save deltaToken from response")
	fmt.Println("2. On next sync, use saved deltaToken in Filter query")
	fmt.Println("3. Process changes: additions, updates, and deletions")
	fmt.Println("4. Save new deltaToken from response for next sync")
	fmt.Println("5. Repeat steps 2-4 for continuous synchronization")
}
