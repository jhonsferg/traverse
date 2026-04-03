# Throughput Testing

Comprehensive throughput and latency analysis of the Traverse OData client library under various load conditions.

## Overview

This category focuses on sustained throughput measurement, peak load analysis, and performance under different configurations and network conditions.

## Benchmarks Included

### 1. **BenchmarkThroughputBaseline**
Establishes baseline throughput with various latency and payload configurations:
- Low latency + small payload
- Medium latency + medium payload
- High latency + large payload
- No latency + medium payload

**Focus:** Baseline performance under different conditions

### 2. **BenchmarkThroughputByPageSize**
Measures throughput impact of different pagination sizes:
- Page sizes: 50, 100, 500, 1K, 5K, 10K records

**Focus:** Optimal page size for throughput

### 3. **BenchmarkThroughputSustained**
Measures sustained throughput over different time periods:
- 1 second sustained load
- 5 second sustained load
- 10 second sustained load

**Focus:** Consistency and stability over time

### 4. **BenchmarkThroughputPeak**
Identifies peak throughput achievable with concurrent requests:
- Concurrency levels: 1, 2, 4, 8, 16, 32, 64

**Focus:** Optimal parallelism and scaling

### 5. **BenchmarkThroughputDistribution**
Measures throughput across different entity types:
- Products entity
- Orders entity
- Customers entity

**Focus:** Entity-specific performance differences

### 6. **BenchmarkThroughputPayloadSize**
Tests impact of response payload size on throughput:
- Small payloads (~100 bytes/record)
- Medium payloads (~500 bytes/record)
- Large payloads (~2KB/record)

**Focus:** Bandwidth and deserialization impact

### 7. **BenchmarkThroughputWithFilters**
Measures throughput with different filter complexities:
- No filter
- Simple filter (Price gt 100)
- Complex filter (multiple conditions with AND)
- Multiple OR conditions

**Focus:** Filter complexity impact

### 8. **BenchmarkThroughputSaturation**
Measures throughput as the system approaches saturation:
- Concurrent loads: 1, 5, 10, 20, 50, 100

**Focus:** System saturation point

### 9. **BenchmarkThroughputVariability**
Measures variance in throughput across scenarios:
- Stable low-latency operations
- Variable high-latency operations
- Balanced medium-latency operations

**Focus:** Performance consistency

## Key Metrics

- **Operations per second** (ops/s)
- **Latency per operation** (ns/op, ms/op)
- **Sustained throughput** over time
- **Peak achievable throughput**
- **Throughput variance**

## Running the Tests

```bash
# Run all throughput benchmarks
go test -bench=. -benchmem ./benchmarks/02_throughput/

# Run specific benchmark
go test -bench=BenchmarkThroughputBaseline -benchmem ./benchmarks/02_throughput/

# Run with specific duration
go test -bench=. -benchtime=10s ./benchmarks/02_throughput/

# Profile CPU during benchmark
go test -bench=. -cpuprofile=cpu.prof ./benchmarks/02_throughput/
go tool pprof -http=:8080 cpu.prof
```

## Interpretation Guide

| Metric | Interpretation |
|--------|-----------------|
| ops/s | Higher is better; operations completed per second |
| ns/op | Lower is better; time per operation |
| Consistent latency | Good; indicates predictable performance |
| Variable latency | May indicate GC pauses or lock contention |
| Degradation with load | Expected; identifies saturation point |

## Expected Baselines

**Single concurrent client:**
- Small payload: 150-250 ops/s
- Medium payload: 100-150 ops/s
- Large payload: 50-100 ops/s

**Peak throughput (8 concurrent):**
- Small payload: 800-1200 ops/s
- Medium payload: 600-900 ops/s
- Large payload: 300-500 ops/s

## Performance Tuning

1. **Increase concurrency** for better throughput (up to ~8x)
2. **Optimize page size** based on workload (typically 100-500)
3. **Use appropriate timeout** for long-running queries
4. **Monitor GC pressure** with high allocation workloads

## Common Patterns

- Network latency dominates total latency (typically 95%+)
- Throughput scales nearly linearly with concurrency up to 8 goroutines
- Larger payloads reduce throughput proportionally to size
- Filter complexity has minimal impact on throughput
