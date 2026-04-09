package traverse

import (
	"context"
	"fmt"
	"strings"

	"github.com/jhonsferg/relay"
)

// DeepUpdateOptions configures the Prefer header for a deep update request.
//
// DeepUpdateOptions allows control over server behavior during a deep update:
//   - ReturnRepresentation: ask the server to return the updated entity
//   - ContinueOnError: ask the server to continue processing remaining operations on error
type DeepUpdateOptions struct {
	// ReturnRepresentation instructs the server to return the updated entity
	// (adds "return=representation" to the Prefer header).
	ReturnRepresentation bool
	// ContinueOnError instructs the server to continue on partial failures
	// (adds "odata.continue-on-error" to the Prefer header).
	ContinueOnError bool
}

// PreferHeader generates the Prefer header value from the options.
// Returns an empty string if no options are set.
func (o DeepUpdateOptions) PreferHeader() string {
	parts := make([]string, 0, 2)
	if o.ReturnRepresentation {
		parts = append(parts, "return=representation")
	}
	if o.ContinueOnError {
		parts = append(parts, "odata.continue-on-error")
	}
	return strings.Join(parts, "; ")
}

// UpdateDeep performs an OData 4.01 deep update of an entity with related entities.
//
// UpdateDeep PATCHes the entity at the keyed URL, embedding related entities in the
// request body for atomic nested updates. The request sets Content-Type to
// "application/json;odata.metadata=minimal" and Prefer to "return=representation".
//
// Call [QueryBuilder.Key] before UpdateDeep to set the entity key:
//
//	resp, err := client.From("Orders").Key(1).UpdateDeep(ctx, OrderWithItems{
//	    Status: "Confirmed",
//	    Items:  []Item{{ID: 10, Qty: 5}},
//	})
//
// Spec ref: OData 4.01 Part 1 - Section 11.4.3 (Update Related Entities When Updating an Entity).
func (q *QueryBuilder) UpdateDeep(ctx context.Context, entity any) (*relay.Response, error) {
	return q.UpdateDeepWithPrefer(ctx, entity, "return=representation")
}

// UpdateDeepWithPrefer performs a deep update with a custom Prefer header value.
//
// Use this when full control over the Prefer header is required, for example to
// combine "return=representation" with "odata.continue-on-error", or to use a
// custom preference. Pass an empty string to omit the Prefer header entirely.
//
//	opts := traverse.DeepUpdateOptions{
//	    ReturnRepresentation: true,
//	    ContinueOnError:      true,
//	}
//	resp, err := client.From("Orders").Key(1).
//	    UpdateDeepWithPrefer(ctx, patch, opts.PreferHeader())
func (q *QueryBuilder) UpdateDeepWithPrefer(ctx context.Context, entity any, prefer string) (*relay.Response, error) {
	var path string
	if q.keyValue != nil {
		keyStr, err := encodeKey(q.keyValue)
		if err != nil {
			return nil, fmt.Errorf("traverse: deep update key encoding failed: %w", err)
		}
		entityPath, rawQuery := splitEntityPath(q.entitySet)
		path = fmt.Sprintf("/%s(%s)%s", entityPath, keyStr, rawQuery)
	} else {
		path = "/" + q.entitySet
	}

	req := q.client.http.Patch(path)
	req = req.WithJSON(entity)
	req = req.WithHeader("Content-Type", "application/json;odata.metadata=minimal")
	if prefer != "" {
		// Strip CR/LF from the Prefer value to prevent HTTP header injection.
		prefer = strings.NewReplacer("\r", "", "\n", "").Replace(prefer)
		req = req.WithHeader("Prefer", prefer)
	}
	req = req.WithContext(ctx)

	resp, err := q.client.http.Execute(req)
	if err != nil {
		return nil, fmt.Errorf("traverse: deep update failed: %w", err)
	}
	if resp.StatusCode != 200 && resp.StatusCode != 204 {
		return resp, fmt.Errorf("traverse: deep update returned status %d", resp.StatusCode)
	}
	return resp, nil
}
