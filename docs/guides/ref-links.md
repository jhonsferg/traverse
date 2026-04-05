# $ref Link Operations

OData supports managing **relationships between entities** via `$ref` links. traverse provides `LinkTo` and `UnlinkFrom` methods to create, update, and delete navigation property references.

## Overview

In OData, relationships between entities are represented as links. For example, linking an Order to a Customer, or associating a User with a Group.

| Operation | HTTP Method | OData URL |
|-----------|-------------|-----------|
| Create link | PUT | `/EntitySet(key)/NavProperty/$ref` |
| Delete link (single) | DELETE | `/EntitySet(key)/NavProperty/$ref` |
| Delete link (collection member) | DELETE | `/EntitySet(key)/NavProperty(targetKey)/$ref` |

## LinkTo

Creates a reference link from the current entity to a target entity:

```go
client := traverse.NewClient(relayClient, "https://api.example.com/odata/")

type Order struct {
    ID         int    `json:"OrderID"`
    CustomerID string `json:"CustomerID,omitempty"`
}

qb := traverse.From[Order](client, "Orders")

// Link Order 1001 to Customer "ALFKI"
err := qb.LinkTo(ctx, 1001, "Customer", "Customers", "ALFKI")
```

This sends:
```http
PUT /Orders(1001)/Customer/$ref
Content-Type: application/json

{"@odata.id": "https://api.example.com/odata/Customers('ALFKI')"}
```

### Signature

```go
func (qb *QueryBuilder[T]) LinkTo(
    ctx         context.Context,
    key         any,      // key of the source entity
    navProperty string,   // navigation property name
    targetSet   string,   // target entity set name
    targetKey   any,      // key of the target entity
) error
```

## UnlinkFrom

Removes a reference link. Behavior depends on whether the navigation property is a single reference or a collection:

### Single Navigation Property

```go
// Unlink Order 1001 from its Customer
err := qb.UnlinkFrom(ctx, 1001, "Customer")
```

Sends: `DELETE /Orders(1001)/Customer/$ref`

### Collection Navigation Property

```go
// Remove User 42 from Group 7's Members collection
type Group struct{ ID int }
groupQB := traverse.From[Group](client, "Groups")

err := groupQB.UnlinkFrom(ctx, 7, "Members", 42)
```

Sends: `DELETE /Groups(7)/Members(42)/$ref`

### Signature

```go
func (qb *QueryBuilder[T]) UnlinkFrom(
    ctx         context.Context,
    key         any,      // key of the source entity
    navProperty string,   // navigation property name
    targetKey   ...any,   // optional: key in collection nav property
) error
```

## Error Handling

| Error | Meaning |
|-------|---------|
| `ErrEntityNotFound` | Source or target entity does not exist (404) |
| `ErrConcurrencyConflict` | ETag mismatch or constraint violation (409/412) |

```go
import "errors"

err := qb.LinkTo(ctx, 1001, "Customer", "Customers", "NOTEXIST")
if errors.Is(err, traverse.ErrEntityNotFound) {
    // Handle missing entity
}
```

## Key Types

Both string and integer keys are supported:

```go
// String key
qb.LinkTo(ctx, "order-uuid-123", "Customer", "Customers", "cust-uuid-456")

// Integer key
qb.LinkTo(ctx, 42, "Department", "Departments", 7)
```

## Practical Example: Assigning a Manager

```go
type Employee struct {
    ID         int    `json:"EmployeeID"`
    Name       string `json:"LastName"`
    ManagerID  *int   `json:"ReportsTo,omitempty"`
}

employees := traverse.From[Employee](client, "Employees")

// Employee 3 now reports to Employee 2
err := employees.LinkTo(ctx, 3, "Manager", "Employees", 2)
if err != nil {
    log.Fatalf("failed to assign manager: %v", err)
}

// Remove the manager relationship
err = employees.UnlinkFrom(ctx, 3, "Manager")
```
