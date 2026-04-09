package traverse

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"
)

// ODataVersion represents the OData protocol version.
//
// ODataVersion is used to distinguish between different OData protocol versions
// supported by different SAP systems. Version detection can happen automatically
// during query execution via HTTP headers.
type ODataVersion int

const (
	ODataV2 ODataVersion = 2 // SAP NetWeaver Gateway, OData v2 (legacy)
	ODataV4 ODataVersion = 4 // SAP S/4HANA, OData v4 (standard)
)

// DateTime represents an OData Edm.DateTime value (OData v2).
//
// DateTime wraps time.Time to provide custom JSON marshaling/unmarshaling for the
// OData v2 DateTime format: /Date(milliseconds)/ or /Date(milliseconds+offset)/.
// This format is common in SAP NetWeaver Gateway and legacy OData v2 services.
//
// Example formats in JSON responses:
//
//	"/Date(1704067200000)/"      // 2024-01-01 UTC
//	"/Date(1704067200000+0100)/" // With timezone offset
//
// Internally stored as UTC time.Time for consistency. Use [DateTime.Time] to convert
// to time.Time for standard Go operations.
//
// Example:
//
//	type Order struct {
//		CreatedAt traverse.DateTime `json:"createdAt"`
//	}
//
//	json.Unmarshal([]byte(`{"createdAt":"/Date(1704067200000)/"}`), &order)
//	t := order.CreatedAt.Time() // Convert to time.Time
type DateTime time.Time

// UnmarshalJSON decodes OData DateTime format: /Date(milliseconds)/ or /Date(milliseconds±offset)/.
//
// UnmarshalJSON parses the OData v2 DateTime format and extracts milliseconds since epoch.
// The timezone offset (if present) is ignored; the result is always in UTC.
//
// Returns an error if the format is invalid or milliseconds cannot be parsed.
func (d *DateTime) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}

	// Handle /Date(1704067200000)/ or /Date(1704067200000+0100)/
	if !strings.HasPrefix(s, "/Date(") || !strings.HasSuffix(s, ")/") {
		return fmt.Errorf("invalid OData DateTime format: %s", s)
	}

	dateStr := s[6 : len(s)-2] // Remove /Date( and )/

	// Extract milliseconds and offset if present
	var millis int64
	var err error
	if idx := strings.IndexAny(dateStr, "+-"); idx != -1 {
		millis, err = strconv.ParseInt(dateStr[:idx], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid DateTime milliseconds: %s", dateStr[:idx])
		}
	} else {
		millis, err = strconv.ParseInt(dateStr, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid DateTime milliseconds: %s", dateStr)
		}
	}

	*d = DateTime(time.UnixMilli(millis).UTC())
	return nil
}

// MarshalJSON encodes DateTime to OData v2 format: /Date(milliseconds)/.
//
// MarshalJSON converts the internal time.Time to milliseconds since epoch and
// formats it as /Date(milliseconds)/.
func (d DateTime) MarshalJSON() ([]byte, error) {
	millis := time.Time(d).UnixMilli()
	return []byte(fmt.Sprintf(`"/Date(%d)/"`, millis)), nil
}

// String returns the time.Time string representation.
//
// String converts DateTime to time.Time and returns its standard string format.
func (d DateTime) String() string {
	return time.Time(d).String()
}

// Time converts DateTime to time.Time.
//
// Time returns the underlying time.Time value for use with standard Go time operations.
func (d DateTime) Time() time.Time {
	return time.Time(d)
}

// DateTimeOffset represents an OData Edm.DateTimeOffset value (OData v4).
//
// DateTimeOffset wraps time.Time to provide custom JSON marshaling/unmarshaling for the
// OData v4 DateTimeOffset format: ISO 8601 (2024-01-01T00:00:00Z or 2024-01-01T00:00:00+01:00).
// This format is standard in modern OData v4 services like SAP S/4HANA.
//
// Example formats in JSON responses:
//
//	"2024-01-01T00:00:00Z"         // UTC (RFC 3339)
//	"2024-01-01T00:00:00+01:00"    // With timezone offset
//	"2024-01-01T00:00:00"          // Without timezone (treated as UTC)
//
// Use [DateTimeOffset.Time] to convert to time.Time for standard Go operations.
//
// Example:
//
//	type Order struct {
//		CreatedAt traverse.DateTimeOffset `json:"createdAt"`
//	}
//
//	json.Unmarshal([]byte(`{"createdAt":"2024-01-01T00:00:00Z"}`), &order)
//	t := order.CreatedAt.Time() // Convert to time.Time
type DateTimeOffset time.Time

// UnmarshalJSON decodes ISO 8601 format (RFC 3339).
//
// UnmarshalJSON parses ISO 8601 formatted datetime strings. Supports both full
// timestamps with timezone (RFC 3339) and timestamps without timezone (treated as UTC).
//
// Returns an error if the format cannot be parsed.
func (d *DateTimeOffset) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}

	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		// Try alternative format without timezone
		t, err = time.Parse("2006-01-02T15:04:05", s)
		if err != nil {
			return fmt.Errorf("invalid DateTimeOffset: %s", s)
		}
	}

	*d = DateTimeOffset(t)
	return nil
}

// MarshalJSON encodes to ISO 8601 (RFC 3339).
//
// MarshalJSON converts DateTimeOffset to RFC 3339 format for JSON output.
func (d DateTimeOffset) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Time(d).Format(time.RFC3339))
}

// Time converts DateTimeOffset to time.Time.
//
// Time returns the underlying time.Time value for use with standard Go time operations.
func (d DateTimeOffset) Time() time.Time {
	return time.Time(d)
}

// Guid represents an OData Edm.Guid value (UUID).
//
// Guid wraps [16]byte to represent OData GUID values in the standard UUID format:
// xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx (8-4-4-4-12 hex digits).
//
// The Guid type provides custom JSON marshaling/unmarshaling to convert between
// UUID string format and the internal 16-byte representation.
//
// Example formats in JSON responses:
//
//	"550e8400-e29b-41d4-a716-446655440000"
//
// Use [Guid.String] to convert to standard UUID string format.
//
// Example:
//
//	type Entity struct {
//		ID traverse.Guid `json:"id"`
//	}
//
//	json.Unmarshal([]byte(`{"id":"550e8400-e29b-41d4-a716-446655440000"}`), &entity)
//	uuidStr := entity.ID.String() // "550e8400-e29b-41d4-a716-446655440000"
type Guid [16]byte

// UnmarshalJSON decodes UUID format: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx.
//
// UnmarshalJSON parses a UUID string and converts it to the internal 16-byte
// representation. The UUID must be in standard format with 8-4-4-4-12 hex digit groups.
//
// Returns an error if the format is invalid or contains non-hexadecimal characters.
func (g *Guid) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}

	// Parse UUID string to 16 bytes
	parts := strings.Split(s, "-")
	if len(parts) != 5 {
		return fmt.Errorf("invalid GUID format: %s", s)
	}

	var result Guid
	// Parse each segment: 8-4-4-4-12 hex digits
	positions := []struct {
		start, length int
		pos           int
	}{
		{0, 8, 0},   // first 8 bytes
		{0, 4, 4},   // next 2 bytes
		{0, 4, 6},   // next 2 bytes
		{0, 4, 8},   // next 2 bytes
		{0, 12, 10}, // last 6 bytes
	}

	idx := 0
	for _, seg := range positions {
		for i := 0; i < len(parts[idx]); i += 2 {
			b1 := hexToByte(parts[idx][i])
			b2 := hexToByte(parts[idx][i+1])
			if b1 < 0 || b2 < 0 {
				return fmt.Errorf("invalid GUID hex: %s", parts[idx])
			}
			result[seg.pos+i/2] = byte((b1 << 4) | b2) // #nosec G115
		}
		idx++
	}

	*g = result
	return nil
}

// String returns the UUID string representation.
//
// String converts the internal 16-byte representation to standard UUID format
// (8-4-4-4-12 hex digits).
func (g Guid) String() string {
	return fmt.Sprintf("%x-%x-%x-%x-%x",
		g[0:4], g[4:6], g[6:8], g[8:10], g[10:16])
}

// hexToByte converts a single hexadecimal character to its integer value.
//
// hexToByte accepts digits 0-9 (return 0-9), lowercase a-f (return 10-15),
// and uppercase A-F (return 10-15). Returns -1 for invalid characters.
func hexToByte(c byte) int {
	switch {
	case c >= '0' && c <= '9':
		return int(c - '0')
	case c >= 'a' && c <= 'f':
		return int(c - 'a' + 10)
	case c >= 'A' && c <= 'F':
		return int(c - 'A' + 10)
	default:
		return -1
	}
}

// Decimal represents an OData Edm.Decimal value (arbitrary precision decimal).
//
// Decimal uses math/big.Float internally to store arbitrary precision decimal numbers.
// This is essential for financial calculations and other high-precision scenarios where
// standard float64 cannot represent the exact value.
//
// The underlying value is stored as a pointer to big.Float to enable nil representation
// for SQL NULL equivalents.
//
// Example formats in JSON responses:
//
//	"123.456789012345678901234567890"
//	123.45 (numeric literal)
//
// Example:
//
//	type Product struct {
//		Price traverse.Decimal `json:"price"`
//	}
//
//	json.Unmarshal([]byte(`{"price":"19.99"}`), &product)
//	fmt.Println(product.Price.String()) // "19.99" with full precision
type Decimal struct {
	value *big.Float
}

// UnmarshalJSON decodes decimal values from JSON (string or numeric).
//
// UnmarshalJSON accepts both string and numeric representations of decimal values.
// Strings are preferred for preserving arbitrary precision. Numeric literals are
// converted to string first for parsing.
//
// Returns an error if the value cannot be parsed as a valid decimal number.
func (d *Decimal) UnmarshalJSON(b []byte) error {
	var v interface{}
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}

	var str string
	switch val := v.(type) {
	case string:
		str = val
	case float64:
		str = fmt.Sprint(val)
	default:
		return fmt.Errorf("invalid Decimal value: %v", v)
	}

	f := new(big.Float)
	if _, ok := f.SetString(str); !ok {
		return fmt.Errorf("cannot parse Decimal: %s", str)
	}

	d.value = f
	return nil
}

// MarshalJSON encodes decimal to JSON string (preserves precision).
//
// MarshalJSON returns the decimal as a JSON string to preserve arbitrary precision
// during round-tripping. Nil values are encoded as null.
func (d Decimal) MarshalJSON() ([]byte, error) {
	if d.value == nil {
		return []byte("null"), nil
	}
	return json.Marshal(d.value.String())
}

// String returns the decimal string representation.
//
// String returns the full precision string representation of the decimal.
// Returns "0" for nil values.
func (d Decimal) String() string {
	if d.value == nil {
		return "0"
	}
	return d.value.String()
}

// Binary represents an OData Edm.Binary value (base64 encoded binary data).
//
// Binary is a byte slice with custom JSON marshaling/unmarshaling for base64 encoded
// binary data. This is used for BLOB/binary fields in OData services.
//
// Example formats in JSON responses:
//
//	"aGVsbG8gd29ybGQ="  // base64 encoded: "hello world"
//
// Example:
//
//	type Document struct {
//		Content traverse.Binary `json:"content"`
//	}
//
//	json.Unmarshal([]byte(`{"content":"aGVsbG8="}`), &doc)
//	fmt.Println(string(doc.Content)) // "hello"
type Binary []byte

// UnmarshalJSON decodes base64 encoded binary data.
//
// UnmarshalJSON accepts a JSON string containing standard or URL-safe base64
// encoded binary data and decodes it to the internal byte slice. OData v4
// uses standard base64 (RFC 4648 §4); some SAP services use URL-safe base64
// (RFC 4648 §5). Both are tried in order.
//
// Returns an error if the input is not a valid JSON string or cannot be
// decoded as base64.
func (b *Binary) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	if s == "" {
		*b = Binary(nil)
		return nil
	}
	// Try standard base64 first (OData v4 default).
	decoded, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		// Fall back to URL-safe base64 (some SAP / Azure services use this).
		decoded, err = base64.URLEncoding.DecodeString(s)
		if err != nil {
			return fmt.Errorf("traverse: Binary.UnmarshalJSON: invalid base64: %w", err)
		}
	}
	*b = Binary(decoded)
	return nil
}

// MarshalJSON encodes binary data as a base64 JSON string.
//
// MarshalJSON converts the raw binary bytes to a standard base64-encoded
// string (RFC 4648 §4) and wraps it in JSON quotes, matching the OData v4
// Edm.Binary wire format.
func (b Binary) MarshalJSON() ([]byte, error) {
	encoded := base64.StdEncoding.EncodeToString(b)
	buf := make([]byte, 0, len(encoded)+2)
	buf = append(buf, '"')
	buf = append(buf, encoded...)
	buf = append(buf, '"')
	return buf, nil
}

// DateTimeValueBytes produces an OData DateTime literal for use in filter expressions as bytes.
//
// DateTimeValueBytes generates the OData v2 DateTime format suitable for use in $filter
// expressions. It pre-allocates a buffer and appends the formatted datetime to minimize
// allocations compared to string concatenation.
//
// Returns bytes in the format: datetime'2024-01-01T00:00:00'
//
// Example:
//
//	filter := string(DateTimeValueBytes(time.Now()))
//	// filter = "datetime'2024-01-01T12:34:56'"
func DateTimeValueBytes(t time.Time) []byte {
	// Pre-allocate buffer: "datetime'" + time (19 chars) + "'"
	buf := make([]byte, 0, 31)
	buf = append(buf, "datetime'"...)
	buf = t.AppendFormat(buf, "2006-01-02T15:04:05")
	buf = append(buf, '\'')
	return buf
}

// DateTimeValue produces an OData DateTime literal for use in filter expressions.
//
// DateTimeValue generates the OData v2 DateTime format suitable for $filter expressions.
// It's a wrapper around [DateTimeValueBytes] for convenience when a string is needed.
//
// Returns string in the format: datetime'2024-01-01T00:00:00'
//
// Example (used in filters):
//
//	qb.Filter(fmt.Sprintf("CreatedAt ge %s", DateTimeValue(startDate)))
func DateTimeValue(t time.Time) string {
	return string(DateTimeValueBytes(t))
}

// DateTimeOffsetValue produces an OData DateTimeOffset literal for use in filter expressions.
//
// DateTimeOffsetValue generates the OData v4 DateTimeOffset format (ISO 8601/RFC 3339)
// suitable for $filter expressions on DateTimeOffset fields.
//
// Returns string in RFC 3339 format: 2024-01-01T00:00:00Z or 2024-01-01T00:00:00+01:00
//
// Example (used in filters):
//
//	qb.Filter(fmt.Sprintf("CreatedAt ge %s", DateTimeOffsetValue(startDate)))
func DateTimeOffsetValue(t time.Time) string {
	return t.Format(time.RFC3339)
}

// GuidValue produces an OData Guid literal for use in filter expressions.
//
// GuidValue wraps the provided GUID string in the OData Guid format suitable for
// $filter expressions on Guid fields.
//
// Returns string in the format: guid'xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx'
//
// Example (used in filters):
//
//	qb.Filter(fmt.Sprintf("ID eq %s", GuidValue("550e8400-e29b-41d4-a716-446655440000")))
func GuidValue(id string) string {
	return fmt.Sprintf("guid'%s'", id)
}

// DecimalValueBytes produces an OData Decimal literal for filters as bytes.
// Returns: []byte("3.14M")
// This is the optimized version that avoids string allocations.
func DecimalValueBytes(v float64) []byte {
	// Use strconv for efficient float formatting
	// Pre-allocate buffer with reasonable size
	buf := make([]byte, 0, 32)
	buf = strconv.AppendFloat(buf, v, 'f', -1, 64)
	buf = append(buf, 'M')
	return buf
}

// DecimalValue produces an OData Decimal literal for filters.
// Returns: 3.14M
// This is a wrapper around DecimalValueBytes for backward compatibility.
func DecimalValue(v float64) string {
	return string(DecimalValueBytes(v))
}

// Response format options
type ResponseFormat int

const (
	FormatJSON ResponseFormat = iota // JSON (default)
	FormatAtom                       // XML/ATOM (legacy OData v2)
)
