package traverse

import (
	"context"
	"crypto/tls"
	"net/http"
	"net/http/cookiejar"
	"testing"
	"time"

	"github.com/jhonsferg/relay"
)

// ── WithRetry ─────────────────────────────────────────────────────────────────

func TestWithRetry_Valid(t *testing.T) {
	_, err := New(
		WithBaseURL("http://example.com"),
		WithRetry(&relay.RetryConfig{
			MaxAttempts:     3,
			InitialInterval: 50 * time.Millisecond,
		}),
	)
	if err != nil {
		t.Fatalf("WithRetry with valid config: %v", err)
	}
}

func TestWithRetry_Nil(t *testing.T) {
	_, err := New(
		WithBaseURL("http://example.com"),
		WithRetry(nil),
	)
	if err == nil {
		t.Fatal("WithRetry(nil) should return error")
	}
}

// ── WithCircuitBreaker ────────────────────────────────────────────────────────

func TestWithCircuitBreaker_Valid(t *testing.T) {
	stateChanges := 0
	_, err := New(
		WithBaseURL("http://example.com"),
		WithCircuitBreaker(&relay.CircuitBreakerConfig{
			MaxFailures:  3,
			ResetTimeout: 5 * time.Second,
			OnStateChange: func(from, to relay.CircuitBreakerState) {
				stateChanges++
			},
		}),
	)
	if err != nil {
		t.Fatalf("WithCircuitBreaker with valid config: %v", err)
	}
}

func TestWithCircuitBreaker_Nil(t *testing.T) {
	_, err := New(
		WithBaseURL("http://example.com"),
		WithCircuitBreaker(nil),
	)
	if err == nil {
		t.Fatal("WithCircuitBreaker(nil) should return error")
	}
}

// ── WithRateLimit ─────────────────────────────────────────────────────────────

func TestWithRateLimit_Valid(t *testing.T) {
	_, err := New(
		WithBaseURL("http://example.com"),
		WithRateLimit(100.0, 110),
	)
	if err != nil {
		t.Fatalf("WithRateLimit with valid params: %v", err)
	}
}

func TestWithRateLimit_ZeroRPS(t *testing.T) {
	_, err := New(
		WithBaseURL("http://example.com"),
		WithRateLimit(0, 1),
	)
	if err == nil {
		t.Fatal("WithRateLimit(0, 1) should return error")
	}
}

func TestWithRateLimit_ZeroBurst(t *testing.T) {
	_, err := New(
		WithBaseURL("http://example.com"),
		WithRateLimit(10.0, 0),
	)
	if err == nil {
		t.Fatal("WithRateLimit(10, 0) should return error")
	}
}

// ── WithProxy ─────────────────────────────────────────────────────────────────

func TestWithProxy_Valid(t *testing.T) {
	_, err := New(
		WithBaseURL("http://example.com"),
		WithProxy("http://proxy.corp.com:3128"),
	)
	if err != nil {
		t.Fatalf("WithProxy with valid URL: %v", err)
	}
}

func TestWithProxy_Empty(t *testing.T) {
	_, err := New(
		WithBaseURL("http://example.com"),
		WithProxy(""),
	)
	if err == nil {
		t.Fatal("WithProxy(\"\") should return error")
	}
}

// ── WithTLSConfig ─────────────────────────────────────────────────────────────

func TestWithTLSConfig_Valid(t *testing.T) {
	_, err := New(
		WithBaseURL("http://example.com"),
		WithTLSConfig(&tls.Config{MinVersion: tls.VersionTLS12}),
	)
	if err != nil {
		t.Fatalf("WithTLSConfig with valid config: %v", err)
	}
}

func TestWithTLSConfig_Nil(t *testing.T) {
	_, err := New(
		WithBaseURL("http://example.com"),
		WithTLSConfig(nil),
	)
	if err == nil {
		t.Fatal("WithTLSConfig(nil) should return error")
	}
}

// ── WithCookieJar ─────────────────────────────────────────────────────────────

func TestWithCookieJar_Valid(t *testing.T) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = New(
		WithBaseURL("http://example.com"),
		WithCookieJar(jar),
	)
	if err != nil {
		t.Fatalf("WithCookieJar with valid jar: %v", err)
	}
}

func TestWithCookieJar_Nil(t *testing.T) {
	_, err := New(
		WithBaseURL("http://example.com"),
		WithCookieJar(nil),
	)
	if err == nil {
		t.Fatal("WithCookieJar(nil) should return error")
	}
}

// ── WithRequestHook ───────────────────────────────────────────────────────────

func TestWithRequestHook_Valid(t *testing.T) {
	called := false
	_, err := New(
		WithBaseURL("http://example.com"),
		WithRequestHook(func(_ context.Context, r *relay.Request) error {
			called = true
			return nil
		}),
	)
	if err != nil {
		t.Fatalf("WithRequestHook with valid func: %v", err)
	}
	_ = called
}

func TestWithRequestHook_Nil(t *testing.T) {
	_, err := New(
		WithBaseURL("http://example.com"),
		WithRequestHook(nil),
	)
	if err == nil {
		t.Fatal("WithRequestHook(nil) should return error")
	}
}

// ── WithResponseHook ──────────────────────────────────────────────────────────

func TestWithResponseHook_Valid(t *testing.T) {
	called := false
	_, err := New(
		WithBaseURL("http://example.com"),
		WithResponseHook(func(_ context.Context, r *relay.Response) error {
			called = true
			return nil
		}),
	)
	if err != nil {
		t.Fatalf("WithResponseHook with valid func: %v", err)
	}
	_ = called
}

func TestWithResponseHook_Nil(t *testing.T) {
	_, err := New(
		WithBaseURL("http://example.com"),
		WithResponseHook(nil),
	)
	if err == nil {
		t.Fatal("WithResponseHook(nil) should return error")
	}
}

// ── WithSigner ────────────────────────────────────────────────────────────────

func TestWithSigner_Valid(t *testing.T) {
	signer := relay.RequestSignerFunc(func(r *http.Request) error {
		r.Header.Set("X-Signature", "test")
		return nil
	})
	_, err := New(
		WithBaseURL("http://example.com"),
		WithSigner(signer),
	)
	if err != nil {
		t.Fatalf("WithSigner with valid signer: %v", err)
	}
}

func TestWithSigner_Nil(t *testing.T) {
	_, err := New(
		WithBaseURL("http://example.com"),
		WithSigner(nil),
	)
	if err == nil {
		t.Fatal("WithSigner(nil) should return error")
	}
}

// ── WithConnectTimeout ────────────────────────────────────────────────────────

func TestWithConnectTimeout_Valid(t *testing.T) {
	_, err := New(
		WithBaseURL("http://example.com"),
		WithConnectTimeout(5*time.Second),
	)
	if err != nil {
		t.Fatalf("WithConnectTimeout with valid duration: %v", err)
	}
}

func TestWithConnectTimeout_Zero(t *testing.T) {
	// Zero is allowed — disables the connection timeout.
	_, err := New(
		WithBaseURL("http://example.com"),
		WithConnectTimeout(0),
	)
	if err != nil {
		t.Fatalf("WithConnectTimeout(0) should be valid: %v", err)
	}
}

// ── WithMaxRedirects ──────────────────────────────────────────────────────────

func TestWithMaxRedirects_Valid(t *testing.T) {
	_, err := New(
		WithBaseURL("http://example.com"),
		WithMaxRedirects(5),
	)
	if err != nil {
		t.Fatalf("WithMaxRedirects(5): %v", err)
	}
}

func TestWithMaxRedirects_Zero(t *testing.T) {
	// Zero disables redirects — valid.
	_, err := New(
		WithBaseURL("http://example.com"),
		WithMaxRedirects(0),
	)
	if err != nil {
		t.Fatalf("WithMaxRedirects(0) should be valid: %v", err)
	}
}

func TestWithMaxRedirects_Negative(t *testing.T) {
	_, err := New(
		WithBaseURL("http://example.com"),
		WithMaxRedirects(-1),
	)
	if err == nil {
		t.Fatal("WithMaxRedirects(-1) should return error")
	}
}

// ── WithODataErrors ───────────────────────────────────────────────────────────

func TestWithODataErrors_Valid(t *testing.T) {
	_, err := New(
		WithBaseURL("http://example.com"),
		WithODataErrors(),
	)
	if err != nil {
		t.Fatalf("WithODataErrors(): %v", err)
	}
}

func TestODataErrorDecoder_V4(t *testing.T) {
	body := []byte(`{"error":{"code":"SY/530","message":"Entity set not found","target":"Products"}}`)
	err := odataErrorDecoder(404, body)
	if err == nil {
		t.Fatal("expected ODataError, got nil")
	}
	odErr, ok := err.(*ODataError)
	if !ok {
		t.Fatalf("expected *ODataError, got %T", err)
	}
	if odErr.Code != "SY/530" {
		t.Errorf("code: want SY/530, got %s", odErr.Code)
	}
	if odErr.Message != "Entity set not found" {
		t.Errorf("message: want 'Entity set not found', got %s", odErr.Message)
	}
	if odErr.Target != "Products" {
		t.Errorf("target: want Products, got %s", odErr.Target)
	}
}

func TestODataErrorDecoder_V2(t *testing.T) {
	body := []byte(`{"error":{"code":"MERR/001","message":{"lang":"en","value":"Invalid key predicate"}}}`)
	err := odataErrorDecoder(400, body)
	if err == nil {
		t.Fatal("expected ODataError, got nil")
	}
	odErr, ok := err.(*ODataError)
	if !ok {
		t.Fatalf("expected *ODataError, got %T", err)
	}
	if odErr.Code != "MERR/001" {
		t.Errorf("code: want MERR/001, got %s", odErr.Code)
	}
	if odErr.Message != "Invalid key predicate" {
		t.Errorf("message: want 'Invalid key predicate', got %s", odErr.Message)
	}
}

func TestODataErrorDecoder_EmptyBody(t *testing.T) {
	err := odataErrorDecoder(500, []byte{})
	if err != nil {
		t.Fatalf("empty body should return nil, got %v", err)
	}
}

func TestODataErrorDecoder_NonODataBody(t *testing.T) {
	err := odataErrorDecoder(500, []byte(`<html><body>Internal Server Error</body></html>`))
	if err != nil {
		t.Fatalf("non-OData body should return nil, got %v", err)
	}
}

// ── CircuitBreakerState method ────────────────────────────────────────────────

func TestCircuitBreakerState_DefaultClosed(t *testing.T) {
	client, err := New(WithBaseURL("http://example.com"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = client.Close() }()

	if client.CircuitBreakerState() != relay.StateClosed {
		t.Errorf("default circuit breaker state should be Closed")
	}
}

func TestResetCircuitBreaker_NoOp(t *testing.T) {
	client, err := New(WithBaseURL("http://example.com"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = client.Close() }()

	// Should not panic even without a custom CB config.
	client.ResetCircuitBreaker()
}
