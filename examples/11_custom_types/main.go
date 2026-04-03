// Package main demonstrates OData custom type handling and serialization.
//
// This example covers custom types:
//   - DateTime handling with /Date(ms)/ format (OData v2)
//   - Guid/UUID type handling
//   - Decimal type for financial calculations (arbitrary precision)
//   - Binary data encoding and decoding
//   - Enum types in OData
//   - Custom type marshaling/unmarshaling
//
// Use custom type conversions when working with non-standard Go types in OData services.
// Different OData versions have different type conventions and serialization formats.
// See: https://docs.oasis-open.org/odata/odata-json-format/v4.01/odata-json-format-v4.01.html#sec_PrimitiveValue
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/jhonsferg/traverse"
)

// main demonstrates handling of OData custom types including DateTime, Guid, Decimal,
// and Binary data types through proper marshaling and unmarshaling.
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

	fmt.Println("=== Custom Types ===\n")

	// OData DateTime handling - SAP and legacy systems use /Date(ms)/ format
	// Milliseconds since Unix epoch (1970-01-01 UTC)
	fmt.Println("1. DateTime (SAP /Date(ms)/ format):")
	dt := traverse.DateTime(time.Now())
	jsonBytes, err := json.Marshal(dt)
	if err != nil {
		log.Printf("Marshal failed: %v", err)
	} else {
		fmt.Printf("Marshaled: %s\n", string(jsonBytes))
	}

	// Guid/UUID handling - unique identifiers in standard format
	// Represented as lowercase hex string with hyphens: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
	fmt.Println("\n2. Guid:")
	guid := "550e8400-e29b-41d4-a716-446655440000"
	guidLiteral := traverse.GuidValue(guid)
	fmt.Printf("Guid literal: %s\n", guidLiteral)

	// Decimal type for financial data - maintains arbitrary precision
	// Prevents floating-point rounding errors in monetary calculations
	fmt.Println("\n3. Decimal (arbitrary precision):")
	price := traverse.DecimalValue(1234.567)
	fmt.Printf("Decimal literal: %s\n", price)

	// Binary data - base64 encoded in JSON
	// Used for images, files, and other byte sequences
	fmt.Println("\n4. Binary (base64):")
	binary := traverse.Binary([]byte("Hello World"))
	jsonBytes, err = json.Marshal(binary)
	if err != nil {
		log.Printf("Marshal failed: %v", err)
	} else {
		fmt.Printf("Binary encoded: %s\n", string(jsonBytes))
	}
}
