package traverse

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/jhonsferg/relay"
)

// AsyncOpStatus represents the state of a long-running OData async operation.
type AsyncOpStatus int

const (
	// AsyncOpRunning indicates the operation is still in progress.
	AsyncOpRunning AsyncOpStatus = iota
	// AsyncOpSucceeded indicates the operation completed successfully.
	AsyncOpSucceeded
	// AsyncOpFailed indicates the operation failed.
	AsyncOpFailed
	// AsyncOpCancelled indicates the operation was cancelled by the server.
	AsyncOpCancelled
)

// String returns a human-readable name for the status.
func (s AsyncOpStatus) String() string {
	switch s {
	case AsyncOpRunning:
		return "Running"
	case AsyncOpSucceeded:
		return "Succeeded"
	case AsyncOpFailed:
		return "Failed"
	case AsyncOpCancelled:
		return "Cancelled"
	default:
		return "Unknown"
	}
}

// AsyncResult holds the outcome of a completed async OData operation.
type AsyncResult struct {
	// Status is the final resolved status.
	Status AsyncOpStatus
	// StatusCode is the HTTP status code of the final poll response.
	StatusCode int
	// Body is the raw response body (may be nil for 204 No Content).
	Body []byte
}

// DefaultPollInterval is the default interval between async operation polls.
const DefaultPollInterval = 5 * time.Second

// DefaultMaxPolls is the default maximum number of poll attempts.
const DefaultMaxPolls = 60

// ErrAsyncOpFailed is returned when the async operation completes with a failure status.
var ErrAsyncOpFailed = errors.New("traverse: async operation failed")

// ErrAsyncOpTimeout is returned when maxPolls is exhausted before the operation completes.
var ErrAsyncOpTimeout = errors.New("traverse: async operation timed out after max polls")

// AsyncOpPoller polls an OData async operation status URL until completion.
//
// Typical OData async flow:
//  1. POST/DELETE/PATCH with "Prefer: respond-async" header
//  2. Server responds 202 Accepted + Location header (status URL)
//  3. Poll Location URL until 200/204 (done) or 4xx/5xx (failed)
//
// Usage:
//
//	resp, err := client.Execute(...)  // initial request returns 202
//	if resp.StatusCode == 202 {
//	    poller := client.NewAsyncPoller(resp.Headers.Get("Location"))
//	    result, err := poller.Wait(ctx)
//	}
type AsyncOpPoller struct {
	client       *Client
	statusURL    string
	pollInterval time.Duration
	maxPolls     int
}

// NewAsyncPoller creates an AsyncOpPoller for the given status URL.
// Default: 5s poll interval, 60 max polls (5 minutes total).
func (c *Client) NewAsyncPoller(statusURL string) *AsyncOpPoller {
	return &AsyncOpPoller{
		client:       c,
		statusURL:    statusURL,
		pollInterval: DefaultPollInterval,
		maxPolls:     DefaultMaxPolls,
	}
}

// WithPollInterval sets the interval between poll attempts.
func (p *AsyncOpPoller) WithPollInterval(d time.Duration) *AsyncOpPoller {
	p.pollInterval = d
	return p
}

// WithMaxPolls sets the maximum number of poll attempts before giving up.
// Use 0 for unlimited (rely on ctx cancellation).
func (p *AsyncOpPoller) WithMaxPolls(n int) *AsyncOpPoller {
	p.maxPolls = n
	return p
}

// Wait polls the status URL until the operation completes or ctx is cancelled.
// Returns ErrAsyncOpFailed if the server signals failure, ErrAsyncOpTimeout
// if maxPolls is exhausted before completion.
func (p *AsyncOpPoller) Wait(ctx context.Context) (*AsyncResult, error) {
	interval := p.pollInterval
	if interval <= 0 {
		interval = DefaultPollInterval
	}

	for poll := 0; p.maxPolls <= 0 || poll < p.maxPolls; poll++ {
		if poll > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(interval):
			}
		}

		req := p.client.http.Get(p.statusURL)
		req = req.WithContext(ctx)
		resp, err := p.client.http.Execute(req)
		if err != nil {
			return nil, fmt.Errorf("traverse: async poll error: %w", err)
		}

		// Respect Retry-After header to pace polls.
		if ra := resp.Headers.Get("Retry-After"); ra != "" {
			if secs, parseErr := strconv.Atoi(ra); parseErr == nil && secs > 0 {
				interval = time.Duration(secs) * time.Second
			}
		}

		result, done := interpretPollResponse(resp)
		if done {
			if result.Status == AsyncOpFailed {
				return result, ErrAsyncOpFailed
			}
			return result, nil
		}
	}

	return nil, ErrAsyncOpTimeout
}

// interpretPollResponse parses a single poll response to determine operation status.
// Returns (result, true) if the operation is done, (partial, false) if still running.
func interpretPollResponse(resp *relay.Response) (*AsyncResult, bool) {
	result := &AsyncResult{
		StatusCode: resp.StatusCode,
		Body:       resp.Body(),
	}
	switch {
	case resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated:
		result.Status = AsyncOpSucceeded
		return result, true
	case resp.StatusCode == http.StatusNoContent:
		result.Status = AsyncOpSucceeded
		return result, true
	case resp.StatusCode == http.StatusAccepted:
		result.Status = AsyncOpRunning
		return result, false
	case resp.StatusCode >= 400:
		result.Status = AsyncOpFailed
		return result, true
	default:
		result.Status = AsyncOpRunning
		return result, false
	}
}
