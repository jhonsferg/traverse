package traverse

import (
	"strings"
	"testing"
	"time"
)

func TestF(t *testing.T) {
	expr := F("Name")
	if expr == nil {
		t.Fatal("F() should not return nil")
	}
	if expr.expr != "Name" {
		t.Errorf("F() should set field to 'Name', got %q", expr.expr)
	}
}

func TestEq(t *testing.T) {
	tests := []struct {
		name     string
		field    string
		value    interface{}
		expected string
	}{
		{"string equality", "Name", "Alice", "Name eq 'Alice'"},
		{"int equality", "Age", 30, "Age eq 30"},
		{"bool equality", "Active", true, "Active eq true"},
		{"bool false", "Deleted", false, "Deleted eq false"},
		{"float equality", "Price", 19.99, "Price eq 19.99"},
		{"nil equality", "Field", nil, "Field eq null"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr := F(tt.field).Eq(tt.value)
			if expr.Build() != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, expr.Build())
			}
		})
	}
}

func TestNe(t *testing.T) {
	expr := F("Status").Ne("Inactive")
	expected := "Status ne 'Inactive'"
	if expr.Build() != expected {
		t.Errorf("expected %q, got %q", expected, expr.Build())
	}
}

func TestLt(t *testing.T) {
	expr := F("Age").Lt(18)
	expected := "Age lt 18"
	if expr.Build() != expected {
		t.Errorf("expected %q, got %q", expected, expr.Build())
	}
}

func TestLe(t *testing.T) {
	expr := F("Score").Le(100)
	expected := "Score le 100"
	if expr.Build() != expected {
		t.Errorf("expected %q, got %q", expected, expr.Build())
	}
}

func TestGt(t *testing.T) {
	expr := F("Age").Gt(18)
	expected := "Age gt 18"
	if expr.Build() != expected {
		t.Errorf("expected %q, got %q", expected, expr.Build())
	}
}

func TestGe(t *testing.T) {
	expr := F("Age").Ge(18)
	expected := "Age ge 18"
	if expr.Build() != expected {
		t.Errorf("expected %q, got %q", expected, expr.Build())
	}
}

func TestContains(t *testing.T) {
	expr := F("Name").Contains("ali")
	expected := "contains(Name,'ali')"
	if expr.Build() != expected {
		t.Errorf("expected %q, got %q", expected, expr.Build())
	}
}

func TestStartsWith(t *testing.T) {
	expr := F("Email").StartsWith("user")
	expected := "startswith(Email,'user')"
	if expr.Build() != expected {
		t.Errorf("expected %q, got %q", expected, expr.Build())
	}
}

func TestEndsWith(t *testing.T) {
	expr := F("Email").EndsWith("@example.com")
	expected := "endswith(Email,'@example.com')"
	if expr.Build() != expected {
		t.Errorf("expected %q, got %q", expected, expr.Build())
	}
}

func TestStringQuoting(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected string
	}{
		{"simple string", "hello", "Name eq 'hello'"},
		{"string with single quote", "it's", "Name eq 'it''s'"},
		{"multiple single quotes", "don't worry", "Name eq 'don''t worry'"},
		{"string with special chars", "user@email.com", "Name eq 'user@email.com'"},
		{"empty string", "", "Name eq ''"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr := F("Name").Eq(tt.value)
			if expr.Build() != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, expr.Build())
			}
		})
	}
}

func TestIntTypes(t *testing.T) {
	tests := []struct {
		name     string
		field    string
		value    interface{}
		expected string
	}{
		{"int", "Count", int(42), "Count eq 42"},
		{"int32", "Count", int32(42), "Count eq 42"},
		{"int64", "Count", int64(42), "Count eq 42"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr := F(tt.field).Eq(tt.value)
			if expr.Build() != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, expr.Build())
			}
		})
	}
}

func TestFloatTypes(t *testing.T) {
	tests := []struct {
		name     string
		field    string
		value    interface{}
		expected string
	}{
		{"float32", "Price", float32(19.99), "Price eq 19.99"},
		{"float64", "Price", float64(19.99), "Price eq 19.99"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr := F(tt.field).Eq(tt.value)
			if expr.Build() != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, expr.Build())
			}
		})
	}
}

func TestTime(t *testing.T) {
	tm := time.Date(2006, 1, 2, 15, 4, 5, 0, time.UTC)
	expr := F("CreatedAt").Eq(tm)
	expected := "CreatedAt eq datetime'2006-01-02T15:04:05Z'"
	if expr.Build() != expected {
		t.Errorf("expected %q, got %q", expected, expr.Build())
	}
}

func TestAnd(t *testing.T) {
	expr := And(F("Age").Ge(18), F("Active").Eq(true))
	expected := "(Age ge 18) and (Active eq true)"
	if expr.Build() != expected {
		t.Errorf("expected %q, got %q", expected, expr.Build())
	}
}

func TestAndMultiple(t *testing.T) {
	expr := And(
		F("Age").Ge(18),
		F("Active").Eq(true),
		F("Status").Ne("Deleted"),
	)
	expected := "(Age ge 18) and (Active eq true) and (Status ne 'Deleted')"
	if expr.Build() != expected {
		t.Errorf("expected %q, got %q", expected, expr.Build())
	}
}

func TestAndEmpty(t *testing.T) {
	expr := And()
	if expr != nil {
		t.Errorf("And() with no args should return nil, got %v", expr)
	}
}

func TestOr(t *testing.T) {
	expr := Or(F("City").Eq("NY"), F("City").Eq("LA"))
	expected := "(City eq 'NY') or (City eq 'LA')"
	if expr.Build() != expected {
		t.Errorf("expected %q, got %q", expected, expr.Build())
	}
}

func TestOrMultiple(t *testing.T) {
	expr := Or(
		F("City").Eq("NY"),
		F("City").Eq("LA"),
		F("City").Eq("SF"),
	)
	expected := "(City eq 'NY') or (City eq 'LA') or (City eq 'SF')"
	if expr.Build() != expected {
		t.Errorf("expected %q, got %q", expected, expr.Build())
	}
}

func TestOrEmpty(t *testing.T) {
	expr := Or()
	if expr != nil {
		t.Errorf("Or() with no args should return nil, got %v", expr)
	}
}

func TestNot(t *testing.T) {
	expr := Not(F("Deleted").Eq(true))
	expected := "not (Deleted eq true)"
	if expr.Build() != expected {
		t.Errorf("expected %q, got %q", expected, expr.Build())
	}
}

func TestNotNil(t *testing.T) {
	expr := Not(nil)
	if expr != nil {
		t.Errorf("Not(nil) should return nil, got %v", expr)
	}
}

func TestNestedAndOr(t *testing.T) {
	expr := And(
		Or(F("City").Eq("NY"), F("City").Eq("LA")),
		F("Age").Ge(18),
	)
	expected := "((City eq 'NY') or (City eq 'LA')) and (Age ge 18)"
	if expr.Build() != expected {
		t.Errorf("expected %q, got %q", expected, expr.Build())
	}
}

func TestNestedOrAnd(t *testing.T) {
	expr := Or(
		And(F("Age").Ge(18), F("Active").Eq(true)),
		F("Status").Eq("Premium"),
	)
	expected := "((Age ge 18) and (Active eq true)) or (Status eq 'Premium')"
	if expr.Build() != expected {
		t.Errorf("expected %q, got %q", expected, expr.Build())
	}
}

func TestNotNested(t *testing.T) {
	expr := Not(And(F("Age").Lt(18), F("Parental").Ne(true)))
	expected := "not ((Age lt 18) and (Parental ne true))"
	if expr.Build() != expected {
		t.Errorf("expected %q, got %q", expected, expr.Build())
	}
}

func TestBuildAndString(t *testing.T) {
	expr := F("Name").Eq("Alice")
	if expr.Build() != expr.String() {
		t.Errorf("Build() and String() should return the same value")
	}
}

func TestStringImplementation(t *testing.T) {
	expr := F("Age").Gt(30)
	result := strings.TrimSpace(expr.String())
	if result != "Age gt 30" {
		t.Errorf("String() should work with fmt functions, got %q", result)
	}
}

func TestChaining(t *testing.T) {
	expr := F("Name").Contains("john")
	if expr == nil {
		t.Fatal("chaining should not return nil")
	}
	result := expr.Build()
	if !strings.Contains(result, "contains") {
		t.Errorf("expected 'contains' in result, got %q", result)
	}
}

func TestComplexFilter(t *testing.T) {
	expr := And(
		F("Status").Eq("Active"),
		Or(
			F("Priority").Gt(5),
			F("VIP").Eq(true),
		),
		Not(F("Archived").Eq(true)),
	)
	result := expr.Build()

	if !strings.Contains(result, "Status eq 'Active'") {
		t.Errorf("expected 'Status eq 'Active'' in result, got %q", result)
	}
	if !strings.Contains(result, "Priority gt 5") {
		t.Errorf("expected 'Priority gt 5' in result, got %q", result)
	}
	if !strings.Contains(result, "VIP eq true") {
		t.Errorf("expected 'VIP eq true' in result, got %q", result)
	}
	if !strings.Contains(result, "not") {
		t.Errorf("expected 'not' in result, got %q", result)
	}
}

// Integration tests with QueryBuilder

func TestQueryBuilderFilterBy(t *testing.T) {
	qb := &QueryBuilder{entitySet: "Users"}

	expr := F("Age").Gt(18)
	result := qb.FilterBy(expr)

	if result != qb {
		t.Errorf("FilterBy() should return the same QueryBuilder for chaining")
	}

	if qb.filterExpr != "Age gt 18" {
		t.Errorf("expected filterExpr to be 'Age gt 18', got %q", qb.filterExpr)
	}

	if !qb.urlDirty {
		t.Errorf("FilterBy() should set urlDirty to true")
	}
}

func TestQueryBuilderFilterByNil(t *testing.T) {
	qb := &QueryBuilder{entitySet: "Users"}
	result := qb.FilterBy(nil)

	if result != qb {
		t.Errorf("FilterBy(nil) should return the same QueryBuilder")
	}

	if qb.filterExpr != "" {
		t.Errorf("FilterBy(nil) should not set filterExpr, got %q", qb.filterExpr)
	}
}

func TestQueryBuilderFilterByComplex(t *testing.T) {
	qb := &QueryBuilder{entitySet: "Users"}

	expr := And(
		F("Status").Eq("Active"),
		F("Age").Ge(18),
	)
	qb.FilterBy(expr)

	expected := "(Status eq 'Active') and (Age ge 18)"
	if qb.filterExpr != expected {
		t.Errorf("expected %q, got %q", expected, qb.filterExpr)
	}
}

func TestFilterByChaining(t *testing.T) {
	qb := &QueryBuilder{entitySet: "Users"}

	expr := F("Status").Eq("Active")
	result := qb.FilterBy(expr).Top(10)

	if result != qb {
		t.Errorf("FilterBy().Top() chaining should work")
	}

	if qb.filterExpr != "Status eq 'Active'" {
		t.Errorf("expected filterExpr to be 'Status eq 'Active'', got %q", qb.filterExpr)
	}

	if qb.top == nil || *qb.top != 10 {
		t.Errorf("Top(10) should set top to 10")
	}
}

func TestStringWithQuotesInContains(t *testing.T) {
	expr := F("Description").Contains("can't do that")
	expected := "contains(Description,'can''t do that')"
	if expr.Build() != expected {
		t.Errorf("expected %q, got %q", expected, expr.Build())
	}
}

func TestNegativeNumbers(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		expected string
	}{
		{"negative int", int(-42), "Value eq -42"},
		{"negative float32", float32(-19.99), "Value eq -19.99"},
		{"negative float64", float64(-19.99), "Value eq -19.99"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr := F("Value").Eq(tt.value)
			if expr.Build() != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, expr.Build())
			}
		})
	}
}

func TestEmptyFieldName(t *testing.T) {
	expr := F("").Eq("value")
	result := expr.Build()
	if result != " eq 'value'" {
		t.Errorf("empty field name should still work, got %q", result)
	}
}

func TestSpecialCharactersInFieldName(t *testing.T) {
	expr := F("User/Name").Eq("Alice")
	expected := "User/Name eq 'Alice'"
	if expr.Build() != expected {
		t.Errorf("expected %q, got %q", expected, expr.Build())
	}
}

func TestMultipleOperationsChained(t *testing.T) {
	// This tests that you can build a single field expression
	expr := F("Name").StartsWith("Ali").Contains("ice")
	result := expr.Build()

	if !strings.Contains(result, "startswith") && !strings.Contains(result, "contains") {
		t.Errorf("expected chained functions in result, got %q", result)
	}
}
