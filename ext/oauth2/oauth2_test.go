package oauth2

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestTokenIsExpired tests the Token.IsExpired method.
func TestTokenIsExpired(t *testing.T) {
	tests := []struct {
		name     string
		token    *Token
		expected bool
	}{
		{
			name:     "nil token",
			token:    nil,
			expected: true,
		},
		{
			name: "empty access token",
			token: &Token{
				AccessToken: "",
				ExpiresAt:   time.Now().Add(1 * time.Hour),
			},
			expected: true,
		},
		{
			name: "token in the future",
			token: &Token{
				AccessToken: "valid_token",
				ExpiresAt:   time.Now().Add(1 * time.Hour),
			},
			expected: false,
		},
		{
			name: "token expired",
			token: &Token{
				AccessToken: "expired_token",
				ExpiresAt:   time.Now().Add(-1 * time.Hour),
			},
			expected: true,
		},
		{
			name: "token expiring soon (within 30 seconds)",
			token: &Token{
				AccessToken: "expiring_token",
				ExpiresAt:   time.Now().Add(10 * time.Second),
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.token.IsExpired(); got != tt.expected {
				t.Errorf("IsExpired() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// TestNewTokenManager tests token manager initialization.
func TestNewTokenManager(t *testing.T) {
	config := &OAuth2Config{
		ClientID:     "test_client",
		ClientSecret: "test_secret",
		TokenURL:     "http://localhost/token",
	}

	tm := NewTokenManager(config)

	if tm == nil {
		t.Fatal("NewTokenManager returned nil")
	}

	if tm.config != config {
		t.Error("config not set correctly")
	}

	if tm.IsTokenValid() {
		t.Error("new token manager should have invalid token initially")
	}
}

// TestNewTokenManagerWithClient tests custom HTTP client.
func TestNewTokenManagerWithClient(t *testing.T) {
	config := &OAuth2Config{
		ClientID:     "test_client",
		ClientSecret: "test_secret",
		TokenURL:     "http://localhost/token",
	}

	customClient := &http.Client{
		Timeout: 60 * time.Second,
	}

	tm := NewTokenManagerWithClient(config, customClient)

	if tm == nil {
		t.Fatal("NewTokenManagerWithClient returned nil")
	}

	if tm.client != customClient {
		t.Error("custom client not set")
	}
}

// TestGetToken tests successful token retrieval and caching.
func TestGetToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/token" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}

		// Verify request parameters
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		if r.FormValue("grant_type") != "client_credentials" {
			http.Error(w, "invalid grant_type", http.StatusBadRequest)
			return
		}

		if r.FormValue("client_id") != "test_client" {
			http.Error(w, "invalid client_id", http.StatusBadRequest)
			return
		}

		// Return valid token response
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token": "test_token_value",
			"token_type":   "Bearer",
			"expires_in":   3600,
		})
	}))
	defer server.Close()

	config := &OAuth2Config{
		ClientID:     "test_client",
		ClientSecret: "test_secret",
		TokenURL:     server.URL + "/token",
	}

	tm := NewTokenManager(config)
	ctx := context.Background()

	token, err := tm.GetToken(ctx)
	if err != nil {
		t.Fatalf("GetToken failed: %v", err)
	}

	if token != "test_token_value" {
		t.Errorf("token = %q, want %q", token, "test_token_value")
	}

	// Verify token is cached
	if !tm.IsTokenValid() {
		t.Error("token should be valid after GetToken")
	}
}

// TestGetTokenCaching tests that tokens are cached and reused.
func TestGetTokenCaching(t *testing.T) {
	callCount := atomic.Int32{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount.Add(1)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token": "cached_token",
			"token_type":   "Bearer",
			"expires_in":   3600,
		})
	}))
	defer server.Close()

	config := &OAuth2Config{
		ClientID:     "test_client",
		ClientSecret: "test_secret",
		TokenURL:     server.URL + "/token",
	}

	tm := NewTokenManager(config)
	ctx := context.Background()

	// First call should fetch token
	token1, err := tm.GetToken(ctx)
	if err != nil {
		t.Fatalf("first GetToken failed: %v", err)
	}

	initialCallCount := callCount.Load()

	// Second call should use cached token
	token2, err := tm.GetToken(ctx)
	if err != nil {
		t.Fatalf("second GetToken failed: %v", err)
	}

	if token1 != token2 {
		t.Errorf("tokens should be identical: %q vs %q", token1, token2)
	}

	if callCount.Load() != initialCallCount {
		t.Errorf("token should be cached, expected %d calls, got %d", initialCallCount, callCount.Load())
	}
}

// TestRefreshToken tests explicit token refresh.
func TestRefreshToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token": "refreshed_token",
			"token_type":   "Bearer",
			"expires_in":   3600,
		})
	}))
	defer server.Close()

	config := &OAuth2Config{
		ClientID:     "test_client",
		ClientSecret: "test_secret",
		TokenURL:     server.URL + "/token",
	}

	tm := NewTokenManager(config)
	ctx := context.Background()

	err := tm.RefreshToken(ctx)
	if err != nil {
		t.Fatalf("RefreshToken failed: %v", err)
	}

	cachedToken := tm.GetCachedToken()
	if cachedToken == nil || cachedToken.AccessToken != "refreshed_token" {
		t.Error("token not refreshed correctly")
	}
}

// TestTokenExpiration tests that expired tokens are refreshed.
func TestTokenExpiration(t *testing.T) {
	callCount := atomic.Int32{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount.Add(1)
		count := callCount.Load()

		w.Header().Set("Content-Type", "application/json")

		var response map[string]interface{}
		if count == 1 {
			response = map[string]interface{}{
				"access_token": "first_token",
				"token_type":   "Bearer",
				"expires_in":   1, // 1 second expiry
			}
		} else {
			response = map[string]interface{}{
				"access_token": "second_token",
				"token_type":   "Bearer",
				"expires_in":   3600,
			}
		}

		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	config := &OAuth2Config{
		ClientID:     "test_client",
		ClientSecret: "test_secret",
		TokenURL:     server.URL + "/token",
	}

	tm := NewTokenManager(config)
	ctx := context.Background()

	// First token
	token1, err := tm.GetToken(ctx)
	if err != nil {
		t.Fatalf("first GetToken failed: %v", err)
	}

	if token1 != "first_token" {
		t.Errorf("first token = %q, want %q", token1, "first_token")
	}

	// Wait for expiration (token expires in 1 second, but we check 30 seconds early)
	time.Sleep(2 * time.Second)

	// Second call should refresh since token is expired
	token2, err := tm.GetToken(ctx)
	if err != nil {
		t.Fatalf("second GetToken failed: %v", err)
	}

	if token2 != "second_token" {
		t.Errorf("second token = %q, want %q", token2, "second_token")
	}

	if callCount.Load() != 2 {
		t.Errorf("expected 2 token fetches, got %d", callCount.Load())
	}
}

// TestInvalidateToken tests token invalidation.
func TestInvalidateToken(t *testing.T) {
	config := &OAuth2Config{
		ClientID:     "test_client",
		ClientSecret: "test_secret",
		TokenURL:     "http://localhost/token",
	}

	tm := NewTokenManager(config)

	// Set a token manually
	tm.mu.Lock()
	tm.token = &Token{
		AccessToken: "valid_token",
		ExpiresAt:   time.Now().Add(1 * time.Hour),
	}
	tm.mu.Unlock()

	if !tm.IsTokenValid() {
		t.Error("token should be valid before invalidation")
	}

	tm.InvalidateToken()

	if tm.IsTokenValid() {
		t.Error("token should be invalid after invalidation")
	}
}

// TestGetCachedToken tests retrieving cached token without refresh.
func TestGetCachedToken(t *testing.T) {
	config := &OAuth2Config{
		ClientID:     "test_client",
		ClientSecret: "test_secret",
		TokenURL:     "http://localhost/token",
	}

	tm := NewTokenManager(config)

	// Initially should be nil
	if tm.GetCachedToken() != nil {
		t.Error("cached token should be nil initially")
	}

	// Set a token manually
	expectedToken := &Token{
		AccessToken: "cached_value",
		ExpiresAt:   time.Now().Add(1 * time.Hour),
	}

	tm.mu.Lock()
	tm.token = expectedToken
	tm.mu.Unlock()

	cachedToken := tm.GetCachedToken()
	if cachedToken != expectedToken {
		t.Error("cached token not retrieved correctly")
	}
}

// TestConcurrentGetToken tests concurrent token access.
func TestConcurrentGetToken(t *testing.T) {
	callCount := atomic.Int32{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount.Add(1)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token": "concurrent_token",
			"token_type":   "Bearer",
			"expires_in":   3600,
		})
	}))
	defer server.Close()

	config := &OAuth2Config{
		ClientID:     "test_client",
		ClientSecret: "test_secret",
		TokenURL:     server.URL + "/token",
	}

	tm := NewTokenManager(config)
	ctx := context.Background()

	// 10 concurrent goroutines requesting tokens
	numGoroutines := 10
	results := make(chan string, numGoroutines)
	var wg sync.WaitGroup

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			token, err := tm.GetToken(ctx)
			if err != nil {
				t.Errorf("GetToken failed: %v", err)
				return
			}
			results <- token
		}()
	}

	wg.Wait()
	close(results)

	// All tokens should be the same
	var tokens []string
	for token := range results {
		tokens = append(tokens, token)
	}

	if len(tokens) != numGoroutines {
		t.Errorf("expected %d tokens, got %d", numGoroutines, len(tokens))
	}

	for i, token := range tokens {
		if token != "concurrent_token" {
			t.Errorf("token %d = %q, want %q", i, token, "concurrent_token")
		}
	}

	// Token should only be fetched once due to caching
	if callCount.Load() > 2 {
		t.Errorf("expected at most 2 token fetches (with some race potential), got %d", callCount.Load())
	}
}

// TestConcurrentRefresh tests concurrent token refresh and validation.
func TestConcurrentRefresh(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token": "test_token",
			"token_type":   "Bearer",
			"expires_in":   3600,
		})
	}))
	defer server.Close()

	config := &OAuth2Config{
		ClientID:     "test_client",
		ClientSecret: "test_secret",
		TokenURL:     server.URL + "/token",
	}

	tm := NewTokenManager(config)
	ctx := context.Background()

	// 5 goroutines calling GetToken, 5 calling IsTokenValid
	numGoroutines := 10
	results := make(chan bool, numGoroutines)
	var wg sync.WaitGroup

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := tm.GetToken(ctx)
			results <- err == nil
		}()
	}

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results <- tm.IsTokenValid()
		}()
	}

	wg.Wait()
	close(results)

	// All operations should succeed
	for valid := range results {
		if !valid {
			t.Error("concurrent operation failed")
		}
	}
}

// TestErrorHandling tests various error conditions.
func TestErrorHandling(t *testing.T) {
	tests := []struct {
		name       string
		setupFunc  func() (*OAuth2Config, *http.Client)
		shouldFail bool
	}{
		{
			name: "nil config",
			setupFunc: func() (*OAuth2Config, *http.Client) {
				return nil, &http.Client{Timeout: 10 * time.Second}
			},
			shouldFail: true,
		},
		{
			name: "empty token URL",
			setupFunc: func() (*OAuth2Config, *http.Client) {
				return &OAuth2Config{
					ClientID:     "id",
					ClientSecret: "secret",
					TokenURL:     "",
				}, &http.Client{Timeout: 10 * time.Second}
			},
			shouldFail: true,
		},
		{
			name: "empty client ID",
			setupFunc: func() (*OAuth2Config, *http.Client) {
				return &OAuth2Config{
					ClientID:     "",
					ClientSecret: "secret",
					TokenURL:     "http://localhost/token",
				}, &http.Client{Timeout: 10 * time.Second}
			},
			shouldFail: true,
		},
		{
			name: "server error",
			setupFunc: func() (*OAuth2Config, *http.Client) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					http.Error(w, "server error", http.StatusInternalServerError)
				}))

				client := server.Client()
				client.Timeout = 10 * time.Second

				config := &OAuth2Config{
					ClientID:     "test",
					ClientSecret: "secret",
					TokenURL:     server.URL + "/token",
				}

				return config, client
			},
			shouldFail: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "server error" {
				config, client := tt.setupFunc()
				tm := NewTokenManagerWithClient(config, client)
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()

				_, err := tm.GetToken(ctx)
				if !tt.shouldFail && err != nil {
					t.Errorf("GetToken failed unexpectedly: %v", err)
				}
				if tt.shouldFail && err == nil {
					t.Error("GetToken should have failed")
				}
				return
			}

			config, client := tt.setupFunc()

			if config == nil {
				tm := &TokenManager{config: nil, client: client}
				ctx := context.Background()
				_, err := tm.GetToken(ctx)
				if !tt.shouldFail && err != nil {
					t.Errorf("GetToken failed unexpectedly: %v", err)
				}
				if tt.shouldFail && err == nil {
					t.Error("GetToken should have failed")
				}
				return
			}

			tm := NewTokenManagerWithClient(config, client)
			ctx := context.Background()

			_, err := tm.GetToken(ctx)
			if !tt.shouldFail && err != nil {
				t.Errorf("GetToken failed unexpectedly: %v", err)
			}
			if tt.shouldFail && err == nil {
				t.Error("GetToken should have failed")
			}
		})
	}
}

// TestTokenWithScopes tests token request with scopes.
func TestTokenWithScopes(t *testing.T) {
	var capturedRequest *http.Request

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedRequest = r

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token": "scoped_token",
			"token_type":   "Bearer",
			"expires_in":   3600,
		})
	}))
	defer server.Close()

	config := &OAuth2Config{
		ClientID:     "test_client",
		ClientSecret: "test_secret",
		TokenURL:     server.URL + "/token",
		Scopes:       []string{"scope1", "scope2"},
	}

	tm := NewTokenManager(config)
	ctx := context.Background()

	_, err := tm.GetToken(ctx)
	if err != nil {
		t.Fatalf("GetToken failed: %v", err)
	}

	// Verify scopes were included in request
	if capturedRequest != nil {
		if err := capturedRequest.ParseForm(); err == nil {
			scopes := capturedRequest.FormValue("scope")
			if scopes != "scope1 scope2" {
				t.Errorf("scopes = %q, want %q", scopes, "scope1 scope2")
			}
		}
	}
}

// TestContextCancellation tests behavior with cancelled context.
func TestContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second) // Simulate slow server

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token": "token",
			"token_type":   "Bearer",
			"expires_in":   3600,
		})
	}))
	defer server.Close()

	config := &OAuth2Config{
		ClientID:     "test_client",
		ClientSecret: "test_secret",
		TokenURL:     server.URL + "/token",
	}

	tm := NewTokenManager(config)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	_, err := tm.GetToken(ctx)
	if err == nil {
		t.Error("GetToken should have failed with context timeout")
	}
}

// TestMissingAccessTokenInResponse tests handling of invalid token response.
func TestMissingAccessTokenInResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Return response without access_token
		json.NewEncoder(w).Encode(map[string]interface{}{
			"token_type": "Bearer",
			"expires_in": 3600,
		})
	}))
	defer server.Close()

	config := &OAuth2Config{
		ClientID:     "test_client",
		ClientSecret: "test_secret",
		TokenURL:     server.URL + "/token",
	}

	tm := NewTokenManager(config)
	ctx := context.Background()

	_, err := tm.GetToken(ctx)
	if err == nil {
		t.Error("GetToken should have failed with missing access_token")
	}
}

// TestTokenResponseParsing tests correct parsing of various token responses.
func TestTokenResponseParsing(t *testing.T) {
	tests := []struct {
		name           string
		response       map[string]interface{}
		expectedToken  string
		expectedExpiry int
		shouldFail     bool
	}{
		{
			name: "minimal response",
			response: map[string]interface{}{
				"access_token": "token123",
				"token_type":   "Bearer",
				"expires_in":   3600,
			},
			expectedToken:  "token123",
			expectedExpiry: 3600,
			shouldFail:     false,
		},
		{
			name: "response with refresh token",
			response: map[string]interface{}{
				"access_token":  "token456",
				"token_type":    "Bearer",
				"expires_in":    7200,
				"refresh_token": "refresh123",
			},
			expectedToken:  "token456",
			expectedExpiry: 7200,
			shouldFail:     false,
		},
		{
			name: "response with scope",
			response: map[string]interface{}{
				"access_token": "token789",
				"token_type":   "Bearer",
				"expires_in":   1800,
				"scope":        "api",
			},
			expectedToken:  "token789",
			expectedExpiry: 1800,
			shouldFail:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(tt.response)
			}))
			defer server.Close()

			config := &OAuth2Config{
				ClientID:     "test_client",
				ClientSecret: "test_secret",
				TokenURL:     server.URL + "/token",
			}

			tm := NewTokenManager(config)
			ctx := context.Background()

			token, err := tm.GetToken(ctx)
			if tt.shouldFail {
				if err == nil {
					t.Error("expected error")
				}
				return
			}

			if err != nil {
				t.Fatalf("GetToken failed: %v", err)
			}

			if token != tt.expectedToken {
				t.Errorf("token = %q, want %q", token, tt.expectedToken)
			}

			cachedToken := tm.GetCachedToken()
			if cachedToken.AccessToken != tt.expectedToken {
				t.Errorf("cached token = %q, want %q", cachedToken.AccessToken, tt.expectedToken)
			}

			if cachedToken.ExpiresIn != tt.expectedExpiry {
				t.Errorf("expires_in = %d, want %d", cachedToken.ExpiresIn, tt.expectedExpiry)
			}
		})
	}
}

// BenchmarkGetToken benchmarks the GetToken method with caching.
func BenchmarkGetToken(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token": "benchmark_token",
			"token_type":   "Bearer",
			"expires_in":   3600,
		})
	}))
	defer server.Close()

	config := &OAuth2Config{
		ClientID:     "test_client",
		ClientSecret: "test_secret",
		TokenURL:     server.URL + "/token",
	}

	tm := NewTokenManager(config)
	ctx := context.Background()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := tm.GetToken(ctx)
		if err != nil {
			b.Fatalf("GetToken failed: %v", err)
		}
	}
}

// BenchmarkConcurrentGetToken benchmarks concurrent GetToken access.
func BenchmarkConcurrentGetToken(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token": "benchmark_token",
			"token_type":   "Bearer",
			"expires_in":   3600,
		})
	}))
	defer server.Close()

	config := &OAuth2Config{
		ClientID:     "test_client",
		ClientSecret: "test_secret",
		TokenURL:     server.URL + "/token",
	}

	tm := NewTokenManager(config)
	ctx := context.Background()

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := tm.GetToken(ctx)
			if err != nil {
				b.Fatalf("GetToken failed: %v", err)
			}
		}
	})
}

// BenchmarkIsTokenValid benchmarks the IsTokenValid method.
func BenchmarkIsTokenValid(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token": "benchmark_token",
			"token_type":   "Bearer",
			"expires_in":   3600,
		})
	}))
	defer server.Close()

	config := &OAuth2Config{
		ClientID:     "test_client",
		ClientSecret: "test_secret",
		TokenURL:     server.URL + "/token",
	}

	tm := NewTokenManager(config)
	ctx := context.Background()

	// Pre-fetch token
	if _, err := tm.GetToken(ctx); err != nil {
		b.Fatalf("GetToken failed: %v", err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		tm.IsTokenValid()
	}
}
