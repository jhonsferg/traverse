package testutil

import (
	"bytes"
	"io"
	"net/http"
	"sync"
)

// RecordedHTTPRequest holds the details of a request intercepted by a [RequestRecorder].
//
// RecordedHTTPRequest captures all the information from an HTTP request without
// executing it. This allows inspection and testing of what the traverse client
// would send without hitting a real service.
type RecordedHTTPRequest struct {
	// Method is the HTTP method (GET, POST, PATCH, DELETE, PUT, etc.)
	Method string
	// URL is the full request URL including query parameters
	URL string
	// Headers are the request headers sent by the client
	Headers http.Header
	// Body contains the raw request body bytes. May be nil for requests with
	// no body (GET, HEAD) or if the body could not be read.
	Body []byte
}

// RequestRecorder intercepts outgoing requests from a traverse client without
// actually sending them.
//
// RequestRecorder works by wrapping the HTTP transport and recording every request
// before it's sent. This is useful when you only need to assert on what was sent,
// not on the response. Unlike MockServer, RequestRecorder doesn't mock responses -
// requests don't actually execute.
//
// Example:
//
//	recorder := testutil.NewRequestRecorder()
//	client, _ := traverse.New(
//		traverse.WithBaseURL("http://fake.service"),
//		traverse.WithRecorder(recorder), // hypothetical option
//	)
//	client.From("Products").Filter("Price gt 100").First(ctx)
//
//	requests := recorder.Requests()
//	// Inspect requests without needing a real server
type RequestRecorder struct {
	mu       sync.Mutex
	requests []*RecordedHTTPRequest
}

// NewRequestRecorder creates a new, empty [RequestRecorder].
//
// NewRequestRecorder initializes a new request recorder ready to capture outgoing
// requests. Create one per test (or call Reset() between tests if reusing).
func NewRequestRecorder() *RequestRecorder { return &RequestRecorder{} }

// Middleware returns an http.RoundTripper wrapping function that records
// every outgoing request.
//
// Middleware returns a transport middleware function suitable for use with
// a relay client or standard http.Client. It intercepts each request,
// records its details, and passes it through to the next transport unchanged.
//
// The middleware allows the actual request to continue (it doesn't prevent execution),
// unlike MockServer which mocks the responses entirely.
//
// Example (with relay client):
//
//	recorder := testutil.NewRequestRecorder()
//	middleware := recorder.Middleware()
//	relayClient := relay.New(
//		relay.WithBaseURL("..."),
//		relay.WithTransportMiddleware(middleware),
//	)
func (r *RequestRecorder) Middleware() func(http.RoundTripper) http.RoundTripper {
	return func(next http.RoundTripper) http.RoundTripper {
		return roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			recorded := &RecordedHTTPRequest{
				Method:  req.Method,
				URL:     req.URL.String(),
				Headers: req.Header.Clone(),
			}
			// Read and restore the request body so the downstream transport
			// still receives the full payload.
			if req.Body != nil && req.Body != http.NoBody {
				body, err := io.ReadAll(req.Body)
				_ = req.Body.Close() //nolint:errcheck
				if err == nil {
					recorded.Body = body
					req.Body = io.NopCloser(bytes.NewReader(body))
				}
			}
			r.mu.Lock()
			r.requests = append(r.requests, recorded)
			r.mu.Unlock()
			return next.RoundTrip(req)
		})
	}
}

// Requests returns a snapshot of all intercepted requests.
//
// Requests returns a copy of the recorded request list. This is safe to call
// concurrently with active recording.
//
// Example:
//
//	requests := recorder.Requests()
//	for _, req := range requests {
//		if strings.Contains(req.URL, "$filter") {
//			fmt.Println("Client sent a filtered query")
//		}
//	}
func (r *RequestRecorder) Requests() []*RecordedHTTPRequest {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]*RecordedHTTPRequest, len(r.requests))
	copy(out, r.requests)
	return out
}

// RequestCount returns the total number of recorded requests.
//
// RequestCount provides a quick count of all requests recorded so far.
// Faster than len(Requests()) when you only need the count.
//
// Example:
//
//	if recorder.RequestCount() == 0 {
//		t.Error("expected at least one request")
//	}
func (r *RequestRecorder) RequestCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.requests)
}

// Reset clears all recorded requests.
//
// Reset allows reusing a RequestRecorder between test cases without creating
// a new instance.
//
// Example (in a test loop):
//
//	recorder := testutil.NewRequestRecorder()
//	for _, testCase := range testCases {
//		recorder.Reset()
//		// Run test with fresh recorder
//	}
func (r *RequestRecorder) Reset() {
	r.mu.Lock()
	r.requests = r.requests[:0]
	r.mu.Unlock()
}

// roundTripperFunc is a helper adapter to create an http.RoundTripper from a function.
// Internal use only.
type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
