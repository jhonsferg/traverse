package traverse

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestStreamProperty_URLConstruction(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("binary data"))
	}))
	defer srv.Close()

	client, err := New(WithBaseURL(srv.URL))
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	rc, err := client.From("Products(42)").StreamProperty(context.Background(), "Photo")
	if err != nil {
		t.Fatalf("StreamProperty() failed: %v", err)
	}
	defer func() { _ = rc.Close() }()

	if gotPath != "/Products(42)/Photo" {
		t.Errorf("path: expected /Products(42)/Photo, got %q", gotPath)
	}
}

func TestStreamProperty_AcceptHeader(t *testing.T) {
	var gotAccept string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAccept = r.Header.Get("Accept")
		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("data"))
	}))
	defer srv.Close()

	client, err := New(WithBaseURL(srv.URL))
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	rc, err := client.From("Documents(1)").StreamProperty(context.Background(), "Content")
	if err != nil {
		t.Fatalf("StreamProperty() failed: %v", err)
	}
	defer func() { _ = rc.Close() }()

	if gotAccept != "application/octet-stream" {
		t.Errorf("Accept: expected application/octet-stream, got %q", gotAccept)
	}
}

func TestStreamProperty_BodyAsReader(t *testing.T) {
	bodyContent := "hello binary world"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(bodyContent))
	}))
	defer srv.Close()

	client, err := New(WithBaseURL(srv.URL))
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	rc, err := client.From("Files(7)").StreamProperty(context.Background(), "Data")
	if err != nil {
		t.Fatalf("StreamProperty() failed: %v", err)
	}
	defer func() { _ = rc.Close() }()

	data, readErr := io.ReadAll(rc)
	if readErr != nil {
		t.Fatalf("ReadAll failed: %v", readErr)
	}
	if string(data) != bodyContent {
		t.Errorf("body: expected %q, got %q", bodyContent, string(data))
	}
}

func TestStreamPropertySize_ReturnsContentLength(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodHead {
			t.Errorf("expected HEAD request, got %s", r.Method)
		}
		w.Header().Set("Content-Length", "1024")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client, err := New(WithBaseURL(srv.URL))
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	size, err := client.From("Files(3)").StreamPropertySize(context.Background(), "Data")
	if err != nil {
		t.Fatalf("StreamPropertySize() failed: %v", err)
	}
	if size != 1024 {
		t.Errorf("size: expected 1024, got %d", size)
	}
}

func TestStreamPropertySize_NoContentLength_ReturnsNegOne(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client, err := New(WithBaseURL(srv.URL))
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	size, err := client.From("Files(9)").StreamPropertySize(context.Background(), "Thumbnail")
	if err != nil {
		t.Fatalf("StreamPropertySize() failed: %v", err)
	}
	if size != -1 {
		t.Errorf("size: expected -1 when no Content-Length, got %d", size)
	}
}

func TestStreamProperty_ErrorStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	client, err := New(WithBaseURL(srv.URL))
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	_, err = client.From("Products(99)").StreamProperty(context.Background(), "Photo")
	if err == nil {
		t.Error("expected error for 404 status, got nil")
	}
	if !strings.Contains(err.Error(), "404") {
		t.Errorf("error should mention 404, got: %v", err)
	}
}
