# Offline Store

Persist OData query responses to a local JSON cache so queries can be served without network access, using the `ext/offline` extension.

## Installation

```bash
go get github.com/jhonsferg/traverse/ext/offline
```

## Quick Start

```go
import "github.com/jhonsferg/traverse/ext/offline"

// Create a persistent store backed by a local directory.
store, err := offline.NewStore("/var/cache/myapp/odata")
if err != nil {
    log.Fatal(err)
}

// Manually cache a response.
body := []byte(`{"value":[{"ID":1,"Name":"Alice"}]}`)
if err := store.Set("/Customers", body); err != nil {
    log.Printf("cache write failed: %v", err)
}

// Retrieve a cached response.
data, err := store.Get("/Customers")
if errors.Is(err, offline.ErrNotFound) {
    // fall back to live request
}
```

## API Reference

### `Store`

Thread-safe store that persists each cached entry as a JSON file under a directory. The file name is derived from a SHA-256 hash of the OData path, so special characters and query parameters are handled safely.

```go
type Store struct { /* unexported */ }

func NewStore(dir string) (*Store, error)
```

`dir` is created with permissions `0750` if it does not exist.

#### `Set`

```go
func (s *Store) Set(path string, data []byte) error
```

Stores the raw OData JSON response for `path`. `path` is the OData entity set path used as a cache key, e.g. `"/Customers"` or `"/Customers(1)"`.

#### `Get`

```go
func (s *Store) Get(path string) ([]byte, error)
```

Returns the cached bytes for `path`. Returns `offline.ErrNotFound` if no entry exists.

#### `Delete`

```go
func (s *Store) Delete(path string) error
```

Removes the cached entry for `path`. A no-op if the entry does not exist.

#### `Keys`

```go
func (s *Store) Keys() ([]string, error)
```

Returns all cached OData paths.

#### `Clear`

```go
func (s *Store) Clear() error
```

Removes all cached entries.

### `ErrNotFound`

```go
var ErrNotFound = errors.New("offline: entry not found")
```

Returned by `Get` when the requested path has no cached entry.

## Cache layout

```
/var/cache/myapp/odata/
├── index.json              # path → hash mapping
├── a3f2...json             # cached response for /Customers
└── 7c1d...json             # cached response for /Customers(1)
```

The index file maps SHA-256 hashes back to their original paths, which is what `Keys()` reads.

## Notes / Limitations

- There is no built-in TTL or expiry; entries remain until explicitly deleted or `Clear()` is called.
- The store does not automatically intercept traverse client HTTP calls. Callers must populate and query the store manually (or wrap their own transport).
- File permissions are `0600` for cache entries and `0750` for the directory.
