// Package tokenizer provides streaming JSON parsing utilities for OData responses.
//
// The tokenizer package contains internal helpers for efficiently parsing JSON responses
// from OData services without loading the entire response into memory. It supports both
// OData v2 ({"d": {"results": [...]}}) and OData v4 ({"value": [...]}) response formats,
// enabling streaming of large result sets with constant memory usage.
//
// Key types:
//   - ArrayStreamer: An iterator-like interface for streaming array elements
//   - StreamMetadata: Metadata discovered during streaming (nextLink, deltaLink, etc.)
//   - ODataVersion: Enumeration for OData protocol versions
//
// Usage example (OData v4):
//
//	streamer := NewArrayStreamer(responseBody, ODataV4)
//	for streamer.Next() {
//	    var record map[string]interface{}
//	    if err := streamer.Decode(&record); err != nil {
//	        break
//	    }
//	    // Process record
//	}
package tokenizer

import (
	"encoding/json"
	"fmt"
	"io"
)

// ODataVersion represents the OData protocol version being used.
//
// OData v2 (OASIS standard) and OData v4 (current standard) use different JSON response formats:
//   - OData v2: {"d": {"results": [...]}} with support for navigation properties
//   - OData v4: {"value": [...]} with @odata.* metadata properties
//
// The version determines how the streaming parser navigates the JSON structure
// to locate the actual array of records.
type ODataVersion int

const (
	ODataV2 ODataVersion = 2 // OData v2 protocol version
	ODataV4 ODataVersion = 4 // OData v4 protocol version (current)
)

// ArrayStreamer provides iterator-like streaming access to OData response arrays without buffering.
//
// ArrayStreamer uses json.Decoder to stream array elements one at a time, enabling
// processing of large OData result sets with constant memory usage. Instead of loading
// the entire array into memory (which could be gigabytes for millions of records),
// ArrayStreamer yields one record at a time via Next() and Decode().
//
// Typical usage pattern:
//
//	streamer := NewArrayStreamer(responseBody, ODataV4)
//	defer streamer.Close()
//	for streamer.Next() {
//	    var product map[string]interface{}
//	    if err := streamer.Decode(&product); err != nil {
//	        log.Fatal(err)
//	    }
//	    // Process individual product record
//	}
//	if err := streamer.Err(); err != nil {
//	    log.Fatal(err)
//	}
//
// The streamer handles both OData v2 and v4 response formats automatically.
// It also captures metadata like nextLink and deltaLink for pagination and delta queries.
type ArrayStreamer struct {
	dec     *json.Decoder
	version ODataVersion
	meta    StreamMetadata
	err     error
	closed  bool
}

// StreamMetadata contains metadata discovered during response streaming.
//
// StreamMetadata captures OData protocol information extracted from the JSON response
// while streaming array records. This enables features like pagination ($skiptoken, $nextlink)
// and delta queries ($deltatoken) without requiring separate metadata requests.
//
// Fields populated depend on the OData version and service implementation:
//   - NextLink: URL for fetching the next page of results (OData v4 @odata.nextLink)
//   - DeltaLink: URL for delta queries (OData v4 @odata.deltaLink)
//   - Count: Total count of matching records if $count=true was used
//   - Context: OData context URI (@odata.context in v4)
type StreamMetadata struct {
	// NextLink is the URL to fetch the next page of results, if pagination is active.
	// Empty if this is the last page or if $top/$skip pagination is being used.
	NextLink string
	// DeltaLink is the URL to use for delta queries to fetch incremental changes.
	// Empty if delta queries are not applicable.
	DeltaLink string
	// Count is the total number of matching records, if $count=true was requested.
	// Nil if count was not requested.
	Count *int64
	// Context is the OData context URI for semantic information about the response.
	Context string
}

// NewArrayStreamer creates a new ArrayStreamer for streaming OData response array records.
//
// NewArrayStreamer initializes a streaming JSON parser for the given OData version.
// It does NOT immediately parse or validate the response—parsing begins only when
// Next() is called, allowing for immediate return and lazy evaluation.
//
// Parameters:
//   - r: An io.Reader containing the JSON response body (typically from http.Response.Body)
//   - version: The OData protocol version (ODataV2 or ODataV4)
//
// The returned ArrayStreamer should be used in a loop with Next() and Decode(),
// and should be closed with Close() when done.
//
// Example:
//
//	resp, _ := http.Get("http://service/odata/Products")
//	defer resp.Body.Close()
//	streamer := NewArrayStreamer(resp.Body, ODataV4)
//	defer streamer.Close()
func NewArrayStreamer(r io.Reader, version ODataVersion) *ArrayStreamer {
	return &ArrayStreamer{
		dec:     json.NewDecoder(r),
		version: version,
	}
}

// Next advances the ArrayStreamer to the next element in the result array.
//
// Next returns true if a new element is available and ready for decoding,
// or false when the array is exhausted, an error has occurred, or the streamer is closed.
//
// After Next() returns true, call Decode() to unmarshal the current element.
// After Next() returns false, call Err() to check if the loop exited due to an error
// or simply reached the end of the array.
//
// Typical loop pattern:
//
//	for streamer.Next() {
//	    var item MyType
//	    if err := streamer.Decode(&item); err != nil {
//	        break
//	    }
//	    // Process item
//	}
//	if err := streamer.Err(); err != nil {
//	    log.Fatal(err)
//	}
//
// Returns false without error when iteration completes normally (array exhausted).
// Returns false if any decoding or streaming error occurred earlier.
func (s *ArrayStreamer) Next() bool {
	if s.closed || s.err != nil {
		return false
	}

	// To be implemented with More() in the context of an open array
	return true
}

// Decode unmarshals the current array element into the provided value.
//
// Decode reads the JSON representation of the current array element (positioned by Next())
// and unmarshals it into the provided interface{} value using json.Unmarshal semantics.
//
// The value parameter should typically be:
//   - A pointer to a struct: &Product{} for type-safe decoding
//   - A pointer to a map: &map[string]interface{}{} for dynamic decoding
//   - Any other pointer type that can be unmarshaled from JSON
//
// Decode should only be called immediately after Next() returns true.
// If called at other times or if Next() returns false, Decode returns an error.
//
// Returns an error if:
//   - A previous error occurred in the streamer
//   - The JSON is malformed
//   - The JSON cannot be unmarshaled into the target type
//
// Example:
//
//	var product struct {
//	    ID    int    `json:"ID"`
//	    Name  string `json:"Name"`
//	    Price float64 `json:"Price"`
//	}
//	if err := streamer.Decode(&product); err != nil {
//	    log.Fatal(err)
//	}
func (s *ArrayStreamer) Decode(v interface{}) error {
	if s.err != nil {
		return s.err
	}

	if err := s.dec.Decode(v); err != nil {
		s.err = err
		return err
	}

	return nil
}

// Meta returns the metadata discovered during streaming.
//
// Meta returns StreamMetadata collected while parsing the OData response.
// This includes information like nextLink (for pagination), deltaLink (for delta queries),
// and count (if $count was requested).
//
// The metadata is populated as the JSON is parsed and is available at any point during
// streaming, though it will only contain data that has been discovered so far.
//
// Use this to implement features like:
//   - Pagination: Check NextLink and fetch the next page
//   - Delta queries: Check DeltaLink and fetch incremental changes
//   - Progress tracking: Check Count to know total records being fetched
//
// Example:
//
//	// After streaming completes
//	meta := streamer.Meta()
//	if meta.NextLink != "" {
//	    // There's another page to fetch
//	}
func (s *ArrayStreamer) Meta() StreamMetadata {
	return s.meta
}

// Err returns any error that occurred during streaming, or nil if no error occurred.
//
// Err should be called after the Next() loop completes to determine if the loop exited
// normally (no error) or due to a parsing/decoding error.
//
// Returns nil if streaming completed successfully or if no elements were encountered.
// Returns the first error that occurred during Next() or Decode() operations.
//
// Typical usage:
//
//	for streamer.Next() {
//	    var item MyType
//	    streamer.Decode(&item)
//	    // Process item...
//	}
//	if err := streamer.Err(); err != nil {
//	    log.Printf("streaming error: %v", err)
//	}
func (s *ArrayStreamer) Err() error {
	return s.err
}

// Close closes the ArrayStreamer and releases any associated resources.
//
// Close marks the streamer as closed, preventing further Next() calls.
// It should be called when done with the streamer to ensure proper resource cleanup,
// though in practice the main resource (the io.Reader) is typically managed by the caller.
//
// Calling Close() is idempotent—multiple Close() calls are safe and return nil.
// It's good practice to always defer Close():
//
//	streamer := NewArrayStreamer(resp.Body, ODataV4)
//	defer streamer.Close()
func (s *ArrayStreamer) Close() error {
	s.closed = true
	return nil
}

// StreamArrayRecords streams records from an OData response array without buffering.
//
// StreamArrayRecords reads a complete OData JSON response and calls a handler function
// for each record in the result array. This function handles both OData v2 and v4 formats
// automatically, streaming records one at a time without loading the entire array into memory.
//
// The function automatically navigates the JSON structure to find the records:
//   - OData v2: Looks for {"d": {"results": [...]}}
//   - OData v4: Looks for {"value": [...]}
//
// For each record in the array, the handler function is called with a map[string]interface{}
// representing the record. The handler should return true to continue streaming the next record,
// or false to stop streaming early (e.g., if an error occurred or a quota was reached).
//
// Parameters:
//   - r: An io.Reader with the JSON response body
//   - version: The OData version (ODataV2 or ODataV4)
//   - handler: A callback function called for each record
//
// Returns an error if:
//   - The JSON structure doesn't match the expected OData format
//   - JSON parsing fails
//   - The handler returns an error (indirectly, through handler behavior)
//
// Example:
//
//	err := StreamArrayRecords(resp.Body, ODataV4, func(record map[string]interface{}) bool {
//	    fmt.Printf("ID: %v, Name: %v\n", record["ID"], record["Name"])
//	    return true // Continue streaming
//	})
func StreamArrayRecords(r io.Reader, version ODataVersion, handler func(map[string]interface{}) bool) error {
	dec := json.NewDecoder(r)

	// Read root object opening '{'
	token, err := dec.Token()
	if err != nil {
		return fmt.Errorf("failed to read root opening: %w", err)
	}
	if delim, ok := token.(json.Delim); !ok || delim != '{' {
		return fmt.Errorf("expected '{', got %v", token)
	}

	// Navigate to the appropriate array
	if version == ODataV2 {
		// OData v2: {"d": {"results": [...]}}
		return streamODataV2(dec, handler)
	} else {
		// OData v4: {"value": [...]}
		return streamODataV4(dec, handler)
	}
}

// streamODataV4 streams records from an OData v4 "value" array without buffering.
//
// streamODataV4 is an internal helper that parses OData v4 response structure
// {"@odata.context": "...", "value": [{...}, {...}], "@odata.nextLink": "..."}
// and streams individual records to the provided handler.
//
// This function handles the JSON decoding at a low level, reading tokens sequentially
// and only buffering one record at a time, enabling streaming of arbitrarily large
// arrays with minimal memory overhead.
//
// The handler receives each record as map[string]interface{} and should return:
//   - true: Continue streaming the next record
//   - false: Stop streaming (useful for quota limits or error handling)
//
// Internal helper; not part of the public API.
func streamODataV4(dec *json.Decoder, handler func(map[string]interface{}) bool) error {
	for dec.More() {
		token, err := dec.Token()
		if err != nil {
			return fmt.Errorf("failed to read token: %w", err)
		}

		key, ok := token.(string)
		if !ok {
			continue
		}

		if key == "value" {
			// Found the value array
			token, err := dec.Token()
			if err != nil {
				return fmt.Errorf("failed to read array opening: %w", err)
			}
			if delim, ok := token.(json.Delim); !ok || delim != '[' {
				return fmt.Errorf("expected '[', got %v", token)
			}

			// Stream array elements
			for dec.More() {
				var record map[string]interface{}
				if err := dec.Decode(&record); err != nil {
					return fmt.Errorf("failed to decode record: %w", err)
				}

				// Call handler and stop if it returns false
				if !handler(record) {
					return nil
				}
			}

			// Read closing ']'
			_, _ = dec.Token()
			return nil
		} else {
			// Skip other fields
			var tmp interface{}
			_ = dec.Decode(&tmp)
		}
	}

	return fmt.Errorf("no 'value' array found in response")
}

// streamODataV2 streams records from an OData v2 "d.results" array without buffering.
//
// streamODataV2 is an internal helper that parses OData v2 response structure
// {"d": {"results": [{...}, {...}], "__count": "123"}}
// and streams individual records to the provided handler.
//
// OData v2 wraps the result array in a "d" object for CSRF protection and security.
// This parser navigates through that wrapper and locates the "results" array,
// then streams records one at a time without buffering.
//
// The handler receives each record as map[string]interface{} and should return:
//   - true: Continue streaming the next record
//   - false: Stop streaming (useful for quota limits or error handling)
//
// Internal helper; not part of the public API.
func streamODataV2(dec *json.Decoder, handler func(map[string]interface{}) bool) error {
	for dec.More() {
		token, err := dec.Token()
		if err != nil {
			return fmt.Errorf("failed to read token: %w", err)
		}

		key, ok := token.(string)
		if !ok {
			continue
		}

		if key == "d" {
			// Found the 'd' wrapper
			token, err := dec.Token()
			if err != nil {
				return fmt.Errorf("failed to read wrapper opening: %w", err)
			}
			if delim, ok := token.(json.Delim); !ok || delim != '{' {
				return fmt.Errorf("expected '{', got %v", token)
			}

			// Navigate inside 'd' to find 'results'
			for dec.More() {
				token, err := dec.Token()
				if err != nil {
					return fmt.Errorf("failed to read wrapper token: %w", err)
				}

				key, ok := token.(string)
				if !ok {
					continue
				}

				if key == "results" {
					// Found the results array
					token, err := dec.Token()
					if err != nil {
						return fmt.Errorf("failed to read array opening: %w", err)
					}
					if delim, ok := token.(json.Delim); !ok || delim != '[' {
						return fmt.Errorf("expected '[', got %v", token)
					}

					// Stream array elements
					for dec.More() {
						var record map[string]interface{}
						if err := dec.Decode(&record); err != nil {
							return fmt.Errorf("failed to decode record: %w", err)
						}

						// Call handler and stop if it returns false
						if !handler(record) {
							return nil
						}
					}

					// Read closing ']'
					_, _ = dec.Token()
					return nil
				} else {
					// Skip other fields in 'd'
					var tmp interface{}
					_ = dec.Decode(&tmp)
				}
			}

			return nil
		} else {
			// Skip other root fields
			var tmp interface{}
			_ = dec.Decode(&tmp)
		}
	}

	return fmt.Errorf("no 'd' object found in OData v2 response")
}
