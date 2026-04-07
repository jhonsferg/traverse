package graphql

import (
	"context"
	"net/http"
	"net/http/httptest"
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

// --- Translate / ToODataParams tests ---

func TestTranslate_SimpleFields(t *testing.T) {
	q, err := Translate(`{ customers { id name } }`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if q.EntitySet != "customers" {
		t.Errorf("EntitySet: got %q, want %q", q.EntitySet, "customers")
	}
	if len(q.Fields) != 2 || q.Fields[0] != "id" || q.Fields[1] != "name" {
		t.Errorf("Fields: got %v, want [id name]", q.Fields)
	}
}

func TestTranslate_WithFilter(t *testing.T) {
	q, err := Translate(`{ orders(filter: "Status eq 'Open'") { id } }`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if q.Filter != "Status eq 'Open'" {
		t.Errorf("Filter: got %q, want %q", q.Filter, "Status eq 'Open'")
	}
	if q.EntitySet != "orders" {
		t.Errorf("EntitySet: got %q, want %q", q.EntitySet, "orders")
	}
}

func TestTranslate_WithTopSkip(t *testing.T) {
	q, err := Translate(`{ products(top: 5, skip: 10) { name } }`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if q.Top != 5 {
		t.Errorf("Top: got %d, want 5", q.Top)
	}
	if q.Skip != 10 {
		t.Errorf("Skip: got %d, want 10", q.Skip)
	}
}

func TestTranslate_NestedExpand(t *testing.T) {
	q, err := Translate(`{ orders { id customer { name email } } }`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(q.Expand) != 1 || q.Expand[0] != "customer" {
		t.Errorf("Expand: got %v, want [customer]", q.Expand)
	}
	subFields := q.ExpandFields["customer"]
	if len(subFields) != 2 || subFields[0] != "name" || subFields[1] != "email" {
		t.Errorf("ExpandFields[customer]: got %v, want [name email]", subFields)
	}

	params := q.ToODataParams()
	expand, ok := params["$expand"]
	if !ok {
		t.Fatal("$expand missing from OData params")
	}
	if expand != "customer($select=name,email)" {
		t.Errorf("$expand: got %q, want %q", expand, "customer($select=name,email)")
	}
}

func TestToODataParams_Select(t *testing.T) {
	q, err := Translate(`{ customers { id name country } }`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	params := q.ToODataParams()
	sel, ok := params["$select"]
	if !ok {
		t.Fatal("$select missing from OData params")
	}
	if sel != "id,name,country" {
		t.Errorf("$select: got %q, want %q", sel, "id,name,country")
	}
}

func TestToODataParams_Filter(t *testing.T) {
	q, err := Translate(`{ orders(filter: "Status eq 'Open'") { id } }`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	params := q.ToODataParams()
	filter, ok := params["$filter"]
	if !ok {
		t.Fatal("$filter missing from OData params")
	}
	if filter != "Status eq 'Open'" {
		t.Errorf("$filter: got %q, want %q", filter, "Status eq 'Open'")
	}
}

func TestTranslate_InvalidQuery(t *testing.T) {
	cases := []string{
		"",
		"{}",
		"not a query",
		"{ }",
	}
	for _, c := range cases {
		_, err := Translate(c)
		if err == nil {
			t.Errorf("Translate(%q): expected error, got nil", c)
		}
	}
}

func TestTranslate_WithOrderBy(t *testing.T) {
	q, err := Translate(`{ products(orderBy: "Name asc") { name price } }`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if q.OrderBy != "Name asc" {
		t.Errorf("OrderBy: got %q, want %q", q.OrderBy, "Name asc")
	}
	params := q.ToODataParams()
	if params["$orderby"] != "Name asc" {
		t.Errorf("$orderby: got %q, want %q", params["$orderby"], "Name asc")
	}
}

func TestExecutor_Execute(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"value":[]}`))
	}))
	defer ts.Close()

	exec := NewExecutor(ts.URL, ts.Client())
	body, err := exec.Execute(context.Background(), `{ orders { id status } }`)
	if err != nil {
		t.Fatalf("Execute: unexpected error: %v", err)
	}
	if string(body) != `{"value":[]}` {
		t.Errorf("body: got %q, want %q", string(body), `{"value":[]}`)
	}
}

func TestExecutor_Execute_InvalidQuery(t *testing.T) {
	exec := NewExecutor("http://localhost", nil)
	_, err := exec.Execute(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty query")
	}
}
