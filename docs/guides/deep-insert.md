# Deep Insert

Deep insert creates a parent entity and its related child entities in a single POST request. The server creates both in the same transaction - if any child fails, the entire operation is rolled back.

## Basic Deep Insert

```go
type Order struct {
    ID         int           `json:"OrderID,omitempty"`
    CustomerID string        `json:"CustomerID"`
    Freight    float64       `json:"Freight"`
    Details    []OrderDetail `json:"OrderDetails"`
}

type OrderDetail struct {
    ProductID int     `json:"ProductID"`
    Quantity  int     `json:"Quantity"`
    UnitPrice float64 `json:"UnitPrice"`
}

newOrder := Order{
    CustomerID: "ALFKI",
    Freight:    15.50,
    Details: []OrderDetail{
        {ProductID: 1, Quantity: 2, UnitPrice: 18.00},
        {ProductID: 5, Quantity: 3, UnitPrice: 21.35},
    },
}

var createdOrder Order
err := client.From("Orders").CreateDeep(ctx, newOrder, &createdOrder)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Created order %d with %d details\n",
    createdOrder.ID, len(createdOrder.Details))
```

HTTP exchange:

```
POST /Orders HTTP/1.1
Content-Type: application/json

{
  "CustomerID": "ALFKI",
  "Freight": 15.50,
  "OrderDetails": [
    {"ProductID": 1, "Quantity": 2, "UnitPrice": 18.00},
    {"ProductID": 5, "Quantity": 3, "UnitPrice": 21.35}
  ]
}
```

## CreateDeepWithPrefer

`CreateDeepWithPrefer` lets you control the `Prefer` header, which tells the server what to return in the response:

```go
var created Order
err := client.From("Orders").CreateDeepWithPrefer(
    ctx,
    newOrder,
    &created,
    "return=representation", // return the full created entity
)
```

Common `Prefer` values:

| Value | Meaning |
|-------|---------|
| `return=representation` | Return the created entity (201 with body) |
| `return=minimal` | Return 204 No Content (faster, no body decode) |
| `odata.continue-on-error` | Continue creating children even if some fail |

## DeepInsertOptions

```go
opts := traverse.DeepInsertOptions{
    ReturnRepresentation: true,   // Prefer: return=representation
    ContinueOnError:      false,  // stop on first child failure
}

var created Order
err := client.From("Orders").CreateDeepWithOptions(ctx, newOrder, &created, opts)
```

## Setting Custom Headers

```go
var created Order
err := client.From("Orders").
    Header("Prefer", "return=representation").
    CreateDeep(ctx, newOrder, &created)
```

## Nested Multi-Level Deep Insert

Deep inserts can be arbitrarily nested:

```go
type Customer struct {
    ID      string   `json:"CustomerID,omitempty"`
    Name    string   `json:"CompanyName"`
    Orders  []Order  `json:"Orders"`
}

type Order struct {
    ID      int           `json:"OrderID,omitempty"`
    Freight float64       `json:"Freight"`
    Details []OrderDetail `json:"OrderDetails"`
}

newCustomer := Customer{
    Name: "New Corp",
    Orders: []Order{
        {
            Freight: 10.00,
            Details: []OrderDetail{
                {ProductID: 1, Quantity: 1, UnitPrice: 18.00},
            },
        },
    },
}

var created Customer
err := client.From("Customers").CreateDeep(ctx, newCustomer, &created)
```

## Partial Success with ContinueOnError

When `ContinueOnError` is set, the server attempts to create all children and reports which ones failed without rolling back the successful ones:

```go
opts := traverse.DeepInsertOptions{
    ContinueOnError: true,
}

var created Order
err := client.From("Orders").CreateDeepWithOptions(ctx, order, &created, opts)
if traverse.IsPartialSuccess(err) {
    partial := err.(*traverse.PartialSuccessError)
    fmt.Printf("Created %d details, %d failed\n",
        partial.Succeeded, partial.Failed)
}
```

!!! warning "Transaction semantics"
    Deep insert transaction behavior depends on the OData service. Most well-implemented services wrap the entire deep insert in a transaction. With `ContinueOnError`, the parent is always created but child creation is best-effort.

!!! note "OData version differences"
    - OData v4: Deep insert is a first-class feature
    - OData v2: Deep insert is service-dependent; SAP ABAP Gateway supports it via `$batch` changesets or inline navigation properties

## Related Pages

- [CRUD Operations](crud.md) - Standard create/update/delete
- [Batch Requests](batch.md) - Atomic multi-operation batches
- [SAP Compatibility](sap.md) - SAP deep insert behavior
