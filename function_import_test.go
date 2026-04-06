package traverse

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/jhonsferg/traverse/testutil"
)

// ---------------------------------------------------------------------------
// FunctionImportBuilder.Invoke - GET requests
// ---------------------------------------------------------------------------

// TestFunctionImportInvoke_GET_NoParams verifies GET FunctionImport with no params.
func TestFunctionImportInvoke_GET_NoParams(t *testing.T) {
	type Result struct {
		Count int `json:"count"`
	}

	server := testutil.NewMockServer()
	defer server.Close()
	server.Enqueue(testutil.MockResponse{Status: 200, Body: `{"count":42}`})

	c, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatal(err)
	}

	var got Result
	if err := c.FunctionImport("GetOrderCount").Invoke(context.Background(), &got); err != nil {
		t.Fatalf("Invoke() error = %v", err)
	}
	if got.Count != 42 {
		t.Errorf("Count = %d, want 42", got.Count)
	}

	reqs := server.RecordedRequests()
	if len(reqs) != 1 {
		t.Fatalf("expected 1 request, got %d", len(reqs))
	}
	if reqs[0].Method != "GET" {
		t.Errorf("method = %s, want GET", reqs[0].Method)
	}
	if !strings.Contains(reqs[0].Path, "GetOrderCount()") {
		t.Errorf("URL %q does not contain GetOrderCount()", reqs[0].Path)
	}
}

// TestFunctionImportInvoke_GET_StringParam verifies GET with string parameter.
func TestFunctionImportInvoke_GET_StringParam(t *testing.T) {
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
	err = c.FunctionImport("FindProduct").
		Param("category", "Electronics").
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
	if !strings.Contains(url, "FindProduct(") {
		t.Errorf("URL %q does not contain FindProduct(", url)
	}
	if !strings.Contains(url, "category='Electronics'") {
		t.Errorf("URL %q missing category parameter", url)
	}
}

// TestFunctionImportInvoke_GET_MultipleParams verifies GET with multiple parameters.
func TestFunctionImportInvoke_GET_MultipleParams(t *testing.T) {
	type Result struct {
		Price float64 `json:"price"`
	}

	server := testutil.NewMockServer()
	defer server.Close()
	server.Enqueue(testutil.MockResponse{Status: 200, Body: `{"price":99.99}`})

	c, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatal(err)
	}

	var got Result
	err = c.FunctionImport("GetDiscountedPrice").
		Param("basePrice", 100).
		Param("discountPercent", 20).
		Param("applyTax", true).
		Invoke(context.Background(), &got)
	if err != nil {
		t.Fatalf("Invoke() error = %v", err)
	}
	if got.Price != 99.99 {
		t.Errorf("Price = %v, want 99.99", got.Price)
	}

	reqs := server.RecordedRequests()
	if len(reqs) != 1 {
		t.Fatalf("expected 1 request, got %d", len(reqs))
	}
	url := reqs[0].Path
	if !strings.Contains(url, "basePrice=100") {
		t.Errorf("URL %q missing basePrice parameter", url)
	}
	if !strings.Contains(url, "discountPercent=20") {
		t.Errorf("URL %q missing discountPercent parameter", url)
	}
	if !strings.Contains(url, "applyTax=true") {
		t.Errorf("URL %q missing applyTax parameter", url)
	}
}

// ---------------------------------------------------------------------------
// FunctionImportBuilder.Invoke - POST requests
// ---------------------------------------------------------------------------

// TestFunctionImportInvoke_POST_NoParams verifies POST FunctionImport with no params.
func TestFunctionImportInvoke_POST_NoParams(t *testing.T) {
	type Result struct {
		Status string `json:"status"`
	}

	server := testutil.NewMockServer()
	defer server.Close()
	server.Enqueue(testutil.MockResponse{Status: 200, Body: `{"status":"completed"}`})

	c, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatal(err)
	}

	var got Result
	err = c.FunctionImport("ProcessQueue").
		Method("POST").
		Invoke(context.Background(), &got)
	if err != nil {
		t.Fatalf("Invoke() error = %v", err)
	}
	if got.Status != "completed" {
		t.Errorf("Status = %q, want completed", got.Status)
	}

	reqs := server.RecordedRequests()
	if len(reqs) != 1 {
		t.Fatalf("expected 1 request, got %d", len(reqs))
	}
	if reqs[0].Method != "POST" {
		t.Errorf("method = %s, want POST", reqs[0].Method)
	}
	if !strings.Contains(reqs[0].Path, "ProcessQueue") {
		t.Errorf("URL %q does not contain ProcessQueue", reqs[0].Path)
	}
}

// TestFunctionImportInvoke_POST_WithParams verifies POST params are sent as JSON body.
func TestFunctionImportInvoke_POST_WithParams(t *testing.T) {
	type Result struct {
		Message string `json:"message"`
	}

	server := testutil.NewMockServer()
	defer server.Close()
	server.Enqueue(testutil.MockResponse{Status: 200, Body: `{"message":"created"}`})

	c, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatal(err)
	}

	var got Result
	err = c.FunctionImport("CreateOrder").
		Method("POST").
		Param("customerID", 123).
		Param("amount", 500.50).
		Invoke(context.Background(), &got)
	if err != nil {
		t.Fatalf("Invoke() error = %v", err)
	}
	if got.Message != "created" {
		t.Errorf("Message = %q, want created", got.Message)
	}

	reqs := server.RecordedRequests()
	if len(reqs) != 1 {
		t.Fatalf("expected 1 request, got %d", len(reqs))
	}

	var body map[string]interface{}
	if err := json.Unmarshal(reqs[0].Body, &body); err != nil {
		t.Fatalf("body is not valid JSON: %v", err)
	}
	if body["customerID"] != float64(123) {
		t.Errorf("body[customerID] = %v, want 123", body["customerID"])
	}
	if body["amount"] != 500.50 {
		t.Errorf("body[amount] = %v, want 500.50", body["amount"])
	}
}

// TestFunctionImportInvoke_POST_ContentType verifies POST sets correct Content-Type.
func TestFunctionImportInvoke_POST_ContentType(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()
	server.Enqueue(testutil.MockResponse{Status: 200, Body: `{}`})

	c, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatal(err)
	}

	var got interface{}
	err = c.FunctionImport("DoSomething").
		Method("POST").
		Param("test", "value").
		Invoke(context.Background(), &got)
	if err != nil {
		t.Fatalf("Invoke() error = %v", err)
	}

	reqs := server.RecordedRequests()
	if len(reqs) != 1 {
		t.Fatalf("expected 1 request, got %d", len(reqs))
	}

	contentType := reqs[0].Headers.Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", contentType)
	}
}

// ---------------------------------------------------------------------------
// FunctionImportBuilder.InvokeCollection
// ---------------------------------------------------------------------------

// TestFunctionImportInvokeCollection_GET verifies GET collection response unmarshaling.
func TestFunctionImportInvokeCollection_GET(t *testing.T) {
	type Product struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}

	server := testutil.NewMockServer()
	defer server.Close()
	server.Enqueue(testutil.MockResponse{Status: 200, Body: `{"d":{"results":[{"id":1,"name":"widget"},{"id":2,"name":"gadget"}]}}`})

	c, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatal(err)
	}

	var got []Product
	err = c.FunctionImport("GetProducts").InvokeCollection(context.Background(), &got)
	if err != nil {
		t.Fatalf("InvokeCollection() error = %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("len(got) = %d, want 2", len(got))
	}
	if got[0].ID != 1 || got[0].Name != "widget" {
		t.Errorf("got[0] = %v, want {ID:1 Name:widget}", got[0])
	}
	if got[1].ID != 2 || got[1].Name != "gadget" {
		t.Errorf("got[1] = %v, want {ID:2 Name:gadget}", got[1])
	}
}

// TestFunctionImportInvokeCollection_POST verifies POST collection response.
func TestFunctionImportInvokeCollection_POST(t *testing.T) {
	type Order struct {
		OrderID    int     `json:"orderID"`
		CustomerID int     `json:"customerID"`
		Total      float64 `json:"total"`
	}

	server := testutil.NewMockServer()
	defer server.Close()
	server.Enqueue(testutil.MockResponse{Status: 200, Body: `{"results":[{"orderID":1001,"customerID":100,"total":250.00},{"orderID":1002,"customerID":101,"total":350.50}]}`})

	c, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatal(err)
	}

	var got []Order
	err = c.FunctionImport("SearchOrders").
		Method("POST").
		Param("query", "urgent").
		InvokeCollection(context.Background(), &got)
	if err != nil {
		t.Fatalf("InvokeCollection() error = %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("len(got) = %d, want 2", len(got))
	}
	if got[0].OrderID != 1001 {
		t.Errorf("got[0].OrderID = %d, want 1001", got[0].OrderID)
	}
	if got[1].Total != 350.50 {
		t.Errorf("got[1].Total = %v, want 350.50", got[1].Total)
	}
}

// TestFunctionImportInvokeCollection_ValueField verifies handling of "value" field.
func TestFunctionImportInvokeCollection_ValueField(t *testing.T) {
	type Item struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}

	server := testutil.NewMockServer()
	defer server.Close()
	server.Enqueue(testutil.MockResponse{Status: 200, Body: `{"value":[{"id":10,"name":"item1"},{"id":20,"name":"item2"}]}`})

	c, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatal(err)
	}

	var got []Item
	err = c.FunctionImport("GetItems").InvokeCollection(context.Background(), &got)
	if err != nil {
		t.Fatalf("InvokeCollection() error = %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("len(got) = %d, want 2", len(got))
	}
	if got[0].ID != 10 {
		t.Errorf("got[0].ID = %d, want 10", got[0].ID)
	}
}

// TestFunctionImportInvokeCollection_EmptyCollection verifies empty collection response.
func TestFunctionImportInvokeCollection_Empty(t *testing.T) {
	type Item struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}

	server := testutil.NewMockServer()
	defer server.Close()
	server.Enqueue(testutil.MockResponse{Status: 200, Body: `{"results":[]}`})

	c, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatal(err)
	}

	var got []Item
	err = c.FunctionImport("GetItems").InvokeCollection(context.Background(), &got)
	if err != nil {
		t.Fatalf("InvokeCollection() error = %v", err)
	}

	if len(got) != 0 {
		t.Fatalf("len(got) = %d, want 0", len(got))
	}
}

// ---------------------------------------------------------------------------
// FunctionImportBuilder.Execute
// ---------------------------------------------------------------------------

// TestFunctionImportExecute_GET verifies Execute returns raw map for GET.
func TestFunctionImportExecute_GET(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()
	server.Enqueue(testutil.MockResponse{Status: 200, Body: `{"status":"ok"}`})

	c, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatal(err)
	}

	result, err := c.FunctionImport("GetStatus").Execute(context.Background())
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if result["status"] != "ok" {
		t.Errorf("result[status] = %v, want ok", result["status"])
	}
}

// TestFunctionImportExecute_POST verifies Execute returns raw map for POST.
func TestFunctionImportExecute_POST(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()
	server.Enqueue(testutil.MockResponse{Status: 200, Body: `{"id":123,"status":"created"}`})

	c, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatal(err)
	}

	result, err := c.FunctionImport("Create").
		Method("POST").
		Param("name", "test").
		Execute(context.Background())
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if result["id"] != float64(123) {
		t.Errorf("result[id] = %v, want 123", result["id"])
	}
	if result["status"] != "created" {
		t.Errorf("result[status] = %v, want created", result["status"])
	}
}

// ---------------------------------------------------------------------------
// FunctionImportBuilder.Invoke - Error handling
// ---------------------------------------------------------------------------

// TestFunctionImportInvoke_HTTP500 verifies error propagation for 500 status.
func TestFunctionImportInvoke_HTTP500(t *testing.T) {
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
	err = c.FunctionImport("Broken").Invoke(context.Background(), &got)
	if err == nil {
		t.Error("expected error on 500, got nil")
	}
}

// TestFunctionImportInvoke_HTTP404 verifies 404 error propagation.
func TestFunctionImportInvoke_HTTP404(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()
	server.Enqueue(testutil.MockResponse{Status: 404, Body: `{"error":"not found"}`})

	c, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatal(err)
	}

	var got map[string]interface{}
	err = c.FunctionImport("Missing").Invoke(context.Background(), &got)
	if err == nil {
		t.Error("expected error on 404, got nil")
	}
}

// ---------------------------------------------------------------------------
// FunctionImportBuilder.Invoke - Nil result
// ---------------------------------------------------------------------------

// TestFunctionImportInvoke_NilResult verifies Invoke accepts nil result.
func TestFunctionImportInvoke_NilResult(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()
	server.Enqueue(testutil.MockResponse{Status: 200, Body: `{}`})

	c, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatal(err)
	}

	if err := c.FunctionImport("Notify").Invoke(context.Background(), nil); err != nil {
		t.Fatalf("Invoke(nil) error = %v", err)
	}
}

// ---------------------------------------------------------------------------
// FunctionImportBuilder - Method chaining
// ---------------------------------------------------------------------------

// TestFunctionImportBuilder_MethodChaining verifies fluent API works.
func TestFunctionImportBuilder_MethodChaining(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()
	server.Enqueue(testutil.MockResponse{Status: 200, Body: `{"result":"ok"}`})

	c, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatal(err)
	}

	var got map[string]interface{}
	err = c.FunctionImport("TestFunc").
		Param("a", 1).
		Param("b", "test").
		Method("GET").
		Invoke(context.Background(), &got)
	if err != nil {
		t.Fatalf("Invoke() error = %v", err)
	}

	if got["result"] != "ok" {
		t.Errorf("result = %v, want ok", got["result"])
	}
}

// TestFunctionImportBuilder_Construction verifies proper initialization.
func TestFunctionImportBuilder_Construction(t *testing.T) {
	c, _ := New(WithBaseURL("http://localhost/odata"))
	fb := c.FunctionImport("TestImport")
	if fb == nil {
		t.Fatal("FunctionImport returned nil")
	}
	if fb.name != "TestImport" {
		t.Errorf("name = %q, want TestImport", fb.name)
	}
	if fb.method != "GET" {
		t.Errorf("method = %q, want GET", fb.method)
	}
}

// ---------------------------------------------------------------------------
// FunctionImportBuilder - Response wrapping
// ---------------------------------------------------------------------------

// TestFunctionImportInvoke_WrappedInD verifies handling of {"d":{...}} wrapper.
func TestFunctionImportInvoke_WrappedInD(t *testing.T) {
	type Result struct {
		Count int `json:"count"`
	}

	server := testutil.NewMockServer()
	defer server.Close()
	server.Enqueue(testutil.MockResponse{Status: 200, Body: `{"d":{"count":99}}`})

	c, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatal(err)
	}

	var got Result
	err = c.FunctionImport("GetStats").Invoke(context.Background(), &got)
	if err != nil {
		t.Fatalf("Invoke() error = %v", err)
	}
	if got.Count != 99 {
		t.Errorf("Count = %d, want 99", got.Count)
	}
}

// TestFunctionImportInvoke_BooleanParam verifies boolean parameter encoding in GET.
func TestFunctionImportInvoke_BooleanParam(t *testing.T) {
	type Result struct {
		Active bool `json:"active"`
	}

	server := testutil.NewMockServer()
	defer server.Close()
	server.Enqueue(testutil.MockResponse{Status: 200, Body: `{"active":true}`})

	c, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatal(err)
	}

	var got Result
	err = c.FunctionImport("CheckStatus").
		Param("includeArchived", false).
		Invoke(context.Background(), &got)
	if err != nil {
		t.Fatalf("Invoke() error = %v", err)
	}
	if !got.Active {
		t.Errorf("Active = %v, want true", got.Active)
	}

	reqs := server.RecordedRequests()
	if len(reqs) != 1 {
		t.Fatalf("expected 1 request, got %d", len(reqs))
	}
	if !strings.Contains(reqs[0].Path, "includeArchived=false") {
		t.Errorf("URL %q missing includeArchived=false", reqs[0].Path)
	}
}

// TestFunctionImportInvoke_FloatParam verifies float parameter encoding in GET.
func TestFunctionImportInvoke_FloatParam(t *testing.T) {
	type Result struct {
		Result float64 `json:"result"`
	}

	server := testutil.NewMockServer()
	defer server.Close()
	server.Enqueue(testutil.MockResponse{Status: 200, Body: `{"result":42.5}`})

	c, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatal(err)
	}

	var got Result
	err = c.FunctionImport("Calculate").
		Param("value", 85.0).
		Invoke(context.Background(), &got)
	if err != nil {
		t.Fatalf("Invoke() error = %v", err)
	}
	if got.Result != 42.5 {
		t.Errorf("Result = %v, want 42.5", got.Result)
	}

	reqs := server.RecordedRequests()
	if len(reqs) != 1 {
		t.Fatalf("expected 1 request, got %d", len(reqs))
	}
	if !strings.Contains(reqs[0].Path, "value=85") {
		t.Errorf("URL %q missing value=85", reqs[0].Path)
	}
}

// TestFunctionImportInvokeCollection_NilResults verifies InvokeCollection handles nil.
func TestFunctionImportInvokeCollection_NilResults(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()
	server.Enqueue(testutil.MockResponse{Status: 200, Body: `{"results":[]}`})

	c, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatal(err)
	}

	err = c.FunctionImport("GetItems").InvokeCollection(context.Background(), nil)
	if err != nil {
		t.Fatalf("InvokeCollection(nil) error = %v", err)
	}
}
