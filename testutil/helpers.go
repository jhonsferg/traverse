package testutil

import (
	"encoding/json"
	"testing"
)

// AssertEqual checks if two values are equal and fails the test if they differ.
//
// AssertEqual is a test helper that provides clear failure messages when values don't match.
// It calls t.Helper() to report failures at the caller's location, not in testutil.
//
// Example:
//
//	testutil.AssertEqual(t, result.Count, 42, "result count mismatch")
func AssertEqual(t *testing.T, got, want interface{}, msg string) {
	t.Helper()
	if got != want {
		t.Errorf("%s: got %v, want %v", msg, got, want)
	}
}

// AssertNoError fails the test if err is not nil.
//
// AssertNoError is a convenience helper for checking that operations succeeded.
// Passes an error message for context in failure reports.
//
// Example:
//
//	err := client.From("Products").First(ctx)
//	testutil.AssertNoError(t, err, "query should succeed")
func AssertNoError(t *testing.T, err error, msg string) {
	t.Helper()
	if err != nil {
		t.Errorf("%s: unexpected error: %v", msg, err)
	}
}

// AssertError fails the test if err is nil (i.e., if no error occurred).
//
// AssertError is used to verify that an operation correctly returned an error.
// Useful for testing error conditions and failure paths.
//
// Example:
//
//	err := client.From("NonExistent").First(ctx)
//	testutil.AssertError(t, err, "query should fail for invalid entity")
func AssertError(t *testing.T, err error, msg string) {
	t.Helper()
	if err == nil {
		t.Errorf("%s: expected error, got nil", msg)
	}
}

// AssertContains fails the test if substring is not found in str.
//
// AssertContains checks for substring presence without requiring imports of strings package.
// Useful for verifying error messages or response content.
//
// Example:
//
//	err := client.From("Invalid").First(ctx)
//	testutil.AssertContains(t, err.Error(), "404", "error should mention 404 status")
func AssertContains(t *testing.T, str, substring, msg string) {
	t.Helper()
	for i := 0; i < len(str)-len(substring)+1; i++ {
		if str[i:i+len(substring)] == substring {
			return
		}
	}
	t.Errorf("%s: %q does not contain %q", msg, str, substring)
}

// AssertStatusCode fails the test if the status codes don't match.
//
// AssertStatusCode provides clear HTTP status code comparison in tests.
//
// Example:
//
//	testutil.AssertStatusCode(t, resp.StatusCode, 200, "should get OK response")
func AssertStatusCode(t *testing.T, got, want int, msg string) {
	t.Helper()
	if got != want {
		t.Errorf("%s: got status %d, want %d", msg, got, want)
	}
}

// AssertJSONEqual parses two JSON strings and compares them structurally.
//
// AssertJSONEqual unmarshals both JSON strings and compares the resulting values.
// This allows comparing JSON despite differences in whitespace or key ordering.
// Useful for verifying OData response formats.
//
// Example:
//
//	expected := `{"ID":1,"Name":"Widget"}`
//	testutil.AssertJSONEqual(t, response, expected, "response JSON mismatch")
func AssertJSONEqual(t *testing.T, gotJSON, wantJSON string, msg string) {
	t.Helper()
	var got, want interface{}
	if err := json.Unmarshal([]byte(gotJSON), &got); err != nil {
		t.Errorf("%s: failed to parse got JSON: %v", msg, err)
		return
	}
	if err := json.Unmarshal([]byte(wantJSON), &want); err != nil {
		t.Errorf("%s: failed to parse want JSON: %v", msg, err)
		return
	}
	if got != want {
		t.Errorf("%s: JSON mismatch\ngot:  %v\nwant: %v", msg, got, want)
	}
}

// ODataResponse creates a minimal OData v4 response body for testing.
//
// ODataResponse wraps entity values in an OData v4 "value" array format.
// Use this to mock service responses in unit tests.
//
// Example:
//
//	response := testutil.ODataResponse(
//		map[string]interface{}{"ID": 1, "Name": "Product A"},
//		map[string]interface{}{"ID": 2, "Name": "Product B"},
//	)
//	// Returns: {"value":[{"ID":1,"Name":"Product A"},{"ID":2,"Name":"Product B"}]}
func ODataResponse(values ...map[string]interface{}) string {
	resp := map[string]interface{}{
		"value": values,
	}
	b, _ := json.Marshal(resp)
	return string(b)
}

// ODataSingleResponse creates an OData v4 single-entity response for testing.
//
// ODataSingleResponse wraps a single entity map directly (no "value" wrapper).
// Use this to mock responses for individual entity retrievals.
//
// Example:
//
//	response := testutil.ODataSingleResponse(
//		map[string]interface{}{"ID": 1, "Name": "Product A"},
//	)
//	// Returns: {"ID":1,"Name":"Product A"}
func ODataSingleResponse(value map[string]interface{}) string {
	b, _ := json.Marshal(value)
	return string(b)
}

// ODataErrorResponse creates an OData v4 error response for testing.
//
// ODataErrorResponse generates a standard OData error format with code and message.
// Use this to test error handling in the client.
//
// Example:
//
//	response := testutil.ODataErrorResponse("NOT_FOUND", "Entity not found")
//	// Returns: {"error":{"code":"NOT_FOUND","message":"Entity not found"}}
func ODataErrorResponse(code, message string) string {
	resp := map[string]interface{}{
		"error": map[string]interface{}{
			"code":    code,
			"message": message,
		},
	}
	b, _ := json.Marshal(resp)
	return string(b)
}
