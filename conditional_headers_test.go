package traverse

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestIfMatch_SendsHeader(t *testing.T) {
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("If-Match")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"value":[]}`))
	}))
	defer srv.Close()

	client, err := New(WithBaseURL(srv.URL))
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	_, _ = client.From("Orders").IfMatch(`W/"abc123"`).Page(context.Background())

	if got != `W/"abc123"` {
		t.Errorf("If-Match: expected W/\"abc123\", got %q", got)
	}
}

func TestIfNoneMatch_SendsHeader(t *testing.T) {
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("If-None-Match")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"value":[]}`))
	}))
	defer srv.Close()

	client, err := New(WithBaseURL(srv.URL))
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	_, _ = client.From("Products").IfNoneMatch(`"xyz789"`).Page(context.Background())

	if got != `"xyz789"` {
		t.Errorf("If-None-Match: expected \"xyz789\", got %q", got)
	}
}

func TestIfModifiedSince_SendsHeader(t *testing.T) {
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("If-Modified-Since")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"value":[]}`))
	}))
	defer srv.Close()

	client, err := New(WithBaseURL(srv.URL))
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	ts := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)
	_, _ = client.From("Orders").IfModifiedSince(ts).Page(context.Background())

	expected := "Mon, 15 Jan 2024 12:00:00 UTC"
	if got != expected {
		t.Errorf("If-Modified-Since: expected %q, got %q", expected, got)
	}
}

func TestIfUnmodifiedSince_SendsHeader(t *testing.T) {
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("If-Unmodified-Since")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"value":[]}`))
	}))
	defer srv.Close()

	client, err := New(WithBaseURL(srv.URL))
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	ts := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	_, _ = client.From("Products").IfUnmodifiedSince(ts).Page(context.Background())

	expected := "Sat, 01 Jun 2024 00:00:00 UTC"
	if got != expected {
		t.Errorf("If-Unmodified-Since: expected %q, got %q", expected, got)
	}
}

func TestConditionalHeaders_Chaining(t *testing.T) {
	var matchGot, noneMatchGot string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		matchGot = r.Header.Get("If-Match")
		noneMatchGot = r.Header.Get("If-None-Match")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"value":[]}`))
	}))
	defer srv.Close()

	client, err := New(WithBaseURL(srv.URL))
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	_, _ = client.From("Orders").
		IfMatch(`"etag1"`).
		IfNoneMatch(`"etag2"`).
		Filter("Status eq 'Open'").
		Page(context.Background())

	if matchGot != `"etag1"` {
		t.Errorf("If-Match: expected \"etag1\", got %q", matchGot)
	}
	if noneMatchGot != `"etag2"` {
		t.Errorf("If-None-Match: expected \"etag2\", got %q", noneMatchGot)
	}
}

func TestConditionalHeaders_FindByKey(t *testing.T) {
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("If-Match")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ID":1}`))
	}))
	defer srv.Close()

	client, err := New(WithBaseURL(srv.URL))
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	_, _ = client.From("Products").IfMatch(`W/"v1"`).FindByKey(context.Background(), 1)

	if got != `W/"v1"` {
		t.Errorf("If-Match on FindByKey: expected W/\"v1\", got %q", got)
	}
}
