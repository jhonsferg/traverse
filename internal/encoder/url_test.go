package encoder

import (
	"strings"
	"testing"
)

func intPtr(i int) *int { return &i }

func TestBuildURL_InvalidBase(t *testing.T) {
	_, err := BuildURL("ftp://example.com", "Products", QueryOptions{})
	if err == nil {
		t.Fatal("expected error for invalid base URL scheme")
	}
}

func TestBuildURL_NoTrailingSlash(t *testing.T) {
	got, err := BuildURL("http://example.com/odata", "Products", QueryOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "http://example.com/odata/Products" {
		t.Fatalf("got %q", got)
	}
}

func TestBuildURL_TrailingSlash(t *testing.T) {
	got, err := BuildURL("http://example.com/odata/", "Products", QueryOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "http://example.com/odata/Products" {
		t.Fatalf("got %q", got)
	}
}

func TestBuildURL_AllOptions(t *testing.T) {
	opts := QueryOptions{
		Select:     []string{"ID", "Name"},
		Filter:     "Price%20gt%20100",
		OrderBy:    "Name desc",
		Expand:     []string{"Orders"},
		Top:        intPtr(10),
		Skip:       intPtr(5),
		Count:      true,
		Search:     "widget",
		Apply:      "groupby((Category))",
		DeltaToken: "abc123",
	}
	got, err := BuildURL("https://example.com/odata", "Products", opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	checks := []string{
		"https://example.com/odata/Products?",
		"$select=ID,Name",
		"$filter=Price%20gt%20100",
		"$orderby=Name+desc",
		"$expand=Orders",
		"$top=10",
		"$skip=5",
		"$count=true",
		"$search=widget",
		"$apply=groupby",
		"$deltatoken=abc123",
	}
	for _, c := range checks {
		if !strings.Contains(got, c) {
			t.Errorf("expected URL to contain %q, got %q", c, got)
		}
	}
}

func TestBuildURL_ZeroTopSkip(t *testing.T) {
	got, err := BuildURL("http://example.com", "Products", QueryOptions{Top: intPtr(0), Skip: intPtr(0)})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(got, "$top=0") || !strings.Contains(got, "$skip=0") {
		t.Fatalf("zero pointer values should still be included, got %q", got)
	}
}

func TestBuildURL_NoOptions(t *testing.T) {
	got, err := BuildURL("http://example.com", "Products", QueryOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(got, "?") {
		t.Fatalf("expected no query string, got %q", got)
	}
}

func TestBuildURL_CustomParams(t *testing.T) {
	got, err := BuildURL("http://example.com", "Products", QueryOptions{
		Custom: map[string]string{"myparam": "my value"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(got, "myparam=my+value") {
		t.Fatalf("expected custom param encoded, got %q", got)
	}
}

func TestBuildNavigationURL_SimpleEntitySet(t *testing.T) {
	got := BuildNavigationURL("/ProductSet", "SalesOrders")
	if got != "/ProductSet/SalesOrders" {
		t.Fatalf("got %q", got)
	}
}

func TestBuildNavigationURL_WithKeyQuoted(t *testing.T) {
	got := BuildNavigationURL("/ProductSet('SKU001')", "Orders")
	if got != "/ProductSet('SKU001')/Orders" {
		t.Fatalf("got %q", got)
	}
}

func TestBuildNavigationURL_WithKeyNumeric(t *testing.T) {
	got := BuildNavigationURL("/ProductSet(1)", "Categories")
	if got != "/ProductSet(1)/Categories" {
		t.Fatalf("got %q", got)
	}
}

func TestEncodeExpandOption_Simple(t *testing.T) {
	got := EncodeExpandOption("Orders", nil, "")
	if got != "Orders" {
		t.Fatalf("got %q", got)
	}
}

func TestEncodeExpandOption_WithSelect(t *testing.T) {
	got := EncodeExpandOption("Orders", []string{"ID", "Amount"}, "")
	if got != "Orders($select=ID,Amount)" {
		t.Fatalf("got %q", got)
	}
}

func TestEncodeExpandOption_WithSelectAndFilter(t *testing.T) {
	got := EncodeExpandOption("Orders", []string{"ID", "Amount"}, "Amount gt 100")
	want := "Orders($select=ID,Amount;$filter=Amount gt 100)"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestEncodeExpandOption_OnlyFilter(t *testing.T) {
	got := EncodeExpandOption("Orders", nil, "Amount gt 100")
	want := "Orders($filter=Amount gt 100)"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
