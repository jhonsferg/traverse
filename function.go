package traverse

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/jhonsferg/relay"
)

// FunctionBuilder provides a fluent API for calling OData Functions (v4).
//
// FunctionBuilder enables execution of OData Function calls, which are read-only
// operations that may take parameters and return values. Functions use HTTP GET
// with parameters encoded in the URL path.
//
// Typical usage:
//
//	result, err := client.Function("GetProducts").
//		Param("category", "Electronics").
//		Param("maxPrice", 1000).
//		Execute(ctx)
type FunctionBuilder struct {
	client     *Client
	name       string
	parameters map[string]interface{}
}

// ActionBuilder provides a fluent API for calling OData Actions (v4).
//
// ActionBuilder enables execution of OData Action calls, which may perform
// side effects (create, update, delete). Actions use HTTP POST and can include
// both body data and parameters.
//
// Typical usage:
//
//	result, err := client.Action("ApproveOrder").
//		WithBody(orderData).
//		Param("approverID", 42).
//		Execute(ctx)
type ActionBuilder struct {
	client     *Client
	name       string
	body       interface{}
	parameters map[string]interface{}
}

// FunctionImportBuilder provides a fluent API for calling OData Function Imports (v2).
//
// FunctionImportBuilder enables execution of OData v2 Function Imports, which are
// similar to v4 Functions but use the v2 protocol. Function Imports use HTTP GET
// with parameters encoded in the URL path.
//
// Typical usage:
//
//	result, err := client.FunctionImport("GetProductsByRating").
//		Param("minRating", 4).
//		Execute(ctx)
type FunctionImportBuilder struct {
	client     *Client
	name       string
	parameters map[string]interface{}
}

// Function starts building a call to an OData Function (v4).
//
// Function creates a new FunctionBuilder for calling OData Functions. Functions
// are read-only operations that may take parameters and return data.
//
// Returns a FunctionBuilder ready for parameter addition and execution.
//
// Example:
//
//	result, err := client.Function("GetProductsByCategory").
//		Param("category", "Electronics").
//		Execute(ctx)
func (c *Client) Function(name string) *FunctionBuilder {
	return &FunctionBuilder{
		client:     c,
		name:       name,
		parameters: make(map[string]interface{}),
	}
}

// Action starts building a call to an OData Action (v4).
//
// Action creates a new ActionBuilder for calling OData Actions. Actions may
// perform side effects and can include both request bodies and parameters.
//
// Returns an ActionBuilder ready for body/parameter configuration and execution.
//
// Example:
//
//	result, err := client.Action("ApproveOrder").
//		WithBody(approvalData).
//		Param("approverID", emp123).
//		Execute(ctx)
func (c *Client) Action(name string) *ActionBuilder {
	return &ActionBuilder{
		client:     c,
		name:       name,
		parameters: make(map[string]interface{}),
	}
}

// FunctionImport starts building a call to an OData Function Import (v2).
//
// FunctionImport creates a new FunctionImportBuilder for calling OData v2 Function Imports.
// Function Imports are similar to v4 Functions and use HTTP GET with URL-encoded parameters.
//
// Returns a FunctionImportBuilder ready for parameter addition and execution.
//
// Example:
//
//	result, err := client.FunctionImport("GetTop10Orders").Execute(ctx)
func (c *Client) FunctionImport(name string) *FunctionImportBuilder {
	return &FunctionImportBuilder{
		client:     c,
		name:       name,
		parameters: make(map[string]interface{}),
	}
}

// Param adds a parameter to the function call.
//
// Param appends a key-value parameter to the function. Parameters are encoded
// in the URL path (e.g., /GetProducts(category='Electronics',maxPrice=1000)).
//
// Returns the receiver for method chaining.
//
// Example:
//
//	f.Param("name", "John").Param("age", 30)
func (f *FunctionBuilder) Param(key string, value interface{}) *FunctionBuilder {
	f.parameters[key] = value
	return f
}

// Param adds a parameter to the action call.
//
// Param appends a key-value parameter for the action. Parameters can be sent
// in the request body as JSON.
//
// Returns the receiver for method chaining.
func (a *ActionBuilder) Param(key string, value interface{}) *ActionBuilder {
	a.parameters[key] = value
	return a
}

// WithBody sets the request body for the action.
//
// WithBody sets the main request body data for the action. The data is
// automatically marshaled to JSON.
//
// Returns the receiver for method chaining.
//
// Example:
//
//	action.WithBody(map[string]interface{}{
//		"orderID": 12345,
//		"approvedBy": "Manager",
//	})
func (a *ActionBuilder) WithBody(data interface{}) *ActionBuilder {
	a.body = data
	return a
}

// Param adds a parameter to the function import.
//
// Param appends a key-value parameter to the function import. Parameters are
// encoded in the URL path, similar to [FunctionBuilder.Param].
//
// Returns the receiver for method chaining.
func (f *FunctionImportBuilder) Param(key string, value interface{}) *FunctionImportBuilder {
	f.parameters[key] = value
	return f
}

// buildParameterString constructs the OData parameter string for function calls.
//
// buildParameterString converts the parameters map to OData format: param1=value1,param2=value2
// Strings are quoted with single quotes and escaped. Numbers, booleans, and null are unquoted.
//
// Returns the formatted parameter string suitable for URL encoding.
func (f *FunctionBuilder) buildParameterString() string {
	if len(f.parameters) == 0 {
		return ""
	}

	var buf bytes.Buffer
	first := true
	for key, val := range f.parameters {
		if !first {
			buf.WriteByte(',')
		}
		buf.Write(formatParameterBytes(key, val))
		first = false
	}

	return buf.String()
}

// buildParameterString constructs the OData parameter string for function imports.
//
// buildParameterString uses the same logic as [FunctionBuilder.buildParameterString],
// converting parameters to OData format for v2 Function Imports.
func (f *FunctionImportBuilder) buildParameterString() string {
	if len(f.parameters) == 0 {
		return ""
	}

	var buf bytes.Buffer
	first := true
	for key, val := range f.parameters {
		if !first {
			buf.WriteByte(',')
		}
		buf.Write(formatParameterBytes(key, val))
		first = false
	}

	return buf.String()
}

// formatParameterBytes formats a single parameter for OData function calls as bytes.
//
// formatParameterBytes converts a key-value pair to OData parameter format:
// - Strings: key='value' (with single quote escaping)
// - Numbers (int, int32, int64, float32, float64): key=value
// - Booleans: key=true or key=false
// - Nil: key=null
// - Complex types: key='<json_encoded>' (serialized as JSON then quoted)
//
// This function optimizes by avoiding intermediate string allocations,
// building the result directly in bytes for URL encoding efficiency.
//
// Example outputs:
//
//	"category='Electronics'"
//	"maxPrice=1000"
//	"active=true"
//	"filter=null"
func formatParameterBytes(key string, value interface{}) []byte {
	var buf bytes.Buffer
	
	switch v := value.(type) {
	case string:
		// Strings are quoted and need escaping
		escaped := strings.ReplaceAll(v, "'", "''")
		buf.WriteString(key)
		buf.WriteByte('=')
		buf.WriteByte('\'')
		buf.WriteString(escaped)
		buf.WriteByte('\'')
	case int:
		buf.WriteString(key)
		buf.WriteByte('=')
		buf.Write(strconv.AppendInt(make([]byte, 0, 20), int64(v), 10))
	case int32:
		buf.WriteString(key)
		buf.WriteByte('=')
		buf.Write(strconv.AppendInt(make([]byte, 0, 20), int64(v), 10))
	case int64:
		buf.WriteString(key)
		buf.WriteByte('=')
		buf.Write(strconv.AppendInt(make([]byte, 0, 20), v, 10))
	case float32:
		buf.WriteString(key)
		buf.WriteByte('=')
		buf.Write(strconv.AppendFloat(make([]byte, 0, 32), float64(v), 'f', -1, 32))
	case float64:
		buf.WriteString(key)
		buf.WriteByte('=')
		buf.Write(strconv.AppendFloat(make([]byte, 0, 32), v, 'f', -1, 64))
	case bool:
		buf.WriteString(key)
		buf.WriteByte('=')
		if v {
			buf.WriteString("true")
		} else {
			buf.WriteString("false")
		}
	case nil:
		buf.WriteString(key)
		buf.WriteString("=null")
	default:
		// For complex types, serialize to JSON
		jsonBytes, _ := json.Marshal(v)
		escaped := strings.ReplaceAll(string(jsonBytes), "'", "''")
		buf.WriteString(key)
		buf.WriteByte('=')
		buf.WriteByte('\'')
		buf.WriteString(escaped)
		buf.WriteByte('\'')
	}
	
	return buf.Bytes()
}

// formatParameter formats a single parameter for OData function calls as string.
//
// formatParameter is a wrapper around [formatParameterBytes] that returns a string
// instead of bytes, for backward compatibility.
func formatParameter(key string, value interface{}) string {
	return string(formatParameterBytes(key, value))
}

// parseResponseValue extracts the response value from an OData response.
//
// parseResponseValue handles various OData response formats:
// - If the response is already a map, return it as-is
// - If the response is an array, wrap it in a "value" property
// - For primitive types, wrap the value in a "value" property
//
// This normalization allows consistent handling of heterogeneous OData responses.
//
// Returns a map containing the response data, or an error if JSON parsing fails.
func parseResponseValue(body []byte) (map[string]interface{}, error) {
	var data interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("traverse: failed to parse response: %w", err)
	}

	// If response is already a map, return it
	if m, ok := data.(map[string]interface{}); ok {
		return m, nil
	}

	// If response is an array, wrap it
	if arr, ok := data.([]interface{}); ok {
		return map[string]interface{}{"value": arr}, nil
	}

	// For primitive types, wrap in value property
	return map[string]interface{}{"value": data}, nil
}

// Execute calls the function and returns the result as a map.
//
// Execute constructs the OData function URL with parameters, sends an HTTP GET
// request, and parses the response into a map. The response format is normalized
// by [parseResponseValue].
//
// Returns a map containing the function result, or an error if the call fails
// or the response status is not 2xx.
//
// Example:
//
//	result, err := client.Function("GetTopProducts").
//		Param("count", 10).
//		Execute(ctx)
//	// result contains the function response data
func (f *FunctionBuilder) Execute(ctx context.Context) (map[string]interface{}, error) {
	// Build URL: /FunctionName(param1=value1,param2=value2)
	paramStr := f.buildParameterString()
	url := fmt.Sprintf("/%s(%s)", f.name, paramStr)

	req := f.client.http.Get(url)
	req = req.WithContext(ctx)

	resp, err := f.client.http.Execute(req)
	if err != nil {
		return nil, fmt.Errorf("traverse: function call failed: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("traverse: function call returned status %d", resp.StatusCode)
	}

	// Parse response
	return parseResponseValue(resp.Body())
}

// Execute calls the action and returns the result as a map.
//
// Execute sends an HTTP POST request to the action with optional body and parameter data.
// The request body is marshaled to JSON automatically. The response is parsed into a map.
//
// If [ActionBuilder.WithBody] was called, that data is sent as the primary body.
// Otherwise, if parameters were added via [ActionBuilder.Param], they are sent as JSON.
// If neither, an empty POST is sent.
//
// Returns a map containing the action response, or an error if the call fails.
//
// Example:
//
//	result, err := client.Action("ApproveOrder").
//		WithBody(approvalData).
//		Execute(ctx)
func (a *ActionBuilder) Execute(ctx context.Context) (map[string]interface{}, error) {
	url := "/" + a.name

	var req *relay.Request
	if a.body != nil {
		req = a.client.http.Post(url)
		req = req.WithJSON(a.body)
	} else if len(a.parameters) > 0 {
		// If no body but parameters exist, POST with JSON parameters
		paramJSON, _ := json.Marshal(a.parameters)
		req = a.client.http.Post(url)
		req = req.WithBody(paramJSON)
		req = req.WithHeader("Content-Type", "application/json")
	} else {
		req = a.client.http.Post(url)
	}

	req = req.WithContext(ctx)

	resp, err := a.client.http.Execute(req)
	if err != nil {
		return nil, fmt.Errorf("traverse: action call failed: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("traverse: action call returned status %d", resp.StatusCode)
	}

	// Parse response
	return parseResponseValue(resp.Body())
}

// Execute calls the function import and returns the result as a map.
//
// Execute constructs the OData v2 function import URL with parameters, sends an HTTP GET
// request, and parses the response into a map. This is similar to [FunctionBuilder.Execute]
// but for OData v2 Function Imports.
//
// Returns a map containing the function import result, or an error if the call fails.
//
// Example:
//
//	result, err := client.FunctionImport("GetTop10Orders").Execute(ctx)
func (f *FunctionImportBuilder) Execute(ctx context.Context) (map[string]interface{}, error) {
	// OData v2 function imports are similar to v4 functions
	paramStr := f.buildParameterString()
	url := fmt.Sprintf("/%s(%s)", f.name, paramStr)

	req := f.client.http.Get(url)
	req = req.WithContext(ctx)

	resp, err := f.client.http.Execute(req)
	if err != nil {
		return nil, fmt.Errorf("traverse: function import call failed: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("traverse: function import call returned status %d", resp.StatusCode)
	}

	// Parse response
	return parseResponseValue(resp.Body())
}

// ExecuteFunctionAs is the generic version of [FunctionBuilder.Execute].
//
// ExecuteFunctionAs calls the function and unmarshals the response result to type T.
// It uses [mapToStruct] for type conversion, supporting all Go types with JSON marshaling.
//
// Returns the function result as type T, or an error if the call fails or type conversion fails.
//
// Example:
//
//	type TopProducts struct {
//		Products []Product `json:"products"`
//	}
//
//	result, err := ExecuteFunctionAs[TopProducts](
//		client.Function("GetTopProducts").Param("count", 10),
//		ctx,
//	)
func ExecuteFunctionAs[T any](f *FunctionBuilder, ctx context.Context) (T, error) {
	result, err := f.Execute(ctx)
	var zero T
	if err != nil {
		return zero, err
	}

	// Use mapToStruct for type conversion
	return mapToStruct[T](result)
}

// ExecuteActionAs is the generic version of [ActionBuilder.Execute].
//
// ExecuteActionAs calls the action and unmarshals the response result to type T.
// It uses [mapToStruct] for type conversion, supporting all Go types with JSON marshaling.
//
// Returns the action result as type T, or an error if the call fails or type conversion fails.
//
// Example:
//
//	type ApprovalResult struct {
//		Approved bool   `json:"approved"`
//		Message  string `json:"message"`
//	}
//
//	result, err := ExecuteActionAs[ApprovalResult](
//		client.Action("ApproveOrder").WithBody(approvalData),
//		ctx,
//	)
func ExecuteActionAs[T any](a *ActionBuilder, ctx context.Context) (T, error) {
	result, err := a.Execute(ctx)
	var zero T
	if err != nil {
		return zero, err
	}

	// Use mapToStruct for type conversion
	return mapToStruct[T](result)
}

// ExecuteFunctionImportAs is the generic version of FunctionImportBuilder.Execute.
func ExecuteFunctionImportAs[T any](f *FunctionImportBuilder, ctx context.Context) (T, error) {
	result, err := f.Execute(ctx)
	var zero T
	if err != nil {
		return zero, err
	}

	// Use mapToStruct for type conversion
	return mapToStruct[T](result)
}
