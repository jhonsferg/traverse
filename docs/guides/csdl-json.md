# CSDL JSON

Parse OData Common Schema Definition Language (CSDL) JSON v4.01 metadata directly from a JSON endpoint or byte slice — no EDMX/XML required.

## Quick Start

```go
import (
    "context"
    "net/http"
    "github.com/jhonsferg/traverse"
)

// From a JSON byte slice:
data := []byte(`{ "northwind.model": { ... } }`)
meta, err := traverse.ParseCSDLJSON(data)

// From an io.Reader (e.g. HTTP response body):
resp, _ := http.Get("https://api.example.com/odata/$metadata")
defer resp.Body.Close()
meta, err := traverse.ParseCSDLJSONReader(resp.Body)
```

## Auto-detection by Content-Type

When fetching `$metadata`, traverse automatically selects the parser based on the `Content-Type` of the response:

| Content-Type | Parser |
|---|---|
| `application/xml`, `text/xml` | EDMX (XML) |
| `application/json`, `application/csdl+json` | CSDL JSON (`ParseCSDLJSONReader`) |

If you download the metadata file manually, pick the right function based on the file contents.

## API Reference

### `ParseCSDLJSON`

```go
func ParseCSDLJSON(data []byte) (*Metadata, error)
```

Parses a CSDL JSON document from a byte slice. Returns `ErrMetadataInvalid` (wrapped) if the input is empty or not valid JSON.

### `ParseCSDLJSONReader`

```go
func ParseCSDLJSONReader(r io.Reader) (*Metadata, error)
```

Streaming variant that reads from any `io.Reader`. Returns `ErrMetadataInvalid` (wrapped) if the reader is nil or the JSON is malformed.

### `*Metadata`

Both functions return the same `*Metadata` type as the EDMX parser, so all subsequent API calls (query building, code generation, OpenAPI export) work identically regardless of the source format.

```go
type Metadata struct {
    EntityTypes  []EntityType
    EntitySets   []EntitySetInfo
    Associations []Association
    Actions      []ActionInfo
    Functions    []FunctionInfo
    ComplexTypes []ComplexType
    EnumTypes    []EnumType
}
```

## CSDL JSON vs EDMX

| Aspect | EDMX (XML) | CSDL JSON |
|--------|-----------|-----------|
| Format | XML | JSON |
| OData version | v2 and v4 | v4.01 only |
| Content-Type | `application/xml` | `application/json` |
| Typical endpoint | `$metadata` | `$metadata` with `Accept: application/json` |
| File extension | `.xml` | `.json` |

## Loading metadata from a live service

```go
req, _ := http.NewRequestWithContext(ctx, http.MethodGet,
    "https://api.example.com/odata/$metadata", nil)
req.Header.Set("Accept", "application/csdl+json")

resp, err := http.DefaultClient.Do(req)
// resp.Body is CSDL JSON → use ParseCSDLJSONReader
meta, err := traverse.ParseCSDLJSONReader(resp.Body)
```

## Notes / Limitations

- CSDL JSON v4.01 is supported. OData v2 services expose EDMX only.
- Schema annotations beyond basic type information (vocabulary terms, etc.) are parsed where present; see the [Vocabulary guide](vocabulary.md) for details.
- `ErrMetadataInvalid` is a sentinel that can be checked with `errors.Is`.
