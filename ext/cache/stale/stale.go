// Package stale provides a stale-while-revalidate cache implementation.
package stale

import (
	"context"
	"sync"
	"time"
)

// Config configures the stale-while-revalidate cache.
type Config struct {
	TTL            time.Duration
	StaleTTL       time.Duration
	BackgroundSync bool
}

// Cache provides stale-while-revalidate semantics for arbitrary data.
type Cache struct {
	cfg       Config
	mu        sync.RWMutex
	entries   map[string]*entry
	onRefresh func(key string, data []byte)
}

type entry struct {
	data       []byte
	timestamp  time.Time
	refreshing bool
}

// New creates a new stale-while-revalidate cache with the given configuration.
func New(cfg Config) *Cache {
	return &Cache{
		cfg:     cfg,
		entries: make(map[string]*entry),
	}
}

// Get returns cached data for key if available and not too stale.
func (c *Cache) Get(ctx context.Context, key string, refreshFn func(ctx context.Context) ([]byte, error)) ([]byte, error) {
	c.mu.RLock()
	_, exists := c.entries[key]
	c.mu.RUnlock()

	if !exists {
		return c.refreshSync(ctx, key, refreshFn)
	}

	now := time.Now()

	c.mu.RLock()
	entry, exists := c.entries[key]
	if !exists {
		c.mu.RUnlock()
		return c.refreshSync(ctx, key, refreshFn)
	}

	data := entry.data
	timestamp := entry.timestamp
	refreshing := entry.refreshing
	c.mu.RUnlock()

	age := now.Sub(timestamp)

	if age < c.cfg.TTL {
		return data, nil
	}

	if age < c.cfg.StaleTTL {
		if c.cfg.BackgroundSync && !refreshing {
			//nolint:contextcheck
			c.startBackgroundRefresh(key, refreshFn)
		}
		return data, nil
	}

	return c.refreshSync(ctx, key, refreshFn)
}

// refreshSync fetches fresh data and updates the cache.
func (c *Cache) refreshSync(ctx context.Context, key string, refreshFn func(ctx context.Context) ([]byte, error)) ([]byte, error) {
	data, err := refreshFn(ctx)
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	c.entries[key] = &entry{
		data:       data,
		timestamp:  time.Now(),
		refreshing: false,
	}
	c.mu.Unlock()

	if c.onRefresh != nil {
		c.onRefresh(key, data)
	}

	return data, nil
}

// startBackgroundRefresh triggers a background refresh without blocking.
func (c *Cache) startBackgroundRefresh(key string, refreshFn func(ctx context.Context) ([]byte, error)) {
	c.mu.Lock()
	if entry, exists := c.entries[key]; exists {
		entry.refreshing = true
	}
	c.mu.Unlock()

	go func() {
		bgCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		data, err := refreshFn(bgCtx)

		c.mu.Lock()
		if entry, exists := c.entries[key]; exists {
			entry.refreshing = false
			if err == nil {
				entry.data = data
				entry.timestamp = time.Now()
			}
		}
		c.mu.Unlock()

		if err == nil && c.onRefresh != nil {
			c.onRefresh(key, data)
		}
	}()
}

// Invalidate removes the cache entry for key.
func (c *Cache) Invalidate(key string) {
	c.mu.Lock()
	delete(c.entries, key)
	c.mu.Unlock()
}

// OnRefresh sets a callback called after each refresh.
func (c *Cache) OnRefresh(fn func(key string, data []byte)) {
	c.mu.Lock()
	c.onRefresh = fn
	c.mu.Unlock()
}

// Clear removes all cached entries.
func (c *Cache) Clear() {
	c.mu.Lock()
	c.entries = make(map[string]*entry)
	c.mu.Unlock()
}
