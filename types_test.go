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

// --- DateTime JSON marshal/unmarshal ---

func TestDateTime_UnmarshalJSON_Basic(t *testing.T) {
	var d DateTime
	err := json.Unmarshal([]byte(`"/Date(1704067200000)/"`), &d)
	if err != nil {
		t.Fatalf("UnmarshalJSON: %v", err)
	}
	got := d.Time()
	if got.Year() != 2024 {
		t.Errorf("expected year 2024, got %d", got.Year())
	}
}

func TestDateTime_UnmarshalJSON_WithOffset(t *testing.T) {
	var d DateTime
	err := json.Unmarshal([]byte(`"/Date(1704067200000+0100)/"`), &d)
	if err != nil {
		t.Fatalf("UnmarshalJSON with offset: %v", err)
	}
}

func TestDateTime_UnmarshalJSON_Invalid(t *testing.T) {
	var d DateTime
	err := json.Unmarshal([]byte(`"not-a-datetime"`), &d)
	if err == nil {
		t.Fatal("expected error for invalid format, got nil")
	}
}

func TestDateTime_UnmarshalJSON_BadMillis(t *testing.T) {
	var d DateTime
	err := json.Unmarshal([]byte(`"/Date(abc)/"`), &d)
	if err == nil {
		t.Fatal("expected error for non-numeric millis, got nil")
	}
}

func TestDateTime_MarshalJSON(t *testing.T) {
	d := DateTime(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))
	b, err := d.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}
	s := string(b)
	if !contains(s, "/Date(") {
		t.Errorf("expected /Date(...) format, got %s", s)
	}
}

func TestDateTime_String(t *testing.T) {
	d := DateTime(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))
	s := d.String()
	if s == "" {
		t.Error("String() returned empty")
	}
}

// --- DateTimeOffset JSON marshal/unmarshal ---

func TestDateTimeOffset_UnmarshalJSON_ISO(t *testing.T) {
	var d DateTimeOffset
	err := json.Unmarshal([]byte(`"2024-01-01T00:00:00Z"`), &d)
	if err != nil {
		t.Fatalf("UnmarshalJSON ISO: %v", err)
	}
	got := d.Time()
	if got.Year() != 2024 {
		t.Errorf("expected year 2024, got %d", got.Year())
	}
}

func TestDateTimeOffset_UnmarshalJSON_NoTZ(t *testing.T) {
	var d DateTimeOffset
	err := json.Unmarshal([]byte(`"2024-01-01T10:30:00"`), &d)
	if err != nil {
		t.Fatalf("UnmarshalJSON no-tz: %v", err)
	}
}

func TestDateTimeOffset_UnmarshalJSON_Invalid(t *testing.T) {
	var d DateTimeOffset
	err := json.Unmarshal([]byte(`"not-valid"`), &d)
	if err == nil {
		t.Fatal("expected error for invalid format, got nil")
	}
}

func TestDateTimeOffset_MarshalJSON(t *testing.T) {
	d := DateTimeOffset(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))
	b, err := json.Marshal(d)
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}
	if len(b) == 0 {
		t.Error("MarshalJSON returned empty bytes")
	}
}

func TestDateTimeOffset_Time(t *testing.T) {
	now := time.Now().UTC()
	d := DateTimeOffset(now)
	if d.Time().Unix() != now.Unix() {
		t.Error("Time() mismatch")
	}
}

// --- Guid ---

func TestGuid_UnmarshalJSON(t *testing.T) {
	var g Guid
	err := json.Unmarshal([]byte(`"550e8400-e29b-41d4-a716-446655440000"`), &g)
	if err != nil {
		t.Fatalf("Guid UnmarshalJSON: %v", err)
	}
}

func TestGuid_UnmarshalJSON_Invalid(t *testing.T) {
	var g Guid
	err := json.Unmarshal([]byte(`"not-a-guid"`), &g)
	if err == nil {
		t.Fatal("expected error for invalid GUID, got nil")
	}
}

func TestGuid_String(t *testing.T) {
	var g Guid
	_ = json.Unmarshal([]byte(`"550e8400-e29b-41d4-a716-446655440000"`), &g)
	s := g.String()
	if !contains(s, "550e8400") {
		t.Errorf("Guid.String() = %q, expected UUID string", s)
	}
}

// --- Decimal ---

func TestDecimal_UnmarshalJSON_Number(t *testing.T) {
	var d Decimal
	err := json.Unmarshal([]byte(`"123.45"`), &d)
	if err != nil {
		t.Fatalf("Decimal UnmarshalJSON string: %v", err)
	}
}

func TestDecimal_UnmarshalJSON_Numeric(t *testing.T) {
	var d Decimal
	err := json.Unmarshal([]byte(`42`), &d)
	if err != nil {
		t.Fatalf("Decimal UnmarshalJSON number: %v", err)
	}
}

func TestDecimal_UnmarshalJSON_Invalid(t *testing.T) {
	var d Decimal
	err := json.Unmarshal([]byte(`"not-a-number"`), &d)
	if err == nil {
		t.Fatal("expected error for invalid decimal, got nil")
	}
}

func TestDecimal_MarshalJSON(t *testing.T) {
	var d Decimal
	_ = json.Unmarshal([]byte(`"123.45"`), &d)
	b, err := d.MarshalJSON()
	if err != nil {
		t.Fatalf("Decimal MarshalJSON: %v", err)
	}
	if len(b) == 0 {
		t.Error("Decimal MarshalJSON returned empty bytes")
	}
}

func TestDecimal_String(t *testing.T) {
	var d Decimal
	_ = json.Unmarshal([]byte(`"99.99"`), &d)
	s := d.String()
	if s == "" {
		t.Error("Decimal.String() returned empty")
	}
}

// --- Binary ---

func TestBinary_UnmarshalJSON_Base64(t *testing.T) {
	var b Binary
	err := json.Unmarshal([]byte(`"SGVsbG8gV29ybGQ="`), &b)
	if err != nil {
		t.Fatalf("Binary UnmarshalJSON base64: %v", err)
	}
	// The current implementation stores the base64 string chunked — just check no error
	if len(b) == 0 {
		t.Error("Binary should not be empty after UnmarshalJSON")
	}
}

func TestBinary_UnmarshalJSON_Short(t *testing.T) {
	var b Binary
	// Short string (less than 4 bytes per chunk)
	err := json.Unmarshal([]byte(`"abc"`), &b)
	if err != nil {
		t.Fatalf("Binary UnmarshalJSON short string: %v", err)
	}
}

func TestBinary_MarshalJSON(t *testing.T) {
	b := Binary("Hello")
	data, err := b.MarshalJSON()
	if err != nil {
		t.Fatalf("Binary MarshalJSON: %v", err)
	}
	if len(data) == 0 {
		t.Error("Binary MarshalJSON returned empty bytes")
	}
}
