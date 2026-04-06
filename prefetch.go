package traverse

import (
	"context"
	"fmt"
)

// prefetchedPage carries a fetched page or the error that occurred while fetching it.
type prefetchedPage struct {
	page    *Page
	pageNum int
	err     error
}

// doPrefetchPages is the prefetch-enabled variant of doStreamPages.
//
// doPrefetchPages starts a background goroutine that fetches pages ahead of
// consumption. The page channel is buffered to bufferPages (clamped 1-3), allowing
// up to that many pages to be ready before the consumer has finished the current one.
// This reduces end-to-end latency for paginated reads by overlapping network I/O
// with the caller's record processing.
//
// Goroutine safety: a child context derived from ctx is used so that stopping
// iteration early (or ctx cancellation) reliably terminates the prefetch goroutine.
func (q *QueryBuilder) doPrefetchPages(ctx context.Context, out chan<- Result[map[string]interface{}]) {
	if q.lastError != nil {
		out <- Result[map[string]interface{}]{Err: q.lastError}
		return
	}

	bufSize := q.prefetchPages
	if bufSize < 1 {
		bufSize = 1
	}
	if bufSize > 3 {
		bufSize = 3
	}

	// fetchCtx lets us cancel the prefetch goroutine when the consumer stops early.
	fetchCtx, cancelFetch := context.WithCancel(ctx)
	defer cancelFetch()

	pageCh := make(chan prefetchedPage, bufSize)

	go func() {
		defer close(pageCh)

		pageNum := 1
		nextLink := q.buildURL()

		for nextLink != "" {
			select {
			case <-fetchCtx.Done():
				return
			default:
			}

			page, err := q.fetchPageStreamed(fetchCtx, nextLink)
			if err != nil {
				select {
				case pageCh <- prefetchedPage{
					err: fmt.Errorf("traverse: failed to fetch page %d: %w", pageNum, err),
				}:
				case <-fetchCtx.Done():
				}
				return
			}

			select {
			case pageCh <- prefetchedPage{page: page, pageNum: pageNum}:
			case <-fetchCtx.Done():
				return
			}

			nextLink = page.NextLink
			pageNum++
		}
	}()

	for p := range pageCh {
		if p.err != nil {
			out <- Result[map[string]interface{}]{Err: p.err}
			return
		}

		for i, record := range p.page.Value {
			select {
			case <-ctx.Done():
				returnPageToPool(p.page, i)
				out <- Result[map[string]interface{}]{Err: ctx.Err()}
				return
			case out <- Result[map[string]interface{}]{
				Value: copyMapDeep(record),
				Page:  p.pageNum,
				Index: i,
			}:
			}
		}

		returnPageToPool(p.page, len(p.page.Value))
	}

	// Propagate context cancellation that caused the prefetch goroutine to stop early.
	if err := ctx.Err(); err != nil {
		out <- Result[map[string]interface{}]{Err: err}
	}
}
