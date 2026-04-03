// Package main demonstrates OData filter query operators and string functions.
//
// This example covers filter operations:
//   - Comparison operators: gt, lt, eq, ne, ge, le
//   - String functions: startswith, endswith, contains, length
//   - Arithmetic operations: add, sub, mul, div, mod
//   - Logical operators: and, or, not
//   - Date comparisons with /Date()/ format (OData v2)
//   - Complex nested conditions with grouping
//
// Use filtering to reduce data transfer and server processing.
// Complex filters are evaluated server-side before transmission to client.
// See: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part2-url-conventions.html#sec_SystemQueryOptionfilter
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/jhonsferg/traverse"
)

// main demonstrates OData filter query operations including comparison operators,
// string functions, and complex filter expressions using standard OData syntax.
func main() {
	client, err := traverse.New(
		traverse.WithBaseURL("http://services.odata.org/V2/Northwind/Northwind.svc"),
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	fmt.Println("=== Advanced Filtering ===\n")

	// String function: startswith() - filter by product names starting with specific letter
	// String functions are evaluated on the server side
	fmt.Println("1. String Functions:")
	qb := client.From("Products").
		Select("ProductName", "UnitPrice").
		Filter("startswith(ProductName, 'C')").
		Top(5)
	results, err := qb.Find(ctx)
	if err != nil {
		log.Printf("Query failed: %v", err)
	} else {
		fmt.Printf("Products starting with 'C': %d\n", len(results))
	}

	// Numeric range filter using comparison operators
	// Chained conditions (and operator) filter by multiple criteria
	fmt.Println("\n2. Numeric Comparisons:")
	qb = client.From("Products").
		Filter("UnitPrice gt 10 and UnitPrice lt 50").
		Top(5)
	results, err = qb.Find(ctx)
	if err != nil {
		log.Printf("Query failed: %v", err)
	} else {
		fmt.Printf("Products with price 10-50: %d\n", len(results))
	}

	// Logical operators: parentheses group complex conditions
	// This filter selects active products that are not discontinued and have stock
	fmt.Println("\n3. Logical Operators:")
	qb = client.From("Products").
		Filter("(Discontinued eq false) and (UnitsInStock gt 0)").
		Top(5)
	results, err = qb.Find(ctx)
	if err != nil {
		log.Printf("Query failed: %v", err)
	} else {
		fmt.Printf("Active products in stock: %d\n", len(results))
	}

	// Date comparisons using OData v2 /Date(millis)/ format
	// Milliseconds since Unix epoch (January 1, 1970 UTC)
	fmt.Println("\n4. Date Comparisons:")
	qb = client.From("Orders").
		Filter("OrderDate ge /Date(946684800000)/").
		Top(5)
	results, err = qb.Find(ctx)
	if err != nil {
		log.Printf("Query failed: %v", err)
	} else {
		fmt.Printf("Orders after 2000-01-01: %d\n", len(results))
	}
}
