package traverse

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/jhonsferg/traverse/testutil"
)

// lastRecorded returns the last recorded request from the mock server.
func lastRecorded(t *testing.T, server *testutil.MockServer) testutil.RecordedRequest {
	t.Helper()
	reqs := server.RecordedRequests()
	if len(reqs) == 0 {
		t.Fatal("no requests recorded")
	}
	return reqs[len(reqs)-1]
}

// ---- A1: Singletons --------------------------------------------------------

func TestSingleton_URLBuilding(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{
		Status: 200,
		Body:   `{"@odata.context":"$metadata#Me","id":"user-1","displayName":"Alice"}`,
	})

	client, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	page, err := client.Singleton("me").Page(context.Background())
	if err != nil {
		t.Fatalf("Singleton().Page() failed: %v", err)
	}
	if page == nil {
		t.Fatal("Page() returned nil")
	}

	req := lastRecorded(t, server)
	if req.Path != "/me" {
		t.Errorf("expected path /me, got %s", req.Path)
	}
}

func TestSingletonAs_TypedRetrieval(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{
		Status: 200,
		Body:   `{"id":"user-1","displayName":"Alice"}`,
	})

	client, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	type UserProfile struct {
		ID          string `json:"id"`
		DisplayName string `json:"displayName"`
	}

	profile, err := SingletonAs[UserProfile](client, "me")
	if err != nil {
		t.Fatalf("SingletonAs() failed: %v", err)
	}
	if profile.ID != "user-1" {
		t.Errorf("expected id user-1, got %s", profile.ID)
	}
	if profile.DisplayName != "Alice" {
		t.Errorf("expected Alice, got %s", profile.DisplayName)
	}
}

func TestSingleton_WithExpand(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{
		Status: 200,
		Body:   `{"id":"user-1","messages":[{"id":"msg-1"}]}`,
	})

	client, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	_, err = client.Singleton("me").Expand("messages").Page(context.Background())
	if err != nil {
		t.Fatalf("Singleton().Expand().Page() failed: %v", err)
	}

	req := lastRecorded(t, server)
	if req.Path != "/me" {
		t.Errorf("expected path /me, got %s", req.Path)
	}
	if req.Query.Get("$expand") != "messages" {
		t.Errorf("expected $expand=messages, got %s", req.Query.Get("$expand"))
	}
}

// ---- A2 + A4: Derived types / type casting ---------------------------------

func TestAsType_AppendsTypeSegment(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{
		Status: 200,
		Body:   `{"value":[]}`,
	})

	client, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	_, err = client.From("Employees").AsType("Model.Manager").Collect(context.Background())
	if err != nil {
		t.Fatalf("AsType().Collect() failed: %v", err)
	}

	req := lastRecorded(t, server)
	if req.Path != "/Employees/Model.Manager" {
		t.Errorf("expected path /Employees/Model.Manager, got %s", req.Path)
	}
}

func TestIsOf_FilterHelper(t *testing.T) {
	expr := IsOf("Model.Manager")
	if expr != "isof(Model.Manager)" {
		t.Errorf("expected isof(Model.Manager), got %s", expr)
	}
}

func TestIsOf_WithProperty(t *testing.T) {
	expr := IsOf("Address", "Model.CnAddress")
	if expr != "isof(Address,Model.CnAddress)" {
		t.Errorf("expected isof(Address,Model.CnAddress), got %s", expr)
	}
}

func TestCast_FilterHelper(t *testing.T) {
	expr := Cast("Budget", "Edm.Decimal")
	if expr != "cast(Budget,Edm.Decimal)" {
		t.Errorf("expected cast(Budget,Edm.Decimal), got %s", expr)
	}
}

// ---- A3: $expand with $levels ----------------------------------------------

func TestWithExpandLevels_Max(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{
		Status: 200,
		Body:   `{"value":[]}`,
	})

	client, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	_, err = client.From("Categories").
		Expand("Children", WithExpandLevels(LevelsMax)).
		Collect(context.Background())
	if err != nil {
		t.Fatalf("Expand with LevelsMax failed: %v", err)
	}

	req := lastRecorded(t, server)
	expand := req.Query.Get("$expand")
	if expand != "Children($levels=max)" {
		t.Errorf("expected $expand=Children($levels=max), got %s", expand)
	}
}

func TestWithExpandLevels_Numeric(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{
		Status: 200,
		Body:   `{"value":[]}`,
	})

	client, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	_, err = client.From("Categories").
		Expand("Children", WithExpandLevels(3)).
		Collect(context.Background())
	if err != nil {
		t.Fatalf("Expand with levels=3 failed: %v", err)
	}

	req := lastRecorded(t, server)
	expand := req.Query.Get("$expand")
	if expand != "Children($levels=3)" {
		t.Errorf("expected $expand=Children($levels=3), got %s", expand)
	}
}

func TestWithExpandLevels_WithSelect(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{
		Status: 200,
		Body:   `{"value":[]}`,
	})

	client, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	_, err = client.From("Categories").
		Expand("Children",
			WithExpandLevels(LevelsMax),
			WithExpandSelect("ID", "Name"),
		).Collect(context.Background())
	if err != nil {
		t.Fatalf("Expand with levels+select failed: %v", err)
	}

	req := lastRecorded(t, server)
	expand := req.Query.Get("$expand")
	if expand == "" {
		t.Errorf("expected $expand, got empty")
	}
	// Should contain both $levels=max and $select=ID,Name
	if expand != "Children($select=ID,Name;$levels=max)" && expand != "Children($levels=max;$select=ID,Name)" {
		t.Errorf("unexpected $expand value: %s", expand)
	}
}

// ---- B1: BulkUpdate --------------------------------------------------------

func TestBulkUpdate_SendsPatchToCollection(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{Status: http.StatusNoContent})

	client, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	err = client.From("Products").
		Filter("Category eq 'Discontinued'").
		BulkUpdate(context.Background(), map[string]any{"Status": "Archived"})
	if err != nil {
		t.Fatalf("BulkUpdate() failed: %v", err)
	}

	req := lastRecorded(t, server)
	if req.Method != http.MethodPatch {
		t.Errorf("expected PATCH, got %s", req.Method)
	}
	if req.Path != "/Products" {
		t.Errorf("expected /Products, got %s", req.Path)
	}
	if req.Query.Get("$filter") != "Category eq 'Discontinued'" {
		t.Errorf("expected filter in query, got %s", req.Query.Get("$filter"))
	}
}

func TestBulkUpdate_OK_ReturnsNil(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{
		Status: http.StatusOK,
		Body:   `{"value":[]}`,
	})

	client, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	err = client.From("Orders").
		Filter("Status eq 'Pending'").
		BulkUpdate(context.Background(), map[string]any{"Priority": 1})
	if err != nil {
		t.Errorf("BulkUpdate with 200 OK should return nil, got: %v", err)
	}
}

func TestBulkUpdate_NotFound_ReturnsError(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{Status: http.StatusNotFound})

	client, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	err = client.From("Products").BulkUpdate(context.Background(), map[string]any{"Status": "X"})
	if err == nil {
		t.Error("expected error for 404, got nil")
	}
}

func TestBulkUpdate_BodyIsJSON(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{Status: http.StatusNoContent})

	client, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	type PatchData struct {
		Status string `json:"status"`
		Count  int    `json:"count"`
	}
	err = client.From("Items").BulkUpdate(context.Background(), PatchData{Status: "active", Count: 5})
	if err != nil {
		t.Fatalf("BulkUpdate() failed: %v", err)
	}

	req := lastRecorded(t, server)
	var body PatchData
	if decErr := json.Unmarshal(req.Body, &body); decErr != nil {
		t.Fatalf("body not valid JSON: %v", decErr)
	}
	if body.Status != "active" || body.Count != 5 {
		t.Errorf("unexpected body: %+v", body)
	}
}

// ---- B3: Prefer headers ----------------------------------------------------

func TestWithPrefer_SetsPreferHeader(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{
		Status: 200,
		Body:   `{"value":[]}`,
	})

	client, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	_, err = client.From("Products").
		WithPrefer(PreferHandlingStrict).
		Collect(context.Background())
	if err != nil {
		t.Fatalf("WithPrefer().Collect() failed: %v", err)
	}

	req := lastRecorded(t, server)
	prefer := req.Headers.Get("Prefer")
	if prefer != PreferHandlingStrict {
		t.Errorf("expected Prefer: %s, got %s", PreferHandlingStrict, prefer)
	}
}

func TestWithPrefer_Lenient(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{
		Status: 200,
		Body:   `{"value":[]}`,
	})

	client, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	_, err = client.From("Products").
		WithPrefer(PreferHandlingLenient).
		Collect(context.Background())
	if err != nil {
		t.Fatalf("WithPrefer(lenient).Collect() failed: %v", err)
	}

	req := lastRecorded(t, server)
	prefer := req.Headers.Get("Prefer")
	if prefer != PreferHandlingLenient {
		t.Errorf("expected Prefer: %s, got %s", PreferHandlingLenient, prefer)
	}
}

func TestPreferConstants_Values(t *testing.T) {
	cases := []struct {
		name     string
		constant string
		expected string
	}{
		{"HandlingStrict", PreferHandlingStrict, "handling=strict"},
		{"HandlingLenient", PreferHandlingLenient, "handling=lenient"},
		{"ReturnMinimal", PreferReturnMinimal, "return=minimal"},
		{"ReturnRepresentation", PreferReturnRepresentation, "return=representation"},
		{"RespondAsync", PreferRespondAsync, "respond-async"},
		{"TrackChanges", PreferTrackChanges, "odata.track-changes"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.constant != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, tc.constant)
			}
		})
	}
}

// ---- B4: $schemaversion -----------------------------------------------------

func TestWithSchemaVersion_ClientLevel(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{
		Status: 200,
		Body:   `{"value":[]}`,
	})

	client, err := New(
		WithBaseURL(server.URL()),
		WithSchemaVersion("2.0"),
	)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	_, err = client.From("Products").Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect() failed: %v", err)
	}

	req := lastRecorded(t, server)
	sv := req.Headers.Get("OData-SchemaVersion")
	if sv != "2.0" {
		t.Errorf("expected OData-SchemaVersion: 2.0, got %q", sv)
	}
}

func TestWithSchemaVersion_PerQuery(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{
		Status: 200,
		Body:   `{"value":[]}`,
	})

	client, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	_, err = client.From("Products").
		WithSchemaVersion("1.5").
		Collect(context.Background())
	if err != nil {
		t.Fatalf("WithSchemaVersion().Collect() failed: %v", err)
	}

	req := lastRecorded(t, server)
	sv := req.Headers.Get("OData-SchemaVersion")
	if sv != "1.5" {
		t.Errorf("expected OData-SchemaVersion: 1.5, got %q", sv)
	}
}

// ---- E2: $inlinecount vs $count normalization --------------------------------

func TestWithCount_V4_EmitsCountTrue(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{
		Status: 200,
		Body:   `{"@odata.count":42,"value":[]}`,
	})

	client, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	page, err := client.From("Products").WithCount().Page(context.Background())
	if err != nil {
		t.Fatalf("WithCount().Page() failed: %v", err)
	}

	req := lastRecorded(t, server)
	countParam := req.Query.Get("$count")
	if countParam != "true" {
		t.Errorf("expected $count=true for v4, got %q", countParam)
	}
	if req.Query.Get("$inlinecount") != "" {
		t.Errorf("should not have $inlinecount for v4")
	}
	if page.Count != nil && *page.Count != 42 {
		t.Errorf("expected count=42, got %d", *page.Count)
	}
	if page.Count == nil {
		t.Error("expected count to be non-nil")
	}
}

func TestWithCount_V2_EmitsInlinecount(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{
		Status: 200,
		Body:   `{"d":{"results":[{"ID":1}],"__count":"10"}}`,
	})

	client, err := New(
		WithBaseURL(server.URL()),
		WithODataVersion(ODataV2),
	)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	page, err := client.From("Materials").WithCount().Page(context.Background())
	if err != nil {
		t.Fatalf("v2 WithCount().Page() failed: %v", err)
	}

	req := lastRecorded(t, server)
	if req.Query.Get("$inlinecount") != "allpages" {
		t.Errorf("expected $inlinecount=allpages for v2, got %q", req.Query.Get("$inlinecount"))
	}
	if req.Query.Get("$count") != "" {
		t.Errorf("should not have $count=true for v2")
	}
	if page.Count != nil && *page.Count != 10 {
		t.Errorf("expected count=10 from d.__count, got %d", *page.Count)
	}
	if page.Count == nil {
		t.Error("expected count to be non-nil for v2 $inlinecount")
	}
}

// ---- Singleton benchmark (zero allocs) -------------------------------------

func BenchmarkSingletonURL(b *testing.B) {
	client, err := New(WithBaseURL("https://graph.microsoft.com/v1.0"))
	if err != nil {
		b.Fatalf("New() failed: %v", err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		q := client.Singleton("me")
		if q == nil {
			b.Fatal("nil QueryBuilder")
		}
	}
}
