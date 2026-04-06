package traverse

import (
	"context"
	"net/http"
	"testing"

	"github.com/jhonsferg/traverse/testutil"
)

// newDeleteClient creates a Client pointed at the given mock server.
func newDeleteClient(t *testing.T, server *testutil.MockServer) *Client {
	t.Helper()
	c, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	return c
}

// ---- DeleteWithOptions: cascade delete ----------------------------------------

func TestDeleteWithOptions_CascadeDelete_SetsPreferHeader(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()
	server.Enqueue(testutil.MockResponse{Status: http.StatusNoContent})

	client := newDeleteClient(t, server)
	err := client.DeleteWithOptions(context.Background(), "Orders", 1, WithDeleteCascade())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	reqs := server.RecordedRequests()
	if len(reqs) != 1 {
		t.Fatalf("expected 1 request, got %d", len(reqs))
	}
	req := reqs[0]

	if req.Method != http.MethodDelete {
		t.Errorf("method: got %q, want DELETE", req.Method)
	}
	if req.Path != "/Orders(1)" {
		t.Errorf("path: got %q, want /Orders(1)", req.Path)
	}
	prefer := req.Headers.Get("Prefer")
	if prefer != "odata.cascade-delete" {
		t.Errorf("Prefer header: got %q, want %q", prefer, "odata.cascade-delete")
	}
}

func TestDeleteWithOptions_IfMatch_SetsHeader(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()
	server.Enqueue(testutil.MockResponse{Status: http.StatusNoContent})

	client := newDeleteClient(t, server)
	err := client.DeleteWithOptions(context.Background(), "Products", 42, WithDeleteIfMatch(`W/"abc123"`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	reqs := server.RecordedRequests()
	req := reqs[0]
	if req.Headers.Get("If-Match") != `W/"abc123"` {
		t.Errorf("If-Match: got %q, want W/\"abc123\"", req.Headers.Get("If-Match"))
	}
}

func TestDeleteWithOptions_ReturnRepresentation_SetsPreferHeader(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()
	server.Enqueue(testutil.MockResponse{Status: http.StatusNoContent})

	client := newDeleteClient(t, server)
	err := client.DeleteWithOptions(context.Background(), "Items", "X1", WithDeleteReturnRepresentation())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	reqs := server.RecordedRequests()
	prefer := reqs[0].Headers.Get("Prefer")
	if prefer != "return=representation" {
		t.Errorf("Prefer: got %q, want %q", prefer, "return=representation")
	}
}

func TestDeleteWithOptions_CascadeAndReturnRepresentation(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()
	server.Enqueue(testutil.MockResponse{Status: http.StatusNoContent})

	client := newDeleteClient(t, server)
	err := client.DeleteWithOptions(context.Background(), "Orders", 1,
		WithDeleteCascade(),
		WithDeleteReturnRepresentation(),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	prefer := server.RecordedRequests()[0].Headers.Get("Prefer")
	if prefer != "odata.cascade-delete,return=representation" {
		t.Errorf("Prefer: got %q, want %q", prefer, "odata.cascade-delete,return=representation")
	}
}

func TestDeleteWithOptions_NoOptions_NoPreferHeader(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()
	server.Enqueue(testutil.MockResponse{Status: http.StatusNoContent})

	client := newDeleteClient(t, server)
	err := client.DeleteWithOptions(context.Background(), "Products", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	req := server.RecordedRequests()[0]
	if req.Headers.Get("Prefer") != "" {
		t.Errorf("Prefer header should be absent, got %q", req.Headers.Get("Prefer"))
	}
}

func TestDeleteWithOptions_StringKey_QuotedInURL(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()
	server.Enqueue(testutil.MockResponse{Status: http.StatusNoContent})

	client := newDeleteClient(t, server)
	err := client.DeleteWithOptions(context.Background(), "Materials", "MAT001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	req := server.RecordedRequests()[0]
	// encodeKey wraps string keys in single quotes, URL-encoded as %27
	if req.Path != "/Materials(%27MAT001%27)" {
		t.Errorf("path: got %q, want /Materials(%%27MAT001%%27)", req.Path)
	}
}

func TestDeleteWithOptions_NotFound_ReturnsErrEntityNotFound(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()
	server.Enqueue(testutil.MockResponse{Status: http.StatusNotFound})

	client := newDeleteClient(t, server)
	err := client.DeleteWithOptions(context.Background(), "Orders", 999)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !isErrEntityNotFound(err) {
		t.Errorf("expected ErrEntityNotFound, got %v", err)
	}
}

func TestDeleteWithOptions_PreconditionFailed_ReturnsErrConcurrencyConflict(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()
	server.Enqueue(testutil.MockResponse{Status: http.StatusPreconditionFailed})

	client := newDeleteClient(t, server)
	err := client.DeleteWithOptions(context.Background(), "Orders", 1, WithDeleteIfMatch(`W/"stale"`))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !isErrConcurrencyConflict(err) {
		t.Errorf("expected ErrConcurrencyConflict, got %v", err)
	}
}

func TestDeleteWithOptions_Conflict_ReturnsErrConcurrencyConflict(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()
	server.Enqueue(testutil.MockResponse{Status: http.StatusConflict})

	client := newDeleteClient(t, server)
	err := client.DeleteWithOptions(context.Background(), "Orders", 1)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !isErrConcurrencyConflict(err) {
		t.Errorf("expected ErrConcurrencyConflict, got %v", err)
	}
}

func TestDeleteWithOptions_InvalidKey_ReturnsError(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	client := newDeleteClient(t, server)
	err := client.DeleteWithOptions(context.Background(), "Orders", true) // bool key unsupported
	if err == nil {
		t.Fatal("expected error for invalid key type, got nil")
	}
}

func TestDeleteWithOptions_InvalidatesCache(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()
	server.Enqueue(testutil.MockResponse{Status: http.StatusNoContent})

	// Ensure no panic when cache is nil (default client has no response cache)
	client := newDeleteClient(t, server)
	if err := client.DeleteWithOptions(context.Background(), "Orders", 1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ---- DeleteLink ---------------------------------------------------------------

func TestDeleteLink_SendsDELETEWithRefAndIDParam(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()
	server.Enqueue(testutil.MockResponse{Status: http.StatusNoContent})

	client := newDeleteClient(t, server)
	err := client.From("Orders").Key(1).DeleteLink(context.Background(), "Items", 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	reqs := server.RecordedRequests()
	if len(reqs) != 1 {
		t.Fatalf("expected 1 request, got %d", len(reqs))
	}
	req := reqs[0]

	if req.Method != http.MethodDelete {
		t.Errorf("method: got %q, want DELETE", req.Method)
	}
	// Path should be /Orders(1)/Items/$ref
	if req.Path != "/Orders(1)/Items/$ref" {
		t.Errorf("path: got %q, want /Orders(1)/Items/$ref", req.Path)
	}
	// $id param should reference Orders(5)
	idParam := req.Query.Get("$id")
	if idParam == "" {
		t.Fatal("$id query parameter missing")
	}
}

func TestDeleteLink_StringSourceKey(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()
	server.Enqueue(testutil.MockResponse{Status: http.StatusNoContent})

	client := newDeleteClient(t, server)
	err := client.From("Customers").Key("ALFKI").DeleteLink(context.Background(), "Orders", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	req := server.RecordedRequests()[0]
	if req.Path != "/Customers(%27ALFKI%27)/Orders/$ref" {
		t.Errorf("path: got %q, want /Customers(%%27ALFKI%%27)/Orders/$ref", req.Path)
	}
}

func TestDeleteLink_NotFound_ReturnsErrEntityNotFound(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()
	server.Enqueue(testutil.MockResponse{Status: http.StatusNotFound})

	client := newDeleteClient(t, server)
	err := client.From("Orders").Key(1).DeleteLink(context.Background(), "Items", 999)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !isErrEntityNotFound(err) {
		t.Errorf("expected ErrEntityNotFound, got %v", err)
	}
}

func TestDeleteLink_InvalidSourceKey_ReturnsError(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	client := newDeleteClient(t, server)
	// nil is not supported by encodeKey
	err := client.From("Orders").Key(true).DeleteLink(context.Background(), "Items", 5)
	if err == nil {
		t.Fatal("expected error for invalid source key, got nil")
	}
}

func TestDeleteLink_InvalidRelatedKey_ReturnsError(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	client := newDeleteClient(t, server)
	err := client.From("Orders").Key(1).DeleteLink(context.Background(), "Items", true)
	if err == nil {
		t.Fatal("expected error for invalid related key, got nil")
	}
}

func TestDeleteLink_Conflict_ReturnsErrConcurrencyConflict(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()
	server.Enqueue(testutil.MockResponse{Status: http.StatusConflict})

	client := newDeleteClient(t, server)
	err := client.From("Orders").Key(1).DeleteLink(context.Background(), "Items", 5)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !isErrConcurrencyConflict(err) {
		t.Errorf("expected ErrConcurrencyConflict, got %v", err)
	}
}

// ---- DeleteLinks --------------------------------------------------------------

func TestDeleteLinks_SendsDELETEToRefEndpoint(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()
	server.Enqueue(testutil.MockResponse{Status: http.StatusNoContent})

	client := newDeleteClient(t, server)
	err := client.From("Orders").Key(1).DeleteLinks(context.Background(), "Items")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	reqs := server.RecordedRequests()
	if len(reqs) != 1 {
		t.Fatalf("expected 1 request, got %d", len(reqs))
	}
	req := reqs[0]

	if req.Method != http.MethodDelete {
		t.Errorf("method: got %q, want DELETE", req.Method)
	}
	if req.Path != "/Orders(1)/Items/$ref" {
		t.Errorf("path: got %q, want /Orders(1)/Items/$ref", req.Path)
	}
	if req.Query.Get("$id") != "" {
		t.Errorf("$id should be absent for DeleteLinks, got %q", req.Query.Get("$id"))
	}
}

func TestDeleteLinks_InvalidSourceKey_ReturnsError(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	client := newDeleteClient(t, server)
	err := client.From("Orders").Key(true).DeleteLinks(context.Background(), "Items")
	if err == nil {
		t.Fatal("expected error for invalid key, got nil")
	}
}

func TestDeleteLinks_NotFound_ReturnsErrEntityNotFound(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()
	server.Enqueue(testutil.MockResponse{Status: http.StatusNotFound})

	client := newDeleteClient(t, server)
	err := client.From("Orders").Key(999).DeleteLinks(context.Background(), "Items")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !isErrEntityNotFound(err) {
		t.Errorf("expected ErrEntityNotFound, got %v", err)
	}
}

func TestDeleteLinks_PreconditionFailed_ReturnsErrConcurrencyConflict(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()
	server.Enqueue(testutil.MockResponse{Status: http.StatusPreconditionFailed})

	client := newDeleteClient(t, server)
	err := client.From("Orders").Key(1).DeleteLinks(context.Background(), "Items")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !isErrConcurrencyConflict(err) {
		t.Errorf("expected ErrConcurrencyConflict, got %v", err)
	}
}

// ---- BulkDelete ---------------------------------------------------------------

func TestBulkDelete_SendsDELETEToCollectionURL(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()
	server.Enqueue(testutil.MockResponse{Status: http.StatusNoContent})

	client := newDeleteClient(t, server)
	err := client.From("TempLogs").BulkDelete(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	reqs := server.RecordedRequests()
	if len(reqs) != 1 {
		t.Fatalf("expected 1 request, got %d", len(reqs))
	}
	req := reqs[0]

	if req.Method != http.MethodDelete {
		t.Errorf("method: got %q, want DELETE", req.Method)
	}
	if req.Path != "/TempLogs" {
		t.Errorf("path: got %q, want /TempLogs", req.Path)
	}
}

func TestBulkDelete_WithFilter_IncludesFilterInURL(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()
	server.Enqueue(testutil.MockResponse{Status: http.StatusNoContent})

	client := newDeleteClient(t, server)
	err := client.From("TempLogs").Filter("Status eq 'archived'").BulkDelete(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	req := server.RecordedRequests()[0]
	filter := req.Query.Get("$filter")
	if filter != "Status eq 'archived'" {
		t.Errorf("$filter: got %q, want %q", filter, "Status eq 'archived'")
	}
}

func TestBulkDelete_NotFound_ReturnsErrEntityNotFound(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()
	server.Enqueue(testutil.MockResponse{Status: http.StatusNotFound})

	client := newDeleteClient(t, server)
	err := client.From("TempLogs").BulkDelete(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !isErrEntityNotFound(err) {
		t.Errorf("expected ErrEntityNotFound, got %v", err)
	}
}

func TestBulkDelete_UnexpectedStatus_ReturnsError(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()
	server.Enqueue(testutil.MockResponse{Status: http.StatusBadRequest})

	client := newDeleteClient(t, server)
	err := client.From("TempLogs").BulkDelete(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestBulkDelete_InvalidatesCache(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()
	server.Enqueue(testutil.MockResponse{Status: http.StatusNoContent})

	client := newDeleteClient(t, server)
	if err := client.From("TempLogs").BulkDelete(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBulkDelete_200OK_Succeeds(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()
	server.Enqueue(testutil.MockResponse{Status: http.StatusOK})

	client := newDeleteClient(t, server)
	if err := client.From("TempLogs").BulkDelete(context.Background()); err != nil {
		t.Fatalf("expected success for 200, got: %v", err)
	}
}

// ---- Key() method -------------------------------------------------------------

func TestKey_ReturnsSameQueryBuilder(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	client := newDeleteClient(t, server)
	q := client.From("Orders")
	q2 := q.Key(42)
	if q2 != q {
		t.Error("Key() should return the same QueryBuilder pointer")
	}
	if q.keyValue != 42 {
		t.Errorf("keyValue: got %v, want 42", q.keyValue)
	}
}

// ---- preferHeader helper -------------------------------------------------------

func TestPreferHeader_Values(t *testing.T) {
	cases := []struct {
		opts DeleteOptions
		want string
	}{
		{DeleteOptions{}, ""},
		{DeleteOptions{CascadeNavigationProperties: true}, "odata.cascade-delete"},
		{DeleteOptions{ReturnRepresentation: true}, "return=representation"},
		{DeleteOptions{CascadeNavigationProperties: true, ReturnRepresentation: true}, "odata.cascade-delete,return=representation"},
	}
	for _, tc := range cases {
		got := preferHeader(tc.opts)
		if got != tc.want {
			t.Errorf("preferHeader(%+v) = %q, want %q", tc.opts, got, tc.want)
		}
	}
}

// ---- helpers ------------------------------------------------------------------

func isErrEntityNotFound(err error) bool {
	return err != nil && containsStr(err.Error(), "not found")
}

func isErrConcurrencyConflict(err error) bool {
	return err != nil && (containsStr(err.Error(), "conflict") || containsStr(err.Error(), "concurrency") || containsStr(err.Error(), "precondition"))
}

func containsStr(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && stringContains(s, sub))
}

func stringContains(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
