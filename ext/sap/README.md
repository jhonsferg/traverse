# SAP OData Extension

The SAP extension provides optimized integration for SAP OData services, including automatic CSRF token handling, authentication, and language routing.

## Installation

```bash
go get github.com/jhonsferg/traverse/ext/sap
```

## Quick Start

```go
package main

import (
    "context"
    "log"
    "github.com/jhonsferg/traverse/ext/sap"
)

func main() {
    client := sap.New(
        "https://sap-system.example.com:8000/sap/opu/odata/sap/ZMY_SERVICE",
        sap.WithBasicAuth("username", "password"),
        sap.WithLanguage("EN"),
    )

    ctx := context.Background()
    
    result, err := client.
        EntitySet("Customers").
        Filter("Status eq 'A'").
        Collect(ctx)
    
    if err != nil {
        log.Fatal(err)
    }
    
    log.Printf("Found %d customers", len(result))
}
```

## Features

### Automatic CSRF Token Management

SAP requires CSRF tokens for mutating operations (POST, PUT, DELETE, PATCH). The extension automatically:

1. Fetches token before create/update/delete operations
2. Includes token in request headers
3. Handles token expiration and retry

**No manual token handling required:**

```go
// Token is automatically managed
result, _ := client.EntitySet("Customers").
    Create(ctx, map[string]interface{}{
        "Name": "New Customer",
    })
```

### Authentication Methods

#### Basic Auth

```go
client := sap.New(url, sap.WithBasicAuth("user", "password"))
```

#### OAuth2 (Placeholder for future implementation)

```go
client := sap.New(url, sap.WithOAuth2Config(config))
```

### Language Routing

SAP systems support multiple languages. Specify the language for UI texts:

```go
// German
client := sap.New(url, sap.WithLanguage("DE"))

// Spanish
client := sap.New(url, sap.WithLanguage("ES"))

// French
client := sap.New(url, sap.WithLanguage("FR"))
```

### Page Size Configuration

SAP OData services respect `$top` parameter. Configure default page size:

```go
client := sap.New(url,
    sap.WithPageSize(1000), // Default: 500
)
```

## CRUD Operations

### Create

```go
newOrder := map[string]interface{}{
    "OrderID": "ORD-001",
    "CustomerID": "CUST-123",
    "Amount": 1500.00,
    "Status": "new",
}

result, err := client.EntitySet("Orders").
    Create(ctx, newOrder)

// result contains the created entity with response data
```

### Read (FindByKey)

```go
customer, err := client.EntitySet("Customers").
    FindByKey("CUST-123").
    First(ctx)
```

### Update

```go
err := client.EntitySet("Orders").
    Key("ORD-001").
    Update(ctx, map[string]interface{}{
        "Status": "completed",
    })
```

### Delete

```go
err := client.EntitySet("Orders").
    Key("ORD-001").
    Delete(ctx)
```

## Query Operations

### Filter

```go
result, _ := client.EntitySet("Customers").
    Filter("Status eq 'A' and Country eq 'US'").
    Collect(ctx)
```

### Ordering

```go
result, _ := client.EntitySet("Orders").
    OrderBy("CreatedDate desc").
    Collect(ctx)
```

### Pagination

```go
page, _ := client.EntitySet("Orders").
    Page(ctx, 1, 100) // Page 1, 100 items

log.Printf("Total: %d, Page items: %d", page.Total, len(page.Data))
```

### Counting

```go
count, _ := client.EntitySet("Orders").
    Filter("Status eq 'open'").
    Count(ctx)

log.Printf("Open orders: %d", count)
```

### Streaming Large Datasets

```go
err := client.EntitySet("SalesOrders").
    Filter("CreatedDate ge datetime'2024-01-01T00:00:00'").
    Stream(ctx, func(data map[string]interface{}) error {
        // Process each order individually
        log.Printf("Order: %v", data)
        return nil
    })
```

## Batch Operations

Combine multiple operations in a single request:

```go
batch := client.Batch()

// Add operations
batch.Create("Customers", map[string]interface{}{
    "CustomerID": "NEW-001",
    "Name": "Customer 1",
})

batch.Create("Customers", map[string]interface{}{
    "CustomerID": "NEW-002",
    "Name": "Customer 2",
})

batch.Update("Customers('CUST-001')", map[string]interface{}{
    "Status": "inactive",
})

// Execute all at once
results, err := batch.Execute(ctx)
```

## Advanced Features

### Navigation Properties (Expand)

Include related entities in response:

```go
result, _ := client.EntitySet("Orders").
    Expand("Customer", "Items").
    Collect(ctx)
```

### Nested Filters on Expanded Entities

```go
result, _ := client.EntitySet("Orders").
    Expand("Items").
    Select("OrderID", "Items/ItemID", "Items/Amount").
    Filter("Items/Amount gt 100").
    Collect(ctx)
```

### Metadata Discovery

Retrieve service metadata dynamically:

```go
metadata, _ := client.Metadata(ctx)

for _, entitySet := range metadata.EntitySets {
    log.Printf("Entity: %s", entitySet.Name)
    for _, prop := range entitySet.Properties {
        log.Printf("  - %s: %s", prop.Name, prop.Type)
    }
}
```

### Service Catalog

List all available entity sets:

```go
service, _ := client.Service(ctx)

for _, entity := range service.EntitySets {
    log.Printf("Available: %s", entity)
}
```

## Error Handling

### Common Errors

```go
err := client.EntitySet("Orders").Key("invalid").First(ctx)

if errors.Is(err, traverse.ErrNotFound) {
    log.Println("Entity not found")
}

if errors.Is(err, traverse.ErrInvalidFilter) {
    log.Println("Invalid OData filter syntax")
}

if errors.Is(err, traverse.ErrUnauthorized) {
    log.Println("Authentication failed")
}
```

### OData Error Responses

SAP OData services return structured error responses:

```go
type ODataError struct {
    Code    string
    Message string
    Details map[string]interface{}
}

// Errors are wrapped but accessible
if err != nil {
    log.Printf("Error: %v", err)
}
```

## Performance Tips

### 1. Use Caching

```go
import "github.com/jhonsferg/traverse/ext/cache/memory"

cache := memory.New(3600) // 1 hour TTL

client := sap.New(url,
    sap.WithBasicAuth("user", "pass"),
    // Requires wrapping with traverse.WithMetadataCache
)
```

### 2. Streaming for Large Datasets

**Bad:** All data in memory
```go
// ❌ 1 million records = 100MB+ memory
result, _ := client.EntitySet("Orders").Collect(ctx)
```

**Good:** Streaming
```go
// ✅ Constant memory usage
_ = client.EntitySet("Orders").Stream(ctx, func(data map[string]interface{}) error {
    process(data)
    return nil
})
```

### 3. Filter on Server

**Bad:** Fetch all, filter locally
```go
// ❌ Transfers unnecessary data
all, _ := client.EntitySet("Orders").Collect(ctx)
active := filter(all)
```

**Good:** Filter in OData
```go
// ✅ Only transfers matching records
active, _ := client.EntitySet("Orders").
    Filter("Status eq 'A'").
    Collect(ctx)
```

### 4. Optimize Page Size

```go
// Find optimal page size for your service
// Typical: 500-5000

client := sap.New(url,
    sap.WithPageSize(2000), // Test different values
)
```

## Troubleshooting

### CSRF Token Errors

If you see "CSRF token invalid" errors:

1. Ensure authentication credentials are correct
2. Check if token endpoint is accessible
3. Verify user has necessary permissions

### Connection Timeouts

```go
// Implement retry logic
for i := 0; i < 3; i++ {
    result, err := client.EntitySet("Orders").Collect(ctx)
    if err == nil {
        break
    }
    
    if i < 2 {
        time.Sleep(time.Second * time.Duration(i+1))
        continue
    }
    
    log.Fatal(err)
}
```

### Memory Issues with Large Datasets

Always use streaming instead of `Collect()`:

```go
// Stream version - constant memory
err := client.EntitySet("BigTable").Stream(ctx, func(row map[string]interface{}) error {
    // Process and discard
    return nil
})

// Collect version - linear memory
data, _ := client.EntitySet("BigTable").Collect(ctx) // Memory = O(n)
```

## Best Practices

1. **Reuse Clients**: Create once, use many times
   ```go
   client := sap.New(url, ...)
   // Use client multiple times
   ```

2. **Implement Retry Logic**: SAP systems may have temporary issues
   ```go
   for i := 0; i < maxRetries; i++ {
       // Execute operation
   }
   ```

3. **Use Context Timeouts**: Prevent hanging requests
   ```go
   ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
   defer cancel()
   result, _ := client.EntitySet("Orders").Collect(ctx)
   ```

4. **Monitor Metrics**: Track performance
   ```go
   import "github.com/jhonsferg/traverse/ext/prometheus"
   metrics := prometheus.New("sap_odata")
   // Track operations
   ```

5. **Enable Tracing**: Debug complex scenarios
   ```go
   import "github.com/jhonsferg/traverse/ext/tracing"
   tracer := tracing.New("sap_client")
   // Trace operations
   ```

## License

MIT License - See LICENSE file in parent directory
