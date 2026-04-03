// Package main demonstrates SAP-specific OData integration using the traverse library.
//
// This example covers SAP integration features:
//   - CSRF token automatic management for write operations
//   - Basic authentication with username/password
//   - SAP system parameters (client, language routing)
//   - Navigation and expand capabilities
//   - SAP error handling and status codes
//
// Use this when connecting to SAP systems (NetWeaver, Business Suite, S/4HANA).
// SAP systems require special header handling and CSRF protection.
// See SAP OData documentation: https://help.sap.com/doc/saphelp_gateway/latest/en-US/
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/jhonsferg/traverse/sap"
)

// main demonstrates SAP-specific OData client initialization and usage patterns
// including CSRF token handling, authentication, and SAP-specific headers.
func main() {
	fmt.Println("=== SAP Integration Example ===\n")

	// Initialize SAP-specific OData client with authentication and system configuration
	// Note: This would connect to a real SAP system with actual credentials
	client, err := sap.NewSAPClient(
		sap.WithSystemURL("http://sap.example.com:8000"),
		sap.WithClient("100"),
		sap.WithServicePath("/sap/opu/odata/sap/C_PRODUCT_SRV"),
		sap.WithBasicAuth("username", "password"),
		sap.WithLanguage("EN"),
	)
	if err != nil {
		log.Fatalf("Failed to create SAP client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	fmt.Println("SAP OData Client Features:")
	fmt.Println("✓ Automatic CSRF token management")
	fmt.Println("✓ Basic authentication support")
	fmt.Println("✓ Language header configuration")
	fmt.Println("✓ Client parameter routing")
	fmt.Println("✓ SAP error handling")
	fmt.Println()

	// Example query structure (not executed without real system)
	fmt.Println("Example Query Structure:")
	qb := client.From("Products").
		Select("Product", "ProductDescription", "Price", "Currency").
		Filter("Status eq 'A'").
		OrderBy("ProductDescription").
		Top(100)

	fmt.Printf("Query built for entity set: %s\n", "Products")
	fmt.Println()

	fmt.Println("Key SAP Integration Points:")
	fmt.Println("1. CSRF Token Handling:")
	fmt.Println("   - Automatic fetch before write operations")
	fmt.Println("   - Token caching with 30-minute TTL")
	fmt.Println("   - Automatic refresh on 403 errors")
	fmt.Println()
	fmt.Println("2. Authentication:")
	fmt.Println("   - Basic auth embedded in requests")
	fmt.Println("   - OAuth2 ready (with custom token provider)")
	fmt.Println()
	fmt.Println("3. Headers:")
	fmt.Println("   - sap-language: EN")
	fmt.Println("   - sap-client: 100")
	fmt.Println()
	fmt.Println("4. OData Versions:")
	fmt.Println("   - v2: SAP NetWeaver Gateway / Business Suite")
	fmt.Println("   - v4: SAP S/4HANA Cloud")
	fmt.Println()

	// Show CRUD pattern for SAP systems
	fmt.Println("CRUD Pattern for SAP Systems:")
	fmt.Println()

	fmt.Println("CREATE (POST):")
	fmt.Println("data := map[string]interface{}{")
	fmt.Println("\"Product\": \"PRODUCT001\",")
	fmt.Println("\"Description\": \"New Product\",")
	fmt.Println("\"Price\": 99.99,")
	fmt.Println("}")
	fmt.Println("result, err := client.Create(ctx, \"Products\", data)")
	fmt.Println()

	fmt.Println("READ (GET):")
	fmt.Println("product, err := client.Read(ctx, \"Products\", \"PRODUCT001\")")
	fmt.Println()

	fmt.Println("UPDATE (PATCH):")
	fmt.Println("updates := map[string]interface{}{")
	fmt.Println("\"Price\": 89.99,")
	fmt.Println("}")
	fmt.Println("err := client.Update(ctx, \"Products\", \"PRODUCT001\", updates)")
	fmt.Println()

	fmt.Println("DELETE:")
	fmt.Println("err := client.Delete(ctx, \"Products\", \"PRODUCT001\")")
}
