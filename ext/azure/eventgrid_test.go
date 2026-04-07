package azure

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newTestServer(t *testing.T, handler http.HandlerFunc) (*httptest.Server, *EventGridClient) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	client := NewEventGridClient(srv.URL, "test-key-123", "traverse/test")
	return srv, client
}

func sampleEvent() EventGridEvent {
	return EventGridEvent{
		ID:          GenerateID(),
		Subject:     "/Customers/42",
		EventType:   EventTypeFor("created"),
		DataVersion: "1.0",
		Data:        map[string]string{"foo": "bar"},
	}
}

func TestPublish_SendsCorrectHeaders(t *testing.T) {
	var gotContentType, gotKey string

	_, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotContentType = r.Header.Get("Content-Type")
		gotKey = r.Header.Get("aeg-sas-key")
		w.WriteHeader(http.StatusOK)
	})

	if err := client.Publish(context.Background(), []EventGridEvent{sampleEvent()}); err != nil {
		t.Fatalf("Publish returned error: %v", err)
	}

	if gotContentType != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", gotContentType)
	}
	if gotKey != "test-key-123" {
		t.Errorf("aeg-sas-key = %q, want test-key-123", gotKey)
	}
}

func TestPublish_EventFormat(t *testing.T) {
	var received []map[string]interface{}

	_, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &received)
		w.WriteHeader(http.StatusOK)
	})

	evt := sampleEvent()
	if err := client.Publish(context.Background(), []EventGridEvent{evt}); err != nil {
		t.Fatalf("Publish returned error: %v", err)
	}

	if len(received) != 1 {
		t.Fatalf("expected 1 event, got %d", len(received))
	}

	e := received[0]
	for _, field := range []string{"id", "subject", "eventType", "eventTime", "dataVersion"} {
		if e[field] == nil || e[field] == "" {
			t.Errorf("field %q is missing or empty in event payload", field)
		}
	}
}

func TestPublishEntityChange_Created(t *testing.T) {
	var received []map[string]interface{}

	_, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &received)
		w.WriteHeader(http.StatusOK)
	})

	if err := client.PublishEntityChange(context.Background(), "Customers", "42", "created", nil, map[string]string{"name": "Alice"}); err != nil {
		t.Fatalf("PublishEntityChange returned error: %v", err)
	}

	if len(received) != 1 {
		t.Fatalf("expected 1 event, got %d", len(received))
	}

	gotType, _ := received[0]["eventType"].(string)
	if gotType != "traverse.entity.created" {
		t.Errorf("eventType = %q, want traverse.entity.created", gotType)
	}
}

func TestPublishEntityChange_Deleted(t *testing.T) {
	var received []map[string]interface{}

	_, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &received)
		w.WriteHeader(http.StatusOK)
	})

	oldVal := map[string]string{"name": "Alice"}
	if err := client.PublishEntityChange(context.Background(), "Customers", "42", "deleted", oldVal, nil); err != nil {
		t.Fatalf("PublishEntityChange returned error: %v", err)
	}

	if len(received) != 1 {
		t.Fatalf("expected 1 event, got %d", len(received))
	}

	data, ok := received[0]["data"].(map[string]interface{})
	if !ok {
		t.Fatal("data field missing or wrong type")
	}

	if data["oldValue"] == nil {
		t.Error("oldValue should be set for deleted events")
	}
	if data["newValue"] != nil {
		t.Errorf("newValue should be nil for deleted events, got %v", data["newValue"])
	}
}

func TestGenerateID_UniqueIDs(t *testing.T) {
	seen := make(map[string]struct{}, 100)
	for i := 0; i < 100; i++ {
		id := GenerateID()
		if _, exists := seen[id]; exists {
			t.Fatalf("duplicate ID generated: %q", id)
		}
		seen[id] = struct{}{}
	}
}
