# OpenTelemetry Tracing Extension

The tracing extension provides distributed tracing capabilities for OData operations, enabling observability across microservices with W3C trace context support.

## Installation

```bash
go get github.com/jhonsferg/traverse/ext/tracing
```

## Quick Start

```go
import (
    "context"
    "github.com/jhonsferg/traverse"
    "github.com/jhonsferg/traverse/ext/tracing"
)

// Create tracer
tracer := tracing.New("odata-client")

// Start a span
_, span := tracer.StartSpan(context.Background(), "query_customers")

// Add attributes
tracer.SetAttribute(span, "entity_set", "Customers")

// Record an event
tracer.AddEvent(span, "query_completed", map[string]interface{}{
    "row_count": 42,
})

// End span (with optional error)
tracer.EndSpan(span, nil)
```

## Core Concepts

### Traces and Spans

A **trace** represents a complete operation (e.g., API request).
A **span** represents a unit of work within that trace (e.g., database query).

```
API Request Trace (trace_id: abc123)
├─ API Handler Span
├─ Database Query Span
│  ├─ Connection Setup Span
│  └─ Query Execution Span
└─ Response Formatting Span
```

### Attributes and Events

**Attributes**: Metadata about a span (immutable)
**Events**: Timestamped occurrences during span execution (timestamped logs)

```go
// Attributes: "what is this span about?"
tracer.SetAttribute(span, "entity_set", "Customers")
tracer.SetAttribute(span, "filter", "Status eq 'A'")

// Events: "what happened during this span?"
tracer.AddEvent(span, "query_started", map[string]interface{}{})
tracer.AddEvent(span, "cache_hit", map[string]interface{}{})
tracer.AddEvent(span, "query_completed", map[string]interface{}{
    "row_count": 1234,
})
```

## Single Service Tracing

### Basic Operation Tracing

```go
client := traverse.New("https://odata.example.com/v2")
tracer := tracing.New("odata-service")

ctx := context.Background()

// Query operation
_, querySpan := tracer.StartSpan(ctx, "fetch_customers")
tracer.SetAttribute(querySpan, "entity_set", "Customers")
tracer.SetAttribute(querySpan, "top", 100)

result, err := client.EntitySet("Customers").Top(100).Collect(ctx)

tracer.AddEvent(querySpan, "query_completed", map[string]interface{}{
    "rows": len(result),
})
tracer.EndSpan(querySpan, err)
```

### CRUD Operation Tracing

```go
// Create
_, span := tracer.StartSpan(ctx, "create_order")
tracer.SetAttribute(span, "operation", "CREATE")
err := client.EntitySet("Orders").Create(ctx, data)
tracer.EndSpan(span, err)

// Update
_, span := tracer.StartSpan(ctx, "update_order")
tracer.SetAttribute(span, "operation", "UPDATE")
err := client.EntitySet("Orders").Key("ID").Update(ctx, data)
tracer.EndSpan(span, err)

// Delete
_, span := tracer.StartSpan(ctx, "delete_order")
tracer.SetAttribute(span, "operation", "DELETE")
err := client.EntitySet("Orders").Key("ID").Delete(ctx)
tracer.EndSpan(span, err)
```

### Nested Spans (Hierarchical Operations)

```go
// Root span
_, rootSpan := tracer.StartSpan(ctx, "process_batch")

// Child span 1
_, validateSpan := tracer.StartSpan(ctx, "validate_data")
// ... validation logic ...
tracer.EndSpan(validateSpan, nil)

// Child span 2
_, importSpan := tracer.StartSpan(ctx, "import_records")
// ... import logic ...
tracer.EndSpan(importSpan, nil)

// Close root
tracer.EndSpan(rootSpan, nil)
```

## Distributed Tracing

### Cross-Service Propagation

Propagate trace context to other services:

```
Service A (Frontend)
  └─ create tracer & start span
  └─ call Service B
     └─ Service B (Backend)
        └─ extract trace context
        └─ continue same span
        └─ call Service C
           └─ Service C (Database)
              └─ extract trace context
              └─ complete span
```

### Implementation

**Service A: Initiator**
```go
tracerA := tracing.New("frontend-api")
_, span := tracerA.StartSpan(ctx, "api_request")

// Inject trace context for propagation
carrier := tracerA.Inject()

// Send carrier to Service B (via HTTP header, message queue, etc.)
```

**Service B: Intermediate**
```go
// Receive carrier from Service A
tracerB := tracing.Extract(carrier)
_, span := tracerB.StartSpan(ctx, "process_request")

// Same trace ID as Service A
log.Printf("Trace ID: %s", tracerB.GetTraceID())

// Propagate to Service C
carrier2 := tracerB.Inject()
```

**Service C: Terminal**
```go
// Receive carrier from Service B
tracerC := tracing.Extract(carrier2)
_, span := tracerC.StartSpan(ctx, "execute_query")

// All have same trace ID
log.Printf("Trace ID: %s", tracerC.GetTraceID())
```

### HTTP Header Propagation

Use W3C trace context format for HTTP:

```go
// Service A: Create request
tracer := tracing.New("api-client")
_, span := tracer.StartSpan(ctx, "http_request")

// Get W3C trace context
w3cContext := tracer.GetTraceContext()

// Add to HTTP header
req.Header.Set("traceparent", w3cContext)

// Service B: Extract from header
w3cValue := req.Header.Get("traceparent")
carrier := parseW3CFormat(w3cValue)
tracer := tracing.Extract(carrier)
```

## W3C Trace Context

The extension uses W3C Trace Context format for distributed tracing compatibility:

```
Format: 00-{traceID}-{spanID}-{traceFlags}

Example: 00-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-01

Fields:
- 00: Version (current W3C version)
- traceID: 32 hex digits (128-bit)
- spanID: 16 hex digits (64-bit)
- traceFlags: 01 = trace must be recorded
```

### Using W3C Context

```go
tracer := tracing.New("service")

// Get W3C context for propagation
w3cContext := tracer.GetTraceContext()

// Use in various transports:

// HTTP header
httpReq.Header.Set("traceparent", w3cContext)

// Message queue
message.Headers["traceparent"] = w3cContext

// gRPC metadata
md := metadata.Pairs("traceparent", w3cContext)
```

## Baggage Propagation

Propagate user/request context across services:

```go
// Service A: Add context
tracer := tracing.New("frontend")
tracer.AddBaggage("user_id", "user@example.com")
tracer.AddBaggage("tenant_id", "tenant-123")
tracer.AddBaggage("request_id", "req-abc")

carrier := tracer.Inject()

// Service B: Extract and use
tracer := tracing.Extract(carrier)

userID := tracer.GetBaggage("user_id")
tenantID := tracer.GetBaggage("tenant_id")
requestID := tracer.GetBaggage("request_id")

log.Printf("Request %s from tenant %s (user: %s)", requestID, tenantID, userID)
```

## Error Tracking

### Recording Errors in Spans

```go
_, span := tracer.StartSpan(ctx, "risky_operation")

// Attempt operation
result, err := client.EntitySet("Data").Collect(ctx)

if err != nil {
    // Record error context
    tracer.AddEvent(span, "error", map[string]interface{}{
        "error_type": reflect.TypeOf(err).String(),
        "error_message": err.Error(),
        "retry_count": 3,
    })
}

tracer.EndSpan(span, err)
```

### Error Statistics

```go
stats := tracer.GetStats()

log.Printf("Total spans: %d", stats["total_spans"])
log.Printf("Successful: %d", stats["successful"])
log.Printf("Errors: %d", stats["errors"])

errorRate := float64(stats["errors"].(int)) / 
             float64(stats["total_spans"].(int))
log.Printf("Error rate: %.2f%%", errorRate*100)
```

## Performance Monitoring

### Latency Tracking

```go
// Track query performance
_, querySpan := tracer.StartSpan(ctx, "query_large_dataset")
tracer.SetAttribute(querySpan, "rows_expected", 10000)

// ... execute query ...

tracer.EndSpan(querySpan, nil)

// Later: Analyze latencies
spans := tracer.GetActiveSpans()
for _, s := range spans {
    if s.Status == "success" {
        log.Printf("%s took %v", s.Name, s.Duration)
    }
}

// Get average
stats := tracer.GetStats()
avgDuration := stats["average_duration"].(time.Duration)
log.Printf("Average latency: %v", avgDuration)
```

### Sampling

To reduce overhead in high-volume scenarios:

```go
// Sample 10% of operations
import "math/rand"

shouldTrace := rand.Float64() < 0.1
tracer.SetEnabled(shouldTrace)

if tracer.IsEnabled() {
    _, span := tracer.StartSpan(ctx, "operation")
    // ... trace operation ...
}
```

## Example: Full Microservices Setup

```go
package main

import (
    "context"
    "log"
    "net/http"
    
    "github.com/jhonsferg/traverse"
    "github.com/jhonsferg/traverse/ext/tracing"
)

// Middleware to extract trace context from incoming request
func tracingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Get W3C trace context from header
        traceparent := r.Header.Get("traceparent")
        
        var tracer *tracing.Tracer
        if traceparent != "" {
            // Extract if header present
            carrier := parseTraceparent(traceparent)
            tracer = tracing.Extract(carrier)
        } else {
            // Start new trace
            tracer = tracing.New("api-service")
            tracer.AddBaggage("user_id", extractUserID(r))
        }
        
        // Continue to handler
        ctx := context.WithValue(r.Context(), "tracer", tracer)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

// Handler for OData queries
func handleQuery(w http.ResponseWriter, r *http.Request) {
    tracer := r.Context().Value("tracer").(*tracing.Tracer)
    
    _, span := tracer.StartSpan(r.Context(), "odata_query")
    tracer.SetAttribute(span, "entity", "Customers")
    
    client := traverse.New("https://odata.example.com/v2")
    result, err := client.EntitySet("Customers").Collect(r.Context())
    
    tracer.AddEvent(span, "query_completed", map[string]interface{}{
        "count": len(result),
    })
    tracer.EndSpan(span, err)
    
    // Return response
    w.Header().Set("X-Trace-ID", tracer.GetTraceID())
    w.Header().Set("traceparent", tracer.GetTraceContext())
}

func main() {
    router := http.NewServeMux()
    router.HandleFunc("/query", handleQuery)
    
    wrapped := tracingMiddleware(router)
    http.ListenAndServe(":8080", wrapped)
}
```

## Statistics and Analysis

### Get All Statistics

```go
stats := tracer.GetStats()

log.Printf("Total Spans: %d", stats["total_spans"])
log.Printf("Successful: %d", stats["successful"])
log.Printf("Errors: %d", stats["errors"])
log.Printf("Total Duration: %v", stats["total_duration"])
log.Printf("Average Duration: %v", stats["average_duration"])
log.Printf("Trace ID: %s", stats["trace_id"])
```

### Active Spans Analysis

```go
spans := tracer.GetActiveSpans()

for _, span := range spans {
    log.Printf("Span: %s", span.Name)
    log.Printf("  Duration: %v", span.Duration)
    log.Printf("  Status: %s", span.Status)
    log.Printf("  Attributes: %v", span.Attributes)
    log.Printf("  Events: %d", len(span.Events))
}
```

### Clear Spans

```go
// Clear recorded spans (for memory management)
tracer.ClearSpans()

// Verify cleared
spans := tracer.GetActiveSpans()
log.Printf("Remaining spans: %d", len(spans)) // 0
```

## Best Practices

1. **Start with key operations**
   ```go
   // Trace important paths first
   tracer.StartSpan(ctx, "api_request")
   tracer.StartSpan(ctx, "database_query")
   tracer.StartSpan(ctx, "cache_lookup")
   ```

2. **Add meaningful attributes**
   ```go
   tracer.SetAttribute(span, "entity_set", "Orders")
   tracer.SetAttribute(span, "filter", filterString)
   tracer.SetAttribute(span, "user_id", userID)
   ```

3. **Record errors immediately**
   ```go
   if err != nil {
       tracer.AddEvent(span, "error", map[string]interface{}{
           "error": err.Error(),
       })
   }
   ```

4. **Use sampling in production**
   ```go
   shouldTrace := rand.Float64() < 0.01 // 1% sampling
   tracer.SetEnabled(shouldTrace)
   ```

5. **Propagate context across services**
   ```go
   carrier := tracer.Inject()
   // Send to downstream service
   ```

## Troubleshooting

### Missing Trace Context in Child Services

Ensure `Extract()` is called before starting new spans:

```go
// ❌ Wrong: Creates new trace ID
tracer := tracing.New("service")

// ✅ Correct: Preserves trace ID
carrier := extractFromRequest(r)
tracer := tracing.Extract(carrier)
```

### High Memory Usage

Clear spans periodically:

```go
go func() {
    ticker := time.NewTicker(time.Hour)
    for range ticker.C {
        tracer.ClearSpans()
    }
}()
```

### No Trace Data

Verify spans are being recorded:

```go
stats := tracer.GetStats()
log.Printf("Total spans: %d", stats["total_spans"])

if stats["total_spans"] == 0 {
    log.Println("No spans recorded - verify StartSpan/EndSpan calls")
}
```

## License

MIT License - See LICENSE file in parent directory
