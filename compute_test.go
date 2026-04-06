package traverse

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCompute_SingleExpression(t *testing.T) {
	qb := &QueryBuilder{client: &Client{}, entitySet: "Products", urlDirty: true}
	qb.Compute("Price mul Quantity as Total")

	u := qb.buildURL()
	if !strings.Contains(u, "$compute=") {
		t.Errorf("expected $compute in URL, got: %s", u)
	}
	if !strings.Contains(u, "Price") || !strings.Contains(u, "Total") {
		t.Errorf("expected compute expression in URL, got: %s", u)
	}
}

func TestCompute_MultipleExpressions(t *testing.T) {
	qb := &QueryBuilder{client: &Client{}, entitySet: "Products", urlDirty: true}
	qb.Compute("Price mul Quantity as Total").Compute("Tax div 100 as TaxRate")

	u := qb.buildURL()
	if !strings.Contains(u, "Total") {
		t.Errorf("expected 'Total' in URL, got: %s", u)
	}
	if !strings.Contains(u, "TaxRate") {
		t.Errorf("expected 'TaxRate' in URL, got: %s", u)
	}
}

func TestCompute_MultipleInOneCall(t *testing.T) {
	qb := &QueryBuilder{client: &Client{}, entitySet: "Products", urlDirty: true}
	qb.Compute("Price mul Quantity as Total", "Tax div 100 as TaxRate")

	u := qb.buildURL()
	if !strings.Contains(u, "Total") {
		t.Errorf("expected 'Total' in URL, got: %s", u)
	}
	if !strings.Contains(u, "TaxRate") {
		t.Errorf("expected 'TaxRate' in URL, got: %s", u)
	}
}

func TestCompute_EmptyExpressionsIgnored(t *testing.T) {
	qb := &QueryBuilder{client: &Client{}, entitySet: "Products", urlDirty: true}
	qb.Compute("", "  ", "Price mul Quantity as Total")

	u := qb.buildURL()
	if !strings.Contains(u, "Total") {
		t.Errorf("expected 'Total' in URL, got: %s", u)
	}
	// Should not have a leading comma before the expression
	if strings.Contains(u, "%2CPrice") || strings.Contains(u, ",Price") {
		t.Errorf("unexpected leading comma in URL, got: %s", u)
	}
}

func TestCompute_WithOtherParams(t *testing.T) {
	qb := &QueryBuilder{
		client:       &Client{},
		entitySet:    "Products",
		urlDirty:     true,
		selectFields: []string{},
		expandProps:  []string{},
		params:       make(map[string]string),
	}
	qb.Select("ID", "Name").Compute("Price mul Quantity as Total").Top(5)

	u := qb.buildURL()
	if !strings.Contains(u, "$select=") {
		t.Errorf("expected $select in URL, got: %s", u)
	}
	if !strings.Contains(u, "$compute=") {
		t.Errorf("expected $compute in URL, got: %s", u)
	}
	if !strings.Contains(u, "$top=5") {
		t.Errorf("expected $top=5 in URL, got: %s", u)
	}
}

func TestComputeExpr_JoinsParts(t *testing.T) {
	expr := ComputeExpr("Price", "mul", "Quantity", "as", "Total")
	expected := "Price mul Quantity as Total"
	if expr != expected {
		t.Errorf("expected %q, got %q", expected, expr)
	}
}

func TestComputeExpr_SinglePart(t *testing.T) {
	expr := ComputeExpr("Total")
	if expr != "Total" {
		t.Errorf("expected %q, got %q", "Total", expr)
	}
}

func TestComputeExpr_EmptyParts(t *testing.T) {
	expr := ComputeExpr()
	if expr != "" {
		t.Errorf("expected empty string, got %q", expr)
	}
}

func TestCompute_URLCacheInvalidated(t *testing.T) {
	qb := &QueryBuilder{client: &Client{}, entitySet: "Orders", urlDirty: true}

	u1 := qb.buildURL()
	qb.Compute("Amount mul 1.1 as WithTax")
	u2 := qb.buildURL()

	if u1 == u2 {
		t.Error("URL should differ after adding a compute expression")
	}
}

func TestCompute_NotInURLWhenEmpty(t *testing.T) {
	qb := &QueryBuilder{client: &Client{}, entitySet: "Products", urlDirty: true}
	u := qb.buildURL()
	if strings.Contains(u, "compute") {
		t.Errorf("expected no $compute in URL when not set, got: %s", u)
	}
}

func TestCompute_Integration(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		full := r.URL.String()
		if !strings.Contains(full, "compute") {
			http.Error(w, "missing $compute in "+full, http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"value":[{"ID":1,"Name":"Widget","Total":49.95}]}`))
	}))
	defer srv.Close()

	client, err := New(WithBaseURL(srv.URL))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	page, err := client.From("Products").
		Compute(ComputeExpr("Price", "mul", "Quantity", "as", "Total")).
		Page(context.Background())
	if err != nil {
		t.Fatalf("Page: %v", err)
	}
	if len(page.Value) != 1 {
		t.Errorf("expected 1 result, got %d", len(page.Value))
	}
}
