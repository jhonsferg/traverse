# Audit Trail

Record every OData HTTP request and response for compliance and debugging using the `ext/audit` extension. The middleware intercepts the relay transport layer and emits structured `AuditEntry` values to any `AuditLogger` implementation.

## Installation

```bash
go get github.com/jhonsferg/traverse/ext/audit
```

## Quick Start

```go
import (
    "log"
    "github.com/jhonsferg/traverse"
    "github.com/jhonsferg/traverse/ext/audit"
)

logger := audit.AuditLoggerFunc(func(ctx context.Context, e audit.AuditEntry) {
    log.Printf("[AUDIT] %s %s %s key=%s status=%d dur=%s user=%s err=%s",
        e.Timestamp.Format(time.RFC3339),
        e.Operation, e.EntitySet, e.EntityKey,
        e.StatusCode, e.Duration, e.UserID, e.Error,
    )
})

client := traverse.New(traverse.Config{
    BaseURL:    "https://api.example.com/odata/",
    HTTPOption: audit.WithAuditTrail(logger),
})

// Attach user/request ID to context for richer audit entries.
ctx := audit.WithUser(context.Background(), "user-123")
ctx  = audit.WithRequestID(ctx, "req-abc")
```

## Audit log format

Each operation produces one `AuditEntry`:

```go
type AuditEntry struct {
    Timestamp  time.Time     // when the request was dispatched
    Operation  OperationType // READ | CREATE | UPDATE | DELETE | BATCH
    EntitySet  string        // e.g. "Orders"
    EntityKey  string        // e.g. "42" (empty for collection operations)
    URL        string        // full request URL
    StatusCode int           // HTTP response status code
    Duration   time.Duration // round-trip time
    UserID     string        // from context via audit.WithUser
    RequestID  string        // from context via audit.WithRequestID
    Error      string        // non-empty if the request failed
}
```

### Operation types

| `OperationType` | Triggered by |
|-----------------|-------------|
| `READ`   | `GET` |
| `CREATE` | `POST` |
| `UPDATE` | `PATCH` or `PUT` |
| `DELETE` | `DELETE` |
| `BATCH`  | `POST /$batch` |

## API Reference

### `AuditLogger`

```go
type AuditLogger interface {
    Log(ctx context.Context, entry AuditEntry)
}
```

Implement this interface to send entries to any sink (structured logger, database, SIEM, etc.).

### `AuditLoggerFunc`

```go
type AuditLoggerFunc func(ctx context.Context, entry AuditEntry)
```

Adapts a plain function to `AuditLogger`.

### `WithAuditTrail`

```go
func WithAuditTrail(logger AuditLogger) relay.Option
```

Returns a `relay.Option` that wraps the HTTP transport. Pass it as `traverse.Config.HTTPOption`.

### `Middleware`

```go
func Middleware(logger AuditLogger) func(http.RoundTripper) http.RoundTripper
```

Lower-level variant for use when constructing a plain `http.Client` or a custom relay client.

### Context helpers

```go
func WithUser(ctx context.Context, userID string) context.Context
func WithRequestID(ctx context.Context, requestID string) context.Context
```

Attach a user ID or request ID to a context. Both values are automatically read and included in `AuditEntry` by the middleware.

### `InMemoryAuditLog`

A thread-safe in-memory logger useful in tests:

```go
log := &audit.InMemoryAuditLog{}
client := traverse.New(traverse.Config{
    BaseURL:    "https://api.example.com/odata/",
    HTTPOption: audit.WithAuditTrail(log),
})
// ... run operations ...
entries := log.Entries() // []AuditEntry
```

## Notes / Limitations

- The middleware records entries after the response is received; entries for timed-out requests will have a non-zero `Error` and a zero `StatusCode`.
- Request/response bodies are not captured to avoid logging PII or large payloads.
- `EntitySet` and `EntityKey` are parsed heuristically from the URL path; custom routing may produce unexpected values.
