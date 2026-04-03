# Prometheus Metrics Extension

The Prometheus extension provides metrics collection for OData operations, enabling performance monitoring and observability.

## Installation

```bash
go get github.com/jhonsferg/traverse/ext/prometheus
```

## Quick Start

```go
import (
    "github.com/jhonsferg/traverse"
    "github.com/jhonsferg/traverse/ext/prometheus"
)

// Create metrics collector
metrics := prometheus.New("odata_client")

// Track query
duration := time.Since(start).Milliseconds()
metrics.RecordQuery("Customers", duration, false)

// Get statistics
stats := metrics.GetStats()
log.Printf("Total queries: %d", stats["total_queries"])
log.Printf("Errors: %d", stats["total_errors"])
```

## Metrics Collection

### Query Metrics

Track SELECT operations:

```go
// Execute query
start := time.Now()
result, err := client.EntitySet("Customers").Collect(ctx)
duration := time.Since(start).Milliseconds()

// Record metrics
hasError := err != nil
metrics.RecordQuery("Customers", duration, hasError)
```

**Tracked metrics:**
- Total queries
- Successful queries
- Failed queries
- Latency distribution

### CRUD Metrics

#### Create Operations

```go
start := time.Now()
result, err := client.EntitySet("Orders").Create(ctx, data)
duration := time.Since(start).Milliseconds()

metrics.RecordCreate("Orders", duration, err != nil)
```

#### Update Operations

```go
start := time.Now()
err := client.EntitySet("Orders").Key("ID").Update(ctx, data)
duration := time.Since(start).Milliseconds()

metrics.RecordUpdate("Orders", duration, err != nil)
```

#### Delete Operations

```go
start := time.Now()
err := client.EntitySet("Orders").Key("ID").Delete(ctx)
duration := time.Since(start).Milliseconds()

metrics.RecordDelete("Orders", duration, err != nil)
```

### Cache Metrics

Track cache effectiveness:

```go
// Cache hit (return latency 0 or actual latency)
metrics.RecordQuery("metadata", 0, false)

// Cache miss (longer latency)
metrics.RecordQuery("metadata", 150, true)

// Calculate hit rate
hitRate := metrics.GetCacheHitRate()
log.Printf("Cache hit rate: %.2f%%", hitRate*100)
```

## Getting Statistics

### Overall Statistics

```go
stats := metrics.GetStats()

// Available stats:
// - total_queries: int
// - total_creates: int
// - total_updates: int
// - total_deletes: int
// - total_errors: int
// - successful_operations: int
// - average_query_latency: time.Duration
// - average_create_latency: time.Duration
// - average_update_latency: time.Duration
// - average_delete_latency: time.Duration

log.Printf("Queries: %d", stats["total_queries"])
log.Printf("Errors: %d", stats["total_errors"])
log.Printf("Avg latency: %v", stats["average_query_latency"])
```

### Cache Hit Rate

```go
hitRate := metrics.GetCacheHitRate()
log.Printf("Cache hit rate: %.2f%%", hitRate*100)

// Hit rate = cache hits / (cache hits + cache misses)
// Range: 0.0 (no hits) to 1.0 (all hits)
```

### Average Query Latency

```go
avgLatency := metrics.GetAverageQueryLatency()
log.Printf("Average query time: %v", avgLatency)

// Weighted average of all query durations
```

## Integration Patterns

### Per-Entity Metrics

Track metrics per entity set:

```go
type EntityMetrics struct {
    Set     string
    Queries int
    Creates int
    Errors  int
    Latency time.Duration
}

metricsMap := make(map[string]*EntityMetrics)

// Record operations
entities := []string{"Customers", "Orders", "Products"}

for _, entity := range entities {
    start := time.Now()
    _, err := client.EntitySet(entity).Collect(ctx)
    
    em := metricsMap[entity]
    if em == nil {
        em = &EntityMetrics{Set: entity}
        metricsMap[entity] = em
    }
    
    em.Queries++
    em.Latency += time.Since(start)
    if err != nil {
        em.Errors++
    }
}
```

### Application-Wide Metrics

Track all OData operations globally:

```go
type AppMetrics struct {
    totalRequests int
    totalErrors   int
    startTime     time.Time
}

metrics := prometheus.New("app_odata")
app := &AppMetrics{startTime: time.Now()}

// Middleware for all queries
func executeQuery(entity string, fn func() (interface{}, error)) (interface{}, error) {
    start := time.Now()
    result, err := fn()
    
    duration := time.Since(start).Milliseconds()
    metrics.RecordQuery(entity, duration, err != nil)
    
    app.totalRequests++
    if err != nil {
        app.totalErrors++
    }
    
    return result, err
}

// Usage
executeQuery("Customers", func() (interface{}, error) {
    return client.EntitySet("Customers").Collect(ctx)
})
```

### Latency Tracking

```go
type LatencyBucket struct {
    Fast    int  // < 100ms
    Normal  int  // 100-500ms
    Slow    int  // 500-1000ms
    VerySlow int // > 1000ms
}

func trackLatency(duration int64, buckets *LatencyBucket) {
    switch {
    case duration < 100:
        buckets.Fast++
    case duration < 500:
        buckets.Normal++
    case duration < 1000:
        buckets.Slow++
    default:
        buckets.VerySlow++
    }
}
```

## Exporters

### Prometheus Client Library Integration

To export to actual Prometheus:

```go
import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promhttp"
)

// Create Prometheus metrics
queryCounter := prometheus.NewCounterVec(
    prometheus.CounterOpts{
        Name: "odata_queries_total",
        Help: "Total OData queries",
    },
    []string{"entity_set", "status"},
)

// Record in metrics
stats := metrics.GetStats()
queryCounter.WithLabelValues("Customers", "success").
    Add(float64(stats["total_queries"].(int) - stats["total_errors"].(int)))

queryCounter.WithLabelValues("Customers", "error").
    Add(float64(stats["total_errors"].(int)))

// Export metrics
prometheus.MustRegister(queryCounter)
```

### JSON Export

```go
import "encoding/json"

stats := metrics.GetStats()
data, _ := json.MarshalIndent(stats, "", "  ")
log.Printf("Metrics: %s", data)
```

## Performance Considerations

### Overhead

- **Memory**: ~100 bytes per recorded operation
- **CPU**: <1ms per RecordXXX() call
- **Thread-safe**: Uses sync.RWMutex (minimal contention)

### Best Practices

1. **Don't Record Every Operation**
   ```go
   // Sample 10% of operations in production
   if rand.Float64() < 0.1 {
       metrics.RecordQuery(entity, duration, hasError)
   }
   ```

2. **Reset Periodically**
   ```go
   // Clear metrics every hour to prevent growth
   go func() {
       ticker := time.NewTicker(time.Hour)
       for range ticker.C {
           metrics = prometheus.New("odata_client")
       }
   }()
   ```

3. **Batch Recording**
   ```go
   // Record in batches instead of per-operation
   var totalDuration int64
   var errorCount int
   
   for i := 0; i < 100; i++ {
       // Do work
       totalDuration += duration
       if err != nil {
           errorCount++
       }
   }
   
   metrics.RecordQuery(entity, totalDuration/100, errorCount > 0)
   ```

## Example: Full Monitoring Setup

```go
package main

import (
    "context"
    "log"
    "time"
    
    "github.com/jhonsferg/traverse"
    "github.com/jhonsferg/traverse/ext/prometheus"
)

type MonitoredClient struct {
    client  *traverse.Client
    metrics *prometheus.Metrics
}

func (m *MonitoredClient) Query(entity string) ([]map[string]interface{}, error) {
    start := time.Now()
    
    result, err := m.client.
        EntitySet(entity).
        Collect(context.Background())
    
    duration := time.Since(start).Milliseconds()
    m.metrics.RecordQuery(entity, duration, err != nil)
    
    return result, err
}

func (m *MonitoredClient) Create(entity string, data map[string]interface{}) error {
    start := time.Now()
    
    err := m.client.
        EntitySet(entity).
        Create(context.Background(), data)
    
    duration := time.Since(start).Milliseconds()
    m.metrics.RecordCreate(entity, duration, err != nil)
    
    return err
}

func (m *MonitoredClient) PrintStats() {
    stats := m.metrics.GetStats()
    
    log.Println("=== OData Metrics ===")
    log.Printf("Total Queries: %d", stats["total_queries"])
    log.Printf("Total Creates: %d", stats["total_creates"])
    log.Printf("Total Errors: %d", stats["total_errors"])
    log.Printf("Avg Latency: %v", stats["average_query_latency"])
    log.Printf("Cache Hit Rate: %.2f%%", m.metrics.GetCacheHitRate()*100)
}

func main() {
    client := traverse.New("https://odata.example.com/v2")
    metrics := prometheus.New("app_odata")
    
    monitored := &MonitoredClient{
        client:  client,
        metrics: metrics,
    }
    
    // Execute operations
    monitored.Query("Customers")
    monitored.Create("Orders", map[string]interface{}{})
    monitored.Query("Products")
    
    // Print statistics
    monitored.PrintStats()
}
```

## Troubleshooting

### Zero Metrics

If all metrics are zero:

```go
// Ensure RecordXXX() is being called
log.Printf("Before: %v", metrics.GetStats())

metrics.RecordQuery("Test", 100, false)

log.Printf("After: %v", metrics.GetStats())
```

### High Error Rates

```go
stats := metrics.GetStats()
errorRate := float64(stats["total_errors"].(int)) / 
             float64(stats["total_queries"].(int))

log.Printf("Error rate: %.2f%%", errorRate*100)

if errorRate > 0.05 { // More than 5%
    log.Println("WARNING: High error rate detected")
}
```

### Memory Usage

```go
// Metrics use minimal memory
// Typical overhead: 1KB per entity set tracked

// If memory is critical, limit entities tracked
trackedEntities := []string{"Customers", "Orders"} // Only track important ones
```

## License

MIT License - See LICENSE file in parent directory
