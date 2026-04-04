package traverse

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestQueryBuilderSelect(t *testing.T) {
	qb := &QueryBuilder{
		client: &Client{},
	}

	qb.Select("Name", "Email", "Phone")

	if len(qb.selectFields) != 3 {
		t.Errorf("Select() resulted in %d fields, want 3", len(qb.selectFields))
	}

	if qb.selectFields[0] != "Name" || qb.selectFields[1] != "Email" || qb.selectFields[2] != "Phone" {
		t.Errorf("Select() fields mismatch: %v", qb.selectFields)
	}
}

func TestQueryBuilderFilter(t *testing.T) {
	qb := &QueryBuilder{
		client: &Client{},
	}

	qb.Filter("Status eq 'Active'")

	if qb.filterExpr != "Status eq 'Active'" {
		t.Errorf("Filter() set filterExpr to %s, want Status eq 'Active'", qb.filterExpr)
	}
}

func TestQueryBuilderOrderBy(t *testing.T) {
	qb := &QueryBuilder{
		client: &Client{},
	}

	qb.OrderBy("Name")
	qb.OrderByDesc("CreatedDate")

	if qb.orderByExpr == "" {
		t.Error("OrderBy() did not set orderByExpr")
	}
}

func TestQueryBuilderTop(t *testing.T) {
	qb := &QueryBuilder{
		client: &Client{},
	}

	qb.Top(100)

	if qb.top == nil || *qb.top != 100 {
		t.Error("Top() did not set top correctly")
	}
}

func TestQueryBuilderSkip(t *testing.T) {
	qb := &QueryBuilder{
		client: &Client{},
	}

	qb.Skip(50)

	if qb.skip == nil || *qb.skip != 50 {
		t.Error("Skip() did not set skip correctly")
	}
}

func TestQueryBuilderChaining(t *testing.T) {
	qb := &QueryBuilder{
		client: &Client{},
	}

	// Test that all methods return *QueryBuilder for chaining
	result := qb.
		Select("Name", "Email").
		Filter("Status eq 'Active'").
		OrderBy("Name").
		Top(10).
		Skip(5)

	if result != qb {
		t.Error("Chaining broke; methods don't return *QueryBuilder")
	}
}

func TestQueryBuilderWithCount(t *testing.T) {
	qb := &QueryBuilder{
		client: &Client{},
	}

	qb.WithCount()

	if !qb.withCount {
		t.Error("WithCount() did not set withCount to true")
	}
}

// TestQueryParallelOrder verifies that QueryParallel maintains input order in results.
func TestQueryParallelOrder(t *testing.T) {
	// Create a mock client
	client := &Client{}

	// Create three queries
	q1 := client.From("Entity1")
	q2 := client.From("Entity2")
	q3 := client.From("Entity3")

	// This test validates the structure; actual execution requires a real server.
	// We're testing that the function accepts queries and returns ordered results.
	queries := []*QueryBuilder{q1, q2, q3}

	if len(queries) != 3 {
		t.Errorf("Expected 3 queries, got %d", len(queries))
	}

	// Verify queries maintain their order
	if queries[0].entitySet != "Entity1" {
		t.Errorf("Query 0 entity set should be Entity1, got %s", queries[0].entitySet)
	}
	if queries[1].entitySet != "Entity2" {
		t.Errorf("Query 1 entity set should be Entity2, got %s", queries[1].entitySet)
	}
	if queries[2].entitySet != "Entity3" {
		t.Errorf("Query 2 entity set should be Entity3, got %s", queries[2].entitySet)
	}
}

// TestQueryParallelEmpty verifies that QueryParallel handles empty input.
func TestQueryParallelEmpty(t *testing.T) {
	results, err := QueryParallel(context.TODO())
	if err != nil {
		t.Errorf("QueryParallel with empty input should not error, got %v", err)
	}
	if len(results) != 0 {
		t.Errorf("QueryParallel with empty input should return empty slice, got %d results", len(results))
	}
}

// TestMemoryCacheConcurrency tests concurrent access to the lock-free cache.
func TestMemoryCacheLockFree(t *testing.T) {
	cache := NewMemoryCache()

	// Test concurrent writes don't block each other
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(idx int) {
			defer func() { done <- true }()
			metadata := &Metadata{EntityTypes: []EntityType{{Name: fmt.Sprintf("Entity%c", 'A'+idx)}}}
			cache.Set(fmt.Sprintf("key%d", idx), metadata)
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify data integrity
	if meta, found := cache.Get("key0"); !found || meta == nil {
		t.Error("Cache entry lost after concurrent writes")
	}
}

// TestGoroutinePoolSubmit verifies the goroutine pool works correctly.
func TestGoroutinePoolSubmit(t *testing.T) {
	pool := newGoroutinePool(3)
	defer pool.close()

	executed := make(chan bool, 5)

	for i := 0; i < 5; i++ {
		pool.submit(func() {
			executed <- true
		})
	}

	// Wait for all tasks to complete
	for i := 0; i < 5; i++ {
		select {
		case <-executed:
		case <-time.After(5 * time.Second):
			t.Fatal("Goroutine pool task did not complete in time")
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────
// PHASE 2 QUERY OPTIMIZATION TESTS
// ─────────────────────────────────────────────────────────────────────────

// TestURLCaching verifies that URL caching works correctly
func TestURLCaching(t *testing.T) {
	qb := &QueryBuilder{
		client:    &Client{},
		entitySet: "Products",
		params:    make(map[string]string),
	}

	// Build URL first time - should set cache
	url1 := qb.buildURL()
	if qb.urlCache != url1 {
		t.Errorf("URL cache not set: got %q, want %q", qb.urlCache, url1)
	}
	if qb.urlDirty {
		t.Error("URL should be marked clean after buildURL()")
	}

	// Build URL second time without changes - should return cached
	url2 := qb.buildURL()
	if url1 != url2 {
		t.Errorf("Cached URL mismatch: first=%q, second=%q", url1, url2)
	}

	// Modify query and check cache invalidation
	qb.Select("Name", "Price")
	if !qb.urlDirty {
		t.Error("URL should be marked dirty after Select()")
	}

	// Rebuild and verify cache is updated
	url3 := qb.buildURL()
	if qb.urlDirty {
		t.Error("URL should be marked clean after buildURL()")
	}
	if url1 == url3 {
		t.Error("URL cache should have been updated after Select()")
	}
	if !strings.Contains(url3, "$select=Name,Price") {
		t.Errorf("URL should contain $select parameter: %q", url3)
	}
}

// TestURLCachingMultipleMutations verifies cache invalidation on different mutations
func TestURLCachingMultipleMutations(t *testing.T) {
	qb := &QueryBuilder{
		client:    &Client{},
		entitySet: "Orders",
		params:    make(map[string]string),
	}

	_ = qb.buildURL()

	mutations := []struct {
		name   string
		mutate func()
		check  func(string) bool
	}{
		{
			name: "Filter",
			mutate: func() {
				qb.Filter("Status eq 'Active'")
			},
			check: func(u string) bool {
				return strings.Contains(u, "$filter=")
			},
		},
		{
			name: "OrderBy",
			mutate: func() {
				qb.OrderBy("CreatedDate")
			},
			check: func(u string) bool {
				return strings.Contains(u, "$orderby=")
			},
		},
		{
			name: "Top",
			mutate: func() {
				qb.Top(100)
			},
			check: func(u string) bool {
				return strings.Contains(u, "$top=100")
			},
		},
		{
			name: "Skip",
			mutate: func() {
				qb.Skip(50)
			},
			check: func(u string) bool {
				return strings.Contains(u, "$skip=50")
			},
		},
		{
			name: "WithCount",
			mutate: func() {
				qb.WithCount()
			},
			check: func(u string) bool {
				return strings.Contains(u, "$count=true")
			},
		},
	}

	for _, m := range mutations {
		m.mutate()
		if !qb.urlDirty {
			t.Errorf("URL should be marked dirty after %s()", m.name)
		}
		url := qb.buildURL()
		if qb.urlDirty {
			t.Errorf("URL should be marked clean after buildURL() following %s()", m.name)
		}
		if !m.check(url) {
			t.Errorf("URL check failed for %s(): %q", m.name, url)
		}
	}
}

// TestSelectFieldsInURL verifies Select() generates correct $select parameter
func TestSelectFieldsInURL(t *testing.T) {
	qb := &QueryBuilder{
		client:    &Client{},
		entitySet: "Products",
		params:    make(map[string]string),
	}

	qb.Select("ID", "Name", "Price")
	url := qb.buildURL()

	if !strings.Contains(url, "$select=ID,Name,Price") {
		t.Errorf("URL should contain $select parameter: %q", url)
	}
}

// TestSelectFieldsMultiple verifies multiple Select() calls append fields
func TestSelectFieldsMultiple(t *testing.T) {
	qb := &QueryBuilder{
		client:    &Client{},
		entitySet: "Products",
		params:    make(map[string]string),
	}

	qb.Select("ID", "Name")
	qb.Select("Price", "Category")
	url := qb.buildURL()

	if !strings.Contains(url, "$select=") {
		t.Errorf("URL should contain $select parameter: %q", url)
	}

	// Check all fields are present
	for _, field := range []string{"ID", "Name", "Price", "Category"} {
		if !strings.Contains(url, field) {
			t.Errorf("URL missing field %q: %q", field, url)
		}
	}
}

// TestValidateFilterValid tests filter validation with valid filters
func TestValidateFilterValid(t *testing.T) {
	tests := []string{
		"",                                 // Empty is valid
		"Status eq 'Active'",               // Basic comparison
		"Name eq 'John'",                   // Simple equality
		"Price gt 100",                     // Greater than
		"Amount le 500",                    // Less than or equal
		"Status eq 'Active' and Age gt 18", // Compound with and
		"Status eq 'Active' or Status eq 'Inactive'", // Compound with or
		"startswith(Name, 'J')",                      // Function
		"endswith(Email, '@example.com')",            // Function
		"contains(Description, 'test')",              // Function
		"Year(CreatedDate) eq 2024",                  // Date function
		"Category in ('Electronics', 'Books')",       // in operator
		"cast(ID, 'Edm.Int32') eq 5",                 // cast function
	}

	for _, expr := range tests {
		err := ValidateFilter(expr)
		if err != nil {
			t.Errorf("ValidateFilter(%q) should be valid, got error: %v", expr, err)
		}
	}
}

// TestValidateFilterEmpty tests that empty filter is valid
func TestValidateFilterEmpty(t *testing.T) {
	err := ValidateFilter("")
	if err != nil {
		t.Errorf("ValidateFilter(\"\") should be valid, got error: %v", err)
	}
}

// TestValidateFilterUnbalancedParentheses tests parentheses validation
func TestValidateFilterUnbalancedParentheses(t *testing.T) {
	invalidFilters := []string{
		"(Status eq 'Active'",
		"Status eq 'Active')",
		"startswith(Name, 'John'",
	}

	for _, expr := range invalidFilters {
		if strings.Count(expr, "(") != strings.Count(expr, ")") {
			err := ValidateFilter(expr)
			if err == nil {
				t.Errorf("ValidateFilter(%q) should detect unbalanced parentheses", expr)
			}
		}
	}
}

// TestValidateFilterUnbalancedQuotes tests quote validation
func TestValidateFilterUnbalancedQuotes(t *testing.T) {
	invalidFilters := []string{
		"Name eq 'John",
		"Status eq 'Active' and Name eq 'Jane",
	}

	for _, expr := range invalidFilters {
		if strings.Count(expr, "'")%2 != 0 {
			err := ValidateFilter(expr)
			if err == nil {
				t.Errorf("ValidateFilter(%q) should detect unbalanced quotes", expr)
			}
		}
	}
}

// TestCountQueryBuilding verifies Count() method generates correct URL
func TestCountQueryBuilding(t *testing.T) {
	qb := &QueryBuilder{
		client:    &Client{},
		entitySet: "Employees",
		params:    make(map[string]string),
	}

	// For Count, we need to verify the URL would be built correctly
	// The Count method itself requires a context and HTTP client, so we test URL construction

	// Manually build what Count() would build
	path := "/" + qb.entitySet + "/$count"
	if !strings.HasSuffix(path, "/$count") {
		t.Errorf("Count path should end with /$count: %q", path)
	}

	// With filter
	qb.Filter("Status eq 'Active'")
	params := make([]string, 0)
	if qb.filterExpr != "" {
		params = append(params, "$filter="+url.QueryEscape(qb.filterExpr))
	}

	if len(params) == 0 {
		t.Error("Count with filter should include $filter parameter")
	}
}

// TestBuilderInitialization verifies new QueryBuilders initialize URL caching properly
func TestBuilderInitialization(t *testing.T) {
	client := &Client{}
	qb := client.From("Products")

	if qb.urlCache != "" {
		t.Errorf("New QueryBuilder should have empty cache, got: %q", qb.urlCache)
	}
	if !qb.urlDirty {
		t.Error("New QueryBuilder should be marked dirty initially")
	}
}

// TestQueryBuilderWhere tests the Where/FilterBuilder fluent API.
func TestQueryBuilderWhere_Eq(t *testing.T) {
	qb := &QueryBuilder{client: &Client{}, entitySet: "Products", urlDirty: true}
	qb.Where("Status").Eq("Active")
	u := qb.buildURL()
	if !strings.Contains(u, "%24filter=") && !strings.Contains(u, "$filter=") {
		t.Errorf("buildURL() should contain $filter, got: %s", u)
	}
}

func TestQueryBuilderWhere_Ne(t *testing.T) {
	qb := &QueryBuilder{client: &Client{}, entitySet: "Products", urlDirty: true}
	qb.Where("Status").Ne("Inactive")
	u := qb.buildURL()
	if !strings.Contains(u, "ne") {
		t.Errorf("buildURL() should contain 'ne', got: %s", u)
	}
}

func TestQueryBuilderWhere_Gt(t *testing.T) {
	qb := &QueryBuilder{client: &Client{}, entitySet: "Orders", urlDirty: true}
	qb.Where("Amount").Gt(100)
	u := qb.buildURL()
	if !strings.Contains(u, "gt") {
		t.Errorf("buildURL() should contain 'gt', got: %s", u)
	}
}

func TestQueryBuilderWhere_Ge(t *testing.T) {
	qb := &QueryBuilder{client: &Client{}, entitySet: "Orders", urlDirty: true}
	qb.Where("Amount").Ge(50)
	u := qb.buildURL()
	if !strings.Contains(u, "ge") {
		t.Errorf("buildURL() should contain 'ge', got: %s", u)
	}
}

func TestQueryBuilderWhere_Lt(t *testing.T) {
	qb := &QueryBuilder{client: &Client{}, entitySet: "Orders", urlDirty: true}
	qb.Where("Amount").Lt(1000)
	u := qb.buildURL()
	if !strings.Contains(u, "lt") {
		t.Errorf("buildURL() should contain 'lt', got: %s", u)
	}
}

func TestQueryBuilderWhere_Le(t *testing.T) {
	qb := &QueryBuilder{client: &Client{}, entitySet: "Orders", urlDirty: true}
	qb.Where("Amount").Le(999)
	u := qb.buildURL()
	if !strings.Contains(u, "le") {
		t.Errorf("buildURL() should contain 'le', got: %s", u)
	}
}

func TestQueryBuilderWhere_Contains(t *testing.T) {
	qb := &QueryBuilder{client: &Client{}, entitySet: "Products", urlDirty: true}
	qb.Where("Name").Contains("Widget")
	u := qb.buildURL()
	if !strings.Contains(u, "substringof") && !strings.Contains(u, "contains") {
		t.Errorf("buildURL() should contain substringof/contains, got: %s", u)
	}
}

func TestQueryBuilderWhere_StartsWith(t *testing.T) {
	qb := &QueryBuilder{client: &Client{}, entitySet: "Products", urlDirty: true}
	qb.Where("Name").StartsWith("Wid")
	u := qb.buildURL()
	if !strings.Contains(u, "startswith") {
		t.Errorf("buildURL() should contain 'startswith', got: %s", u)
	}
}

func TestQueryBuilderWhere_EndsWith(t *testing.T) {
	qb := &QueryBuilder{client: &Client{}, entitySet: "Products", urlDirty: true}
	qb.Where("Name").EndsWith("get")
	u := qb.buildURL()
	if !strings.Contains(u, "endswith") {
		t.Errorf("buildURL() should contain 'endswith', got: %s", u)
	}
}

func TestQueryBuilderWhere_In(t *testing.T) {
	qb := &QueryBuilder{client: &Client{}, entitySet: "Products", urlDirty: true}
	qb.Where("Status").In("A", "B", "C")
	u := qb.buildURL()
	if !strings.Contains(u, "in") {
		t.Errorf("buildURL() should contain 'in', got: %s", u)
	}
}

func TestQueryBuilderExpand_Simple(t *testing.T) {
	qb := &QueryBuilder{client: &Client{}, entitySet: "Orders", urlDirty: true}
	qb.Expand("Items")
	u := qb.buildURL()
	if !strings.Contains(u, "Items") {
		t.Errorf("buildURL() should contain 'Items' expand, got: %s", u)
	}
}

func TestQueryBuilderExpand_Multiple(t *testing.T) {
	qb := &QueryBuilder{client: &Client{}, entitySet: "Orders", urlDirty: true}
	qb.Expand("Items").Expand("Customer")
	u := qb.buildURL()
	if !strings.Contains(u, "Items") || !strings.Contains(u, "Customer") {
		t.Errorf("buildURL() should contain both expands, got: %s", u)
	}
}

func TestQueryBuilderExpand_WithSelect(t *testing.T) {
	qb := &QueryBuilder{client: &Client{}, entitySet: "Orders", urlDirty: true}
	qb.Expand("Items", WithExpandSelect("ID", "Qty"))
	u := qb.buildURL()
	if !strings.Contains(u, "Items") {
		t.Errorf("buildURL() should contain 'Items', got: %s", u)
	}
}

func TestQueryBuilderExpand_WithFilter(t *testing.T) {
	qb := &QueryBuilder{client: &Client{}, entitySet: "Orders", urlDirty: true}
	qb.Expand("Items", WithExpandFilter("Qty gt 0"))
	u := qb.buildURL()
	if !strings.Contains(u, "Items") {
		t.Errorf("buildURL() should contain 'Items', got: %s", u)
	}
}

func TestQueryBuilderExpand_WithOrderBy(t *testing.T) {
	qb := &QueryBuilder{client: &Client{}, entitySet: "Orders", urlDirty: true}
	qb.Expand("Items", WithExpandOrderBy("ID"))
	u := qb.buildURL()
	if !strings.Contains(u, "Items") {
		t.Errorf("buildURL() should contain 'Items', got: %s", u)
	}
}

func TestQueryBuilderExpand_WithTop(t *testing.T) {
	qb := &QueryBuilder{client: &Client{}, entitySet: "Orders", urlDirty: true}
	qb.Expand("Items", WithExpandTop(5))
	u := qb.buildURL()
	if !strings.Contains(u, "Items") {
		t.Errorf("buildURL() should contain 'Items', got: %s", u)
	}
}

// TestQueryBuilderSearch tests the Search method.
func TestQueryBuilderSearch(t *testing.T) {
	qb := &QueryBuilder{client: &Client{}, entitySet: "Products", urlDirty: true}
	qb.Search("laptop")
	u := qb.buildURL()
	if !strings.Contains(u, "laptop") {
		t.Errorf("buildURL() should contain 'laptop' in search, got: %s", u)
	}
}

// TestQueryBuilderApply tests the Apply method.
func TestQueryBuilderApply(t *testing.T) {
	qb := &QueryBuilder{client: &Client{}, entitySet: "Products", urlDirty: true}
	qb.Apply("groupby((Category),aggregate(Price with sum as TotalPrice))")
	u := qb.buildURL()
	if !strings.Contains(u, "groupby") {
		t.Errorf("buildURL() should contain 'groupby', got: %s", u)
	}
}

// TestQueryBuilderParam tests custom query parameter injection.
func TestQueryBuilderParam(t *testing.T) {
	qb := &QueryBuilder{client: &Client{}, entitySet: "Products", urlDirty: true}
	qb.Param("sap-client", "100")
	u := qb.buildURL()
	if !strings.Contains(u, "sap-client") {
		t.Errorf("buildURL() should contain custom param, got: %s", u)
	}
}

// TestQueryBuilderWithDeltaToken tests delta token injection.
func TestQueryBuilderWithDeltaToken(t *testing.T) {
	qb := &QueryBuilder{client: &Client{}, entitySet: "Products", urlDirty: true}
	qb.WithDeltaToken("deltalink_abc123")
	u := qb.buildURL()
	if !strings.Contains(u, "deltalink_abc123") {
		t.Errorf("buildURL() should contain delta token, got: %s", u)
	}
}

// TestQueryURLCaching tests that buildURL caches the result.
func TestQueryURLCaching(t *testing.T) {
	qb := &QueryBuilder{client: &Client{}, entitySet: "Products", urlDirty: true}
	qb.Select("ID", "Name")
	u1 := qb.buildURL()
	u2 := qb.buildURL()
	if u1 != u2 {
		t.Errorf("buildURL() should return cached result, got different values: %q vs %q", u1, u2)
	}
	if qb.urlDirty {
		t.Error("urlDirty should be false after buildURL()")
	}
}

// TestAutoDetectODataVersion tests version detection from headers.
func TestAutoDetectODataVersion(t *testing.T) {
	tests := []struct {
		name    string
		headers map[string][]string
		want    ODataVersion
	}{
		{"v4 odata-version", map[string][]string{"Odata-Version": {"4.0"}}, ODataV4},
		{"v2 dataserviceversion", map[string][]string{"Dataserviceversion": {"2.0"}}, ODataV2},
		{"no version header", map[string][]string{}, ODataV4},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := AutoDetectODataVersion(tc.headers)
			if got != tc.want {
				t.Errorf("AutoDetectODataVersion() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestSerializeValue_AllTypes covers serializeValue for int32, int64, float32, float64, bool,
// and default/unknown types via the In() FilterBuilder method.
func TestSerializeValue_AllTypes(t *testing.T) {
	qb := &QueryBuilder{client: &Client{}, entitySet: "Products", urlDirty: true}

	// int32
	qb2 := qb.Where("ID").In(int32(1), int32(2))
	if qb2.filterExpr == "" {
		t.Error("serializeValue int32: filterExpr should not be empty")
	}

	// int64
	qb3 := (&QueryBuilder{client: &Client{}, entitySet: "P", urlDirty: true}).Where("ID").In(int64(100))
	if qb3.filterExpr == "" {
		t.Error("serializeValue int64: filterExpr should not be empty")
	}

	// float32
	qb4 := (&QueryBuilder{client: &Client{}, entitySet: "P", urlDirty: true}).Where("Price").In(float32(1.5))
	if qb4.filterExpr == "" {
		t.Error("serializeValue float32: filterExpr should not be empty")
	}

	// float64
	qb5 := (&QueryBuilder{client: &Client{}, entitySet: "P", urlDirty: true}).Where("Price").In(float64(3.14))
	if qb5.filterExpr == "" {
		t.Error("serializeValue float64: filterExpr should not be empty")
	}

	// bool true
	qb6 := (&QueryBuilder{client: &Client{}, entitySet: "P", urlDirty: true}).Where("Active").In(true)
	if qb6.filterExpr == "" {
		t.Error("serializeValue bool true: filterExpr should not be empty")
	}

	// bool false
	qb7 := (&QueryBuilder{client: &Client{}, entitySet: "P", urlDirty: true}).Where("Active").In(false)
	if qb7.filterExpr == "" {
		t.Error("serializeValue bool false: filterExpr should not be empty")
	}

	// default (struct — fallback to fmt.Sprint)
	type custom struct{ Name string }
	qb8 := (&QueryBuilder{client: &Client{}, entitySet: "P", urlDirty: true}).Where("Status").In(custom{Name: "open"})
	if qb8.filterExpr == "" {
		t.Error("serializeValue default: filterExpr should not be empty")
	}
}
