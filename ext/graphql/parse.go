package graphql

import (
	"fmt"
	"strconv"
	"strings"
)

// Query represents a parsed GraphQL query for translation.
type Query struct {
	EntitySet    string              // top-level field name → OData entity set
	Fields       []string            // selected scalar fields → $select
	Filter       string              // filter argument → $filter
	OrderBy      string              // orderBy argument → $orderby
	Top          int                 // top argument → $top
	Skip         int                 // skip argument → $skip
	Expand       []string            // nested object field names → $expand
	ExpandFields map[string][]string // navigation property name → selected sub-fields
}

// Translate converts a simple GraphQL query string into OData query parameters.
// Only supports single-level queries (no mutations, no fragments, no variables).
//
// Example input:
//
//	{ customers(filter: "Country eq 'Germany'", top: 10) { id name country } }
//
// Example output:
//
//	Query{EntitySet: "customers", Fields: ["id","name","country"],
//	      Filter: "Country eq 'Germany'", Top: 10}
func Translate(query string) (*Query, error) {
	p := &gqlParser{input: strings.TrimSpace(query)}
	return p.parse()
}

// ToODataParams converts a Query to OData URL query parameters.
// Returns a map suitable for use with url.Values.
func (q *Query) ToODataParams() map[string]string {
	params := map[string]string{}

	if len(q.Fields) > 0 {
		params["$select"] = strings.Join(q.Fields, ",")
	}
	if q.Filter != "" {
		params["$filter"] = q.Filter
	}
	if q.OrderBy != "" {
		params["$orderby"] = q.OrderBy
	}
	if q.Top > 0 {
		params["$top"] = strconv.Itoa(q.Top)
	}
	if q.Skip > 0 {
		params["$skip"] = strconv.Itoa(q.Skip)
	}
	if len(q.Expand) > 0 {
		expandParts := make([]string, 0, len(q.Expand))
		for _, nav := range q.Expand {
			if subFields, ok := q.ExpandFields[nav]; ok && len(subFields) > 0 {
				expandParts = append(expandParts, fmt.Sprintf("%s($select=%s)", nav, strings.Join(subFields, ",")))
			} else {
				expandParts = append(expandParts, nav)
			}
		}
		params["$expand"] = strings.Join(expandParts, ",")
	}
	return params
}

// gqlParser is a hand-written recursive-descent parser for simple GraphQL queries.
type gqlParser struct {
	input string
	pos   int
}

func (p *gqlParser) skipWS() {
	for p.pos < len(p.input) {
		c := p.input[p.pos]
		if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
			p.pos++
		} else {
			break
		}
	}
}

func (p *gqlParser) peek() byte {
	p.skipWS()
	if p.pos >= len(p.input) {
		return 0
	}
	return p.input[p.pos]
}

func (p *gqlParser) consume(ch byte) error {
	p.skipWS()
	if p.pos >= len(p.input) {
		return fmt.Errorf("expected '%c' at position %d, got EOF", ch, p.pos)
	}
	if p.input[p.pos] != ch {
		snippet := p.input[p.pos:]
		if len(snippet) > 10 {
			snippet = snippet[:10]
		}
		return fmt.Errorf("expected '%c' at position %d, got %q", ch, p.pos, snippet)
	}
	p.pos++
	return nil
}

func (p *gqlParser) readIdent() (string, error) {
	p.skipWS()
	if p.pos >= len(p.input) {
		return "", fmt.Errorf("expected identifier at position %d, got EOF", p.pos)
	}
	c := p.input[p.pos]
	if !isAlpha(c) && c != '_' {
		return "", fmt.Errorf("expected identifier at position %d, got %q", p.pos, string(c))
	}
	start := p.pos
	for p.pos < len(p.input) && (isAlphaNum(p.input[p.pos]) || p.input[p.pos] == '_') {
		p.pos++
	}
	return p.input[start:p.pos], nil
}

func (p *gqlParser) readString() (string, error) {
	p.skipWS()
	if p.pos >= len(p.input) || p.input[p.pos] != '"' {
		return "", fmt.Errorf("expected '\"' at position %d", p.pos)
	}
	p.pos++ // skip opening quote
	start := p.pos
	for p.pos < len(p.input) && p.input[p.pos] != '"' {
		if p.input[p.pos] == '\\' {
			p.pos++ // skip escape character
		}
		p.pos++
	}
	if p.pos >= len(p.input) {
		return "", fmt.Errorf("unterminated string starting at position %d", start-1)
	}
	val := p.input[start:p.pos]
	p.pos++ // skip closing quote
	return val, nil
}

func (p *gqlParser) readInt() (int, error) {
	p.skipWS()
	start := p.pos
	if p.pos < len(p.input) && p.input[p.pos] == '-' {
		p.pos++
	}
	for p.pos < len(p.input) && p.input[p.pos] >= '0' && p.input[p.pos] <= '9' {
		p.pos++
	}
	if start == p.pos || (start == p.pos-1 && p.input[start] == '-') {
		return 0, fmt.Errorf("expected integer at position %d", p.pos)
	}
	return strconv.Atoi(p.input[start:p.pos])
}

func (p *gqlParser) parse() (*Query, error) {
	if p.input == "" {
		return nil, fmt.Errorf("empty query")
	}

	if err := p.consume('{'); err != nil {
		return nil, fmt.Errorf("invalid query: %w", err)
	}

	entitySet, err := p.readIdent()
	if err != nil {
		return nil, fmt.Errorf("invalid query: expected entity set name: %w", err)
	}

	q := &Query{
		EntitySet:    entitySet,
		ExpandFields: make(map[string][]string),
	}

	// Optional argument list
	if p.peek() == '(' {
		p.pos++ // consume '('
		if parseErr := p.parseArgs(q); parseErr != nil {
			return nil, parseErr
		}
		if consumeErr := p.consume(')'); consumeErr != nil {
			return nil, fmt.Errorf("invalid query: %w", consumeErr)
		}
	}

	// Field selection block
	if blockErr := p.consume('{'); blockErr != nil {
		return nil, fmt.Errorf("invalid query: expected field selection block: %w", blockErr)
	}

	fields, expand, expandFields, err := p.parseFields()
	if err != nil {
		return nil, err
	}

	if err := p.consume('}'); err != nil {
		return nil, fmt.Errorf("invalid query: expected '}' closing field block: %w", err)
	}
	if err := p.consume('}'); err != nil {
		return nil, fmt.Errorf("invalid query: expected '}' closing query: %w", err)
	}

	if len(fields) == 0 && len(expand) == 0 {
		return nil, fmt.Errorf("invalid query: no fields selected")
	}

	q.Fields = fields
	q.Expand = expand
	for k, v := range expandFields {
		q.ExpandFields[k] = v
	}

	return q, nil
}

func (p *gqlParser) parseArgs(q *Query) error {
	for {
		p.skipWS()
		if p.peek() == ')' {
			break
		}

		key, err := p.readIdent()
		if err != nil {
			return fmt.Errorf("invalid argument name: %w", err)
		}

		if err := p.consume(':'); err != nil {
			return fmt.Errorf("argument %q: expected ':': %w", key, err)
		}

		switch key {
		case "filter":
			val, err := p.readString()
			if err != nil {
				return fmt.Errorf("argument %q: %w", key, err)
			}
			q.Filter = val
		case "orderBy":
			val, err := p.readString()
			if err != nil {
				return fmt.Errorf("argument %q: %w", key, err)
			}
			q.OrderBy = val
		case "top":
			val, err := p.readInt()
			if err != nil {
				return fmt.Errorf("argument %q: %w", key, err)
			}
			q.Top = val
		case "skip":
			val, err := p.readInt()
			if err != nil {
				return fmt.Errorf("argument %q: %w", key, err)
			}
			q.Skip = val
		default:
			// Skip unknown arguments (string or integer value)
			p.skipWS()
			if p.pos < len(p.input) && p.input[p.pos] == '"' {
				if _, err := p.readString(); err != nil {
					return fmt.Errorf("argument %q: %w", key, err)
				}
			} else {
				if _, err := p.readInt(); err != nil {
					return fmt.Errorf("argument %q: expected string or integer: %w", key, err)
				}
			}
		}

		p.skipWS()
		if p.peek() == ',' {
			p.pos++ // consume ','
		}
	}
	return nil
}

// parseFields reads a list of field names and nested selection sets.
// Returns scalar fields, expand navigation property names, and their sub-fields.
func (p *gqlParser) parseFields() ([]string, []string, map[string][]string, error) {
	var fields []string
	var expand []string
	expandFields := make(map[string][]string)

	for {
		p.skipWS()
		if p.peek() == '}' || p.pos >= len(p.input) {
			break
		}

		name, readErr := p.readIdent()
		if readErr != nil {
			return nil, nil, nil, readErr
		}

		p.skipWS()
		if p.peek() == '{' {
			p.pos++ // consume '{'
			subFields, _, _, parseErr := p.parseFields()
			if parseErr != nil {
				return nil, nil, nil, parseErr
			}
			if closeErr := p.consume('}'); closeErr != nil {
				return nil, nil, nil, fmt.Errorf("expected '}' closing nested field %q: %w", name, closeErr)
			}
			expand = append(expand, name)
			if len(subFields) > 0 {
				expandFields[name] = subFields
			}
		} else {
			fields = append(fields, name)
		}
	}
	return fields, expand, expandFields, nil
}

func isAlpha(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func isAlphaNum(c byte) bool {
	return isAlpha(c) || (c >= '0' && c <= '9')
}
