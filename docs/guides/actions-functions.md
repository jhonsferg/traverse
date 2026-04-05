# Actions and Functions

OData supports two kinds of server-side operations beyond standard CRUD:

- **Actions** - operations that may have side effects (like sending an email, processing a payment, or triggering a workflow). They use `POST`.
- **Functions** - read-only operations that return data without side effects (like computing a discount or running a report). They use `GET`.

Both can be **bound** (attached to a specific entity or collection) or **unbound** (standalone operations at the service root).

---

## Unbound Actions

Unbound actions are called at the service root level:

```go
client := traverse.NewClient(relayClient, "https://api.example.com/odata/")

// Simple action with no parameters
err := traverse.NewActionBuilder(client, "SendAllNotifications").
    Execute(ctx)

// Action with parameters
err = traverse.NewActionBuilder(client, "ResetDatabase").
    Param("ConfirmCode", "RESET-2024").
    Param("WipeAll", true).
    Execute(ctx)
```

This produces: `POST /SendAllNotifications` with body `{"ConfirmCode":"RESET-2024","WipeAll":true}`

---

## Bound Actions

Bound actions are attached to a specific entity instance or collection:

```go
type Order struct {
    ID     int    `json:"OrderID"`
    Status string `json:"Status"`
}

orders := traverse.From[Order](client, "Orders")

// Bound to a specific entity: POST /Orders(1001)/Cancel
err := orders.Action(ctx, 1001, "Cancel").
    Param("Reason", "Customer request").
    Execute(ctx)

// Bound to collection: POST /Orders/CloseAll
err = orders.CollectionAction(ctx, "CloseAll").
    Param("Before", "2024-01-01").
    Execute(ctx)
```

---

## Unbound Functions

Unbound functions return data and support typed results:

```go
type DiscountResult struct {
    Code    string  `json:"DiscountCode"`
    Percent float64 `json:"Percent"`
}

// Function with parameters: GET /GetBestDiscount(CustomerID='ALFKI',MinOrder=100)
result, err := traverse.NewFunctionBuilder[DiscountResult](client, "GetBestDiscount").
    Param("CustomerID", "ALFKI").
    Param("MinOrder", 100).
    Execute(ctx)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Use code %s for %.0f%% off\n", result.Code, result.Percent)
```

---

## Bound Functions

Bound functions are attached to specific entities:

```go
type Invoice struct {
    Number string  `json:"InvoiceNumber"`
    Total  float64 `json:"Total"`
}

orders := traverse.From[Order](client, "Orders")

// GET /Orders(1001)/GetInvoice(Format='PDF')
invoice, err := orders.Function(ctx, 1001, "GetInvoice", new(Invoice)).
    Param("Format", "PDF").
    Execute(ctx)
```

---

## Actions vs Functions at a Glance

| Property | Action | Function |
|----------|--------|----------|
| HTTP method | POST | GET |
| Side effects | Yes | No |
| Parameters in body | Yes (JSON) | Yes (URL inline) |
| Can return data | Yes | Yes |
| Idempotent | Not required | Required |

---

## ActionBuilder API

```go
builder := traverse.NewActionBuilder(client, "ActionName")
builder.Param(name string, value any) *ActionBuilder
builder.Execute(ctx context.Context) error
```

For bound actions via `QueryBuilder`:

```go
// Bound to entity
qb.Action(ctx, key any, actionName string) *ActionBuilder

// Bound to collection
qb.CollectionAction(ctx, actionName string) *ActionBuilder
```

---

## FunctionBuilder API

```go
builder := traverse.NewFunctionBuilder[T](client, "FunctionName")
builder.Param(name string, value any) *FunctionBuilder[T]
builder.Execute(ctx context.Context) (*T, error)
```

For bound functions via `QueryBuilder`:

```go
qb.Function(ctx, key any, functionName string, result *T) *FunctionBuilder[T]
qb.CollectionFunction(ctx, functionName string, result *T) *FunctionBuilder[T]
```

---

## Practical Example: Approval Workflow

```go
type ApprovalResult struct {
    Approved  bool   `json:"Approved"`
    ApprovedBy string `json:"ApprovedBy"`
    Comment   string `json:"Comment"`
}

type PurchaseOrder struct {
    ID     int    `json:"POID"`
    Amount float64 `json:"Amount"`
    Status string `json:"Status"`
}

pos := traverse.From[PurchaseOrder](client, "PurchaseOrders")

// Submit for approval (action - has side effects)
err := pos.Action(ctx, 42, "SubmitForApproval").
    Param("Urgency", "HIGH").
    Param("NotifyEmail", "manager@example.com").
    Execute(ctx)

// Check approval status (function - read-only)
result, err := pos.Function(ctx, 42, "GetApprovalStatus", new(ApprovalResult)).
    Execute(ctx)
if result.Approved {
    fmt.Printf("Approved by %s: %s\n", result.ApprovedBy, result.Comment)
}
```

---

## Error Handling

```go
err := orders.Action(ctx, 9999, "Cancel").Execute(ctx)
if errors.Is(err, traverse.ErrEntityNotFound) {
    // Order 9999 does not exist
}
if errors.Is(err, traverse.ErrActionNotFound) {
    // Server does not expose this action
}
```
