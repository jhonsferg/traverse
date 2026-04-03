// Package parser provides OData filter expression parsing and serialization.
package parser

import (
	"fmt"
	"strings"
	"time"
)

// SerializeValue converts a Go value to an OData filter literal string representation.
//
// SerializeValue takes a Go value and its corresponding OData Edm type, and converts it
// to a properly formatted OData literal. It handles all standard OData type conversions
// including string escaping, numeric formatting, boolean representation, and datetime
// handling according to OData v2 and v4 specifications.
//
// Supported types:
//   - string: Escaped with single quotes; internal quotes doubled: 'text”s value'
//   - int, int32, int64: Numeric literals without quotes: 42
//   - float32, float64: Decimal (with M suffix) or Double: 3.14M or 3.14
//   - bool: Boolean literals: true or false
//   - time.Time: DateTime or DateTimeOffset format depending on edmType
//   - nil: OData null literal
//
// The edmType parameter controls the representation for types with multiple formats
// (e.g., Edm.Decimal vs Edm.Double for floats, Edm.DateTime vs Edm.DateTimeOffset).
//
// Returns an error if the value type is not supported for OData serialization.
//
// Examples:
//
//	SerializeValue("John's", "") → "'John''s'", nil
//	SerializeValue(42, "Edm.Int32") → "42", nil
//	SerializeValue(3.14, "Edm.Decimal") → "3.14M", nil
//	SerializeValue(time.Now(), "Edm.DateTime") → "datetime'2024-01-01T00:00:00'", nil
func SerializeValue(v interface{}, edmType string) (string, error) {
	switch val := v.(type) {
	case string:
		// String literal: 'value with ''escaped quotes'''
		escaped := strings.ReplaceAll(val, "'", "''")
		return fmt.Sprintf("'%s'", escaped), nil

	case int, int32, int64:
		// Integer: 42
		return fmt.Sprint(val), nil

	case float32, float64:
		// Decimal/Double: 3.14 or 3.14M (Decimal)
		if edmType == "Edm.Decimal" {
			return fmt.Sprintf("%vM", val), nil
		}
		return fmt.Sprint(val), nil

	case bool:
		// Boolean: true / false
		if val {
			return "true", nil
		}
		return "false", nil

	case time.Time:
		// DateTime: datetime'2024-01-01T00:00:00'
		// DateTimeOffset: 2024-01-01T00:00:00Z
		switch edmType {
		case "Edm.DateTime":
			return fmt.Sprintf("datetime'%s'", val.Format("2006-01-02T15:04:05")), nil
		case "Edm.DateTimeOffset":
			return val.Format(time.RFC3339), nil
		default:
			return fmt.Sprintf("datetime'%s'", val.Format("2006-01-02T15:04:05")), nil
		}

	case nil:
		// Null value
		return "null", nil

	default:
		return "", fmt.Errorf("unsupported type for OData filter: %T", v)
	}
}

// EncodeKey encodes a single key value for use in an OData key predicate.
//
// EncodeKey converts a Go value to its OData key representation, which can be used
// to construct entity identifiers (e.g., Product(123) or Material('MAT001')).
//
// Type-specific formatting:
//   - string: Escaped and quoted: 'value'
//   - Numeric types: No quotes, decimal format as-is
//   - bool: Lowercase true/false
//   - time.Time: OData datetime format: datetime'YYYY-MM-DDTHH:MM:SS'
//
// String values are escaped by doubling single quotes per OData specification.
//
// Returns an error if the key type is unsupported or cannot be safely encoded.
//
// Example:
//
//	EncodeKey("MAT001") → "'MAT001'", nil
//	EncodeKey(42) → "42", nil
//	EncodeKey("John's") → "'John''s'", nil
func EncodeKey(key interface{}) (string, error) {
	switch v := key.(type) {
	case string:
		// String keys need single quotes and escaping
		escaped := strings.ReplaceAll(v, "'", "''")
		return fmt.Sprintf("'%s'", escaped), nil

	case int, int32, int64, float32, float64:
		// Numeric keys don't need quotes
		return fmt.Sprint(v), nil

	case bool:
		// Boolean keys
		if v {
			return "true", nil
		}
		return "false", nil

	case time.Time:
		// DateTime keys: datetime'2024-01-01T00:00:00'
		return fmt.Sprintf("datetime'%s'", v.Format("2006-01-02T15:04:05")), nil

	default:
		return "", fmt.Errorf("unsupported key type: %T", v)
	}
}

// EncodeCompositeKey encodes a composite (multi-part) key for OData URL construction.
//
// EncodeCompositeKey takes a map of key field names to values and produces a comma-separated
// key predicate string suitable for OData URLs. Each key component is individually encoded
// using EncodeKey, and then formatted as "name=value" pairs joined by commas.
//
// Format: Material='MAT001',Plant='1000',Year=2024
//
// Important limitations:
//   - Map iteration order is non-deterministic in Go, but OData key order matters for SAP systems.
//   - For production use with ordered keys, consider passing an ordered structure or key name slice.
//   - All values are encoded individually; names should be valid OData property names.
//
// Returns an error if:
//   - The keys map is empty
//   - Any key value cannot be encoded
//
// Example:
//
//	keys := map[string]interface{}{"Material": "MAT001", "Plant": "1000"}
//	result, err := EncodeCompositeKey(keys)
//	// result might be: "Material='MAT001',Plant='1000'" or "Plant='1000',Material='MAT001'"
func EncodeCompositeKey(keys map[string]interface{}) (string, error) {
	if len(keys) == 0 {
		return "", fmt.Errorf("composite key cannot be empty")
	}

	parts := make([]string, 0, len(keys))

	// Note: map iteration is random in Go, but OData key order matters
	// In production, should use ordered map or accept a slice of tuples
	for name, value := range keys {
		encoded, err := EncodeKey(value)
		if err != nil {
			return "", err
		}
		parts = append(parts, fmt.Sprintf("%s=%s", name, encoded))
	}

	return strings.Join(parts, ","), nil
}

// EscapeFilterExpression escapes special characters in OData filter expressions.
//
// EscapeFilterExpression applies OData-specific escaping rules to filter expression strings.
// The primary escaping rule in OData filters is: single quotes (') are escaped by doubling (”)
// to distinguish them from string literal delimiters.
//
// This function should be called on string values being embedded in filter expressions
// to prevent injection or syntax errors.
//
// Example:
//
//	input: "John's Company"
//	output: "John''s Company"
//
// Note: This only escapes quotes. URL encoding (percent-encoding) is handled separately
// by the URL builder when the complete filter expression is added to the query string.
func EscapeFilterExpression(expr string) string {
	// OData filter expressions have limited escaping needs
	// Main rule: single quotes are escaped by doubling: ' -> ''
	return strings.ReplaceAll(expr, "'", "''")
}

// ParseFilterExpression performs basic validation of an OData filter expression syntax.
//
// ParseFilterExpression checks the structural validity of a filter expression by verifying:
//  1. Balanced parentheses: every '(' has a matching ')'
//  2. Balanced quotes: every unescaped quote ends a quoted string
//  3. Proper quote escaping: single quotes within strings are doubled (”)
//
// This is a lightweight validation suitable for catching obvious syntax errors.
// It does NOT validate OData operator names, function calls, or semantic correctness—
// those are validated by the OData service itself.
//
// Returns nil if the expression is structurally valid, or an error describing
// the first detected syntax issue:
//   - "unbalanced parentheses in filter"
//   - "unclosed quote in filter expression"
//
// Examples:
//
//	"(Price gt 100)" → nil (valid)
//	"(Price gt 100" → error (unbalanced parentheses)
//	"'John''s Company'" → nil (valid, with escaped quote)
//	"'John's Company'" → error (unclosed quote)
func ParseFilterExpression(expr string) error {
	// Basic validation: check for balanced parentheses and quotes
	parenCount := 0
	quoteOpen := false

	for i := 0; i < len(expr); i++ {
		ch := expr[i]

		if ch == '\'' {
			if i+1 < len(expr) && expr[i+1] == '\'' {
				// Escaped quote ''
				i++
			} else {
				quoteOpen = !quoteOpen
			}
		} else if !quoteOpen {
			if ch == '(' {
				parenCount++
			} else if ch == ')' {
				parenCount--
				if parenCount < 0 {
					return fmt.Errorf("unbalanced parentheses in filter")
				}
			}
		}
	}

	if quoteOpen {
		return fmt.Errorf("unclosed quote in filter expression")
	}

	if parenCount != 0 {
		return fmt.Errorf("unbalanced parentheses in filter")
	}

	return nil
}
