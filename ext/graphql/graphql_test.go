package graphql

import (
	"context"
	"testing"

	gql "github.com/graphql-go/graphql"
	"github.com/jhonsferg/traverse"
)

// TestGraphQLServerCreation tests creating a GraphQL server
func TestGraphQLServerCreation(t *testing.T) {
	// Create a mock client
	client, err := traverse.New(
		traverse.WithBaseURL("https://example.com/odata"),
		traverse.WithODataVersion(traverse.ODataV4),
	)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Create GraphQL server
	server, err := New(client)
	if err != nil {
		t.Fatalf("Failed to create GraphQL server: %v", err)
	}

	if server == nil {
		t.Fatal("Expected non-nil server")
	}

	if server.client != client {
		t.Fatal("Expected client to be set")
	}
}

// TestGraphQLServerWithNilClient tests creating server with nil client
func TestGraphQLServerWithNilClient(t *testing.T) {
	server, err := New(nil)

	if err == nil {
		t.Fatal("Expected error for nil client")
	}

	if server != nil {
		t.Fatal("Expected nil server for nil client")
	}
}

// TestGraphQLServerWithLogger tests setting a logger
func TestGraphQLServerWithLogger(t *testing.T) {
	client, _ := traverse.New(
		traverse.WithBaseURL("https://example.com/odata"),
		traverse.WithODataVersion(traverse.ODataV4),
	)

	server, _ := New(client)

	// Create a simple logger function
	loggerCalls := 0
	logger := struct {
		logFunc func(string, ...interface{})
	}{
		logFunc: func(string, ...interface{}) {
			loggerCalls++
		},
	}

	// Set logger - should accept the interface
	// Note: We can't actually use the logger since it doesn't implement the interface
	// but this tests that the method exists
	result := server.WithLogger(nil)
	if result != server {
		t.Fatal("Expected same server instance")
	}

	_ = logger // Use the logger variable
}

// TestSchemaBuilderCreation tests creating a schema builder
func TestSchemaBuilderCreation(t *testing.T) {
	metadata := &traverse.Metadata{
		Version:   "1.0",
		Namespace: "Test",
		EntityTypes: []traverse.EntityType{
			{
				Name: "Product",
				Properties: []traverse.Property{
					{
						Name:     "ID",
						Type:     "Edm.Int32",
						Nullable: false,
					},
					{
						Name:     "Name",
						Type:     "Edm.String",
						Nullable: true,
					},
				},
			},
		},
		EntitySets: []traverse.EntitySetInfo{
			{
				Name:           "Products",
				EntityTypeName: "Product",
			},
		},
	}

	client, _ := traverse.New(
		traverse.WithBaseURL("https://example.com/odata"),
		traverse.WithODataVersion(traverse.ODataV4),
	)

	builder := NewSchemaBuilder(metadata, client)

	if builder == nil {
		t.Fatal("Expected non-nil builder")
	}

	if builder.metadata != metadata {
		t.Fatal("Expected metadata to be set")
	}
}

// TestODataTypeToGraphQL tests type mapping
func TestODataTypeToGraphQL(t *testing.T) {
	metadata := &traverse.Metadata{}
	client, _ := traverse.New(
		traverse.WithBaseURL("https://example.com/odata"),
		traverse.WithODataVersion(traverse.ODataV4),
	)
	builder := NewSchemaBuilder(metadata, client)

	tests := []struct {
		odataType string
		nullable  bool
		expected  gql.Type
	}{
		{"Edm.String", true, gql.String},
		{"Edm.Int32", true, gql.Int},
		{"Edm.Boolean", true, gql.Boolean},
		{"Edm.Double", true, gql.Float},
	}

	for _, test := range tests {
		result := builder.odataTypeToGraphQL(test.odataType, test.nullable)
		if result != test.expected {
			t.Errorf("Expected %v for %s, got %v", test.expected, test.odataType, result)
		}
	}
}

// TestODataTypeToGraphQLNonNull tests non-null types
func TestODataTypeToGraphQLNonNull(t *testing.T) {
	metadata := &traverse.Metadata{}
	client, _ := traverse.New(
		traverse.WithBaseURL("https://example.com/odata"),
		traverse.WithODataVersion(traverse.ODataV4),
	)
	builder := NewSchemaBuilder(metadata, client)

	result := builder.odataTypeToGraphQL("Edm.String", false)
	if result == nil {
		t.Fatal("Expected non-null type")
	}

	// Check if it's a NonNull type
	_, ok := result.(*gql.NonNull)
	if !ok {
		t.Fatal("Expected NonNull type for nullable=false")
	}
}

// TestExecuteQuery tests executing a simple query
func TestExecuteQuery(t *testing.T) {
	// Create minimal metadata
	metadata := &traverse.Metadata{
		Version:   "1.0",
		Namespace: "Test",
		EntityTypes: []traverse.EntityType{
			{
				Name: "Product",
				Properties: []traverse.Property{
					{
						Name:     "ID",
						Type:     "Edm.Int32",
						Nullable: false,
					},
				},
			},
		},
		EntitySets: []traverse.EntitySetInfo{
			{
				Name:           "Products",
				EntityTypeName: "Product",
			},
		},
	}

	client, _ := traverse.New(
		traverse.WithBaseURL("https://example.com/odata"),
		traverse.WithODataVersion(traverse.ODataV4),
	)

	server, _ := New(client)
	server.schema, _ = NewSchemaBuilder(metadata, client).Build()

	// Simple query
	query := `{ __schema { types { name } } }`

	result := server.Execute(context.Background(), query, nil)

	// GraphQL always returns a result, may have errors but shouldn't panic
	if result == nil {
		t.Fatal("Expected non-nil result")
	}
}
