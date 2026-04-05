package traverse

import (
	"net/url"
	"strings"
	"testing"
)

func TestLambdaAny_BasicEq(t *testing.T) {
	qb := &QueryBuilder{
		client:             &Client{},
		entitySet:          "Products",
		urlDirty:           true,
		conditionalHeaders: make(map[string]string),
	}
	qb.LambdaAny("Tags", func(b *LambdaBuilder) {
		b.Field("Name").Eq("admin")
	})

	if qb.filterExpr != "Tags/any(t: t/Name eq 'admin')" {
		t.Errorf("unexpected filter: %q", qb.filterExpr)
	}
}

func TestLambdaAll_BasicGt(t *testing.T) {
	qb := &QueryBuilder{
		client:             &Client{},
		entitySet:          "Orders",
		urlDirty:           true,
		conditionalHeaders: make(map[string]string),
	}
	qb.LambdaAll("Items", func(b *LambdaBuilder) {
		b.Field("Price").Gt(100)
	})

	if qb.filterExpr != "Items/all(i: i/Price gt 100)" {
		t.Errorf("unexpected filter: %q", qb.filterExpr)
	}
}

func TestLambdaAny_ContainsFn(t *testing.T) {
	qb := &QueryBuilder{
		client:             &Client{},
		entitySet:          "Posts",
		urlDirty:           true,
		conditionalHeaders: make(map[string]string),
	}
	qb.LambdaAny("Tags", func(b *LambdaBuilder) {
		b.Field("Value").Contains("go")
	})

	if qb.filterExpr != "Tags/any(t: contains(t/Value, 'go'))" {
		t.Errorf("unexpected filter: %q", qb.filterExpr)
	}
}

func TestLambdaAny_StartsWithFn(t *testing.T) {
	qb := &QueryBuilder{
		client:             &Client{},
		entitySet:          "Posts",
		urlDirty:           true,
		conditionalHeaders: make(map[string]string),
	}
	qb.LambdaAny("Tags", func(b *LambdaBuilder) {
		b.Field("Name").StartsWith("pre")
	})

	if qb.filterExpr != "Tags/any(t: startswith(t/Name, 'pre'))" {
		t.Errorf("unexpected filter: %q", qb.filterExpr)
	}
}

func TestLambdaAny_EndsWithFn(t *testing.T) {
	qb := &QueryBuilder{
		client:             &Client{},
		entitySet:          "Posts",
		urlDirty:           true,
		conditionalHeaders: make(map[string]string),
	}
	qb.LambdaAny("Tags", func(b *LambdaBuilder) {
		b.Field("Name").EndsWith("suffix")
	})

	if qb.filterExpr != "Tags/any(t: endswith(t/Name, 'suffix'))" {
		t.Errorf("unexpected filter: %q", qb.filterExpr)
	}
}

func TestLambdaCondition_Ne(t *testing.T) {
	qb := &QueryBuilder{
		client:             &Client{},
		entitySet:          "Orders",
		urlDirty:           true,
		conditionalHeaders: make(map[string]string),
	}
	qb.LambdaAny("Items", func(b *LambdaBuilder) {
		b.Field("Status").Ne("Cancelled")
	})

	if qb.filterExpr != "Items/any(i: i/Status ne 'Cancelled')" {
		t.Errorf("unexpected filter: %q", qb.filterExpr)
	}
}

func TestLambdaCondition_Le(t *testing.T) {
	qb := &QueryBuilder{
		client:             &Client{},
		entitySet:          "Orders",
		urlDirty:           true,
		conditionalHeaders: make(map[string]string),
	}
	qb.LambdaAll("Items", func(b *LambdaBuilder) {
		b.Field("Qty").Le(10)
	})

	if qb.filterExpr != "Items/all(i: i/Qty le 10)" {
		t.Errorf("unexpected filter: %q", qb.filterExpr)
	}
}

func TestLambdaCondition_Lt(t *testing.T) {
	qb := &QueryBuilder{
		client:             &Client{},
		entitySet:          "Orders",
		urlDirty:           true,
		conditionalHeaders: make(map[string]string),
	}
	qb.LambdaAll("Items", func(b *LambdaBuilder) {
		b.Field("Weight").Lt(5.0)
	})

	if qb.filterExpr != "Items/all(i: i/Weight lt 5)" {
		t.Errorf("unexpected filter: %q", qb.filterExpr)
	}
}

func TestLambdaCondition_Ge(t *testing.T) {
	qb := &QueryBuilder{
		client:             &Client{},
		entitySet:          "Products",
		urlDirty:           true,
		conditionalHeaders: make(map[string]string),
	}
	qb.LambdaAll("Scores", func(b *LambdaBuilder) {
		b.Field("Value").Ge(90)
	})

	if qb.filterExpr != "Scores/all(s: s/Value ge 90)" {
		t.Errorf("unexpected filter: %q", qb.filterExpr)
	}
}

func TestLambdaAny_MultipleConditions(t *testing.T) {
	qb := &QueryBuilder{
		client:             &Client{},
		entitySet:          "Products",
		urlDirty:           true,
		conditionalHeaders: make(map[string]string),
	}
	qb.LambdaAny("Tags", func(b *LambdaBuilder) {
		b.Field("Name").Eq("featured")
		b.Field("Active").Eq(true)
	})

	expected := "Tags/any(t: t/Name eq 'featured' and t/Active eq true)"
	if qb.filterExpr != expected {
		t.Errorf("unexpected filter:\n  got:  %q\n  want: %q", qb.filterExpr, expected)
	}
}

func TestLambdaAny_CombinesWithFilter(t *testing.T) {
	qb := &QueryBuilder{
		client:             &Client{},
		entitySet:          "Products",
		urlDirty:           true,
		conditionalHeaders: make(map[string]string),
	}
	qb.Filter("Price gt 50")
	qb.LambdaAny("Tags", func(b *LambdaBuilder) {
		b.Field("Name").Eq("sale")
	})

	expected := "Price gt 50 and Tags/any(t: t/Name eq 'sale')"
	if qb.filterExpr != expected {
		t.Errorf("unexpected filter:\n  got:  %q\n  want: %q", qb.filterExpr, expected)
	}
}

func TestFilterLambda_RawExpression(t *testing.T) {
	qb := &QueryBuilder{
		client:             &Client{},
		entitySet:          "Products",
		urlDirty:           true,
		conditionalHeaders: make(map[string]string),
	}
	qb.FilterLambda("tags/any(t: t/name eq 'admin')")

	if qb.filterExpr != "tags/any(t: t/name eq 'admin')" {
		t.Errorf("unexpected filter: %q", qb.filterExpr)
	}
}

func TestFilterLambda_CombinesWithExisting(t *testing.T) {
	qb := &QueryBuilder{
		client:             &Client{},
		entitySet:          "Products",
		urlDirty:           true,
		conditionalHeaders: make(map[string]string),
	}
	qb.Filter("Active eq true")
	qb.FilterLambda("Tags/any(t: t/Name eq 'featured')")

	expected := "Active eq true and Tags/any(t: t/Name eq 'featured')"
	if qb.filterExpr != expected {
		t.Errorf("unexpected filter:\n  got:  %q\n  want: %q", qb.filterExpr, expected)
	}
}

func TestLambdaAny_URLEncoded(t *testing.T) {
	qb := &QueryBuilder{
		client:             &Client{version: ODataV4},
		entitySet:          "Products",
		urlDirty:           true,
		conditionalHeaders: make(map[string]string),
		selectFields:       []string{},
		expandProps:        []string{},
		params:             map[string]string{},
	}
	qb.LambdaAny("Tags", func(b *LambdaBuilder) {
		b.Field("Name").Eq("admin")
	})

	rawURL := qb.buildURL()
	parsed, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("failed to parse URL: %v", err)
	}
	filter := parsed.Query().Get("$filter")
	if !strings.Contains(filter, "Tags/any(t: t/Name eq 'admin')") {
		t.Errorf("URL filter not found; rawURL=%q, filter=%q", rawURL, filter)
	}
}
