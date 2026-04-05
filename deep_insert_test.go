package traverse

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateDeep_SetsHeaders(t *testing.T) {
	var gotContentType, gotPrefer string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotContentType = r.Header.Get("Content-Type")
		gotPrefer = r.Header.Get("Prefer")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"ID":1}`))
	}))
	defer srv.Close()

	client, err := New(WithBaseURL(srv.URL))
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	_, _ = client.From("Orders").CreateDeep(context.Background(), map[string]any{"CustomerID": "CUST1"})

	if gotContentType != "application/json;odata.metadata=minimal" {
		t.Errorf("Content-Type: expected application/json;odata.metadata=minimal, got %q", gotContentType)
	}
	if gotPrefer != "return=representation" {
		t.Errorf("Prefer: expected return=representation, got %q", gotPrefer)
	}
}

func TestCreateDeep_NestedBodyMarshaling(t *testing.T) {
	type Line struct {
		ProductID int `json:"ProductID"`
		Quantity  int `json:"Quantity"`
	}
	type Order struct {
		CustomerID string `json:"CustomerID"`
		Lines      []Line `json:"Lines"`
	}

	var receivedBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error
		receivedBody, err = json.Marshal(struct{}{})
		receivedBody = make([]byte, r.ContentLength)
		_, _ = r.Body.Read(receivedBody)
		_ = err
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"CustomerID":"CUST1"}`))
	}))
	defer srv.Close()

	client, err := New(WithBaseURL(srv.URL))
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	order := Order{
		CustomerID: "CUST1",
		Lines: []Line{
			{ProductID: 42, Quantity: 5},
		},
	}
	resp, err := client.From("Orders").CreateDeep(context.Background(), order)
	if err != nil {
		t.Fatalf("CreateDeep() failed: %v", err)
	}
	if resp == nil {
		t.Fatal("CreateDeep() returned nil response")
	}

	var parsed Order
	if jsonErr := json.Unmarshal(receivedBody, &parsed); jsonErr != nil {
		t.Fatalf("failed to parse sent body: %v", jsonErr)
	}
	if parsed.CustomerID != "CUST1" {
		t.Errorf("CustomerID: got %q, want CUST1", parsed.CustomerID)
	}
	if len(parsed.Lines) != 1 || parsed.Lines[0].ProductID != 42 {
		t.Errorf("Lines: got %+v, want [{42 5}]", parsed.Lines)
	}
}

func TestCreateDeepWithPrefer_CustomHeader(t *testing.T) {
	var gotPrefer string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPrefer = r.Header.Get("Prefer")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	client, err := New(WithBaseURL(srv.URL))
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	opts := DeepInsertOptions{ReturnRepresentation: true, ContinueOnError: true}
	_, _ = client.From("Orders").CreateDeepWithPrefer(context.Background(), map[string]any{}, opts.PreferHeader())

	expected := "return=representation; odata.continue-on-error"
	if gotPrefer != expected {
		t.Errorf("Prefer: expected %q, got %q", expected, gotPrefer)
	}
}

func TestCreateDeepWithPrefer_EmptyPrefer_OmitsHeader(t *testing.T) {
	var gotPrefer string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPrefer = r.Header.Get("Prefer")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	client, err := New(WithBaseURL(srv.URL))
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	_, _ = client.From("Orders").CreateDeepWithPrefer(context.Background(), map[string]any{}, "")

	if gotPrefer != "" {
		t.Errorf("Prefer: expected empty, got %q", gotPrefer)
	}
}

func TestDeepInsertOptions_PreferHeader(t *testing.T) {
	tests := []struct {
		name     string
		opts     DeepInsertOptions
		expected string
	}{
		{"empty", DeepInsertOptions{}, ""},
		{"return only", DeepInsertOptions{ReturnRepresentation: true}, "return=representation"},
		{"continue only", DeepInsertOptions{ContinueOnError: true}, "odata.continue-on-error"},
		{"both", DeepInsertOptions{ReturnRepresentation: true, ContinueOnError: true}, "return=representation; odata.continue-on-error"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.opts.PreferHeader()
			if got != tc.expected {
				t.Errorf("PreferHeader() = %q, want %q", got, tc.expected)
			}
		})
	}
}
