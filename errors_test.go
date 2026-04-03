package traverse

import (
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
