package traverse

import (
	"fmt"

	"github.com/jhonsferg/relay"
)

type GraphConfig struct {
	Version          string
	AccessToken      string
	ConsistencyLevel string
}

type GraphError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *GraphError) Error() string {
	if e.Code != "" && e.Message != "" {
		return fmt.Sprintf("traverse: Graph error %s: %s", e.Code, e.Message)
	}
	if e.Message != "" {
		return fmt.Sprintf("traverse: %s", e.Message)
	}
	return "traverse: unknown Graph error"
}

func NewGraphClient(relayClient *relay.Client, cfg GraphConfig) *Client {
	version := cfg.Version
	if version == "" {
		version = "v1.0"
	}

	baseURL := fmt.Sprintf("https://graph.microsoft.com/%s", version)

	opts := []Option{
		WithODataVersion(ODataV4),
		WithODataErrors(),
		WithHeader("Authorization", fmt.Sprintf("Bearer %s", cfg.AccessToken)),
	}

	// Inject the caller-provided relay client so its transport configuration
	// (TLS, timeouts, circuit breakers, tracing hooks, etc.) is honoured.
	// WithBaseURL must come after WithRelayClient so the Graph endpoint URL
	// takes precedence over whatever base URL the relay client was built with.
	if relayClient != nil {
		opts = append(opts, WithRelayClient(relayClient))
	}
	opts = append(opts, WithBaseURL(baseURL))

	if cfg.ConsistencyLevel != "" {
		opts = append(opts, WithHeader("ConsistencyLevel", cfg.ConsistencyLevel))
	}

	client, err := New(opts...)
	if err != nil {
		// New() only fails on programmer errors (e.g. empty base URL). The options
		// above are always valid, so this path should never be reached in practice.
		panic(fmt.Sprintf("traverse: NewGraphClient: %v", err))
	}
	return client
}
