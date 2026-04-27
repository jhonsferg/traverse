package sap

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jhonsferg/relay"
)

// TestCSRFTokenReuse verifies that tokens are reused within the 30-minute window
// instead of being invalidated preventively before each write operation (BUG-003 fix).
func TestCSRFTokenReuse(t *testing.T) {
	fetchCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			// CSRF fetch request
			fetchCount++
			w.Header().Set("X-CSRF-Token", "token-reused")
			w.WriteHeader(http.StatusOK)
		} else if r.Method == "POST" {
			// Write operation
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	relayClient := relay.New(relay.WithBaseURL(server.URL))
	csrf := NewCSRFMiddleware(relayClient, "/$metadata")

	ctx := context.Background()

	// First write operation - should fetch token
	token1, err := csrf.GetToken(ctx)
	if err != nil {
		t.Fatalf("First GetToken failed: %v", err)
	}
	if token1 != "token-reused" {
		t.Errorf("Expected token 'token-reused', got %s", token1)
	}
	if fetchCount != 1 {
		t.Errorf("Expected 1 fetch, got %d", fetchCount)
	}

	// Second write operation - should reuse token (no new fetch)
	token2, err := csrf.GetToken(ctx)
	if err != nil {
		t.Fatalf("Second GetToken failed: %v", err)
	}
	if token2 != token1 {
		t.Errorf("Token should be reused, but got different token")
	}
	if fetchCount != 1 {
		t.Errorf("Expected 1 fetch (token reused), got %d", fetchCount)
	}

	// Third write operation - token still valid, should NOT fetch
	token3, err := csrf.GetToken(ctx)
	if err != nil {
		t.Fatalf("Third GetToken failed: %v", err)
	}
	if token3 != token1 {
		t.Errorf("Token should still be reused")
	}
	if fetchCount != 1 {
		t.Errorf("Expected 1 fetch (no preventive invalidation), got %d", fetchCount)
	}
}

// TestCSRFTokenExpiration verifies that invalidated tokens trigger a new fetch.
func TestCSRFTokenExpiration(t *testing.T) {
	tokenVersion := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			// CSRF fetch request
			tokenVersion++
			w.Header().Set("X-CSRF-Token", "token-v"+string(rune(48+tokenVersion)))
			w.WriteHeader(http.StatusOK)
		} else if r.Method == "POST" {
			// Write operation
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	relayClient := relay.New(relay.WithBaseURL(server.URL))
	csrf := NewCSRFMiddleware(relayClient, "/$metadata")

	ctx := context.Background()

	// Get first token
	token1, err := csrf.GetToken(ctx)
	if err != nil {
		t.Fatalf("First GetToken failed: %v", err)
	}
	if token1 != "token-v1" {
		t.Errorf("Expected token 'token-v1', got %s", token1)
	}

	// Invalidate token (simulating 403 response)
	csrf.InvalidateToken()

	// Get new token - should fetch
	token2, err := csrf.GetToken(ctx)
	if err != nil {
		t.Fatalf("Second GetToken after invalidation failed: %v", err)
	}
	if token2 == token1 {
		t.Errorf("Token should be different after invalidation")
	}
}

// TestHandleResponse403 verifies that HandleResponse invalidates token on 403.
func TestHandleResponse403(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			w.Header().Set("X-CSRF-Token", "valid-token")
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	relayClient := relay.New(relay.WithBaseURL(server.URL))
	csrf := NewCSRFMiddleware(relayClient, "/$metadata")

	ctx := context.Background()

	// Get token
	token, err := csrf.GetToken(ctx)
	if err != nil {
		t.Fatalf("GetToken failed: %v", err)
	}
	if token != "valid-token" {
		t.Errorf("Expected token 'valid-token', got %s", token)
	}

	// Check token is valid
	if !csrf.IsValid() {
		t.Error("Token should be valid after fetch")
	}

	// Simulate 403 response by calling HandleResponse
	resp := &relay.Response{StatusCode: 403}
	err = csrf.HandleResponse(ctx, resp, nil)
	if err == nil {
		t.Error("HandleResponse should return an error for 403")
	}

	// Check token is invalidated
	if csrf.IsValid() {
		t.Error("Token should be invalidated after 403 response")
	}

	// New GetToken should fetch fresh token
	token, err = csrf.GetToken(ctx)
	if err != nil {
		t.Fatalf("GetToken after 403 failed: %v", err)
	}
	if token != "valid-token" {
		t.Errorf("Expected fresh token 'valid-token', got %s", token)
	}
}

// TestTokenIsValidBeforeExpiration verifies IsValid() returns true before expiration.
func TestTokenIsValidBeforeExpiration(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			w.Header().Set("X-CSRF-Token", "test-token")
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	relayClient := relay.New(relay.WithBaseURL(server.URL))
	csrf := NewCSRFMiddleware(relayClient, "/$metadata")

	ctx := context.Background()

	// Token should be invalid before fetch
	if csrf.IsValid() {
		t.Error("Token should be invalid before fetch")
	}

	// Fetch token
	_, err := csrf.GetToken(ctx)
	if err != nil {
		t.Fatalf("GetToken failed: %v", err)
	}

	// Token should be valid after fetch
	if !csrf.IsValid() {
		t.Error("Token should be valid after fetch")
	}
}
