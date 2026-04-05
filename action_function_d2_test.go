package traverse

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/jhonsferg/traverse/testutil"
)

// ---------------------------------------------------------------------------
// FunctionBuilder.Invoke
// ---------------------------------------------------------------------------

// TestFunctionInvoke_NoParams verifies that Invoke with no params builds the right URL
// and unmarshals the response into the result struct.
func TestFunctionInvoke_NoParams(t *testing.T) {
	type Result struct {
		Count int `json:"count"`
	}

	server := testutil.NewMockServer()
	defer server.Close()
	server.Enqueue(testutil.MockResponse{Status: 200, Body: `{"count":7}`})

	c, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatal(err)
	}

	var got Result
	if err := c.Function("GetCount").Invoke(context.Background(), &got); err != nil {
		t.Fatalf("Invoke() error = %v", err)
	}
	if got.Count != 7 {
		t.Errorf("Count = %d, want 7", got.Count)
	}

	reqs := server.RecordedRequests()
	if len(reqs) != 1 {
		t.Fatalf("expected 1 request, got %d", len(reqs))
	}
	if reqs[0].Method != "GET" {
		t.Errorf("method = %s, want GET", reqs[0].Method)
	}
	if !strings.Contains(reqs[0].Path, "GetCount()") {
		t.Errorf("URL %q does not contain GetCount()", reqs[0].Path)
	}
}

// TestFunctionInvoke_MultipleParams verifies string, int, and bool parameters are
// encoded inline in the URL when Invoke is called.
func TestFunctionInvoke_MultipleParams(t *testing.T) {
	type Result struct {
		Name string `json:"name"`
	}

	server := testutil.NewMockServer()
	defer server.Close()
	server.Enqueue(testutil.MockResponse{Status: 200, Body: `{"name":"widget"}`})

	c, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatal(err)
	}

	var got Result
	err = c.Function("FindProduct").
		Param("category", "Electronics").
		Param("maxPrice", 500).
		Param("inStock", true).
		Invoke(context.Background(), &got)
	if err != nil {
		t.Fatalf("Invoke() error = %v", err)
	}
	if got.Name != "widget" {
		t.Errorf("Name = %q, want widget", got.Name)
	}

	reqs := server.RecordedRequests()
	if len(reqs) != 1 {
		t.Fatalf("expected 1 request, got %d", len(reqs))
	}
	url := reqs[0].Path
	if !strings.Contains(url, "category='Electronics'") {
		t.Errorf("URL %q missing category parameter", url)
	}
	if !strings.Contains(url, "maxPrice=500") {
		t.Errorf("URL %q missing maxPrice parameter", url)
	}
	if !strings.Contains(url, "inStock=true") {
		t.Errorf("URL %q missing inStock parameter", url)
	}
}

// TestFunctionInvoke_Error verifies Invoke propagates HTTP errors.
func TestFunctionInvoke_Error(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()
	// relay retries 500 three times
	server.Enqueue(testutil.MockResponse{Status: 500, Body: `{"error":"fail"}`})
	server.Enqueue(testutil.MockResponse{Status: 500, Body: `{"error":"fail"}`})
	server.Enqueue(testutil.MockResponse{Status: 500, Body: `{"error":"fail"}`})

	c, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatal(err)
	}

	var got map[string]interface{}
	err = c.Function("Broken").Invoke(context.Background(), &got)
	if err == nil {
		t.Error("expected error on 500, got nil")
	}
}

// ---------------------------------------------------------------------------
// FunctionBuilder.Invoke - bound to entity set
// ---------------------------------------------------------------------------

// TestFunctionInvoke_BoundToEntitySet verifies BoundFunction constructs the URL as
// /EntitySet/FunctionName(params).
func TestFunctionInvoke_BoundToEntitySet(t *testing.T) {
	type Result struct {
		Discount float64 `json:"discount"`
	}

	server := testutil.NewMockServer()
	defer server.Close()
	server.Enqueue(testutil.MockResponse{Status: 200, Body: `{"discount":0.15}`})

	c, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatal(err)
	}

	var got Result
	err = c.From("Products").
		BoundFunction("Namespace.GetDiscount").
		Param("tier", "gold").
		Invoke(context.Background(), &got)
	if err != nil {
		t.Fatalf("BoundFunction.Invoke() error = %v", err)
	}
	if got.Discount != 0.15 {
		t.Errorf("Discount = %v, want 0.15", got.Discount)
	}

	reqs := server.RecordedRequests()
	if len(reqs) != 1 {
		t.Fatalf("expected 1 request, got %d", len(reqs))
	}
	url := reqs[0].Path
	if !strings.Contains(url, "Products/Namespace.GetDiscount(") {
		t.Errorf("URL %q does not contain Products/Namespace.GetDiscount(", url)
	}
	if !strings.Contains(url, "tier='gold'") {
		t.Errorf("URL %q missing tier parameter", url)
	}
}

// ---------------------------------------------------------------------------
// ActionBuilder.Invoke
// ---------------------------------------------------------------------------

// TestActionInvoke_NoBody verifies an action with no body sends an empty POST.
func TestActionInvoke_NoBody(t *testing.T) {
	type Result struct {
		OK bool `json:"ok"`
	}

	server := testutil.NewMockServer()
	defer server.Close()
	server.Enqueue(testutil.MockResponse{Status: 200, Body: `{"ok":true}`})

	c, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatal(err)
	}

	var got Result
	if err := c.Action("Ping").Invoke(context.Background(), &got); err != nil {
		t.Fatalf("Invoke() error = %v", err)
	}
	if !got.OK {
		t.Errorf("OK = false, want true")
	}

	reqs := server.RecordedRequests()
	if len(reqs) != 1 {
		t.Fatalf("expected 1 request, got %d", len(reqs))
	}
	if reqs[0].Method != "POST" {
		t.Errorf("method = %s, want POST", reqs[0].Method)
	}
}

// TestActionInvoke_WithParams verifies params become the JSON body.
func TestActionInvoke_WithParams(t *testing.T) {
	type Result struct {
		Status string `json:"status"`
	}

	server := testutil.NewMockServer()
	defer server.Close()
	server.Enqueue(testutil.MockResponse{Status: 200, Body: `{"status":"approved"}`})

	c, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatal(err)
	}

	var got Result
	err = c.Action("ApproveOrder").
		Param("orderID", 42).
		Param("approver", "manager").
		Invoke(context.Background(), &got)
	if err != nil {
		t.Fatalf("Invoke() error = %v", err)
	}
	if got.Status != "approved" {
		t.Errorf("Status = %q, want approved", got.Status)
	}

	reqs := server.RecordedRequests()
	if len(reqs) != 1 {
		t.Fatalf("expected 1 request, got %d", len(reqs))
	}
	// Verify the body contains the params as JSON
	var body map[string]interface{}
	if err := json.Unmarshal(reqs[0].Body, &body); err != nil {
		t.Fatalf("body is not valid JSON: %v", err)
	}
	if body["approver"] != "manager" {
		t.Errorf("body[approver] = %v, want manager", body["approver"])
	}
}

// TestActionInvoke_NilResult verifies Invoke accepts nil result (fire-and-forget).
func TestActionInvoke_NilResult(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()
	server.Enqueue(testutil.MockResponse{Status: 200, Body: `{}`})

	c, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatal(err)
	}

	if err := c.Action("Notify").Invoke(context.Background(), nil); err != nil {
		t.Fatalf("Invoke(nil) error = %v", err)
	}
}

// TestActionInvoke_Error verifies Invoke propagates HTTP errors.
func TestActionInvoke_Error(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()
	// relay retries 500 three times
	server.Enqueue(testutil.MockResponse{Status: 500, Body: `{"error":"fail"}`})
	server.Enqueue(testutil.MockResponse{Status: 500, Body: `{"error":"fail"}`})
	server.Enqueue(testutil.MockResponse{Status: 500, Body: `{"error":"fail"}`})

	c, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatal(err)
	}

	var got map[string]interface{}
	if err := c.Action("Broken").Invoke(context.Background(), &got); err == nil {
		t.Error("expected error on 500, got nil")
	}
}

// ---------------------------------------------------------------------------
// ActionBuilder.Invoke - bound to entity set
// ---------------------------------------------------------------------------

// TestActionInvoke_BoundToEntitySet verifies BoundAction constructs the URL as
// /EntitySet/ActionName and sends params as the JSON body.
func TestActionInvoke_BoundToEntitySet(t *testing.T) {
	type Result struct {
		Updated int `json:"updated"`
	}

	server := testutil.NewMockServer()
	defer server.Close()
	server.Enqueue(testutil.MockResponse{Status: 200, Body: `{"updated":5}`})

	c, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatal(err)
	}

	var got Result
	err = c.From("Products").
		BoundAction("Namespace.BulkDiscount").
		Param("percent", 10).
		Invoke(context.Background(), &got)
	if err != nil {
		t.Fatalf("BoundAction.Invoke() error = %v", err)
	}
	if got.Updated != 5 {
		t.Errorf("Updated = %d, want 5", got.Updated)
	}

	reqs := server.RecordedRequests()
	if len(reqs) != 1 {
		t.Fatalf("expected 1 request, got %d", len(reqs))
	}
	url := reqs[0].Path
	if !strings.Contains(url, "Products/Namespace.BulkDiscount") {
		t.Errorf("URL %q does not contain Products/Namespace.BulkDiscount", url)
	}
	if reqs[0].Method != "POST" {
		t.Errorf("method = %s, want POST", reqs[0].Method)
	}

	var body map[string]interface{}
	if err := json.Unmarshal(reqs[0].Body, &body); err != nil {
		t.Fatalf("body is not valid JSON: %v", err)
	}
}

// ---------------------------------------------------------------------------
// BoundFunction / BoundAction on QueryBuilder construction tests
// ---------------------------------------------------------------------------

// TestBoundFunction_Construction verifies BoundFunction propagates entity set as basePath.
func TestBoundFunction_Construction(t *testing.T) {
	c, _ := New(WithBaseURL("http://localhost/odata"))
	fb := c.From("Orders").BoundFunction("Namespace.GetSummary")
	if fb == nil {
		t.Fatal("BoundFunction returned nil")
	}
	if fb.name != "Namespace.GetSummary" {
		t.Errorf("name = %q, want Namespace.GetSummary", fb.name)
	}
	if fb.basePath != "Orders" {
		t.Errorf("basePath = %q, want Orders", fb.basePath)
	}
}

// TestBoundAction_Construction verifies BoundAction propagates entity set as basePath.
func TestBoundAction_Construction(t *testing.T) {
	c, _ := New(WithBaseURL("http://localhost/odata"))
	ab := c.From("Orders").BoundAction("Namespace.Cancel")
	if ab == nil {
		t.Fatal("BoundAction returned nil")
	}
	if ab.name != "Namespace.Cancel" {
		t.Errorf("name = %q, want Namespace.Cancel", ab.name)
	}
	if ab.basePath != "Orders" {
		t.Errorf("basePath = %q, want Orders", ab.basePath)
	}
}

// TestUnboundFunction_BasePathEmpty verifies unbound functions have an empty basePath.
func TestUnboundFunction_BasePathEmpty(t *testing.T) {
	c, _ := New(WithBaseURL("http://localhost/odata"))
	fb := c.Function("GetGlobal")
	if fb.basePath != "" {
		t.Errorf("basePath = %q, want empty", fb.basePath)
	}
}

// TestUnboundAction_BasePathEmpty verifies unbound actions have an empty basePath.
func TestUnboundAction_BasePathEmpty(t *testing.T) {
	c, _ := New(WithBaseURL("http://localhost/odata"))
	ab := c.Action("GlobalNotify")
	if ab.basePath != "" {
		t.Errorf("basePath = %q, want empty", ab.basePath)
	}
}
