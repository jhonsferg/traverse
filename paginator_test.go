package traverse

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/jhonsferg/traverse/testutil"
)

// ---- NewPaginator -----------------------------------------------------------

func TestNewPaginator_NotNil(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	c, _ := New(WithBaseURL(server.URL()))
	p := NewPaginator[map[string]any](c.From("Products").Top(10))
	if p == nil {
		t.Fatal("NewPaginator returned nil")
	}
}

func TestNewPaginatorWithDecoder_NotNil(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	c, _ := New(WithBaseURL(server.URL()))
	p := NewPaginatorWithDecoder[map[string]any](
		c.From("Products"),
		func(raw json.RawMessage) (map[string]any, error) {
			var v map[string]any
			return v, json.Unmarshal(raw, &v)
		},
	)
	if p == nil {
		t.Fatal("NewPaginatorWithDecoder returned nil")
	}
}

// ---- HasMorePages -----------------------------------------------------------

func TestHasMorePages_TrueBeforeFirstFetch(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	c, _ := New(WithBaseURL(server.URL()))
	p := NewPaginator[map[string]any](c.From("Products"))
	if !p.HasMorePages() {
		t.Error("HasMorePages() should be true before first fetch")
	}
}

func TestHasMorePages_FalseAfterLastPage(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{
		Status: 200,
		Body:   `{"value":[{"ID":1}]}`,
	})

	c, _ := New(WithBaseURL(server.URL()))
	p := NewPaginator[map[string]any](c.From("Products"))
	_, err := p.NextPage(context.Background())
	if err != nil {
		t.Fatalf("NextPage() error: %v", err)
	}
	if p.HasMorePages() {
		t.Error("HasMorePages() should be false after consuming last page")
	}
}

// ---- NextPage single page ---------------------------------------------------

func TestNextPage_SinglePage(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{
		Status: 200,
		Body:   `{"value":[{"ID":1,"Name":"Widget"},{"ID":2,"Name":"Gadget"}]}`,
	})

	type Product struct {
		ID   int    `json:"ID"`
		Name string `json:"Name"`
	}

	c, _ := New(WithBaseURL(server.URL()))
	p := NewPaginator[Product](c.From("Products"))

	items, err := p.NextPage(context.Background())
	if err != nil {
		t.Fatalf("NextPage() error: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	if items[0].Name != "Widget" {
		t.Errorf("items[0].Name = %q, want Widget", items[0].Name)
	}
	if items[1].Name != "Gadget" {
		t.Errorf("items[1].Name = %q, want Gadget", items[1].Name)
	}
}

// ---- NextPage multi-page (nextLink) -----------------------------------------

func TestNextPage_MultiPage_FollowsNextLink(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	// Page 1: has nextLink pointing back to the same test server.
	server.Enqueue(testutil.MockResponse{
		Status: 200,
		Body:   fmt.Sprintf(`{"value":[{"ID":1}],"@odata.nextLink":"%s/Products?$skiptoken=page2"}`, server.URL()),
	})
	// Page 2: no nextLink, end of results.
	server.Enqueue(testutil.MockResponse{
		Status: 200,
		Body:   `{"value":[{"ID":2}]}`,
	})

	type Item struct{ ID int }

	c, _ := New(WithBaseURL(server.URL()))
	p := NewPaginator[Item](c.From("Products"))

	var all []Item
	for p.HasMorePages() {
		page, err := p.NextPage(context.Background())
		if err != nil {
			t.Fatalf("NextPage() error: %v", err)
		}
		all = append(all, page...)
	}

	if len(all) != 2 {
		t.Fatalf("expected 2 total items, got %d", len(all))
	}
	if all[0].ID != 1 || all[1].ID != 2 {
		t.Errorf("items IDs = %v, want [1 2]", []int{all[0].ID, all[1].ID})
	}
}

// ---- NextPage after done ----------------------------------------------------

func TestNextPage_AfterDone_ReturnsNilNoError(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{
		Status: 200,
		Body:   `{"value":[{"ID":1}]}`,
	})

	c, _ := New(WithBaseURL(server.URL()))
	p := NewPaginator[map[string]any](c.From("Products"))

	_, _ = p.NextPage(context.Background())
	// Second call after done.
	items, err := p.NextPage(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if items != nil {
		t.Errorf("expected nil items after done, got %v", items)
	}
}

// ---- Reset ------------------------------------------------------------------

func TestReset_RestoresPaginator(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{Status: 200, Body: `{"value":[{"ID":1}]}`})
	server.Enqueue(testutil.MockResponse{Status: 200, Body: `{"value":[{"ID":1}]}`})

	c, _ := New(WithBaseURL(server.URL()))
	p := NewPaginator[map[string]any](c.From("Products"))

	_, _ = p.NextPage(context.Background())
	if p.HasMorePages() {
		t.Error("should be done after first page")
	}

	p.Reset()
	if !p.HasMorePages() {
		t.Error("HasMorePages() should be true after Reset()")
	}

	items, err := p.NextPage(context.Background())
	if err != nil {
		t.Fatalf("NextPage() after reset error: %v", err)
	}
	if len(items) == 0 {
		t.Error("expected items after reset")
	}
}

// ---- TotalCount -------------------------------------------------------------

func TestTotalCount_WithCount(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	// TotalCount calls Page() which re-executes the original query.
	server.Enqueue(testutil.MockResponse{
		Status: 200,
		Body:   `{"value":[{"ID":1}],"@odata.count":42}`,
	})

	c, _ := New(WithBaseURL(server.URL()))
	p := NewPaginator[map[string]any](c.From("Products").WithCount())

	count, err := p.TotalCount(context.Background())
	if err != nil {
		t.Fatalf("TotalCount() error: %v", err)
	}
	if count != 42 {
		t.Errorf("TotalCount() = %d, want 42", count)
	}
}

func TestTotalCount_NoCount_ReturnsZero(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{
		Status: 200,
		Body:   `{"value":[{"ID":1}]}`,
	})

	c, _ := New(WithBaseURL(server.URL()))
	p := NewPaginator[map[string]any](c.From("Products"))

	count, err := p.TotalCount(context.Background())
	if err != nil {
		t.Fatalf("TotalCount() error: %v", err)
	}
	if count != 0 {
		t.Errorf("TotalCount() = %d, want 0", count)
	}
}

// ---- FetchPageAt ------------------------------------------------------------

func TestFetchPageAt_Success(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{
		Status: 200,
		Body:   `{"value":[{"ID":10}]}`,
	})

	c, _ := New(WithBaseURL(server.URL()))
	page, err := c.FetchPageAt(context.Background(), server.URL()+"/Products?$skiptoken=page2")
	if err != nil {
		t.Fatalf("FetchPageAt() error: %v", err)
	}
	if len(page.Value) != 1 {
		t.Fatalf("expected 1 item, got %d", len(page.Value))
	}
}

func TestFetchPageAt_ServerError(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	// Exhaust retries with 500s.
	for range 3 {
		server.Enqueue(testutil.MockResponse{Status: 500, Body: `{}`})
	}

	c, _ := New(WithBaseURL(server.URL()))
	_, err := c.FetchPageAt(context.Background(), server.URL()+"/Products?$skiptoken=page99")
	if err == nil {
		t.Fatal("expected error on 500")
	}
}

func TestFetchPageAt_Non200Status(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{Status: 404, Body: `{}`})

	c, _ := New(WithBaseURL(server.URL()))
	_, err := c.FetchPageAt(context.Background(), server.URL()+"/Missing")
	if err == nil {
		t.Fatal("expected error on 404")
	}
}

// ---- CustomDecoder ----------------------------------------------------------

func TestPaginator_CustomDecoder(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{
		Status: 200,
		Body:   `{"value":[{"ID":7,"Name":"Custom"}]}`,
	})

	type Product struct {
		ID   int
		Name string
	}

	c, _ := New(WithBaseURL(server.URL()))
	p := NewPaginatorWithDecoder[Product](
		c.From("Products"),
		func(raw json.RawMessage) (Product, error) {
			var pr Product
			return pr, json.Unmarshal(raw, &pr)
		},
	)

	items, err := p.NextPage(context.Background())
	if err != nil {
		t.Fatalf("NextPage() error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].Name != "Custom" {
		t.Errorf("Name = %q, want Custom", items[0].Name)
	}
}

// ---- NextPage empty result --------------------------------------------------

func TestNextPage_EmptyPage(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{
		Status: 200,
		Body:   `{"value":[]}`,
	})

	c, _ := New(WithBaseURL(server.URL()))
	p := NewPaginator[map[string]any](c.From("Products"))

	items, err := p.NextPage(context.Background())
	if err != nil {
		t.Fatalf("NextPage() error: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("expected 0 items, got %d", len(items))
	}
	if p.HasMorePages() {
		t.Error("HasMorePages() should be false after empty page")
	}
}
