# Traverse OData Client - Comprehensive Benchmarks

## Overview

This directory contains exhaustive benchmarks for the traverse OData client library, covering all major aspects of functionality, resource consumption, and performance characteristics.

## Benchmark Categories

### 1. **Streaming Performance** (`streaming_bench_test.go`)
Evaluates streaming efficiency with various data volumes:
- **Data volumes**: 10K, 100K, 1M, 10M records
- **Metrics**: Throughput, latency, memory footprint, GC pressure
- **Comparison**: Collect() vs Stream() vs Batch operations
- **Focus**: Memory efficiency with large result sets

**Expected outcomes**:
- Stream: O(1) memory regardless of size
- Collect: O(n) memory for all results
- Batch: Optimal for pagination

### 2. **Cache Performance** (`cache_bench_test.go`)
Memory vs Redis cache under various scenarios:
- **Data sizes**: 1K, 10K, 100K, 1M items
- **Operations**: Get (hit/miss), Set, Clear, Delete
- **TTL variations**: 0s (no expiry), 1h, 24h
- **Concurrency**: 1, 10, 100 goroutines
- **Comparison**: Memory cache vs Redis vs no-cache

**Expected outcomes**:
- Memory cache: ~1-5ms per operation
- Redis cache: ~5-10ms per operation
- Cache hit rate analysis
- Memory usage comparison

### 3. **Query Complexity** (`query_bench_test.go`)
Impact of query complexity on performance:
- **Simple queries**: Single entity, no filters
- **Complex queries**: Multiple filters, nested expands, selects
- **Filter depth**: AND, OR combinations up to 5 levels
- **Expand depth**: Up to 3 levels of navigation properties
- **Metrics**: Query build time, network latency, result parsing

**Expected outcomes**:
- Simple query: <10ms
- Complex query: 10-50ms depending on nesting
- Count() optimization: 100x faster than full query

### 4. **Concurrency & Parallelism** (`concurrency_bench_test.go`)
Multi-threaded access patterns:
- **Goroutine counts**: 1, 10, 50, 100, 500
- **Scenarios**: 
  - Same query repeated
  - Different queries (no cache conflicts)
  - Mixed read/write operations
- **Contention measurement**: Lock contention, context switching
- **Metrics**: Throughput, latency variance, resource usage

**Expected outcomes**:
- Linear scaling up to CPU cores
- No deadlocks or race conditions
- Cache contention handled gracefully

### 5. **CRUD Operations** (`crud_bench_test.go`)
Create, Update, Delete performance:
- **Batch sizes**: 1, 10, 100, 1000
- **Response sizes**: Small (10 fields), Large (100 fields)
- **Operations**: Individual vs batch
- **Error scenarios**: Validation failures, network delays
- **Metrics**: Throughput, latency, error rate

**Expected outcomes**:
- Individual CRUD: 50-200ms per operation
- Batch CRUD: 5-20ms per item (amortized)
- Error handling: <5% performance impact

### 6. **Advanced Scenarios** (`advanced_bench_test.go`)
Real-world usage patterns:
- **Delta sync**: Change tracking performance
- **Service document parsing**: Large metadata
- **Function imports**: Complex parameter handling
- **Nested operations**: Expand with filters at multiple levels

## Mock OData Server

A built-in mock server simulates OData v4 responses for reproducible benchmarks.

**Features**:
- Configurable response sizes
- Latency simulation (realistic network delays)
- Supports all major OData operations
- Thread-safe concurrent requests
- Runs on localhost during tests

## Running Benchmarks

### Run all benchmarks:
```bash
go test -bench=. -benchmem -benchtime=10s ./benchmarks/...
```

### Run specific category:
```bash
go test -bench=BenchmarkStreaming -benchmem ./benchmarks/...
```

### With CPU profiling:
```bash
go test -bench=. -cpuprofile=cpu.prof -memprofile=mem.prof ./benchmarks/...
go tool pprof cpu.prof
```

### With detailed output:
```bash
go test -bench=. -benchmem -v ./benchmarks/... 2>&1 | tee benchmark_results.txt
```

## Benchmark Results Interpretation

### Throughput (ops/sec)
Higher is better. Expected ranges by operation:
- Simple filter: 1000+ ops/sec
- Complex filter: 100+ ops/sec
- Cache hit: 10000+ ops/sec
- Network-dependent ops: 10-100 ops/sec

### Memory (B/op)
Lower is better. Expected ranges:
- Cache get: <100 B/op
- Query build: 1-10 KB/op
- Result parsing: Variable with result size

### Allocations (allocs/op)
Lower is better. Target: <10 allocations per operation

### Latency variance
Measured by comparing min/max latency across runs.
Lower variance = more predictable performance

## Output Formats

Benchmarks generate multiple output files:

1. **BENCHMARKS.md** - Human-readable summary with analysis
2. **benchmark_results.json** - Machine-readable results for tracking
3. **benchmark_graphs.html** - Visual comparisons (optional)

## Continuous Benchmarking

For tracking performance regressions:

```bash
# Baseline
go test -bench=. -benchmem ./benchmarks/... > baseline.txt

# After changes
go test -bench=. -benchmem ./benchmarks/... > current.txt

# Compare
benchstat baseline.txt current.txt
```

## Performance Targets (v2.0)

These are aspirational targets for production readiness:

| Operation | Target | Current |
|-----------|--------|---------|
| Simple query | <10ms | TBD |
| Complex query | <50ms | TBD |
| Cache hit | <5ms | TBD |
| Stream 1M records | <5s | TBD |
| Concurrent 100 goroutines | Linear scaling | TBD |
| Memory per item | <1KB | TBD |

## Notes

- All benchmarks are I/O-bound (network latency simulated)
- Memory measurements exclude initial allocations
- CPU profiling requires: `go test -cpuprofile=...`
- Results vary by machine; focus on relative comparisons

---

**Generated**: 2026-04-01
**Library Version**: traverse v2.0
