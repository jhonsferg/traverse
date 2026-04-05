package traverse

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

// refBody is the JSON body for a $ref link operation.
type refBody struct {
	ODataID string `json:"@odata.id"`
}

// LinkTo creates a reference link from the current entity to a target entity.
//
// LinkTo sends a PUT request to EntitySet(key)/NavProperty/$ref with a body
// containing the OData ID of the target entity. This establishes a navigation
// property relationship between two existing entities.
//
// OData: PUT /EntitySet(key)/NavProperty/$ref
// Body:  {"@odata.id": "serviceRoot/TargetEntitySet(targetKey)"}
//
// Parameters:
//   - ctx: Request context
//   - key: Primary key of the source entity
//   - navProperty: Navigation property name on the source entity
//   - targetEntitySet: Entity set name of the target entity
//   - targetKey: Primary key of the target entity
//
// Example:
//
//	err := client.From("Orders").LinkTo(ctx, 1, "Customer", "Customers", "ALFKI")
func (q *QueryBuilder) LinkTo(ctx context.Context, key any, navProperty string, targetEntitySet string, targetKey any) error {
	keyStr, err := encodeKey(key)
	if err != nil {
		return fmt.Errorf("traverse: LinkTo invalid source key: %w", err)
	}

	targetKeyStr, err := encodeKey(targetKey)
	if err != nil {
		return fmt.Errorf("traverse: LinkTo invalid target key: %w", err)
	}

	path := fmt.Sprintf("/%s(%s)/%s/$ref", q.entitySet, keyStr, navProperty)

	// Build the serviceRoot - strip trailing slash for consistent URL assembly.
	serviceRoot := strings.TrimRight(q.client.baseURL, "/")
	body := refBody{
		ODataID: fmt.Sprintf("%s/%s(%s)", serviceRoot, targetEntitySet, targetKeyStr),
	}

	req := q.client.http.Put(path)
	req = req.WithJSON(body)
	req = req.WithContext(ctx)

	resp, execErr := q.client.http.Execute(req)
	if execErr != nil {
		return fmt.Errorf("traverse: LinkTo failed: %w", execErr)
	}

	switch resp.StatusCode {
	case http.StatusOK, http.StatusCreated, http.StatusNoContent:
		return nil
	case http.StatusNotFound:
		return fmt.Errorf("traverse: LinkTo: %w", ErrEntityNotFound)
	case http.StatusConflict:
		return fmt.Errorf("traverse: LinkTo conflict: %w", ErrConcurrencyConflict)
	case http.StatusPreconditionFailed:
		return fmt.Errorf("traverse: LinkTo precondition failed: %w", ErrConcurrencyConflict)
	default:
		return fmt.Errorf("traverse: LinkTo returned status %d", resp.StatusCode)
	}
}

// UnlinkFrom removes a reference link from the current entity.
//
// For a single-valued navigation property (no targetKey), it sends:
//
//	DELETE /EntitySet(key)/NavProperty/$ref
//
// For a collection-valued navigation property (with targetKey), it sends:
//
//	DELETE /EntitySet(key)/NavProperty(targetKey)/$ref
//
// Parameters:
//   - ctx: Request context
//   - key: Primary key of the source entity
//   - navProperty: Navigation property name on the source entity
//   - targetKey: (optional) Primary key of the target entity for collection nav properties
//
// Example (single-valued):
//
//	err := client.From("Orders").UnlinkFrom(ctx, 1, "Customer")
//
// Example (collection-valued):
//
//	err := client.From("Orders").UnlinkFrom(ctx, 1, "Items", 42)
func (q *QueryBuilder) UnlinkFrom(ctx context.Context, key any, navProperty string, targetKey ...any) error {
	keyStr, err := encodeKey(key)
	if err != nil {
		return fmt.Errorf("traverse: UnlinkFrom invalid source key: %w", err)
	}

	var path string
	if len(targetKey) > 0 {
		targetKeyStr, err := encodeKey(targetKey[0])
		if err != nil {
			return fmt.Errorf("traverse: UnlinkFrom invalid target key: %w", err)
		}
		path = fmt.Sprintf("/%s(%s)/%s(%s)/$ref", q.entitySet, keyStr, navProperty, targetKeyStr)
	} else {
		path = fmt.Sprintf("/%s(%s)/%s/$ref", q.entitySet, keyStr, navProperty)
	}

	req := q.client.http.Delete(path)
	req = req.WithContext(ctx)

	resp, execErr := q.client.http.Execute(req)
	if execErr != nil {
		return fmt.Errorf("traverse: UnlinkFrom failed: %w", execErr)
	}

	switch resp.StatusCode {
	case http.StatusOK, http.StatusNoContent:
		return nil
	case http.StatusNotFound:
		return fmt.Errorf("traverse: UnlinkFrom: %w", ErrEntityNotFound)
	case http.StatusConflict:
		return fmt.Errorf("traverse: UnlinkFrom conflict: %w", ErrConcurrencyConflict)
	case http.StatusPreconditionFailed:
		return fmt.Errorf("traverse: UnlinkFrom precondition failed: %w", ErrConcurrencyConflict)
	default:
		return fmt.Errorf("traverse: UnlinkFrom returned status %d", resp.StatusCode)
	}
}
