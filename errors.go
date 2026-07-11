package traverse

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"strings"
)

// ODataError represents an error returned by an OData service.
//
// ODataError encapsulates structured error information from OData responses,
// including error codes, messages, affected targets, and optional details.
// It also includes support for SAP-specific inner error extensions.
//
// OData services return errors in a standardized JSON format, which traverse
// parses and wraps in this type for easier error handling and inspection.
//
// Example:
//
//	err := client.From("InvalidSet").First(ctx)
//	if odErr, ok := IsODataError(err); ok {
//		fmt.Println("OData Error:", odErr.Code, odErr.Message)
//		for _, detail := range odErr.Details {
//			fmt.Println("Detail:", detail.Code, detail.Message)
//		}
//	}
type ODataError struct {
	Code       string                 `json:"code"`
	Message    string                 `json:"message"`
	Target     string                 `json:"target,omitempty"`
	Details    []ODataErrorDetail     `json:"details,omitempty"`
	InnerError map[string]interface{} `json:"innererror,omitempty"` // SAP extensions
}

// ODataErrorDetail represents a detail in an OData error response.
//
// ODataErrorDetail provides granular error information for specific fields
// or sub-operations that failed within a larger request.
type ODataErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Target  string `json:"target,omitempty"`
}

// Error implements the error interface.
//
// Error formats the ODataError into a human-readable string for logging and display.
// It includes the code (if available), message, and traverse prefix.
func (e *ODataError) Error() string {
	var b strings.Builder

	// Pre-allocate with estimated size
	b.Grow(50)
	b.WriteString("traverse: ")

	if e.Code != "" && e.Message != "" {
		b.WriteString("OData error ")
		b.WriteString(e.Code)
		b.WriteString(": ")
		b.WriteString(e.Message)
	} else if e.Message != "" {
		b.WriteString(e.Message)
	} else {
		b.WriteString("unknown OData error")
	}

	return b.String()
}

// Sentinel errors define common error conditions in OData operations.
//
// These are error values that can be compared directly with errors.Is()
// for categorizing failures in OData interactions.
var (
	// ErrEntityNotFound is returned when a requested entity is not found (HTTP 404).
	ErrEntityNotFound = errors.New("traverse: entity not found (404)")

	// ErrConcurrencyConflict is returned when an ETag mismatch occurs (HTTP 412).
	// This indicates the entity was modified by another client.
	ErrConcurrencyConflict = errors.New("traverse: ETag mismatch (412)")

	// ErrServiceUnavailable is returned when the OData service is unavailable.
	ErrServiceUnavailable = errors.New("traverse: OData service unavailable")

	// ErrCSRFTokenRequired is returned when a CSRF token is required (SAP-specific).
	// This is common in SAP systems; include a CSRF token in requests.
	ErrCSRFTokenRequired = errors.New("traverse: CSRF token required (SAP)")

	// ErrInvalidFilter is returned when a $filter expression is invalid.
	ErrInvalidFilter = errors.New("traverse: invalid $filter expression")

	// ErrMetadataInvalid is returned when $metadata parsing fails.
	ErrMetadataInvalid = errors.New("traverse: failed to parse $metadata")

	// ErrPageSizeExceeded is returned when server rejects $top value.
	ErrPageSizeExceeded = errors.New("traverse: server rejected $top value")

	// ErrStreamClosed is returned when stream is closed or context cancelled.
	ErrStreamClosed = errors.New("traverse: stream closed or context cancelled")

	// ErrClientClosed is returned when client is closed.
	ErrClientClosed = errors.New("traverse: client closed")
)

// SAPError represents a structured error from a SAP OData service.
//
// SAPError encapsulates both JSON and XML error formats that SAP systems return,
// parsing and normalizing them into a unified structure. This allows consumers
// to branch on error type (configuration, authorization, CSRF, transient) rather
// than inspecting raw response bodies.
//
// SAP returns errors in multiple formats:
//   - JSON format (OData v4): {"error":{"code":"...","message":{"value":"..."},"innererror":{...}}}
//   - XML Atom format (OData v2): <error><code>...</code><message>...</message><innererror>...</innererror></error>
//
// Example:
//
//	if sapErr := ParseSAPError(resp); sapErr != nil {
//		switch sapErr.ErrorType {
//		case SAPErrorTypeNotFound:
//			// Handle entity not found
//		case SAPErrorTypeCSRF:
//			// Handle CSRF token expiration
//		case SAPErrorTypeUnauthorized:
//			// Handle auth failure
//		}
//	}
type SAPError struct {
	Code       string       // SAP error code (e.g., "/IWFND/MED/170")
	Message    string       // Human-readable error message
	InnerError string       // SAP inner error details if present
	ErrorType  SAPErrorType // Categorized error type
	RawBody    string       // Original response body for debugging
	IsXML      bool         // Whether original response was XML format
}

// SAPErrorType categorizes SAP errors for programmatic handling.
type SAPErrorType int

const (
	SAPErrorTypeUnknown       SAPErrorType = iota // Unknown or uncategorized error
	SAPErrorTypeNotFound                          // Service or entity not found
	SAPErrorTypeCSRF                              // CSRF token invalid or missing
	SAPErrorTypeUnauthorized                      // Authentication or authorization failure
	SAPErrorTypeServiceConfig                     // SAP configuration or setup issue
	SAPErrorTypeTransient                         // Transient error (timeout, gateway issue)
)

// Error implements the error interface for SAPError.
func (e *SAPError) Error() string {
	if e == nil {
		return ""
	}
	var b strings.Builder
	b.Grow(100)
	b.WriteString("traverse: SAP error ")
	if e.Code != "" {
		b.WriteString(e.Code)
		b.WriteString(" ")
	}
	b.WriteString(e.Message)
	if e.InnerError != "" {
		b.WriteString(" (inner: ")
		b.WriteString(e.InnerError)
		b.WriteString(")")
	}
	return b.String()
}

// ParseSAPError extracts and categorizes a structured error from SAP response body.
//
// ParseSAPError attempts to parse the response body as either JSON (OData v4 format)
// or XML (OData v2 Atom format), categorizing the error type for programmatic handling.
// Returns nil if the body is not a recognized SAP error format.
//
// Supported categorizations:
//   - CSRF errors: Contains "csrf", "token invalid", "token expired", "x-csrf-token"
//   - Authorization errors: Contains "forbidden", "unauthorized", "permission"
//   - Configuration errors: Contains "/IWFND/MED", "not found", "unknown service"
//   - Transient errors: Contains "timeout", "gateway", "temporarily"
func ParseSAPError(body string) *SAPError {
	if body == "" {
		return nil
	}

	// Try JSON format first
	var jsonErr struct {
		Error struct {
			Code       string                   `json:"code"`
			Message    struct{ Value string }   `json:"message"`
			InnerError struct{ Message string } `json:"innererror"`
		} `json:"error"`
	}

	if err := json.Unmarshal([]byte(body), &jsonErr); err == nil && jsonErr.Error.Code != "" {
		return categorizeSAPError(&SAPError{
			Code:       jsonErr.Error.Code,
			Message:    jsonErr.Error.Message.Value,
			InnerError: jsonErr.Error.InnerError.Message,
			RawBody:    body,
			IsXML:      false,
		})
	}

	// Try XML format (OData v2 Atom)
	var xmlErr struct {
		Code       string `xml:"code"`
		Message    string `xml:"message"`
		InnerError string `xml:"innererror"`
	}

	if err := xml.Unmarshal([]byte(body), &xmlErr); err == nil && xmlErr.Code != "" {
		return categorizeSAPError(&SAPError{
			Code:       xmlErr.Code,
			Message:    xmlErr.Message,
			InnerError: xmlErr.InnerError,
			RawBody:    body,
			IsXML:      true,
		})
	}

	return nil
}

// categorizeSAPError assigns an error type based on code and message content.
func categorizeSAPError(sapErr *SAPError) *SAPError {
	lowerCode := strings.ToLower(sapErr.Code)
	lowerMsg := strings.ToLower(sapErr.Message + " " + sapErr.InnerError)

	switch {
	case strings.Contains(lowerMsg, "csrf") || strings.Contains(lowerMsg, "token invalid") ||
		strings.Contains(lowerMsg, "token expired") || strings.Contains(lowerMsg, "x-csrf-token"):
		sapErr.ErrorType = SAPErrorTypeCSRF

	case strings.Contains(lowerMsg, "forbidden") || strings.Contains(lowerMsg, "unauthorized") ||
		strings.Contains(lowerMsg, "permission") || strings.Contains(lowerMsg, "access denied"):
		sapErr.ErrorType = SAPErrorTypeUnauthorized

	case strings.Contains(lowerCode, "/iwfnd/med/") || strings.Contains(lowerMsg, "not found") ||
		strings.Contains(lowerMsg, "unknown service") || strings.Contains(lowerMsg, "service definition"):
		sapErr.ErrorType = SAPErrorTypeServiceConfig

	case strings.Contains(lowerMsg, "timeout") || strings.Contains(lowerMsg, "gateway") ||
		strings.Contains(lowerMsg, "temporarily"):
		sapErr.ErrorType = SAPErrorTypeTransient

	default:
		sapErr.ErrorType = SAPErrorTypeUnknown
	}

	return sapErr
}

// IsSAPError extracts a *SAPError from an error value.
//
// IsSAPError uses errors.As to unwrap and retrieve the *SAPError from
// a potentially wrapped error. Returns the *SAPError and true if found,
// otherwise returns nil and false.
func IsSAPError(err error) (*SAPError, bool) {
	if err == nil {
		return nil, false
	}

	var sapErr *SAPError
	if errors.As(err, &sapErr) {
		return sapErr, true
	}

	return nil, false
}

// IsODataError extracts an *ODataError from an error value.
//
// IsODataError uses errors.As to unwrap and retrieve the *ODataError from
// a potentially wrapped error. Returns the *ODataError and true if found,
// otherwise returns nil and false.
//
// Example:
//
//	if odErr, ok := IsODataError(err); ok {
//		fmt.Println("Code:", odErr.Code)
//	}
func IsODataError(err error) (*ODataError, bool) {
	if err == nil {
		return nil, false
	}

	var odErr *ODataError
	if errors.As(err, &odErr) {
		return odErr, true
	}

	return nil, false
}

// IsEntityNotFound returns true if err is or wraps [ErrEntityNotFound].
//
// IsEntityNotFound provides a convenient way to check if a specific error
// is a "not found" error without type assertion.
func IsEntityNotFound(err error) bool {
	return errors.Is(err, ErrEntityNotFound)
}

// IsConcurrencyConflict returns true if err is or wraps [ErrConcurrencyConflict].
//
// IsConcurrencyConflict indicates whether the error is due to an ETag mismatch
// (concurrent modification by another client). This typically requires the
// caller to retry with a fresh entity version.
func IsConcurrencyConflict(err error) bool {
	return errors.Is(err, ErrConcurrencyConflict)
}
