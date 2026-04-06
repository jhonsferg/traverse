# Extensions Overview

traverse extensions add optional capabilities that are not part of the core OData client. Each extension is a separate Go module to keep the core dependency-free.

## Available extensions

| Extension | Module | Description |
|-----------|--------|-------------|
| [SAP](sap.md) | `ext/sap` | CSRF tokens, SAP error parsing, BTP/XSUAA token exchange |
| [OAuth2](oauth2.md) | `ext/oauth2` | OAuth2 token management and automatic refresh |
| [Prometheus](prometheus.md) | `ext/prometheus` | Request metrics exposed as Prometheus counters/histograms |
| [OpenTelemetry](tracing.md) | `ext/otel` | Distributed tracing with OTel spans |
| [Cache](cache.md) | `ext/cache` | Response caching with ETag and `Cache-Control` awareness |
| [Stale Cache](stale-cache.md) | `ext/cache/stale` | Stale-while-revalidate cache for metadata and reference data |
| [Webhooks](webhooks.md) | `ext/webhooks` | OData v4 webhook subscription lifecycle and dispatch |
| [GraphQL Bridge](graphql.md) | `ext/graphql` | Translate OData queries to GraphQL automatically |
| [Microsoft Graph](graph.md) | `ext/graph` | Microsoft Graph API helpers |

## Installing extensions

Each extension is a separate module:

```bash
go get github.com/jhonsferg/traverse/ext/sap@latest
go get github.com/jhonsferg/traverse/ext/otel@latest
```

## Using multiple extensions

Extensions are composable. Pass them via `traverse.Config.Extension` or chain with `traverse.Chain`:

```go
client := traverse.New(traverse.Config{
    BaseURL: "https://api.example.com/odata/",
    Extension: traverse.Chain(
        sap.Extension(),
        otel.Extension(otel.Config{ServiceName: "my-service"}),
        prometheus.Extension(),
    ),
})
```

## Writing a custom extension

Implement the `traverse.Extension` interface:

```go
type Extension interface {
    WrapTransport(http.RoundTripper) http.RoundTripper
    OnRequest(ctx context.Context, req *http.Request) error
    OnResponse(ctx context.Context, req *http.Request, resp *http.Response) error
}
```

Extensions can wrap the transport (for low-level control) or use hooks (for request/response inspection).
