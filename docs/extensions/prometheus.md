# Prometheus Extension

The Prometheus extension (`ext/prometheus`) exposes request metrics for traverse clients as Prometheus counters and histograms.

## Installation

```bash
go get github.com/jhonsferg/traverse/ext/prometheus@latest
```

## Quick start

```go
import (
    "github.com/jhonsferg/traverse"
    "github.com/jhonsferg/traverse/ext/prometheus"
)

client := traverse.New(traverse.Config{
    BaseURL:   "https://api.example.com/odata/",
    Extension: prometheus.Extension(),
})
```

## Metrics exposed

| Metric | Type | Labels |
|--------|------|--------|
| `traverse_requests_total` | Counter | `entity`, `method`, `status` |
| `traverse_request_duration_seconds` | Histogram | `entity`, `method` |
| `traverse_errors_total` | Counter | `entity`, `error_type` |
| `traverse_retries_total` | Counter | `entity` |

## Custom namespace and labels

```go
prometheus.Extension(prometheus.Config{
    Namespace: "myapp",
    ConstLabels: map[string]string{
        "service": "order-service",
        "env":     "production",
    },
})
```

## Custom registerer

Register metrics with a custom Prometheus registry:

```go
reg := prometheus.NewRegistry()

prometheus.Extension(prometheus.Config{
    Registerer: reg,
})
```

## See also

- [OpenTelemetry Tracing](tracing.md)
- [Extensions Overview](index.md)
