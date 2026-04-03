package traverse

import (
	"encoding/json"
	"testing"
	"time"
)

func TestDateTimeValue(t *testing.T) {
	dt := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	lit := DateTimeValue(dt)

	// Should produce datetime'2024-01-15T10:30:00' format
	if lit == "" || !contains(lit, "2024-01-15") {
		t.Errorf("DateTimeValue() = %s, expected datetime literal", lit)
	}
}

func TestGuidValue(t *testing.T) {
	lit := GuidValue("550e8400-e29b-41d4-a716-446655440000")

	if lit != "guid'550e8400-e29b-41d4-a716-446655440000'" {
		t.Errorf("GuidValue() = %s, want guid'550e8400-e29b-41d4-a716-446655440000'", lit)
	}
}

func TestDecimalValue(t *testing.T) {
	lit := DecimalValue(123.45)

	if !contains(lit, "123.45") || !contains(lit, "M") {
		t.Errorf("DecimalValue() = %s, expected decimal with M suffix", lit)
	}
}

func TestDateTimeJSONUnmarshal(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		wantErr bool
	}{
		{
			name:    "SAP Date format",
			json:    `"/Date(1705314600000)/"`,
			wantErr: false,
		},
		{
			name:    "Invalid JSON",
			json:    `invalid`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var dt DateTime
			err := json.Unmarshal([]byte(tt.json), &dt)
			if (err != nil) != tt.wantErr {
				t.Errorf("DateTime.UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
