package traverse

import (
	"context"
	"fmt"
	"testing"

	"github.com/jhonsferg/traverse/testutil"
)

// ---- helpers ----------------------------------------------------------------

func newETagClient(t *testing.T, server *testutil.MockServer) *Client {
	t.Helper()
	c, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	return c
}

// lastRequest returns the most recent recorded request.
func lastRequest(t *testing.T, server *testutil.MockServer) testutil.RecordedRequest {
	t.Helper()
	reqs := server.RecordedRequests()
	if len(reqs) == 0 {
		t.Fatal("no requests recorded")
	}
	return reqs[len(reqs)-1]
}

// ---- ReadWithETag -----------------------------------------------------------

func TestReadWithETag_ReturnsEntityAndETag(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{
		Status:  200,
		Headers: map[string]string{"ETag": `W/"abc123"`},
		Body:    `{"ID":1,"Name":"Widget"}`,
	})

	client := newETagClient(t, server)
	result, err := client.ReadWithETag(context.Background(), "Products", "1")
	if err != nil {
		t.Fatalf("ReadWithETag() error: %v", err)
	}
	if result.ETag.Value != `W/"abc123"` {
		t.Errorf("ETag = %q, want %q", result.ETag.Value, `W/"abc123"`)
	}
	if result.ETag.IsWeak() != true {
		t.Error("expected weak ETag")
	}
	if result.Entity == nil {
		t.Error("expected non-nil data")
	}
}

func TestReadWithETag_ServerError(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{Status: 404, Body: `{"error":{"code":"NotFound"}}`})

	client := newETagClient(t, server)
	_, err := client.ReadWithETag(context.Background(), "Products", "999")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestReadWithETag_NoETagHeader(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{
		Status: 200,
		Body:   `{"ID":2,"Name":"Gadget"}`,
	})

	client := newETagClient(t, server)
	result, err := client.ReadWithETag(context.Background(), "Products", "2")
	if err != nil {
		t.Fatalf("ReadWithETag() error: %v", err)
	}
	if result.ETag.Value != "" {
		t.Errorf("expected empty ETag, got %q", result.ETag.Value)
	}
}

// ---- ETag helpers -----------------------------------------------------------

func TestETag_IsWeak(t *testing.T) {
	tests := []struct {
		value string
		weak  bool
	}{
		{`W/"abc"`, true},
		{`"abc"`, false},
		{``, false},
	}
	for _, tt := range tests {
		e := ETag{Value: tt.value}
		if got := e.IsWeak(); got != tt.weak {
			t.Errorf("ETag(%q).IsWeak() = %v, want %v", tt.value, got, tt.weak)
		}
	}
}

func TestETag_String(t *testing.T) {
	e := ETag{Value: `"xyz"`}
	if e.String() != `"xyz"` {
		t.Errorf("String() = %q, want %q", e.String(), `"xyz"`)
	}
}

// ---- UpdateWithETag ---------------------------------------------------------

func TestUpdateWithETag_MatchingETag_Returns204(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{Status: 204, Body: ``})

	client := newETagClient(t, server)
	etag := ETag{Value: `"abc"`}
	err := client.UpdateWithETag(context.Background(), "Products", "1", map[string]any{"Name": "Updated"}, etag)
	if err != nil {
		t.Fatalf("UpdateWithETag() error: %v", err)
	}

	req := lastRequest(t, server)
	if req.Headers.Get("If-Match") != `"abc"` {
		t.Errorf("If-Match = %q, want %q", req.Headers.Get("If-Match"), `"abc"`)
	}
}

func TestUpdateWithETag_Conflict_Returns412(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{
		Status: 412,
		Body:   `{"error":{"code":"PreconditionFailed"}}`,
	})

	client := newETagClient(t, server)
	etag := ETag{Value: `"stale"`}
	err := client.UpdateWithETag(context.Background(), "Products", "1", map[string]any{"Name": "x"}, etag)
	if err == nil {
		t.Fatal("expected ErrConcurrencyConflict, got nil")
	}
}

func TestUpdateWithETag_409Conflict(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{Status: 409, Body: `{"error":{"code":"Conflict"}}`})

	client := newETagClient(t, server)
	etag := ETag{Value: `"x"`}
	err := client.UpdateWithETag(context.Background(), "Products", "1", map[string]any{}, etag)
	if err == nil {
		t.Fatal("expected error on 409, got nil")
	}
}

// ---- ReplaceWithETag --------------------------------------------------------

func TestReplaceWithETag_SendsPUTWithIfMatch(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{Status: 204})

	client := newETagClient(t, server)
	etag := ETag{Value: `"v1"`}
	err := client.ReplaceWithETag(context.Background(), "Products", "1", map[string]any{"ID": 1, "Name": "Full"}, etag)
	if err != nil {
		t.Fatalf("ReplaceWithETag() error: %v", err)
	}

	req := lastRequest(t, server)
	if req.Method != "PUT" {
		t.Errorf("method = %q, want PUT", req.Method)
	}
	if req.Headers.Get("If-Match") != `"v1"` {
		t.Errorf("If-Match = %q, want %q", req.Headers.Get("If-Match"), `"v1"`)
	}
}

// ---- DeleteWithETag ---------------------------------------------------------

func TestDeleteWithETag_SendsDELETEWithIfMatch(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{Status: 204})

	client := newETagClient(t, server)
	etag := ETag{Value: `"v2"`}
	err := client.DeleteWithETag(context.Background(), "Products", "1", etag)
	if err != nil {
		t.Fatalf("DeleteWithETag() error: %v", err)
	}

	req := lastRequest(t, server)
	if req.Method != "DELETE" {
		t.Errorf("method = %q, want DELETE", req.Method)
	}
	if req.Headers.Get("If-Match") != `"v2"` {
		t.Errorf("If-Match = %q, want %q", req.Headers.Get("If-Match"), `"v2"`)
	}
}

func TestDeleteWithETag_ServerError(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{Status: 500, Body: `{"error":{"code":"ServerError"}}`})

	client := newETagClient(t, server)
	etag := ETag{Value: `"v1"`}
	err := client.DeleteWithETag(context.Background(), "Products", "1", etag)
	if err == nil {
		t.Fatal("expected error on 500")
	}
}

// ---- Upsert -----------------------------------------------------------------

func TestUpsert_Created201(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{Status: 201, Body: `{"ID":1,"Name":"New"}`})

	client := newETagClient(t, server)
	err := client.Upsert(context.Background(), "Products", "1", map[string]any{"ID": 1, "Name": "New"})
	if err != nil {
		t.Fatalf("Upsert() error: %v", err)
	}

	req := lastRequest(t, server)
	if req.Method != "PUT" {
		t.Errorf("method = %q, want PUT", req.Method)
	}
	if req.Headers.Get("If-None-Match") != "*" {
		t.Errorf("If-None-Match = %q, want *", req.Headers.Get("If-None-Match"))
	}
}

func TestUpsert_Replaced200(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{Status: 200, Body: ``})

	client := newETagClient(t, server)
	err := client.Upsert(context.Background(), "Products", "1", map[string]any{"ID": 1})
	if err != nil {
		t.Fatalf("Upsert() error: %v", err)
	}
}

func TestUpsert_Replaced204(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{Status: 204})

	client := newETagClient(t, server)
	err := client.Upsert(context.Background(), "Products", "1", map[string]any{})
	if err != nil {
		t.Fatalf("Upsert() 204 error: %v", err)
	}
}

func TestUpsert_ServerError(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	// Relay retries 500 up to 3 times by default; enqueue enough to exhaust all attempts.
	for range 3 {
		server.Enqueue(testutil.MockResponse{Status: 500, Body: `{"error":{"code":"err"}}`})
	}

	client := newETagClient(t, server)
	err := client.Upsert(context.Background(), "Products", "1", map[string]any{})
	if err == nil {
		t.Fatal("expected error on 500")
	}
}

// ---- checkUpdateResponse ----------------------------------------------------

func TestCheckUpdateResponse_412(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{Status: 412, Body: `{}`})

	c, _ := New(WithBaseURL(server.URL()))
	req := c.http.Get("/")
	resp, _ := c.http.Execute(req)

	err := checkUpdateResponse(resp)
	if err == nil {
		t.Fatal("expected ErrConcurrencyConflict")
	}
}

func TestCheckUpdateResponse_409(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{Status: 409, Body: `{}`})

	c, _ := New(WithBaseURL(server.URL()))
	req := c.http.Get("/")
	resp, _ := c.http.Execute(req)

	err := checkUpdateResponse(resp)
	if err == nil {
		t.Fatal("expected ErrConcurrencyConflict on 409")
	}
}

func TestCheckUpdateResponse_204_NoError(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{Status: 204})

	c, _ := New(WithBaseURL(server.URL()))
	req := c.http.Get("/")
	resp, _ := c.http.Execute(req)

	if err := checkUpdateResponse(resp); err != nil {
		t.Errorf("unexpected error on 204: %v", err)
	}
}

// ---- EntityWithETag ---------------------------------------------------------

func TestEntityWithETag_Fields(t *testing.T) {
	data := map[string]any{"ID": 1}
	e := EntityWithETag{
		Entity: data,
		ETag:   ETag{Value: `"abc"`},
	}
	if fmt.Sprintf("%v", e.Entity) != "map[ID:1]" {
		t.Errorf("unexpected data: %v", e.Entity)
	}
	if e.ETag.Value != `"abc"` {
		t.Errorf("ETag = %q", e.ETag.Value)
	}
}
