package traverse

import (
	"testing"

	"github.com/jhonsferg/relay"
)

// TestClient_Close_NilHTTP covers the return nil path of Close when http is nil.
func TestClient_Close_NilHTTP(t *testing.T) {
	c := &Client{}
	if err := c.Close(); err != nil {
		t.Errorf("Close() with nil http should return nil, got %v", err)
	}
}

// TestClient_CircuitBreakerState_NilHTTP covers the return StateClosed path when http is nil.
func TestClient_CircuitBreakerState_NilHTTP(t *testing.T) {
	c := &Client{}
	if state := c.CircuitBreakerState(); state != relay.StateClosed {
		t.Errorf("CircuitBreakerState() with nil http should return StateClosed, got %v", state)
	}
}

// TestClient_Close_WithHTTP covers the c.http.Shutdown path when http is set.
func TestClient_Close_WithHTTP(t *testing.T) {
	c, err := New(WithBaseURL("http://example.com"))
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Close(); err != nil {
		t.Errorf("Close() returned error: %v", err)
	}
}

// TestClient_CircuitBreakerState_WithHTTP covers the c.http.CircuitBreakerState() path.
func TestClient_CircuitBreakerState_WithHTTP(t *testing.T) {
	c, err := New(WithBaseURL("http://example.com"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = c.Close() }()
	// Default circuit breaker should be closed (healthy)
	if state := c.CircuitBreakerState(); state != relay.StateClosed {
		t.Errorf("CircuitBreakerState() = %v, want StateClosed", state)
	}
}
