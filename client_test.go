package traverse

import (
	"testing"
)

func TestClientConstruction(t *testing.T) {
	// Test that New() creates a valid client
	c, err := New(
		WithBaseURL("http://localhost:8080/odata"),
	)

	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	if c == nil {
		t.Fatalf("New() returned nil client")
	}

	if c.baseURL != "http://localhost:8080/odata" {
		t.Errorf("baseURL = %s, want http://localhost:8080/odata", c.baseURL)
	}
}

func TestClientFrom(t *testing.T) {
	c, _ := New(
		WithBaseURL("http://localhost:8080/odata"),
	)

	qb := c.From("Products")
	if qb == nil {
		t.Fatalf("From() returned nil")
	}

	if qb.entitySet != "Products" {
		t.Errorf("entitySet = %s, want Products", qb.entitySet)
	}
}

// TestFrom_StripLeadingSlash verifies that From("/Products") and From("Products")
// are equivalent. A leading slash in the entity-set name would cause buildURL to
// emit "//Products", which servers such as SAP reject with 401 instead of 404.
func TestFrom_StripLeadingSlash(t *testing.T) {
	t.Parallel()

	c, _ := New(WithBaseURL("http://localhost/odata"))

	cases := []string{"/Products", "//Products", "Products"}
	for _, input := range cases {
		qb := c.From(input)
		if qb.entitySet != "Products" {
			t.Errorf("From(%q): entitySet = %q, want %q", input, qb.entitySet, "Products")
		}
	}
}

func TestClientWithOptions(t *testing.T) {
	c, _ := New(
		WithBaseURL("http://localhost:8080/odata"),
		WithODataVersion(ODataV4),
		WithPageSize(5000),
	)

	if c.version != ODataV4 {
		t.Errorf("version = %v, want ODataV4", c.version)
	}

	if c.pageSize != 5000 {
		t.Errorf("pageSize = %d, want 5000", c.pageSize)
	}
}
