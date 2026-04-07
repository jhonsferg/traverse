# Schema Validation

traverse can validate `$filter` and `$orderby` expressions against a declared entity schema before sending a request. This catches typos and invalid field names at the client side rather than receiving a cryptic server error.

## Declaring a Schema

```go
type Product struct {
    ID       int     `json:"ProductID"`
    Name     string  `json:"ProductName"`
    Price    float64 `json:"UnitPrice"`
    Category string  `json:"Category"`
    InStock  bool    `json:"InStock"`
}

schema := traverse.NewEntitySchema(
    traverse.Field("ProductID"),
    traverse.Field("ProductName"),
    traverse.Field("UnitPrice"),
    traverse.Field("Category"),
    traverse.Field("InStock"),
)
```

## Attaching a Schema to a QueryBuilder

```go
client, err := traverse.New(traverse.WithBaseURL("https://api.example.com/odata/"))
if err != nil {
    log.Fatal(err)
}

products := traverse.From[Product](client, "Products").
    WithSchema(schema)
```

Once attached, every call to `.Filter()` and `.OrderBy()` is validated automatically.

## What Gets Validated

### Filter Expressions

```go
// OK - known field
results, err := products.Filter("ProductName eq 'Chai'").List(ctx)

// Error - unknown field "Title" (typo for ProductName)
results, err = products.Filter("Title eq 'Chai'").List(ctx)
// -> SchemaValidationError{Field: "Title", Message: "unknown field"}
```

### OrderBy Expressions

```go
// OK
results, err := products.OrderBy("UnitPrice desc").List(ctx)

// Error - unknown sort field
results, err = products.OrderBy("Cost asc").List(ctx)
// -> SchemaValidationError{Field: "Cost", Message: "unknown field"}
```

## Error Type

```go
type SchemaValidationError struct {
    Field   string // the unknown field name
    Message string // human-readable explanation
}

func (e *SchemaValidationError) Error() string
```

Usage:

```go
var ve *traverse.SchemaValidationError
if errors.As(err, &ve) {
    fmt.Printf("invalid field %q: %s\n", ve.Field, ve.Message)
}
```

## Full Example

```go
type Order struct {
    ID         int     `json:"OrderID"`
    CustomerID string  `json:"CustomerID"`
    Total      float64 `json:"Total"`
    Status     string  `json:"Status"`
    OrderDate  string  `json:"OrderDate"`
}

schema := traverse.NewEntitySchema(
    traverse.Field("OrderID"),
    traverse.Field("CustomerID"),
    traverse.Field("Total"),
    traverse.Field("Status"),
    traverse.Field("OrderDate"),
)

orders := traverse.From[Order](client, "Orders").WithSchema(schema)

// Build the query safely
results, err := orders.
    Filter("CustomerID eq 'ALFKI' and Total gt 100").
    OrderBy("OrderDate desc").
    Top(10).
    List(ctx)
if err != nil {
    var ve *traverse.SchemaValidationError
    if errors.As(err, &ve) {
        log.Fatalf("query uses unknown field %q", ve.Field)
    }
    log.Fatal(err)
}
```

## Combining with the Type-safe Filter Builder

Schema validation works well with the [type-safe filter builder](../guides/filter-builder.md):

```go
filter := traverse.F("CustomerID").Eq("ALFKI").
    And(traverse.F("Total").Gt(100))

results, err := orders.
    WithSchema(schema).
    FilterBy(filter).
    List(ctx)
```

## Notes

- Schema validation runs client-side before any network call.
- An entity without a schema passes all requests through unchanged.
- Validation only checks field names, not types. Server-side type errors are still possible.
- Navigation property expansions are not validated against the schema.
