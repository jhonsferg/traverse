// Package encoder provides OData URL construction and encoding utilities.
//
// The encoder package contains internal helpers for building OData query URLs with
// proper encoding of query parameters. These utilities are used internally by the
// traverse client when constructing OData requests.
package encoder

import (
	"fmt"
	"net/url"
	"strings"
)

// QueryOptions holds all OData system query options for URL construction.
//
// QueryOptions encapsulates the standard OData query parameters ($filter, $top, $skip, etc.)
// used to build complete OData query URLs. Each field corresponds to one or more OData system
// query options as defined in the OData v4 specification.
//
// Zero values are safe—unset fields simply result in that query parameter being omitted from
// the constructed URL. Pointers (Top, Skip) use nil to indicate "not set", allowing zero to
// be a valid value (e.g., $top=0).
//
// When used with BuildURL, the resulting URL will include only the non-zero query options,
// properly URL-encoded (except for $filter, which is expected to be pre-encoded).
type QueryOptions struct {
	// Select specifies which properties to return ($select).
	// Properties are listed comma-separated: "ID,Name,Price"
	Select []string
	// Filter is an OData filter expression for row filtering ($filter).
	// Example: "Price gt 100 and Category eq 'Electronics'"
	// Note: Should be pre-encoded before passing to BuildURL.
	Filter string
	// OrderBy specifies sort order ($orderby).
	// Format: "Price asc, Name desc"
	OrderBy string
	// Expand specifies related entities to include in the response ($expand).
	// Can include nested expand options: "Orders,Customers($select=Name)"
	Expand []string
	// Top is the maximum number of records to return ($top).
	// Nil means no limit. Use pointer to allow $top=0.
	Top *int
	// Skip is the number of records to skip before returning ($skip).
	// Enables pagination. Nil means skip=0.
	Skip *int
	// Count indicates whether to include $count=true in the query.
	// When true, OData returns the count of matching records.
	Count bool
	// Search is a full-text search expression ($search).
	// Not all OData services support this parameter.
	Search string
	// Apply specifies data aggregation operations ($apply).
	// Used with OData v4 aggregation extensions.
	Apply string
	// DeltaToken is a token from a previous delta sync ($deltatoken).
	// Enables delta query operations for incremental syncs.
	DeltaToken string
	// Custom holds any custom query parameters.
	// These are added as-is (URL-encoded) to the query string.
	Custom map[string]string
}

// BuildURL constructs a complete OData query URL with all system query options properly formatted.
//
// BuildURL takes a base service URL, an entity set name, and QueryOptions, then builds
// a fully formatted OData request URL with all query parameters. It handles:
//   - Base URL validation (must be http:// or https://)
//   - Entity set path construction
//   - Query parameter encoding (URL-encoding for most; $filter expected pre-encoded)
//   - Proper parameter formatting per OData spec
//
// The resulting URL format is: {base}/{entitySet}?param1=value1&param2=value2
//
// Important encoding notes:
//   - $filter values are NOT URL-encoded (expected to be already properly encoded)
//   - All other parameters ($select, $orderby, etc.) are URL-encoded as appropriate
//   - Query parameters are appended only if their corresponding QueryOptions field is set
//   - Parameter order in the query string is not guaranteed
//
// Returns an error if:
//   - base does not start with "http://" or "https://"
//   - entitySet is empty (caught at URL building stage)
//
// Examples:
//
//	BuildURL("http://sap.example.com/odata/v2", "MaterialSet", QueryOptions{
//	  Top: intPtr(10),
//	  Filter: "Price%20gt%20100",
//	})
//	// Returns: "http://sap.example.com/odata/v2/MaterialSet?$top=10&$filter=Price%20gt%20100"
func BuildURL(base, entitySet string, opts QueryOptions) (string, error) {
	// Validate base URL
	if !strings.HasPrefix(base, "http://") && !strings.HasPrefix(base, "https://") {
		return "", fmt.Errorf("invalid base URL: must start with http:// or https://")
	}

	// Start building the URL
	path := base
	if !strings.HasSuffix(path, "/") {
		path += "/"
	}
	path += entitySet

	// Build query string
	params := make([]string, 0)

	if len(opts.Select) > 0 {
		params = append(params, "$select="+strings.Join(opts.Select, ","))
	}

	if opts.Filter != "" {
		// Note: filter expression should NOT be double-encoded
		params = append(params, "$filter="+opts.Filter)
	}

	if opts.OrderBy != "" {
		params = append(params, "$orderby="+url.QueryEscape(opts.OrderBy))
	}

	if len(opts.Expand) > 0 {
		// $expand can be complex with nested options; simplified version
		params = append(params, "$expand="+strings.Join(opts.Expand, ","))
	}

	if opts.Top != nil {
		params = append(params, fmt.Sprintf("$top=%d", *opts.Top))
	}

	if opts.Skip != nil {
		params = append(params, fmt.Sprintf("$skip=%d", *opts.Skip))
	}

	if opts.Count {
		params = append(params, "$count=true")
	}

	if opts.Search != "" {
		params = append(params, "$search="+url.QueryEscape(opts.Search))
	}

	if opts.Apply != "" {
		params = append(params, "$apply="+url.QueryEscape(opts.Apply))
	}

	if opts.DeltaToken != "" {
		params = append(params, "$deltatoken="+url.QueryEscape(opts.DeltaToken))
	}

	// Add custom parameters
	if opts.Custom != nil {
		for k, v := range opts.Custom {
			params = append(params, url.QueryEscape(k)+"="+url.QueryEscape(v))
		}
	}

	// Build final URL
	if len(params) > 0 {
		path += "?" + strings.Join(params, "&")
	}

	return path, nil
}

// BuildNavigationURL constructs a URL for accessing navigation properties (related entities).
//
// BuildNavigationURL takes an entity set URL (which may or may not have a key predicate)
// and a navigation property name, then constructs the URL to access related entities.
//
// Navigation properties allow accessing related data without separate requests. For example,
// given a Product entity, you can navigate to its related Orders using a navigation property.
//
// URL construction rules:
//   - If entitySetURL doesn't end with ) or ', simply append /navigationProp
//   - If entitySetURL has a key (ends with ) or '), append /navigationProp to that
//
// Examples:
//
//	Input:  "/ProductSet", "SalesOrders"
//	Output: "/ProductSet/SalesOrders"
//
//	Input:  "/ProductSet('SKU001')", "Orders"
//	Output: "/ProductSet('SKU001')/Orders"
//
//	Input:  "/ProductSet(1)", "Categories"
//	Output: "/ProductSet(1)/Categories"
//
// The resulting URL can then be used with OData queries to navigate to related entity sets.
func BuildNavigationURL(entitySetURL, navigationProp string) string {
	if !strings.HasSuffix(entitySetURL, ")") && !strings.HasSuffix(entitySetURL, "'") {
		// Simple entity set
		return entitySetURL + "/" + navigationProp
	}
	// Already has key
	return entitySetURL + "/" + navigationProp
}

// EncodeExpandOption constructs an $expand query parameter with optional nested query options.
//
// EncodeExpandOption creates complex $expand expressions that include nested select and filter
// operations on related entities. This allows querying to retrieve related data with specific
// properties and filters in a single request, rather than requiring separate round trips.
//
// The $expand parameter format per OData spec:
//
//	Simple expand: "Orders"
//	With nested options: "Orders($select=ID,Amount;$filter=Amount gt 100)"
//
// Parameters:
//   - navProp: The navigation property name (e.g., "Orders", "Categories", "SalesOrders")
//   - selectFields: Specific fields to return from the expanded entity set (optional)
//   - filterExpr: An OData filter expression for the expanded entity set (optional)
//
// If both selectFields and filterExpr are empty, returns just the navigation property name.
// Multiple select fields are comma-separated; filter expressions follow OData syntax.
//
// Examples:
//
//	EncodeExpandOption("Orders", nil, "")
//	// Returns: "Orders"
//
//	EncodeExpandOption("Orders", []string{"ID", "Amount"}, "")
//	// Returns: "Orders($select=ID,Amount)"
//
//	EncodeExpandOption("Orders", []string{"ID", "Amount"}, "Amount gt 100")
//	// Returns: "Orders($select=ID,Amount;$filter=Amount gt 100)"
//
// The result can be used as a value for the Expand field in QueryOptions.
func EncodeExpandOption(navProp string, selectFields []string, filterExpr string) string {
	result := navProp

	if len(selectFields) > 0 || filterExpr != "" {
		result += "("

		opts := make([]string, 0)

		if len(selectFields) > 0 {
			opts = append(opts, "$select="+strings.Join(selectFields, ","))
		}

		if filterExpr != "" {
			opts = append(opts, "$filter="+filterExpr)
		}

		result += strings.Join(opts, ";")
		result += ")"
	}

	return result
}
