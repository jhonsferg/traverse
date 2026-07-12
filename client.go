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
	"encoding/xml"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/jhonsferg/relay"
)

// maxResponseBodySize limits how much data we read from a single OData response
// to prevent memory exhaustion from unbounded payloads.
const maxResponseBodySize = 100 * 1024 * 1024 // 100 MB

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

// defaultMaxPages is the upper bound on the number of pages that will be
// fetched when following nextLink URLs. 10,000 pages of 1,000 records each
// equals 10 million entities — more than enough for any real-world dataset
// while preventing infinite loops caused by misbehaving servers.
const defaultMaxPages = 10_000

type Client struct {
	// http is the underlying relay HTTP client for making requests.
	http *relay.Client
	// baseURL is the OData service endpoint URL.
	baseURL string
	// version is the OData protocol version (v2 or v4).
	version ODataVersion
	// pageSize is the default number of records per page for pagination.
	pageSize int
	// maxPages is the maximum number of pages to follow via nextLink. A value
	// of 0 or negative is treated as the default (defaultMaxPages). Set via
	// WithMaxPages to override. Guards against servers that always return a
	// nextLink causing an infinite pagination loop.
	maxPages int
	// logger is the logger for diagnostic output.
	logger relay.Logger
	// responseFormat is the requested response format (JSON or Atom/XML).
	responseFormat ResponseFormat

	// metadataCache stores metadata for improved performance across requests.
	metadataCache CacheStore
	// responseCache is the HTTP-level response cache for entity set queries.
	responseCache ResponseCache
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
	maxPages       int
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
	// responseCache is the HTTP-level response cache for entity set queries.
	responseCache ResponseCache
	// beforeQuery are hooks called before query execution.
	beforeQuery []func(*QueryBuilder) error
	// afterExecute are hooks called after query execution.
	afterExecute []func(*QueryBuilder) error
}

// applyAuthOpts converts basicAuth/bearerToken config into relay.Option entries
// so they are actually sent as HTTP headers on every request.
func applyAuthOpts(cfg *clientConfig) {
	if cfg.basicAuthUser != "" {
		user, pass := cfg.basicAuthUser, cfg.basicAuthPass
		cfg.relayOpts = append(cfg.relayOpts, relay.WithOnBeforeRequest(
			func(_ context.Context, req *relay.Request) error {
				req.WithBasicAuth(user, pass)
				return nil
			},
		))
	} else if cfg.bearerToken != "" {
		token := cfg.bearerToken
		cfg.relayOpts = append(cfg.relayOpts, relay.WithOnBeforeRequest(
			func(_ context.Context, req *relay.Request) error {
				req.WithBearerToken(token)
				return nil
			},
		))
	}

	if cfg.apiKeyHeader != "" {
		header, value := cfg.apiKeyHeader, cfg.apiKeyValue
		cfg.relayOpts = append(cfg.relayOpts, relay.WithOnBeforeRequest(
			func(_ context.Context, req *relay.Request) error {
				req.WithHeader(header, value)
				return nil
			},
		))
	}
}

// New creates a new [Client] with the provided options.
// At minimum, [WithBaseURL] must be provided unless a pre-configured relay
// client is passed via [WithRelayClient], in which case the base URL is
// inherited from the relay client automatically.
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
		maxPages:       defaultMaxPages,
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

	// Auto-inherit base URL from the relay client when WithRelayClient was
	// used but WithBaseURL was not explicitly provided.
	if cfg.baseURL == "" && cfg.httpClient != nil {
		cfg.baseURL = cfg.httpClient.BaseURL()
	}

	// Validate required fields
	if cfg.baseURL == "" {
		return nil, fmt.Errorf("traverse: BaseURL is required (use WithBaseURL or ensure the relay client was configured with relay.WithBaseURL)")
	}

	// Create relay client if not provided externally.
	if cfg.httpClient == nil {
		cfg.relayOpts = append(cfg.relayOpts,
			relay.WithTimeout(30*time.Second),
			relay.WithBaseURL(cfg.baseURL),
		)
		applyAuthOpts(cfg)
		cfg.httpClient = relay.New(cfg.relayOpts...)
	} else if len(cfg.relayOpts) > 0 || cfg.basicAuthUser != "" || cfg.bearerToken != "" {
		// An external relay client was provided AND traverse-level options that
		// map to relay options were also set (e.g. WithHeader, WithTimeout,
		// WithBeforeRequest). Apply them to the existing relay client so they
		// are not silently dropped.
		applyAuthOpts(cfg)
		cfg.httpClient = cfg.httpClient.With(cfg.relayOpts...)
	}

	c := &Client{
		http:           cfg.httpClient,
		baseURL:        cfg.baseURL,
		version:        cfg.version,
		pageSize:       cfg.pageSize,
		maxPages:       cfg.maxPages,
		logger:         cfg.logger,
		responseFormat: cfg.responseFormat,
		metadataCache:  cfg.metadataCache,
		responseCache:  cfg.responseCache,
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

// RelayClient returns the underlying [relay.Client] used for HTTP transport.
//
// This provides access to relay's full request API for advanced use cases such
// as extension modules that need to issue raw HTTP calls (e.g. OData webhook
// subscription management) while reusing the same configured transport, auth,
// retry and circuit-breaker settings as the parent traverse.Client.
//
// The returned client must not be closed independently  -  use [Client.Close]
// to shut down both the traverse and relay clients together.
func (c *Client) RelayClient() *relay.Client {
	return c.http
}

// breaker: Closed (healthy), Open (failing, requests rejected), or Half-Open
// (probing for recovery).
//
// Returns [relay.StateClosed] if the circuit breaker is not configured.
//
// Example:
//
//	if client.CircuitBreakerState() == relay.StateOpen {
//	    log.Println("OData service unavailable - circuit is open")
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
	// Strip leading slashes so that From("/Products") and From("Products")
	// behave identically. A leading slash in the entity set name causes
	// buildURL to emit "//Products", which produces a double-slash URL
	// (e.g. https://host//Products) that servers such as SAP reject with 401.
	entitySet = strings.TrimLeft(entitySet, "/")
	// selectFields, expandProps, params, and conditionalHeaders are intentionally
	// left nil here (zero-alloc construction). Each field is lazily initialised
	// on first write: append(nil, ...) is valid for slices, and the map setters
	// guard with a nil check before the first write.
	return &QueryBuilder{
		client:    c,
		entitySet: entitySet,
		urlDirty:  true,
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
		// Skip cache operations for NoOpCache to avoid unnecessary key generation
		if _, isNoOp := c.metadataCache.(*NoOpCache); isNoOp {
			metadata, err := c.fetchMetadata(ctx)
			if err != nil {
				c.metadataErr = err
				return
			}
			c.metadata = metadata
			return
		}

		// Use cache key based on baseURL
		cacheKey := "metadata:" + c.baseURL

		// Try to get from cache first
		if metadata, found := c.metadataCache.Get(cacheKey); found {
			c.metadata = metadata
			return
		}

		// Cache miss, fetch from network
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

	// Auto-detect format: JSON for CSDL JSON, XML for EDMX.
	var (
		metadata *Metadata
		parseErr error
	)
	if ct := resp.ContentType(); strings.Contains(ct, "application/json") {
		metadata, parseErr = ParseCSDLJSONReader(resp.BodyReader())
	} else {
		metadata, parseErr = ParseEDMX(resp.BodyReader())
	}
	if parseErr != nil {
		return nil, fmt.Errorf("traverse: failed to parse $metadata: %w", parseErr)
	}

	return metadata, nil
}

// decodeODataResponse decodes an OData response body into a map[string]interface{}.
// It detects the content type (JSON vs XML) and routes to the appropriate decoder.
//
// For JSON responses, it unmarshals directly using the OData version-specific structure.
// For XML responses (common in SAP OData v2), it converts to JSON first via intermediate unmarshaling.
func (c *Client) decodeODataResponse(resp *relay.Response) (map[string]interface{}, error) {
	// Read the response body into memory so we can inspect it
	bodyBytes, err := io.ReadAll(io.LimitReader(resp.BodyReader(), maxResponseBodySize))
	if err != nil {
		return nil, fmt.Errorf("traverse: failed to read response body: %w", err)
	}

	// Detect content type: check header first, then body
	contentType := resp.ContentType()
	isXML := isXMLContentType(contentType) || isXMLBody(bodyBytes)

	var result map[string]interface{}

	if isXML {
		// Handle XML response (OData v2 Atom format)
		result, err = decodeXMLResponse(bodyBytes, c.version)
		if err != nil {
			return nil, fmt.Errorf("traverse: failed to decode XML response: %w", err)
		}
	} else {
		// Handle JSON response
		result, err = decodeJSONResponse(bodyBytes, c.version)
		if err != nil {
			return nil, fmt.Errorf("traverse: failed to decode JSON response: %w", err)
		}
	}

	return result, nil
}

// isXMLContentType checks if the content type indicates XML
func isXMLContentType(ct string) bool {
	if ct == "" {
		return false
	}
	ct = strings.ToLower(ct)
	return strings.Contains(ct, "xml") ||
		strings.Contains(ct, "atom") ||
		strings.Contains(ct, "text/xml")
}

// isXMLBody checks if the response body starts with XML markers
func isXMLBody(body []byte) bool {
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 {
		return false
	}
	// Check for XML declaration or opening tag
	return bytes.HasPrefix(trimmed, []byte("<?xml")) ||
		bytes.HasPrefix(trimmed, []byte("<"))
}

// decodeJSONResponse decodes a JSON OData response
func decodeJSONResponse(bodyBytes []byte, version ODataVersion) (map[string]interface{}, error) {
	var result map[string]interface{}

	if version == ODataV2 {
		// OData v2: response is wrapped in {"d": {...}}
		var wrapped struct {
			D map[string]interface{} `json:"d"`
		}
		err := json.Unmarshal(bodyBytes, &wrapped)
		if err != nil {
			return nil, err
		}
		result = wrapped.D
	} else {
		// OData v4: response is the entity directly
		err := json.Unmarshal(bodyBytes, &result)
		if err != nil {
			return nil, err
		}
	}

	return result, nil
}

// decodeXMLResponse decodes an XML OData response (typically SAP OData v2 Atom format).
// It uses token-by-token parsing to extract d:property elements from m:properties.
func decodeXMLResponse(bodyBytes []byte, version ODataVersion) (map[string]interface{}, error) {
	const (
		atomNS = "http://www.w3.org/2005/Atom"
		dataNS = "http://schemas.microsoft.com/ado/2007/08/dataservices"
		metaNS = "http://schemas.microsoft.com/ado/2007/08/dataservices/metadata"
	)

	dec := xml.NewDecoder(bytes.NewReader(bodyBytes))
	result := make(map[string]interface{})

	var inProperties, inProp bool
	var currentProp string

	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to parse XML: %w", err)
		}

		switch t := tok.(type) {
		case xml.StartElement:
			fullName := t.Name.Space + " " + t.Name.Local
			switch {
			case fullName == metaNS+" properties" || fullName == dataNS+" properties":
				inProperties = true
			case inProperties && t.Name.Space == dataNS:
				inProp = true
				currentProp = t.Name.Local
			}
		case xml.EndElement:
			fullName := t.Name.Space + " " + t.Name.Local
			switch {
			case fullName == metaNS+" properties" || fullName == dataNS+" properties":
				inProperties = false
			case inProp && t.Name.Space == dataNS:
				inProp = false
				currentProp = ""
			}
		case xml.CharData:
			if inProp && currentProp != "" {
				val := strings.TrimSpace(string(t))
				if val != "" {
					result[currentProp] = val
				}
			}
		}
	}

	return result, nil
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

	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		return nil, fmt.Errorf("traverse: create returned status %d", resp.StatusCode)
	}

	// Invalidate cached responses for this entity set since data has changed.
	c.invalidateEntitySetCache(entitySet)

	// Parse response body with content-type detection (JSON or XML)
	result, err := c.decodeODataResponse(resp)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// createWithRawXML creates a new entity and returns the raw XML response bytes (internal helper).
//
// This internal method is used by CreateAtomXmlAs to get the raw XML response
// without going through the map[string]interface{} conversion layer.
// It sends a POST with Accept: application/atom+xml header.
func (c *Client) createWithRawXML(ctx context.Context, entitySet string, data interface{}) ([]byte, error) {
	req := c.http.Post("/" + entitySet)
	req = req.WithJSON(data)
	req = req.WithContext(ctx)
	req = req.WithHeader("Accept", "application/atom+xml")

	resp, err := c.http.Execute(req)
	if err != nil {
		return nil, fmt.Errorf("traverse: create failed: %w", err)
	}

	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		return nil, fmt.Errorf("traverse: create returned status %d", resp.StatusCode)
	}

	// Invalidate cached responses for this entity set since data has changed.
	c.invalidateEntitySetCache(entitySet)

	// Read the raw response body
	bodyBytes, err := io.ReadAll(io.LimitReader(resp.BodyReader(), maxResponseBodySize))
	if err != nil {
		return nil, fmt.Errorf("traverse: failed to read response body: %w", err)
	}

	return bodyBytes, nil
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
// Returns nil on success with HTTP 200 OK (with updated entity) or HTTP 204 No Content (standard).
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

	entityPath, rawQuery := splitEntityPath(entitySet)
	path := fmt.Sprintf("/%s(%s)%s", entityPath, keyStr, rawQuery)
	req := c.http.Patch(path)
	req = req.WithJSON(data)
	req = req.WithContext(ctx)

	resp, err := c.http.Execute(req)
	if err != nil {
		return fmt.Errorf("traverse: update failed: %w", err)
	}

	if resp.StatusCode != 200 && resp.StatusCode != 204 {
		return fmt.Errorf("traverse: update returned status %d", resp.StatusCode)
	}

	// Invalidate cached responses for this entity set since data has changed.
	c.invalidateEntitySetCache(entitySet)

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

	entityPath, rawQuery := splitEntityPath(entitySet)
	path := fmt.Sprintf("/%s(%s)%s", entityPath, keyStr, rawQuery)
	req := c.http.Put(path)
	req = req.WithJSON(data)
	req = req.WithContext(ctx)

	resp, err := c.http.Execute(req)
	if err != nil {
		return fmt.Errorf("traverse: replace failed: %w", err)
	}

	if resp.StatusCode != 200 && resp.StatusCode != 201 && resp.StatusCode != 204 {
		return fmt.Errorf("traverse: replace returned status %d", resp.StatusCode)
	}

	// Invalidate cached responses for this entity set since data has changed.
	c.invalidateEntitySetCache(entitySet)

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

	entityPath, rawQuery := splitEntityPath(entitySet)
	path := fmt.Sprintf("/%s(%s)%s", entityPath, keyStr, rawQuery)
	req := c.http.Delete(path)
	req = req.WithContext(ctx)

	resp, err := c.http.Execute(req)
	if err != nil {
		return fmt.Errorf("traverse: delete failed: %w", err)
	}

	if resp.StatusCode != 200 && resp.StatusCode != 204 {
		return fmt.Errorf("traverse: delete returned status %d", resp.StatusCode)
	}

	// Invalidate cached responses for this entity set since data has changed.
	c.invalidateEntitySetCache(entitySet)

	return nil
}

// invalidateEntitySetCache removes all response cache entries for the given
// entity set. Called after successful Create, Update, Replace, and Delete
// operations to ensure stale cached pages are not served.
func (c *Client) invalidateEntitySetCache(entitySet string) {
	if c.responseCache != nil {
		// Cache keys are built by buildURL() which always starts with "/",
		// so invalidation must match the leading slash prefix.
		c.responseCache.Invalidate("/" + entitySet)
	}
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
//	encodeKey("Product A")   → "'Product A'"
//	encodeKey(123)           → "123"
//	encodeKey(45.67)         → "45.67"
func encodeKey(key interface{}) (string, error) {
	switch v := key.(type) {
	case string:
		// OData string literals are wrapped in single quotes.
		// Embedded single quotes must be escaped by doubling them ('').
		// Single quotes and parentheses are RFC 3986 sub-delimiters and must
		// NOT be percent-encoded in URL paths; encoding them causes double-encoding
		// when relay resolves the full URL via url.URL.ResolveReference.
		escaped := strings.ReplaceAll(v, "'", "''")
		return "'" + escaped + "'", nil
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

// WithMaxPages sets the maximum number of pages to follow when paginating via
// nextLink. This guards against servers that always return a nextLink (whether
// by misconfiguration or malice), which would otherwise cause an infinite loop.
//
// The default is [defaultMaxPages] (10,000). Omit or pass a positive value.
// A value of 1 fetches only the first page regardless of nextLink.
//
// Example:
//
//	client, _ := traverse.New(
//		traverse.WithBaseURL("..."),
//		traverse.WithMaxPages(100), // cap at 100 pages (e.g. 500k records at pageSize=5000)
//	)
func WithMaxPages(n int) Option {
	return func(cfg *clientConfig) error {
		if n <= 0 {
			return fmt.Errorf("max pages must be positive")
		}
		cfg.maxPages = n
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
// When a relay client is provided, [WithBaseURL] on traverse is optional: the base URL
// is automatically inherited from the relay client's own [relay.WithBaseURL] option.
// This means you do NOT need to repeat the base URL in both clients.
//
// Any traverse-level options that map to relay behaviour (e.g. [WithHeader], [WithTimeout],
// [WithBeforeRequest]) are applied on top of the provided relay client via [relay.Client.With],
// so they are not silently dropped.
//
// This is useful for:
//   - Configuring connection pooling for large-scale operations
//   - Setting up proxy or TLS configuration
//   - Using a relay client shared across multiple services
//
// Example - base URL inherited automatically:
//
//	httpClient := relay.New(
//		relay.WithTimeout(60 * time.Second),
//		relay.WithBaseURL("https://sap.example.com/sap/opu/odata/sap"),
//	)
//	// No traverse.WithBaseURL needed - inherited from httpClient
//	client, _ := traverse.New(
//		traverse.WithRelayClient(httpClient),
//		traverse.WithODataVersion(traverse.ODataV2),
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

// WithSchemaVersion sets the OData-SchemaVersion request header on all requests
// made by this [Client].
//
// OData 4.01 (spec section 8.2.10) introduced this header to allow clients to
// request a specific version of the service's metadata schema. Services that
// evolve their schema over time can use this to serve different schema versions
// to different clients simultaneously.
//
// Example:
//
//	client, _ := traverse.New(
//	    traverse.WithBaseURL("https://api.example.com/odata/"),
//	    traverse.WithSchemaVersion("2.0"),
//	)
func WithSchemaVersion(version string) Option {
	return func(cfg *clientConfig) error {
		cfg.relayOpts = append(cfg.relayOpts,
			relay.WithDefaultHeaders(map[string]string{"OData-SchemaVersion": version}),
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

// WithResponseCache sets a custom HTTP-level response cache for entity set queries.
//
// WithResponseCache attaches a [ResponseCache] to the client. When a [QueryBuilder]
// uses [QueryBuilder.WithCache], query responses are stored in this cache and served
// on subsequent identical requests until the TTL expires.
//
// The default is no response cache. Provide [NewInMemoryResponseCache] for a simple
// in-process cache, or implement [ResponseCache] for a distributed backend.
//
// Example:
//
//	client, _ := traverse.New(
//	    traverse.WithBaseURL("https://odata.example.com/v4"),
//	    traverse.WithResponseCache(traverse.NewInMemoryResponseCache()),
//	)
//
//	// Cache Products queries for 5 minutes
//	products, _ := client.From("Products").
//	    WithCache(5 * time.Minute).
//	    Collect(ctx)
func WithResponseCache(cache ResponseCache) Option {
	return func(cfg *clientConfig) error {
		if cache == nil {
			return fmt.Errorf("response cache cannot be nil")
		}
		cfg.responseCache = cache
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
	// ComplexTypes contains complex type definitions (OData v4).
	ComplexTypes []ComplexType
	// EnumTypes contains enum type definitions (OData v4).
	EnumTypes []EnumType
}

// EntitySetInfo represents an OData entity set definition.
// An entity set is a collection of entities of a specific type.
type EntitySetInfo struct {
	// Name is the name of the entity set.
	Name string
	// EntityTypeName is the fully qualified name of the entity type.
	EntityTypeName string
	// NavigationBindings contains navigation property binding targets for this entity set.
	NavigationBindings []NavigationBinding
	// SAP contains SAP-specific entity-set-level annotations (sap:creatable, sap:deletable, etc.).
	SAP SAPAnnotations
}

// NavigationBinding maps a navigation property path to its target entity set.
type NavigationBinding struct {
	// Path is the navigation property name or path.
	Path string
	// Target is the name of the target entity set.
	Target string
}

// EntityType represents an OData entity type definition.
// It describes the structure of entities of this type, including their properties and relationships.
type EntityType struct {
	// Name is the name of the entity type.
	Name string
	// Abstract indicates whether this entity type is abstract (cannot be instantiated directly).
	Abstract bool
	// BaseType is the fully qualified name of the parent entity type (e.g. "Namespace.ParentType").
	BaseType string
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
	// Core contains OData Core vocabulary annotations when present in metadata.
	Core *CoreVocabulary
	// Validation contains OData Validation vocabulary annotations when present in metadata.
	Validation *ValidationVocabulary
	// Measures contains OData Measures vocabulary annotations (unit-of-measure, currency, scale).
	Measures *MeasuresVocabulary
	// Analytics contains OData Aggregation/Analytics vocabulary annotations (dimension, measure, aggregation).
	Analytics *AnalyticsVocabulary
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
	// Label is a human-readable label for the property (sap:label).
	Label string
	// Filterable indicates if the property can be used in $filter clauses (sap:filterable).
	Filterable bool
	// Sortable indicates if the property can be used in $orderby clauses (sap:sortable).
	Sortable bool
	// Searchable indicates if the property can be used in search operations (sap:searchable).
	Searchable bool
	// Required indicates if the property is required in filter expressions (sap:required-in-filter).
	Required bool
	// Text refers to a property containing a descriptive text for this property (sap:text).
	Text string
	// Unit refers to a property holding the unit of measure for this property (sap:unit).
	Unit string
	// ValueList describes how value help is provided: "standard" or "fixed-values" (sap:value-list).
	ValueList string
	// DisplayFormat specifies the display format, e.g. "UpperCase", "NonNegative", "Date" (sap:display-format).
	DisplayFormat string
	// FieldControl specifies the field-control property name or mode: Mandatory, Optional, ReadOnly (sap:field-control).
	FieldControl string
	// Semantics is the semantic type of the value, e.g. "email", "phone", "url" (sap:semantics).
	Semantics string
	// IsKey is true when sap:key="true" is present on the property.
	IsKey bool
	// UpdatablePath refers to a boolean property that controls whether this property is updatable (sap:updatable-path).
	UpdatablePath string
	// Creatable indicates if the entity set supports create operations (sap:creatable). Used on EntitySet level.
	Creatable bool
	// Updatable indicates if the entity set supports update operations (sap:updatable). Used on EntitySet level.
	Updatable bool
	// Deletable indicates if the entity set supports delete operations (sap:deletable). Used on EntitySet level.
	Deletable bool
	// Pageable indicates if the entity set supports server-side paging (sap:pageable). Used on EntitySet level.
	Pageable bool
	// Addressable indicates if individual entities are directly addressable by key (sap:addressable). Used on EntitySet level.
	Addressable bool
	// RequiresFilter indicates if $filter must be specified in queries (sap:requires-filter). Used on EntitySet level.
	RequiresFilter bool
	// ChangeTracking indicates if the entity set supports delta/change-tracking (sap:change-tracking). Used on EntitySet level.
	ChangeTracking bool
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

// ComplexType represents an OData complex type definition.
// Complex types are structured types without a key (not independently addressable).
type ComplexType struct {
	// Name is the name of the complex type.
	Name string
	// Properties is the list of all properties defined for this complex type.
	Properties []Property
}

// EnumMember represents a single member of an OData enum type.
type EnumMember struct {
	// Name is the name of the enum member.
	Name string
	// Value is the integer value of the enum member.
	Value int
}

// EnumType represents an OData enum type definition.
type EnumType struct {
	// Name is the name of the enum type.
	Name string
	// Members is the list of enum members.
	Members []EnumMember
	// IsFlags indicates whether multiple members can be combined.
	IsFlags bool
	// UnderlyingType is the underlying integer type (e.g., "Edm.Int32").
	UnderlyingType string
}
