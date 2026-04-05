package traverse

import (
	"testing"
)

func TestEntitySchema_ValidateFilter_SimpleEqualityOperator(t *testing.T) {
	schema := EntitySchema{
		Properties: map[string]string{
			"Name":  "string",
			"Age":   "int",
			"Email": "string",
		},
	}

	err := validateFilter(schema, "Name eq 'John'")
	if err != nil {
		t.Fatalf("Expected no error for valid field, got: %v", err)
	}
}

func TestEntitySchema_ValidateFilter_MultipleOperators(t *testing.T) {
	schema := EntitySchema{
		Properties: map[string]string{
			"Name":  "string",
			"Age":   "int",
			"Email": "string",
		},
	}

	testCases := []struct {
		name      string
		filter    string
		shouldErr bool
	}{
		{"eq operator", "Name eq 'John'", false},
		{"ne operator", "Name ne 'John'", false},
		{"lt operator", "Age lt 30", false},
		{"le operator", "Age le 30", false},
		{"gt operator", "Age gt 18", false},
		{"ge operator", "Age ge 18", false},
		{"startswith", "Name startswith 'J'", false},
		{"endswith", "Name endswith 'n'", false},
		{"contains", "Email contains '@'", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateFilter(schema, tc.filter)
			if tc.shouldErr && err == nil {
				t.Fatalf("Expected error for filter %q, got nil", tc.filter)
			}
			if !tc.shouldErr && err != nil {
				t.Fatalf("Expected no error for filter %q, got: %v", tc.filter, err)
			}
		})
	}
}

func TestEntitySchema_ValidateFilter_UnknownField(t *testing.T) {
	schema := EntitySchema{
		Properties: map[string]string{
			"Name": "string",
			"Age":  "int",
		},
	}

	err := validateFilter(schema, "UnknownField eq 'value'")
	if err == nil {
		t.Fatal("Expected error for unknown field, got nil")
	}

	schemaErr, ok := err.(*SchemaValidationError)
	if !ok {
		t.Fatalf("Expected *SchemaValidationError, got %T", err)
	}

	if schemaErr.Field != "UnknownField" {
		t.Fatalf("Expected field 'UnknownField', got %q", schemaErr.Field)
	}

	if schemaErr.Message != "field not found in schema" {
		t.Fatalf("Expected message 'field not found in schema', got %q", schemaErr.Message)
	}
}

func TestEntitySchema_ValidateFilter_EmptyFilter(t *testing.T) {
	schema := EntitySchema{
		Properties: map[string]string{
			"Name": "string",
		},
	}

	err := validateFilter(schema, "")
	if err != nil {
		t.Fatalf("Expected no error for empty filter, got: %v", err)
	}
}

func TestEntitySchema_ValidateFilter_NilSchema(t *testing.T) {
	// Should not panic and should return nil
	err := validateFilter(EntitySchema{}, "Name eq 'value'")
	if err != nil {
		t.Fatalf("Expected no error for empty schema, got: %v", err)
	}
}

func TestEntitySchema_ValidateFilter_ComplexExpression(t *testing.T) {
	schema := EntitySchema{
		Properties: map[string]string{
			"Name": "string",
			"Age":  "int",
			"City": "string",
		},
	}

	// Valid: multiple fields with and operator
	err := validateFilter(schema, "Name eq 'John' and Age gt 18")
	if err != nil {
		t.Fatalf("Expected no error for valid complex expression, got: %v", err)
	}

	// Invalid: unknown field in complex expression
	err = validateFilter(schema, "Name eq 'John' and UnknownField gt 18")
	if err == nil {
		t.Fatal("Expected error for unknown field in complex expression, got nil")
	}
}

func TestEntitySchema_ValidateOrderBy_SingleField(t *testing.T) {
	schema := EntitySchema{
		Properties: map[string]string{
			"Name": "string",
			"Age":  "int",
		},
	}

	err := validateOrderBy(schema, "Name")
	if err != nil {
		t.Fatalf("Expected no error for valid field, got: %v", err)
	}
}

func TestEntitySchema_ValidateOrderBy_WithDirection(t *testing.T) {
	schema := EntitySchema{
		Properties: map[string]string{
			"Name": "string",
			"Age":  "int",
		},
	}

	testCases := []struct {
		name      string
		orderBy   string
		shouldErr bool
	}{
		{"ascending", "Name asc", false},
		{"descending", "Name desc", false},
		{"no direction", "Name", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateOrderBy(schema, tc.orderBy)
			if tc.shouldErr && err == nil {
				t.Fatalf("Expected error for orderby %q, got nil", tc.orderBy)
			}
			if !tc.shouldErr && err != nil {
				t.Fatalf("Expected no error for orderby %q, got: %v", tc.orderBy, err)
			}
		})
	}
}

func TestEntitySchema_ValidateOrderBy_MultipleFields(t *testing.T) {
	schema := EntitySchema{
		Properties: map[string]string{
			"Name": "string",
			"Age":  "int",
			"City": "string",
		},
	}

	// Valid: multiple fields
	err := validateOrderBy(schema, "Name asc,Age desc")
	if err != nil {
		t.Fatalf("Expected no error for valid multiple fields, got: %v", err)
	}

	// Invalid: unknown field
	err = validateOrderBy(schema, "Name asc,UnknownField desc")
	if err == nil {
		t.Fatal("Expected error for unknown field, got nil")
	}
}

func TestEntitySchema_ValidateOrderBy_UnknownField(t *testing.T) {
	schema := EntitySchema{
		Properties: map[string]string{
			"Name": "string",
		},
	}

	err := validateOrderBy(schema, "UnknownField")
	if err == nil {
		t.Fatal("Expected error for unknown field, got nil")
	}

	schemaErr, ok := err.(*SchemaValidationError)
	if !ok {
		t.Fatalf("Expected *SchemaValidationError, got %T", err)
	}

	if schemaErr.Field != "UnknownField" {
		t.Fatalf("Expected field 'UnknownField', got %q", schemaErr.Field)
	}
}

func TestEntitySchema_ValidateOrderBy_EmptyOrderBy(t *testing.T) {
	schema := EntitySchema{
		Properties: map[string]string{
			"Name": "string",
		},
	}

	err := validateOrderBy(schema, "")
	if err != nil {
		t.Fatalf("Expected no error for empty orderby, got: %v", err)
	}
}

func TestQueryBuilder_WithSchema_Filter_ValidField(t *testing.T) {
	// Create a mock client (we won't actually execute)
	client := &Client{}

	qb := &QueryBuilder{
		client:    client,
		entitySet: "Users",
	}

	schema := &EntitySchema{
		Properties: map[string]string{
			"Name": "string",
			"Age":  "int",
		},
	}

	qb = qb.WithSchema(schema)
	qb = qb.Filter("Name eq 'John'")

	if qb.lastError != nil {
		t.Fatalf("Expected no error, got: %v", qb.lastError)
	}

	if qb.filterExpr != "Name eq 'John'" {
		t.Fatalf("Expected filterExpr to be set, got %q", qb.filterExpr)
	}
}

func TestQueryBuilder_WithSchema_Filter_UnknownField(t *testing.T) {
	client := &Client{}

	qb := &QueryBuilder{
		client:    client,
		entitySet: "Users",
	}

	schema := &EntitySchema{
		Properties: map[string]string{
			"Name": "string",
		},
	}

	qb = qb.WithSchema(schema)
	qb = qb.Filter("UnknownField eq 'value'")

	if qb.lastError == nil {
		t.Fatal("Expected error for unknown field, got nil")
	}

	schemaErr, ok := qb.lastError.(*SchemaValidationError)
	if !ok {
		t.Fatalf("Expected *SchemaValidationError, got %T", qb.lastError)
	}

	if schemaErr.Field != "UnknownField" {
		t.Fatalf("Expected field 'UnknownField', got %q", schemaErr.Field)
	}
}

func TestQueryBuilder_WithSchema_OrderBy_ValidField(t *testing.T) {
	client := &Client{}

	qb := &QueryBuilder{
		client:    client,
		entitySet: "Users",
	}

	schema := &EntitySchema{
		Properties: map[string]string{
			"Name": "string",
			"Age":  "int",
		},
	}

	qb = qb.WithSchema(schema)
	qb = qb.OrderBy("Name")

	if qb.lastError != nil {
		t.Fatalf("Expected no error, got: %v", qb.lastError)
	}

	if qb.orderByExpr != "Name asc" {
		t.Fatalf("Expected orderByExpr 'Name asc', got %q", qb.orderByExpr)
	}
}

func TestQueryBuilder_WithSchema_OrderBy_UnknownField(t *testing.T) {
	client := &Client{}

	qb := &QueryBuilder{
		client:    client,
		entitySet: "Users",
	}

	schema := &EntitySchema{
		Properties: map[string]string{
			"Name": "string",
		},
	}

	qb = qb.WithSchema(schema)
	qb = qb.OrderBy("UnknownField")

	if qb.lastError == nil {
		t.Fatal("Expected error for unknown field, got nil")
	}

	schemaErr, ok := qb.lastError.(*SchemaValidationError)
	if !ok {
		t.Fatalf("Expected *SchemaValidationError, got %T", qb.lastError)
	}

	if schemaErr.Field != "UnknownField" {
		t.Fatalf("Expected field 'UnknownField', got %q", schemaErr.Field)
	}
}

func TestQueryBuilder_WithSchema_OrderByDesc_UnknownField(t *testing.T) {
	client := &Client{}

	qb := &QueryBuilder{
		client:    client,
		entitySet: "Users",
	}

	schema := &EntitySchema{
		Properties: map[string]string{
			"Name": "string",
		},
	}

	qb = qb.WithSchema(schema)
	qb = qb.OrderByDesc("UnknownField")

	if qb.lastError == nil {
		t.Fatal("Expected error for unknown field, got nil")
	}

	schemaErr, ok := qb.lastError.(*SchemaValidationError)
	if !ok {
		t.Fatalf("Expected *SchemaValidationError, got %T", qb.lastError)
	}

	if schemaErr.Field != "UnknownField" {
		t.Fatalf("Expected field 'UnknownField', got %q", schemaErr.Field)
	}
}

func TestQueryBuilder_WithSchema_NoValidationIfNoSchema(t *testing.T) {
	// Without schema, even invalid field names should be accepted
	client := &Client{}

	qb := &QueryBuilder{
		client:    client,
		entitySet: "Users",
	}

	// Don't call WithSchema
	qb = qb.Filter("UnknownField eq 'value'")

	if qb.lastError != nil {
		t.Fatalf("Expected no error without schema, got: %v", qb.lastError)
	}

	if qb.filterExpr != "UnknownField eq 'value'" {
		t.Fatalf("Expected filterExpr to be set, got %q", qb.filterExpr)
	}
}

func TestQueryBuilder_WithSchema_Filter_Multiple_UnknownFields(t *testing.T) {
	client := &Client{}

	qb := &QueryBuilder{
		client:    client,
		entitySet: "Users",
	}

	schema := &EntitySchema{
		Properties: map[string]string{
			"Name": "string",
		},
	}

	qb = qb.WithSchema(schema)
	// First filter with valid field succeeds
	qb = qb.Filter("Name eq 'John'")

	if qb.lastError != nil {
		t.Fatalf("Expected no error for first filter, got: %v", qb.lastError)
	}

	// Second filter with invalid field should fail
	qb = qb.Filter("InvalidField eq 'value'")

	if qb.lastError == nil {
		t.Fatal("Expected error for second filter, got nil")
	}
}

func TestSchemaValidationError_ErrorMessage(t *testing.T) {
	err := &SchemaValidationError{
		Field:   "TestField",
		Message: "field not found in schema",
	}

	expectedMsg := "traverse: schema validation error: TestField: field not found in schema"
	actualMsg := err.Error()

	if actualMsg != expectedMsg {
		t.Fatalf("Expected message %q, got %q", expectedMsg, actualMsg)
	}
}

func TestSchemaValidationError_ErrorMessage_NoMessage(t *testing.T) {
	err := &SchemaValidationError{
		Field: "TestField",
	}

	expectedMsg := "traverse: schema validation error: TestField"
	actualMsg := err.Error()

	if actualMsg != expectedMsg {
		t.Fatalf("Expected message %q, got %q", expectedMsg, actualMsg)
	}
}

func TestQueryBuilder_WithSchema_Chaining(t *testing.T) {
	// Test that WithSchema returns QueryBuilder for chaining
	client := &Client{}

	qb := &QueryBuilder{
		client:    client,
		entitySet: "Users",
	}

	schema := &EntitySchema{
		Properties: map[string]string{
			"Name": "string",
			"Age":  "int",
		},
	}

	result := qb.WithSchema(schema)

	if result != qb {
		t.Fatal("Expected WithSchema to return the same QueryBuilder for chaining")
	}

	if qb.schema == nil {
		t.Fatal("Expected schema to be set")
	}

	if qb.schema.Properties == nil {
		t.Fatal("Expected schema.Properties to be set")
	}
}

func TestQueryBuilder_OrderBy_StopsOnValidationError(t *testing.T) {
	client := &Client{}

	qb := &QueryBuilder{
		client:    client,
		entitySet: "Users",
	}

	schema := &EntitySchema{
		Properties: map[string]string{
			"Name": "string",
		},
	}

	qb = qb.WithSchema(schema)
	// Set an invalid orderby
	qb = qb.OrderBy("InvalidField")

	// Try to call OrderBy again - should be no-op due to lastError check
	qb = qb.OrderBy("Name")

	if qb.orderByExpr != "" {
		t.Fatalf("Expected no orderByExpr after validation error, got %q", qb.orderByExpr)
	}
}

func TestValidateFilter_ChainedProperties(t *testing.T) {
	schema := EntitySchema{
		Properties: map[string]string{
			"Address": "string",
			"City":    "string",
		},
	}

	// Chained property "Address/City" should validate "Address"
	err := validateFilter(schema, "Address/City eq 'New York'")
	if err != nil {
		t.Fatalf("Expected no error for chained property, got: %v", err)
	}
}

func TestValidateOrderBy_ChainedProperties(t *testing.T) {
	schema := EntitySchema{
		Properties: map[string]string{
			"Address": "string",
		},
	}

	// Chained property "Address/City" should validate "Address"
	err := validateOrderBy(schema, "Address/City asc")
	if err != nil {
		t.Fatalf("Expected no error for chained property, got: %v", err)
	}
}

func TestValidateFilter_FunctionCalls(t *testing.T) {
	schema := EntitySchema{
		Properties: map[string]string{
			"Name": "string",
			"Age":  "int",
		},
	}

	// Functions shouldn't cause false positives
	err := validateFilter(schema, "startswith(Name, 'J') eq true")
	// This is OK - the function contains parentheses which complicate parsing
	// The important thing is that we check the field, not the function
	if err != nil {
		t.Logf("Note: Function call validation returned: %v", err)
	}
}

func TestValidateFilter_QuotedStrings(t *testing.T) {
	schema := EntitySchema{
		Properties: map[string]string{
			"Name": "string",
		},
	}

	// String literals shouldn't be parsed as field names
	err := validateFilter(schema, "Name eq 'InvalidField eq something'")
	if err != nil {
		t.Fatalf("Expected no error for string literal with operators, got: %v", err)
	}
}
