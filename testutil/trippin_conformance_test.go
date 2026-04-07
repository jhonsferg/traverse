//go:build conformance

package testutil_test

import (
	"context"
	"os"
	"testing"

	"github.com/jhonsferg/traverse"
)

var trippinBaseURL = func() string {
	if u := os.Getenv("ODATA_TRIPPIN_BASE_URL"); u != "" {
		return u
	}
	return "https://services.odata.org/TripPinRESTierService"
}()

func newTripPinClient(t *testing.T) *traverse.Client {
	t.Helper()
	c, err := traverse.New(
		traverse.WithBaseURL(trippinBaseURL),
		traverse.WithODataVersion(traverse.ODataV4),
	)
	if err != nil {
		t.Fatalf("failed to create traverse client: %v", err)
	}
	t.Cleanup(func() { c.Close() })
	return c
}

// TestTripPinGetPeople verifies basic entity set listing against TripPin.
func TestTripPinGetPeople(t *testing.T) {
	c := newTripPinClient(t)
	results, err := c.From("People").Collect(context.Background())
	if err != nil {
		t.Fatalf("People.Collect: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected at least 1 person, got 0")
	}
	// Verify basic decode  -  each record should have a UserName field
	for i, r := range results {
		if _, ok := r["UserName"]; !ok {
			t.Errorf("result[%d] missing UserName field", i)
		}
	}
}

// TestTripPinFilter verifies $filter works against TripPin.
func TestTripPinFilter(t *testing.T) {
	c := newTripPinClient(t)
	results, err := c.From("People").
		Filter("FirstName eq 'Russell'").
		Collect(context.Background())
	if err != nil {
		t.Fatalf("People$filter.Collect: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected at least 1 result for FirstName eq 'Russell', got 0")
	}
	for i, r := range results {
		if r["FirstName"] != "Russell" {
			t.Errorf("result[%d]: FirstName = %q, want Russell", i, r["FirstName"])
		}
	}
}

// TestTripPinSelect verifies $select works.
func TestTripPinSelect(t *testing.T) {
	c := newTripPinClient(t)
	results, err := c.From("People").
		Select("FirstName", "LastName").
		Top(5).
		Collect(context.Background())
	if err != nil {
		t.Fatalf("People$select.Collect: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected results, got 0")
	}
	for i, r := range results {
		if _, ok := r["FirstName"]; !ok {
			t.Errorf("result[%d] missing FirstName", i)
		}
		if _, ok := r["LastName"]; !ok {
			t.Errorf("result[%d] missing LastName", i)
		}
	}
}

// TestTripPinTop verifies $top pagination.
func TestTripPinTop(t *testing.T) {
	c := newTripPinClient(t)
	results, err := c.From("People").
		Top(3).
		Collect(context.Background())
	if err != nil {
		t.Fatalf("People$top.Collect: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("expected exactly 3 results, got %d", len(results))
	}
}

// TestTripPinExpand verifies $expand.
func TestTripPinExpand(t *testing.T) {
	c := newTripPinClient(t)
	result, err := c.From("People").
		FindByKey(context.Background(), "russellwhyte")
	if err != nil {
		t.Fatalf("People('russellwhyte').FindByKey: %v", err)
	}
	if result == nil {
		t.Fatal("expected a result for russellwhyte, got nil")
	}
	if result["UserName"] != "russellwhyte" {
		t.Errorf("UserName = %q, want russellwhyte", result["UserName"])
	}
}
