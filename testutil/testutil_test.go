package testutil

import (
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

	resp, err := http.Get(ms.URL() + "/test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

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

	req, err := http.NewRequest(http.MethodPost, ms.URL()+"/api/data", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer token123")

	client := &http.Client{}
	_, err = client.Do(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
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

	for i, expectedStatus := range []int{http.StatusOK, http.StatusCreated, http.StatusNotFound} {
		resp, err := http.Get(ms.URL())
		if err != nil {
			t.Fatalf("request %d: unexpected error: %v", i, err)
		}
		resp.Body.Close()

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
	resp, err := http.Get(ms.URL())
	elapsed := time.Since(start)
	resp.Body.Close()

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

	for i := 0; i < 3; i++ {
		http.Get(ms.URL())
	}

	if count := ms.RequestCount(); count != 3 {
		t.Errorf("got request count %d, want 3", count)
	}
}

func TestRequestRecorder_RecordRequests(t *testing.T) {
	recorder := NewRequestRecorder()

	roundTrip := recorder.Middleware()(http.DefaultTransport)

	req1, _ := http.NewRequest(http.MethodGet, "https://example.com/api", nil)
	req1.Header.Set("X-Custom", "value1")
	roundTrip.RoundTrip(req1)

	req2, _ := http.NewRequest(http.MethodPost, "https://example.com/api", nil)
	req2.Header.Set("X-Custom", "value2")
	roundTrip.RoundTrip(req2)

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

	req, _ := http.NewRequest(http.MethodGet, "https://example.com", nil)
	roundTrip.RoundTrip(req)

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
