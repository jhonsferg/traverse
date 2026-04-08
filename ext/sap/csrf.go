package sap

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/jhonsferg/relay"
)

// CSRFMiddleware manages the SAP CSRF token lifecycle.
// SAP systems require an X-CSRF-Token header for all write operations (POST, PATCH, PUT, DELETE).
// This middleware:
// - Fetches tokens on-demand with X-CSRF-Token: Fetch header
// - Caches tokens with ~30-minute expiry
// - Auto-injects tokens on write operations
// - Handles 403 errors by refreshing the token
// - Is thread-safe via sync.RWMutex + fetchMu to prevent thundering herd
type CSRFMiddleware struct {
	client  *relay.Client
	baseURL string

	mu        sync.RWMutex
	token     string
	expiresAt time.Time

	// fetchMu serialises concurrent token fetch attempts (thundering herd
	// prevention). After acquiring fetchMu, callers re-check validity under
	// mu.RLock before actually calling Fetch so that only one network round-trip
	// is made even when many goroutines simultaneously observe an expired token.
	fetchMu sync.Mutex
}

// NewCSRFMiddleware creates a new CSRF token middleware.
// The relay.Client is used for all HTTP operations.
// fetchEndpoint is the URL (absolute or relay-relative) used to fetch the CSRF
// token via GET + "X-CSRF-Token: Fetch". Pass the OData service root URL or a
// specific path known to return the token. If empty, "/$metadata" is used as a
// safe default that works on all OData services.
func NewCSRFMiddleware(client *relay.Client, fetchEndpoint string) *CSRFMiddleware {
	return &CSRFMiddleware{
		client:  client,
		baseURL: fetchEndpoint,
	}
}

// Fetch obtains a new CSRF token from the server.
// Sends a GET request with X-CSRF-Token: Fetch header and extracts the token from response.
// SAP returns the token in the X-CSRF-Token response header.
// The request is sent to the endpoint provided to NewCSRFMiddleware; if none
// was given the relative path "/$metadata" is used as a safe default available
// on every OData service.
func (c *CSRFMiddleware) Fetch(ctx context.Context) error {
	endpoint := c.baseURL
	if endpoint == "" {
		endpoint = "/$metadata"
	}
	req := c.client.Get(endpoint)
	req = req.WithHeader("X-CSRF-Token", "Fetch")
	req = req.WithContext(ctx)

	resp, err := c.client.Execute(req)
	if err != nil {
		return fmt.Errorf("traverse: csrf fetch failed: %w", err)
	}

	// Extract token from X-CSRF-Token response header
	token := resp.Headers.Get("X-CSRF-Token")
	if token == "" {
		// Some systems might return in different case
		token = resp.Headers.Get("x-csrf-token")
	}

	if token == "" {
		return fmt.Errorf("traverse: csrf token not returned by server")
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.token = token
	// SAP tokens typically expire in 30 minutes
	c.expiresAt = time.Now().Add(30 * time.Minute)

	return nil
}

// GetToken returns the current token, fetching a new one if expired.
// This is the main API for obtaining a token - it handles refresh logic.
// Concurrent callers that simultaneously observe an expired token are
// serialised by fetchMu so only one token fetch round-trip is made.
func (c *CSRFMiddleware) GetToken(ctx context.Context) (string, error) {
	// Fast path: valid token already cached.
	c.mu.RLock()
	if c.token != "" && time.Now().Before(c.expiresAt) {
		t := c.token
		c.mu.RUnlock()
		return t, nil
	}
	c.mu.RUnlock()

	// Slow path: token missing or expired. Serialise fetch attempts so that
	// only one goroutine makes the network call even under high concurrency
	// (double-checked locking pattern).
	c.fetchMu.Lock()
	defer c.fetchMu.Unlock()

	// Re-check under read lock: another goroutine may have fetched while we
	// waited for fetchMu.
	c.mu.RLock()
	if c.token != "" && time.Now().Before(c.expiresAt) {
		t := c.token
		c.mu.RUnlock()
		return t, nil
	}
	c.mu.RUnlock()

	// Token still missing or expired - fetch a new one.
	if err := c.Fetch(ctx); err != nil {
		return "", err
	}

	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.token, nil
}

// InvalidateToken marks the current token as invalid.
// Called when a 403 error is received (token expired on server).
func (c *CSRFMiddleware) InvalidateToken() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.token = ""
	c.expiresAt = time.Now()
}

// Inject injects the CSRF token into a request if needed.
// Only injects for write operations (POST, PATCH, PUT, DELETE).
// For GET requests, no token is needed.
//
// Note: this method's signature (returns *relay.Request) does not match the
// relay.WithOnBeforeRequest hook type. Use Hook() to obtain a ready-to-use
// relay hook function instead.
func (c *CSRFMiddleware) Inject(ctx context.Context, req *relay.Request) (*relay.Request, error) {
	// Only inject for write operations
	method := req.Method()
	if method != "POST" && method != "PATCH" && method != "PUT" && method != "DELETE" {
		return req, nil
	}

	// Get a valid token
	token, err := c.GetToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("traverse: failed to get csrf token: %w", err)
	}

	// Inject token header
	req = req.WithHeader("X-CSRF-Token", token)
	return req, nil
}

// Hook returns a relay.WithOnBeforeRequest-compatible hook function that
// injects the CSRF token into write requests. Use this when registering
// the middleware with a relay client:
//
//	relay.WithOnBeforeRequest(csrfMiddleware.Hook())
func (c *CSRFMiddleware) Hook() func(context.Context, *relay.Request) error {
	return func(ctx context.Context, req *relay.Request) error {
		method := req.Method()
		if method != "POST" && method != "PATCH" && method != "PUT" && method != "DELETE" {
			return nil
		}
		token, err := c.GetToken(ctx)
		if err != nil {
			return fmt.Errorf("traverse: failed to get csrf token: %w", err)
		}
		req.WithHeader("X-CSRF-Token", token)
		return nil
	}
}

// HandleResponse processes responses for CSRF-related errors.
// If a 403 error is received, the token is invalidated (likely expired on server).
// The caller should retry the request after invalidation.
// This method can be used as a relay hook via WithOnAfterResponse.
func (c *CSRFMiddleware) HandleResponse(ctx context.Context, resp *relay.Response, err error) error {
	// Only handle successful responses that have CSRF-related errors
	if resp == nil {
		return err
	}

	// Status 403 Forbidden often means CSRF token is invalid/expired
	if resp.StatusCode == 403 {
		// Check if it's a CSRF error (look for token-related message)
		// This is optional - we invalidate on any 403 as precaution
		c.InvalidateToken()

		// Return a meaningful error that indicates retry might help
		return fmt.Errorf("traverse: csrf token invalid (403 Forbidden), token invalidated - retry with new token")
	}

	return err
}

// Token returns the current cached token without checking expiry.
// Used primarily for testing or diagnostic purposes.
// For production code, use GetToken() instead.
func (c *CSRFMiddleware) Token() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.token
}

// IsValid checks if the current token is still valid.
// Returns true only if a token exists and hasn't expired.
func (c *CSRFMiddleware) IsValid() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.token != "" && time.Now().Before(c.expiresAt)
}
