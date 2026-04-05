# Paginator[T]

`Paginator[T]` is a generic iterator for OData collections that handles all standard OData pagination strategies automatically.

## Type definition

```go
type Paginator[T any] struct { /* unexported */ }

func (p *Paginator[T]) Next(ctx context.Context) bool
func (p *Paginator[T]) Items() []T
func (p *Paginator[T]) Err() error
func (p *Paginator[T]) Page() int
func (p *Paginator[T]) TotalCount() (int, bool)
func (p *Paginator[T]) Close()
```

## PageOptions

```go
type PageOptions struct {
    // Size is the number of records per page. Default: 100.
    Size int

    // Strategy selects the pagination mode. Default: auto-detect.
    Strategy PaginationStrategy

    // MaxPages limits total pages fetched. 0 = unlimited.
    MaxPages int
}

type PaginationStrategy int

const (
    PaginationAuto         PaginationStrategy = iota // auto-detect
    PaginationNextLink                               // @odata.nextLink
    PaginationSkipToken                              // $skiptoken
    PaginationOffsetLimit                            // $top/$skip
    PaginationCursor                                 // custom cursor header
)
```

## Basic usage

```go
paginator, err := client.Collection("Products").
    Filter("Discontinued eq false").
    OrderBy("ProductName").
    Paginate[Product](ctx, traverse.PageOptions{Size: 50})
if err != nil {
    log.Fatal(err)
}
defer paginator.Close()

for paginator.Next(ctx) {
    for _, product := range paginator.Items() {
        fmt.Println(product.ProductName)
    }
}
if err := paginator.Err(); err != nil {
    log.Fatal(err)
}
```

## Collecting all pages

```go
var all []Product
for paginator.Next(ctx) {
    all = append(all, paginator.Items()...)
}
```

## Total count

When `$count=true` is included, `TotalCount` returns the server-reported total:

```go
paginator, _ := client.Collection("Products").Count().
    Paginate[Product](ctx, traverse.PageOptions{Size: 20})

for paginator.Next(ctx) {
    if total, ok := paginator.TotalCount(); ok {
        fmt.Printf("Page %d (total: %d)\n", paginator.Page(), total)
    }
    // process paginator.Items()
}
```

## Limiting pages

```go
// Fetch at most 3 pages regardless of more data
paginator, _ := client.Collection("Orders").
    Paginate[Order](ctx, traverse.PageOptions{
        Size:     100,
        MaxPages: 3,
    })
```

## See also

- [Typed Pagination guide](../guides/pagination.md)
- [Query Builder API](query-builder.md)
- [Client Reference](client.md)
