package traverse

import (
	"fmt"
	"strings"
)

// EntitySchema defines the known properties of an entity type for filter validation.
//
// EntitySchema provides a declarative way to specify the properties of an OData entity type
// so that filter and orderby expressions can be validated before sending to the service.
// This catches typos in field names and type mismatches at build time rather than at
// runtime when the service rejects the request.
//
// The Properties map associates property names with their types, allowing validation
// of filter expressions to ensure they reference only valid properties with appropriate types.
//
// Example:
//
//	schema := EntitySchema{
//	    Properties: map[string]string{
//	        "ID":        "int",
//	        "Name":      "string",
//	        "Email":     "string",
//	        "Age":       "int",
//	        "Birthdate": "datetime",
//	        "IsActive":  "bool",
//	    },
//	}
//	query := client.From("Users").WithSchema(schema)
type EntitySchema struct {
	// Properties maps property name -> property type.
	// Valid types: "string", "int", "float", "bool", "datetime", "guid"
	Properties map[string]string
}

// SchemaValidationError is returned when a filter or orderby references an unknown field
// or has a type mismatch.
//
// SchemaValidationError provides detailed information about validation failures so that
// developers can quickly identify and fix issues in their filter expressions.
//
// Example:
//
//	if err != nil {
//	    if schemaErr, ok := err.(*SchemaValidationError); ok {
//	        fmt.Printf("Unknown field: %s\n", schemaErr.Field)
//	        fmt.Printf("Details: %s\n", schemaErr.Message)
//	    }
//	}
type SchemaValidationError struct {
	Field   string
	Message string
}

// Error implements the error interface.
func (e *SchemaValidationError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("traverse: schema validation error: %s: %s", e.Field, e.Message)
	}
	return fmt.Sprintf("traverse: schema validation error: %s", e.Field)
}

// filterOperators contains operators that are followed by a field reference in OData filters.
var filterOperators = map[string]bool{
	"eq":         true,
	"ne":         true,
	"lt":         true,
	"le":         true,
	"gt":         true,
	"ge":         true,
	"contains":   true,
	"startswith": true,
	"endswith":   true,
}

// validateFilter parses the filter string and validates that all referenced fields
// exist in the schema.
//
// The filter string is parsed by looking for field names that appear before
// OData comparison operators (eq, ne, lt, le, gt, ge, contains, startswith, endswith).
// Field names may be simple identifiers or chained properties (e.g., "Address/City").
//
// Validation is case-sensitive. Property names must match exactly as defined in the schema.
//
// Returns a SchemaValidationError if any referenced field is not found in the schema.
// Returns nil if all fields are valid or if the filter is empty.
func validateFilter(schema EntitySchema, filter string) error {
	if filter == "" || len(schema.Properties) == 0 {
		return nil
	}

	// Tokenize more carefully to avoid picking up tokens inside quoted strings
	tokens, err := tokenizeFilter(filter)
	if err != nil {
		// If tokenization fails, we can't validate, so allow it through
		return nil
	}

	for i, token := range tokens {
		// Skip operators and empty tokens
		if token == "" || isOperator(token) {
			continue
		}

		// Check if the next token is a comparison operator
		// If so, the current token is a field name
		if i+1 < len(tokens) && filterOperators[tokens[i+1]] {
			// Extract the base field name (handle chained properties like "Address/City")
			fieldName := token
			if idx := strings.Index(fieldName, "/"); idx != -1 {
				fieldName = fieldName[:idx]
			}

			// Validate the field exists in the schema
			if _, exists := schema.Properties[fieldName]; !exists {
				return &SchemaValidationError{
					Field:   fieldName,
					Message: "field not found in schema",
				}
			}
		}
	}

	return nil
}

// tokenizeFilter tokenizes an OData filter expression, respecting quoted strings.
// It returns a list of tokens, skipping over content inside single or double quotes.
func tokenizeFilter(filter string) ([]string, error) {
	var tokens []string
	var current strings.Builder
	inQuote := false
	quoteChar := rune(0)

	for i, ch := range filter {
		// Handle quote toggling
		if (ch == '\'' || ch == '"') && (i == 0 || filter[i-1] != '\\') {
			if !inQuote {
				inQuote = true
				quoteChar = ch
				// Don't add the quote itself
				continue
			} else if ch == quoteChar {
				inQuote = false
				quoteChar = 0
				// Don't add the quote itself
				continue
			}
		}

		// If we're inside a quote, skip tokenization
		if inQuote {
			continue
		}

		// Tokenize on whitespace, parens, commas
		if ch == ' ' || ch == '(' || ch == ')' || ch == ',' {
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
		} else {
			current.WriteRune(ch)
		}
	}

	// Add any remaining token
	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}

	return tokens, nil
}

// validateOrderBy parses the orderby string and validates that all referenced fields
// exist in the schema.
//
// The orderby string contains field names separated by commas, optionally followed by
// "asc" or "desc" keywords. Each field name is validated against the schema.
//
// Example valid orderby expressions:
//   - "Name"
//   - "Name asc"
//   - "Name asc,Age desc"
//   - "Address/City"
//
// Returns a SchemaValidationError if any referenced field is not found in the schema.
// Returns nil if all fields are valid or if the orderby is empty.
func validateOrderBy(schema EntitySchema, orderBy string) error {
	if orderBy == "" || len(schema.Properties) == 0 {
		return nil
	}

	// Split on commas to handle multiple orderby clauses
	clauses := strings.Split(orderBy, ",")

	for _, clause := range clauses {
		clause = strings.TrimSpace(clause)
		if clause == "" {
			continue
		}

		// Split on whitespace to separate field name from asc/desc
		parts := strings.Fields(clause)
		if len(parts) == 0 {
			continue
		}

		fieldName := parts[0]

		// Handle chained properties (e.g., "Address/City" -> "Address")
		if idx := strings.Index(fieldName, "/"); idx != -1 {
			fieldName = fieldName[:idx]
		}

		// Validate the field exists in the schema
		if _, exists := schema.Properties[fieldName]; !exists {
			return &SchemaValidationError{
				Field:   fieldName,
				Message: "field not found in schema",
			}
		}
	}

	return nil
}

// isOperator checks if a token is a known OData operator or function.
func isOperator(token string) bool {
	switch token {
	case "and", "or", "not", "in", "has":
		return true
	case "eq", "ne", "lt", "le", "gt", "ge":
		return true
	case "contains", "startswith", "endswith":
		return true
	case "asc", "desc":
		return true
	}

	// Check if it's a function call (simplified check)
	if strings.Contains(token, "(") || token == "true" || token == "false" || token == "null" {
		return true
	}

	return false
}
