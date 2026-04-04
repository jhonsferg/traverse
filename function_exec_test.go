package traverse

import (
	"context"
	"testing"

	"github.com/jhonsferg/traverse/testutil"
)

// TestFunctionBuilder_Execute_Success verifies that Execute makes a GET request and parses the result.
func TestFunctionBuilder_Execute_Success(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{
		Status: 200,
		Body:   `{"value":[{"ID":1,"Name":"Top"}]}`,
	})

	client, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	result, err := client.Function("GetTopProducts").
		Param("count", 1).
		Execute(ctx)
	if err != nil {
		t.Fatalf("Function.Execute() error = %v", err)
	}
	if result == nil {
		t.Fatal("Function.Execute() returned nil result")
	}

	reqs := server.RecordedRequests()
	if len(reqs) != 1 {
		t.Fatalf("expected 1 request, got %d", len(reqs))
	}
	if reqs[0].Method != "GET" {
		t.Errorf("expected GET request, got %s", reqs[0].Method)
	}
}

// TestFunctionBuilder_Execute_Error verifies that a 500 response returns an error.
func TestFunctionBuilder_Execute_Error(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{
		Status: 500,
		Body:   `{"error":"server error"}`,
	})

	client, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	_, err = client.Function("GetTopProducts").Execute(ctx)
	if err == nil {
		t.Error("Function.Execute() expected error on 500, got nil")
	}
}

// TestActionBuilder_Execute_Success verifies that Execute makes a POST request and parses the result.
func TestActionBuilder_Execute_Success(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{
		Status: 200,
		Body:   `{"approved":true,"message":"Order approved"}`,
	})

	client, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	result, err := client.Action("ApproveOrder").
		WithBody(map[string]interface{}{"orderID": 42}).
		Execute(ctx)
	if err != nil {
		t.Fatalf("Action.Execute() error = %v", err)
	}
	if result == nil {
		t.Fatal("Action.Execute() returned nil result")
	}

	reqs := server.RecordedRequests()
	if len(reqs) != 1 {
		t.Fatalf("expected 1 request, got %d", len(reqs))
	}
	if reqs[0].Method != "POST" {
		t.Errorf("expected POST request, got %s", reqs[0].Method)
	}
}

// TestActionBuilder_Execute_Error verifies that a 500 response returns an error.
func TestActionBuilder_Execute_Error(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{
		Status: 500,
		Body:   `{"error":"server error"}`,
	})

	client, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	_, err = client.Action("ApproveOrder").
		WithBody(map[string]interface{}{"orderID": 99}).
		Execute(ctx)
	if err == nil {
		t.Error("Action.Execute() expected error on 500, got nil")
	}
}

// TestActionBuilder_Execute_WithParams verifies that params-only (no body) actions work.
func TestActionBuilder_Execute_WithParams(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{
		Status: 200,
		Body:   `{"result":"ok"}`,
	})

	client, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	result, err := client.Action("SendNotification").
		Param("message", "hello").
		Execute(ctx)
	if err != nil {
		t.Fatalf("Action.Execute() error = %v", err)
	}
	if result == nil {
		t.Fatal("Action.Execute() returned nil")
	}
}

// TestActionBuilder_Execute_NoBody verifies that an action with no body or params works.
func TestActionBuilder_Execute_NoBody(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{
		Status: 200,
		Body:   `{"status":"done"}`,
	})

	client, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	result, err := client.Action("Ping").Execute(ctx)
	if err != nil {
		t.Fatalf("Action.Execute() error = %v", err)
	}
	if result == nil {
		t.Fatal("Action.Execute() returned nil")
	}
}

// TestFunctionImportBuilder_Execute_Success verifies FunctionImport execute makes a GET request.
func TestFunctionImportBuilder_Execute_Success(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{
		Status: 200,
		Body:   `{"value":[{"OrderID":"ORD-001","Total":99.99}]}`,
	})

	client, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	result, err := client.FunctionImport("GetTop10Orders").Execute(ctx)
	if err != nil {
		t.Fatalf("FunctionImport.Execute() error = %v", err)
	}
	if result == nil {
		t.Fatal("FunctionImport.Execute() returned nil")
	}

	reqs := server.RecordedRequests()
	if len(reqs) != 1 {
		t.Fatalf("expected 1 request, got %d", len(reqs))
	}
	if reqs[0].Method != "GET" {
		t.Errorf("expected GET request, got %s", reqs[0].Method)
	}
}

// TestFunctionImportBuilder_Execute_Error verifies FunctionImport returns error on 500.
func TestFunctionImportBuilder_Execute_Error(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{
		Status: 500,
		Body:   `{"error":"server error"}`,
	})

	client, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	_, err = client.FunctionImport("GetTop10Orders").Execute(ctx)
	if err == nil {
		t.Error("FunctionImport.Execute() expected error on 500, got nil")
	}
}

// TestExecuteFunctionAs_Success verifies the generic ExecuteFunctionAs returns typed result.
func TestExecuteFunctionAs_Success(t *testing.T) {
	type FuncResult struct {
		Count int    `json:"count"`
		Label string `json:"label"`
	}

	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{
		Status: 200,
		Body:   `{"count":42,"label":"top"}`,
	})

	client, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	result, err := ExecuteFunctionAs[FuncResult](
		client.Function("GetStats"),
		ctx,
	)
	if err != nil {
		t.Fatalf("ExecuteFunctionAs() error = %v", err)
	}
	if result.Count != 42 || result.Label != "top" {
		t.Errorf("ExecuteFunctionAs() = %+v, want {Count:42 Label:top}", result)
	}
}

// TestExecuteActionAs_Success verifies the generic ExecuteActionAs returns typed result.
func TestExecuteActionAs_Success(t *testing.T) {
	type ActionResult struct {
		Approved bool   `json:"approved"`
		Message  string `json:"message"`
	}

	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{
		Status: 200,
		Body:   `{"approved":true,"message":"done"}`,
	})

	client, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	result, err := ExecuteActionAs[ActionResult](
		client.Action("ApproveOrder").WithBody(map[string]interface{}{"id": 1}),
		ctx,
	)
	if err != nil {
		t.Fatalf("ExecuteActionAs() error = %v", err)
	}
	if !result.Approved || result.Message != "done" {
		t.Errorf("ExecuteActionAs() = %+v, want {Approved:true Message:done}", result)
	}
}

// TestExecuteFunctionImportAs_Success verifies generic ExecuteFunctionImportAs.
func TestExecuteFunctionImportAs_Success(t *testing.T) {
	type ImportResult struct {
		Total int `json:"total"`
	}

	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{
		Status: 200,
		Body:   `{"total":100}`,
	})

	client, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	result, err := ExecuteFunctionImportAs[ImportResult](
		client.FunctionImport("GetOrderCount"),
		ctx,
	)
	if err != nil {
		t.Fatalf("ExecuteFunctionImportAs() error = %v", err)
	}
	if result.Total != 100 {
		t.Errorf("ExecuteFunctionImportAs() = %+v, want {Total:100}", result)
	}
}
