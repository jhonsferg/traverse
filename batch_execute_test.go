package traverse

import (
	"context"
	"fmt"
	"mime/multipart"
	"net/textproto"
	"strings"
	"testing"

	"github.com/jhonsferg/traverse/testutil"
)

// buildBatchResponseBody constructs a multipart/mixed response body for batch tests.
func buildBatchResponseBody(boundary string, httpParts []string) string {
	var sb strings.Builder
	for _, p := range httpParts {
		sb.WriteString("--")
		sb.WriteString(boundary)
		sb.WriteString("\r\nContent-Type: application/http\r\n\r\n")
		sb.WriteString(p)
		sb.WriteString("\r\n")
	}
	sb.WriteString("--")
	sb.WriteString(boundary)
	sb.WriteString("--\r\n")
	return sb.String()
}

// buildBatchChangesetResponseBody wraps changeset parts inside the outer multipart batch.
func buildBatchChangesetResponseBody(outerBoundary, csBoundary string, csParts []string) string {
	var csBody strings.Builder
	for _, p := range csParts {
		csBody.WriteString("--")
		csBody.WriteString(csBoundary)
		csBody.WriteString("\r\nContent-Type: application/http\r\n\r\n")
		csBody.WriteString(p)
		csBody.WriteString("\r\n")
	}
	csBody.WriteString("--")
	csBody.WriteString(csBoundary)
	csBody.WriteString("--\r\n")

	var sb strings.Builder
	sb.WriteString("--")
	sb.WriteString(outerBoundary)
	sb.WriteString("\r\nContent-Type: multipart/mixed; boundary=")
	sb.WriteString(csBoundary)
	sb.WriteString("\r\n\r\n")
	sb.WriteString(csBody.String())
	sb.WriteString("--")
	sb.WriteString(outerBoundary)
	sb.WriteString("--\r\n")
	return sb.String()
}

func TestBatchExecute_Get(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	boundary := "batch_exec_get"
	part := "HTTP/1.1 200 OK\r\nContent-Type: application/json\r\n\r\n{\"ID\":1,\"Name\":\"Widget\"}"
	body := buildBatchResponseBody(boundary, []string{part})

	server.Enqueue(testutil.MockResponse{
		Status: 200,
		Headers: map[string]string{
			"Content-Type": fmt.Sprintf("multipart/mixed; boundary=%s", boundary),
		},
		Body: body,
	})

	c, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	resp, err := c.Batch().Get("Products", 1).Execute(context.Background())
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(resp.Results) != 1 {
		t.Fatalf("want 1 result, got %d", len(resp.Results))
	}
	if resp.Results[0].StatusCode != 200 {
		t.Errorf("want status 200, got %d", resp.Results[0].StatusCode)
	}
}

func TestBatchExecute_MultipleOps(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	boundary := "batch_exec_multi"
	parts := []string{
		"HTTP/1.1 200 OK\r\nContent-Type: application/json\r\n\r\n{\"ID\":1}",
		"HTTP/1.1 200 OK\r\nContent-Type: application/json\r\n\r\n{\"ID\":2}",
		"HTTP/1.1 204 No Content\r\n\r\n",
	}
	body := buildBatchResponseBody(boundary, parts)

	server.Enqueue(testutil.MockResponse{
		Status: 200,
		Headers: map[string]string{
			"Content-Type": fmt.Sprintf("multipart/mixed; boundary=%s", boundary),
		},
		Body: body,
	})

	c, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	resp, err := c.Batch().
		Get("Products", 1).
		Get("Products", 2).
		Delete("Orders", 99).
		Execute(context.Background())
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(resp.Results) != 3 {
		t.Fatalf("want 3 results, got %d", len(resp.Results))
	}
	if resp.Results[2].StatusCode != 204 {
		t.Errorf("want 204, got %d", resp.Results[2].StatusCode)
	}
}

func TestBatchExecute_Create(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	boundary := "batch_exec_create"
	part := "HTTP/1.1 201 Created\r\nContent-Type: application/json\r\n\r\n{\"ID\":42}"
	body := buildBatchResponseBody(boundary, []string{part})

	server.Enqueue(testutil.MockResponse{
		Status: 200,
		Headers: map[string]string{
			"Content-Type": fmt.Sprintf("multipart/mixed; boundary=%s", boundary),
		},
		Body: body,
	})

	c, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	data := map[string]interface{}{"Name": "New"}
	resp, err := c.Batch().Create("Products", data).Execute(context.Background())
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if resp.Results[0].StatusCode != 201 {
		t.Errorf("want 201, got %d", resp.Results[0].StatusCode)
	}
}

func TestBatchExecute_Update(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	boundary := "batch_exec_update"
	part := "HTTP/1.1 204 No Content\r\n\r\n"
	body := buildBatchResponseBody(boundary, []string{part})

	server.Enqueue(testutil.MockResponse{
		Status: 200,
		Headers: map[string]string{
			"Content-Type": fmt.Sprintf("multipart/mixed; boundary=%s", boundary),
		},
		Body: body,
	})

	c, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	data := map[string]interface{}{"Name": "Updated"}
	resp, err := c.Batch().Update("Products", 1, data).Execute(context.Background())
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if resp.Results[0].StatusCode != 204 {
		t.Errorf("want 204, got %d", resp.Results[0].StatusCode)
	}
}

func TestBatchExecute_Changeset(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	outerBoundary := "batch_exec_outer"
	csBoundary := "batch_exec_cs"
	csParts := []string{
		"HTTP/1.1 201 Created\r\nContent-Type: application/json\r\n\r\n{\"ID\":1}",
		"HTTP/1.1 204 No Content\r\n\r\n",
	}
	body := buildBatchChangesetResponseBody(outerBoundary, csBoundary, csParts)

	server.Enqueue(testutil.MockResponse{
		Status: 200,
		Headers: map[string]string{
			"Content-Type": fmt.Sprintf("multipart/mixed; boundary=%s", outerBoundary),
		},
		Body: body,
	})

	c, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	data := map[string]interface{}{"Name": "Txn"}
	resp, err := c.Batch().
		BeginChangeset("tx1").
		Create("Products", data).
		Delete("Orders", 5).
		EndChangeset().
		Execute(context.Background())
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(resp.Results) != 2 {
		t.Fatalf("want 2 results, got %d", len(resp.Results))
	}
}

func TestBatchExecute_AutoCloseChangeset(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	outerBoundary := "batch_exec_auto"
	csBoundary := "batch_exec_cs_auto"
	csParts := []string{
		"HTTP/1.1 201 Created\r\n\r\n{\"ID\":7}",
	}
	body := buildBatchChangesetResponseBody(outerBoundary, csBoundary, csParts)

	server.Enqueue(testutil.MockResponse{
		Status: 200,
		Headers: map[string]string{
			"Content-Type": fmt.Sprintf("multipart/mixed; boundary=%s", outerBoundary),
		},
		Body: body,
	})

	c, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	// No EndChangeset call — Execute should auto-close it.
	data := map[string]interface{}{"Name": "Auto"}
	resp, err := c.Batch().
		BeginChangeset("auto-cs").
		Create("Products", data).
		Execute(context.Background())
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(resp.Results) != 1 {
		t.Fatalf("want 1 result, got %d", len(resp.Results))
	}
}

func TestBatchExecute_ServerError(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{
		Status: 500,
		Body:   "Internal Server Error",
	})

	c, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	_, err = c.Batch().Get("Products", 1).Execute(context.Background())
	if err == nil {
		t.Fatal("expected error for 500 response, got nil")
	}
}

func TestBatchExecute_MissingContentType(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{
		Status: 200,
		Body:   "some body without content-type",
	})

	c, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	_, err = c.Batch().Get("Products", 1).Execute(context.Background())
	if err == nil {
		t.Fatal("expected error when Content-Type is missing")
	}
}

func TestBatchExecute_InvalidBoundary(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{
		Status: 200,
		Headers: map[string]string{
			"Content-Type": "multipart/mixed",
		},
		Body: "no boundary here",
	})

	c, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	_, err = c.Batch().Get("Products", 1).Execute(context.Background())
	if err == nil {
		t.Fatal("expected error when boundary is missing from Content-Type")
	}
}

func TestBatchExecuteStream_Success(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	boundary := "batch_stream_exec"
	parts := []string{
		"HTTP/1.1 200 OK\r\nContent-Type: application/json\r\n\r\n{\"ID\":1}",
		"HTTP/1.1 204 No Content\r\n\r\n",
	}
	body := buildBatchResponseBody(boundary, parts)

	server.Enqueue(testutil.MockResponse{
		Status: 200,
		Headers: map[string]string{
			"Content-Type": fmt.Sprintf("multipart/mixed; boundary=%s", boundary),
		},
		Body: body,
	})

	c, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	results := c.Batch().Get("Products", 1).Delete("Orders", 2).ExecuteStream(context.Background())
	var collected []BatchResult
	for r := range results {
		collected = append(collected, r)
	}
	if len(collected) != 2 {
		t.Fatalf("want 2 streamed results, got %d", len(collected))
	}
	if collected[0].StatusCode != 200 {
		t.Errorf("want 200, got %d", collected[0].StatusCode)
	}
}

func TestBatchExecuteStream_ServerError(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{
		Status: 503,
		Body:   "Service Unavailable",
	})

	c, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	results := c.Batch().Get("Products", 1).ExecuteStream(context.Background())
	var errs []error
	for r := range results {
		if r.Err != nil {
			errs = append(errs, r.Err)
		}
	}
	if len(errs) == 0 {
		t.Fatal("expected error result in stream for 503 response")
	}
}

func TestBatchExecuteStream_Changeset(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	outerBoundary := "batch_stream_cs_outer"
	csBoundary := "batch_stream_cs_inner"
	csParts := []string{
		"HTTP/1.1 201 Created\r\n\r\n{\"ID\":99}",
		"HTTP/1.1 204 No Content\r\n\r\n",
	}
	body := buildBatchChangesetResponseBody(outerBoundary, csBoundary, csParts)

	server.Enqueue(testutil.MockResponse{
		Status: 200,
		Headers: map[string]string{
			"Content-Type": fmt.Sprintf("multipart/mixed; boundary=%s", outerBoundary),
		},
		Body: body,
	})

	c, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	data := map[string]interface{}{"Name": "StreamTxn"}
	results := c.Batch().
		BeginChangeset("cs-stream").
		Create("Items", data).
		Delete("Items", 5).
		EndChangeset().
		ExecuteStream(context.Background())

	var collected []BatchResult
	for r := range results {
		if r.Err != nil {
			t.Fatalf("unexpected error in stream: %v", r.Err)
		}
		collected = append(collected, r)
	}
	if len(collected) != 2 {
		t.Fatalf("want 2 stream results, got %d", len(collected))
	}
}

func TestBatchSetBody_Error(t *testing.T) {
	op := &BatchOperation{}
	// channels cannot be marshalled to JSON — expect error.
	err := op.SetBody(make(chan int))
	if err == nil {
		t.Fatal("expected marshal error for channel value, got nil")
	}
}

func TestBatchSetBody_ValidStruct(t *testing.T) {
	op := &BatchOperation{}
	data := map[string]interface{}{"ID": 1, "Name": "Test"}
	err := op.SetBody(data)
	if err != nil {
		t.Fatalf("SetBody: %v", err)
	}
	if len(op.Body) == 0 {
		t.Error("SetBody should populate op.Body")
	}
}

func TestExtractBoundaryHelper(t *testing.T) {
	tests := []struct {
		ct   string
		want string
	}{
		{"multipart/mixed; boundary=batch_123", "batch_123"},
		{"multipart/mixed; boundary=\"quoted\"", "quoted"},
		{"multipart/mixed", ""},
		{"application/json", ""},
	}
	for _, tc := range tests {
		got := extractBoundary(tc.ct)
		if got != tc.want {
			t.Errorf("extractBoundary(%q) = %q, want %q", tc.ct, got, tc.want)
		}
	}
}

func TestBatchReleaseHeaders(t *testing.T) {
	h := acquireHeaders()
	if h == nil {
		t.Fatal("acquireHeaders returned nil")
	}
	h["X-Test"] = "value"
	releaseHeaders(h) // Should not panic; returns map to pool
}

func TestParseResponsePart_InvalidStatusLine(t *testing.T) {
	b := &BatchRequest{}
	h := make(textproto.MIMEHeader)
	h.Set("Content-Type", "application/http")

	var buf strings.Builder
	mw := multipart.NewWriter(&buf)
	_ = mw.SetBoundary("parse_test_boundary")
	pw, _ := mw.CreatePart(h)
	_, _ = fmt.Fprintf(pw, "BADLINE\r\n")
	_ = mw.Close()

	mr := multipart.NewReader(strings.NewReader(buf.String()), "parse_test_boundary")
	part, err := mr.NextPart()
	if err != nil {
		t.Fatalf("NextPart: %v", err)
	}

	// "BADLINE" has only one field — parseResponsePart returns an error for short status lines.
	_, err = b.parseResponsePart(part)
	if err == nil {
		t.Fatal("expected error for invalid status line, got nil")
	}
}
