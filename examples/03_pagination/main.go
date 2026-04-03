// Package main demonstrates OData pagination patterns for efficient data retrieval.
//
// This example covers pagination concepts:
//   - Skip/Top pattern for offset-based pagination
//   - Page number calculation from skip/top parameters
//   - Detecting the last page (empty result set)
//   - Iteration over large datasets with fixed page sizes
//   - Memory-efficient pagination without streaming
//
// Use this pattern when you need to iterate through data in pages
// and the client can buffer one page at a time into memory.
// See: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part2-url-conventions.html
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/jhonsferg/traverse"
)

// main demonstrates offset-based pagination using Skip/Top parameters
// to efficiently retrieve data in manageable page-sized chunks.
func main() {
	client, err := traverse.New(
		traverse.WithBaseURL("http://services.odata.org/V2/Northwind/Northwind.svc"),
		traverse.WithODataVersion(traverse.ODataV2),
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	fmt.Println("=== Pagination Example ===\n")

	pageSize := 10
	totalProcessed := 0

	// Iterate through pages using offset-based pagination
	// Each page skip by (page * pageSize) records and return pageSize records
	for page := 0; page < 5; page++ {
		skip := page * pageSize

		qb := client.From("Orders").
			Select("OrderID", "CustomerID", "OrderDate", "Freight").
			OrderBy("OrderID").
			Top(pageSize).
			Skip(skip)

		results, err := qb.Find(ctx)
		if err != nil {
			log.Fatalf("Query failed: %v", err)
		}

		// Empty result set indicates we've reached the end of data
		if len(results) == 0 {
			fmt.Printf("\nNo more results at page %d\n", page+1)
			break
		}

		fmt.Printf("Page %d (Skip: %d, Top: %d): %d records\n",
			page+1, skip, pageSize, len(results))

		totalProcessed += len(results)

		// Display details of the first record on this page for verification
		if len(results) > 0 {
			fmt.Printf("  First record: %v\n", results[0])
		}
	}

	fmt.Printf("\nTotal processed: %d records\n", totalProcessed)

	// Alternative: Using page numbers for clearer code
	// Convert page number (1-indexed) to skip calculation
	fmt.Println("\n=== Page Number Pagination ===")
	for page := 1; page <= 3; page++ {
		// Calculate skip from page number: (page - 1) * pageSize
		qb := client.From("Customers").
			Select("CustomerID", "CompanyName", "City").
			Top(5).
			Skip((page - 1) * 5)

		results, err := qb.Find(ctx)
		if err != nil {
			log.Fatalf("Query failed: %v", err)
		}

		fmt.Printf("Page %d: %d customers\n", page, len(results))
		// When no results are returned, we've paginated past all available data
		if len(results) == 0 {
			break
		}
	}
}
