package traverse

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/jhonsferg/traverse/testutil"
)

// TestBatchExecuteJSON tests the OData 4.01 JSON batch format.
func TestBatchExecuteJSON_Basic(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	// JSON batch response: {"responses": [...]}
	batchResp := `{"responses":[{"id":"1","status":200,"headers":{"Content-Type":"application/json"},"body":{"@odata.context":"$metadata#Products/$entity","ID":1,"Name":"Widget"}},{"id":"2","status":200,"headers":{"Content-Type":"application/json"},"body":{"@odata.context":"$metadata#Products/$entity","ID":2,"Name":"Gadget"}}]}`
	server.Enqueue(testutil.MockResponse{
		Status: 200,
		Body:   batchResp,
	})

	c, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer func() { _ = c.Close() }()

	resp, err := c.Batch().
		Get("Products", 1).
		Get("Products", 2).
		ExecuteJSON(context.Background())
	if err != nil {
		t.Fatalf("ExecuteJSON: %v", err)
	}

	if len(resp.Results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(resp.Results))
	}
	if resp.Results[0].StatusCode != 200 {
		t.Errorf("result[0] status: want 200, got %d", resp.Results[0].StatusCode)
	}
	if resp.Results[1].StatusCode != 200 {
		t.Errorf("result[1] status: want 200, got %d", resp.Results[1].StatusCode)
	}

	// Verify the request used JSON content type
	reqs := server.RecordedRequests()
	if len(reqs) != 1 {
		t.Fatalf("expected 1 recorded request, got %d", len(reqs))
	}
	if ct := reqs[0].Headers.Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected Content-Type: application/json, got %q", ct)
	}
}

// TestBatchExecuteJSON_ErrorStatus verifies that 4xx status codes are returned as errors.
func TestBatchExecuteJSON_ErrorStatus(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	batchResp := `{"responses":[{"id":"1","status":404,"headers":{},"body":{"error":{"code":"NotFound","message":"Entity not found"}}}]}`
	server.Enqueue(testutil.MockResponse{Status: 200, Body: batchResp})

	c, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer func() { _ = c.Close() }()

	resp, err := c.Batch().Get("Products", 999).ExecuteJSON(context.Background())
	if err != nil {
		t.Fatalf("ExecuteJSON: %v", err)
	}

	if len(resp.Results) != 1 {
		t.Fatalf("expected 1 result")
	}
	if resp.Results[0].Err == nil {
		t.Error("expected error for 404 response, got nil")
	}
}

// TestBuildJSONBatchBody verifies the JSON batch request body structure.
func TestBuildJSONBatchBody(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()
	server.Enqueue(testutil.MockResponse{
		Status: 200,
		Body:   `{"responses":[{"id":"1","status":200,"headers":{},"body":{}}]}`,
	})

	c, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer func() { _ = c.Close() }()

	_, _ = c.Batch().
		Get("Products", 1).
		ExecuteJSON(context.Background())

	reqs := server.RecordedRequests()
	if len(reqs) == 0 {
		t.Fatal("no requests recorded")
	}

	var body jsonBatchRequest
	if err := json.Unmarshal(reqs[0].Body, &body); err != nil {
		t.Fatalf("failed to parse request body as jsonBatchRequest: %v", err)
	}
	if len(body.Requests) == 0 {
		t.Fatal("expected at least one request item")
	}
	item := body.Requests[0]
	if item.ID == "" {
		t.Error("expected non-empty request ID")
	}
	if item.Method != http.MethodGet {
		t.Errorf("expected method GET, got %s", item.Method)
	}
}

// TestBuildJSONBatchBody_Changeset verifies that changeset operations use atomicityGroup.
func TestBuildJSONBatchBody_Changeset(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()
	server.Enqueue(testutil.MockResponse{
		Status: 200,
		Body:   `{"responses":[{"id":"1","status":201,"headers":{},"body":{}},{"id":"2","status":204,"headers":{},"body":null}]}`,
	})

	c, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer func() { _ = c.Close() }()

	_, _ = c.Batch().
		BeginChangeset("cs1").
		Create("Orders", map[string]any{"Total": 100}).
		Update("Products", 1, map[string]any{"Stock": 9}).
		EndChangeset().
		ExecuteJSON(context.Background())

	reqs := server.RecordedRequests()
	if len(reqs) == 0 {
		t.Fatal("no requests recorded")
	}

	var body jsonBatchRequest
	if err := json.Unmarshal(reqs[0].Body, &body); err != nil {
		t.Fatalf("failed to parse request body: %v", err)
	}
	if len(body.Requests) < 2 {
		t.Fatalf("expected at least 2 request items, got %d", len(body.Requests))
	}
	// Both items should share the same atomicityGroup
	g0 := body.Requests[0].AtomicityGroup
	g1 := body.Requests[1].AtomicityGroup
	if g0 == "" || g1 == "" {
		t.Error("expected non-empty atomicityGroup for changeset items")
	}
	if g0 != g1 {
		t.Errorf("expected same atomicityGroup for changeset items, got %q and %q", g0, g1)
	}
}

// TestBatchExecuteJSONStream_Basic verifies streaming JSON batch execution.
func TestBatchExecuteJSONStream_Basic(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	batchResp := `{"responses":[{"id":"1","status":200,"headers":{},"body":{"ID":1}},{"id":"2","status":200,"headers":{},"body":{"ID":2}}]}`
	server.Enqueue(testutil.MockResponse{Status: 200, Body: batchResp})

	c, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer func() { _ = c.Close() }()

	var results []BatchResult
	for r := range c.Batch().Get("Products", 1).Get("Products", 2).ExecuteJSONStream(context.Background()) {
		results = append(results, r)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 streaming results, got %d", len(results))
	}
	for i, r := range results {
		if r.Err != nil {
			t.Errorf("result[%d] unexpected error: %v", i, r.Err)
		}
		if r.StatusCode != 200 {
			t.Errorf("result[%d] status: want 200, got %d", i, r.StatusCode)
		}
	}
}

// TestBatchExecuteJSON_InvalidResponse verifies error handling on malformed response.
func TestBatchExecuteJSON_InvalidResponse(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()
	server.Enqueue(testutil.MockResponse{Status: 200, Body: `not json`})

	c, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer func() { _ = c.Close() }()

	_, err = c.Batch().Get("Products", 1).ExecuteJSON(context.Background())
	if err == nil {
		t.Fatal("expected error for invalid JSON response")
	}
}
