package traverse

import (
	"context"
	"testing"

	"github.com/jhonsferg/traverse/testutil"
)

func TestQueryCount_Success(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{
		Status: 200,
		Body:   "42",
	})

	c, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	count, err := c.From("Products").Count(context.Background())
	if err != nil {
		t.Fatalf("Count: %v", err)
	}
	if count != 42 {
		t.Errorf("want 42, got %d", count)
	}
}

func TestQueryCount_WithFilter(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{Status: 200, Body: "10"})

	c, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	count, err := c.From("Products").Filter("Status eq 'Active'").Count(context.Background())
	if err != nil {
		t.Fatalf("Count with filter: %v", err)
	}
	if count != 10 {
		t.Errorf("want 10, got %d", count)
	}
}

func TestQueryCount_HTTPError(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{Status: 403, Body: "forbidden"})

	c, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	_, err = c.From("Products").Count(context.Background())
	if err == nil {
		t.Fatal("expected error for 403, got nil")
	}
}

func TestQueryCount_InvalidBody(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{Status: 200, Body: "not-a-number"})

	c, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	_, err = c.From("Products").Count(context.Background())
	if err == nil {
		t.Fatal("expected error for non-numeric count body, got nil")
	}
}

func TestQueryFindByCompositeKey_Success(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{
		Status: 200,
		Body:   `{"ID":1,"Plant":"1000","Material":"MAT001"}`,
	})

	c, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	keys := map[string]interface{}{"Plant": "1000", "Material": "MAT001"}
	result, err := c.From("PlantMaterials").FindByCompositeKey(context.Background(), keys)
	if err != nil {
		t.Fatalf("FindByCompositeKey: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
}

func TestQueryFindByCompositeKey_HTTPError(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{Status: 404, Body: `{"error":{"code":"404","message":"not found"}}`})

	c, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	keys := map[string]interface{}{"Plant": "1000", "Material": "MISSING"}
	_, err = c.From("PlantMaterials").FindByCompositeKey(context.Background(), keys)
	if err == nil {
		t.Fatal("expected error for 404, got nil")
	}
}

func TestQueryStreamAs_Success(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{
		Status: 200,
		Body:   `{"value":[{"ID":1,"Name":"A"},{"ID":2,"Name":"B"}]}`,
	})

	c, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ch := c.From("Products").StreamAs(context.Background())
	var count int
	for r := range ch {
		if r.Err != nil {
			t.Fatalf("stream error: %v", r.Err)
		}
		count++
	}
	if count != 2 {
		t.Fatalf("want 2 records, got %d", count)
	}
}

func TestQueryParallel_Success(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	// 3 responses for 3 parallel queries
	server.Enqueue(testutil.MockResponse{Status: 200, Body: `{"value":[{"ID":1}]}`})
	server.Enqueue(testutil.MockResponse{Status: 200, Body: `{"value":[{"ID":2}]}`})
	server.Enqueue(testutil.MockResponse{Status: 200, Body: `{"value":[{"ID":3}]}`})

	c, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	q1 := c.From("Products")
	q2 := c.From("Orders")
	q3 := c.From("Customers")

	pages, err := QueryParallel(context.Background(), q1, q2, q3)
	if err != nil {
		t.Fatalf("QueryParallel: %v", err)
	}
	if len(pages) != 3 {
		t.Fatalf("want 3 pages, got %d", len(pages))
	}
}

func TestQueryParallel_Empty(t *testing.T) {
	pages, err := QueryParallel(context.Background())
	if err != nil {
		t.Fatalf("QueryParallel empty: %v", err)
	}
	if len(pages) != 0 {
		t.Errorf("want 0 pages for empty input, got %d", len(pages))
	}
}

func TestQueryParallel_WithError(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{Status: 200, Body: `{"value":[{"ID":1}]}`})
	server.Enqueue(testutil.MockResponse{Status: 500, Body: `{"error":"fail"}`})

	c, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	q1 := c.From("Products")
	q2 := c.From("Orders")

	_, err = QueryParallel(context.Background(), q1, q2)
	if err == nil {
		t.Fatal("expected error when one query fails, got nil")
	}
}

func TestQueryWithExpandOrderByDesc(t *testing.T) {
	qb := &QueryBuilder{client: &Client{}, entitySet: "Orders", urlDirty: true}
	qb.Expand("Items", WithExpandOrderByDesc("CreatedAt"))
	u := qb.buildURL()
	if u == "" {
		t.Error("buildURL() should not be empty")
	}
}

func TestQueryWithExpandSkip(t *testing.T) {
	qb := &QueryBuilder{client: &Client{}, entitySet: "Orders", urlDirty: true}
	qb.Expand("Items", WithExpandSkip(5))
	u := qb.buildURL()
	if u == "" {
		t.Error("buildURL() should not be empty")
	}
}

// TestQueryFindByCompositeKey_V2 covers the OData v2 path of FindByCompositeKey.
func TestQueryFindByCompositeKey_V2(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{
		Status: 200,
		Body:   `{"d":{"Plant":"1000","Material":"MAT001"}}`,
	})

	c, err := New(WithBaseURL(server.URL()), WithODataVersion(ODataV2))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer func() { _ = c.Close() }()

	keys := map[string]interface{}{"Plant": "1000", "Material": "MAT001"}
	result, err := c.From("PlantMaterials").FindByCompositeKey(context.Background(), keys)
	if err != nil {
		t.Fatalf("FindByCompositeKey V2: %v", err)
	}
	if result == nil {
		t.Fatal("FindByCompositeKey V2 returned nil result")
	}
}

// TestQueryFindByKey_V2 covers the OData v2 response-unwrapping path in FindByKey.
func TestQueryFindByKey_V2(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	// OData v2 wraps the result in {"d": {...}}
	server.Enqueue(testutil.MockResponse{
		Status: 200,
		Body:   `{"d":{"ID":42,"Name":"Widget"}}`,
	})

	c, err := New(WithBaseURL(server.URL()), WithODataVersion(ODataV2))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer func() { _ = c.Close() }()

	result, err := c.From("Products").FindByKey(context.Background(), 42)
	if err != nil {
		t.Fatalf("FindByKey V2: %v", err)
	}
	if result == nil {
		t.Fatal("FindByKey V2 returned nil result")
	}
	if result["Name"] != "Widget" {
		t.Errorf("FindByKey V2: got Name=%v, want Widget", result["Name"])
	}
}

// TestQueryCollect_Error covers the error path in Collect when Stream returns an error.
func TestQueryCollect_Error(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	// Enqueue a 500 error to trigger stream error
	server.Enqueue(testutil.MockResponse{
		Status: 500,
		Body:   `{"error":{"code":"500","message":"Internal Server Error"}}`,
	})

	c, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer func() { _ = c.Close() }()

	_, err = c.From("Products").Collect(context.Background())
	if err == nil {
		t.Fatal("Collect: expected error on 500, got nil")
	}
}

// TestQueryCollect_WithTopAndSkip covers the capacity estimation path with both Top and Skip set.
func TestQueryCollect_WithTopAndSkip(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{
		Status: 200,
		Body:   `{"value":[{"ID":6},{"ID":7},{"ID":8}]}`,
	})

	c, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer func() { _ = c.Close() }()

	results, err := c.From("Products").Top(10).Skip(5).Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect with Top+Skip: %v", err)
	}
	if len(results) != 3 {
		t.Errorf("want 3 results, got %d", len(results))
	}
}
