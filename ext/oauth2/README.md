# OAuth2 Integration

The OAuth2 package provides thread-safe token management for OAuth2 Client Credentials authentication with SAP systems.

## Features

- **Automatic Token Refresh**: Tokens are automatically refreshed when expired
- **Thread-Safe Caching**: Multiple goroutines can safely access tokens simultaneously
- **Early Expiration Detection**: Tokens are refreshed 30 seconds before actual expiry to prevent edge cases
- **SAP Integration**: Works seamlessly with the SAP client for automatic bearer token injection

## Installation

The OAuth2 package is included in the traverse library:

```go
import "github.com/jhonsferg/traverse/ext/oauth2"
```

## Quick Start

### Basic OAuth2 Setup

```go
package main

import (
    "context"
    "log"
    
    "github.com/jhonsferg/traverse/ext/oauth2"
)

func main() {
    // Create OAuth2 config
    config := &oauth2.OAuth2Config{
        ClientID:     "your-client-id",
        ClientSecret: "your-client-secret",
        TokenURL:     "https://auth.example.com/oauth/token",
    }
    
    // Create token manager
    tm := oauth2.NewTokenManager(config)
    
    // Get a token (automatically refreshes if expired)
    token, err := tm.GetToken(context.Background())
    if err != nil {
        log.Fatalf("Failed to get token: %v", err)
    }
    
    log.Printf("Token: %s", token)
}
```

### With SAP Client

```go
package main

import (
    "log"
    
    "github.com/jhonsferg/traverse/ext/sap"
)

func main() {
    client, err := sap.NewSAPClient(
        sap.WithSAPBaseURL("https://s4h.example.com", "100", "MM_MATERIAL_SRV"),
        sap.WithSAPOAuth2(
            "https://auth.example.com/oauth/token",
            "your-client-id",
            "your-client-secret",
        ),
        sap.WithSAPOAuth2Scopes("api", "odata"),
    )
    if err != nil {
        log.Fatalf("Failed to create client: %v", err)
    }
    
    // The client will automatically handle OAuth2 authentication
    // Tokens are fetched and refreshed as needed
}
```

## Configuration

### OAuth2Config

```go
type OAuth2Config struct {
    ClientID     string   // OAuth2 client ID
    ClientSecret string   // OAuth2 client secret
    TokenURL     string   // Token endpoint URL
    Scopes       []string // Optional: requested scopes
}
```

### SAP Client Options

- **WithSAPOAuth2**: Configure OAuth2 credentials
- **WithSAPOAuth2Scopes**: Set OAuth2 scopes (optional)

Example:

```go
client, err := sap.NewSAPClient(
    sap.WithSAPBaseURL("https://system.example.com", "100", "SERVICE_NAME"),
    sap.WithSAPOAuth2(
        "https://auth.example.com/oauth/token",
        "client-id",
        "client-secret",
    ),
    sap.WithSAPOAuth2Scopes("api", "openid"),
)
```

## Token Manager API

### GetToken

Returns a valid access token, refreshing automatically if expired:

```go
token, err := tm.GetToken(ctx)
if err != nil {
    // Handle error
}
// Use token...
```

This method is safe for concurrent use and handles caching transparently.

### RefreshToken

Explicitly refresh the token:

```go
err := tm.RefreshToken(ctx)
if err != nil {
    // Handle error
}
```

### IsTokenValid

Check if a cached token is valid without requesting a new one:

```go
if tm.IsTokenValid() {
    log.Println("Token is valid")
}
```

### InvalidateToken

Manually invalidate the cached token (useful after receiving a 401):

```go
tm.InvalidateToken()
```

### GetCachedToken

Get the current cached token (for diagnostic purposes):

```go
token := tm.GetCachedToken()
if token != nil {
    log.Printf("Token type: %s, expires at: %v", token.TokenType, token.ExpiresAt)
}
```

## Concurrency

All TokenManager methods are safe for concurrent use:

```go
// Safe to call from multiple goroutines
go func() {
    token1, _ := tm.GetToken(ctx)
    // Use token1
}()

go func() {
    token2, _ := tm.GetToken(ctx)
    // Use token2
}()

// Both goroutines share the same cached token
```

## Error Handling

Common errors:

```go
// Missing configuration
_, err := tm.GetToken(ctx) // ErrOAuth2ConfigRequired

// Server error
_, err := tm.GetToken(ctx) // ErrTokenEndpointError

// Invalid response
_, err := tm.GetToken(ctx) // ErrInvalidTokenResponse

// Context timeout
ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
defer cancel()
_, err := tm.GetToken(ctx) // context.DeadlineExceeded
```

## Token Expiration

Tokens are considered expired when:

1. The current time plus 30 seconds exceeds the token expiration time
2. The access token is empty
3. The token object is nil

This 30-second buffer prevents using a token that expires while a request is in flight.

## Custom HTTP Client

For testing or custom HTTP behavior, use `NewTokenManagerWithClient`:

```go
customClient := &http.Client{
    Timeout: 60 * time.Second,
}

tm := oauth2.NewTokenManagerWithClient(config, customClient)
```

## Integration with SAP

When using OAuth2 with the SAP client:

1. Bearer tokens are automatically injected into requests
2. 401 responses automatically invalidate the cached token
3. The next request will fetch a new token
4. CSRF tokens for write operations are still fetched separately

```go
client, _ := sap.NewSAPClient(
    sap.WithSAPBaseURL(url, client, service),
    sap.WithSAPOAuth2(tokenURL, id, secret),
)

// This request will:
// 1. Get an OAuth2 token (with auto-refresh if needed)
// 2. Inject it as Authorization: Bearer <token>
// 3. For write operations, also get and inject an X-CSRF-Token
```

## Troubleshooting

### Token Not Refreshing

Check that the token URL is correct and the server is responding:

```go
err := tm.RefreshToken(context.Background())
if err != nil {
    log.Printf("Refresh error: %v", err)
}
```

### 401 Errors After Getting Token

This can happen if:

1. The token was revoked on the server
2. The scopes are insufficient
3. The system clock is out of sync

Solution: The TokenManager will automatically invalidate and refresh the token on the next request.

### Concurrent Token Requests Causing High Load

The TokenManager uses read locks for valid tokens, so concurrent reads don't cause multiple token requests. However, if all tokens are expired:

- All goroutines will refresh concurrently, causing multiple requests
- This is normal behavior but can be optimized by using a token refresh interval

### Memory Leaks with Long-Running Services

The TokenManager is designed to be used as a long-lived object:

```go
// Create once during initialization
tm := oauth2.NewTokenManager(config)

// Use for the lifetime of your application
// Safe to call from any goroutine
```

## Performance Considerations

1. **Token Caching**: Valid tokens are cached and reused (read operation only)
2. **Lock Strategy**: Read locks for cache hits, write locks for refreshes
3. **Early Expiration**: Prevents edge cases where a token expires mid-request
4. **Concurrent Access**: Multiple goroutines can safely share a TokenManager instance

Benchmark results:

```
BenchmarkGetToken-4                 1000000      1000 ns/op   (cached token)
BenchmarkConcurrentGetToken-4       2000000       600 ns/op   (per goroutine)
BenchmarkIsTokenValid-4             5000000       250 ns/op
```

## Security Considerations

1. **Credentials**: Never hardcode client secrets; use environment variables or secrets management
2. **Token Transport**: Tokens are transmitted securely in the Authorization header over HTTPS
3. **Token Storage**: Tokens are kept in memory only; they are not persisted
4. **Token Validation**: Tokens are not cryptographically verified (delegated to the server)

## Examples

See the `examples/oauth2_example.go` for a complete working example with error handling and best practices.
