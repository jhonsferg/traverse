package sap

import (
	"fmt"
	"strings"
)

// ErrorDiagnostic provides utilities for diagnosing and categorizing SAP OData errors.
type ErrorDiagnostic struct {
	StatusCode    int
	ResponseBody  string
	ErrorMessage  string
	SuggestedFix  string
	IsRetryable   bool
	ErrorCategory string
}

// DiagnoseError analyzes an error and provides diagnostic information.
// This helps users understand what went wrong and how to fix it.
func DiagnoseError(statusCode int, responseBody string, errorMessage string) ErrorDiagnostic {
	diag := ErrorDiagnostic{
		StatusCode:   statusCode,
		ResponseBody: responseBody,
		ErrorMessage: errorMessage,
		IsRetryable:  false,
	}

	// Categorize based on HTTP status and content
	switch statusCode {
	case 401:
		diag.ErrorCategory = "AUTHENTICATION_FAILED"
		diag.SuggestedFix = "Check your SAP credentials (username/password or OAuth2 token)"
		diag.IsRetryable = false

	case 403:
		if strings.Contains(strings.ToUpper(responseBody), "CSRF") ||
			strings.Contains(strings.ToUpper(errorMessage), "CSRF") ||
			strings.Contains(strings.ToUpper(errorMessage), "TOKEN") {
			diag.ErrorCategory = "CSRF_TOKEN_INVALID"
			diag.SuggestedFix = "CSRF token expired or session invalid. System will retry with fresh token"
			diag.IsRetryable = true
		} else {
			diag.ErrorCategory = "AUTHORIZATION_FAILED"
			diag.SuggestedFix = "Your SAP user account lacks permission for this operation"
			diag.IsRetryable = false
		}

	case 404:
		diag.ErrorCategory = "SERVICE_NOT_FOUND"
		diag.SuggestedFix = "Check that OData service name and path are correct; verify service is activated in SAP"
		diag.IsRetryable = false

	case 406:
		diag.ErrorCategory = "UNSUPPORTED_MEDIA_TYPE"
		diag.SuggestedFix = "Check Accept header format (try application/json instead of application/xml)"
		diag.IsRetryable = false

	case 500:
		diag.ErrorCategory = "SAP_SERVER_ERROR"
		diag.SuggestedFix = "Check SAP server logs and contact SAP administrator"
		diag.IsRetryable = true

	case 503:
		diag.ErrorCategory = "SERVICE_UNAVAILABLE"
		diag.SuggestedFix = "SAP service temporarily unavailable. Retry after a delay"
		diag.IsRetryable = true

	case 504:
		diag.ErrorCategory = "GATEWAY_TIMEOUT"
		diag.SuggestedFix = "Request timed out (SAP backend overloaded or network delay). Try with smaller dataset or longer timeout"
		diag.IsRetryable = true

	default:
		if strings.Contains(strings.ToUpper(errorMessage), "TIMEOUT") {
			diag.ErrorCategory = "NETWORK_TIMEOUT"
			diag.SuggestedFix = "Connection timeout. Check network connectivity and SAP server availability"
			diag.IsRetryable = true
		} else if strings.Contains(strings.ToUpper(errorMessage), "CONNECTION REFUSED") {
			diag.ErrorCategory = "CONNECTION_REFUSED"
			diag.SuggestedFix = "Cannot connect to SAP server. Verify SAP URL and network connectivity"
			diag.IsRetryable = false
		} else if strings.Contains(strings.ToUpper(errorMessage), "NO SUCH HOST") {
			diag.ErrorCategory = "DNS_RESOLUTION_FAILED"
			diag.SuggestedFix = "Cannot resolve SAP hostname. Check URL and DNS configuration"
			diag.IsRetryable = false
		} else {
			diag.ErrorCategory = "UNKNOWN_ERROR"
			diag.SuggestedFix = "Unknown error. Check error message and SAP server logs"
			diag.IsRetryable = false
		}
	}

	return diag
}

// String returns a human-readable error diagnostic.
func (d ErrorDiagnostic) String() string {
	return fmt.Sprintf(
		"[%s] %s\nStatus: %d\nSuggestion: %s\nRetryable: %v",
		d.ErrorCategory,
		d.ErrorMessage,
		d.StatusCode,
		d.SuggestedFix,
		d.IsRetryable,
	)
}

// DetailedString returns a more detailed diagnostic including raw response.
func (d ErrorDiagnostic) DetailedString() string {
	msg := fmt.Sprintf(
		"ERROR DIAGNOSTIC REPORT\n"+
			"=======================\n"+
			"Category: %s\n"+
			"Status Code: %d\n"+
			"Message: %s\n"+
			"Suggestion: %s\n"+
			"Retryable: %v\n"+
			"Response Body:\n%s",
		d.ErrorCategory,
		d.StatusCode,
		d.ErrorMessage,
		d.SuggestedFix,
		d.IsRetryable,
		d.ResponseBody,
	)
	return msg
}

// IsCSRFError checks if this error is specifically a CSRF token error.
func (d ErrorDiagnostic) IsCSRFError() bool {
	return d.ErrorCategory == "CSRF_TOKEN_INVALID"
}

// IsAuthError checks if this error is an authentication or authorization error.
func (d ErrorDiagnostic) IsAuthError() bool {
	return d.ErrorCategory == "AUTHENTICATION_FAILED" ||
		d.ErrorCategory == "AUTHORIZATION_FAILED"
}

// IsConfigError checks if this error is a configuration-related error.
func (d ErrorDiagnostic) IsConfigError() bool {
	return d.ErrorCategory == "SERVICE_NOT_FOUND" ||
		d.ErrorCategory == "DNS_RESOLUTION_FAILED" ||
		d.ErrorCategory == "CONNECTION_REFUSED"
}

// IsNetworkError checks if this error is a network-related error.
func (d ErrorDiagnostic) IsNetworkError() bool {
	return d.ErrorCategory == "NETWORK_TIMEOUT" ||
		d.ErrorCategory == "GATEWAY_TIMEOUT" ||
		d.ErrorCategory == "CONNECTION_REFUSED" ||
		d.ErrorCategory == "DNS_RESOLUTION_FAILED"
}
