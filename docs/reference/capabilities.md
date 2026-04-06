# Capabilities Validation

The `capabilities` API lets you parse the OData v4 Capabilities vocabulary from a service's `$metadata` document and enforce restrictions at query-build time -- before any HTTP request is sent.

## Overview

OData services annotate their entity sets with vocabulary terms such as `Capabilities.FilterRestrictions` and `Capabilities.SortRestrictions` to advertise what operations are permitted. traverse can read these annotations and surface clear errors when a query violates them.

## Parsing capabilities

```go
import "github.com/jhonsferg/traverse"

// Fetch raw metadata XML
metaBytes, err := client.MetadataBytes(ctx)
if err != nil {
    log.Fatal(err)
}

registry, err := traverse.ParseCapabilities(metaBytes)
if err != nil {
    log.Fatal(err)
}
```

`ParseCapabilities` returns a `*CapabilitiesRegistry` populated from every `Capabilities.*` annotation in the EDMX document.

## Enabling validation on a client

Pass the registry as a client option:

```go
client := traverse.New(traverse.Config{
    BaseURL: "https://api.example.com/odata/",
}, traverse.WithCapabilitiesValidation(registry))
```

From this point on, query builders that violate the capabilities will return a `*CapabilityError` immediately, without making a network call.

## CapabilityError

```go
type CapabilityError struct {
    EntitySet string
    Operation string // "filter", "sort", or "expand"
    Property  string // populated for property-level restrictions
    Message   string
}
```

```go
products := traverse.From[Product](client, "Products")

_, err := products.Filter("CreatedAt gt 2024-01-01").List(ctx)
var capErr *traverse.CapabilityError
if errors.As(err, &capErr) {
    fmt.Printf("%s: %s on %s\n", capErr.EntitySet, capErr.Operation, capErr.Property)
    // Products: filter on CreatedAt
}
```

## EntityCapabilities

`CapabilitiesRegistry.Get` returns an `EntityCapabilities` struct:

```go
type EntityCapabilities struct {
    Filterable                     bool
    NonFilterableProperties        []string
    Sortable                       bool
    NonSortableProperties          []string
    ExpandableNavigationProperties []string
    Insertable                     bool
    Updatable                      bool
    Deletable                      bool
}
```

When an entity set has no annotations, all capabilities default to `true` and both restriction lists are empty.

## Inspecting capabilities directly

```go
cap := registry.Get("Products")
if !cap.Filterable {
    fmt.Println("Products cannot be filtered")
}
fmt.Println("Non-filterable:", cap.NonFilterableProperties)
fmt.Println("Non-sortable:",   cap.NonSortableProperties)
```

## Supported Capabilities annotations

| Annotation term | Fields checked |
|----------------|----------------|
| `Capabilities.FilterRestrictions` | `Filterable`, `NonFilterableProperties` |
| `Capabilities.SortRestrictions` | `Sortable`, `NonSortableProperties` |
| `Capabilities.ExpandRestrictions` | `ExpandableProperties` |
| `Capabilities.InsertRestrictions` | `Insertable` |
| `Capabilities.UpdateRestrictions` | `Updatable` |
| `Capabilities.DeleteRestrictions` | `Deletable` |

## Complete example

```go
package main

import (
    "context"
    "errors"
    "fmt"
    "log"

    "github.com/jhonsferg/traverse"
)

type Product struct {
    ID    int    `json:"ProductID"`
    Name  string `json:"ProductName"`
    Price float64 `json:"UnitPrice"`
}

func main() {
    ctx := context.Background()

    // 1. Create a bare client to fetch metadata
    bare := traverse.New(traverse.Config{
        BaseURL: "https://api.example.com/odata/",
    })

    metaBytes, err := bare.MetadataBytes(ctx)
    if err != nil {
        log.Fatal(err)
    }

    // 2. Parse capabilities
    registry, err := traverse.ParseCapabilities(metaBytes)
    if err != nil {
        log.Fatal(err)
    }

    // 3. Create a validated client
    client := traverse.New(traverse.Config{
        BaseURL: "https://api.example.com/odata/",
    }, traverse.WithCapabilitiesValidation(registry))

    // 4. Queries are checked before any HTTP call
    products := traverse.From[Product](client, "Products")

    _, err = products.Filter("CreatedAt gt 2024-01-01").List(ctx)
    var capErr *traverse.CapabilityError
    if errors.As(err, &capErr) {
        fmt.Println("Caught capability error:", capErr)
    }
}
```

## See also

- [Query Builder](../guides/query-builder.md)
- [Schema Validation](../guides/schema-validation.md)
- [Reference: Query Builder API](query-builder.md)
