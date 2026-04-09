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
	"strings"
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

// IsExpired reports whether the token is expired or will expire within 30 s.
func (t *Token) IsExpired() bool {
	if t == nil || t.AccessToken == "" {
		return true
	}
	return time.Now().Add(30 * time.Second).After(t.ExpiresAt)
}

// inflightFetch tracks an in-progress token fetch so that concurrent callers
// can wait for the result of a single HTTP request instead of all racing to
// make their own. The channel is closed once the fetch completes.
type inflightFetch struct {
	done chan struct{}
	err  error
}

// TokenManager manages OAuth2 token lifecycle with caching and refresh.
// It is thread-safe and handles concurrent access to the token.
// Only one goroutine makes the token endpoint HTTP request at a time; all
// others wait for the result via an inflight channel (singleflight pattern).
type TokenManager struct {
	config *OAuth2Config
	client *http.Client

	mu       sync.Mutex
	token    *Token
	inflight *inflightFetch // non-nil when a fetch is in progress
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
func (tm *TokenManager) GetToken(ctx context.Context) (string, error) {
	tm.mu.Lock()
	if tm.token != nil && !tm.token.IsExpired() {
		t := tm.token.AccessToken
		tm.mu.Unlock()
		return t, nil
	}
	tm.mu.Unlock()

	if err := tm.RefreshToken(ctx); err != nil {
		return "", err
	}

	tm.mu.Lock()
	defer tm.mu.Unlock()

	if tm.token == nil || tm.token.AccessToken == "" {
		return "", fmt.Errorf("oauth2: failed to obtain access token")
	}
	return tm.token.AccessToken, nil
}

// RefreshToken fetches a new access token using the Client Credentials flow.
// Only one goroutine makes the HTTP call at a time; concurrent callers wait
// for the in-flight request to complete (singleflight pattern). The HTTP
// request is performed WITHOUT holding the internal mutex.
func (tm *TokenManager) RefreshToken(ctx context.Context) error {
	tm.mu.Lock()

	// Double-check: another goroutine may have refreshed the token already.
	if tm.token != nil && !tm.token.IsExpired() {
		tm.mu.Unlock()
		return nil
	}

	// A fetch is already in progress — wait for it without holding the lock.
	if tm.inflight != nil {
		inflight := tm.inflight
		tm.mu.Unlock()
		select {
		case <-inflight.done:
			return inflight.err
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	// We are the leader: register an in-flight marker and release the lock
	// BEFORE making the HTTP call so other goroutines are not blocked.
	f := &inflightFetch{done: make(chan struct{})}
	tm.inflight = f
	tm.mu.Unlock()

	token, err := tm.fetchToken(ctx)

	tm.mu.Lock()
	if err == nil {
		tm.token = token
	}
	f.err = err
	tm.inflight = nil
	close(f.done)
	tm.mu.Unlock()

	return err
}

// IsTokenValid reports whether the cached token is still valid.
func (tm *TokenManager) IsTokenValid() bool {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	return tm.token != nil && !tm.token.IsExpired()
}

// GetCachedToken returns the currently cached token without checking expiry.
// For production code, use GetToken() instead.
func (tm *TokenManager) GetCachedToken() *Token {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	return tm.token
}

// InvalidateToken clears the cached token, forcing a refresh on the next call.
func (tm *TokenManager) InvalidateToken() {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	tm.token = nil
}

// fetchToken performs the actual OAuth2 token exchange using Client Credentials flow.
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

	data := url.Values{}
	data.Set("grant_type", "client_credentials")
	data.Set("client_id", tm.config.ClientID)
	data.Set("client_secret", tm.config.ClientSecret)
	if len(tm.config.Scopes) > 0 {
		data.Set("scope", strings.Join(tm.config.Scopes, " "))
	}

	req, err := http.NewRequestWithContext(ctx, "POST", tm.config.TokenURL, bytes.NewBufferString(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("oauth2: failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := tm.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("oauth2: failed to fetch token: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("oauth2: failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		// Try to extract a structured RFC 6749 error before falling back to a
		// status-only message. Never echo the raw body — it may contain
		// sensitive diagnostic data or partial credential information.
		var errResp struct {
			Error     string `json:"error"`
			ErrorDesc string `json:"error_description"`
		}
		if jsonErr := json.Unmarshal(body, &errResp); jsonErr == nil && errResp.Error != "" {
			if errResp.ErrorDesc != "" {
				return nil, fmt.Errorf("oauth2: token endpoint returned status %d: %s: %s",
					resp.StatusCode, errResp.Error, errResp.ErrorDesc)
			}
			return nil, fmt.Errorf("oauth2: token endpoint returned status %d: %s",
				resp.StatusCode, errResp.Error)
		}
		return nil, fmt.Errorf("oauth2: token endpoint returned status %d", resp.StatusCode)
	}

	var token Token
	if err := json.Unmarshal(body, &token); err != nil {
		return nil, fmt.Errorf("oauth2: failed to parse token response: %w", err)
	}
	if token.AccessToken == "" {
		return nil, fmt.Errorf("oauth2: no access token in response")
	}

	if token.ExpiresIn > 0 {
		token.ExpiresAt = time.Now().Add(time.Duration(token.ExpiresIn) * time.Second)
	}

	return &token, nil
}
