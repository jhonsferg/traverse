# OpenAPI Export

Convert an OData `*Metadata` document into an OpenAPI 3.1 specification using the `ext/openapi` extension.

## Installation

```bash
go get github.com/jhonsferg/traverse/ext/openapi
```

## Quick Start

```go
import (
    "encoding/json"
    "os"

    "github.com/jhonsferg/traverse"
    "github.com/jhonsferg/traverse/ext/openapi"
)

// 1. Parse OData metadata (EDMX or CSDL JSON).
meta, err := traverse.ParseMetadataFromURL(ctx, "https://api.example.com/odata/$metadata")
if err != nil {
    log.Fatal(err)
}

// 2. Export to OpenAPI 3.1.
doc, err := openapi.Export(meta,
    openapi.WithTitle("My OData API"),
    openapi.WithVersion("2.0.0"),
    openapi.WithServerURL("https://api.example.com/odata"),
    openapi.WithTagsFromNamespace(),
)
if err != nil {
    log.Fatal(err)
}

// 3. Serialize to JSON.
enc := json.NewEncoder(os.Stdout)
enc.SetIndent("", "  ")
enc.Encode(doc)
```

## Full flow: EDMX → OpenAPI → YAML

```go
// Parse from a local EDMX file.
f, _ := os.Open("metadata.xml")
defer f.Close()
meta, _ := traverse.ParseMetadata(f)

doc, _ := openapi.Export(meta, openapi.WithTitle("Northwind"))

// Serialize to YAML using gopkg.in/yaml.v3.
raw, _ := json.Marshal(doc)
var m interface{}
json.Unmarshal(raw, &m)
yamlBytes, _ := yaml.Marshal(m)
os.WriteFile("openapi.yaml", yamlBytes, 0o644)
```

## API Reference

### `Export`

```go
func Export(meta *traverse.Metadata, opts ...Option) (*OpenAPI, error)
```

Converts an OData `*Metadata` into an `*OpenAPI` document. Returns an error if `meta` is nil.

For each entity set, `Export` generates:

- `GET /{EntitySet}` – list operation with `$filter`, `$select`, `$orderby`, `$top`, `$skip` parameters
- `POST /{EntitySet}` – create operation with a request body schema
- `GET /{EntitySet}({key})` – read by key
- `PATCH /{EntitySet}({key})` – update by key
- `DELETE /{EntitySet}({key})` – delete by key

Entity type properties are mapped to JSON Schema types inside `components/schemas`.

### Options

| Option | Description |
|--------|-------------|
| `WithTitle(title string)` | Sets `info.title`. Default: `"OData Service"` |
| `WithVersion(version string)` | Sets `info.version`. Default: `"1.0.0"` |
| `WithServerURL(url string)` | Adds a server entry. Can be called multiple times. |
| `WithTagsFromNamespace()` | Groups operations by OData namespace as tags. |

### Output types

```go
type OpenAPI struct {
    OpenAPI    string               `json:"openapi"`    // "3.1.0"
    Info       Info                 `json:"info"`
    Servers    []Server             `json:"servers,omitempty"`
    Paths      map[string]*PathItem `json:"paths"`
    Components *Components          `json:"components,omitempty"`
}
```

The struct is JSON-serialisable directly. Use `encoding/json` or `gopkg.in/yaml.v3` to write JSON or YAML output.

## Notes / Limitations

- Only entity sets are exported as paths; OData actions and functions are not currently mapped.
- Navigation properties generate `$expand` query parameters but not nested path items.
- The output targets OpenAPI 3.1.0; OpenAPI 2.0 (Swagger) is not supported.
