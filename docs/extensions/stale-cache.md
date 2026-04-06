# Stale-While-Revalidate Cache

The `ext/cache/stale` package provides a general-purpose stale-while-revalidate cache. It serves cached data immediately, even if it is past its TTL, while optionally refreshing the entry in the background so the *next* caller gets fresh data.

## Installation

```bash
go get github.com/jhonsferg/traverse/ext/cache/stale@latest
```

## How it works

Every cache entry passes through three states based on its age:

| Age | Behaviour |
|-----|-----------|
| `< TTL` | Returned as-is -- no refresh needed |
| `TTL <= age < StaleTTL` | Stale data returned immediately; background refresh triggered when `BackgroundSync: true` |
| `>= StaleTTL` | Synchronous refresh -- caller waits for fresh data |

## Quick start

```go
import (
    "context"
    "github.com/jhonsferg/traverse/ext/cache/stale"
)

cache := stale.New(stale.Config{
    TTL:            30 * time.Second,  // serve fresh for 30 s
    StaleTTL:       5 * time.Minute,   // serve stale for up to 5 min
    BackgroundSync: true,              // refresh in the background
})

data, err := cache.Get(ctx, "products", func(ctx context.Context) ([]byte, error) {
    // called only when data is absent or too stale
    return fetchProductsFromAPI(ctx)
})
```

## Configuration

```go
type Config struct {
    // TTL is how long data is considered fresh.
    TTL time.Duration

    // StaleTTL is the maximum age at which stale data may still be served.
    // Must be >= TTL.
    StaleTTL time.Duration

    // BackgroundSync triggers an async refresh when stale data is served.
    BackgroundSync bool
}
```

## Cache operations

### Get

```go
data, err := c.Get(ctx, key, refreshFn)
```

`refreshFn` is called with a `context.Context` and must return `([]byte, error)`. Background refreshes run with a 30-second timeout derived from `context.Background()`.

### Invalidate

```go
c.Invalidate("products") // remove one entry
```

### Clear

```go
c.Clear() // remove all entries
```

### OnRefresh callback

Register a hook that fires after every successful refresh -- useful for logging or metrics:

```go
c.OnRefresh(func(key string, data []byte) {
    log.Printf("cache refreshed: key=%s bytes=%d", key, len(data))
})
```

## Integration with traverse

The stale cache works well as a pre-fetch layer for metadata or reference data that changes infrequently:

```go
metaCache := stale.New(stale.Config{
    TTL:            1 * time.Minute,
    StaleTTL:       10 * time.Minute,
    BackgroundSync: true,
})

func getMetadata(ctx context.Context) (*traverse.Metadata, error) {
    raw, err := metaCache.Get(ctx, "metadata", func(ctx context.Context) ([]byte, error) {
        md, err := client.Metadata(ctx)
        if err != nil {
            return nil, err
        }
        return json.Marshal(md)
    })
    if err != nil {
        return nil, err
    }

    var md traverse.Metadata
    return &md, json.Unmarshal(raw, &md)
}
```

## Thread safety

`Cache` is safe for concurrent use. All internal state is protected by a `sync.RWMutex`.

## See also

- [Cache extension](cache.md) -- HTTP-level response caching with ETag support
- [Extensions Overview](index.md)
