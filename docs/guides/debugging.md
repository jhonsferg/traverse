# Debugging & Raw Response Handling

traverse provides several tools for debugging OData responses and testing unexpected formats.

## CreateRawAs - Raw Response Bytes (v0.20.0+)

The `CreateRawAs()` method returns raw response bytes from Create operations, complementing the typed `CreateJsonAs()` and `CreateXmlAs()` methods. This is useful for:

- Debugging SAP response formats
- Testing unexpected content types
- Handling non-standard OData formats
- Transparently capturing both JSON and XML responses
- Troubleshooting CSRF or authentication issues

### Basic Usage

```go
import (
    "context"
    "encoding/json"
    "github.com/jhonsferg/traverse"
)

type Order struct {
    OrderID string `json:"OrderID"`
    Amount  float64 `json:"Amount"`
}

client, _ := traverse.New(
    traverse.WithBaseURL("https://api.example.com/odata/v4/"),
)

order := Order{OrderID: "1001", Amount: 99.99}

// Get raw response bytes
rawBytes, err := traverse.CreateRawAs(
    client.From("Orders"),
    context.Background(),
    order,
)
if err != nil {
    log.Fatal(err)
}

// Inspect raw content
log.Printf("Raw response: %s\n", string(rawBytes))

// Parse manually if needed
var custom map[string]interface{}
json.Unmarshal(rawBytes, &custom)
```

### SAP Backend Debugging

When debugging SAP OData integrations, use `CreateRawAs()` to inspect actual response formats:

```go
import (
    "github.com/jhonsferg/traverse"
    "github.com/jhonsferg/traverse/ext/sap"
)

client, _ := traverse.New(
    traverse.WithBaseURL("https://sap.example.com/sap/opu/odata/sap/API_SALES_ORDER_SRV/"),
    sap.WithCSRFMiddleware(),
)

order := Order{
    SalesOrderType: "TA",
    SoldToParty:    "100001",
}

// Capture raw response for analysis
rawBytes, err := traverse.CreateRawAs(
    client.From("A_SalesOrder"),
    context.Background(),
    order,
)
if err != nil {
    log.Printf("Error: %v", err)
    return
}

// Log actual response received
log.Printf("SAP returned:\n%s\n", string(rawBytes))

// Check if response is JSON or XML
contentType := /* check Content-Type header */
log.Printf("Content-Type: %s", contentType)
```

### Handling Mixed Formats

Some backends may return XML instead of JSON, even when JSON is requested. Use `CreateRawAs()` to detect and handle both:

```go
// Get raw bytes
rawBytes, err := traverse.CreateRawAs(...)
if err != nil {
    log.Fatal(err)
}

// Detect format
var result interface{}
if err := json.Unmarshal(rawBytes, &result); err == nil {
    log.Println("Response is JSON")
    // Process as JSON
} else if isXML(rawBytes) {
    log.Println("Response is XML")
    // Process as XML using CreateXmlAs
} else {
    log.Printf("Unknown format: %s", string(rawBytes))
}
```

## Error Analysis

When CSRF or authentication fails, use raw response inspection:

```go
import (
    "github.com/jhonsferg/traverse"
    "github.com/jhonsferg/traverse/ext/sap"
)

client, _ := traverse.New(
    traverse.WithBaseURL("https://sap.example.com/..."),
    sap.WithCSRFMiddleware(),
)

rawBytes, err := traverse.CreateRawAs(
    client.From("Orders"),
    context.Background(),
    order,
)

if err != nil {
    // Inspect error details
    log.Printf("Error: %v", err)
    
    // Check if we got a response body
    var errDetail map[string]interface{}
    if err := json.Unmarshal(rawBytes, &errDetail); err == nil {
        log.Printf("Server response: %v", errDetail)
    }
    
    // For SAP errors
    var sapErr *sap.Error
    if errors.As(err, &sapErr) {
        log.Printf("SAP error code: %s", sapErr.Code)
        for _, detail := range sapErr.Details {
            log.Printf("  - [%s] %s", detail.Code, detail.Message)
        }
    }
}
```

## Comparison: JsonAs vs XmlAs vs RawAs

| Method | Return Type | Best For | Format |
|--------|-------------|----------|--------|
| `CreateJsonAs[T]()` | `T` (struct) | Typed responses when JSON format guaranteed | JSON only |
| `CreateXmlAs[T]()` | `T` (struct) | Typed responses when XML format guaranteed | XML only |
| `CreateRawAs()` | `[]byte` | Debugging, format detection, mixed formats | JSON or XML |

### Workflow Recommendation

1. **Development/testing**: Use `CreateRawAs()` to verify actual response formats
2. **Format known**: Switch to `CreateJsonAs[T]()` or `CreateXmlAs[T]()`  
3. **Format varies**: Keep using `CreateRawAs()` with manual parsing based on detection
4. **Debugging failures**: Always use `CreateRawAs()` to inspect what server actually returned

## Logging & Observability

Use relay's built-in hooks to log all requests/responses:

```go
import (
    "github.com/jhonsferg/relay"
)

client, _ := traverse.New(
    traverse.WithBaseURL("https://api.example.com/odata/v4/"),
    relay.WithOnBeforeRequest(func(ctx context.Context, req *http.Request) error {
        log.Printf("→ %s %s", req.Method, req.URL.String())
        return nil
    }),
    relay.WithOnAfterResponse(func(ctx context.Context, resp *http.Response) error {
        log.Printf("← %d", resp.StatusCode)
        return nil
    }),
)
```

## See also

- [CRUD Operations](crud.md)
- [XML Support](../guides/sap.md#xml-support)
- [SAP Compatibility - Raw Response Debugging](sap.md#raw-response-debugging-v0200)
- [CHANGELOG v0.20.0](../changelog.md#0200-2026-04-27)
