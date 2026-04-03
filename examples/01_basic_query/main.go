// Package main demonstrates basic OData query operations using the traverse library.
//
// This example covers fundamental query patterns:
//   - SELECT projections to retrieve specific columns
//   - WHERE conditions (Filter) for data selection
//   - ORDER BY sorting clauses
//   - TOP/LIMIT for result limiting
//   - Pagination with Skip/Top pattern
//
// Use these patterns when you need to fetch and filter data from OData services.
// See OData 4.0 standard: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part2-url-conventions.html
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/jhonsferg/traverse"
)

// main demonstrates fundamental OData query operations including selection,
// filtering, ordering, and pagination using the traverse client builder pattern.
func main() {
	// Initialize OData client with Northwind test service
	client, err := traverse.New(
		traverse.WithBaseURL("http://services.odata.org/V2/Northwind/Northwind.svc"),
		traverse.WithODataVersion(traverse.ODataV2),
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Example 1: Simple SELECT query - select specific fields from an entity set
	fmt.Println("=== Example 1: Simple Query ===")
	qb := client.From("Products").
		Select("ProductID", "ProductName", "UnitPrice").
		Top(5)

	results, err := qb.Find(ctx)
	if err != nil {
		log.Fatalf("Query failed: %v", err)
	}
	fmt.Printf("Found %d products\n", len(results))

	// Example 2: Query with WHERE clause (Filter in OData) and ORDER BY
	// Filter uses OData comparison operators: gt (greater than), lt (less than), eq (equal), etc.
	fmt.Println("\n=== Example 2: Query with Filter ===")
	qb = client.From("Products").
		Select("ProductName", "UnitPrice").
		Filter("UnitPrice gt 20").
		OrderBy("UnitPrice").
		Top(10)

	results, err = qb.Find(ctx)
	if err != nil {
		log.Fatalf("Query failed: %v", err)
	}
	fmt.Printf("Found %d products with price > 20\n", len(results))

	// Example 3: Multiple filters are combined with logical AND
	// Each Filter() call adds another condition to the query
	fmt.Println("\n=== Example 3: Multiple Filters ===")
	qb = client.From("Products").
		Filter("UnitPrice gt 10").
		Filter("UnitsInStock gt 0").
		Select("ProductName", "UnitPrice", "UnitsInStock").
		Top(5)

	results, err = qb.Find(ctx)
	if err != nil {
		log.Fatalf("Query failed: %v", err)
	}
	fmt.Printf("Found %d products\n", len(results))

	// Example 4: Count functionality placeholder
	fmt.Println("\n=== Example 4: Count ===")
	// Note: Count() method needs to be implemented in traverse library
	fmt.Println("Count functionality (to be implemented)")

	// Example 5: Offset-based pagination pattern
	// This pattern is ideal for incremental data retrieval with Top (page size)
	// and Skip (number of records to skip).
	fmt.Println("\n=== Example 5: Pagination ===")
	pageSize := 5
	for page := 0; page < 2; page++ {
		skip := page * pageSize
		qb = client.From("Products").
			Select("ProductID", "ProductName").
			Top(pageSize).
			Skip(skip)

		results, err := qb.Find(ctx)
		if err != nil {
			log.Fatalf("Query failed: %v", err)
		}
		fmt.Printf("Page %d: Found %d products (skip: %d, top: %d)\n",
			page+1, len(results), skip, pageSize)

		// Empty result set indicates no more pages available
		if len(results) == 0 {
			break
		}
	}
}
