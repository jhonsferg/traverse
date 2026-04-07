// Package dataverse provides a Microsoft Dataverse (OData v4) adapter for traverse.
package dataverse

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// Config holds Dataverse connection settings.
type Config struct {
	// OrgURL is the Dataverse organisation URL, e.g. https://myorg.api.crm.dynamics.com
	OrgURL string
	// APIVersion is the Dataverse API version, defaults to "9.2".
	APIVersion string
	// BearerToken provides the OAuth2 access token. Called before each request.
	// If nil, requests are sent without authorization (useful for testing).
	BearerToken func() (string, error)
	// CallerID (MSCRMCallerID) optionally impersonates a user by their system user GUID.
	CallerID string
	// MaxPageSize sets the Prefer: odata.maxpagesize header. Defaults to 100.
	MaxPageSize int
}

// Client wraps an HTTP client preconfigured for Dataverse.
type Client struct {
	inner  *http.Client
	config Config
}

// New creates a Dataverse client.
func New(cfg Config) (*Client, error) {
	if cfg.OrgURL == "" {
		return nil, fmt.Errorf("dataverse: OrgURL is required")
	}
	if cfg.APIVersion == "" {
		cfg.APIVersion = "9.2"
	}
	if cfg.MaxPageSize == 0 {
		cfg.MaxPageSize = 100
	}
	return &Client{
		inner:  &http.Client{Timeout: 30 * time.Second},
		config: cfg,
	}, nil
}

// ServiceURL returns the full API base URL, e.g. https://myorg.api.crm.dynamics.com/api/data/v9.2/
func (c *Client) ServiceURL() string {
	base := strings.TrimRight(c.config.OrgURL, "/")
	return fmt.Sprintf("%s/api/data/v%s/", base, c.config.APIVersion)
}

// Do executes a raw HTTP request with Dataverse-specific headers applied.
//
//nolint:gosec // G704: SSRF is expected; callers control the request URL.
func (c *Client) Do(req *http.Request) (*http.Response, error) {
	req.Header.Set("OData-MaxVersion", "4.0")
	req.Header.Set("OData-Version", "4.0")
	req.Header.Set("Accept", "application/json; odata.metadata=minimal")

	if c.config.BearerToken != nil {
		token, err := c.config.BearerToken()
		if err != nil {
			return nil, fmt.Errorf("dataverse: failed to get bearer token: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+token)
	}

	if c.config.CallerID != "" {
		req.Header.Set("MSCRMCallerID", c.config.CallerID)
	}

	return c.inner.Do(req)
}

// QueryOption configures an OData query.
type QueryOption func(url.Values)

// Select applies a $select OData query option.
func Select(fields ...string) QueryOption {
	return func(v url.Values) {
		v.Set("$select", strings.Join(fields, ","))
	}
}

// Filter applies a $filter OData query option.
func Filter(expr string) QueryOption {
	return func(v url.Values) {
		v.Set("$filter", expr)
	}
}

// OrderBy applies a $orderby OData query option.
func OrderBy(field string, desc bool) QueryOption {
	return func(v url.Values) {
		val := field
		if desc {
			val += " desc"
		}
		v.Set("$orderby", val)
	}
}

// Top applies a $top OData query option.
func Top(n int) QueryOption {
	return func(v url.Values) {
		v.Set("$top", fmt.Sprintf("%d", n))
	}
}

// Expand applies a $expand OData query option.
func Expand(nav string) QueryOption {
	return func(v url.Values) {
		v.Set("$expand", nav)
	}
}

// List retrieves a page of entities from the given entity set.
// Returns the raw JSON response bytes.
func (c *Client) List(ctx context.Context, entitySet string, opts ...QueryOption) ([]byte, error) {
	u, err := url.Parse(c.ServiceURL() + entitySet)
	if err != nil {
		return nil, fmt.Errorf("dataverse: invalid URL: %w", err)
	}

	q := u.Query()
	for _, opt := range opts {
		opt(q)
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("dataverse: failed to create request: %w", err)
	}
	req.Header.Set("Prefer", fmt.Sprintf("odata.maxpagesize=%d", c.config.MaxPageSize))

	resp, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("dataverse: request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("dataverse: failed to read response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("dataverse: unexpected status %d: %s", resp.StatusCode, body)
	}
	return body, nil
}

// Get retrieves a single entity by ID.
func (c *Client) Get(ctx context.Context, entitySet, id string, opts ...QueryOption) ([]byte, error) {
	u, err := url.Parse(fmt.Sprintf("%s%s(%s)", c.ServiceURL(), entitySet, id))
	if err != nil {
		return nil, fmt.Errorf("dataverse: invalid URL: %w", err)
	}

	q := u.Query()
	for _, opt := range opts {
		opt(q)
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("dataverse: failed to create request: %w", err)
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("dataverse: request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("dataverse: failed to read response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("dataverse: unexpected status %d: %s", resp.StatusCode, body)
	}
	return body, nil
}

// reEntityID extracts the GUID from an OData-EntityId header value such as
// https://org/api/data/v9.2/accounts(some-guid).
var reEntityID = regexp.MustCompile(`\(([^)]+)\)$`)

// Create creates a new entity. body is the JSON-encoded entity.
// Returns the ID of the created entity (from OData-EntityId response header).
func (c *Client) Create(ctx context.Context, entitySet string, body []byte) (string, error) {
	u := c.ServiceURL() + entitySet

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("dataverse: failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.Do(req)
	if err != nil {
		return "", fmt.Errorf("dataverse: request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("dataverse: failed to read response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("dataverse: unexpected status %d: %s", resp.StatusCode, respBody)
	}

	entityID := resp.Header.Get("OData-EntityId")
	if entityID == "" {
		return "", fmt.Errorf("dataverse: OData-EntityId header missing from response")
	}

	matches := reEntityID.FindStringSubmatch(entityID)
	if len(matches) < 2 {
		return "", fmt.Errorf("dataverse: could not extract ID from OData-EntityId: %s", entityID)
	}
	return matches[1], nil
}

// Update updates an entity using PATCH.
func (c *Client) Update(ctx context.Context, entitySet, id string, body []byte) error {
	u := fmt.Sprintf("%s%s(%s)", c.ServiceURL(), entitySet, id)

	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, u, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("dataverse: failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.Do(req)
	if err != nil {
		return fmt.Errorf("dataverse: request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("dataverse: unexpected status %d: %s", resp.StatusCode, b)
	}
	return nil
}

// Delete deletes an entity.
func (c *Client) Delete(ctx context.Context, entitySet, id string) error {
	u := fmt.Sprintf("%s%s(%s)", c.ServiceURL(), entitySet, id)

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, u, nil)
	if err != nil {
		return fmt.Errorf("dataverse: failed to create request: %w", err)
	}

	resp, err := c.Do(req)
	if err != nil {
		return fmt.Errorf("dataverse: request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("dataverse: unexpected status %d: %s", resp.StatusCode, b)
	}
	return nil
}
