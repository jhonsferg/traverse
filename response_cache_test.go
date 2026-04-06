package traverse

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// oDataResp builds a minimal OData v4 JSON response for a slice of records.
func oDataResp(t *testing.T, records []map[string]interface{}) []byte {
	t.Helper()
	body, err := json.Marshal(map[string]interface{}{"value": records})
	if err != nil {
		t.Fatalf("failed to marshal test response: %v", err)
	}
	return body
}

// TestInMemoryResponseCache_GetSet verifies basic get/set semantics.
func TestInMemoryResponseCache_GetSet(t *testing.T) {
	c := NewInMemoryResponseCache()

	// Miss on empty cache.
	if _, ok := c.Get("k1"); ok {
		t.Fatal("expected cache miss on empty cache")
	}

	entry := &ResponseCacheEntry{Body: []byte(`{"value":[]}`), ETag: `"abc"`}
	c.Set("k1", entry, time.Minute)

	got, ok := c.Get("k1")
	if !ok {
		t.Fatal("expected cache hit")
	}
	if string(got.Body) != string(entry.Body) {
		t.Errorf("body mismatch: got %q, want %q", got.Body, entry.Body)
	}
	if got.ETag != entry.ETag {
		t.Errorf("ETag mismatch: got %q, want %q", got.ETag, entry.ETag)
	}
}

// TestInMemoryResponseCache_TTLExpiry checks that expired entries are still
// returned (callers use isExpired) but ExpiresAt is set correctly.
func TestInMemoryResponseCache_TTLExpiry(t *testing.T) {
	c := NewInMemoryResponseCache()

	entry := &ResponseCacheEntry{Body: []byte(`{}`)}
	c.Set("k", entry, 10*time.Millisecond)

	// Immediately: should hit and not be expired.
	got, ok := c.Get("k")
	if !ok {
		t.Fatal("expected hit immediately after Set")
	}
	if got.isExpired() {
		t.Fatal("entry should not be expired immediately after Set")
	}

	// After TTL: should still return the entry (callers check expiry).
	time.Sleep(20 * time.Millisecond)
	got2, ok2 := c.Get("k")
	if !ok2 {
		t.Fatal("expected entry to remain in cache after expiry (lazy eviction)")
	}
	if !got2.isExpired() {
		t.Error("entry should be expired after TTL elapsed")
	}
}

// TestInMemoryResponseCache_NoTTL verifies entries without TTL never expire.
func TestInMemoryResponseCache_NoTTL(t *testing.T) {
	c := NewInMemoryResponseCache()

	entry := &ResponseCacheEntry{Body: []byte(`{}`)}
	c.Set("k", entry, 0)

	got, ok := c.Get("k")
	if !ok {
		t.Fatal("expected cache hit")
	}
	if got.isExpired() {
		t.Error("entry with zero TTL should never expire")
	}
}

// TestInMemoryResponseCache_Delete verifies Delete removes a single entry.
func TestInMemoryResponseCache_Delete(t *testing.T) {
	c := NewInMemoryResponseCache()

	c.Set("k1", &ResponseCacheEntry{Body: []byte(`{}`)}, 0)
	c.Set("k2", &ResponseCacheEntry{Body: []byte(`{}`)}, 0)

	c.Delete("k1")

	if _, ok := c.Get("k1"); ok {
		t.Error("k1 should be deleted")
	}
	if _, ok := c.Get("k2"); !ok {
		t.Error("k2 should still exist")
	}
}

// TestInMemoryResponseCache_Invalidate verifies prefix-based invalidation.
func TestInMemoryResponseCache_Invalidate(t *testing.T) {
	c := NewInMemoryResponseCache()

	c.Set("Products", &ResponseCacheEntry{Body: []byte(`{}`)}, 0)
	c.Set("Products?$filter=Price gt 10", &ResponseCacheEntry{Body: []byte(`{}`)}, 0)
	c.Set("Orders", &ResponseCacheEntry{Body: []byte(`{}`)}, 0)

	c.Invalidate("Products")

	if _, ok := c.Get("Products"); ok {
		t.Error("Products should be invalidated")
	}
	if _, ok := c.Get("Products?$filter=Price gt 10"); ok {
		t.Error("Products?$filter=... should be invalidated")
	}
	if _, ok := c.Get("Orders"); !ok {
		t.Error("Orders should not be invalidated")
	}
}

// TestInMemoryResponseCache_Clear verifies all entries are removed.
func TestInMemoryResponseCache_Clear(t *testing.T) {
	c := NewInMemoryResponseCache()

	for i := 0; i < 5; i++ {
		c.Set(fmt.Sprintf("k%d", i), &ResponseCacheEntry{Body: []byte(`{}`)}, 0)
	}

	c.Clear()

	for i := 0; i < 5; i++ {
		if _, ok := c.Get(fmt.Sprintf("k%d", i)); ok {
			t.Errorf("k%d should be cleared", i)
		}
	}
}

// TestQueryBuilder_WithCache_CachesResponse verifies that WithCache stores the
// first response and serves subsequent identical requests from cache without
// making additional HTTP calls.
func TestQueryBuilder_WithCache_CachesResponse(t *testing.T) {
	var calls int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(oDataResp(t, []map[string]interface{}{
			{"ID": 1, "Name": "Widget"},
		}))
	}))
	defer srv.Close()

	cache := NewInMemoryResponseCache()
	client, err := New(
		WithBaseURL(srv.URL),
		WithResponseCache(cache),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer client.Close() //nolint:errcheck

	ctx := context.Background()

	// First call: cache miss, fetches from server.
	page1, err := client.From("Products").WithCache(time.Minute).Page(ctx)
	if err != nil {
		t.Fatalf("first Page: %v", err)
	}
	if len(page1.Value) != 1 {
		t.Fatalf("expected 1 record, got %d", len(page1.Value))
	}

	// Second call: should hit cache, no additional HTTP request.
	page2, err := client.From("Products").WithCache(time.Minute).Page(ctx)
	if err != nil {
		t.Fatalf("second Page: %v", err)
	}
	if len(page2.Value) != 1 {
		t.Fatalf("expected 1 record from cache, got %d", len(page2.Value))
	}

	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Errorf("expected exactly 1 HTTP call, got %d", got)
	}
}

// TestQueryBuilder_WithCache_ExpiredRevalidates verifies that an expired entry
// causes a new HTTP request and updates the cache.
func TestQueryBuilder_WithCache_ExpiredRevalidates(t *testing.T) {
	var calls int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&calls, 1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(oDataResp(t, []map[string]interface{}{
			{"ID": int(n)},
		}))
	}))
	defer srv.Close()

	cache := NewInMemoryResponseCache()
	client, err := New(
		WithBaseURL(srv.URL),
		WithResponseCache(cache),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer client.Close() //nolint:errcheck

	ctx := context.Background()
	ttl := 20 * time.Millisecond

	// First call: populate cache.
	page1, err := client.From("Items").WithCache(ttl).Page(ctx)
	if err != nil {
		t.Fatalf("first Page: %v", err)
	}
	if idVal, ok := page1.Value[0]["ID"]; !ok || fmt.Sprint(idVal) != "1" {
		t.Fatalf("unexpected first result: %v", page1.Value)
	}

	// Wait for TTL to expire.
	time.Sleep(40 * time.Millisecond)

	// Second call: cache is stale, re-fetches.
	page2, err := client.From("Items").WithCache(ttl).Page(ctx)
	if err != nil {
		t.Fatalf("second Page: %v", err)
	}
	if idVal, ok := page2.Value[0]["ID"]; !ok || fmt.Sprint(idVal) != "2" {
		t.Fatalf("unexpected second result (expected fresh data): %v", page2.Value)
	}

	if got := atomic.LoadInt32(&calls); got != 2 {
		t.Errorf("expected 2 HTTP calls, got %d", got)
	}
}

// TestQueryBuilder_WithCache_ETag304 verifies that a stale cached entry with
// an ETag triggers an If-None-Match request, and a 304 response renews the entry.
func TestQueryBuilder_WithCache_ETag304(t *testing.T) {
	const etag = `"v1"`
	var calls int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		if r.Header.Get("If-None-Match") == etag {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		w.Header().Set("ETag", etag)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(oDataResp(t, []map[string]interface{}{
			{"ID": 1},
		}))
	}))
	defer srv.Close()

	cache := NewInMemoryResponseCache()
	client, err := New(
		WithBaseURL(srv.URL),
		WithResponseCache(cache),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer client.Close() //nolint:errcheck

	ctx := context.Background()
	ttl := 20 * time.Millisecond

	// First call: cache miss, server returns 200 with ETag.
	page1, err := client.From("Entities").WithCache(ttl).Page(ctx)
	if err != nil {
		t.Fatalf("first Page: %v", err)
	}
	if len(page1.Value) != 1 {
		t.Fatalf("expected 1 record, got %d", len(page1.Value))
	}

	// Expire the entry.
	time.Sleep(40 * time.Millisecond)

	// Second call: stale entry, sends If-None-Match; server replies 304.
	page2, err := client.From("Entities").WithCache(ttl).Page(ctx)
	if err != nil {
		t.Fatalf("second Page (304): %v", err)
	}
	if len(page2.Value) != 1 {
		t.Fatalf("expected 1 record from renewed cache, got %d", len(page2.Value))
	}

	if got := atomic.LoadInt32(&calls); got != 2 {
		t.Errorf("expected 2 HTTP calls (200 + 304), got %d", got)
	}
}

// TestQueryBuilder_NoCache_SendsHeader verifies that NoCache adds Cache-Control: no-cache.
func TestQueryBuilder_NoCache_SendsHeader(t *testing.T) {
	var receivedCC string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedCC = r.Header.Get("Cache-Control")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(oDataResp(t, []map[string]interface{}{}))
	}))
	defer srv.Close()

	client, err := New(WithBaseURL(srv.URL))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer client.Close() //nolint:errcheck

	ctx := context.Background()
	if _, err := client.From("Products").NoCache().Page(ctx); err != nil {
		t.Fatalf("Page: %v", err)
	}

	if !strings.Contains(receivedCC, "no-cache") {
		t.Errorf("expected Cache-Control: no-cache header, got %q", receivedCC)
	}
}

// TestQueryBuilder_WithCache_Collect verifies that Collect also uses the cache.
func TestQueryBuilder_WithCache_Collect(t *testing.T) {
	var calls int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(oDataResp(t, []map[string]interface{}{
			{"ID": 1},
			{"ID": 2},
		}))
	}))
	defer srv.Close()

	cache := NewInMemoryResponseCache()
	client, err := New(
		WithBaseURL(srv.URL),
		WithResponseCache(cache),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer client.Close() //nolint:errcheck

	ctx := context.Background()

	records1, err := client.From("Items").WithCache(time.Minute).Collect(ctx)
	if err != nil {
		t.Fatalf("first Collect: %v", err)
	}
	if len(records1) != 2 {
		t.Fatalf("expected 2 records, got %d", len(records1))
	}

	records2, err := client.From("Items").WithCache(time.Minute).Collect(ctx)
	if err != nil {
		t.Fatalf("second Collect: %v", err)
	}
	if len(records2) != 2 {
		t.Fatalf("expected 2 records from cache, got %d", len(records2))
	}

	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Errorf("expected 1 HTTP call for Collect, got %d", got)
	}
}

// TestClient_CreateInvalidatesCache verifies that Create invalidates cached
// entries for the same entity set.
func TestClient_CreateInvalidatesCache(t *testing.T) {
	var calls int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			atomic.AddInt32(&calls, 1)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"ID":99}`))
		case http.MethodGet:
			atomic.AddInt32(&calls, 1)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(oDataResp(t, []map[string]interface{}{{"ID": 1}}))
		}
	}))
	defer srv.Close()

	cache := NewInMemoryResponseCache()
	client, err := New(
		WithBaseURL(srv.URL),
		WithResponseCache(cache),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer client.Close() //nolint:errcheck

	ctx := context.Background()

	// Populate cache.
	if _, err := client.From("Products").WithCache(time.Minute).Page(ctx); err != nil {
		t.Fatalf("initial Page: %v", err)
	}
	beforeCreate := atomic.LoadInt32(&calls) // should be 1

	// Create: should invalidate the cache entry.
	if _, err := client.Create(ctx, "Products", map[string]interface{}{"Name": "New"}); err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Next read must hit the server again, not the cache.
	if _, err := client.From("Products").WithCache(time.Minute).Page(ctx); err != nil {
		t.Fatalf("post-create Page: %v", err)
	}
	afterCreate := atomic.LoadInt32(&calls)

	if afterCreate-beforeCreate < 2 {
		// We expect at least POST + GET after Create.
		t.Errorf("expected at least 2 calls after Create (POST + GET), got %d total", afterCreate)
	}
}

// TestClient_DeleteInvalidatesCache verifies that Delete invalidates cached
// entries for the same entity set.
func TestClient_DeleteInvalidatesCache(t *testing.T) {
	var calls int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		switch r.Method {
		case http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		case http.MethodGet:
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(oDataResp(t, []map[string]interface{}{{"ID": 1}}))
		}
	}))
	defer srv.Close()

	cache := NewInMemoryResponseCache()
	client, err := New(
		WithBaseURL(srv.URL),
		WithResponseCache(cache),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer client.Close() //nolint:errcheck

	ctx := context.Background()

	// Populate cache.
	if _, err := client.From("Products").WithCache(time.Minute).Page(ctx); err != nil {
		t.Fatalf("initial Page: %v", err)
	}
	callsAfterGet1 := atomic.LoadInt32(&calls)

	// Delete: invalidates cache.
	if err := client.Delete(ctx, "Products", 1); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	// Next read must re-fetch.
	if _, err := client.From("Products").WithCache(time.Minute).Page(ctx); err != nil {
		t.Fatalf("post-delete Page: %v", err)
	}
	callsAfterGet2 := atomic.LoadInt32(&calls)

	if callsAfterGet2-callsAfterGet1 < 2 {
		t.Errorf("expected DELETE + GET calls after deletion, total calls=%d", callsAfterGet2)
	}
}

// TestClient_UpdateInvalidatesCache verifies that Update invalidates cached entries.
func TestClient_UpdateInvalidatesCache(t *testing.T) {
	var calls int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		switch r.Method {
		case http.MethodPatch:
			w.WriteHeader(http.StatusNoContent)
		case http.MethodGet:
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(oDataResp(t, []map[string]interface{}{{"ID": 1}}))
		}
	}))
	defer srv.Close()

	cache := NewInMemoryResponseCache()
	client, err := New(
		WithBaseURL(srv.URL),
		WithResponseCache(cache),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer client.Close() //nolint:errcheck

	ctx := context.Background()

	if _, err := client.From("Tasks").WithCache(time.Minute).Page(ctx); err != nil {
		t.Fatalf("initial Page: %v", err)
	}

	if err := client.Update(ctx, "Tasks", 1, map[string]interface{}{"Done": true}); err != nil {
		t.Fatalf("Update: %v", err)
	}

	// After update the cache entry for Tasks is gone; must re-fetch.
	if _, ok := cache.Get("Tasks"); ok {
		entry, _ := cache.Get("Tasks")
		if !entry.isExpired() {
			t.Error("cache entry for Tasks should have been invalidated after Update")
		}
	}
}

// TestWithResponseCache_NilReturnsError verifies that a nil cache returns an error.
func TestWithResponseCache_NilReturnsError(t *testing.T) {
	_, err := New(
		WithBaseURL("http://example.com"),
		WithResponseCache(nil),
	)
	if err == nil {
		t.Error("expected error when passing nil ResponseCache")
	}
}
