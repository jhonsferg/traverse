package graphql

import (
	"fmt"
	"strings"
)

// QueryTranslator converts GraphQL queries to OData query parameters.
type QueryTranslator struct {
}

// NewQueryTranslator creates a new query translator.
func NewQueryTranslator() *QueryTranslator {
	return &QueryTranslator{}
}

// FilterToOData converts a GraphQL filter expression to OData $filter format.
// For now, this is a simple pass-through that assumes the filter is already in OData format.
// In a full implementation, this would parse and convert GraphQL filter syntax.
func (qt *QueryTranslator) FilterToOData(filter string) string {
	if filter == "" {
		return ""
	}

	// TODO: Parse and convert GraphQL filter syntax to OData
	// For now, assume it's already in OData format
	return filter
}

// SelectToOData converts GraphQL field selections to OData $select format.
func (qt *QueryTranslator) SelectToOData(selectedFields []string) string {
	if len(selectedFields) == 0 {
		return ""
	}

	return strings.Join(selectedFields, ",")
}

// OrderByToOData converts GraphQL order by to OData $orderby format.
func (qt *QueryTranslator) OrderByToOData(orderBy string) string {
	if orderBy == "" {
		return ""
	}

	// TODO: Parse and convert GraphQL orderBy syntax to OData
	// Format: "field1 asc, field2 desc" -> "field1 asc,field2 desc"
	fields := strings.Split(orderBy, ",")
	for i := range fields {
		fields[i] = strings.TrimSpace(fields[i])
	}

	return strings.Join(fields, ",")
}

// TopToOData converts GraphQL limit to OData $top format.
func (qt *QueryTranslator) TopToOData(top int) int {
	if top <= 0 {
		return 0
	}
	return top
}

// SkipToOData converts GraphQL offset to OData $skip format.
func (qt *QueryTranslator) SkipToOData(skip int) int {
	if skip <= 0 {
		return 0
	}
	return skip
}

// ExpandToOData converts navigation properties to OData $expand format.
// This is called when GraphQL selects nested objects/collections.
func (qt *QueryTranslator) ExpandToOData(navigationProperties []string) string {
	if len(navigationProperties) == 0 {
		return ""
	}

	return strings.Join(navigationProperties, ",")
}

// BuildODataQuery builds a complete OData query from GraphQL arguments.
func (qt *QueryTranslator) BuildODataQuery(args map[string]interface{}) string {
	var parts []string

	// Filter
	if filter, ok := args["filter"].(string); ok && filter != "" {
		parts = append(parts, fmt.Sprintf("$filter=%s", filter))
	}

	// Select
	if select_, ok := args["select"].(string); ok && select_ != "" {
		parts = append(parts, fmt.Sprintf("$select=%s", select_))
	}

	// Expand
	if expand, ok := args["expand"].(string); ok && expand != "" {
		parts = append(parts, fmt.Sprintf("$expand=%s", expand))
	}

	// Top
	if top, ok := args["top"].(int); ok && top > 0 {
		parts = append(parts, fmt.Sprintf("$top=%d", top))
	}

	// Skip
	if skip, ok := args["skip"].(int); ok && skip > 0 {
		parts = append(parts, fmt.Sprintf("$skip=%d", skip))
	}

	// OrderBy
	if orderBy, ok := args["orderBy"].(string); ok && orderBy != "" {
		parts = append(parts, fmt.Sprintf("$orderby=%s", orderBy))
	}

	// Count
	if count, ok := args["count"].(bool); ok && count {
		parts = append(parts, "$count=true")
	}

	return strings.Join(parts, "&")
}
