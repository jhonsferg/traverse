package traverse

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

// CrossJoinBuilder constructs an OData $crossjoin query against two or more
// entity sets, returning tuples of related entities.
//
// $crossjoin is an OData v4 feature (section 11.2.10) that produces the
// Cartesian product of the specified entity sets, optionally narrowed by a
// $filter expression. The response is an array of objects where each key is
// the entity set name and the value is a single entity instance.
//
// Build a cross-join with [Client.CrossJoin], then call [CrossJoinBuilder.Filter],
// [CrossJoinBuilder.Select], and [CrossJoinBuilder.Expand] to refine the query.
// Execute with [CrossJoinBuilder.Collect] or [CrossJoinBuilder.Stream].
//
// Example:
//
//	pairs, err := client.CrossJoin("Customers", "Orders").
//	    Filter("Customers/ID eq Orders/CustomerID").
//	    Collect(ctx)
type CrossJoinBuilder struct {
	client      *Client
	entitySets  []string
	filterExpr  string
	selectParts []string // namespace-qualified: "EntitySet/Field"
	expandParts []string // navigation properties to expand
	params      map[string]string
}

// CrossJoin creates a CrossJoinBuilder for the given entity sets.
//
// At least two entity set names are required; the method panics if fewer than
// two are provided to surface the configuration error at init time rather than
// silently producing an invalid URL.
//
// Example:
//
//	client.CrossJoin("Products", "Categories").
//	    Filter("Products/CategoryID eq Categories/ID")
func (c *Client) CrossJoin(entitySets ...string) *CrossJoinBuilder {
	if len(entitySets) < 2 {
		panic("traverse: CrossJoin requires at least two entity set names")
	}
	return &CrossJoinBuilder{
		client:     c,
		entitySets: entitySets,
		params:     make(map[string]string),
	}
}

// Filter adds an OData $filter expression to narrow the cross-join result.
//
// The filter can reference properties from any of the joined entity sets using
// the qualified form "EntitySetName/PropertyName".
//
// Example:
//
//	.Filter("Products/CategoryID eq Categories/ID and Products/Price gt 100.0")
func (b *CrossJoinBuilder) Filter(expr string) *CrossJoinBuilder {
	b.filterExpr = expr
	return b
}

// Select limits the properties returned for each entity set in the result.
//
// Each selector must use the qualified form "EntitySetName/PropertyName".
//
// Example:
//
//	.Select("Products/Name", "Products/Price", "Categories/Name")
func (b *CrossJoinBuilder) Select(fields ...string) *CrossJoinBuilder {
	b.selectParts = append(b.selectParts, fields...)
	return b
}

// Expand requests inline expansion of navigation properties.
//
// Each value must use the qualified form "EntitySetName/NavigationProperty".
//
// Example:
//
//	.Expand("Products/Supplier")
func (b *CrossJoinBuilder) Expand(navProps ...string) *CrossJoinBuilder {
	b.expandParts = append(b.expandParts, navProps...)
	return b
}

// Param adds a custom query parameter.
//
// Use this for service-specific OData parameters not covered by the builder
// (e.g., SAP sap-client or service-defined preference hints).
func (b *CrossJoinBuilder) Param(key, value string) *CrossJoinBuilder {
	b.params[key] = value
	return b
}

// buildURL assembles the OData $crossjoin URL path and query string.
func (b *CrossJoinBuilder) buildURL() string {
	buf := bufferPool.Get().(*bytes.Buffer)
	defer func() {
		buf.Reset()
		bufferPool.Put(buf)
	}()

	// Path: $crossjoin(EntitySet1,EntitySet2,...)
	buf.WriteString("$crossjoin(")
	buf.WriteString(strings.Join(b.entitySets, ","))
	buf.WriteByte(')')

	hasParams := false
	addSep := func() {
		if !hasParams {
			buf.WriteByte('?')
			hasParams = true
		} else {
			buf.WriteByte('&')
		}
	}

	if len(b.selectParts) > 0 {
		addSep()
		buf.WriteString("$select=")
		buf.WriteString(strings.Join(b.selectParts, ","))
	}

	if b.filterExpr != "" {
		addSep()
		buf.WriteString("$filter=")
		buf.WriteString(url.QueryEscape(b.filterExpr))
	}

	if len(b.expandParts) > 0 {
		addSep()
		buf.WriteString("$expand=")
		buf.WriteString(strings.Join(b.expandParts, ","))
	}

	for k, v := range b.params {
		addSep()
		buf.WriteString(url.QueryEscape(k))
		buf.WriteByte('=')
		buf.WriteString(url.QueryEscape(v))
	}

	return buf.String()
}

// CrossJoinResult is a single row returned from a $crossjoin query.
// Each key is an entity set name; the value is the raw JSON of one entity.
//
// Use [CrossJoinResult.Decode] to unmarshal a specific entity set's data into a
// typed struct.
type CrossJoinResult map[string]json.RawMessage

// Decode unmarshals the entity from the specified entity set into dest.
//
// Example:
//
//	var product Product
//	err := row.Decode("Products", &product)
func (r CrossJoinResult) Decode(entitySet string, dest any) error {
	raw, ok := r[entitySet]
	if !ok {
		return fmt.Errorf("traverse: entity set %q not present in cross-join result row", entitySet)
	}
	return json.Unmarshal(raw, dest)
}

// crossJoinResponse is the raw OData $crossjoin response envelope.
type crossJoinResponse struct {
	Value    []CrossJoinResult `json:"value"`
	NextLink string            `json:"@odata.nextLink,omitempty"`
	Count    *int64            `json:"@odata.count,omitempty"`
}

// Collect executes the $crossjoin query and returns all result rows.
//
// Collect follows @odata.nextLink pagination automatically, accumulating all
// pages before returning. For very large result sets, prefer iterating over
// pages manually with repeated Collect calls using [CrossJoinBuilder.Param].
//
// Example:
//
//	rows, err := client.CrossJoin("Products", "Categories").
//	    Filter("Products/CategoryID eq Categories/ID").
//	    Collect(ctx)
//	if err != nil {
//	    return err
//	}
//	for _, row := range rows {
//	    var p Product
//	    var c Category
//	    _ = row.Decode("Products", &p)
//	    _ = row.Decode("Categories", &c)
//	}
func (b *CrossJoinBuilder) Collect(ctx context.Context) ([]CrossJoinResult, error) {
	var all []CrossJoinResult
	path := b.buildURL()
	for {
		req := b.client.http.Get(path)
		req = req.WithContext(ctx)

		httpResp, err := b.client.http.Execute(req)
		if err != nil {
			return nil, fmt.Errorf("traverse: $crossjoin: %w", err)
		}
		if httpResp.StatusCode != 200 {
			return nil, fmt.Errorf("traverse: $crossjoin: unexpected status %d", httpResp.StatusCode)
		}

		var page crossJoinResponse
		if err := json.NewDecoder(httpResp.BodyReader()).Decode(&page); err != nil {
			return nil, fmt.Errorf("traverse: $crossjoin: parse response: %w", err)
		}
		all = append(all, page.Value...)
		if page.NextLink == "" {
			break
		}
		// Strip the base URL if nextLink is absolute so the relay client uses
		// just the relative path (relay prepends the base URL itself).
		next := page.NextLink
		if strings.HasPrefix(next, b.client.baseURL) {
			next = strings.TrimPrefix(next, b.client.baseURL)
			next = strings.TrimPrefix(next, "/")
		}
		path = next
	}
	return all, nil
}
