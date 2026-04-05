package traverse

import (
	"context"
	"fmt"
	"strings"

	"github.com/jhonsferg/relay"
)

// DeepInsertOptions configures the Prefer header for a deep insert request.
//
// DeepInsertOptions allows control over server behavior during a deep insert:
//   - ReturnRepresentation: ask the server to return the created entity
//   - ContinueOnError: ask the server to continue processing remaining operations on error
type DeepInsertOptions struct {
	// ReturnRepresentation instructs the server to return the created entity
	// (adds "return=representation" to the Prefer header).
	ReturnRepresentation bool
	// ContinueOnError instructs the server to continue on partial failures
	// (adds "odata.continue-on-error" to the Prefer header).
	ContinueOnError bool
}

// PreferHeader generates the Prefer header value from the options.
// Returns an empty string if no options are set.
func (o DeepInsertOptions) PreferHeader() string {
	parts := make([]string, 0, 2)
	if o.ReturnRepresentation {
		parts = append(parts, "return=representation")
	}
	if o.ContinueOnError {
		parts = append(parts, "odata.continue-on-error")
	}
	return strings.Join(parts, "; ")
}

// CreateDeep performs an OData 4.01 deep insert of an entity with related entities.
//
// CreateDeep POSTs the entity to the entity set, setting the required headers for
// deep insert: Content-Type is set to "application/json;odata.metadata=minimal" and
// Prefer is set to "return=representation". The caller constructs nested structs or maps
// with related entities embedded.
//
// Returns the raw relay.Response so the caller can decode nested result structures.
//
// Example:
//
//	type OrderLine struct {
//	    ProductID int    `json:"ProductID"`
//	    Quantity  int    `json:"Quantity"`
//	}
//	type Order struct {
//	    CustomerID string      `json:"CustomerID"`
//	    Lines      []OrderLine `json:"Lines"`
//	}
//	resp, err := client.From("Orders").CreateDeep(ctx, Order{
//	    CustomerID: "CUST1",
//	    Lines: []OrderLine{{ProductID: 1, Quantity: 5}},
//	})
func (q *QueryBuilder) CreateDeep(ctx context.Context, entity any) (*relay.Response, error) {
	return q.CreateDeepWithPrefer(ctx, entity, "return=representation")
}

// CreateDeepWithPrefer performs a deep insert with a custom Prefer header value.
//
// Use this when you need full control over the Prefer header, for example to
// combine "return=representation" with "odata.continue-on-error", or to use
// a custom preference. Pass an empty string to omit the Prefer header.
//
//	opts := traverse.DeepInsertOptions{
//	    ReturnRepresentation: true,
//	    ContinueOnError:      true,
//	}
//	resp, err := client.From("Orders").
//	    CreateDeepWithPrefer(ctx, order, opts.PreferHeader())
func (q *QueryBuilder) CreateDeepWithPrefer(ctx context.Context, entity any, prefer string) (*relay.Response, error) {
	path := "/" + q.entitySet

	req := q.client.http.Post(path)
	req = req.WithJSON(entity)
	req = req.WithHeader("Content-Type", "application/json;odata.metadata=minimal")
	if prefer != "" {
		req = req.WithHeader("Prefer", prefer)
	}
	req = req.WithContext(ctx)

	resp, err := q.client.http.Execute(req)
	if err != nil {
		return nil, fmt.Errorf("traverse: deep insert failed: %w", err)
	}
	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		return resp, fmt.Errorf("traverse: deep insert returned status %d", resp.StatusCode)
	}
	return resp, nil
}
