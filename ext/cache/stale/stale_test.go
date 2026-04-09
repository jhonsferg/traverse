package stale

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestCacheMiss(t *testing.T) {
	cache := New(Config{
		TTL:      100 * time.Millisecond,
		StaleTTL: 1 * time.Second,
	})

	var called int
	refreshFn := func(ctx context.Context) ([]byte, error) {
		called++
		return []byte("fresh data"), nil
	}

	ctx := context.Background()
	data, err := cache.Get(ctx, "key1", refreshFn)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(data) != "fresh data" {
		t.Errorf("expected 'fresh data', got %q", string(data))
	}
	if called != 1 {
		t.Errorf("expected refreshFn to be called once, called %d times", called)
	}
}

func TestCacheHitWithinTTL(t *testing.T) {
	cache := New(Config{
		TTL:      100 * time.Millisecond,
		StaleTTL: 1 * time.Second,
	})

	var called int
	refreshFn := func(ctx context.Context) ([]byte, error) {
		called++
		return []byte("fresh data"), nil
	}

	ctx := context.Background()

	data1, err := cache.Get(ctx, "key1", refreshFn)
	if err != nil || string(data1) != "fresh data" {
		t.Fatalf("first get failed: err=%v, data=%q", err, string(data1))
	}

	data2, err := cache.Get(ctx, "key1", refreshFn)
	if err != nil || string(data2) != "fresh data" {
		t.Fatalf("second get failed: err=%v, data=%q", err, string(data2))
	}

	if called != 1 {
		t.Errorf("expected refreshFn to be called once (cache hit), called %d times", called)
	}
}

func TestCacheStaleWithinStaleTTL(t *testing.T) {
	cache := New(Config{
		TTL:            50 * time.Millisecond,
		StaleTTL:       200 * time.Millisecond,
		BackgroundSync: false,
	})

	var called int
	refreshFn := func(ctx context.Context) ([]byte, error) {
		called++
		return []byte("data"), nil
	}

	ctx := context.Background()

	_, err := cache.Get(ctx, "key1", refreshFn)
	if err != nil {
		t.Fatalf("first get failed: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	data, err := cache.Get(ctx, "key1", refreshFn)
	if err != nil || string(data) != "data" {
		t.Fatalf("stale get failed: err=%v, data=%q", err, string(data))
	}

	if called != 1 {
		t.Errorf("expected refreshFn called once (stale, no sync), called %d times", called)
	}
}

func TestCacheBackgroundRefresh(t *testing.T) {
	cache := New(Config{
		TTL:            50 * time.Millisecond,
		StaleTTL:       500 * time.Millisecond,
		BackgroundSync: true,
	})

	var called int32
	refreshFn := func(ctx context.Context) ([]byte, error) {
		atomic.AddInt32(&called, 1)
		return []byte("data"), nil
	}

	ctx := context.Background()

	_, err := cache.Get(ctx, "key1", refreshFn)
	if err != nil {
		t.Fatalf("first get failed: %v", err)
	}

	if atomic.LoadInt32(&called) != 1 {
		t.Errorf("expected 1 call after first get, got %d", atomic.LoadInt32(&called))
	}

	time.Sleep(100 * time.Millisecond)

	_, err = cache.Get(ctx, "key1", refreshFn)
	if err != nil {
		t.Fatalf("stale get failed: %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	if atomic.LoadInt32(&called) != 2 {
		t.Errorf("expected 2 calls (background refresh), got %d", atomic.LoadInt32(&called))
	}
}

func TestCacheTooStale(t *testing.T) {
	cache := New(Config{
		TTL:      50 * time.Millisecond,
		StaleTTL: 100 * time.Millisecond,
	})

	var called int
	refreshFn := func(ctx context.Context) ([]byte, error) {
		called++
		return []byte("data"), nil
	}

	ctx := context.Background()

	_, err := cache.Get(ctx, "key1", refreshFn)
	if err != nil {
		t.Fatalf("first get failed: %v", err)
	}

	time.Sleep(150 * time.Millisecond)

	_, err = cache.Get(ctx, "key1", refreshFn)
	if err != nil {
		t.Fatalf("too-stale get failed: %v", err)
	}

	if called != 2 {
		t.Errorf("expected 2 calls (synchronous refresh after StaleTTL), called %d times", called)
	}
}

func TestRefreshFnError(t *testing.T) {
	cache := New(Config{
		TTL:      100 * time.Millisecond,
		StaleTTL: 1 * time.Second,
	})

	testErr := errors.New("refresh failed")
	refreshFn := func(ctx context.Context) ([]byte, error) {
		return nil, testErr
	}

	ctx := context.Background()
	data, err := cache.Get(ctx, "key1", refreshFn)

	if err != testErr {
		t.Errorf("expected error %v, got %v", testErr, err)
	}
	if data != nil {
		t.Errorf("expected nil data on error, got %q", string(data))
	}
}

func TestInvalidate(t *testing.T) {
	cache := New(Config{
		TTL:      100 * time.Millisecond,
		StaleTTL: 1 * time.Second,
	})

	var called int
	refreshFn := func(ctx context.Context) ([]byte, error) {
		called++
		return []byte("data"), nil
	}

	ctx := context.Background()

	_, err := cache.Get(ctx, "key1", refreshFn)
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}

	if called != 1 {
		t.Errorf("expected 1 call, got %d", called)
	}

	cache.Invalidate("key1")

	_, err = cache.Get(ctx, "key1", refreshFn)
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}

	if called != 2 {
		t.Errorf("expected 2 calls after invalidate, got %d", called)
	}
}

func TestConcurrentAccess(t *testing.T) {
	cache := New(Config{
		TTL:            100 * time.Millisecond,
		StaleTTL:       500 * time.Millisecond,
		BackgroundSync: true,
	})

	var called int32
	refreshFn := func(ctx context.Context) ([]byte, error) {
		atomic.AddInt32(&called, 1)
		return []byte("data"), nil
	}

	ctx := context.Background()

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := cache.Get(ctx, "key1", refreshFn)
			if err != nil {
				t.Errorf("concurrent get failed: %v", err)
			}
		}()
	}

	wg.Wait()

	if atomic.LoadInt32(&called) == 0 {
		t.Fatal("expected at least one call")
	}

	data, err := cache.Get(ctx, "key1", refreshFn)
	if err != nil || len(data) == 0 {
		t.Errorf("concurrent access resulted in invalid cache state: err=%v", err)
	}
}

func TestMultipleKeys(t *testing.T) {
	cache := New(Config{
		TTL:      100 * time.Millisecond,
		StaleTTL: 1 * time.Second,
	})

	var called int
	refreshFn := func(ctx context.Context) ([]byte, error) {
		called++
		return []byte("data"), nil
	}

	ctx := context.Background()

	for i := 0; i < 3; i++ {
		_, err := cache.Get(ctx, "key"+string(rune('0'+i)), refreshFn)
		if err != nil {
			t.Fatalf("get failed: %v", err)
		}
	}

	if called != 3 {
		t.Errorf("expected 3 calls, got %d", called)
	}

	for i := 0; i < 3; i++ {
		_, err := cache.Get(ctx, "key"+string(rune('0'+i)), refreshFn)
		if err != nil {
			t.Fatalf("get failed: %v", err)
		}
	}

	if called != 3 {
		t.Errorf("expected 3 calls (all cache hits), got %d", called)
	}
}

func TestCache_BackgroundRefreshErrorCallback(t *testing.T) {
	done := make(chan struct{}, 1)
	var mu sync.Mutex
	var cbKey string
	var cbErr error

	cfg := Config{
		TTL:            10 * time.Millisecond,
		StaleTTL:       500 * time.Millisecond,
		BackgroundSync: true,
		OnError: func(key string, err error) {
			mu.Lock()
			cbKey = key
			cbErr = err
			mu.Unlock()
			select {
			case done <- struct{}{}:
			default:
			}
		},
	}
	c := New(cfg)

	// Seed cache with valid data.
	_, _ = c.Get(context.Background(), "k", func(_ context.Context) ([]byte, error) {
		return []byte("v"), nil
	})

	// Let TTL expire so next Get sees stale and triggers background refresh.
	time.Sleep(20 * time.Millisecond)

	fetchErr := errors.New("backend down")
	_, _ = c.Get(context.Background(), "k", func(_ context.Context) ([]byte, error) {
		return nil, fetchErr
	})

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("OnError callback was not called within 5s timeout")
	}

	mu.Lock()
	gotKey, gotErr := cbKey, cbErr
	mu.Unlock()

	if gotKey != "k" || gotErr != fetchErr {
		t.Fatalf("expected OnError callback with key=k err=%v, got key=%q err=%v", fetchErr, gotKey, gotErr)
	}
}
