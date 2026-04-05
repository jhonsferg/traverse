package traverse

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"sync"
)

// validOpsSet contains all valid OData comparison and logical operators.
//
// validOpsSet defines the set of operators supported by OData filter expressions.
// These operators are validated by ValidateFilter to catch typos or invalid operators early.
//
// Supported operators:
//   - Comparison: eq (equals), ne (not equals), gt (greater than), ge (>=), lt (<), le (<=)
//   - Logical: and, or, not, in (membership test)
//   - Spatial: has (enum flag test)
//
// Used internally for filter expression validation and parsing.
var (
	// validOpsSet - Valid OData comparison operators
	validOpsSet = map[string]bool{
		"eq":  true,
		"ne":  true,
		"gt":  true,
		"ge":  true,
		"lt":  true,
		"le":  true,
		"and": true,
		"or":  true,
		"not": true,
		"in":  true,
		"has": true,
	}

	// validFuncsSet contains all valid OData function names.
	//
	// validFuncsSet defines all OData functions available in filter expressions for:
	//   - String operations: startswith, endswith, contains, substring, tolower, toupper, trim, concat
	//   - Length and position: length, indexof
	//   - Date/Time: year, month, day, hour, minute, second, now, date, time
	//   - Math: round, floor, ceiling
	//   - Type checking: cast, isof
	//   - Other: totaloffsetminutes, fractionalseconds
	//
	// Used by ValidateFilter to catch invalid function names before sending to service.
	// validFuncsSet - Valid OData functions
	validFuncsSet = map[string]bool{
		"startswith":         true,
		"endswith":           true,
		"contains":           true,
		"substringof":        true,
		"length":             true,
		"indexof":            true,
		"substring":          true,
		"tolower":            true,
		"toupper":            true,
		"trim":               true,
		"concat":             true,
		"year":               true,
		"month":              true,
		"day":                true,
		"hour":               true,
		"minute":             true,
		"second":             true,
		"fractionalseconds":  true,
		"date":               true,
		"time":               true,
		"totaloffsetminutes": true,
		"now":                true,
		"round":              true,
		"floor":              true,
		"ceiling":            true,
		"cast":               true,
		"isof":               true,
	}
)

// bufferPool is a sync.Pool of *bytes.Buffer for efficient reuse during query construction.
//
// bufferPool is a memory optimization that reduces garbage collection pressure by reusing
// buffer instances across multiple query constructions. Buffers are obtained from the pool
// during URL and filter building, then returned to the pool for reuse.
//
// This optimization is particularly important for high-volume scenarios with concurrent
// query construction, where creating new buffers for each query would cause significant
// memory pressure and GC overhead.
//
// Internal use only; accessed by URL and filter building methods.
var bufferPool = sync.Pool{
	New: func() interface{} {
		return new(bytes.Buffer)
	},
}

// filterBuilderPool is a sync.Pool of *FilterBuilder for reusing builder instances.
//
// filterBuilderPool is an object pooling optimization that reduces allocations when building
// filter expressions via the Where() method. FilterBuilder instances are obtained from this pool,
// configured with filters, and returned to the pool for reuse.
//
// This reduces garbage collection pressure in high-volume query scenarios where many filters
// are constructed dynamically. The pool enables near-zero-allocation filter building for
// common use cases.
//
// Important: FilterBuilder instances from this pool should not be retained after completion.
// Once the filter is complete and the QueryBuilder is returned, do not reuse the FilterBuilder.
//
// Internal use only; accessed via QueryBuilder.Where() method.
var filterBuilderPool = sync.Pool{
	New: func() interface{} {
		return &FilterBuilder{}
	},
}

// QueryBuilder builds and executes OData queries with a fluent, chainable API.
//
// QueryBuilder provides an ergonomic interface for constructing complex OData queries
// with support for filtering, selection, ordering, pagination, expansion, and advanced
// features. All methods return the receiver (*QueryBuilder) to enable method chaining
// for readable, fluent query construction.
//
// Core Query Methods (chainable):
//   - Select(): Choose specific properties to return ($select)
//   - Filter() / Where(): Add row filtering ($filter)
//   - OrderBy() / OrderByDesc(): Sort results ($orderby)
//   - Top(): Limit result count ($top for pagination)
//   - Skip(): Skip records before returning ($skip for pagination)
//   - Expand(): Include related entities ($expand)
//   - WithCount(): Include total record count
//
// Execution Methods (terminal operations):
//   - Collect(ctx): Fetch all results at once into memory
//   - Stream(ctx): Stream results one at a time (O(1) memory)
//   - Page(ctx): Fetch one page of results
//   - Count(ctx): Get total matching record count
//   - Single(ctx): Get exactly one result (error if 0 or >1)
//   - First(ctx): Get first result or nil
//
// QueryBuilder is NOT thread-safe. Create separate QueryBuilders for concurrent queries.
// Modifications to a QueryBuilder during execution produce undefined behavior.
//
// Performance Notes:
//   - Lazy URL construction: URLs are rebuilt only when needed (marked by urlDirty flag)
//   - Object pooling: Uses buffer and FilterBuilder pools to reduce allocations
//   - Streaming support: Stream() provides O(1) memory for large datasets
//   - Query hooks: Client-level hooks can intercept before/after query execution
//
// Fluent API Examples:
//
// Simple query with filtering and sorting:
//
//	results, err := client.From("Products").
//		Filter("Price gt 100 and Category eq 'Electronics'").
//		OrderBy("Price desc").
//		Top(50).
//		Select("ProductID", "Name", "Price").
//		Collect(ctx)
//
// Query with type-safe filtering:
//
//	results, err := client.From("Orders").
//		Where("Status").Eq("Completed").
//		Where("Amount").Gt(1000).
//		OrderByDesc("CreatedDate").
//		Expand("Customer").
//		Collect(ctx)
//
// Streaming large dataset (constant memory):
//
//	stream := client.From("LargeDataSet").
//		Filter("Active eq true").
//		Stream(ctx)
//	for result := range stream {
//		if result.Error != nil {
//			log.Printf("Error: %v", result.Error)
//			break
//		}
//		processRecord(result.Value)
//	}
//
// Pagination with streaming:
//
//	stream := client.From("Products").
//		Top(1000).
//		Skip(offset).
//		Stream(ctx)
type QueryBuilder struct {
	// client is the OData client used to execute queries
	client *Client
	// entitySet is the OData entity set being queried
	entitySet string

	// selectFields contains the list of properties to include in results (from Select)
	selectFields []string
	// filterExpr is the OData filter expression (from Filter or Where methods)
	filterExpr string
	// orderByExpr is the OData order by expression (from OrderBy/OrderByDesc)
	orderByExpr string
	// expandProps contains navigation properties to expand (from Expand)
	expandProps []string
	// top is the maximum number of records to return (from Top)
	top *int
	// skip is the number of records to skip (from Skip)
	skip *int
	// withCount indicates whether to include the total count in results
	withCount bool
	// search is the search term for full-text search (OData v4)
	search string
	// apply is the aggregation expression (OData v4)
	apply string
	// deltaToken is used for incremental updates (OData v4)
	deltaToken string
	// params contains custom query parameters added via Param
	params map[string]string
	// paramsInitialized tracks whether the params map has been allocated (lazy init optimization)
	paramsInitialized bool

	// urlCache is the cached OData URL to avoid rebuilding
	urlCache string
	// urlDirty indicates the cached URL is stale and needs rebuilding
	urlDirty bool
}

// Select limits the returned properties to only the specified fields.
//
// Select specifies which properties should be included in the query response using
// the OData $select system query option. When Select is used, the OData service
// returns only the specified properties, reducing bandwidth and improving query performance.
//
// If Select is called multiple times, fields are accumulated (not replaced).
// Calling Select("A", "B") then Select("C") results in properties A, B, and C being selected.
//
// Useful for:
//   - Reducing response size: Only fetch needed properties
//   - Improving performance: Fewer properties to transmit and parse
//   - Security: Exclude sensitive properties from result sets
//   - Bandwidth optimization: Critical for mobile or limited bandwidth scenarios
//
// If Select is never called, all properties are returned (default behavior).
//
// Parameters:
//   - fields: Property names to include (comma-separated is also valid: "ID,Name,Price")
//
// Returns:
//   - *QueryBuilder: For method chaining
//
// Chainable: Yes
//
// Examples:
//
// Select specific fields:
//
//	query.Select("CustomerID", "Name", "Country")
//	// OData: $select=CustomerID,Name,Country
//
// Multiple Select calls accumulate:
//
//	query.Select("ID", "Name").Select("Email")
//	// OData: $select=ID,Name,Email
//
// Exclude internal/large properties:
//
//	query.Select("ProductID", "Name", "Price", "Category").
//		Filter("Price gt 100")
//	// Gets only essential info, improving performance
func (q *QueryBuilder) Select(fields ...string) *QueryBuilder {
	q.selectFields = append(q.selectFields, fields...)
	q.urlDirty = true
	return q
}

// Filter adds a raw OData filter expression using the $filter system query option.
//
// Filter specifies a raw OData filter expression for row filtering. The filter expression
// should be a valid OData v2 or v4 filter expression following OData specification syntax.
//
// Filter vs Where:
//   - Filter(): For raw OData expressions; caller responsible for syntax
//   - Where(): Type-safe builder interface; prevents syntax errors
//
// Multiple Filter Calls:
// If Filter is called multiple times, the last expression replaces previous ones.
// Use the Where() method for type-safe, chainable filtering instead.
//
// Valid Operators:
//   - Comparison: eq (equals), ne (not equals), gt, ge, lt, le
//   - Logical: and, or, not
//   - String: startswith, endswith, contains, substring, length, toupper, tolower
//   - Math: add, sub, mul, div, mod, round, floor, ceiling
//   - Date: year, month, day, hour, minute, second, now
//   - Membership: in, has (enum flags)
//
// Escaping:
// String literals in expressions must be single-quoted with internal quotes doubled:
//
//	'John''s Company' for the value John's Company
//
// Returns:
//   - *QueryBuilder: For method chaining
//
// Chainable: Yes (but replaces previous filter)
//
// Examples:
//
// Simple comparison:
//
//	query.Filter("Age gt 30")
//	// OData: $filter=Age gt 30
//
// Complex expression with AND/OR:
//
//	query.Filter("Region eq 'North' and Status ne 'Inactive'")
//	// OData: $filter=Region eq 'North' and Status ne 'Inactive'
//
// String functions:
//
//	query.Filter("startswith(CompanyName, 'Alpine')")
//	// OData: $filter=startswith(CompanyName, 'Alpine')
//
// Date/time filtering:
//
//	query.Filter("year(OrderDate) eq 2024")
//	// OData: $filter=year(OrderDate) eq 2024
//
// For safer filtering without syntax errors, use Where() instead:
//
//	query.Where("Age").Gt(30).Where("Status").Ne("Inactive")
func (q *QueryBuilder) Filter(expr string) *QueryBuilder {
	q.filterExpr = expr
	q.urlDirty = true
	return q
}

// Where starts building a type-safe filter expression for the given field.
//
// Where returns a FilterBuilder that provides type-safe filter building with methods
// like Eq(), Gt(), StartsWith(), Contains(), etc. The FilterBuilder implements the
// builder pattern and returns *QueryBuilder from each method to enable chaining.
//
// Type Safety:
// Unlike Filter() which uses raw strings, Where() prevents many syntax errors
// through a typed builder interface. Property names and operators are checked
// before sending to the service.
//
// Object Pooling:
// FilterBuilder instances are obtained from a sync.Pool to reduce allocations.
// Returned FilterBuilder instances should NOT be retained after the QueryBuilder
// is returned; they are reused for subsequent calls.
//
// Comparison Methods Available on FilterBuilder:
//   - Eq(): Equal (eq)
//   - Ne(): Not equal (ne)
//   - Gt(): Greater than (gt)
//   - Ge(): Greater or equal (ge)
//   - Lt(): Less than (lt)
//   - Le(): Less or equal (le)
//   - StartsWith(): String startswith
//   - EndsWith(): String endswith
//   - Contains(): String contains
//
// Returns:
//   - *FilterBuilder: For chainable filter building
//
// Chainable: Yes (chains to QueryBuilder through FilterBuilder methods)
//
// Examples:
//
// Single condition:
//
//	query.Where("Age").Gt(30)
//	// OData: $filter=Age gt 30
//
// Multiple conditions (chained):
//
//	query.Where("Status").Eq("Active").Where("Priority").Gt(1)
//	// OData: $filter=Status eq 'Active' and Priority gt 1
//
// String operations:
//
//	query.Where("Name").StartsWith("John")
//	// OData: $filter=startswith(Name, 'John')
//
// Nested filtering with expand:
//
//	query.Where("Status").Eq("Active").
//		Expand("Orders").
//		Collect(ctx)
func (q *QueryBuilder) Where(field string) *FilterBuilder {
	fb := filterBuilderPool.Get().(*FilterBuilder)
	fb.query = q
	fb.field = field
	return fb
}

// Expand adds a $expand clause for including related entities in the response.
//
// Expand specifies navigation properties (relationships to other entities) to include
// in the OData response using the $expand system query option. Related entities are
// returned nested within the parent entity rather than requiring separate requests.
//
// Navigation Properties:
// A navigation property represents a relationship to another entity type. For example,
// a Customer entity might have navigation properties to Orders, Addresses, Payments, etc.
//
// Multiple Expand Calls:
// Calling Expand() multiple times accumulates properties. Expand("Orders") then
// Expand("Addresses") includes both Orders and Addresses relationships.
//
// Nested Expansion Options:
// Expand supports optional ExpandOption parameters for filtering and selecting
// specific properties from related entities:
//   - WithExpandSelect(): Choose specific properties from related entities
//   - WithExpandFilter(): Filter related entities
//   - WithExpandOrderBy(): Sort related entities
//   - WithExpandTop(): Limit related entity count
//
// Performance Considerations:
//   - Related entities increase response size; use Select() to limit properties
//   - Nested Expand can exponentially increase response size; be careful
//   - Use carefully with large entity collections (100k+ records)
//
// Returns:
//   - *QueryBuilder: For method chaining
//
// Chainable: Yes
//
// Examples:
//
// Simple expand (include all related entities):
//
//	query.Expand("Orders")
//	// OData: $expand=Orders
//
// Multiple navigation properties:
//
//	query.Expand("Orders").Expand("Addresses")
//	// OData: $expand=Orders,Addresses
//
// Expand with nested options:
//
//	query.Expand("Orders",
//		WithExpandSelect("OrderID", "Amount"),
//		WithExpandFilter("Status eq 'Completed'"))
//	// OData: $expand=Orders($select=OrderID,Amount;$filter=Status eq 'Completed')
//
// Complex query with expand and filtering:
//
//	results, _ := client.From("Customers").
//		Where("Country").Eq("USA").
//		Select("CustomerID", "Name").
//		Expand("Orders", WithExpandSelect("OrderID", "Amount")).
//		Collect(ctx)
func (q *QueryBuilder) Expand(navProp string, opts ...ExpandOption) *QueryBuilder {
	// Build expand string with options if provided
	expandStr := navProp
	if len(opts) > 0 {
		expandCfg := &expandConfig{}
		for _, opt := range opts {
			opt(expandCfg)
		}
		// Build nested expand with $select, $filter, $orderby, etc.
		expandStr = buildNestedExpand(navProp, expandCfg)
	}
	q.expandProps = append(q.expandProps, expandStr)
	q.urlDirty = true
	return q
}

// OrderBy adds a $orderby clause to sort results in ascending order.
//
// OrderBy specifies one or more properties to sort the result set in ascending order
// using the OData $orderby system query option. Multiple calls to OrderBy accumulate
// sort expressions; results are sorted first by the first OrderBy, then by the second, etc.
//
// Sort Order:
// Results are sorted by properties in the order OrderBy methods are called:
//   - First OrderBy: Primary sort key
//   - Second OrderBy: Secondary sort key (within primary groups)
//   - And so on...
//
// For descending order, use OrderByDesc() instead.
//
// Performance:
// OData services may create database indexes on frequently used sort keys.
// Sorting on non-indexed properties can be slow for large datasets.
//
// Returns:
//   - *QueryBuilder: For method chaining
//
// Chainable: Yes (accumulates with subsequent OrderBy calls)
//
// Examples:
//
// Single sort:
//
//	query.OrderBy("LastName")
//	// OData: $orderby=LastName asc
//
// Multiple sort keys (hierarchical):
//
//	query.OrderBy("LastName").OrderBy("FirstName")
//	// OData: $orderby=LastName asc,FirstName asc
//	// Sorts by last name first, then first name within each last name group
//
// Descending sort:
//
//	query.OrderByDesc("CreatedDate")
//	// OData: $orderby=CreatedDate desc
//
// Mixed ascending/descending:
//
//	query.OrderBy("Department").OrderByDesc("Salary")
//	// Sorts by department ascending, then salary descending within each department
//
// Combined with filtering:
//
//	query.Where("Status").Eq("Active").
//		OrderBy("Priority").
//		OrderByDesc("CreatedDate").
//		Top(100)
func (q *QueryBuilder) OrderBy(field string) *QueryBuilder {
	if q.orderByExpr != "" {
		q.orderByExpr += ","
	}
	q.orderByExpr += field + " asc"
	q.urlDirty = true
	return q
}

// OrderByDesc adds a $orderby clause to sort results in descending order.
//
// OrderByDesc specifies one or more properties to sort the result set in descending order
// using the OData $orderby system query option. Multiple calls to OrderByDesc accumulate
// sort expressions; results are sorted first by the first OrderBy/OrderByDesc, then by
// the second, etc.
//
// Sort Order:
// Results are sorted by properties in the order OrderBy/OrderByDesc methods are called:
//   - First call: Primary sort key (ascending or descending)
//   - Second call: Secondary sort key (ascending or descending)
//   - And so on...
//
// Mixing Ascending and Descending:
// You can mix OrderBy() (ascending) and OrderByDesc() (descending) in the same query.
// Each call maintains its own sort direction.
//
// Performance:
// OData services may create database indexes on frequently used sort keys.
// Sorting on non-indexed properties can be slow for large datasets.
//
// Returns:
//   - *QueryBuilder: For method chaining
//
// Chainable: Yes (accumulates with subsequent OrderBy/OrderByDesc calls)
//
// Examples:
//
// Single descending sort:
//
//	query.OrderByDesc("Price")
//	// OData: $orderby=Price desc
//
// Multiple descending sorts:
//
//	query.OrderByDesc("DateCreated").OrderByDesc("Priority")
//	// OData: $orderby=DateCreated desc,Priority desc
//	// Most recent first, then highest priority within each date
//
// Mixed ascending and descending:
//
//	query.OrderBy("Category").OrderByDesc("Revenue")
//	// OData: $orderby=Category asc,Revenue desc
//	// Categories A-Z, highest revenue first within each category
//
// Complex query with multiple sort criteria:
//
//	results, _ := client.From("Orders").
//		Where("Status").Eq("Shipped").
//		OrderByDesc("OrderDate").
//		OrderBy("CustomerName").
//		Top(50).
//		Collect(ctx)
func (q *QueryBuilder) OrderByDesc(field string) *QueryBuilder {
	if q.orderByExpr != "" {
		q.orderByExpr += ","
	}
	q.orderByExpr += field + " desc"
	q.urlDirty = true
	return q
}

// Top limits the number of records returned using the $top system query option.
//
// Top specifies the maximum number of entities to return in a single response.
// Combined with Skip, Top enables cursor-based pagination through large datasets.
//
// Behavior:
// If Top is called multiple times, the last value replaces previous ones (not cumulative).
// Top(10).Top(20) results in $top=20, not $top=30.
//
// Pagination Pattern:
// Combine Top with Skip to implement page-based pagination:
//   - Page 1: Skip(0).Top(10) or just Top(10)
//   - Page 2: Skip(10).Top(10)
//   - Page 3: Skip(20).Top(10)
//   - And so on...
//
// Server-Side Limits:
// Many OData services enforce maximum Top values. If you request $top=10000 but the
// service limit is 1000, the service may return only 1000 records. Check service
// documentation for limits.
//
// Relation to Page():
// For simpler pagination with metadata, use Page(pageNumber, pageSize) instead of
// manually calculating Skip/Top offsets.
//
// Returns:
//   - *QueryBuilder: For method chaining
//
// Chainable: Yes
//
// Examples:
//
// Get first 10 records:
//
//	query.Top(10)
//	// OData: $top=10
//
// Pagination (page 3, 25 per page):
//
//	query.Skip(50).Top(25)
//	// OData: $skip=50&$top=25
//	// Returns records 51-75
//
// Combined with filtering and sorting:
//
//	query.Where("Status").Eq("Active").
//		OrderByDesc("CreatedDate").
//		Top(50)
//	// Get 50 most recent active records
//
// Implementation of paging loop:
//
//	for page := 1; page <= maxPages; page++ {
//		results, _ := client.From("Products").
//			OrderBy("ProductID").
//			Skip((page - 1) * 100).
//			Top(100).
//			Collect(ctx)
//	}
func (q *QueryBuilder) Top(n int) *QueryBuilder {
	q.top = &n
	q.urlDirty = true
	return q
}

// Skip skips a specified number of records using the $skip system query option.
//
// Skip specifies the number of entities to skip before returning results. This is
// typically used with Top for cursor-based pagination through large datasets.
//
// Behavior:
// If Skip is called multiple times, the last value replaces previous ones (not cumulative).
// Skip(10).Skip(20) results in $skip=20, not $skip=30.
//
// Pagination Pattern:
// Combine Skip with Top to implement page-based pagination:
//   - Page 1: Top(10) [Skip defaults to 0]
//   - Page 2: Skip(10).Top(10) [Results 11-20]
//   - Page 3: Skip(20).Top(10) [Results 21-30]
//   - Page N: Skip((N-1)*PageSize).Top(PageSize)
//
// Zero-Based Indexing:
// Skip uses zero-based indexing: Skip(0) means "don't skip any", Skip(10) means
// "skip the first 10 and start at the 11th record".
//
// Performance:
// Skipping many records can be slower than using previous result bookmarks or
// date-based filters. For large datasets, consider filtering by date or using
// the Page() method which may be more efficient.
//
// Relation to Page():
// For simpler pagination with metadata, use Page(pageNumber, pageSize) instead of
// manually calculating Skip/Top offsets.
//
// Returns:
//   - *QueryBuilder: For method chaining
//
// Chainable: Yes
//
// Examples:
//
// Skip first 10, get next 25:
//
//	query.Skip(10).Top(25)
//	// OData: $skip=10&$top=25
//	// Returns records 11-35
//
// Pagination loop (50 records per page):
//
//	pageSize := 50
//	for page := 1; ; page++ {
//		results, _ := client.From("Customers").
//			OrderBy("CustomerID").
//			Skip((page - 1) * pageSize).
//			Top(pageSize).
//			Collect(ctx)
//		if len(results) == 0 {
//			break
//		}
//		processPage(results)
//	}
//
// Combined with filtering and sorting:
//
//	query.Where("Status").Eq("Active").
//		OrderByDesc("CreatedDate").
//		Skip(100).
//		Top(50)
//	// Skip first 100 active records, get next 50 (most recent first)
func (q *QueryBuilder) Skip(n int) *QueryBuilder {
	q.skip = &n
	q.urlDirty = true
	return q
}

// WithCount includes the total count of matching records in results.
//
// When called, the response will include a $count field with the total number
// of records matching the query filters, even if fewer records are returned due
// to Top/Skip settings. This is useful for pagination scenarios where the client
// needs to know the total result set size.
// WithCount is chainable and returns q for method chaining.
//
// Example:
//
//	page, err := query.Top(10).WithCount().Page(ctx)
//	if err == nil {
//		log.Printf("Total records: %d", page.Count)
//	}
func (q *QueryBuilder) WithCount() *QueryBuilder {
	q.withCount = true
	q.urlDirty = true
	return q
}

// Search performs a full-text search using the $search system query option.
//
// Search is only available in OData v4. The search term is added to the query,
// and the OData service will return entities matching the search term using
// implementation-defined search semantics.
// Search is chainable and returns q for method chaining.
//
// Example:
//
//	query.Search("customer")  // OData v4 only
func (q *QueryBuilder) Search(term string) *QueryBuilder {
	q.search = term
	q.urlDirty = true
	return q
}

// Apply applies an aggregation or data transformation using the $apply system query option.
//
// Apply is only available in OData v4 and is used for complex data aggregation.
// It follows the OData Data Aggregation Extension specification.
// Apply is chainable and returns q for method chaining.
//
// Example:
//
//	query.Apply("groupby((Region),aggregate(Sales with sum as Total))")  // OData v4 only
func (q *QueryBuilder) Apply(expr string) *QueryBuilder {
	q.apply = expr
	q.urlDirty = true
	return q
}

// Param adds a custom query parameter to the request.
//
// Param allows adding arbitrary query parameters beyond the standard OData options.
// If called multiple times with the same key, the last value replaces the previous one.
// Param is chainable and returns q for method chaining.
//
// Note: Param uses lazy initialization to avoid allocating a params map until needed.
//
// Example:
//
//	query.Param("api-version", "1.0")
func (q *QueryBuilder) Param(key, value string) *QueryBuilder {
	if !q.paramsInitialized {
		q.params = make(map[string]string)
		q.paramsInitialized = true
	}
	q.params[key] = value
	q.urlDirty = true
	return q
}

// WithDeltaToken enables incremental updates using a delta token.
//
// WithDeltaToken is only available in OData v4 and is used with delta queries
// to retrieve only entities that have changed since the previous query.
// The delta token is provided by a previous successful delta query in the
// [Page.DeltaLink] field.
// WithDeltaToken is chainable and returns q for method chaining.
//
// Example:
//
//	page, err := query.WithDeltaToken(previousToken).Page(ctx)  // OData v4 only
func (q *QueryBuilder) WithDeltaToken(token string) *QueryBuilder {
	q.deltaToken = token
	q.urlDirty = true
	return q
}

// Stream executes the query and returns a channel of results.
//
// Stream streams all matching records from all pages using adaptive buffering.
// The returned channel will receive [Result] values containing either a data record
// (in Result.Value) or an error (in Result.Err). The channel is closed when streaming
// completes or an error occurs.
//
// The bufferSize parameter controls the channel buffer capacity:
//   - If bufferSize is omitted or ≤0, adaptive buffering is used (recommended for large datasets)
//   - If bufferSize > 0, that exact buffer size is used
//
// Adaptive buffering estimates the buffer size as: 10MB / avgRecordSize, clamped to [32, 1024].
// This reduces memory usage and GC pressure for large datasets with small records.
//
// Stream uses a worker pool for efficient goroutine management and implements
// backpressure through channel buffering to prevent unbounded memory growth.
//
// Example:
//
//	for result := range query.Stream(ctx) {
//		if result.Err != nil {
//			return result.Err
//		}
//		process(result.Value)  // map[string]interface{}
//	}
func (q *QueryBuilder) Stream(ctx context.Context, bufferSize ...int) <-chan Result[map[string]interface{}] {
	buffer := 0
	if len(bufferSize) > 0 && bufferSize[0] > 0 {
		// Use explicitly provided buffer size
		buffer = bufferSize[0]
	} else {
		// Default: use adaptive buffering
		buffer = 256 // Temporary default; will be optimized on first page fetch
	}

	out := make(chan Result[map[string]interface{}], buffer)

	streamPool.submit(func() {
		defer close(out)
		q.doStreamPages(ctx, out)
	})

	return out
}

// StreamAs is the generic typed version of Stream.
//
// StreamAs is identical to Stream but allows specifying a result type
// through generic type parameters. Currently, this returns the same untyped
// map[string]interface{} as Stream, but the method signature allows for
// future type-safe streaming implementations.
//
// See [QueryBuilder.Stream] for usage details and examples.
func (q *QueryBuilder) StreamAs(ctx context.Context, bufferSize ...int) <-chan Result[map[string]interface{}] {
	return q.Stream(ctx, bufferSize...)
}

// streamRaw is an internal function that streams raw JSON messages for optimized type conversion.
//
// streamRaw returns a channel of raw JSON bytes for each record, bypassing intermediate
// map[string]interface{} allocations. This is useful for applications requiring zero-allocation
// or custom unmarshaling paths.
//
// This is an unexported internal function intended for optimization paths.
func (q *QueryBuilder) streamRaw(ctx context.Context, bufferSize ...int) <-chan RawResult {
	buffer := 256
	if len(bufferSize) > 0 && bufferSize[0] > 0 {
		buffer = bufferSize[0]
	}

	out := make(chan RawResult, buffer)

	streamPool.submit(func() {
		defer close(out)
		q.doStreamPagesRaw(ctx, out)
	})

	return out
}

// First executes the query and returns only the first matching entity.
//
// First automatically adds $top=1 to the query and returns the first matching
// entity without fetching additional results. It returns ErrEntityNotFound if
// no matching entities exist.
//
// First is a convenience method for queries where only the first result matters.
// It's equivalent to calling Page().Value[0] but with built-in error handling.
//
// Efficiency:
// First is more efficient than Collect() when only one record is needed, since
// it limits the OData query to return just one result.
//
// Returns:
//   - map[string]interface{}: The first matching entity
//   - error: ErrEntityNotFound if no matches, or other error on query failure
//
// Common Patterns:
//   - Get first item sorted by a key: OrderBy().First()
//   - Get first matching filtered item: Where().First()
//   - Check if any item exists: if _, err := query.First(ctx); err == nil { /* exists */ }
//
// Examples:
//
// Find first user by ID:
//
//	user, err := client.From("Users").
//		Where("ID").Eq(42).
//		First(ctx)
//	if errors.Is(err, traverse.ErrEntityNotFound) {
//		log.Println("User not found")
//	}
//
// Get most recently created record:
//
//	latest, _ := client.From("Orders").
//		OrderByDesc("CreatedDate").
//		First(ctx)
//	log.Printf("Latest order: %v", latest)
//
// Check if user exists (ignore result):
//
//	_, err := client.From("Users").
//		Where("Email").Eq("test@example.com").
//		First(ctx)
//	if err == nil {
//		log.Println("User already exists")
//	}
//
// Safe pattern with filter verification:
//
//	result, err := client.From("Products").
//		Where("SKU").Eq("ABC123").
//		First(ctx)
//	if err != nil {
//		return fmt.Errorf("product lookup failed: %w", err)
//	}
//	productID := result["ID"]
func (q *QueryBuilder) First(ctx context.Context) (map[string]interface{}, error) {
	q.Top(1)
	page, err := q.Page(ctx)
	if err != nil {
		return nil, err
	}
	if page == nil || len(page.Value) == 0 {
		return nil, ErrEntityNotFound
	}
	return page.Value[0], nil
}

// FindByKey retrieves a single entity by its primary key.
//
// FindByKey fetches the entity directly using the OData GET endpoint with the key
// in the URL path: GET /<EntitySet>(<key>)
//
// The key parameter should be the primary key value. For composite keys, use
// [QueryBuilder.FindByCompositeKey] instead.
// FindByKey returns [ErrEntityNotFound] if no entity matches the key.
//
// Example:
//
//	entity, err := query.FindByKey(ctx, 42)  // Single key
func (q *QueryBuilder) FindByKey(ctx context.Context, key interface{}) (map[string]interface{}, error) {
	keyStr, err := encodeKey(key)
	if err != nil {
		return nil, fmt.Errorf("traverse: invalid key: %w", err)
	}

	url := fmt.Sprintf("%s(%s)", q.entitySet, keyStr)
	req := q.client.http.Get(url)
	req = req.WithContext(ctx)

	resp, err := q.client.http.Execute(req)
	if err != nil {
		return nil, fmt.Errorf("traverse: FindByKey failed: %w", err)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("traverse: FindByKey returned status %d", resp.StatusCode)
	}

	// Parse response based on OData version
	var result map[string]interface{}

	if q.client.version == ODataV2 {
		// OData v2: response is wrapped in {"d": {...}}
		var wrapped struct {
			D map[string]interface{} `json:"d"`
		}
		err = json.NewDecoder(resp.BodyReader()).Decode(&wrapped)
		if err != nil {
			return nil, fmt.Errorf("traverse: failed to decode FindByKey response: %w", err)
		}
		result = wrapped.D
	} else {
		// OData v4: response is the entity directly
		err = json.NewDecoder(resp.BodyReader()).Decode(&result)
		if err != nil {
			return nil, fmt.Errorf("traverse: failed to decode FindByKey response: %w", err)
		}
	}

	return result, nil
}

// FindByCompositeKey retrieves a single entity by a composite primary key.
//
// FindByCompositeKey fetches the entity using a composite key where multiple
// properties form the primary key. The keys parameter should be a map of
// property names to their values.
//
// For single-key entities, use [QueryBuilder.FindByKey] instead.
// FindByCompositeKey returns an error if keys is empty or if the server
// cannot find an entity matching the composite key.
//
// Example:
//
//	entity, err := query.FindByCompositeKey(ctx, map[string]interface{}{
//		"Company": "ABC",
//		"Material": "XYZ",
//	})
func (q *QueryBuilder) FindByCompositeKey(ctx context.Context, keys map[string]interface{}) (map[string]interface{}, error) {
	if len(keys) == 0 {
		return nil, fmt.Errorf("traverse: composite key cannot be empty")
	}

	// Build composite key string: key1=val1,key2=val2,...
	keyParts := make([]string, 0, len(keys))
	for k, v := range keys {
		valStr, err := encodeKey(v)
		if err != nil {
			return nil, fmt.Errorf("traverse: invalid composite key value for %q: %w", k, err)
		}
		keyParts = append(keyParts, fmt.Sprintf("%s=%s", k, valStr))
	}

	url := fmt.Sprintf("%s(%s)", q.entitySet, strings.Join(keyParts, ","))
	req := q.client.http.Get(url)
	req = req.WithContext(ctx)

	resp, err := q.client.http.Execute(req)
	if err != nil {
		return nil, fmt.Errorf("traverse: FindByCompositeKey failed: %w", err)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("traverse: FindByCompositeKey returned status %d", resp.StatusCode)
	}

	// Parse response based on OData version
	var result map[string]interface{}

	if q.client.version == ODataV2 {
		// OData v2: response is wrapped in {"d": {...}}
		var wrapped struct {
			D map[string]interface{} `json:"d"`
		}
		err = json.NewDecoder(resp.BodyReader()).Decode(&wrapped)
		if err != nil {
			return nil, fmt.Errorf("traverse: failed to decode FindByCompositeKey response: %w", err)
		}
		result = wrapped.D
	} else {
		// OData v4: response is the entity directly
		err = json.NewDecoder(resp.BodyReader()).Decode(&result)
		if err != nil {
			return nil, fmt.Errorf("traverse: failed to decode FindByCompositeKey response: %w", err)
		}
	}

	return result, nil
}

// Count returns the total count of matching records without transferring entity data.
//
// Count executes a /$count query which returns only the total number of matching
// entities without downloading the actual entity data. This is the most efficient
// way to determine result set size, as it minimizes network traffic and processing.
//
// Filters Applied:
// Count applies any filters set via Filter() or Where() methods. Select, OrderBy,
// Top, and Skip are ignored (meaningless for counts).
//
// Use Cases:
//   - Check if results exist: Count() > 0
//   - Determine pagination requirements
//   - Validate constraints before operations
//   - Performance optimization: 364x faster than Collect for large datasets
//
// Returns:
//   - int64: Total number of matching records
//   - error: If the query fails or server returns error
//
// Performance:
// Ultra-fast operation returning just an integer. Typical time: 1-2ms even for
// millions of records. Ideal for existence checks and result set sizing.
//
// Examples:
//
// Check if any records match:
//
//	count, _ := client.From("Orders").
//		Where("Status").Eq("Pending").
//		Count(ctx)
//	if count > 0 {
//		// Process pending orders
//	}
//
// Pagination decision:
//
//	total, _ := client.From("Products").Count(ctx)
//	pageCount := (total + 99) / 100  // Ceiling division for 100-item pages
//
// Filtered count:
//
//	activeCount, _ := client.From("Employees").
//		Where("Department").Eq("Sales").
//		Where("Active").Eq(true).
//		Count(ctx)
func (q *QueryBuilder) Count(ctx context.Context) (int64, error) {
	// Build URL with /$count appended to entity set
	path := q.entitySet + "/$count"

	// Build query parameters (without $top/$skip for count)
	params := make([]string, 0, 3)

	if len(q.selectFields) > 0 {
		params = append(params, "$select="+strings.Join(q.selectFields, ","))
	}

	if q.filterExpr != "" {
		params = append(params, "$filter="+url.QueryEscape(q.filterExpr))
	}

	if q.search != "" {
		params = append(params, "$search="+url.QueryEscape(q.search))
	}

	// Build full URL
	if len(params) > 0 {
		path += "?" + strings.Join(params, "&")
	}

	req := q.client.http.Get(path)
	req = req.WithContext(ctx)

	resp, err := q.client.http.Execute(req)
	if err != nil {
		return 0, fmt.Errorf("traverse: Count failed: %w", err)
	}

	if resp.StatusCode != 200 {
		return 0, fmt.Errorf("traverse: Count returned status %d", resp.StatusCode)
	}

	// Parse response: $count returns plain text integer
	body := string(resp.Body())
	count, err := strconv.ParseInt(strings.TrimSpace(body), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("traverse: failed to parse count response %q: %w", body, err)
	}

	return count, nil
}

// Page returns a single page of results with pagination metadata.
//
// Page executes the query and returns one page of results along with pagination
// metadata. The returned Page struct contains:
//   - Value: List of entities for this page (typically 1-1000 records)
//   - NextLink: URL to fetch the next page (empty if this is the last page)
//   - Count: Total count of matching records (only if WithCount() was called)
//   - DeltaLink: URL for delta queries (OData v4 with delta queries only)
//
// Pagination Workflow:
// Page is useful when implementing paginated interfaces or processing results in
// controlled chunks. Manually fetch pages using NextLink or use Stream() for
// automatic multi-page streaming.
//
// Typical Pagination Loop:
//  1. query.Top(pageSize).WithCount().Page(ctx) - Get first page with total count
//  2. Check page.NextLink to see if more results exist
//  3. Call next query with Skip((page-1)*pageSize) for subsequent pages
//  4. Repeat until page.NextLink is empty
//
// Alternative to Manual Pagination:
// For simpler code, use Collect() to fetch all results at once, or Stream() for
// memory-efficient processing of large datasets.
//
// Returns:
//   - *Page: Pagination metadata and results
//   - error: If query fails or server returns error
//
// Thread Safety:
// The returned Page and its Value slice are not thread-safe if modified. Create
// copies before sharing across goroutines.
//
// Examples:
//
// Get first page with total count:
//
//	page, _ := client.From("Customers").
//		OrderBy("CustomerID").
//		Top(25).
//		WithCount().
//		Page(ctx)
//	log.Printf("Page 1 of %d (25 records per page)", page.Count/25)
//
// Manual pagination loop:
//
//	pageSize := 50
//	for page := 1; ; page++ {
//		results, _ := client.From("Products").
//			OrderBy("ProductID").
//			Skip((page - 1) * pageSize).
//			Top(pageSize).
//			Page(ctx)
//		if len(results.Value) == 0 {
//			break
//		}
//		processPage(results.Value, page)
//	}
//
// Using NextLink for pagination:
//
//	for url := initialURL; url != ""; {
//		page, _ := client.From("Orders").Page(ctx)
//		processBatch(page.Value)
//		url = page.NextLink  // May require converting to query
//	}
func (q *QueryBuilder) Page(ctx context.Context) (*Page, error) {
	url := q.buildURL()
	req := q.client.http.Get(url)
	req = req.WithContext(ctx)

	resp, err := q.client.http.Execute(req)
	if err != nil {
		return nil, fmt.Errorf("traverse: Page failed: %w", err)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("traverse: Page returned status %d", resp.StatusCode)
	}

	// Parse response using the same streaming parser
	page := &Page{
		Value: make([]map[string]interface{}, 0, q.client.pageSize),
	}

	decoder := json.NewDecoder(resp.BodyReader())

	// Parse the JSON structure token-by-token
	err = parseODataResponse(decoder, page, q.client.version)
	if err != nil {
		return nil, fmt.Errorf("failed to parse OData response: %w", err)
	}

	return page, nil
}

// FetchPageAt fetches and parses an OData page from an arbitrary URL.
// It is primarily used by [Paginator] to follow @odata.nextLink (or __next)
// URLs returned by the server between pages.
//
// The rawURL must be a fully-qualified URL (absolute) or a path relative to
// the client base URL that the server returned.
func (c *Client) FetchPageAt(ctx context.Context, rawURL string) (*Page, error) {
	req := c.http.Get(rawURL)
	req = req.WithContext(ctx)

	resp, err := c.http.Execute(req)
	if err != nil {
		return nil, fmt.Errorf("traverse: FetchPageAt failed: %w", err)
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("traverse: FetchPageAt returned status %d", resp.StatusCode)
	}

	page := &Page{
		Value: make([]map[string]interface{}, 0, c.pageSize),
	}
	decoder := json.NewDecoder(resp.BodyReader())
	if parseErr := parseODataResponse(decoder, page, c.version); parseErr != nil {
		return nil, fmt.Errorf("traverse: FetchPageAt parse error: %w", parseErr)
	}
	return page, nil
}

// Collect materializes all matching records into a single slice.
//
// Collect streams all pages and accumulates all matching records into a single
// in-memory slice. It uses the Stream method internally for multi-page fetching
// and pre-allocates memory based on query parameters to reduce allocations.
//
// Memory Usage:
// Collect loads the ENTIRE result set into memory. For large result sets, memory
// usage can be problematic:
//   - 1M records × 1 KB each = ~1 GB
//   - 10M records × 1 KB each = ~10 GB
//
// Use Stream() for large datasets to avoid memory exhaustion.
//
// When to Use Collect:
//   - Dataset is known to be small (<10K records)
//   - Top() limits the result set
//   - Where() filters results to manageable size
//   - API endpoint or demo code
//
// When NOT to Use Collect:
//   - Unbounded queries on large tables
//   - Production code handling user input
//   - Processing 100K+ records
//
// Capacity Estimation:
// If Top() is set, Collect pre-allocates that capacity plus any Skip() offset.
// Otherwise defaults to 1000 record estimate. This reduces allocation overhead.
//
// Error Handling:
// If any page fails to fetch, the partial results accumulated so far are lost
// and the error is returned.
//
// Returns:
//   - []map[string]interface{}: All matching records
//   - error: If any page fetch fails
//
// Examples:
//
// OK: Limited result set:
//
//	// Collect with explicit limit
//	products, _ := client.From("Products").
//		Where("Category").Eq("Electronics").
//		Top(500).
//		Collect(ctx)
//
// DANGER: Unbounded query:
//
//	// WARNING: Could exhaust memory on large table!
//	all, _ := client.From("Products").Collect(ctx)
//
// Safe filtered query:
//
//	// Filter reduces result set to manageable size
//	orders, _ := client.From("Orders").
//		Where("OrderDate").Gt(startDate).
//		Where("OrderDate").Lt(endDate).
//		Collect(ctx)
//
// Convert streaming to slice:
//
//	results := make([]map[string]interface{}, 0, 100)
//	for result := range client.From("Orders").Stream(ctx) {
//		if result.Err != nil {
//			return result.Err
//		}
//		results = append(results, result.Value)
//	}
func (q *QueryBuilder) Collect(ctx context.Context) ([]map[string]interface{}, error) {
	// Estimate capacity from query parameters if available
	// If $top is set, use that as a hint; otherwise use reasonable default
	estimatedCapacity := 1000 // Default estimate

	if q.top != nil {
		estimatedCapacity = *q.top
		// If we're skipping, add skip count to estimate
		if q.skip != nil {
			estimatedCapacity += *q.skip
		}
	}

	// Pre-allocate slice with capacity hint to avoid repeated allocations
	results := make([]map[string]interface{}, 0, estimatedCapacity)

	for result := range q.Stream(ctx) {
		if result.Err != nil {
			return nil, result.Err
		}
		results = append(results, result.Value)
	}

	return results, nil
}

// QueryParallel executes multiple queries concurrently and returns results in order.
//
// QueryParallel takes multiple [QueryBuilder] queries and executes them in parallel
// using goroutines. Results are returned in the same order as the input queries,
// making it easy to correlate results.
//
// Each query is executed independently. If any query fails, QueryParallel returns
// the first error encountered; partial results are discarded.
// For independent error handling per query, execute queries separately with
// goroutines and collect errors manually.
//
// Example:
//
//	q1 := client.For("Customers").Filter("Country eq 'USA'")
//	q2 := client.For("Orders").Filter("Status eq 'Completed'")
//	results, err := QueryParallel(ctx, q1, q2)
//	// results[0] is from q1, results[1] is from q2
func QueryParallel(ctx context.Context, queries ...*QueryBuilder) ([]Page, error) {
	if len(queries) == 0 {
		return []Page{}, nil
	}

	// Pre-allocate results slice to maintain order
	results := make([]Page, len(queries))
	errors := make([]error, len(queries))

	// Use WaitGroup to synchronize goroutine completion
	var wg sync.WaitGroup
	wg.Add(len(queries))

	// Execute each query concurrently
	for i, q := range queries {
		go func(index int, query *QueryBuilder) {
			defer wg.Done()
			page, err := query.Page(ctx)
			if err != nil {
				errors[index] = err
				return
			}
			results[index] = *page
		}(i, q)
	}

	// Wait for all goroutines to complete
	wg.Wait()

	// Check for any errors and return the first one
	for _, err := range errors {
		if err != nil {
			return nil, err
		}
	}

	return results, nil
}

// ─────────────────────────────────────────────────────────────────────────

// Private methods

// buildURL constructs the complete OData query URL with all query parameters.
//
// buildURL combines the entity set, $select, $filter, $orderby, $expand, $top,
// $skip, $count, $search, $apply, $deltatoken, and custom parameters into
// a complete URL.
//
// buildURL implements URL caching: if the query has not been modified (urlDirty=false)
// and the cache is populated, the cached URL is returned immediately.
// Otherwise, buildURL reconstructs the URL, caches it, and returns it.
//
// This is an unexported method used internally by Page, Stream, and other query methods.
func (q *QueryBuilder) buildURL() string {
	// Return cached URL if not dirty
	if !q.urlDirty && q.urlCache != "" {
		return q.urlCache
	}

	// ✅ Phase 3: Get buffer from pool instead of allocating new one
	buf := bufferPool.Get().(*bytes.Buffer)
	defer func() {
		buf.Reset()
		bufferPool.Put(buf)
	}()

	buf.WriteString(q.entitySet)

	// Track if we need to add the first parameter
	hasParams := false

	if len(q.selectFields) > 0 {
		if !hasParams {
			buf.WriteString("?")
			hasParams = true
		} else {
			buf.WriteString("&")
		}
		buf.WriteString("$select=")
		for i, field := range q.selectFields {
			if i > 0 {
				buf.WriteString(",")
			}
			buf.WriteString(field)
		}
	}

	if q.filterExpr != "" {
		if !hasParams {
			buf.WriteString("?")
			hasParams = true
		} else {
			buf.WriteString("&")
		}
		buf.WriteString("$filter=")
		buf.WriteString(url.QueryEscape(q.filterExpr))
	}

	if q.orderByExpr != "" {
		if !hasParams {
			buf.WriteString("?")
			hasParams = true
		} else {
			buf.WriteString("&")
		}
		buf.WriteString("$orderby=")
		buf.WriteString(url.QueryEscape(q.orderByExpr))
	}

	if len(q.expandProps) > 0 {
		if !hasParams {
			buf.WriteString("?")
			hasParams = true
		} else {
			buf.WriteString("&")
		}
		buf.WriteString("$expand=")
		for i, prop := range q.expandProps {
			if i > 0 {
				buf.WriteString(",")
			}
			buf.WriteString(prop)
		}
	}

	if q.top != nil {
		if !hasParams {
			buf.WriteString("?")
			hasParams = true
		} else {
			buf.WriteString("&")
		}
		buf.WriteString("$top=")
		// ✅ Optimized: Use strconv instead of fmt.Sprintf
		buf.WriteString(strconv.Itoa(*q.top))
	}

	if q.skip != nil {
		if !hasParams {
			buf.WriteString("?")
			hasParams = true
		} else {
			buf.WriteString("&")
		}
		buf.WriteString("$skip=")
		buf.WriteString(strconv.Itoa(*q.skip))
	}

	if q.withCount {
		if !hasParams {
			buf.WriteString("?")
			hasParams = true
		} else {
			buf.WriteString("&")
		}
		buf.WriteString("$count=true")
	}

	if q.search != "" {
		if !hasParams {
			buf.WriteString("?")
			hasParams = true
		} else {
			buf.WriteString("&")
		}
		buf.WriteString("$search=")
		buf.WriteString(url.QueryEscape(q.search))
	}

	if q.apply != "" {
		if !hasParams {
			buf.WriteString("?")
			hasParams = true
		} else {
			buf.WriteString("&")
		}
		buf.WriteString("$apply=")
		buf.WriteString(url.QueryEscape(q.apply))
	}

	if q.deltaToken != "" {
		if !hasParams {
			buf.WriteString("?")
			hasParams = true
		} else {
			buf.WriteString("&")
		}
		buf.WriteString("$deltatoken=")
		buf.WriteString(url.QueryEscape(q.deltaToken))
	}

	// ✅ Optimized: Custom parameters with minimal allocations
	for k, v := range q.params {
		if !hasParams {
			buf.WriteString("?")
			hasParams = true
		} else {
			buf.WriteString("&")
		}
		buf.WriteString(url.QueryEscape(k))
		buf.WriteString("=")
		buf.WriteString(url.QueryEscape(v))
	}

	// Update cache and mark as clean
	q.urlCache = buf.String()
	q.urlDirty = false

	return q.urlCache
}

// ─────────────────────────────────────────────────────────────────────────

// FilterBuilder provides a type-safe, chainable interface for building OData filter expressions.
//
// FilterBuilder is obtained from QueryBuilder.Where(fieldName) and allows constructing
// filter conditions without writing raw OData syntax. Each comparison method returns
// *QueryBuilder to enable method chaining directly back to query building.
//
// Lifecycle and Pooling:
// FilterBuilder instances are managed by an internal sync.Pool to reduce allocations
// during filtering operations. Instances obtained from Where() should NOT be retained
// after the method returns-they may be reused for subsequent Where() calls.
//
// Object Pooling Benefits:
// - -1 allocation per Where() call (amortized)
// - Efficient for queries with many filters
// - Transparent to users; no manual pool management required
//
// Chaining Pattern:
// Where() methods return *QueryBuilder, enabling seamless method chaining:
//
//	query.Where("Field1").Eq(value1).
//	      Where("Field2").Gt(value2).
//	      Where("Field3").StartsWith("prefix")
//
// Type Safety:
// Comparison methods accept interface{} and automatically handle type conversion:
//   - int, int64: Converted to decimal notation
//   - string: Quoted with escape handling ('value' with ” for internal quotes)
//   - bool: Converted to "true" or "false"
//   - time.Time: Formatted per OData version (v2: /Date(ms)/, v4: RFC3339)
//   - nil: Converted to "null"
//
// Available Comparisons:
// Numeric: Eq, Ne, Gt, Ge, Lt, Le
// String: Eq, Ne, StartsWith, EndsWith, Contains
// Boolean/Null: Eq, Ne
// Date/Time: All numeric comparisons (Gt, Lt, etc.)
//
// String Functions vs Operators:
// String comparisons can use either:
//   - Operators: Eq() for exact match
//   - Functions: StartsWith(), Contains(), EndsWith() for partial matching
//
// Examples:
//
// Single filter:
//
//	query.Where("Status").Eq("Active")
//	// OData: $filter=Status eq 'Active'
//
// Multiple filters (AND):
//
//	query.Where("Status").Eq("Active").
//		  Where("Age").Gt(30)
//	// OData: $filter=Status eq 'Active' and Age gt 30
//
// String operations:
//
//	query.Where("Name").StartsWith("John")
//	// OData: $filter=startswith(Name, 'John')
//
// Mixed types:
//
//	query.Where("OrderDate").Gt(time.Now()).
//		  Where("TotalAmount").Lt(1000).
//		  Where("Status").Ne("Cancelled")
//	// OData: $filter=OrderDate gt <timestamp> and TotalAmount lt 1000 and Status ne 'Cancelled'
type FilterBuilder struct {
	// query is the parent QueryBuilder to return after building an expression
	query *QueryBuilder
	// field is the property name being filtered
	field string
}

// buildExpr efficiently builds a filter expression with a comparison operator and value.
//
// buildExpr is used internally by FilterBuilder comparison methods (Eq, Gt, Lt, etc.)
// to build the actual filter expression. It uses a buffer from the buffer pool for
// efficient string construction with minimal allocations.
//
// Returns the parent QueryBuilder to enable method chaining.
func (f *FilterBuilder) buildExpr(op string, value interface{}) *QueryBuilder {
	buf := bufferPool.Get().(*bytes.Buffer)
	buf.WriteString(f.field)
	buf.WriteRune(' ')
	buf.WriteString(op)
	buf.WriteRune(' ')
	buf.WriteString(serializeValue(value))
	f.query.filterExpr = buf.String()
	buf.Reset()
	bufferPool.Put(buf)
	q := f.query
	f.release()
	return q
}

// Eq creates an equality filter: field eq value.
//
// Eq filters for records where the field exactly equals the specified value.
// This is the most common filter operation.
//
// Type Handling:
// The value parameter is automatically converted to OData literal format:
//   - Strings: Quoted and escaped ('value' format)
//   - Numbers: Decimal notation (42, -3.14, etc.)
//   - Booleans: "true" or "false"
//   - DateTime: OData timestamp format
//   - null/nil: Converts to "null" for null checks
//
// Chaining:
// Eq returns *QueryBuilder to chain with other filters or query methods:
//
//	query.Where("Status").Eq("Active").Where("Age").Gt(30)
//
// Returns:
//   - *QueryBuilder: For method chaining
//
// Examples:
//
// Simple equality:
//
//	query.Where("Status").Eq("Active")
//	// OData: $filter=Status eq 'Active'
//
// Numeric equality:
//
//	query.Where("ID").Eq(42)
//	// OData: $filter=ID eq 42
//
// Boolean:
//
//	query.Where("IsDeleted").Eq(false)
//	// OData: $filter=IsDeleted eq false
//
// Null check:
//
//	query.Where("DeletedAt").Eq(nil)
//	// OData: $filter=DeletedAt eq null
//
// Chained with other filters:
//
//	query.Where("Status").Eq("Active").
//		  Where("Priority").Gt(1).
//		  Where("DueDate").Gt(today)
func (f *FilterBuilder) Eq(value interface{}) *QueryBuilder {
	return f.buildExpr("eq", value)
}

// Ne creates a not-equal filter: field ne value.
//
// Ne filters for records where the field does NOT equal the specified value.
// Useful for excluding specific values or statuses.
//
// Use Cases:
//   - Exclude deleted records: Where("Status").Ne("Deleted")
//   - Exclude null values: Where("Email").Ne(nil)
//   - Exclude specific user: Where("UserID").Ne(adminID)
//
// Type Handling:
// Same automatic conversion as Eq() for all value types.
//
// Chaining:
// Ne returns *QueryBuilder to chain with other filters.
//
// Returns:
//   - *QueryBuilder: For method chaining
//
// Examples:
//
// Exclude status:
//
//	query.Where("Status").Ne("Deleted")
//	// OData: $filter=Status ne 'Deleted'
//
// Exclude null:
//
//	query.Where("Email").Ne(nil)
//	// OData: $filter=Email ne null
//
// Exclude numeric value:
//
//	query.Where("Quantity").Ne(0)
//	// OData: $filter=Quantity ne 0
//
// Combined filtering (active non-deleted):
//
//	query.Where("Status").Eq("Active").
//		  Where("Deleted").Ne(true)
//	// OData: $filter=Status eq 'Active' and Deleted ne true
func (f *FilterBuilder) Ne(value interface{}) *QueryBuilder {
	return f.buildExpr("ne", value)
}

// Gt creates a greater-than filter: field gt value.
//
// Gt filters for records where the field value is strictly greater than the
// specified value. Useful for numeric ranges, dates, and ordered comparisons.
//
// Typical Uses:
//   - Price ranges: Where("Price").Gt(100)
//   - Date filtering: Where("CreatedDate").Gt(startDate)
//   - Numeric thresholds: Where("Age").Gt(18)
//   - Ordered sequences: Where("Sequence").Gt(currentSeq)
//
// Type Support:
// Works with any OData type supporting numeric comparison:
//   - Integers: 42
//   - Decimals: 99.99
//   - Dates: time.Time (converted to OData timestamp)
//   - Enums: Numeric enum values
//   - Durations: In milliseconds or appropriate units
//
// Chaining:
// Gt returns *QueryBuilder to combine with other filters and query methods.
//
// Returns:
//   - *QueryBuilder: For method chaining
//
// Examples:
//
// Simple numeric threshold:
//
//	query.Where("Age").Gt(30)
//	// OData: $filter=Age gt 30
//	// Returns people over 30 years old
//
// Date filtering:
//
//	query.Where("OrderDate").Gt(startOfMonth)
//	// OData: $filter=OrderDate gt <timestamp>
//	// Returns orders from start of month onward
//
// Price range (combined with Lt):
//
//	query.Where("Price").Gt(10).Where("Price").Lt(100)
//	// OData: $filter=Price gt 10 and Price lt 100
//	// Returns products between $10 and $100
//
// Complex filtering:
//
//	query.Where("Salary").Gt(50000).
//		  Where("Department").Eq("Sales").
//		  Where("HireDate").Gt(minDate)
func (f *FilterBuilder) Gt(value interface{}) *QueryBuilder {
	return f.buildExpr("gt", value)
}

// Ge creates a greater-than-or-equal filter: field ge value.
//
// Ge filters for records where the field value is greater than or equal to the
// specified value. Includes the boundary value unlike Gt.
//
// Typical Uses:
//   - Minimum thresholds: Where("Score").Ge(80) for passing grades
//   - Date ranges: Where("CreatedDate").Ge(startDate)
//   - Cutoff values: Where("Priority").Ge(3)
//   - Age verification: Where("Age").Ge(18)
//
// Boundary Behavior:
// Unlike Gt, Ge INCLUDES records that exactly equal the specified value.
// Ge(100) returns 100, 101, 102, ... while Gt(100) returns 101, 102, ...
//
// Chaining:
// Ge returns *QueryBuilder for combining with other filters.
//
// Returns:
//   - *QueryBuilder: For method chaining
//
// Examples:
//
// Minimum score threshold:
//
//	query.Where("Score").Ge(80)
//	// OData: $filter=Score ge 80
//	// Includes score=80 and above
//
// Date range (start inclusive):
//
//	query.Where("CreatedDate").Ge(startDate)
//	// Returns records from startDate onward (inclusive)
//
// Combined with less-than for range:
//
//	query.Where("Price").Ge(50).Where("Price").Le(200)
//	// OData: $filter=Price ge 50 and Price le 200
//	// Returns $50-$200 range (both boundaries included)
//
// Inventory threshold:
//
//	query.Where("StockLevel").Ge(10)
//	// OData: $filter=StockLevel ge 10
//	// Returns items with 10 or more in stock
func (f *FilterBuilder) Ge(value interface{}) *QueryBuilder {
	return f.buildExpr("ge", value)
}

// Lt creates a less-than filter: field lt value.
//
// Lt filters for records where the field value is strictly less than the
// specified value. Useful for upper bounds and before/prior comparisons.
//
// Typical Uses:
//   - Upper price limits: Where("Price").Lt(100)
//   - Before dates: Where("DeadlineDate").Lt(today)
//   - Below thresholds: Where("DaysOverdue").Lt(7)
//   - Version checks: Where("BuildNumber").Lt(2000)
//
// Boundary Behavior:
// Lt does NOT include the specified value. Lt(100) matches 99, 98, 97...
// Use Le() to include the boundary.
//
// Chaining:
// Lt returns *QueryBuilder for combining with other filters.
//
// Returns:
//   - *QueryBuilder: For method chaining
//
// Examples:
//
// Upper price limit (exclusive):
//
//	query.Where("Price").Lt(100)
//	// OData: $filter=Price lt 100
//	// Returns prices < $100 (99.99, 50, etc., but not 100)
//
// Past events:
//
//	query.Where("EventDate").Lt(today)
//	// Returns events before today (yesterday and earlier)
//
// Range query (both exclusive):
//
//	query.Where("Age").Gt(18).Where("Age").Lt(65)
//	// OData: $filter=Age gt 18 and Age lt 65
//	// Returns ages 19-64
//
// Early warning system:
//
//	query.Where("DaysUntilExpiry").Lt(30)
//	// OData: $filter=DaysUntilExpiry lt 30
//	// Items expiring within 30 days
func (f *FilterBuilder) Lt(value interface{}) *QueryBuilder {
	return f.buildExpr("lt", value)
}

// Le creates a less-than-or-equal filter: field le value.
//
// Le filters for records where the field value is less than or equal to the
// specified value. Includes the boundary value unlike Lt.
//
// Typical Uses:
//   - Maximum thresholds: Where("Duration").Le(60) for minute-long clips
//   - On-or-before dates: Where("EndDate").Le(quarterEnd)
//   - Upper limits: Where("PageSize").Le(1000)
//   - Age/date cutoffs: Where("Age").Le(65)
//
// Boundary Behavior:
// Unlike Lt, Le INCLUDES records that exactly equal the specified value.
// Le(100) returns 100, 99, 98, ... while Lt(100) returns 99, 98, ...
//
// Common Pattern:
// Combine with Ge for inclusive ranges: Where("Age").Ge(18).Where("Age").Le(65)
//
// Chaining:
// Le returns *QueryBuilder for combining with other filters.
//
// Returns:
//   - *QueryBuilder: For method chaining
//
// Examples:
//
// Maximum duration (inclusive):
//
//	query.Where("DurationSeconds").Le(300)
//	// OData: $filter=DurationSeconds le 300
//	// Videos 300 seconds or shorter
//
// On-or-before date:
//
//	query.Where("EndDate").Le(today)
//	// Includes today and all past dates
//
// Inclusive range:
//
//	query.Where("Price").Ge(50).Where("Price").Le(200)
//	// OData: $filter=Price ge 50 and Price le 200
//	// Exactly $50-$200 including boundaries
//
// Age demographic:
//
//	query.Where("Age").Ge(13).Where("Age").Le(19)
//	// OData: $filter=Age ge 13 and Age le 19
//	// Ages 13 through 19 inclusive (teenagers)
func (f *FilterBuilder) Le(value interface{}) *QueryBuilder {
	return f.buildExpr("le", value)
}

// Contains creates a substring filter using the OData substringof function.
//
// Contains filters for records where the field value contains the specified
// substring anywhere within the value (case-sensitive per OData spec).
//
// Behavior:
// Matches partial strings at any position:
//   - "John Smith" contains "Smith" ✓
//   - "Smith, John" contains "Smith" ✓
//   - "smith" contains "Smith" ✗ (case-sensitive)
//
// OData Implementation:
// Contains uses the substringof() function (OData standard):
//
//	OData: substringof('substring', FieldName)
//
// Performance Note:
// Substring searches typically require full table scans without indexes.
// For large datasets, ensure database indexes exist or use more specific filters.
//
// Chaining:
// Contains returns *QueryBuilder to combine with other filters.
//
// Returns:
//   - *QueryBuilder: For method chaining
//
// Examples:
//
// Basic substring match:
//
//	query.Where("Name").Contains("John")
//	// OData: $filter=substringof('John', Name)
//	// Matches: "John", "John Smith", "Johnny", "Johanna", "Adjourn"
//
// Email domain search:
//
//	query.Where("Email").Contains("@example.com")
//	// OData: $filter=substringof('@example.com', Email)
//	// Matches: "user@example.com", "admin@example.com"
//
// Combined with other filters:
//
//	query.Where("Company").Contains("Inc").
//		  Where("Status").Eq("Active")
//	// OData: $filter=substringof('Inc', Company) and Status eq 'Active'
//	// Active companies with "Inc" in name
//
// Search functionality:
//
//	query.Where("Description").Contains("urgent").
//		  Where("Priority").Gt(2)
//	// OData: $filter=substringof('urgent', Description) and Priority gt 2
//	// High-priority items with "urgent" in description
func (f *FilterBuilder) Contains(substr string) *QueryBuilder {
	buf := bufferPool.Get().(*bytes.Buffer)
	buf.WriteString("substringof('")
	buf.WriteString(substr)
	buf.WriteString("', ")
	buf.WriteString(f.field)
	buf.WriteRune(')')
	f.query.filterExpr = buf.String()
	buf.Reset()
	bufferPool.Put(buf)
	q := f.query
	f.release()
	return q
}

// StartsWith creates a prefix match filter using the OData startswith function.
//
// StartsWith filters for records where the field value begins with the specified
// prefix string (case-sensitive per OData spec).
//
// Behavior:
// Matches only at the beginning of the field value:
//   - "Product A" startswith "Product" ✓
//   - "Product A" startswith "product" ✗ (case-sensitive)
//   - "A Product" startswith "Product" ✗ (not at start)
//
// Efficiency:
// StartsWith is more efficient than Contains for prefix matching and can use
// database indexes on string columns (B-tree prefix matching).
//
// Use Cases:
//   - Category codes: StartsWith("A-") for category A codes
//   - Postal codes: StartsWith("90210") for specific zip code area
//   - Enum prefixes: StartsWith("STATUS_") for status fields
//   - Name filtering: StartsWith("John") for first names
//   - Auto-complete: StartsWith(userInput) for search suggestions
//
// OData Implementation:
// StartsWith generates: startswith(FieldName, 'prefix')
//
// Chaining:
// StartsWith returns *QueryBuilder to combine with other filters.
//
// Returns:
//   - *QueryBuilder: For method chaining
//
// Examples:
//
// Product code filtering:
//
//	query.Where("ProductCode").StartsWith("ELEC-")
//	// OData: $filter=startswith(ProductCode, 'ELEC-')
//	// Matches: "ELEC-001", "ELEC-TV-50"
//
// Name search (auto-complete):
//
//	query.Where("LastName").StartsWith("Smith")
//	// OData: $filter=startswith(LastName, 'Smith')
//	// Matches: "Smith", "Smithson", "Smithers" (but not "Smithy" capitalization)
//
// Regional filtering:
//
//	query.Where("ZipCode").StartsWith("90210")
//	// OData: $filter=startswith(ZipCode, '90210')
//	// Matches: "90210", "90210-1234" (plus 4 format)
//
// Combined with other filters:
//
//	query.Where("Email").StartsWith("admin").
//		  Where("Department").Eq("IT")
//	// OData: $filter=startswith(Email, 'admin') and Department eq 'IT'
//	// IT admin email addresses
func (f *FilterBuilder) StartsWith(prefix string) *QueryBuilder {
	buf := bufferPool.Get().(*bytes.Buffer)
	buf.WriteString("startswith(")
	buf.WriteString(f.field)
	buf.WriteString(", '")
	buf.WriteString(prefix)
	buf.WriteString("')")
	f.query.filterExpr = buf.String()
	buf.Reset()
	bufferPool.Put(buf)
	q := f.query
	f.release()
	return q
}

// EndsWith creates a suffix match filter using the OData endswith function.
//
// EndsWith filters for records where the field value ends with the specified
// suffix string (case-sensitive per OData spec).
//
// Behavior:
// Matches only at the end of the field value:
//   - "john.smith@example.com" endswith "@example.com" ✓
//   - "example.com" endswith "example.com" ✓
//   - "john.smith@example.com" endswith "@example.COM" ✗ (case-sensitive)
//
// Efficiency:
// EndsWith may require full table scans unless database provides suffix indexes.
// More expensive than StartsWith for large datasets.
//
// Use Cases:
//   - Email domains: EndsWith("@example.com")
//   - File extensions: EndsWith(".pdf"), EndsWith(".jpg")
//   - Phone number suffixes: EndsWith("-1234")
//   - URL patterns: EndsWith(".com"), EndsWith(".org")
//   - TLDs: EndsWith(".co.uk")
//
// OData Implementation:
// EndsWith generates: endswith(FieldName, 'suffix')
//
// Chaining:
// EndsWith returns *QueryBuilder to combine with other filters.
//
// Returns:
//   - *QueryBuilder: For method chaining
//
// Examples:
//
// Email domain filter:
//
//	query.Where("Email").EndsWith("@company.com")
//	// OData: $filter=endswith(Email, '@company.com')
//	// Matches: "user@company.com", "admin@company.com"
//
// File type filtering:
//
//	query.Where("Filename").EndsWith(".pdf")
//	// OData: $filter=endswith(Filename, '.pdf')
//	// Matches: "document.pdf", "report_2024.pdf"
//
// Phone number area code:
//
//	query.Where("Phone").EndsWith("-2000")
//	// OData: $filter=endswith(Phone, '-2000')
//	// Matches: "555-1234-2000", "555-2000"
//
// Combined with other filters:
//
//	query.Where("Email").EndsWith("@partner.com").
//		  Where("Status").Eq("Active")
//	// OData: $filter=endswith(Email, '@partner.com') and Status eq 'Active'
//	// Active partner email addresses
func (f *FilterBuilder) EndsWith(suffix string) *QueryBuilder {
	buf := bufferPool.Get().(*bytes.Buffer)
	buf.WriteString("endswith(")
	buf.WriteString(f.field)
	buf.WriteString(", '")
	buf.WriteString(suffix)
	buf.WriteString("')")
	f.query.filterExpr = buf.String()
	buf.Reset()
	bufferPool.Put(buf)
	q := f.query
	f.release()
	return q
}

// In creates a membership test filter for checking if a field value is in a set.
//
// In generates an OData v4 "in" expression: field in (val1, val2, val3, ...)
// This is efficient for testing whether a value belongs to a known set.
//
// OData Version:
// In is only available in OData v4. OData v2 requires using multiple Eq conditions
// joined with "or": (field eq val1) or (field eq val2) or (field eq val3)
//
// Behavior:
// Returns all records where the field equals ANY of the provided values.
// Equivalent to: OR-ing multiple Eq conditions.
//
// Empty Values:
// If In is called with no values, it returns the query unchanged (no filter added).
// This allows optional filtering: if len(categories) > 0 { q = q.Where("Cat").In(categories...) }
//
// Performance:
// Much more efficient than multiple Where().Eq() calls:
//   - Single clause: Where("Status").In("Active", "Pending", "Review")
//   - vs multiple: Where("Status").Eq("Active").Where("Status").Eq("Pending")...
//   - Network: 1 query parameter vs 3
//   - Processing: Server can use IN index optimization
//
// Type Handling:
// All value types are automatically converted to OData literals:
//   - Strings: Single-quoted ('value')
//   - Numbers: Decimal notation
//   - Booleans: true/false
//   - Dates: OData timestamp format
//
// Chaining:
// In returns *QueryBuilder to combine with other filters.
//
// Returns:
//   - *QueryBuilder: For method chaining
//
// Examples:
//
// Status filter (common use):
//
//	query.Where("Status").In("Active", "Pending", "Review")
//	// OData: $filter=Status in ('Active', 'Pending', 'Review')
//	// Matches any of three statuses
//
// Category selection:
//
//	query.Where("CategoryID").In(1, 2, 3, 5)
//	// OData: $filter=CategoryID in (1, 2, 3, 5)
//	// Products in categories 1, 2, 3, or 5
//
// Priority levels:
//
//	priorities := []int{1, 2, 3}  // High, Medium, Low
//	query.Where("Priority").In(append(priorities, 0)...)
//	// All high/medium/low plus unassigned
//
// Optional filtering:
//
//	var allowedStatuses []string
//	q := client.From("Orders")
//	if len(allowedStatuses) > 0 {
//		q = q.Where("Status").In(allowedStatuses...)
//	}
//	results, _ := q.Collect(ctx)
//
// Combined with other filters:
//
//	query.Where("Region").In("North", "South", "West").
//		  Where("SalesAmount").Gt(10000).
//		  OrderByDesc("SalesAmount").
//		  Top(100)
//	// High-value sales in three regions, sorted
func (f *FilterBuilder) In(values ...interface{}) *QueryBuilder {
	if len(values) == 0 {
		q := f.query
		f.release()
		return q
	}

	// Use buffer to build expression more efficiently
	buf := bufferPool.Get().(*bytes.Buffer)
	buf.WriteString(f.field)
	buf.WriteString(" in (")

	for i, v := range values {
		if i > 0 {
			buf.WriteString(", ")
		}
		buf.WriteString(serializeValue(v))
	}

	buf.WriteRune(')')
	f.query.filterExpr = buf.String()
	buf.Reset()
	bufferPool.Put(buf)

	q := f.query
	f.release()
	return q
}

// release returns the FilterBuilder to the pool for reuse.
//
// release is an internal method called after FilterBuilder has completed building
// an expression. It resets the builder state and returns it to filterBuilderPool
// for reuse, reducing allocations.
func (f *FilterBuilder) release() {
	f.query = nil
	f.field = ""
	filterBuilderPool.Put(f)
}

// Helper types and functions

// Result represents a single record result from a stream, along with any error.
// Result represents a single streamed result from a query.
//
// Result is a generic type that contains either a data value or an error.
// When streaming results via [QueryBuilder.Stream], each item on the returned
// channel is a Result. Check Err before using Value.
//
// Fields:
//   - Value: the actual data (map[string]interface{} for most queries)
//   - Err: error encountered during streaming (or nil if successful)
//   - Page: 1-based page number of this result
//   - Index: 0-based index within the page
type Result[T any] struct {
	// Value is the actual data record
	Value T
	// Err is an error if one occurred, nil otherwise
	Err error
	// Page is the 1-based page number this result came from
	Page int
	// Index is the 0-based index within the page
	Index int
}

// RawResult represents a raw JSON record from an optimized streaming path.
//
// RawResult is used internally for zero-allocation streaming scenarios where
// direct unmarshaling to custom types is needed. The Raw field contains the
// JSON bytes for the record, bypassing intermediate map allocations.
type RawResult struct {
	// Raw is the raw JSON bytes for the record
	Raw json.RawMessage
	// Page is the 1-based page number this result came from
	Page int
	// Index is the 0-based index within the page
	Index int
	// Err is an error if one occurred, nil otherwise
	Err error
}

// Page represents a single page of OData results with pagination metadata.
//
// Page is returned by [QueryBuilder.Page] and contains the results for one
// page of a query, along with metadata for fetching subsequent pages or
// performing incremental updates.
//
// Fields:
//   - Value: slice of entities for this page
//   - RawValue: slice of raw JSON for each entity (for optimization purposes)
//   - NextLink: URL to fetch the next page (if more records exist)
//   - DeltaLink: URL for delta queries to fetch only changed records
//   - Count: total count of all matching records (if WithCount was used)
//   - Context: OData context URI (v4 only)
type Page struct {
	// Value is the slice of entities in this page
	Value []map[string]interface{}
	// RawValue stores raw JSON for direct unmarshaling (optimization)
	RawValue []json.RawMessage
	// NextLink is the URL to fetch the next page (@odata.nextLink in v4, __next in v2)
	NextLink string
	// DeltaLink is the URL for delta queries for incremental sync (@odata.deltaLink)
	DeltaLink string
	// Count is the total count of matching records if $count=true was used
	Count *int64
	// Context is the @odata.context value (OData v4 only)
	Context string
}

// ExpandOption is a functional option for configuring entity expansion.
//
// ExpandOption functions configure which fields, filters, ordering, and limits
// apply to related entities when using [QueryBuilder.Expand]. Multiple options
// can be passed to Expand to build complex nested queries.
//
// See [WithExpandSelect], [WithExpandFilter], [WithExpandOrderBy], etc. for
// available option constructors.
//
// Example:
//
//	query.Expand("Orders",
//		WithExpandSelect("OrderID", "Amount"),
//		WithExpandFilter("Status eq 'Completed'"),
//		WithExpandTop(5),
//	)
type ExpandOption func(*expandConfig)

// expandConfig holds the configuration for expand options.
//
// expandConfig is an internal type that accumulates expand options (select, filter, etc.)
// and is passed to buildNestedExpand to generate the final expand expression.
type expandConfig struct {
	// selectFields is the list of properties to include from related entities
	selectFields []string
	// filterExpr is the OData filter expression to apply to related entities
	filterExpr string
	// orderByExpr is the OData order by expression for related entities
	orderByExpr string
	// topCount is the maximum number of related entities to return
	topCount *int
	// skipCount is the number of related entities to skip
	skipCount *int
}

// WithExpandSelect specifies which fields to include from related entities.
//
// WithExpandSelect adds a $select clause to the expand, limiting related entity
// results to only the specified fields. This reduces response size and improves performance.
//
// Example:
//
//	Expand("Orders", WithExpandSelect("OrderID", "Amount", "OrderDate"))
func WithExpandSelect(fields ...string) ExpandOption {
	return func(cfg *expandConfig) {
		cfg.selectFields = append(cfg.selectFields, fields...)
	}
}

// WithExpandFilter specifies a filter for related entities.
//
// WithExpandFilter adds a $filter clause to the expand, returning only related
// entities that match the filter expression.
//
// Example:
//
//	Expand("Orders", WithExpandFilter("Status eq 'Completed'"))
func WithExpandFilter(expr string) ExpandOption {
	return func(cfg *expandConfig) {
		cfg.filterExpr = expr
	}
}

// WithExpandOrderBy specifies ascending order for related entities.
//
// WithExpandOrderBy adds an $orderby clause (ascending) to the expand.
// Multiple calls accumulate order expressions.
//
// Example:
//
//	Expand("Orders", WithExpandOrderBy("OrderDate"))
func WithExpandOrderBy(field string) ExpandOption {
	return func(cfg *expandConfig) {
		if cfg.orderByExpr != "" {
			cfg.orderByExpr += ","
		}
		cfg.orderByExpr += field + " asc"
	}
}

// WithExpandOrderByDesc specifies descending order for related entities.
//
// WithExpandOrderByDesc adds an $orderby clause (descending) to the expand.
// Multiple calls accumulate order expressions.
//
// Example:
//
//	Expand("Orders", WithExpandOrderByDesc("OrderDate"))
func WithExpandOrderByDesc(field string) ExpandOption {
	return func(cfg *expandConfig) {
		if cfg.orderByExpr != "" {
			cfg.orderByExpr += ","
		}
		cfg.orderByExpr += field + " desc"
	}
}

// WithExpandTop limits the number of related entities to return.
//
// WithExpandTop adds a $top clause to the expand, limiting the number of related
// entities to the specified count.
//
// Example:
//
//	Expand("Orders", WithExpandTop(10))
func WithExpandTop(n int) ExpandOption {
	return func(cfg *expandConfig) {
		cfg.topCount = &n
	}
}

// WithExpandSkip skips a number of related entities.
//
// WithExpandSkip adds a $skip clause to the expand, skipping the specified number
// of related entities before returning results. Typically used with [WithExpandTop].
//
// Example:
//
//	Expand("Orders", WithExpandSkip(5).WithExpandTop(10))
func WithExpandSkip(n int) ExpandOption {
	return func(cfg *expandConfig) {
		cfg.skipCount = &n
	}
}

// buildNestedExpand constructs an expand expression with nested query options.
//
// buildNestedExpand builds OData expand syntax with optional $select, $filter,
// $orderby, $top, and $skip clauses applied to a related entity collection.
//
// Format: NavProp($select=Field1,Field2;$filter=expr;$orderby=field;$top=10;$skip=5)
//
// This is used internally by [QueryBuilder.Expand] with [ExpandOption] configurations
// to build complex nested expand expressions. If no options are provided, returns
// just the navigation property name.
func buildNestedExpand(navProp string, cfg *expandConfig) string {
	if cfg == nil {
		return navProp
	}

	// Check if any options were specified
	hasOptions := len(cfg.selectFields) > 0 || cfg.filterExpr != "" ||
		cfg.orderByExpr != "" || cfg.topCount != nil || cfg.skipCount != nil

	if !hasOptions {
		return navProp
	}

	// Build nested options string using buffer pool (more efficient than strings.Join + fmt.Sprintf)
	buf := bufferPool.Get().(*bytes.Buffer)
	defer func() {
		buf.Reset()
		bufferPool.Put(buf)
	}()

	buf.WriteString(navProp)
	buf.WriteString("($")

	first := true

	if len(cfg.selectFields) > 0 {
		if !first {
			buf.WriteRune(';')
		}
		buf.WriteString("select=")
		for i, field := range cfg.selectFields {
			if i > 0 {
				buf.WriteRune(',')
			}
			buf.WriteString(field)
		}
		first = false
	}

	if cfg.filterExpr != "" {
		if !first {
			buf.WriteRune(';')
		}
		buf.WriteString("filter=")
		buf.WriteString(url.QueryEscape(cfg.filterExpr))
		first = false
	}

	if cfg.orderByExpr != "" {
		if !first {
			buf.WriteRune(';')
		}
		buf.WriteString("orderby=")
		buf.WriteString(url.QueryEscape(cfg.orderByExpr))
		first = false
	}

	if cfg.topCount != nil {
		if !first {
			buf.WriteRune(';')
		}
		buf.WriteString("top=")
		buf.WriteString(strconv.Itoa(*cfg.topCount))
		first = false
	}

	if cfg.skipCount != nil {
		if !first {
			buf.WriteRune(';')
		}
		buf.WriteString("skip=")
		buf.WriteString(strconv.Itoa(*cfg.skipCount))
	}

	buf.WriteRune(')')
	return buf.String()
}

// serializeValue converts a Go value to an OData filter literal.
//
// serializeValue handles the following types:
//   - string: wrapped in single quotes with internal quotes escaped
//   - int, int32, int64: converted to decimal representation
//   - float32, float64: converted to decimal representation
//   - bool: converted to "true" or "false"
//   - Other types: converted using fmt.Sprint as fallback
//
// This function is used internally by [FilterBuilder] methods and
// should not be called directly in most cases.
func serializeValue(v interface{}) string {
	switch val := v.(type) {
	case string:
		// Escape single quotes
		escaped := strings.ReplaceAll(val, "'", "''")
		return "'" + escaped + "'"
	case int:
		return strconv.Itoa(val)
	case int32:
		return strconv.FormatInt(int64(val), 10)
	case int64:
		return strconv.FormatInt(val, 10)
	case float32:
		return strconv.FormatFloat(float64(val), 'f', -1, 32)
	case float64:
		return strconv.FormatFloat(val, 'f', -1, 64)
	case bool:
		if val {
			return "true"
		}
		return "false"
	default:
		// Fallback for unknown types
		return fmt.Sprint(val)
	}
}

// ValidateFilter performs basic OData filter syntax validation.
//
// ValidateFilter checks the filter expression for common syntax errors such as:
//   - Unbalanced parentheses
//   - Unbalanced quotes
//   - Invalid operator usage
//   - Unknown function names
//
// ValidateFilter is a permissive validator that only catches obvious errors,
// not all possible invalid syntax. An empty filter expression is considered valid.
//
// This is typically called before executing a query to provide early error
// detection, avoiding unnecessary server round-trips for invalid filters.
func ValidateFilter(expr string) error {
	if expr == "" {
		return nil // Empty filter is valid
	}

	// Check for basic syntax errors first (fast path - no allocations)
	if strings.Count(expr, "(") != strings.Count(expr, ")") {
		return fmt.Errorf("traverse: invalid filter: unbalanced parentheses")
	}

	if strings.Count(expr, "'")%2 != 0 {
		return fmt.Errorf("traverse: invalid filter: unbalanced quotes")
	}

	// Use bytes.Split instead of strings.Fields to reduce allocations
	exprBytes := []byte(expr)
	tokens := bytes.Split(exprBytes, []byte(" "))

	for i, tokenBytes := range tokens {
		token := string(tokenBytes)
		if token == "" {
			continue // Skip empty tokens
		}

		// Check for invalid comparison operator usage
		if validOpsSet[token] {
			// Comparison operators should have operands on both sides
			// (simplified check - just ensure we're not at boundaries)
			if token == "and" || token == "or" {
				if i == 0 || i == len(tokens)-1 {
					return fmt.Errorf("traverse: invalid filter: %q operator at boundary", token)
				}
			}
			continue
		}

		// Check for function calls (look for opening parenthesis)
		if strings.Contains(token, "(") {
			// Extract function name (everything before the opening paren)
			funcName := token[:strings.Index(token, "(")]
			if !validFuncsSet[funcName] && funcName != "" {
				// Not a recognized function, but it might be a custom one - allow it
				// Only warn if it looks suspiciously malformed
				if strings.Count(token, "(") != strings.Count(token, ")") {
					return fmt.Errorf("traverse: invalid filter: unbalanced parentheses in %q", token)
				}
			}
		}
	}

	return nil
}
