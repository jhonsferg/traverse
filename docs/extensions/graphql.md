# GraphQL Bridge Extension

The GraphQL Bridge (`ext/graphql`) translates OData queries into GraphQL queries automatically, allowing you to use the traverse API against a GraphQL backend.

## Installation

```bash
go get github.com/jhonsferg/traverse/ext/graphql@latest
```

## Configuration

```go
import (
    "github.com/jhonsferg/traverse"
    "github.com/jhonsferg/traverse/ext/graphql"
)

client := traverse.New(traverse.Config{
    BaseURL: "https://api.example.com/graphql",
    Extension: graphql.Extension(graphql.Config{
        Endpoint: "https://api.example.com/graphql",
    }),
})
```

## Query translation

| OData operation | GraphQL equivalent |
|-----------------|-------------------|
| `$filter` | `where` argument |
| `$select` | field selection |
| `$expand` | nested fields |
| `$top` / `$skip` | `first` / `offset` |
| `$orderby` | `orderBy` argument |
| `$count` | `totalCount` field |

## Example

```go
// OData: GET /Products?$filter=Price gt 10&$select=Name,Price&$top=5
// Translated to GraphQL: query { products(where:{price:{gt:10}}, first:5) { name price } }

var products []Product
_, err := client.Collection("Products").
    Filter("Price gt 10").
    Select("Name", "Price").
    Top(5).
    List(ctx, &products)
```

## Schema introspection

```go
schema, err := graphql.IntrospectSchema(ctx, "https://api.example.com/graphql")
// Use schema to generate traverse entity definitions
```

## Limitations

- Mutations (Create/Update/Delete) require explicit GraphQL mutation templates.
- Not all OData functions have a direct GraphQL equivalent.
- Custom scalar types require explicit type mapping configuration.

## See also

- [Extensions Overview](index.md)
- [Query Builder API](../reference/query-builder.md)
