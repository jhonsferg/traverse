# SAP Extension

The SAP extension (`ext/sap`) provides compatibility with SAP NetWeaver Gateway and SAP S/4HANA OData services.

## Installation

```bash
go get github.com/jhonsferg/traverse/ext/sap@latest
```

## Features

- Automatic CSRF token fetch and caching
- SAP-specific error message parsing (`innererror` / `errordetails`)
- Function import helpers
- x-csrf-token header injection on all mutating requests
- Compatible with SAP Gateway (OData v2) and S/4HANA (OData v4)

## Quick start

```go
import (
    "github.com/jhonsferg/traverse"
    "github.com/jhonsferg/traverse/ext/sap"
)

client := traverse.New(traverse.Config{
    BaseURL:   "https://my.sap.host/sap/opu/odata/sap/API_PRODUCT_SRV/",
    Extension: sap.Extension(),
})
```

## CSRF token lifecycle

1. On the first mutating request (POST/PUT/PATCH/DELETE), the extension issues a `HEAD` request with `X-CSRF-Token: Fetch`.
2. The returned token is cached in memory.
3. All subsequent mutating requests include `X-CSRF-Token: <cached>`.
4. On a 403 `X-CSRF-Token: Required` response, the token is invalidated and re-fetched automatically.

## Error handling

```go
var sapErr *sap.Error
if errors.As(err, &sapErr) {
    fmt.Println(sapErr.Code)
    fmt.Println(sapErr.Message.Value)
    for _, d := range sapErr.Details {
        fmt.Printf("  %s: %s\n", d.Code, d.Message.Value)
    }
}
```

## See also

- [SAP Guide](../guides/sap.md)
- [Extensions Overview](index.md)
