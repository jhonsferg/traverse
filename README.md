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

Traverse is a Go library for consuming OData v2 and v4 services. It handles all protocol details - pagination, CSRF tokens, ETag concurrency control, delta sync, batch requests, async long-running operations, actions, functions, and schema validation - so you can focus on the data.

Built on [relay](https://github.com/jhonsferg/relay) for HTTP transport. Well-suited for SAP environments (ABAP Gateway / OData v2, S/4HANA / OData v4), Microsoft Graph, and any standards-compliant OData service.

```bash
go get github.com/jhonsferg/traverse
```

Requires Go 1.24 or later.

---

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/jhonsferg/relay"
    "github.com/jhonsferg/traverse"
)

type Product struct {
    ID    int     `json:"ProductID"`
    Name  string  `json:"ProductName"`
    Price float64 `json:"UnitPrice"`
}

func main() {
    rc := relay.NewClient(relay.WithBaseURL("https://services.odata.org/V4/Northwind/Northwind.svc/"))
    client := traverse.NewClient(rc, "")

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
| **Batch requests** | `$batch` with transactional changesets |
| **Delta sync** | `$deltatoken` tracking for incremental data sync |
| **Lambda filters** | `any()` / `all()` on collection navigation properties |
| **Deep insert** | Create entity graphs in a single request |
| **$ref link operations** | `LinkTo()` / `UnlinkFrom()` for managing navigation property references |
| **Actions & Functions** | `ActionBuilder` / `FunctionBuilder` - bound and unbound |
| **Schema validation** | Client-side field name validation on `$filter` / `$orderby` |
| **Code generation** | `traverse-gen` generates typed clients from `$metadata` |
| **SAP compatibility** | CSRF tokens, X-Requested-With, SAP-specific OData quirks |

> Full feature documentation: **[jhonsferg.github.io/traverse](https://jhonsferg.github.io/traverse)**

---

## Extensions

| Module | Import path | Description |
|--------|-------------|-------------|
| `ext/sap` | `github.com/jhonsferg/traverse/ext/sap` | SAP Gateway CSRF & session handling |
| `ext/oauth2` | `github.com/jhonsferg/traverse/ext/oauth2` | OAuth2 token provider |
| `ext/prometheus` | `github.com/jhonsferg/traverse/ext/prometheus` | Prometheus metrics |
| `ext/tracing` | `github.com/jhonsferg/traverse/ext/tracing` | OpenTelemetry tracing |
| `ext/graphql` | `github.com/jhonsferg/traverse/ext/graphql` | GraphQL bridge |
| `ext/cache` | `github.com/jhonsferg/traverse/ext/cache` | Response caching |

> Extension documentation: **[jhonsferg.github.io/traverse/extensions](https://jhonsferg.github.io/traverse/extensions/index/)**

---

## Microsoft Graph

```go
gc := traverse.NewGraphClient(traverse.GraphConfig{
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
