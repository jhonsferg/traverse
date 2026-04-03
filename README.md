<div align="center">

# Traverse

**An OData v2/v4 client for Go - streaming, batch, SAP-compatible.**

[![Go Reference](https://pkg.go.dev/badge/github.com/jhonsferg/traverse.svg)](https://pkg.go.dev/github.com/jhonsferg/traverse)
[![CI](https://img.shields.io/github/actions/workflow/status/jhonsferg/traverse/ci.yml?style=flat-square&logo=github&label=CI)](https://github.com/jhonsferg/traverse/actions/workflows/ci.yml)
[![Tests](https://img.shields.io/badge/tests-6%20OS%2FGo%20combos-0099ff?style=flat-square&logo=github)](https://github.com/jhonsferg/traverse/actions/workflows/ci.yml)
[![Codecov](https://img.shields.io/badge/coverage-tracked-41B883?style=flat-square&logo=codecov)](https://codecov.io/gh/jhonsferg/traverse)
[![CodeQL](https://img.shields.io/github/actions/workflow/status/jhonsferg/traverse/codeql.yml?style=flat-square&logo=github&label=CodeQL)](https://github.com/jhonsferg/traverse/actions/workflows/codeql.yml)
[![Release](https://img.shields.io/github/v/release/jhonsferg/traverse?style=flat-square&logo=github&color=orange)](https://github.com/jhonsferg/traverse/releases/latest)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg?style=flat-square)](LICENSE)
[![Go 1.24+](https://img.shields.io/badge/Go-1.24%2B-00ADD8?style=flat-square&logo=go)](https://golang.org/)

---

**[Installation](#installation) · [Quick start](#quick-start) · [Features](#features) · [Extensions](#extension-modules) · [Contributing](#contributing)**

</div>

## Overview

Traverse is a Go library for consuming OData v2 and v4 services. It handles the protocol details - pagination, CSRF tokens, delta sync, batch requests - so you can focus on the data.

It is built on top of [relay](https://github.com/jhonsferg/relay) for HTTP transport and is well-suited for SAP environments (classic ABAP Gateway / OData v2, S/4HANA / OData v4), though it works with any standards-compliant OData service.

Large result sets are handled through streaming (`json.Decoder` + Go channels), keeping memory usage constant regardless of payload size.

---

## Installation

```bash
go get github.com/jhonsferg/traverse
```

Requires Go 1.24 or later. The core module has no external dependencies beyond relay.

Optional extensions can be installed independently - see [Extension modules](#extension-modules).

---

## Quick start

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/jhonsferg/traverse"
)

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

    // Stream all records - memory stays constant as pages are fetched
    for result := range client.From("MaterialSet").Stream(ctx) {
        if result.Err != nil {
            log.Fatal(result.Err)
        }
        fmt.Println(result.Value) // map[string]any
    }
}
```

### Queries

```go
// Fetch a filtered, projected result set into a typed slice
var materials []Material
err := client.From("MaterialSet").
    Filter("Activated eq true").
    Select("MaterialID", "Description", "UnitOfMeasure").
    OrderBy("MaterialID asc").
    Top(50).
    Into(ctx, &materials)

// Count records matching a filter
count, err := client.From("SalesOrderSet").
    Filter("CustomerID eq '1001'").
    Count(ctx)

// Expand a navigation property
err = client.From("SalesOrderSet").
    Expand("Items").
    Filter("Status eq 'OPEN'").
    Into(ctx, &orders)
```

### CRUD

```go
// Create
id, err := client.Create(ctx, "SalesOrderSet", newOrder)

// Read by key
var order SalesOrder
err = client.Read(ctx, "SalesOrderSet", "SO_NUMBER='4500001'", &order)

// Update (PATCH / MERGE)
err = client.Update(ctx, "SalesOrderSet", "SO_NUMBER='4500001'", changes)

// Delete
err = client.Delete(ctx, "SalesOrderSet", "SO_NUMBER='4500001'")
```

### Batch requests

```go
batch := client.NewBatch()
batch.Get("MaterialSet('MAT-001')")
batch.Get("MaterialSet('MAT-002')")
results, err := batch.Execute(ctx)
```

### Delta sync

```go
// First run: fetch all and store the token
stream, deltaToken, err := client.From("SalesOrderSet").Delta(ctx, "")

// Later runs: fetch only what changed
stream, newToken, err := client.From("SalesOrderSet").Delta(ctx, deltaToken)
```

---

## Features

- **OData v2 and v4** - automatic version detection from service metadata
- **Streaming** - server-side pagination via Go channels; constant memory on large datasets
- **Fluent query builder** - `$filter`, `$select`, `$expand`, `$orderby`, `$top`, `$skip`, `$count`
- **CRUD** - create, read, update, delete with OData-compliant error handling
- **Delta sync** - incremental updates via delta tokens
- **Batch** - `$batch` request support
- **SAP compatibility** - CSRF token fetching, ETag handling, basic auth and OAuth2
- **Object pooling** - `sync.Pool` reuse of buffers and decoders to reduce GC pressure on high-throughput workloads
- **Thread-safe** - safe for concurrent use across goroutines
- **Tested** - 94%+ coverage; CI runs on Linux, macOS, Windows (Go 1.24 & 1.25)

---

## Extension modules

Extensions are optional and independently versioned. Install only what you need.

| Module | Purpose | Install |
|--------|---------|---------|
| `ext/cache` | Metadata and response caching | `go get github.com/jhonsferg/traverse/ext/cache` |
| `ext/oauth2` | OAuth2 token management | `go get github.com/jhonsferg/traverse/ext/oauth2` |
| `ext/sap` | SAP-specific request handling | `go get github.com/jhonsferg/traverse/ext/sap` |
| `ext/prometheus` | Prometheus metrics | `go get github.com/jhonsferg/traverse/ext/prometheus` |
| `ext/tracing` | OpenTelemetry distributed tracing | `go get github.com/jhonsferg/traverse/ext/tracing` |
| `ext/graphql` | GraphQL bridge (experimental) | `go get github.com/jhonsferg/traverse/ext/graphql` |

Each extension has its own `go.mod`. See the README inside each `ext/` subdirectory.

---

## Running locally

```bash
# Tests
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

- Tests on Linux, macOS and Windows (Go 1.24 and 1.25)
- `go vet` and golangci-lint v2
- CodeQL static analysis
- Trivy vulnerability scan
- TruffleHog secrets scan

Tags are created automatically on every merge to master using Conventional Commits semantics (`feat` → minor bump, `fix`/`ci`/`chore` → patch, `BREAKING CHANGE` footer → major).

---

## Contributing

Contributions are welcome. Please read [CONTRIBUTING.md](CONTRIBUTING.md) before opening a pull request.

Commit messages must follow [Conventional Commits](https://www.conventionalcommits.org/):

```
feat(query): add $apply aggregation support
fix(client): resolve context cancellation race condition
docs(readme): update streaming example
```

Issues and discussion are open on GitHub.

---

## License

MIT — see [LICENSE](LICENSE).
