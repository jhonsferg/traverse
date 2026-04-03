// Package main demonstrates CRUD (Create, Read, Update, Delete) operations on OData services.
//
// This example covers CRUD patterns:
//   - Create (POST) - insert new entities
//   - Read (GET) - retrieve single or multiple entities
//   - Update (PATCH) - modify entity properties
//   - Delete (DELETE) - remove entities
//   - Error handling for each operation
//   - Transaction and batch operation principles
//
// Use CRUD operations when you need to manipulate data directly on the OData service.
// For bulk operations, consider using batch requests to reduce network round-trips.
// See: https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/jhonsferg/traverse"
)

// main demonstrates basic CRUD operations: Create, Read, Update, and Delete
// using the traverse client with proper error handling for production use.
func main() {
	client, err := traverse.New(
		traverse.WithBaseURL("http://services.odata.org/V2/Northwind/Northwind.svc"),
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	fmt.Println("=== CRUD Operations ===\n")

	// CREATE: Insert a new entity into the Products collection
	// Returns the created entity with generated keys (if any)
	fmt.Println("1. CREATE:")
	newProduct := map[string]interface{}{
		"ProductName":  "New Product",
		"UnitPrice":    99.99,
		"Discontinued": false,
	}
	result, err := client.Create(ctx, "Products", newProduct)
	if err != nil {
		log.Printf("Create failed: %v", err)
	} else {
		fmt.Printf("Created: %v\n", result)
	}

	// READ: Retrieve a single entity by its key
	// Key format depends on the OData service and entity type
	fmt.Println("\n2. READ:")
	product, err := client.Read(ctx, "Products", "'1'")
	if err != nil {
		log.Printf("Read failed: %v", err)
	} else {
		fmt.Printf("Read: %v\n", product)
	}

	// UPDATE: Modify specific properties of an existing entity
	// Uses PATCH semantics - only specified properties are updated
	fmt.Println("\n3. UPDATE:")
	updates := map[string]interface{}{
		"UnitPrice": 89.99,
	}
	err = client.Update(ctx, "Products", "'1'", updates)
	if err != nil {
		log.Printf("Update failed: %v", err)
	} else {
		fmt.Println("Updated product price to 89.99")
	}

	// DELETE: Remove an entity from the service
	// Typically idempotent - deleting non-existent entities may succeed
	fmt.Println("\n4. DELETE:")
	err = client.Delete(ctx, "Products", "'999'")
	if err != nil {
		log.Printf("Delete failed: %v", err)
	} else {
		fmt.Println("Deleted product")
	}
}
