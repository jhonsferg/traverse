package traverse

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/jhonsferg/relay"
)

// ── Relay-backed traverse options ────────────────────────────────────────────
//
// The following options expose relay's advanced HTTP transport features
// directly through the traverse API. They are alternatives to injecting a
// fully pre-configured relay client via [WithRelayClient].
//
// Any option here can be combined with the OData-specific options in client.go.
// All options are ignored if a relay client has already been supplied via
// [WithRelayClient].

// WithRetry configures automatic retry behaviour for transient failures.
//
// Relay will retry requests that fail with a network error or a 429/5xx status
// code. The retry strategy is exponential back-off with jitter.
//
// Example:
//
//	client, _ := traverse.New(
//	    traverse.WithBaseURL("https://odata.example.com/v4"),
//	    traverse.WithRetry(&relay.RetryConfig{
//	        MaxAttempts:     4,
//	        InitialInterval: 200 * time.Millisecond,
//	        MaxInterval:     10 * time.Second,
//	        Multiplier:      2.0,
//	        RandomFactor:    0.3,
//	    }),
//	)
func WithRetry(rc *relay.RetryConfig) Option {
	return func(c *clientConfig) error {
		if rc == nil {
			return errorf("WithRetry: RetryConfig must not be nil")
		}
		c.relayOpts = append(c.relayOpts, relay.WithRetry(rc))
		return nil
	}
}

// WithCircuitBreaker enables the circuit-breaker pattern for all requests.
//
// The circuit breaker opens after [relay.CircuitBreakerConfig.MaxFailures]
// consecutive failures, preventing further requests until the reset timeout
// elapses. This protects downstream OData services from cascading failure.
//
// Example:
//
//	client, _ := traverse.New(
//	    traverse.WithBaseURL("https://odata.example.com/v4"),
//	    traverse.WithCircuitBreaker(&relay.CircuitBreakerConfig{
//	        MaxFailures:      5,
//	        ResetTimeout:     30 * time.Second,
//	        HalfOpenRequests: 2,
//	        OnStateChange: func(from, to relay.CircuitBreakerState) {
//	            log.Printf("circuit breaker: %s → %s", from, to)
//	        },
//	    }),
//	)
func WithCircuitBreaker(cfg *relay.CircuitBreakerConfig) Option {
	return func(c *clientConfig) error {
		if cfg == nil {
			return errorf("WithCircuitBreaker: CircuitBreakerConfig must not be nil")
		}
		c.relayOpts = append(c.relayOpts, relay.WithCircuitBreaker(cfg))
		return nil
	}
}

// WithRateLimit constrains the number of outgoing HTTP requests per second.
//
// rps is the sustained request rate (e.g. 50.0 for 50 req/s).
// burst is the maximum number of requests allowed in a single instant above rps.
//
// This is particularly useful when querying SAP OData services that enforce
// throttling via HTTP 429 responses.
//
// Example:
//
//	// Allow 100 req/s with a burst of up to 110
//	traverse.WithRateLimit(100, 110)
func WithRateLimit(rps float64, burst int) Option {
	return func(c *clientConfig) error {
		if rps <= 0 {
			return errorf("WithRateLimit: rps must be > 0")
		}
		if burst < 1 {
			return errorf("WithRateLimit: burst must be >= 1")
		}
		c.relayOpts = append(c.relayOpts, relay.WithRateLimit(rps, burst))
		return nil
	}
}

// WithProxy sets an HTTP or HTTPS proxy for all outgoing requests.
//
// proxyURL must be a fully qualified URL, e.g. "http://proxy.corp.com:3128" or
// "https://user:pass@proxy.corp.com:8080".
//
// Example:
//
//	traverse.WithProxy("http://proxy.internal.corp:3128")
func WithProxy(proxyURL string) Option {
	return func(c *clientConfig) error {
		if proxyURL == "" {
			return errorf("WithProxy: proxy URL must not be empty")
		}
		c.relayOpts = append(c.relayOpts, relay.WithProxy(proxyURL))
		return nil
	}
}

// WithTLSConfig sets a custom [crypto/tls.Config] for the HTTP transport.
//
// Use this to configure mutual TLS, disable certificate verification for
// on-premise services with self-signed certificates, or pin specific CAs.
//
// Example - disable TLS verification for a local development OData service:
//
//	traverse.WithTLSConfig(&tls.Config{InsecureSkipVerify: true}) // #nosec G402
func WithTLSConfig(cfg *tls.Config) Option {
	return func(c *clientConfig) error {
		if cfg == nil {
			return errorf("WithTLSConfig: tls.Config must not be nil")
		}
		c.relayOpts = append(c.relayOpts, relay.WithTLSConfig(cfg))
		return nil
	}
}

// WithCookieJar sets a custom [net/http.CookieJar] for session-based authentication.
//
// Many OData services (particularly SAP) rely on session cookies after an initial
// CSRF token handshake. Providing a cookie jar allows traverse to maintain and
// reuse those session cookies across requests automatically.
//
// Example - use the standard library cookie jar:
//
//	jar, _ := cookiejar.New(nil)
//	traverse.WithCookieJar(jar)
func WithCookieJar(jar http.CookieJar) Option {
	return func(c *clientConfig) error {
		if jar == nil {
			return errorf("WithCookieJar: CookieJar must not be nil")
		}
		c.relayOpts = append(c.relayOpts, relay.WithCookieJar(jar))
		return nil
	}
}

// WithRequestHook registers a low-level HTTP hook called before every request.
//
// The hook receives the underlying [relay.Request] and may modify headers,
// inject tracing metadata, or cancel the context. It is called after OData-level
// hooks ([WithBeforeQuery]) but before the request is sent over the wire.
//
// To register OData-level hooks (called before query building), use
// [WithBeforeQuery] instead.
//
// Example - add a request-ID header to every request:
//
//	traverse.WithRequestHook(func(ctx context.Context, r *relay.Request) error {
//	    r.WithHeader("X-Request-ID", uuid.New().String())
//	    return nil
//	})
func WithRequestHook(fn func(context.Context, *relay.Request) error) Option {
	return func(c *clientConfig) error {
		if fn == nil {
			return errorf("WithRequestHook: hook function must not be nil")
		}
		c.relayOpts = append(c.relayOpts, relay.WithOnBeforeRequest(fn))
		return nil
	}
}

// WithResponseHook registers a low-level HTTP hook called after every response.
//
// The hook receives the underlying [relay.Response] and may inspect headers,
// record metrics, or return an error to abort further processing. It is called
// before OData-level hooks ([WithAfterExecute]).
//
// Example - record response latency to a metrics system:
//
//	traverse.WithResponseHook(func(ctx context.Context, r *relay.Response) error {
//	    metrics.Record("odata.latency", r.Timing.Total)
//	    return nil
//	})
func WithResponseHook(fn func(context.Context, *relay.Response) error) Option {
	return func(c *clientConfig) error {
		if fn == nil {
			return errorf("WithResponseHook: hook function must not be nil")
		}
		c.relayOpts = append(c.relayOpts, relay.WithOnAfterResponse(fn))
		return nil
	}
}

// WithSigner sets a [relay.RequestSigner] that signs every outgoing HTTP request.
//
// This is useful for OData services that require request signing such as
// AWS API Gateway (SigV4) or custom HMAC schemes.
//
// Example - sign requests with a custom HMAC scheme:
//
//	traverse.WithSigner(relay.RequestSignerFunc(func(r *http.Request) error {
//	    mac := hmac.New(sha256.New, secretKey)
//	    mac.Write([]byte(r.Method + r.URL.Path))
//	    r.Header.Set("X-Signature", hex.EncodeToString(mac.Sum(nil)))
//	    return nil
//	}))
func WithSigner(s relay.RequestSigner) Option {
	return func(c *clientConfig) error {
		if s == nil {
			return errorf("WithSigner: RequestSigner must not be nil")
		}
		c.relayOpts = append(c.relayOpts, relay.WithSigner(s))
		return nil
	}
}

// WithConnectTimeout sets the TCP/TLS connection timeout independently of the
// overall transfer timeout set by [WithTimeout].
//
// Use [WithTimeout] to limit the total time for a request (including reading
// the response body). Use WithConnectTimeout to limit only the time spent
// establishing the connection and reading response headers.
//
// The default is 30 seconds. Setting 0 disables the connection timeout.
//
// Example:
//
//	// 5-second connection deadline, no overall transfer limit
//	traverse.WithConnectTimeout(5 * time.Second)
func WithConnectTimeout(d time.Duration) Option {
	return func(c *clientConfig) error {
		c.relayOpts = append(c.relayOpts,
			relay.WithDialTimeout(d),
			relay.WithResponseHeaderTimeout(d),
		)
		return nil
	}
}

// WithMaxRedirects sets the maximum number of HTTP redirects to follow.
//
// The default is 10. Set to 0 to disable redirect following entirely.
//
// Example:
//
//	traverse.WithMaxRedirects(0) // never follow redirects
func WithMaxRedirects(n int) Option {
	return func(c *clientConfig) error {
		if n < 0 {
			return errorf("WithMaxRedirects: value must be >= 0")
		}
		c.relayOpts = append(c.relayOpts, relay.WithMaxRedirects(n))
		return nil
	}
}

// WithHTTPOption passes a raw [relay.Option] through to the underlying relay client.
//
// Use this to inject low-level relay transport options that do not have a
// dedicated traverse wrapper - for example, audit trail or custom transport
// middleware provided by a third-party extension.
//
// Example:
//
//	import "github.com/jhonsferg/traverse/ext/audit"
//
//	logger := audit.AuditLoggerFunc(func(ctx context.Context, e audit.AuditEntry) {
//	    log.Printf("[AUDIT] %s %s %d", e.Operation, e.EntitySet, e.StatusCode)
//	})
//	client, _ := traverse.New(url,
//	    traverse.WithHTTPOption(audit.WithAuditTrail(logger)),
//	)
func WithHTTPOption(opt relay.Option) Option {
	return func(c *clientConfig) error {
		if opt == nil {
			return errorf("WithHTTPOption: option must not be nil")
		}
		c.relayOpts = append(c.relayOpts, opt)
		return nil
	}
}

// WithODataErrors configures the client to decode OData error responses into
// structured [ODataError] values.
//
// When enabled, any 4xx or 5xx response whose body follows the OData v2 or v4
// error format is decoded into an [ODataError] instead of a generic HTTP error.
// Use [IsODataError] to inspect the result.
//
// This option is recommended for all production clients as it surfaces the
// OData error code and message directly from the service.
//
// Example:
//
//	client, _ := traverse.New(
//	    traverse.WithBaseURL("https://odata.example.com/v4"),
//	    traverse.WithODataErrors(),
//	)
//
//	_, err := client.From("InvalidEntity").First(ctx)
//	if odErr, ok := traverse.IsODataError(err); ok {
//	    fmt.Println("OData error code:", odErr.Code)
//	}
func WithODataErrors() Option {
	return func(c *clientConfig) error {
		c.relayOpts = append(c.relayOpts, relay.WithErrorDecoder(odataErrorDecoder))
		return nil
	}
}

// odataErrorDecoder attempts to parse OData v4 and v2 error response bodies.
// Returns nil if the body does not match either format, letting relay fall
// back to its default error handling.
func odataErrorDecoder(statusCode int, body []byte) error {
	if len(body) == 0 {
		return nil
	}

	// OData v4: {"error":{"code":"...","message":"..."}}
	var v4 struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
			Target  string `json:"target,omitempty"`
			Details []struct {
				Code    string `json:"code"`
				Message string `json:"message"`
				Target  string `json:"target,omitempty"`
			} `json:"details,omitempty"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &v4); err == nil && (v4.Error.Code != "" || v4.Error.Message != "") {
		oderr := &ODataError{
			Code:    v4.Error.Code,
			Message: v4.Error.Message,
			Target:  v4.Error.Target,
		}
		for _, d := range v4.Error.Details {
			oderr.Details = append(oderr.Details, ODataErrorDetail{
				Code:    d.Code,
				Message: d.Message,
				Target:  d.Target,
			})
		}
		return oderr
	}

	// OData v2: {"error":{"code":"...","message":{"lang":"en","value":"..."}}}
	var v2 struct {
		Error struct {
			Code    string `json:"code"`
			Message struct {
				Value string `json:"value"`
			} `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &v2); err == nil && v2.Error.Code != "" {
		return &ODataError{
			Code:    v2.Error.Code,
			Message: v2.Error.Message.Value,
		}
	}

	return nil
}

// errorf is a helper that returns a formatted traverse option error.
func errorf(format string, args ...any) error {
	return fmt.Errorf("traverse: "+format, args...)
}
