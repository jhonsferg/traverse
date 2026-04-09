package traverse

import (
	"context"
	"fmt"
	"net/http"
)

// DeleteOptions configures a delete operation.
type DeleteOptions struct {
	// CascadeNavigationProperties triggers cascade delete on related entities.
	// Sends header: Prefer: odata.cascade-delete
	CascadeNavigationProperties bool

	// IfMatch sets the ETag condition for the delete.
	IfMatch string

	// ReturnRepresentation requests the deleted entity in the response.
	// Sends header: Prefer: return=representation
	ReturnRepresentation bool
}

// DeleteOption is a functional option for configuring delete behaviour.
type DeleteOption func(*DeleteOptions)

// WithDeleteCascade enables cascade delete on related navigation properties.
// The Prefer: odata.cascade-delete header is sent with the DELETE request.
//
// Example:
//
//	err := client.Delete(ctx, "Orders", 1, traverse.WithDeleteCascade())
func WithDeleteCascade() DeleteOption {
	return func(o *DeleteOptions) {
		o.CascadeNavigationProperties = true
	}
}

// WithDeleteIfMatch adds an If-Match ETag condition to the delete request.
// The server will only delete the entity if its current ETag matches.
// Returns ErrConcurrencyConflict if the ETag does not match (HTTP 412).
//
// Example:
//
//	err := client.Delete(ctx, "Products", 42, traverse.WithDeleteIfMatch(`W/"abc123"`))
func WithDeleteIfMatch(etag string) DeleteOption {
	return func(o *DeleteOptions) {
		o.IfMatch = etag
	}
}

// WithDeleteReturnRepresentation requests that the deleted entity be returned in the response.
// Sends header: Prefer: return=representation
//
// Note: Most OData services return HTTP 204 No Content for deletes; this option
// is only useful with services that support return=representation on deletes.
//
// Example:
//
//	err := client.Delete(ctx, "Orders", 1, traverse.WithDeleteReturnRepresentation())
func WithDeleteReturnRepresentation() DeleteOption {
	return func(o *DeleteOptions) {
		o.ReturnRepresentation = true
	}
}

// applyDeleteOptions applies a list of DeleteOption functions to a new DeleteOptions struct.
func applyDeleteOptions(opts []DeleteOption) DeleteOptions {
	var o DeleteOptions
	for _, fn := range opts {
		fn(&o)
	}
	return o
}

// preferHeader builds the value for the Prefer header from delete options.
// Returns an empty string if no Prefer header values are required.
func preferHeader(o DeleteOptions) string {
	switch {
	case o.CascadeNavigationProperties && o.ReturnRepresentation:
		return "odata.cascade-delete,return=representation"
	case o.CascadeNavigationProperties:
		return "odata.cascade-delete"
	case o.ReturnRepresentation:
		return "return=representation"
	default:
		return ""
	}
}

// DeleteWithOptions deletes an entity, applying the provided DeleteOption values.
//
// DeleteWithOptions extends the base Delete method with support for cascade delete,
// ETag conditions, and return-representation semantics.
//
// Parameters:
//   - ctx: Request context for cancellation and timeout
//   - entitySet: Name of the entity set (e.g. "Orders")
//   - key: Entity key (int or string)
//   - opts: Zero or more DeleteOption values
//
// Example:
//
//	err := client.DeleteWithOptions(ctx, "Orders", 1,
//	    traverse.WithDeleteCascade(),
//	    traverse.WithDeleteIfMatch(`W/"abc"`),
//	)
func (c *Client) DeleteWithOptions(ctx context.Context, entitySet string, key interface{}, opts ...DeleteOption) error {
	keyStr, err := encodeKey(key)
	if err != nil {
		return fmt.Errorf("traverse: invalid key: %w", err)
	}

	o := applyDeleteOptions(opts)

	entityPath, rawQuery := splitEntityPath(entitySet)
	path := fmt.Sprintf("/%s(%s)%s", entityPath, keyStr, rawQuery)
	req := c.http.Delete(path)
	req = req.WithContext(ctx)

	if prefer := preferHeader(o); prefer != "" {
		req = req.WithHeader("Prefer", prefer)
	}
	if o.IfMatch != "" {
		req = req.WithHeader("If-Match", o.IfMatch)
	}

	resp, execErr := c.http.Execute(req)
	if execErr != nil {
		return fmt.Errorf("traverse: delete failed: %w", execErr)
	}

	switch resp.StatusCode {
	case http.StatusOK, http.StatusNoContent:
		c.invalidateEntitySetCache(entitySet)
		return nil
	case http.StatusNotFound:
		return fmt.Errorf("traverse: delete: %w", ErrEntityNotFound)
	case http.StatusConflict:
		return fmt.Errorf("traverse: delete conflict: %w", ErrConcurrencyConflict)
	case http.StatusPreconditionFailed:
		return fmt.Errorf("traverse: delete precondition failed: %w", ErrConcurrencyConflict)
	default:
		return fmt.Errorf("traverse: delete returned status %d", resp.StatusCode)
	}
}

// DeleteLink removes a specific navigation link between two entities.
//
// DeleteLink sends a DELETE request to:
//
//	/{EntitySet}({key})/{navProperty}/$ref?$id={relatedKey}
//
// This removes the relationship between the source entity and the target entity
// without deleting either entity from the data store.
//
// Parameters:
//   - ctx: Request context
//   - navProperty: Name of the navigation property on the source entity
//   - relatedKey: Primary key of the target entity to unlink
//
// Example:
//
//	// Remove item 5 from order 1's Items collection
//	err := client.From("Orders").Key(1).DeleteLink(ctx, "Items", 5)
func (q *QueryBuilder) DeleteLink(ctx context.Context, navProperty string, relatedKey any) error {
	keyStr, err := encodeKey(q.keyValue)
	if err != nil {
		return fmt.Errorf("traverse: DeleteLink invalid source key: %w", err)
	}

	relatedKeyStr, err := encodeKey(relatedKey)
	if err != nil {
		return fmt.Errorf("traverse: DeleteLink invalid related key: %w", err)
	}

	entityPath, rawQuery := splitEntityPath(q.entitySet)
	path := fmt.Sprintf("/%s(%s)/%s/$ref%s", entityPath, keyStr, navProperty, rawQuery)
	idValue := fmt.Sprintf("%s(%s)", entityPath, relatedKeyStr)

	req := q.client.http.Delete(path).
		WithQueryParam("$id", idValue).
		WithContext(ctx)

	resp, execErr := q.client.http.Execute(req)
	if execErr != nil {
		return fmt.Errorf("traverse: DeleteLink failed: %w", execErr)
	}

	switch resp.StatusCode {
	case http.StatusOK, http.StatusNoContent:
		return nil
	case http.StatusNotFound:
		return fmt.Errorf("traverse: DeleteLink: %w", ErrEntityNotFound)
	case http.StatusConflict:
		return fmt.Errorf("traverse: DeleteLink conflict: %w", ErrConcurrencyConflict)
	case http.StatusPreconditionFailed:
		return fmt.Errorf("traverse: DeleteLink precondition failed: %w", ErrConcurrencyConflict)
	default:
		return fmt.Errorf("traverse: DeleteLink returned status %d", resp.StatusCode)
	}
}

// DeleteLinks removes all navigation links for a given navigation property.
//
// DeleteLinks sends a DELETE request to:
//
//	/{EntitySet}({key})/{navProperty}/$ref
//
// This removes all relationships for the navigation property without deleting
// any of the related entities.
//
// Parameters:
//   - ctx: Request context
//   - navProperty: Name of the navigation property whose links should be removed
//
// Example:
//
//	// Remove all items from order 1's Items collection
//	err := client.From("Orders").Key(1).DeleteLinks(ctx, "Items")
func (q *QueryBuilder) DeleteLinks(ctx context.Context, navProperty string) error {
	keyStr, err := encodeKey(q.keyValue)
	if err != nil {
		return fmt.Errorf("traverse: DeleteLinks invalid source key: %w", err)
	}

	entityPath, rawQuery := splitEntityPath(q.entitySet)
	path := fmt.Sprintf("/%s(%s)/%s/$ref%s", entityPath, keyStr, navProperty, rawQuery)

	req := q.client.http.Delete(path)
	req = req.WithContext(ctx)

	resp, execErr := q.client.http.Execute(req)
	if execErr != nil {
		return fmt.Errorf("traverse: DeleteLinks failed: %w", execErr)
	}

	switch resp.StatusCode {
	case http.StatusOK, http.StatusNoContent:
		return nil
	case http.StatusNotFound:
		return fmt.Errorf("traverse: DeleteLinks: %w", ErrEntityNotFound)
	case http.StatusConflict:
		return fmt.Errorf("traverse: DeleteLinks conflict: %w", ErrConcurrencyConflict)
	case http.StatusPreconditionFailed:
		return fmt.Errorf("traverse: DeleteLinks precondition failed: %w", ErrConcurrencyConflict)
	default:
		return fmt.Errorf("traverse: DeleteLinks returned status %d", resp.StatusCode)
	}
}
