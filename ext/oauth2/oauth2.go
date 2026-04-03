// Package oauth2 provides OAuth2 token management for SAP integration.
package oauth2

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"
)

// OAuth2Config holds OAuth2 client credentials and configuration.
type OAuth2Config struct {
	ClientID     string
	ClientSecret string
	TokenURL     string
	Scopes       []string
}

// Token represents an OAuth2 access token response.
type Token struct {
	AccessToken  string    `json:"access_token"`
	TokenType    string    `json:"token_type"`
	ExpiresIn    int       `json:"expires_in"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	Scope        string    `json:"scope,omitempty"`
	ExpiresAt    time.Time `json:"-"`
}

// IsExpired checks if the token is expired.
// A token is considered expired 30 seconds before its actual expiry time
// to avoid using an expired token.
func (t *Token) IsExpired() bool {
	if t == nil || t.AccessToken == "" {
		return true
	}
	// Refresh 30 seconds early to avoid edge cases
	return time.Now().Add(30 * time.Second).After(t.ExpiresAt)
}

// TokenManager manages OAuth2 token lifecycle with caching and refresh.
// It is thread-safe and handles concurrent access to the token.
type TokenManager struct {
	config *OAuth2Config
	client *http.Client

	mu    sync.RWMutex
	token *Token
}

// NewTokenManager creates a new token manager with the given OAuth2 config.
// Uses a default HTTP client with a 30-second timeout.
func NewTokenManager(config *OAuth2Config) *TokenManager {
	return NewTokenManagerWithClient(config, &http.Client{
		Timeout: 30 * time.Second,
	})
}

// NewTokenManagerWithClient creates a new token manager with a custom HTTP client.
// Useful for testing or when you need to customize the HTTP client behavior.
func NewTokenManagerWithClient(config *OAuth2Config, client *http.Client) *TokenManager {
	return &TokenManager{
		config: config,
		client: client,
	}
}

// GetToken returns a valid access token, refreshing if necessary.
// This is the main API for obtaining a token.
// It handles refresh logic transparently and is safe for concurrent use.
func (tm *TokenManager) GetToken(ctx context.Context) (string, error) {
	tm.mu.RLock()

	// Check if we have a valid token
	if tm.token != nil && !tm.token.IsExpired() {
		defer tm.mu.RUnlock()
		return tm.token.AccessToken, nil
	}

	tm.mu.RUnlock()

	// Token missing or expired - refresh it
	if err := tm.RefreshToken(ctx); err != nil {
		return "", err
	}

	tm.mu.RLock()
	defer tm.mu.RUnlock()

	if tm.token == nil || tm.token.AccessToken == "" {
		return "", fmt.Errorf("oauth2: failed to obtain access token")
	}

	return tm.token.AccessToken, nil
}

// RefreshToken fetches a new access token using the Client Credentials flow.
// This method acquires an exclusive lock and should not be called directly
// in most cases - use GetToken instead.
// However, it can be called explicitly to force a token refresh.
func (tm *TokenManager) RefreshToken(ctx context.Context) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	token, err := tm.fetchToken(ctx)
	if err != nil {
		return err
	}

	tm.token = token
	return nil
}

// IsTokenValid checks if the current token is valid without acquiring the write lock.
// This is a non-blocking check for the validity of the cached token.
func (tm *TokenManager) IsTokenValid() bool {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	return tm.token != nil && !tm.token.IsExpired()
}

// GetCachedToken returns the currently cached token without checking expiry.
// Used primarily for testing or diagnostic purposes.
// For production code, use GetToken() instead.
func (tm *TokenManager) GetCachedToken() *Token {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	return tm.token
}

// InvalidateToken marks the current token as invalid.
// Useful when the server returns a 401 response indicating token expiry.
func (tm *TokenManager) InvalidateToken() {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	tm.token = nil
}

// fetchToken performs the actual OAuth2 token exchange using Client Credentials flow.
// It sends a POST request to the token endpoint with the client credentials.
// The token endpoint response is parsed and a Token is returned with expiry calculated.
func (tm *TokenManager) fetchToken(ctx context.Context) (*Token, error) {
	if tm.config == nil {
		return nil, fmt.Errorf("oauth2: config not set")
	}

	if tm.config.TokenURL == "" {
		return nil, fmt.Errorf("oauth2: token URL not configured")
	}

	if tm.config.ClientID == "" || tm.config.ClientSecret == "" {
		return nil, fmt.Errorf("oauth2: client credentials not configured")
	}

	// Build the request body using application/x-www-form-urlencoded format
	data := url.Values{}
	data.Set("grant_type", "client_credentials")
	data.Set("client_id", tm.config.ClientID)
	data.Set("client_secret", tm.config.ClientSecret)

	if len(tm.config.Scopes) > 0 {
		scopes := ""
		for i, scope := range tm.config.Scopes {
			if i > 0 {
				scopes += " "
			}
			scopes += scope
		}
		data.Set("scope", scopes)
	}

	// Create the HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", tm.config.TokenURL, bytes.NewBufferString(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("oauth2: failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	// Execute the request
	resp, err := tm.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("oauth2: failed to fetch token: %w", err)
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("oauth2: failed to read response: %w", err)
	}

	// Check for HTTP errors
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("oauth2: token endpoint returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse the token response
	var token Token
	if err := json.Unmarshal(body, &token); err != nil {
		return nil, fmt.Errorf("oauth2: failed to parse token response: %w", err)
	}

	if token.AccessToken == "" {
		return nil, fmt.Errorf("oauth2: no access token in response")
	}

	// Set the expiry time based on expires_in
	if token.ExpiresIn > 0 {
		token.ExpiresAt = time.Now().Add(time.Duration(token.ExpiresIn) * time.Second)
	}

	return &token, nil
}
