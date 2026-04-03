package traverse

import (
	"fmt"
	"strings"
	"testing"
)

// TestBatchRequestConstruction tests basic batch request building.
func TestBatchRequestConstruction(t *testing.T) {
	mockClient := &Client{}
	batch := mockClient.Batch()

	if batch == nil {
		t.Fatalf("Batch() returned nil")
	}

	if batch.client != mockClient {
		t.Error("Client not set correctly")
	}

	if len(batch.ops) != 0 {
		t.Error("Initial ops should be empty")
	}
}

// TestBatchAddGet tests adding GET operations.
func TestBatchAddGet(t *testing.T) {
	mockClient := &Client{}
	batch := mockClient.Batch()

	batch.Get("Materials", "MAT001")
	batch.Get("Plants", "PLANT01")

	if len(batch.ops) != 2 {
		t.Errorf("Expected 2 ops, got %d", len(batch.ops))
	}

	if batch.ops[0].Method != "GET" {
		t.Errorf("Expected GET method, got %s", batch.ops[0].Method)
	}

	if !strings.Contains(batch.ops[0].URL, "Materials") {
		t.Errorf("Expected Materials in URL, got %s", batch.ops[0].URL)
	}
}

// TestBatchAddCreate tests adding POST (create) operations.
func TestBatchAddCreate(t *testing.T) {
	mockClient := &Client{}
	batch := mockClient.Batch()

	data := map[string]interface{}{
		"MatID": "MAT002",
		"Name":  "Aluminum",
	}

	batch.Create("Materials", data)

	if len(batch.ops) != 1 {
		t.Errorf("Expected 1 op, got %d", len(batch.ops))
	}

	if batch.ops[0].Method != "POST" {
		t.Errorf("Expected POST method, got %s", batch.ops[0].Method)
	}

	if batch.ops[0].Body == nil {
		t.Error("Body should not be nil")
	}
}

// TestBatchAddUpdate tests adding PATCH (update) operations.
func TestBatchAddUpdate(t *testing.T) {
	mockClient := &Client{}
	batch := mockClient.Batch()

	data := map[string]interface{}{
		"Name": "Updated Aluminum",
	}

	batch.Update("Materials", "MAT002", data)

	if len(batch.ops) != 1 {
		t.Errorf("Expected 1 op, got %d", len(batch.ops))
	}

	if batch.ops[0].Method != "PATCH" {
		t.Errorf("Expected PATCH method, got %s", batch.ops[0].Method)
	}

	if !strings.Contains(batch.ops[0].URL, "MAT002") {
		t.Errorf("Expected MAT002 in URL, got %s", batch.ops[0].URL)
	}
}

// TestBatchAddDelete tests adding DELETE operations.
func TestBatchAddDelete(t *testing.T) {
	mockClient := &Client{}
	batch := mockClient.Batch()

	batch.Delete("Materials", "MAT003")

	if len(batch.ops) != 1 {
		t.Errorf("Expected 1 op, got %d", len(batch.ops))
	}

	if batch.ops[0].Method != "DELETE" {
		t.Errorf("Expected DELETE method, got %s", batch.ops[0].Method)
	}

	if !strings.Contains(batch.ops[0].URL, "MAT003") {
		t.Errorf("Expected MAT003 in URL, got %s", batch.ops[0].URL)
	}
}

// TestBatchChangeset tests changeset tracking.
func TestBatchChangeset(t *testing.T) {
	mockClient := &Client{}
	batch := mockClient.Batch()

	// Add operation outside changeset
	batch.Get("Materials", "MAT001")

	// Start changeset
	batch.BeginChangeset("cs1")
	batch.Create("Materials", map[string]interface{}{"MatID": "MAT004"})
	batch.Update("Materials", "MAT005", map[string]interface{}{"Name": "Updated"})

	// End changeset
	batch.EndChangeset()

	// Add another operation outside changeset
	batch.Delete("Materials", "MAT006")

	if len(batch.ops) != 2 {
		t.Errorf("Expected 2 non-changeset ops, got %d", len(batch.ops))
	}

	if len(batch.changesets) != 1 {
		t.Errorf("Expected 1 changeset, got %d", len(batch.changesets))
	}

	cs, exists := batch.changesets["cs1"]
	if !exists {
		t.Error("Changeset cs1 not found")
	}

	if len(cs.ops) != 2 {
		t.Errorf("Expected 2 ops in changeset, got %d", len(cs.ops))
	}
}

// TestBatchChaining tests method chaining.
func TestBatchChaining(t *testing.T) {
	mockClient := &Client{}
	batch := mockClient.Batch().
		Get("Materials", "MAT001").
		Create("Materials", map[string]interface{}{"MatID": "MAT002"}).
		Update("Materials", "MAT003", map[string]interface{}{}).
		Delete("Materials", "MAT004")

	if len(batch.ops) != 4 {
		t.Errorf("Expected 4 ops after chaining, got %d", len(batch.ops))
	}
}

// TestExtractBoundary tests boundary extraction from Content-Type.
func TestExtractBoundary(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		expected    string
	}{
		{
			name:        "Standard boundary",
			contentType: "multipart/mixed; boundary=batch_12345",
			expected:    "batch_12345",
		},
		{
			name:        "Quoted boundary",
			contentType: `multipart/mixed; boundary="batch_12345"`,
			expected:    "batch_12345",
		},
		{
			name:        "No boundary",
			contentType: "multipart/mixed",
			expected:    "",
		},
		{
			name:        "Multiple params",
			contentType: "multipart/mixed; charset=utf-8; boundary=abc123; other=value",
			expected:    "abc123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractBoundary(tt.contentType)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// TestBatchResponseParsing tests parsing batch responses (basic structure).
func TestBatchResponseStructure(t *testing.T) {
	response := &BatchResponse{
		Results: []BatchResult{
			{StatusCode: 200, Body: []byte("OK")},
			{StatusCode: 201, Body: []byte("Created")},
		},
	}

	if len(response.Results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(response.Results))
	}

	if response.Results[0].StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", response.Results[0].StatusCode)
	}
}

// TestBatchMultipleChangesets tests handling multiple changesets.
func TestBatchMultipleChangesets(t *testing.T) {
	mockClient := &Client{}
	batch := mockClient.Batch()

	// First changeset
	batch.BeginChangeset("cs1")
	batch.Create("Materials", map[string]interface{}{"MatID": "MAT001"})
	batch.EndChangeset()

	// Second changeset
	batch.BeginChangeset("cs2")
	batch.Create("Plants", map[string]interface{}{"PlantID": "PLANT01"})
	batch.EndChangeset()

	if len(batch.changesets) != 2 {
		t.Errorf("Expected 2 changesets, got %d", len(batch.changesets))
	}

	if _, exists := batch.changesets["cs1"]; !exists {
		t.Error("Changeset cs1 not found")
	}

	if _, exists := batch.changesets["cs2"]; !exists {
		t.Error("Changeset cs2 not found")
	}
}

// TestBatchOperationHeaders tests that headers can be set on operations.
func TestBatchOperationHeaders(t *testing.T) {
	mockClient := &Client{}
	batch := mockClient.Batch()

	batch.Create("Materials", map[string]interface{}{"MatID": "MAT001"})

	if batch.ops[0].Headers == nil {
		t.Error("Headers should not be nil")
	}

	// Headers should be an empty map ready for use
	if len(batch.ops[0].Headers) != 0 {
		t.Errorf("Headers should start empty, got %d entries", len(batch.ops[0].Headers))
	}
}

// TestBatchOperationURLs tests URL construction for different operations.
func TestBatchOperationURLs(t *testing.T) {
	mockClient := &Client{}
	batch := mockClient.Batch()

	// Test different URL patterns
	batch.Get("Materials", "MAT001")
	batch.Create("Plants", nil)
	batch.Update("Vendors", "VEND01", nil)
	batch.Delete("Stores", "STORE1")

	urls := []string{
		batch.ops[0].URL,
		batch.ops[1].URL,
		batch.ops[2].URL,
		batch.ops[3].URL,
	}

	// Verify URLs don't start with "/" (they should be relative)
	for i, url := range urls {
		if strings.HasPrefix(url, "/") {
			t.Errorf("Op %d URL should not start with /, got %s", i, url)
		}
	}

	// Verify pattern matching
	if !strings.Contains(urls[0], "Materials") {
		t.Errorf("Get URL should contain entity set, got %s", urls[0])
	}

	if !strings.Contains(urls[1], "Plants") {
		t.Errorf("Create URL should contain entity set, got %s", urls[1])
	}
}

// BenchmarkBatchBuilding benchmarks batch request construction.
func BenchmarkBatchBuilding(b *testing.B) {
	mockClient := &Client{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		batch := mockClient.Batch()
		for j := 0; j < 10; j++ {
			batch.Get("Materials", fmt.Sprintf("MAT%03d", j))
		}
	}
}

// BenchmarkChangesetBuilding benchmarks changeset construction.
func BenchmarkChangesetBuilding(b *testing.B) {
	mockClient := &Client{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		batch := mockClient.Batch()
		batch.BeginChangeset("cs1")
		for j := 0; j < 5; j++ {
			batch.Create("Materials", map[string]interface{}{"MatID": fmt.Sprintf("MAT%03d", j)})
		}
		batch.EndChangeset()
	}
}

// TestBatchResponseResult tests individual batch results.
func TestBatchResponseResult(t *testing.T) {
	result := BatchResult{
		StatusCode: 200,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: []byte(`{"MatID":"MAT001"}`),
	}

	if result.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", result.StatusCode)
	}

	if result.Headers["Content-Type"] != "application/json" {
		t.Errorf("Expected Content-Type header, got %s", result.Headers["Content-Type"])
	}

	if string(result.Body) != `{"MatID":"MAT001"}` {
		t.Errorf("Body mismatch: %s", string(result.Body))
	}
}
