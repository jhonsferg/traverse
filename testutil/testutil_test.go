package testutil

import (
	"context"
	"net/http"
	"testing"
	"time"
)

func TestMockServer_EnqueueAndServe(t *testing.T) {
	ms := NewMockServer()
	defer ms.Close()

	ms.Enqueue(MockResponse{
		Status: http.StatusOK,
		Body:   `{"value":[]}`,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
	})

	ctx := context.Background()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, ms.URL()+"/test", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("got status %d, want %d", resp.StatusCode, http.StatusOK)
	}

	if ct := resp.Header.Get("Content-Type"); ct != "application/json" {
		t.Errorf("got Content-Type %q, want application/json", ct)
	}
}

func TestMockServer_RecordedRequests(t *testing.T) {
	ms := NewMockServer()
	defer ms.Close()

	ms.Enqueue(MockResponse{Status: http.StatusOK})

	ctx := context.Background()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, ms.URL()+"/api/data", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer token123")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp != nil && resp.Body != nil {
		_ = resp.Body.Close()
	}

	recorded := ms.RecordedRequests()
	if len(recorded) != 1 {
		t.Errorf("got %d recorded requests, want 1", len(recorded))
	}

	if recorded[0].Method != http.MethodPost {
		t.Errorf("got method %q, want POST", recorded[0].Method)
	}

	if recorded[0].Path != "/api/data" {
		t.Errorf("got path %q, want /api/data", recorded[0].Path)
	}

	if recorded[0].Headers.Get("Authorization") != "Bearer token123" {
		t.Errorf("got Authorization header %q, want Bearer token123", recorded[0].Headers.Get("Authorization"))
	}
}

func TestMockServer_MultipleResponses(t *testing.T) {
	ms := NewMockServer()
	defer ms.Close()

	ms.Enqueue(MockResponse{Status: http.StatusOK, Body: "first"})
	ms.Enqueue(MockResponse{Status: http.StatusCreated, Body: "second"})
	ms.Enqueue(MockResponse{Status: http.StatusNotFound, Body: "third"})

	ctx := context.Background()
	for i, expectedStatus := range []int{http.StatusOK, http.StatusCreated, http.StatusNotFound} {
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, ms.URL(), nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("request %d: unexpected error: %v", i, err)
		}
		_ = resp.Body.Close()

		if resp.StatusCode != expectedStatus {
			t.Errorf("request %d: got status %d, want %d", i, resp.StatusCode, expectedStatus)
		}
	}
}

func TestMockServer_Delay(t *testing.T) {
	ms := NewMockServer()
	defer ms.Close()

	ms.Enqueue(MockResponse{
		Status: http.StatusOK,
		Delay:  100 * time.Millisecond,
	})

	start := time.Now()
	ctx := context.Background()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, ms.URL(), nil)
	resp, err := http.DefaultClient.Do(req)
	elapsed := time.Since(start)
	if resp != nil && resp.Body != nil {
		_ = resp.Body.Close()
	}

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if elapsed < 100*time.Millisecond {
		t.Errorf("request completed too quickly: %v", elapsed)
	}
}

func TestMockServer_RequestCount(t *testing.T) {
	ms := NewMockServer()
	defer ms.Close()

	ms.Enqueue(MockResponse{Status: http.StatusOK})
	ms.Enqueue(MockResponse{Status: http.StatusOK})
	ms.Enqueue(MockResponse{Status: http.StatusOK})

	ctx := context.Background()
	for i := 0; i < 3; i++ {
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, ms.URL(), nil)
		resp, _ := http.DefaultClient.Do(req)
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
	}

	if count := ms.RequestCount(); count != 3 {
		t.Errorf("got request count %d, want 3", count)
	}
}

func TestRequestRecorder_RecordRequests(t *testing.T) {
	recorder := NewRequestRecorder()

	roundTrip := recorder.Middleware()(http.DefaultTransport)

	req1, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "https://example.com/api", nil)
	req1.Header.Set("X-Custom", "value1")
	resp1, _ := roundTrip.RoundTrip(req1)
	if resp1 != nil && resp1.Body != nil {
		_ = resp1.Body.Close()
	}

	req2, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "https://example.com/api", nil)
	req2.Header.Set("X-Custom", "value2")
	resp2, _ := roundTrip.RoundTrip(req2)
	if resp2 != nil && resp2.Body != nil {
		_ = resp2.Body.Close()
	}

	requests := recorder.Requests()
	if len(requests) != 2 {
		t.Errorf("got %d recorded requests, want 2", len(requests))
	}

	if requests[0].Method != http.MethodGet {
		t.Errorf("got method %q, want GET", requests[0].Method)
	}

	if requests[1].Method != http.MethodPost {
		t.Errorf("got method %q, want POST", requests[1].Method)
	}
}

func TestRequestRecorder_Reset(t *testing.T) {
	recorder := NewRequestRecorder()
	roundTrip := recorder.Middleware()(http.DefaultTransport)

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "https://example.com", nil)
	resp, _ := roundTrip.RoundTrip(req)
	if resp != nil && resp.Body != nil {
		_ = resp.Body.Close()
	}

	if count := recorder.RequestCount(); count != 1 {
		t.Errorf("before reset: got %d requests, want 1", count)
	}

	recorder.Reset()

	if count := recorder.RequestCount(); count != 0 {
		t.Errorf("after reset: got %d requests, want 0", count)
	}
}

func TestODataResponse(t *testing.T) {
	resp := ODataResponse(
		map[string]interface{}{"id": 1, "name": "Item 1"},
		map[string]interface{}{"id": 2, "name": "Item 2"},
	)

	if !contains(resp, `"value"`) {
		t.Errorf("response missing 'value' field: %s", resp)
	}
}

func TestODataErrorResponse(t *testing.T) {
	resp := ODataErrorResponse("INVALID_REQUEST", "Invalid request format")

	if !contains(resp, `"error"`) {
		t.Errorf("error response missing 'error' field: %s", resp)
	}
	if !contains(resp, `"INVALID_REQUEST"`) {
		t.Errorf("error response missing code: %s", resp)
	}
}

func contains(s, substr string) bool {
	for i := 0; i < len(s)-len(substr)+1; i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
