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
// SaveChanges is safe to call concurrently: it atomically snapshots the dirty
// fields and clears only those fields on success. Any Set calls that race with
// the in-flight PATCH remain dirty and will be included in the next SaveChanges.
//
//	t := traverse.TrackEntity(entity)
//	t.Set("Name", "New Name")
//	err := t.SaveChanges(ctx, client, "Products", 42)
func (t *TrackedEntity) SaveChanges(ctx context.Context, client *Client, entitySet string, key interface{}) error {
	// Phase 1: atomically snapshot dirty fields and their pre-save originals.
	// We optimistically advance original[field] to current[field] and remove
	// the dirty flag so that concurrent Set() calls on other fields are not
	// affected. On failure we restore the previous originals and re-dirty only
	// the fields we attempted to save.
	t.mu.Lock()
	if len(t.dirty) == 0 {
		t.mu.Unlock()
		return nil
	}
	patch := make(map[string]interface{}, len(t.dirty))
	prevOriginals := make(map[string]interface{}, len(t.dirty))
	for field := range t.dirty {
		patch[field] = t.current[field]
		prevOriginals[field] = t.original[field]
		t.original[field] = t.current[field]
		delete(t.dirty, field)
	}
	t.mu.Unlock()

	keyStr, err := encodeKey(key)
	if err != nil {
		t.restoreDirty(prevOriginals)
		return fmt.Errorf("traverse: invalid key: %w", err)
	}

	entityPath, rawQuery := splitEntityPath(entitySet)
	path := fmt.Sprintf("/%s(%s)%s", entityPath, keyStr, rawQuery)
	r := client.http.Patch(path).WithJSON(patch).WithContext(ctx)

	resp, execErr := client.http.Execute(r)
	if execErr != nil {
		t.restoreDirty(prevOriginals)
		return fmt.Errorf("traverse: SaveChanges failed: %w", execErr)
	}
	if resp.StatusCode != 200 && resp.StatusCode != 204 {
		t.restoreDirty(prevOriginals)
		return fmt.Errorf("traverse: SaveChanges returned status %d", resp.StatusCode)
	}
	return nil
}

// restoreDirty is called after a failed PATCH. It reinstates the pre-send
// original values and re-marks each field as dirty so the next SaveChanges
// retries with the correct values. Fields that were set again by a concurrent
// Set() call between the snapshot and the failure are already in dirty and
// keep their newer value; we only restore the original pointer without
// overwriting the dirty flag.
func (t *TrackedEntity) restoreDirty(prevOriginals map[string]interface{}) {
	t.mu.Lock()
	defer t.mu.Unlock()
	for field, prev := range prevOriginals {
		t.original[field] = prev
		t.dirty[field] = true
	}
}
