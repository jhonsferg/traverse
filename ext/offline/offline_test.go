package offline

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"testing"
)

func TestStore_SetGet(t *testing.T) {
	dir := t.TempDir()
	s, err := NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}

	want := []byte(`{"value":[{"ID":1}]}`)
	if setErr := s.Set("/Customers", want); setErr != nil {
		t.Fatal(setErr)
	}

	got, err := s.Get("/Customers")
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(want) {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestStore_GetNotFound(t *testing.T) {
	dir := t.TempDir()
	s, err := NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}

	_, err = s.Get("/DoesNotExist")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestStore_Delete(t *testing.T) {
	dir := t.TempDir()
	s, err := NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}

	if setErr := s.Set("/Orders", []byte(`{}`)); setErr != nil {
		t.Fatal(setErr)
	}
	if delErr := s.Delete("/Orders"); delErr != nil {
		t.Fatal(delErr)
	}
	_, err = s.Get("/Orders")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestStore_Keys(t *testing.T) {
	dir := t.TempDir()
	s, err := NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}

	paths := []string{"/A", "/B", "/C"}
	for _, p := range paths {
		if setErr := s.Set(p, []byte(`{}`)); setErr != nil {
			t.Fatal(setErr)
		}
	}

	keys, err := s.Keys()
	if err != nil {
		t.Fatal(err)
	}
	if len(keys) != 3 {
		t.Fatalf("expected 3 keys, got %d", len(keys))
	}
	sort.Strings(keys)
	sort.Strings(paths)
	for i, k := range keys {
		if k != paths[i] {
			t.Errorf("key[%d] = %q, want %q", i, k, paths[i])
		}
	}
}

func TestStore_Clear(t *testing.T) {
	dir := t.TempDir()
	s, err := NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}

	for _, p := range []string{"/X", "/Y", "/Z"} {
		if setErr := s.Set(p, []byte(`{}`)); setErr != nil {
			t.Fatal(setErr)
		}
	}
	if clearErr := s.Clear(); clearErr != nil {
		t.Fatal(clearErr)
	}

	keys, err := s.Keys()
	if err != nil {
		t.Fatal(err)
	}
	if len(keys) != 0 {
		t.Fatalf("expected 0 keys after Clear, got %d: %v", len(keys), keys)
	}
}

func TestStore_Persistence(t *testing.T) {
	dir := t.TempDir()

	s1, err := NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	want := []byte(`{"value":[{"Name":"Alice"}]}`)
	if setErr := s1.Set("/People", want); setErr != nil {
		t.Fatal(setErr)
	}

	// Create a second store pointing at the same directory.
	s2, err := NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	got, err := s2.Get("/People")
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(want) {
		t.Fatalf("got %q, want %q", got, want)
	}
}

// TestMiddleware_CacheOnSuccess verifies that OfflineMiddleware persists a 2xx response.
func TestMiddleware_CacheOnSuccess(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}

	body := []byte(`{"value":[]}`)
	mw := OfflineMiddleware(store)
	wrapped := mw(roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		return fakeResponse(200, body), nil
	}))

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://example.com/Products", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := wrapped.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	closeErr := resp.Body.Close()
	if closeErr != nil {
		t.Fatal(closeErr)
	}

	cached, err := store.Get("/Products")
	if err != nil {
		t.Fatalf("expected cached entry after 200: %v", err)
	}
	if string(cached) != string(body) {
		t.Fatalf("cached %q, want %q", cached, body)
	}
}

// TestMiddleware_ServesCacheOnNetworkError verifies offline fallback.
func TestMiddleware_ServesCacheOnNetworkError(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}

	data := []byte(`{"value":[{"ID":99}]}`)
	if setErr := store.Set("/Items", data); setErr != nil {
		t.Fatal(setErr)
	}

	netErr := fmt.Errorf("dial tcp: connection refused")
	mw := OfflineMiddleware(store)
	wrapped := mw(roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		return nil, netErr
	}))

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://example.com/Items", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := wrapped.RoundTrip(req)
	if err != nil {
		t.Fatalf("expected cached response, got error: %v", err)
	}
	if closeErr := resp.Body.Close(); closeErr != nil {
		t.Fatal(closeErr)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

// TestMiddleware_NetworkErrorCacheMiss verifies that the original error is returned
// when there is no cached entry and a network error occurs.
func TestMiddleware_NetworkErrorCacheMiss(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}

	netErr := fmt.Errorf("no route to host")
	mw := OfflineMiddleware(store)
	wrapped := mw(roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		return nil, netErr
	}))

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://example.com/Missing", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := wrapped.RoundTrip(req)
	if resp != nil {
		resp.Body.Close() //nolint:errcheck
	}
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, netErr) {
		t.Fatalf("expected netErr, got %v", err)
	}
}

// fakeResponse builds a minimal http.Response for testing.
func fakeResponse(code int, body []byte) *http.Response {
	return &http.Response{
		StatusCode: code,
		Status:     fmt.Sprintf("%d OK", code),
		Header:     http.Header{},
		Body:       io.NopCloser(bytes.NewReader(body)),
	}
}
