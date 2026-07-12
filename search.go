package traverse

import "strings"

// SearchExpression represents an OData $search expression.
// OData $search is distinct from $filter  -  it performs full-text search
// across all searchable properties of an entity.
//
// Example:
//
//	results, err := client.From("Products").
//	    Search(SearchAnd(SearchWord("mountain"), SearchNot(SearchWord("bike")))).
//	    Collect(ctx)
type SearchExpression interface {
	searchString() string
}

// searchTerm is a single word or quoted phrase search term.
type searchTerm struct {
	value  string
	phrase bool // true → wrap in double quotes
}

func (s searchTerm) searchString() string {
	if s.phrase {
		escaped := strings.ReplaceAll(s.value, `"`, `\"`)
		return `"` + escaped + `"`
	}
	return s.value
}

// SearchWord creates a single-word OData $search term.
//
// The word is emitted unquoted. Use SearchPhrase for multi-word expressions.
//
// Example:
//
//	SearchWord("mountain").searchString() // "mountain"
func SearchWord(word string) SearchExpression {
	return searchTerm{value: word, phrase: false}
}

// SearchPhrase creates a quoted-phrase OData $search term.
//
// The phrase is wrapped in double quotes so the OData service treats it as
// a literal phrase rather than individual words.
//
// Example:
//
//	SearchPhrase("mountain bike").searchString() // `"mountain bike"`
func SearchPhrase(phrase string) SearchExpression {
	return searchTerm{value: phrase, phrase: true}
}

// Search is a convenience constructor that auto-detects whether the input
// should be a word (no spaces) or a phrase (contains spaces).
//
// Example:
//
//	Search("mountain")       // same as SearchWord("mountain")
//	Search("mountain bike")  // same as SearchPhrase("mountain bike")
func Search(s string) SearchExpression {
	if strings.ContainsAny(s, " \t") {
		return SearchPhrase(s)
	}
	return SearchWord(s)
}

// searchAndExpr combines expressions with OData AND.
type searchAndExpr struct {
	exprs []SearchExpression
}

func (s searchAndExpr) searchString() string {
	if len(s.exprs) == 1 {
		return s.exprs[0].searchString()
	}
	parts := make([]string, len(s.exprs))
	for i, e := range s.exprs {
		parts[i] = e.searchString()
	}
	return "(" + strings.Join(parts, " AND ") + ")"
}

// SearchAnd combines two or more search expressions with logical AND.
//
// Example:
//
//	SearchAnd(SearchWord("mountain"), SearchWord("view"))
//	// "(mountain AND view)"
func SearchAnd(exprs ...SearchExpression) SearchExpression {
	return searchAndExpr{exprs: exprs}
}

// searchOrExpr combines expressions with OData OR.
type searchOrExpr struct {
	exprs []SearchExpression
}

func (s searchOrExpr) searchString() string {
	if len(s.exprs) == 1 {
		return s.exprs[0].searchString()
	}
	parts := make([]string, len(s.exprs))
	for i, e := range s.exprs {
		parts[i] = e.searchString()
	}
	return "(" + strings.Join(parts, " OR ") + ")"
}

// SearchOr combines two or more search expressions with logical OR.
//
// Example:
//
//	SearchOr(SearchWord("mountain"), SearchWord("hill"))
//	// "(mountain OR hill)"
func SearchOr(exprs ...SearchExpression) SearchExpression {
	return searchOrExpr{exprs: exprs}
}

// searchNotExpr negates a search expression.
type searchNotExpr struct {
	expr SearchExpression
}

func (s searchNotExpr) searchString() string {
	return "NOT " + s.expr.searchString()
}

// SearchNot negates a search expression.
//
// Example:
//
//	SearchNot(SearchWord("bike")).searchString() // "NOT bike"
func SearchNot(expr SearchExpression) SearchExpression {
	return searchNotExpr{expr: expr}
}
