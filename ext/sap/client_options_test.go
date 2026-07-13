package sap

import (
	"testing"

	traverse "github.com/jhonsferg/traverse"
)

func TestWithSAPOAuth2Scopes(t *testing.T) {
	cfg := &sapConfig{}
	opt := WithSAPOAuth2Scopes("scope1", "scope2")

	if err := opt(cfg); err != nil {
		t.Fatalf("Failed to apply option: %v", err)
	}
	if len(cfg.oauth2Scopes) != 2 || cfg.oauth2Scopes[0] != "scope1" || cfg.oauth2Scopes[1] != "scope2" {
		t.Errorf("oauth2Scopes not set correctly, got %v", cfg.oauth2Scopes)
	}
}

func TestWithSAPOAuth2Scopes_Empty(t *testing.T) {
	cfg := &sapConfig{}
	opt := WithSAPOAuth2Scopes()

	if err := opt(cfg); err == nil {
		t.Fatal("expected error for empty scopes, got nil")
	}
}

func TestWithSAPFormatJSON(t *testing.T) {
	cfg := &sapConfig{}
	opt := WithSAPFormatJSON()

	if err := opt(cfg); err != nil {
		t.Fatalf("Failed to apply option: %v", err)
	}
	if !cfg.formatJSON {
		t.Error("expected formatJSON to be true")
	}
}

func TestWithSAPMaxPageSize(t *testing.T) {
	cfg := &sapConfig{}
	opt := WithSAPMaxPageSize(50)

	if err := opt(cfg); err != nil {
		t.Fatalf("Failed to apply option: %v", err)
	}
	if cfg.pageSize != 50 {
		t.Errorf("pageSize = %d, want 50", cfg.pageSize)
	}
}

func TestWithSAPMaxPageSize_Invalid(t *testing.T) {
	cfg := &sapConfig{}

	for _, n := range []int{0, -1} {
		if err := WithSAPMaxPageSize(n)(cfg); err == nil {
			t.Errorf("WithSAPMaxPageSize(%d) expected error, got nil", n)
		}
	}
}

func TestWithSAPODataVersion(t *testing.T) {
	cfg := &sapConfig{}
	opt := WithSAPODataVersion(traverse.ODataV4)

	if err := opt(cfg); err != nil {
		t.Fatalf("Failed to apply option: %v", err)
	}
	if cfg.version != int(traverse.ODataV4) {
		t.Errorf("version = %d, want %d", cfg.version, int(traverse.ODataV4))
	}
}

func TestNewSAPClient_WithOAuth2AndFormatJSON(t *testing.T) {
	client, err := NewSAPClient(
		WithSAPBaseURL("http://sap.example.com", "100", "MATERIAL_SRV"),
		WithSAPOAuth2("https://auth.example.com/oauth/token", "client-id", "secret"),
		WithSAPOAuth2Scopes("read", "write"),
		WithSAPFormatJSON(),
		WithSAPMaxPageSize(200),
		WithSAPODataVersion(traverse.ODataV2),
	)
	if err != nil {
		t.Fatalf("NewSAPClient() error = %v", err)
	}
	if client == nil {
		t.Fatal("NewSAPClient() returned nil client")
	}
}

func TestNewSAPClient_OptionError(t *testing.T) {
	_, err := NewSAPClient(
		WithSAPBaseURL("", "100", ""),
	)
	if err == nil {
		t.Fatal("NewSAPClient() expected error for invalid option, got nil")
	}
}
