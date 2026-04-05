package traverse

import (
	"strings"
	"testing"
)

func TestExpandNested_BuildsNestedExpression(t *testing.T) {
	qb := &QueryBuilder{
		client:    &Client{},
		entitySet: "Orders",
		urlDirty:  true,
	}
	qb.ExpandNested("Items").Select("ID").Done()

	if len(qb.expandProps) != 1 {
		t.Fatalf("expected 1 expand prop, got %d", len(qb.expandProps))
	}
	got := qb.expandProps[0]
	if got != "Items($select=ID)" {
		t.Errorf("expected Items($select=ID), got %q", got)
	}
}

func TestExpandNested_WithFilter(t *testing.T) {
	qb := &QueryBuilder{
		client:    &Client{},
		entitySet: "Orders",
		urlDirty:  true,
	}
	qb.ExpandNested("Items").Filter("Qty gt 0").Done()

	got := qb.expandProps[0]
	if got != "Items($filter=Qty gt 0)" {
		t.Errorf("expected Items($filter=Qty gt 0), got %q", got)
	}
}

func TestExpandNested_WithOrderBy(t *testing.T) {
	qb := &QueryBuilder{
		client:    &Client{},
		entitySet: "Orders",
		urlDirty:  true,
	}
	qb.ExpandNested("Items").OrderBy("Price desc").Done()

	got := qb.expandProps[0]
	if got != "Items($orderby=Price desc)" {
		t.Errorf("expected Items($orderby=Price desc), got %q", got)
	}
}

func TestExpandNested_WithTop(t *testing.T) {
	qb := &QueryBuilder{
		client:    &Client{},
		entitySet: "Orders",
		urlDirty:  true,
	}
	qb.ExpandNested("Items").Top(5).Done()

	got := qb.expandProps[0]
	if got != "Items($top=5)" {
		t.Errorf("expected Items($top=5), got %q", got)
	}
}

func TestExpandNested_MultipleOptions(t *testing.T) {
	qb := &QueryBuilder{
		client:    &Client{},
		entitySet: "Orders",
		urlDirty:  true,
	}
	qb.ExpandNested("Items").Select("ID", "Qty").Filter("Qty gt 0").Top(10).Done()

	got := qb.expandProps[0]
	if !strings.HasPrefix(got, "Items(") {
		t.Errorf("expected Items(...), got %q", got)
	}
	if !strings.Contains(got, "$select=ID,Qty") {
		t.Errorf("missing $select in %q", got)
	}
	if !strings.Contains(got, "$filter=Qty gt 0") {
		t.Errorf("missing $filter in %q", got)
	}
	if !strings.Contains(got, "$top=10") {
		t.Errorf("missing $top in %q", got)
	}
}

func TestExpandNested_FurtherExpand(t *testing.T) {
	qb := &QueryBuilder{
		client:    &Client{},
		entitySet: "Orders",
		urlDirty:  true,
	}
	qb.ExpandNested("Items").Expand("Product").Done()

	got := qb.expandProps[0]
	if got != "Items($expand=Product)" {
		t.Errorf("expected Items($expand=Product), got %q", got)
	}
}

func TestExpandNested_NoOptions(t *testing.T) {
	qb := &QueryBuilder{
		client:    &Client{},
		entitySet: "Orders",
		urlDirty:  true,
	}
	qb.ExpandNested("Customer").Done()

	got := qb.expandProps[0]
	if got != "Customer" {
		t.Errorf("expected plain Customer, got %q", got)
	}
}

func TestExpandNested_URLIntegration(t *testing.T) {
	qb := &QueryBuilder{
		client:    &Client{},
		entitySet: "Orders",
		urlDirty:  true,
	}
	qb.ExpandNested("Items").Select("ID", "Qty").Filter("Qty gt 0").Done()

	u := qb.buildURL()
	if !strings.Contains(u, "$expand=") {
		t.Errorf("URL missing $expand, got: %s", u)
	}
	if !strings.Contains(u, "Items") {
		t.Errorf("URL missing Items expand, got: %s", u)
	}
}

func TestExpandNested_Chaining_ReturnsParent(t *testing.T) {
	qb := &QueryBuilder{
		client:    &Client{},
		entitySet: "Orders",
		urlDirty:  true,
	}
	result := qb.ExpandNested("Items").Select("ID").Done()
	if result != qb {
		t.Error("Done() should return the parent QueryBuilder")
	}
}
