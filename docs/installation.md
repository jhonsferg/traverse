# Installation

## Requirements

- Go 1.24 or later
- Module-aware project (`go mod init`)

traverse uses generics (Go 1.18+) and other features that require Go 1.24.

## Core Module

```bash
go get github.com/jhonsferg/traverse
```

This installs the core traverse module and its only external dependency, [relay](https://github.com/jhonsferg/relay), which is resolved automatically.

## Verify Installation

```go
package main

import (
    "fmt"
    "github.com/jhonsferg/traverse"
)

func main() {
    c, _ := traverse.New(traverse.WithBaseURL("https://example.com/odata"))
    defer c.Close()
    fmt.Println("traverse ready")
}
```

```bash
go run .
# traverse ready
```

## Extension Modules

Each extension is a separate Go module with its own `go get`:

| Extension | Import Path | Install |
|-----------|-------------|---------|
| SAP adapter | `github.com/jhonsferg/traverse/ext/sap` | `go get github.com/jhonsferg/traverse/ext/sap` |
| OAuth2 | `github.com/jhonsferg/traverse/ext/oauth2` | `go get github.com/jhonsferg/traverse/ext/oauth2` |
| Prometheus | `github.com/jhonsferg/traverse/ext/prometheus` | `go get github.com/jhonsferg/traverse/ext/prometheus` |
| OpenTelemetry | `github.com/jhonsferg/traverse/ext/tracing` | `go get github.com/jhonsferg/traverse/ext/tracing` |
| Memory Cache | `github.com/jhonsferg/traverse/ext/cache/memory` | `go get github.com/jhonsferg/traverse/ext/cache/memory` |
| Redis Cache | `github.com/jhonsferg/traverse/ext/cache/redis` | `go get github.com/jhonsferg/traverse/ext/cache/redis` |
| GraphQL Bridge | `github.com/jhonsferg/traverse/ext/graphql` | `go get github.com/jhonsferg/traverse/ext/graphql` |

Extensions are optional - only add what you need.

## Code Generator

Install `traverse-gen` to generate typed Go structs and query builders from OData `$metadata`:

```bash
go install github.com/jhonsferg/traverse/cmd/traverse-gen@latest
```

Verify:

```bash
traverse-gen --version
```

See [Code Generation](codegen/index.md) for usage details.

## Using traverse in a Monorepo (go.work)

If your project uses a Go workspace (`go.work`), add traverse alongside your own modules:

```
go 1.24

use (
    .
    ./services/api
    ./services/worker
)
```

```bash
go work download github.com/jhonsferg/traverse
```

Or pin a specific version in your workspace:

```bash
go work edit -replace github.com/jhonsferg/traverse=../local-traverse
```

!!! tip "Updating traverse"
    Run `go get github.com/jhonsferg/traverse@latest` to upgrade to the latest release, then `go mod tidy` to clean up.

!!! note "relay dependency"
    relay is traverse's HTTP transport layer. It is resolved automatically when you `go get traverse`. You do not need to add relay separately unless you want to configure it directly.
