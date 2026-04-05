# Lambda Filters

Lambda filter expressions (`any()` and `all()`) filter entities based on conditions applied to collection-valued navigation properties or inline collections. traverse provides a type-safe DSL for building lambda expressions.

## When to Use Lambda Filters

Use `any()` to find entities where at least one item in a sub-collection matches:

```
Products?$filter=Tags/any(t: t eq 'featured')
```

Use `all()` to find entities where every item in a sub-collection matches:

```
Orders?$filter=OrderDetails/all(d: d/Quantity gt 0)
```

## LambdaAny

```go
traverse.LambdaAny(field string, fn func(b *traverse.LambdaBuilder) string) string
```

Generates an `any()` lambda expression:

```go
filter := traverse.LambdaAny("Tags", func(b *traverse.LambdaBuilder) string {
    return b.Field("").Eq("featured")
})
// Tags/any(x0: x0 eq 'featured')

client.From("Products").Filter(filter).Into(ctx, &products)
```

## LambdaAll

```go
traverse.LambdaAll(field string, fn func(b *traverse.LambdaBuilder) string) string
```

Generates an `all()` lambda expression:

```go
filter := traverse.LambdaAll("OrderDetails", func(b *traverse.LambdaBuilder) string {
    return b.Field("Quantity").Gt(0)
})
// OrderDetails/all(x0: x0/Quantity gt 0)
```

## LambdaBuilder Methods

The `LambdaBuilder` passed to your function provides fluent operators:

### Field

```go
b.Field(name string) *LambdaCondition
```

Returns a condition builder for the specified field. Use `""` for the lambda variable itself (for scalar collections), or a field name for navigation property collections.

### LambdaCondition Operators

| Method | OData operator | Example |
|--------|---------------|---------|
| `Eq(v)` | `eq` | `b.Field("Status").Eq("Active")` |
| `Ne(v)` | `ne` | `b.Field("Status").Ne("Deleted")` |
| `Lt(v)` | `lt` | `b.Field("Price").Lt(100)` |
| `Le(v)` | `le` | `b.Field("Price").Le(100)` |
| `Gt(v)` | `gt` | `b.Field("Quantity").Gt(0)` |
| `Ge(v)` | `ge` | `b.Field("Score").Ge(50)` |
| `Contains(v)` | `contains()` | `b.Field("Name").Contains("coffee")` |
| `StartsWith(v)` | `startswith()` | `b.Field("Name").StartsWith("A")` |
| `EndsWith(v)` | `endswith()` | `b.Field("Name").EndsWith("Ltd")` |

## Combining Lambda with Filter

Lambda expressions return plain strings, so you can combine them with `Filter()`:

```go
// Orders with at least one detail over $100
lambda := traverse.LambdaAny("OrderDetails", func(b *traverse.LambdaBuilder) string {
    return b.Field("UnitPrice").Gt(100.0)
})

var orders []Order
err := client.From("Orders").
    Filter(lambda).
    Filter("Freight gt 0").  // AND'd with the lambda
    Top(20).
    Into(ctx, &orders)
// $filter=OrderDetails/any(x0: x0/UnitPrice gt 100) and Freight gt 0
```

Multiple lambda expressions can be AND'd:

```go
hasExpensive := traverse.LambdaAny("OrderDetails", func(b *traverse.LambdaBuilder) string {
    return b.Field("UnitPrice").Gt(100.0)
})
allDelivered := traverse.LambdaAll("Shipments", func(b *traverse.LambdaBuilder) string {
    return b.Field("Status").Eq("Delivered")
})

err := client.From("Orders").
    Filter(hasExpensive + " and " + allDelivered).
    Into(ctx, &orders)
```

## Generated OData URLs

```go
// Scalar collection: Tags/any(t: t eq 'featured')
traverse.LambdaAny("Tags", func(b *traverse.LambdaBuilder) string {
    return b.Field("").Eq("featured")
})

// Navigation collection field: OrderDetails/any(x0: x0/Quantity gt 5)
traverse.LambdaAny("OrderDetails", func(b *traverse.LambdaBuilder) string {
    return b.Field("Quantity").Gt(5)
})

// Nested navigation: Categories/any(c: c/Products/any(p: p/Price gt 100))
outer := traverse.LambdaAny("Categories", func(b *traverse.LambdaBuilder) string {
    inner := traverse.LambdaAny("Products", func(b2 *traverse.LambdaBuilder) string {
        return b2.Field("Price").Gt(100)
    })
    return inner
})
```

## Real-World Example

```go
// Find products that have any review with rating >= 4 AND all inventory
// entries show in-stock status

hasGoodReview := traverse.LambdaAny("Reviews", func(b *traverse.LambdaBuilder) string {
    return b.Field("Rating").Ge(4)
})

allInStock := traverse.LambdaAll("InventoryItems", func(b *traverse.LambdaBuilder) string {
    return b.Field("Status").Eq("InStock")
})

var products []Product
err := client.From("Products").
    Filter(hasGoodReview + " and " + allInStock).
    Select("ProductID", "ProductName", "UnitPrice").
    OrderBy("UnitPrice asc").
    Into(ctx, &products)
if err != nil {
    log.Fatal(err)
}
```

Generated URL:

```
/Products?$filter=Reviews/any(x0: x0/Rating ge 4) and InventoryItems/all(x1: x1/Status eq 'InStock')&$select=ProductID,ProductName,UnitPrice&$orderby=UnitPrice asc
```

!!! note "OData v4 only"
    Lambda expressions (`any()`/`all()`) are an OData v4 feature. OData v2 services do not support them. For SAP ABAP Gateway v2 services, use regular `$filter` expressions instead.

## Related Pages

- [Query Builder](query-builder.md) - The Filter() method
- [OData Primer](../odata-primer.md) - OData filter syntax background
