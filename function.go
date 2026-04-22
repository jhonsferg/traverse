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
	basePath   string
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
	basePath   string
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
	method     string
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
		method:     "GET",
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
		jsonBytes, err := json.Marshal(v)
		if err != nil {
			// If marshaling fails, use string representation
			fmt.Fprintf(&buf, "%s=%v", key, v)
			break
		}
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
	paramStr := f.buildParameterString()
	var url string
	if f.basePath != "" {
		url = fmt.Sprintf("/%s/%s(%s)", f.basePath, f.name, paramStr)
	} else {
		url = fmt.Sprintf("/%s(%s)", f.name, paramStr)
	}

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
	var url string
	if a.basePath != "" {
		url = fmt.Sprintf("/%s/%s", a.basePath, a.name)
	} else {
		url = "/" + a.name
	}

	var req *relay.Request
	if a.body != nil {
		req = a.client.http.Post(url)
		req = req.WithJSON(a.body)
	} else if len(a.parameters) > 0 {
		// If no body but parameters exist, POST with JSON parameters
		paramJSON, err := json.Marshal(a.parameters)
		if err != nil {
			return nil, fmt.Errorf("traverse: failed to marshal parameters: %w", err)
		}
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
// Method sets the HTTP method for the function import call (default is "GET").
//
// Use Method("POST") for function imports that modify state or when parameters
// must be sent in the request body instead of the URL.
//
// Returns the receiver for method chaining.
//
// Example:
//
//	err := client.FunctionImport("ProcessQueue").Method("POST").Invoke(ctx, &result)
func (f *FunctionImportBuilder) Method(m string) *FunctionImportBuilder {
	f.method = m
	return f
}

// Invoke calls the function import and decodes the response into result.
//
// For GET requests, parameters are encoded in the URL as FuncName(k=v,...).
// For POST requests, parameters are sent as a JSON body.
//
// The response is unwrapped from the OData {"d":{...}} envelope when present
// before decoding into result. Pass nil to discard the response body.
//
// Returns an error if the HTTP call fails, the status is not 2xx, or JSON
// decoding fails.
//
// Example:
//
//	var stats Stats
//	err := client.FunctionImport("GetStats").Invoke(ctx, &stats)
func (f *FunctionImportBuilder) Invoke(ctx context.Context, result any) error {
	body, err := f.executeRaw(ctx)
	if err != nil {
		return err
	}
	if result == nil {
		return nil
	}

	// Unwrap {"d":{...}} envelope (OData v2).
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		return fmt.Errorf("traverse: function import response parse failed: %w", err)
	}
	payload := body
	if inner, ok := raw["d"]; ok {
		payload = inner
	}
	return json.Unmarshal(payload, result)
}

// InvokeCollection calls the function import and decodes a collection response.
//
// It handles the following OData collection envelope formats:
//   - {"d":{"results":[...]}}  -  OData v2 collection
//   - {"results":[...]}        -  flat results array
//   - {"value":[...]}          -  OData v4 collection
//   - [...]                    -  bare JSON array
//
// results must be a pointer to a slice, or nil to discard.
//
// Returns an error if the HTTP call fails, the status is not 2xx, or JSON
// decoding fails.
//
// Example:
//
//	var orders []Order
//	err := client.FunctionImport("GetOrders").InvokeCollection(ctx, &orders)
func (f *FunctionImportBuilder) InvokeCollection(ctx context.Context, results any) error {
	body, err := f.executeRaw(ctx)
	if err != nil {
		return err
	}
	if results == nil {
		return nil
	}

	// Try to find the collection under common OData wrappers.
	var raw map[string]json.RawMessage
	if jsonErr := json.Unmarshal(body, &raw); jsonErr == nil {
		// {"d":{"results":[...]}}
		if d, ok := raw["d"]; ok {
			var inner map[string]json.RawMessage
			if jsonErr2 := json.Unmarshal(d, &inner); jsonErr2 == nil {
				if arr, ok2 := inner["results"]; ok2 {
					return json.Unmarshal(arr, results)
				}
			}
		}
		// {"results":[...]}
		if arr, ok := raw["results"]; ok {
			return json.Unmarshal(arr, results)
		}
		// {"value":[...]}
		if arr, ok := raw["value"]; ok {
			return json.Unmarshal(arr, results)
		}
	}

	// Bare array fallback.
	return json.Unmarshal(body, results)
}

// executeRaw issues the HTTP request (GET or POST) and returns the raw body bytes.
// It validates the response status and wraps any error.
func (f *FunctionImportBuilder) executeRaw(ctx context.Context) ([]byte, error) {
	var (
		req *relay.Request
		err error
	)

	if f.method == "POST" {
		url := fmt.Sprintf("/%s", f.name)
		var bodyBytes []byte
		if len(f.parameters) > 0 {
			bodyBytes, err = json.Marshal(f.parameters)
			if err != nil {
				return nil, fmt.Errorf("traverse: function import marshal params: %w", err)
			}
		}
		req = f.client.http.Post(url).
			WithBody(bodyBytes).
			WithHeader("Content-Type", "application/json")
	} else {
		paramStr := f.buildParameterString()
		url := fmt.Sprintf("/%s(%s)", f.name, paramStr)
		req = f.client.http.Get(url)
	}

	req = req.WithContext(ctx)
	resp, respErr := f.client.http.Execute(req)
	if respErr != nil {
		return nil, fmt.Errorf("traverse: function import call failed: %w", respErr)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("traverse: function import call returned status %d", resp.StatusCode)
	}
	return resp.Body(), nil
}

// Example:
//
//	result, err := client.FunctionImport("GetTop10Orders").Execute(ctx)
func (f *FunctionImportBuilder) Execute(ctx context.Context) (map[string]interface{}, error) {
	body, err := f.executeRaw(ctx)
	if err != nil {
		return nil, err
	}
	return parseResponseValue(body)
}

// ExecuteFunctionJsonAs is the JSON-format generic version of [FunctionBuilder.Execute].
//
// ExecuteFunctionJsonAs calls the function and unmarshals the response result to type T using JSON.
// It uses [mapToJsonStruct] for type conversion, supporting all Go types with JSON marshaling.
//
// Returns the function result as type T, or an error if the call fails or type conversion fails.
//
// Example:
//
//	type TopProducts struct {
//		Products []Product `json:"products"`
//	}
//
//	result, err := ExecuteFunctionJsonAs[TopProducts](
//		client.Function("GetTopProducts").Param("count", 10),
//		ctx,
//	)
func ExecuteFunctionJsonAs[T any](f *FunctionBuilder, ctx context.Context) (T, error) {
	result, err := f.Execute(ctx)
	var zero T
	if err != nil {
		return zero, err
	}

	return mapToJsonStruct[T](result)
}

// ExecuteFunctionXmlAs is the XML-format generic version of [FunctionBuilder.Execute].
//
// ExecuteFunctionXmlAs calls the function and unmarshals the response result to type T using XML struct tags.
// It uses [mapToJsonStruct] for conversion but expects struct tags to be in xml:"..." format.
//
// Returns the function result as type T, or an error if the call fails or type conversion fails.
//
// Example:
//
//	type TopProducts struct {
//		Products []Product `xml:"products"`
//	}
//
//	result, err := ExecuteFunctionXmlAs[TopProducts](
//		client.Function("GetTopProducts").Param("count", 10),
//		ctx,
//	)
func ExecuteFunctionXmlAs[T any](f *FunctionBuilder, ctx context.Context) (T, error) {
	result, err := f.Execute(ctx)
	var zero T
	if err != nil {
		return zero, err
	}

	return mapToJsonStruct[T](result)
}

// ExecuteFunctionAs is an alias for [ExecuteFunctionJsonAs] for backward compatibility.
// Deprecated: Use [ExecuteFunctionJsonAs] or [ExecuteFunctionXmlAs] instead.
func ExecuteFunctionAs[T any](f *FunctionBuilder, ctx context.Context) (T, error) {
	return ExecuteFunctionJsonAs[T](f, ctx)
}

// ExecuteActionJsonAs is the JSON-format generic version of [ActionBuilder.Execute].
//
// ExecuteActionJsonAs calls the action and unmarshals the response result to type T using JSON.
// It uses [mapToJsonStruct] for type conversion, supporting all Go types with JSON marshaling.
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
//	result, err := ExecuteActionJsonAs[ApprovalResult](
//		client.Action("ApproveOrder").WithBody(approvalData),
//		ctx,
//	)
func ExecuteActionJsonAs[T any](a *ActionBuilder, ctx context.Context) (T, error) {
	result, err := a.Execute(ctx)
	var zero T
	if err != nil {
		return zero, err
	}

	return mapToJsonStruct[T](result)
}

// ExecuteActionXmlAs is the XML-format generic version of [ActionBuilder.Execute].
//
// ExecuteActionXmlAs calls the action and unmarshals the response result to type T using XML struct tags.
// It uses [mapToJsonStruct] for conversion but expects struct tags to be in xml:"..." format.
//
// Returns the action result as type T, or an error if the call fails or type conversion fails.
//
// Example:
//
//	type ApprovalResult struct {
//		Approved bool   `xml:"approved"`
//		Message  string `xml:"message"`
//	}
//
//	result, err := ExecuteActionXmlAs[ApprovalResult](
//		client.Action("ApproveOrder").WithBody(approvalData),
//		ctx,
//	)
func ExecuteActionXmlAs[T any](a *ActionBuilder, ctx context.Context) (T, error) {
	result, err := a.Execute(ctx)
	var zero T
	if err != nil {
		return zero, err
	}

	return mapToJsonStruct[T](result)
}

// ExecuteActionAs is an alias for [ExecuteActionJsonAs] for backward compatibility.
// Deprecated: Use [ExecuteActionJsonAs] or [ExecuteActionXmlAs] instead.
func ExecuteActionAs[T any](a *ActionBuilder, ctx context.Context) (T, error) {
	return ExecuteActionJsonAs[T](a, ctx)
}

// ExecuteFunctionImportJsonAs is the JSON-format generic version of FunctionImportBuilder.Execute.
func ExecuteFunctionImportJsonAs[T any](f *FunctionImportBuilder, ctx context.Context) (T, error) {
	result, err := f.Execute(ctx)
	var zero T
	if err != nil {
		return zero, err
	}

	return mapToJsonStruct[T](result)
}

// ExecuteFunctionImportXmlAs is the XML-format generic version of FunctionImportBuilder.Execute.
func ExecuteFunctionImportXmlAs[T any](f *FunctionImportBuilder, ctx context.Context) (T, error) {
	result, err := f.Execute(ctx)
	var zero T
	if err != nil {
		return zero, err
	}

	return mapToJsonStruct[T](result)
}

// ExecuteFunctionImportAs is an alias for [ExecuteFunctionImportJsonAs] for backward compatibility.
// Deprecated: Use [ExecuteFunctionImportJsonAs] or [ExecuteFunctionImportXmlAs] instead.
func ExecuteFunctionImportAs[T any](f *FunctionImportBuilder, ctx context.Context) (T, error) {
	return ExecuteFunctionImportJsonAs[T](f, ctx)
}

// Invoke calls the function and unmarshals the response into result.
//
// Invoke is the result-receiver variant of [FunctionBuilder.Execute]. It sends an
// HTTP GET request to the OData function URL (with inline parameters) and unmarshals
// the response body into the value pointed to by result.
//
// result must be a non-nil pointer. Invoke uses JSON round-trip via [mapToJsonStruct]
// for the unmarshaling, so standard json struct tags are respected.
//
// Returns an error if the HTTP call fails, the response status is not 2xx, or
// unmarshaling fails.
//
// Example:
//
//	type StatsResult struct {
//		Count int    `json:"count"`
//		Label string `json:"label"`
//	}
//
//	var stats StatsResult
//	err := client.Function("GetStats").Param("top", 10).Invoke(ctx, &stats)
func (f *FunctionBuilder) Invoke(ctx context.Context, result any) error {
	raw, err := f.Execute(ctx)
	if err != nil {
		return err
	}
	return unmarshalInto(raw, result)
}

// Invoke calls the action and unmarshals the response into result.
//
// Invoke is the result-receiver variant of [ActionBuilder.Execute]. It sends an
// HTTP POST request to the OData action URL and unmarshals the response body into
// the value pointed to by result. result may be nil when no response body is expected.
//
// If [ActionBuilder.WithBody] was called, that data forms the request body.
// Otherwise parameters added via [ActionBuilder.Param] are sent as JSON.
//
// Returns an error if the HTTP call fails, the response status is not 2xx, or
// unmarshaling fails.
//
// Example:
//
//	type ApprovalResult struct {
//		Approved bool   `json:"approved"`
//		Message  string `json:"message"`
//	}
//
//	var res ApprovalResult
//	err := client.Action("ApproveOrder").Param("orderID", 42).Invoke(ctx, &res)
func (a *ActionBuilder) Invoke(ctx context.Context, result any) error {
	raw, err := a.Execute(ctx)
	if err != nil {
		return err
	}
	if result == nil {
		return nil
	}
	return unmarshalInto(raw, result)
}

// unmarshalInto converts a map[string]interface{} response into the provided destination.
// It uses JSON round-trip so that standard json struct tags are respected.
func unmarshalInto(raw map[string]interface{}, dest any) error {
	b, err := json.Marshal(raw)
	if err != nil {
		return fmt.Errorf("traverse: failed to marshal response: %w", err)
	}
	if err := json.Unmarshal(b, dest); err != nil {
		return fmt.Errorf("traverse: failed to unmarshal response: %w", err)
	}
	return nil
}
