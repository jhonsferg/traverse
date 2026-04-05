# Quick Start

This guide walks through the most common traverse operations, from connecting to an OData service through streaming large datasets.

All examples use the public [Northwind OData service](https://services.odata.org/V4/Northwind/Northwind.svc/).

## 1. Connect to an OData Service

```go
package main

import (
    "log"
    "github.com/jhonsferg/traverse"
)

func main() {
    client, err := traverse.New(
        traverse.WithBaseURL("https://services.odata.org/V4/Northwind/Northwind.svc"),
    )
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()
    // client is ready for use
}
```

For authenticated services:

```go
client, err := traverse.New(
    traverse.WithBaseURL("https://sap.example.com/sap/opu/odata/sap/MY_SRV"),
    traverse.WithBasicAuth("username", "password"),
)
```

## 2. List Entities with a Filter

Define a struct matching the OData entity shape, then use `From` to start a query:

```go
type Product struct {
    ID       int     `json:"ProductID"`
    Name     string  `json:"ProductName"`
    Price    float64 `json:"UnitPrice"`
    InStock  bool    `json:"UnitsInStock"`
}

var products []Product
err := client.From("Products").
    Filter("UnitPrice gt 20 and UnitsInStock gt 0").
    Select("ProductID", "ProductName", "UnitPrice", "UnitsInStock").
    OrderBy("UnitPrice desc").
    Top(10).
    Into(ctx, &products)
if err != nil {
    log.Fatal(err)
}

for _, p := range products {
    fmt.Printf("%s: $%.2f\n", p.Name, p.Price)
}
```

The OData URL generated:

```
/Products?$filter=UnitPrice gt 20 and UnitsInStock gt 0&$select=ProductID,ProductName,UnitPrice,UnitsInStock&$orderby=UnitPrice desc&$top=10
```

## 3. Get a Single Entity

Use `Key` to retrieve one entity by its primary key:

```go
type Order struct {
    ID         int    `json:"OrderID"`
    CustomerID string `json:"CustomerID"`
    OrderDate  string `json:"OrderDate"`
    Freight    float64 `json:"Freight"`
}

var order Order
err := client.From("Orders").
    Key(10248).
    Into(ctx, &order)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Order %d: customer=%s\n", order.ID, order.CustomerID)
```

For composite keys:

```go
err := client.From("OrderDetails").
    Key(map[string]any{"OrderID": 10248, "ProductID": 11}).
    Into(ctx, &detail)
```

## 4. Create an Entity

Use `Create` to POST a new entity:

```go
type Category struct {
    ID          int    `json:"CategoryID,omitempty"`
    Name        string `json:"CategoryName"`
    Description string `json:"Description"`
}

newCategory := Category{
    Name:        "Electronics",
    Description: "Electronic devices and accessories",
}

var created Category
err := client.From("Categories").Create(ctx, newCategory, &created)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Created category %d: %s\n", created.ID, created.Name)
```

!!! note "Server-generated keys"
    When the server generates the primary key, set `omitempty` on the ID field. The created entity returned in `&created` will contain the server-assigned ID.

## 5. Update with ETag

Use `ReadWithETag` to fetch the current ETag, then `UpdateWithETag` to PATCH with optimistic concurrency:

```go
var category Category
etag, err := client.From("Categories").Key(1).ReadWithETag(ctx, &category)
if err != nil {
    log.Fatal(err)
}

category.Description = "Updated description"

err = client.From("Categories").Key(1).UpdateWithETag(ctx, category, etag)
if err != nil {
    // 412 Precondition Failed means the entity was modified by someone else
    log.Fatal(err)
}
```

## 6. Delete an Entity

```go
err := client.From("Categories").Key(99).Delete(ctx)
if err != nil {
    log.Fatal(err)
}
```

Delete with ETag (to prevent accidental deletion of a modified entity):

```go
err := client.From("Categories").Key(99).DeleteWithETag(ctx, etag)
```

## 7. Stream a Large Result Set

For large datasets, use `Stream` to receive results over a channel - memory stays constant regardless of result size:

```go
type Order struct {
    ID         int    `json:"OrderID"`
    CustomerID string `json:"CustomerID"`
}

stream, err := client.From("Orders").
    Filter("Freight gt 0").
    OrderBy("OrderDate asc").
    Stream(ctx, Order{})
if err != nil {
    log.Fatal(err)
}
defer stream.Close()

count := 0
for item := range stream.Items() {
    order := item.(Order)
    count++
    // process order - memory is reused per iteration
}

if err := stream.Err(); err != nil {
    log.Fatal(err)
}
fmt.Printf("Processed %d orders\n", count)
```

!!! tip "When to stream vs paginate"
    Use streaming when you need to process all results in sequence (ETL, aggregation, export). Use [pagination](guides/pagination.md) when you need random page access or want to display a page at a time.

## Next Steps

- [Query Builder](guides/query-builder.md) - All query options in depth
- [CRUD Operations](guides/crud.md) - Full create/read/update/delete patterns
- [ETag & Concurrency](guides/etag-concurrency.md) - Optimistic concurrency control
- [Streaming](guides/streaming.md) - Processing millions of records
- [OData Primer](odata-primer.md) - OData concepts explained for Go developers
