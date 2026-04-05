# Working with Generated Code

After running `traverse-gen`, you get three files in your output directory. This guide explains how to use them effectively.

## Initialising the generated client

```go
import (
    "github.com/jhonsferg/traverse"
    "myapp/gen/northwind"
)

base := traverse.New(traverse.Config{
    BaseURL: "https://services.odata.org/V4/Northwind/Northwind.svc/",
})

client := northwind.NewClient(base)
```

## Listing entities

```go
products, err := client.Products().
    Filter(northwind.ProductFilter.Price.Gt(10)).
    OrderBy(northwind.Product.ProductName).
    Top(25).
    List(ctx)
if err != nil {
    log.Fatal(err)
}
for _, p := range products {
    fmt.Println(p.ProductName, p.UnitPrice)
}
```

## Getting a single entity

```go
product, err := client.Products().Get(ctx, 42)
if err != nil {
    log.Fatal(err)
}
fmt.Println(product.ProductName)
```

## Creating an entity

```go
newProduct := northwind.Product{
    ProductName:     "New Widget",
    QuantityPerUnit: "10 boxes",
    UnitPrice:       ptr(9.99),
    Discontinued:    false,
}
created, err := client.Products().Create(ctx, newProduct)
```

## Updating an entity

```go
patch := northwind.ProductPatch{
    UnitPrice: ptr(12.50),
}
err := client.Products().Update(ctx, 42, patch)
```

## Deleting an entity

```go
err := client.Products().Delete(ctx, 42)
```

## Expanding navigation properties

```go
orders, err := client.Orders().
    Expand(northwind.Order.OrderDetails).
    Filter(northwind.OrderFilter.CustomerID.Eq("ALFKI")).
    List(ctx)
for _, o := range orders {
    fmt.Printf("Order %d: %d items\n", o.OrderID, len(o.OrderDetails))
}
```

## Typed pagination

```go
page, err := client.Products().
    OrderBy(northwind.Product.ProductID).
    Paginate(ctx, traverse.PageOptions{Size: 20})

for page.Next(ctx) {
    for _, p := range page.Items() {
        fmt.Println(p.ProductName)
    }
}
if err := page.Err(); err != nil {
    log.Fatal(err)
}
```

## Keeping generated code up to date

Add a `go generate` comment near the top of a non-generated file in the same package:

```go
// in api/api.go
//go:generate go run github.com/jhonsferg/traverse/cmd/traverse-gen@latest --metadata ../../api/metadata.xml --out . --package northwind
```

Run `go generate ./api/...` whenever the service metadata changes.

!!! tip "Commit generated code"
    Generated files should be committed to your repository. This makes diffs reviewable and avoids requiring the generator at runtime or in CI (unless you want to detect drift).

## See also

- [Type Mapping](type-mapping.md)
- [Query Builder API](../reference/query-builder.md)
- [Paginator[T]](../reference/paginator.md)
