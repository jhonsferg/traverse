package traverse

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// FilterExpr is a chainable OData filter expression builder.
//
// FilterExpr provides a fluent API for constructing type-safe OData $filter expressions
// without string concatenation. Each method returns *FilterExpr to enable method chaining,
// and the final Build() or String() method generates the OData filter string.
//
// Example:
//
//	expr := F("Name").Eq("Alice")
//	filter := expr.Build() // "Name eq 'Alice'"
type FilterExpr struct {
	expr string
}

// F starts a filter expression for a field.
//
// F creates a new FilterExpr targeting the specified field. This is the entry point
// for building type-safe OData filter expressions.
//
// Example:
//
//	F("Age").Gt(18).Build()
func F(field string) *FilterExpr {
	return &FilterExpr{expr: field}
}

// formatValue converts a value to its OData literal representation.
//
// formatValue handles type-specific formatting:
//   - string: 'value' (single quotes, with escaping)
//   - int/int32/int64: unquoted decimal
//   - float32/float64: unquoted decimal
//   - bool: true or false (lowercase)
//   - time.Time: OData datetime format
//   - nil: null
func formatValue(v interface{}) string {
	switch val := v.(type) {
	case string:
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
	case time.Time:
		// OData v2 format: datetime'2006-01-02T15:04:05Z'
		return "datetime'" + val.UTC().Format("2006-01-02T15:04:05Z") + "'"
	case nil:
		return "null"
	default:
		return fmt.Sprint(val)
	}
}

// applyOp applies a comparison operator to the current expression.
func (f *FilterExpr) applyOp(op string, value interface{}) *FilterExpr {
	f.expr = f.expr + " " + op + " " + formatValue(value)
	return f
}

// Eq creates an equality filter: field eq value.
func (f *FilterExpr) Eq(value interface{}) *FilterExpr {
	return f.applyOp("eq", value)
}

// Ne creates an inequality filter: field ne value.
func (f *FilterExpr) Ne(value interface{}) *FilterExpr {
	return f.applyOp("ne", value)
}

// Lt creates a less-than filter: field lt value.
func (f *FilterExpr) Lt(value interface{}) *FilterExpr {
	return f.applyOp("lt", value)
}

// Le creates a less-than-or-equal filter: field le value.
func (f *FilterExpr) Le(value interface{}) *FilterExpr {
	return f.applyOp("le", value)
}

// Gt creates a greater-than filter: field gt value.
func (f *FilterExpr) Gt(value interface{}) *FilterExpr {
	return f.applyOp("gt", value)
}

// Ge creates a greater-than-or-equal filter: field ge value.
func (f *FilterExpr) Ge(value interface{}) *FilterExpr {
	return f.applyOp("ge", value)
}

// Contains creates a string contains filter: contains(field, value).
//
// Contains checks if the field (string) contains the specified substring.
func (f *FilterExpr) Contains(value string) *FilterExpr {
	f.expr = "contains(" + f.expr + "," + formatValue(value) + ")"
	return f
}

// StartsWith creates a string starts-with filter: startswith(field, value).
//
// StartsWith checks if the field (string) starts with the specified prefix.
func (f *FilterExpr) StartsWith(value string) *FilterExpr {
	f.expr = "startswith(" + f.expr + "," + formatValue(value) + ")"
	return f
}

// EndsWith creates a string ends-with filter: endswith(field, value).
//
// EndsWith checks if the field (string) ends with the specified suffix.
func (f *FilterExpr) EndsWith(value string) *FilterExpr {
	f.expr = "endswith(" + f.expr + "," + formatValue(value) + ")"
	return f
}

// And combines multiple filter expressions with logical AND.
//
// And takes multiple FilterExpr arguments and combines them with the 'and' operator,
// wrapping each in parentheses. If no expressions are provided, returns nil.
//
// Example:
//
//	And(F("Age").Ge(18), F("Active").Eq(true)).Build()
//	// "(Age ge 18) and (Active eq true)"
func And(exprs ...*FilterExpr) *FilterExpr {
	if len(exprs) == 0 {
		return nil
	}
	var sb strings.Builder
	for i, expr := range exprs {
		if i > 0 {
			sb.WriteString(" and ")
		}
		sb.WriteString("(")
		sb.WriteString(expr.expr)
		sb.WriteString(")")
	}
	return &FilterExpr{expr: sb.String()}
}

// Or combines multiple filter expressions with logical OR.
//
// Or takes multiple FilterExpr arguments and combines them with the 'or' operator,
// wrapping each in parentheses. If no expressions are provided, returns nil.
//
// Example:
//
//	Or(F("City").Eq("NY"), F("City").Eq("LA")).Build()
//	// "(City eq 'NY') or (City eq 'LA')"
func Or(exprs ...*FilterExpr) *FilterExpr {
	if len(exprs) == 0 {
		return nil
	}
	var sb strings.Builder
	for i, expr := range exprs {
		if i > 0 {
			sb.WriteString(" or ")
		}
		sb.WriteString("(")
		sb.WriteString(expr.expr)
		sb.WriteString(")")
	}
	return &FilterExpr{expr: sb.String()}
}

// Not creates a logical NOT filter: not (expression).
//
// Not negates the given filter expression.
//
// Example:
//
//	Not(F("Deleted").Eq(true)).Build()
//	// "not (Deleted eq true)"
func Not(expr *FilterExpr) *FilterExpr {
	if expr == nil {
		return nil
	}
	return &FilterExpr{expr: "not (" + expr.expr + ")"}
}

// Build returns the OData filter string.
//
// Build finalizes the filter expression and returns the OData $filter string.
// This method does not modify the FilterExpr and can be called multiple times.
func (f *FilterExpr) Build() string {
	return f.expr
}

// String implements the Stringer interface, returning the OData filter string.
//
// String is equivalent to Build() and allows FilterExpr to be used with
// fmt.Print and other functions expecting Stringer.
func (f *FilterExpr) String() string {
	return f.expr
}
