# OData v4 Test Server

A comprehensive OData v4 compliant test server for testing the traverse OData client library.

## Features

- **OData v4 Compliant**: Implements OData v4 specification with proper metadata and service documents
- **Content Negotiation**: Supports both JSON and XML response formats based on `Accept` header
- **Query Operators**: Supports `$filter`, `$top`, `$skip`, and `$count` OData query operators
- **Sample Data**: Pre-populated with realistic product and category data for testing
- **Well Documented**: Comprehensive inline documentation and examples

## Building

From the repository root:

```bash
go build -o demo-server ./cmd/demo
```

Or run directly:

```bash
go run ./cmd/demo
```

## Running

```bash
./demo-server
```

The server will start on `http://localhost:9999` and display available endpoints:

```
OData v4 Test Server
====================

Service running on: http://localhost:9999

Endpoints:
  Service root:  http://localhost:9999/odata/v4/
  Metadata:      http://localhost:9999/odata/v4/$metadata
  Products:      http://localhost:9999/odata/v4/Products
  Categories:    http://localhost:9999/odata/v4/Categories

Features:
  ✓ JSON and XML response formats (Accept header)
  ✓ OData query operators: $filter, $top, $skip, $count
  ✓ Sample data: 10 products, 3 categories
  ✓ Full OData v4 metadata

Press Ctrl+C to stop
```

## Endpoints

### Service Root
Returns the OData service document listing available entity sets.

```bash
GET http://localhost:9999/odata/v4/
```

**Response (JSON):**
```json
{
  "@odata.context": "http://localhost:9999/odata/v4/$metadata",
  "value": [
    {"name": "Products", "url": "Products"},
    {"name": "Categories", "url": "Categories"}
  ]
}
```

### Metadata
Returns the OData entity data model (EDM) in XML format, defining all entity types and properties.

```bash
GET http://localhost:9999/odata/v4/$metadata
```

### Entity Set: Products

Returns all products in JSON format (default):

```bash
GET http://localhost:9999/odata/v4/Products
```

Returns products in XML format:

```bash
GET http://localhost:9999/odata/v4/Products \
  -H "Accept: application/xml"
```

**Response (JSON):**
```json
{
  "@odata.context": "http://localhost:9999/odata/v4/$metadata#Products",
  "value": [
    {
      "ID": 1,
      "Name": "Laptop",
      "Price": 999.99,
      "Category": "Electronics",
      "InStock": true
    },
    ...
  ]
}
```

### Entity Set: Categories

Returns all categories:

```bash
GET http://localhost:9999/odata/v4/Categories
```

## Query Examples

### Filtering

Filter products by category:

```bash
curl "http://localhost:9999/odata/v4/Products?$filter=Category%20eq%20'Electronics'"
```

Filter by stock status:

```bash
curl "http://localhost:9999/odata/v4/Products?$filter=InStock%20eq%20true"
```

Filter by price:

```bash
curl "http://localhost:9999/odata/v4/Products?$filter=Price%20gt%20100"
```

### Pagination

Get first 5 products:

```bash
curl "http://localhost:9999/odata/v4/Products?$top=5"
```

Get products starting from position 10:

```bash
curl "http://localhost:9999/odata/v4/Products?$skip=10&$top=5"
```

### Count

Get total count of products:

```bash
curl http://localhost:9999/odata/v4/Products/$count
```

Get count of filtered results:

```bash
curl "http://localhost:9999/odata/v4/Products/$count?$filter=Category%20eq%20'Electronics'"
```

## Content Negotiation

The server automatically selects response format based on the `Accept` header:

| Accept Header | Response Format |
|---------------|-----------------|
| `application/json` | JSON |
| `application/xml` | XML |
| `*/*` or not specified | JSON (default) |

Examples:

```bash
# Request JSON (default)
curl http://localhost:9999/odata/v4/Products

# Request XML
curl -H "Accept: application/xml" http://localhost:9999/odata/v4/Products
```

## Sample Data

### Products

The server includes 10 sample products:

| ID | Name | Price | Category | InStock |
|----|------|-------|----------|---------|
| 1 | Laptop | 999.99 | Electronics | true |
| 2 | Mouse | 29.99 | Electronics | true |
| 3 | Keyboard | 79.99 | Electronics | false |
| 4 | Monitor | 299.99 | Electronics | true |
| 5 | USB Cable | 9.99 | Accessories | true |
| 6 | HDMI Cable | 14.99 | Accessories | true |
| 7 | Desk Chair | 199.99 | Furniture | true |
| 8 | Standing Desk | 499.99 | Furniture | false |
| 9 | Monitor Stand | 49.99 | Accessories | true |
| 10 | Webcam | 79.99 | Electronics | true |

### Categories

| ID | Name | Description |
|----|------|-------------|
| 1 | Electronics | Electronic devices and gadgets |
| 2 | Accessories | Computer accessories |
| 3 | Furniture | Office furniture |

## Testing with traverse

Use this server to test the traverse OData client library:

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/jhonsferg/traverse"
)

func main() {
	// Create client pointing to the test server
	client, err := traverse.New(
		traverse.WithBaseURL("http://localhost:9999/odata/v4"),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	ctx := context.Background()

	// Query products
	qb := client.From("Products").Top(5)
	results, err := qb.Find(ctx)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Found %d products\n", len(results))
}
```

## Code Structure

- `main.go` - Server implementation with handlers and data models

### Key Functions

- `getAcceptFormat(r)` - Parses Accept header for content negotiation
- `writeJSONResponse(w, data)` - Writes JSON response with proper formatting
- `writeXMLResponse(w, data)` - Writes XML response with proper formatting
- `handleMetadata(w, r)` - Returns OData metadata (EDM)
- `handleService(w, r)` - Returns service document
- `handleProducts(w, r)` - Returns Products entity set with query support
- `handleCategories(w, r)` - Returns Categories entity set
- `filterProducts(list, filter)` - Applies OData filter expressions

## Supported OData Query Operators

- `$filter` - Filter results based on property conditions
  - Examples: `Category eq 'Electronics'`, `InStock eq true`, `Price gt 100`
- `$top` - Limit the number of results
  - Example: `$top=5` returns first 5 records
- `$skip` - Skip a number of records (for pagination)
  - Example: `$skip=10` skips first 10 records
- `$count` - Return count of matching records
  - Example: `/Products/$count` or `?$count=true`

## Limitations

This is a **test server** with simplified implementations:

- Filter parsing is basic and supports simple patterns only
- No complex OData expressions (use, group by, etc.)
- No authentication or authorization
- In-memory data (no persistence)
- Single-threaded (sequential request handling)

For production OData services, use a full OData framework or service like:
- Microsoft Dynamics
- SAP OData services
- Custom implementations using [OData libraries](https://www.odata.org/libraries/)

## Performance Considerations

This server is optimized for **testing and development**:

- Suitable for testing with small datasets
- Adequate for single concurrent client testing
- Not recommended for load testing or production use

## Troubleshooting

### Server won't start on port 9999

Check if port 9999 is already in use:

```bash
# Windows
netstat -ano | findstr :9999

# Linux/macOS
lsof -i :9999
```

Kill the process or use a different port (edit `main.go`).

### XML responses not working

Ensure you're sending the correct `Accept` header:

```bash
curl -H "Accept: application/xml" http://localhost:9999/odata/v4/Products
```

### Filters not working as expected

Remember that filter support is simplified. Supported patterns:

```
Category eq 'Electronics'
Category eq 'Accessories'
Category eq 'Furniture'
InStock eq true
InStock eq false
Price gt 100
```

## Contributing

To extend this test server:

1. Modify sample data in `main.go` (products, categories)
2. Add new entity types (new structs and sample data)
3. Implement new handlers following the pattern
4. Update documentation

## Related

- [traverse Library](../../../README.md) - OData client library
- [OData Specification](https://www.odata.org/documentation/) - OData v4 standard
- [OData Best Practices](https://www.odata.org/guidance/) - OData design guidance

## License

See LICENSE file in repository root.
