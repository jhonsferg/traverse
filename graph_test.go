package traverse

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jhonsferg/relay"
)

// TestNewGraphClient_BaseURLv1 verifies that NewGraphClient sets the correct base URL for v1.0.
func TestNewGraphClient_BaseURLv1(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"value": []interface{}{},
		})
	}))
	defer server.Close()

	relayClient := relay.New(relay.WithBaseURL(server.URL))
	defer func() {
		_ = relayClient.Shutdown(context.Background())
	}()

	client := NewGraphClient(relayClient, GraphConfig{
		Version:     "v1.0",
		AccessToken: "test-token",
	})
	defer func() {
		_ = client.Close()
	}()

	expectedURL := "https://graph.microsoft.com/v1.0"
	if client.BaseURL() != expectedURL {
		t.Errorf("expected BaseURL %s, got %s", expectedURL, client.BaseURL())
	}
}

// TestNewGraphClient_BaseURLBeta verifies that NewGraphClient sets the correct base URL for beta.
func TestNewGraphClient_BaseURLBeta(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"value": []interface{}{},
		})
	}))
	defer server.Close()

	relayClient := relay.New(relay.WithBaseURL(server.URL))
	defer func() {
		_ = relayClient.Shutdown(context.Background())
	}()

	client := NewGraphClient(relayClient, GraphConfig{
		Version:     "beta",
		AccessToken: "test-token",
	})
	defer func() {
		_ = client.Close()
	}()

	expectedURL := "https://graph.microsoft.com/beta"
	if client.BaseURL() != expectedURL {
		t.Errorf("expected BaseURL %s, got %s", expectedURL, client.BaseURL())
	}
}

// TestNewGraphClient_DefaultVersion verifies that NewGraphClient defaults to v1.0 when Version is empty.
func TestNewGraphClient_DefaultVersion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"value": []interface{}{},
		})
	}))
	defer server.Close()

	relayClient := relay.New(relay.WithBaseURL(server.URL))
	defer func() {
		_ = relayClient.Shutdown(context.Background())
	}()

	client := NewGraphClient(relayClient, GraphConfig{
		AccessToken: "test-token",
	})
	defer func() {
		_ = client.Close()
	}()

	expectedURL := "https://graph.microsoft.com/v1.0"
	if client.BaseURL() != expectedURL {
		t.Errorf("expected BaseURL %s (default v1.0), got %s", expectedURL, client.BaseURL())
	}
}

// TestNewGraphClient_ODataV4Version verifies that the client is configured with OData v4.
func TestNewGraphClient_ODataV4Version(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"value": []interface{}{},
		})
	}))
	defer server.Close()

	relayClient := relay.New(relay.WithBaseURL(server.URL))
	defer func() {
		_ = relayClient.Shutdown(context.Background())
	}()

	client := NewGraphClient(relayClient, GraphConfig{
		Version:     "v1.0",
		AccessToken: "test-token",
	})
	defer func() {
		_ = client.Close()
	}()

	if client.Version() != ODataV4 {
		t.Errorf("expected OData version %d, got %d", ODataV4, client.Version())
	}
}

// TestGraphError_Error verifies that GraphError.Error() returns formatted error string.
func TestGraphError_Error(t *testing.T) {
	tests := []struct {
		name        string
		err         *GraphError
		expectedMsg string
	}{
		{
			name: "both code and message",
			err: &GraphError{
				Code:    "InvalidRequest",
				Message: "The request is invalid",
			},
			expectedMsg: "traverse: Graph error InvalidRequest: The request is invalid",
		},
		{
			name: "message only",
			err: &GraphError{
				Message: "The request is invalid",
			},
			expectedMsg: "traverse: The request is invalid",
		},
		{
			name: "code only",
			err: &GraphError{
				Code: "InvalidRequest",
			},
			expectedMsg: "traverse: unknown Graph error",
		},
		{
			name: "empty",
			err: &GraphError{
				Code:    "",
				Message: "",
			},
			expectedMsg: "traverse: unknown Graph error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.err.Error()
			if result != tt.expectedMsg {
				t.Errorf("expected %q, got %q", tt.expectedMsg, result)
			}
		})
	}
}

// TestNewGraphClient_EmptyAccessToken verifies graceful handling of empty access token.
func TestNewGraphClient_EmptyAccessToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]interface{}{
				"code":    "Authentication",
				"message": "Missing authorization token",
			},
		})
	}))
	defer server.Close()

	relayClient := relay.New(relay.WithBaseURL(server.URL))
	defer func() {
		_ = relayClient.Shutdown(context.Background())
	}()

	client := NewGraphClient(relayClient, GraphConfig{
		Version:     "v1.0",
		AccessToken: "",
	})
	defer func() {
		_ = client.Close()
	}()

	if client == nil {
		t.Fatal("expected client to be created, got nil")
	}
}

// TestNewGraphClient_Integration verifies end-to-end Graph client creation.
func TestNewGraphClient_Integration(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"value": []interface{}{
				map[string]interface{}{
					"id":   "user1",
					"name": "Test User",
				},
			},
		})
	}))
	defer server.Close()

	relayClient := relay.New(relay.WithBaseURL(server.URL))
	defer func() {
		_ = relayClient.Shutdown(context.Background())
	}()

	client := NewGraphClient(relayClient, GraphConfig{
		Version:     "v1.0",
		AccessToken: "test-token",
	})
	defer func() {
		_ = client.Close()
	}()

	if client.BaseURL() != "https://graph.microsoft.com/v1.0" {
		t.Errorf("unexpected BaseURL: %s", client.BaseURL())
	}
	if client.Version() != ODataV4 {
		t.Errorf("unexpected OData version: %d", client.Version())
	}
}
