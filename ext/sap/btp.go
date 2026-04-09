package sap

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	traverse "github.com/jhonsferg/traverse"
	"github.com/jhonsferg/traverse/ext/oauth2"
)

// XSUAABinding represents a parsed XSUAA service binding from VCAP_SERVICES.
// XSUAA (Extended Services for UAA) is the identity and access management service
// in SAP Business Technology Platform (BTP). The binding contains OAuth2 credentials
// and endpoint information for token exchange.
type XSUAABinding struct {
	ClientID     string
	ClientSecret string
	TokenURL     string // {url}/oauth/token
	APIURL       string // base URL of the BTP service
	ZoneID       string // identity zone / subaccount ID
}

// vcapXSUAACredentials represents the credentials structure within VCAP_SERVICES.
type vcapXSUAACredentials struct {
	ClientID       string `json:"clientid"`
	ClientSecret   string `json:"clientsecret"`
	URL            string `json:"url"`
	APIURL         string `json:"apiurl,omitempty"`
	IdentityZone   string `json:"identityzone,omitempty"`
	IdentityZoneID string `json:"identityzoneid,omitempty"`
}

// vcapServiceBinding represents a single service binding in VCAP_SERVICES.
type vcapServiceBinding struct {
	Credentials vcapXSUAACredentials `json:"credentials"`
}

// ParseVCAPServices parses a VCAP_SERVICES JSON string and extracts the XSUAA binding.
// serviceName is the label of the service binding (e.g. "xsuaa", "business-rules").
// If the service is not found or the JSON is invalid, an error is returned.
func ParseVCAPServices(vcapJSON string, serviceName string) (*XSUAABinding, error) {
	if vcapJSON == "" {
		return nil, fmt.Errorf("btp: vcap json is empty")
	}

	if serviceName == "" {
		serviceName = "xsuaa"
	}

	// Parse the JSON
	var vcapData map[string][]vcapServiceBinding
	if err := json.Unmarshal([]byte(vcapJSON), &vcapData); err != nil {
		return nil, fmt.Errorf("btp: failed to parse vcap json: %w", err)
	}

	// Look for the requested service
	services, exists := vcapData[serviceName]
	if !exists || len(services) == 0 {
		return nil, fmt.Errorf("btp: service '%s' not found in vcap_services", serviceName)
	}

	creds := services[0].Credentials

	// Validate required fields
	if creds.ClientID == "" || creds.ClientSecret == "" || creds.URL == "" {
		return nil, fmt.Errorf("btp: missing required credentials (clientid, clientsecret, url)")
	}

	// Build token URL
	tokenURL := creds.URL + "/oauth/token"

	// Determine Zone ID
	zoneID := creds.IdentityZoneID
	if zoneID == "" {
		zoneID = creds.IdentityZone
	}

	binding := &XSUAABinding{
		ClientID:     creds.ClientID,
		ClientSecret: creds.ClientSecret,
		TokenURL:     tokenURL,
		APIURL:       creds.APIURL,
		ZoneID:       zoneID,
	}

	return binding, nil
}

// ParseVCAPServicesEnv reads VCAP_SERVICES from the environment and parses it.
// serviceName defaults to "xsuaa" if not specified.
func ParseVCAPServicesEnv(serviceName string) (*XSUAABinding, error) {
	vcapJSON := os.Getenv("VCAP_SERVICES")
	if vcapJSON == "" {
		return nil, fmt.Errorf("btp: VCAP_SERVICES environment variable not set")
	}

	return ParseVCAPServices(vcapJSON, serviceName)
}

// BTPTokenProvider fetches OAuth2 tokens from XSUAA using client credentials flow.
// It wraps the oauth2.TokenManager and provides BTP-specific token management.
type BTPTokenProvider struct {
	binding       XSUAABinding
	mu            sync.Mutex
	tokenManager  *oauth2.TokenManager
	lastError     error
	errorTime     time.Time
	errorResetTTL time.Duration
}

// NewBTPTokenProvider creates a token provider from an XSUAA binding.
func NewBTPTokenProvider(binding XSUAABinding) *BTPTokenProvider {
	oauth2Config := &oauth2.OAuth2Config{
		ClientID:     binding.ClientID,
		ClientSecret: binding.ClientSecret,
		TokenURL:     binding.TokenURL,
	}

	return &BTPTokenProvider{
		binding:       binding,
		tokenManager:  oauth2.NewTokenManager(oauth2Config),
		errorResetTTL: 30 * time.Second,
	}
}

// Token returns a valid Bearer token, refreshing if expired.
// The returned token is ready to use in the "Authorization: Bearer {token}" header.
// It is safe for concurrent use.
func (p *BTPTokenProvider) Token(ctx context.Context) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Check if we have a recent error and are still in the reset TTL window
	if p.lastError != nil && time.Since(p.errorTime) < p.errorResetTTL {
		return "", p.lastError
	}

	// Clear old errors
	if time.Since(p.errorTime) >= p.errorResetTTL {
		p.lastError = nil
	}

	// Get token from the manager
	token, err := p.tokenManager.GetToken(ctx)
	if err != nil {
		p.lastError = err
		p.errorTime = time.Now()
		return "", fmt.Errorf("btp: failed to get token from xsuaa: %w", err)
	}

	return token, nil
}

// NewBTPClient creates a traverse client pre-configured for SAP BTP.
// It reads VCAP_SERVICES from the environment and sets up XSUAA authentication.
// The returned client is fully configured with OAuth2 token injection and CSRF handling.
// Note: For this to work, the VCAP_SERVICES must contain a BTP service binding that includes
// the base URL or API URL for the SAP service endpoint.
func NewBTPClient(ctx context.Context, serviceURL string) (*traverse.Client, error) {
	binding, err := ParseVCAPServicesEnv("xsuaa")
	if err != nil {
		return nil, fmt.Errorf("btp: failed to parse vcap services: %w", err)
	}

	return NewBTPClientFromBinding(ctx, *binding, serviceURL)
}

// NewBTPClientFromBinding creates a traverse client from an explicit XSUAA binding.
// This is useful for testing or when VCAP_SERVICES is not available.
// serviceURL should be the OData service endpoint URL (e.g., "https://service.sap.example.com/odata").
// The returned client is fully configured with OAuth2 token injection and CSRF handling.
func NewBTPClientFromBinding(ctx context.Context, binding XSUAABinding, serviceURL string) (*traverse.Client, error) {
	if binding.TokenURL == "" {
		return nil, fmt.Errorf("btp: token url is empty")
	}

	if binding.ClientID == "" || binding.ClientSecret == "" {
		return nil, fmt.Errorf("btp: client credentials are empty")
	}

	if serviceURL == "" {
		return nil, fmt.Errorf("btp: service url is required")
	}

	// Create BTP token provider to verify we can authenticate
	tokenProvider := NewBTPTokenProvider(binding)

	// Ensure we have a valid token upfront
	if _, err := tokenProvider.Token(ctx); err != nil {
		return nil, fmt.Errorf("btp: initial token fetch failed: %w", err)
	}

	// Create a custom SAP option for BTP that sets base URL directly
	btpOption := func(cfg *sapConfig) error {
		cfg.baseURL = serviceURL
		return nil
	}

	// Create a SAP client with the BTP binding's OAuth2 configuration and service URL
	client, err := NewSAPClient(
		btpOption,
		WithSAPOAuth2(binding.TokenURL, binding.ClientID, binding.ClientSecret),
	)

	if err != nil {
		return nil, fmt.Errorf("btp: failed to create sap client: %w", err)
	}

	return client, nil
}
