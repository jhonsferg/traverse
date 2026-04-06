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

## OData v2 Function Imports

OData v2 services use **Function Imports** rather than v4 Functions. The `FunctionImportBuilder` API has been expanded with `Method`, `Invoke`, and `InvokeCollection`.

### Basic usage

```go
// GET /GetTop10Orders()
result, err := client.FunctionImport("GetTop10Orders").
    Execute(ctx)
```

### Method -- choosing HTTP verb

Some Function Imports require `POST` (for example when parameters are too large for a URL, or the import has side effects):

```go
err := client.FunctionImport("ProcessQueue").
    Method("POST").
    Invoke(ctx, &result)
```

The default method is `GET`. Parameters for `GET` requests are encoded inline: `FuncName(key='value',...)`. For `POST` requests they are sent as a JSON body.

### Invoke -- typed single-entity response

`Invoke` unwraps the OData v2 `{"d":{...}}` envelope and decodes the payload into your struct:

```go
type Stats struct {
    Count int     `json:"Count"`
    Avg   float64 `json:"Avg"`
}

var stats Stats
err := client.FunctionImport("GetStats").
    Param("top", 10).
    Invoke(ctx, &stats)
```

Pass `nil` to discard the response body.

### InvokeCollection -- typed collection response

`InvokeCollection` handles all common OData v2/v4 collection wrappers automatically:

| Response format | Handled |
|-----------------|---------|
| `{"d":{"results":[...]}}` | OData v2 collection |
| `{"results":[...]}` | Flat results array |
| `{"value":[...]}` | OData v4 collection |
| `[...]` | Bare JSON array |

```go
var orders []Order
err := client.FunctionImport("GetOrders").
    Param("status", "pending").
    InvokeCollection(ctx, &orders)
```

### FunctionImportBuilder API summary

| Method | Description |
|--------|-------------|
| `Param(key, value)` | Add a URL or body parameter |
| `Method(m)` | Set HTTP method (`"GET"` or `"POST"`, default `"GET"`) |
| `Execute(ctx)` | Call and return `map[string]interface{}` |
| `Invoke(ctx, result)` | Call and decode single-entity response |
| `InvokeCollection(ctx, results)` | Call and decode collection response |

The generic helper `ExecuteFunctionImportAs[T]` is also available for cases where you prefer to pass the builder rather than a receiver pointer.

---

## CLI playground -- traverse actions

The `traverse` CLI includes an `actions` sub-command that connects to any OData service and lists every Function Import, Function, and Action exposed in its metadata. This is handy for exploring an unfamiliar service.

### Usage

```
traverse actions [options]

Options:
  -url     string   OData service URL (required)
  -user    string   Username for basic authentication
  -pass    string   Password for basic authentication
  -token   string   Bearer token for authentication
  -profile string   Use a saved connection profile
  -format  string   Output format: json, text (default "text")
  -timeout int      Request timeout in seconds (default 30)
```

### Text output

```
$ traverse actions -url https://services.odata.org/V4/Northwind/

OData Service Actions and Functions
====================================

Function Imports (OData v2):
---
  GetProductsByRating
    Return Type: Collection(NorthwindModel.Product)
    Parameters:
      - rating: Edm.Int32

Actions (OData v4):
---
  ResetDataSource
    Return Type:
```

### JSON output

```
$ traverse actions -url https://services.odata.org/V4/Northwind/ -format json
```

```json
{
  "actions": [...],
  "function_imports": [...],
  "functions": [...]
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
