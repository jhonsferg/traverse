package traverse

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

// pageResponse builds an OData v4 JSON page body.
// nextURL, if non-empty, is written as @odata.nextLink.
func pageResponse(records []map[string]interface{}, nextURL string) string {
	b, _ := json.Marshal(records)
	if nextURL != "" {
		return fmt.Sprintf(`{"value":%s,"@odata.nextLink":%q}`, b, nextURL)
	}
	return fmt.Sprintf(`{"value":%s}`, b)
}

// newPrefetchClient creates a Client pointed at the given base URL.
func newPrefetchClient(t *testing.T, baseURL string) *Client {
	t.Helper()
	c, err := New(WithBaseURL(baseURL))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return c
}

// collectStream drains a Stream channel into a slice, returning the first error seen.
func collectStream(ch <-chan Result[map[string]interface{}]) ([]map[string]interface{}, error) {
	var out []map[string]interface{}
	for r := range ch {
		if r.Err != nil {
			return out, r.Err
		}
		out = append(out, r.Value)
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// TestWithPrefetch_DefaultNoOp - no prefetch still works
// ---------------------------------------------------------------------------

func TestWithPrefetch_DefaultNoOp(t *testing.T) {
	page1 := []map[string]interface{}{{"ID": 1}, {"ID": 2}}
	page2 := []map[string]interface{}{{"ID": 3}}

	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Query().Get("$skip") == "" && calls.Load() == 1 {
			_, _ = fmt.Fprint(w, pageResponse(page1, ""))
		} else {
			_, _ = fmt.Fprint(w, pageResponse(page2, ""))
		}
	}))
	defer srv.Close()

	c := newPrefetchClient(t, srv.URL)
	results, err := collectStream(c.From("Items").Stream(context.Background()))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("want 2 records, got %d", len(results))
	}
}

// ---------------------------------------------------------------------------
// TestWithPrefetch_SingleBuffer - WithPrefetch(1) fetches all pages
// ---------------------------------------------------------------------------

func TestWithPrefetch_SingleBuffer(t *testing.T) {
	// Two pages: page 1 links to page 2.
	var requestCount atomic.Int32
	var srvURL string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := requestCount.Add(1)
		w.Header().Set("Content-Type", "application/json")

		switch n {
		case 1:
			next := srvURL + "/Products?page=2"
			_, _ = fmt.Fprint(w, pageResponse([]map[string]interface{}{{"ID": 1}}, next))
		default:
			_, _ = fmt.Fprint(w, pageResponse([]map[string]interface{}{{"ID": 2}}, ""))
		}
	}))
	srvURL = srv.URL
	defer srv.Close()

	c := newPrefetchClient(t, srv.URL)
	results, err := collectStream(c.From("Products").WithPrefetch(1).Stream(context.Background()))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("want 2 results, got %d", len(results))
	}
	if requestCount.Load() != 2 {
		t.Fatalf("want 2 HTTP requests, got %d", requestCount.Load())
	}
}

// ---------------------------------------------------------------------------
// TestWithPrefetch_ZeroDefaultsToOne - bufferPages=0 treated as 1
// ---------------------------------------------------------------------------

func TestWithPrefetch_ZeroDefaultsToOne(t *testing.T) {
	var requestCount atomic.Int32
	var srvURL string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := requestCount.Add(1)
		w.Header().Set("Content-Type", "application/json")
		if n == 1 {
			_, _ = fmt.Fprint(w, pageResponse([]map[string]interface{}{{"X": 1}}, srvURL+"/E?p=2"))
		} else {
			_, _ = fmt.Fprint(w, pageResponse([]map[string]interface{}{{"X": 2}}, ""))
		}
	}))
	srvURL = srv.URL
	defer srv.Close()

	c := newPrefetchClient(t, srv.URL)
	// WithPrefetch(0) should clamp to 1 internally (builder normalises to 1 before storing).
	results, err := collectStream(c.From("E").WithPrefetch(0).Stream(context.Background()))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("want 2 results, got %d", len(results))
	}
}

// ---------------------------------------------------------------------------
// TestWithPrefetch_ContextCancellation - honours ctx cancellation
// ---------------------------------------------------------------------------

func TestWithPrefetch_ContextCancellation(t *testing.T) {
	// Serve pages very slowly - longer than the context timeout.
	// This guarantees the context expires before the first page is even returned.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, pageResponse([]map[string]interface{}{{"ID": 1}}, r.URL.String()+"&x=1"))
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	c := newPrefetchClient(t, srv.URL)
	ch := c.From("Entities").WithPrefetch(1).Stream(ctx)

	var gotErr bool
	for r := range ch {
		if r.Err != nil {
			gotErr = true
		}
	}
	if !gotErr {
		t.Fatal("expected a context cancellation error, got none")
	}
}

// ---------------------------------------------------------------------------
// TestWithPrefetch_ErrorPropagates - fetch error reaches the caller
// ---------------------------------------------------------------------------

func TestWithPrefetch_ErrorPropagates(t *testing.T) {
	var requestCount atomic.Int32
	var srvURL string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := requestCount.Add(1)
		w.Header().Set("Content-Type", "application/json")
		if n == 1 {
			_, _ = fmt.Fprint(w, pageResponse([]map[string]interface{}{{"ID": 1}}, srvURL+"/E?p=2"))
		} else {
			// Return a server error on the second page.
			http.Error(w, "internal server error", http.StatusInternalServerError)
		}
	}))
	srvURL = srv.URL
	defer srv.Close()

	c := newPrefetchClient(t, srv.URL)
	_, err := collectStream(c.From("E").WithPrefetch(1).Stream(context.Background()))
	if err == nil {
		t.Fatal("expected an error from the failed second page, got nil")
	}
}

// ---------------------------------------------------------------------------
// TestWithNoPrefetch_DisablesPrefetch - WithNoPrefetch overrides WithPrefetch
// ---------------------------------------------------------------------------

func TestWithNoPrefetch_DisablesPrefetch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, pageResponse([]map[string]interface{}{{"ID": 1}}, ""))
	}))
	defer srv.Close()

	c := newPrefetchClient(t, srv.URL)
	q := c.From("Items").WithPrefetch(2).WithNoPrefetch()

	// After WithNoPrefetch, prefetchPages should be -1 (disabled).
	if q.prefetchPages != -1 {
		t.Fatalf("want prefetchPages=-1 after WithNoPrefetch, got %d", q.prefetchPages)
	}

	// Stream should work normally (sequential path).
	results, err := collectStream(q.Stream(context.Background()))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("want 1 result, got %d", len(results))
	}
}

// ---------------------------------------------------------------------------
// TestWithPrefetch_MultiplePages - three-page dataset collected correctly
// ---------------------------------------------------------------------------

func TestWithPrefetch_MultiplePages(t *testing.T) {
	var requestCount atomic.Int32
	var srvURL string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := requestCount.Add(1)
		w.Header().Set("Content-Type", "application/json")
		switch n {
		case 1:
			_, _ = fmt.Fprint(w, pageResponse([]map[string]interface{}{{"N": 1}, {"N": 2}}, srvURL+"/Items?p=2"))
		case 2:
			_, _ = fmt.Fprint(w, pageResponse([]map[string]interface{}{{"N": 3}, {"N": 4}}, srvURL+"/Items?p=3"))
		default:
			_, _ = fmt.Fprint(w, pageResponse([]map[string]interface{}{{"N": 5}}, ""))
		}
	}))
	srvURL = srv.URL
	defer srv.Close()

	c := newPrefetchClient(t, srv.URL)
	results, err := collectStream(c.From("Items").WithPrefetch(2).Stream(context.Background()))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 5 {
		t.Fatalf("want 5 results, got %d", len(results))
	}
}

// ---------------------------------------------------------------------------
// TestWithPrefetch_PageMetadata - page numbers propagate correctly
// ---------------------------------------------------------------------------

func TestWithPrefetch_PageMetadata(t *testing.T) {
	var requestCount atomic.Int32
	var srvURL string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := requestCount.Add(1)
		w.Header().Set("Content-Type", "application/json")
		if n == 1 {
			_, _ = fmt.Fprint(w, pageResponse([]map[string]interface{}{{"ID": "a"}}, srvURL+"/X?p=2"))
		} else {
			_, _ = fmt.Fprint(w, pageResponse([]map[string]interface{}{{"ID": "b"}}, ""))
		}
	}))
	srvURL = srv.URL
	defer srv.Close()

	c := newPrefetchClient(t, srv.URL)
	ch := c.From("X").WithPrefetch(1).Stream(context.Background())

	var pages []int
	for r := range ch {
		if r.Err != nil {
			t.Fatalf("unexpected error: %v", r.Err)
		}
		pages = append(pages, r.Page)
	}
	if len(pages) != 2 {
		t.Fatalf("want 2 results, got %d", len(pages))
	}
	if pages[0] != 1 || pages[1] != 2 {
		t.Errorf("page numbers: got %v, want [1 2]", pages)
	}
}
