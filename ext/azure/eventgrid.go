// Package azure provides Azure Event Grid integration for traverse OData change events.
package azure

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// EventGridClient publishes events to an Azure Event Grid topic.
type EventGridClient struct {
	topicEndpoint string
	topicKey      string
	httpClient    *http.Client
	source        string // event source (e.g. "traverse/myapp")
}

// NewEventGridClient creates a client for publishing to an Event Grid topic.
//   - topicEndpoint: the full topic URL, e.g. https://mytopic.westus2-1.eventgrid.azure.net/api/events
//   - topicKey: the access key from Azure Portal
//   - source: identifies the event source in the CloudEvents/EventGrid schema
func NewEventGridClient(topicEndpoint, topicKey, source string) *EventGridClient {
	return &EventGridClient{
		topicEndpoint: topicEndpoint,
		topicKey:      topicKey,
		httpClient:    &http.Client{Timeout: 30 * time.Second},
		source:        source,
	}
}

// EventGridEvent represents an Azure Event Grid event (schema v1).
type EventGridEvent struct {
	ID          string      `json:"id"`
	Subject     string      `json:"subject"`   // e.g. "/customers/42"
	EventType   string      `json:"eventType"` // e.g. "traverse.entity.created"
	EventTime   time.Time   `json:"eventTime"`
	DataVersion string      `json:"dataVersion"` // "1.0"
	Data        interface{} `json:"data"`
}

// ODataChangeData is the event payload for OData entity changes.
type ODataChangeData struct {
	EntitySet  string      `json:"entitySet"`
	EntityID   string      `json:"entityId"`
	ChangeType string      `json:"changeType"` // "created", "updated", "deleted"
	OldValue   interface{} `json:"oldValue,omitempty"`
	NewValue   interface{} `json:"newValue,omitempty"`
}

// Publish sends one or more events to the Event Grid topic.
// Events are sent as a JSON array (up to 1MB per batch).
func (c *EventGridClient) Publish(ctx context.Context, events []EventGridEvent) error {
	body, err := json.Marshal(events)
	if err != nil {
		return fmt.Errorf("azure: failed to marshal events: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.topicEndpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("azure: failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("aeg-sas-key", c.topicKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("azure: failed to send events: %w", err)
	}
	// Always drain and close so the underlying connection can be reused.
	defer func() {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
		resp.Body.Close() //nolint:errcheck
	}()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("azure: event grid returned status %d", resp.StatusCode)
	}

	return nil
}

// PublishEntityChange is a convenience method that creates and publishes a
// single OData entity change event.
//   - entitySet: the OData entity set name (e.g. "Customers")
//   - entityID: the entity key value
//   - changeType: "created", "updated", or "deleted"
//   - oldValue: previous state (nil for creates)
//   - newValue: new state (nil for deletes)
func (c *EventGridClient) PublishEntityChange(ctx context.Context, entitySet, entityID, changeType string, oldValue, newValue interface{}) error {
	event := EventGridEvent{
		ID:          GenerateID(),
		Subject:     fmt.Sprintf("/%s/%s", entitySet, entityID),
		EventType:   EventTypeFor(changeType),
		EventTime:   time.Now().UTC(),
		DataVersion: "1.0",
		Data: ODataChangeData{
			EntitySet:  entitySet,
			EntityID:   entityID,
			ChangeType: changeType,
			OldValue:   oldValue,
			NewValue:   newValue,
		},
	}

	return c.Publish(ctx, []EventGridEvent{event})
}

// EventTypeFor returns the standard event type string for a change type.
// e.g. "created" → "traverse.entity.created"
func EventTypeFor(changeType string) string {
	return "traverse.entity." + changeType
}

// GenerateID returns a new random event ID (UUID v4 format without external deps).
func GenerateID() string {
	var uuid [16]byte
	_, _ = rand.Read(uuid[:])
	// Set version 4 and variant bits
	uuid[6] = (uuid[6] & 0x0f) | 0x40
	uuid[8] = (uuid[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:16])
}
