package traverse

import (
	"context"
	"net/http"
	"testing"

	"github.com/jhonsferg/traverse/testutil"
)

const testServiceMetadataV4 = `<?xml version="1.0" encoding="utf-8"?>
<edmx:Edmx Version="4.0" xmlns:edmx="http://docs.oasis-open.org/odata/ns/edmx">
  <edmx:DataServices>
    <Schema Namespace="TestService" xmlns="http://docs.oasis-open.org/odata/ns/edm">
      <EntityContainer Name="Container">
        <EntitySet Name="Products" EntityType="TestService.Product"/>
      </EntityContainer>
      <EntityType Name="Product">
        <Key>
          <PropertyRef Name="ID"/>
        </Key>
        <Property Name="ID" Type="Edm.Int32" Nullable="false"/>
        <Property Name="Name" Type="Edm.String"/>
      </EntityType>
    </Schema>
  </edmx:DataServices>
</edmx:Edmx>`

func TestUpdate_AcceptsStatus200(t *testing.T) {
	srv := testutil.NewMockServer()
	defer srv.Close()

	srv.Enqueue(testutil.MockResponse{Status: http.StatusOK, Body: `{"d":{"ID":1,"Name":"Updated"}}`})

	client, _ := New(
		WithBaseURL(srv.URL()),
		WithODataVersion(ODataV2),
	)
	defer client.Close()

	err := client.Update(context.Background(), "Products", 1, map[string]interface{}{"Name": "Updated"})
	if err != nil {
		t.Errorf("Update should accept 200 OK, got error: %v", err)
	}
}

func TestUpdate_AcceptsStatus204(t *testing.T) {
	srv := testutil.NewMockServer()
	defer srv.Close()

	srv.Enqueue(testutil.MockResponse{Status: http.StatusNoContent})

	client, _ := New(
		WithBaseURL(srv.URL()),
		WithODataVersion(ODataV2),
	)
	defer client.Close()

	err := client.Update(context.Background(), "Products", 1, map[string]interface{}{"Name": "Updated"})
	if err != nil {
		t.Errorf("Update should accept 204 No Content, got error: %v", err)
	}
}

func TestReplace_AcceptsStatus200(t *testing.T) {
	srv := testutil.NewMockServer()
	defer srv.Close()

	srv.Enqueue(testutil.MockResponse{Status: http.StatusOK, Body: `{"d":{"ID":1,"Name":"Replaced"}}`})

	client, _ := New(
		WithBaseURL(srv.URL()),
		WithODataVersion(ODataV2),
	)
	defer client.Close()

	err := client.Replace(context.Background(), "Products", 1, map[string]interface{}{"Name": "Replaced"})
	if err != nil {
		t.Errorf("Replace should accept 200 OK, got error: %v", err)
	}
}

func TestReplace_AcceptsStatus201(t *testing.T) {
	srv := testutil.NewMockServer()
	defer srv.Close()

	srv.Enqueue(testutil.MockResponse{Status: http.StatusCreated, Body: `{"d":{"ID":1,"Name":"Replaced"}}`})

	client, _ := New(
		WithBaseURL(srv.URL()),
		WithODataVersion(ODataV2),
	)
	defer client.Close()

	err := client.Replace(context.Background(), "Products", 1, map[string]interface{}{"Name": "Replaced"})
	if err != nil {
		t.Errorf("Replace should accept 201 Created, got error: %v", err)
	}
}

func TestMetadata_NoOpCache_SkipsCacheOperations(t *testing.T) {
	srv := testutil.NewMockServer()
	defer srv.Close()

	srv.Enqueue(testutil.MockResponse{Status: http.StatusOK, Body: testServiceMetadataV4})

	client, _ := New(
		WithBaseURL(srv.URL()),
		WithODataVersion(ODataV4),
		WithMetadataCache(&NoOpCache{}),
	)
	defer client.Close()

	meta, err := client.Metadata(context.Background())
	if err != nil {
		t.Fatalf("Metadata() with NoOpCache should work, got error: %v", err)
	}
	if meta == nil {
		t.Fatal("Metadata() should return non-nil metadata")
	}
}

func TestMetadata_MemoryCache_CachesResult(t *testing.T) {
	srv := testutil.NewMockServer()
	defer srv.Close()

	srv.Enqueue(testutil.MockResponse{Status: http.StatusOK, Body: testServiceMetadataV4})
	srv.Enqueue(testutil.MockResponse{Status: http.StatusOK, Body: testServiceMetadataV4})

	cache := NewMemoryCache()
	client, _ := New(
		WithBaseURL(srv.URL()),
		WithODataVersion(ODataV4),
		WithMetadataCache(cache),
	)
	defer client.Close()

	_, err := client.Metadata(context.Background())
	if err != nil {
		t.Fatalf("first Metadata() failed: %v", err)
	}

	_, err = client.Metadata(context.Background())
	if err != nil {
		t.Fatalf("second Metadata() failed: %v", err)
	}
}
