package sap

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jhonsferg/relay"
)

// --- Inject ---

func TestCSRFMiddleware_Inject_ReadOperation(t *testing.T) {
	relayClient := relay.New(relay.WithBaseURL("http://example.com"))
	csrf := NewCSRFMiddleware(relayClient, "/$metadata")

	req := relayClient.Get("/Products")
	got, err := csrf.Inject(context.Background(), req)
	if err != nil {
		t.Fatalf("Inject() error = %v", err)
	}
	if got != req {
		t.Error("expected Inject() to return the same request unmodified for GET")
	}
}

func TestCSRFMiddleware_Inject_WriteOperation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-CSRF-Token", "fresh-token")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	relayClient := relay.New(relay.WithBaseURL(server.URL))
	csrf := NewCSRFMiddleware(relayClient, "/$metadata")

	req := relayClient.Post("/Products")
	got, err := csrf.Inject(context.Background(), req)
	if err != nil {
		t.Fatalf("Inject() error = %v", err)
	}
	if got == nil {
		t.Fatal("Inject() returned nil request")
	}
}

func TestCSRFMiddleware_Inject_TokenFetchError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	relayClient := relay.New(relay.WithBaseURL(server.URL))
	csrf := NewCSRFMiddleware(relayClient, "/$metadata")

	_, err := csrf.Inject(context.Background(), relayClient.Post("/Products"))
	if err == nil {
		t.Fatal("Inject() expected error when token fetch fails, got nil")
	}
}

// --- Hook ---

func TestCSRFMiddleware_Hook_ReadOperation(t *testing.T) {
	relayClient := relay.New(relay.WithBaseURL("http://example.com"))
	csrf := NewCSRFMiddleware(relayClient, "/$metadata")

	hook := csrf.Hook()
	if err := hook(context.Background(), relayClient.Get("/Products")); err != nil {
		t.Fatalf("Hook() error = %v", err)
	}
}

func TestCSRFMiddleware_Hook_WriteOperation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-CSRF-Token", "fresh-token")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	relayClient := relay.New(relay.WithBaseURL(server.URL))
	csrf := NewCSRFMiddleware(relayClient, "/$metadata")

	hook := csrf.Hook()
	if err := hook(context.Background(), relayClient.Post("/Products")); err != nil {
		t.Fatalf("Hook() error = %v", err)
	}
}

func TestCSRFMiddleware_Hook_TokenFetchError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	relayClient := relay.New(relay.WithBaseURL(server.URL))
	csrf := NewCSRFMiddleware(relayClient, "/$metadata")

	hook := csrf.Hook()
	if err := hook(context.Background(), relayClient.Post("/Products")); err == nil {
		t.Fatal("Hook() expected error, got nil")
	}
}

// --- isCsrfError / HandleResponse with real body content ---

func fetchResponse(t *testing.T, body string, status int) *relay.Response {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	defer server.Close()

	relayClient := relay.New(relay.WithBaseURL(server.URL))
	resp, err := relayClient.Execute(relayClient.Get("/Products"))
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	return resp
}

func TestCSRFMiddleware_HandleResponse_CSRFError(t *testing.T) {
	relayClient := relay.New(relay.WithBaseURL("http://example.com"))
	csrf := NewCSRFMiddleware(relayClient, "/$metadata")

	resp := fetchResponse(t, "CSRF token validation failed", 403)
	err := csrf.HandleResponse(context.Background(), resp, nil)
	if err == nil {
		t.Fatal("HandleResponse() expected error for CSRF body, got nil")
	}
	if csrf.IsValid() {
		t.Error("expected token to be invalidated")
	}
}

func TestCSRFMiddleware_HandleResponse_NonCSRF403(t *testing.T) {
	relayClient := relay.New(relay.WithBaseURL("http://example.com"))
	csrf := NewCSRFMiddleware(relayClient, "/$metadata")

	resp := fetchResponse(t, "insufficient authorization", 403)
	err := csrf.HandleResponse(context.Background(), resp, nil)
	if err == nil {
		t.Fatal("HandleResponse() expected a non-CSRF 403 error, got nil")
	}
}

func TestCSRFMiddleware_HandleResponse_NonCSRF403_PropagatesOriginalError(t *testing.T) {
	relayClient := relay.New(relay.WithBaseURL("http://example.com"))
	csrf := NewCSRFMiddleware(relayClient, "/$metadata")

	resp := fetchResponse(t, "insufficient authorization", 403)
	orig := errBoom
	err := csrf.HandleResponse(context.Background(), resp, orig)
	if err != orig {
		t.Errorf("expected original error to propagate, got %v", err)
	}
}

func TestCSRFMiddleware_HandleResponse_NotForbidden(t *testing.T) {
	relayClient := relay.New(relay.WithBaseURL("http://example.com"))
	csrf := NewCSRFMiddleware(relayClient, "/$metadata")

	resp := &relay.Response{StatusCode: 200}
	if err := csrf.HandleResponse(context.Background(), resp, nil); err != nil {
		t.Errorf("HandleResponse() expected nil for non-403, got %v", err)
	}
}

func TestCSRFMiddleware_HandleResponse_NilResponse(t *testing.T) {
	relayClient := relay.New(relay.WithBaseURL("http://example.com"))
	csrf := NewCSRFMiddleware(relayClient, "/$metadata")

	if err := csrf.HandleResponse(context.Background(), nil, errBoom); err != errBoom {
		t.Errorf("HandleResponse(nil) = %v, want %v", err, errBoom)
	}
}

// --- ExecuteWithCSRFRetry ---

func TestCSRFMiddleware_ExecuteWithCSRFRetry_SuccessNoRetry(t *testing.T) {
	relayClient := relay.New(relay.WithBaseURL("http://example.com"))
	csrf := NewCSRFMiddleware(relayClient, "/$metadata")

	calls := 0
	resp, err := csrf.ExecuteWithCSRFRetry(context.Background(), func() (*relay.Response, error) {
		calls++
		return &relay.Response{StatusCode: 200}, nil
	})
	if err != nil {
		t.Fatalf("ExecuteWithCSRFRetry() error = %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("StatusCode = %d, want 200", resp.StatusCode)
	}
	if calls != 1 {
		t.Errorf("expected 1 call, got %d", calls)
	}
}

func TestCSRFMiddleware_ExecuteWithCSRFRetry_FnError(t *testing.T) {
	relayClient := relay.New(relay.WithBaseURL("http://example.com"))
	csrf := NewCSRFMiddleware(relayClient, "/$metadata")

	_, err := csrf.ExecuteWithCSRFRetry(context.Background(), func() (*relay.Response, error) {
		return nil, errBoom
	})
	if err != errBoom {
		t.Errorf("expected errBoom, got %v", err)
	}
}

func TestCSRFMiddleware_ExecuteWithCSRFRetry_CSRFRetrySucceeds(t *testing.T) {
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-CSRF-Token", "fresh-token")
		w.WriteHeader(http.StatusOK)
	}))
	defer tokenServer.Close()

	relayClient := relay.New(relay.WithBaseURL(tokenServer.URL))
	csrf := NewCSRFMiddleware(relayClient, "/$metadata")

	calls := 0
	resp, err := csrf.ExecuteWithCSRFRetry(context.Background(), func() (*relay.Response, error) {
		calls++
		if calls == 1 {
			return fetchResponse(t, "CSRF token validation failed", 403), nil
		}
		return &relay.Response{StatusCode: 200}, nil
	})
	if err != nil {
		t.Fatalf("ExecuteWithCSRFRetry() error = %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("StatusCode = %d, want 200", resp.StatusCode)
	}
	if calls != 2 {
		t.Errorf("expected 2 calls (retry), got %d", calls)
	}
}

func TestCSRFMiddleware_ExecuteWithCSRFRetry_NonCSRF403NoRetry(t *testing.T) {
	relayClient := relay.New(relay.WithBaseURL("http://example.com"))
	csrf := NewCSRFMiddleware(relayClient, "/$metadata")

	calls := 0
	resp, err := csrf.ExecuteWithCSRFRetry(context.Background(), func() (*relay.Response, error) {
		calls++
		return fetchResponse(t, "insufficient authorization", 403), nil
	})
	if err != nil {
		t.Fatalf("ExecuteWithCSRFRetry() unexpected error = %v", err)
	}
	if resp.StatusCode != 403 {
		t.Errorf("StatusCode = %d, want 403", resp.StatusCode)
	}
	if calls != 1 {
		t.Errorf("expected 1 call (no retry for non-CSRF 403), got %d", calls)
	}
}

func TestCSRFMiddleware_ExecuteWithCSRFRetry_RefreshFails(t *testing.T) {
	relayClient := relay.New(relay.WithBaseURL("http://example.com"))
	csrf := NewCSRFMiddleware(relayClient, "/$metadata")

	_, err := csrf.ExecuteWithCSRFRetry(context.Background(), func() (*relay.Response, error) {
		return fetchResponse(t, "CSRF token validation failed", 403), nil
	})
	if err == nil {
		t.Fatal("ExecuteWithCSRFRetry() expected error when token refresh fails, got nil")
	}
}

// --- WithWriteMethodDetection / HandleResponseForWriteOps ---

func TestCSRFMiddleware_WriteMethodDetection_ReadOpsSkipped(t *testing.T) {
	relayClient := relay.New(relay.WithBaseURL("http://example.com"))
	csrf := NewCSRFMiddleware(relayClient, "/$metadata")

	detectHook := csrf.WithWriteMethodDetection()
	responseHook := csrf.HandleResponseForWriteOps()

	if err := detectHook(context.Background(), relayClient.Get("/Products")); err != nil {
		t.Fatalf("detectHook() error = %v", err)
	}
	resp := &relay.Response{StatusCode: 403}
	if err := responseHook(context.Background(), resp, nil); err != nil {
		t.Errorf("expected nil error for read-op 403, got %v", err)
	}
}

func TestCSRFMiddleware_WriteMethodDetection_WriteOpsHandled(t *testing.T) {
	relayClient := relay.New(relay.WithBaseURL("http://example.com"))
	csrf := NewCSRFMiddleware(relayClient, "/$metadata")

	detectHook := csrf.WithWriteMethodDetection()
	responseHook := csrf.HandleResponseForWriteOps()

	if err := detectHook(context.Background(), relayClient.Post("/Products")); err != nil {
		t.Fatalf("detectHook() error = %v", err)
	}
	resp := fetchResponse(t, "CSRF token validation failed", 403)
	if err := responseHook(context.Background(), resp, nil); err == nil {
		t.Error("expected error for write-op CSRF 403")
	}
}

var errBoom = &boomError{}

type boomError struct{}

func (*boomError) Error() string { return "boom" }
