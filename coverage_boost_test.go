package traverse

import (
	"context"
	"strings"
	"testing"

	"github.com/jhonsferg/traverse/testutil"
)

// TestQueryWhere tests the Where method as an alias to Filter with Eq.
func TestQueryWhere(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{
		Status: 200,
		Body:   `{"value":[{"ID":1,"Name":"Product1"}]}`,
	})

	c, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer func() { _ = c.Close() }()

	result, err := c.From("Products").Where("Name").Eq("Product1").First(context.Background())
	if err != nil {
		t.Fatalf("Where+Eq: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
}

// TestQueryOrderBy tests the OrderBy method constructs proper OData URL.
func TestQueryOrderBy(t *testing.T) {
	qb := &QueryBuilder{client: &Client{}, entitySet: "Products", urlDirty: true}
	qb.OrderBy("Name")
	u := qb.buildURL()
	if !strings.Contains(u, "%24orderby=") && !strings.Contains(u, "$orderby=") {
		t.Errorf("buildURL() should contain $orderby, got: %s", u)
	}
	if !strings.Contains(u, "asc") && !strings.Contains(u, "%20asc") {
		t.Errorf("buildURL() should contain 'asc', got: %s", u)
	}
}

// TestQueryOrderBy_Multiple tests chaining multiple OrderBy calls.
func TestQueryOrderBy_Multiple(t *testing.T) {
	qb := &QueryBuilder{client: &Client{}, entitySet: "Products", urlDirty: true}
	qb.OrderBy("Name").OrderBy("Price")
	u := qb.buildURL()
	if !strings.Contains(u, "%24orderby=") && !strings.Contains(u, "$orderby=") {
		t.Errorf("buildURL() should contain $orderby, got: %s", u)
	}
	if !strings.Contains(u, ",") && !strings.Contains(u, "2C") {
		t.Errorf("buildURL() should contain comma for multiple orderby, got: %s", u)
	}
}

// TestQueryOrderByDesc tests the OrderByDesc method.
func TestQueryOrderByDesc(t *testing.T) {
	qb := &QueryBuilder{client: &Client{}, entitySet: "Products", urlDirty: true}
	qb.OrderByDesc("Name")
	u := qb.buildURL()
	if !strings.Contains(u, "%24orderby=") && !strings.Contains(u, "$orderby=") {
		t.Errorf("buildURL() should contain $orderby, got: %s", u)
	}
	if !strings.Contains(u, "desc") && !strings.Contains(u, "%20desc") {
		t.Errorf("buildURL() should contain 'desc', got: %s", u)
	}
}

// TestQueryOrderByDesc_Multiple tests chaining OrderByDesc multiple times.
func TestQueryOrderByDesc_Multiple(t *testing.T) {
	qb := &QueryBuilder{client: &Client{}, entitySet: "Products", urlDirty: true}
	qb.OrderByDesc("Price").OrderByDesc("Name")
	u := qb.buildURL()
	if !strings.Contains(u, "%24orderby=") && !strings.Contains(u, "$orderby=") {
		t.Errorf("buildURL() should contain $orderby, got: %s", u)
	}
	if !strings.Contains(u, "desc") && !strings.Contains(u, "%20desc") {
		t.Errorf("buildURL() should contain 'desc', got: %s", u)
	}
}

// TestQueryTop tests the Top method.
func TestQueryTop(t *testing.T) {
	qb := &QueryBuilder{client: &Client{}, entitySet: "Products", urlDirty: true}
	qb.Top(10)
	u := qb.buildURL()
	if !strings.Contains(u, "%24top=") && !strings.Contains(u, "$top=") {
		t.Errorf("buildURL() should contain $top, got: %s", u)
	}
	if !strings.Contains(u, "10") {
		t.Errorf("buildURL() should contain '10', got: %s", u)
	}
}

// TestQuerySkip tests the Skip method.
func TestQuerySkip(t *testing.T) {
	qb := &QueryBuilder{client: &Client{}, entitySet: "Products", urlDirty: true}
	qb.Skip(5)
	u := qb.buildURL()
	if !strings.Contains(u, "%24skip=") && !strings.Contains(u, "$skip=") {
		t.Errorf("buildURL() should contain $skip, got: %s", u)
	}
	if !strings.Contains(u, "5") {
		t.Errorf("buildURL() should contain '5', got: %s", u)
	}
}

// TestQuerySearch tests the Search method.
func TestQuerySearch(t *testing.T) {
	qb := &QueryBuilder{client: &Client{}, entitySet: "Products", urlDirty: true}
	qb.Search(SearchWord("Product"))
	u := qb.buildURL()
	if !strings.Contains(u, "%24search=") && !strings.Contains(u, "$search=") {
		t.Errorf("buildURL() should contain $search, got: %s", u)
	}
	if !strings.Contains(u, "Product") {
		t.Errorf("buildURL() should contain 'Product', got: %s", u)
	}
}

// TestQueryApply tests the Apply method.
func TestQueryApply(t *testing.T) {
	qb := &QueryBuilder{client: &Client{}, entitySet: "Products", urlDirty: true}
	qb.Apply("aggregate(Amount with sum as Total)")
	u := qb.buildURL()
	if !strings.Contains(u, "%24apply=") && !strings.Contains(u, "$apply=") {
		t.Errorf("buildURL() should contain $apply, got: %s", u)
	}
	if !strings.Contains(u, "aggregate") && !strings.Contains(u, "%24aggregate") {
		t.Errorf("buildURL() should contain 'aggregate', got: %s", u)
	}
}

// TestQueryParam tests the Param method for custom parameters.
func TestQueryParam(t *testing.T) {
	qb := &QueryBuilder{client: &Client{}, entitySet: "Products", urlDirty: true}
	qb.Param("custom", "value123")
	u := qb.buildURL()
	if !strings.Contains(u, "custom=") {
		t.Errorf("buildURL() should contain 'custom=', got: %s", u)
	}
	if !strings.Contains(u, "value123") {
		t.Errorf("buildURL() should contain 'value123', got: %s", u)
	}
}

// TestQueryWithDeltaToken tests the WithDeltaToken method.
func TestQueryWithDeltaToken(t *testing.T) {
	qb := &QueryBuilder{client: &Client{}, entitySet: "Products", urlDirty: true}
	qb.WithDeltaToken("delta-token-123")
	u := qb.buildURL()
	if !strings.Contains(u, "%24deltatoken=") && !strings.Contains(u, "$deltatoken=") {
		t.Errorf("buildURL() should contain $deltatoken, got: %s", u)
	}
	if !strings.Contains(u, "delta-token-123") {
		t.Errorf("buildURL() should contain 'delta-token-123', got: %s", u)
	}
}

// TestQueryStreamAs tests the StreamAs method for streaming generic results.
func TestQueryStreamAs(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{
		Status: 200,
		Body:   `{"value":[{"ID":1,"Name":"A"},{"ID":2,"Name":"B"}]}`,
	})

	c, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer func() { _ = c.Close() }()

	ch := c.From("Products").StreamAs(context.Background())
	var count int
	for r := range ch {
		if r.Err != nil {
			t.Fatalf("stream error: %v", r.Err)
		}
		if r.Value == nil {
			t.Fatal("expected value in stream result")
		}
		count++
	}
	if count != 2 {
		t.Fatalf("want 2 records, got %d", count)
	}
}

// TestQueryStreamAs_Error tests StreamAs with HTTP error.
func TestQueryStreamAs_Error(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{
		Status: 500,
		Body:   `{"error":{"code":"500","message":"Internal Server Error"}}`,
	})

	c, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer func() { _ = c.Close() }()

	ch := c.From("Products").StreamAs(context.Background())
	var errorFound bool
	for r := range ch {
		if r.Err != nil {
			errorFound = true
			break
		}
	}
	if !errorFound {
		t.Fatal("expected error in stream")
	}
}

// TestClientFetchPageAt tests the FetchPageAt method.
func TestClientFetchPageAt(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{
		Status: 200,
		Body:   `{"value":[{"ID":3,"Name":"C"}]}`,
	})

	c, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer func() { _ = c.Close() }()

	pageURL := server.URL() + "/Products?$skip=20&$top=10"
	page, err := c.FetchPageAt(context.Background(), pageURL)
	if err != nil {
		t.Fatalf("FetchPageAt: %v", err)
	}
	if len(page.Value) != 1 {
		t.Fatalf("want 1 result in fetched page, got %d", len(page.Value))
	}
}

// TestFilterBuilderLt tests less-than filter.
func TestFilterBuilderLt(t *testing.T) {
	qb := &QueryBuilder{client: &Client{}, entitySet: "Orders", urlDirty: true}
	qb.Where("Amount").Lt(200)
	u := qb.buildURL()
	if !strings.Contains(u, "lt") {
		t.Errorf("buildURL() should contain 'lt', got: %s", u)
	}
}

// TestFilterBuilderLe tests less-than-or-equal filter.
func TestFilterBuilderLe(t *testing.T) {
	qb := &QueryBuilder{client: &Client{}, entitySet: "Orders", urlDirty: true}
	qb.Where("Amount").Le(200)
	u := qb.buildURL()
	if !strings.Contains(u, "le") {
		t.Errorf("buildURL() should contain 'le', got: %s", u)
	}
}

// TestFilterBuilderContains tests contains filter.
func TestFilterBuilderContains(t *testing.T) {
	qb := &QueryBuilder{client: &Client{}, entitySet: "Products", urlDirty: true}
	qb.Where("Name").Contains("Product")
	u := qb.buildURL()
	if !strings.Contains(u, "substringof") && !strings.Contains(u, "contains") {
		t.Errorf("buildURL() should contain substring/contains function, got: %s", u)
	}
}

// TestFilterBuilderStartsWith tests startswith filter.
func TestFilterBuilderStartsWith(t *testing.T) {
	qb := &QueryBuilder{client: &Client{}, entitySet: "Products", urlDirty: true}
	qb.Where("Name").StartsWith("Product")
	u := qb.buildURL()
	if !strings.Contains(u, "startswith") {
		t.Errorf("buildURL() should contain 'startswith', got: %s", u)
	}
}

// TestFilterBuilderEndsWith tests endswith filter.
func TestFilterBuilderEndsWith(t *testing.T) {
	qb := &QueryBuilder{client: &Client{}, entitySet: "Products", urlDirty: true}
	qb.Where("Name").EndsWith("A")
	u := qb.buildURL()
	if !strings.Contains(u, "endswith") {
		t.Errorf("buildURL() should contain 'endswith', got: %s", u)
	}
}

// TestFilterBuilderIn tests the In method for multiple values.
func TestFilterBuilderIn(t *testing.T) {
	qb := &QueryBuilder{client: &Client{}, entitySet: "Products", urlDirty: true}
	qb.Where("Status").In("Active", "Pending")
	u := qb.buildURL()
	if !strings.Contains(u, "%24filter=") && !strings.Contains(u, "$filter=") {
		t.Errorf("buildURL() should contain $filter, got: %s", u)
	}
	if !strings.Contains(u, "in") {
		t.Errorf("buildURL() should contain 'in' for In, got: %s", u)
	}
}

// TestQueryBoundFunction tests BoundFunction.
func TestQueryBoundFunction(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{
		Status: 200,
		Body:   `true`,
	})

	c, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer func() { _ = c.Close() }()

	fb := c.From("Products").BoundFunction("IsInStock")
	result, err := fb.Execute(context.Background())
	if err != nil {
		t.Fatalf("BoundFunction Execute: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result from BoundFunction")
	}
}

// TestWithExpandSelect tests the WithExpandSelect option.
func TestWithExpandSelect(t *testing.T) {
	qb := &QueryBuilder{client: &Client{}, entitySet: "Orders", urlDirty: true}
	qb.Expand("Items", WithExpandSelect("LineNo", "Amount"))
	u := qb.buildURL()
	if !strings.Contains(u, "%24expand=") && !strings.Contains(u, "$expand=") {
		t.Errorf("buildURL() should contain $expand, got: %s", u)
	}
	if !strings.Contains(u, "select") && !strings.Contains(u, "%24select") {
		t.Errorf("buildURL() should contain nested select, got: %s", u)
	}
}

// TestQueryWithCount tests the WithCount method.
func TestQueryWithCount(t *testing.T) {
	qb := &QueryBuilder{client: &Client{}, entitySet: "Products", urlDirty: true}
	qb.WithCount()
	u := qb.buildURL()
	if !strings.Contains(u, "%24count=") && !strings.Contains(u, "$count=") {
		t.Errorf("buildURL() should contain $count, got: %s", u)
	}
	if !strings.Contains(u, "true") {
		t.Errorf("buildURL() should contain 'true' for count, got: %s", u)
	}
}

// TestQueryBuildExpr_Indirect tests buildExpr indirectly through complex filter chains.
func TestQueryBuildExpr_Indirect(t *testing.T) {
	qb := &QueryBuilder{client: &Client{}, entitySet: "Products", urlDirty: true}
	qb.Filter("(Status eq 'Active' or Status eq 'Pending')")
	u := qb.buildURL()
	if !strings.Contains(u, "%24filter=") && !strings.Contains(u, "$filter=") {
		t.Errorf("buildURL() should contain $filter, got: %s", u)
	}
}

// TestQueryRelease_Indirect tests query builder pool reuse indirectly.
func TestQueryRelease_Indirect(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{
		Status: 200,
		Body:   `{"value":[{"ID":1}]}`,
	})
	server.Enqueue(testutil.MockResponse{
		Status: 200,
		Body:   `{"value":[{"ID":2}]}`,
	})

	c, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer func() { _ = c.Close() }()

	result1, err := c.From("Products").First(context.Background())
	if err != nil {
		t.Fatalf("First query: %v", err)
	}
	if result1 == nil {
		t.Fatal("expected non-nil result in first query")
	}

	result2, err := c.From("Products").First(context.Background())
	if err != nil {
		t.Fatalf("Second query: %v", err)
	}
	if result2 == nil {
		t.Fatal("expected non-nil result in second query")
	}
}

// TestQueryWithCount_ViaCollect tests WithCount with Collect.
func TestQueryWithCount_ViaCollect(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{
		Status: 200,
		Body:   `{"@odata.count":100,"value":[{"ID":1}]}`,
	})

	c, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer func() { _ = c.Close() }()

	results, err := c.From("Products").WithCount().Collect(context.Background())
	if err != nil {
		t.Fatalf("WithCount+Collect: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected at least one result")
	}
}
