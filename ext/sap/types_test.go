package sap

import "testing"

func TestSAPError_String(t *testing.T) {
	var e SAPError
	e.Error.Message.Value = "entity not found"

	if got := e.String(); got != "entity not found" {
		t.Errorf("String() = %q, want %q", got, "entity not found")
	}
}

func TestSAPError_String_Empty(t *testing.T) {
	var e SAPError
	if got := e.String(); got != "" {
		t.Errorf("String() = %q, want empty string", got)
	}
}
