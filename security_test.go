package traverse

import (
	"strings"
	"testing"
)

func TestSanitizeHeaderValue_RemovesCRLF(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"clean value", "abc", "abc"},
		{"carriage return", "abc\rdef", "abcdef"},
		{"line feed", "abc\ndef", "abcdef"},
		{"crlf pair", "abc\r\ndef", "abcdef"},
		{"multiple injections", "a\r\nb\r\nc", "abc"},
		{"only cr", "\r", ""},
		{"only lf", "\n", ""},
		{"empty", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeHeaderValue(tt.input)
			if got != tt.expected {
				t.Errorf("sanitizeHeaderValue(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestSearchPhrase_EscapesDoubleQuotes(t *testing.T) {
	tests := []struct {
		name     string
		phrase   string
		expected string
	}{
		{"no quotes", "hello", `"hello"`},
		{"embedded quote", `say "hi"`, `"say \"hi\""`},
		{"double embedded", `a"b"c`, `"a\"b\"c"`},
		{"only quotes", `"`, `"` + `\"` + `"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr := SearchPhrase(tt.phrase)
			got := expr.searchString()
			if got != tt.expected {
				t.Errorf("SearchPhrase(%q).searchString() = %q, want %q", tt.phrase, got, tt.expected)
			}
		})
	}
}

func TestValidateFieldName(t *testing.T) {
	tests := []struct {
		name    string
		field   string
		wantErr bool
	}{
		{"valid simple", "Name", false},
		{"valid with underscore", "First_Name", false},
		{"valid with dot", "Address.City", false},
		{"valid with slash", "User/Name", false},
		{"valid alphanumeric", "Field123", false},
		{"empty field", "", false},
		{"reserved keyword and", "and", true},
		{"reserved keyword or", "or", true},
		{"reserved keyword not", "not", true},
		{"reserved keyword eq", "eq", true},
		{"reserved keyword null", "null", true},
		{"space injection", "Name eq", true},
		{"paren injection", "Name()", true},
		{"quote injection", `Name"`, true},
		{"semicolon injection", "Name;", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateFieldName(tt.field)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateFieldName(%q) error = %v, wantErr %v", tt.field, err, tt.wantErr)
			}
		})
	}
}

func TestFilterExpr_FieldValidation(t *testing.T) {
	expr := F("Name").Eq("Alice")
	if !strings.Contains(expr.Build(), "Name") {
		t.Errorf("expected valid filter, got %s", expr.Build())
	}

	badExpr := F("or")
	if badExpr.Build() != "INVALID_FIELD" {
		t.Errorf("expected INVALID_FIELD for reserved keyword, got %s", badExpr.Build())
	}
}
