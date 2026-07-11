package sap

import (
	"context"
	"fmt"
	"io"
	"strings"
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

	// lastMethod stores the HTTP method of the most recent request seen by
	// WithWriteMethodDetection, so HandleResponseForWriteOps can determine
	// whether the request was a write operation without relying on context
	// (relay.WithOnBeforeRequest hooks cannot return a modified context).
	lastMethod string
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
	defer relay.PutResponse(resp)

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

// HandleResponse processes responses and invalidates CSRF token only if it's a genuine CSRF error.
// This method should be used as a relay hook via WithOnAfterResponse.
//
// A CSRF error is only considered genuine if:
// 1. HTTP Status is 403 Forbidden
// 2. The response body contains CSRF-specific language (e.g., "csrf", "token", "validation")
//
// This prevents false positives where SAP returns 403 for other reasons (e.g., authorization,
// service not found) that have nothing to do with CSRF token validity.
func (c *CSRFMiddleware) HandleResponse(ctx context.Context, resp *relay.Response, err error) error {
	// Only handle 403 responses
	if resp == nil || resp.StatusCode != 403 {
		return err
	}

	// Read response body to check for CSRF-specific errors
	bodyBytes, readErr := readResponseBody(resp)
	if readErr != nil || len(bodyBytes) == 0 {
		// If we can't read the body or it's empty, assume it might be CSRF and refresh.
		// This is a safe precaution for SAP systems.
		c.InvalidateToken()
		return fmt.Errorf("traverse: 403 Forbidden received (token may be invalid) - token invalidated")
	}

	// Check if the body contains CSRF-specific language
	if isCsrfError(string(bodyBytes)) {
		c.InvalidateToken()
		return fmt.Errorf("traverse: CSRF token validation failed (403 Forbidden) - token invalidated, fresh token will be obtained on retry")
	}

	// This is a 403 but not CSRF-related - propagate the original error
	// This allows SAP business errors to surface correctly
	if err != nil {
		return err
	}

	return fmt.Errorf("traverse: request rejected with 403 Forbidden (not CSRF-related)")
}

// readResponseBody safely reads response body bytes
func readResponseBody(resp *relay.Response) ([]byte, error) {
	if resp == nil {
		return nil, fmt.Errorf("response is nil")
	}

	// Try to read from BodyReader
	reader := resp.BodyReader()
	if reader == nil {
		return nil, fmt.Errorf("body reader is nil")
	}

	bodyBytes, err := io.ReadAll(reader)
	return bodyBytes, err
}

// isCsrfError checks if the response body contains CSRF-related error messages
func isCsrfError(body string) bool {
	// Look for common CSRF error markers in SAP responses
	lowerBody := strings.ToLower(body)

	csrfMarkers := []string{
		"csrf",
		"token validation",
		"token invalid",
		"token expired",
		"x-csrf-token",
	}

	for _, marker := range csrfMarkers {
		if strings.Contains(lowerBody, marker) {
			return true
		}
	}

	return false
}

// ExecuteWithCSRFRetry executes a function that returns a relay.Response,
// and automatically retries once if it receives a 403 CSRF error.
//
// This method implements transparent CSRF token refresh + retry:
// 1. Call the provided function
// 2. If 403 + CSRF error is detected:
//    a. Invalidate current token
//    b. Fetch fresh token
//    c. Re-execute the function
// 3. Return result (success or final error)
//
// This eliminates boilerplate retry logic from consumers and makes CSRF
// token management completely transparent.
//
// Example usage in a middleware or client decorator:
//
//	result, err := csrfMiddleware.ExecuteWithCSRFRetry(ctx, func() (*relay.Response, error) {
//		return httpClient.Execute(request)
//	})
//
// The callback function is responsible for constructing and executing the HTTP request.
// Token injection should still happen via the Hook() method (as a relay WithOnBeforeRequest hook).
func (c *CSRFMiddleware) ExecuteWithCSRFRetry(ctx context.Context, fn func() (*relay.Response, error)) (*relay.Response, error) {
	// First attempt
	resp, err := fn()
	if err != nil {
		return resp, err
	}

	// Check if this is a genuine CSRF error
	if resp.StatusCode == 403 {
		// Check response body for CSRF markers
		bodyBytes, readErr := readResponseBody(resp)
		var bodyStr string
		if readErr == nil {
			bodyStr = string(bodyBytes)
		}

		if isCsrfError(bodyStr) {
			// This is a genuine CSRF error - refresh token and retry
			c.InvalidateToken()

			// Fetch fresh token
			_, getErr := c.GetToken(ctx)
			if getErr != nil {
				return resp, fmt.Errorf("traverse: CSRF token refresh failed after 403: %w", getErr)
			}

			// Retry the operation with fresh token
			retryResp, retryErr := fn()
			return retryResp, retryErr
		}
	}

	return resp, err
}

// WithWriteMethodDetection returns a relay hook that records the HTTP method
// of write-operation requests so that HandleResponseForWriteOps can apply
// CSRF logic selectively.
//
// Write operations (POST, PATCH, PUT, DELETE) may have CSRF requirements.
// Read operations (GET, HEAD, OPTIONS) never require CSRF handling.
//
// This hook should be registered via relay.WithOnBeforeRequest(...) on the
// relay client. It stores the method in the middleware so that
// HandleResponseForWriteOps can check it later.
//
// Example:
//
//	relay.WithOnBeforeRequest(csrfMiddleware.WithWriteMethodDetection())
//	relay.WithOnAfterResponse(csrfMiddleware.HandleResponseForWriteOps())
func (c *CSRFMiddleware) WithWriteMethodDetection() func(context.Context, *relay.Request) error {
	return func(ctx context.Context, req *relay.Request) error {
		c.mu.Lock()
		c.lastMethod = req.Method()
		c.mu.Unlock()
		return nil
	}
}

// HandleResponseForWriteOps is a method-aware version of HandleResponse that only
// applies CSRF logic to write operations (POST, PATCH, PUT, DELETE).
//
// This prevents false positives where SAP returns 403 for non-CSRF reasons
// (e.g., authorization, service not found) on read operations.
//
// Usage:
//
//	relay.WithOnAfterResponse(csrfMiddleware.HandleResponseForWriteOps())
func (c *CSRFMiddleware) HandleResponseForWriteOps() func(context.Context, *relay.Response, error) error {
	return func(ctx context.Context, resp *relay.Response, err error) error {
		c.mu.RLock()
		method := c.lastMethod
		c.mu.RUnlock()

		isWrite := method == "POST" || method == "PATCH" || method == "PUT" || method == "DELETE"
		if !isWrite {
			return err
		}

		return c.HandleResponse(ctx, resp, err)
	}
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
