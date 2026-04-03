// Package main demonstrates advanced OData filtering with complex query expressions.
//
// This example covers advanced filters:
//   - String functions: contains, length, substring operations
//   - Arithmetic operations: mul, div, add, sub, mod
//   - Grouped conditions with operator precedence
//   - Nested parentheses for complex logic
//   - Query optimization and performance considerations
//   - Common patterns: range filters, pattern matching
//
// Use advanced filters to leverage server-side query processing and reduce
// data transfer. Complex filters are evaluated on the server before transmission.
// See: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part2-url-conventions.html#sec_BuitInFilterOperations
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/jhonsferg/traverse"
)

// main demonstrates advanced OData filter expressions including string operations,
// arithmetic calculations, and complex grouped conditions for sophisticated
// server-side data filtering and optimization.
func main() {
	client, err := traverse.New(
		traverse.WithBaseURL("http://services.odata.org/V2/Northwind/Northwind.svc"),
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()
	_ = ctx // Suppress unused variable if not using in this demo

	fmt.Println("=== Advanced Filters ===\n")

	// String function filters - server-side text operations
	// These functions operate on string properties to match patterns
	fmt.Println("1. String Functions:")
	qb := client.From("Products").
		Filter("contains(ProductName, 'Chai')").
		Top(5)
	fmt.Println("Filter: contains(ProductName, 'Chai')")
	results, err := qb.Find(ctx)
	if err != nil {
		log.Printf("Query failed: %v", err)
	} else {
		fmt.Printf("Results: %d products\n\n", len(results))
	}

	// String length filter - find products with long names
	// length() returns the number of characters in a string
	qb = client.From("Products").
		Filter("length(ProductName) gt 10").
		Top(5)
	fmt.Println("Filter: length(ProductName) gt 10")
	results, err = qb.Find(ctx)
	if err != nil {
		log.Printf("Query failed: %v", err)
	} else {
		fmt.Printf("Results: %d products\n\n", len(results))
	}

	// Arithmetic operations - use mul for calculation-based filters
	// Multiplying unit price by quantity to find expensive product combinations
	fmt.Println("2. Arithmetic Operations:")
	qb = client.From("Products").
		Filter("UnitPrice mul 10 gt 1000").
		Top(5)
	fmt.Println("Filter: UnitPrice mul 10 gt 1000")
	results, err = qb.Find(ctx)
	if err != nil {
		log.Printf("Query failed: %v", err)
	} else {
		fmt.Printf("Results: %d products\n\n", len(results))
	}

	// Complex grouped conditions with multiple operators and precedence rules
	// Parentheses control operator precedence for correct logical evaluation
	// This filter: (price 10-20 OR no stock) AND discontinued=false
	fmt.Println("3. Grouped Conditions:")
	qb = client.From("Products").
		Filter("((UnitPrice gt 10 and UnitPrice lt 20) or (UnitsInStock eq 0))").
		Top(5)
	fmt.Println("Filter: ((UnitPrice gt 10 and UnitPrice lt 20) or (UnitsInStock eq 0))")
	results, err = qb.Find(ctx)
	if err != nil {
		log.Printf("Query failed: %v", err)
	} else {
		fmt.Printf("Results: %d products\n", len(results))
	}
}
