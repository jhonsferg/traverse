# Batch Requests

OData `$batch` lets you bundle multiple requests into a single HTTP call, reducing round-trip latency. Changesets within a batch are atomic: all operations in a changeset succeed or all are rolled back.

## Creating a Batch

```go
batch := client.NewBatch()
```

## Adding Requests

```go
// GET requests
batch.Get("Products(1)")
batch.Get("Customers('ALFKI')")

// Mutation requests
batch.Post("Orders", newOrder)
batch.Patch("Products(1)", map[string]any{"UnitPrice": 29.99})
batch.Put("Categories(5)", fullCategory)
batch.Delete("Products(99)")
```

## Atomic Changesets

Group mutations that must all succeed or all fail:

```go
cs := batch.NewChangeset()
cs.Post("Orders", order)
cs.Post("OrderDetails", detail1)
cs.Post("OrderDetails", detail2)
batch.AddChangeset(cs)
```

If `Post("OrderDetails", detail2)` fails, the server rolls back the `Post("Orders", ...)` and `Post("OrderDetails", detail1)` as well.

## Executing the Batch

```go
ctx := context.Background()
results, err := batch.Execute(ctx)
if err != nil {
    log.Fatal(err)
}
```

## Processing Results

`results` is a slice of `BatchResult`, one per request (in order):

```go
for i, r := range results {
    if r.Err != nil {
        fmt.Printf("Request %d failed: %s\n", i, r.Err)
        continue
    }
    fmt.Printf("Request %d: HTTP %d\n", i, r.StatusCode)
}
```

Decode a specific result:

```go
var createdOrder Order
if err := results[0].Decode(&createdOrder); err != nil {
    log.Fatal(err)
}
fmt.Printf("Created order: %d\n", createdOrder.ID)
```

## Complete Example

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/jhonsferg/traverse"
)

type Order struct {
    ID         int    `json:"OrderID,omitempty"`
    CustomerID string `json:"CustomerID"`
    Freight    float64 `json:"Freight"`
}

type OrderDetail struct {
    OrderID   int     `json:"OrderID,omitempty"`
    ProductID int     `json:"ProductID"`
    Quantity  int     `json:"Quantity"`
    UnitPrice float64 `json:"UnitPrice"`
}

func main() {
    client, err := traverse.New(
        traverse.WithBaseURL("https://services.odata.org/V4/Northwind/Northwind.svc"),
    )
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    ctx := context.Background()
    batch := client.NewBatch()

    // Read requests (outside changeset, not atomic)
    batch.Get("Products(1)")
    batch.Get("Customers('ALFKI')")

    // Atomic changeset: create order + details together
    cs := batch.NewChangeset()
    cs.Post("Orders", Order{
        CustomerID: "ALFKI",
        Freight:    12.50,
    })
    cs.Post("OrderDetails", OrderDetail{
        ProductID: 1,
        Quantity:  5,
        UnitPrice: 18.00,
    })
    batch.AddChangeset(cs)

    results, err := batch.Execute(ctx)
    if err != nil {
        log.Fatal(err)
    }

    for i, r := range results {
        if r.Err != nil {
            fmt.Printf("[%d] ERROR: %v\n", i, r.Err)
        } else {
            fmt.Printf("[%d] HTTP %d\n", i, r.StatusCode)
        }
    }
}
```

## OData Batch Format

traverse serializes the batch as `multipart/mixed`:

```
POST /odata/$batch HTTP/1.1
Content-Type: multipart/mixed; boundary=batch_abc123

--batch_abc123
Content-Type: application/http
Content-Transfer-Encoding: binary

GET /Products(1) HTTP/1.1
Accept: application/json

--batch_abc123
Content-Type: multipart/mixed; boundary=changeset_xyz456

--changeset_xyz456
Content-Type: application/http
Content-Transfer-Encoding: binary

POST /Orders HTTP/1.1
Content-Type: application/json

{"CustomerID":"ALFKI","Freight":12.50}
--changeset_xyz456--
--batch_abc123--
```

## Changeset Rollback Behavior

When a changeset fails:
- OData v4: the server rolls back all operations in the changeset
- OData v2: rollback behavior depends on the service implementation (SAP ABAP Gateway supports rollback)

Operations outside of changesets (plain GET/POST outside a changeset) are independent and are not affected by changeset failure.

!!! warning "Service capability"
    Not all OData services support `$batch`. Check the service's `$metadata` for the `Batch` capability. SAP ABAP Gateway supports batch for all registered services.

## Related Pages

- [CRUD Operations](crud.md) - Individual operations
- [Delta Sync](delta-sync.md) - Sync changed records efficiently
- [SAP Compatibility](sap.md) - SAP batch considerations
