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

## SAP BTP / XSUAA token exchange

The `ext/sap` package also supports **SAP Business Technology Platform (BTP)** via the XSUAA identity service, enabling automatic OAuth2 token exchange for cloud-native SAP services.

### Parsing VCAP_SERVICES

In a Cloud Foundry or BTP environment, service credentials are injected via the `VCAP_SERVICES` environment variable. Parse them with:

```go
import "github.com/jhonsferg/traverse/ext/sap"

// Read from the environment (defaults to the "xsuaa" binding)
binding, err := sap.ParseVCAPServicesEnv("xsuaa")
if err != nil {
    log.Fatal(err)
}
```

Or supply the JSON string directly:

```go
binding, err := sap.ParseVCAPServices(os.Getenv("VCAP_SERVICES"), "xsuaa")
```

The returned `XSUAABinding` contains the `ClientID`, `ClientSecret`, `TokenURL`, optional `APIURL`, and `ZoneID`.

### Creating a BTP client

The easiest path reads `VCAP_SERVICES` automatically:

```go
client, err := sap.NewBTPClient(ctx, "https://my-service.sap.example.com/odata/")
if err != nil {
    log.Fatal(err)
}
```

Or build from an explicit binding (useful in tests):

```go
client, err := sap.NewBTPClientFromBinding(ctx, binding, serviceURL)
```

`NewBTPClientFromBinding` validates credentials, performs an initial token fetch to verify connectivity, and returns a fully configured traverse client with OAuth2 token injection and CSRF handling.

### BTPTokenProvider

`BTPTokenProvider` wraps the OAuth2 token manager and adds a 30-second error back-off -- repeated failures within the window return the cached error without hammering the token endpoint:

```go
provider := sap.NewBTPTokenProvider(binding)
token, err := provider.Token(ctx)
```

### XSUAABinding fields

| Field | Description |
|-------|-------------|
| `ClientID` | OAuth2 client ID from the service binding |
| `ClientSecret` | OAuth2 client secret |
| `TokenURL` | Full token endpoint (`{url}/oauth/token`) |
| `APIURL` | Base URL of the BTP service (optional) |
| `ZoneID` | Identity zone / subaccount ID |

## See also

- [SAP Guide](../guides/sap.md)
- [OAuth2 extension](oauth2.md)
- [Extensions Overview](index.md)
