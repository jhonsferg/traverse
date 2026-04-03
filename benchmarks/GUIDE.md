# Traverse Benchmarks Suite

This directory contains comprehensive benchmarks for the traverse OData client library.

## Quick Start

### Run All Benchmarks
```bash
go test -bench=. -benchmem ./benchmarks/...
```

### Run Specific Categories
```bash
# Cache benchmarks
go test -bench=BenchmarkCache -benchmem ./benchmarks/...

# Streaming benchmarks
go test -bench=BenchmarkStream -benchmem ./benchmarks/...

# Concurrency benchmarks
go test -bench=BenchmarkConcurrent -benchmem ./benchmarks/...
```

### Save Results
```bash
go test -bench=. -benchmem ./benchmarks/... | tee results.txt
```

## Benchmark Categories

### 1. Cache Performance (cache_bench_test.go)
Tests memory cache efficiency across various scenarios:
- Get operations (hit/miss)
- Set operations
- Concurrent read/write/mixed
- TTL expiration
- Cache clearing
- Scaling with different cache sizes

**Key Results**:
- Cache get hit: **32.93 ns/op**
- Cache set: **110.1 ns/op**
- Concurrent operations: linear scaling

### 2. Streaming (streaming_bench_test.go)
Evaluates streaming efficiency with different data volumes:
- Small datasets (10K records): 426 ms
- Medium datasets (100K records): 3,579 ms
- Stream vs Collect comparison
- First record retrieval
- Page operations

**Key Results**:
- Stream is memory-efficient: O(1) regardless of size
- Collect for <10K records is practical
- Stream for >100K records recommended

### 3. Query Complexity (query_bench_test.go)
Impact of query complexity on performance:
- Simple queries
- Filtered queries (single and complex)
- OrderBy (single and multiple fields)
- Select (field projection)
- Fully qualified queries
- Count optimization
- Response parsing with different payload sizes

**Key Results**:
- Query complexity doesn't affect latency (~71-73ms)
- Count is 364x faster than full query
- Pagination overhead is constant

### 4. Concurrency (concurrency_bench_test.go)
Multi-threaded access patterns:
- Same query repeated
- Different queries (no cache conflict)
- Stream and Collect mix
- Metadata access concurrency
- Mixed operations (metadata/count/query/stream)
- Goroutine scaling (1-16)
- Context cancellation
- Rate limiting

**Key Results**:
- Linear scaling up to 8 goroutines
- ~10-15% latency increase per 2x goroutines
- No cache contention between different queries

## Mock OData Server

The `mock_server.go` provides a configurable mock OData v4 server:

```go
server := NewMockODataServer(ServerConfig{
    Latency:         5 * time.Millisecond,  // Network delay
    DefaultPageSize: 100,                    // Default $top
    MaxRecords:      10000,                  // Dataset size
    RecordSize:      RecordSizeMedium,      // Small/Medium/Large payloads
})
defer server.Close()
```

Features:
- Configurable network latency
- Adjustable response sizes
- Support for $filter, $orderby, $select, $skip, $top
- Error rate simulation
- Request counting

## Interpreting Results

### Throughput (ops/sec)
- Cache operations: **millions per second** (expected to be high)
- Query operations: **tens per second** (network-bound)
- Count operations: **hundreds per second** (smaller payload)

### Memory (B/op)
- Cache hit: **0 B** (reuse cached data)
- Query collect: **2-12MB** (depends on result size)
- Stream operations: **0 B** (doesn't buffer)

### Allocations (allocs/op)
- Cache operations: **0-2 allocs** (very efficient)
- Query operations: **64K-646K allocs** (result parsing)
- Concurrent ops: **increased allocations** (goroutine overhead)

## Performance Targets (v2.0)

| Operation | Target | Status |
|-----------|--------|--------|
| Cache get | <100 ns | ✅ 33 ns |
| Cache set | <200 ns | ✅ 110 ns |
| Count query | <10ms | ✅ 1.95ms |
| Simple query | <100ms | ✅ 71ms |
| Stream 10K | <1s | ✅ 426ms |
| Concurrent (8 goroutines) | Linear | ✅ 10% overhead/2x |

## Running with Profiling

### CPU Profiling
```bash
go test -bench=BenchmarkStreamingMedium -cpuprofile=cpu.prof ./benchmarks/...
go tool pprof cpu.prof
```

### Memory Profiling
```bash
go test -bench=BenchmarkCollectMedium -memprofile=mem.prof ./benchmarks/...
go tool pprof mem.prof
```

### Benchstat Comparison
```bash
# Baseline
go test -bench=. -benchmem ./benchmarks/... > baseline.txt

# After optimization
go test -bench=. -benchmem ./benchmarks/... > current.txt

# Compare
benchstat baseline.txt current.txt
```

## Benchmark Details

For detailed analysis and interpretation of results, see **BENCHMARKS.md**

---

**Generated**: April 1, 2026  
**System**: AMD Ryzen 9 5950X 16-Core Processor  
**Library Version**: traverse v2.0
