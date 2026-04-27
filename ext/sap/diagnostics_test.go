package sap

import (
	"strings"
	"testing"
)

// TestDiagnoseError_CSRF verifies CSRF error diagnosis.
func TestDiagnoseError_CSRF(t *testing.T) {
	diag := DiagnoseError(403, "CSRF token invalid", "failed csrf")
	if diag.ErrorCategory != "CSRF_TOKEN_INVALID" {
		t.Errorf("Expected CSRF_TOKEN_INVALID, got %s", diag.ErrorCategory)
	}
	if !diag.IsRetryable {
		t.Error("Expected retryable=true for CSRF error")
	}
	if !diag.IsCSRFError() {
		t.Error("Expected IsCSRFError to return true")
	}
}

// TestDiagnoseError_Auth verifies auth error diagnosis.
func TestDiagnoseError_Auth(t *testing.T) {
	diag := DiagnoseError(401, "", "unauthorized")
	if diag.ErrorCategory != "AUTHENTICATION_FAILED" {
		t.Errorf("Expected AUTHENTICATION_FAILED, got %s", diag.ErrorCategory)
	}
	if !diag.IsAuthError() {
		t.Error("Expected IsAuthError to return true for 401")
	}
}

// TestDiagnoseError_Forbidden403NonCSRF verifies non-CSRF 403 diagnosis.
func TestDiagnoseError_Forbidden403NonCSRF(t *testing.T) {
	diag := DiagnoseError(403, "forbidden", "access denied")
	if diag.ErrorCategory != "AUTHORIZATION_FAILED" {
		t.Errorf("Expected AUTHORIZATION_FAILED, got %s", diag.ErrorCategory)
	}
	if !diag.IsAuthError() {
		t.Error("Expected IsAuthError to return true for 403 without CSRF")
	}
}

// TestDiagnoseError_NotFound verifies 404 diagnosis.
func TestDiagnoseError_NotFound(t *testing.T) {
	diag := DiagnoseError(404, "", "not found")
	if diag.ErrorCategory != "SERVICE_NOT_FOUND" {
		t.Errorf("Expected SERVICE_NOT_FOUND, got %s", diag.ErrorCategory)
	}
	if !diag.IsConfigError() {
		t.Error("Expected IsConfigError to return true for 404")
	}
}

// TestDiagnoseError_UnsupportedMediaType verifies 406 diagnosis.
func TestDiagnoseError_UnsupportedMediaType(t *testing.T) {
	diag := DiagnoseError(406, "", "not acceptable")
	if diag.ErrorCategory != "UNSUPPORTED_MEDIA_TYPE" {
		t.Errorf("Expected UNSUPPORTED_MEDIA_TYPE, got %s", diag.ErrorCategory)
	}
}

// TestDiagnoseError_ServerError verifies 500 diagnosis.
func TestDiagnoseError_ServerError(t *testing.T) {
	diag := DiagnoseError(500, "", "internal error")
	if diag.ErrorCategory != "SAP_SERVER_ERROR" {
		t.Errorf("Expected SAP_SERVER_ERROR, got %s", diag.ErrorCategory)
	}
	if !diag.IsRetryable {
		t.Error("Expected retryable=true for 500 error")
	}
}

// TestDiagnoseError_Timeout verifies timeout diagnosis.
func TestDiagnoseError_Timeout(t *testing.T) {
	diag := DiagnoseError(504, "", "gateway timeout")
	if diag.ErrorCategory != "GATEWAY_TIMEOUT" {
		t.Errorf("Expected GATEWAY_TIMEOUT, got %s", diag.ErrorCategory)
	}
	if !diag.IsRetryable {
		t.Error("Expected retryable=true for timeout")
	}
	if !diag.IsNetworkError() {
		t.Error("Expected IsNetworkError to return true for timeout")
	}
}

// TestDiagnoseError_NetworkTimeout verifies network timeout in error message.
func TestDiagnoseError_NetworkTimeout(t *testing.T) {
	diag := DiagnoseError(0, "", "connection timeout")
	if diag.ErrorCategory != "NETWORK_TIMEOUT" {
		t.Errorf("Expected NETWORK_TIMEOUT, got %s", diag.ErrorCategory)
	}
	if !diag.IsNetworkError() {
		t.Error("Expected IsNetworkError to return true")
	}
}

// TestDiagnoseError_DNSError verifies DNS resolution error diagnosis.
func TestDiagnoseError_DNSError(t *testing.T) {
	diag := DiagnoseError(0, "", "no such host")
	if diag.ErrorCategory != "DNS_RESOLUTION_FAILED" {
		t.Errorf("Expected DNS_RESOLUTION_FAILED, got %s", diag.ErrorCategory)
	}
	if !diag.IsConfigError() {
		t.Error("Expected IsConfigError to return true for DNS error")
	}
}

// TestErrorDiagnosticString verifies String() method.
func TestErrorDiagnosticString(t *testing.T) {
	diag := DiagnoseError(403, "", "CSRF failed")
	str := diag.String()

	if !strings.Contains(str, "CSRF_TOKEN_INVALID") {
		t.Error("Expected error category in string output")
	}
	if !strings.Contains(str, "403") {
		t.Error("Expected status code in string output")
	}
}

// TestErrorDiagnosticDetailedString verifies DetailedString() method.
func TestErrorDiagnosticDetailedString(t *testing.T) {
	diag := DiagnoseError(500, "server error response", "internal error")
	detailed := diag.DetailedString()

	if !strings.Contains(detailed, "ERROR DIAGNOSTIC REPORT") {
		t.Error("Expected header in detailed string")
	}
	if !strings.Contains(detailed, "server error response") {
		t.Error("Expected response body in detailed string")
	}
}

// TestDiagnoseError_AllCategories verifies that all HTTP status codes are handled.
func TestDiagnoseError_AllCategories(t *testing.T) {
	statuses := []int{401, 403, 404, 406, 500, 503, 504}
	for _, status := range statuses {
		diag := DiagnoseError(status, "", "test error")
		if diag.ErrorCategory == "UNKNOWN_ERROR" {
			t.Errorf("Status %d resulted in UNKNOWN_ERROR", status)
		}
		if diag.SuggestedFix == "" {
			t.Errorf("Status %d has no suggested fix", status)
		}
	}
}
