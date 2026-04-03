# Memory Profile Analysis

Comprehensive memory profiling and analysis of the Traverse OData client library.

## Overview

This category focuses on memory allocation patterns, heap behavior, and memory efficiency across different query scenarios.

## Benchmarks Included

### 1. **BenchmarkMemoryAllocation**
Tests memory allocation patterns with different query sizes and page configurations:
- Small single-page queries
- Medium multi-page queries  
- Large multi-page queries
- Extra-large datasets
- Large page sizes

**Focus:** Understanding memory usage for different result set sizes

### 2. **BenchmarkMemoryGrowth**
Analyzes how memory scales with increasing data sizes and payload types:
- Small/Medium/Large record combinations
- 1K, 10K, 50K record counts
- Different payload sizes

**Focus:** Identifying memory scaling characteristics

### 3. **BenchmarkMemoryPressure**
Tests behavior under memory-intensive scenarios:
- Multiple concurrent queries
- Large filter expressions
- Selecting many fields

**Focus:** Real-world high-load memory patterns

### 4. **BenchmarkHeapFragmentation**
Tests heap fragmentation patterns:
- Sequential queries (many small allocations)
- Batch queries (few large allocations)
- Mixed page sizes

**Focus:** GC efficiency and fragmentation impact

### 5. **BenchmarkMemoryLeakDetection**
Tests for potential memory leaks in edge cases:
- Client reuse patterns
- Client creation/destruction cycles

**Focus:** Long-running stability

## Key Metrics

- **Allocations per operation** (allocs/op)
- **Total bytes allocated** (B/op)
- **Memory efficiency** across different scenarios
- **GC impact** on performance

## Running the Tests

```bash
# Run all memory profile benchmarks
go test -bench=. -benchmem ./benchmarks/01_memory_profile/

# Run specific benchmark
go test -bench=BenchmarkMemoryAllocation -benchmem ./benchmarks/01_memory_profile/

# With memory profiling
go test -bench=. -benchmem -memprofile=mem.prof ./benchmarks/01_memory_profile/
go tool pprof -http=:8080 mem.prof
```

## Interpretation Guide

| Metric | Interpretation |
|--------|-----------------|
| Allocations/op | Lower is better; indicates fewer GC cycles |
| Bytes/op | Lower is better; indicates efficient memory usage |
| Consistent allocs | Indicates predictable memory behavior |
| Growing allocs | May indicate inefficient collection strategies |

## Expected Baselines

- Small queries: ~11K-12K allocs/op, ~540KB/op
- Medium queries: ~60K-70K allocs/op, ~2-3MB/op
- Large queries: ~600K-700K allocs/op, ~20-30MB/op

## Common Issues

1. **High allocations**: May indicate inefficient buffering or unnecessary copying
2. **Variable allocations**: Suggests different code paths or GC interference
3. **Growing trends**: May indicate memory leaks or inefficient collection strategies
