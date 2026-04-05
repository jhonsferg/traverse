# Query Builder API

The `CollectionBuilder` provides a fluent interface for building OData query options.

## Filter expressions

### String expressions

```go
client.Collection("Products").
    Filter("Price gt 10 and Category eq 'Electronics'").
    List(ctx, &products)
```

### Lambda DSL

Use the type-safe lambda helpers to avoid string concatenation:

```go
import "github.com/jhonsferg/traverse/query"

client.Collection("Products").
    FilterExpr(
        query.And(
            query.Field("Price").Gt(10),
            query.Field("Category").Eq("Electronics"),
        ),
    ).
    List(ctx, &products)
```

### Lambda filter functions

| Function | OData equivalent |
|----------|-----------------|
| `query.Field(name).Eq(v)` | `field eq v` |
| `query.Field(name).Ne(v)` | `field ne v` |
| `query.Field(name).Gt(v)` | `field gt v` |
| `query.Field(name).Ge(v)` | `field ge v` |
| `query.Field(name).Lt(v)` | `field lt v` |
| `query.Field(name).Le(v)` | `field le v` |
| `query.Field(name).Contains(s)` | `contains(field,'s')` |
| `query.Field(name).StartsWith(s)` | `startswith(field,'s')` |
| `query.Field(name).EndsWith(s)` | `endswith(field,'s')` |
| `query.And(a, b)` | `a and b` |
| `query.Or(a, b)` | `a or b` |
| `query.Not(a)` | `not a` |
| `query.Any(nav, lambda)` | `nav/any(x: lambda)` |
| `query.All(nav, lambda)` | `nav/all(x: lambda)` |

## Select

```go
client.Collection("Products").
    Select("ProductID", "ProductName", "UnitPrice").
    List(ctx, &products)
// $select=ProductID,ProductName,UnitPrice
```

## Expand

```go
client.Collection("Orders").
    Expand("Customer", "OrderDetails/Product").
    List(ctx, &orders)
// $expand=Customer,OrderDetails/Product
```

### Expand with nested options (OData v4)

```go
client.Collection("Orders").
    ExpandWith("OrderDetails", query.ExpandOptions{
        Select: []string{"Quantity", "UnitPrice"},
        Filter: query.Field("Quantity").Gt(5),
        OrderBy: "UnitPrice desc",
    }).
    List(ctx, &orders)
// $expand=OrderDetails($select=Quantity,UnitPrice;$filter=Quantity gt 5;$orderby=UnitPrice desc)
```

## OrderBy

```go
// Ascending
client.Collection("Products").OrderBy("ProductName").List(ctx, &p)

// Descending
client.Collection("Products").OrderBy("Price desc").List(ctx, &p)

// Multiple fields
client.Collection("Products").OrderBy("Category", "Price desc").List(ctx, &p)
```

## Pagination

```go
// Manual offset/limit
client.Collection("Products").Top(20).Skip(40).List(ctx, &p)

// Typed paginator
paginator, _ := client.Collection("Products").
    OrderBy("ProductID").
    Paginate[Product](ctx, traverse.PageOptions{Size: 20})
```

## Apply ($apply - aggregation)

```go
client.Collection("Sales").
    Apply("groupby((Region),aggregate(Amount with sum as Total))").
    List(ctx, &summary)
```

## Count

```go
result, err := client.Collection("Products").Count().List(ctx, &products)
fmt.Println(result.Count) // total matching records
```

## See also

- [Lambda Filters Guide](../guides/lambda-filters.md)
- [Paginator[T]](paginator.md)
- [Client Reference](client.md)
