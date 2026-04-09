// Package graphql provides an experimental GraphQL gateway over OData for traverse.
//
// EXPERIMENTAL: This package is experimental and the API may change.
package graphql

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	gql "github.com/graphql-go/graphql"

	"github.com/jhonsferg/traverse"
)

// GraphQLServer represents a GraphQL server that wraps an OData client.
type GraphQLServer struct {
	client *traverse.Client
	logger interface{ Printf(string, ...interface{}) }

	// schemaMu protects schema and schemaBuilt. Using an explicit mutex
	// instead of sync.Once so that a transient buildSchema failure
	// (e.g. metadata endpoint temporarily unavailable) does not permanently
	// poison the server — subsequent Execute calls will retry.
	schemaMu    sync.Mutex
	schema      *gql.Schema
	schemaBuilt bool
}

// New creates a new GraphQL server with the given OData client.
func New(client *traverse.Client) (*GraphQLServer, error) {
	if client == nil {
		return nil, fmt.Errorf("graphql: client cannot be nil")
	}

	server := &GraphQLServer{
		client: client,
		logger: noOpLogger{},
	}

	return server, nil
}

// WithLogger sets a logger for debugging.
func (s *GraphQLServer) WithLogger(logger interface{ Printf(string, ...interface{}) }) *GraphQLServer {
	if logger != nil {
		s.logger = logger
	}
	return s
}

// Handler returns an http.Handler that serves GraphQL queries.
func (s *GraphQLServer) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			Query         string                 `json:"query"`
			Variables     map[string]interface{} `json:"variables"`
			OperationName string                 `json:"operationName"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
			return
		}

		result := s.Execute(r.Context(), req.Query, req.Variables)
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(result); err != nil {
			s.logger.Printf("failed to encode response: %v", err)
		}
	})
}

// Execute executes a GraphQL query against the schema.
// The schema is built lazily on the first successful call. If schema
// construction fails (e.g. metadata endpoint unreachable), the error is
// logged and an empty result is returned; the next call will retry.
func (s *GraphQLServer) Execute(ctx context.Context, query string, variables map[string]interface{}) *gql.Result {
	s.schemaMu.Lock()
	if !s.schemaBuilt {
		if err := s.buildSchema(ctx); err != nil {
			s.schemaMu.Unlock()
			s.logger.Printf("failed to build schema: %v", err)
			return &gql.Result{}
		}
		s.schemaBuilt = true
	}
	schema := *s.schema
	s.schemaMu.Unlock()

	params := gql.Params{
		Schema:         schema,
		RequestString:  query,
		VariableValues: variables,
		Context:        ctx,
	}

	return gql.Do(params)
}

// buildSchema generates a GraphQL schema from OData metadata.
func (s *GraphQLServer) buildSchema(ctx context.Context) error {
	metadata, err := s.client.Metadata(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch metadata: %w", err)
	}

	builder := NewSchemaBuilder(metadata, s.client)
	schema, err := builder.Build()
	if err != nil {
		return fmt.Errorf("failed to build schema: %w", err)
	}

	s.schema = schema
	return nil
}

// noOpLogger is a no-op logger.
type noOpLogger struct{}

func (noOpLogger) Printf(string, ...interface{}) {}
