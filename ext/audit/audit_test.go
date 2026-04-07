package audit_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jhonsferg/relay"

	"github.com/jhonsferg/traverse/ext/audit"
)

// newTestServer returns an httptest.Server that responds with the given status
// code and a minimal JSON body for every request.
func newTestServer(statusCode int) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		_, _ = w.Write([]byte(`{"value":[]}`))
	}))
}

// newRelayClient builds a relay.Client pointing at srv with the audit
// middleware installed.
func newRelayClient(srv *httptest.Server, log audit.AuditLogger) *relay.Client {
	return relay.New(
		relay.WithBaseURL(srv.URL),
		relay.WithDisableRetry(),
		relay.WithTransportMiddleware(audit.Middleware(log)),
	)
}

// doRequest fires a request with the given method and path through the relay client.
func doRequest(t *testing.T, client *relay.Client, ctx context.Context, method, path string) {
	t.Helper()
	var req *relay.Request
	switch method {
	case http.MethodGet:
		req = client.Get(path)
	case http.MethodPost:
		req = client.Post(path)
	case http.MethodPatch:
		req = client.Patch(path)
	case http.MethodPut:
		req = client.Put(path)
	case http.MethodDelete:
		req = client.Delete(path)
	default:
		t.Fatalf("unsupported method: %s", method)
	}
	req = req.WithContext(ctx)
	_, err := client.Execute(req)
	if err != nil {
		t.Fatalf("execute request: %v", err)
	}
}

func TestGET_records_READ(t *testing.T) {
	srv := newTestServer(http.StatusOK)
	defer srv.Close()

	log := audit.NewInMemoryAuditLog()
	client := newRelayClient(srv, log)

	doRequest(t, client, context.Background(), http.MethodGet, "/odata/Orders")

	entries := log.Entries()
	if len(entries) != 1 {
		t.Fatalf("want 1 entry, got %d", len(entries))
	}
	e := entries[0]
	if e.Operation != audit.OperationRead {
		t.Errorf("operation: want READ, got %s", e.Operation)
	}
	if e.EntitySet != "Orders" {
		t.Errorf("entity set: want Orders, got %q", e.EntitySet)
	}
	if e.EntityKey != "" {
		t.Errorf("entity key: want empty, got %q", e.EntityKey)
	}
	if e.StatusCode != http.StatusOK {
		t.Errorf("status code: want 200, got %d", e.StatusCode)
	}
	if e.Duration == 0 {
		t.Error("duration should be non-zero")
	}
	if e.Timestamp.IsZero() {
		t.Error("timestamp should not be zero")
	}
}

func TestGET_with_key_extracts_EntityKey(t *testing.T) {
	srv := newTestServer(http.StatusOK)
	defer srv.Close()

	log := audit.NewInMemoryAuditLog()
	client := newRelayClient(srv, log)

	doRequest(t, client, context.Background(), http.MethodGet, "/odata/Orders(42)")

	e := log.Entries()[0]
	if e.EntitySet != "Orders" {
		t.Errorf("entity set: want Orders, got %q", e.EntitySet)
	}
	if e.EntityKey != "42" {
		t.Errorf("entity key: want 42, got %q", e.EntityKey)
	}
}

func TestPOST_records_CREATE(t *testing.T) {
	srv := newTestServer(http.StatusCreated)
	defer srv.Close()

	log := audit.NewInMemoryAuditLog()
	client := newRelayClient(srv, log)

	doRequest(t, client, context.Background(), http.MethodPost, "/odata/Orders")

	e := log.Entries()[0]
	if e.Operation != audit.OperationCreate {
		t.Errorf("operation: want CREATE, got %s", e.Operation)
	}
}

func TestPATCH_records_UPDATE(t *testing.T) {
	srv := newTestServer(http.StatusNoContent)
	defer srv.Close()

	log := audit.NewInMemoryAuditLog()
	client := newRelayClient(srv, log)

	doRequest(t, client, context.Background(), http.MethodPatch, "/odata/Orders(1)")

	e := log.Entries()[0]
	if e.Operation != audit.OperationUpdate {
		t.Errorf("operation: want UPDATE, got %s", e.Operation)
	}
}

func TestPUT_records_UPDATE(t *testing.T) {
	srv := newTestServer(http.StatusOK)
	defer srv.Close()

	log := audit.NewInMemoryAuditLog()
	client := newRelayClient(srv, log)

	doRequest(t, client, context.Background(), http.MethodPut, "/odata/Orders(1)")

	e := log.Entries()[0]
	if e.Operation != audit.OperationUpdate {
		t.Errorf("operation: want UPDATE, got %s", e.Operation)
	}
}

func TestDELETE_records_DELETE(t *testing.T) {
	srv := newTestServer(http.StatusNoContent)
	defer srv.Close()

	log := audit.NewInMemoryAuditLog()
	client := newRelayClient(srv, log)

	doRequest(t, client, context.Background(), http.MethodDelete, "/odata/Orders(5)")

	e := log.Entries()[0]
	if e.Operation != audit.OperationDelete {
		t.Errorf("operation: want DELETE, got %s", e.Operation)
	}
	if e.EntityKey != "5" {
		t.Errorf("entity key: want 5, got %q", e.EntityKey)
	}
}

func TestBatch_POST_records_BATCH(t *testing.T) {
	srv := newTestServer(http.StatusOK)
	defer srv.Close()

	log := audit.NewInMemoryAuditLog()
	client := newRelayClient(srv, log)

	doRequest(t, client, context.Background(), http.MethodPost, "/odata/$batch")

	e := log.Entries()[0]
	if e.Operation != audit.OperationBatch {
		t.Errorf("operation: want BATCH, got %s", e.Operation)
	}
	if e.EntitySet != "$batch" {
		t.Errorf("entity set: want $batch, got %q", e.EntitySet)
	}
}

func TestError_response_sets_Error_field(t *testing.T) {
	srv := newTestServer(http.StatusNotFound)
	defer srv.Close()

	log := audit.NewInMemoryAuditLog()
	client := newRelayClient(srv, log)

	doRequest(t, client, context.Background(), http.MethodGet, "/odata/Orders(99)")

	e := log.Entries()[0]
	if e.StatusCode != http.StatusNotFound {
		t.Errorf("status code: want 404, got %d", e.StatusCode)
	}
	if e.Error == "" {
		t.Error("Error field should be non-empty for 4xx response")
	}
}

func TestContext_UserID_and_RequestID_propagate(t *testing.T) {
	srv := newTestServer(http.StatusOK)
	defer srv.Close()

	log := audit.NewInMemoryAuditLog()
	client := newRelayClient(srv, log)

	ctx := audit.WithUser(context.Background(), "alice")
	ctx = audit.WithRequestID(ctx, "req-123")

	doRequest(t, client, ctx, http.MethodGet, "/odata/Products")

	e := log.Entries()[0]
	if e.UserID != "alice" {
		t.Errorf("UserID: want alice, got %q", e.UserID)
	}
	if e.RequestID != "req-123" {
		t.Errorf("RequestID: want req-123, got %q", e.RequestID)
	}
}

func TestDuration_is_non_zero(t *testing.T) {
	// Use a server with a deliberate delay so the measured duration is non-zero
	// even on Windows, where the default clock resolution is ~15 ms.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(20 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"value":[]}`))
	}))
	defer srv.Close()

	log := audit.NewInMemoryAuditLog()
	client := newRelayClient(srv, log)

	doRequest(t, client, context.Background(), http.MethodGet, "/odata/Customers")

	if d := log.Entries()[0].Duration; d == 0 {
		t.Error("Duration should be non-zero")
	}
}

func TestInMemoryAuditLog_Clear(t *testing.T) {
	log := audit.NewInMemoryAuditLog()
	log.Log(context.Background(), audit.AuditEntry{EntitySet: "Orders"})
	log.Clear()
	if n := len(log.Entries()); n != 0 {
		t.Errorf("after Clear: want 0 entries, got %d", n)
	}
}

func TestAuditLoggerFunc_adapts_function(t *testing.T) {
	var called bool
	fn := audit.AuditLoggerFunc(func(_ context.Context, _ audit.AuditEntry) {
		called = true
	})
	fn.Log(context.Background(), audit.AuditEntry{})
	if !called {
		t.Error("AuditLoggerFunc did not call the underlying function")
	}
}
