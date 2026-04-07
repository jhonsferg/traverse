// Package sap provides SAP-specific OData adaptations and helpers.
package sap

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/url"
	"time"

	"github.com/jhonsferg/relay"
	traverse "github.com/jhonsferg/traverse"
	"github.com/jhonsferg/traverse/ext/oauth2"
)

// NewSAPClient creates a traverse Client preconfigured for SAP systems.
// Handles CSRF tokens, authentication, and SAP-specific conventions automatically.
func NewSAPClient(opts ...SAPOption) (*traverse.Client, error) {
	cfg := &sapConfig{
		version:   2, // SAP typically uses v2
		pageSize:  1000,
		language:  "EN",
		csrfToken: "", // Will be fetched on first write
	}

	// Apply SAP-specific options
	for _, opt := range opts {
		if err := opt(cfg); err != nil {
			return nil, fmt.Errorf("sap: invalid option: %w", err)
		}
	}

	// Create CSRF middleware for write operations
	// Note: We'll create it before the relay client so we can inject it
	var csrfMiddleware *CSRFMiddleware
	var tokenManager *oauth2.TokenManager

	// Build the relay client with SAP-specific configuration
	relayOpts := []relay.Option{
		relay.WithBaseURL(cfg.baseURL),
		relay.WithTimeout(30 * time.Second),
	}

	// Apply custom TLS config when provided (e.g. InsecureSkipVerify for DEV/QAS).
	if cfg.tlsConfig != nil {
		relayOpts = append(relayOpts, relay.WithTLSConfig(cfg.tlsConfig))
	}

	// Add SAP language header
	relayOpts = append(relayOpts,
		relay.WithDefaultHeaders(map[string]string{
			"sap-language": cfg.language,
		}),
	)

	// Add authentication (OAuth2 or Basic Auth)
	if cfg.oauth2URL != "" && cfg.oauth2ID != "" && cfg.oauth2Secret != "" {
		// OAuth2 configuration provided
		oauth2Config := &oauth2.OAuth2Config{
			ClientID:     cfg.oauth2ID,
			ClientSecret: cfg.oauth2Secret,
			TokenURL:     cfg.oauth2URL,
			Scopes:       cfg.oauth2Scopes,
		}
		tokenManager = oauth2.NewTokenManager(oauth2Config)
	}

	// Add CSRF token injection and OAuth2 auth via hooks
	relayOpts = append(relayOpts,
		relay.WithOnBeforeRequest(func(ctx context.Context, req *relay.Request) error {
			// Inject OAuth2 bearer token if configured
			if tokenManager != nil {
				token, err := tokenManager.GetToken(ctx)
				if err != nil {
					return fmt.Errorf("traverse: failed to get oauth2 token: %w", err)
				}
				req.WithHeader("Authorization", "Bearer "+token)
			}

			// Inject CSRF token for write operations
			if csrfMiddleware != nil {
				// Only inject for write operations
				method := req.Method()
				if method != "POST" && method != "PATCH" && method != "PUT" && method != "DELETE" {
					return nil
				}

				// Get a valid token
				csrfToken, err := csrfMiddleware.GetToken(ctx)
				if err != nil {
					return fmt.Errorf("traverse: failed to get csrf token: %w", err)
				}

				// Inject token header (modifies in-place)
				req.WithHeader("X-CSRF-Token", csrfToken)
			}
			return nil
		}),
		relay.WithOnAfterResponse(func(ctx context.Context, resp *relay.Response) error {
			// Handle 401 Unauthorized - invalidate OAuth2 token
			if tokenManager != nil && resp != nil && resp.StatusCode == 401 {
				tokenManager.InvalidateToken()
			}

			// Handle CSRF token errors
			if csrfMiddleware != nil {
				return csrfMiddleware.HandleResponse(ctx, resp, nil)
			}
			return nil
		}),
	)

	// Create the relay client
	relayClient := relay.New(relayOpts...)

	// Now create the CSRF middleware with the relay client
	csrfMiddleware = NewCSRFMiddleware(relayClient, cfg.baseURL)

	// Create the traverse client
	traverseOpts := []traverse.Option{
		traverse.WithBaseURL(cfg.baseURL),
		traverse.WithODataVersion(traverse.ODataVersion(cfg.version)),
		traverse.WithPageSize(cfg.pageSize),
		traverse.WithRelayClient(relayClient),
	}

	if cfg.formatJSON {
		traverseOpts = append(traverseOpts,
			traverse.WithFormat(traverse.FormatJSON),
		)
	}

	return traverse.New(traverseOpts...)
}

// SAPOption is a functional option for SAP client configuration.
type SAPOption func(*sapConfig) error

// sapConfig holds SAP-specific configuration.
type sapConfig struct {
	baseURL       string
	client        string // SAP client number (e.g., "100")
	service       string // Service name (e.g., "MM_MATERIAL_SRV")
	system        string // System name (e.g., "S4H")
	version       int    // OData version (2 or 4)
	pageSize      int
	language      string
	formatJSON    bool
	basicAuthUser string
	basicAuthPass string
	oauth2URL     string
	oauth2ID      string
	oauth2Secret  string
	oauth2Scopes  []string
	oauth2Token   string
	csrfToken     string
	tlsConfig     *tls.Config // optional custom TLS config (e.g. InsecureSkipVerify for DEV/QAS)
}

// WithSAPBaseURL configures the SAP system URL components.
// Builds: https://{systemURL}/sap/opu/odata/sap/{service}?sap-client={client}
func WithSAPBaseURL(systemURL, client, service string) SAPOption {
	return func(cfg *sapConfig) error {
		if systemURL == "" || service == "" {
			return fmt.Errorf("systemURL and service cannot be empty")
		}

		// Build the base URL
		baseURL := fmt.Sprintf("%s/sap/opu/odata/sap/%s", systemURL, service)

		// Add client parameter if provided
		if client != "" {
			baseURL += fmt.Sprintf("?sap-client=%s", url.QueryEscape(client))
			cfg.client = client
		}

		cfg.baseURL = baseURL
		cfg.system = systemURL
		cfg.service = service

		return nil
	}
}

// WithSAPBasicAuth sets basic authentication for SAP.
func WithSAPBasicAuth(user, pass string) SAPOption {
	return func(cfg *sapConfig) error {
		if user == "" || pass == "" {
			return fmt.Errorf("username and password cannot be empty")
		}
		cfg.basicAuthUser = user
		cfg.basicAuthPass = pass
		return nil
	}
}

// WithSAPOAuth2 configures OAuth2 Client Credentials for S/4HANA.
// The tokenManager will handle token refresh automatically.
// Calling this option enables OAuth2 authentication instead of basic auth.
func WithSAPOAuth2(tokenURL, clientID, secret string) SAPOption {
	return func(cfg *sapConfig) error {
		if tokenURL == "" || clientID == "" || secret == "" {
			return fmt.Errorf("tokenURL, clientID, and secret cannot be empty")
		}
		cfg.oauth2URL = tokenURL
		cfg.oauth2ID = clientID
		cfg.oauth2Secret = secret
		return nil
	}
}

// WithSAPOAuth2Scopes sets the OAuth2 scopes to request.
// This is optional and can be used to customize the token permissions.
func WithSAPOAuth2Scopes(scopes ...string) SAPOption {
	return func(cfg *sapConfig) error {
		if len(scopes) == 0 {
			return fmt.Errorf("at least one scope must be provided")
		}
		cfg.oauth2Scopes = scopes
		return nil
	}
}

// WithSAPLanguage sets the SAP language parameter.
func WithSAPLanguage(lang string) SAPOption {
	return func(cfg *sapConfig) error {
		if lang == "" {
			return fmt.Errorf("language cannot be empty")
		}
		cfg.language = lang
		return nil
	}
}

// WithSAPFormatJSON forces JSON format (default is ATOM/XML for v2).
func WithSAPFormatJSON() SAPOption {
	return func(cfg *sapConfig) error {
		cfg.formatJSON = true
		return nil
	}
}

// WithSAPMaxPageSize sets the maximum page size.
func WithSAPMaxPageSize(n int) SAPOption {
	return func(cfg *sapConfig) error {
		if n <= 0 {
			return fmt.Errorf("page size must be positive")
		}
		cfg.pageSize = n
		return nil
	}
}

// WithSAPODataVersion sets the OData version (v2 or v4).
func WithSAPODataVersion(v traverse.ODataVersion) SAPOption {
	return func(cfg *sapConfig) error {
		cfg.version = int(v)
		return nil
	}
}

// WithSAPTLSConfig sets a custom [crypto/tls.Config] for the underlying HTTP transport.
//
// Use this option when the SAP system presents a self-signed or internal CA
// certificate that is not trusted by the default system trust store  -  a common
// situation in DEV and QAS environments.
//
//	// DEV / QAS only  -  never use InsecureSkipVerify in production.
//	client, err := sap.NewSAPClient(
//	    sap.WithSAPBaseURL("https://s4h-dev.example.com:44300", "100", ""),
//	    sap.WithSAPBasicAuth("user", "pass"),
//	    sap.WithSAPTLSConfig(&tls.Config{InsecureSkipVerify: true}), // #nosec G402
//	)
//
// ⛔ Never pass InsecureSkipVerify: true to a production environment.
func WithSAPTLSConfig(cfg *tls.Config) SAPOption {
	return func(c *sapConfig) error {
		if cfg == nil {
			return fmt.Errorf("sap: WithSAPTLSConfig: tls.Config must not be nil")
		}
		c.tlsConfig = cfg
		return nil
	}
}
