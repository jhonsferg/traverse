// Package testutil provides testing utilities for traverse users.
package testutil

import (
	"fmt"
	"math/rand"
	"time"
)

// Fixtures provide common test data for OData integration tests.

// MaterialFixture generates n sample material/product records with realistic SAP fields.
//
// MaterialFixture returns a slice of maps representing materials with fields:
//   - Material: formatted ID (MAT000001, MAT000002, ...)
//   - MaterialType: cycles through FERT (finished), HALBF (semi-finished), ROHST (raw), HANDW (trading)
//   - Plant: cycles through plants (1000, 2000, 3000, 4000)
//   - BaseUnit: cycles through units (EA, KG, LTR, M)
//   - Description: text describing the material
//   - Created: dates going backwards from today
//
// Useful for testing streaming, filtering, and large dataset operations.
//
// Example:
//
//	materials := testutil.MaterialFixture(1000)
//	for record := range client.From("Materials").Stream(ctx) {
//		// Test with realistic material data
//	}
func MaterialFixture(n int) []map[string]interface{} {
	return GenerateFixture(n, map[string]func(i int) interface{}{
		"Material": func(i int) interface{} {
			return fmt.Sprintf("MAT%06d", i+1)
		},
		"MaterialType": func(i int) interface{} {
			types := []string{"FERT", "HALBF", "ROHST", "HANDW"}
			return types[i%len(types)]
		},
		"Plant": func(i int) interface{} {
			plants := []string{"1000", "2000", "3000", "4000"}
			return plants[i%len(plants)]
		},
		"BaseUnit": func(i int) interface{} {
			units := []string{"EA", "KG", "LTR", "M"}
			return units[i%len(units)]
		},
		"Description": func(i int) interface{} {
			return fmt.Sprintf("Material Description %d", i+1)
		},
		"Created": func(i int) interface{} {
			return time.Now().AddDate(0, 0, -i).Format("2006-01-02")
		},
	})
}

// SalesOrderFixture generates n sample sales order records with realistic SAP fields.
//
// SalesOrderFixture returns a slice of maps representing sales orders with fields:
//   - SalesOrder: formatted order ID (SO00000001, SO00000002, ...)
//   - Customer: formatted customer ID (CUST000001, CUST000002, ...)
//   - OrderDate: dates going backwards from today
//   - NetAmount: random amounts between 1000-11000 (realistic order values)
//   - Currency: always "USD" in this fixture
//   - Status: cycles through A, B, C, D (different order statuses)
//
// Useful for testing filtering, aggregation, and pagination with realistic business data.
//
// Example:
//
//	orders := testutil.SalesOrderFixture(500)
//	// Test filtering by status, date range, amount, etc.
func SalesOrderFixture(n int) []map[string]interface{} {
	return GenerateFixture(n, map[string]func(i int) interface{}{
		"SalesOrder": func(i int) interface{} {
			return fmt.Sprintf("SO%08d", i+1)
		},
		"Customer": func(i int) interface{} {
			return fmt.Sprintf("CUST%06d", i+1)
		},
		"OrderDate": func(i int) interface{} {
			return time.Now().AddDate(0, 0, -i).Format("2006-01-02")
		},
		"NetAmount": func(i int) interface{} {
			return 1000.00 + float64(rand.Intn(10000))
		},
		"Currency": func(i int) interface{} {
			return "USD"
		},
		"Status": func(i int) interface{} {
			statuses := []string{"A", "B", "C", "D"}
			return statuses[i%len(statuses)]
		},
	})
}

// GenerateFixture generates n records with specified field generators.
//
// GenerateFixture is a flexible fixture factory that creates test records by applying
// generator functions to each field. This allows building domain-specific fixtures
// while reusing the core generation logic.
//
// The fields parameter is a map of field names to generator functions.
// Each generator function receives the record index (0 to n-1) and returns a value
// for that field. This allows:
//   - Sequential IDs (using index)
//   - Cycling through enumerations
//   - Computing derived values based on position
//   - Random values (if using rand in generator)
//
// The returned slice contains n maps, each with all specified fields.
//
// Example:
//
//	records := testutil.GenerateFixture(100, map[string]func(i int) interface{}{
//		"ID": func(i int) interface{} {
//			return i + 1
//		},
//		"Status": func(i int) interface{} {
//			statuses := []string{"New", "Active", "Closed"}
//			return statuses[i % len(statuses)]
//		},
//	})
func GenerateFixture(n int, fields map[string]func(i int) interface{}) []map[string]interface{} {
	records := make([]map[string]interface{}, n)

	for i := 0; i < n; i++ {
		record := make(map[string]interface{})

		for fieldName, generator := range fields {
			record[fieldName] = generator(i)
		}

		records[i] = record
	}

	return records
}

// CustomerFixture generates n sample customer records with realistic SAP fields.
//
// CustomerFixture returns a slice of maps representing customers with fields:
//   - CustomerID: formatted ID (CUST000001, CUST000002, ...)
//   - Name: descriptive name (Customer 1, Customer 2, ...)
//   - City: cycles through US cities (New York, Los Angeles, Chicago, Houston, Miami)
//   - Country: always "USA" in this fixture
//   - Status: cycles A (Active) every third record, others I (Inactive)
//
// Useful for testing filtering by status, city, and customer relationships.
//
// Example:
//
//	customers := testutil.CustomerFixture(250)
//	// Test filtering for active customers in specific cities
func CustomerFixture(n int) []map[string]interface{} {
	return GenerateFixture(n, map[string]func(i int) interface{}{
		"CustomerID": func(i int) interface{} {
			return fmt.Sprintf("CUST%06d", i+1)
		},
		"Name": func(i int) interface{} {
			return fmt.Sprintf("Customer %d", i+1)
		},
		"City": func(i int) interface{} {
			cities := []string{"New York", "Los Angeles", "Chicago", "Houston", "Miami"}
			return cities[i%len(cities)]
		},
		"Country": func(i int) interface{} {
			return "USA"
		},
		"Status": func(i int) interface{} {
			if i%3 == 0 {
				return "A" // Active
			}
			return "I" // Inactive
		},
	})
}
