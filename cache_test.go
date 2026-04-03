package traverse

import (
	"testing"
)

// TestNoOpCache tests the NoOpCache implementation.
func TestNoOpCache(t *testing.T) {
	cache := &NoOpCache{}

	// Get should always return nil, false
	if metadata, found := cache.Get("any-key"); found || metadata != nil {
		t.Errorf("NoOpCache.Get() should return (nil, false), got (%v, %v)", metadata, found)
	}

	// Set should be no-op
	testMetadata := &Metadata{}
	cache.Set("key", testMetadata)

	// Get should still return nil, false
	if metadata, found := cache.Get("key"); found || metadata != nil {
		t.Errorf("NoOpCache.Get() after Set() should still return (nil, false), got (%v, %v)", metadata, found)
	}

	// Clear should be no-op
	cache.Clear()
}

// TestMemoryCache tests the MemoryCache implementation.
func TestMemoryCache(t *testing.T) {
	cache := NewMemoryCache()

	// Initially empty
	if metadata, found := cache.Get("key1"); found || metadata != nil {
		t.Errorf("MemoryCache.Get() on empty cache should return (nil, false), got (%v, %v)", metadata, found)
	}

	// Set and Get
	testMetadata := &Metadata{
		EntityTypes: []EntityType{
			{
				Name: "TestEntity",
				Properties: []Property{
					{Name: "ID", Type: "Edm.String"},
				},
			},
		},
	}

	cache.Set("key1", testMetadata)

	if metadata, found := cache.Get("key1"); !found || metadata == nil {
		t.Errorf("MemoryCache.Get() after Set() should return (metadata, true), got (nil, %v)", found)
	} else if metadata.EntityTypes[0].Name != "TestEntity" {
		t.Errorf("MemoryCache.Get() returned different metadata, got Name=%s", metadata.EntityTypes[0].Name)
	}

	// Multiple entries
	testMetadata2 := &Metadata{EntityTypes: []EntityType{{Name: "AnotherEntity"}}}
	cache.Set("key2", testMetadata2)

	if metadata, found := cache.Get("key2"); !found || metadata.EntityTypes[0].Name != "AnotherEntity" {
		t.Errorf("MemoryCache.Get(key2) failed, expected AnotherEntity")
	}

	// Original key should still be there
	if metadata, found := cache.Get("key1"); !found || metadata.EntityTypes[0].Name != "TestEntity" {
		t.Errorf("MemoryCache.Get(key1) after Set(key2) should still return TestEntity")
	}

	// Overwrite
	testMetadata3 := &Metadata{EntityTypes: []EntityType{{Name: "OverwrittenEntity"}}}
	cache.Set("key1", testMetadata3)

	if metadata, found := cache.Get("key1"); !found || metadata.EntityTypes[0].Name != "OverwrittenEntity" {
		t.Errorf("MemoryCache.Get(key1) after overwrite should return OverwrittenEntity")
	}

	// Clear
	cache.Clear()

	if metadata, found := cache.Get("key1"); found || metadata != nil {
		t.Errorf("MemoryCache.Get() after Clear() should return (nil, false), got (%v, %v)", metadata, found)
	}

	if metadata, found := cache.Get("key2"); found || metadata != nil {
		t.Errorf("MemoryCache.Get() after Clear() should return (nil, false), got (%v, %v)", metadata, found)
	}
}

// TestMemoryCacheConcurrency tests that MemoryCache is safe for concurrent use.
func TestMemoryCacheConcurrency(t *testing.T) {
	cache := NewMemoryCache()

	// Concurrent writes
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(idx int) {
			metadata := &Metadata{EntityTypes: []EntityType{{Name: string(rune(idx))}}}
			cache.Set("key", metadata)
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	// Should have last value
	if metadata, found := cache.Get("key"); !found || metadata == nil {
		t.Errorf("MemoryCache.Get() after concurrent writes should return a value")
	}

	// Concurrent reads and writes
	for i := 0; i < 10; i++ {
		go func() {
			cache.Get("key")
			done <- true
		}()
		go func() {
			cache.Set("key", &Metadata{})
			done <- true
		}()
	}

	for i := 0; i < 20; i++ {
		<-done
	}

	t.Logf("Concurrency test passed")
}

// TestClientWithMemoryCache tests Client integration with MemoryCache.
func TestClientWithMemoryCache(t *testing.T) {
	cache := NewMemoryCache()

	// Create client with cache
	client, err := New(
		WithBaseURL("http://example.com"),
		WithMetadataCache(cache),
	)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Verify cache is set
	if client.metadataCache == nil {
		t.Fatalf("Client.metadataCache is nil")
	}

	if _, ok := client.metadataCache.(*MemoryCache); !ok {
		t.Fatalf("Client.metadataCache is not MemoryCache")
	}

	// Verify metadata caching interface
	testMetadata := &Metadata{EntityTypes: []EntityType{{Name: "TestEntity"}}}
	client.metadataCache.Set("test-key", testMetadata)

	if cached, found := client.metadataCache.Get("test-key"); !found {
		t.Fatalf("Metadata cache lookup failed")
	} else if cached.EntityTypes[0].Name != "TestEntity" {
		t.Fatalf("Cached metadata is incorrect")
	}
}

// TestClientWithNoOpCacheDefault tests that Client uses NoOpCache by default.
func TestClientWithNoOpCacheDefault(t *testing.T) {
	client, err := New(
		WithBaseURL("http://example.com"),
	)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Verify default cache is NoOpCache
	if _, ok := client.metadataCache.(*NoOpCache); !ok {
		t.Fatalf("Default Client.metadataCache should be NoOpCache, got %T", client.metadataCache)
	}

	// Verify it's truly a no-op
	testMetadata := &Metadata{EntityTypes: []EntityType{{Name: "TestEntity"}}}
	client.metadataCache.Set("test-key", testMetadata)

	if cached, found := client.metadataCache.Get("test-key"); found {
		t.Fatalf("NoOpCache should not cache, but found entry: %v", cached)
	}
}
