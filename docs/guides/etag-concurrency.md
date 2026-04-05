# ETag & Concurrency

ETags provide optimistic concurrency control: you can update or delete an entity only if it has not been modified since you last read it. traverse has first-class ETag support in all mutating operations.

## The ETag Type

```go
type ETag struct {
    Value string
    Weak  bool
}
```

| Method | Description |
|--------|-------------|
| `IsWeak() bool` | Returns true if the ETag is prefixed with `W/` |
| `IsEmpty() bool` | Returns true if Value is empty |
| `String() string` | Returns the full ETag string including `W/` prefix if weak |

Weak ETags (prefixed `W/"..."`) indicate content equivalence, not byte-for-byte identity. SAP services commonly return weak ETags.

```go
etag := traverse.ETag{Value: `"abc123"`, Weak: false}
fmt.Println(etag.String()) // "abc123"

weakEtag := traverse.ETag{Value: `"abc123"`, Weak: true}
fmt.Println(weakEtag.String()) // W/"abc123"
```

## Reading with ETag

`ReadWithETag` performs a GET and returns both the entity and its current ETag:

```go
var product Product
etag, err := client.From("Products").Key(1).ReadWithETag(ctx, &product)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("ETag: %s\n", etag.String())
```

HTTP exchange:

```
GET /Products(1)
< ETag: W/"datetime'2024-01-15T10%3A30%3A00'"
```

## Updating with ETag (PATCH + If-Match)

`UpdateWithETag` sends `If-Match: <etag>` with the PATCH request:

```go
err = client.From("Products").Key(1).UpdateWithETag(ctx, map[string]any{
    "UnitPrice": 24.99,
}, etag)
```

HTTP exchange:

```
PATCH /Products(1)
If-Match: W/"datetime'2024-01-15T10%3A30%3A00'"
Content-Type: application/json

{"UnitPrice": 24.99}
```

If the entity was modified by another client between your read and update, the server returns `412 Precondition Failed`.

## Replacing with ETag (PUT + If-Match)

`ReplaceWithETag` sends a full PUT with `If-Match`:

```go
err = client.From("Products").Key(1).ReplaceWithETag(ctx, fullProduct, etag)
```

## Deleting with ETag (DELETE + If-Match)

`DeleteWithETag` sends `If-Match` with the DELETE request:

```go
err = client.From("Products").Key(1).DeleteWithETag(ctx, etag)
```

## Upsert (If-None-Match: *)

`If-None-Match: *` tells the server to succeed only if the entity does NOT exist (create, not update). Use this for safe upsert semantics:

```go
err = client.From("Products").Key(newID).Upsert(ctx, product)
```

This generates:

```
PUT /Products(newID)
If-None-Match: *
```

- If the entity does not exist: creates it (201 Created)
- If the entity already exists: returns 412 Precondition Failed

## EntityWithETag

`EntityWithETag[T]` bundles an entity with its ETag for convenient passing between functions:

```go
type EntityWithETag[T any] struct {
    Entity T
    ETag   ETag
}

// Fetch and wrap
result, err := traverse.FetchWithETag[Product](ctx, client, "Products", 1)
// result.Entity is the product, result.ETag is the current ETag
```

## Optimistic Concurrency Pattern

A complete read-modify-write loop with retry:

```go
func updateProductPrice(ctx context.Context, client *traverse.Client, id int, newPrice float64) error {
    const maxRetries = 3
    for attempt := 0; attempt < maxRetries; attempt++ {
        var product Product
        etag, err := client.From("Products").Key(id).ReadWithETag(ctx, &product)
        if err != nil {
            return err
        }

        err = client.From("Products").Key(id).UpdateWithETag(ctx,
            map[string]any{"UnitPrice": newPrice},
            etag,
        )
        if err == nil {
            return nil // success
        }
        if !traverse.IsPreconditionFailed(err) {
            return err // unexpected error
        }
        // 412: another client modified the entity, retry
    }
    return fmt.Errorf("failed after %d retries", maxRetries)
}
```

## Handling 412 Precondition Failed

```go
err = client.From("Products").Key(1).UpdateWithETag(ctx, patch, etag)
if traverse.IsPreconditionFailed(err) {
    // Re-read the entity to get the new ETag and current values
    etag, err = client.From("Products").Key(1).ReadWithETag(ctx, &product)
    // ... merge changes and retry
}
```

!!! tip "ETags in SAP"
    SAP ABAP Gateway services return weak ETags in the format `W/"datetime'2024-01-15T10%3A30%3A00'"`. traverse handles these automatically. See [SAP Compatibility](sap.md) for SAP-specific details.

## Related Pages

- [CRUD Operations](crud.md) - Basic create/read/update/delete
- [Entity Change Tracking](change-tracking.md) - Emit only changed fields in PATCH
- [ETag Types Reference](../reference/etag.md) - Full API reference
