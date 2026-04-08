package testutil

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"sync/atomic"
	"time"
)

// MockResponse defines the response the MockServer should return for a queued request.
//
// MockResponse allows configuring status codes, headers, body content, and artificial delays.
// It can also simulate network errors for testing error handling.
//
// Example:
//
//	resp := testutil.MockResponse{
//		Status: 200,
//		Headers: map[string]string{"Content-Type": "application/json"},
//		Body: `{"value":[{"ID":1,"Name":"Test"}]}`,
//		Delay: 100 * time.Millisecond,
//	}
type MockResponse struct {
	// Status is the HTTP status code (defaults to 200 if 0).
	Status int

	// Headers are response headers merged into the reply.
	Headers map[string]string

	// Body is the response body as a string.
	Body string

	// Delay introduces artificial latency before the response is written.
	// Useful for testing timeout behavior or simulating slow services.
	Delay time.Duration

	// isError causes the server to close the connection instead of writing a response.
	// Use EnqueueError() to set this conveniently.
	isError bool
}

// RecordedRequest holds details of an HTTP request captured by the MockServer.
//
// RecordedRequest stores everything needed to verify that the client made
// the expected HTTP request with correct method, path, headers, query parameters, and body.
type RecordedRequest struct {
	// Method is the HTTP method (GET, POST, PATCH, DELETE, etc.)
	Method string
	// Path is the request path (e.g., "/Products")
	Path string
	// Headers are the request headers
	Headers http.Header
	// Body is the request body as bytes
	Body []byte
	// Query contains parsed query parameters (e.g., $filter, $top, $skip)
	Query url.Values
}

// MockServer is a test HTTP server that serves queued responses in FIFO order
// and records all incoming requests.
//
// MockServer is useful for testing traverse clients without accessing a real OData service.
// Responses are served in the order they are enqueued. Requests are recorded and can be
// inspected to verify the client made the correct HTTP calls.
//
// Example:
//
//	server := testutil.NewMockServer()
//	defer server.Close()
//
//	server.Enqueue(testutil.MockResponse{
//		Status: 200,
//		Body: testutil.ODataResponse(
//			map[string]interface{}{"ID": 1, "Name": "Product"},
//		),
//	})
//
//	client, _ := traverse.New(traverse.WithBaseURL(server.URL()))
//	result, _ := client.From("Products").First(ctx)
//
//	requests := server.RecordedRequests()
//	// Verify the client made a GET request to /Products
type MockServer struct {
	server   *httptest.Server
	mu       sync.Mutex
	queue    []MockResponse
	recorded []RecordedRequest
	count    atomic.Int64
	newReqCh chan struct{}
}

// NewMockServer starts a new local HTTP test server.
//
// NewMockServer creates an in-process test HTTP server that can be used to mock
// an OData service. Call [MockServer.Close] when the test is done to shut down
// the server and clean up resources.
//
// Example:
//
//	server := testutil.NewMockServer()
//	defer server.Close()
//	// Use server.URL() as the base URL for traverse client
func NewMockServer() *MockServer {
	ms := &MockServer{
		newReqCh: make(chan struct{}, 128),
	}
	ms.server = httptest.NewServer(http.HandlerFunc(ms.handle))
	return ms
}

func (s *MockServer) handle(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	var resp MockResponse
	if len(s.queue) > 0 {
		resp = s.queue[0]
		s.queue = s.queue[1:]
	} else {
		// Default: 200 OK with empty body.
		resp = MockResponse{Status: http.StatusOK}
	}
	s.mu.Unlock()

	// Record the request.
	body, _ := io.ReadAll(r.Body)
	recorded := RecordedRequest{
		Method:  r.Method,
		Path:    r.URL.EscapedPath(), // EscapedPath returns RawPath (percent-encoded) when available
		Headers: r.Header.Clone(),
		Body:    body,
		Query:   r.URL.Query(),
	}
	s.mu.Lock()
	s.recorded = append(s.recorded, recorded)
	s.mu.Unlock()
	s.count.Add(1)
	select {
	case s.newReqCh <- struct{}{}:
	default:
	}

	// Simulate a network error.
	if resp.isError {
		panic(errors.New("simulated error"))
	}

	// Apply delay if set.
	if resp.Delay > 0 {
		time.Sleep(resp.Delay)
	}

	// Write response headers.
	for k, v := range resp.Headers {
		w.Header().Set(k, v)
	}
	if resp.Status == 0 {
		resp.Status = http.StatusOK
	}
	w.WriteHeader(resp.Status)

	// Write response body.
	if resp.Body != "" {
		_, _ = io.WriteString(w, resp.Body)
	}
}

// URL returns the base URL of the mock server.
//
// URL provides the HTTP endpoint to use when creating a traverse client.
// The returned URL is valid only while the server is running.
func (s *MockServer) URL() string {
	return s.server.URL
}

// Enqueue adds a response to the queue in FIFO order.
//
// Enqueue appends a MockResponse to the response queue. Responses are served
// in the order they are enqueued. If the queue is empty when a request arrives,
// the server responds with a default 200 OK with empty body.
//
// Example:
//
//	server.Enqueue(testutil.MockResponse{Status: 200, Body: "{\"value\":[]}"})
//	server.Enqueue(testutil.MockResponse{Status: 200, Body: "{\"value\":[]}"})
//	// First request gets first response, second request gets second response
func (s *MockServer) Enqueue(resp MockResponse) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.queue = append(s.queue, resp)
}

// EnqueueError enqueues a response that simulates a network error.
//
// EnqueueError adds a MockResponse that will cause the server to close the
// connection abruptly, simulating a network error or timeout. Useful for
// testing error handling in traverse clients.
//
// Example:
//
//	server.EnqueueError()
//	// Next request will fail with a network error
func (s *MockServer) EnqueueError() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.queue = append(s.queue, MockResponse{isError: true})
}

// RecordedRequests returns a snapshot of all recorded requests.
//
// RecordedRequests returns a copy of all HTTP requests received by the server
// since it was created. This allows verification of client behavior after
// running traverse operations.
//
// Example:
//
//	client.From("Products").First(ctx)
//	requests := server.RecordedRequests()
//	if len(requests) != 1 {
//		t.Fatalf("expected 1 request, got %d", len(requests))
//	}
//	if requests[0].Method != "GET" {
//		t.Fatalf("expected GET, got %s", requests[0].Method)
//	}
func (s *MockServer) RecordedRequests() []RecordedRequest {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]RecordedRequest, len(s.recorded))
	copy(out, s.recorded)
	return out
}

// RequestCount returns the total number of requests received by the server.
//
// RequestCount provides an atomic count of all requests received since the
// server was created. Useful for verifying that the client makes the expected
// number of network calls.
//
// Example:
//
//	initialCount := server.RequestCount()
//	client.From("Products").Top(1000).Collect(ctx)
//	finalCount := server.RequestCount()
//	if finalCount-initialCount < 1 {
//		t.Error("expected at least one request")
//	}
func (s *MockServer) RequestCount() int64 {
	return s.count.Load()
}

// Close stops the server and releases all resources.
//
// Close shuts down the mock server. It should be called at the end of each test.
// After Close, the server is no longer available and the URL is invalid.
func (s *MockServer) Close() {
	s.server.Close()
}

// WaitForRequest waits for the next request to arrive with a timeout.
//
// WaitForRequest blocks until a request arrives at the server or the timeout
// expires. This is useful in concurrent tests where you need to synchronize
// on incoming requests.
//
// Returns an error if the timeout expires before a request arrives.
//
// Example:
//
//	go client.From("Products").Stream(ctx) // async request
//	if err := server.WaitForRequest(2 * time.Second); err != nil {
//		t.Fatal("request did not arrive in time")
//	}
func (s *MockServer) WaitForRequest(timeout time.Duration) error {
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case <-s.newReqCh:
		return nil
	case <-timer.C:
		return errors.New("timeout waiting for request")
	}
}
