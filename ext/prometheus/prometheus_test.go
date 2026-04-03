// Package prometheus provides tests for metrics tracking.
package prometheus

import (
	"errors"
	"testing"
	"time"
)

// TestMetrics_Creation verifies Metrics can be created.
func TestMetrics_Creation(t *testing.T) {
	metrics := New()
	if metrics == nil {
		t.Fatalf("Expected non-nil Metrics instance")
	}
	if metrics.GetQueryCount() != 0 {
		t.Fatalf("Expected initial query count to be 0")
	}

	t.Logf("✅ Metrics creation test passed")
}

// TestMetrics_RecordQuery tracks query metrics.
func TestMetrics_RecordQuery(t *testing.T) {
	metrics := New()
	latency := 100 * time.Millisecond

	// Record successful query
	metrics.RecordQuery(latency, nil)

	if metrics.GetQueryCount() != 1 {
		t.Fatalf("Expected query count to be 1, got %d", metrics.GetQueryCount())
	}
	if metrics.GetErrorCount() != 0 {
		t.Fatalf("Expected error count to be 0")
	}

	t.Logf("✅ Query recording test passed")
}

// TestMetrics_RecordQueryWithError tracks query errors.
func TestMetrics_RecordQueryWithError(t *testing.T) {
	metrics := New()
	latency := 50 * time.Millisecond

	// Record failed query
	metrics.RecordQuery(latency, errors.New("query failed"))

	if metrics.GetQueryCount() != 1 {
		t.Fatalf("Expected query count to be 1")
	}
	if metrics.GetErrorCount() != 1 {
		t.Fatalf("Expected error count to be 1, got %d", metrics.GetErrorCount())
	}

	t.Logf("✅ Query error recording test passed")
}

// TestMetrics_RecordCreate tracks create operations.
func TestMetrics_RecordCreate(t *testing.T) {
	metrics := New()
	latency := 150 * time.Millisecond

	metrics.RecordCreate(latency, nil)

	stats := metrics.GetStats()
	ops := stats["operations"].(map[string]int64)
	if ops["create"] != 1 {
		t.Fatalf("Expected create count to be 1")
	}

	t.Logf("✅ Create recording test passed")
}

// TestMetrics_RecordUpdate tracks update operations.
func TestMetrics_RecordUpdate(t *testing.T) {
	metrics := New()
	latency := 120 * time.Millisecond

	metrics.RecordUpdate(latency, nil)

	stats := metrics.GetStats()
	ops := stats["operations"].(map[string]int64)
	if ops["update"] != 1 {
		t.Fatalf("Expected update count to be 1")
	}

	t.Logf("✅ Update recording test passed")
}

// TestMetrics_RecordDelete tracks delete operations.
func TestMetrics_RecordDelete(t *testing.T) {
	metrics := New()
	latency := 80 * time.Millisecond

	metrics.RecordDelete(latency, nil)

	stats := metrics.GetStats()
	ops := stats["operations"].(map[string]int64)
	if ops["delete"] != 1 {
		t.Fatalf("Expected delete count to be 1")
	}

	t.Logf("✅ Delete recording test passed")
}

// TestMetrics_CacheHitRate calculates cache hit rate.
func TestMetrics_CacheHitRate(t *testing.T) {
	metrics := New()

	// Record 7 hits, 3 misses = 70% hit rate
	for i := 0; i < 7; i++ {
		metrics.RecordCacheHit()
	}
	for i := 0; i < 3; i++ {
		metrics.RecordCacheMiss()
	}

	hitRate := metrics.GetCacheHitRate()
	expected := 0.7

	if hitRate != expected {
		t.Fatalf("Expected hit rate %.1f, got %.1f", expected, hitRate)
	}

	t.Logf("✅ Cache hit rate test passed (%.0f%%)", hitRate*100)
}

// TestMetrics_AverageLatency calculates average latency.
func TestMetrics_AverageLatency(t *testing.T) {
	metrics := New()

	// Record queries with different latencies
	metrics.RecordQuery(100*time.Millisecond, nil)
	metrics.RecordQuery(200*time.Millisecond, nil)
	metrics.RecordQuery(150*time.Millisecond, nil)

	avgLatency := metrics.GetAverageQueryLatency()
	expected := 150 * time.Millisecond

	if avgLatency != expected {
		t.Fatalf("Expected avg latency %v, got %v", expected, avgLatency)
	}

	t.Logf("✅ Average latency test passed (%v)", avgLatency)
}

// TestMetrics_GetStats returns structured stats.
func TestMetrics_GetStats(t *testing.T) {
	metrics := New()

	metrics.RecordQuery(100*time.Millisecond, nil)
	metrics.RecordCreate(150*time.Millisecond, nil)
	metrics.RecordCacheHit()

	stats := metrics.GetStats()

	// Verify structure
	if _, ok := stats["queries"]; !ok {
		t.Fatalf("Expected queries key in stats")
	}
	if _, ok := stats["operations"]; !ok {
		t.Fatalf("Expected operations key in stats")
	}
	if _, ok := stats["cache"]; !ok {
		t.Fatalf("Expected cache key in stats")
	}

	queries := stats["queries"].(map[string]int64)
	if queries["total"] != 1 {
		t.Fatalf("Expected 1 total query in stats")
	}

	t.Logf("✅ Get stats test passed")
}

// TestMetrics_Reset clears all metrics.
func TestMetrics_Reset(t *testing.T) {
	metrics := New()

	// Record some metrics
	metrics.RecordQuery(100*time.Millisecond, nil)
	metrics.RecordCreate(150*time.Millisecond, nil)
	metrics.RecordCacheHit()

	if metrics.GetQueryCount() == 0 {
		t.Fatalf("Expected metrics to be recorded before reset")
	}

	// Reset
	metrics.Reset()

	// Verify everything is cleared
	if metrics.GetQueryCount() != 0 {
		t.Fatalf("Expected query count to be 0 after reset")
	}
	if metrics.GetErrorCount() != 0 {
		t.Fatalf("Expected error count to be 0 after reset")
	}
	hitRate := metrics.GetCacheHitRate()
	if hitRate != 0 {
		t.Fatalf("Expected hit rate to be 0 after reset")
	}

	t.Logf("✅ Reset test passed")
}

// TestMetrics_Concurrency verifies thread-safe operation.
func TestMetrics_Concurrency(t *testing.T) {
	metrics := New()
	done := make(chan bool, 10)

	// Concurrent metric recording
	for i := 0; i < 10; i++ {
		go func() {
			metrics.RecordQuery(100*time.Millisecond, nil)
			metrics.RecordCacheHit()
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	if metrics.GetQueryCount() != 10 {
		t.Fatalf("Expected 10 queries, got %d", metrics.GetQueryCount())
	}

	t.Logf("✅ Concurrency test passed (10 concurrent operations)")
}

// TestMetrics_ErrorTracking tracks different error types.
func TestMetrics_ErrorTracking(t *testing.T) {
	metrics := New()

	// Record mixed success and failure
	metrics.RecordQuery(100*time.Millisecond, nil)      // success
	metrics.RecordQuery(50*time.Millisecond, errors.New("error1"))  // error
	metrics.RecordQuery(200*time.Millisecond, nil)      // success
	metrics.RecordCreate(150*time.Millisecond, errors.New("error2")) // error

	if metrics.GetQueryCount() != 3 {
		t.Fatalf("Expected 3 queries")
	}
	if metrics.GetErrorCount() != 2 {
		t.Fatalf("Expected 2 total errors, got %d", metrics.GetErrorCount())
	}

	t.Logf("✅ Error tracking test passed")
}

// TestMetrics_CacheMetrics tracks cache operations.
func TestMetrics_CacheMetrics(t *testing.T) {
	metrics := New()

	// Record cache operations
	for i := 0; i < 15; i++ {
		metrics.RecordCacheHit()
	}
	for i := 0; i < 5; i++ {
		metrics.RecordCacheMiss()
	}

	stats := metrics.GetStats()
	cache := stats["cache"].(map[string]interface{})
	
	if cache["hits"].(int64) != 15 {
		t.Fatalf("Expected 15 cache hits")
	}
	if cache["misses"].(int64) != 5 {
		t.Fatalf("Expected 5 cache misses")
	}

	hitRate := metrics.GetCacheHitRate()
	if hitRate != 0.75 {
		t.Fatalf("Expected 75%% hit rate, got %.0f%%", hitRate*100)
	}

	t.Logf("✅ Cache metrics test passed (75%% hit rate)")
}

// TestMetrics_LatencyTracking verifies latency storage.
func TestMetrics_LatencyTracking(t *testing.T) {
	metrics := New()

	latencies := []time.Duration{
		50 * time.Millisecond,
		100 * time.Millisecond,
		150 * time.Millisecond,
	}

	for _, latency := range latencies {
		metrics.RecordQuery(latency, nil)
	}

	avgLatency := metrics.GetAverageQueryLatency()
	expectedAvg := 100 * time.Millisecond

	if avgLatency != expectedAvg {
		t.Fatalf("Expected avg latency %v, got %v", expectedAvg, avgLatency)
	}

	t.Logf("✅ Latency tracking test passed (%v avg)", avgLatency)
}
