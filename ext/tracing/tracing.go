// Package tracing provides OpenTelemetry tracing integration for traverse OData client.
// It supports W3C trace context and distributed tracing.
package tracing

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// Global counter to ensure unique span IDs across goroutines.
var spanCounter int64

// maxCompletedSpans is the maximum number of completed spans retained for stats/retrieval.
// Older spans are evicted when the limit is reached.
const maxCompletedSpans = 1000

// contextKey is an unexported type for context keys in this package,
// preventing collisions with keys from other packages.
type contextKey string

const (
	contextKeyTraceID contextKey = "trace_id"
	contextKeySpanID  contextKey = "span_id"
)

// Tracer provides distributed tracing capabilities for OData operations.
type Tracer struct {
	// Trace context
	traceID      string
	spanID       string
	parentSpanID string

	// Baggage (key-value pairs propagated with trace)
	baggage map[string]string

	// activeSpans holds spans that have been started but not yet ended.
	activeSpans map[string]*Span

	// completedSpans is a capped ring buffer of finished spans retained for
	// stats and retrieval. Capped at maxCompletedSpans to prevent OOM.
	completedSpans []*Span

	// Configuration
	serviceName string
	enabled     bool

	mu sync.RWMutex
}

// Span represents a single unit of work in a trace.
// Span represents a single unit of work in a trace.
//
// SpanID, TraceID, ParentID, Name and StartTime are set at creation and never
// modified — they are safe to read at any time. All other fields are
// protected by an internal mutex; use the accessor methods (Status, EndTime,
// Duration, Err, Events, Attributes) for concurrent-safe reads.
type Span struct {
	SpanID    string
	TraceID   string
	ParentID  string
	Name      string
	StartTime time.Time

	mu         sync.RWMutex
	endTime    time.Time
	duration   time.Duration
	status     string // "active" | "success" | "error"
	attributes map[string]interface{}
	events     []SpanEvent
	spanErr    error
}

// Status returns the current span status ("active", "success", or "error").
func (s *Span) Status() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.status
}

// EndTime returns the time the span was ended (zero if still active).
func (s *Span) EndTime() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.endTime
}

// Duration returns how long the span ran (zero if still active).
func (s *Span) Duration() time.Duration {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.duration
}

// Err returns the error recorded when the span ended, if any.
func (s *Span) Err() error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.spanErr
}

// Events returns a copy of the span's event list.
func (s *Span) Events() []SpanEvent {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]SpanEvent, len(s.events))
	copy(out, s.events)
	return out
}

// Attributes returns a copy of the span's attribute map.
func (s *Span) Attributes() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string]interface{}, len(s.attributes))
	for k, v := range s.attributes {
		out[k] = v
	}
	return out
}

// SpanEvent represents an event that occurred during span execution.
type SpanEvent struct {
	Timestamp  time.Time
	Name       string
	Attributes map[string]interface{}
}

// New creates a new Tracer instance.
func New(serviceName string) *Tracer {
	return &Tracer{
		traceID:        generateTraceID(),
		spanID:         generateSpanID(),
		baggage:        make(map[string]string),
		activeSpans:    make(map[string]*Span),
		completedSpans: make([]*Span, 0, maxCompletedSpans),
		serviceName:    serviceName,
		enabled:        true,
	}
}

// StartSpan creates a new span for an operation.
func (t *Tracer) StartSpan(ctx context.Context, spanName string) (context.Context, *Span) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.enabled {
		return ctx, nil
	}

	span := &Span{
		SpanID:     generateSpanID(),
		TraceID:    t.traceID,
		ParentID:   t.spanID,
		Name:       spanName,
		StartTime:  time.Now(),
		status:     "active",
		attributes: make(map[string]interface{}),
		events:     make([]SpanEvent, 0),
	}

	t.activeSpans[span.SpanID] = span

	// Propagate trace context via unexported keys to avoid package collisions.
	newCtx := context.WithValue(ctx, contextKeyTraceID, t.traceID)
	newCtx = context.WithValue(newCtx, contextKeySpanID, span.SpanID)

	return newCtx, span
}

// EndSpan finishes a span and records it.
func (t *Tracer) EndSpan(span *Span, err error) {
	if span == nil {
		return
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	span.mu.Lock()
	span.endTime = time.Now()
	span.duration = span.endTime.Sub(span.StartTime)
	if err != nil {
		span.status = "error"
		span.spanErr = err
	} else {
		span.status = "success"
	}
	span.mu.Unlock()

	// Move from active to completed, capped at maxCompletedSpans.
	delete(t.activeSpans, span.SpanID)
	if len(t.completedSpans) >= maxCompletedSpans {
		// Evict oldest half to amortise the cost of eviction.
		copy(t.completedSpans, t.completedSpans[maxCompletedSpans/2:])
		t.completedSpans = t.completedSpans[:maxCompletedSpans/2]
	}
	t.completedSpans = append(t.completedSpans, span)
}

// AddEvent adds an event to a span.
func (t *Tracer) AddEvent(span *Span, eventName string, attrs map[string]interface{}) {
	if span == nil {
		return
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	event := SpanEvent{
		Timestamp:  time.Now(),
		Name:       eventName,
		Attributes: attrs,
	}

	span.mu.Lock()
	span.events = append(span.events, event)
	span.mu.Unlock()
}

// SetAttribute sets an attribute on a span.
func (t *Tracer) SetAttribute(span *Span, key string, value interface{}) {
	if span == nil {
		return
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	span.mu.Lock()
	span.attributes[key] = value
	span.mu.Unlock()
}

// AddBaggage adds a baggage item (propagated with trace).
func (t *Tracer) AddBaggage(key, value string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.baggage[key] = value
}

// GetBaggage retrieves a baggage item.
func (t *Tracer) GetBaggage(key string) string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return t.baggage[key]
}

// GetTraceContext returns W3C trace context header value.
func (t *Tracer) GetTraceContext() string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if !t.enabled {
		return ""
	}

	// Format: version-traceID-spanID-traceFlags
	// version: 00 (current)
	// traceFlags: 01 (trace must be recorded)
	return fmt.Sprintf("00-%s-%s-01", t.traceID, t.spanID)
}

// GetTraceID returns the trace ID.
func (t *Tracer) GetTraceID() string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return t.traceID
}

// GetSpanID returns the current span ID.
func (t *Tracer) GetSpanID() string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return t.spanID
}

// GetActiveSpans returns all currently in-progress (not yet ended) spans.
func (t *Tracer) GetActiveSpans() []*Span {
	t.mu.RLock()
	defer t.mu.RUnlock()

	spans := make([]*Span, 0, len(t.activeSpans))
	for _, span := range t.activeSpans {
		spans = append(spans, span)
	}
	return spans
}

// GetSpan retrieves a specific span by ID from either active or completed spans.
func (t *Tracer) GetSpan(spanID string) *Span {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if s, ok := t.activeSpans[spanID]; ok {
		return s
	}
	for i := len(t.completedSpans) - 1; i >= 0; i-- {
		if t.completedSpans[i].SpanID == spanID {
			return t.completedSpans[i]
		}
	}
	return nil
}

// ClearSpans removes all recorded spans (active and completed).
func (t *Tracer) ClearSpans() {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.activeSpans = make(map[string]*Span)
	t.completedSpans = t.completedSpans[:0]
}

// SetEnabled enables or disables tracing.
func (t *Tracer) SetEnabled(enabled bool) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.enabled = enabled
}

// IsEnabled returns whether tracing is enabled.
func (t *Tracer) IsEnabled() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return t.enabled
}

// GetStats returns statistics about recorded spans.
func (t *Tracer) GetStats() map[string]interface{} {
	t.mu.RLock()
	defer t.mu.RUnlock()

	totalSpans := len(t.completedSpans)
	successCount := 0
	errorCount := 0
	totalDuration := time.Duration(0)

	for _, span := range t.completedSpans {
		if span.Status() == "success" {
			successCount++
		} else if span.Status() == "error" {
			errorCount++
		}
		totalDuration += span.Duration()
	}

	avgDuration := time.Duration(0)
	if totalSpans > 0 {
		avgDuration = totalDuration / time.Duration(totalSpans)
	}

	return map[string]interface{}{
		"total_spans":      totalSpans,
		"successful":       successCount,
		"errors":           errorCount,
		"total_duration":   totalDuration,
		"average_duration": avgDuration,
		"trace_id":         t.traceID,
		"service":          t.serviceName,
	}
}

// Helper functions

func generateTraceID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		// Fallback to counter-based ID when crypto/rand is unavailable.
		counter := atomic.AddInt64(&spanCounter, 1)
		return fmt.Sprintf("%016x%016x", time.Now().UnixNano(), counter)
	}
	return hex.EncodeToString(b[:])
}

func generateSpanID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		counter := atomic.AddInt64(&spanCounter, 1)
		return fmt.Sprintf("%016x", time.Now().UnixNano()^counter)
	}
	return hex.EncodeToString(b[:])
}

// ContextCarrier implements context propagation for distributed tracing.
type ContextCarrier struct {
	TraceID      string
	SpanID       string
	ParentSpanID string
	Baggage      map[string]string
}

// Extract extracts trace context from a carrier.
func Extract(carrier *ContextCarrier) *Tracer {
	t := &Tracer{
		traceID:        carrier.TraceID,
		spanID:         carrier.SpanID,
		parentSpanID:   carrier.ParentSpanID,
		baggage:        carrier.Baggage,
		activeSpans:    make(map[string]*Span),
		completedSpans: make([]*Span, 0, maxCompletedSpans),
		enabled:        true,
	}
	if t.baggage == nil {
		t.baggage = make(map[string]string)
	}
	return t
}

// Inject injects trace context into a carrier.
func (t *Tracer) Inject() *ContextCarrier {
	t.mu.RLock()
	defer t.mu.RUnlock()

	carrier := &ContextCarrier{
		TraceID:      t.traceID,
		SpanID:       t.spanID,
		ParentSpanID: t.parentSpanID,
		Baggage:      make(map[string]string),
	}

	for k, v := range t.baggage {
		carrier.Baggage[k] = v
	}

	return carrier
}
