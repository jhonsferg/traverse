# Entity Change Tracking

Entity change tracking lets you load an entity, modify specific fields, and then save only the changed fields as a PATCH request. This is the safest and most efficient way to update entities - you never accidentally overwrite fields you did not intend to change.

## TrackEntity

`TrackEntity[T]` wraps an entity in a `TrackedEntity[T]` that records which fields have changed:

```go
var product Product
err := client.From("Products").Key(1).Into(ctx, &product)
if err != nil {
    log.Fatal(err)
}

tracked := traverse.TrackEntity(product)
```

## Making Changes

Use `Set` to change a field. Only fields changed through `Set` are considered dirty:

```go
tracked.Set("UnitPrice", 29.99)
tracked.Set("UnitsInStock", 100)
```

You can also modify the entity directly and call `MarkDirty`:

```go
tracked.Get().UnitPrice = 29.99
tracked.MarkDirty("UnitPrice")
```

## Checking Dirty State

```go
fmt.Println(tracked.IsDirty())          // true
fmt.Println(tracked.DirtyFields())      // ["UnitPrice", "UnitsInStock"]

changes := tracked.Changes()
// map[string]any{"UnitPrice": 29.99, "UnitsInStock": 100}
```

## SaveChanges

`SaveChanges` sends a PATCH request containing only the dirty fields:

```go
err = tracked.SaveChanges(ctx, client, "Products", 1)
```

HTTP exchange:

```
PATCH /Products(1)
Content-Type: application/json

{"UnitPrice": 29.99, "UnitsInStock": 100}
```

Fields like `ProductName`, `CategoryID`, etc. are not included even though they exist on the struct.

## MarshalJSON

`TrackedEntity[T]` implements `json.Marshaler` and emits only dirty fields:

```go
data, err := json.Marshal(tracked)
// {"UnitPrice": 29.99, "UnitsInStock": 100}
```

This makes `TrackedEntity[T]` safe to pass directly to `Update`:

```go
err = client.From("Products").Key(1).Update(ctx, tracked)
```

## Reset and Discard

`Reset` clears the dirty set but keeps the current values (marks as clean):

```go
tracked.Reset()
fmt.Println(tracked.IsDirty()) // false
```

`Discard` reverts the entity to its original state (both values and dirty set):

```go
tracked.Set("UnitPrice", 999.99)
tracked.Discard()

p := tracked.Get()
fmt.Println(p.UnitPrice) // original value
fmt.Println(tracked.IsDirty()) // false
```

## Full Example

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/jhonsferg/traverse"
)

type Product struct {
    ID          int     `json:"ProductID"`
    Name        string  `json:"ProductName"`
    UnitPrice   float64 `json:"UnitPrice"`
    UnitsInStock int    `json:"UnitsInStock"`
    Discontinued bool   `json:"Discontinued"`
}

func main() {
    client, _ := traverse.New(traverse.WithBaseURL("https://example.com/odata"))
    defer client.Close()
    ctx := context.Background()

    // Load entity
    var product Product
    err := client.From("Products").Key(1).Into(ctx, &product)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Loaded: %s at $%.2f\n", product.Name, product.UnitPrice)

    // Track and modify
    tracked := traverse.TrackEntity(product)
    tracked.Set("UnitPrice", 24.99)
    tracked.Set("UnitsInStock", tracked.Get().UnitsInStock + 10)

    fmt.Println("Dirty fields:", tracked.DirtyFields())
    // Dirty fields: [UnitPrice, UnitsInStock]

    // Save only changed fields
    err = tracked.SaveChanges(ctx, client, "Products", 1)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println("Saved successfully")
    // PATCH body: {"UnitPrice": 24.99, "UnitsInStock": <updated value>}

    // Entity is still usable after save
    fmt.Println(tracked.IsDirty()) // false (reset after save)
}
```

## TrackedEntity[T] API

| Method | Description |
|--------|-------------|
| `Get() *T` | Returns a pointer to the tracked entity |
| `Set(field string, value any)` | Sets a field value and marks it dirty |
| `IsDirty() bool` | Returns true if any field has been modified |
| `DirtyFields() []string` | Returns the names of all dirty fields |
| `Changes() map[string]any` | Returns a map of dirty field names to new values |
| `Reset()` | Clears the dirty set, keeping current values |
| `Discard()` | Reverts all changes to the original loaded state |
| `MarshalJSON() ([]byte, error)` | Emits only dirty fields as JSON |
| `SaveChanges(ctx, client, entitySet, key)` | PATCHes only dirty fields |

## Related Pages

- [CRUD Operations](crud.md) - Standard update patterns
- [ETag & Concurrency](etag-concurrency.md) - Combine change tracking with ETags
- [Change Tracking Reference](../reference/change-tracking.md) - Full API reference
