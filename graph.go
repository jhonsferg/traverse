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

func NewGraphClient(relay *relay.Client, cfg GraphConfig) *Client {
	version := cfg.Version
	if version == "" {
		version = "v1.0"
	}

	baseURL := fmt.Sprintf("https://graph.microsoft.com/%s", version)

	opts := []Option{
		WithBaseURL(baseURL),
		WithODataVersion(ODataV4),
		WithHeader("Authorization", fmt.Sprintf("Bearer %s", cfg.AccessToken)),
		WithODataErrors(),
	}

	if cfg.ConsistencyLevel != "" {
		opts = append(opts, WithHeader("ConsistencyLevel", cfg.ConsistencyLevel))
	}

	client, _ := New(opts...)
	return client
}
