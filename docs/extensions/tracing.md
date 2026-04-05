# OpenTelemetry Tracing Extension

The OpenTelemetry extension (`ext/otel`) instruments traverse requests with distributed tracing using the OpenTelemetry Go SDK.

## Installation

```bash
go get github.com/jhonsferg/traverse/ext/otel@latest
```

## Quick start

```go
import (
    "github.com/jhonsferg/traverse"
    "github.com/jhonsferg/traverse/ext/otel"
)

client := traverse.New(traverse.Config{
    BaseURL: "https://api.example.com/odata/",
    Extension: otel.Extension(otel.Config{
        ServiceName: "order-service",
    }),
})
```

## Span attributes

Each request creates a span with these attributes:

| Attribute | Value |
|-----------|-------|
| `http.method` | GET, POST, PATCH, DELETE |
| `http.url` | Full request URL |
| `http.status_code` | Response status |
| `odata.entity` | Entity set name (e.g. Products) |
| `odata.operation` | list, get, create, update, delete |

## Custom tracer provider

```go
otel.Extension(otel.Config{
    ServiceName:    "my-service",
    TracerProvider: myTracerProvider, // inject your own
})
```

## Propagation

The extension automatically injects W3C `traceparent` and `tracestate` headers into every outgoing request, enabling trace propagation to the OData backend if it supports OpenTelemetry.

## Combining with Prometheus

```go
client := traverse.New(traverse.Config{
    BaseURL: "https://api.example.com/odata/",
    Extension: traverse.Chain(
        otel.Extension(otel.Config{ServiceName: "svc"}),
        prometheus.Extension(),
    ),
})
```

## See also

- [Prometheus Extension](prometheus.md)
- [Extensions Overview](index.md)
