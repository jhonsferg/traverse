package webhooks

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jhonsferg/traverse"
)

// TestSubscribe tests subscription creation with the OData service.
func TestSubscribe(t *testing.T) {
	svc := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/$subscriptions" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		var req subscriptionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if req.EntitySetName == "" || req.CallbackURL == "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		resp := subscriptionResponse{
			SubscriptionID:     "sub-123",
			EntitySetName:      req.EntitySetName,
			CallbackURL:        req.CallbackURL,
			ExpirationDateTime: req.ExpirationDateTime,
			ClientState:        req.ClientState,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}))
	defer svc.Close()

	client, _ := traverse.New(
		traverse.WithBaseURL(svc.URL),
		traverse.WithODataVersion(traverse.ODataV4),
	)
	defer client.Close() //nolint:errcheck

	cfg := Config{
		EntitySet:   "Products",
		CallbackURL: "https://example.com/webhook",
		Expiry:      24 * time.Hour,
	}

	sub, err := Subscribe(context.Background(), client, cfg)
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	if sub.SubscriptionID() != "sub-123" {
		t.Errorf("Expected subscription ID 'sub-123', got %q", sub.SubscriptionID())
	}
}

// TestSubscribeNoSupport tests error when OData service doesn't support subscriptions.
func TestSubscribeNoSupport(t *testing.T) {
	svc := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer svc.Close()

	client, _ := traverse.New(
		traverse.WithBaseURL(svc.URL),
		traverse.WithODataVersion(traverse.ODataV4),
	)
	defer client.Close() //nolint:errcheck

	cfg := Config{
		EntitySet:   "Products",
		CallbackURL: "https://example.com/webhook",
	}

	_, err := Subscribe(context.Background(), client, cfg)
	if err == nil {
		t.Fatal("Expected error for unsupported $subscriptions")
	}
}

// TestHandlerValidationRequest tests validation token handling.
func TestHandlerValidationRequest(t *testing.T) {
	sub := &Subscription{
		id:       "sub-1",
		cfg:      Config{EntitySet: "Products"},
		handlers: make(map[ChangeType][]func(context.Context, Notification)),
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequestWithContext(context.Background(), "GET", "/?validationToken=token-xyz", nil)

	sub.Handler().ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	if w.Body.String() != "token-xyz" {
		t.Errorf("Expected response body 'token-xyz', got %q", w.Body.String())
	}
}

// TestHandlerNotification tests receiving and dispatching notifications.
func TestHandlerNotification(t *testing.T) {
	sub := &Subscription{
		id:       "sub-1",
		cfg:      Config{EntitySet: "Products"},
		handlers: make(map[ChangeType][]func(context.Context, Notification)),
	}

	var received Notification
	var receivedMu sync.Mutex

	sub.OnCreated(func(ctx context.Context, n Notification) {
		receivedMu.Lock()
		defer receivedMu.Unlock()
		received = n
	})

	payload := notificationPayload{
		Value: []notificationValue{
			{
				SubscriptionID: "sub-1",
				ClientState:    "state-1",
				ChangeType:     "created",
				EntityKey:      "1",
				Data:           json.RawMessage(`{"name":"Product 1"}`),
			},
		},
	}

	bodyBytes, _ := json.Marshal(payload)
	w := httptest.NewRecorder()
	r := httptest.NewRequestWithContext(context.Background(), "POST", "/", bytes.NewReader(bodyBytes))
	r.Header.Set("Content-Type", "application/json")

	sub.Handler().ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	time.Sleep(100 * time.Millisecond)
	receivedMu.Lock()
	if received.SubscriptionID != "sub-1" {
		t.Errorf("Expected subscription ID 'sub-1', got %q", received.SubscriptionID)
	}
	if received.ChangeType != ChangeTypeCreated {
		t.Errorf("Expected change type 'created', got %q", received.ChangeType)
	}
	receivedMu.Unlock()
}

// TestHandlerHMACVerification tests HMAC signature verification.
func TestHandlerHMACVerification(t *testing.T) {
	secret := "my-secret"
	sub := &Subscription{
		id:       "sub-1",
		cfg:      Config{EntitySet: "Products", Secret: secret},
		handlers: make(map[ChangeType][]func(context.Context, Notification)),
	}

	payload := notificationPayload{
		Value: []notificationValue{
			{
				SubscriptionID: "sub-1",
				ChangeType:     "created",
				EntityKey:      "1",
			},
		},
	}

	bodyBytes, _ := json.Marshal(payload)

	// Test valid signature
	hmacSig := hmac.New(sha256.New, []byte(secret))
	hmacSig.Write(bodyBytes)
	validSignature := hex.EncodeToString(hmacSig.Sum(nil))

	w := httptest.NewRecorder()
	r := httptest.NewRequestWithContext(context.Background(), "POST", "/", bytes.NewReader(bodyBytes))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-Signature", validSignature)

	sub.Handler().ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 for valid signature, got %d", w.Code)
	}

	// Test invalid signature
	w = httptest.NewRecorder()
	r = httptest.NewRequestWithContext(context.Background(), "POST", "/", bytes.NewReader(bodyBytes))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-Signature", "invalid-signature")

	sub.Handler().ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401 for invalid signature, got %d", w.Code)
	}

	// Test missing signature when secret is set
	w = httptest.NewRecorder()
	r = httptest.NewRequestWithContext(context.Background(), "POST", "/", bytes.NewReader(bodyBytes))
	r.Header.Set("Content-Type", "application/json")

	sub.Handler().ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401 for missing signature, got %d", w.Code)
	}
}

// TestHandlerUpdateNotification tests receiving update notifications.
func TestHandlerUpdateNotification(t *testing.T) {
	sub := &Subscription{
		id:       "sub-1",
		cfg:      Config{EntitySet: "Products"},
		handlers: make(map[ChangeType][]func(context.Context, Notification)),
	}

	var received Notification
	var receivedMu sync.Mutex

	sub.OnUpdated(func(ctx context.Context, n Notification) {
		receivedMu.Lock()
		defer receivedMu.Unlock()
		received = n
	})

	payload := notificationPayload{
		Value: []notificationValue{
			{
				SubscriptionID: "sub-1",
				ChangeType:     "updated",
				EntityKey:      "5",
			},
		},
	}

	bodyBytes, _ := json.Marshal(payload)
	w := httptest.NewRecorder()
	r := httptest.NewRequestWithContext(context.Background(), "POST", "/", bytes.NewReader(bodyBytes))

	sub.Handler().ServeHTTP(w, r)

	time.Sleep(100 * time.Millisecond)
	receivedMu.Lock()
	if received.ChangeType != ChangeTypeUpdated {
		t.Errorf("Expected change type 'updated', got %q", received.ChangeType)
	}
	if received.EntityID != "5" {
		t.Errorf("Expected entity ID '5', got %q", received.EntityID)
	}
	receivedMu.Unlock()
}

// TestHandlerDeleteNotification tests receiving delete notifications.
func TestHandlerDeleteNotification(t *testing.T) {
	sub := &Subscription{
		id:       "sub-1",
		cfg:      Config{EntitySet: "Products"},
		handlers: make(map[ChangeType][]func(context.Context, Notification)),
	}

	var received Notification
	var receivedMu sync.Mutex

	sub.OnDeleted(func(ctx context.Context, n Notification) {
		receivedMu.Lock()
		defer receivedMu.Unlock()
		received = n
	})

	payload := notificationPayload{
		Value: []notificationValue{
			{
				SubscriptionID: "sub-1",
				ChangeType:     "deleted",
				EntityKey:      "10",
			},
		},
	}

	bodyBytes, _ := json.Marshal(payload)
	w := httptest.NewRecorder()
	r := httptest.NewRequestWithContext(context.Background(), "POST", "/", bytes.NewReader(bodyBytes))

	sub.Handler().ServeHTTP(w, r)

	time.Sleep(100 * time.Millisecond)
	receivedMu.Lock()
	if received.ChangeType != ChangeTypeDeleted {
		t.Errorf("Expected change type 'deleted', got %q", received.ChangeType)
	}
	receivedMu.Unlock()
}

// TestHandlerMethodNotAllowed tests non-POST/GET requests.
func TestHandlerMethodNotAllowed(t *testing.T) {
	sub := &Subscription{
		id:       "sub-1",
		cfg:      Config{EntitySet: "Products"},
		handlers: make(map[ChangeType][]func(context.Context, Notification)),
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequestWithContext(context.Background(), "PUT", "/", nil)

	sub.Handler().ServeHTTP(w, r)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", w.Code)
	}
}

// TestRenew tests subscription renewal.
func TestRenew(t *testing.T) {
	requestCount := int32(0)
	svc := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/$subscriptions/sub-1" && r.Method == http.MethodPatch {
			atomic.AddInt32(&requestCount, 1)
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer svc.Close()

	client, _ := traverse.New(
		traverse.WithBaseURL(svc.URL),
		traverse.WithODataVersion(traverse.ODataV4),
	)
	defer client.Close() //nolint:errcheck

	sub := &Subscription{
		id:       "sub-1",
		cfg:      Config{EntitySet: "Products", Expiry: 24 * time.Hour},
		client:   client,
		handlers: make(map[ChangeType][]func(context.Context, Notification)),
	}

	err := sub.Renew(context.Background(), 0)
	if err != nil {
		t.Fatalf("Renew failed: %v", err)
	}

	if atomic.LoadInt32(&requestCount) != 1 {
		t.Errorf("Expected 1 PATCH request, got %d", atomic.LoadInt32(&requestCount))
	}
}

// TestDelete tests subscription deletion.
func TestDelete(t *testing.T) {
	requestCount := int32(0)
	svc := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/$subscriptions/sub-1" && r.Method == http.MethodDelete {
			atomic.AddInt32(&requestCount, 1)
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer svc.Close()

	client, _ := traverse.New(
		traverse.WithBaseURL(svc.URL),
		traverse.WithODataVersion(traverse.ODataV4),
	)
	defer client.Close() //nolint:errcheck

	_, stopRenew := context.WithCancel(context.Background())
	sub := &Subscription{
		id:            "sub-1",
		cfg:           Config{EntitySet: "Products"},
		client:        client,
		handlers:      make(map[ChangeType][]func(context.Context, Notification)),
		stopRenew:     stopRenew,
		autoRenewDone: make(chan struct{}),
	}
	close(sub.autoRenewDone) // no auto-renew goroutine started

	err := sub.Delete(context.Background())
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	if atomic.LoadInt32(&requestCount) != 1 {
		t.Errorf("Expected 1 DELETE request, got %d", atomic.LoadInt32(&requestCount))
	}
}

// TestMultipleHandlers tests multiple handlers for same change type.
func TestMultipleHandlers(t *testing.T) {
	sub := &Subscription{
		id:       "sub-1",
		cfg:      Config{EntitySet: "Products"},
		handlers: make(map[ChangeType][]func(context.Context, Notification)),
	}

	var count1, count2 int32

	sub.OnCreated(func(ctx context.Context, n Notification) {
		atomic.AddInt32(&count1, 1)
	})

	sub.OnCreated(func(ctx context.Context, n Notification) {
		atomic.AddInt32(&count2, 1)
	})

	payload := notificationPayload{
		Value: []notificationValue{
			{
				SubscriptionID: "sub-1",
				ChangeType:     "created",
			},
		},
	}

	bodyBytes, _ := json.Marshal(payload)
	w := httptest.NewRecorder()
	r := httptest.NewRequestWithContext(context.Background(), "POST", "/", bytes.NewReader(bodyBytes))

	sub.Handler().ServeHTTP(w, r)

	time.Sleep(100 * time.Millisecond)

	if atomic.LoadInt32(&count1) != 1 {
		t.Errorf("Expected handler1 to be called once, got %d", atomic.LoadInt32(&count1))
	}
	if atomic.LoadInt32(&count2) != 1 {
		t.Errorf("Expected handler2 to be called once, got %d", atomic.LoadInt32(&count2))
	}
}

// TestConcurrentHandlers tests thread safety with multiple goroutines (use with -race).
func TestConcurrentHandlers(t *testing.T) {
	sub := &Subscription{
		id:       "sub-1",
		cfg:      Config{EntitySet: "Products"},
		handlers: make(map[ChangeType][]func(context.Context, Notification)),
	}

	var totalCount int32

	sub.OnCreated(func(ctx context.Context, n Notification) {
		atomic.AddInt32(&totalCount, 1)
	})
	sub.OnUpdated(func(ctx context.Context, n Notification) {
		atomic.AddInt32(&totalCount, 1)
	})
	sub.OnDeleted(func(ctx context.Context, n Notification) {
		atomic.AddInt32(&totalCount, 1)
	})

	// Send concurrent notifications
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			changeType := []string{"created", "updated", "deleted"}[idx%3]
			payload := notificationPayload{
				Value: []notificationValue{
					{
						SubscriptionID: "sub-1",
						ChangeType:     changeType,
					},
				},
			}

			bodyBytes, _ := json.Marshal(payload)
			w := httptest.NewRecorder()
			r := httptest.NewRequestWithContext(context.Background(), "POST", "/", bytes.NewReader(bodyBytes))

			sub.Handler().ServeHTTP(w, r)
		}(i)
	}

	wg.Wait()
	time.Sleep(200 * time.Millisecond)

	if atomic.LoadInt32(&totalCount) != 10 {
		t.Errorf("Expected 10 total notifications, got %d", atomic.LoadInt32(&totalCount))
	}
}

// TestClientState tests that client state is preserved and returned.
func TestClientState(t *testing.T) {
	sub := &Subscription{
		id:       "sub-1",
		cfg:      Config{EntitySet: "Products", ClientState: "my-state"},
		handlers: make(map[ChangeType][]func(context.Context, Notification)),
	}

	var received Notification
	var receivedMu sync.Mutex

	sub.OnCreated(func(ctx context.Context, n Notification) {
		receivedMu.Lock()
		defer receivedMu.Unlock()
		received = n
	})

	payload := notificationPayload{
		Value: []notificationValue{
			{
				SubscriptionID: "sub-1",
				ClientState:    "my-state",
				ChangeType:     "created",
			},
		},
	}

	bodyBytes, _ := json.Marshal(payload)
	w := httptest.NewRecorder()
	r := httptest.NewRequestWithContext(context.Background(), "POST", "/", bytes.NewReader(bodyBytes))

	sub.Handler().ServeHTTP(w, r)

	time.Sleep(100 * time.Millisecond)
	receivedMu.Lock()
	if received.ClientState != "my-state" {
		t.Errorf("Expected client state 'my-state', got %q", received.ClientState)
	}
	receivedMu.Unlock()
}

// TestSubscriptionID tests that SubscriptionID returns the correct ID.
func TestSubscriptionID(t *testing.T) {
	sub := &Subscription{
		id:       "sub-xyz",
		cfg:      Config{EntitySet: "Products"},
		handlers: make(map[ChangeType][]func(context.Context, Notification)),
	}

	if sub.SubscriptionID() != "sub-xyz" {
		t.Errorf("Expected subscription ID 'sub-xyz', got %q", sub.SubscriptionID())
	}
}

// TestInvalidConfig tests invalid configuration.
func TestInvalidConfig(t *testing.T) {
	svc := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer svc.Close()

	client, _ := traverse.New(
		traverse.WithBaseURL(svc.URL),
		traverse.WithODataVersion(traverse.ODataV4),
	)
	defer client.Close() //nolint:errcheck

	// Missing EntitySet
	cfg := Config{
		CallbackURL: "https://example.com/webhook",
	}

	_, err := Subscribe(context.Background(), client, cfg)
	if err == nil || !bytes.Contains([]byte(err.Error()), []byte("EntitySet")) {
		t.Errorf("Expected error about missing EntitySet, got: %v", err)
	}

	// Missing CallbackURL
	cfg = Config{
		EntitySet: "Products",
	}

	_, err = Subscribe(context.Background(), client, cfg)
	if err == nil || !bytes.Contains([]byte(err.Error()), []byte("CallbackURL")) {
		t.Errorf("Expected error about missing CallbackURL, got: %v", err)
	}
}

func TestOnRenewErrorCallback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{ //nolint:errcheck
				"id":                 "sub-callback-test",
				"expirationDateTime": time.Now().Add(1 * time.Hour).Format(time.RFC3339),
			})
		case http.MethodPatch:
			// Renewal always fails.
			http.Error(w, `{"error":"service_unavailable"}`, http.StatusServiceUnavailable)
		default:
			http.Error(w, "not found", http.StatusNotFound)
		}
	}))
	defer server.Close()

	client, _ := traverse.New(
		traverse.WithBaseURL(server.URL),
		traverse.WithODataVersion(traverse.ODataV4),
	)
	defer client.Close() //nolint:errcheck

	errCh := make(chan error, 5)
	sub, err := Subscribe(context.Background(), client, Config{
		EntitySet:          "Products",
		CallbackURL:        "https://example.com/webhook",
		Expiry:             500 * time.Millisecond,
		RenewAutomatically: true,
		OnRenewError: func(e error) {
			errCh <- e
		},
	})
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}
	defer sub.Delete(context.Background()) //nolint:errcheck

	select {
	case e := <-errCh:
		if e == nil {
			t.Error("OnRenewError called with nil error")
		}
	case <-time.After(5 * time.Second):
		t.Error("OnRenewError was never called after renewal failure")
	}
}
