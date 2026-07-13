package graphql

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jhonsferg/traverse"
)

const handlerTestEdmxXML = `<?xml version="1.0" encoding="utf-8"?>
<edmx:Edmx Version="1.0" xmlns:edmx="http://schemas.microsoft.com/ado/2007/06/edmx">
  <edmx:DataServices>
    <Schema Namespace="TestService" xmlns="http://schemas.microsoft.com/ado/2008/09/edm">
      <EntityType Name="Product">
        <Key><PropertyRef Name="ID"/></Key>
        <Property Name="ID" Type="Edm.Int32" Nullable="false"/>
        <Property Name="Name" Type="Edm.String"/>
      </EntityType>
      <EntityContainer Name="TestContainer" m:IsDefaultEntityContainer="true">
        <EntitySet Name="Products" EntityType="TestService.Product"/>
      </EntityContainer>
    </Schema>
  </edmx:DataServices>
</edmx:Edmx>`

// --- Execute: lazy schema build (success and failure) ---

func TestGraphQLServer_Execute_BuildSchemaSuccess(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		_, _ = w.Write([]byte(handlerTestEdmxXML))
	}))
	defer ts.Close()

	client, err := traverse.New(traverse.WithBaseURL(ts.URL + "/"))
	if err != nil {
		t.Fatalf("traverse.New() error = %v", err)
	}
	server, err := New(client)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	result := server.Execute(context.Background(), `{ __schema { types { name } } }`, nil)
	if result == nil {
		t.Fatal("Execute() returned nil result")
	}
	if !server.schemaBuilt {
		t.Error("expected schemaBuilt to be true after successful build")
	}
}

func TestGraphQLServer_Execute_BuildSchemaFailure(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer ts.Close()

	client, err := traverse.New(traverse.WithBaseURL(ts.URL + "/"))
	if err != nil {
		t.Fatalf("traverse.New() error = %v", err)
	}
	server, err := New(client)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	var loggedCalls int
	server.WithLogger(logFunc(func(string, ...interface{}) { loggedCalls++ }))

	result := server.Execute(context.Background(), `{ __schema { types { name } } }`, nil)
	if result == nil {
		t.Fatal("Execute() returned nil result")
	}
	if server.schemaBuilt {
		t.Error("expected schemaBuilt to remain false after failed build")
	}
	if loggedCalls == 0 {
		t.Error("expected logger to be called on build failure")
	}
}

// logFunc adapts a plain function to the server's logger interface.
type logFunc func(string, ...interface{})

func (f logFunc) Printf(format string, args ...interface{}) { f(format, args...) }

// --- noOpLogger ---

func TestNoOpLogger_Printf(t *testing.T) {
	// Calling Printf on the zero-value logger should be a no-op and never panic.
	noOpLogger{}.Printf("format %s", "arg")
}

func TestGraphQLServer_Execute_DefaultLoggerOnFailure(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer ts.Close()

	client, err := traverse.New(traverse.WithBaseURL(ts.URL + "/"))
	if err != nil {
		t.Fatalf("traverse.New() error = %v", err)
	}
	server, err := New(client)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Uses the default noOpLogger; just verify it doesn't panic.
	result := server.Execute(context.Background(), `{ __schema { types { name } } }`, nil)
	if result == nil {
		t.Fatal("Execute() returned nil result")
	}
}

// --- WithLogger nil ---

func TestGraphQLServer_WithLogger_Nil(t *testing.T) {
	client, err := traverse.New(traverse.WithBaseURL("https://example.com/odata"))
	if err != nil {
		t.Fatalf("traverse.New() error = %v", err)
	}
	server, err := New(client)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	before := server.logger
	server.WithLogger(nil)
	if server.logger != before {
		t.Error("expected logger to remain unchanged when nil is passed")
	}
}

// --- Handler ---

func newHandlerTestServer(t *testing.T) *GraphQLServer {
	t.Helper()
	metadata := &traverse.Metadata{
		Version:   "1.0",
		Namespace: "Test",
		EntityTypes: []traverse.EntityType{
			{
				Name: "Product",
				Properties: []traverse.Property{
					{Name: "ID", Type: "Edm.Int32", Nullable: false},
				},
			},
		},
		EntitySets: []traverse.EntitySetInfo{
			{Name: "Products", EntityTypeName: "Product"},
		},
	}
	client, err := traverse.New(traverse.WithBaseURL("https://example.com/odata"))
	if err != nil {
		t.Fatalf("traverse.New() error = %v", err)
	}
	server, err := New(client)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	schema, err := NewSchemaBuilder(metadata, client).Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	server.schema = schema
	server.schemaBuilt = true
	return server
}

func TestGraphQLServer_Handler_MethodNotAllowed(t *testing.T) {
	server := newHandlerTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/graphql", nil)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

func TestGraphQLServer_Handler_InvalidJSON(t *testing.T) {
	server := newHandlerTestServer(t)
	req := httptest.NewRequest(http.MethodPost, "/graphql", bytes.NewBufferString("not json"))
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestGraphQLServer_Handler_Success(t *testing.T) {
	server := newHandlerTestServer(t)

	body, err := json.Marshal(map[string]interface{}{
		"query": `{ __schema { types { name } } }`,
	})
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/graphql", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
	var decoded map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("response is not valid JSON: %v (%s)", err, rec.Body.String())
	}
}
