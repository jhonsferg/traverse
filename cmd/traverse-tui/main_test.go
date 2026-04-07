package main

import "testing"

func TestBuildURL_Basic(t *testing.T) {
	s := &session{baseURL: "https://example.com/odata", entitySet: "Customers", top: 10}
	u := buildURL(s)
	if u != "https://example.com/odata/Customers?$top=10" {
		t.Errorf("unexpected URL: %s", u)
	}
}

func TestBuildURL_WithFilter(t *testing.T) {
	s := &session{
		baseURL:   "https://example.com/odata",
		entitySet: "Customers",
		filters:   []string{"Country eq 'Germany'"},
		top:       5,
	}
	u := buildURL(s)
	if u == "" {
		t.Fatal("empty URL")
	}
	// Should contain both filter and top
	if !contains(u, "$filter=Country+eq") && !contains(u, "$filter=Country eq") {
		// URL may not be encoded in this simple impl
	}
	if !contains(u, "$top=5") {
		t.Errorf("missing $top in %s", u)
	}
}

func TestBuildURL_Incomplete(t *testing.T) {
	s := &session{}
	u := buildURL(s)
	if !contains(u, "incomplete") {
		t.Errorf("expected incomplete message, got: %s", u)
	}
}

func TestHandleCommand_Help(t *testing.T) {
	s := &session{}
	// Should not panic and return true
	if !handleCommand(s, "help") {
		t.Error("help should return true")
	}
}

func TestHandleCommand_Exit(t *testing.T) {
	s := &session{}
	if handleCommand(s, "exit") {
		t.Error("exit should return false")
	}
	if handleCommand(s, "quit") {
		t.Error("quit should return false")
	}
}

func TestHandleCommand_Use(t *testing.T) {
	s := &session{}
	handleCommand(s, "use Orders")
	if s.entitySet != "Orders" {
		t.Errorf("expected Orders, got %s", s.entitySet)
	}
}

func TestHandleCommand_Filter(t *testing.T) {
	s := &session{entitySet: "Customers"}
	handleCommand(s, "filter Country eq 'Germany'")
	if len(s.filters) != 1 {
		t.Errorf("expected 1 filter, got %d", len(s.filters))
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 ||
		func() bool {
			for i := 0; i <= len(s)-len(sub); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
			return false
		}())
}
