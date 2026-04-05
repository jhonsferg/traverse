package traverse

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jhonsferg/relay"

	"github.com/jhonsferg/traverse/testutil"
)

// newAsyncClient creates a traverse client with retries disabled so mock server
// responses are consumed exactly once.
func newAsyncClient(t *testing.T, server *testutil.MockServer) *Client {
	t.Helper()
	rc := relay.New(relay.WithDisableRetry(), relay.WithBaseURL(server.URL()))
	c, err := New(WithBaseURL(server.URL()), WithRelayClient(rc))
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	return c
}

func TestAsyncOpPoller_SuccessAfterPolling(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	// First poll returns 202, second returns 200 with body.
	server.Enqueue(testutil.MockResponse{Status: 202, Body: ""})
	server.Enqueue(testutil.MockResponse{
		Status:  200,
		Headers: map[string]string{"Content-Type": "application/json"},
		Body:    `{"status":"done"}`,
	})

	c := newAsyncClient(t, server)
	poller := c.NewAsyncPoller(server.URL() + "/status/op1").WithPollInterval(1 * time.Millisecond)

	result, err := poller.Wait(context.Background())
	if err != nil {
		t.Fatalf("Wait() returned error: %v", err)
	}
	if result.Status != AsyncOpSucceeded {
		t.Errorf("expected AsyncOpSucceeded, got %v", result.Status)
	}
	if result.StatusCode != 200 {
		t.Errorf("expected StatusCode 200, got %d", result.StatusCode)
	}
}

func TestAsyncOpPoller_ImmediateSuccess(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{Status: 200, Body: `{"done":true}`})

	c := newAsyncClient(t, server)
	poller := c.NewAsyncPoller(server.URL() + "/status/op2").WithPollInterval(1 * time.Millisecond)

	result, err := poller.Wait(context.Background())
	if err != nil {
		t.Fatalf("Wait() error: %v", err)
	}
	if result.Status != AsyncOpSucceeded {
		t.Errorf("expected AsyncOpSucceeded, got %v", result.Status)
	}
}

func TestAsyncOpPoller_204Success(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{Status: 204, Body: ""})

	c := newAsyncClient(t, server)
	poller := c.NewAsyncPoller(server.URL() + "/status/op3").WithPollInterval(1 * time.Millisecond)

	result, err := poller.Wait(context.Background())
	if err != nil {
		t.Fatalf("Wait() error: %v", err)
	}
	if result.Status != AsyncOpSucceeded {
		t.Errorf("expected AsyncOpSucceeded, got %v", result.Status)
	}
	if result.StatusCode != 204 {
		t.Errorf("expected StatusCode 204, got %d", result.StatusCode)
	}
}

func TestAsyncOpPoller_FailureStatus(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{Status: 500, Body: `{"error":"internal error"}`})

	c := newAsyncClient(t, server)
	poller := c.NewAsyncPoller(server.URL() + "/status/op4").WithPollInterval(1 * time.Millisecond)

	result, err := poller.Wait(context.Background())
	if !errors.Is(err, ErrAsyncOpFailed) {
		t.Errorf("expected ErrAsyncOpFailed, got %v", err)
	}
	if result == nil {
		t.Fatal("result should not be nil on failure")
	}
	if result.Status != AsyncOpFailed {
		t.Errorf("expected AsyncOpFailed, got %v", result.Status)
	}
}

func TestAsyncOpPoller_MaxPolls(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	// Always return 202 - should time out.
	for i := 0; i < 5; i++ {
		server.Enqueue(testutil.MockResponse{Status: 202, Body: ""})
	}

	c := newAsyncClient(t, server)
	poller := c.NewAsyncPoller(server.URL() + "/status/op5").
		WithPollInterval(1 * time.Millisecond).
		WithMaxPolls(2)

	_, err := poller.Wait(context.Background())
	if !errors.Is(err, ErrAsyncOpTimeout) {
		t.Errorf("expected ErrAsyncOpTimeout, got %v", err)
	}
}

func TestAsyncOpPoller_ContextCancel(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	// Enqueue several 202 responses to allow multiple polls.
	for i := 0; i < 10; i++ {
		server.Enqueue(testutil.MockResponse{Status: 202, Body: "", Delay: 5 * time.Millisecond})
	}

	c := newAsyncClient(t, server)
	ctx, cancel := context.WithCancel(context.Background())

	poller := c.NewAsyncPoller(server.URL() + "/status/op6").
		WithPollInterval(20 * time.Millisecond).
		WithMaxPolls(0) // unlimited - rely on ctx

	// Cancel after a brief delay.
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	_, err := poller.Wait(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

func TestAsyncOpPoller_WithPollInterval(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	c := newAsyncClient(t, server)
	poller := c.NewAsyncPoller(server.URL() + "/status/op7")

	result := poller.WithPollInterval(10 * time.Second)
	if result != poller {
		t.Error("WithPollInterval should return the same poller")
	}
	if poller.pollInterval != 10*time.Second {
		t.Errorf("expected 10s, got %v", poller.pollInterval)
	}
}

func TestAsyncOpPoller_RetryAfterHeader(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	// First response returns 202 with Retry-After: 0 (immediate retry).
	server.Enqueue(testutil.MockResponse{
		Status:  202,
		Headers: map[string]string{"Retry-After": "0"},
		Body:    "",
	})
	server.Enqueue(testutil.MockResponse{Status: 200, Body: `{"done":true}`})

	c := newAsyncClient(t, server)
	poller := c.NewAsyncPoller(server.URL() + "/status/op8").
		WithPollInterval(1 * time.Millisecond)

	result, err := poller.Wait(context.Background())
	if err != nil {
		t.Fatalf("Wait() error: %v", err)
	}
	if result.Status != AsyncOpSucceeded {
		t.Errorf("expected AsyncOpSucceeded, got %v", result.Status)
	}
}

func TestNewAsyncPoller_Defaults(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	c := newAsyncClient(t, server)
	poller := c.NewAsyncPoller("http://example.com/status")

	if poller.pollInterval != DefaultPollInterval {
		t.Errorf("expected DefaultPollInterval %v, got %v", DefaultPollInterval, poller.pollInterval)
	}
	if poller.maxPolls != DefaultMaxPolls {
		t.Errorf("expected DefaultMaxPolls %d, got %d", DefaultMaxPolls, poller.maxPolls)
	}
	if poller.statusURL != "http://example.com/status" {
		t.Errorf("unexpected statusURL: %q", poller.statusURL)
	}
}

func TestAsyncOpStatus_String(t *testing.T) {
	cases := []struct {
		status AsyncOpStatus
		want   string
	}{
		{AsyncOpRunning, "Running"},
		{AsyncOpSucceeded, "Succeeded"},
		{AsyncOpFailed, "Failed"},
		{AsyncOpCancelled, "Cancelled"},
		{AsyncOpStatus(99), "Unknown"},
	}
	for _, tc := range cases {
		if got := tc.status.String(); got != tc.want {
			t.Errorf("AsyncOpStatus(%d).String() = %q, want %q", tc.status, got, tc.want)
		}
	}
}

func TestInterpretPollResponse_202_Running(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()
	server.Enqueue(testutil.MockResponse{Status: 202, Body: ""})

	c := newAsyncClient(t, server)
	req := c.http.Get(server.URL() + "/status")
	resp, err := c.http.Execute(req)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	result, done := interpretPollResponse(resp)
	if done {
		t.Error("202 should not be done")
	}
	if result.Status != AsyncOpRunning {
		t.Errorf("expected AsyncOpRunning, got %v", result.Status)
	}
}

func TestInterpretPollResponse_200_Done(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()
	server.Enqueue(testutil.MockResponse{Status: 200, Body: `{"value":1}`})

	c := newAsyncClient(t, server)
	req := c.http.Get(server.URL() + "/status")
	resp, err := c.http.Execute(req)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	result, done := interpretPollResponse(resp)
	if !done {
		t.Error("200 should be done")
	}
	if result.Status != AsyncOpSucceeded {
		t.Errorf("expected AsyncOpSucceeded, got %v", result.Status)
	}
}
