package graphql

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// Executor executes translated GraphQL queries against an OData endpoint.
type Executor struct {
	client  *http.Client
	baseURL string
}

// NewExecutor creates a new Executor targeting the given OData base URL.
// If client is nil, http.DefaultClient is used.
func NewExecutor(baseURL string, client *http.Client) *Executor {
	if client == nil {
		client = http.DefaultClient
	}
	return &Executor{
		client:  client,
		baseURL: baseURL,
	}
}

// Execute translates the GraphQL query and fetches the OData response.
// Returns the raw JSON response bytes.
func (e *Executor) Execute(ctx context.Context, gqlQuery string) ([]byte, error) {
	q, err := Translate(gqlQuery)
	if err != nil {
		return nil, fmt.Errorf("graphql: translate: %w", err)
	}

	u, err := url.Parse(e.baseURL)
	if err != nil {
		return nil, fmt.Errorf("graphql: invalid base URL: %w", err)
	}
	u.Path = strings.TrimRight(u.Path, "/") + "/" + q.EntitySet

	qv := url.Values{}
	for k, v := range q.ToODataParams() {
		qv.Set(k, v)
	}
	u.RawQuery = qv.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("graphql: create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := e.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("graphql: execute request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("graphql: read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("graphql: unexpected status %d: %s", resp.StatusCode, string(body))
	}

	return body, nil
}
