# Concurrency Scaling Benchmarks

Concurrent operation analysis including:
- Goroutine scaling (1-64 concurrent)
- Cache contention patterns
- Concurrent vs sequential performance
- Context cancellation overhead
- Rate limiting impact
- Mixed operation workloads

## Files

- `concurrency_bench_test.go` - Concurrency patterns
- `contention_bench_test.go` - Lock contention analysis
- `scaling_bench_test.go` - Horizontal scaling
