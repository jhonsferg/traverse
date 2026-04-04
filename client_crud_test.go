package traverse

import (
	"context"
	"testing"
	"time"

	"github.com/jhonsferg/relay"

	"github.com/jhonsferg/traverse/testutil"
)

const edmxXML = `<?xml version="1.0" encoding="utf-8"?>
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

// ── Service ───────────────────────────────────────────────────────────────────

func TestClientService_V4(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{
		Status: 200,
		Body:   `{"@odata.context":"$metadata","value":[{"name":"Products","kind":"EntitySet","url":"Products"},{"name":"Orders","kind":"EntitySet","url":"Orders"}]}`,
	})

	client, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	doc, err := client.Service(context.Background())
	if err != nil {
		t.Fatalf("Service() failed: %v", err)
	}
	if doc == nil {
		t.Fatal("Service() returned nil document")
	}
	if len(doc.EntitySets) != 2 {
		t.Fatalf("expected 2 entity sets, got %d", len(doc.EntitySets))
	}
	if doc.EntitySets[0].Name != "Products" {
		t.Errorf("EntitySets[0].Name = %q, want %q", doc.EntitySets[0].Name, "Products")
	}
	if doc.EntitySets[1].Name != "Orders" {
		t.Errorf("EntitySets[1].Name = %q, want %q", doc.EntitySets[1].Name, "Orders")
	}
}

func TestClientService_V4_HTTPError(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{Status: 404, Body: `Not Found`})

	client, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	_, err = client.Service(context.Background())
	if err == nil {
		t.Fatal("Service() expected error for 404, got nil")
	}
}

func TestClientService_V2(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{
		Status: 200,
		Body:   `{"d":{"EntitySets":[{"Name":"Products","Url":"Products"},{"Name":"Orders","Url":"Orders"}]}}`,
	})

	client, err := New(WithBaseURL(server.URL()), WithODataVersion(ODataV2))
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	doc, err := client.Service(context.Background())
	if err != nil {
		t.Fatalf("Service() failed: %v", err)
	}
	if len(doc.EntitySets) != 2 {
		t.Fatalf("expected 2 entity sets, got %d", len(doc.EntitySets))
	}
	if doc.EntitySets[0].Name != "Products" {
		t.Errorf("EntitySets[0].Name = %q, want %q", doc.EntitySets[0].Name, "Products")
	}
}

// ── Metadata ──────────────────────────────────────────────────────────────────

func TestClientMetadata_Success(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{Status: 200, Body: edmxXML})

	client, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	meta, err := client.Metadata(context.Background())
	if err != nil {
		t.Fatalf("Metadata() failed: %v", err)
	}
	if meta == nil {
		t.Fatal("Metadata() returned nil")
	}
	if len(meta.EntityTypes) == 0 {
		t.Fatal("expected at least one entity type in metadata")
	}
	found := false
	for _, et := range meta.EntityTypes {
		if et.Name == "Product" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected entity type 'Product' in metadata")
	}
}

func TestClientMetadata_HTTPError(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{Status: 500, Body: `Internal Server Error`})

	client, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	_, err = client.Metadata(context.Background())
	if err == nil {
		t.Fatal("Metadata() expected error for 500, got nil")
	}
}

func TestClientMetadata_Cached(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	// Only enqueue one response — the second Metadata() call must use the cache.
	server.Enqueue(testutil.MockResponse{Status: 200, Body: edmxXML})

	client, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	ctx := context.Background()

	meta1, err := client.Metadata(ctx)
	if err != nil {
		t.Fatalf("first Metadata() failed: %v", err)
	}

	meta2, err := client.Metadata(ctx)
	if err != nil {
		t.Fatalf("second Metadata() failed: %v", err)
	}

	// Both calls should return the same pointer (cached).
	if meta1 != meta2 {
		t.Error("expected Metadata() to return the cached instance on second call")
	}

	// Only one HTTP request should have been made.
	if count := server.RequestCount(); count != 1 {
		t.Errorf("expected 1 HTTP request, got %d", count)
	}
}

// ── Create ────────────────────────────────────────────────────────────────────

func TestClientCreate_V4(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{
		Status: 201,
		Body:   `{"ID":42,"Name":"Widget"}`,
	})

	client, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	result, err := client.Create(context.Background(), "Products", map[string]interface{}{
		"Name": "Widget",
	})
	if err != nil {
		t.Fatalf("Create() failed: %v", err)
	}
	if result == nil {
		t.Fatal("Create() returned nil map")
	}
	if result["Name"] != "Widget" {
		t.Errorf("Name = %v, want Widget", result["Name"])
	}
}

func TestClientCreate_V4_ErrorStatus(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{Status: 400, Body: `{"error":{"message":"Bad Request"}}`})

	client, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	_, err = client.Create(context.Background(), "Products", map[string]interface{}{"Name": "X"})
	if err == nil {
		t.Fatal("Create() expected error for 400, got nil")
	}
}

func TestClientCreate_V2(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{
		Status: 201,
		Body:   `{"d":{"ID":7,"Name":"Gadget"}}`,
	})

	client, err := New(WithBaseURL(server.URL()), WithODataVersion(ODataV2))
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	result, err := client.Create(context.Background(), "Products", map[string]interface{}{
		"Name": "Gadget",
	})
	if err != nil {
		t.Fatalf("Create() (v2) failed: %v", err)
	}
	if result["Name"] != "Gadget" {
		t.Errorf("Name = %v, want Gadget", result["Name"])
	}
}

// ── Update ────────────────────────────────────────────────────────────────────

func TestClientUpdate_Success(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{Status: 204})

	client, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	err = client.Update(context.Background(), "Products", 1, map[string]interface{}{"Name": "Updated"})
	if err != nil {
		t.Fatalf("Update() failed: %v", err)
	}

	reqs := server.RecordedRequests()
	if len(reqs) != 1 {
		t.Fatalf("expected 1 request, got %d", len(reqs))
	}
	if reqs[0].Method != "PATCH" {
		t.Errorf("Method = %s, want PATCH", reqs[0].Method)
	}
}

func TestClientUpdate_ErrorStatus(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{Status: 409, Body: `{"error":"Conflict"}`})

	client, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	err = client.Update(context.Background(), "Products", 1, map[string]interface{}{"Name": "X"})
	if err == nil {
		t.Fatal("Update() expected error for 409, got nil")
	}
}

// ── Replace ───────────────────────────────────────────────────────────────────

func TestClientReplace_Success(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{Status: 204})

	client, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	err = client.Replace(context.Background(), "Products", 1, map[string]interface{}{"ID": 1, "Name": "Full"})
	if err != nil {
		t.Fatalf("Replace() failed: %v", err)
	}

	reqs := server.RecordedRequests()
	if len(reqs) != 1 {
		t.Fatalf("expected 1 request, got %d", len(reqs))
	}
	if reqs[0].Method != "PUT" {
		t.Errorf("Method = %s, want PUT", reqs[0].Method)
	}
}

func TestClientReplace_ErrorStatus(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{Status: 404, Body: `{"error":"Not Found"}`})

	client, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	err = client.Replace(context.Background(), "Products", 99, map[string]interface{}{"Name": "X"})
	if err == nil {
		t.Fatal("Replace() expected error for 404, got nil")
	}
}

// ── Delete ────────────────────────────────────────────────────────────────────

func TestClientDelete_Success(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{Status: 204})

	client, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	err = client.Delete(context.Background(), "Products", 1)
	if err != nil {
		t.Fatalf("Delete() failed: %v", err)
	}

	reqs := server.RecordedRequests()
	if len(reqs) != 1 {
		t.Fatalf("expected 1 request, got %d", len(reqs))
	}
	if reqs[0].Method != "DELETE" {
		t.Errorf("Method = %s, want DELETE", reqs[0].Method)
	}
}

func TestClientDelete_ErrorStatus(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{Status: 404, Body: `{"error":"Not Found"}`})

	client, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	err = client.Delete(context.Background(), "Products", 99)
	if err == nil {
		t.Fatal("Delete() expected error for 404, got nil")
	}
}

// ── Options ───────────────────────────────────────────────────────────────────

func TestWithBearerToken(t *testing.T) {
	_, err := New(
		WithBaseURL("http://example.com"),
		WithBearerToken("my-token-abc"),
	)
	if err != nil {
		t.Fatalf("New() with WithBearerToken failed: %v", err)
	}
}

func TestWithBearerToken_Empty(t *testing.T) {
	_, err := New(
		WithBaseURL("http://example.com"),
		WithBearerToken(""),
	)
	if err == nil {
		t.Fatal("expected error for empty bearer token")
	}
}

func TestWithAPIKey(t *testing.T) {
	_, err := New(
		WithBaseURL("http://example.com"),
		WithAPIKey("X-API-Key", "secret-key"),
	)
	if err != nil {
		t.Fatalf("New() with WithAPIKey failed: %v", err)
	}
}

func TestWithAPIKey_Empty(t *testing.T) {
	_, err := New(
		WithBaseURL("http://example.com"),
		WithAPIKey("", "value"),
	)
	if err == nil {
		t.Fatal("expected error for empty API key header")
	}
}

func TestWithLogger(t *testing.T) {
	_, err := New(
		WithBaseURL("http://example.com"),
		WithLogger(relay.NoopLogger()),
	)
	if err != nil {
		t.Fatalf("New() with WithLogger failed: %v", err)
	}
}

func TestWithHeader(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{Status: 204})

	client, err := New(
		WithBaseURL(server.URL()),
		WithHeader("X-Custom-Header", "custom-value"),
	)
	if err != nil {
		t.Fatalf("New() with WithHeader failed: %v", err)
	}

	// Make a request to verify the header is sent.
	_ = client.Delete(context.Background(), "Products", 1)

	reqs := server.RecordedRequests()
	if len(reqs) == 0 {
		t.Fatal("no requests recorded")
	}
	got := reqs[0].Headers.Get("X-Custom-Header")
	if got != "custom-value" {
		t.Errorf("X-Custom-Header = %q, want %q", got, "custom-value")
	}
}

func TestWithTimeout(t *testing.T) {
	_, err := New(
		WithBaseURL("http://example.com"),
		WithTimeout(15*time.Second),
	)
	if err != nil {
		t.Fatalf("New() with WithTimeout failed: %v", err)
	}
}
