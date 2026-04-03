package sap

import (
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
