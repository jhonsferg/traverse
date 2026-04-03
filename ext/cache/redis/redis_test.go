package redis

import (
	"testing"
	"time"

	"github.com/jhonsferg/traverse"
	"github.com/stretchr/testify/assert"
)

// TestRedisConnection tests basic Redis connectivity
func TestRedisConnection(t *testing.T) {
	// Skip if Redis is not available
	if testing.Short() {
		t.Skip("Skipping Redis integration test in short mode")
	}

	cfg := &Config{
		Addr: "localhost:6379",
	}

	cache, err := New(cfg)
	if err != nil {
		t.Skipf("Redis not available: %v", err)
	}
	defer cache.Close()

	// Connection successful
	assert.NotNil(t, cache)
}

// TestCacheGetSet tests basic get/set functionality
func TestCacheGetSet(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Redis integration test in short mode")
	}

	cfg := &Config{
		Addr:      "localhost:6379",
		KeyPrefix: "test:traverse:",
		TTL:       time.Hour,
	}

	cache, err := New(cfg)
	if err != nil {
		t.Skipf("Redis not available: %v", err)
	}
	defer cache.Close()

	metadata := &traverse.Metadata{
		Version:   "1.0",
		Namespace: "test",
		EntityTypes: []traverse.EntityType{
			{Name: "Product"},
		},
	}

	// Test Set and Get
	cache.Set("test_key", metadata)
	retrieved, exists := cache.Get("test_key")

	assert.True(t, exists)
	assert.NotNil(t, retrieved)
	if retrieved != nil {
		assert.Equal(t, len(metadata.EntityTypes), len(retrieved.EntityTypes))
	}
}

// TestCacheGetMissing tests get on non-existent key
func TestCacheGetMissing(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Redis integration test in short mode")
	}

	cfg := &Config{
		Addr:      "localhost:6379",
		KeyPrefix: "test:traverse:",
	}

	cache, err := New(cfg)
	if err != nil {
		t.Skipf("Redis not available: %v", err)
	}
	defer cache.Close()

	retrieved, exists := cache.Get("nonexistent_key_" + time.Now().Format("20060102150405"))

	assert.False(t, exists)
	assert.Nil(t, retrieved)
}

// TestCacheClear tests clear functionality
func TestCacheClear(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Redis integration test in short mode")
	}

	cfg := &Config{
		Addr:      "localhost:6379",
		KeyPrefix: "test:clear:",
	}

	cache, err := New(cfg)
	if err != nil {
		t.Skipf("Redis not available: %v", err)
	}
	defer cache.Close()

	metadata := &traverse.Metadata{
		Version:   "1.0",
		Namespace: "test",
		EntityTypes: []traverse.EntityType{
			{Name: "Product"},
		},
	}

	cache.Set("key1", metadata)
	cache.Set("key2", metadata)

	size := cache.Size()
	assert.Greater(t, size, int64(0))

	cache.Clear()

	size = cache.Size()
	assert.Equal(t, int64(0), size)
}

// TestCacheDelete tests delete functionality
func TestCacheDelete(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Redis integration test in short mode")
	}

	cfg := &Config{
		Addr:      "localhost:6379",
		KeyPrefix: "test:delete:",
	}

	cache, err := New(cfg)
	if err != nil {
		t.Skipf("Redis not available: %v", err)
	}
	defer cache.Close()

	metadata := &traverse.Metadata{
		Version:   "1.0",
		Namespace: "test",
		EntityTypes: []traverse.EntityType{
			{Name: "Product"},
		},
	}

	cache.Set("test_key", metadata)
	assert.True(t, cache.Exists("test_key"))

	cache.Delete("test_key")
	assert.False(t, cache.Exists("test_key"))

	retrieved, exists := cache.Get("test_key")
	assert.False(t, exists)
	assert.Nil(t, retrieved)
}

// TestCacheExists tests exists functionality
func TestCacheExists(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Redis integration test in short mode")
	}

	cfg := &Config{
		Addr:      "localhost:6379",
		KeyPrefix: "test:exists:",
	}

	cache, err := New(cfg)
	if err != nil {
		t.Skipf("Redis not available: %v", err)
	}
	defer cache.Close()

	metadata := &traverse.Metadata{
		Version:   "1.0",
		Namespace: "test",
		EntityTypes: []traverse.EntityType{
			{Name: "Product"},
		},
	}

	cache.Set("test_key", metadata)
	assert.True(t, cache.Exists("test_key"))
	assert.False(t, cache.Exists("nonexistent_key"))
}

// TestCacheSetTTL tests TTL functionality
func TestCacheSetTTL(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Redis integration test in short mode")
	}

	cfg := &Config{
		Addr:      "localhost:6379",
		KeyPrefix: "test:ttl:",
		TTL:       10 * time.Second,
	}

	cache, err := New(cfg)
	if err != nil {
		t.Skipf("Redis not available: %v", err)
	}
	defer cache.Close()

	metadata := &traverse.Metadata{
		Version:   "1.0",
		Namespace: "test",
		EntityTypes: []traverse.EntityType{
			{Name: "Product"},
		},
	}

	cache.Set("test_key", metadata)
	assert.True(t, cache.Exists("test_key"))

	// Update TTL
	result := cache.SetTTL("test_key", 100*time.Millisecond)
	assert.True(t, result)

	// Should still exist immediately
	assert.True(t, cache.Exists("test_key"))

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Should be expired
	assert.False(t, cache.Exists("test_key"))
}

// TestCacheMultipleKeys tests multiple keys functionality
func TestCacheMultipleKeys(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Redis integration test in short mode")
	}

	cfg := &Config{
		Addr:      "localhost:6379",
		KeyPrefix: "test:multi:",
	}

	cache, err := New(cfg)
	if err != nil {
		t.Skipf("Redis not available: %v", err)
	}
	defer cache.Close()

	meta1 := &traverse.Metadata{
		Version:   "1.0",
		Namespace: "test",
		EntityTypes: []traverse.EntityType{
			{Name: "E1"},
		},
	}
	meta2 := &traverse.Metadata{
		Version:   "1.0",
		Namespace: "test",
		EntityTypes: []traverse.EntityType{
			{Name: "E2"},
		},
	}

	cache.Set("key1", meta1)
	cache.Set("key2", meta2)

	retrieved1, exists1 := cache.Get("key1")
	retrieved2, exists2 := cache.Get("key2")

	assert.True(t, exists1)
	assert.True(t, exists2)
	assert.NotNil(t, retrieved1)
	assert.NotNil(t, retrieved2)
}

// TestCacheImplementsInterface tests that Cache implements CacheStore
func TestCacheImplementsInterface(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Redis integration test in short mode")
	}

	cfg := &Config{
		Addr: "localhost:6379",
	}

	cache, err := New(cfg)
	if err != nil {
		t.Skipf("Redis not available: %v", err)
	}
	defer cache.Close()

	var _ traverse.CacheStore = cache
}

// TestCacheKeyPrefix tests key prefixing
func TestCacheKeyPrefix(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Redis integration test in short mode")
	}

	cfg1 := &Config{
		Addr:      "localhost:6379",
		KeyPrefix: "app1:",
	}

	cfg2 := &Config{
		Addr:      "localhost:6379",
		KeyPrefix: "app2:",
	}

	cache1, err := New(cfg1)
	if err != nil {
		t.Skipf("Redis not available: %v", err)
	}
	defer cache1.Close()

	cache2, err := New(cfg2)
	if err != nil {
		t.Skipf("Redis not available: %v", err)
	}
	defer cache2.Close()

	metadata := &traverse.Metadata{
		Version:   "1.0",
		Namespace: "test",
		EntityTypes: []traverse.EntityType{
			{Name: "Product"},
		},
	}

	cache1.Set("key", metadata)

	// Should be accessible from cache1
	retrieved1, exists1 := cache1.Get("key")
	assert.True(t, exists1)
	assert.NotNil(t, retrieved1)

	// Should NOT be accessible from cache2 (different prefix)
	retrieved2, exists2 := cache2.Get("key")
	assert.False(t, exists2)
	assert.Nil(t, retrieved2)

	cache2.Clear()
	cache1.Clear()
}

// BenchmarkRedisGet benchmarks Get operation
func BenchmarkRedisGet(b *testing.B) {
	cfg := &Config{
		Addr:      "localhost:6379",
		KeyPrefix: "bench:traverse:",
	}

	cache, err := New(cfg)
	if err != nil {
		b.Skipf("Redis not available: %v", err)
	}
	defer cache.Close()

	metadata := &traverse.Metadata{
		Version:   "1.0",
		Namespace: "test",
		EntityTypes: []traverse.EntityType{
			{Name: "Product"},
		},
	}
	cache.Set("bench_key", metadata)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Get("bench_key")
	}
}

// BenchmarkRedisSet benchmarks Set operation
func BenchmarkRedisSet(b *testing.B) {
	cfg := &Config{
		Addr:      "localhost:6379",
		KeyPrefix: "bench:traverse:",
	}

	cache, err := New(cfg)
	if err != nil {
		b.Skipf("Redis not available: %v", err)
	}
	defer cache.Close()

	metadata := &traverse.Metadata{
		Version:   "1.0",
		Namespace: "test",
		EntityTypes: []traverse.EntityType{
			{Name: "Product"},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Set("bench_key", metadata)
	}
}
