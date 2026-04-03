package traverse

import (
	"strings"
	"testing"
)

func TestBuilderBasics(t *testing.T) {
	c, _ := New(WithBaseURL("http://localhost:8080/odata"))

	qb := c.From("Products")
	if qb == nil {
		t.Fatalf("From() returned nil")
	}

	if qb.entitySet != "Products" {
		t.Errorf("entitySet = %s, want Products", qb.entitySet)
	}
}

// TestFunctionBuilderConstruction tests Function builder creation.
func TestFunctionBuilderConstruction(t *testing.T) {
	mockClient := &Client{}
	fb := mockClient.Function("GetMaterials")

	if fb == nil {
		t.Fatalf("Function() returned nil")
	}

	if fb.name != "GetMaterials" {
		t.Errorf("Expected name 'GetMaterials', got %q", fb.name)
	}

	if len(fb.parameters) != 0 {
		t.Error("Initial parameters should be empty")
	}
}

// TestActionBuilderConstruction tests Action builder creation.
func TestActionBuilderConstruction(t *testing.T) {
	mockClient := &Client{}
	ab := mockClient.Action("CreateMaterial")

	if ab == nil {
		t.Fatalf("Action() returned nil")
	}

	if ab.name != "CreateMaterial" {
		t.Errorf("Expected name 'CreateMaterial', got %q", ab.name)
	}
}

// TestFunctionImportBuilderConstruction tests FunctionImport builder creation.
func TestFunctionImportBuilderConstruction(t *testing.T) {
	mockClient := &Client{}
	fib := mockClient.FunctionImport("GetMaterialInfo")

	if fib == nil {
		t.Fatalf("FunctionImport() returned nil")
	}

	if fib.name != "GetMaterialInfo" {
		t.Errorf("Expected name 'GetMaterialInfo', got %q", fib.name)
	}
}

// TestFunctionParamString tests parameter string construction for functions.
func TestFunctionParamString(t *testing.T) {
	tests := []struct {
		name   string
		params map[string]interface{}
		want   string
	}{
		{
			name:   "No parameters",
			params: map[string]interface{}{},
			want:   "",
		},
		{
			name: "String parameter",
			params: map[string]interface{}{
				"MatID": "MAT001",
			},
			want: "MatID='MAT001'",
		},
		{
			name: "Integer parameter",
			params: map[string]interface{}{
				"Quantity": 100,
			},
			want: "Quantity=100",
		},
		{
			name: "Boolean parameter",
			params: map[string]interface{}{
				"IsActive": true,
			},
			want: "IsActive=true",
		},
		{
			name: "Multiple parameters",
			params: map[string]interface{}{
				"MatID":    "MAT001",
				"Quantity": 100,
			},
			// Order may vary in maps, so we'll check separately
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fb := &FunctionBuilder{
				parameters: tt.params,
			}
			result := fb.buildParameterString()

			if tt.want != "" && result != tt.want {
				t.Errorf("Expected %q, got %q", tt.want, result)
			}

			// For multi-param test, verify all params are present
			if tt.name == "Multiple parameters" {
				if !strings.Contains(result, "MatID='MAT001'") {
					t.Errorf("Expected MatID parameter, got %q", result)
				}
				if !strings.Contains(result, "Quantity=100") {
					t.Errorf("Expected Quantity parameter, got %q", result)
				}
			}
		})
	}
}

// TestFormatParameter tests individual parameter formatting.
func TestFormatParameter(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		value interface{}
		want  string
	}{
		{
			name:  "String value",
			key:   "MatID",
			value: "MAT001",
			want:  "MatID='MAT001'",
		},
		{
			name:  "String with quotes",
			key:   "Name",
			value: "Material's Name",
			want:  "Name='Material''s Name'",
		},
		{
			name:  "Integer value",
			key:   "Quantity",
			value: 100,
			want:  "Quantity=100",
		},
		{
			name:  "Float value",
			key:   "Price",
			value: 99.99,
			want:  "Price=99.99",
		},
		{
			name:  "Boolean true",
			key:   "IsActive",
			value: true,
			want:  "IsActive=true",
		},
		{
			name:  "Boolean false",
			key:   "IsDeleted",
			value: false,
			want:  "IsDeleted=false",
		},
		{
			name:  "Nil value",
			key:   "Optional",
			value: nil,
			want:  "Optional=null",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatParameter(tt.key, tt.value)
			if result != tt.want {
				t.Errorf("Expected %q, got %q", tt.want, result)
			}
		})
	}
}

// TestFunctionBuilderChaining tests method chaining for functions.
func TestFunctionBuilderChaining(t *testing.T) {
	mockClient := &Client{}
	fb := mockClient.Function("GetMaterials").
		Param("Plant", "P001").
		Param("Type", "Raw")

	if len(fb.parameters) != 2 {
		t.Errorf("Expected 2 parameters, got %d", len(fb.parameters))
	}

	if fb.parameters["Plant"] != "P001" {
		t.Error("Plant parameter not set correctly")
	}

	if fb.parameters["Type"] != "Raw" {
		t.Error("Type parameter not set correctly")
	}
}

// TestActionBuilderChaining tests method chaining for actions.
func TestActionBuilderChaining(t *testing.T) {
	mockClient := &Client{}
	data := map[string]interface{}{"MatID": "MAT001"}

	ab := mockClient.Action("CreateMaterial").
		WithBody(data).
		Param("Version", 1)

	if ab.body == nil {
		t.Error("Body not set")
	}

	if len(ab.parameters) != 1 {
		t.Errorf("Expected 1 parameter, got %d", len(ab.parameters))
	}
}

// TestParseResponseValue tests response value parsing.
func TestParseResponseValue(t *testing.T) {
	tests := []struct {
		name    string
		body    []byte
		wantKey string
	}{
		{
			name:    "Object response",
			body:    []byte(`{"MatID":"MAT001","Name":"Steel"}`),
			wantKey: "MatID",
		},
		{
			name:    "Wrapped response",
			body:    []byte(`{"value":{"MatID":"MAT001"}}`),
			wantKey: "value",
		},
		{
			name:    "Array response",
			body:    []byte(`[{"MatID":"MAT001"},{"MatID":"MAT002"}]`),
			wantKey: "value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseResponseValue(tt.body)
			if err != nil {
				t.Errorf("parseResponseValue failed: %v", err)
			}

			if _, exists := result[tt.wantKey]; !exists {
				t.Errorf("Expected key %q in result, got %v", tt.wantKey, result)
			}
		})
	}
}

// TestFunctionBuilderURL tests URL construction.
func TestFunctionBuilderURL(t *testing.T) {
	fb := &FunctionBuilder{
		name: "GetMaterials",
		parameters: map[string]interface{}{
			"Plant": "P001",
		},
	}

	paramStr := fb.buildParameterString()

	if !strings.Contains(paramStr, "Plant") {
		t.Errorf("Expected Plant parameter in %q", paramStr)
	}
}

// TestFunctionImportParamString tests parameter string for v2 function imports.
func TestFunctionImportParamString(t *testing.T) {
	fib := &FunctionImportBuilder{
		parameters: map[string]interface{}{
			"SalesOrg": "SO1",
			"Quarter":  4,
		},
	}

	result := fib.buildParameterString()

	if !strings.Contains(result, "SalesOrg='SO1'") {
		t.Errorf("Expected SalesOrg parameter, got %q", result)
	}

	if !strings.Contains(result, "Quarter=4") {
		t.Errorf("Expected Quarter parameter, got %q", result)
	}
}

// TestActionParameterOverwrite tests parameter overwriting.
func TestActionParameterOverwrite(t *testing.T) {
	mockClient := &Client{}
	ab := mockClient.Action("DoSomething").
		Param("Value", "First").
		Param("Value", "Second")

	if ab.parameters["Value"] != "Second" {
		t.Errorf("Expected parameter to be overwritten to 'Second', got %v", ab.parameters["Value"])
	}
}

// TestFunctionResponseParsing tests response parsing for different formats.
func TestFunctionResponseParsing(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		wantErr bool
	}{
		{
			name:    "Valid JSON object",
			json:    `{"MatID":"MAT001","Name":"Steel"}`,
			wantErr: false,
		},
		{
			name:    "Valid JSON array",
			json:    `[1, 2, 3]`,
			wantErr: false,
		},
		{
			name:    "Valid JSON primitive",
			json:    `"string value"`,
			wantErr: false,
		},
		{
			name:    "Invalid JSON",
			json:    `{invalid json}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseResponseValue([]byte(tt.json))
			if (err != nil) != tt.wantErr {
				t.Errorf("Expected error=%v, got %v", tt.wantErr, err != nil)
			}

			if err == nil && result == nil {
				t.Error("Expected non-nil result for valid JSON")
			}
		})
	}
}

// TestActionWithoutBody tests action execution without body.
func TestActionWithoutBody(t *testing.T) {
	mockClient := &Client{}
	ab := mockClient.Action("SimpleAction").Param("Code", "TEST")

	if ab.body != nil {
		t.Error("Body should be nil")
	}

	if len(ab.parameters) != 1 {
		t.Errorf("Expected 1 parameter, got %d", len(ab.parameters))
	}
}

// TestParameterStringComplexity tests complex parameter combinations.
func TestParameterStringComplexity(t *testing.T) {
	fb := &FunctionBuilder{
		parameters: map[string]interface{}{
			"StringParam": "value",
			"IntParam":    42,
			"FloatParam":  3.14,
			"BoolParam":   true,
			"NullParam":   nil,
		},
	}

	result := fb.buildParameterString()

	// All parameters should be present
	if !strings.Contains(result, "StringParam='value'") {
		t.Error("StringParam missing from parameter string")
	}
	if !strings.Contains(result, "IntParam=42") {
		t.Error("IntParam missing from parameter string")
	}
	if !strings.Contains(result, "BoolParam=true") {
		t.Error("BoolParam missing from parameter string")
	}
	if !strings.Contains(result, "NullParam=null") {
		t.Error("NullParam missing from parameter string")
	}
}

// BenchmarkParameterBuilding benchmarks parameter string construction.
func BenchmarkParameterBuilding(b *testing.B) {
	params := map[string]interface{}{
		"Param1": "value1",
		"Param2": 100,
		"Param3": true,
		"Param4": 99.99,
		"Param5": nil,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		fb := &FunctionBuilder{parameters: params}
		_ = fb.buildParameterString()
	}
}

// BenchmarkResponseParsing benchmarks response value parsing.
func BenchmarkResponseParsing(b *testing.B) {
	body := []byte(`{"MatID":"MAT001","Name":"Steel","Price":99.99,"Active":true}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = parseResponseValue(body)
	}
}
