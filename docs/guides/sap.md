# SAP Compatibility

traverse includes a dedicated SAP compatibility layer for connecting to SAP NetWeaver Gateway and SAP S/4HANA OData services, which implement a subset of OData v2 with SAP-specific extensions.

## Configuration

```go
import (
    "github.com/jhonsferg/traverse"
    "github.com/jhonsferg/traverse/ext/sap"
)

client := traverse.New(traverse.Config{
    BaseURL:    "https://my.sap.host/sap/opu/odata/sap/API_SALES_ORDER_SRV/",
    Extension:  sap.Extension(),
})
```

## CSRF token handling

SAP services require a CSRF token for all mutating requests (POST, PUT, PATCH, DELETE). The SAP extension fetches and caches the token automatically:

```go
// The extension handles token fetch transparently
order := map[string]any{
    "SalesOrderType": "TA",
    "SoldToParty":    "100001",
}

var created map[string]any
_, err := client.Collection("A_SalesOrder").Create(ctx, order, &created)
// CSRF token was automatically fetched on the first mutating request
```

## SAP message parsing

SAP returns structured error messages in its own format. The extension translates them to standard traverse errors:

```go
_, err := client.Collection("A_SalesOrder").Create(ctx, order, &result)
var sapErr *sap.Error
if errors.As(err, &sapErr) {
    log.Printf("SAP code: %s", sapErr.Code)
    for _, detail := range sapErr.Details {
        log.Printf("  detail: [%s] %s", detail.Code, detail.Message)
    }
}
```

## Function imports (SAP-style)

Call SAP function imports using the typed action helper:

```go
result, err := client.FunctionImport("BAPI_SALESORDER_SIMULATE").
    Param("OrderType", "TA").
    Param("SoldToParty", "100001").
    Execute(ctx)
```

## Batch requests

SAP Gateway supports OData batch but with stricter requirements. The SAP extension handles the `--changeset` boundary and encoding:

```go
batch := client.Batch()
batch.Create("A_SalesOrder", order1)
batch.Create("A_SalesOrder", order2)
results, err := batch.Execute(ctx)
```

## Authentication

### Basic authentication

```go
client := traverse.New(traverse.Config{
    BaseURL:   "https://my.sap.host/sap/opu/odata/...",
    Extension: sap.Extension(),
    Auth: traverse.BasicAuth{
        Username: "sapuser",
        Password: "sappass",
    },
})
```

### SAP OAuth2 (Cloud)

```go
client := traverse.New(traverse.Config{
    BaseURL:   "https://my.sap.host/.../",
    Extension: sap.Extension(),
    Auth: sap.OAuth2{
        TokenURL:     "https://mytenant.authentication.eu10.hana.ondemand.com/oauth/token",
        ClientID:     os.Getenv("SAP_CLIENT_ID"),
        ClientSecret: os.Getenv("SAP_CLIENT_SECRET"),
    },
})
```

## See also

- [Extensions - SAP](../extensions/sap.md)
- [Batch Requests](batch.md)
- [Async Operations](async-operations.md)
