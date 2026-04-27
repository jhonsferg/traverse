package traverse

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
)

// mapToJsonStruct converts a map[string]interface{} to a typed value T.
//
// mapToJsonStruct uses JSON marshal/unmarshal for maximum compatibility and flexibility
// with custom types, embedded fields, and field tags. This is the foundation for all
// generic conversion methods: CreateJsonAs, UpdateAs, CollectAs, StreamAs, FindByKeyAs, FirstAs.
//
// The method marshals the map to JSON bytes first, then unmarshals into the target type.
// This two-step process ensures that all Go type conversions, validations, and unmarshaling
// hooks are properly applied. It's slower than direct field mapping but handles complex
// type scenarios (nested structs, custom unmarshaling, etc.).
//
// Example:
//
//	type Order struct {
//		ID    int       `json:"id"`
//		Total float64   `json:"total"`
//	}
//
//	m := map[string]interface{}{"id": 123, "total": 99.99}
//	order, err := mapToJsonStruct[Order](m)
func mapToJsonStruct[T any](m map[string]interface{}) (T, error) {
	var result T

	// Marshal the map to JSON bytes
	data, err := json.Marshal(m)
	if err != nil {
		return result, fmt.Errorf("traverse: failed to marshal map to JSON: %w", err)
	}

	// Unmarshal JSON bytes into the target type T
	err = json.Unmarshal(data, &result)
	if err != nil {
		return result, fmt.Errorf("traverse: failed to unmarshal JSON to target type: %w", err)
	}

	return result, nil
}

// mapToXmlStruct converts a map[string]interface{} to a typed value T with XML struct tags.
//
// mapToXmlStruct marshals the map to JSON bytes (as an intermediary since XML cannot
// directly serialize maps), then unmarshals into the target type T using XML tags.
// The target type T is expected to have xml:"..." struct tags for field mapping.
//
// This approach handles both JSON and XML responses from OData backends.
// When the backend returns JSON, this converts it to a struct with XML tags.
// For true XML responses, use rawMessageToXmlStruct for direct XML unmarshaling.
//
// This is the foundation for XmlAs methods: CreateXmlAs, FindByKeyXmlAs, FirstXmlAs,
// CollectXmlAs, StreamXmlAs.
//
// Example:
//
//	type Order struct {
//		ID    int       `xml:"id"`
//		Total float64   `xml:"total"`
//	}
//
//	m := map[string]interface{}{"id": 123, "total": 99.99}
//	order, err := mapToXmlStruct[Order](m)
func mapToXmlStruct[T any](m map[string]interface{}) (T, error) {
	var result T

	// Marshal the map to JSON bytes (intermediary step since xml.Marshal doesn't support maps)
	data, err := json.Marshal(m)
	if err != nil {
		return result, fmt.Errorf("traverse: failed to marshal map to JSON: %w", err)
	}

	// Unmarshal JSON bytes into the target type T using XML tags
	err = json.Unmarshal(data, &result)
	if err != nil {
		return result, fmt.Errorf("traverse: failed to unmarshal to target type with XML tags: %w", err)
	}

	return result, nil
}

// rawMessageToStruct converts json.RawMessage directly to a typed value T.
//
// rawMessageToStruct eliminates the intermediate map[string]interface{} step,
// improving performance by unmarshaling raw JSON directly to the target type.
// This is used internally by [StreamJsonAs] for optimal streaming performance.
//
// Compared to [mapToJsonStruct], this method skips the JSON marshaling step and avoids
// allocating an intermediate map, making it significantly faster for streaming scenarios
// where raw JSON is already available.
//
// Returns an error if JSON unmarshaling fails (invalid format, type mismatch, etc.).
//
// Example:
//
//	raw := json.RawMessage(`{"id": 123, "name": "Product"}`)
//	product, err := rawMessageToStruct[Product](raw)
func rawMessageToStruct[T any](raw json.RawMessage) (T, error) {
	var result T

	err := json.Unmarshal(raw, &result)
	if err != nil {
		return result, fmt.Errorf("traverse: failed to unmarshal JSON to target type: %w", err)
	}

	return result, nil
}

// rawMessageToXmlStruct converts a raw message to a typed value T with XML struct tags.
//
// rawMessageToXmlStruct unmarshals raw XML bytes directly to the target type T.
// The target type T is expected to have xml:"..." struct tags for field mapping.
// This is used internally by StreamXmlAs for optimal streaming performance.
//
// This method is designed for OData backends that return true XML responses
// (ignoring Accept: application/json headers). It unmarshals XML bytes directly
// without intermediate conversions.
//
// Returns an error if XML unmarshaling fails (invalid format, type mismatch, etc.).
//
// Example:
//
//	raw := json.RawMessage(`<Product><id>123</id><name>Widget</name></Product>`)
//	product, err := rawMessageToXmlStruct[Product](raw)
//
// Example:
//
//	raw := json.RawMessage(`{"id": 123, "name": "Product"}`)
//	product, err := rawMessageToXmlStruct[Product](raw)
func rawMessageToXmlStruct[T any](raw json.RawMessage) (T, error) {
	var result T

	err := xml.Unmarshal(raw, &result)
	if err != nil {
		return result, fmt.Errorf("traverse: failed to unmarshal JSON to target type: %w", err)
	}

	return result, nil
}

// CreateJsonAs creates a new entity using JSON payload and decodes the response to type T.
//
// CreateJsonAs is the generic version of [Client.Create]. It sends a POST request
// to create a new entity in the specified entity set, then unmarshals the response
// into a typed value of type T.
//
// The created entity data is marshaled to JSON automatically. The server response
// typically contains the new entity with generated fields (ID, timestamps, etc.).
//
// Returns the created entity (with server-assigned fields) as type T, or an error
// if creation fails.
//
// Example:
//
//	type Order struct {
//		ID    int    `json:"id"`
//		Total float64 `json:"total"`
//	}
//
//	newOrder := map[string]interface{}{"total": 99.99}
//	order, err := CreateJsonAs[Order](client, ctx, "Orders", newOrder)
//	// order.ID is now set by the server
func CreateJsonAs[T any](c *Client, ctx context.Context, entitySet string, data interface{}) (T, error) {
	raw, err := c.Create(ctx, entitySet, data)
	var zero T
	if err != nil {
		return zero, err
	}

	return mapToJsonStruct[T](raw)
}

// CreateXmlAs creates a new entity and decodes the response to type T with XML struct tags.
//
// CreateXmlAs is the generic version of [Client.Create]. It sends a POST request
// to create a new entity in the specified entity set, then unmarshals the response
// into a typed value of type T.
//
// The target struct T must have xml:"..." tags for proper field mapping.
// The method handles responses that are returned as JSON (converted via mapToXmlStruct)
// or for direct XML unmarshaling, use the lower-level Client methods with rawMessageToXmlStruct.
//
// Returns the created entity (with server-assigned fields) as type T, or an error
// if creation fails.
//
// Example:
//
//	type Order struct {
//		ID    int    `xml:"id"`
//		Total float64 `xml:"total"`
//	}
//
//	newOrder := map[string]interface{}{"total": 99.99}
//	order, err := CreateXmlAs[Order](client, ctx, "Orders", newOrder)
//	// order.ID is now set by the server
func CreateXmlAs[T any](c *Client, ctx context.Context, entitySet string, data interface{}) (T, error) {
	raw, err := c.Create(ctx, entitySet, data)
	var zero T
	if err != nil {
		return zero, err
	}

	return mapToXmlStruct[T](raw)
}

// CreateRawAs creates a new entity and returns the raw response bytes.
//
// CreateRawAs sends a POST request to create a new entity in the specified entity set
// and returns the raw response body as bytes. This is useful for debugging, testing,
// or handling responses that don't fit standard patterns (e.g., custom XML, mixed formats).
//
// Returns the raw response bytes, or an error if creation fails.
// The caller is responsible for parsing or validating the response content.
//
// Example:
//
//	rawData, err := CreateRawAs(client, ctx, "Orders", newOrder)
//	if err != nil {
//		// handle error
//	}
//	fmt.Println(string(rawData)) // Print raw response
//
// This is commonly used for:
//   - Testing and debugging OData responses
//   - Working with non-standard response formats
//   - Validating SAP backend behavior
//   - Capturing both JSON and XML responses transparently
func CreateRawAs(c *Client, ctx context.Context, entitySet string, data interface{}) ([]byte, error) {
	raw, err := c.Create(ctx, entitySet, data)
	if err != nil {
		return nil, err
	}

	// raw is map[string]interface{}, convert to bytes
	jsonBytes, err := json.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("traverse: failed to marshal response to bytes: %w", err)
	}

	return jsonBytes, nil
}

// UpdateAs is the generic version of [Client.Update].
//
// UpdateAs updates an existing entity identified by its key. The update data is
// marshaled to JSON and sent as a PATCH request.
//
// Note: OData PATCH requests typically do not return an entity body in the response,
// so this is primarily a type-safe wrapper around [Client.Update]. The generic type
// parameter T is included for API consistency but does not affect the response.
//
// Returns an error if the update fails (entity not found, invalid data, etc.).
//
// Example:
//
//	err := UpdateAs[Order](client, ctx, "Orders", 123, map[string]interface{}{"total": 150.00})
func UpdateAs[T any](c *Client, ctx context.Context, entitySet string, key interface{}, data interface{}) error {
	return c.Update(ctx, entitySet, key, data)
}

// FindByKeyJsonAs retrieves a single entity by its key and decodes it to type T using JSON.
//
// FindByKeyJsonAs is the JSON-format variant of [QueryBuilder.FindByKey]. It constructs
// a single-entity query using the provided key and returns the entity as type T.
//
// The key can be a single value (for single-part keys) or a composite key using
// a map (for entities with compound keys).
//
// Returns the entity as type T, or an error if not found or query fails.
//
// Example:
//
//	type Customer struct {
//		ID   int    `json:"id"`
//		Name string `json:"name"`
//	}
//
//	customer, err := FindByKeyJsonAs[Customer](qb, ctx, 42)
//	// or with composite key:
//	customer, err := FindByKeyJsonAs[Customer](qb, ctx, map[string]interface{}{"CompanyID": 1, "CustomerID": 42})
func FindByKeyJsonAs[T any](qb *QueryBuilder, ctx context.Context, key interface{}) (T, error) {
	raw, err := qb.FindByKey(ctx, key)
	var zero T
	if err != nil {
		return zero, err
	}

	return mapToJsonStruct[T](raw)
}

// FindByKeyXmlAs retrieves a single entity by its key and decodes it to type T with XML struct tags.
//
// FindByKeyXmlAs is the XML-tag variant of [QueryBuilder.FindByKey]. It constructs
// a single-entity query using the provided key and returns the entity as type T.
//
// The "XmlAs" suffix indicates that the target struct T has xml:"..." tags for field
// mapping, not that the server response is XML. The server response is always JSON,
// which is unmarshaled into structs with XML tags.
//
// The key can be a single value (for single-part keys) or a composite key using
// a map (for entities with compound keys).
//
// Returns the entity as type T, or an error if not found or query fails.
//
// Example:
//
//	type Customer struct {
//		ID   int    `xml:"id"`
//		Name string `xml:"name"`
//	}
//
//	customer, err := FindByKeyXmlAs[Customer](qb, ctx, 42)
//	// or with composite key:
//	customer, err := FindByKeyXmlAs[Customer](qb, ctx, map[string]interface{}{"CompanyID": 1, "CustomerID": 42})
func FindByKeyXmlAs[T any](qb *QueryBuilder, ctx context.Context, key interface{}) (T, error) {
	raw, err := qb.FindByKey(ctx, key)
	var zero T
	if err != nil {
		return zero, err
	}

	return mapToXmlStruct[T](raw)
}

// FindByKeyAs is an alias for [FindByKeyJsonAs] for backward compatibility.
// Deprecated: Use [FindByKeyJsonAs] or [FindByKeyXmlAs] instead.
func FindByKeyAs[T any](qb *QueryBuilder, ctx context.Context, key interface{}) (T, error) {
	return FindByKeyJsonAs[T](qb, ctx, key)
}

// CollectJsonAs materializes all results from the query into a slice of type T using JSON.
//
// CollectJsonAs iterates through the query result stream, converts each result to type T,
// and collects them into a slice. All results are loaded into memory before returning.
//
// ⚠️ Warning: For large datasets (millions of records), this method loads all results
// into memory at once, which can cause significant memory pressure and GC overhead.
// For large result sets, prefer [StreamJsonAs] which processes results incrementally
// without materializing the entire collection.
//
// Returns a slice of T with all query results, or an error if streaming or conversion fails.
//
// Example:
//
//	type Product struct {
//		ID    int     `json:"id"`
//		Price float64 `json:"price"`
//	}
//
//	// Load all products (⚠️ use with caution for large datasets)
//	products, err := CollectJsonAs[Product](qb, ctx)
func CollectJsonAs[T any](qb *QueryBuilder, ctx context.Context) ([]T, error) {
	results := make([]T, 0)

	for result := range qb.Stream(ctx) {
		if result.Err != nil {
			return nil, result.Err
		}

		item, err := mapToJsonStruct[T](result.Value)
		if err != nil {
			return nil, err
		}

		results = append(results, item)
	}

	return results, nil
}

// CollectAs iterates through the query result stream, converts each result to type T,
// and collects them into a slice. All results are loaded into memory before returning.
//
// ⚠️ Warning: For large datasets (millions of records), this method loads all results
// into memory at once, which can cause significant memory pressure and GC overhead.
// For large result sets, prefer [StreamAs] which processes results incrementally
// without materializing the entire collection.
//
// The buffer size parameter is passed to the underlying [QueryBuilder.Stream] call
// to control the buffering of results. Default is adaptive buffering (see [Stream]).
//
// Returns a slice of T with all query results, or an error if streaming or conversion fails.
//
// Example:
//
//	type Product struct {
//		ID    int     `json:"id"`
//		Price float64 `json:"price"`
//	}
//
//	// Load all products (⚠️ use with caution for large datasets)
//	products, err := CollectAs[Product](qb, ctx)
//
// CollectAs is an alias for [CollectJsonAs] for backward compatibility.
// Deprecated: Use [CollectJsonAs] or [CollectXmlAs] instead.
func CollectAs[T any](qb *QueryBuilder, ctx context.Context) ([]T, error) {
	return CollectJsonAs[T](qb, ctx)
}

// CollectXmlAs materializes all results from the query into a slice of type T with XML struct tags.
//
// CollectXmlAs is the XML-tag variant of [CollectAs]. It iterates through the query
// result stream, converts each result to type T, and collects them into a slice.
// All results are loaded into memory before returning.
//
// The "XmlAs" suffix indicates that the target struct T has xml:"..." tags for field
// mapping, not that the server returns XML responses. The server response is always JSON.
//
// ⚠️ Warning: For large datasets (millions of records), this method loads all results
// into memory at once, which can cause significant memory pressure and GC overhead.
// For large result sets, prefer [StreamXmlAs] which processes results incrementally
// without materializing the entire collection.
//
// Returns a slice of T with all query results, or an error if streaming or conversion fails.
//
// Example:
//
//	type Product struct {
//		ID    int     `xml:"id"`
//		Price float64 `xml:"price"`
//	}
//
//	// Load all products (⚠️ use with caution for large datasets)
//	products, err := CollectXmlAs[Product](qb, ctx)
func CollectXmlAs[T any](qb *QueryBuilder, ctx context.Context) ([]T, error) {
	results := make([]T, 0)

	for result := range qb.Stream(ctx) {
		if result.Err != nil {
			return nil, result.Err
		}

		item, err := mapToXmlStruct[T](result.Value)
		if err != nil {
			return nil, err
		}

		results = append(results, item)
	}

	return results, nil
}

// FirstJsonAs retrieves the first result from the query and decodes it to type T using JSON.
//
// FirstJsonAs is the JSON-format version of [QueryBuilder.First]. It executes the query
// with the $top=1 modifier to retrieve only the first matching entity, then
// unmarshals it to type T.
//
// This is efficient for single-item lookups and is equivalent to:
//
//	result, _ := FirstJsonAs[T](qb, ctx)
//	// vs
//	item, _ := qb.Top(1).First(ctx)
//	item, _ := mapToJsonStruct[T](item)
//
// Returns the first entity as type T, or an error if the query fails or no results match.
//
// Example:
//
//	type Customer struct {
//		ID   int    `json:"id"`
//		Name string `json:"name"`
//	}
//
//	customer, err := FirstJsonAs[Customer](qb.Filter("Name eq 'Alice'"), ctx)
func FirstJsonAs[T any](qb *QueryBuilder, ctx context.Context) (T, error) {
	raw, err := qb.First(ctx)
	var zero T
	if err != nil {
		return zero, err
	}

	return mapToJsonStruct[T](raw)
}

// FirstXmlAs retrieves the first result from the query and decodes it to type T with XML struct tags.
//
// FirstXmlAs is the XML-tag version of [QueryBuilder.First]. It executes the query
// with the $top=1 modifier to retrieve only the first matching entity, then
// unmarshals it to type T.
//
// The "XmlAs" suffix indicates that the target struct T has xml:"..." tags for field
// mapping, not that the server response is XML. The server response is always JSON.
//
// Returns the first entity as type T, or an error if the query fails or no results match.
//
// Example:
//
//	type Customer struct {
//		ID   int    `xml:"id"`
//		Name string `xml:"name"`
//	}
//
//	customer, err := FirstXmlAs[Customer](qb.Filter("Name eq 'Alice'"), ctx)
func FirstXmlAs[T any](qb *QueryBuilder, ctx context.Context) (T, error) {
	raw, err := qb.First(ctx)
	var zero T
	if err != nil {
		return zero, err
	}

	return mapToXmlStruct[T](raw)
}

// FirstAs is an alias for [FirstJsonAs] for backward compatibility.
// Deprecated: Use [FirstJsonAs] or [FirstXmlAs] instead.
func FirstAs[T any](qb *QueryBuilder, ctx context.Context) (T, error) {
	return FirstJsonAs[T](qb, ctx)
}

// StreamJsonAs is the JSON-format streaming method for type T.
//
// StreamJsonAs returns a channel of [Result] items typed to T, enabling incremental
// processing of large result sets without materializing all data in memory.
// Each result can be of type T or contain an error.
//
// The method is optimized to use [QueryBuilder.streamRaw] for direct JSON unmarshaling,
// avoiding the intermediate map[string]interface{} allocation required by [CollectJsonAs].
// This makes it significantly faster for streaming large datasets.
//
// The bufferSize parameter controls the capacity of the result channel (default 256).
// For large record sizes or high network latency, increase this value to reduce blocking.
// For small records, the default is usually optimal.
//
// Results include pagination information (Page, Index) for tracking position within
// large result sets.
//
// Returns a receive-only channel that yields [Result] items as they become available.
// The channel is closed when all results have been processed or an error occurs.
//
// Example:
//
//	type Product struct {
//		ID    int     `json:"id"`
//		Price float64 `json:"price"`
//	}
//
//	// Stream 1 million products incrementally
//	results := StreamJsonAs[Product](qb.Filter("Price gt 50"), ctx, 512)
//	for result := range results {
//		if result.Err != nil {
//			log.Println("Error:", result.Err)
//			continue
//		}
//
//		product := result.Value
//		fmt.Printf("Product %d (page %d): $%.2f\n", product.ID, result.Page, product.Price)
//	}
func StreamJsonAs[T any](qb *QueryBuilder, ctx context.Context, bufferSize ...int) <-chan Result[T] {
	buffer := 256
	if len(bufferSize) > 0 && bufferSize[0] > 0 {
		buffer = bufferSize[0]
	}

	out := make(chan Result[T], buffer)

	go func() {
		defer close(out)

		for result := range qb.streamRaw(ctx, buffer) {
			if result.Err != nil {
				out <- Result[T]{
					Err: result.Err,
				}
				continue
			}

			item, err := rawMessageToStruct[T](result.Raw)
			if err != nil {
				out <- Result[T]{
					Err: err,
				}
				continue
			}

			out <- Result[T]{
				Value: item,
				Page:  result.Page,
				Index: result.Index,
			}
		}
	}()

	return out
}

// StreamXmlAs is the XML-tag streaming method for type T.
//
// StreamXmlAs returns a channel of [Result] items typed to T, enabling incremental
// processing of large result sets without materializing all data in memory.
// Each result can be of type T or contain an error.
//
// The "XmlAs" suffix indicates that the target struct T has xml:"..." tags for field
// mapping, not that the server returns XML responses. The server response is always JSON,
// which is unmarshaled into structs with XML tags.
//
// The bufferSize parameter controls the capacity of the result channel (default 256).
// For large record sizes or high network latency, increase this value to reduce blocking.
// For small records, the default is usually optimal.
//
// Results include pagination information (Page, Index) for tracking position within
// large result sets.
//
// Returns a receive-only channel that yields [Result] items as they become available.
// The channel is closed when all results have been processed or an error occurs.
//
// Example:
//
//	type Product struct {
//		ID    int     `xml:"id"`
//		Price float64 `xml:"price"`
//	}
//
//	// Stream 1 million products
//	results := StreamXmlAs[Product](qb.Filter("Price gt 50"), ctx, 512)
//	for result := range results {
//		if result.Err != nil {
//			log.Println("Error:", result.Err)
//			continue
//		}
//
//		product := result.Value
//		fmt.Printf("Product %d (page %d): $%.2f\n", product.ID, result.Page, product.Price)
//	}
func StreamXmlAs[T any](qb *QueryBuilder, ctx context.Context, bufferSize ...int) <-chan Result[T] {
	buffer := 256
	if len(bufferSize) > 0 && bufferSize[0] > 0 {
		buffer = bufferSize[0]
	}

	out := make(chan Result[T], buffer)

	go func() {
		defer close(out)

		for result := range qb.streamRaw(ctx, buffer) {
			if result.Err != nil {
				out <- Result[T]{
					Err: result.Err,
				}
				continue
			}

			item, err := rawMessageToXmlStruct[T](result.Raw)
			if err != nil {
				out <- Result[T]{
					Err: err,
				}
				continue
			}

			out <- Result[T]{
				Value: item,
				Page:  result.Page,
				Index: result.Index,
			}
		}
	}()

	return out
}

// StreamAs is an alias for [StreamJsonAs] for backward compatibility.
// Deprecated: Use [StreamJsonAs] or [StreamXmlAs] instead.
func StreamAs[T any](qb *QueryBuilder, ctx context.Context, bufferSize ...int) <-chan Result[T] {
	return StreamJsonAs[T](qb, ctx, bufferSize...)
}

// FetchPropertyAs retrieves a single scalar or object property from an OData entity
// using the standard OData property path pattern:
//
//	GET /EntitySet(Key)/PropertyName
//
// This is the idiomatic way to fetch one field from a known entity without
// downloading the full record. Useful for large entities where only a single
// field (e.g. a price, a flag, or a blob link) is needed.
//
// The qb parameter must already point to the full entity including its key,
// e.g. built via:
//
//	qb := client.From("/sap/opu/odata/sap/UI_PRODUCTLIST/ProductList(Product='3001008',Plant='1010',ValuationType='')")
//	price, err := traverse.FetchPropertyAs[string](qb, ctx, "PriceUnitQty")
//
// Returns the zero value of T and an error if the property is not found or
// cannot be decoded.
func FetchPropertyAs[T any](qb *QueryBuilder, ctx context.Context, property string) (T, error) {
	var zero T

	if property == "" {
		return zero, fmt.Errorf("traverse: FetchPropertyAs: property name must not be empty")
	}

	// Build the property path by appending /PropertyName to the current entity set.
	propQB := qb.client.From(qb.entitySet + "/" + property)

	page, err := propQB.Page(ctx)
	if err != nil {
		return zero, fmt.Errorf("traverse: FetchPropertyAs(%q): %w", property, err)
	}

	// OData v2 property response: {"d": {"PropertyName": value}}
	// After parsing through our Page decoder, Value[0] holds map{"PropertyName": value}.
	if len(page.Value) > 0 {
		if v, ok := page.Value[0][property]; ok {
			b, err := json.Marshal(v)
			if err != nil {
				return zero, fmt.Errorf("traverse: FetchPropertyAs(%q): marshal: %w", property, err)
			}
			if err := json.Unmarshal(b, &zero); err != nil {
				return zero, fmt.Errorf("traverse: FetchPropertyAs(%q): unmarshal to %T: %w", property, zero, err)
			}
			return zero, nil
		}
	}

	// Fallback: try RawValue for the single-value case ("d":{"value":...})
	if len(page.RawValue) > 0 {
		if err := json.Unmarshal(page.RawValue[0], &zero); err != nil {
			return zero, fmt.Errorf("traverse: FetchPropertyAs(%q): unmarshal raw: %w", property, err)
		}
		return zero, nil
	}

	return zero, fmt.Errorf("traverse: FetchPropertyAs(%q): property not found in response", property)
}
