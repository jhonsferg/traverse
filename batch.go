package traverse

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/textproto"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jhonsferg/relay"
)

// headerMapPool is a sync.Pool for reusing map[string]string allocations.
//
// headerMapPool reduces allocations in batch operations where multiple operations
// each need header maps. Maps are cleared and reused from the pool.
var headerMapPool = sync.Pool{
	New: func() interface{} {
		return make(map[string]string, 10)
	},
}

// BatchRequest represents a collection of OData operations to execute as a batch.
//
// BatchRequest builds a $batch request combining multiple operations (GET, POST, PATCH, DELETE)
// into a single HTTP request. Operations can be grouped into changesets for atomic transactions.
//
// Typical usage:
//
//	resp, err := client.Batch().
//		Get("Customers", 1).
//		Create("Orders", orderData).
//		Execute(ctx)
//
// For transactional operations, use BeginChangeset/EndChangeset:
//
//	resp, err := client.Batch().
//		BeginChangeset("tx1").
//		Create("Orders", order).
//		Update("Customers", 1, updateData).
//		EndChangeset().
//		Execute(ctx)
type BatchRequest struct {
	// client is the OData client
	client *Client
	// ops contains operations outside changesets
	ops []BatchOperation
	// changesets contains all changesets by ID
	changesets map[string]*changeset
	// currentCS is the currently open changeset being built
	currentCS *changeset
	// buildErrs collects errors from builder methods (e.g. invalid key types)
	// that cannot be returned directly due to method-chaining API.
	buildErrs []error
}

// changeset represents a transaction group within a batch request.
//
// changeset groups operations that must be executed atomically.
type changeset struct {
	// id is the changeset identifier
	id string
	// ops contains the operations in this changeset
	ops []BatchOperation
}

// BatchOperation represents a single operation within a batch request.
//
// BatchOperation encapsulates a single OData operation (GET, POST, PATCH, DELETE)
// with its method, URL, headers, and optional body data.
type BatchOperation struct {
	// Method is the HTTP method (GET, POST, PATCH, DELETE)
	Method string
	// URL is the entity set or entity reference
	URL string
	// Headers are operation-specific HTTP headers
	Headers map[string]string
	// Body is the request body as raw JSON (for POST/PATCH)
	Body json.RawMessage
	// ChangesetID identifies the changeset this operation belongs to
	ChangesetID string
}

// acquireHeaders gets a headers map from the pool.
//
// acquireHeaders returns a cleared map[string]string from headerMapPool for reuse.
func acquireHeaders() map[string]string {
	return headerMapPool.Get().(map[string]string)
}

// releaseHeaders returns a headers map to the pool after clearing.
//
// releaseHeaders clears all entries from the map and returns it to headerMapPool
// for reuse, reducing allocations in batch operation processing.
func releaseHeaders(h map[string]string) {
	if h == nil {
		return
	}
	// Clear all entries before returning to pool
	for k := range h {
		delete(h, k)
	}
	headerMapPool.Put(h)
}

// SetBody sets the Body field, marshaling data if needed.
//
// SetBody converts the provided data to JSON and stores it as a [json.RawMessage]
// in the Body field. Returns an error if JSON marshaling fails. Nil data results
// in a nil Body.
//
// This helper maintains API compatibility while optimizing allocations by using
// raw JSON instead of unmarshaling on the server side.
func (op *BatchOperation) SetBody(data interface{}) error {
	if data == nil {
		op.Body = nil
		return nil
	}

	b, err := json.Marshal(data)
	if err != nil {
		return err
	}
	op.Body = json.RawMessage(b)
	return nil
}

// batchMaxResponseCount is the maximum number of batch response parts traverse
// will parse. This prevents OOM from server responses with unbounded part counts.
const batchMaxResponseCount = 10_000

// batchMaxPartBodySize is the maximum number of bytes read per batch response part.
const batchMaxPartBodySize = 10 * 1024 * 1024 // 10 MB
// BatchResponse contains the results of all operations in a batch request,
// with one [BatchResult] entry per operation in the same order as submitted.
type BatchResponse struct {
	// Results is the slice of results, one per operation
	Results []BatchResult
}

// BatchResult represents the result of a single operation in the batch.
//
// BatchResult contains the HTTP response status, headers, and body from one
// operation within a batch request.
type BatchResult struct {
	// StatusCode is the HTTP status code of the operation
	StatusCode int
	// Headers are the response headers
	Headers map[string]string
	// Body is the response body
	Body []byte
	// Err is an error if one occurred
	Err error
}

// Batch starts building a batch request.
//
// Batch returns a new [BatchRequest] associated with the client, ready to add
// operations (Get, Create, Update, Delete) or changesets.
//
// Example:
//
//	batch := client.Batch().
//		Get("Customers", 1).
//		Create("Orders", orderData)
//	resp, err := batch.Execute(ctx)
func (c *Client) Batch() *BatchRequest {
	return &BatchRequest{
		client:     c,
		ops:        make([]BatchOperation, 0),
		changesets: make(map[string]*changeset),
	}
}

// Get adds a GET operation to the batch.
//
// Get retrieves a single entity by its key and adds the operation to the batch.
// The operation is read-only and cannot be part of a changeset.
//
// Returns the receiver for method chaining.
//
// Example:
//
//	batch.Get("Customers", 1)
func (b *BatchRequest) Get(entitySet string, key interface{}) *BatchRequest {
	keyStr, err := encodeKey(key)
	if err != nil {
		b.buildErrs = append(b.buildErrs, fmt.Errorf("traverse batch.Get: invalid key (%T %v): %w", key, key, err))
		fmt.Fprintf(os.Stderr, "warning: traverse batch.Get: invalid key (%T %v): %v\n", key, key, err)
	}
	var urlBuilder strings.Builder
	urlBuilder.WriteString(entitySet)
	urlBuilder.WriteString("(")
	urlBuilder.WriteString(keyStr)
	urlBuilder.WriteString(")")
	op := BatchOperation{
		Method:  "GET",
		URL:     urlBuilder.String(),
		Headers: acquireHeaders(),
	}

	if b.currentCS != nil {
		op.ChangesetID = b.currentCS.id
		b.currentCS.ops = append(b.currentCS.ops, op)
	} else {
		b.ops = append(b.ops, op)
	}
	return b
}

// Create adds a POST (create) operation to the batch.
//
// Create inserts a new entity into the specified entity set. The data is marshaled
// to JSON automatically. If Create is called within a changeset (between
// BeginChangeset and EndChangeset), the operation becomes part of that transaction.
//
// Returns the receiver for method chaining.
//
// Example:
//
//	batch.Create("Orders", map[string]interface{}{"OrderID": 123, "Total": 99.99})
func (b *BatchRequest) Create(entitySet string, data interface{}) *BatchRequest {
	op := BatchOperation{
		Method:  "POST",
		URL:     entitySet,
		Headers: acquireHeaders(),
	}

	if data != nil {
		if err := op.SetBody(data); err != nil {
			// Log error but continue - don't break the chain
			fmt.Fprintf(os.Stderr, "warning: failed to marshal batch data: %v\n", err)
		}
	}

	if b.currentCS != nil {
		op.ChangesetID = b.currentCS.id
		b.currentCS.ops = append(b.currentCS.ops, op)
	} else {
		b.ops = append(b.ops, op)
	}
	return b
}

// Update adds a PATCH (update) operation to the batch.
//
// Update modifies an existing entity identified by its key. The data is marshaled
// to JSON and sent as PATCH. If called within a changeset, the operation becomes
// part of that atomic transaction.
//
// Returns the receiver for method chaining.
//
// Example:
//
//	batch.Update("Orders", 123, map[string]interface{}{"Total": 150.00})
func (b *BatchRequest) Update(entitySet string, key interface{}, data interface{}) *BatchRequest {
	keyStr, err := encodeKey(key)
	if err != nil {
		b.buildErrs = append(b.buildErrs, fmt.Errorf("traverse batch.Update: invalid key (%T %v): %w", key, key, err))
		fmt.Fprintf(os.Stderr, "warning: traverse batch.Update: invalid key (%T %v): %v\n", key, key, err)
	}
	var urlBuilder strings.Builder
	urlBuilder.WriteString(entitySet)
	urlBuilder.WriteString("(")
	urlBuilder.WriteString(keyStr)
	urlBuilder.WriteString(")")
	op := BatchOperation{
		Method:  "PATCH",
		URL:     urlBuilder.String(),
		Headers: acquireHeaders(),
	}

	if data != nil {
		if err := op.SetBody(data); err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to marshal batch data: %v\n", err)
		}
	}

	if b.currentCS != nil {
		op.ChangesetID = b.currentCS.id
		b.currentCS.ops = append(b.currentCS.ops, op)
	} else {
		b.ops = append(b.ops, op)
	}
	return b
}

// Delete adds a DELETE operation to the batch.
//
// Delete removes an entity identified by its key. If called within a changeset,
// the deletion becomes part of that atomic transaction.
//
// Returns the receiver for method chaining.
//
// Example:
//
//	batch.Delete("Orders", 123)
func (b *BatchRequest) Delete(entitySet string, key interface{}) *BatchRequest {
	keyStr, err := encodeKey(key)
	if err != nil {
		b.buildErrs = append(b.buildErrs, fmt.Errorf("traverse batch.Delete: invalid key (%T %v): %w", key, key, err))
		fmt.Fprintf(os.Stderr, "warning: traverse batch.Delete: invalid key (%T %v): %v\n", key, key, err)
	}
	var urlBuilder strings.Builder
	urlBuilder.WriteString(entitySet)
	urlBuilder.WriteString("(")
	urlBuilder.WriteString(keyStr)
	urlBuilder.WriteString(")")
	op := BatchOperation{
		Method:  "DELETE",
		URL:     urlBuilder.String(),
		Headers: acquireHeaders(),
	}

	if b.currentCS != nil {
		op.ChangesetID = b.currentCS.id
		b.currentCS.ops = append(b.currentCS.ops, op)
	} else {
		b.ops = append(b.ops, op)
	}
	return b
}

// BeginChangeset starts a changeset (atomic transaction group).
//
// BeginChangeset marks the beginning of a set of operations that should be
// executed atomically. All operations added after this call (until EndChangeset)
// belong to this changeset. Multiple changesets can exist in a single batch.
//
// The id parameter is used to group related operations. If a changeset was
// already open, it is closed first.
//
// Returns the receiver for method chaining.
//
// Example:
//
//	batch.BeginChangeset("group-1").
//		Create("Orders", order).
//		Create("OrderItems", item1).
//		EndChangeset()
func (b *BatchRequest) BeginChangeset(id string) *BatchRequest {
	// End previous changeset if exists
	if b.currentCS != nil {
		b.changesets[b.currentCS.id] = b.currentCS
	}

	// Start new changeset
	b.currentCS = &changeset{
		id:  id,
		ops: make([]BatchOperation, 0),
	}
	return b
}

// EndChangeset ends the current changeset.
//
// EndChangeset closes the active changeset, storing it for later execution.
// All subsequent operations will be standalone batch operations (not in a changeset)
// until another BeginChangeset is called.
//
// Returns the receiver for method chaining.
//
// Example:
//
//	batch.BeginChangeset("tx1").
//		Create("Orders", orderData).
//		EndChangeset().
//		Get("Customers", 1)
func (b *BatchRequest) EndChangeset() *BatchRequest {
	if b.currentCS != nil {
		b.changesets[b.currentCS.id] = b.currentCS
		b.currentCS = nil
	}
	return b
}

// Execute sends the batch request and returns all results at once.
//
// Execute constructs a multipart/mixed HTTP request containing all batch operations
// and changesets, sends it to the OData service, and parses the response. The operation
// automatically closes any open changeset before execution.
//
// All results are materialized into memory before returning. For large batches,
// consider using [BatchRequest.ExecuteStream] for memory-efficient incremental processing.
//
// On error, returns a non-nil error. The returned [BatchResponse] contains individual
// results for each operation, even if some operations failed.
//
// Example:
//
//	resp, err := batch.Execute(ctx)
//	if err != nil {
//		log.Fatal(err)
//	}
//	for _, result := range resp.Results {
//		fmt.Println(result.Status, result.Data)
//	}
func (b *BatchRequest) Execute(ctx context.Context) (*BatchResponse, error) {
	// Ensure any open changeset is closed
	if b.currentCS != nil {
		b.EndChangeset()
	}

	// Surface any build-time errors (e.g. invalid key types in Get/Update/Delete).
	if len(b.buildErrs) > 0 {
		return nil, fmt.Errorf("traverse: batch has build errors: %w", errors.Join(b.buildErrs...))
	}

	// Build multipart/mixed request body
	body, boundary, err := b.buildMultipartBody()
	if err != nil {
		return nil, fmt.Errorf("traverse: batch build failed: %w", err)
	}

	// POST /$batch
	var ctBuilder strings.Builder
	ctBuilder.WriteString("multipart/mixed; boundary=")
	ctBuilder.WriteString(boundary)
	url := "/$batch"

	req := b.client.http.Post(url)
	req = req.WithContext(ctx)
	req = req.WithHeader("Content-Type", ctBuilder.String())
	req = req.WithBody(body)

	resp, err := b.client.http.Execute(req)
	if err != nil {
		return nil, fmt.Errorf("traverse: batch execute failed: %w", err)
	}

	// Parse multipart response
	results, err := b.parseMultipartResponse(resp)
	if err != nil {
		return nil, fmt.Errorf("traverse: batch parse failed: %w", err)
	}

	// Clean up and return header maps to pool
	b.release()

	return &BatchResponse{Results: results}, nil
}

// ExecuteStream sends the batch request and streams results incrementally via a channel.
//
// ExecuteStream is more memory-efficient than [BatchRequest.Execute] for large batches.
// Results are parsed and sent to the returned channel as they arrive, allowing processing
// to begin before the entire response is received. The operation automatically closes any
// open changeset before execution.
//
// The returned channel is buffered with capacity 8 and will be closed when all results
// have been sent or an error occurs. Errors are sent as [BatchResult] items with non-nil Err field.
//
// Example:
//
//	results := batch.ExecuteStream(ctx)
//	for result := range results {
//		if result.Err != nil {
//			log.Println("Operation failed:", result.Err)
//			continue
//		}
//		fmt.Println(result.Status, string(result.Body))
//	}
func (b *BatchRequest) ExecuteStream(ctx context.Context) <-chan BatchResult {
	out := make(chan BatchResult, 8) // Buffered channel for streaming results

	go func() {
		defer close(out)
		defer b.release()

		// Ensure any open changeset is closed
		if b.currentCS != nil {
			b.EndChangeset()
		}

		// Surface any build-time errors.
		if len(b.buildErrs) > 0 {
			out <- BatchResult{
				Err: fmt.Errorf("traverse: batch has build errors: %w", errors.Join(b.buildErrs...)),
			}
			return
		}

		// Build multipart/mixed request body
		body, boundary, err := b.buildMultipartBody()
		if err != nil {
			out <- BatchResult{
				Err: fmt.Errorf("traverse: batch build failed: %w", err),
			}
			return
		}

		// POST /$batch
		url := "/$batch"
		contentType := fmt.Sprintf("multipart/mixed; boundary=%s", boundary)

		req := b.client.http.Post(url)
		req = req.WithContext(ctx)
		req = req.WithHeader("Content-Type", contentType)
		req = req.WithBody(body)

		resp, err := b.client.http.Execute(req)
		if err != nil {
			out <- BatchResult{
				Err: fmt.Errorf("traverse: batch execute failed: %w", err),
			}
			return
		}

		// Stream multipart response incrementally.
		// Use a non-blocking send for the terminal error: if the caller cancelled
		// and stopped reading, the buffer may be full and the goroutine would
		// otherwise leak trying to deliver the final error.
		if err = b.streamMultipartResponse(ctx, resp, out); err != nil {
			select {
			case out <- BatchResult{Err: fmt.Errorf("traverse: batch parse failed: %w", err)}:
			default:
			}
		}
	}()

	return out
}

// streamMultipartResponse streams batch response results to a channel incrementally.
//
// streamMultipartResponse parses the multipart/mixed response from the OData service
// and sends each result (or error) to the output channel as it's parsed. This approach
// avoids loading the entire response into memory, making it ideal for large batches.
//
// The method respects context cancellation and will stop parsing if the context is
// canceled. Changesets are handled recursively; their results are also streamed to
// the same output channel.
//
// Returns an error if the response cannot be parsed or if an HTTP error status is received.
func (b *BatchRequest) streamMultipartResponse(ctx context.Context, resp *relay.Response, out chan<- BatchResult) error {
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("traverse: batch request failed with status %d: %w", resp.StatusCode, ErrBatchFailed)
	}

	// Extract boundary from Content-Type header
	contentType := resp.Headers.Get("Content-Type")
	if contentType == "" {
		return fmt.Errorf("traverse: batch response missing Content-Type header")
	}

	// Parse boundary from "multipart/mixed; boundary=..."
	boundary := extractBoundary(contentType)
	if boundary == "" {
		return fmt.Errorf("traverse: could not extract boundary from Content-Type")
	}

	// Parse multipart response
	reader := bytes.NewReader(resp.Body())
	mr := multipart.NewReader(reader, boundary)

	var partCount int
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		part, err := mr.NextPart()
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("traverse: error reading batch response: %w", err)
		}

		partCount++
		if partCount > batchMaxResponseCount {
			return fmt.Errorf("traverse: batch response exceeds maximum of %d parts", batchMaxResponseCount)
		}

		// Read response part
		contentTypeHeader := part.Header.Get("Content-Type")
		if strings.HasPrefix(contentTypeHeader, "multipart/mixed") {
			// This is a changeset response - stream its results
			csBoundary := extractBoundary(contentTypeHeader)
			err = b.streamChangesetResponse(ctx, part, csBoundary, out)
			if err != nil {
				return err
			}
		} else {
			// Regular response
			var result BatchResult
			result, err = b.parseResponsePart(part)
			if err != nil {
				return err
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case out <- result:
			}
		}
	}

	return nil
}

// streamChangesetResponse streams changeset response parts to a channel.
//
// streamChangesetResponse is a helper that processes the multipart/mixed changeset
// response, parsing each operation's result and sending them to the output channel.
// This is called recursively during batch response streaming to handle changesets.
//
// Returns an error if parsing fails or context is canceled.
func (b *BatchRequest) streamChangesetResponse(ctx context.Context, part *multipart.Part, boundary string, out chan<- BatchResult) error {
	reader := multipart.NewReader(part, boundary)

	var partCount int
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		part, err := reader.NextPart()
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("traverse: error reading changeset response: %w", err)
		}

		partCount++
		if partCount > batchMaxResponseCount {
			return fmt.Errorf("traverse: changeset response exceeds maximum of %d parts", batchMaxResponseCount)
		}

		var result BatchResult
		result, err = b.parseResponsePart(part)
		if err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case out <- result:
		}
	}

	return nil
}

// buildMultipartBody constructs the multipart/mixed request body for the batch.
//
// buildMultipartBody creates a multipart/mixed MIME message containing all batch
// operations and changesets. It generates a unique boundary and organizes operations
// into the correct structure: standalone operations first, then changesets.
//
// Each operation is written with appropriate MIME headers (Content-Type: application/http).
// Changesets are wrapped in their own multipart/mixed sections to ensure transactional semantics.
//
// Returns the serialized body as bytes, the boundary string used, or an error if
// serialization fails.
func (b *BatchRequest) buildMultipartBody() ([]byte, string, error) {
	buf := &bytes.Buffer{}
	var boundaryBuilder strings.Builder
	boundaryBuilder.WriteString("batch_")
	boundaryBuilder.WriteString(strconv.FormatInt(time.Now().UnixNano(), 10))
	boundary := boundaryBuilder.String()
	w := multipart.NewWriter(buf)
	if err := w.SetBoundary(boundary); err != nil {
		return nil, "", err
	}

	// Write non-changeset operations first
	for _, op := range b.ops {
		if op.ChangesetID == "" {
			if err := b.writeBatchOperation(w, &op); err != nil {
				return nil, "", err
			}
		}
	}

	// Write changesets
	for _, cs := range b.changesets {
		if err := b.writeChangeset(w, cs, boundary); err != nil {
			return nil, "", err
		}
	}

	if err := w.Close(); err != nil {
		return nil, "", err
	}
	return buf.Bytes(), boundary, nil
}

// writeBatchOperation writes a single operation to the multipart writer.
//
// writeBatchOperation serializes a [BatchOperation] as an HTTP request within the
// multipart/mixed MIME structure. It writes the operation method, URL, headers,
// and optional body. The result is a complete HTTP request embedded in the
// multipart section.
//
// The function uses strings.Builder with pre-allocation for efficient formatting.
//
// Returns an error if writing to the multipart part fails.
func (b *BatchRequest) writeBatchOperation(w *multipart.Writer, op *BatchOperation) error {
	h := make(textproto.MIMEHeader)
	h.Set("Content-Type", "application/http")
	h.Set("Content-Transfer-Encoding", "binary")

	part, err := w.CreatePart(h)
	if err != nil {
		return err
	}

	// Use strings.Builder to efficiently construct request line and headers
	var reqBuilder strings.Builder
	reqBuilder.Grow(256) // Pre-allocate for typical request

	// Serialize the operation as HTTP request
	reqBuilder.WriteString(op.Method)
	reqBuilder.WriteByte(' ')
	reqBuilder.WriteString(sanitizeHeaderValue(op.URL))
	reqBuilder.WriteString(" HTTP/1.1\r\n")

	// Write headers
	for k, v := range op.Headers {
		reqBuilder.WriteString(sanitizeHeaderValue(k))
		reqBuilder.WriteString(": ")
		reqBuilder.WriteString(sanitizeHeaderValue(v))
		reqBuilder.WriteString("\r\n")
	}

	// Write body if present
	if op.Body != nil {
		bodyBytes := []byte(op.Body)

		reqBuilder.WriteString("Content-Length: ")
		reqBuilder.WriteString(strconv.Itoa(len(bodyBytes)))
		reqBuilder.WriteString("\r\n\r\n")

		if _, err := part.Write([]byte(reqBuilder.String())); err != nil {
			return err
		}

		if _, err := part.Write(bodyBytes); err != nil {
			return err
		}
	} else {
		// Empty line separates headers from body (even if no body)
		reqBuilder.WriteString("\r\n")
		if _, err := part.Write([]byte(reqBuilder.String())); err != nil {
			return err
		}
	}

	return nil
}

// writeChangeset writes a changeset (atomic transaction group) in multipart/mixed format.
//
// writeChangeset packages all operations belonging to a changeset into a nested
// multipart/mixed section. Each operation is assigned a Content-ID for tracking.
// This structure ensures the OData service processes all changeset operations
// atomically (all succeed or all fail).
//
// The parentBoundary parameter is used to generate unique changeset boundaries
// by incorporating the changeset ID and timestamp.
//
// Returns an error if multipart writing fails.
func (b *BatchRequest) writeChangeset(w *multipart.Writer, cs *changeset, parentBoundary string) error {
	h := make(textproto.MIMEHeader)
	csBoundary := fmt.Sprintf("changeset_%s_%d", cs.id, time.Now().UnixNano())
	h.Set("Content-Type", fmt.Sprintf("multipart/mixed; boundary=%s", csBoundary))

	part, err := w.CreatePart(h)
	if err != nil {
		return err
	}

	// Create changeset writer
	csWriter := multipart.NewWriter(part)
	if err := csWriter.SetBoundary(csBoundary); err != nil {
		return err
	}

	// Write all operations in the changeset
	for i, op := range cs.ops {
		// Add content ID for change tracking
		h := make(textproto.MIMEHeader)
		h.Set("Content-Type", "application/http")
		h.Set("Content-Transfer-Encoding", "binary")
		h.Set("Content-ID", fmt.Sprintf("%s-%d", cs.id, i))

		csPart, err := csWriter.CreatePart(h)
		if err != nil {
			return err
		}

		// Serialize operation
		if _, err := fmt.Fprintf(csPart, "%s %s HTTP/1.1\r\n", op.Method, sanitizeHeaderValue(op.URL)); err != nil {
			return err
		}

		// Write headers
		for k, v := range op.Headers {
			if _, err := fmt.Fprintf(csPart, "%s: %s\r\n", sanitizeHeaderValue(k), sanitizeHeaderValue(v)); err != nil {
				return err
			}
		}

		// Write body if present
		if op.Body != nil {
			bodyBytes := []byte(op.Body)

			if _, err := fmt.Fprintf(csPart, "Content-Length: %d\r\n", len(bodyBytes)); err != nil {
				return err
			}

			if _, err := csPart.Write([]byte("\r\n")); err != nil {
				return err
			}

			if _, err := csPart.Write(bodyBytes); err != nil {
				return err
			}
		} else {
			if _, err := csPart.Write([]byte("\r\n")); err != nil {
				return err
			}
		}
	}

	if err := csWriter.Close(); err != nil {
		return err
	}
	return nil
}

// parseMultipartResponse parses the multipart/mixed response from the OData service.
//
// parseMultipartResponse extracts all batch operation results (including changeset
// results) from the multipart/mixed HTTP response. It handles both standalone
// responses and nested changeset responses.
//
// The response is fully materialized into memory. For large batches,
// consider using [BatchRequest.ExecuteStream] instead.
//
// Returns a slice of [BatchResult] for each operation, or an error if parsing fails.
func (b *BatchRequest) parseMultipartResponse(resp *relay.Response) ([]BatchResult, error) {
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("traverse: batch request failed with status %d", resp.StatusCode)
	}

	// Extract boundary from Content-Type header
	contentType := resp.Headers.Get("Content-Type")
	if contentType == "" {
		return nil, fmt.Errorf("traverse: batch response missing Content-Type header")
	}

	// Parse boundary from "multipart/mixed; boundary=..."
	boundary := extractBoundary(contentType)
	if boundary == "" {
		return nil, fmt.Errorf("traverse: could not extract boundary from Content-Type")
	}

	// Parse multipart response
	reader := bytes.NewReader(resp.Body())
	mr := multipart.NewReader(reader, boundary)

	var results []BatchResult

	for {
		part, err := mr.NextPart()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, fmt.Errorf("traverse: error reading batch response: %w", err)
		}

		// Read response part
		contentType := part.Header.Get("Content-Type")
		if strings.HasPrefix(contentType, "multipart/mixed") {
			// This is a changeset response
			csBoundary := extractBoundary(contentType)
			var csResults []BatchResult
			csResults, err = b.parseChangesetResponse(part, csBoundary)
			if err != nil {
				return nil, err
			}
			results = append(results, csResults...)
		} else {
			// Regular response
			var result BatchResult
			result, err = b.parseResponsePart(part)
			if err != nil {
				return nil, err
			}
			results = append(results, result)
		}

		if len(results) > batchMaxResponseCount {
			return nil, fmt.Errorf("traverse: batch response exceeds maximum of %d parts", batchMaxResponseCount)
		}
	}

	return results, nil
}

// parseChangesetResponse parses results from a changeset response.
//
// parseChangesetResponse extracts all operation results from a nested
// multipart/mixed changeset section. It is called recursively from
// [BatchRequest.parseMultipartResponse] to handle changesets.
//
// Returns a slice of [BatchResult] for operations in the changeset, or an error
// if parsing fails.
func (b *BatchRequest) parseChangesetResponse(part *multipart.Part, boundary string) ([]BatchResult, error) {
	var results []BatchResult

	reader := multipart.NewReader(part, boundary)

	for {
		part, err := reader.NextPart()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, fmt.Errorf("traverse: error reading changeset response: %w", err)
		}

		var result BatchResult
		result, err = b.parseResponsePart(part)
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}

	return results, nil
}

// parseResponsePart parses a single HTTP response part from a multipart/mixed section.
//
// parseResponsePart extracts the HTTP response embedded in a multipart MIME part.
// The part contains a complete HTTP response (status line, headers, body) that
// was embedded according to the RFC 2616 HTTP/1.1 message format.
//
// The method parses the status code, extracts all response headers into a map,
// and returns the response body as raw bytes.
//
// Returns a [BatchResult] containing the status code, headers, and body.
func (b *BatchRequest) parseResponsePart(part *multipart.Part) (BatchResult, error) {
	result := BatchResult{
		Headers: make(map[string]string),
	}

	// Read the part content, capped to batchMaxPartBodySize to prevent OOM.
	body, err := io.ReadAll(io.LimitReader(part, batchMaxPartBodySize))
	if err != nil {
		return result, fmt.Errorf("traverse: error reading response part: %w", err)
	}

	// Parse HTTP response from the part
	// Format: HTTP/1.1 200 OK\r\nHeaders\r\n\r\nBody
	lines := strings.Split(string(body), "\r\n")
	if len(lines) == 0 {
		return result, fmt.Errorf("traverse: invalid response part format")
	}

	// Parse status line: HTTP/1.1 200 OK
	statusLine := strings.Fields(lines[0])
	if len(statusLine) < 2 {
		return result, fmt.Errorf("traverse: invalid status line: %s", lines[0])
	}

	// Parse status code
	if _, err := fmt.Sscanf(statusLine[1], "%d", &result.StatusCode); err != nil {
		result.StatusCode = 500
	}

	// Parse headers and body
	bodyStart := 0
	for i, line := range lines {
		if line == "" {
			bodyStart = i + 1
			break
		}

		if i > 0 {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				result.Headers[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
			}
		}
	}

	// Extract body
	if bodyStart < len(lines) {
		result.Body = []byte(strings.Join(lines[bodyStart:], "\r\n"))
	}

	return result, nil
}

// extractBoundary extracts the boundary parameter from a multipart Content-Type header.
//
// extractBoundary parses a Content-Type header like "multipart/mixed; boundary=..."
// and returns the boundary value. It handles quoted boundaries (e.g., boundary="xyz")
// by removing quotes.
//
// Returns an empty string if no boundary parameter is found.
//
// Example:
//
//	boundary := extractBoundary("multipart/mixed; boundary=batch_12345")
//	// Returns: "batch_12345"
func extractBoundary(contentType string) string {
	parts := strings.Split(contentType, ";")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "boundary=") {
			boundary := strings.TrimPrefix(part, "boundary=")
			// Remove quotes if present
			boundary = strings.Trim(boundary, "\"")
			return boundary
		}
	}
	return ""
}

// release returns all header maps from operations back to the pool.
func (b *BatchRequest) release() {
	for i := range b.ops {
		releaseHeaders(b.ops[i].Headers)
		b.ops[i].Headers = nil
	}
	for _, cs := range b.changesets {
		for i := range cs.ops {
			releaseHeaders(cs.ops[i].Headers)
			cs.ops[i].Headers = nil
		}
	}
}

// jsonBatchRequest is the OData 4.01 JSON batch request envelope.
//
// Per OData 4.01 spec section 18.2, a JSON batch request is a single JSON
// document with a "requests" array, each item being a JSON batch request object.
type jsonBatchRequest struct {
	Requests []jsonBatchItem `json:"requests"`
}

// jsonBatchItem represents a single request within a JSON batch envelope.
type jsonBatchItem struct {
	ID             string            `json:"id"`
	Method         string            `json:"method"`
	URL            string            `json:"url"`
	Headers        map[string]string `json:"headers,omitempty"`
	Body           json.RawMessage   `json:"body,omitempty"`
	AtomicityGroup string            `json:"atomicityGroup,omitempty"`
	DependsOn      []string          `json:"dependsOn,omitempty"`
}

// jsonBatchResponse is the OData 4.01 JSON batch response envelope.
type jsonBatchResponse struct {
	Responses []jsonBatchResponseItem `json:"responses"`
}

// jsonBatchResponseItem is a single response item in a JSON batch response.
type jsonBatchResponseItem struct {
	ID      string            `json:"id"`
	Status  int               `json:"status"`
	Headers map[string]string `json:"headers,omitempty"`
	Body    json.RawMessage   `json:"body,omitempty"`
}

// ExecuteJSON sends the batch request using the OData 4.01 JSON batch format.
//
// ExecuteJSON is an alternative to [BatchRequest.Execute] that uses JSON encoding
// instead of multipart/mixed. The OData 4.01 JSON batch format is more compact,
// easier to debug, and supported by modern OData services.
//
// OData 4.01 spec reference: section 18 (Batch Requests and Responses).
//
// Operations within changesets are grouped using the "atomicityGroup" property.
// All operations in a changeset must succeed or all will be rolled back.
//
// Example:
//
//	resp, err := client.Batch().
//	    Get("Products", 1).
//	    Get("Products", 2).
//	    ExecuteJSON(ctx)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	for _, result := range resp.Results {
//	    fmt.Println(result.StatusCode, string(result.Body))
//	}
func (b *BatchRequest) ExecuteJSON(ctx context.Context) (*BatchResponse, error) {
	if b.currentCS != nil {
		b.EndChangeset()
	}

	reqBody, err := b.buildJSONBatchBody()
	if err != nil {
		return nil, fmt.Errorf("traverse: json batch build failed: %w", err)
	}

	req := b.client.http.Post("/$batch")
	req = req.WithContext(ctx)
	req = req.WithHeader("Content-Type", "application/json")
	req = req.WithHeader("OData-MaxVersion", "4.01")
	req = req.WithBody(reqBody)

	resp, err := b.client.http.Execute(req)
	if err != nil {
		return nil, fmt.Errorf("traverse: json batch execute failed: %w", err)
	}

	raw := resp.Body()

	var envelope jsonBatchResponse
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, fmt.Errorf("traverse: json batch parse response failed: %w", err)
	}

	results := make([]BatchResult, len(envelope.Responses))
	for i, item := range envelope.Responses {
		results[i] = BatchResult{
			StatusCode: item.Status,
			Headers:    item.Headers,
			Body:       item.Body,
		}
		if item.Status >= 400 {
			results[i].Err = fmt.Errorf("traverse: batch operation %s failed with status %d", item.ID, item.Status)
		}
	}

	b.release()
	return &BatchResponse{Results: results}, nil
}

// ExecuteJSONStream sends the batch using JSON format and streams results via a channel.
//
// ExecuteJSONStream is the streaming variant of [BatchRequest.ExecuteJSON].
// It reads the entire JSON response but emits results to the channel one by one
// to allow concurrent processing.
//
// Example:
//
//	for result := range batch.ExecuteJSONStream(ctx) {
//	    if result.Err != nil {
//	        log.Println("Operation failed:", result.Err)
//	        continue
//	    }
//	    fmt.Println(result.StatusCode, string(result.Body))
//	}
func (b *BatchRequest) ExecuteJSONStream(ctx context.Context) <-chan BatchResult {
	out := make(chan BatchResult, 8)

	go func() {
		defer close(out)

		if b.currentCS != nil {
			b.EndChangeset()
		}

		reqBody, err := b.buildJSONBatchBody()
		if err != nil {
			out <- BatchResult{Err: fmt.Errorf("traverse: json batch build failed: %w", err)}
			return
		}

		req := b.client.http.Post("/$batch")
		req = req.WithContext(ctx)
		req = req.WithHeader("Content-Type", "application/json")
		req = req.WithHeader("OData-MaxVersion", "4.01")
		req = req.WithBody(reqBody)

		resp, err := b.client.http.Execute(req)
		if err != nil {
			out <- BatchResult{Err: fmt.Errorf("traverse: json batch execute failed: %w", err)}
			return
		}

		raw := resp.Body()

		var envelope jsonBatchResponse
		if err := json.Unmarshal(raw, &envelope); err != nil {
			out <- BatchResult{Err: fmt.Errorf("traverse: json batch parse failed: %w", err)}
			return
		}

		for _, item := range envelope.Responses {
			result := BatchResult{
				StatusCode: item.Status,
				Headers:    item.Headers,
				Body:       item.Body,
			}
			if item.Status >= 400 {
				result.Err = fmt.Errorf("traverse: batch operation %s failed with status %d", item.ID, item.Status)
			}
			select {
			case out <- result:
			case <-ctx.Done():
				out <- BatchResult{Err: ctx.Err()}
				return
			}
		}
	}()

	return out
}

// buildJSONBatchBody constructs the OData 4.01 JSON batch request body.
//
// buildJSONBatchBody serializes all batch operations and changesets into a single
// JSON document per OData 4.01 spec section 18.2. Operations within changesets
// are assigned the same "atomicityGroup" identifier to signal atomicity.
func (b *BatchRequest) buildJSONBatchBody() ([]byte, error) {
	items := make([]jsonBatchItem, 0, len(b.ops)+10)
	counter := 0

	// Standalone operations (not in a changeset)
	for _, op := range b.ops {
		counter++
		item := jsonBatchItem{
			ID:     strconv.Itoa(counter),
			Method: op.Method,
			URL:    op.URL,
		}
		if len(op.Headers) > 0 {
			item.Headers = op.Headers
		}
		if len(op.Body) > 0 {
			item.Body = op.Body
		}
		items = append(items, item)
	}

	// Changeset operations - each changeset becomes an atomicityGroup
	for _, cs := range b.changesets {
		for _, op := range cs.ops {
			counter++
			item := jsonBatchItem{
				ID:             strconv.Itoa(counter),
				Method:         op.Method,
				URL:            op.URL,
				AtomicityGroup: cs.id,
			}
			if len(op.Headers) > 0 {
				item.Headers = op.Headers
			}
			if len(op.Body) > 0 {
				item.Body = op.Body
			}
			items = append(items, item)
		}
	}

	return json.Marshal(jsonBatchRequest{Requests: items})
}
