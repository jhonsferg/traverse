package traverse

import (
	"fmt"
	"testing"
)

func TestODataErrors(t *testing.T) {
	tests := []struct {
		name    string
		err     interface{}
		wantErr bool
	}{
		{
			name:    "ODataError",
			err:     &ODataError{Code: "404", Message: "Not found"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if (tt.err != nil) != tt.wantErr {
				t.Errorf("error = %v, wantErr %v", tt.err, tt.wantErr)
			}
		})
	}
}

func TestODataErrorMessage(t *testing.T) {
	errMsg := "Resource not found"
	err := &ODataError{
		Code:    "ResourceNotFound",
		Message: errMsg,
	}

	// Error() returns a formatted message, not just the message
	errStr := err.Error()
	if !contains(errStr, errMsg) {
		t.Errorf("ODataError.Error() = %s, expected to contain %s", errStr, errMsg)
	}
}

func TestIsODataError_Nil(t *testing.T) {
	oErr, ok := IsODataError(nil)
	if ok || oErr != nil {
		t.Error("IsODataError(nil) should return nil, false")
	}
}

func TestIsODataError_WithODataError(t *testing.T) {
	err := &ODataError{Code: "400", Message: "Bad Request"}
	oErr, ok := IsODataError(err)
	if !ok {
		t.Fatal("expected ok=true for *ODataError")
	}
	if oErr.Code != "400" {
		t.Errorf("Code = %q, want 400", oErr.Code)
	}
}

func TestIsODataError_WrappedError(t *testing.T) {
	import_err := &ODataError{Code: "500", Message: "Server Error"}
	wrapped := fmt.Errorf("wrapped: %w", import_err)
	oErr, ok := IsODataError(wrapped)
	if !ok {
		t.Fatal("expected ok=true for wrapped *ODataError")
	}
	if oErr.Code != "500" {
		t.Errorf("Code = %q, want 500", oErr.Code)
	}
}

func TestIsODataError_NonODataError(t *testing.T) {
	_, ok := IsODataError(ErrEntityNotFound)
	if ok {
		t.Error("IsODataError should return false for non-ODataError")
	}
}

func TestIsEntityNotFound_True(t *testing.T) {
	if !IsEntityNotFound(ErrEntityNotFound) {
		t.Error("expected true for ErrEntityNotFound")
	}
}

func TestIsEntityNotFound_Wrapped(t *testing.T) {
	if !IsEntityNotFound(fmt.Errorf("wrapped: %w", ErrEntityNotFound)) {
		t.Error("expected true for wrapped ErrEntityNotFound")
	}
}

func TestIsEntityNotFound_False(t *testing.T) {
	if IsEntityNotFound(ErrConcurrencyConflict) {
		t.Error("expected false for ErrConcurrencyConflict")
	}
}

func TestIsConcurrencyConflict_True(t *testing.T) {
	if !IsConcurrencyConflict(ErrConcurrencyConflict) {
		t.Error("expected true for ErrConcurrencyConflict")
	}
}

func TestIsConcurrencyConflict_Wrapped(t *testing.T) {
	if !IsConcurrencyConflict(fmt.Errorf("ops: %w", ErrConcurrencyConflict)) {
		t.Error("expected true for wrapped ErrConcurrencyConflict")
	}
}

func TestIsConcurrencyConflict_False(t *testing.T) {
	if IsConcurrencyConflict(ErrEntityNotFound) {
		t.Error("expected false for ErrEntityNotFound")
	}
}

func TestODataError_OnlyMessage(t *testing.T) {
	err := &ODataError{Message: "only message"}
	s := err.Error()
	if !contains(s, "only message") {
		t.Errorf("Error() = %q, want to contain 'only message'", s)
	}
}

func TestODataError_NoCodeNoMessage(t *testing.T) {
	err := &ODataError{}
	s := err.Error()
	if !contains(s, "unknown OData error") {
		t.Errorf("Error() = %q, expected 'unknown OData error'", s)
	}
}
