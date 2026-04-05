# Typed Pagination

`Paginator[T]` handles OData pagination automatically, following `@odata.nextLink`, `@odata.skipToken`, and OData v2 `$skiptoken` responses. You get typed pages without managing URLs or tokens manually.

## Creating a Paginator

```go
type Product struct {
    ID    int     `json:"ProductID"`
    Name  string  `json:"ProductName"`
    Price float64 `json:"UnitPrice"`
}

paginator := traverse.NewPaginator[Product](
    client.From("Products").
        Filter("UnitPrice gt 0").
        OrderBy("ProductName asc").
        Top(20), // page size
)
```

`Top(20)` sets the page size for the first request. Subsequent pages are determined by the server's `nextLink`.

## Iterating Pages

```go
ctx := context.Background()

for paginator.HasMorePages() {
    page, err := paginator.NextPage(ctx)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Page: %d items\n", len(page))
    for _, p := range page {
        fmt.Printf("  %s: $%.2f\n", p.Name, p.Price)
    }
}
```

`HasMorePages` returns true initially (before the first request) and after any page that returned a nextLink. After the final page (no nextLink), it returns false.

## Total Count

If you requested `$count=true`, `TotalCount()` returns the total number of matching entities:

```go
paginator := traverse.NewPaginator[Product](
    client.From("Products").Count().Top(20),
)

page, err := paginator.NextPage(ctx)
// After first page:
total := paginator.TotalCount()
fmt.Printf("%d total products, first page has %d\n", total, len(page))
```

## Reset

`Reset` returns the paginator to its initial state, restarting from the first page:

```go
paginator.Reset()
// Now HasMorePages() returns true again, NextPage will fetch page 1
```

## Custom Decoder

`NewPaginatorWithDecoder` lets you provide a custom decode function for non-standard response shapes:

```go
paginator := traverse.NewPaginatorWithDecoder[Product](
    client.From("Products").Top(20),
    func(data []byte) ([]Product, error) {
        var wrapper struct {
            Value []Product `json:"value"`
        }
        if err := json.Unmarshal(data, &wrapper); err != nil {
            return nil, err
        }
        return wrapper.Value, nil
    },
)
```

## Processing All Pages

```go
func fetchAll[T any](ctx context.Context, p *traverse.Paginator[T]) ([]T, error) {
    var all []T
    for p.HasMorePages() {
        page, err := p.NextPage(ctx)
        if err != nil {
            return nil, err
        }
        all = append(all, page...)
    }
    return all, nil
}

products, err := fetchAll(ctx, paginator)
```

!!! tip "Paginator vs Streaming"
    Use `Paginator[T]` when you need random page access, want to display results a page at a time, or need `TotalCount()`. Use [Streaming](streaming.md) when you need to process all results in sequence with constant memory.

## OData nextLink Behavior

When the server returns more results than fit on one page, it includes a `@odata.nextLink` (OData v4) or `__next` (OData v2) in the response. The paginator extracts this URL and uses it verbatim for the next request - it may contain a `$skiptoken` that the server manages.

```json
{
    "@odata.context": "...",
    "@odata.count": 1234,
    "@odata.nextLink": "Products?$skiptoken=5",
    "value": [...]
}
```

The paginator follows this link exactly as provided by the server, so pagination works even when the server uses opaque continuation tokens.

## Paginator API Summary

| Method | Description |
|--------|-------------|
| `NewPaginator[T](qb)` | Create a paginator from a QueryBuilder |
| `NewPaginatorWithDecoder[T](qb, fn)` | Create a paginator with a custom decode function |
| `NextPage(ctx) ([]T, error)` | Fetch the next page |
| `HasMorePages() bool` | Returns true if there are more pages |
| `Reset()` | Restart from the first page |
| `TotalCount() int64` | Returns total count (requires $count=true) |

See [Paginator Reference](../reference/paginator.md) for the full API.

## Related Pages

- [Streaming](streaming.md) - Process all results at constant memory
- [Query Builder](query-builder.md) - Build the initial query
- [Delta Sync](delta-sync.md) - Sync only changed records since last run
