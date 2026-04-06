package traverse

import "strings"

// Compute adds a $compute expression to the OData query.
//
// The $compute system query option (OData v4.01) allows defining computed properties
// that can be referenced in $select, $filter, and $orderby. Multiple calls append
// expressions separated by commas.
//
// Compute is chainable and returns q for method chaining.
//
// Example:
//
//	query.Compute("Price mul Quantity as Total")
//	query.Compute("Price mul Quantity as Total", "Tax div 100 as TaxRate")
func (q *QueryBuilder) Compute(expressions ...string) *QueryBuilder {
	for _, expr := range expressions {
		expr = strings.TrimSpace(expr)
		if expr == "" {
			continue
		}
		if q.compute == "" {
			q.compute = expr
		} else {
			q.compute += "," + expr
		}
	}
	q.urlDirty = true
	return q
}

// ComputeExpr joins the provided parts with spaces to form a $compute expression.
//
// This is a convenience helper for constructing compute expressions from individual
// tokens without manual string concatenation.
//
// Example:
//
//	ComputeExpr("Price", "mul", "Quantity", "as", "Total") // "Price mul Quantity as Total"
func ComputeExpr(parts ...string) string {
	return strings.Join(parts, " ")
}
