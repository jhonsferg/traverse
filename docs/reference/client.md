# Client Reference

## traverse.New

```go
func New(config Config) *Client
```

Creates a new traverse client with the given configuration.

## Config

```go
type Config struct {
    // BaseURL is the service root URL (required).
    // Example: "https://api.example.com/odata/"
    BaseURL string

    // Version specifies OData protocol version.
    // Accepted: "v2", "v4". Default: auto-detect from metadata.
    Version string

    // Transport is the underlying relay client.
    // If nil, a default relay.Client is created.
    Transport *relay.Client

    // Extension adds optional capabilities (SAP, OAuth2, tracing, etc.).
    Extension Extension

    // Headers sets default headers sent on every request.
    Headers map[string]string

    // Timeout is the default request timeout. Default: 30s.
    Timeout time.Duration

    // MaxRetries configures the retry count for transient errors. Default: 3.
    MaxRetries int
}
```

## Client methods

### Collection

```go
func (c *Client) Collection(name string) *CollectionBuilder
```

Returns a builder for the named entity set.

### Entity

```go
func (c *Client) Entity(collection string, key any) *EntityBuilder
```

Returns a builder for a single entity by key. Key can be a scalar value or a struct for composite keys.

### Batch

```go
func (c *Client) Batch() *BatchBuilder
```

Returns a builder for constructing an OData `$batch` request.

### FunctionImport

```go
func (c *Client) FunctionImport(name string) *FunctionBuilder
```

Returns a builder for calling an OData function or action import.

### Metadata

```go
func (c *Client) Metadata(ctx context.Context) (*Metadata, error)
```

Fetches and parses the service `$metadata` document.

### Relay

```go
func (c *Client) Relay() *relay.Client
```

Returns the underlying relay client for low-level HTTP access.

## CollectionBuilder methods

| Method | Description |
|--------|-------------|
| `Filter(expr string) *CollectionBuilder` | Append `$filter` clause |
| `Select(fields ...string) *CollectionBuilder` | Set `$select` |
| `Expand(nav ...string) *CollectionBuilder` | Set `$expand` |
| `OrderBy(field string, desc ...bool) *CollectionBuilder` | Set `$orderby` |
| `Top(n int) *CollectionBuilder` | Set `$top` |
| `Skip(n int) *CollectionBuilder` | Set `$skip` |
| `Count() *CollectionBuilder` | Include `$count=true` |
| `List(ctx, dest) (ListResult, error)` | Execute and decode list |
| `Get(ctx, key, dest) error` | Get single entity |
| `Create(ctx, body, dest) error` | POST new entity |
| `Update(ctx, key, body) error` | PATCH entity |
| `Replace(ctx, key, body) error` | PUT entity |
| `Delete(ctx, key) error` | DELETE entity |
| `Paginate(ctx, opts) (*Paginator[T], error)` | Typed pagination |

## See also

- [Query Builder API](query-builder.md)
- [Paginator[T]](paginator.md)
- [Errors](errors.md)
