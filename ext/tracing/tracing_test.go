// Package tracing provides tests for OpenTelemetry integration.
package tracing

import (
	"context"
	"errors"
	"testing"
	"time"
)

// TestTracer_Creation verifies Tracer can be created.
func TestTracer_Creation(t *testing.T) {
	tracer := New("test-service")
	if tracer == nil {
		t.Fatalf("Expected non-nil Tracer")
	}
	if tracer.GetTraceID() == "" {
		t.Fatalf("Expected trace ID to be generated")
	}
	if tracer.GetSpanID() == "" {
		t.Fatalf("Expected span ID to be generated")
	}

	t.Logf("✅ Tracer creation test passed")
}

// TestTracer_StartEndSpan verifies span lifecycle.
func TestTracer_StartEndSpan(t *testing.T) {
	tracer := New("test-service")
	ctx := context.Background()

	// Start span
	newCtx, span := tracer.StartSpan(ctx, "query")
	if span == nil {
		t.Fatalf("Expected non-nil span")
	}
	if newCtx == nil {
		t.Fatalf("Expected context with trace info")
	}

	// Simulate work
	time.Sleep(10 * time.Millisecond)

	// End span
	tracer.EndSpan(span, nil)

	if span.Status != "success" {
		t.Fatalf("Expected success status")
	}
	if span.Duration == 0 {
		t.Fatalf("Expected non-zero duration")
	}

	t.Logf("✅ Span lifecycle test passed (duration: %v)", span.Duration)
}

// TestTracer_SpanWithError verifies error handling in spans.
func TestTracer_SpanWithError(t *testing.T) {
	tracer := New("test-service")
	ctx := context.Background()

	_, span := tracer.StartSpan(ctx, "failing-query")

	testErr := errors.New("query failed")
	tracer.EndSpan(span, testErr)

	if span.Status != "error" {
		t.Fatalf("Expected error status")
	}
	if span.Error != testErr {
		t.Fatalf("Expected error to be recorded")
	}

	t.Logf("✅ Span error handling test passed")
}

// TestTracer_AddEvent verifies events can be added to spans.
func TestTracer_AddEvent(t *testing.T) {
	tracer := New("test-service")
	ctx := context.Background()

	_, span := tracer.StartSpan(ctx, "operation")

	attrs := map[string]interface{}{
		"key": "value",
	}
	tracer.AddEvent(span, "phase1_complete", attrs)
	tracer.AddEvent(span, "phase2_complete", attrs)

	if len(span.Events) != 2 {
		t.Fatalf("Expected 2 events, got %d", len(span.Events))
	}

	t.Logf("✅ Add event test passed")
}

// TestTracer_SetAttribute verifies attributes can be set on spans.
func TestTracer_SetAttribute(t *testing.T) {
	tracer := New("test-service")
	ctx := context.Background()

	_, span := tracer.StartSpan(ctx, "query")

	tracer.SetAttribute(span, "entity_set", "Customers")
	tracer.SetAttribute(span, "row_count", 100)

	if span.Attributes["entity_set"] != "Customers" {
		t.Fatalf("Expected entity_set attribute")
	}
	if span.Attributes["row_count"] != 100 {
		t.Fatalf("Expected row_count attribute")
	}

	t.Logf("✅ Set attribute test passed")
}

// TestTracer_Baggage verifies baggage propagation.
func TestTracer_Baggage(t *testing.T) {
	tracer := New("test-service")

	tracer.AddBaggage("user_id", "user123")
	tracer.AddBaggage("correlation_id", "corr456")

	if tracer.GetBaggage("user_id") != "user123" {
		t.Fatalf("Expected user_id baggage")
	}
	if tracer.GetBaggage("correlation_id") != "corr456" {
		t.Fatalf("Expected correlation_id baggage")
	}

	t.Logf("✅ Baggage test passed")
}

// TestTracer_W3CTraceContext verifies W3C trace context format.
func TestTracer_W3CTraceContext(t *testing.T) {
	tracer := New("test-service")

	ctx := tracer.GetTraceContext()
	if ctx == "" {
		t.Fatalf("Expected trace context string")
	}

	// Should be in format: 00-traceID-spanID-01
	if len(ctx) < 5 {
		t.Fatalf("Invalid trace context format: %s", ctx)
	}

	t.Logf("✅ W3C trace context test passed (%s)", ctx)
}

// TestTracer_GetStats verifies statistics collection.
func TestTracer_GetStats(t *testing.T) {
	tracer := New("test-service")
	ctx := context.Background()

	// Record several spans
	for i := 0; i < 3; i++ {
		_, span := tracer.StartSpan(ctx, "operation")
		tracer.EndSpan(span, nil)
	}

	stats := tracer.GetStats()

	if stats["total_spans"].(int) != 3 {
		t.Fatalf("Expected 3 spans in stats")
	}
	if stats["successful"].(int) != 3 {
		t.Fatalf("Expected 3 successful spans")
	}

	t.Logf("✅ Get stats test passed")
}

// TestTracer_GetActiveSpans verifies active spans retrieval.
// After EndSpan, spans move to completedSpans — GetActiveSpans returns only in-progress spans.
func TestTracer_GetActiveSpans(t *testing.T) {
	tracer := New("test-service")
	ctx := context.Background()

	_, span1 := tracer.StartSpan(ctx, "op1")
	_, span2 := tracer.StartSpan(ctx, "op2")

	// Both spans are in-flight — should appear as active.
	active := tracer.GetActiveSpans()
	if len(active) != 2 {
		t.Fatalf("Expected 2 active spans before EndSpan, got %d", len(active))
	}

	tracer.EndSpan(span1, nil)
	tracer.EndSpan(span2, nil)

	// After EndSpan they are completed — active list must be empty.
	if active = tracer.GetActiveSpans(); len(active) != 0 {
		t.Fatalf("Expected 0 active spans after EndSpan, got %d", len(active))
	}

	// Completed spans are still retrievable via GetSpan.
	if tracer.GetSpan(span1.SpanID) == nil {
		t.Fatal("Expected to retrieve span1 from completed spans")
	}

	t.Logf("✅ Get active spans test passed")
}

// TestTracer_ClearSpans verifies spans can be cleared.
func TestTracer_ClearSpans(t *testing.T) {
	tracer := New("test-service")
	ctx := context.Background()

	_, span := tracer.StartSpan(ctx, "op")
	tracer.EndSpan(span, nil)

	// After EndSpan, the span is in completedSpans — verify retrievable.
	if tracer.GetSpan(span.SpanID) == nil {
		t.Fatalf("Expected span before clear")
	}

	tracer.ClearSpans()

	if tracer.GetSpan(span.SpanID) != nil {
		t.Fatalf("Expected span to be gone after clear")
	}
	if len(tracer.GetActiveSpans()) != 0 {
		t.Fatalf("Expected no active spans after clear")
	}

	t.Logf("✅ Clear spans test passed")
}

// TestTracer_EnableDisable verifies tracing can be toggled.
func TestTracer_EnableDisable(t *testing.T) {
	tracer := New("test-service")

	if !tracer.IsEnabled() {
		t.Fatalf("Expected tracing to be enabled by default")
	}

	tracer.SetEnabled(false)
	if tracer.IsEnabled() {
		t.Fatalf("Expected tracing to be disabled")
	}

	tracer.SetEnabled(true)
	if !tracer.IsEnabled() {
		t.Fatalf("Expected tracing to be enabled")
	}

	t.Logf("✅ Enable/disable test passed")
}

// TestTracer_ContextCarrier verifies context injection/extraction.
func TestTracer_ContextCarrier(t *testing.T) {
	tracer := New("test-service")
	tracer.AddBaggage("key1", "value1")

	// Inject context
	carrier := tracer.Inject()
	if carrier.TraceID == "" {
		t.Fatalf("Expected trace ID in carrier")
	}

	// Extract context
	tracer2 := Extract(carrier)
	if tracer2.GetTraceID() != tracer.GetTraceID() {
		t.Fatalf("Expected same trace ID after extraction")
	}

	t.Logf("✅ Context carrier test passed")
}

// TestTracer_MultipleSpans verifies multiple spans are correctly completed and retrievable.
func TestTracer_MultipleSpans(t *testing.T) {
	tracer := New("test-service")
	ctx := context.Background()

	spanIDs := make([]string, 0, 5)
	for i := 0; i < 5; i++ {
		_, span := tracer.StartSpan(ctx, "op"+string(rune('0'+i)))
		tracer.EndSpan(span, nil)
		spanIDs = append(spanIDs, span.SpanID)
	}

	// Active spans must be 0 — all ended.
	if n := len(tracer.GetActiveSpans()); n != 0 {
		t.Fatalf("Expected 0 active spans, got %d", n)
	}

	// All 5 spans must appear in stats.
	stats := tracer.GetStats()
	if stats["total_spans"].(int) != 5 {
		t.Fatalf("Expected 5 spans in stats, got %d", stats["total_spans"])
	}

	// All spans must be retrievable.
	for _, id := range spanIDs {
		if tracer.GetSpan(id) == nil {
			t.Fatalf("Expected to retrieve span %s", id)
		}
	}

	t.Logf("✅ Multiple spans test passed")
}

// TestTracer_Concurrency verifies thread-safe operations.
func TestTracer_Concurrency(t *testing.T) {
	tracer := New("test-service")
	ctx := context.Background()
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(id int) {
			_, span := tracer.StartSpan(ctx, "concurrent_op")
			tracer.SetAttribute(span, "id", id)
			tracer.EndSpan(span, nil)
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	// After all goroutines complete, active list must be empty.
	if n := len(tracer.GetActiveSpans()); n != 0 {
		t.Fatalf("Expected 0 active spans, got %d", n)
	}

	stats := tracer.GetStats()
	if stats["total_spans"].(int) != 10 {
		t.Fatalf("Expected 10 completed spans in stats, got %d", stats["total_spans"])
	}

	t.Logf("✅ Concurrency test passed (10 concurrent spans)")
}

// TestTracer_SpanDuration verifies duration calculation.
func TestTracer_SpanDuration(t *testing.T) {
	tracer := New("test-service")
	ctx := context.Background()

	_, span := tracer.StartSpan(ctx, "timed_op")

	sleepTime := 50 * time.Millisecond
	time.Sleep(sleepTime)

	tracer.EndSpan(span, nil)

	if span.Duration < sleepTime {
		t.Fatalf("Expected duration >= %v, got %v", sleepTime, span.Duration)
	}

	t.Logf("✅ Span duration test passed (%v)", span.Duration)
}
