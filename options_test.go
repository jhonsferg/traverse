package traverse

import (
	"testing"

	"github.com/jhonsferg/relay"
)

// TestWithMetadataCacheOption tests the WithMetadataCache option.
func TestWithMetadataCacheOption(t *testing.T) {
	cache := NewMemoryCache()

	client, err := New(
		WithBaseURL("http://example.com"),
		WithMetadataCache(cache),
	)
	if err != nil {
		t.Fatalf("Failed to create client with WithMetadataCache: %v", err)
	}

	// Verify the cache is set
	if client.metadataCache != cache {
		t.Fatalf("Client.metadataCache should be the provided cache instance")
	}
}

// TestWithMetadataCacheOptionNil tests that WithMetadataCache rejects nil cache.
func TestWithMetadataCacheOptionNil(t *testing.T) {
	_, err := New(
		WithBaseURL("http://example.com"),
		WithMetadataCache(nil),
	)

	if err == nil {
		t.Fatalf("WithMetadataCache(nil) should return error")
	}
}

// TestWithBeforeQueryOption tests the WithBeforeQuery option.
func TestWithBeforeQueryOption(t *testing.T) {
	hookCalled := false

	hook := func(qb *QueryBuilder) error {
		hookCalled = true
		return nil
	}

	client, err := New(
		WithBaseURL("http://example.com"),
		WithBeforeQuery(hook),
	)
	if err != nil {
		t.Fatalf("Failed to create client with WithBeforeQuery: %v", err)
	}

	// Verify the hook is registered
	if len(client.beforeQuery) != 1 {
		t.Fatalf("Expected 1 before query hook, got %d", len(client.beforeQuery))
	}

	// Verify the hook is callable
	qb := &QueryBuilder{}
	err = client.beforeQuery[0](qb)
	if err != nil || !hookCalled {
		t.Fatalf("Before query hook was not called correctly")
	}
}

// TestWithBeforeQueryMultipleHooks tests multiple WithBeforeQuery hooks.
func TestWithBeforeQueryMultipleHooks(t *testing.T) {
	callOrder := []int{}

	hook1 := func(qb *QueryBuilder) error {
		callOrder = append(callOrder, 1)
		return nil
	}

	hook2 := func(qb *QueryBuilder) error {
		callOrder = append(callOrder, 2)
		return nil
	}

	client, err := New(
		WithBaseURL("http://example.com"),
		WithBeforeQuery(hook1),
		WithBeforeQuery(hook2),
	)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Verify both hooks are registered
	if len(client.beforeQuery) != 2 {
		t.Fatalf("Expected 2 before query hooks, got %d", len(client.beforeQuery))
	}

	// Call hooks in order
	qb := &QueryBuilder{}
	for _, h := range client.beforeQuery {
		h(qb)
	}

	if len(callOrder) != 2 || callOrder[0] != 1 || callOrder[1] != 2 {
		t.Fatalf("Hooks were not called in order, got %v", callOrder)
	}
}

// TestWithBeforeQueryOptionNil tests that WithBeforeQuery rejects nil hook.
func TestWithBeforeQueryOptionNil(t *testing.T) {
	_, err := New(
		WithBaseURL("http://example.com"),
		WithBeforeQuery(nil),
	)

	if err == nil {
		t.Fatalf("WithBeforeQuery(nil) should return error")
	}
}

// TestWithAfterExecuteOption tests the WithAfterExecute option.
func TestWithAfterExecuteOption(t *testing.T) {
	hookCalled := false

	hook := func(qb *QueryBuilder) error {
		hookCalled = true
		return nil
	}

	client, err := New(
		WithBaseURL("http://example.com"),
		WithAfterExecute(hook),
	)
	if err != nil {
		t.Fatalf("Failed to create client with WithAfterExecute: %v", err)
	}

	// Verify the hook is registered
	if len(client.afterExecute) != 1 {
		t.Fatalf("Expected 1 after execute hook, got %d", len(client.afterExecute))
	}

	// Verify the hook is callable
	qb := &QueryBuilder{}
	err = client.afterExecute[0](qb)
	if err != nil || !hookCalled {
		t.Fatalf("After execute hook was not called correctly")
	}
}

// TestWithAfterExecuteMultipleHooks tests multiple WithAfterExecute hooks.
func TestWithAfterExecuteMultipleHooks(t *testing.T) {
	callOrder := []int{}

	hook1 := func(qb *QueryBuilder) error {
		callOrder = append(callOrder, 1)
		return nil
	}

	hook2 := func(qb *QueryBuilder) error {
		callOrder = append(callOrder, 2)
		return nil
	}

	client, err := New(
		WithBaseURL("http://example.com"),
		WithAfterExecute(hook1),
		WithAfterExecute(hook2),
	)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Verify both hooks are registered
	if len(client.afterExecute) != 2 {
		t.Fatalf("Expected 2 after execute hooks, got %d", len(client.afterExecute))
	}

	// Call hooks in order
	qb := &QueryBuilder{}
	for _, h := range client.afterExecute {
		h(qb)
	}

	if len(callOrder) != 2 || callOrder[0] != 1 || callOrder[1] != 2 {
		t.Fatalf("Hooks were not called in order, got %v", callOrder)
	}
}

// TestWithAfterExecuteOptionNil tests that WithAfterExecute rejects nil hook.
func TestWithAfterExecuteOptionNil(t *testing.T) {
	_, err := New(
		WithBaseURL("http://example.com"),
		WithAfterExecute(nil),
	)

	if err == nil {
		t.Fatalf("WithAfterExecute(nil) should return error")
	}
}

// TestBackwardCompatibility tests that existing options still work.
func TestBackwardCompatibility(t *testing.T) {
	client, err := New(
		WithBaseURL("http://example.com/odata"),
		WithODataVersion(ODataV2),
		WithPageSize(500),
		WithFormat(FormatJSON),
		WithBasicAuth("user", "pass"),
	)
	if err != nil {
		t.Fatalf("Failed to create client with backward compatible options: %v", err)
	}

	// Verify options were applied
	if client.baseURL != "http://example.com/odata" {
		t.Fatalf("BaseURL not set correctly")
	}
	if client.version != ODataV2 {
		t.Fatalf("OData version not set correctly")
	}
	if client.pageSize != 500 {
		t.Fatalf("PageSize not set correctly")
	}
	if client.responseFormat != FormatJSON {
		t.Fatalf("ResponseFormat not set correctly")
	}

	// Verify Phase 1 additions are initialized
	if client.metadataCache == nil {
		t.Fatalf("metadataCache should not be nil")
	}
	if len(client.beforeQuery) != 0 {
		t.Fatalf("beforeQuery should be empty slice initially")
	}
	if len(client.afterExecute) != 0 {
		t.Fatalf("afterExecute should be empty slice initially")
	}
}

// TestCombinedOptions tests using all Phase 1 options together.
func TestCombinedOptions(t *testing.T) {
	cache := NewMemoryCache()
	beforeQueryHookCalled := false
	afterExecuteHookCalled := false

	client, err := New(
		WithBaseURL("http://example.com"),
		WithODataVersion(ODataV4),
		WithMetadataCache(cache),
		WithBeforeQuery(func(qb *QueryBuilder) error {
			beforeQueryHookCalled = true
			return nil
		}),
		WithAfterExecute(func(qb *QueryBuilder) error {
			afterExecuteHookCalled = true
			return nil
		}),
	)

	if err != nil {
		t.Fatalf("Failed to create client with combined options: %v", err)
	}

	// Verify all options were applied
	if client.metadataCache != cache {
		t.Fatalf("Cache not set correctly")
	}
	if client.version != ODataV4 {
		t.Fatalf("OData version not set correctly")
	}
	if len(client.beforeQuery) != 1 {
		t.Fatalf("Expected 1 before query hook")
	}
	if len(client.afterExecute) != 1 {
		t.Fatalf("Expected 1 after execute hook")
	}

	// Execute hooks
	qb := &QueryBuilder{}
	client.beforeQuery[0](qb)
	client.afterExecute[0](qb)

	if !beforeQueryHookCalled || !afterExecuteHookCalled {
		t.Fatalf("Hooks were not called")
	}
}

// TestRelayClientIntegration tests that relay.Client integration still works.
func TestRelayClientIntegration(t *testing.T) {
	relayClient := relay.New()

	client, err := New(
		WithBaseURL("http://example.com"),
		WithRelayClient(relayClient),
	)
	if err != nil {
		t.Fatalf("Failed to create client with relay client: %v", err)
	}

	if client.http != relayClient {
		t.Fatalf("Relay client not set correctly")
	}
}
