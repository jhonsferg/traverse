package traverse

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/jhonsferg/relay"
)

// ETag wraps an entity value together with its HTTP ETag for concurrency control.
// Obtain an ETag value via [Client.ReadWithETag] and pass it back to
// [Client.UpdateWithETag], [Client.ReplaceWithETag], or [Client.DeleteWithETag]
// to detect concurrent modifications.
type ETag struct {
	// Value is the raw ETag string as returned by the server in the ETag header
	// (e.g. W/"123", "abc456").
	Value string
}

// IsEmpty reports whether the ETag is unset.
func (e ETag) IsEmpty() bool { return e.Value == "" }

// IsWeak reports whether this is a weak ETag (prefixed with W/).
func (e ETag) IsWeak() bool { return len(e.Value) > 2 && e.Value[:2] == `W/` }

// String returns the raw ETag value.
func (e ETag) String() string { return e.Value }

// EntityWithETag pairs a fetched entity map with its server ETag.
type EntityWithETag struct {
	// Entity is the entity properties as returned by the server.
	Entity map[string]interface{}
	// ETag is the concurrency token for this entity revision.
	ETag ETag
}

// ReadWithETag fetches a single entity by key and returns it together with
// the server ETag. The ETag can be passed to [Client.UpdateWithETag],
// [Client.ReplaceWithETag], or [Client.DeleteWithETag] to guard against
// concurrent modifications.
//
// The ETag is extracted from the response ETag header. If the server does not
// return an ETag header, EntityWithETag.ETag.IsEmpty() returns true - the
// entity is still returned normally.
//
//	entity, err := client.ReadWithETag(ctx, "Products", 42)
//	if err != nil { ... }
//	// modify entity.Entity ...
//	err = client.UpdateWithETag(ctx, "Products", 42, changes, entity.ETag)
func (c *Client) ReadWithETag(ctx context.Context, entitySet string, key interface{}) (*EntityWithETag, error) {
	keyStr, err := encodeKey(key)
	if err != nil {
		return nil, fmt.Errorf("traverse: invalid key: %w", err)
	}

	entityPath, rawQuery := splitEntityPath(entitySet)
	path := fmt.Sprintf("/%s(%s)%s", entityPath, keyStr, rawQuery)
	req := c.http.Get(path)
	req = req.WithContext(ctx)

	resp, err := c.http.Execute(req)
	if err != nil {
		return nil, fmt.Errorf("traverse: read failed: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("traverse: read returned status %d", resp.StatusCode)
	}

	var entity map[string]interface{}
	if c.version == ODataV2 {
		var wrapped struct {
			D map[string]interface{} `json:"d"`
		}
		if decErr := json.NewDecoder(resp.BodyReader()).Decode(&wrapped); decErr != nil {
			return nil, fmt.Errorf("traverse: failed to decode entity: %w", decErr)
		}
		entity = wrapped.D
	} else {
		if decErr := json.NewDecoder(resp.BodyReader()).Decode(&entity); decErr != nil {
			return nil, fmt.Errorf("traverse: failed to decode entity: %w", decErr)
		}
	}

	return &EntityWithETag{
		Entity: entity,
		ETag:   ETag{Value: resp.Headers.Get("ETag")},
	}, nil
}

// UpdateWithETag performs a partial update (PATCH/MERGE) guarded by an ETag.
// If etag is non-empty, an If-Match header is sent. The server will reject
// the update with HTTP 412 Precondition Failed if the entity was modified
// since the ETag was fetched; this is translated to [ErrConcurrencyConflict].
//
// Pass a zero ETag (or ETag{}) to skip the concurrency check.
//
//	entity, err := client.ReadWithETag(ctx, "Products", 42)
//	// ...
//	err = client.UpdateWithETag(ctx, "Products", 42, map[string]any{"Price": 9.99}, entity.ETag)
func (c *Client) UpdateWithETag(ctx context.Context, entitySet string, key interface{}, data interface{}, etag ETag) error {
	keyStr, err := encodeKey(key)
	if err != nil {
		return fmt.Errorf("traverse: invalid key: %w", err)
	}

	entityPath, rawQuery := splitEntityPath(entitySet)
	path := fmt.Sprintf("/%s(%s)%s", entityPath, keyStr, rawQuery)

	var resp *relay.Response
	var execErr error

	if c.version == ODataV2 {
		// OData v2 uses PUT with X-HTTP-Method: MERGE to emulate MERGE.
		r := c.http.Put(path)
		r = r.WithHeader("X-HTTP-Method", "MERGE")
		r = r.WithJSON(data)
		r = r.WithContext(ctx)
		if !etag.IsEmpty() {
			r = r.WithHeader("If-Match", etag.Value)
		}
		resp, execErr = c.http.Execute(r)
	} else {
		r := c.http.Patch(path)
		r = r.WithJSON(data)
		r = r.WithContext(ctx)
		if !etag.IsEmpty() {
			r = r.WithHeader("If-Match", etag.Value)
		}
		resp, execErr = c.http.Execute(r)
	}

	if execErr != nil {
		return fmt.Errorf("traverse: update failed: %w", execErr)
	}
	return checkUpdateResponse(resp)
}

// ReplaceWithETag performs a full replacement (PUT) guarded by an ETag.
// If etag is non-empty, an If-Match header is sent to guard against concurrent
// modifications (HTTP 412 -> [ErrConcurrencyConflict]).
func (c *Client) ReplaceWithETag(ctx context.Context, entitySet string, key interface{}, data interface{}, etag ETag) error {
	keyStr, err := encodeKey(key)
	if err != nil {
		return fmt.Errorf("traverse: invalid key: %w", err)
	}

	entityPath, rawQuery := splitEntityPath(entitySet)
	path := fmt.Sprintf("/%s(%s)%s", entityPath, keyStr, rawQuery)
	r := c.http.Put(path)
	r = r.WithJSON(data)
	r = r.WithContext(ctx)
	if !etag.IsEmpty() {
		r = r.WithHeader("If-Match", etag.Value)
	}
	resp, execErr := c.http.Execute(r)
	if execErr != nil {
		return fmt.Errorf("traverse: replace failed: %w", execErr)
	}
	return checkUpdateResponse(resp)
}

// DeleteWithETag deletes an entity guarded by an ETag.
// If etag is non-empty, an If-Match header is sent. HTTP 412 is translated to
// [ErrConcurrencyConflict].
func (c *Client) DeleteWithETag(ctx context.Context, entitySet string, key interface{}, etag ETag) error {
	keyStr, err := encodeKey(key)
	if err != nil {
		return fmt.Errorf("traverse: invalid key: %w", err)
	}

	entityPath, rawQuery := splitEntityPath(entitySet)
	path := fmt.Sprintf("/%s(%s)%s", entityPath, keyStr, rawQuery)
	r := c.http.Delete(path)
	r = r.WithContext(ctx)
	if !etag.IsEmpty() {
		r = r.WithHeader("If-Match", etag.Value)
	}
	resp, execErr := c.http.Execute(r)
	if execErr != nil {
		return fmt.Errorf("traverse: delete failed: %w", execErr)
	}
	if resp.StatusCode == http.StatusPreconditionFailed || resp.StatusCode == http.StatusConflict {
		return ErrConcurrencyConflict
	}
	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("traverse: delete returned status %d", resp.StatusCode)
	}
	return nil
}

// checkUpdateResponse translates update/replace relay.Response status codes into errors.
func checkUpdateResponse(resp *relay.Response) error {
	switch resp.StatusCode {
	case http.StatusPreconditionFailed, http.StatusConflict:
		return ErrConcurrencyConflict
	case http.StatusNoContent, http.StatusOK:
		return nil
	default:
		return fmt.Errorf("traverse: unexpected status %d", resp.StatusCode)
	}
}

// ---- Upsert ------------------------------------------------------------------

// Upsert creates a new entity or replaces an existing one using OData upsert
// semantics (PUT with If-None-Match: * header for create-or-replace).
//
// Upsert behaviour depends on whether the entity already exists:
//   - If the entity does NOT exist: the server creates it (HTTP 201 Created).
//   - If the entity already exists: the server replaces it (HTTP 200/204).
//
// This uses the OData upsert pattern (If-None-Match: *) as described in
// OData 4.01 section 11.4.4. Not all servers support this pattern; SAP
// OData services typically do via the PUT endpoint.
//
//	err := client.Upsert(ctx, "Products", 42, map[string]any{
//	    "ID":    42,
//	    "Name":  "Widget",
//	    "Price": 9.99,
//	})
func (c *Client) Upsert(ctx context.Context, entitySet string, key interface{}, data interface{}) error {
	keyStr, err := encodeKey(key)
	if err != nil {
		return fmt.Errorf("traverse: invalid key: %w", err)
	}

	entityPath, rawQuery := splitEntityPath(entitySet)
	path := fmt.Sprintf("/%s(%s)%s", entityPath, keyStr, rawQuery)
	r := c.http.Put(path)
	r = r.WithJSON(data)
	r = r.WithContext(ctx)
	r = r.WithHeader("If-None-Match", "*")

	resp, execErr := c.http.Execute(r)
	if execErr != nil {
		return fmt.Errorf("traverse: upsert failed: %w", execErr)
	}

	switch resp.StatusCode {
	case http.StatusOK, http.StatusCreated, http.StatusNoContent:
		return nil
	default:
		return fmt.Errorf("traverse: upsert returned status %d", resp.StatusCode)
	}
}
