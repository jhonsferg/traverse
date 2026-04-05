package traverse

import (
	"fmt"
	"strings"
)

// LambdaBuilder constructs the expression inside an OData lambda operator (any/all).
//
// LambdaBuilder is obtained from [QueryBuilder.LambdaAny] or [QueryBuilder.LambdaAll]
// via a callback function. Use Field() to start building conditions within the lambda.
//
// The variable name used in the lambda expression is automatically derived from the
// first letter of the collection field name (e.g. "tags" -> "t", "items" -> "i").
type LambdaBuilder struct {
	varName string
	exprs   []string
}

// Field begins a condition expression for the given field within the lambda.
// The field is referenced as varName/fieldName in the generated OData expression.
//
//	b.Field("name").Eq("admin")
//	// produces: t/name eq 'admin'  (when varName is "t")
func (b *LambdaBuilder) Field(name string) *LambdaCondition {
	return &LambdaCondition{
		builder: b,
		path:    b.varName + "/" + name,
	}
}

// build returns the combined lambda body expression, joining multiple conditions with " and ".
func (b *LambdaBuilder) build() string {
	return strings.Join(b.exprs, " and ")
}

// LambdaCondition builds a comparison or function expression within a lambda.
//
// LambdaCondition is obtained from [LambdaBuilder.Field] and provides comparison
// and string function operators. Each operator appends an expression to the parent
// LambdaBuilder and returns the LambdaBuilder for further chaining.
type LambdaCondition struct {
	builder *LambdaBuilder
	path    string
}

func (c *LambdaCondition) addExpr(expr string) *LambdaBuilder {
	c.builder.exprs = append(c.builder.exprs, expr)
	return c.builder
}

// Eq adds an equality condition: path eq value.
func (c *LambdaCondition) Eq(v any) *LambdaBuilder {
	return c.addExpr(fmt.Sprintf("%s eq %s", c.path, serializeValue(v)))
}

// Ne adds a not-equal condition: path ne value.
func (c *LambdaCondition) Ne(v any) *LambdaBuilder {
	return c.addExpr(fmt.Sprintf("%s ne %s", c.path, serializeValue(v)))
}

// Lt adds a less-than condition: path lt value.
func (c *LambdaCondition) Lt(v any) *LambdaBuilder {
	return c.addExpr(fmt.Sprintf("%s lt %s", c.path, serializeValue(v)))
}

// Le adds a less-than-or-equal condition: path le value.
func (c *LambdaCondition) Le(v any) *LambdaBuilder {
	return c.addExpr(fmt.Sprintf("%s le %s", c.path, serializeValue(v)))
}

// Gt adds a greater-than condition: path gt value.
func (c *LambdaCondition) Gt(v any) *LambdaBuilder {
	return c.addExpr(fmt.Sprintf("%s gt %s", c.path, serializeValue(v)))
}

// Ge adds a greater-than-or-equal condition: path ge value.
func (c *LambdaCondition) Ge(v any) *LambdaBuilder {
	return c.addExpr(fmt.Sprintf("%s ge %s", c.path, serializeValue(v)))
}

// Contains adds a contains() function condition.
func (c *LambdaCondition) Contains(s string) *LambdaBuilder {
	return c.addExpr(fmt.Sprintf("contains(%s, %s)", c.path, serializeValue(s)))
}

// StartsWith adds a startswith() function condition.
func (c *LambdaCondition) StartsWith(s string) *LambdaBuilder {
	return c.addExpr(fmt.Sprintf("startswith(%s, %s)", c.path, serializeValue(s)))
}

// EndsWith adds an endswith() function condition.
func (c *LambdaCondition) EndsWith(s string) *LambdaBuilder {
	return c.addExpr(fmt.Sprintf("endswith(%s, %s)", c.path, serializeValue(s)))
}

// lambdaVarName derives a short variable name from a collection field.
// "tags" -> "t", "items" -> "i", "orderItems" -> "o".
func lambdaVarName(collectionField string) string {
	// Strip any navigation path prefix (e.g. "Order/Tags" -> use "Tags")
	if idx := strings.LastIndex(collectionField, "/"); idx >= 0 {
		collectionField = collectionField[idx+1:]
	}
	if len(collectionField) == 0 {
		return "x"
	}
	return strings.ToLower(string(collectionField[0]))
}

// appendFilter combines a new expression with any existing filter using "and".
func (q *QueryBuilder) appendFilter(expr string) *QueryBuilder {
	if q.filterExpr == "" {
		q.filterExpr = expr
	} else {
		q.filterExpr = q.filterExpr + " and " + expr
	}
	q.urlDirty = true
	return q
}

// FilterLambda appends a raw lambda expression string to the query filter.
// Use this for complex lambda expressions that cannot be expressed through
// the typed LambdaAny/LambdaAll builders.
//
// The expression is combined with any existing filter using "and".
//
//	query.FilterLambda("tags/any(t: t/name eq 'admin')")
func (q *QueryBuilder) FilterLambda(expression string) *QueryBuilder {
	return q.appendFilter(expression)
}

// LambdaAny appends an OData "any" lambda filter to the query.
// The fn callback receives a LambdaBuilder; use Field() to build conditions.
//
// Generated OData: collectionField/any(v: <expression>)
//
// The result is combined with any existing filter using "and".
//
//	query.LambdaAny("Tags", func(b *LambdaBuilder) {
//	    b.Field("Name").Eq("admin")
//	})
//	// $filter=Tags/any(t: t/Name eq 'admin')
func (q *QueryBuilder) LambdaAny(collectionField string, fn func(*LambdaBuilder)) *QueryBuilder {
	varName := lambdaVarName(collectionField)
	lb := &LambdaBuilder{varName: varName}
	fn(lb)
	expr := fmt.Sprintf("%s/any(%s: %s)", collectionField, varName, lb.build())
	return q.appendFilter(expr)
}

// LambdaAll appends an OData "all" lambda filter to the query.
// The fn callback receives a LambdaBuilder; use Field() to build conditions.
//
// Generated OData: collectionField/all(v: <expression>)
//
// The result is combined with any existing filter using "and".
//
//	query.LambdaAll("Items", func(b *LambdaBuilder) {
//	    b.Field("Price").Gt(100)
//	})
//	// $filter=Items/all(i: i/Price gt 100)
func (q *QueryBuilder) LambdaAll(collectionField string, fn func(*LambdaBuilder)) *QueryBuilder {
	varName := lambdaVarName(collectionField)
	lb := &LambdaBuilder{varName: varName}
	fn(lb)
	expr := fmt.Sprintf("%s/all(%s: %s)", collectionField, varName, lb.build())
	return q.appendFilter(expr)
}
