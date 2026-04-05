package traverse

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/jhonsferg/traverse/testutil"
)

// newRefClient creates a traverse Client pointed at the given mock server.
func newRefClient(t *testing.T, server *testutil.MockServer) *Client {
	t.Helper()
	c, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	return c
}

// ---- LinkTo -----------------------------------------------------------------

func TestLinkTo_SendsPUTWithCorrectURLAndBody(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{Status: 204})

	client := newRefClient(t, server)
	err := client.From("Orders").LinkTo(context.Background(), 1, "Customer", "Customers", "ALFKI")
	if err != nil {
		t.Fatalf("LinkTo() unexpected error: %v", err)
	}

	reqs := server.RecordedRequests()
	if len(reqs) != 1 {
		t.Fatalf("expected 1 request, got %d", len(reqs))
	}
	req := reqs[0]

	if req.Method != "PUT" {
		t.Errorf("method: got %q, want PUT", req.Method)
	}
	if req.Path != "/Orders(1)/Customer/$ref" {
		t.Errorf("path: got %q, want /Orders(1)/Customer/$ref", req.Path)
	}

	var body map[string]string
	if err := json.Unmarshal(req.Body, &body); err != nil {
		t.Fatalf("failed to parse request body: %v", err)
	}
	odataID, ok := body["@odata.id"]
	if !ok {
		t.Fatal("body missing @odata.id field")
	}
	// encodeKey wraps strings in single quotes and URL-encodes: 'ALFKI' -> %27ALFKI%27
	if !strings.HasSuffix(odataID, "/Customers(%27ALFKI%27)") {
		t.Errorf("@odata.id: got %q, want suffix /Customers(%%27ALFKI%%27)", odataID)
	}
}

func TestLinkTo_IntegerTargetKey(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{Status: 204})

	client := newRefClient(t, server)
	err := client.From("Orders").LinkTo(context.Background(), 42, "Product", "Products", 99)
	if err != nil {
		t.Fatalf("LinkTo() unexpected error: %v", err)
	}

	reqs := server.RecordedRequests()
	if len(reqs) != 1 {
		t.Fatalf("expected 1 request, got %d", len(reqs))
	}
	req := reqs[0]

	if req.Path != "/Orders(42)/Product/$ref" {
		t.Errorf("path: got %q, want /Orders(42)/Product/$ref", req.Path)
	}

	var body map[string]string
	if err := json.Unmarshal(req.Body, &body); err != nil {
		t.Fatalf("failed to parse request body: %v", err)
	}
	odataID := body["@odata.id"]
	if !strings.HasSuffix(odataID, "/Products(99)") {
		t.Errorf("@odata.id: got %q, want suffix /Products(99)", odataID)
	}
}

func TestLinkTo_Returns200AsSuccess(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{Status: 200, Body: `{}`})

	client := newRefClient(t, server)
	err := client.From("Orders").LinkTo(context.Background(), 1, "Customer", "Customers", 5)
	if err != nil {
		t.Errorf("LinkTo() expected no error on 200, got: %v", err)
	}
}

func TestLinkTo_Returns404AsErrEntityNotFound(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{Status: 404, Body: `{"error":{"code":"NotFound"}}`})

	client := newRefClient(t, server)
	err := client.From("Orders").LinkTo(context.Background(), 1, "Customer", "Customers", 5)
	if err == nil {
		t.Fatal("expected error on 404, got nil")
	}
	if !IsEntityNotFound(err) {
		t.Errorf("expected ErrEntityNotFound, got: %v", err)
	}
}

func TestLinkTo_Returns409AsConflict(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{Status: 409, Body: `{"error":{"code":"Conflict"}}`})

	client := newRefClient(t, server)
	err := client.From("Orders").LinkTo(context.Background(), 1, "Customer", "Customers", 5)
	if err == nil {
		t.Fatal("expected error on 409, got nil")
	}
	if !IsConcurrencyConflict(err) {
		t.Errorf("expected ErrConcurrencyConflict, got: %v", err)
	}
}

func TestLinkTo_Returns412AsConcurrencyConflict(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{Status: 412, Body: `{"error":{"code":"PreconditionFailed"}}`})

	client := newRefClient(t, server)
	err := client.From("Orders").LinkTo(context.Background(), 1, "Customer", "Customers", 5)
	if err == nil {
		t.Fatal("expected error on 412, got nil")
	}
	if !IsConcurrencyConflict(err) {
		t.Errorf("expected ErrConcurrencyConflict, got: %v", err)
	}
}

func TestLinkTo_InvalidSourceKey_ReturnsError(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	client := newRefClient(t, server)
	err := client.From("Orders").LinkTo(context.Background(), true, "Customer", "Customers", 5)
	if err == nil {
		t.Fatal("expected error for unsupported key type, got nil")
	}
	if !strings.Contains(err.Error(), "invalid source key") {
		t.Errorf("error message: got %q, want it to contain 'invalid source key'", err.Error())
	}
}

func TestLinkTo_InvalidTargetKey_ReturnsError(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	client := newRefClient(t, server)
	err := client.From("Orders").LinkTo(context.Background(), 1, "Customer", "Customers", true)
	if err == nil {
		t.Fatal("expected error for unsupported target key type, got nil")
	}
	if !strings.Contains(err.Error(), "invalid target key") {
		t.Errorf("error message: got %q, want it to contain 'invalid target key'", err.Error())
	}
}

// ---- UnlinkFrom -------------------------------------------------------------

func TestUnlinkFrom_Single_SendsDELETEWithCorrectURL(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{Status: 204})

	client := newRefClient(t, server)
	err := client.From("Orders").UnlinkFrom(context.Background(), 1, "Customer")
	if err != nil {
		t.Fatalf("UnlinkFrom() unexpected error: %v", err)
	}

	reqs := server.RecordedRequests()
	if len(reqs) != 1 {
		t.Fatalf("expected 1 request, got %d", len(reqs))
	}
	req := reqs[0]

	if req.Method != "DELETE" {
		t.Errorf("method: got %q, want DELETE", req.Method)
	}
	if req.Path != "/Orders(1)/Customer/$ref" {
		t.Errorf("path: got %q, want /Orders(1)/Customer/$ref", req.Path)
	}
}

func TestUnlinkFrom_Collection_SendsDELETEWithTargetKeyInURL(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{Status: 204})

	client := newRefClient(t, server)
	err := client.From("Orders").UnlinkFrom(context.Background(), 1, "Items", 42)
	if err != nil {
		t.Fatalf("UnlinkFrom() unexpected error: %v", err)
	}

	reqs := server.RecordedRequests()
	if len(reqs) != 1 {
		t.Fatalf("expected 1 request, got %d", len(reqs))
	}
	req := reqs[0]

	if req.Method != "DELETE" {
		t.Errorf("method: got %q, want DELETE", req.Method)
	}
	if req.Path != "/Orders(1)/Items(42)/$ref" {
		t.Errorf("path: got %q, want /Orders(1)/Items(42)/$ref", req.Path)
	}
}

func TestUnlinkFrom_StringKey(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{Status: 204})

	client := newRefClient(t, server)
	err := client.From("Customers").UnlinkFrom(context.Background(), "ALFKI", "Orders", 7)
	if err != nil {
		t.Fatalf("UnlinkFrom() unexpected error: %v", err)
	}

	reqs := server.RecordedRequests()
	if len(reqs) != 1 {
		t.Fatalf("expected 1 request, got %d", len(reqs))
	}
	// String keys are URL-encoded: 'ALFKI' -> '%27ALFKI%27
	req := reqs[0]
	if !strings.Contains(req.Path, "/$ref") {
		t.Errorf("path: got %q, expected /$ref suffix", req.Path)
	}
}

func TestUnlinkFrom_Returns404AsErrEntityNotFound(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{Status: 404, Body: `{"error":{"code":"NotFound"}}`})

	client := newRefClient(t, server)
	err := client.From("Orders").UnlinkFrom(context.Background(), 1, "Customer")
	if err == nil {
		t.Fatal("expected error on 404, got nil")
	}
	if !IsEntityNotFound(err) {
		t.Errorf("expected ErrEntityNotFound, got: %v", err)
	}
}

func TestUnlinkFrom_Returns409AsConflict(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{Status: 409, Body: `{"error":{"code":"Conflict"}}`})

	client := newRefClient(t, server)
	err := client.From("Orders").UnlinkFrom(context.Background(), 1, "Customer")
	if err == nil {
		t.Fatal("expected error on 409, got nil")
	}
	if !IsConcurrencyConflict(err) {
		t.Errorf("expected ErrConcurrencyConflict, got: %v", err)
	}
}

func TestUnlinkFrom_Returns412AsConcurrencyConflict(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{Status: 412, Body: `{"error":{"code":"PreconditionFailed"}}`})

	client := newRefClient(t, server)
	err := client.From("Orders").UnlinkFrom(context.Background(), 1, "Customer")
	if err == nil {
		t.Fatal("expected error on 412, got nil")
	}
	if !IsConcurrencyConflict(err) {
		t.Errorf("expected ErrConcurrencyConflict, got: %v", err)
	}
}

func TestUnlinkFrom_InvalidSourceKey_ReturnsError(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	client := newRefClient(t, server)
	err := client.From("Orders").UnlinkFrom(context.Background(), true, "Customer")
	if err == nil {
		t.Fatal("expected error for unsupported key type, got nil")
	}
	if !strings.Contains(err.Error(), "invalid source key") {
		t.Errorf("error message: got %q, want it to contain 'invalid source key'", err.Error())
	}
}

func TestUnlinkFrom_InvalidTargetKey_ReturnsError(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	client := newRefClient(t, server)
	err := client.From("Orders").UnlinkFrom(context.Background(), 1, "Items", true)
	if err == nil {
		t.Fatal("expected error for unsupported target key type, got nil")
	}
	if !strings.Contains(err.Error(), "invalid target key") {
		t.Errorf("error message: got %q, want it to contain 'invalid target key'", err.Error())
	}
}

func TestUnlinkFrom_Returns200AsSuccess(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{Status: 200, Body: `{}`})

	client := newRefClient(t, server)
	err := client.From("Orders").UnlinkFrom(context.Background(), 1, "Customer")
	if err != nil {
		t.Errorf("UnlinkFrom() expected no error on 200, got: %v", err)
	}
}
