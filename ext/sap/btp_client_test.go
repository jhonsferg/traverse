package sap

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestNewBTPClient_Success(t *testing.T) {
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token": "btp-test-token",
			"token_type":   "Bearer",
			"expires_in":   3600,
		})
	}))
	defer tokenServer.Close()

	vcapJSON := `{
"xsuaa": [{
"credentials": {
"clientid": "sb-test-app",
"clientsecret": "secret123",
"url": "` + tokenServer.URL + `",
"identityzoneid": "test-zone-id"
}
}]
}`

	oldVal := os.Getenv("VCAP_SERVICES")
	defer func() { _ = os.Setenv("VCAP_SERVICES", oldVal) }()
	_ = os.Setenv("VCAP_SERVICES", vcapJSON)

	client, err := NewBTPClient(context.Background(), "https://api.sap.example.com/odata")
	if err != nil {
		t.Fatalf("NewBTPClient() error = %v", err)
	}
	if client == nil {
		t.Fatal("NewBTPClient() returned nil client")
	}
}

func TestNewBTPClient_ParseVCAPError(t *testing.T) {
	oldVal := os.Getenv("VCAP_SERVICES")
	defer func() { _ = os.Setenv("VCAP_SERVICES", oldVal) }()
	_ = os.Unsetenv("VCAP_SERVICES")

	_, err := NewBTPClient(context.Background(), "https://api.sap.example.com/odata")
	if err == nil {
		t.Fatal("NewBTPClient() expected error when VCAP_SERVICES is not set, got nil")
	}
}
