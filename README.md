<div align="center">

# Traverse

**An OData v2/v4 client for Go - streaming, batch, SAP-compatible.**

[![Go Version](https://img.shields.io/badge/Go-1.24%2B-00ADD8?style=for-the-badge&logo=go)](https://pkg.go.dev/github.com/jhonsferg/traverse)
[![CI](https://img.shields.io/github/actions/workflow/status/jhonsferg/traverse/ci.yml?style=for-the-badge&logo=github&label=CI)](https://github.com/jhonsferg/traverse/actions/workflows/ci.yml)
[![Tests](https://img.shields.io/badge/tests-6%20OS%2FGo%20combos-0099ff?style=for-the-badge&logo=github)](https://github.com/jhonsferg/traverse/actions/workflows/ci.yml)
[![Coverage](https://img.shields.io/codecov/c/github/jhonsferg/traverse?style=for-the-badge&logo=codecov&label=coverage)](https://codecov.io/gh/jhonsferg/traverse)
[![CodeQL](https://img.shields.io/github/actions/workflow/status/jhonsferg/traverse/codeql.yml?style=for-the-badge&logo=github&label=CodeQL)](https://github.com/jhonsferg/traverse/actions/workflows/codeql.yml)
[![Trivy](https://img.shields.io/badge/vulnerability%20scan-Trivy-1f77b4?style=for-the-badge&logo=github)](https://github.com/jhonsferg/traverse/actions/workflows/trivy.yml)
[![API Check](https://img.shields.io/badge/api%20compatibility-checked-4CAF50?style=for-the-badge&logo=github)](https://github.com/jhonsferg/traverse/actions/workflows/api-check.yml)
[![License Check](https://img.shields.io/badge/license%20compliance-checked-FFC107?style=for-the-badge&logo=github)](https://github.com/jhonsferg/traverse/actions/workflows/license-check.yml)
[![Release](https://img.shields.io/github/v/release/jhonsferg/traverse?style=for-the-badge&logo=github&color=orange)](https://github.com/jhonsferg/traverse/releases/latest)
[![pkg.go.dev](https://img.shields.io/badge/pkg.go.dev-reference-007D9C?style=for-the-badge&logo=go)](https://pkg.go.dev/github.com/jhonsferg/traverse)
[![Go Report Card](https://img.shields.io/badge/go%20report-A%2B-brightgreen?style=for-the-badge)](https://goreportcard.com/report/github.com/jhonsferg/traverse)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg?style=for-the-badge)](LICENSE)

---

**[Installation](#installation) - [Quick Start](#quick-start) - [Query Builder](#query-builder) - [CRUD](#crud) - [ETag & Concurrency](#etag--concurrency-control) - [Change Tracking](#entity-change-tracking) - [Pagination](#typed-pagination) - [Async Ops](#async-operation-polling) - [Streaming](#streaming) - [Batch](#batch-requests) - [Delta Sync](#delta-sync) - [Extensions](#extension-modules)**

</div>

## Overview

Traverse is a Go library for consuming OData v2 and v4 services. It handles all protocol details - pagination, CSRF tokens, ETag concurrency control, delta sync, batch requests, async long-running operations - so you can focus on the data.

Built on top of [relay](https://github.com/jhonsferg/relay) for HTTP transport. Well-suited for SAP environments (classic ABAP Gateway / OData v2, S/4HANA / OData v4), but compatible with any standards-compliant OData service.

Large result sets are handled through streaming (`json.Decoder` + Go channels), keeping memory constant regardless of payload size.

---

## Installation

```bash
go get github.com/jhonsferg/traverse
```

Requires Go 1.24 or later. The core module has no external dependencies beyond relay.

---

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/jhonsferg/traverse"
)

type SalesOrder struct {
    ID          string  `json:"SalesOrderID"`
    CustomerID  string  `json:"CustomerID"`
    TotalAmount float64 `json:"TotalAmount"`
}

func main() {
    client, err := traverse.New(
        traverse.WithBaseURL("https://sap.example.com/sap/opu/odata/sap/MY_SRV"),
        traverse.WithBasicAuth("user", "pass"),
    )
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    ctx := context.Background()

    // Typed query into a slice
    var orders []SalesOrder
    err = client.From("SalesOrderSet").
        Filter("TotalAmount gt 1000").
        Select("SalesOrderID", "CustomerID", "TotalAmount").
        OrderBy("TotalAmount desc").
        Top(20).
        Into(ctx, &orders)
    if err != nil {
        log.Fatal(err)
    }

    for _, o := range orders {
        fmt.Printf("%s: %.2f\n", o.ID, o.TotalAmount)
    }
}
```

---

## Query Builder

The fluent query builder covers every OData system query option:

```go
query := client.From("Products").
    Select("ID", "Name", "Price", "Category").
    Filter("Price gt 10 and Category eq 'Electronics'").
    OrderBy("Price asc").
    Top(50).
    Skip(100).
    Expand("Supplier").
    WithCount()
```

### Filtering

```go
// Simple equality
.Filter("Status eq 'OPEN'")

// Compound conditions
.Filter("Price gt 10 and Price lt 100")

// String functions
.Filter("startswith(Name,'Widget')")

// Nested property
.Filter("Address/City eq 'Berlin'")
```

### Deep Expand (Nested Navigation)

Configure per-level options for expanded navigation properties:

```go
orders, err := client.From("Orders").
    ExpandNested("Items").
        Select("ID", "Qty", "UnitPrice").
        Filter("Qty gt 0").
        OrderBy("UnitPrice desc").
        Top(10).
    Done().
    ExpandNested("Customer").
        Select("Name", "Email").
    Done().
    Into(ctx, &orders)
```

Generates: `$expand=Items($select=ID,Qty,UnitPrice;$filter=Qty gt 0;$orderby=UnitPrice desc;$top=10),Customer($select=Name,Email)`

### Aggregation with $apply

Server-side aggregation using the OData v4 `$apply` extension:

```go
page, err := client.From("SalesItems").
    Apply("groupby((Category),aggregate(Amount with sum as TotalAmount))").
    Page(ctx)
```

### Count

```go
// Count without fetching data
n, err := client.From("Products").Filter("Active eq true").Count(ctx)

// Count inline with results
page, err := client.From("Products").WithCount().Page(ctx)
if page.Count != nil {
    fmt.Printf("total: %d, returned: %d\n", *page.Count, len(page.Value))
}
```

---

## CRUD

### Create

```go
id, err := client.Create(ctx, "SalesOrderSet", map[string]any{
    "CustomerID":  "C-001",
    "Description": "New order",
})
```

### Read

```go
var order SalesOrder
err := client.Read(ctx, "SalesOrderSet", "SalesOrderID='4500001'", &order)
```

### Update (PATCH / MERGE)

```go
// Partial update - only specified fields are changed
err := client.Update(ctx, "SalesOrderSet", "SalesOrderID='4500001'", map[string]any{
    "Status": "CONFIRMED",
})
```

### Delete

```go
err := client.Delete(ctx, "SalesOrderSet", "SalesOrderID='4500001'")
```

### Upsert

Create a new entity or replace an existing one in a single operation (OData 4.01 §11.4.4).
Uses `PUT` with `If-None-Match: *`:

```go
err := client.Upsert(ctx, "Products", "ProductID=42", map[string]any{
    "ProductID": 42,
    "Name":      "Widget Pro",
    "Price":     29.99,
})
// 201 Created if new, 200/204 if replaced
```

---

## ETag & Concurrency Control

ETag-aware operations prevent lost updates when multiple clients modify the same entity.

### Read with ETag

```go
result, err := client.ReadWithETag(ctx, "Products", "42")
// result.ETag holds the token; result.Entity holds the data

fmt.Println(result.ETag.Value)     // "W/"abc123""
fmt.Println(result.ETag.IsWeak())  // true
```

### Update with ETag

```go
// Fetch and capture ETag
result, err := client.ReadWithETag(ctx, "Products", "42")

// Apply conditional update - fails with ErrConcurrencyConflict if stale
err = client.UpdateWithETag(ctx, "Products", "42", map[string]any{
    "Price": 19.99,
}, result.ETag)

if errors.Is(err, traverse.ErrConcurrencyConflict) {
    // Another client modified the entity - re-read and retry
}
```

### Replace and Delete with ETag

```go
// Full replacement (PUT)
err = client.ReplaceWithETag(ctx, "Products", "42", fullEntity, etag)

// Conditional delete
err = client.DeleteWithETag(ctx, "Products", "42", etag)
```

---

## Entity Change Tracking

Track which fields of an entity have been modified and generate minimal PATCH payloads.
Inspired by the dirty-bit pattern from .NET EF Core and pnpcore.

```go
// 1. Fetch the entity
var raw map[string]any
client.Read(ctx, "Products", "42", &raw)

// 2. Start tracking
tracked := traverse.TrackEntity(raw)

// 3. Modify fields
tracked.Set("Price", 24.99)
tracked.Set("Status", "ACTIVE")

// 4. Check what changed
fmt.Println(tracked.IsDirty())       // true
fmt.Println(tracked.DirtyFields())   // ["Price", "Status"]

// 5. Send only changed fields as a PATCH
changes := tracked.Changes()         // map[string]any{"Price": 24.99, "Status": "ACTIVE"}
err := client.Update(ctx, "Products", "42", changes)

// Or use SaveChanges for a one-step save + reset
err = tracked.SaveChanges(ctx, client, "Products", "42")
```

### Discard and Reset

```go
tracked.Discard()   // revert all changes back to original snapshot
tracked.Reset()     // mark current state as the new baseline (no more dirty fields)
```

### JSON Serialization

`TrackedEntity` implements `json.Marshaler` and encodes only changed fields:

```go
body, _ := json.Marshal(tracked) // {"Price":24.99,"Status":"ACTIVE"}
```

---

## Typed Pagination

`Paginator[T]` provides a type-safe, iterator-style interface for paging through
large result sets. Automatically follows `@odata.nextLink` (v4) or `__next` (v2).

```go
type Product struct {
    ID    int    `json:"ProductID"`
    Name  string `json:"Name"`
    Price float64 `json:"Price"`
}

p := traverse.NewPaginator[Product](
    client.From("Products").Top(100).OrderBy("ProductID asc"),
)

for p.HasMorePages() {
    items, err := p.NextPage(ctx)
    if err != nil {
        log.Fatal(err)
    }
    for _, item := range items {
        fmt.Printf("%d: %s - %.2f\n", item.ID, item.Name, item.Price)
    }
}
```

### Custom Decoder

```go
p := traverse.NewPaginatorWithDecoder[Product](
    client.From("Products"),
    func(raw json.RawMessage) (Product, error) {
        var p Product
        return p, json.Unmarshal(raw, &p)
    },
)
```

### Reset and Total Count

```go
p.Reset()                           // restart from the first page

count, err := p.TotalCount(ctx)     // requires .WithCount() on the query
```

### Fetch Page at Arbitrary URL

Follow server-provided links directly:

```go
page, err := client.FetchPageAt(ctx, "https://service.example.com/Products?$skiptoken=xyz")
```

---

## Async Operation Polling

Many OData services (SAP, Dynamics 365) process long-running operations asynchronously.
The server returns `202 Accepted` with a `Location` header pointing to a status URL.

```go
// 1. Send the initial request with Prefer: respond-async
req := client.http.Post("/SalesOrderSet").
    WithHeader("Prefer", "respond-async").
    WithJSON(newOrder)

resp, err := client.Execute(req)
if resp.StatusCode != 202 {
    // synchronous completion - no polling needed
    return
}

// 2. Create a poller from the Location header
poller := client.NewAsyncPoller(resp.Headers.Get("Location")).
    WithPollInterval(2 * time.Second).
    WithMaxPolls(30)

// 3. Wait for completion (respects Retry-After header from server)
result, err := poller.Wait(ctx)
if errors.Is(err, traverse.ErrAsyncOpFailed) {
    log.Printf("operation failed: %s", result.Body)
}
if err == nil {
    fmt.Printf("completed with status %d\n", result.StatusCode)
}
```

### Status Values

| Status | Meaning |
|--------|---------|
| `AsyncOpRunning` | Operation still in progress (202) |
| `AsyncOpSucceeded` | Completed successfully (200, 201, 204) |
| `AsyncOpFailed` | Server reported failure (4xx, 5xx) |
| `AsyncOpCancelled` | Operation was cancelled server-side |

### Sentinel Errors

| Error | When returned |
|-------|--------------|
| `ErrAsyncOpFailed` | Server returned 4xx/5xx during polling |
| `ErrAsyncOpTimeout` | MaxPolls exhausted before completion |

---

## Streaming

For large datasets, streaming processes results record-by-record with constant memory.

### Channel-based Streaming

```go
for result := range client.From("MaterialSet").Stream(ctx) {
    if result.Err != nil {
        log.Fatal(result.Err)
    }
    fmt.Println(result.Value["MaterialID"])
}
```

### Typed Streaming with Callback

```go
err := client.From("SalesOrderSet").
    Filter("Status eq 'OPEN'").
    Stream(ctx).
    ForEach(func(record map[string]any, page, index int) error {
        return processRecord(record)
    })
```

### Raw JSONL Streaming

```go
stream, err := client.ExecuteStream(ctx, req)
defer stream.Close()
for stream.Next() {
    fmt.Println(stream.Value()) // each raw JSON entity
}
```

Memory stays constant regardless of total result count because records are processed
as they arrive from the network, without buffering the full response.

---

## Batch Requests

Execute multiple operations in a single HTTP round-trip.

### Read Batch

```go
batch := client.NewBatch()
batch.Get("MaterialSet('MAT-001')")
batch.Get("MaterialSet('MAT-002')")
batch.Get("SalesOrderSet('SO-100')")

results, err := batch.Execute(ctx)
for i, r := range results {
    fmt.Printf("request %d: status %d\n", i, r.StatusCode)
}
```

### Changeset (Atomic Batch)

Group multiple mutations into a changeset - all succeed or all roll back:

```go
batch := client.NewBatch()
cs := batch.NewChangeset()
cs.Post("SalesOrderSet", newOrder)
cs.Patch("SalesOrderSet('SO-100')", patch)
cs.Delete("SalesOrderSet('SO-099')")

results, err := batch.Execute(ctx)
```

---

## Delta Sync

Incremental updates fetch only records that changed since the last sync.

```go
// First run: fetch all + store the delta token
stream, deltaToken, err := client.From("SalesOrderSet").Delta(ctx, "")

// Process all records...
for result := range stream { /* ... */ }

// Subsequent runs: only what changed
stream, newToken, err := client.From("SalesOrderSet").Delta(ctx, deltaToken)
```

Delta links are preserved across sessions and service restarts.

---

## SAP Compatibility

Traverse handles SAP-specific OData protocol quirks automatically:

- **CSRF token fetching** - automatically fetches and rotates `x-csrf-token` for write operations
- **MERGE method** - OData v2 uses `X-HTTP-Method: MERGE` tunneling (since MERGE is not standard HTTP)
- **ETag handling** - parses both `ETag` and `W/` weak ETags from SAP ABAP Gateway responses
- **Basic auth** - `WithBasicAuth(user, pass)` for classic ABAP Gateway authentication
- **Client cert** - `WithClientCert(certFile, keyFile)` for mutual TLS

```go
client, err := traverse.New(
    traverse.WithBaseURL("https://s4hana.example.com/sap/opu/odata/sap/API_SALES_ORDER_SRV"),
    traverse.WithBasicAuth(os.Getenv("SAP_USER"), os.Getenv("SAP_PASS")),
    traverse.WithODataVersion(traverse.ODataV2),
)
```

---

## Code Generation (`traverse-gen`)

Generate typed Go clients directly from your OData `$metadata` endpoint:

```bash
# Install
go install github.com/jhonsferg/traverse/cmd/traverse-gen@latest

# Generate from live endpoint
traverse-gen --metadata-url https://services.odata.org/V4/Northwind/Northwind.svc/$metadata \
             --output-dir ./odata --package-name odata

# Generate from local EDMX file
traverse-gen --metadata-file northwind.edmx --output-dir ./odata --package-name odata
```

The generator produces three files:

| File | Contents |
|------|----------|
| `types.go` | Go structs for every EntityType and ComplexType with json + odata tags |
| `client.go` | `GeneratedClient` with typed entity set accessor methods |
| `queries.go` | `*XxxQuery` typed QueryBuilder wrappers per entity set |

**Generated usage:**

```go
c, err := odata.NewGeneratedClient("https://services.odata.org/V4/Northwind/Northwind.svc")
if err != nil {
    log.Fatal(err)
}

// Fully typed - no string-based entity set names
var customers []odata.Customer
err = c.Customers().Top(10).Filter("Country eq 'Germany'").List(ctx, &customers)
```

---

## Lambda Filter DSL

Build OData lambda expressions (`any`/`all`) fluently:

```go
// Orders where any tag name is 'priority'
query := client.From("Orders").
    LambdaAny("Tags", func(b *traverse.LambdaBuilder) {
        b.Field("Name").Eq("priority")
    })

// Products where all variants have stock > 0
query = client.From("Products").
    LambdaAll("Variants", func(b *traverse.LambdaBuilder) {
        b.Field("Stock").Gt(0)
    })

// Combine with regular filters using and
query = client.From("Items").
    Filter("Active eq true").
    LambdaAny("Tags", func(b *traverse.LambdaBuilder) {
        b.Field("Category").Contains("sale")
    })
```

Supported operators: `Eq`, `Ne`, `Lt`, `Le`, `Gt`, `Ge`, `Contains`, `StartsWith`, `EndsWith`.

---

## Deep Insert

Create an entity with related entities in a single request (OData 4.01):

```go
type Order struct {
    CustomerID string       `json:"CustomerID"`
    Lines      []OrderLine  `json:"Lines"`
}

order := Order{
    CustomerID: "ALFKI",
    Lines: []OrderLine{
        {ProductID: 1, Quantity: 10},
        {ProductID: 2, Quantity: 5},
    },
}

resp, err := client.From("Orders").CreateDeep(ctx, order)
```

For custom `Prefer` semantics:

```go
opts := traverse.DeepInsertOptions{
    ReturnRepresentation: true,
    ContinueOnError:      false,
}
resp, err := client.From("Orders").CreateDeepWithPrefer(ctx, order, opts.Header())
```

---

## Conditional Request Headers

Attach standard HTTP conditional headers to any query:

```go
// Optimistic update - only apply if ETag matches
_, err := client.From("Products(1)").
    IfMatch(`"abc123"`).
    Update(ctx, patch)

// Only fetch if modified since last sync
resp, err := client.From("Reports('Q4')").
    IfModifiedSince(lastSyncTime).
    Get(ctx, &report)

// Create only if not exists
resp, err := client.From("Settings('global')").
    IfNoneMatch("*").
    Create(ctx, defaults)
```

All four standard conditional headers are supported: `IfMatch`, `IfNoneMatch`, `IfModifiedSince`, `IfUnmodifiedSince`.

---

## Named Stream Properties

Read binary/media stream properties without loading them into memory:

```go
// Get a document attachment as a streaming reader
rc, err := client.From("Documents(42)").StreamProperty(ctx, "Attachment")
if err != nil {
    log.Fatal(err)
}
defer rc.Close()

// Stream directly to disk
f, _ := os.Create("attachment.pdf")
_, _ = io.Copy(f, rc)

// Check size before downloading
size, err := client.From("Documents(42)").StreamPropertySize(ctx, "Attachment")
fmt.Printf("attachment is %d bytes\n", size)
```

---

## Extension Modules

Extensions are optional and independently versioned. Install only what you need.

| Module | Purpose | Install |
|--------|---------|---------|
| `ext/cache/memory` | In-memory metadata + response caching | `go get github.com/jhonsferg/traverse/ext/cache/memory` |
| `ext/cache/redis` | Redis-backed shared cache | `go get github.com/jhonsferg/traverse/ext/cache/redis` |
| `ext/oauth2` | OAuth2 token management with auto-refresh | `go get github.com/jhonsferg/traverse/ext/oauth2` |
| `ext/sap` | SAP-specific CSRF, headers, and gateway quirks | `go get github.com/jhonsferg/traverse/ext/sap` |
| `ext/prometheus` | Prometheus metrics (requests, latency, errors) | `go get github.com/jhonsferg/traverse/ext/prometheus` |
| `ext/tracing` | OpenTelemetry distributed tracing | `go get github.com/jhonsferg/traverse/ext/tracing` |
| `ext/graphql` | GraphQL bridge (experimental) | `go get github.com/jhonsferg/traverse/ext/graphql` |

---

## Performance

- **Constant memory streaming** - channel-based pagination never buffers the full response; ideal for datasets with millions of records
- **Object pooling** - `sync.Pool` for internal buffers and JSON decoders to reduce GC pressure
- **Zero allocations on hot paths** - string interning for common OData keys and values
- **Lazy URL construction** - query URLs are rebuilt only when options change (dirty flag)
- **Thread-safe** - the client is safe for concurrent use across goroutines

---

## Running Locally

```bash
# Tests (all platforms)
go test ./...
go test -race ./...

# Benchmarks
go test -bench=. -benchmem ./...

# Lint (requires golangci-lint v2)
golangci-lint run
```

---

## CI/CD

Every push and pull request runs:

- Tests on Linux, macOS, and Windows (Go 1.24 and 1.25)
- `go vet` and golangci-lint v2
- CodeQL static analysis
- Trivy vulnerability scan
- TruffleHog secrets scan
- API compatibility check

Tags are created automatically on every merge to master using Conventional Commits semantics (`feat` - minor bump, `fix`/`ci`/`chore` - patch, `BREAKING CHANGE` footer - major).

---

## Contributing

Contributions are welcome. Please read [CONTRIBUTING.md](CONTRIBUTING.md) before opening a pull request.

Commit messages must follow [Conventional Commits](https://www.conventionalcommits.org/):

```
feat(query): add $apply aggregation support
fix(client): resolve context cancellation race condition
docs(readme): update streaming example
```

---

<div align="center">

Distributed under the MIT License. See [LICENSE](LICENSE) for details.

Built with care by [jhonsferg](https://github.com/jhonsferg)

</div>
