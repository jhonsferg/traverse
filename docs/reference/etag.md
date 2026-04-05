# ETag Types

traverse exposes strongly-typed ETag values to prevent accidental use of raw strings in concurrency-safe operations.

## Types

```go
// ETag represents a validated OData ETag value.
type ETag struct {
    value string
    weak  bool
}

// Strong returns a strong ETag from a quoted string value.
func Strong(v string) ETag

// Weak returns a weak ETag (W/"value").
func Weak(v string) ETag

// Any returns the wildcard ETag (*) used with If-Match to match any version.
var Any ETag
```

## Constructing ETags

```go
// From a server response header
etag := traverse.ETagFromHeader(resp.Header.Get("ETag"))

// Explicit construction
etag := traverse.Strong("abc123")
weak  := traverse.Weak("abc123")  // W/"abc123"
```

## Using ETags in requests

```go
// Conditional update - only succeeds if the entity still matches the ETag
err := client.Collection("Products").
    IfMatch(etag).
    Update(ctx, productID, patch)

// Conditional get - returns 304 if unchanged
err := client.Collection("Products").
    IfNoneMatch(etag).
    Get(ctx, productID, &product)
```

## ETag from a prior read

```go
var product Product
result, err := client.Collection("Products").Get(ctx, 1, &product)
if err != nil {
    log.Fatal(err)
}

etag := result.ETag() // typed ETag from the response

// Later: conditional update
patch := ProductPatch{Price: 99.99}
err = client.Collection("Products").
    IfMatch(etag).
    Update(ctx, 1, patch)
if errors.Is(err, traverse.ErrPreconditionFailed) {
    log.Println("entity was modified by someone else")
}
```

## ETag string representation

```go
etag := traverse.Strong("v42")
fmt.Println(etag.String()) // "v42"
fmt.Println(etag.Header()) // "v42" (for use in headers)

weak := traverse.Weak("v42")
fmt.Println(weak.Header()) // W/"v42"
```

## See also

- [ETag & Concurrency guide](../guides/etag-concurrency.md)
- [Client Reference](client.md)
