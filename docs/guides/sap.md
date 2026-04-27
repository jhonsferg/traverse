# SAP Compatibility

traverse includes a dedicated SAP compatibility layer for connecting to SAP NetWeaver Gateway and SAP S/4HANA OData services, which implement a subset of OData v2 with SAP-specific extensions.

## Configuration

```go
import (
    "github.com/jhonsferg/traverse"
    "github.com/jhonsferg/traverse/ext/sap"
)

client := traverse.New(
    traverse.WithBaseURL("https://my.sap.host/sap/opu/odata/sap/API_SALES_ORDER_SRV/"),
    sap.WithCSRFMiddleware(),
)
```

## CSRF Token Handling (Automatic - v0.19.0+)

SAP services require a CSRF token for all mutating requests (POST, PUT, PATCH, DELETE). Starting with v0.19.0, the SAP extension handles CSRF tokens and session cookies **completely transparently**—no manual management needed.

### What's New (v0.19.0)

- ✅ **Automatic session persistence** - Session cookies are captured and reused across requests (via relay v0.4.0)
- ✅ **Atomic token + cookie pairing** - CSRF tokens and session cookies are managed as an inseparable unit
- ✅ **Token reuse** - Tokens are reused for their full 30-minute validity window (no preventive invalidation)
- ✅ **Automatic 403 recovery** - When tokens expire, the middleware automatically fetches a fresh one
- ✅ **Better error diagnostics** - Distinguishes CSRF failures from auth failures and network errors

### Usage

```go
import (
    "context"
    "github.com/jhonsferg/traverse"
    "github.com/jhonsferg/traverse/ext/sap"
)

client, _ := traverse.New(
    traverse.WithBaseURL("https://my.sap.host/sap/opu/odata/sap/API_SALES_ORDER_SRV/"),
    sap.WithCSRFMiddleware(),
)

// CSRF token and session cookies are handled automatically
// No explicit token management needed
order := Order{
    SalesOrderType: "TA",
    SoldToParty:    "100001",
}

created, err := sap.CreateJsonAs[Order](
    client.From("A_SalesOrder"),
    context.Background(),
    order,
)

// If token expires mid-flight, middleware automatically:
// 1. Detects the 403 error
// 2. Fetches a fresh token (with updated session cookie)
// 3. Retries the request
// This is all transparent to application code
```

### Under the Hood (v0.19.0 Architecture)

The CSRF middleware now follows a 4-phase protocol:

1. **Token Fetch** - GET `/metadata` (or similar) with `X-CSRF-Token: Fetch`
   - Server responds with `X-CSRF-Token` header
   - Response includes `Set-Cookie` header(s) for session
   - Middleware captures both token and cookies

2. **Cookie Persistence** - All cookies are automatically stored by `http.CookieJar`
   - No manual cookie management needed
   - Cookies are automatically included in all subsequent requests

3. **Mutating Request** - POST/PUT/PATCH/DELETE
   - Middleware injects both `X-CSRF-Token` header and `Cookie` header
   - Request uses stored session cookies from phase 1

4. **403 Recovery** - If token expires
   - Middleware detects `403 Forbidden` response
   - Automatically returns to phase 1 (token fetch)
   - Retries the original request with fresh token

### Error Diagnostics (v0.19.0+)

The extension now provides detailed error categorization:

```go
created, err := sap.CreateJsonAs[Order](...)
if err != nil {
    var diag *sap.ErrorDiagnostic
    if errors.As(err, &diag) {
        switch diag.Category {
        case sap.ErrorCSRFExpired:
            log.Println("Token expired—middleware will retry automatically")
        case sap.ErrorAuthFailed:
            log.Println("Authentication failure—check credentials")
        case sap.ErrorConfigError:
            log.Println("Configuration issue—check service URL")
        case sap.ErrorNetworkError:
            log.Println("Network connectivity issue")
        }
    }
}
```

## SAP Message Parsing

SAP returns structured error messages in its own format. The extension translates them to standard traverse errors:

```go
_, err := sap.CreateJsonAs[Order](
    client.From("A_SalesOrder"),
    context.Background(),
    order,
)
var sapErr *sap.Error
if errors.As(err, &sapErr) {
    log.Printf("SAP code: %s", sapErr.Code)
    for _, detail := range sapErr.Details {
        log.Printf("  detail: [%s] %s", detail.Code, detail.Message)
    }
}
```

## Function Imports (SAP-style)

Call SAP function imports using the typed action helper:

```go
result, err := client.FunctionImport("BAPI_SALESORDER_SIMULATE").
    Param("OrderType", "TA").
    Param("SoldToParty", "100001").
    Execute(ctx)
```

## Batch Requests

SAP Gateway supports OData batch but with stricter requirements. The SAP extension handles the `--changeset` boundary and encoding:

```go
batch := client.Batch()
batch.Create("A_SalesOrder", order1)
batch.Create("A_SalesOrder", order2)
results, err := batch.Execute(ctx)
```

## Raw Response Debugging (v0.20.0+)

For debugging or handling non-standard response formats, use `CreateRawAs()` to capture raw response bytes:

```go
import "github.com/jhonsferg/traverse"

// Get raw response bytes instead of unmarshaled struct
rawBytes, err := traverse.CreateRawAs(
    client.From("A_SalesOrder"),
    context.Background(),
    order,
)
if err != nil {
    log.Fatal(err)
}

// Inspect raw content for troubleshooting
log.Printf("Response: %s\n", string(rawBytes))

// Parse manually if needed
var custom map[string]interface{}
json.Unmarshal(rawBytes, &custom)
```

This is useful for:
- Debugging SAP response formats
- Testing unexpected content types
- Capturing both JSON and XML responses transparently
- Analyzing response structure before mapping to structs

## Authentication

### Basic Authentication

```go
client, _ := traverse.New(
    traverse.WithBaseURL("https://my.sap.host/sap/opu/odata/..."),
    traverse.WithBasicAuth("sapuser", "sappass"),
    sap.WithCSRFMiddleware(),
)
```

### SAP OAuth2 (Cloud)

```go
client, _ := traverse.New(
    traverse.WithBaseURL("https://my.sap.host/.../"),
    sap.WithCSRFMiddleware(),
    sap.WithOAuth2(
        "https://mytenant.authentication.eu10.hana.ondemand.com/oauth/token",
        os.Getenv("SAP_CLIENT_ID"),
        os.Getenv("SAP_CLIENT_SECRET"),
    ),
)
```

## See also

- [Extensions - SAP](../extensions/sap.md)
- [Batch Requests](batch.md)
- [Async Operations](async-operations.md)
- [CHANGELOG - Phase 10 CSRF Architecture](../changelog.md#0190-2026-04-27---phase-10-critical-sap-csrf-architecture-fix)
