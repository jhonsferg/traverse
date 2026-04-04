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
