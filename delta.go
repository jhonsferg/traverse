package traverse

import (
	"context"
	"fmt"
)

// DeltaSync manages incremental synchronization using OData delta links (OData v4).
//
// DeltaSync enables efficient synchronization of large datasets by supporting incremental
// updates. On the first sync, all records are read. On subsequent syncs, only changes
// (modifications and deletions) are returned using a delta token.
//
// Delta sync uses the $deltatoken query parameter to mark a point in time.
// The server returns only records that have changed since that token was issued.
// Deleted records are marked with the @removed annotation.
//
// Typical workflow:
//
//	1. Initialize: ds := client.NewDeltaSync("Customers")
//	2. Full sync: records, token, err := ds.Full(ctx)   // Read all records
//	3. Save token for later
//	4. Incremental: changes, newToken, err := ds.Incremental(ctx, token)  // Only changes
//	5. Process changes, save new token, repeat step 4
//
// This approach significantly reduces bandwidth and latency for large datasets
// that change infrequently.
type DeltaSync struct {
	client    *Client
	entitySet string
	token     string // Current delta token
}

// NewDeltaSync creates a new delta sync handler for an entity set.
//
// NewDeltaSync initializes a [DeltaSync] for incremental synchronization of the
// specified entity set. The delta token is initially empty; call [DeltaSync.Full]
// first to obtain an initial token.
//
// Returns a DeltaSync instance ready for use.
//
// Example:
//
//	ds := client.NewDeltaSync("Customers")
func (c *Client) NewDeltaSync(entitySet string) *DeltaSync {
	return &DeltaSync{
		client:    c,
		entitySet: entitySet,
	}
}

// Full performs a complete (initial) synchronization, returning all records.
//
// Full returns a channel streaming all records from the entity set and automatically
// extracts the delta token from the server response metadata (DeltaLink). This token
// can be used for subsequent incremental syncs via [DeltaSync.Incremental].
//
// The method also stores the token internally for convenience, so you can call
// [DeltaSync.Incremental] without providing the token explicitly.
//
// The bufferSize parameter controls the result channel capacity (default 256).
// For large record sizes or high network latency, increase this value to reduce blocking.
//
// Returns:
//   - A receive-only channel of Result items containing all records
//   - The delta token for use in subsequent incremental syncs
//   - An error if the query fails
//
// Example:
//
//	records, token, err := ds.Full(ctx)
//	if err != nil {
//		log.Fatal(err)
//	}
//	for result := range records {
//		if result.Err != nil {
//			log.Println("Error:", result.Err)
//			continue
//		}
//		processRecord(result.Value)
//	}
//	// Save token for next sync: ds.Incremental(ctx, token)
func (d *DeltaSync) Full(ctx context.Context, bufferSize ...int) (<-chan Result[map[string]interface{}], string, error) {
	q := d.client.From(d.entitySet).WithCount()

	buffer := 256
	if len(bufferSize) > 0 && bufferSize[0] > 0 {
		buffer = bufferSize[0]
	}

	// Stream all records
	out := make(chan Result[map[string]interface{}], buffer)

	go func() {
		defer close(out)

		// Track the delta token from the first page's metadata
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

			// Fetch page with delta link extraction
			page, err := q.fetchPageStreamed(ctx, nextLink)
			if err != nil {
				out <- Result[map[string]interface{}]{
					Err: fmt.Errorf("traverse: failed to fetch page %d: %w", pageNum, err),
				}
				return
			}

			// Extract delta token from metadata if available
			if page.DeltaLink != "" {
				d.token = extractDeltaToken(page.DeltaLink)
			}

			// Stream records from this page
			for i, record := range page.Value {
				select {
				case <-ctx.Done():
					out <- Result[map[string]interface{}]{
						Err: ctx.Err(),
					}
					return
				case out <- Result[map[string]interface{}]{
					Value: record,
					Page:  pageNum,
					Index: i,
				}:
					// Record sent successfully
				}
			}

			// Check for next page
			nextLink = page.NextLink
			pageNum++
		}
	}()

	// Use stored token if available
	if d.token != "" {
		return out, d.token, nil
	}

	return out, "", nil
}

// Incremental performs an incremental sync using a delta token.
//
// Incremental returns only records that have changed or been deleted since the
// provided delta token was issued. Changes include both new records and modifications
// to existing records. Deleted records are marked with Removed=true and include
// a Reason (typically "changed" or "deleted").
//
// The token parameter specifies which delta point to start from. If empty, the
// internally stored token (from previous [DeltaSync.Full] or [DeltaSync.Incremental])
// is used. Returns an error if no token is available.
//
// A new delta token is extracted from the response and can be used for the next
// incremental sync. It's automatically stored internally as well.
//
// The bufferSize parameter controls the result channel capacity (default 256).
//
// Returns:
//   - A receive-only channel of [DeltaResult] items containing changed records
//   - The new delta token for use in subsequent incremental syncs
//   - An error if the query fails or no token is available
//
// Example:
//
//	// First, get initial token from Full sync
//	_, token, _ := ds.Full(ctx)
//
//	// Later, sync only changes
//	for result := range ds.Incremental(ctx, token) {
//		if result.Removed {
//			deleteRecord(result.Value)
//		} else {
//			updateRecord(result.Value)
//		}
//	}
func (d *DeltaSync) Incremental(ctx context.Context, token string, bufferSize ...int) (<-chan DeltaResult, string, error) {
	if token == "" {
		token = d.token
	}

	if token == "" {
		return nil, "", fmt.Errorf("traverse: no delta token available, run Full() first")
	}

	q := d.client.From(d.entitySet).WithDeltaToken(token)

	buffer := 256
	if len(bufferSize) > 0 && bufferSize[0] > 0 {
		buffer = bufferSize[0]
	}

	out := make(chan DeltaResult, buffer)
	var newToken string

	go func() {
		defer close(out)

		pageNum := 1
		nextLink := q.buildURL()

		for nextLink != "" {
			select {
			case <-ctx.Done():
				out <- DeltaResult{
					Err: ctx.Err(),
				}
				return
			default:
			}

			// Stream delta records
			page, err := q.fetchPageStreamed(ctx, nextLink)
			if err != nil {
				out <- DeltaResult{
					Err: fmt.Errorf("traverse: failed to fetch page %d: %w", pageNum, err),
				}
				return
			}

			// Extract delta token from metadata
			if page.DeltaLink != "" {
				newToken = extractDeltaToken(page.DeltaLink)
				d.token = newToken
			}

			// Stream records with removed/modified tracking
			for _, record := range page.Value {
				removed := false
				reason := ""

				// Check for @removed annotation
				if removedObj, ok := record["@removed"]; ok {
					removed = true
					// Extract reason if available
					if removedMap, ok := removedObj.(map[string]interface{}); ok {
						if r, ok := removedMap["reason"].(string); ok {
							reason = r
						}
					}
					// Remove the annotation from the record
					delete(record, "@removed")
				}

				select {
				case <-ctx.Done():
					out <- DeltaResult{
						Err: ctx.Err(),
					}
					return
				case out <- DeltaResult{
					Value:   record,
					Removed: removed,
					Reason:  reason,
				}:
					// Result sent successfully
				}
			}

			// Check for next page
			nextLink = page.NextLink
			pageNum++
		}
	}()

	// Return the new token (could be same as input on subsequent pages)
	if newToken == "" {
		newToken = token
	}

	return out, newToken, nil
}

// extractDeltaToken extracts the delta token from a delta link URL.
// Delta links typically look like: /path?$deltatoken='token_value'
func extractDeltaToken(deltaLink string) string {
	// Find $deltatoken parameter
	start := 0
	for i := 0; i < len(deltaLink); i++ {
		if i+11 <= len(deltaLink) && deltaLink[i:i+11] == "$deltatoken" {
			start = i + 11
			break
		}
	}

	if start == 0 {
		return ""
	}

	// Find the value after '='
	eqIdx := -1
	for i := start; i < len(deltaLink); i++ {
		if deltaLink[i] == '=' {
			eqIdx = i
			break
		}
	}

	if eqIdx == -1 {
		return ""
	}

	// Extract quoted value: 'token_value'
	startIdx := eqIdx + 1
	if startIdx >= len(deltaLink) {
		return ""
	}

	// Skip leading quote if present
	if deltaLink[startIdx] == '\'' {
		startIdx++
	}

	// Find ending quote
	endIdx := startIdx
	for i := startIdx; i < len(deltaLink); i++ {
		if deltaLink[i] == '\'' || deltaLink[i] == '&' {
			endIdx = i
			break
		}
		if i == len(deltaLink)-1 {
			endIdx = i + 1
		}
	}

	if endIdx > startIdx {
		return deltaLink[startIdx:endIdx]
	}

	return ""
}

// SetToken sets the current delta token.
func (d *DeltaSync) SetToken(token string) {
	d.token = token
}

// Token returns the current delta token.
func (d *DeltaSync) Token() string {
	return d.token
}

// DeltaSyncAs is the generic version of delta sync.
type DeltaSyncAs[T any] struct {
	delta *DeltaSync
}

// NewDeltaSyncAs creates a typed delta sync handler.
func NewDeltaSyncAs[T any](c *Client, entitySet string) *DeltaSyncAs[T] {
	return &DeltaSyncAs[T]{
		delta: c.NewDeltaSync(entitySet),
	}
}

// Full performs a complete sync with type T.
func (d *DeltaSyncAs[T]) Full(ctx context.Context, bufferSize ...int) (<-chan Result[T], string, error) {
	buffer := 256
	if len(bufferSize) > 0 && bufferSize[0] > 0 {
		buffer = bufferSize[0]
	}

	out := make(chan Result[T], buffer)
	mapChan, token, err := d.delta.Full(ctx, buffer)

	if err != nil {
		close(out)
		return out, "", err
	}

	go func() {
		defer close(out)
		for result := range mapChan {
			if result.Err != nil {
				out <- Result[T]{
					Err: result.Err,
				}
				continue
			}

			// Convert result.Value to T using mapToStruct
			value, err := mapToStruct[T](result.Value)
			if err != nil {
				out <- Result[T]{
					Err: err,
				}
				continue
			}

			out <- Result[T]{
				Value: value,
				Page:  result.Page,
				Index: result.Index,
			}
		}
	}()

	return out, token, nil
}

// Incremental performs an incremental sync with type T.
func (d *DeltaSyncAs[T]) Incremental(ctx context.Context, token string, bufferSize ...int) (<-chan DeltaResultAs[T], string, error) {
	buffer := 256
	if len(bufferSize) > 0 && bufferSize[0] > 0 {
		buffer = bufferSize[0]
	}

	out := make(chan DeltaResultAs[T], buffer)
	deltaChan, newToken, err := d.delta.Incremental(ctx, token, buffer)

	if err != nil {
		close(out)
		return out, "", err
	}

	go func() {
		defer close(out)
		for result := range deltaChan {
			if result.Err != nil {
				out <- DeltaResultAs[T]{
					Err: result.Err,
				}
				continue
			}

			// Convert result.Value to T using mapToStruct
			value, err := mapToStruct[T](result.Value)
			if err != nil {
				out <- DeltaResultAs[T]{
					Err: err,
				}
				continue
			}

			out <- DeltaResultAs[T]{
				Value:   value,
				Removed: result.Removed,
				Reason:  result.Reason,
			}
		}
	}()

	return out, newToken, nil
}

// DeltaResultAs is the typed version of DeltaResult.
type DeltaResultAs[T any] struct {
	Value   T
	Removed bool
	Reason  string
	Err     error
}
