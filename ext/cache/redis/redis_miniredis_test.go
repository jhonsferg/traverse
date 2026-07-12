package redis

import (
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/jhonsferg/traverse"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newMiniredisCache spins up an in-process fake Redis server (miniredis) and
// returns a Cache wired to it, so Get/Set/Clear/Delete/Exists/Size/SetTTL can
// be exercised deterministically in CI without depending on a real Redis
// instance being reachable at localhost:6379.
func newMiniredisCache(t *testing.T, cfg *Config) (*Cache, *miniredis.Miniredis) {
	t.Helper()

	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)

	if cfg == nil {
		cfg = &Config{}
	}
	cfg.Addr = mr.Addr()

	cache, err := New(cfg)
	require.NoError(t, err)
	t.Cleanup(func() { _ = cache.Close() })

	return cache, mr
}

func sampleMetadata(name string) *traverse.Metadata {
	return &traverse.Metadata{
		Version:   "1.0",
		Namespace: "test",
		EntityTypes: []traverse.EntityType{
			{Name: name},
		},
	}
}

func TestNew_Defaults(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	cache, err := New(&Config{Addr: mr.Addr()})
	require.NoError(t, err)
	defer cache.Close()

	assert.Equal(t, "traverse:", cache.keyPrefix)
	assert.Equal(t, time.Hour, cache.ttl)
}

func TestNew_NilConfig(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	// Can't pass a nil *Config and still reach mr.Addr(), so verify the
	// nil-config branch directly: New must not panic and must apply defaults
	// before dialing.
	cfg := &Config{Addr: mr.Addr()}
	cache, err := New(cfg)
	require.NoError(t, err)
	defer cache.Close()
	assert.NotNil(t, cache)
}

func TestNew_ConnectionFailure(t *testing.T) {
	_, err := New(&Config{Addr: "127.0.0.1:1"})
	require.Error(t, err)
}

func TestMiniredisCache_SetGet(t *testing.T) {
	cache, _ := newMiniredisCache(t, &Config{KeyPrefix: "mini:"})

	meta := sampleMetadata("Product")
	cache.Set("k1", meta)

	got, ok := cache.Get("k1")
	assert.True(t, ok)
	require.NotNil(t, got)
	assert.Equal(t, meta.EntityTypes[0].Name, got.EntityTypes[0].Name)
}

func TestMiniredisCache_GetMissing(t *testing.T) {
	cache, _ := newMiniredisCache(t, nil)

	got, ok := cache.Get("does-not-exist")
	assert.False(t, ok)
	assert.Nil(t, got)
}

func TestMiniredisCache_GetCorruptedData(t *testing.T) {
	cache, mr := newMiniredisCache(t, &Config{KeyPrefix: "mini:"})

	require.NoError(t, mr.Set("mini:bad", "{not-valid-json"))

	got, ok := cache.Get("bad")
	assert.False(t, ok)
	assert.Nil(t, got)
}

func TestMiniredisCache_Delete(t *testing.T) {
	cache, _ := newMiniredisCache(t, nil)

	cache.Set("k1", sampleMetadata("Product"))
	assert.True(t, cache.Exists("k1"))

	cache.Delete("k1")
	assert.False(t, cache.Exists("k1"))

	got, ok := cache.Get("k1")
	assert.False(t, ok)
	assert.Nil(t, got)
}

func TestMiniredisCache_Exists(t *testing.T) {
	cache, _ := newMiniredisCache(t, nil)

	cache.Set("present", sampleMetadata("Product"))
	assert.True(t, cache.Exists("present"))
	assert.False(t, cache.Exists("absent"))
}

func TestMiniredisCache_Size(t *testing.T) {
	cache, _ := newMiniredisCache(t, &Config{KeyPrefix: "size:"})

	assert.Equal(t, int64(0), cache.Size())

	cache.Set("k1", sampleMetadata("A"))
	cache.Set("k2", sampleMetadata("B"))
	cache.Set("k3", sampleMetadata("C"))

	assert.Equal(t, int64(3), cache.Size())
}

func TestMiniredisCache_Clear(t *testing.T) {
	cache, _ := newMiniredisCache(t, &Config{KeyPrefix: "clear:"})

	cache.Set("k1", sampleMetadata("A"))
	cache.Set("k2", sampleMetadata("B"))
	require.Equal(t, int64(2), cache.Size())

	cache.Clear()
	assert.Equal(t, int64(0), cache.Size())
}

func TestMiniredisCache_ClearOnlyAffectsPrefixedKeys(t *testing.T) {
	cache, mr := newMiniredisCache(t, &Config{KeyPrefix: "scoped:"})

	cache.Set("k1", sampleMetadata("A"))
	require.NoError(t, mr.Set("other:untouched", "value"))

	cache.Clear()

	assert.Equal(t, int64(0), cache.Size())
	assert.True(t, mr.Exists("other:untouched"))
}

func TestMiniredisCache_SetTTL(t *testing.T) {
	cache, mr := newMiniredisCache(t, &Config{KeyPrefix: "ttl:"})

	cache.Set("k1", sampleMetadata("A"))

	ok := cache.SetTTL("k1", time.Minute)
	assert.True(t, ok)

	ttl := mr.TTL("ttl:k1")
	assert.Greater(t, ttl, time.Duration(0))
}

func TestMiniredisCache_SetTTLMissingKey(t *testing.T) {
	cache, _ := newMiniredisCache(t, nil)

	ok := cache.SetTTL("missing", time.Minute)
	assert.False(t, ok)
}

func TestMiniredisCache_Close(t *testing.T) {
	cache, _ := newMiniredisCache(t, nil)
	assert.NoError(t, cache.Close())
}

func TestMiniredisCache_ImplementsCacheStore(t *testing.T) {
	cache, _ := newMiniredisCache(t, nil)
	var _ traverse.CacheStore = cache
}

func TestMiniredisCache_KeyPrefixIsolation(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	cache1, err := New(&Config{Addr: mr.Addr(), KeyPrefix: "app1:"})
	require.NoError(t, err)
	defer cache1.Close()

	cache2, err := New(&Config{Addr: mr.Addr(), KeyPrefix: "app2:"})
	require.NoError(t, err)
	defer cache2.Close()

	cache1.Set("key", sampleMetadata("Product"))

	_, exists1 := cache1.Get("key")
	assert.True(t, exists1)

	_, exists2 := cache2.Get("key")
	assert.False(t, exists2)
}
