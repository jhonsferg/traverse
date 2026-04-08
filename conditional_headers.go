package traverse

import (
	"strings"
	"time"
)

// sanitizeHeaderValue strips CRLF characters that could allow HTTP response splitting.
func sanitizeHeaderValue(v string) string {
	v = strings.ReplaceAll(v, "\r", "")
	v = strings.ReplaceAll(v, "\n", "")
	return v
}

// IfMatch sets the If-Match conditional request header on the query.
// The server will only process the request if the entity's current ETag matches.
// Returns HTTP 412 Precondition Failed if the ETags do not match.
//
// Commonly used to guard updates and deletes against concurrent modifications:
//
//	page, err := client.From("Orders").
//	    IfMatch(`W/"abc123"`).
//	    Page(ctx)
func (q *QueryBuilder) IfMatch(etag string) *QueryBuilder {
	if q.conditionalHeaders == nil {
		q.conditionalHeaders = make(map[string]string, 2)
	}
	q.conditionalHeaders["If-Match"] = sanitizeHeaderValue(etag)
	return q
}

// IfNoneMatch sets the If-None-Match conditional request header on the query.
// The server will only return data if none of the given ETags match the current entity.
// Used for cache validation: the server returns 304 Not Modified if the ETag matches.
//
//	page, err := client.From("Products").
//	    IfNoneMatch(`"xyz789"`).
//	    Page(ctx)
func (q *QueryBuilder) IfNoneMatch(etag string) *QueryBuilder {
	if q.conditionalHeaders == nil {
		q.conditionalHeaders = make(map[string]string, 2)
	}
	q.conditionalHeaders["If-None-Match"] = sanitizeHeaderValue(etag)
	return q
}

// IfModifiedSince sets the If-Modified-Since conditional request header.
// The server will only return data if the resource has been modified after t.
// Returns HTTP 304 Not Modified if unchanged.
//
//	since := time.Now().Add(-24 * time.Hour)
//	page, err := client.From("Orders").
//	    IfModifiedSince(since).
//	    Page(ctx)
func (q *QueryBuilder) IfModifiedSince(t time.Time) *QueryBuilder {
	if q.conditionalHeaders == nil {
		q.conditionalHeaders = make(map[string]string, 2)
	}
	q.conditionalHeaders["If-Modified-Since"] = t.UTC().Format(time.RFC1123)
	return q
}

// IfUnmodifiedSince sets the If-Unmodified-Since conditional request header.
// The server will only process the request if the resource has not been
// modified since t. Returns HTTP 412 Precondition Failed if it was modified.
//
//	checkpoint := time.Now().Add(-1 * time.Hour)
//	page, err := client.From("Products").
//	    IfUnmodifiedSince(checkpoint).
//	    Page(ctx)
func (q *QueryBuilder) IfUnmodifiedSince(t time.Time) *QueryBuilder {
	if q.conditionalHeaders == nil {
		q.conditionalHeaders = make(map[string]string, 2)
	}
	q.conditionalHeaders["If-Unmodified-Since"] = t.UTC().Format(time.RFC1123)
	return q
}
