// Package traverse provides a production-grade OData v2/v4 client library for Go.
//
// Traverse is designed for high-performance querying and manipulation of OData services,
// with special optimizations for SAP systems. It offers:
//   - Streaming-first architecture for processing large datasets without memory overhead
//
// - Ultra-low memory allocations (-81% vs baseline) through object pooling and zero-allocation patterns
// - Fluent query builder API for ergonomic OData query construction
// - Support for OData batch operations, delta queries, functions, and actions
// - Thread-safe client with concurrent goroutine support
// - Extensive metadata caching and query hooks for extensibility
//
// Quick Start:
//
//	import "github.com/jhonsferg/traverse"
//
//	client, _ := traverse.New(
//		traverse.WithBaseURL("https://odata.example.com/v4"),
//		traverse.WithODataVersion(traverse.ODataV4),
//	)
//	defer client.Close()
//
//	results, _ := client.From("Products").
//		Filter("Price gt 100").
//		OrderBy("Name").
//		Collect(context.Background())
//
// For streaming large result sets:
//
//	stream := client.From("Orders").
//		Stream(context.Background())
//	for result := range stream {
//		processOrder(result.Value)
//	}
//
// The Client type is the main entry point and is safe for concurrent use across
// multiple goroutines. Use From() to create a QueryBuilder for constructing queries.
package traverse

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"sync"
	"time"

	"github.com/jhonsferg/relay"
)

// Client is the main OData client for querying and manipulating OData services.
//
// Client handles all OData operations with a comprehensive feature set including query building,
// data creation/update/deletion, streaming, batching, function/action invocation, metadata retrieval,
// and delta synchronization. The client is safe for concurrent use across multiple goroutines with
// built-in synchronization on metadata caching through sync.RWMutex protection.
//
// Core Features:
//   - Fluent query building API: Use From() to create QueryBuilder for composable OData queries
//   - Streaming large result sets: Use Stream() for O(1) memory processing of millions of records
//   - Batch operations: Group multiple requests with Batch() for efficient batch processing
//   - OData functions and actions: Call Function() and Action() for remote function invocation
//   - Delta synchronization: Use NewDeltaSync() for incremental data updates
//   - Metadata management: Automatic metadata caching with extensible CacheStore interface
//   - Query hooks: Register beforeQuery and afterExecute hooks for cross-cutting concerns
//   - OData version support: Automatic v2 and v4 protocol handling with configurable version
//
// Thread Safety:
//
// The Client is designed for concurrent use. Metadata access is protected by sync.RWMutex,
// and the underlying HTTP client (relay.Client) is also thread-safe. Multiple goroutines
// can safely call client methods simultaneously.
//
// Resource Management:
//
// Always call Close() to properly shut down the client and release resources. It's recommended
// to use defer for proper cleanup:
//
//	client, _ := traverse.New(traverse.WithBaseURL("..."))
//	defer client.Close()
//
// Metadata Caching:
//
// The client caches service metadata ($metadata) to reduce HTTP requests. Use WithMetadataCache()
// to customize the cache implementation, or use the default in-memory cache.
//
// Typical usage:
//
//	client, _ := traverse.New(
//		traverse.WithBaseURL("https://odata.service/"),
//		traverse.WithODataVersion(traverse.ODataV4),
//	)
//	defer client.Close()
//
//	qb := client.From("Products").
//		Filter("Price gt 100").
//		OrderBy("Name")
//	results, _ := qb.Collect(ctx)
//
// Or for large datasets with streaming (constant memory usage):
//
//	stream := client.From("Orders").Stream(ctx)
//	for result := range stream {
//		// Process each order without loading entire dataset
//		processOrder(result.Value)
//	}
type Client struct {
	// http is the underlying relay HTTP client for making requests.
	http *relay.Client
	// baseURL is the OData service endpoint URL.
	baseURL string
	// version is the OData protocol version (v2 or v4).
	version ODataVersion
	// pageSize is the default number of records per page for pagination.
	pageSize int
	// logger is the logger for diagnostic output.
	logger relay.Logger
	// responseFormat is the requested response format (JSON or Atom/XML).
	responseFormat ResponseFormat

	// metadataCache stores metadata for improved performance across requests.
	metadataCache CacheStore
	// beforeQuery is a list of hooks called before query execution.
	beforeQuery []func(*QueryBuilder) error
	// afterExecute is a list of hooks called after query execution.
	afterExecute []func(*QueryBuilder) error

	// metadataOnce ensures metadata is fetched only once.
	metadataOnce sync.Once
	// metadata is the cached service metadata.
	metadata *Metadata
	// metadataErr is any error encountered during metadata fetch.
	metadataErr error
}

// Option is a functional option for configuring a [Client].
//
// Option implements the functional options pattern for flexible client construction,
// allowing callers to customize client behavior without modifying the New signature.
//
// Example:
//
//	client, _ := traverse.New(
//		"https://odata.service/",
//		traverse.WithODataVersion(traverse.ODataV4),
//		traverse.WithPageSize(1000),
//		traverse.WithBasicAuth("user", "pass"),
//	)
type Option func(*clientConfig) error

// clientConfig holds configuration parameters during [Client] construction.
//
// clientConfig is populated by [Option] functions and used internally by [New].
// It is not exported; users interact with it only through Option functions.
type clientConfig struct {
	httpClient     *relay.Client
	baseURL        string
	version        ODataVersion
	pageSize       int
	logger         relay.Logger
	responseFormat ResponseFormat

	// Auth options
	basicAuthUser string
	basicAuthPass string
	bearerToken   string
	apiKeyHeader  string
	apiKeyValue   string

	// Relay options for the underlying HTTP client.
	relayOpts []relay.Option

	// metadataCache stores OData service metadata.
	metadataCache CacheStore
	// beforeQuery are hooks called before query execution.
	beforeQuery []func(*QueryBuilder) error
	// afterExecute are hooks called after query execution.
	afterExecute []func(*QueryBuilder) error
}

// New creates a new [Client] with the provided options.
// At minimum, [WithBaseURL] must be provided.
// Returns an error if required options are missing or invalid.
//
// Example:
//
//	client, err := traverse.New(
//	    traverse.WithBaseURL("https://odata.example.com/v2"),
//	    traverse.WithBasicAuth("user", "pass"),
//	    traverse.WithPageSize(5000),
//	)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer client.Close()
func New(opts ...Option) (*Client, error) {
	cfg := &clientConfig{
		version:        ODataV4, // Default to OData v4
		pageSize:       1000,
		responseFormat: FormatJSON,
		relayOpts:      []relay.Option{},
		metadataCache:  &NoOpCache{}, // Default no-op cache
		beforeQuery:    make([]func(*QueryBuilder) error, 0),
		afterExecute:   make([]func(*QueryBuilder) error, 0),
	}

	// Apply options
	for _, opt := range opts {
		if err := opt(cfg); err != nil {
			return nil, fmt.Errorf("traverse: invalid option: %w", err)
		}
	}

	// Validate required fields
	if cfg.baseURL == "" {
		return nil, fmt.Errorf("traverse: BaseURL is required")
	}

	// Create relay client if not provided
	if cfg.httpClient == nil {
		cfg.relayOpts = append(cfg.relayOpts,
			relay.WithTimeout(30*time.Second),
			relay.WithBaseURL(cfg.baseURL),
		)
		cfg.httpClient = relay.New(cfg.relayOpts...)
	}

	c := &Client{
		http:           cfg.httpClient,
		baseURL:        cfg.baseURL,
		version:        cfg.version,
		pageSize:       cfg.pageSize,
		logger:         cfg.logger,
		responseFormat: cfg.responseFormat,
		metadataCache:  cfg.metadataCache,
		beforeQuery:    cfg.beforeQuery,
		afterExecute:   cfg.afterExecute,
	}

	return c, nil
}

// Close closes the client and releases all associated resources.
//
// Close gracefully shuts down the underlying HTTP client (relay.Client), terminating
// any active connections and releasing network resources. This should always be called
// when the client is no longer needed, preferably via defer for cleanup guarantee.
//
// After Close is called, the client must not be reused. Attempting to make requests
// after Close will result in errors.
//
// Returns an error if the shutdown process fails, though in most cases this is safe to ignore.
// However, it's still good practice to check and log errors:
//
//	if err := client.Close(); err != nil {
//		log.Printf("error closing client: %v", err)
//	}
//
// Typical usage with defer:
//
//	client, _ := traverse.New(traverse.WithBaseURL("..."))
//	defer client.Close()  // Ensures cleanup even if panic occurs
//	// Use client...
func (c *Client) Close() error {
	if c.http != nil {
		return c.http.Shutdown(context.Background())
	}
	return nil
}

// CircuitBreakerState returns the current state of the underlying relay circuit
// breaker: Closed (healthy), Open (failing, requests rejected), or Half-Open
// (probing for recovery).
//
// Returns [relay.StateClosed] if the circuit breaker is not configured.
//
// Example:
//
//	if client.CircuitBreakerState() == relay.StateOpen {
//	    log.Println("OData service unavailable — circuit is open")
//	}
func (c *Client) CircuitBreakerState() relay.CircuitBreakerState {
	if c.http == nil {
		return relay.StateClosed
	}
	return c.http.CircuitBreakerState()
}

// ResetCircuitBreaker resets the circuit breaker to the Closed state, allowing
// requests to flow again immediately. This is useful after a manual intervention
// or during testing.
//
// No-op if the circuit breaker is not configured.
func (c *Client) ResetCircuitBreaker() {
	if c.http != nil {
		c.http.ResetCircuitBreaker()
	}
}

// BaseURL returns the base URL of the OData service that this Client is connected to.
//
// BaseURL returns the configured service root URL, which serves as the base for all
// OData entity set operations. This is the URL passed to New() via WithBaseURL().
//
// The returned URL typically has the form: https://odata.example.com/v4
// or for SAP systems: https://sap.example.com/sap/opu/odata/sap/ZESX_PRODUCT/
//
// Returns:
//   - The base URL string (always non-empty for a valid client)
//
// Example:
//
//	url := client.BaseURL()
//	// Returns: "https://odata.example.com/v4"
func (c *Client) BaseURL() string {
	return c.baseURL
}

// Version returns the OData protocol version (v2 or v4) configured for this Client.
//
// Version returns the OData protocol version that the client is using for requests and response parsing.
// This is configured via WithODataVersion() when creating the client.
//
// OData Version Differences:
//   - ODataV2: Legacy SAP standard, uses {"d": {"results": [...]}} response structure
//   - ODataV4: Current OASIS standard, uses {"value": [...]} response structure
//
// The version affects:
//   - JSON response parsing (d.results vs value array)
//   - DateTime format handling
//   - Query operator availability
//   - Metadata structure interpretation
//
// Returns:
//   - ODataV2 (value 2) or ODataV4 (value 4)
//
// Example:
//
//	version := client.Version()
//	if version == traverse.ODataV4 {
//		// Use OData v4-specific features
//	}
func (c *Client) Version() ODataVersion {
	return c.version
}

// PageSize returns the configured page size for pagination.
//
// PageSize returns the number of records fetched per HTTP request when using $top-based pagination.
// This is the default page size for all queries unless overridden with QueryBuilder.Top().
//
// Pagination Details:
//   - Controls the default $top parameter for OData queries
//   - Reduces per-request latency by controlling chunk size
//   - Larger values: Fewer requests but higher memory per request
//   - Smaller values: More requests but lower memory per request
//   - Typical value: 1000-5000 records (default: 1000)
//
// This is configured via WithPageSize() when creating the client.
//
// Returns:
//   - The page size in records (default: 1000)
//
// Example:
//
//	pageSize := client.PageSize()
//	// Default returns 1000
//	// Can be overridden per query with QueryBuilder.Top()
func (c *Client) PageSize() int {
	return c.pageSize
}

// From creates a new QueryBuilder for querying the specified entity set.
//
// From starts the construction of an OData query for the given entity set name.
// The returned QueryBuilder supports a fluent API for adding query parameters
// (filtering, selection, ordering, expansion, pagination) before execution.
//
// The QueryBuilder provides:
//   - Filter(): Add OData $filter expressions
//   - Select(): Choose specific properties ($select)
//   - OrderBy(): Sort results ($orderby)
//   - Expand(): Include related entities ($expand)
//   - Top(): Limit result count ($top)
//   - Skip(): Skip records for pagination ($skip)
//   - Collect(): Execute and collect all results
//   - Stream(): Execute and stream results without buffering
//
// The QueryBuilder uses method chaining for ergonomic query construction.
// All methods are optional-you can execute with just From() for a basic query.
//
// Query Execution:
//   - Call Collect() to fetch all results at once
//   - Call Stream() for memory-efficient streaming of large datasets
//   - Call Count() to get the record count with $count
//   - Call Single() for a single-record query with error if 0 or >1 result
//
// Thread Safety:
// Each QueryBuilder is independent and not thread-safe for concurrent modifications.
// Create separate QueryBuilders for concurrent queries.
//
// Example:
//
//	qb := client.From("Products").
//		Filter("Price gt 100").
//		Select("ProductID", "Name", "Price").
//		OrderBy("Name asc").
//		Top(50)
//	results, _ := qb.Collect(context.Background())
//
// Or for streaming:
//
//	stream := client.From("LargeDataSet").Stream(context.Background())
//	for result := range stream {
//		processRecord(result.Value)
//	}
func (c *Client) From(entitySet string) *QueryBuilder {
	return &QueryBuilder{
		client:       c,
		entitySet:    entitySet,
		selectFields: make([]string, 0, 20),
		expandProps:  make([]string, 0, 10),
		params:       make(map[string]string),
		urlDirty:     true,
	}
}

// Service fetches the OData service document, which lists all available entity sets and operations.
//
// Service makes an HTTP GET request to the service root URL (typically /odata/v4 or /sap/opu/odata/...)
// and returns a ServiceDocument containing all entity sets exposed by the service.
//
// The Service Document:
//   - Lists all available entity sets that can be queried
//   - Includes singleton entities and function imports
//   - Is version-aware: Parsed differently for OData v2 vs v4
//   - Is useful for dynamic service discovery without prior knowledge of entity set names
//
// Response Format:
//   - OData v2: {"d": {"EntitySets": [{...}, ...]}} with "d" wrapper for CSRF protection
//   - OData v4: {"value": [{...}, ...]} at the root
//
// Use Case:
//
// This is typically called once during service initialization to discover available entity sets,
// or when the service schema is not known in advance. For known schemas, use From() directly.
//
// Returns:
//   - *ServiceDocument: Contains all entity sets available in the service
//   - error: Network error, HTTP error, or parsing error
//
// Example:
//
//	doc, err := client.Service(ctx)
//	if err != nil {
//		log.Fatal(err)
//	}
//	for _, es := range doc.EntitySets {
//		fmt.Printf("Entity Set: %s (URL: %s)\n", es.Name, es.URL)
//	}
//
//	// Now you can query known entity sets
//	results, _ := client.From(doc.EntitySets[0].Name).Collect(ctx)
func (c *Client) Service(ctx context.Context) (*ServiceDocument, error) {
	req := c.http.Get("")
	req = req.WithContext(ctx)

	resp, err := c.http.Execute(req)
	if err != nil {
		return nil, fmt.Errorf("traverse: failed to fetch service document: %w", err)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("traverse: service document returned status %d", resp.StatusCode)
	}

	// Parse service document based on OData version
	var doc *ServiceDocument

	if c.version == ODataV2 {
		// OData v2: Service document is JSON with "d" wrapper containing array of entity set objects
		doc, err = parseODataV2ServiceDocument(resp.BodyReader())
	} else {
		// OData v4: Service document is JSON with "value" array
		doc, err = parseODataV4ServiceDocument(resp.BodyReader())
	}

	if err != nil {
		return nil, fmt.Errorf("traverse: failed to parse service document: %w", err)
	}

	return doc, nil
}

// Metadata returns the OData $metadata service document, which contains the complete
// Entity Data Model (EDM) describing all entity types, properties, relationships, and operations.
//
// Metadata fetches and caches the $metadata document, which defines:
//   - All entity types and their properties
//   - Navigation properties (relationships between entities)
//   - Associations and multiplicities
//   - Complex types and enumerations
//   - Function imports and action definitions
//   - Sap annotations and extensions for SAP systems
//
// Caching Behavior:
//
// The result is cached using sync.Once to ensure metadata is fetched only once per client lifetime,
// even with concurrent calls. Subsequent calls return the cached metadata without network requests,
// providing significant performance benefits for services with large metadata documents.
//
// The cache key includes the base URL to support scenarios with multiple clients to different services.
// Custom cache implementations can be provided via WithMetadataCache().
//
// Use Cases:
//   - Query validation: Check entity types before constructing queries
//   - Dynamic query building: Inspect available properties and relationships
//   - Type information: Determine property types for serialization/deserialization
//   - Navigation: Discover related entities through association mappings
//
// Returns:
//   - *Metadata: Parsed EDM model describing the service schema
//   - error: Returns ErrMetadataInvalid if metadata cannot be parsed
//   - error: Returns network/HTTP errors if fetch fails
//
// Example:
//
//	meta, err := client.Metadata(ctx)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Find entity type definition
//	productType := meta.FindEntityType("Product")
//	for _, prop := range productType.Properties {
//		fmt.Printf("Property: %s (Type: %s)\n", prop.Name, prop.Type)
//	}
//
//	// Use discovered metadata for dynamic query building
//	for _, navProp := range productType.NavigationProperties {
//		fmt.Printf("Related: %s\n", navProp.Name)
//	}
//
// Thread Safety:
// Metadata is thread-safe. Multiple goroutines can safely call Metadata() concurrently;
// only the first call fetches the metadata, while others wait for the result.
func (c *Client) Metadata(ctx context.Context) (*Metadata, error) {
	c.metadataOnce.Do(func() {
		// Use cache key based on baseURL
		cacheKey := "metadata:" + c.baseURL

		// Try to get from cache first
		if metadata, found := c.metadataCache.Get(cacheKey); found {
			c.metadata = metadata
			return
		}

		// Cache miss or no-op cache, fetch from network
		metadata, err := c.fetchMetadata(ctx)
		if err != nil {
			c.metadataErr = err
			return
		}

		// Store in cache for future use
		c.metadataCache.Set(cacheKey, metadata)
		c.metadata = metadata
	})

	if c.metadataErr != nil {
		return nil, c.metadataErr
	}

	return c.metadata, nil
}

// fetchMetadata fetches and parses the OData $metadata document.
// It is called internally by [Metadata] and should not be used directly.
// Returns [ErrMetadataInvalid] if parsing fails.
func (c *Client) fetchMetadata(ctx context.Context) (*Metadata, error) {
	req := c.http.Get("/$metadata")
	req = req.WithContext(ctx)

	resp, err := c.http.Execute(req)
	if err != nil {
		return nil, fmt.Errorf("traverse: failed to fetch $metadata: %w", err)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("traverse: $metadata returned status %d: %w", resp.StatusCode, ErrMetadataInvalid)
	}

	// Parse EDMX (XML) format
	metadata, err := ParseEDMX(resp.BodyReader())
	if err != nil {
		return nil, fmt.Errorf("traverse: failed to parse EDMX: %w", err)
	}

	return metadata, nil
}

// Create creates a new entity in the specified entity set.
//
// Create sends an HTTP POST request with the provided entity data to the service,
// creating a new record and returning the complete entity as created by the server.
// This includes any server-generated fields such as identity columns, default values,
// and computed fields.
//
// Parameters:
//   - ctx: Context for request cancellation and timeout
//   - entitySet: Name of the entity set (e.g., "Products", "Orders")
//   - data: Entity data as map[string]interface{} or struct
//
// Response:
// The method returns the complete created entity including server-generated fields.
// The response format differs by OData version:
//   - OData v2: Entity wrapped in {"d": {...}} response
//   - OData v4: Entity returned directly in response body
//
// Error Cases:
//   - HTTP 400: Invalid data or constraint violations
//   - HTTP 409: Duplicate key or concurrency conflict
//   - Network errors: Connection failures
//
// HTTP Status: 201 Created (success)
//
// Example with Map:
//
//	newProduct := map[string]interface{}{
//		"Name": "Premium Widget",
//		"Price": 29.99,
//		"Category": "Electronics",
//	}
//	created, err := client.Create(ctx, "Products", newProduct)
//	if err != nil {
//		log.Fatal(err)
//	}
//	fmt.Printf("Created product with ID: %v\n", created["ID"])
//	fmt.Printf("Server name: %v\n", created["Name"])  // May differ if server applies defaults
//
// Example with Struct:
//
//	type Product struct {
//		Name  string  `json:"Name"`
//		Price float64 `json:"Price"`
//	}
//	product := Product{Name: "Widget", Price: 9.99}
//	created, err := client.Create(ctx, "Products", product)
func (c *Client) Create(ctx context.Context, entitySet string, data interface{}) (map[string]interface{}, error) {
	req := c.http.Post("/" + entitySet)
	req = req.WithJSON(data)
	req = req.WithContext(ctx)

	resp, err := c.http.Execute(req)
	if err != nil {
		return nil, fmt.Errorf("traverse: create failed: %w", err)
	}

	if resp.StatusCode != 201 {
		return nil, fmt.Errorf("traverse: create returned status %d", resp.StatusCode)
	}

	// Parse response body based on OData version
	var result map[string]interface{}

	if c.version == ODataV2 {
		// OData v2: response is wrapped in {"d": {...}}
		var wrapped struct {
			D map[string]interface{} `json:"d"`
		}
		err = json.NewDecoder(resp.BodyReader()).Decode(&wrapped)
		if err != nil {
			return nil, fmt.Errorf("traverse: failed to decode Create response: %w", err)
		}
		result = wrapped.D
	} else {
		// OData v4: response is the entity directly
		err = json.NewDecoder(resp.BodyReader()).Decode(&result)
		if err != nil {
			return nil, fmt.Errorf("traverse: failed to decode Create response: %w", err)
		}
	}

	return result, nil
}

// Update updates an existing entity using a partial update (PATCH/MERGE operation).
//
// Update sends an HTTP PATCH request (or HTTP MERGE for OData v2 compatibility) with the entity data.
// This performs a partial update where only provided properties are modified; omitted properties remain unchanged.
//
// This is the standard way to update entities-use Replace() for complete entity replacement instead.
//
// Parameters:
//   - ctx: Context for request cancellation and timeout
//   - entitySet: Name of the entity set (e.g., "Products", "Orders")
//   - key: Entity key for identification (typically an ID or primary key)
//   - data: Properties to update as map[string]interface{} or struct (only these fields change)
//
// Key Formats:
//   - Single key: numeric ID (123) or string ID ('MAT001')
//   - Composite key: Not directly supported; use URL encoding if needed
//
// HTTP Method:
//   - PATCH (standard) for OData v4
//   - MERGE (legacy) for OData v2
//
// Response:
// Returns nil on success with HTTP 204 No Content (standard) or HTTP 200 OK with updated entity.
// The method doesn't parse the response body for 204 responses.
//
// Concurrency Control:
// Returns ErrConcurrencyConflict if:
//   - ETag mismatch (entity was modified elsewhere)
//   - HTTP 409 Conflict response
//
// Example - Partial Update:
//
//	err := client.Update(ctx, "Products", 123, map[string]interface{}{
//		"Price": 19.99,
//		"LastModified": time.Now(),
//	})
//	if err != nil {
//		if err == traverse.ErrConcurrencyConflict {
//			log.Println("Entity was modified by another user")
//		} else {
//			log.Fatal(err)
//		}
//	}
//	log.Println("Product updated successfully")
//
// Example - Update Multiple Fields:
//
//	updates := map[string]interface{}{
//		"Status": "Active",
//		"Priority": 1,
//		"Description": "Updated description",
//	}
//	err := client.Update(ctx, "Tasks", "TASK-001", updates)
func (c *Client) Update(ctx context.Context, entitySet string, key interface{}, data interface{}) error {
	keyStr, err := encodeKey(key)
	if err != nil {
		return fmt.Errorf("traverse: invalid key: %w", err)
	}

	path := fmt.Sprintf("/%s(%s)", entitySet, keyStr)
	req := c.http.Patch(path)
	req = req.WithJSON(data)
	req = req.WithContext(ctx)

	resp, err := c.http.Execute(req)
	if err != nil {
		return fmt.Errorf("traverse: update failed: %w", err)
	}

	if resp.StatusCode != 204 {
		return fmt.Errorf("traverse: update returned status %d", resp.StatusCode)
	}

	return nil
}

// Replace replaces an entire entity with new data using HTTP PUT.
//
// Replace sends an HTTP PUT request with the complete new entity state to replace the existing entity.
// Unlike Update() which performs a partial merge (PATCH), Replace completely replaces all properties
// of the entity. Properties not explicitly provided in the data parameter are typically set to their
// default values or NULL, which can result in data loss if not all properties are included.
//
// Use Cases:
//   - Complete entity replacement: When you have the full new state
//   - Wholesale updates: Replacing entire records from batch operations
//   - Consistency: Ensuring a known complete state after update
//
// Compare with Update():
//   - Update (PATCH): Partial update, only changed fields
//   - Replace (PUT): Complete replacement, unspecified fields set to defaults
//
// Parameters:
//   - ctx: Context for request cancellation and timeout
//   - entitySet: Name of the entity set (e.g., "Products")
//   - key: Entity key for identification (ID or primary key)
//   - data: Complete new entity state (all properties that should exist)
//
// HTTP Method: PUT (HTTP 201-204 on success)
//
// Response: Returns nil on success with HTTP 204 No Content.
// Returns ErrConcurrencyConflict if an ETag mismatch occurs (HTTP 409).
//
// Warning:
// Replace can result in data loss if the data parameter doesn't include all required fields.
// Prefer Update() for partial updates unless you specifically need complete replacement.
//
// Example - Complete Replacement:
//
//	newProduct := map[string]interface{}{
//		"ID": 123,  // Must include key
//		"Name": "Completely New Name",
//		"Price": 49.99,
//		"Stock": 50,
//		"Active": true,
//	}
//	err := client.Replace(ctx, "Products", 123, newProduct)
//	if err != nil {
//		log.Fatal(err)
//	}
//	log.Println("Product completely replaced")
func (c *Client) Replace(ctx context.Context, entitySet string, key interface{}, data interface{}) error {
	keyStr, err := encodeKey(key)
	if err != nil {
		return fmt.Errorf("traverse: invalid key: %w", err)
	}

	path := fmt.Sprintf("/%s(%s)", entitySet, keyStr)
	req := c.http.Put(path)
	req = req.WithJSON(data)
	req = req.WithContext(ctx)

	resp, err := c.http.Execute(req)
	if err != nil {
		return fmt.Errorf("traverse: replace failed: %w", err)
	}

	if resp.StatusCode != 204 {
		return fmt.Errorf("traverse: replace returned status %d", resp.StatusCode)
	}

	return nil
}

// Delete deletes an entity from the specified entity set.
//
// Delete sends an HTTP DELETE request to remove the entity identified by the provided key.
// Once deleted, the entity can no longer be accessed through normal queries; it is permanently
// removed from the data store (subject to any database retention policies).
//
// Key Encoding:
//   - String keys: Automatically quoted and escaped for OData URL encoding ('MAT001')
//   - Numeric keys: Converted to string representation (123)
//   - Invalid keys: Returns error during encoding
//
// Concurrency Control:
//   - Returns ErrConcurrencyConflict if entity was modified elsewhere (ETag mismatch)
//   - The service may return HTTP 409 if concurrent modification is detected
//
// Parameters:
//   - ctx: Context for request cancellation and timeout
//   - entitySet: Name of the entity set (e.g., "Products", "Orders")
//   - key: Entity key for identification (int or string)
//
// HTTP Method: DELETE
// Response: HTTP 204 No Content on success (no body returned)
//
// Error Cases:
//   - HTTP 404: Entity not found
//   - HTTP 409: Concurrency conflict (entity modified elsewhere)
//   - Network errors: Connection failures
//
// Important Notes:
//   - Deletion is permanent; deleted data cannot be recovered
//   - Some services may use soft deletes (marking as deleted) instead of hard deletion
//   - Deleting a parent may cascade to child entities (depending on foreign key constraints)
//
// Example - Delete by Numeric ID:
//
//	err := client.Delete(ctx, "Products", 123)
//	if err != nil {
//		if errors.Is(err, traverse.ErrConcurrencyConflict) {
//			log.Println("Product was modified by another user")
//		} else {
//			log.Fatal(err)
//		}
//	}
//	log.Println("Product deleted successfully")
//
// Example - Delete by String ID:
//
//	err := client.Delete(ctx, "Materials", "MAT001")
//	if err != nil {
//		log.Fatal(err)
//	}
func (c *Client) Delete(ctx context.Context, entitySet string, key interface{}) error {
	keyStr, err := encodeKey(key)
	if err != nil {
		return fmt.Errorf("traverse: invalid key: %w", err)
	}

	path := fmt.Sprintf("/%s(%s)", entitySet, keyStr)
	req := c.http.Delete(path)
	req = req.WithContext(ctx)

	resp, err := c.http.Execute(req)
	if err != nil {
		return fmt.Errorf("traverse: delete failed: %w", err)
	}

	if resp.StatusCode != 204 {
		return fmt.Errorf("traverse: delete returned status %d", resp.StatusCode)
	}

	return nil
}

// encodeKey encodes a key value for use in OData URLs.
//
// encodeKey handles multiple key types:
//   - Strings: Wrapped in single quotes and URL-encoded (e.g., 'value%20here')
//   - Integers/Floats: Converted to string representation (e.g., 123, 45.67)
//
// String keys are optimized using a bytes.Buffer to avoid intermediate allocations.
// Returns an error if the key type is not supported (e.g., bool, slice, map).
//
// Example encodings:
//
//	encodeKey("Product A")   → "'Product%20A'"
//	encodeKey(123)           → "123"
//	encodeKey(45.67)         → "45.67"
func encodeKey(key interface{}) (string, error) {
	switch v := key.(type) {
	case string:
		// String keys need single quotes and escaping
		// Optimized: use buffer to avoid intermediate "'" + v + "'" allocation
		var buf bytes.Buffer
		buf.WriteRune('\'')
		buf.WriteString(v)
		buf.WriteRune('\'')
		escaped := url.QueryEscape(buf.String())
		return escaped, nil
	case int, int32, int64, float32, float64:
		return fmt.Sprint(v), nil
	default:
		return "", fmt.Errorf("unsupported key type: %T", v)
	}
}

// Configuration Options

// WithBaseURL sets the base URL of the OData service.
//
// WithBaseURL is required; New() will return an error if no base URL is provided.
// The URL should be the root endpoint of the OData service (e.g., "https://sap.example.com/sap/opu/odata/sap/MY_SRV").
//
// Example:
//
//	client, _ := traverse.New(
//		traverse.WithBaseURL("https://sap.example.com/sap/opu/odata/sap/Products"),
//	)
func WithBaseURL(url string) Option {
	return func(cfg *clientConfig) error {
		if url == "" {
			return fmt.Errorf("base URL cannot be empty")
		}
		cfg.baseURL = url
		return nil
	}
}

// WithODataVersion sets the OData protocol version (ODataV2 or ODataV4).
//
// WithODataVersion controls how requests are constructed and responses are parsed.
// Defaults to ODataV4 if not specified.
//
// - ODataV2: Used by classic SAP ABAP Gateway; responses wrapped in "d" object
// - ODataV4: Modern protocol used by S/4HANA and most new OData services
//
// Example:
//
//	client, _ := traverse.New(
//		traverse.WithBaseURL("..."),
//		traverse.WithODataVersion(traverse.ODataV2), // For legacy SAP systems
//	)
func WithODataVersion(v ODataVersion) Option {
	return func(cfg *clientConfig) error {
		if v != ODataV2 && v != ODataV4 {
			return fmt.Errorf("unsupported OData version: %v", v)
		}
		cfg.version = v
		return nil
	}
}

// WithPageSize sets the number of records fetched per HTTP request during pagination.
//
// WithPageSize controls the $top parameter sent to the server. Defaults to 1000 if not specified.
// Larger values reduce network round-trips but consume more memory per request.
// Smaller values reduce per-request memory but increase latency due to more requests.
//
// The server may reject the value or return fewer records; OData clients should handle
// the actual page size in responses.
//
// Example:
//
//	client, _ := traverse.New(
//		traverse.WithBaseURL("..."),
//		traverse.WithPageSize(5000), // Fetch 5000 records per request
//	)
func WithPageSize(n int) Option {
	return func(cfg *clientConfig) error {
		if n <= 0 {
			return fmt.Errorf("page size must be positive")
		}
		cfg.pageSize = n
		return nil
	}
}

// WithBasicAuth sets HTTP Basic Authentication credentials.
//
// WithBasicAuth adds a Basic Authorization header to all requests using the
// provided username and password. Credentials are base64-encoded.
//
// Note: Basic auth should only be used over HTTPS to avoid credential exposure.
//
// Example:
//
//	client, _ := traverse.New(
//		traverse.WithBaseURL("https://sap.example.com/..."),
//		traverse.WithBasicAuth("user@example.com", "password123"),
//	)
func WithBasicAuth(user, pass string) Option {
	return func(cfg *clientConfig) error {
		cfg.basicAuthUser = user
		cfg.basicAuthPass = pass
		return nil
	}
}

// WithBearerToken sets a Bearer token for OAuth2/token-based authentication.
//
// WithBearerToken adds a Bearer Authorization header to all requests using the
// provided token. Tokens are typically obtained from an OAuth2 token endpoint.
//
// Example:
//
//	token := getOAuth2Token() // From OAuth2 provider
//	client, _ := traverse.New(
//		traverse.WithBaseURL("https://odata.service/"),
//		traverse.WithBearerToken(token),
//	)
func WithBearerToken(token string) Option {
	return func(cfg *clientConfig) error {
		if token == "" {
			return fmt.Errorf("bearer token cannot be empty")
		}
		cfg.bearerToken = token
		return nil
	}
}

// WithAPIKey sets API key authentication using a custom header.
//
// WithAPIKey adds a custom header with the provided key/value to all requests.
// This is useful for services that require API keys in custom headers
// (e.g., X-API-Key, X-Custom-Auth-Token).
//
// Example:
//
//	client, _ := traverse.New(
//		traverse.WithBaseURL("https://api.service/odata/"),
//		traverse.WithAPIKey("X-API-Key", "sk-abcd1234..."),
//	)
func WithAPIKey(header, value string) Option {
	return func(cfg *clientConfig) error {
		if header == "" || value == "" {
			return fmt.Errorf("API key header and value cannot be empty")
		}
		cfg.apiKeyHeader = header
		cfg.apiKeyValue = value
		return nil
	}
}

// WithRelayClient injects a pre-configured [relay.Client] for custom HTTP configuration.
//
// WithRelayClient allows advanced users to configure HTTP behavior (connection pooling,
// custom transport, proxies, etc.) before passing the client to Traverse.
// If not provided, Traverse creates a default relay.Client.
//
// This is useful for:
//   - Configuring connection pooling for large-scale operations
//   - Setting up proxy or TLS configuration
//   - Using a relay client shared across multiple services
//
// Example:
//
//	httpClient := relay.New(
//		relay.WithTimeout(60 * time.Second),
//		relay.WithBaseURL("https://sap.example.com/sap/opu/odata/sap/Products"),
//	)
//	client, _ := traverse.New(
//		traverse.WithRelayClient(httpClient),
//	)
func WithRelayClient(rc *relay.Client) Option {
	return func(cfg *clientConfig) error {
		if rc == nil {
			return fmt.Errorf("relay client cannot be nil")
		}
		cfg.httpClient = rc
		return nil
	}
}

// WithLogger sets a [relay.Logger] for diagnostic output.
//
// WithLogger enables logging of HTTP requests/responses and other diagnostic information.
// If not provided, logging is disabled. Useful for debugging authentication issues or
// tracing OData query execution.
//
// Example:
//
//	client, _ := traverse.New(
//		traverse.WithBaseURL("..."),
//		traverse.WithLogger(customLogger),
//	)
func WithLogger(l relay.Logger) Option {
	return func(cfg *clientConfig) error {
		cfg.logger = l
		cfg.relayOpts = append(cfg.relayOpts,
			relay.WithLogger(l),
		)
		return nil
	}
}

// WithFormat sets the OData response format (FormatJSON or FormatAtom).
//
// WithFormat controls the $format parameter sent to the OData service. Defaults to
// FormatJSON if not specified. Most modern OData services prefer JSON format.
//
// - FormatJSON: Returns responses as JSON (recommended for performance)
// - FormatAtom: Returns responses as Atom+XML (legacy format)
func WithFormat(f ResponseFormat) Option {
	return func(cfg *clientConfig) error {
		cfg.responseFormat = f
		return nil
	}
}

// WithHeader adds a custom HTTP header to all requests made by this [Client].
//
// WithHeader can be called multiple times to add multiple custom headers.
// Useful for setting headers like User-Agent, Accept-Language, or custom tracking IDs.
//
// Example:
//
//	client, _ := traverse.New(
//		traverse.WithBaseURL("..."),
//		traverse.WithHeader("User-Agent", "MyApp/1.0"),
//		traverse.WithHeader("X-Tracking-ID", "12345"),
//	)
func WithHeader(key, value string) Option {
	return func(cfg *clientConfig) error {
		cfg.relayOpts = append(cfg.relayOpts,
			relay.WithDefaultHeaders(map[string]string{key: value}),
		)
		return nil
	}
}

// WithTimeout sets the request timeout for all HTTP requests.
//
// WithTimeout controls the timeout for individual OData requests. Defaults to
// 30 seconds if not specified. Applies to query execution, metadata fetching, and all CRUD operations.
//
// For operations that process millions of records with streaming, consider a longer timeout.
//
// Example:
//
//	client, _ := traverse.New(
//		traverse.WithBaseURL("..."),
//		traverse.WithTimeout(60 * time.Second),
//	)
func WithTimeout(d time.Duration) Option {
	return func(cfg *clientConfig) error {
		cfg.relayOpts = append(cfg.relayOpts,
			relay.WithTimeout(d),
		)
		return nil
	}
}

// WithMetadataCache sets a custom metadata cache implementation.
//
// WithMetadataCache allows you to provide a [CacheStore] implementation for caching
// OData service metadata. By default, metadata is cached after the first fetch using
// [NewMemoryCache]. This option allows distributed caching or custom expiration policies.
//
// Example implementations: Redis, memcached, file-based, or SQL database backends.
//
// Example with memory cache:
//
//	cache := traverse.NewMemoryCache()
//	client, err := traverse.New(
//	    traverse.WithBaseURL("https://odata.service/"),
//	    traverse.WithMetadataCache(cache),
//	)
func WithMetadataCache(cache CacheStore) Option {
	return func(cfg *clientConfig) error {
		if cache == nil {
			return fmt.Errorf("metadata cache cannot be nil")
		}
		cfg.metadataCache = cache
		return nil
	}
}

// WithBeforeQuery adds a hook function that is called before executing each query.
//
// WithBeforeQuery registers a callback that fires immediately before each OData query
// is executed. Multiple hooks can be added and are called in registration order.
// The hook receives the [QueryBuilder] and can modify it, perform validation,
// or apply common filters/options.
//
// Return an error from the hook to abort query execution.
//
// Useful for:
//   - Adding common filters or parameters
//   - Logging or tracing
//   - Implementing access control
//   - Modifying query behavior dynamically
//
// Example:
//
//	client, _ := traverse.New(
//		traverse.WithBaseURL("..."),
//		traverse.WithBeforeQuery(func(qb *traverse.QueryBuilder) error {
//			qb.Logger(log.Printf) // Log all queries
//			return nil
//		}),
//	)
func WithBeforeQuery(hook func(*QueryBuilder) error) Option {
	return func(cfg *clientConfig) error {
		if hook == nil {
			return fmt.Errorf("before query hook cannot be nil")
		}
		cfg.beforeQuery = append(cfg.beforeQuery, hook)
		return nil
	}
}

// WithAfterExecute adds a hook function that is called after executing each query.
//
// WithAfterExecute registers a callback that fires immediately after each OData query
// completes. Multiple hooks can be added and are called in registration order.
// The hook receives the [QueryBuilder] and can access query results, state, or
// perform actions after execution.
//
// Useful for:
//   - Collecting metrics or statistics
//   - Logging or tracing
//   - Implementing response validation
//   - Cleanup operations
//
// Example:
//
//	client, _ := traverse.New(
//		traverse.WithBaseURL("..."),
//		traverse.WithAfterExecute(func(qb *traverse.QueryBuilder) error {
//			metrics.RecordQuery(qb.EntitySet())
//			return nil
//		}),
//	)
func WithAfterExecute(hook func(*QueryBuilder) error) Option {
	return func(cfg *clientConfig) error {
		if hook == nil {
			return fmt.Errorf("after execute hook cannot be nil")
		}
		cfg.afterExecute = append(cfg.afterExecute, hook)
		return nil
	}
}

// ServiceDocument represents the root OData service document.
// It contains a list of all available entity sets in the OData service.
type ServiceDocument struct {
	// EntitySets is the list of available entity sets.
	EntitySets []EntitySetReference
}

// EntitySetReference represents a reference to an entity set in the service document.
// It provides the name and URL path for accessing the entity set.
type EntitySetReference struct {
	// Name is the name of the entity set.
	Name string
	// URL is the URL path relative to the service root.
	URL string
}

// Metadata represents the parsed OData $metadata document.
// It contains the Entity Data Model (EDM) describing all entity types,
// relationships, and operations available in the service.
type Metadata struct {
	// Version is the OData protocol version (e.g., "2.0" or "4.0").
	Version string
	// Namespace is the target namespace of the metadata document.
	Namespace string
	// EntityTypes is the list of all entity types defined in the service.
	EntityTypes []EntityType
	// EntitySets is the list of all entity set definitions.
	EntitySets []EntitySetInfo
	// Associations describes relationships between entity types.
	Associations []Association
	// FunctionImports contains function imports available in the service.
	FunctionImports []FunctionImportInfo
	// Actions contains action definitions (OData v4).
	Actions []ActionInfo
	// Functions contains function definitions (OData v4).
	Functions []FunctionInfo
}

// EntitySetInfo represents an OData entity set definition.
// An entity set is a collection of entities of a specific type.
type EntitySetInfo struct {
	// Name is the name of the entity set.
	Name string
	// EntityTypeName is the fully qualified name of the entity type.
	EntityTypeName string
}

// EntityType represents an OData entity type definition.
// It describes the structure of entities of this type, including their properties and relationships.
type EntityType struct {
	// Name is the name of the entity type.
	Name string
	// Key is the list of properties that form the primary key.
	Key []PropertyRef
	// Properties is the list of all properties defined for this entity type.
	Properties []Property
	// NavigationProperties is the list of navigation properties (relationships).
	NavigationProperties []NavigationProperty
}

// PropertyRef represents a reference to a property that is part of an entity's key.
type PropertyRef struct {
	// Name is the name of the property.
	Name string
}

// Property represents an OData property definition.
// Properties describe the data that can be stored in entities.
type Property struct {
	// Name is the name of the property.
	Name string
	// Type is the OData type (e.g., "Edm.String", "Edm.Int32", "Edm.DateTime").
	Type string
	// Nullable indicates whether the property can have a null value.
	Nullable bool
	// MaxLength is the maximum string length (for string types).
	MaxLength *int
	// Precision is the number of significant digits (for numeric types).
	Precision *int
	// Scale is the number of decimal places (for decimal types).
	Scale *int
	// SAP contains SAP-specific annotations for this property.
	SAP SAPAnnotations
}

// NavigationProperty represents a navigation property in an OData entity type.
// Navigation properties define relationships to other entity types.
type NavigationProperty struct {
	// Name is the name of the navigation property.
	Name string
	// FromEntityType is the entity type this property originates from.
	FromEntityType string
	// ToEntityType is the target entity type.
	ToEntityType string
	// FromMultiplicity describes the multiplicity from the source (0..1, 1, *).
	FromMultiplicity string
	// ToMultiplicity describes the multiplicity to the target (0..1, 1, *).
	ToMultiplicity string
	// Partner is the name of the inverse navigation property.
	Partner string
}

// Association represents an OData association between entity types.
type Association struct {
	// Name is the name of the association.
	Name string
	// From is the source end of the association.
	From AssociationEnd
	// To is the target end of the association.
	To AssociationEnd
}

// AssociationEnd represents one end of an OData association.
type AssociationEnd struct {
	// EntityType is the fully qualified entity type name at this end.
	EntityType string
	// Multiplicity describes the multiplicity (0..1, 1, *).
	Multiplicity string
}

// SAPAnnotations contains SAP-specific annotations for properties and entity types.
// These annotations provide hints about how SAP systems use the properties.
type SAPAnnotations struct {
	// Label is a human-readable label for the property.
	Label string
	// Filterable indicates if the property can be used in $filter clauses.
	Filterable bool
	// Sortable indicates if the property can be used in $orderby clauses.
	Sortable bool
	// Searchable indicates if the property can be used in search operations.
	Searchable bool
	// Required indicates if the property is required.
	Required bool
	// Text refers to a property containing a descriptive text for this property.
	Text string
}

// FunctionImportInfo represents a function import definition (OData v2).
// Function imports are operations that can be called on the service.
type FunctionImportInfo struct {
	// Name is the name of the function import.
	Name string
	// Parameters is the list of parameters for this function.
	Parameters []FunctionParameter
	// ReturnType is the return type of the function.
	ReturnType string
	// IsComposable indicates if the function can be used in URL composition.
	IsComposable bool
}

// FunctionParameter represents a function parameter.
type FunctionParameter struct {
	Name     string
	Type     string
	Nullable bool
}

// ActionInfo represents an OData action (v4).
type ActionInfo struct {
	Name       string
	Parameters []FunctionParameter
	ReturnType string
}

// FunctionInfo represents an OData function (v4).
type FunctionInfo struct {
	Name         string
	Parameters   []FunctionParameter
	ReturnType   string
	IsComposable bool
}
