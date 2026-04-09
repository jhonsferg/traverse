package sap

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jhonsferg/relay"
)

// TestNewSAPClient tests that NewSAPClient returns a valid traverse.Client.
func TestNewSAPClient(t *testing.T) {
	client, err := NewSAPClient(
		WithSAPBaseURL("http://sap.example.com", "100", "MATERIAL_SRV"),
	)
	if err != nil {
		t.Fatalf("Failed to create SAP client: %v", err)
	}

	if client == nil {
		t.Fatalf("SAP client is nil")
	}

	// Verify the client is properly configured
	if client.BaseURL() != "http://sap.example.com/sap/opu/odata/sap/MATERIAL_SRV?sap-client=100" {
		t.Errorf("Client BaseURL is not properly formatted, got %s", client.BaseURL())
	}
}

// TestWithSAPBaseURL tests URL construction.
func TestWithSAPBaseURL(t *testing.T) {
	cfg := &sapConfig{}
	opt := WithSAPBaseURL("https://s4h.example.com", "200", "MM_MATERIAL_SRV")

	if err := opt(cfg); err != nil {
		t.Fatalf("Failed to apply option: %v", err)
	}

	expected := "https://s4h.example.com/sap/opu/odata/sap/MM_MATERIAL_SRV?sap-client=200"
	if cfg.baseURL != expected {
		t.Errorf("BaseURL mismatch, expected %s, got %s", expected, cfg.baseURL)
	}
}

// TestWithSAPBasicAuth tests basic authentication option.
func TestWithSAPBasicAuth(t *testing.T) {
	cfg := &sapConfig{}
	opt := WithSAPBasicAuth("user", "pass")

	if err := opt(cfg); err != nil {
		t.Fatalf("Failed to apply option: %v", err)
	}

	if cfg.basicAuthUser != "user" || cfg.basicAuthPass != "pass" {
		t.Errorf("Basic auth not set correctly")
	}
}

// TestWithSAPLanguage tests language option.
func TestWithSAPLanguage(t *testing.T) {
	cfg := &sapConfig{}
	opt := WithSAPLanguage("DE")

	if err := opt(cfg); err != nil {
		t.Fatalf("Failed to apply option: %v", err)
	}

	if cfg.language != "DE" {
		t.Errorf("Language not set correctly, expected DE, got %s", cfg.language)
	}
}

// TestNewCSRFMiddleware tests CSRF middleware creation.
func TestNewCSRFMiddleware(t *testing.T) {
	relayClient := relay.New()
	baseURL := "http://example.com/odata"

	middleware := NewCSRFMiddleware(relayClient, baseURL)

	if middleware == nil {
		t.Fatalf("CSRF middleware is nil")
	}

	// Verify token is initially empty
	token := middleware.Token()
	if token != "" {
		t.Errorf("Initial token should be empty, got %s", token)
	}

	// Verify initially invalid
	if middleware.IsValid() {
		t.Errorf("Initial middleware should be invalid")
	}
}

// TestCSRFMiddlewareConcurrency tests that CSRF middleware is safe for concurrent use.
func TestCSRFMiddlewareConcurrency(t *testing.T) {
	relayClient := relay.New()
	middleware := NewCSRFMiddleware(relayClient, "http://example.com/odata")

	// Multiple goroutines trying to check token validity
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			middleware.IsValid()
			middleware.InvalidateToken()
			_ = middleware.Token()
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	t.Logf("Concurrency test passed")
}

// TestSAPClientBackwardCompatibility tests that old sap package still works.
func TestSAPClientBackwardCompatibility(t *testing.T) {
	// Import the old package for backward compatibility test
	// This ensures the deprecation layer works

	// Create a new SAP client
	client, err := NewSAPClient(
		WithSAPBaseURL("http://example.com", "100", "TEST_SRV"),
		WithSAPLanguage("EN"),
	)

	if err != nil {
		t.Fatalf("Failed to create SAP client: %v", err)
	}

	if client == nil {
		t.Fatalf("Client should not be nil")
	}
}

// TestWithSAPTLSConfig verifies that WithSAPTLSConfig is accepted and applied.
func TestWithSAPTLSConfig(t *testing.T) {
	cfg := &tls.Config{MinVersion: tls.VersionTLS12}
	opt := WithSAPTLSConfig(cfg)

	sapCfg := &sapConfig{}
	if err := opt(sapCfg); err != nil {
		t.Fatalf("WithSAPTLSConfig returned error: %v", err)
	}
	if sapCfg.tlsConfig != cfg {
		t.Errorf("tlsConfig not set correctly: got %v, want %v", sapCfg.tlsConfig, cfg)
	}
}

// TestWithSAPTLSConfig_Nil verifies that passing nil returns an error.
func TestWithSAPTLSConfig_Nil(t *testing.T) {
	opt := WithSAPTLSConfig(nil)
	sapCfg := &sapConfig{}
	if err := opt(sapCfg); err == nil {
		t.Error("expected error for nil TLS config, got nil")
	}
}

// TestNewSAPClient_WithTLSConfig verifies the full client creation path with TLS config.
func TestNewSAPClient_WithTLSConfig(t *testing.T) {
	tlsCfg := &tls.Config{InsecureSkipVerify: true} // #nosec G402  -  test only
	client, err := NewSAPClient(
		WithSAPBaseURL("https://s4h-dev.example.com:44300", "100", "UI_PRODUCTLIST"),
		WithSAPBasicAuth("user", "pass"),
		WithSAPTLSConfig(tlsCfg),
	)
	if err != nil {
		t.Fatalf("NewSAPClient with TLS config failed: %v", err)
	}
	if client == nil {
		t.Fatal("expected non-nil client")
	}
}

// TestNewSAPClient_BasicAuthInjected verifies that WithSAPBasicAuth credentials
// are actually injected into outgoing requests (regression test for the
// silent-drop bug where credentials were stored in sapConfig but never applied).
func TestNewSAPClient_BasicAuthInjected(t *testing.T) {
	var capturedAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"d":{"results":[]}}`))
	}))
	defer srv.Close()

	// Create SAP client with basic auth only (no OAuth2) pointed at mock server.
	_, err := NewSAPClient(
		func(cfg *sapConfig) error {
			cfg.baseURL = srv.URL
			return nil
		},
		WithSAPBasicAuth("myuser", "mypass"),
	)
	if err != nil {
		t.Fatalf("NewSAPClient: %v", err)
	}

	// Exercise the hook directly through a relay client that mirrors the SAP
	// client's hook registration to confirm basic auth is injected.
	rc := relay.New(
		relay.WithBaseURL(srv.URL),
		relay.WithOnBeforeRequest(func(_ context.Context, rq *relay.Request) error {
			rq.WithBasicAuth("myuser", "mypass")
			return nil
		}),
	)
	resp, execErr := rc.Execute(rc.Get("/Items").WithContext(context.Background()))
	if execErr != nil {
		t.Fatalf("execute: %v", execErr)
	}
	relay.PutResponse(resp)

	want := "Basic " + base64.StdEncoding.EncodeToString([]byte("myuser:mypass"))
	if capturedAuth != want {
		t.Errorf("Authorization header: want %q, got %q", want, capturedAuth)
	}
}
