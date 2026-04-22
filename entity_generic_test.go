package traverse

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/jhonsferg/traverse/testutil"
)

type testProduct struct {
	ID   int    `json:"ID"`
	Name string `json:"Name"`
}

// TestRawMessageToStruct_Valid verifies that a valid JSON RawMessage is correctly decoded.
func TestRawMessageToStruct_Valid(t *testing.T) {
	raw := json.RawMessage(`{"ID":42,"Name":"Widget"}`)
	got, err := rawMessageToStruct[testProduct](raw)
	if err != nil {
		t.Fatalf("rawMessageToStruct() error = %v", err)
	}
	if got.ID != 42 || got.Name != "Widget" {
		t.Errorf("got %+v, want {ID:42 Name:Widget}", got)
	}
}

// TestRawMessageToStruct_Invalid verifies that invalid JSON returns an error.
func TestRawMessageToStruct_Invalid(t *testing.T) {
	raw := json.RawMessage(`{not valid json`)
	_, err := rawMessageToStruct[testProduct](raw)
	if err == nil {
		t.Error("rawMessageToStruct() expected error for invalid JSON, got nil")
	}
}

// TestCreateAsJson_Success verifies CreateJsonAs returns a typed entity from a 201 response.
func TestCreateAsJson_Success(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{
		Status: 201,
		Body:   `{"ID":1,"Name":"NewProduct"}`,
	})

	client, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	product, err := CreateJsonAs[testProduct](client, ctx, "Products", map[string]interface{}{"Name": "NewProduct"})
	if err != nil {
		t.Fatalf("CreateJsonAs() error = %v", err)
	}
	if product.ID != 1 || product.Name != "NewProduct" {
		t.Errorf("CreateJsonAs() = %+v, want {ID:1 Name:NewProduct}", product)
	}
}

// TestCreateAsJson_Error verifies CreateJsonAs returns an error on non-201 status.
func TestCreateAsJson_Error(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{
		Status: 400,
		Body:   `{"error":"bad request"}`,
	})

	client, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	_, err = CreateJsonAs[testProduct](client, ctx, "Products", map[string]interface{}{"Name": ""})
	if err == nil {
		t.Error("CreateJsonAs() expected error on 400, got nil")
	}
}

// TestUpdateAs_Success verifies UpdateAs returns nil on a 204 response.
func TestUpdateAs_Success(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{
		Status: 204,
		Body:   "",
	})

	client, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	err = UpdateAs[testProduct](client, ctx, "Products", 1, map[string]interface{}{"Name": "Updated"})
	if err != nil {
		t.Errorf("UpdateAs() unexpected error: %v", err)
	}
}

// TestFindByKeyAs_Success verifies FindByKeyAs returns a typed entity.
func TestFindByKeyAs_Success(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{
		Status: 200,
		Body:   `{"ID":5,"Name":"Found"}`,
	})

	client, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	qb := client.From("Products")
	product, err := FindByKeyAs[testProduct](qb, ctx, 5)
	if err != nil {
		t.Fatalf("FindByKeyAs() error = %v", err)
	}
	if product.ID != 5 || product.Name != "Found" {
		t.Errorf("FindByKeyAs() = %+v, want {ID:5 Name:Found}", product)
	}
}

// TestFindByKeyAs_Error verifies FindByKeyAs returns an error on a 404 response.
func TestFindByKeyAs_Error(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{
		Status: 404,
		Body:   `{"error":"not found"}`,
	})

	client, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	qb := client.From("Products")
	_, err = FindByKeyAs[testProduct](qb, ctx, 999)
	if err == nil {
		t.Error("FindByKeyAs() expected error on 404, got nil")
	}
}

// TestCollectAs_Success verifies CollectAs returns a typed slice.
func TestCollectAs_Success(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{
		Status: 200,
		Body:   `{"value":[{"ID":1,"Name":"Alpha"},{"ID":2,"Name":"Beta"}]}`,
	})

	client, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	qb := client.From("Products")
	results, err := CollectAs[testProduct](qb, ctx)
	if err != nil {
		t.Fatalf("CollectAs() error = %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("CollectAs() got %d results, want 2", len(results))
	}
	if results[0].ID != 1 || results[0].Name != "Alpha" {
		t.Errorf("results[0] = %+v, want {ID:1 Name:Alpha}", results[0])
	}
	if results[1].ID != 2 || results[1].Name != "Beta" {
		t.Errorf("results[1] = %+v, want {ID:2 Name:Beta}", results[1])
	}
}

// TestCollectAs_Empty verifies CollectAs returns an empty slice for empty value array.
func TestCollectAs_Empty(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{
		Status: 200,
		Body:   `{"value":[]}`,
	})

	client, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	qb := client.From("Products")
	results, err := CollectAs[testProduct](qb, ctx)
	if err != nil {
		t.Fatalf("CollectAs() error = %v", err)
	}
	if len(results) != 0 {
		t.Errorf("CollectAs() got %d results, want 0", len(results))
	}
}

// TestFirstAs_Success verifies FirstAs returns the first typed entity.
func TestFirstAs_Success(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{
		Status: 200,
		Body:   `{"value":[{"ID":7,"Name":"First"}]}`,
	})

	client, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	qb := client.From("Products")
	product, err := FirstAs[testProduct](qb, ctx)
	if err != nil {
		t.Fatalf("FirstAs() error = %v", err)
	}
	if product.ID != 7 || product.Name != "First" {
		t.Errorf("FirstAs() = %+v, want {ID:7 Name:First}", product)
	}
}

// TestFirstAs_NotFound verifies FirstAs returns ErrEntityNotFound for empty list.
func TestFirstAs_NotFound(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{
		Status: 200,
		Body:   `{"value":[]}`,
	})

	client, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	qb := client.From("Products")
	_, err = FirstAs[testProduct](qb, ctx)
	if !errors.Is(err, ErrEntityNotFound) {
		t.Errorf("FirstAs() error = %v, want ErrEntityNotFound", err)
	}
}

// TestStreamAs_Success verifies StreamAs yields correctly typed results.
func TestStreamAs_Success(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{
		Status: 200,
		Body:   `{"value":[{"ID":100,"Name":"Streamed1"},{"ID":200,"Name":"Streamed2"}]}`,
	})

	client, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	qb := client.From("Products")
	ch := StreamAs[testProduct](qb, ctx)

	var results []testProduct
	for result := range ch {
		if result.Err != nil {
			t.Fatalf("StreamAs() unexpected error: %v", result.Err)
		}
		results = append(results, result.Value)
	}

	if len(results) != 2 {
		t.Fatalf("StreamAs() got %d results, want 2", len(results))
	}
	if results[0].ID != 100 || results[0].Name != "Streamed1" {
		t.Errorf("results[0] = %+v, want {ID:100 Name:Streamed1}", results[0])
	}
	if results[1].ID != 200 || results[1].Name != "Streamed2" {
		t.Errorf("results[1] = %+v, want {ID:200 Name:Streamed2}", results[1])
	}
}

// TestStreamAs_WithError verifies StreamAs propagates server errors through the channel.
func TestStreamAs_WithError(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{
		Status: 500,
		Body:   `{"error":"internal server error"}`,
	})

	client, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	qb := client.From("Products")
	ch := StreamAs[testProduct](qb, ctx)

	var gotErr error
	for result := range ch {
		if result.Err != nil {
			gotErr = result.Err
		}
	}

	if gotErr == nil {
		t.Error("StreamAs() expected error for 500 response, got none")
	}
}
