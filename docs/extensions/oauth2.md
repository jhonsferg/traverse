# OAuth2 Extension

The OAuth2 extension (`ext/oauth2`) provides automatic token acquisition and refresh for OData services protected by OAuth2.

## Installation

```bash
go get github.com/jhonsferg/traverse/ext/oauth2@latest
```

## Client credentials flow

```go
import (
    "github.com/jhonsferg/traverse"
    "github.com/jhonsferg/traverse/ext/oauth2"
)

client := traverse.New(traverse.Config{
    BaseURL: "https://api.example.com/odata/",
    Extension: oauth2.Extension(oauth2.Config{
        TokenURL:     "https://auth.example.com/oauth/token",
        ClientID:     os.Getenv("CLIENT_ID"),
        ClientSecret: os.Getenv("CLIENT_SECRET"),
        Scopes:       []string{"api.read", "api.write"},
    }),
})
```

## Password flow

```go
oauth2.Extension(oauth2.Config{
    TokenURL:  "https://auth.example.com/token",
    ClientID:  "my-client",
    Username:  os.Getenv("API_USER"),
    Password:  os.Getenv("API_PASS"),
    GrantType: oauth2.PasswordGrant,
})
```

## Token caching and refresh

The extension caches the access token and refreshes it automatically before it expires (with a 30-second buffer by default):

```go
oauth2.Extension(oauth2.Config{
    TokenURL:       "https://...",
    ClientID:       "id",
    ClientSecret:   "secret",
    RefreshBuffer:  60 * time.Second, // refresh 60s before expiry
})
```

## On-behalf-of / bearer passthrough

Pass a user-supplied bearer token directly (no token fetch):

```go
oauth2.Extension(oauth2.Config{
    StaticToken: userBearerToken,
})
```

## See also

- [Extensions Overview](index.md)
- [SAP Extension](sap.md)
