# Code Generation Overview

`traverse-gen` reads an EDMX metadata document and generates a fully typed Go client, entity structs, and a query builder - eliminating hand-written boilerplate and catching type mismatches at compile time.

## What gets generated

Given an EDMX file, `traverse-gen` produces:

| Output | Description |
|--------|-------------|
| Entity structs | Go structs with correct field types and JSON tags |
| Client | Service client with one method per entity set |
| Query builders | Typed `.Filter()`, `.Select()`, `.Expand()` per entity |
| Enum types | Go `const` blocks for OData enum members |

## Prerequisites

- Go 1.24+
- Access to the OData service metadata endpoint (`$metadata`)

## Quickstart

```bash
# Fetch metadata and generate
traverse-gen \
  --metadata https://services.odata.org/V4/Northwind/Northwind.svc/$metadata \
  --out gen/ \
  --package northwind
```

This creates `gen/client.go`, `gen/entities.go`, and `gen/queries.go`.

## Using the generated client

```go
import "myapp/gen/northwind"

client := northwind.NewClient(traverse.New(traverse.Config{
    BaseURL: "https://services.odata.org/V4/Northwind/Northwind.svc/",
}))

products, err := client.Products().
    Filter(northwind.Product.Price.Gt(100)).
    Select(northwind.Product.Name, northwind.Product.Price).
    Top(10).
    List(ctx)
```

## Next steps

- [Installation](installation.md) - install `traverse-gen`
- [Usage](usage.md) - all CLI flags and options
- [Type Mapping](type-mapping.md) - how OData types map to Go
- [Generated Client](generated-client.md) - working with generated code
