# Microsoft Dataverse

OData adapter for Microsoft Dataverse (Power Platform / Dynamics 365), using the `ext/dataverse` extension. Handles Dataverse-specific headers, API versioning, OAuth2 bearer tokens, and user impersonation automatically.

## Installation

```bash
go get github.com/jhonsferg/traverse/ext/dataverse
```

## Quick Start

```go
import "github.com/jhonsferg/traverse/ext/dataverse"

client, err := dataverse.New(dataverse.Config{
    OrgURL: "https://myorg.api.crm.dynamics.com",
    BearerToken: func() (string, error) {
        return getTokenFromAAD() // your token acquisition logic
    },
})
if err != nil {
    log.Fatal(err)
}

// Build the entity URL and issue a request.
req, _ := http.NewRequestWithContext(ctx, http.MethodGet,
    client.ServiceURL()+"accounts?$select=name,accountid", nil)

resp, err := client.Do(req)
```

## API Reference

### `Config`

```go
type Config struct {
    // OrgURL is the Dataverse organisation URL, e.g. https://myorg.api.crm.dynamics.com
    OrgURL string

    // APIVersion is the Dataverse Web API version. Defaults to "9.2".
    APIVersion string

    // BearerToken is called before each request to supply an OAuth2 access token.
    // If nil, requests are sent without an Authorization header.
    BearerToken func() (string, error)

    // CallerID optionally impersonates a user by their system user GUID.
    // Sets the MSCRMCallerID request header.
    CallerID string

    // MaxPageSize sets the Prefer: odata.maxpagesize header. Defaults to 100.
    MaxPageSize int
}
```

### `Client`

```go
type Client struct { /* unexported */ }

func New(cfg Config) (*Client, error)
```

Returns an error if `OrgURL` is empty.

#### `ServiceURL`

```go
func (c *Client) ServiceURL() string
```

Returns the fully qualified Web API base URL, e.g. `https://myorg.api.crm.dynamics.com/api/data/v9.2/`.

#### `Do`

```go
func (c *Client) Do(req *http.Request) (*http.Response, error)
```

Executes an HTTP request with Dataverse-specific headers applied:

| Header | Value |
|--------|-------|
| `OData-MaxVersion` | `4.0` |
| `OData-Version` | `4.0` |
| `Accept` | `application/json; odata.metadata=minimal` |
| `Authorization` | `Bearer <token>` (when `BearerToken` is set) |
| `MSCRMCallerID` | `<CallerID>` (when set) |
| `Prefer` | `odata.maxpagesize=<MaxPageSize>` |

### Query option helpers

```go
func Select(fields ...string) QueryOption
func Filter(expr string) QueryOption
func OrderBy(field string, desc bool) QueryOption
func Top(n int) QueryOption
func Skip(n int) QueryOption
func Expand(nav string) QueryOption
```

Apply options to a `url.Values` map when building query URLs:

```go
params := url.Values{}
dataverse.Select("name", "accountid")(params)
dataverse.Filter("statecode eq 0")(params)
url := client.ServiceURL() + "accounts?" + params.Encode()
```

## Dataverse vs standard OData

| Aspect | Standard OData | Dataverse |
|--------|---------------|-----------|
| Auth | varies | OAuth2 (Azure AD) via `BearerToken` |
| Headers | none required | `OData-Version`, `OData-MaxVersion`, `Accept` mandatory |
| Impersonation | not standard | `MSCRMCallerID` header |
| API path | service-defined | `/api/data/v{version}/` |
| Pagination | `$skiptoken` | `odata.maxpagesize` preference |

## Notes / Limitations

- Token acquisition (Azure AD / MSAL) is the caller's responsibility. Pass a `BearerToken` function that calls your MSAL/confidential-client library.
- Only the Dataverse Web API (OData v4) is supported. The legacy 2011 SOAP endpoint is not.
- `MaxPageSize` applies globally; per-request overrides require constructing the `Prefer` header manually.
