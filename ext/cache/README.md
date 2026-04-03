# Cache Extensions

Traverse provides multiple caching implementations for metadata caching to reduce network round-trips and improve performance.

## Available Implementations

- **Memory Cache**: In-process, TTL-based, no external dependencies
- **Redis Cache**: Distributed, multi-instance, network-based

## Memory Cache

Fast, in-process caching with automatic TTL expiration.

### Installation

Memory cache is included in traverse. No additional dependencies required.

### Usage

```go
import (
    "time"
    "github.com/jhonsferg/traverse"
    "github.com/jhonsferg/traverse/ext/cache/memory"
)

// Create cache with 1-hour TTL
cache := memory.New(time.Hour)

client, _ := traverse.New(
    traverse.WithBaseURL("https://odata.example.com/v2"),
    traverse.WithVersion(traverse.ODataV4),
    traverse.WithMetadataCache(cache),
)

// All metadata operations now use cache
metadata, _ := client.Metadata()
```

### Features

- **Thread-safe**: Uses `sync.RWMutex` for concurrent access
- **TTL-based expiration**: Entries automatically expire after TTL
- **Lazy cleanup**: Expired entries are removed on access
- **Fast lookups**: ~1-5ms per operation
- **Zero dependencies**: No external packages required

### Configuration

```go
import "time"

// Development: 1 hour
cache := memory.New(time.Hour)

// Production with frequent changes: 15 minutes
cache := memory.New(15 * time.Minute)

// Production stable: 24 hours
cache := memory.New(24 * time.Hour)

// No expiration (manual Clear only)
cache := memory.New(0)
```

### Performance Characteristics

| Operation | Latency | Memory |
|-----------|---------|--------|
| **Cache Hit** | 1-5ms | Minimal |
| **Cache Miss** | 100-500ms | ~1KB per entity |
| **Concurrent Reads** | 1-5ms | Lock-free on read |

### When to Use Memory Cache

✅ **Use when:**
- Single instance deployment
- Metadata rarely changes
- Development/testing environments
- Low-volume services (<100 queries/sec)

❌ **Don't use when:**
- Multiple instances (each needs separate cache)
- Frequent metadata updates across instances
- Memory is severely constrained
- Need shared cache across services

### Example: Development Setup

```go
package main

import (
    "time"
    "github.com/jhonsferg/traverse"
    "github.com/jhonsferg/traverse/ext/cache/memory"
)

func main() {
    // Create cache
    cache := memory.New(time.Hour)
    
    client, _ := traverse.New(
        traverse.WithBaseURL("https://dev-sap.example.com/odata/v2"),
        traverse.WithVersion(traverse.ODataV4),
        traverse.WithMetadataCache(cache),
    )
    
    // First call: fetches metadata from service
    m1, _ := client.Metadata()
    println("Loaded:", len(m1.EntityTypes), "entity types")
    
    // Subsequent calls: use cached data (instant)
    m2, _ := client.Metadata()
    println("Cache size:", cache.Size(), "entries")
    
    // Manual cache operations
    cache.Clear()
    println("Cache cleared")
}
```

## Redis Cache

Distributed caching for multi-instance deployments. Share cache across multiple services.

### Installation

```bash
go get github.com/redis/go-redis/v9
```

### Usage

```go
import (
    "time"
    "github.com/jhonsferg/traverse"
    "github.com/jhonsferg/traverse/ext/cache/redis"
    redisLib "github.com/redis/go-redis/v9"
)

// Connect to Redis
cfg := &redis.Config{
    Addr:      "localhost:6379",
    TTL:       time.Hour,
    KeyPrefix: "traverse:",
}

cache, err := redis.New(cfg)
if err != nil {
    panic(err)
}
defer cache.Close()

client, _ := traverse.New(
    traverse.WithBaseURL("https://odata.example.com/v2"),
    traverse.WithVersion(traverse.ODataV4),
    traverse.WithMetadataCache(cache),
)

// All instances share the same cache
metadata, _ := client.Metadata()
```

### Configuration

```go
// Basic configuration
cfg := &redis.Config{
    Addr: "localhost:6379",
    TTL:  time.Hour,
}

// With authentication
cfg := &redis.Config{
    Addr:     "redis.example.com:6379",
    Password: "secret",
    DB:       0,
    TTL:      time.Hour,
}

// With custom prefix
cfg := &redis.Config{
    Addr:      "localhost:6379",
    KeyPrefix: "myservice:odata:",
    TTL:       2 * time.Hour,
}

cache, err := redis.New(cfg)
```

### Performance Characteristics

| Operation | Latency | Network | Memory |
|-----------|---------|---------|--------|
| **Cache Hit** | 5-10ms | ~1KB | Server-side |
| **Cache Miss** | 100-500ms | ~1KB | Server-side |
| **Network Round-trip** | 1-5ms | Per operation | N/A |

### When to Use Redis Cache

✅ **Use when:**
- Multiple instances (horizontal scaling)
- Kubernetes deployments
- Load-balanced services
- Metadata rarely changes but used frequently
- Want persistent cache across restarts
- High-volume services (>100 queries/sec)

❌ **Don't use when:**
- Single instance (memory cache is faster)
- Redis infrastructure unavailable
- Metadata highly volatile
- Network latency is critical (<5ms required)

### Example: Kubernetes Deployment

```go
package main

import (
    "os"
    "time"
    "github.com/jhonsferg/traverse"
    "github.com/jhonsferg/traverse/ext/cache/redis"
)

func init() {
    // Redis address from K8s service DNS
    redisAddr := os.Getenv("REDIS_URL")
    if redisAddr == "" {
        redisAddr = "redis:6379"
    }
    
    cfg := &redis.Config{
        Addr:      redisAddr,
        TTL:       time.Hour,
        KeyPrefix: "odata:",
    }
    
    cache, _ := redis.New(cfg)
    
    client, _ := traverse.New(
        traverse.WithBaseURL("https://sap-odata.example.com/v2"),
        traverse.WithVersion(traverse.ODataV4),
        traverse.WithMetadataCache(cache),
    )
}
```

### Redis Key Format

Keys are stored with format: `{keyPrefix}{serviceName}:{entitySet}`

Example with prefix `"traverse:"`:
```
traverse:metadata_edmx
traverse:service_document
```

## Implementing Custom Cache

Implement the `traverse.CacheStore` interface:

```go
import "github.com/jhonsferg/traverse"

type CustomCache struct {
    // Your implementation
}

// Required: Get metadata from cache
func (c *CustomCache) Get(key string) (*traverse.Metadata, bool) {
    // Return cached metadata or (nil, false) if not found
    return metadata, found
}

// Required: Store metadata in cache
func (c *CustomCache) Set(key string, metadata *traverse.Metadata) {
    // Store metadata in cache with appropriate TTL/expiration
}

// Required: Clear all cache entries
func (c *CustomCache) Clear() {
    // Clear all cached data
}

// Usage
cache := &CustomCache{}
client, _ := traverse.New(
    traverse.WithBaseURL("https://example.com"),
    traverse.WithMetadataCache(cache),
)
```

### Example: Database-Backed Cache

```go
import (
    "database/sql"
    "encoding/json"
    "github.com/jhonsferg/traverse"
)

type DBCache struct {
    db *sql.DB
}

func (c *DBCache) Get(key string) (*traverse.Metadata, bool) {
    var data []byte
    err := c.db.QueryRow(
        "SELECT data FROM metadata_cache WHERE key=? AND expires_at > CURRENT_TIMESTAMP",
        key,
    ).Scan(&data)
    
    if err != nil {
        return nil, false
    }
    
    var metadata traverse.Metadata
    json.Unmarshal(data, &metadata)
    return &metadata, true
}

func (c *DBCache) Set(key string, metadata *traverse.Metadata) {
    data, _ := json.Marshal(metadata)
    c.db.Exec(
        "INSERT OR REPLACE INTO metadata_cache VALUES(?, ?, datetime('now', '+1 hour'))",
        key, data,
    )
}

func (c *DBCache) Clear() {
    c.db.Exec("DELETE FROM metadata_cache")
}
```

## Cache Strategy Selection

### Decision Tree

```
Is it single instance?
├─ YES → Use Memory Cache
│        - Fastest (~1-5ms)
│        - No external dependencies
│        - Set TTL to 1-24 hours
│
└─ NO → Multiple instances?
       ├─ YES → Use Redis Cache
       │        - Shared across instances
       │        - Set TTL to 15 mins - 24 hours
       │        - Monitor Redis health
       │
       └─ NO → Implement Custom Cache
              - Database-backed
              - Or other specialized storage
```

## Best Practices

### 1. Choose Appropriate TTL

```go
// Short TTL: Service has frequent changes
cache := memory.New(5 * time.Minute)

// Medium TTL: Standard setup
cache := memory.New(time.Hour)

// Long TTL: Stable metadata
cache := memory.New(24 * time.Hour)
```

### 2. Handle Cache Failures Gracefully

```go
// If cache fails, client falls back to direct service calls
cache, err := redis.New(&redis.Config{
    Addr: "localhost:6379",
})
if err != nil {
    // Fall back to no cache (NoOpCache)
    cache = &traverse.NoOpCache{}
}

client, _ := traverse.New(
    traverse.WithBaseURL(url),
    traverse.WithMetadataCache(cache),
)
```

### 3. Size Cache Appropriately

**Memory Cache:**
```go
// ~1KB per cached metadata
// With typical EDMX: ~50-100KB overhead
cache := memory.New(time.Hour)
```

**Redis Cache:**
```go
// Same as above but on Redis server
// Monitor Redis memory usage
// Set Redis max memory policy: allkeys-lru
```

## Troubleshooting

### Memory Cache Not Working

```go
// Verify cache is being used
cache := memory.New(time.Hour)

// Check that Metadata() is called (triggers caching)
m1, _ := client.Metadata()
m2, _ := client.Metadata() // Should be instant

// Check TTL is sufficient
cache := memory.New(24 * time.Hour) // Longer TTL

// Check cache size
println("Cache entries:", cache.Size())
```

### Redis Cache Not Working

```go
// Verify Redis connection
cfg := &redis.Config{
    Addr: "localhost:6379",
}

cache, err := redis.New(cfg)
if err != nil {
    log.Fatal("Redis connection failed:", err)
}
defer cache.Close()

// Verify keys are being stored
size := cache.Size()
println("Cached entries:", size)

// Check if entry exists
exists := cache.Exists("some_key")
println("Entry exists:", exists)
```

### Stale Data Issues

If you see stale data:

```go
// Reduce TTL
cache := memory.New(5 * time.Minute)

// Or clear cache manually
cache.Clear()
```

## License

MIT License - See LICENSE file in parent directory

