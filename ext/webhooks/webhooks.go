// Package webhooks provides OData v4 webhook subscription support.
//
// It enables clients to register subscriptions with an OData service and receive
// change notifications via HTTP webhooks. The package handles subscription lifecycle
// (Subscribe/Renew/Delete), provides an http.Handler for incoming notifications,
// and verifies HMAC-SHA256 signatures for security.
package webhooks

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/jhonsferg/traverse"
)

// ChangeType represents the type of data change.
type ChangeType string

const (
	ChangeTypeCreated ChangeType = "created"
	ChangeTypeUpdated ChangeType = "updated"
	ChangeTypeDeleted ChangeType = "deleted"
)

// Config configures a webhook subscription.
type Config struct {
	// EntitySet is the OData entity set to subscribe to.
	EntitySet string
	// CallbackURL is the HTTPS URL where notifications will be sent.
	CallbackURL string
	// Expiry is how long the subscription should live.
	Expiry time.Duration
	// Secret is used for HMAC-SHA256 signature verification.
	// Optional but strongly recommended.
	Secret string
	// ClientState is an optional opaque string returned with every notification.
	ClientState string
	// RenewAutomatically enables automatic renewal before expiry.
	RenewAutomatically bool
	// OnRenewError is called when an automatic renewal attempt fails. If nil,
	// renewal errors are silently ignored and the goroutine keeps retrying.
	// The callback must not block; offload any heavy work to a goroutine.
	OnRenewError func(err error)
}

// Notification is a parsed OData change notification.
type Notification struct {
	SubscriptionID string
	ClientState    string
	EntitySet      string
	ChangeType     ChangeType
	EntityID       string // key value of the changed entity
	Entity         []byte // raw JSON of the entity (may be nil for deletes)
	ReceivedAt     time.Time
}

// Subscription manages an OData webhook subscription lifecycle.
type Subscription struct {
	id            string
	cfg           Config
	client        *traverse.Client
	handlers      map[ChangeType][]func(context.Context, Notification)
	mu            sync.RWMutex
	stopAutoRenew chan struct{}
	autoRenewDone chan struct{}
}

// subscriptionRequest is the body sent to OData $subscriptions endpoint.
type subscriptionRequest struct {
	EntitySetName      string `json:"entitySetName"`
	CallbackURL        string `json:"callbackURL"`
	ExpirationDateTime string `json:"expirationDateTime"`
	ClientState        string `json:"clientState,omitempty"`
	NotificationURL    string `json:"notificationURL,omitempty"`
	ChangeTypes        string `json:"changeTypes,omitempty"`
	Resource           string `json:"resource,omitempty"`
}

// subscriptionResponse contains the response from OData $subscriptions endpoint.
type subscriptionResponse struct {
	SubscriptionID     string `json:"subscriptionId"`
	EntitySetName      string `json:"entitySetName"`
	CallbackURL        string `json:"callbackURL"`
	ExpirationDateTime string `json:"expirationDateTime"`
	ClientState        string `json:"clientState"`
}

// notificationValue represents a single change notification in the webhook payload.
type notificationValue struct {
	SubscriptionID string          `json:"subscriptionId"`
	ClientState    string          `json:"clientState"`
	ChangeType     string          `json:"changeType"`
	EntityKey      string          `json:"entityKey,omitempty"`
	SequenceNumber int             `json:"sequenceNumber,omitempty"`
	Data           json.RawMessage `json:"data,omitempty"`
}

// notificationPayload is the body of a webhook notification.
type notificationPayload struct {
	Value []notificationValue `json:"value"`
}

// Subscribe creates a new subscription with the OData service.
// Returns a Subscription that can register handlers and be used as an http.Handler.
func Subscribe(ctx context.Context, client *traverse.Client, cfg Config) (*Subscription, error) {
	if cfg.EntitySet == "" {
		return nil, fmt.Errorf("EntitySet is required")
	}
	if cfg.CallbackURL == "" {
		return nil, fmt.Errorf("CallbackURL is required")
	}
	if cfg.Expiry == 0 {
		cfg.Expiry = 24 * time.Hour
	}

	// Calculate expiration datetime
	expiryTime := time.Now().Add(cfg.Expiry).UTC()
	expiryStr := expiryTime.Format(time.RFC3339)

	// Build subscription request
	reqBody := subscriptionRequest{
		EntitySetName:      cfg.EntitySet,
		CallbackURL:        cfg.CallbackURL,
		ExpirationDateTime: expiryStr,
		ClientState:        cfg.ClientState,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal subscription request: %w", err)
	}

	// Send subscription request to OData service
	req := client.RelayClient().Post("/$subscriptions")
	req = req.WithBody(bodyBytes)
	req = req.WithHeader("Content-Type", "application/json")
	req = req.WithContext(ctx)

	resp, err := client.RelayClient().Execute(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute subscription request: %w", err)
	}

	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusMethodNotAllowed {
		return nil, fmt.Errorf("OData service does not support $subscriptions: status %d", resp.StatusCode)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("subscription request failed with status %d: %s", resp.StatusCode, string(resp.Body()))
	}

	var respData subscriptionResponse
	if err := json.Unmarshal(resp.Body(), &respData); err != nil {
		return nil, fmt.Errorf("failed to parse subscription response: %w", err)
	}

	sub := &Subscription{
		id:            respData.SubscriptionID,
		cfg:           cfg,
		client:        client,
		handlers:      make(map[ChangeType][]func(context.Context, Notification)),
		stopAutoRenew: make(chan struct{}, 1),
		autoRenewDone: make(chan struct{}),
	}

	if cfg.RenewAutomatically {
		go sub.runAutoRenew() //nolint:contextcheck,gosec
	}

	return sub, nil
}

// OnCreated registers a handler called for created entities.
func (s *Subscription) OnCreated(fn func(ctx context.Context, n Notification)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.handlers[ChangeTypeCreated] = append(s.handlers[ChangeTypeCreated], fn)
}

// OnUpdated registers a handler called for updated entities.
func (s *Subscription) OnUpdated(fn func(ctx context.Context, n Notification)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.handlers[ChangeTypeUpdated] = append(s.handlers[ChangeTypeUpdated], fn)
}

// OnDeleted registers a handler called for deleted entities.
func (s *Subscription) OnDeleted(fn func(ctx context.Context, n Notification)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.handlers[ChangeTypeDeleted] = append(s.handlers[ChangeTypeDeleted], fn)
}

// Handler returns an http.Handler that processes incoming webhook notifications.
// Mount this at the CallbackURL path in your HTTP server.
func (s *Subscription) Handler() http.Handler {
	return http.HandlerFunc(s.handleNotification)
}

// handleNotification processes incoming webhook notifications or validation requests.
func (s *Subscription) handleNotification(w http.ResponseWriter, r *http.Request) {
	// Handle validation request (GET with validationToken query param)
	if r.Method == http.MethodGet {
		validationToken := r.URL.Query().Get("validationToken")
		if validationToken != "" {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(validationToken)) //nolint:errcheck,gosec
			return
		}
	}

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// Verify HMAC signature if secret is set
	if s.cfg.Secret != "" {
		signature := r.Header.Get("X-Signature")
		if signature == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		r.Body = io.NopCloser(bytes.NewReader(bodyBytes))

		expectedSig := hmac.New(sha256.New, []byte(s.cfg.Secret))
		expectedSig.Write(bodyBytes)
		expectedStr := hex.EncodeToString(expectedSig.Sum(nil))

		if !hmac.Equal([]byte(signature), []byte(expectedStr)) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
	}

	// Parse notification payload
	var payload notificationPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Process each notification in the payload
	for _, nv := range payload.Value {
		notification := Notification{
			SubscriptionID: nv.SubscriptionID,
			ClientState:    nv.ClientState,
			EntitySet:      s.cfg.EntitySet,
			ChangeType:     ChangeType(nv.ChangeType),
			EntityID:       nv.EntityKey,
			Entity:         nv.Data,
			ReceivedAt:     time.Now(),
		}

		// Dispatch to registered handlers.
		// Deep-copy the slice under the read lock so that concurrent OnCreated/
		// OnUpdated/OnDeleted calls cannot race with our iteration.
		ctx := r.Context()
		s.mu.RLock()
		orig := s.handlers[notification.ChangeType]
		handlers := make([]func(context.Context, Notification), len(orig))
		copy(handlers, orig)
		s.mu.RUnlock()

		for _, handler := range handlers {
			handler(ctx, notification)
		}
	}

	w.WriteHeader(http.StatusOK)
}

// Renew extends the subscription expiry.
func (s *Subscription) Renew(ctx context.Context, expiry time.Duration) error {
	if expiry == 0 {
		expiry = s.cfg.Expiry
	}

	expiryTime := time.Now().Add(expiry).UTC()
	expiryStr := expiryTime.Format(time.RFC3339)

	// PATCH request to renew subscription
	reqBody := map[string]string{
		"expirationDateTime": expiryStr,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal renew request: %w", err)
	}

	req := s.client.RelayClient().Patch(fmt.Sprintf("/$subscriptions/%s", s.id))
	req = req.WithBody(bodyBytes)
	req = req.WithHeader("Content-Type", "application/json")
	req = req.WithContext(ctx)

	resp, err := s.client.RelayClient().Execute(req)
	if err != nil {
		return fmt.Errorf("failed to execute renew request: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("renew request failed with status %d: %s", resp.StatusCode, string(resp.Body()))
	}

	return nil
}

// Delete cancels the subscription with the OData service.
func (s *Subscription) Delete(ctx context.Context) error {
	// Stop auto-renewal if running
	select {
	case s.stopAutoRenew <- struct{}{}:
	default:
	}

	// Wait for auto-renew goroutine to finish
	select {
	case <-s.autoRenewDone:
	case <-time.After(5 * time.Second):
	}

	req := s.client.RelayClient().Delete(fmt.Sprintf("/$subscriptions/%s", s.id))
	req = req.WithContext(ctx)

	resp, err := s.client.RelayClient().Execute(req)
	if err != nil {
		return fmt.Errorf("failed to execute delete request: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("delete request failed with status %d: %s", resp.StatusCode, string(resp.Body()))
	}

	return nil
}

// SubscriptionID returns the ID assigned by the OData service.
func (s *Subscription) SubscriptionID() string {
	return s.id
}

// runAutoRenew periodically renews the subscription before expiry.
func (s *Subscription) runAutoRenew() {
	defer close(s.autoRenewDone)

	// Renew 5 minutes before expiry
	renewBefore := 5 * time.Minute
	if s.cfg.Expiry < renewBefore*2 {
		renewBefore = s.cfg.Expiry / 4
	}

	ticker := time.NewTicker(s.cfg.Expiry - renewBefore)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopAutoRenew:
			return
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			if err := s.Renew(ctx, s.cfg.Expiry); err != nil {
				if s.cfg.OnRenewError != nil {
					s.cfg.OnRenewError(err)
				}
			}
			cancel()
		}
	}
}
