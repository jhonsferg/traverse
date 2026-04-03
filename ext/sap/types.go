package sap

// types.go - SAP-specific types and structures.

// SAPMetadata represents SAP metadata with id, uri, and type.
type SAPMetadata struct {
	ID   string `json:"id"`
	URI  string `json:"uri"`
	Type string `json:"type"`
}

// SAPDeferred represents a deferred navigation property link in OData v2.
type SAPDeferred struct {
	URI string `json:"uri"`
}

// SAPDateTime represents OData DateTime in SAP format.
// Format: /Date(milliseconds)/ or /Date(milliseconds+offset)/
type SAPDateTime string

// SAPDateTimeWithOffset represents OData DateTime with timezone offset.
type SAPDateTimeWithOffset string

// SAPError represents an error response from SAP OData service.
type SAPError struct {
	Error struct {
		Code       string `json:"code"`
		Message    struct {
			Lang  string `json:"lang"`
			Value string `json:"value"`
		} `json:"message"`
		InnerError map[string]interface{} `json:"innererror,omitempty"`
	} `json:"error"`
}

// String returns the error message.
func (e *SAPError) String() string {
	return e.Error.Message.Value
}
