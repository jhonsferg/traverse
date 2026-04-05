# Errors Reference

traverse uses structured error types that carry context about what went wrong.

## Error types

### traverse.Error

The base error type for all traverse errors:

```go
type Error struct {
    Code       string // OData error code
    Message    string // human-readable message
    StatusCode int    // HTTP status code (0 if not an HTTP error)
    Details    []ErrorDetail
    Inner      error  // wrapped underlying error
}
```

### Sentinel errors

```go
var (
    // ErrNotFound is returned when the entity does not exist (404).
    ErrNotFound = errors.New("entity not found")

    // ErrPreconditionFailed is returned on ETag mismatch (412).
    ErrPreconditionFailed = errors.New("precondition failed")

    // ErrConflict is returned on duplicate key or concurrent modification (409).
    ErrConflict = errors.New("conflict")

    // ErrUnauthorized is returned on 401 responses.
    ErrUnauthorized = errors.New("unauthorized")

    // ErrForbidden is returned on 403 responses.
    ErrForbidden = errors.New("forbidden")

    // ErrTimeout is returned when the request context deadline is exceeded.
    ErrTimeout = errors.New("request timeout")

    // ErrBadRequest wraps 400 responses.
    ErrBadRequest = errors.New("bad request")

    // ErrAsyncFailed is returned when a polled async operation reports failure.
    ErrAsyncFailed = errors.New("async operation failed")
)
```

## Error handling patterns

### Check for specific errors

```go
_, err := client.Collection("Products").Get(ctx, 999, &product)
if errors.Is(err, traverse.ErrNotFound) {
    log.Println("product does not exist")
    return nil, nil
}
if err != nil {
    return nil, fmt.Errorf("fetching product: %w", err)
}
```

### Access structured error details

```go
var te *traverse.Error
if errors.As(err, &te) {
    log.Printf("OData error [%s] %d: %s", te.Code, te.StatusCode, te.Message)
    for _, d := range te.Details {
        log.Printf("  detail: %s - %s", d.Code, d.Message)
    }
}
```

### Concurrency conflict

```go
err := client.Collection("Products").
    IfMatch(etag).
    Update(ctx, id, patch)
if errors.Is(err, traverse.ErrPreconditionFailed) {
    // Reload and retry
    client.Collection("Products").Get(ctx, id, &fresh)
    etag = result.ETag()
    // ... apply patch and retry
}
```

## See also

- [ETag Types](etag.md)
- [Async Poller](async-op.md)
- [Client Reference](client.md)
