package traverse

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync"
)

// mapPool is a global object pool for reusing map[string]interface{} allocations.
//
// mapPool reduces garbage collection pressure when processing large datasets by
// reusing map allocations across multiple records. Maps are checked out, filled
// with data, sent to the caller, and returned to the pool for reuse.
var mapPool = sync.Pool{
	New: func() interface{} {
		return make(map[string]interface{})
	},
}

// copyMapDeep creates a deep copy of a map[string]interface{}.
// This prevents race conditions where pool-managed maps are returned to the pool
// before the receiver has finished processing them.
func copyMapDeep(src map[string]interface{}) map[string]interface{} {
	if src == nil {
		return nil
	}
	dst := make(map[string]interface{}, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

// estimateBufferSize calculates optimal channel buffer size based on average record size.
//
// estimateBufferSize uses the formula: bufferSize = 10MB / avgRecordSizeBytes,
// clamped to [32, 1024]. Smaller records get larger buffers (more records fit in 10MB),
// while larger records get smaller buffers (fewer records fit in 10MB).
//
// This optimization balances memory usage with throughput, preventing unbounded
// channel growth while maintaining backpressure for large records.
//
// Returns 256 as fallback if avgRecordSizeBytes is invalid.
func estimateBufferSize(avgRecordSizeBytes int) int {
	if avgRecordSizeBytes <= 0 {
		return 256 // default fallback
	}

	const tenMB = 10 * 1024 * 1024
	estimated := tenMB / avgRecordSizeBytes

	// Clamp between min(32) and max(1024)
	if estimated < 32 {
		return 32
	}
	if estimated > 1024 {
		return 1024
	}
	return estimated
}

// ✅ Optimized: Removed calculateAverageRecordSize() - unnecessary json.Marshal overhead
// Use estimateBufferSize with default value instead

// goroutinePool is an internal pool of workers for executing streaming tasks.
//
// goroutinePool manages a fixed number of goroutines (default 5) that execute
// streaming tasks sequentially. This limits concurrent goroutine creation and
// reduces resource contention, making it efficient for large-scale streaming
// operations.
//
// The pool can be closed with [goroutinePool.close], which waits for all
// pending tasks to complete.
type goroutinePool struct {
	// workers is the number of worker goroutines in the pool
	workers int
	// taskChan is the channel for submitting tasks to the pool
	taskChan chan func()
	// wg synchronizes worker goroutine shutdown
	wg sync.WaitGroup
	// closeOnce ensures close is called only once
	closeOnce sync.Once
	// doneChan signals pool closure
	doneChan chan struct{}
}

// newGoroutinePool creates a new goroutine pool with the given number of workers.
//
// newGoroutinePool creates a pool of worker goroutines that execute tasks
// submitted via [goroutinePool.submit]. If workers <= 0, defaults to 5 workers.
//
// The pool maintains a task channel with capacity 2*workers to balance
// buffering and goroutine blocking.
//
// Example:
//
//	pool := newGoroutinePool(10)
//	defer pool.close()
//	pool.submit(func() { /* task */ })
func newGoroutinePool(workers int) *goroutinePool {
	if workers <= 0 {
		workers = 5 // default pool size
	}
	pool := &goroutinePool{
		workers:  workers,
		taskChan: make(chan func(), workers*2),
		doneChan: make(chan struct{}),
	}

	// Start worker goroutines
	pool.wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer pool.wg.Done()
			for task := range pool.taskChan {
				task()
			}
		}()
	}

	return pool
}

// submit submits a task to the pool for execution.
//
// submit sends a task function to the pool for execution by one of the worker
// goroutines. If the pool is closed, the task is executed inline immediately.
// Otherwise, submit blocks if the pool is at capacity until a worker becomes available.
func (p *goroutinePool) submit(task func()) {
	select {
	case <-p.doneChan:
		// Pool is closed, run inline
		task()
	default:
		p.taskChan <- task
	}
}

// close closes the pool and waits for all pending tasks to complete.
//
// close signals the pool to stop accepting new tasks and waits for all worker
// goroutines to finish executing existing tasks. Uses sync.Once to ensure
// close can be safely called multiple times.
func (p *goroutinePool) close() {
	p.closeOnce.Do(func() {
		close(p.doneChan)
		close(p.taskChan)
		p.wg.Wait()
	})
}

// streamPool is the global singleton goroutine pool for executing streaming operations.
//
// streamPool is created with 5 worker goroutines and is used by [QueryBuilder.Stream]
// and [QueryBuilder.StreamAs] to manage concurrent streaming tasks efficiently.
var streamPool = newGoroutinePool(5)

// resetMapForPool clears a map for reuse in the object pool.
//
// resetMapForPool resets a map by deleting all entries if it's small enough
// to reuse (≤512 entries). If the map has grown very large, it discards the
// map and creates a new small one instead, preventing large maps from being
// held in the pool and wasting memory.
//
// Returns either the cleared map or a new empty map.
func resetMapForPool(m map[string]interface{}) map[string]interface{} {
	// If map is small, clear and reuse it
	if len(m) <= 512 {
		for k := range m {
			delete(m, k)
		}
		return m
	}
	// If map has grown large, discard it and get a new one from pool
	// This prevents large maps from being held in the pool
	return make(map[string]interface{})
}

// returnPageToPool returns processed records from a page to the object pool.
//
// returnPageToPool resets and returns the first count records from page.Value
// to the mapPool for reuse. This is called after records have been sent to
// the caller via the result channel, allowing their map allocations to be reused.
//
// This optimization significantly reduces memory allocations and GC pressure
// when streaming large datasets.
func returnPageToPool(page *Page, count int) {
	for i := 0; i < count && i < len(page.Value); i++ {
		mapPool.Put(resetMapForPool(page.Value[i]))
	}
}

// doStreamPages implements the core streaming logic using json.Decoder.
//
// doStreamPages streams results from a query across all pages without buffering
// entire arrays. It uses token-by-token JSON parsing for memory efficiency and
// applies backpressure via the output channel to prevent unbounded memory growth.
//
// doStreamPages handles:
//   - Pagination across multiple pages using NextLink
//   - Context cancellation with cleanup
//   - Object pool reuse of record maps
//   - Per-page and per-record indexing for [Result] metadata
//
// This is an unexported method called internally by [QueryBuilder.Stream].
func (q *QueryBuilder) doStreamPages(ctx context.Context, out chan<- Result[map[string]interface{}]) {
	pageNum := 1
	nextLink := q.buildURL()

	for nextLink != "" {
		select {
		case <-ctx.Done():
			out <- Result[map[string]interface{}]{
				Err: ctx.Err(),
			}
			return
		default:
		}

		// Fetch and decode the current page
		page, err := q.fetchPageStreamed(ctx, nextLink)
		if err != nil {
			out <- Result[map[string]interface{}]{
				Err: fmt.Errorf("traverse: failed to fetch page %d: %w", pageNum, err),
			}
			return
		}

		// Stream records from this page
		for i, record := range page.Value {
			select {
			case <-ctx.Done():
				// Return remaining records to pool on context cancellation
				returnPageToPool(page, i)
				out <- Result[map[string]interface{}]{
					Err: ctx.Err(),
				}
				return
			case out <- Result[map[string]interface{}]{
				Value: copyMapDeep(record),
				Page:  pageNum,
				Index: i,
			}:
				// Record sent successfully
			}
		}

		// Return all records from this page to pool
		returnPageToPool(page, len(page.Value))

		// Check for next page
		nextLink = page.NextLink
		pageNum++

		// If there's a next link, fetch it in the next iteration
	}
}

// doStreamPagesRaw implements optimized streaming using raw JSON messages.
//
// doStreamPagesRaw streams results without intermediate map allocations, returning
// raw JSON bytes in [RawResult] for direct unmarshaling to custom types. This is
// useful for zero-allocation scenarios and custom type conversions.
//
// Like [QueryBuilder.doStreamPages], it handles pagination, context cancellation,
// and backpressure, but operates on raw JSON data instead of unmarshaled maps.
//
// This is an unexported method called internally by [QueryBuilder.streamRaw].
func (q *QueryBuilder) doStreamPagesRaw(ctx context.Context, out chan<- RawResult) {
	pageNum := 1
	nextLink := q.buildURL()

	for nextLink != "" {
		select {
		case <-ctx.Done():
			out <- RawResult{
				Err: ctx.Err(),
			}
			return
		default:
		}

		// Fetch and decode the current page
		page, err := q.fetchPageStreamed(ctx, nextLink)
		if err != nil {
			out <- RawResult{
				Err: fmt.Errorf("traverse: failed to fetch page %d: %w", pageNum, err),
			}
			return
		}

		// Stream raw JSON records from this page
		for i, rawRecord := range page.RawValue {
			select {
			case <-ctx.Done():
				out <- RawResult{
					Err: ctx.Err(),
				}
				return
			case out <- RawResult{
				Raw:   rawRecord,
				Page:  pageNum,
				Index: i,
			}:
				// Record sent successfully
			}
		}

		// Check for next page
		nextLink = page.NextLink
		pageNum++

		// If there's a next link, fetch it in the next iteration
	}
}

// fetchPageStreamed fetches and parses a single page of results using streaming.
//
// fetchPageStreamed executes an HTTP GET request for the given pageURL and
// uses token-by-token JSON parsing to extract results without buffering the
// entire response body. This minimizes memory usage for large responses.
//
// The response is parsed based on the client's detected OData version (v2 or v4),
// automatically handling version-specific JSON wrapping and field names.
//
// Returns an error if the HTTP status is not 200 or if JSON parsing fails.
//
// This is an unexported method called internally by streaming functions.
func (q *QueryBuilder) fetchPageStreamed(ctx context.Context, pageURL string) (*Page, error) {
	req := q.client.http.Get(pageURL)
	req = req.WithContext(ctx)

	// Use ExecuteStream to avoid buffering the entire response body
	stream, err := q.client.http.ExecuteStream(req)
	if err != nil {
		return nil, fmt.Errorf("ExecuteStream failed: %w", err)
	}
	defer func() { _ = stream.Body.Close() }()

	// Check response status
	if stream.StatusCode != 200 {
		// Try to parse error response
		body, _ := io.ReadAll(stream.Body)
		return nil, fmt.Errorf("HTTP %d: %s", stream.StatusCode, string(body))
	}

	// Create json.Decoder and stream-parse the response
	page := &Page{
		Value: make([]map[string]interface{}, 0, q.client.pageSize),
	}

	decoder := json.NewDecoder(stream.Body)

	// Parse the JSON structure token-by-token
	if err := parseODataResponse(decoder, page, q.client.version); err != nil {
		return nil, fmt.Errorf("failed to parse OData response: %w", err)
	}

	return page, nil
}

// parseODataResponse decodes an OData response using token-by-token JSON streaming.
//
// parseODataResponse parses the top-level response object and extracts:
//   - value: array of entities (OData v4) or via "d.results" (OData v2)
//   - @odata.nextLink/@odata.nextLink or __next: pagination link
//   - @odata.count: total count of matching records
//   - @odata.context: metadata context URI (v4 only)
//   - @odata.deltaLink: delta sync link (v4 only)
//
// Automatically handles version-specific formats and field names.
// Unknown fields are skipped without error.
//
// This is an unexported function used internally for response parsing.
func parseODataResponse(decoder *json.Decoder, page *Page, version ODataVersion) error {
	// First token should be '{'
	token, err := decoder.Token()
	if err != nil {
		return fmt.Errorf("failed to read first token: %w", err)
	}
	if delim, ok := token.(json.Delim); !ok || delim != '{' {
		return fmt.Errorf("expected '{', got %v", token)
	}

	// Navigate through the root object
	for decoder.More() {
		token, err := decoder.Token()
		if err != nil {
			return fmt.Errorf("failed to read token: %w", err)
		}

		// Must be a string key
		key, ok := token.(string)
		if !ok {
			continue
		}

		switch key {
		case "value":
			// OData v4 format: "value": [...]
			if version == ODataV4 {
				if err := parseValueArray(decoder, page); err != nil {
					return err
				}
			}

		case "d":
			// OData v2 format: "d": {"results": [...], "__next": "..."}
			if version == ODataV2 {
				if err := parseODataV2Wrapper(decoder, page); err != nil {
					return err
				}
			}

		case "@odata.nextLink":
			// OData v4 next link
			var nextLink string
			if err := decoder.Decode(&nextLink); err != nil {
				return fmt.Errorf("failed to decode @odata.nextLink: %w", err)
			}
			page.NextLink = nextLink

		case "__next":
			// OData v2 next link (also can be in the "d" wrapper)
			var nextLink string
			if err := decoder.Decode(&nextLink); err != nil {
				return fmt.Errorf("failed to decode __next: %w", err)
			}
			page.NextLink = nextLink

		case "@odata.count":
			// Total count
			var count int64
			if err := decoder.Decode(&count); err != nil {
				return fmt.Errorf("failed to decode @odata.count: %w", err)
			}
			page.Count = &count

		case "@odata.context":
			// Metadata context URI
			var ctx string
			if err := decoder.Decode(&ctx); err != nil {
				return fmt.Errorf("failed to decode @odata.context: %w", err)
			}
			page.Context = ctx

		case "@odata.deltaLink":
			// Delta link for incremental sync
			var deltaLink string
			if err := decoder.Decode(&deltaLink); err != nil {
				return fmt.Errorf("failed to decode @odata.deltaLink: %w", err)
			}
			page.DeltaLink = deltaLink

		default:
			// Skip unknown fields
			var tmp interface{}
			if err := decoder.Decode(&tmp); err != nil && err != io.EOF {
				return fmt.Errorf("failed to skip field %q: %w", key, err)
			}
		}
	}

	// Final closing '}'
	_, err = decoder.Token()
	if err != nil && err != io.EOF {
		return fmt.Errorf("failed to read closing token: %w", err)
	}

	return nil
}

// parseODataV2Wrapper parses the OData v2 "d" wrapper object.
//
// parseODataV2Wrapper handles the OData v2-specific wrapper format where results
// are nested inside a "d" object: { "d": { "results": [...], "__next": "...", "__count": ... } }
//
// Extracts the "results" array (entity records), "__next" link for pagination,
// and "__count" for total count (if $count=true was used).
//
// This is an unexported function used internally for OData v2 response parsing.
func parseODataV2Wrapper(decoder *json.Decoder, page *Page) error {
	// Should see '{'
	token, err := decoder.Token()
	if err != nil {
		return fmt.Errorf("failed to read wrapper opening: %w", err)
	}
	if delim, ok := token.(json.Delim); !ok || delim != '{' {
		return fmt.Errorf("expected '{' in d wrapper, got %v", token)
	}

	for decoder.More() {
		token, err := decoder.Token()
		if err != nil {
			return fmt.Errorf("failed to read wrapper token: %w", err)
		}

		key, ok := token.(string)
		if !ok {
			continue
		}

		switch key {
		case "results":
			if err := parseValueArray(decoder, page); err != nil {
				return err
			}

		case "__next":
			var nextLink string
			if err := decoder.Decode(&nextLink); err != nil {
				return fmt.Errorf("failed to decode __next: %w", err)
			}
			page.NextLink = nextLink

		case "__count":
			// OData v2 count format
			var count interface{}
			if err := decoder.Decode(&count); err != nil {
				return fmt.Errorf("failed to decode __count: %w", err)
			}
			// Convert to int64 if possible
			if c, ok := count.(float64); ok {
				cnt := int64(c)
				page.Count = &cnt
			}

		default:
			// Skip unknown fields
			var tmp interface{}
			if err := decoder.Decode(&tmp); err != nil && err != io.EOF {
				return fmt.Errorf("failed to skip wrapper field %q: %w", key, err)
			}
		}
	}

	// Closing '}'
	_, err = decoder.Token()
	if err != nil && err != io.EOF {
		return fmt.Errorf("failed to read wrapper closing: %w", err)
	}

	return nil
}

// parseValueArray parses the "value" array of entity records.
//
// parseValueArray uses [json.Decoder.More] and [json.Decoder.Decode] to stream
// records one-by-one without buffering the entire array in memory. This enables
// efficient processing of very large result sets.
//
// The function implements object pooling: map[string]interface{} allocations
// are reused from mapPool to reduce GC pressure on large datasets.
//
// Also captures raw JSON messages in page.RawValue for optimized direct type
// conversion paths (see [QueryBuilder.streamRaw]).
//
// This is an unexported function used internally for response parsing.
func parseValueArray(decoder *json.Decoder, page *Page) error {
	// Should see '['
	token, err := decoder.Token()
	if err != nil {
		return fmt.Errorf("failed to read array opening: %w", err)
	}
	if delim, ok := token.(json.Delim); !ok || delim != '[' {
		return fmt.Errorf("expected '[', got %v", token)
	}

	// Stream each array element
	for decoder.More() {
		// Capture raw JSON first by decoding to RawMessage
		var rawMsg json.RawMessage
		if err := decoder.Decode(&rawMsg); err != nil {
			return fmt.Errorf("failed to decode record: %w", err)
		}
		page.RawValue = append(page.RawValue, rawMsg)

		// Also decode to map for backward compatibility with existing code
		// Get a map from the pool
		record := mapPool.Get().(map[string]interface{})
		if err := json.Unmarshal(rawMsg, &record); err != nil {
			// Return the map to pool even on error
			mapPool.Put(resetMapForPool(record))
			return fmt.Errorf("failed to unmarshal raw record to map: %w", err)
		}
		page.Value = append(page.Value, record)
		// Note: The map is not returned to the pool here because it's stored
		// in page.Value. The caller is responsible for pool management after
		// the page is processed.
	}

	// Closing ']'
	token, err = decoder.Token()
	if err != nil && err != io.EOF {
		return fmt.Errorf("failed to read array closing: %w", err)
	}
	if delim, ok := token.(json.Delim); ok && delim != ']' {
		return fmt.Errorf("expected ']', got %v", token)
	}

	return nil
}

// AutoDetectODataVersion attempts to detect the OData version from the response.
// Checks the "OData-Version" header first, then falls back to structural detection.
// AutoDetectODataVersion detects the OData version from HTTP response headers.
//
// AutoDetectODataVersion examines the "Odata-Version" and "Dataserviceversion"
// headers to determine whether the response follows OData v2 or v4 conventions.
//
// Returns [ODataV4] as default if the version cannot be determined from headers.
//
// Supported versions:
//   - OData v4: "Odata-Version: 4.0"
//   - OData v2: "Odata-Version: 2.0" or "Dataserviceversion: 2.0"
func AutoDetectODataVersion(headers map[string][]string) ODataVersion {
	// Check for OData-Version header
	if versions, ok := headers["Odata-Version"]; ok && len(versions) > 0 {
		switch versions[0] {
		case "4.0":
			return ODataV4
		case "2.0":
			return ODataV2
		}
	}

	// Check for DataServiceVersion header (older v2)
	if versions, ok := headers["Dataserviceversion"]; ok && len(versions) > 0 {
		switch versions[0] {
		case "2.0":
			return ODataV2
		}
	}

	// Default to v4 if cannot detect
	return ODataV4
}

// DeltaResult represents a record from a delta sync operation.
//
// DeltaResult is used with delta queries to track both added/modified records
// (in Value) and deleted records (indicated by Removed=true).
//
// Fields:
//   - Value: the entity record (for non-deleted records)
//   - Removed: true if the record was deleted
//   - Reason: optional deletion reason (if provided by server)
//   - Err: error if one occurred during parsing
//
// See [QueryBuilder.WithDeltaToken] for incremental sync usage.
type DeltaResult struct {
	// Value is the entity record (nil for deleted records)
	Value map[string]interface{}
	// Removed is true if the record was deleted
	Removed bool
	// Reason is the deletion reason if available
	Reason string
	// Err is an error if one occurred
	Err error
}
