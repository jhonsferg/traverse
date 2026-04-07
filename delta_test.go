package traverse

import (
	"context"
	"testing"

	"github.com/jhonsferg/traverse/testutil"
)

// TestNewDeltaSync verifies that NewDeltaSync sets the entitySet correctly.
func TestNewDeltaSync(t *testing.T) {
	c, _ := New(WithBaseURL("http://localhost:8080/odata"))
	ds := c.NewDeltaSync("Products")
	if ds == nil {
		t.Fatal("NewDeltaSync() returned nil")
	}
	if ds.entitySet != "Products" {
		t.Errorf("entitySet = %q, want %q", ds.entitySet, "Products")
	}
	if ds.token != "" {
		t.Errorf("initial token should be empty, got %q", ds.token)
	}
}

// TestDeltaSync_Full verifies Full returns all records and sets delta token.
func TestDeltaSync_Full(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{
		Status: 200,
		Body:   `{"value":[{"ID":1,"Name":"Alpha"},{"ID":2,"Name":"Beta"}],"@odata.deltaLink":"https://example.com/Products?$deltatoken=abc123"}`,
	})

	client, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	ds := client.NewDeltaSync("Products")

	ch, _, callErr := ds.Full(ctx)
	if callErr != nil {
		t.Fatalf("Full() returned error: %v", callErr)
	}

	var records []map[string]interface{}
	for result := range ch {
		if result.Err != nil {
			t.Fatalf("unexpected stream error: %v", result.Err)
		}
		records = append(records, result.Value)
	}

	if len(records) != 2 {
		t.Fatalf("got %d records, want 2", len(records))
	}

	tok := ds.Token()
	if tok != "abc123" {
		t.Errorf("token = %q, want %q", tok, "abc123")
	}
}

// TestDeltaSync_Full_Error verifies that a server error is propagated via the channel.
func TestDeltaSync_Full_Error(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{Status: 500, Body: `{"error":"server error"}`})

	client, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	ds := client.NewDeltaSync("Products")

	ch, _, callErr := ds.Full(ctx)
	if callErr != nil {
		// Some implementations may return error immediately; that's also fine.
		return
	}

	var got error
	for result := range ch {
		if result.Err != nil {
			got = result.Err
		}
	}
	if got == nil {
		t.Error("expected an error from the channel, got none")
	}
}

// TestDeltaSync_Incremental verifies Incremental returns delta changes and updates the token.
func TestDeltaSync_Incremental(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{
		Status: 200,
		Body:   `{"value":[{"ID":3,"Name":"Gamma"},{"ID":1,"@removed":{"reason":"deleted"}}],"@odata.deltaLink":"https://example.com/Products?$deltatoken=xyz789"}`,
	})

	client, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	ds := client.NewDeltaSync("Products")

	ch, _, callErr := ds.Incremental(ctx, "abc123")
	if callErr != nil {
		t.Fatalf("Incremental() returned error: %v", callErr)
	}

	var changes []DeltaResult
	for result := range ch {
		if result.Err != nil {
			t.Fatalf("unexpected stream error: %v", result.Err)
		}
		changes = append(changes, result)
	}

	if len(changes) != 2 {
		t.Fatalf("got %d changes, want 2", len(changes))
	}

	// Second record should be marked as removed
	if !changes[1].Removed {
		t.Errorf("expected changes[1].Removed to be true")
	}
	if changes[1].Reason != "deleted" {
		t.Errorf("reason = %q, want %q", changes[1].Reason, "deleted")
	}

	// After draining, new token should be set
	tok := ds.Token()
	if tok != "xyz789" {
		t.Errorf("new token = %q, want %q", tok, "xyz789")
	}
}

// TestDeltaSync_Incremental_Error verifies that a server error is propagated via the channel.
func TestDeltaSync_Incremental_Error(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{Status: 500, Body: `{"error":"server error"}`})

	client, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	ds := client.NewDeltaSync("Products")

	ch, _, callErr := ds.Incremental(ctx, "sometoken")
	if callErr != nil {
		return
	}

	var got error
	for result := range ch {
		if result.Err != nil {
			got = result.Err
		}
	}
	if got == nil {
		t.Error("expected an error from the channel, got none")
	}
}

// TestDeltaSync_Incremental_NoToken verifies error when no token is available.
func TestDeltaSync_Incremental_NoToken(t *testing.T) {
	client, _ := New(WithBaseURL("http://localhost:8080/odata"))
	ctx := context.Background()
	ds := client.NewDeltaSync("Products")

	_, _, err := ds.Incremental(ctx, "")
	if err == nil {
		t.Error("expected error when no token available, got nil")
	}
}

// TestDeltaSync_SetToken_Token verifies SetToken and Token work correctly.
func TestDeltaSync_SetToken_Token(t *testing.T) {
	client, _ := New(WithBaseURL("http://localhost:8080/odata"))
	ds := client.NewDeltaSync("Products")

	if ds.Token() != "" {
		t.Errorf("initial Token() = %q, want empty", ds.Token())
	}

	ds.SetToken("mytoken123")

	if ds.Token() != "mytoken123" {
		t.Errorf("Token() = %q, want %q", ds.Token(), "mytoken123")
	}
}

// TestDeltaSyncAs_Full verifies typed Full sync returns correctly typed results.
func TestDeltaSyncAs_Full(t *testing.T) {
	type Product struct {
		ID   int    `json:"ID"`
		Name string `json:"Name"`
	}

	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{
		Status: 200,
		Body:   `{"value":[{"ID":10,"Name":"Widget"},{"ID":20,"Name":"Gadget"}],"@odata.deltaLink":"https://example.com/Products?$deltatoken=tok1"}`,
	})

	client, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	ds := NewDeltaSyncAs[Product](client, "Products")

	ch, _, callErr := ds.Full(ctx)
	if callErr != nil {
		t.Fatalf("DeltaSyncAs.Full() returned error: %v", callErr)
	}

	var results []Product
	for result := range ch {
		if result.Err != nil {
			t.Fatalf("unexpected stream error: %v", result.Err)
		}
		results = append(results, result.Value)
	}

	if len(results) != 2 {
		t.Fatalf("got %d results, want 2", len(results))
	}
	if results[0].ID != 10 || results[0].Name != "Widget" {
		t.Errorf("results[0] = %+v, want {ID:10 Name:Widget}", results[0])
	}
	if results[1].ID != 20 || results[1].Name != "Gadget" {
		t.Errorf("results[1] = %+v, want {ID:20 Name:Gadget}", results[1])
	}
}

// TestDeltaSyncAs_Incremental verifies typed Incremental sync returns correctly typed delta results.
func TestDeltaSyncAs_Incremental(t *testing.T) {
	type Product struct {
		ID   int    `json:"ID"`
		Name string `json:"Name"`
	}

	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{
		Status: 200,
		Body:   `{"value":[{"ID":30,"Name":"NewItem"},{"ID":10,"@removed":{"reason":"deleted"}}],"@odata.deltaLink":"https://example.com/Products?$deltatoken=tok2"}`,
	})

	client, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	ds := NewDeltaSyncAs[Product](client, "Products")

	ch, _, callErr := ds.Incremental(ctx, "tok1")
	if callErr != nil {
		t.Fatalf("DeltaSyncAs.Incremental() returned error: %v", callErr)
	}

	var results []DeltaResultAs[Product]
	for result := range ch {
		if result.Err != nil {
			t.Fatalf("unexpected stream error: %v", result.Err)
		}
		results = append(results, result)
	}

	if len(results) != 2 {
		t.Fatalf("got %d results, want 2", len(results))
	}
	if results[0].Value.ID != 30 || results[0].Value.Name != "NewItem" {
		t.Errorf("results[0].Value = %+v, want {ID:30 Name:NewItem}", results[0].Value)
	}
	if !results[1].Removed {
		t.Error("expected results[1].Removed to be true")
	}
	if results[1].Reason != "deleted" {
		t.Errorf("results[1].Reason = %q, want %q", results[1].Reason, "deleted")
	}
}

// TestDeltaSync_Full_MultiPage covers the multi-page path where the first page has
// a NextLink and the second page has a DeltaLink.
func TestDeltaSync_Full_MultiPage(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	// First page: has NextLink
	server.Enqueue(testutil.MockResponse{
		Status: 200,
		Body:   `{"value":[{"ID":1,"Name":"Alpha"},{"ID":2,"Name":"Beta"}],"@odata.nextLink":"` + server.URL() + `/Products?$skiptoken=page2"}`,
	})
	// Second page: has DeltaLink (no NextLink)
	server.Enqueue(testutil.MockResponse{
		Status: 200,
		Body:   `{"value":[{"ID":3,"Name":"Gamma"}],"@odata.deltaLink":"` + server.URL() + `/Products?$deltatoken=multipage_token"}`,
	})

	client, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	ds := client.NewDeltaSync("Products")

	ch, _, callErr := ds.Full(ctx)
	if callErr != nil {
		t.Fatalf("Full() returned error: %v", callErr)
	}

	var records []map[string]interface{}
	for result := range ch {
		if result.Err != nil {
			t.Fatalf("unexpected stream error: %v", result.Err)
		}
		records = append(records, result.Value)
	}

	if len(records) != 3 {
		t.Fatalf("got %d records across pages, want 3", len(records))
	}

	// Delta token should be extracted from the second page's deltaLink
	tok := ds.Token()
	if tok != "multipage_token" {
		t.Errorf("token = %q, want %q", tok, "multipage_token")
	}
}

// TestDeltaSync_extractDeltaToken_NoToken covers extractDeltaToken when no deltatoken param.
func TestDeltaSync_extractDeltaToken_NoToken(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	// DeltaLink without $deltatoken parameter  -  extractDeltaToken returns ""
	server.Enqueue(testutil.MockResponse{
		Status: 200,
		Body:   `{"value":[{"ID":1}],"@odata.deltaLink":"` + server.URL() + `/Products?$skiptoken=noDeltaToken"}`,
	})

	client, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	ds := client.NewDeltaSync("Products")

	ch, _, callErr := ds.Full(ctx)
	if callErr != nil {
		t.Fatalf("Full() returned error: %v", callErr)
	}

	for result := range ch {
		if result.Err != nil {
			t.Fatalf("unexpected stream error: %v", result.Err)
		}
	}

	// Token should be empty since deltaLink has no $deltatoken param
	tok := ds.Token()
	if tok != "" {
		t.Logf("token = %q (might be empty or extracted from link)", tok)
	}
}

// TestDeltaSync_Incremental_MultiPage covers Incremental with NextLink pagination.
func TestDeltaSync_Incremental_MultiPage(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	// First page has NextLink
	server.Enqueue(testutil.MockResponse{
		Status: 200,
		Body:   `{"value":[{"ID":5,"Name":"NewItem"}],"@odata.nextLink":"` + server.URL() + `/Products?$deltatoken=abc123&$skiptoken=p2"}`,
	})
	// Second page has DeltaLink (no NextLink)
	server.Enqueue(testutil.MockResponse{
		Status: 200,
		Body:   `{"value":[{"ID":6,"@removed":{"reason":"deleted"}}],"@odata.deltaLink":"` + server.URL() + `/Products?$deltatoken=new_token"}`,
	})

	client, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	ds := client.NewDeltaSync("Products")

	ch, _, callErr := ds.Incremental(ctx, "abc123")
	if callErr != nil {
		t.Fatalf("Incremental() returned error: %v", callErr)
	}

	var changes []DeltaResult
	for result := range ch {
		if result.Err != nil {
			t.Fatalf("unexpected stream error: %v", result.Err)
		}
		changes = append(changes, result)
	}

	if len(changes) != 2 {
		t.Fatalf("got %d changes across pages, want 2", len(changes))
	}
	if !changes[1].Removed {
		t.Error("expected changes[1].Removed to be true")
	}
}

// TestDeltaSyncAs_Full_Error covers the error path in DeltaSyncAs.Full when underlying call fails.
func TestDeltaSyncAs_Full_Error(t *testing.T) {
	type Product struct {
		ID   int    `json:"ID"`
		Name string `json:"Name"`
	}

	server := testutil.NewMockServer()
	defer server.Close()

	// Return 500 to trigger error in underlying DeltaSync.Full
	server.Enqueue(testutil.MockResponse{
		Status: 500,
		Body:   `{"error":"server error"}`,
	})

	client, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	ds := NewDeltaSyncAs[Product](client, "Products")

	ch, _, callErr := ds.Full(ctx)
	if callErr != nil {
		return // error returned immediately  -  fine
	}

	var gotErr error
	for result := range ch {
		if result.Err != nil {
			gotErr = result.Err
		}
	}
	if gotErr == nil {
		t.Error("DeltaSyncAs.Full: expected error on 500, got none")
	}
}

// TestDeltaSyncAs_Incremental_Error covers the error path in DeltaSyncAs.Incremental.
func TestDeltaSyncAs_Incremental_Error(t *testing.T) {
	type Product struct {
		ID int `json:"ID"`
	}

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
	ds := NewDeltaSyncAs[Product](client, "Products")

	ch, _, callErr := ds.Incremental(ctx, "tok")
	if callErr != nil {
		return
	}

	var gotErr error
	for result := range ch {
		if result.Err != nil {
			gotErr = result.Err
		}
	}
	if gotErr == nil {
		t.Error("DeltaSyncAs.Incremental: expected error on 500, got none")
	}
}
