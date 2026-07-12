package graphql

import "testing"

func TestNewQueryTranslator(t *testing.T) {
	qt := NewQueryTranslator()
	if qt == nil {
		t.Fatal("expected non-nil translator")
	}
}

func TestFilterToOData(t *testing.T) {
	qt := NewQueryTranslator()

	if got := qt.FilterToOData(""); got != "" {
		t.Fatalf("expected empty string for empty filter, got %q", got)
	}
	if got := qt.FilterToOData("Price gt 100"); got != "Price gt 100" {
		t.Fatalf("expected pass-through, got %q", got)
	}
}

func TestSelectToOData(t *testing.T) {
	qt := NewQueryTranslator()

	if got := qt.SelectToOData(nil); got != "" {
		t.Fatalf("expected empty string for nil fields, got %q", got)
	}
	if got := qt.SelectToOData([]string{}); got != "" {
		t.Fatalf("expected empty string for empty fields, got %q", got)
	}
	if got := qt.SelectToOData([]string{"ID"}); got != "ID" {
		t.Fatalf("expected ID, got %q", got)
	}
	if got := qt.SelectToOData([]string{"ID", "Name", "Price"}); got != "ID,Name,Price" {
		t.Fatalf("expected comma-joined fields, got %q", got)
	}
}

func TestOrderByToOData(t *testing.T) {
	qt := NewQueryTranslator()

	if got := qt.OrderByToOData(""); got != "" {
		t.Fatalf("expected empty string, got %q", got)
	}
	if got := qt.OrderByToOData("Name asc"); got != "Name asc" {
		t.Fatalf("got %q", got)
	}
	if got := qt.OrderByToOData("Name asc, Price desc"); got != "Name asc,Price desc" {
		t.Fatalf("expected trimmed and rejoined fields, got %q", got)
	}
	if got := qt.OrderByToOData("Name asc,  Price desc  , ID"); got != "Name asc,Price desc,ID" {
		t.Fatalf("expected whitespace trimmed around every field, got %q", got)
	}
}

func TestTopToOData(t *testing.T) {
	qt := NewQueryTranslator()

	if got := qt.TopToOData(0); got != 0 {
		t.Fatalf("expected 0 for zero top, got %d", got)
	}
	if got := qt.TopToOData(-5); got != 0 {
		t.Fatalf("expected 0 for negative top, got %d", got)
	}
	if got := qt.TopToOData(10); got != 10 {
		t.Fatalf("expected 10, got %d", got)
	}
}

func TestSkipToOData(t *testing.T) {
	qt := NewQueryTranslator()

	if got := qt.SkipToOData(0); got != 0 {
		t.Fatalf("expected 0 for zero skip, got %d", got)
	}
	if got := qt.SkipToOData(-3); got != 0 {
		t.Fatalf("expected 0 for negative skip, got %d", got)
	}
	if got := qt.SkipToOData(20); got != 20 {
		t.Fatalf("expected 20, got %d", got)
	}
}

func TestExpandToOData(t *testing.T) {
	qt := NewQueryTranslator()

	if got := qt.ExpandToOData(nil); got != "" {
		t.Fatalf("expected empty string for nil, got %q", got)
	}
	if got := qt.ExpandToOData([]string{"Orders", "Categories"}); got != "Orders,Categories" {
		t.Fatalf("got %q", got)
	}
}

func TestBuildODataQuery_Empty(t *testing.T) {
	qt := NewQueryTranslator()

	got := qt.BuildODataQuery(map[string]interface{}{})
	if got != "" {
		t.Fatalf("expected empty query for empty args, got %q", got)
	}
}

func TestBuildODataQuery_AllFields(t *testing.T) {
	qt := NewQueryTranslator()

	args := map[string]interface{}{
		"filter":  "Price gt 100",
		"select":  "ID,Name",
		"expand":  "Orders",
		"top":     10,
		"skip":    5,
		"orderBy": "Name asc",
		"count":   true,
	}
	got := qt.BuildODataQuery(args)
	want := "$filter=Price gt 100&$select=ID,Name&$expand=Orders&$top=10&$skip=5&$orderby=Name asc&$count=true"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestBuildODataQuery_IgnoresZeroAndFalseValues(t *testing.T) {
	qt := NewQueryTranslator()

	args := map[string]interface{}{
		"filter":  "",
		"select":  "",
		"expand":  "",
		"top":     0,
		"skip":    0,
		"orderBy": "",
		"count":   false,
	}
	got := qt.BuildODataQuery(args)
	if got != "" {
		t.Fatalf("expected empty query when all values are zero/empty/false, got %q", got)
	}
}

func TestBuildODataQuery_IgnoresWrongTypes(t *testing.T) {
	qt := NewQueryTranslator()

	// Wrong Go types for each key (e.g. int where string expected) should be
	// silently skipped rather than panicking, since the type assertions use
	// the ", ok" form.
	args := map[string]interface{}{
		"filter": 123,
		"select": true,
		"top":    "not-an-int",
		"count":  "not-a-bool",
	}
	got := qt.BuildODataQuery(args)
	if got != "" {
		t.Fatalf("expected empty query for mistyped args, got %q", got)
	}
}

func TestBuildODataQuery_PartialArgs(t *testing.T) {
	qt := NewQueryTranslator()

	args := map[string]interface{}{
		"filter": "Active eq true",
		"top":    5,
	}
	got := qt.BuildODataQuery(args)
	want := "$filter=Active eq true&$top=5"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
