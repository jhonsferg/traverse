# Type-safe Filter Builder

Instead of writing raw OData filter strings, traverse provides a fluent, type-safe `FilterExpr` API that constructs correct filter expressions with proper quoting, escaping, and operator formatting.

## Entry Point

```go
import "github.com/jhonsferg/traverse"

// F(field) starts a filter expression for a named field
expr := traverse.F("LastName").Eq("Smith")
fmt.Println(expr.String()) // LastName eq 'Smith'
```

## Comparison Operators

```go
traverse.F("Age").Eq(30)          // Age eq 30
traverse.F("Age").Ne(30)          // Age ne 30
traverse.F("Price").Lt(100.0)     // Price lt 100
traverse.F("Price").Le(99.99)     // Price le 99.99
traverse.F("Score").Gt(0)         // Score gt 0
traverse.F("Score").Ge(1)         // Score ge 1
```

## String Functions

```go
traverse.F("Name").Contains("acme")     // contains(Name,'acme')
traverse.F("Name").StartsWith("ACME")   // startswith(Name,'ACME')
traverse.F("Email").EndsWith(".com")    // endswith(Email,'.com')
```

## Logical Combinators

```go
// And
filter := traverse.And(
    traverse.F("Category").Eq("Beverages"),
    traverse.F("Price").Lt(20.0),
)
// (Category eq 'Beverages') and (Price lt 20)

// Or
filter = traverse.Or(
    traverse.F("Status").Eq("Active"),
    traverse.F("Status").Eq("Pending"),
)
// (Status eq 'Active') or (Status eq 'Pending')

// Not
filter = traverse.Not(traverse.F("Discontinued").Eq(true))
// not (Discontinued eq true)
```

## Nesting

Combinators can be nested arbitrarily:

```go
filter := traverse.And(
    traverse.Or(
        traverse.F("Region").Eq("US"),
        traverse.F("Region").Eq("EU"),
    ),
    traverse.And(
        traverse.F("Active").Eq(true),
        traverse.F("Total").Gt(500.0),
    ),
)
// ((Region eq 'US') or (Region eq 'EU')) and ((Active eq true) and (Total gt 500))
```

## Value Type Handling

The builder automatically formats values correctly for OData:

| Go type | OData format |
|---------|-------------|
| `string` | `'value'` (single-quoted, inner `'` escaped as `''`) |
| `int`, `int32`, `int64` | `42` |
| `float32`, `float64` | `3.14` |
| `bool` | `true` / `false` |
| `time.Time` | `2024-01-15T10:30:00Z` |
| `nil` | `null` |

```go
// String with apostrophe - auto-escaped
traverse.F("Name").Eq("O'Brien")  // Name eq 'O''Brien'

// Time value
t := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
traverse.F("Created").Ge(t)  // Created ge 2024-01-15T00:00:00Z

// Null check
traverse.F("DeletedAt").Eq(nil)  // DeletedAt eq null
```

## Using with QueryBuilder

Pass a `*FilterExpr` directly to `FilterBy`:

```go
type Product struct {
    ID    int     `json:"ProductID"`
    Name  string  `json:"ProductName"`
    Price float64 `json:"UnitPrice"`
}

products := traverse.From[Product](client, "Products")

filter := traverse.And(
    traverse.F("ProductName").StartsWith("C"),
    traverse.F("UnitPrice").Lt(20.0),
)

results, err := products.FilterBy(filter).OrderBy("UnitPrice").List(ctx)
```

This is equivalent to (and safer than):

```go
results, err = products.Filter("startswith(ProductName,'C') and UnitPrice lt 20").List(ctx)
```

## Combining with Schema Validation

The filter builder pairs naturally with [schema validation](schema-validation.md):

```go
schema := traverse.NewEntitySchema(
    traverse.Field("ProductID"),
    traverse.Field("ProductName"),
    traverse.Field("UnitPrice"),
)

filter := traverse.And(
    traverse.F("ProductName").Contains("chai"),
    traverse.F("UnitPrice").Le(50.0),
)

results, err := products.
    WithSchema(schema).
    FilterBy(filter).
    List(ctx)
```

## FilterExpr API Reference

### Constructors

```go
func F(field string) *FilterExpr
func And(exprs ...*FilterExpr) *FilterExpr
func Or(exprs ...*FilterExpr) *FilterExpr
func Not(expr *FilterExpr) *FilterExpr
```

### Methods on *FilterExpr

```go
// Comparison
func (e *FilterExpr) Eq(value any) *FilterExpr
func (e *FilterExpr) Ne(value any) *FilterExpr
func (e *FilterExpr) Lt(value any) *FilterExpr
func (e *FilterExpr) Le(value any) *FilterExpr
func (e *FilterExpr) Gt(value any) *FilterExpr
func (e *FilterExpr) Ge(value any) *FilterExpr

// String functions
func (e *FilterExpr) Contains(value string) *FilterExpr
func (e *FilterExpr) StartsWith(value string) *FilterExpr
func (e *FilterExpr) EndsWith(value string) *FilterExpr

// Output
func (e *FilterExpr) Build() string
func (e *FilterExpr) String() string
```

### QueryBuilder method

```go
func (qb *QueryBuilder[T]) FilterBy(expr *FilterExpr) *QueryBuilder[T]
```
