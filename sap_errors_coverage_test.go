package traverse

import (
	"errors"
	"fmt"
	"testing"
)

func TestSAPError_Error_Nil(t *testing.T) {
	var e *SAPError
	if got := e.Error(); got != "" {
		t.Errorf("nil SAPError.Error() = %q, want empty", got)
	}
}

func TestSAPError_Error_FullFields(t *testing.T) {
	e := &SAPError{
		Code:       "/IWFND/MED/170",
		Message:    "Service not found",
		InnerError: "Could not find service",
	}
	got := e.Error()
	assertContains(t, got, "traverse: SAP error")
	assertContains(t, got, "/IWFND/MED/170")
	assertContains(t, got, "Service not found")
	assertContains(t, got, "inner: Could not find service")
}

func TestSAPError_Error_CodeOnly(t *testing.T) {
	e := &SAPError{Code: "ERR001", Message: "something"}
	got := e.Error()
	assertContains(t, got, "ERR001")
	assertContains(t, got, "something")
}

func TestSAPError_Error_MessageOnly(t *testing.T) {
	e := &SAPError{Message: "just message"}
	got := e.Error()
	assertContains(t, got, "just message")
}

func TestParseSAPError_Empty(t *testing.T) {
	if got := ParseSAPError(""); got != nil {
		t.Errorf("ParseSAPError(\"\") = %v, want nil", got)
	}
}

func TestParseSAPError_InvalidBody(t *testing.T) {
	if got := ParseSAPError("not json or xml"); got != nil {
		t.Errorf("ParseSAPError(invalid) = %v, want nil", got)
	}
}

func TestParseSAPError_JSON(t *testing.T) {
	body := `{"error":{"code":"/IWBEP/CX_SD_MED/100","message":{"value":"Invalid binding"},"innererror":{"message":"Details here"}}}`
	sapErr := ParseSAPError(body)
	if sapErr == nil {
		t.Fatal("ParseSAPError returned nil for valid JSON")
	}
	if sapErr.Code != "/IWBEP/CX_SD_MED/100" {
		t.Errorf("Code = %q, want /IWBEP/CX_SD_MED/100", sapErr.Code)
	}
	if sapErr.Message != "Invalid binding" {
		t.Errorf("Message = %q, want 'Invalid binding'", sapErr.Message)
	}
	if sapErr.InnerError != "Details here" {
		t.Errorf("InnerError = %q, want 'Details here'", sapErr.InnerError)
	}
	if sapErr.IsXML {
		t.Error("IsXML should be false for JSON format")
	}
	if sapErr.RawBody != body {
		t.Error("RawBody should contain original body")
	}
}

func TestParseSAPError_XML(t *testing.T) {
	body := `<error><code>/IWFND/MED/170</code><message>Service not found</message><innererror>detail</innererror></error>`
	sapErr := ParseSAPError(body)
	if sapErr == nil {
		t.Fatal("ParseSAPError returned nil for valid XML")
	}
	if sapErr.Code != "/IWFND/MED/170" {
		t.Errorf("Code = %q, want /IWFND/MED/170", sapErr.Code)
	}
	if !sapErr.IsXML {
		t.Error("IsXML should be true for XML format")
	}
}

func TestCategorizeSAPError_CSRF(t *testing.T) {
	e := &SAPError{Code: "CSRF", Message: "token expired"}
	got := categorizeSAPError(e)
	if got.ErrorType != SAPErrorTypeCSRF {
		t.Errorf("ErrorType = %v, want SAPErrorTypeCSRF", got.ErrorType)
	}
}

func TestCategorizeSAPError_TokenInvalid(t *testing.T) {
	e := &SAPError{Code: "X", Message: "token invalid"}
	got := categorizeSAPError(e)
	if got.ErrorType != SAPErrorTypeCSRF {
		t.Errorf("ErrorType = %v, want SAPErrorTypeCSRF", got.ErrorType)
	}
}

func TestCategorizeSAPError_XCSRF(t *testing.T) {
	e := &SAPError{Code: "X", Message: "x-csrf-token failed"}
	got := categorizeSAPError(e)
	if got.ErrorType != SAPErrorTypeCSRF {
		t.Errorf("ErrorType = %v, want SAPErrorTypeCSRF", got.ErrorType)
	}
}

func TestCategorizeSAPError_Unauthorized(t *testing.T) {
	e := &SAPError{Code: "AUTH", Message: "access denied"}
	got := categorizeSAPError(e)
	if got.ErrorType != SAPErrorTypeUnauthorized {
		t.Errorf("ErrorType = %v, want SAPErrorTypeUnauthorized", got.ErrorType)
	}
}

func TestCategorizeSAPError_Forbidden(t *testing.T) {
	e := &SAPError{Code: "X", Message: "forbidden resource"}
	got := categorizeSAPError(e)
	if got.ErrorType != SAPErrorTypeUnauthorized {
		t.Errorf("ErrorType = %v, want SAPErrorTypeUnauthorized", got.ErrorType)
	}
}

func TestCategorizeSAPError_Permission(t *testing.T) {
	e := &SAPError{Code: "X", Message: "permission denied"}
	got := categorizeSAPError(e)
	if got.ErrorType != SAPErrorTypeUnauthorized {
		t.Errorf("ErrorType = %v, want SAPErrorTypeUnauthorized", got.ErrorType)
	}
}

func TestCategorizeSAPError_ServiceConfig(t *testing.T) {
	e := &SAPError{Code: "/IWFND/MED/170", Message: "unknown service"}
	got := categorizeSAPError(e)
	if got.ErrorType != SAPErrorTypeServiceConfig {
		t.Errorf("ErrorType = %v, want SAPErrorTypeServiceConfig", got.ErrorType)
	}
}

func TestCategorizeSAPError_NotFound(t *testing.T) {
	e := &SAPError{Code: "X", Message: "entity not found"}
	got := categorizeSAPError(e)
	if got.ErrorType != SAPErrorTypeServiceConfig {
		t.Errorf("ErrorType = %v, want SAPErrorTypeServiceConfig", got.ErrorType)
	}
}

func TestCategorizeSAPError_Transient(t *testing.T) {
	e := &SAPError{Code: "X", Message: "gateway timeout"}
	got := categorizeSAPError(e)
	if got.ErrorType != SAPErrorTypeTransient {
		t.Errorf("ErrorType = %v, want SAPErrorTypeTransient", got.ErrorType)
	}
}

func TestCategorizeSAPError_TransientTemporarily(t *testing.T) {
	e := &SAPError{Code: "X", Message: "service temporarily unavailable"}
	got := categorizeSAPError(e)
	if got.ErrorType != SAPErrorTypeTransient {
		t.Errorf("ErrorType = %v, want SAPErrorTypeTransient", got.ErrorType)
	}
}

func TestCategorizeSAPError_Unknown(t *testing.T) {
	e := &SAPError{Code: "X", Message: "something weird"}
	got := categorizeSAPError(e)
	if got.ErrorType != SAPErrorTypeUnknown {
		t.Errorf("ErrorType = %v, want SAPErrorTypeUnknown", got.ErrorType)
	}
}

func TestCategorizeSAPError_InnerError(t *testing.T) {
	e := &SAPError{Code: "X", Message: "normal", InnerError: "x-csrf-token mismatch"}
	got := categorizeSAPError(e)
	if got.ErrorType != SAPErrorTypeCSRF {
		t.Errorf("ErrorType = %v, want SAPErrorTypeCSRF (inner error)", got.ErrorType)
	}
}

func TestIsSAPError_Nil(t *testing.T) {
	_, ok := IsSAPError(nil)
	if ok {
		t.Error("IsSAPError(nil) should return false")
	}
}

func TestIsSAPError_WithSAPError(t *testing.T) {
	sapErr := &SAPError{Code: "E001", Message: "test"}
	got, ok := IsSAPError(sapErr)
	if !ok {
		t.Fatal("IsSAPError should return true for *SAPError")
	}
	if got.Code != "E001" {
		t.Errorf("Code = %q, want E001", got.Code)
	}
}

func TestIsSAPError_Wrapped(t *testing.T) {
	sapErr := &SAPError{Code: "E002", Message: "wrapped test"}
	wrapped := errors.New("outer: %w")
	_ = wrapped
	err := fmt.Errorf("outer: %w", sapErr)
	got, ok := IsSAPError(err)
	if !ok {
		t.Fatal("IsSAPError should return true for wrapped *SAPError")
	}
	if got.Code != "E002" {
		t.Errorf("Code = %q, want E002", got.Code)
	}
}

func TestIsSAPError_NonSAPError(t *testing.T) {
	_, ok := IsSAPError(errors.New("普通的error"))
	if ok {
		t.Error("IsSAPError should return false for non-SAPError")
	}
}

func assertContains(t *testing.T, s, substr string) {
	t.Helper()
	if !contains(s, substr) {
		t.Errorf("%q does not contain %q", s, substr)
	}
}
