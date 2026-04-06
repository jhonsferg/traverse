package traverse

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// SearchExpression unit tests
// ---------------------------------------------------------------------------

func TestSearchWord(t *testing.T) {
	got := SearchWord("laptop").searchString()
	if got != "laptop" {
		t.Errorf("SearchWord: got %q, want %q", got, "laptop")
	}
}

func TestSearchPhrase(t *testing.T) {
	got := SearchPhrase("mountain bike").searchString()
	if got != `"mountain bike"` {
		t.Errorf("SearchPhrase: got %q, want %q", got, `"mountain bike"`)
	}
}

func TestSearchConvenienceSingleWord(t *testing.T) {
	expr := Search("laptop")
	if _, ok := expr.(searchTerm); !ok {
		t.Fatalf("Search(single word) should return searchTerm, got %T", expr)
	}
	if expr.searchString() != "laptop" {
		t.Errorf("got %q", expr.searchString())
	}
}

func TestSearchConvenienceMultiWord(t *testing.T) {
	expr := Search("mountain bike")
	if st, ok := expr.(searchTerm); !ok || !st.phrase {
		t.Fatalf("Search(multi word) should return phrase searchTerm")
	}
	if expr.searchString() != `"mountain bike"` {
		t.Errorf("got %q", expr.searchString())
	}
}

func TestSearchAnd(t *testing.T) {
	expr := SearchAnd(SearchWord("mountain"), SearchWord("view"))
	got := expr.searchString()
	want := "(mountain AND view)"
	if got != want {
		t.Errorf("SearchAnd: got %q, want %q", got, want)
	}
}

func TestSearchAndSingleExpr(t *testing.T) {
	expr := SearchAnd(SearchWord("only"))
	if expr.searchString() != "only" {
		t.Errorf("SearchAnd with one expr should not wrap: got %q", expr.searchString())
	}
}

func TestSearchOr(t *testing.T) {
	expr := SearchOr(SearchWord("mountain"), SearchWord("hill"))
	got := expr.searchString()
	want := "(mountain OR hill)"
	if got != want {
		t.Errorf("SearchOr: got %q, want %q", got, want)
	}
}

func TestSearchNot(t *testing.T) {
	expr := SearchNot(SearchWord("bike"))
	got := expr.searchString()
	want := "NOT bike"
	if got != want {
		t.Errorf("SearchNot: got %q, want %q", got, want)
	}
}

func TestSearchComplex(t *testing.T) {
	expr := SearchAnd(SearchWord("mountain"), SearchNot(SearchWord("bike")))
	got := expr.searchString()
	want := "(mountain AND NOT bike)"
	if got != want {
		t.Errorf("SearchComplex: got %q, want %q", got, want)
	}
}

func TestSearchOrNested(t *testing.T) {
	inner := SearchAnd(SearchWord("a"), SearchWord("b"))
	outer := SearchOr(inner, SearchWord("c"))
	got := outer.searchString()
	want := "((a AND b) OR c)"
	if got != want {
		t.Errorf("SearchOrNested: got %q, want %q", got, want)
	}
}

func TestQueryBuilderSearchURL(t *testing.T) {
	qb := &QueryBuilder{client: &Client{}, entitySet: "Products", urlDirty: true}
	qb.Search(SearchAnd(SearchWord("mountain"), SearchNot(SearchWord("bike"))))
	u := qb.buildURL()
	if !strings.Contains(u, "%24search=") && !strings.Contains(u, "$search=") {
		t.Errorf("URL should contain $search, got: %s", u)
	}
	if !strings.Contains(u, "mountain") {
		t.Errorf("URL should contain search term 'mountain', got: %s", u)
	}
}

// ---------------------------------------------------------------------------
// CrossJoinBuilder unit tests
// ---------------------------------------------------------------------------

func TestCrossJoinBuildURL_basic(t *testing.T) {
	c := &Client{}
	b := c.CrossJoin("Products", "Categories")
	u := b.buildURL()
	if u != "$crossjoin(Products,Categories)" {
		t.Errorf("buildURL basic: got %q", u)
	}
}

func TestCrossJoinBuildURL_threeEntities(t *testing.T) {
	c := &Client{}
	b := c.CrossJoin("A", "B", "C")
	u := b.buildURL()
	if u != "$crossjoin(A,B,C)" {
		t.Errorf("buildURL three entities: got %q", u)
	}
}

func TestCrossJoinBuildURL_withFilter(t *testing.T) {
	c := &Client{}
	b := c.CrossJoin("Products", "Categories").
		Filter("Products/CategoryID eq Categories/ID")
	u := b.buildURL()
	if !strings.HasPrefix(u, "$crossjoin(Products,Categories)?") {
		t.Errorf("URL should have path then params, got: %q", u)
	}
	if !strings.Contains(u, "$filter=") {
		t.Errorf("URL should contain $filter, got: %q", u)
	}
}

func TestCrossJoinBuildURL_withSelect(t *testing.T) {
	c := &Client{}
	b := c.CrossJoin("Products", "Categories").
		Select("Products/Name", "Categories/Name")
	u := b.buildURL()
	if !strings.Contains(u, "$select=Products%2FName") && !strings.Contains(u, "$select=Products/Name") {
		t.Errorf("URL should contain $select, got: %q", u)
	}
}

func TestCrossJoinBuildURL_withExpand(t *testing.T) {
	c := &Client{}
	b := c.CrossJoin("Products", "Orders").
		Expand("Products/Supplier")
	u := b.buildURL()
	if !strings.Contains(u, "$expand=") {
		t.Errorf("URL should contain $expand, got: %q", u)
	}
}

func TestCrossJoinBuildURL_withCustomParam(t *testing.T) {
	c := &Client{}
	b := c.CrossJoin("X", "Y").Param("sap-client", "100")
	u := b.buildURL()
	if !strings.Contains(u, "sap-client") {
		t.Errorf("URL should contain custom param, got: %q", u)
	}
}

func TestCrossJoinPanicLessThanTwo(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("CrossJoin with < 2 entity sets should panic")
		}
	}()
	c := &Client{}
	c.CrossJoin("OnlyOne")
}

func TestCrossJoinResultDecode(t *testing.T) {
	type Product struct {
		Name string `json:"Name"`
	}
	row := CrossJoinResult{
		"Products": json.RawMessage(`{"Name":"Widget"}`),
	}
	var p Product
	if err := row.Decode("Products", &p); err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if p.Name != "Widget" {
		t.Errorf("got Name=%q, want Widget", p.Name)
	}
}

func TestCrossJoinResultDecodeMissingSet(t *testing.T) {
	row := CrossJoinResult{}
	var dest map[string]any
	err := row.Decode("Missing", &dest)
	if err == nil {
		t.Error("Decode with missing entity set should return error")
	}
}

func TestCrossJoinCollect(t *testing.T) {
	// Serve a minimal OData $crossjoin response.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "$crossjoin") {
			http.Error(w, "not found", 404)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"value": []map[string]any{
				{"Products": map[string]any{"Name": "Widget"}, "Categories": map[string]any{"Name": "Gadgets"}},
			},
		})
	}))
	defer srv.Close()

	client, err := New(WithBaseURL(srv.URL))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer func() { _ = client.Close() }()

	rows, err := client.CrossJoin("Products", "Categories").
		Filter("Products/CategoryID eq Categories/ID").
		Collect(t.Context())
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	var prod map[string]any
	if err := rows[0].Decode("Products", &prod); err != nil {
		t.Fatalf("Decode Products: %v", err)
	}
	if prod["Name"] != "Widget" {
		t.Errorf("expected Widget, got %v", prod["Name"])
	}
}

func TestCrossJoinCollectPaginated(t *testing.T) {
	page := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		page++
		resp := map[string]any{
			"value": []map[string]any{
				{"A": map[string]any{"ID": page}},
			},
		}
		if page == 1 {
			resp["@odata.nextLink"] = r.URL.Scheme + "://" + r.Host + "/$crossjoin(A,B)?$skip=1"
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	client, err := New(WithBaseURL(srv.URL))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer func() { _ = client.Close() }()

	rows, err := client.CrossJoin("A", "B").Collect(t.Context())
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if len(rows) != 2 {
		t.Errorf("expected 2 rows across 2 pages, got %d", len(rows))
	}
}
