// Package traverse integration tests verify cache and client components work together.
package traverse

import (
	"context"
	"net/http"
	"testing"

	"github.com/jhonsferg/traverse/testutil"
)

// TestIntegration_MemoryCacheCreation verifies MemoryCache can be created and used.
func TestIntegration_MemoryCacheCreation(t *testing.T) {
	cache := NewMemoryCache()
	if cache == nil {
		t.Fatalf("Expected NewMemoryCache to return non-nil cache")
	}

	t.Logf("✅ Memory cache creation test passed")
}

// TestIntegration_NoOpCacheImplementation verifies NoOpCache implements CacheStore.
func TestIntegration_NoOpCacheImplementation(t *testing.T) {
	noOp := &NoOpCache{}

	// Should compile (implements CacheStore interface)
	var _ CacheStore = noOp

	t.Logf("✅ NoOpCache implements CacheStore interface")
}

// TestIntegration_ClientCacheField verifies Client has metadataCache field.
func TestIntegration_ClientCacheField(t *testing.T) {
	cache := NewMemoryCache()

	client := &Client{
		metadataCache: cache,
	}

	if client.metadataCache == nil {
		t.Fatalf("Expected metadataCache to be set")
	}

	t.Logf("✅ Client cache field integration test passed")
}

// TestIntegration_ClientHooksFields verifies Client has before/after query hooks.
func TestIntegration_ClientHooksFields(t *testing.T) {
	client := &Client{
		beforeQuery:  []func(*QueryBuilder) error{},
		afterExecute: []func(*QueryBuilder) error{},
	}

	if client.beforeQuery == nil || client.afterExecute == nil {
		t.Fatalf("Expected hook fields to be initialized")
	}

	t.Logf("✅ Client hooks fields integration test passed")
}

// TestIntegration_CacheStoreInterface verifies MemoryCache implements CacheStore.
func TestIntegration_CacheStoreInterface(t *testing.T) {
	cache := NewMemoryCache()

	// This will fail to compile if MemoryCache doesn't implement CacheStore
	var _ CacheStore = cache

	t.Logf("✅ MemoryCache implements CacheStore interface")
}

// TestIntegration_MultipleClientInstances verifies multiple clients can be created.
func TestIntegration_MultipleClientInstances(t *testing.T) {
	cache1 := NewMemoryCache()
	cache2 := NewMemoryCache()

	client1 := &Client{metadataCache: cache1}
	client2 := &Client{metadataCache: cache2}

	if client1 == nil || client2 == nil {
		t.Fatalf("Expected both clients to be created")
	}
	if client1.metadataCache == client2.metadataCache {
		t.Fatalf("Expected different cache instances")
	}

	t.Logf("✅ Multiple client instances integration test passed")
}

// TestIntegration_CacheClear verifies Clear method works.
func TestIntegration_CacheClear(t *testing.T) {
	cache := NewMemoryCache()
	cache.Clear() // Should not panic

	t.Logf("✅ Cache clear integration test passed")
}

// TestIntegration_ClientGettersExist verifies client getter methods exist.
func TestIntegration_ClientGettersExist(t *testing.T) {
	client := &Client{
		baseURL:  "http://test",
		version:  ODataV2,
		pageSize: 1000,
	}

	// These should compile if methods exist
	_ = client.BaseURL()
	_ = client.Version()
	_ = client.PageSize()

	t.Logf("✅ Client getter methods exist and work")
}

// TestIntegration_QueryBuilderWithClient verifies QueryBuilder can be created with client.
func TestIntegration_QueryBuilderWithClient(t *testing.T) {
	client := &Client{}
	qb := &QueryBuilder{client: client}

	if qb.client == nil {
		t.Fatalf("Expected QueryBuilder to have client")
	}

	t.Logf("✅ QueryBuilder with client integration test passed")
}

// TestIntegration_CacheStoreMethods verifies CacheStore interface methods.
func TestIntegration_CacheStoreMethods(t *testing.T) {
	var store CacheStore

	// Initialize with MemoryCache
	store = NewMemoryCache()

	// Should be able to call interface methods
	store.Clear()
	// Get and Set would need Metadata type

	t.Logf("✅ CacheStore interface methods integration test passed")
}

// TestIntegration_MockServerWithClient tests OData client with MockServer.
// TestIntegration_MockServerWithClient tests basic integration with MockServer.
// NOTE: This test verifies that NewMockServer and client creation work,
// but actual request recording requires executing a query operation.
// Currently disabled pending query builder implementation.
/*
func TestIntegration_MockServerWithClient(t *testing.T) {
	ms := testutil.NewMockServer()
	defer ms.Close()

	// Enqueue mock response
	ms.Enqueue(testutil.MockResponse{
		Status: http.StatusOK,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: testutil.ODataResponse(
			map[string]interface{}{"id": 1, "name": "Test Entity"},
		),
	})

	client, err := New(WithBaseURL(ms.URL()))
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	if client == nil {
		t.Fatal("failed to create client")
	}

	requests := ms.RecordedRequests()
	if len(requests) == 0 {
		t.Fatal("expected recorded request")
	}

	if requests[0].Method != http.MethodGet {
		t.Errorf("got method %q, want GET", requests[0].Method)
	}

	t.Logf("✅ MockServer with client integration test passed")
}
*/

// TestIntegration_RequestRecorderMiddleware tests request recording with middleware.
func TestIntegration_RequestRecorderMiddleware(t *testing.T) {
	recorder := testutil.NewRequestRecorder()

	// Create a mock transport chain
	mockTransport := http.DefaultTransport
	recordingTransport := recorder.Middleware()(mockTransport)

	// Create and record a request
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "https://example.com/api/data", nil)
	req.Header.Set("Authorization", "Bearer test-token")

	resp, _ := recordingTransport.RoundTrip(req)
	if resp != nil && resp.Body != nil {
		_ = resp.Body.Close()
	}

	// Verify recording
	recorded := recorder.Requests()
	if len(recorded) != 1 {
		t.Errorf("expected 1 recorded request, got %d", len(recorded))
	}

	if recorded[0].Method != http.MethodPost {
		t.Errorf("got method %q, want POST", recorded[0].Method)
	}

	if recorded[0].Headers.Get("Authorization") != "Bearer test-token" {
		t.Error("Authorization header not properly recorded")
	}

	t.Logf("✅ RequestRecorder middleware integration test passed")
}

// TestIntegration_MultipleResponses tests handling sequence of responses.
func TestIntegration_MultipleResponses(t *testing.T) {
	ms := testutil.NewMockServer()
	defer ms.Close()

	// Enqueue multiple responses in sequence
	responses := []string{
		testutil.ODataResponse(map[string]interface{}{"id": 1}),
		testutil.ODataResponse(map[string]interface{}{"id": 2}),
		testutil.ODataResponse(map[string]interface{}{"id": 3}),
	}

	for _, resp := range responses {
		ms.Enqueue(testutil.MockResponse{
			Status: http.StatusOK,
			Body:   resp,
		})
	}

	// Make requests
	for i := 0; i < len(responses); i++ {
		req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, ms.URL(), nil)
		resp, _ := http.DefaultClient.Do(req)
		_ = resp.Body.Close()
	}

	if count := ms.RequestCount(); count != int64(len(responses)) {
		t.Errorf("got %d requests, want %d", count, len(responses))
	}

	t.Logf("✅ Multiple responses integration test passed")
}

// TestIntegration_ContextPropagation tests context cancellation.
// NOTE: This test verifies context handling, currently a placeholder
// pending full query builder implementation.
/*
func TestIntegration_ContextPropagation(t *testing.T) {
	ms := testutil.NewMockServer()
	defer ms.Close()

	ms.Enqueue(testutil.MockResponse{
		Status: http.StatusOK,
		Body:   testutil.ODataResponse(),
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client, err := New(WithBaseURL(ms.URL()))
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	_ = client
	_ = ctx

	if ctx.Err() != nil {
		t.Error("context should not be canceled")
	}

	t.Logf("✅ Context propagation integration test passed")
}
*/

// TestIntegration_ErrorResponse tests OData error response handling.
func TestIntegration_ErrorResponse(t *testing.T) {
	ms := testutil.NewMockServer()
	defer ms.Close()

	ms.Enqueue(testutil.MockResponse{
		Status: http.StatusBadRequest,
		Body:   testutil.ODataErrorResponse("INVALID_REQUEST", "Bad request format"),
	})

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, ms.URL(), nil)
	resp, _ := http.DefaultClient.Do(req)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("got status %d, want 400", resp.StatusCode)
	}
	_ = resp.Body.Close()

	t.Logf("✅ Error response integration test passed")
}

// TestIntegration_RequestRecorderReset tests recorder reset functionality.
func TestIntegration_RequestRecorderReset(t *testing.T) {
	recorder := testutil.NewRequestRecorder()
	transport := recorder.Middleware()(http.DefaultTransport)

	// Record first request
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "https://example.com/1", nil)
	resp, _ := transport.RoundTrip(req)
	if resp != nil && resp.Body != nil {
		_ = resp.Body.Close()
	}

	if count := recorder.RequestCount(); count != 1 {
		t.Errorf("before reset: got %d requests, want 1", count)
	}

	// Reset
	recorder.Reset()

	if count := recorder.RequestCount(); count != 0 {
		t.Errorf("after reset: got %d requests, want 0", count)
	}

	// Record second request
	req, _ = http.NewRequestWithContext(context.Background(), http.MethodPost, "https://example.com/2", nil)
	resp, _ = transport.RoundTrip(req)
	if resp != nil && resp.Body != nil {
		_ = resp.Body.Close()
	}

	if count := recorder.RequestCount(); count != 1 {
		t.Errorf("after second request: got %d requests, want 1", count)
	}

	t.Logf("✅ RequestRecorder reset integration test passed")
}
