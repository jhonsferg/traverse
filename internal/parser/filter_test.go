package parser

import (
	"strings"
	"testing"
	"time"
)

// ── SerializeValue ──────────────────────────────────────────────────────────

func TestSerializeValue_String(t *testing.T) {
	got, err := SerializeValue("hello", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "'hello'" {
		t.Errorf("got %q, want %q", got, "'hello'")
	}
}

func TestSerializeValue_StringWithQuote(t *testing.T) {
	got, err := SerializeValue("John's", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "'John''s'" {
		t.Errorf("got %q, want %q", got, "'John''s'")
	}
}

func TestSerializeValue_Int(t *testing.T) {
	for _, tc := range []struct {
		val  interface{}
		want string
	}{
		{42, "42"},
		{int32(100), "100"},
		{int64(-7), "-7"},
	} {
		got, err := SerializeValue(tc.val, "")
		if err != nil {
			t.Fatalf("SerializeValue(%v): unexpected error: %v", tc.val, err)
		}
		if got != tc.want {
			t.Errorf("SerializeValue(%v) = %q, want %q", tc.val, got, tc.want)
		}
	}
}

func TestSerializeValue_Float_Decimal(t *testing.T) {
	got, err := SerializeValue(float64(3.14), "Edm.Decimal")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasSuffix(got, "M") {
		t.Errorf("got %q, expected decimal suffix M", got)
	}
}

func TestSerializeValue_Float32_Decimal(t *testing.T) {
	got, err := SerializeValue(float32(2.5), "Edm.Decimal")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasSuffix(got, "M") {
		t.Errorf("got %q, expected decimal suffix M", got)
	}
}

func TestSerializeValue_Float_Double(t *testing.T) {
	got, err := SerializeValue(float64(3.14), "Edm.Double")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.HasSuffix(got, "M") {
		t.Errorf("got %q, double should not have M suffix", got)
	}
}

func TestSerializeValue_Bool_True(t *testing.T) {
	got, err := SerializeValue(true, "")
	if err != nil || got != "true" {
		t.Errorf("got %q err %v, want true nil", got, err)
	}
}

func TestSerializeValue_Bool_False(t *testing.T) {
	got, err := SerializeValue(false, "")
	if err != nil || got != "false" {
		t.Errorf("got %q err %v, want false nil", got, err)
	}
}

func TestSerializeValue_DateTime(t *testing.T) {
	ts := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	got, err := SerializeValue(ts, "Edm.DateTime")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasPrefix(got, "datetime'") {
		t.Errorf("got %q, expected datetime' prefix", got)
	}
}

func TestSerializeValue_DateTimeOffset(t *testing.T) {
	ts := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	got, err := SerializeValue(ts, "Edm.DateTimeOffset")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(got, "2024-01-15") {
		t.Errorf("got %q, expected ISO date", got)
	}
}

func TestSerializeValue_DateTime_DefaultEdm(t *testing.T) {
	ts := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	got, err := SerializeValue(ts, "Edm.Unknown")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasPrefix(got, "datetime'") {
		t.Errorf("got %q, expected datetime fallback", got)
	}
}

func TestSerializeValue_Nil(t *testing.T) {
	got, err := SerializeValue(nil, "")
	if err != nil || got != "null" {
		t.Errorf("got %q err %v, want null nil", got, err)
	}
}

func TestSerializeValue_UnsupportedType(t *testing.T) {
	_, err := SerializeValue([]int{1, 2, 3}, "")
	if err == nil {
		t.Error("expected error for unsupported type, got nil")
	}
}

// ── EncodeKey ───────────────────────────────────────────────────────────────

func TestEncodeKey_String(t *testing.T) {
	got, err := EncodeKey("MAT001")
	if err != nil || got != "'MAT001'" {
		t.Errorf("got %q err %v, want 'MAT001' nil", got, err)
	}
}

func TestEncodeKey_StringWithQuote(t *testing.T) {
	got, err := EncodeKey("John's")
	if err != nil || got != "'John''s'" {
		t.Errorf("got %q err %v, want 'John''s' nil", got, err)
	}
}

func TestEncodeKey_Int(t *testing.T) {
	got, err := EncodeKey(42)
	if err != nil || got != "42" {
		t.Errorf("got %q err %v, want 42 nil", got, err)
	}
}

func TestEncodeKey_Int32(t *testing.T) {
	got, err := EncodeKey(int32(99))
	if err != nil || got != "99" {
		t.Errorf("got %q err %v, want 99 nil", got, err)
	}
}

func TestEncodeKey_Int64(t *testing.T) {
	got, err := EncodeKey(int64(12345))
	if err != nil || got != "12345" {
		t.Errorf("got %q err %v", got, err)
	}
}

func TestEncodeKey_Float(t *testing.T) {
	got, err := EncodeKey(float64(3.14))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(got, "3.14") {
		t.Errorf("got %q, expected to contain 3.14", got)
	}
}

func TestEncodeKey_Float32(t *testing.T) {
	got, err := EncodeKey(float32(1.5))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == "" {
		t.Error("expected non-empty result")
	}
}

func TestEncodeKey_BoolTrue(t *testing.T) {
	got, err := EncodeKey(true)
	if err != nil || got != "true" {
		t.Errorf("got %q err %v, want true nil", got, err)
	}
}

func TestEncodeKey_BoolFalse(t *testing.T) {
	got, err := EncodeKey(false)
	if err != nil || got != "false" {
		t.Errorf("got %q err %v, want false nil", got, err)
	}
}

func TestEncodeKey_Time(t *testing.T) {
	ts := time.Date(2024, 3, 15, 9, 0, 0, 0, time.UTC)
	got, err := EncodeKey(ts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasPrefix(got, "datetime'") {
		t.Errorf("got %q, expected datetime' prefix", got)
	}
}

func TestEncodeKey_UnsupportedType(t *testing.T) {
	_, err := EncodeKey(struct{ Name string }{Name: "x"})
	if err == nil {
		t.Error("expected error for unsupported type, got nil")
	}
}

// ── EncodeCompositeKey ──────────────────────────────────────────────────────

func TestEncodeCompositeKey_SingleKey(t *testing.T) {
	keys := map[string]interface{}{"ID": 42}
	got, err := EncodeCompositeKey(keys)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(got, "ID=42") {
		t.Errorf("got %q, expected ID=42", got)
	}
}

func TestEncodeCompositeKey_MultipleKeys(t *testing.T) {
	keys := map[string]interface{}{
		"Company":  "ABC",
		"OrderNum": 100,
	}
	got, err := EncodeCompositeKey(keys)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(got, "Company='ABC'") {
		t.Errorf("got %q, expected Company='ABC'", got)
	}
	if !strings.Contains(got, "OrderNum=100") {
		t.Errorf("got %q, expected OrderNum=100", got)
	}
}

func TestEncodeCompositeKey_Empty(t *testing.T) {
	_, err := EncodeCompositeKey(map[string]interface{}{})
	if err == nil {
		t.Error("expected error for empty keys, got nil")
	}
}

func TestEncodeCompositeKey_InvalidValue(t *testing.T) {
	_, err := EncodeCompositeKey(map[string]interface{}{
		"Key": []string{"bad"},
	})
	if err == nil {
		t.Error("expected error for invalid value type, got nil")
	}
}

// ── EscapeFilterExpression ──────────────────────────────────────────────────

func TestEscapeFilterExpression_NoQuotes(t *testing.T) {
	got := EscapeFilterExpression("Price gt 100")
	if got != "Price gt 100" {
		t.Errorf("got %q, want unchanged", got)
	}
}

func TestEscapeFilterExpression_SingleQuote(t *testing.T) {
	got := EscapeFilterExpression("John's")
	if got != "John''s" {
		t.Errorf("got %q, want John''s", got)
	}
}

func TestEscapeFilterExpression_MultipleQuotes(t *testing.T) {
	got := EscapeFilterExpression("a'b'c")
	if got != "a''b''c" {
		t.Errorf("got %q, want a''b''c", got)
	}
}

func TestEscapeFilterExpression_Empty(t *testing.T) {
	if EscapeFilterExpression("") != "" {
		t.Error("empty string should return empty string")
	}
}

// ── ParseFilterExpression ───────────────────────────────────────────────────

func TestParseFilterExpression_Valid(t *testing.T) {
	cases := []string{
		"Price gt 100",
		"(Price gt 100)",
		"(Price gt 100 and Status eq 'Active')",
		"Name eq 'John''s Company'",
		"",
	}
	for _, expr := range cases {
		if err := ParseFilterExpression(expr); err != nil {
			t.Errorf("ParseFilterExpression(%q) = %v, want nil", expr, err)
		}
	}
}

func TestParseFilterExpression_UnbalancedParen_Open(t *testing.T) {
	err := ParseFilterExpression("(Price gt 100")
	if err == nil {
		t.Error("expected error for unbalanced open paren, got nil")
	}
}

func TestParseFilterExpression_UnbalancedParen_Close(t *testing.T) {
	err := ParseFilterExpression("Price gt 100)")
	if err == nil {
		t.Error("expected error for unbalanced close paren, got nil")
	}
}

func TestParseFilterExpression_UnclosedQuote(t *testing.T) {
	err := ParseFilterExpression("Name eq 'unclosed")
	if err == nil {
		t.Error("expected error for unclosed quote, got nil")
	}
}

func TestParseFilterExpression_EscapedQuoteValid(t *testing.T) {
	// John''s → valid escaped quote
	err := ParseFilterExpression("Name eq 'John''s'")
	if err != nil {
		t.Errorf("valid escaped quote should not error: %v", err)
	}
}

func TestParseFilterExpression_ParenInsideQuote(t *testing.T) {
	// Parens inside quotes should not count
	err := ParseFilterExpression("Name eq 'a(b)c'")
	if err != nil {
		t.Errorf("parens inside quotes should be ignored: %v", err)
	}
}
