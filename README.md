<div align="center">

# Traverse

**A declarative OData v2/v4 client for Go.**

[![Go Version](https://img.shields.io/badge/Go-1.24%2B-00ADD8?style=for-the-badge&logo=go)](https://pkg.go.dev/github.com/jhonsferg/traverse)
[![CI](https://img.shields.io/github/actions/workflow/status/jhonsferg/traverse/ci.yml?style=for-the-badge&logo=github&label=CI)](https://github.com/jhonsferg/traverse/actions/workflows/ci.yml)
[![Tests](https://img.shields.io/badge/tests-6%20OS%2FGo%20combos-0099ff?style=for-the-badge&logo=github)](https://github.com/jhonsferg/traverse/actions/workflows/ci.yml)
[![Coverage](https://img.shields.io/codecov/c/github/jhonsferg/traverse?style=for-the-badge&logo=codecov&label=coverage)](https://codecov.io/gh/jhonsferg/traverse)
[![CodeQL](https://img.shields.io/github/actions/workflow/status/jhonsferg/traverse/codeql.yml?style=for-the-badge&logo=github&label=CodeQL)](https://github.com/jhonsferg/traverse/actions/workflows/codeql.yml)
[![Trivy](https://img.shields.io/badge/vulnerability%20scan-Trivy-1f77b4?style=for-the-badge&logo=github)](https://github.com/jhonsferg/traverse/actions/workflows/trivy.yml)
[![Release](https://img.shields.io/github/v/release/jhonsferg/traverse?style=for-the-badge&logo=github&color=orange)](https://github.com/jhonsferg/traverse/releases/latest)
[![pkg.go.dev](https://img.shields.io/badge/pkg.go.dev-reference-007D9C?style=for-the-badge&logo=go)](https://pkg.go.dev/github.com/jhonsferg/traverse)
[![Go Report Card](https://img.shields.io/badge/go%20report-A%2B-brightgreen?style=for-the-badge)](https://goreportcard.com/report/github.com/jhonsferg/traverse)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg?style=for-the-badge)](LICENSE)

---

**[Documentation](https://jhonsferg.github.io/traverse) - [pkg.go.dev](https://pkg.go.dev/github.com/jhonsferg/traverse) - [Quick Start](#quick-start) - [Features](#features) - [Extensions](#extensions)**

</div>

## Overview

Traverse is a Go library for consuming OData v2 and v4 services. It handles all protocol details  -  pagination, CSRF tokens, ETag concurrency control, delta sync, batch requests, async long-running operations, actions, functions, and schema validation  -  so you can focus on the data.

Built on [relay](https://github.com/jhonsferg/relay) for HTTP transport. Well-suited for SAP environments (ABAP Gateway / OData v2, S/4HANA / OData v4), Microsoft Graph, and any standards-compliant OData service.

```bash
go get github.com/jhonsferg/traverse
```

Requires Go 1.24 or later.

---

## Why *traverse*?

The verb *to traverse* means to walk through something large, complex, or extended  -  one step at a time, without needing to hold the whole thing in memory. In computer science, *tree traversal* and *graph traversal* describe exactly that: visiting every node of a structure incrementally rather than materialising it all at once.

That is the problem this library solves:

```
other clients:  GET /MaterialSet → load 1 000 000 records into RAM → out of memory
traverse:       GET /MaterialSet → visit each record one by one   → constant memory
```

The difference is not *what* you fetch  -  it is *how* you move through it. Traverse treats a remote OData collection the way a graph traversal treats a tree: as a path to walk, not a payload to download.

Three principles follow naturally from the name:

**The path matters more than the destination.** You do not wait to have all the data before you start working. Each record is actionable the moment it arrives  -  that is exactly the `for result := range client.From("MaterialSet").Stream(ctx)` pattern at the core of the library.

**Respect for the terrain.** A careful traversal does not tear up the ground beneath it. Traverse is deliberately gentle on the services it talks to: rate limiting and circuit breaking are inherited from relay, page size follows the server's own `nextLink` rhythm, and CSRF tokens are managed transparently without extra round-trips.

**The map is not the territory.** A tree traversal does not require the full tree in memory  -  only the current node and a pointer to the next. Traverse does the same with OData: you can walk a million SAP materials without keeping a million structs alive simultaneously.

The name also has an intentional honesty to it: it does not promise to be an OData library or a SAP library. It promises to help you *move through* large, remote datasets. Today that means OData. Tomorrow it could mean any cursor-based protocol  -  the name stays valid.

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

type Product struct {
    ID    int     `json:"ProductID"`
    Name  string  `json:"ProductName"`
    Price float64 `json:"UnitPrice"`
}

func main() {
    client, err := traverse.New(
        traverse.WithBaseURL("https://services.odata.org/V4/Northwind/Northwind.svc/"),
    )
    if err != nil {
        log.Fatal(err)
    }

    products := traverse.From[Product](client, "Products")

    results, err := products.
        Filter("UnitPrice lt 20").
        OrderBy("ProductName").
        Top(10).
        List(context.Background())
    if err != nil {
        log.Fatal(err)
    }

    for _, p := range results {
        fmt.Printf("%s - $%.2f\n", p.Name, p.Price)
    }
}
```

---

## Features

| Feature | Description |
|---------|-------------|
| **Typed query builder** | `From[T]()` with `.Filter()`, `.Select()`, `.Expand()`, `.OrderBy()`, `.Top()`, `.Skip()` |
| **Type-safe filter builder** | `F("Field").Eq(value)`, `And()`, `Or()`, `Not()` - no raw strings |
| **CRUD operations** | `.Get()`, `.List()`, `.Create()`, `.Update()`, `.Delete()`, `.Upsert()` |
| **ETag & concurrency** | Automatic ETag tracking for optimistic concurrency on updates |
| **Entity change tracking** | Track and PATCH only modified fields |
| **Typed pagination** | `Paginator[T]` with `.NextPage()` and `.Stream()` |
| **Async operations** | Automatic polling for `202 Accepted` long-running operations |
| **Streaming** | Channel-based streaming via `json.Decoder` - constant memory |
| **Batch requests** | `$batch` with transactional changesets; OData 4.01 JSON batch format |
| **Delta sync** | `$deltatoken` tracking for incremental data sync |
| **Lambda filters** | `any()` / `all()` on collection navigation properties |
| **Deep insert** | Create entity graphs in a single request |
| **Deep update** | `PATCH` nested entity graphs in a single round-trip |
| **BulkUpdate** | `PATCH /EntitySet?$filter=...` for mass updates (OData 4.01) |
| **Singletons** | First-class singleton access: `client.Singleton("me").Page(ctx)` |
| **Type casting** | `AsType("Model.Manager")` path segments; `IsOf()` / `Cast()` filter helpers |
| **$expand $levels** | `Expand("Children", traverse.WithExpandLevels(traverse.LevelsMax))` |
| **Geospatial** | `GeographyPoint`, `GeometryPoint`, `GeoDistance`, `GeoIntersects` filter functions |
| **$ref link operations** | `LinkTo()` / `UnlinkFrom()` for managing navigation property references |
| **Actions & Functions** | `ActionBuilder` / `FunctionBuilder` - bound and unbound |
| **Schema validation** | Client-side field name validation on `$filter` / `$orderby` |
| **Prefer headers** | `PreferHandlingStrict`, `ReturnMinimal`, `ReturnRepresentation`, `PreferTrackChanges` |
| **$schemaversion** | `WithSchemaVersion("2.0")` at client or per-query level |
| **Atom/XML responses** | OData v2 `application/atom+xml` streaming parser (auto-detected) |
| **OData v2 $inlinecount** | `$inlinecount=allpages` emitted for v2 services; `d.__count` parsed |
| **SAP TLS** | `WithSAPTLSConfig(cfg)` for custom CA bundles and self-signed certs |
| **SAP property path** | `FetchPropertyAs[T]` for scalar/complex property fetch by key and path |
| **Code generation** | `traverse-gen` generates typed clients from `$metadata` |
| **SAP compatibility** | CSRF tokens, X-Requested-With, SAP sap:* metadata attributes |

> Full feature documentation: **[jhonsferg.github.io/traverse](https://jhonsferg.github.io/traverse)**

---

## SAP CSRF Token Management (Automatic)

Traverse now handles SAP CSRF tokens completely transparently, including session cookie management. This fixes a critical architectural issue where CSRF tokens and session cookies must be paired atomically:

```go
import (
    "context"
    "github.com/jhonsferg/traverse"
    "github.com/jhonsferg/traverse/ext/sap"
)

// CSRF token and session cookie are fetched and managed automatically
// No explicit token management needed
client, _ := traverse.New(
    traverse.WithBaseURL("https://sap.example.com/sap/opu/odata/"),
    sap.WithCSRFMiddleware(), // Automatic token fetch + session persistence
)

// Create operation - token is reused, session cookies are automatic
newOrder := Order{ID: "1001", Amount: 99.99}
created, err := sap.CreateJsonAs[Order](
    client.From("Orders"),
    context.Background(),
    newOrder,
)

// If token expires, middleware automatically handles 403 recovery
// No retry logic needed in application code
```

**What's Fixed (v0.19.0+):**
- ✅ Session cookies are now captured and reused (via relay v0.4.0 CookieJar support)
- ✅ CSRF tokens are reused for their full validity window (no preventive invalidation)
- ✅ URL construction handles edge cases (no multiple slashes)
- ✅ Better error diagnostics distinguish between CSRF, auth, config, and network failures
- ✅ Automatic 403 recovery when tokens expire in-flight

---

## CSDL JSON Support

Traverse can parse both EDMX/XML and CSDL JSON (the OData v4.01 JSON format used by Microsoft Graph). The client auto-detects the format by `Content-Type` when fetching `$metadata`:

```go
// Auto-detected - no code change needed when the service returns JSON metadata
client, _ := traverse.New(traverse.WithBaseURL("https://api.example.com/odata/v4/"))
meta, err := client.Metadata(ctx)
```

For direct parsing:

```go
import "github.com/jhonsferg/traverse"

// From bytes
meta, err := traverse.ParseCSDLJSON(data)

// From a reader (e.g. http.Response.Body)
meta, err := traverse.ParseCSDLJSONReader(resp.Body)
```

---

## XML Support

Traverse supports both JSON and XML response formats. Some OData backends (particularly SAP) may return XML instead of JSON, even when JSON is requested. Use the explicit format methods to handle both:

```go
type Product struct {
    ID    int    `json:"ProductID" xml:"ProductID"`
    Name  string `json:"ProductName" xml:"ProductName"`
}

client, _ := traverse.New(traverse.WithBaseURL("https://api.example.com/odata/"))

ctx := context.Background()

qb := client.From("Products").Filter("Price lt 100")

products, err := traverse.CollectJsonAs[Product](qb, ctx)

if err != nil {
    products, err = traverse.CollectXmlAs[Product](qb, ctx)
    if err != nil {
        log.Fatal(err)
    }
}

for _, p := range products {
    fmt.Printf("%s\n", p.Name)
}
```

Streaming with XML:

```go
type Order struct {
    OrderID string `json:"OrderID" xml:"OrderID"`
    Amount  float64 `json:"Amount" xml:"Amount"`
}

qb := client.From("Orders").Top(1000)

for result := range traverse.StreamXmlAs[Order](qb, context.Background()) {
    if result.Err != nil {
        log.Printf("stream error: %v", result.Err)
        break
    }
    fmt.Printf("Order %s: $%.2f\n", result.Value.OrderID, result.Value.Amount)
}
```

All CRUD and query operations have explicit JSON/XML variants:

| Operation | JSON | XML |
|-----------|------|-----|
| Create | `CreateJsonAs[T]()` | `CreateXmlAs[T]()` |
| Collect (list) | `CollectJsonAs[T]()` | `CollectXmlAs[T]()` |
| Stream (paginated) | `StreamJsonAs[T]()` | `StreamXmlAs[T]()` |
| First (single) | `FirstJsonAs[T]()` | `FirstXmlAs[T]()` |
| FindByKey (get) | `FindByKeyJsonAs[T]()` | `FindByKeyXmlAs[T]()` |
| Functions | `ExecuteFunctionJsonAs[T]()` | `ExecuteFunctionXmlAs[T]()` |
| Actions | `ExecuteActionJsonAs[T]()` | `ExecuteActionXmlAs[T]()` |
| Delta sync | `DeltaSyncJsonAs[T]` | `DeltaSyncXmlAs[T]` |

Struct tags determine marshaling behavior:

```go
type Material struct {
    ID    string `json:"MatID" xml:"MatID"`
    Name  string `json:"Name" xml:"Name"`
    Stock int    `json:"StockQty" xml:"StockQty"`
}

json_mats, _ := traverse.CollectJsonAs[Material](qb, ctx)

xml_mats, _ := traverse.CollectXmlAs[Material](qb, ctx)
```

---

## OpenAPI 3.1 Export

Convert OData metadata to an OpenAPI 3.1 document:

```go
import (
    "encoding/json"
    "github.com/jhonsferg/traverse"
    "github.com/jhonsferg/traverse/ext/openapi"
)

meta, _ := client.Metadata(ctx)
doc, err := openapi.Export(meta,
    openapi.WithTitle("My OData API"),
    openapi.WithVersion("1.0.0"),
    openapi.WithServerURL("https://api.example.com/odata/v4/"),
)

out, _ := json.MarshalIndent(doc, "", "  ")
fmt.Println(string(out))
```

```bash
go get github.com/jhonsferg/traverse/ext/openapi
```

---

## OData Vocabularies

Properties carry parsed Core and Validation vocabulary annotations:

```go
meta, _ := client.Metadata(ctx)
for _, et := range meta.EntityTypes {
    for _, prop := range et.Properties {
        core := traverse.ParseCoreVocabulary(prop.Annotations)
        val  := traverse.ParseValidationVocabulary(prop.Annotations)

        fmt.Printf("%s: %s", prop.Name, core.Description)
        if val.Pattern != "" {
            fmt.Printf(" (pattern: %s)", val.Pattern)
        }
        if val.Required {
            fmt.Print(" [required]")
        }
        fmt.Println()
    }
}
```

Available types: `CoreVocabulary` (Description, LongDescription, Immutable, Computed, Permissions, …) and `ValidationVocabulary` (Minimum, Maximum, Pattern, AllowedValues, Required).

---

## Tools

### traverse-gen

`traverse-gen` generates type-safe Go clients from an OData `$metadata` endpoint:

```bash
go run github.com/jhonsferg/traverse/cmd/traverse-gen \
  -metadata https://services.odata.org/V4/Northwind/Northwind.svc/$metadata \
  -out ./northwind
```

### traverse-tui

Interactive terminal UI for exploring OData endpoints, building queries, and inspecting results:

```bash
go run github.com/jhonsferg/traverse/cmd/traverse-tui
```

### SAP OData Mock Server

A local SAP NetWeaver OData v2 simulator for integration testing without a real SAP system:

```bash
go run github.com/jhonsferg/traverse/cmd/sap-mock
```

Simulates CSRF token lifecycle, Basic Auth, `$metadata` responses, entity-set queries, key-predicate lookups, and property-path navigation. Logs all incoming requests with headers, query parameters, and body for inspection.

```
SAP OData Mock Server
  Listen: http://localhost:44300
  Auth:   enabled (user=sapuser pass=sappass)
```

---

## Extensions

| Module | Import path | Description |
|--------|-------------|-------------|
| `ext/sap` | `github.com/jhonsferg/traverse/ext/sap` | SAP Gateway CSRF, session handling, and Fiori UI annotations |
| `ext/openapi` | `github.com/jhonsferg/traverse/ext/openapi` | OpenAPI 3.1 export from OData metadata |
| `ext/oauth2` | `github.com/jhonsferg/traverse/ext/oauth2` | OAuth2 token provider |
| `ext/prometheus` | `github.com/jhonsferg/traverse/ext/prometheus` | Prometheus metrics |
| `ext/tracing` | `github.com/jhonsferg/traverse/ext/tracing` | OpenTelemetry tracing |
| `ext/graphql` | `github.com/jhonsferg/traverse/ext/graphql` | GraphQL-to-OData bridge |
| `ext/cache` | `github.com/jhonsferg/traverse/ext/cache` | HTTP response and metadata caching |
| `ext/offline` | `github.com/jhonsferg/traverse/ext/offline` | Persistent offline store with JSON cache |
| `ext/dataverse` | `github.com/jhonsferg/traverse/ext/dataverse` | Microsoft Dataverse adapter |
| `ext/azure` | `github.com/jhonsferg/traverse/ext/azure` | Azure Event Grid change events |
| `ext/webhooks` | `github.com/jhonsferg/traverse/ext/webhooks` | OData webhook subscriptions |
| `ext/audit` | `github.com/jhonsferg/traverse/ext/audit` | Audit trail middleware |

> Extension documentation: **[jhonsferg.github.io/traverse/extensions](https://jhonsferg.github.io/traverse/extensions/index/)**

### SAP Fiori UI Annotations

`ext/sap` includes support for SAP UI annotations parsed from EDMX attributes (`sap:label`, `sap:sortable`, `sap:filterable`, etc.):

```go
import "github.com/jhonsferg/traverse/ext/sap"

ann := sap.ParseSAPUIAnnotation(property.RawAttributes)
fmt.Printf("Label: %s, Filterable: %v\n", ann.Label, ann.Filterable)

// Get all annotated properties from an entity type
props := sap.AnnotatedProperties(entityType, meta)
for _, p := range props {
    fmt.Printf("%s → label=%s sortable=%v\n", p.Property.Name, p.Annotation.Label, p.Annotation.Sortable)
}
```

---

## Microsoft Graph

```go
rc := relay.New(relay.WithBearerToken(token))
gc := traverse.NewGraphClient(rc, traverse.GraphConfig{
    AccessToken: token,
})

type User struct {
    ID          string `json:"id"`
    DisplayName string `json:"displayName"`
}

users, err := traverse.From[User](gc, "users").
    Filter("department eq 'Engineering'").
    Select("id", "displayName").
    List(ctx)
```

> [Microsoft Graph guide](https://jhonsferg.github.io/traverse/extensions/graph/)

---

## Documentation

The full documentation is at **[jhonsferg.github.io/traverse](https://jhonsferg.github.io/traverse)**:

- [Getting Started](https://jhonsferg.github.io/traverse/quickstart/)
- [Query Builder Guide](https://jhonsferg.github.io/traverse/guides/query-builder/)
- [All Guides](https://jhonsferg.github.io/traverse/guides/)
- [Code Generation](https://jhonsferg.github.io/traverse/codegen/)
- [API Reference](https://pkg.go.dev/github.com/jhonsferg/traverse) on pkg.go.dev

---

## License

MIT - see [LICENSE](LICENSE).
