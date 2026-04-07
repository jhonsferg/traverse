# traverse

**A declarative OData v2/v4 client for Go**

[![Go Version](https://img.shields.io/badge/Go-1.24%2B-00ADD8?style=flat-square&logo=go)](https://pkg.go.dev/github.com/jhonsferg/traverse)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg?style=flat-square)](https://github.com/jhonsferg/traverse/blob/master/LICENSE)
[![pkg.go.dev](https://img.shields.io/badge/pkg.go.dev-reference-007D9C?style=flat-square&logo=go)](https://pkg.go.dev/github.com/jhonsferg/traverse)

traverse is a declarative OData v2/v4 client for Go, built on top of [relay](https://github.com/jhonsferg/relay) for HTTP transport. It handles all OData protocol details - pagination, CSRF tokens, ETag concurrency, delta sync, batch requests, async long-running operations - so you can focus on the data.

---

## :rocket: Why traverse?

<div class="grid cards" markdown>

- :material-filter: **Fluent Query Builder**

    Compose OData queries with a type-safe builder: `$filter`, `$select`, `$expand`, `$orderby`, `$top`, `$skip`, `$count`, `$search`, `$apply`.

- :material-sync: **Full CRUD + ETags**

    Create, Read, Update (PATCH/PUT), Delete with full ETag-aware variants for optimistic concurrency control.

- :material-track-changes: **Entity Change Tracking**

    Track field-level changes, emit PATCH bodies with only the dirty fields, and roll back with Discard.

- :material-format-list-numbered: **Typed Pagination**

    Generic `Paginator[T]` follows `@odata.nextLink` and `$skiptoken` automatically.

- :material-broadcast: **Streaming**

    Channel-based streaming for large datasets. Process millions of records at constant memory.

- :material-package-variant: **Batch Requests**

    Compose multi-operation `$batch` requests with atomic changesets.

- :material-delta: **Delta Sync**

    Incremental updates via OData delta links - sync only what changed since last run.

- :material-robot: **Code Generation**

    Generate typed Go structs and query builders from any `$metadata` EDMX document.

</div>

---

## :vs: Comparison

| Feature | traverse | go-odata | raw http.Client |
|---------|----------|----------|-----------------|
| OData v2 | :white_check_mark: | :white_check_mark: | manual |
| OData v4 | :white_check_mark: | partial | manual |
| Fluent query builder | :white_check_mark: | limited | manual |
| ETag concurrency | :white_check_mark: | :x: | manual |
| Entity change tracking | :white_check_mark: | :x: | :x: |
| Generic paginator | :white_check_mark: | :x: | manual |
| Streaming (channels) | :white_check_mark: | :x: | manual |
| Batch + changesets | :white_check_mark: | :x: | manual |
| Delta sync | :white_check_mark: | :x: | manual |
| Async op polling | :white_check_mark: | :x: | manual |
| Lambda filters | :white_check_mark: | :x: | manual |
| Deep insert | :white_check_mark: | :x: | manual |
| CSRF token mgmt | :white_check_mark: | :x: | manual |
| Code generation | :white_check_mark: | :x: | :x: |
| SAP compatibility | :white_check_mark: | partial | manual |

---

## :zap: 30-Second Example

```bash
go get github.com/jhonsferg/traverse
```

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/jhonsferg/traverse"
)

type Product struct {
    ID       int     `json:"ProductID"`
    Name     string  `json:"ProductName"`
    Price    float64 `json:"UnitPrice"`
    Category string  `json:"CategoryName"`
}

func main() {
    client, err := traverse.New(
        traverse.WithBaseURL("https://services.odata.org/V4/Northwind/Northwind.svc"),
    )
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    ctx := context.Background()

    var products []Product
    err = client.From("Products").
        Filter("UnitPrice gt 20").
        Select("ProductID", "ProductName", "UnitPrice").
        OrderBy("UnitPrice desc").
        Top(10).
        Into(ctx, &products)
    if err != nil {
        log.Fatal(err)
    }

    for _, p := range products {
        fmt.Printf("%s: $%.2f\n", p.Name, p.Price)
    }
}
```

---

## :books: Documentation

- [Installation](installation.md) - Get started in under a minute
- [Quick Start](quickstart.md) - Full working examples from connect to delete
- [OData Primer](odata-primer.md) - OData concepts every Go developer needs to know
- [Guides](guides/query-builder.md) - Deep dives into every feature
- [Code Generation](codegen/index.md) - Generate typed clients from `$metadata`
- [Extensions](extensions/index.md) - SAP, OAuth2, Prometheus, OpenTelemetry, Cache, GraphQL, Azure Event Grid, Offline Store, Dataverse, OpenAPI Export, Audit Trail
- [CSDL JSON](guides/csdl-json.md) - Parse OData metadata from JSON endpoints
- [Vocabularies](guides/vocabulary.md) - Core and Validation vocabulary annotations
- [traverse-tui](guides/tui-cli.md) - Interactive terminal OData query builder
- [Reference](reference/client.md) - Full API reference
