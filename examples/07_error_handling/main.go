// Package main demonstrates error handling patterns for OData client operations.
//
// This example covers error handling:
//   - OData-specific error types and error codes
//   - HTTP status code interpretation
//   - Context cancellation handling
//   - Retry strategies and backoff patterns
//   - Logging and debugging techniques
//   - Error classification for recovery decisions
//
// Use robust error handling to distinguish between retryable and fatal errors.
// Different error types require different handling strategies and recovery paths.
// See: https://docs.oasis-open.org/odata/odata-json-format/v4.01/odata-json-format-v4.01.html#sec_ErrorResponse
package main

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/jhonsferg/traverse"
)

// main demonstrates various error handling scenarios including OData errors,
// HTTP errors, and context cancellation for robust client applications.
func main() {
	client, err := traverse.New(
		traverse.WithBaseURL("http://services.odata.org/V2/Northwind/Northwind.svc"),
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	fmt.Println("=== Error Handling ===\n")

	// Example 1: Handle OData-specific errors from invalid entity names
	// OData errors are returned with code and message explaining the failure
	fmt.Println("Example 1: OData Error (Invalid Entity)")
	qb := client.From("NonExistentEntity").Select("ID").Top(1)
	results, err := qb.Find(ctx)
	if err != nil {
		// Check if this is an OData error with structured response
		if odataErr, ok := err.(*traverse.ODataError); ok {
			fmt.Printf("OData Error: %s - %s\n", odataErr.Code, odataErr.Message)
		} else {
			fmt.Printf("General Error: %v\n", err)
		}
	}

	// Example 2: Check for specific error types using errors.Is()
	// NotFound errors indicate the resource doesn't exist (HTTP 404)
	fmt.Println("\nExample 2: Checking for Specific Error Type")
	_, err = client.Read(ctx, "Products", "'999999'")
	if errors.Is(err, traverse.ErrNotFound) {
		fmt.Println("Entity not found - expected behavior for non-existent ID")
	} else if err != nil {
		fmt.Printf("Unexpected error: %v\n", err)
	}

	// Example 3: Handle context cancellation
	// When a context is canceled, in-flight requests return context.Canceled error
	fmt.Println("\nExample 3: Context Cancellation")
	cancelCtx, cancel := context.WithCancel(context.Background())
	cancel() // Immediately cancel the context

	_, err = client.From("Products").Top(100).Find(cancelCtx)
	if err == context.Canceled {
		fmt.Println("Query was cancelled - context deadline exceeded or explicitly canceled")
	} else if err != nil {
		fmt.Printf("Other error: %v\n", err)
	}
}
