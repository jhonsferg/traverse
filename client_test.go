package traverse

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/jhonsferg/traverse/testutil"
)

func TestClientConstruction(t *testing.T) {
	// Test that New() creates a valid client
	c, err := New(
		WithBaseURL("http://localhost:8080/odata"),
	)

	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	if c == nil {
		t.Fatalf("New() returned nil client")
	}

	if c.baseURL != "http://localhost:8080/odata" {
		t.Errorf("baseURL = %s, want http://localhost:8080/odata", c.baseURL)
	}
}

func TestClientFrom(t *testing.T) {
	c, _ := New(
		WithBaseURL("http://localhost:8080/odata"),
	)

	qb := c.From("Products")
	if qb == nil {
		t.Fatalf("From() returned nil")
	}

	if qb.entitySet != "Products" {
		t.Errorf("entitySet = %s, want Products", qb.entitySet)
	}
}

// TestFrom_StripLeadingSlash verifies that From("/Products") and From("Products")
// are equivalent. A leading slash in the entity-set name would cause buildURL to
// emit "//Products", which servers such as SAP reject with 401 instead of 404.
func TestFrom_StripLeadingSlash(t *testing.T) {
	t.Parallel()

	c, _ := New(WithBaseURL("http://localhost/odata"))

	cases := []string{"/Products", "//Products", "Products"}
	for _, input := range cases {
		qb := c.From(input)
		if qb.entitySet != "Products" {
			t.Errorf("From(%q): entitySet = %q, want %q", input, qb.entitySet, "Products")
		}
	}
}

func TestClientWithOptions(t *testing.T) {
	c, _ := New(
		WithBaseURL("http://localhost:8080/odata"),
		WithODataVersion(ODataV4),
		WithPageSize(5000),
	)

	if c.version != ODataV4 {
		t.Errorf("version = %v, want ODataV4", c.version)
	}

	if c.pageSize != 5000 {
		t.Errorf("pageSize = %d, want 5000", c.pageSize)
	}
}

// TestWithMaxPages_DefaultIsSet verifies defaultMaxPages is applied when no
// explicit WithMaxPages option is provided.
func TestWithMaxPages_DefaultIsSet(t *testing.T) {
	c, err := New(WithBaseURL("http://localhost/odata"))
	if err != nil {
		t.Fatalf("New(): %v", err)
	}
	if c.maxPages != defaultMaxPages {
		t.Errorf("maxPages = %d, want defaultMaxPages (%d)", c.maxPages, defaultMaxPages)
	}
}

// TestWithMaxPages_Override verifies WithMaxPages changes the limit.
func TestWithMaxPages_Override(t *testing.T) {
	c, err := New(WithBaseURL("http://localhost/odata"), WithMaxPages(50))
	if err != nil {
		t.Fatalf("New(): %v", err)
	}
	if c.maxPages != 50 {
		t.Errorf("maxPages = %d, want 50", c.maxPages)
	}
}

// TestStream_MaxPagesGuardFires verifies that Stream stops with an error once
// the per-client maxPages limit is reached, preventing infinite pagination loops.
func TestStream_MaxPagesGuardFires(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	// Enqueue 3 pages. Each page returns a nextLink pointing to the next.
	// With maxPages=2 the third page must never be fetched and the stream must
	// return an error after the second page.
	for i := 1; i <= 3; i++ {
		body := fmt.Sprintf(`{"value":[{"ID":%d}],"@odata.nextLink":"%s/Items?$skiptoken=%d"}`,
			i, server.URL(), i)
		if i == 3 {
			// Last page has no nextLink.
			body = fmt.Sprintf(`{"value":[{"ID":%d}]}`, i)
		}
		server.Enqueue(testutil.MockResponse{Status: 200, Body: body})
	}

	c, _ := New(WithBaseURL(server.URL()), WithMaxPages(2))

	var gotErr error
	var count int
	for result := range c.From("Items").Stream(context.Background()) {
		if result.Err != nil {
			gotErr = result.Err
			break
		}
		count++
	}

	if gotErr == nil {
		t.Fatal("expected pagination limit error, got nil")
	}
	if !strings.Contains(gotErr.Error(), "WithMaxPages") {
		t.Errorf("error message should mention WithMaxPages, got: %v", gotErr)
	}
	// Pages 1 and 2 emitted one record each before hitting the guard.
	if count != 2 {
		t.Errorf("expected 2 records before limit, got %d", count)
	}
}

// TestStream_MaxPagesNotHitForShortDataset verifies that normal datasets that
// fit within the limit stream completely without error.
func TestStream_MaxPagesNotHitForShortDataset(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	// Two pages, limit is 10 — should complete cleanly.
	server.Enqueue(testutil.MockResponse{Status: 200, Body: fmt.Sprintf(
		`{"value":[{"ID":1}],"@odata.nextLink":"%s/Items?$skiptoken=1"}`, server.URL())})
	server.Enqueue(testutil.MockResponse{Status: 200, Body: `{"value":[{"ID":2}]}`})

	c, _ := New(WithBaseURL(server.URL()), WithMaxPages(10))

	var records []map[string]interface{}
	for result := range c.From("Items").Stream(context.Background()) {
		if result.Err != nil {
			t.Fatalf("unexpected error: %v", result.Err)
		}
		records = append(records, result.Value)
	}

	if len(records) != 2 {
		t.Errorf("expected 2 records, got %d", len(records))
	}
}
