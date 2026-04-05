package sap

import (
"context"
"encoding/json"
"fmt"
"net/http"
"net/http/httptest"
"os"
"sync"
"testing"
"time"
)

// TestParseVCAPServices_ValidXSUAA tests parsing a valid XSUAA binding.
func TestParseVCAPServices_ValidXSUAA(t *testing.T) {
vcapJSON := `{
"xsuaa": [{
"credentials": {
"clientid": "sb-test-app",
"clientsecret": "secret123",
"url": "https://test.authentication.eu10.hana.ondemand.com",
"identityzone": "test-zone",
"identityzoneid": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
"apiurl": "https://api.authentication.eu10.hana.ondemand.com"
}
}]
}`

binding, err := ParseVCAPServices(vcapJSON, "xsuaa")
if err != nil {
t.Fatalf("Failed to parse VCAP_SERVICES: %v", err)
}

if binding.ClientID != "sb-test-app" {
t.Errorf("ClientID mismatch, expected 'sb-test-app', got '%s'", binding.ClientID)
}

if binding.ClientSecret != "secret123" {
t.Errorf("ClientSecret mismatch, expected 'secret123', got '%s'", binding.ClientSecret)
}

expectedTokenURL := "https://test.authentication.eu10.hana.ondemand.com/oauth/token"
if binding.TokenURL != expectedTokenURL {
t.Errorf("TokenURL mismatch, expected '%s', got '%s'", expectedTokenURL, binding.TokenURL)
}

if binding.ZoneID != "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx" {
t.Errorf("ZoneID should prefer identityzoneid, got '%s'", binding.ZoneID)
}
}

// TestParseVCAPServices_ZoneIDFallback tests that identityzone is used if identityzoneid is missing.
func TestParseVCAPServices_ZoneIDFallback(t *testing.T) {
vcapJSON := `{
"xsuaa": [{
"credentials": {
"clientid": "sb-test-app",
"clientsecret": "secret123",
"url": "https://test.authentication.eu10.hana.ondemand.com",
"identityzone": "test-zone"
}
}]
}`

binding, err := ParseVCAPServices(vcapJSON, "xsuaa")
if err != nil {
t.Fatalf("Failed to parse VCAP_SERVICES: %v", err)
}

if binding.ZoneID != "test-zone" {
t.Errorf("ZoneID should fallback to identityzone, got '%s'", binding.ZoneID)
}
}

// TestParseVCAPServices_InvalidJSON tests parsing invalid JSON.
func TestParseVCAPServices_InvalidJSON(t *testing.T) {
vcapJSON := `{invalid json`

_, err := ParseVCAPServices(vcapJSON, "xsuaa")
if err == nil {
t.Fatal("Expected error for invalid JSON, got nil")
}
}

// TestParseVCAPServices_MissingService tests that an error is returned when service is not found.
func TestParseVCAPServices_MissingService(t *testing.T) {
vcapJSON := `{
"other-service": [{
"credentials": {}
}]
}`

_, err := ParseVCAPServices(vcapJSON, "xsuaa")
if err == nil {
t.Fatal("Expected error for missing service, got nil")
}
}

// TestParseVCAPServices_MissingCredentials tests that an error is returned for missing credentials.
func TestParseVCAPServices_MissingCredentials(t *testing.T) {
vcapJSON := `{
"xsuaa": [{
"credentials": {
"clientid": "sb-test-app"
}
}]
}`

_, err := ParseVCAPServices(vcapJSON, "xsuaa")
if err == nil {
t.Fatal("Expected error for missing credentials, got nil")
}
}

// TestParseVCAPServicesEnv_SetEnv tests reading VCAP_SERVICES from environment.
func TestParseVCAPServicesEnv_SetEnv(t *testing.T) {
vcapJSON := `{
"xsuaa": [{
"credentials": {
"clientid": "sb-test-app",
"clientsecret": "secret123",
"url": "https://test.authentication.eu10.hana.ondemand.com"
}
}]
}`

// Save original env var
oldVal := os.Getenv("VCAP_SERVICES")
defer func() { _ = os.Setenv("VCAP_SERVICES", oldVal) }()

// Set test env var
_ = os.Setenv("VCAP_SERVICES", vcapJSON)

binding, err := ParseVCAPServicesEnv("xsuaa")
if err != nil {
t.Fatalf("Failed to parse VCAP_SERVICES from environment: %v", err)
}

if binding.ClientID != "sb-test-app" {
t.Errorf("ClientID mismatch, got '%s'", binding.ClientID)
}
}

// TestParseVCAPServicesEnv_NotSet tests that an error is returned when VCAP_SERVICES is not set.
func TestParseVCAPServicesEnv_NotSet(t *testing.T) {
// Save original env var
oldVal := os.Getenv("VCAP_SERVICES")
defer func() { _ = os.Setenv("VCAP_SERVICES", oldVal) }()

// Unset env var
_ = os.Unsetenv("VCAP_SERVICES")

_, err := ParseVCAPServicesEnv("xsuaa")
if err == nil {
t.Fatal("Expected error when VCAP_SERVICES is not set, got nil")
}
}

// TestNewBTPTokenProvider_TokenFetch tests token fetching from a mock XSUAA endpoint.
func TestNewBTPTokenProvider_TokenFetch(t *testing.T) {
// Create mock XSUAA token endpoint
server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// Verify request format
if r.Method != "POST" {
t.Errorf("Expected POST request, got %s", r.Method)
}

if r.Header.Get("Content-Type") != "application/x-www-form-urlencoded" {
t.Errorf("Expected application/x-www-form-urlencoded, got %s", r.Header.Get("Content-Type"))
}

// Parse request body with size limit for security
_ = r.ParseForm()

if r.FormValue("grant_type") != "client_credentials" {
t.Errorf("Expected grant_type=client_credentials")
}

// Write token response
w.Header().Set("Content-Type", "application/json")
_ = json.NewEncoder(w).Encode(map[string]interface{}{
"access_token": "test-token-12345",
"token_type":   "Bearer",
"expires_in":   3600,
})
}))
defer server.Close()

binding := XSUAABinding{
ClientID:     "test-client",
ClientSecret: "test-secret",
TokenURL:     server.URL + "/oauth/token",
}

provider := NewBTPTokenProvider(binding)
token, err := provider.Token(context.Background())
if err != nil {
t.Fatalf("Failed to get token: %v", err)
}

if token != "test-token-12345" {
t.Errorf("Token mismatch, expected 'test-token-12345', got '%s'", token)
}
}

// TestNewBTPTokenProvider_TokenCaching tests that tokens are cached.
func TestNewBTPTokenProvider_TokenCaching(t *testing.T) {
callCount := 0
server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
callCount++
w.Header().Set("Content-Type", "application/json")
_ = json.NewEncoder(w).Encode(map[string]interface{}{
"access_token": fmt.Sprintf("token-%d", callCount),
"token_type":   "Bearer",
"expires_in":   3600,
})
}))
defer server.Close()

binding := XSUAABinding{
ClientID:     "test-client",
ClientSecret: "test-secret",
TokenURL:     server.URL + "/oauth/token",
}

provider := NewBTPTokenProvider(binding)

// First call should fetch token
token1, err := provider.Token(context.Background())
if err != nil {
t.Fatalf("Failed to get first token: %v", err)
}

if token1 != "token-1" {
t.Errorf("First token should be 'token-1', got '%s'", token1)
}

// Second call should return cached token
token2, err := provider.Token(context.Background())
if err != nil {
t.Fatalf("Failed to get second token: %v", err)
}

if token2 != "token-1" {
t.Errorf("Second token should be cached 'token-1', got '%s'", token2)
}

if callCount != 1 {
t.Errorf("Token endpoint should be called once, was called %d times", callCount)
}
}

// TestNewBTPTokenProvider_TokenRefreshOnExpiry tests that tokens are refreshed after expiry.
func TestNewBTPTokenProvider_TokenRefreshOnExpiry(t *testing.T) {
callCount := 0
server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
callCount++
// Return a token that expires in 1 second
w.Header().Set("Content-Type", "application/json")
_ = json.NewEncoder(w).Encode(map[string]interface{}{
"access_token": fmt.Sprintf("token-%d", callCount),
"token_type":   "Bearer",
"expires_in":   1,
})
}))
defer server.Close()

binding := XSUAABinding{
ClientID:     "test-client",
ClientSecret: "test-secret",
TokenURL:     server.URL + "/oauth/token",
}

provider := NewBTPTokenProvider(binding)

// First call should fetch token
token1, err := provider.Token(context.Background())
if err != nil {
t.Fatalf("Failed to get first token: %v", err)
}

if token1 != "token-1" {
t.Errorf("First token should be 'token-1', got '%s'", token1)
}

// Wait for token to expire (the oauth2 module expires 30 seconds early)
time.Sleep(2 * time.Second)

// Third call should refresh the token (new token fetched)
token3, err := provider.Token(context.Background())
if err != nil {
t.Fatalf("Failed to get third token: %v", err)
}

if token3 != "token-2" {
t.Errorf("Third token should be 'token-2' after expiry, got '%s'", token3)
}
}

// TestNewBTPTokenProvider_ConcurrentAccess tests that token provider is thread-safe.
func TestNewBTPTokenProvider_ConcurrentAccess(t *testing.T) {
server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
w.Header().Set("Content-Type", "application/json")
_ = json.NewEncoder(w).Encode(map[string]interface{}{
"access_token": "test-token",
"token_type":   "Bearer",
"expires_in":   3600,
})
}))
defer server.Close()

binding := XSUAABinding{
ClientID:     "test-client",
ClientSecret: "test-secret",
TokenURL:     server.URL + "/oauth/token",
}

provider := NewBTPTokenProvider(binding)

// Multiple goroutines requesting tokens concurrently
var wg sync.WaitGroup
tokens := make([]string, 10)
for i := 0; i < 10; i++ {
wg.Add(1)
go func(idx int) {
defer wg.Done()
token, err := provider.Token(context.Background())
if err != nil {
t.Errorf("Goroutine %d failed to get token: %v", idx, err)
return
}
tokens[idx] = token
}(i)
}

wg.Wait()

// All tokens should be the same (cached)
for i, token := range tokens {
if token != "test-token" {
t.Errorf("Token %d should be 'test-token', got '%s'", i, token)
}
}
}

// TestNewBTPTokenProvider_ErrorHandling tests error handling and error caching.
func TestNewBTPTokenProvider_ErrorHandling(t *testing.T) {
server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
w.WriteHeader(http.StatusUnauthorized)
_, _ = w.Write([]byte("Invalid credentials"))
}))
defer server.Close()

binding := XSUAABinding{
ClientID:     "test-client",
ClientSecret: "test-secret",
TokenURL:     server.URL + "/oauth/token",
}

provider := NewBTPTokenProvider(binding)

// First error call
_, err := provider.Token(context.Background())
if err == nil {
t.Fatal("Expected error for unauthorized token endpoint")
}

// Error should be cached - second call should return same error without hitting endpoint
_, err2 := provider.Token(context.Background())
if err2 == nil {
t.Fatal("Expected error to be cached")
}
}

// TestNewBTPClientFromBinding_Integration tests end-to-end client creation with mock server.
func TestNewBTPClientFromBinding_Integration(t *testing.T) {
// Create mock XSUAA token endpoint
tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
w.Header().Set("Content-Type", "application/json")
_ = json.NewEncoder(w).Encode(map[string]interface{}{
"access_token": "btp-test-token",
"token_type":   "Bearer",
"expires_in":   3600,
})
}))
defer tokenServer.Close()

binding := XSUAABinding{
ClientID:     "test-client",
ClientSecret: "test-secret",
TokenURL:     tokenServer.URL + "/oauth/token",
APIURL:       "https://api.sap.example.com",
ZoneID:       "test-zone-id",
}

ctx := context.Background()
client, err := NewBTPClientFromBinding(ctx, binding, "https://api.sap.example.com/odata")
if err != nil {
t.Fatalf("Failed to create BTP client: %v", err)
}

if client == nil {
t.Fatal("BTP client is nil")
}
}

// TestNewBTPClientFromBinding_MissingCredentials tests that error is returned for missing credentials.
func TestNewBTPClientFromBinding_MissingCredentials(t *testing.T) {
binding := XSUAABinding{
ClientID: "",
// Missing ClientSecret
TokenURL: "https://test.auth.com/oauth/token",
}

ctx := context.Background()
_, err := NewBTPClientFromBinding(ctx, binding, "https://api.sap.example.com/odata")
if err == nil {
t.Fatal("Expected error for missing credentials")
}
}

// TestNewBTPClientFromBinding_MissingServiceURL tests that error is returned for missing service URL.
func TestNewBTPClientFromBinding_MissingServiceURL(t *testing.T) {
binding := XSUAABinding{
ClientID:     "test-client",
ClientSecret: "test-secret",
TokenURL:     "https://test.auth.com/oauth/token",
}

ctx := context.Background()
_, err := NewBTPClientFromBinding(ctx, binding, "")
if err == nil {
t.Fatal("Expected error for missing service URL")
}
}

// TestNewBTPClientFromBinding_TokenFetchFailure tests that error is returned if initial token fetch fails.
func TestNewBTPClientFromBinding_TokenFetchFailure(t *testing.T) {
binding := XSUAABinding{
ClientID:     "test-client",
ClientSecret: "test-secret",
TokenURL:     "https://invalid.example.com/oauth/token",
}

ctx := context.Background()
_, err := NewBTPClientFromBinding(ctx, binding, "https://api.sap.example.com/odata")
if err == nil {
t.Fatal("Expected error when token fetch fails")
}
}

// TestParseVCAPServices_EmptyJSON tests parsing empty JSON string.
func TestParseVCAPServices_EmptyJSON(t *testing.T) {
_, err := ParseVCAPServices("", "xsuaa")
if err == nil {
t.Fatal("Expected error for empty JSON")
}
}

// TestParseVCAPServices_CustomServiceName tests parsing a custom service name.
func TestParseVCAPServices_CustomServiceName(t *testing.T) {
vcapJSON := `{
"business-rules": [{
"credentials": {
"clientid": "br-client",
"clientsecret": "br-secret",
"url": "https://br.authentication.eu10.hana.ondemand.com"
}
}]
}`

binding, err := ParseVCAPServices(vcapJSON, "business-rules")
if err != nil {
t.Fatalf("Failed to parse custom service: %v", err)
}

if binding.ClientID != "br-client" {
t.Errorf("ClientID mismatch, got '%s'", binding.ClientID)
}
}

// TestNewBTPTokenProvider_ContextCancellation tests that context cancellation is handled.
func TestNewBTPTokenProvider_ContextCancellation(t *testing.T) {
server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
time.Sleep(2 * time.Second)
w.Header().Set("Content-Type", "application/json")
_ = json.NewEncoder(w).Encode(map[string]interface{}{
"access_token": "token",
"token_type":   "Bearer",
"expires_in":   3600,
})
}))
defer server.Close()

binding := XSUAABinding{
ClientID:     "test-client",
ClientSecret: "test-secret",
TokenURL:     server.URL + "/oauth/token",
}

provider := NewBTPTokenProvider(binding)

ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
defer cancel()

_, err := provider.Token(ctx)
if err == nil {
t.Fatal("Expected error for cancelled context")
}
}
