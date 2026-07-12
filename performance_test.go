package traverse

import (
	"strings"
	"testing"
)

func TestFilterExpr_Reset(t *testing.T) {
	expr := F("Name").Eq("Alice")
	filter1 := expr.Build()
	if filter1 != "Name eq 'Alice'" {
		t.Errorf("expected 'Name eq Alice', got %q", filter1)
	}

	// Reset clears the expression
	expr.Reset()
	if expr.Build() != "" {
		t.Errorf("expected empty string after Reset, got %q", expr.Build())
	}
}

func TestFormatValue_StringWithoutQuote(t *testing.T) {
	result := formatValue("hello")
	if result != "'hello'" {
		t.Errorf("expected 'hello', got %q", result)
	}
}

func TestFormatValue_StringWithQuote(t *testing.T) {
	result := formatValue("it's")
	if result != "'it''s'" {
		t.Errorf("expected 'it''s', got %q", result)
	}
}

func TestFormatValue_OtherTypes(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{"int", 42, "42"},
		{"int32", int32(42), "42"},
		{"int64", int64(42), "42"},
		{"float32", float32(3.14), "3.14"},
		{"float64", 3.14, "3.14"},
		{"bool true", true, "true"},
		{"bool false", false, "false"},
		{"nil", nil, "null"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatValue(tt.input)
			if result != tt.expected {
				t.Errorf("formatValue(%v) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestAtomParser_EntryMapPreallocated(t *testing.T) {
	xml := `<?xml version="1.0" encoding="utf-8"?>
<feed xmlns="http://www.w3.org/2005/Atom"
      xmlns:d="http://schemas.microsoft.com/ado/2007/08/dataservices"
      xmlns:m="http://schemas.microsoft.com/ado/2007/08/dataservices/metadata">
  <entry>
    <content type="application/xml">
      <m:properties>
        <d:ID>1</d:ID>
        <d:Name>Test</d:Name>
      </m:properties>
    </content>
  </entry>
</feed>`
	page := &Page{}
	err := ParseAtomFeed(strings.NewReader(xml), page)
	if err != nil {
		t.Fatalf("ParseAtomFeed failed: %v", err)
	}
	if len(page.Value) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(page.Value))
	}
	if page.Value[0]["ID"] != "1" {
		t.Errorf("expected ID=1, got %v", page.Value[0]["ID"])
	}
	if page.Value[0]["Name"] != "Test" {
		t.Errorf("expected Name=Test, got %v", page.Value[0]["Name"])
	}
}
