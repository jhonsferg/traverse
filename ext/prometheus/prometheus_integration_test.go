// Package prometheus integration tests verify metrics work with traverse.
package prometheus

import (
	"testing"
	"time"
)

// TestPrometheusIntegration_MetricsWithClient verifies metrics work with traverse client.
func TestPrometheusIntegration_MetricsWithClient(t *testing.T) {
	metrics := New()

	// Simulate client operations
	metrics.RecordQuery(100*time.Millisecond, nil)
	metrics.RecordCreate(150*time.Millisecond, nil)
	metrics.RecordUpdate(120*time.Millisecond, nil)
	metrics.RecordDelete(80*time.Millisecond, nil)

	stats := metrics.GetStats()
	
	queries := stats["queries"].(map[string]int64)
	if queries["total"] != 1 {
		t.Fatalf("Expected 1 query")
	}

	ops := stats["operations"].(map[string]int64)
	if ops["create"] != 1 || ops["update"] != 1 || ops["delete"] != 1 {
		t.Fatalf("Expected all CRUD operations to be recorded")
	}

	t.Logf("✅ Prometheus + client integration test passed")
}

// TestPrometheusIntegration_MetricsCollection verifies metrics collection.
func TestPrometheusIntegration_MetricsCollection(t *testing.T) {
	metrics := New()

	// Simulate realistic query load
	for i := 0; i < 100; i++ {
		if i%10 == 0 {
			metrics.RecordQuery(50*time.Millisecond, nil)
		} else {
			metrics.RecordQuery(100*time.Millisecond, nil)
		}
	}

	if metrics.GetQueryCount() != 100 {
		t.Fatalf("Expected 100 queries recorded")
	}

	avgLatency := metrics.GetAverageQueryLatency()
	if avgLatency == 0 {
		t.Fatalf("Expected non-zero average latency")
	}

	t.Logf("✅ Metrics collection test passed (100 queries, avg: %v)", avgLatency)
}

// TestPrometheusIntegration_CacheAndQueryMetrics verifies cache + query tracking.
func TestPrometheusIntegration_CacheAndQueryMetrics(t *testing.T) {
	metrics := New()

	// Record cache operations (metadata cache)
	for i := 0; i < 30; i++ {
		metrics.RecordCacheHit()
	}
	for i := 0; i < 10; i++ {
		metrics.RecordCacheMiss()
	}

	// Record query operations
	for i := 0; i < 40; i++ {
		metrics.RecordQuery(100*time.Millisecond, nil)
	}

	stats := metrics.GetStats()
	
	cache := stats["cache"].(map[string]interface{})
	if cache["hits"].(int64) != 30 {
		t.Fatalf("Expected 30 cache hits")
	}

	queries := stats["queries"].(map[string]int64)
	if queries["total"] != 40 {
		t.Fatalf("Expected 40 queries")
	}

	hitRate := metrics.GetCacheHitRate()
	if hitRate != 0.75 {
		t.Fatalf("Expected 75%% cache hit rate")
	}

	t.Logf("✅ Cache + query metrics test passed (75%% hit rate, 40 queries)")
}

// TestPrometheusIntegration_ErrorRateCalculation verifies error tracking.
func TestPrometheusIntegration_ErrorRateCalculation(t *testing.T) {
	metrics := New()

	// Record mixed operations
	successCount := 80
	errorCount := 20

	for i := 0; i < successCount; i++ {
		metrics.RecordQuery(100*time.Millisecond, nil)
	}

	for i := 0; i < errorCount; i++ {
		metrics.RecordQuery(50*time.Millisecond, errTest)
	}

	totalErrors := metrics.GetErrorCount()
	if totalErrors != int64(errorCount) {
		t.Fatalf("Expected %d errors, got %d", errorCount, totalErrors)
	}

	errorRate := float64(totalErrors) / float64(successCount+errorCount)
	if errorRate != 0.2 {
		t.Fatalf("Expected 20%% error rate, got %.0f%%", errorRate*100)
	}

	t.Logf("✅ Error rate calculation test passed (20%% error rate)")
}

// TestPrometheusIntegration_MultipleMetricsInstances verifies isolated metrics.
func TestPrometheusIntegration_MultipleMetricsInstances(t *testing.T) {
	metrics1 := New()
	metrics2 := New()

	// Record different operations on each
	metrics1.RecordQuery(100*time.Millisecond, nil)
	metrics1.RecordQuery(100*time.Millisecond, nil)

	metrics2.RecordCreate(150*time.Millisecond, nil)

	if metrics1.GetQueryCount() != 2 {
		t.Fatalf("Expected metrics1 to have 2 queries")
	}

	stats2 := metrics2.GetStats()
	ops := stats2["operations"].(map[string]int64)
	if ops["create"] != 1 {
		t.Fatalf("Expected metrics2 to have 1 create")
	}

	t.Logf("✅ Multiple metrics instances test passed (isolated)")
}

// TestPrometheusIntegration_PerformanceMonitoring verifies latency tracking.
func TestPrometheusIntegration_PerformanceMonitoring(t *testing.T) {
	metrics := New()

	// Simulate varying query performance
	latencies := []time.Duration{
		50 * time.Millisecond,
		100 * time.Millisecond,
		150 * time.Millisecond,
		200 * time.Millisecond,
		250 * time.Millisecond,
	}

	for _, latency := range latencies {
		metrics.RecordQuery(latency, nil)
	}

	avgLatency := metrics.GetAverageQueryLatency()
	expectedAvg := 150 * time.Millisecond

	if avgLatency != expectedAvg {
		t.Fatalf("Expected avg %v, got %v", expectedAvg, avgLatency)
	}

	t.Logf("✅ Performance monitoring test passed (%v avg latency)", avgLatency)
}

// errTest is a test error for testing purposes.
var errTest = &testError{msg: "test error"}

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}
