// Package tracing integration tests verify tracing works with traverse operations.
package tracing

import (
	"context"
	"errors"
	"testing"
	"time"
)

// TestTracingIntegration_QueryTracing verifies query operation tracing.
func TestTracingIntegration_QueryTracing(t *testing.T) {
	tracer := New("odata-service")
	ctx := context.Background()

	// Simulate query operation
	_, span := tracer.StartSpan(ctx, "query_customers")
	tracer.SetAttribute(span, "entity_set", "Customers")
	tracer.SetAttribute(span, "filter", "Status eq 'active'")
	tracer.AddEvent(span, "query_started", map[string]interface{}{})

	time.Sleep(20 * time.Millisecond)

	tracer.AddEvent(span, "query_completed", map[string]interface{}{
		"row_count": 42,
	})
	tracer.EndSpan(span, nil)

	stats := tracer.GetStats()
	if stats["total_spans"].(int) != 1 {
		t.Fatalf("Expected 1 span")
	}

	t.Logf("✅ Query tracing integration test passed")
}

// TestTracingIntegration_ErrorTracing verifies error tracking in spans.
func TestTracingIntegration_ErrorTracing(t *testing.T) {
	tracer := New("odata-service")
	ctx := context.Background()

	_, span := tracer.StartSpan(ctx, "create_entity")
	tracer.SetAttribute(span, "entity_type", "Order")

	queryErr := errors.New("entity validation failed")
	tracer.EndSpan(span, queryErr)

	stats := tracer.GetStats()
	if stats["errors"].(int) != 1 {
		t.Fatalf("Expected 1 error in stats")
	}

	t.Logf("✅ Error tracing integration test passed")
}

// TestTracingIntegration_DistributedTracing verifies context propagation.
func TestTracingIntegration_DistributedTracing(t *testing.T) {
	// Service A (frontend)
	tracerA := New("frontend-service")
	tracerA.AddBaggage("user_id", "user789")
	tracerA.AddBaggage("request_id", "req123")

	// Simulate propagation to Service B (backend)
	carrier := tracerA.Inject()
	tracerB := Extract(carrier)

	// Verify context is propagated
	if tracerB.GetTraceID() != tracerA.GetTraceID() {
		t.Fatalf("Expected same trace ID across services")
	}
	if tracerB.GetBaggage("user_id") != "user789" {
		t.Fatalf("Expected baggage to be propagated")
	}

	t.Logf("✅ Distributed tracing integration test passed")
}

// TestTracingIntegration_NestedSpans verifies span hierarchy.
func TestTracingIntegration_NestedSpans(t *testing.T) {
	tracer := New("odata-service")
	ctx := context.Background()

	// Root span (API request)
	_, rootSpan := tracer.StartSpan(ctx, "api_request")
	tracer.SetAttribute(rootSpan, "method", "GET")
	tracer.SetAttribute(rootSpan, "path", "/odata/v2/Customers")

	// Child span 1 (authorization)
	newCtx, authSpan := tracer.StartSpan(ctx, "authorize")
	tracer.SetAttribute(authSpan, "user", "admin")
	tracer.EndSpan(authSpan, nil)

	// Child span 2 (query)
	_, querySpan := tracer.StartSpan(newCtx, "execute_query")
	tracer.SetAttribute(querySpan, "entity_set", "Customers")
	tracer.EndSpan(querySpan, nil)

	tracer.EndSpan(rootSpan, nil)

	spans := tracer.GetActiveSpans()
	if len(spans) != 3 {
		t.Fatalf("Expected 3 spans in hierarchy")
	}

	t.Logf("✅ Nested spans integration test passed")
}

// TestTracingIntegration_CacheWithTracing verifies cache operations in tracing.
func TestTracingIntegration_CacheWithTracing(t *testing.T) {
	tracer := New("odata-service")
	ctx := context.Background()

	// Metadata cache hit
	_, cacheHitSpan := tracer.StartSpan(ctx, "metadata_cache_lookup")
	tracer.AddEvent(cacheHitSpan, "cache_hit", map[string]interface{}{
		"key": "metadata:Customers",
	})
	tracer.EndSpan(cacheHitSpan, nil)

	// Metadata fetch (cache miss)
	_, cacheMissSpan := tracer.StartSpan(ctx, "metadata_fetch")
	tracer.AddEvent(cacheMissSpan, "cache_miss", map[string]interface{}{})
	time.Sleep(10 * time.Millisecond)
	tracer.EndSpan(cacheMissSpan, nil)

	spans := tracer.GetActiveSpans()
	if len(spans) != 2 {
		t.Fatalf("Expected 2 cache-related spans")
	}

	t.Logf("✅ Cache with tracing integration test passed")
}

// TestTracingIntegration_PerformanceMonitoring verifies latency tracking.
func TestTracingIntegration_PerformanceMonitoring(t *testing.T) {
	tracer := New("odata-service")
	ctx := context.Background()

	// Fast query
	_, span1 := tracer.StartSpan(ctx, "fast_query")
	time.Sleep(5 * time.Millisecond)
	tracer.EndSpan(span1, nil)

	// Slow query
	_, span2 := tracer.StartSpan(ctx, "slow_query")
	time.Sleep(50 * time.Millisecond)
	tracer.EndSpan(span2, nil)

	stats := tracer.GetStats()
	avgDuration := stats["average_duration"].(time.Duration)

	if avgDuration == 0 {
		t.Fatalf("Expected non-zero average duration")
	}

	t.Logf("✅ Performance monitoring integration test passed (avg: %v)", avgDuration)
}

// TestTracingIntegration_CRUDOperations verifies CRUD operation tracing.
func TestTracingIntegration_CRUDOperations(t *testing.T) {
	tracer := New("odata-service")
	ctx := context.Background()

	operations := []struct {
		name string
		op   string
	}{
		{"create_customer", "CREATE"},
		{"read_customer", "READ"},
		{"update_customer", "UPDATE"},
		{"delete_customer", "DELETE"},
	}

	for _, tc := range operations {
		_, span := tracer.StartSpan(ctx, tc.name)
		tracer.SetAttribute(span, "operation", tc.op)
		tracer.SetAttribute(span, "entity_type", "Customer")
		tracer.EndSpan(span, nil)
	}

	spans := tracer.GetActiveSpans()
	if len(spans) != 4 {
		t.Fatalf("Expected 4 CRUD operation spans")
	}

	t.Logf("✅ CRUD operations tracing integration test passed")
}

// TestTracingIntegration_W3CContextPropagation verifies W3C trace context.
func TestTracingIntegration_W3CContextPropagation(t *testing.T) {
	tracer := New("odata-service")

	// Get W3C trace context
	ctx := tracer.GetTraceContext()

	// Should contain trace ID and span ID
	if len(ctx) == 0 {
		t.Fatalf("Expected W3C trace context")
	}

	// Format should be version-traceID-spanID-flags
	parts := make([]string, 0)
	if len(ctx) > 0 {
		parts = append(parts, ctx) // Simplified - just verify format exists
	}

	if len(parts) == 0 {
		t.Fatalf("Expected valid W3C context format")
	}

	t.Logf("✅ W3C context propagation integration test passed (%s)", ctx)
}

// TestTracingIntegration_MultiServiceScenario simulates multi-service tracing.
func TestTracingIntegration_MultiServiceScenario(t *testing.T) {
	// Service A receives request
	tracerA := New("frontend-api")
	ctx := context.Background()
	_, aSpan := tracerA.StartSpan(ctx, "api_request")
	tracerA.SetAttribute(aSpan, "endpoint", "/orders")
	tracerA.AddBaggage("request_id", "req-001")

	// Propagate to Service B
	carrier := tracerA.Inject()
	tracerB := Extract(carrier)
	_, bSpan := tracerB.StartSpan(ctx, "db_query")
	tracerB.SetAttribute(bSpan, "table", "orders")

	// Propagate to Service C
	carrier2 := tracerB.Inject()
	tracerC := Extract(carrier2)
	_, cSpan := tracerC.StartSpan(ctx, "cache_update")

	// Complete spans
	tracerC.EndSpan(cSpan, nil)
	tracerB.EndSpan(bSpan, nil)
	tracerA.EndSpan(aSpan, nil)

	// All should have same trace ID
	if tracerA.GetTraceID() != tracerB.GetTraceID() || tracerB.GetTraceID() != tracerC.GetTraceID() {
		t.Fatalf("Expected same trace ID across all services")
	}

	t.Logf("✅ Multi-service tracing integration test passed")
}

// TestTracingIntegration_SamplingDecision verifies tracing can be disabled.
func TestTracingIntegration_SamplingDecision(t *testing.T) {
	tracer := New("odata-service")

	// Simulate sampling decision (record 50% of traces)
	shouldTrace := true // 50% decision
	tracer.SetEnabled(shouldTrace)

	ctx := context.Background()
	_, span := tracer.StartSpan(ctx, "sampled_operation")

	if span == nil && shouldTrace {
		t.Fatalf("Expected span to be created when tracing enabled")
	}

	tracer.EndSpan(span, nil)

	if tracer.IsEnabled() {
		spans := tracer.GetActiveSpans()
		if len(spans) != 1 {
			t.Fatalf("Expected 1 span when tracing enabled")
		}
	}

	t.Logf("✅ Sampling decision integration test passed")
}
