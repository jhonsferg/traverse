package traverse

import (
	"strconv"
	"strings"
)

// ExpandBuilder configures a nested expand expression with optional sub-query options.
//
// ExpandBuilder is obtained via [QueryBuilder.ExpandNested] and allows specifying
// per-navigation-property $select, $filter, $orderby, $top, and further nested
// $expand options.
type ExpandBuilder struct {
	parent   *QueryBuilder
	property string
	selects  []string
	filter   string
	orderBy  string
	top      *int
	expands  []string
}

// ExpandNested begins a nested expand configuration for the given navigation property.
// Call the builder methods to configure sub-query options, then call Done() to
// return to the parent QueryBuilder.
//
//	query := client.From("Orders").
//	    ExpandNested("Items").
//	        Select("ID", "Qty").
//	        Filter("Qty gt 0").
//	        Top(10).
//	    Done()
func (q *QueryBuilder) ExpandNested(property string) *ExpandBuilder {
	return &ExpandBuilder{parent: q, property: property}
}

// Select limits the properties returned for the expanded entity.
func (b *ExpandBuilder) Select(fields ...string) *ExpandBuilder {
	b.selects = append(b.selects, fields...)
	return b
}

// Filter applies a filter expression to the expanded entities.
func (b *ExpandBuilder) Filter(expr string) *ExpandBuilder {
	b.filter = expr
	return b
}

// OrderBy sets the sort order for expanded entities.
func (b *ExpandBuilder) OrderBy(expr string) *ExpandBuilder {
	b.orderBy = expr
	return b
}

// Top limits the number of expanded entities returned.
func (b *ExpandBuilder) Top(n int) *ExpandBuilder {
	b.top = &n
	return b
}

// Expand adds a further nested expand within this navigation property.
func (b *ExpandBuilder) Expand(property string) *ExpandBuilder {
	b.expands = append(b.expands, property)
	return b
}

// Done finalizes the nested expand and returns the parent QueryBuilder.
func (b *ExpandBuilder) Done() *QueryBuilder {
	expr := b.build()
	b.parent.expandProps = append(b.parent.expandProps, expr)
	b.parent.urlDirty = true
	return b.parent
}

// build constructs the OData $expand sub-expression for this level.
func (b *ExpandBuilder) build() string {
	var opts []string
	if len(b.selects) > 0 {
		opts = append(opts, "$select="+strings.Join(b.selects, ","))
	}
	if b.filter != "" {
		opts = append(opts, "$filter="+b.filter)
	}
	if b.orderBy != "" {
		opts = append(opts, "$orderby="+b.orderBy)
	}
	if b.top != nil {
		opts = append(opts, "$top="+strconv.Itoa(*b.top))
	}
	if len(b.expands) > 0 {
		opts = append(opts, "$expand="+strings.Join(b.expands, ","))
	}
	if len(opts) == 0 {
		return b.property
	}
	return b.property + "(" + strings.Join(opts, ";") + ")"
}
