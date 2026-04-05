# CRUD Operations

traverse provides typed Create, Read, Update, and Delete operations. Each maps to a standard HTTP method and OData URL pattern.

## Read

### Reading a Collection

```go
var products []Product
err := client.From("Products").
    Filter("UnitPrice gt 10").
    Top(50).
    Into(ctx, &products)
```

HTTP: `GET /Products?$filter=UnitPrice gt 10&$top=50`

### Reading a Single Entity

```go
var product Product
err := client.From("Products").Key(1).Into(ctx, &product)
```

HTTP: `GET /Products(1)`

**Handling 404:**

```go
err := client.From("Products").Key(999).Into(ctx, &product)
if err != nil {
    if traverse.IsNotFound(err) {
        fmt.Println("product not found")
        return
    }
    log.Fatal(err)
}
```

### Reading with Expand

```go
type Order struct {
    ID       int          `json:"OrderID"`
    Customer Customer     `json:"Customer"`
    Details  []OrderDetail `json:"OrderDetails"`
}

var order Order
err := client.From("Orders").
    Key(10248).
    Expand("Customer", "OrderDetails").
    Into(ctx, &order)
```

HTTP: `GET /Orders(10248)?$expand=Customer,OrderDetails`

## Create

`Create` sends a POST request and decodes the created entity into the response pointer.

```go
type Category struct {
    ID          int    `json:"CategoryID,omitempty"`
    Name        string `json:"CategoryName"`
    Description string `json:"Description,omitempty"`
}

newCat := Category{Name: "Beverages", Description: "Drinks"}

var created Category
err := client.From("Categories").Create(ctx, newCat, &created)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Created: ID=%d\n", created.ID)
```

HTTP: `POST /Categories` with JSON body

**Server-generated ID:** The `omitempty` tag on `CategoryID` ensures the field is omitted from the POST body, letting the server assign the key.

**Handling 409 Conflict:**

```go
err := client.From("Categories").Create(ctx, cat, &created)
if traverse.IsConflict(err) {
    fmt.Println("entity already exists")
}
```

### Create Without Reading Response

Pass `nil` as the response pointer to skip decoding:

```go
err := client.From("Logs").Create(ctx, logEntry, nil)
```

## Update (PATCH)

`Update` sends a PATCH request, merging only the provided fields. Fields not included in the body are unchanged on the server.

```go
patch := map[string]any{
    "Description": "Updated description",
    "Price":       29.99,
}
err := client.From("Products").Key(1).Update(ctx, patch)
```

HTTP: `PATCH /Products(1)`

Using a struct:

```go
type ProductPatch struct {
    Description string  `json:"Description"`
    UnitPrice   float64 `json:"UnitPrice"`
}

err := client.From("Products").Key(1).Update(ctx, ProductPatch{
    Description: "New description",
    UnitPrice:   29.99,
})
```

!!! note "PATCH vs PUT"
    PATCH (Update) merges fields. PUT (Replace) replaces the entire entity. Use PATCH unless you specifically need full replacement semantics.

## Replace (PUT)

`Replace` sends a PUT request, replacing the entire entity. All fields must be provided.

```go
fullProduct := Product{
    ID:          1,
    Name:        "Chai",
    SupplierID:  1,
    CategoryID:  1,
    UnitPrice:   19.99,
    UnitsInStock: 50,
}
err := client.From("Products").Key(1).Replace(ctx, fullProduct)
```

HTTP: `PUT /Products(1)`

!!! warning "PUT replaces everything"
    Fields not included in a PUT body may be set to their default or null values on the server. Use PATCH for partial updates.

## Delete

```go
err := client.From("Products").Key(99).Delete(ctx)
```

HTTP: `DELETE /Products(99)`

**Handling 404:**

```go
err := client.From("Products").Key(99).Delete(ctx)
if traverse.IsNotFound(err) {
    // already deleted, nothing to do
}
```

## ETag-Aware Operations

For optimistic concurrency control, use the `WithETag` variants:

```go
// Read with ETag
var product Product
etag, err := client.From("Products").Key(1).ReadWithETag(ctx, &product)

// Update only if ETag matches (If-Match header)
err = client.From("Products").Key(1).UpdateWithETag(ctx, patch, etag)

// Replace only if ETag matches
err = client.From("Products").Key(1).ReplaceWithETag(ctx, fullProduct, etag)

// Delete only if ETag matches
err = client.From("Products").Key(1).DeleteWithETag(ctx, etag)
```

**Handling 412 Precondition Failed:**

```go
err = client.From("Products").Key(1).UpdateWithETag(ctx, patch, etag)
if traverse.IsPreconditionFailed(err) {
    fmt.Println("entity was modified by another client - re-read and retry")
}
```

See [ETag & Concurrency](etag-concurrency.md) for the full concurrency guide.

## Error Handling Reference

| HTTP Status | traverse helper | Meaning |
|-------------|----------------|---------|
| 404 Not Found | `traverse.IsNotFound(err)` | Entity does not exist |
| 409 Conflict | `traverse.IsConflict(err)` | Duplicate key or business rule violation |
| 412 Precondition Failed | `traverse.IsPreconditionFailed(err)` | ETag mismatch |
| 422 Unprocessable Entity | `traverse.IsValidationError(err)` | Server-side validation failed |
| 500+ | `traverse.IsServerError(err)` | Server error |

See [Errors Reference](../reference/errors.md) for all error types.

## Related Pages

- [ETag & Concurrency](etag-concurrency.md) - Optimistic concurrency with ETags
- [Entity Change Tracking](change-tracking.md) - Track field changes, emit minimal PATCH bodies
- [Batch Requests](batch.md) - Bundle multiple operations in one HTTP request
