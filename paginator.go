package traverse

import (
	"context"
	"encoding/json"
	"fmt"
)

// Paginator provides a typed, iterator-style interface for paging through
// an OData entity set one page at a time. It automatically follows
// @odata.nextLink (OData v4) or __next (OData v2) links between pages.
//
// Obtain a Paginator via [NewPaginator] or [NewPaginatorJSON].
//
//	p := traverse.NewPaginator[Product](client.From("Products").Top(50))
//	for p.HasMorePages() {
//	    items, err := p.NextPage(ctx)
//	    if err != nil { break }
//	    for _, item := range items {
//	        process(item)
//	    }
//	}
//
// A Paginator is NOT safe for concurrent use; use separate Paginators in
// separate goroutines.
type Paginator[T any] struct {
	query    *QueryBuilder
	nextLink string
	done     bool
	// unmarshal converts a raw JSON value to T.
	unmarshal func(json.RawMessage) (T, error)
}

// NewPaginator creates a typed Paginator that deserialises each entity via
// [json.Unmarshal] into T. The query is executed lazily - no network call is
// made until the first call to [Paginator.NextPage].
//
//	type Product struct {
//	    ID   int    `json:"ProductID"`
//	    Name string `json:"Name"`
//	}
//	p := traverse.NewPaginator[Product](client.From("Products").Top(100))
func NewPaginator[T any](query *QueryBuilder) *Paginator[T] {
	return &Paginator[T]{
		query: query,
		unmarshal: func(raw json.RawMessage) (T, error) {
			var v T
			err := json.Unmarshal(raw, &v)
			return v, err
		},
	}
}

// NewPaginatorWithDecoder creates a Paginator that uses a custom decode
// function to convert each raw JSON entity into T. Use this when you need
// non-standard unmarshalling (e.g. case-insensitive keys, custom types).
//
//	p := traverse.NewPaginatorWithDecoder[Product](
//	    client.From("Products"),
//	    func(raw json.RawMessage) (Product, error) {
//	        var p Product
//	        return p, json.Unmarshal(raw, &p)
//	    },
//	)
func NewPaginatorWithDecoder[T any](query *QueryBuilder, decode func(json.RawMessage) (T, error)) *Paginator[T] {
	return &Paginator[T]{
		query:     query,
		unmarshal: decode,
	}
}

// HasMorePages reports whether there are more pages to fetch.
// It returns true before the first call to [Paginator.NextPage] and after
// any page that included a nextLink. It returns false once the final page
// has been consumed.
func (p *Paginator[T]) HasMorePages() bool { return !p.done }

// Reset resets the Paginator to the beginning of the result set, discarding
// any accumulated nextLink. The next call to [Paginator.NextPage] will re-issue
// the original query from the start.
func (p *Paginator[T]) Reset() {
	p.nextLink = ""
	p.done = false
}

// NextPage fetches the next page of results and returns a typed slice.
// On the first call the original query is executed; subsequent calls follow
// the nextLink returned by the server until no more pages are available.
//
// After the last page is returned, [Paginator.HasMorePages] returns false and
// subsequent calls to NextPage return an empty slice with no error.
func (p *Paginator[T]) NextPage(ctx context.Context) ([]T, error) {
	if p.done {
		return nil, nil
	}

	var page *Page
	var err error

	if p.nextLink != "" {
		// Follow the server-provided nextLink directly.
		page, err = p.fetchNextLink(ctx, p.nextLink)
	} else {
		page, err = p.query.Page(ctx)
	}
	if err != nil {
		return nil, err
	}
	if page == nil {
		p.done = true
		return nil, nil
	}

	p.nextLink = page.NextLink
	if p.nextLink == "" {
		p.done = true
	} else {
		if err := validateNextLink(p.nextLink, p.query.client.baseURL); err != nil {
			return nil, err
		}
	}

	items, decErr := p.decodeRaw(page)
	if decErr != nil {
		return nil, decErr
	}
	return items, nil
}

// fetchNextLink issues a GET to the server-provided nextLink URL and parses
// the response as an OData page.
func (p *Paginator[T]) fetchNextLink(ctx context.Context, link string) (*Page, error) {
	page, err := p.query.client.FetchPageAt(ctx, link)
	if err != nil {
		return nil, fmt.Errorf("paginator: follow nextLink: %w", err)
	}
	return page, nil
}

// decodeRaw converts the RawValue slice in a Page into a typed slice.
func (p *Paginator[T]) decodeRaw(page *Page) ([]T, error) {
	if len(page.RawValue) == 0 {
		return nil, nil
	}
	items := make([]T, 0, len(page.RawValue))
	for i, raw := range page.RawValue {
		v, err := p.unmarshal(raw)
		if err != nil {
			return items, fmt.Errorf("paginator: decode item %d: %w", i, err)
		}
		items = append(items, v)
	}
	return items, nil
}

// TotalCount returns the total count of matching records as reported by the
// server (requires $count=true on the original query). Returns 0 if the
// server did not return a count. Note that this value is only available after
// the first page has been fetched.
func (p *Paginator[T]) TotalCount(ctx context.Context) (int64, error) {
	page, err := p.query.Page(ctx)
	if err != nil {
		return 0, err
	}
	if page.Count != nil {
		return *page.Count, nil
	}
	return 0, nil
}
