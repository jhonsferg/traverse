package offline

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
)

// OfflineMiddleware returns a transport middleware that:
//  1. Tries the real request normally.
//  2. On success (2xx): persists the response body to the store under the request path.
//  3. On network error (not HTTP error): serves the cached response if available.
//  4. On cache miss and network error: returns the network error.
//
// The returned function is compatible with relay.WithTransportMiddleware.
func OfflineMiddleware(store *Store) func(http.RoundTripper) http.RoundTripper {
	return func(next http.RoundTripper) http.RoundTripper {
		return roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			path := req.URL.Path

			resp, err := next.RoundTrip(req)
			if err != nil {
				// Network error  -  try the cache.
				cached, cacheErr := store.Get(path)
				if cacheErr != nil {
					// Cache miss: return original network error.
					return nil, err
				}
				return cachedResponse(req, cached), nil
			}

			// Persist successful (2xx) responses to the store.
			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				body, readErr := io.ReadAll(resp.Body)
				resp.Body.Close() //nolint:errcheck
				if readErr != nil {
					// Body could not be fully read; propagate the error rather than
					// silently returning a partial or empty body to the caller.
					return nil, fmt.Errorf("offline: buffer response body: %w", readErr)
				}
				_ = store.Set(path, body)
				resp.Body = io.NopCloser(bytes.NewReader(body))
			}

			return resp, nil
		})
	}
}

// roundTripperFunc adapts a function to http.RoundTripper.
type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

// cachedResponse constructs a synthetic 200 response from cached bytes.
func cachedResponse(req *http.Request, body []byte) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Status:     "200 OK (cached)",
		Header:     http.Header{"Content-Type": {"application/json"}},
		Body:       io.NopCloser(bytes.NewReader(body)),
		Request:    req,
	}
}
