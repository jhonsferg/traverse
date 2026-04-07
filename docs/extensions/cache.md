# Cache Extension

The Cache extension (`ext/cache`) adds transparent response caching to traverse, respecting OData ETags and HTTP `Cache-Control` headers.

## Installation

```bash
go get github.com/jhonsferg/traverse/ext/cache@latest
```

## In-memory cache

```go
import (
    "github.com/jhonsferg/traverse"
    "github.com/jhonsferg/traverse/ext/cache/memory"
)

client, err := traverse.New(
    traverse.WithBaseURL("https://api.example.com/odata/"),
    traverse.WithMetadataCache(memory.New(5 * time.Minute)),
)
```

## Redis cache

```go
import "github.com/redis/go-redis/v9"

rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})

cache.Extension(cache.Redis(cache.RedisConfig{
    Client: rdb,
    TTL:    10 * time.Minute,
    Prefix: "traverse:",
}))
```

## ETag-aware caching

When the server returns an `ETag`, subsequent requests automatically include `If-None-Match`. On a 304 response, the cached body is returned without re-parsing:

```go
// First request: GET /Products(1) -> 200 + ETag: "v1"
product, _ := client.Collection("Products").Get(ctx, 1, &p)

// Second request: GET /Products(1) + If-None-Match: "v1" -> 304
// Returns cached value instantly
product2, _ := client.Collection("Products").Get(ctx, 1, &p)
```

## Cache bypass

```go
// Force a fresh request ignoring cache
_, err := client.R().
    SetHeader("Cache-Control", "no-cache").
    GET(ctx, "/Products")
```

## See also

- [Stale Cache](stale-cache.md) -- stale-while-revalidate caching for metadata and reference data
- [ETag & Concurrency](../guides/etag-concurrency.md)
- [Extensions Overview](index.md)
