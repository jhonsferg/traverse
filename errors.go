package traverse

import (
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
