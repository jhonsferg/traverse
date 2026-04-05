package traverse

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// StreamProperty fetches a named binary stream property and returns its content
// as an io.ReadCloser. The caller is responsible for closing the returned reader.
//
// StreamProperty issues a GET request to EntitySet/PropertyName with the
// Accept: application/octet-stream header. The response body is returned
// directly without buffering, making this suitable for large binary properties.
//
// The entitySet on the QueryBuilder should include the entity key, e.g.:
//
//	reader, err := client.From("Products(42)").StreamProperty(ctx, "Photo")
//	if err != nil { ... }
//	defer reader.Close()
//	io.Copy(file, reader)
func (q *QueryBuilder) StreamProperty(ctx context.Context, propertyName string) (io.ReadCloser, error) {
	path := q.entitySet + "/" + propertyName

	req := q.client.http.Get(path)
	req = req.WithContext(ctx)
	req = req.WithHeader("Accept", "application/octet-stream")

	stream, err := q.client.http.ExecuteStream(req)
	if err != nil {
		return nil, fmt.Errorf("traverse: StreamProperty failed: %w", err)
	}
	if stream.StatusCode != 200 {
		_ = stream.Body.Close()
		return nil, fmt.Errorf("traverse: StreamProperty returned status %d", stream.StatusCode)
	}
	return stream.Body, nil
}

// StreamPropertySize issues a HEAD request to determine the byte size of a
// named stream property without downloading its content.
//
// Returns the value of the Content-Length response header. Returns -1 if the
// server does not include a Content-Length header.
//
//	size, err := client.From("Products(42)").StreamPropertySize(ctx, "Photo")
func (q *QueryBuilder) StreamPropertySize(ctx context.Context, propertyName string) (int64, error) {
	path := q.entitySet + "/" + propertyName

	req := q.client.http.Head(path)
	req = req.WithContext(ctx)
	req = req.WithHeader("Accept", "application/octet-stream")

	resp, err := q.client.http.Execute(req)
	if err != nil {
		return 0, fmt.Errorf("traverse: StreamPropertySize failed: %w", err)
	}
	if resp.StatusCode != 200 {
		return 0, fmt.Errorf("traverse: StreamPropertySize returned status %d", resp.StatusCode)
	}

	cl := strings.TrimSpace(resp.Headers.Get("Content-Length"))
	if cl == "" {
		return -1, nil
	}
	size, err := strconv.ParseInt(cl, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("traverse: failed to parse Content-Length %q: %w", cl, err)
	}
	return size, nil
}
