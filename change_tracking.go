package traverse

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
)

// TrackedEntity wraps an entity snapshot and tracks which fields have been
// modified. Use it to generate minimal PATCH payloads with only changed fields.
//
// Usage:
//
//	// Fetch an entity
//	entity := map[string]interface{}{"ID": 42, "Name": "Widget", "Price": 9.99}
//
//	// Start tracking
//	t := traverse.TrackEntity(entity)
//	t.Set("Name", "Updated Widget")
//	t.Set("Price", 19.99)
//
//	// Save only changed fields
//	err = t.SaveChanges(ctx, client, "Products", 42)
type TrackedEntity struct {
	mu       sync.RWMutex
	original map[string]interface{}
	current  map[string]interface{}
	dirty    map[string]bool
}

// TrackEntity creates a TrackedEntity from an entity snapshot.
// The snapshot is copied so the original is preserved for comparison.
func TrackEntity(entity map[string]interface{}) *TrackedEntity {
	orig := make(map[string]interface{}, len(entity))
	for k, v := range entity {
		orig[k] = v
	}
	curr := make(map[string]interface{}, len(entity))
	for k, v := range entity {
		curr[k] = v
	}
	return &TrackedEntity{
		original: orig,
		current:  curr,
		dirty:    make(map[string]bool),
	}
}

// Set updates a field value and marks it dirty.
func (t *TrackedEntity) Set(field string, value interface{}) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.current[field] = value
	t.dirty[field] = true
}

// Get returns the current value of a field.
func (t *TrackedEntity) Get(field string) (interface{}, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	v, ok := t.current[field]
	return v, ok
}

// IsDirty reports whether any field has been modified.
func (t *TrackedEntity) IsDirty() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return len(t.dirty) > 0
}

// DirtyFields returns the names of all modified fields.
func (t *TrackedEntity) DirtyFields() []string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	fields := make([]string, 0, len(t.dirty))
	for f := range t.dirty {
		fields = append(fields, f)
	}
	return fields
}

// Changes returns a map containing only the fields that have been modified.
// This is the payload to send in a PATCH request.
func (t *TrackedEntity) Changes() map[string]interface{} {
	t.mu.RLock()
	defer t.mu.RUnlock()
	patch := make(map[string]interface{}, len(t.dirty))
	for field := range t.dirty {
		patch[field] = t.current[field]
	}
	return patch
}

// Original returns the unmodified snapshot of the entity as it was when
// TrackEntity was called.
func (t *TrackedEntity) Original() map[string]interface{} {
	t.mu.RLock()
	defer t.mu.RUnlock()
	cp := make(map[string]interface{}, len(t.original))
	for k, v := range t.original {
		cp[k] = v
	}
	return cp
}

// Reset clears all dirty flags, making the current state the new baseline.
// Use this after a successful save to reset the tracker.
func (t *TrackedEntity) Reset() {
	t.mu.Lock()
	defer t.mu.Unlock()
	for k, v := range t.current {
		t.original[k] = v
	}
	t.dirty = make(map[string]bool)
}

// Discard reverts all changes back to the original snapshot.
func (t *TrackedEntity) Discard() {
	t.mu.Lock()
	defer t.mu.Unlock()
	for k, v := range t.original {
		t.current[k] = v
	}
	t.dirty = make(map[string]bool)
}

// MarshalJSON implements json.Marshaler and encodes only the changed fields.
func (t *TrackedEntity) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.Changes())
}

// SaveChanges saves the dirty fields to the OData service using a PATCH request.
// If no fields are dirty, it is a no-op and returns nil.
//
//	t := traverse.TrackEntity(entity)
//	t.Set("Name", "New Name")
//	err := t.SaveChanges(ctx, client, "Products", 42)
func (t *TrackedEntity) SaveChanges(ctx context.Context, client *Client, entitySet string, key interface{}) error {
	if !t.IsDirty() {
		return nil
	}
	changes := t.Changes()

	keyStr, err := encodeKey(key)
	if err != nil {
		return fmt.Errorf("traverse: invalid key: %w", err)
	}

	path := fmt.Sprintf("/%s(%s)", entitySet, keyStr)
	r := client.http.Patch(path)
	r = r.WithJSON(changes)
	r = r.WithContext(ctx)

	resp, execErr := client.http.Execute(r)
	if execErr != nil {
		return fmt.Errorf("traverse: SaveChanges failed: %w", execErr)
	}
	if resp.StatusCode != 200 && resp.StatusCode != 204 {
		return fmt.Errorf("traverse: SaveChanges returned status %d", resp.StatusCode)
	}
	t.Reset()
	return nil
}
