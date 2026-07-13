package graphql

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	gql "github.com/graphql-go/graphql"

	"github.com/jhonsferg/traverse"
)

func newTestClient(t *testing.T, handler http.HandlerFunc) (*traverse.Client, *httptest.Server) {
	t.Helper()
	ts := httptest.NewServer(handler)
	client, err := traverse.New(traverse.WithBaseURL(ts.URL + "/"))
	if err != nil {
		t.Fatalf("traverse.New() error = %v", err)
	}
	return client, ts
}

// --- QueryResolver.Resolve ---

func TestQueryResolver_Resolve_Success(t *testing.T) {
	client, ts := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/Products" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"value": []map[string]interface{}{
				{"ID": 1, "Name": "Widget"},
				{"ID": 2, "Name": "Gadget"},
			},
		})
	})
	defer ts.Close()

	qr := NewQueryResolver(client, "Products", nil)
	result, err := qr.Resolve(gql.ResolveParams{
		Context: context.Background(),
		Args: map[string]interface{}{
			"filter":  "Price gt 10",
			"orderBy": "Name asc",
			"top":     5,
			"skip":    1,
		},
	})
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	rows, ok := result.([]map[string]interface{})
	if !ok || len(rows) != 2 {
		t.Fatalf("Resolve() = %#v, want 2 rows", result)
	}
}

func TestQueryResolver_Resolve_NilContext(t *testing.T) {
	client, ts := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"value": []map[string]interface{}{}})
	})
	defer ts.Close()

	qr := NewQueryResolver(client, "Products", nil)
	_, err := qr.Resolve(gql.ResolveParams{Args: map[string]interface{}{}})
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
}

func TestQueryResolver_Resolve_QueryError(t *testing.T) {
	client, ts := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	})
	defer ts.Close()

	qr := NewQueryResolver(client, "Products", nil)
	_, err := qr.Resolve(gql.ResolveParams{Context: context.Background(), Args: map[string]interface{}{}})
	if err == nil {
		t.Fatal("Resolve() expected error, got nil")
	}
}

// --- QueryResolver.ResolveByKey ---

func TestQueryResolver_ResolveByKey_Success(t *testing.T) {
	client, ts := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/Products('1')" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"ID": 1, "Name": "Widget"})
	})
	defer ts.Close()

	qr := NewQueryResolver(client, "Products", nil)
	result, err := qr.ResolveByKey(gql.ResolveParams{
		Context: context.Background(),
		Args:    map[string]interface{}{"key": "1"},
	})
	if err != nil {
		t.Fatalf("ResolveByKey() error = %v", err)
	}
	row, ok := result.(map[string]interface{})
	if !ok || row["Name"] != "Widget" {
		t.Fatalf("ResolveByKey() = %#v", result)
	}
}

func TestQueryResolver_ResolveByKey_MissingKey(t *testing.T) {
	client, ts := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server should not be called when key is missing")
	})
	defer ts.Close()

	qr := NewQueryResolver(client, "Products", nil)

	cases := []map[string]interface{}{
		{},
		{"key": ""},
		{"key": 123},
	}
	for _, args := range cases {
		_, err := qr.ResolveByKey(gql.ResolveParams{Context: context.Background(), Args: args})
		if err == nil {
			t.Errorf("ResolveByKey(%v) expected error, got nil", args)
		}
	}
}

func TestQueryResolver_ResolveByKey_NotFound(t *testing.T) {
	client, ts := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	})
	defer ts.Close()

	qr := NewQueryResolver(client, "Products", nil)
	_, err := qr.ResolveByKey(gql.ResolveParams{
		Context: context.Background(),
		Args:    map[string]interface{}{"key": "missing"},
	})
	if err == nil {
		t.Fatal("ResolveByKey() expected error, got nil")
	}
}

// --- MutationResolver.ResolveCreate ---

func TestMutationResolver_ResolveCreate_Success(t *testing.T) {
	client, ts := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/Products" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"ID": 3, "Name": "New"})
	})
	defer ts.Close()

	mr := NewMutationResolver(client, "Products")
	result, err := mr.ResolveCreate(gql.ResolveParams{
		Context: context.Background(),
		Args:    map[string]interface{}{"input": map[string]interface{}{"Name": "New"}},
	})
	if err != nil {
		t.Fatalf("ResolveCreate() error = %v", err)
	}
	row, ok := result.(map[string]interface{})
	if !ok || row["Name"] != "New" {
		t.Fatalf("ResolveCreate() = %#v", result)
	}
}

func TestMutationResolver_ResolveCreate_MissingInput(t *testing.T) {
	client, ts := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server should not be called when input is missing")
	})
	defer ts.Close()

	mr := NewMutationResolver(client, "Products")
	_, err := mr.ResolveCreate(gql.ResolveParams{Context: context.Background(), Args: map[string]interface{}{}})
	if err == nil {
		t.Fatal("ResolveCreate() expected error, got nil")
	}
}

func TestMutationResolver_ResolveCreate_Error(t *testing.T) {
	client, ts := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	})
	defer ts.Close()

	mr := NewMutationResolver(client, "Products")
	_, err := mr.ResolveCreate(gql.ResolveParams{
		Context: context.Background(),
		Args:    map[string]interface{}{"input": map[string]interface{}{"Name": "New"}},
	})
	if err == nil {
		t.Fatal("ResolveCreate() expected error, got nil")
	}
}

// --- MutationResolver.ResolveUpdate ---

func TestMutationResolver_ResolveUpdate_Success(t *testing.T) {
	client, ts := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPatch && r.URL.Path == "/Products('1')":
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodGet && r.URL.Path == "/Products('1')":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"ID": 1, "Name": "Updated"})
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})
	defer ts.Close()

	mr := NewMutationResolver(client, "Products")
	result, err := mr.ResolveUpdate(gql.ResolveParams{
		Context: context.Background(),
		Args: map[string]interface{}{
			"key":   "1",
			"input": map[string]interface{}{"Name": "Updated"},
		},
	})
	if err != nil {
		t.Fatalf("ResolveUpdate() error = %v", err)
	}
	row, ok := result.(map[string]interface{})
	if !ok || row["Name"] != "Updated" {
		t.Fatalf("ResolveUpdate() = %#v", result)
	}
}

func TestMutationResolver_ResolveUpdate_MissingArgs(t *testing.T) {
	client, ts := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server should not be called when args are missing")
	})
	defer ts.Close()

	mr := NewMutationResolver(client, "Products")

	cases := []map[string]interface{}{
		{"input": map[string]interface{}{"Name": "x"}},
		{"key": "1"},
		{"key": "", "input": map[string]interface{}{"Name": "x"}},
	}
	for _, args := range cases {
		_, err := mr.ResolveUpdate(gql.ResolveParams{Context: context.Background(), Args: args})
		if err == nil {
			t.Errorf("ResolveUpdate(%v) expected error, got nil", args)
		}
	}
}

func TestMutationResolver_ResolveUpdate_UpdateError(t *testing.T) {
	client, ts := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	})
	defer ts.Close()

	mr := NewMutationResolver(client, "Products")
	_, err := mr.ResolveUpdate(gql.ResolveParams{
		Context: context.Background(),
		Args: map[string]interface{}{
			"key":   "1",
			"input": map[string]interface{}{"Name": "Updated"},
		},
	})
	if err == nil {
		t.Fatal("ResolveUpdate() expected error, got nil")
	}
}

func TestMutationResolver_ResolveUpdate_FetchAfterUpdateError(t *testing.T) {
	client, ts := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPatch:
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodGet:
			http.Error(w, "not found", http.StatusNotFound)
		}
	})
	defer ts.Close()

	mr := NewMutationResolver(client, "Products")
	_, err := mr.ResolveUpdate(gql.ResolveParams{
		Context: context.Background(),
		Args: map[string]interface{}{
			"key":   "1",
			"input": map[string]interface{}{"Name": "Updated"},
		},
	})
	if err == nil {
		t.Fatal("ResolveUpdate() expected error, got nil")
	}
}

// --- MutationResolver.ResolveDelete ---

func TestMutationResolver_ResolveDelete_Success(t *testing.T) {
	client, ts := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/Products('1')" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	defer ts.Close()

	mr := NewMutationResolver(client, "Products")
	result, err := mr.ResolveDelete(gql.ResolveParams{
		Context: context.Background(),
		Args:    map[string]interface{}{"key": "1"},
	})
	if err != nil {
		t.Fatalf("ResolveDelete() error = %v", err)
	}
	if result != true {
		t.Fatalf("ResolveDelete() = %#v, want true", result)
	}
}

func TestMutationResolver_ResolveDelete_MissingKey(t *testing.T) {
	client, ts := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server should not be called when key is missing")
	})
	defer ts.Close()

	mr := NewMutationResolver(client, "Products")

	cases := []map[string]interface{}{
		{},
		{"key": ""},
	}
	for _, args := range cases {
		_, err := mr.ResolveDelete(gql.ResolveParams{Context: context.Background(), Args: args})
		if err == nil {
			t.Errorf("ResolveDelete(%v) expected error, got nil", args)
		}
	}
}

func TestMutationResolver_ResolveDelete_Error(t *testing.T) {
	client, ts := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	})
	defer ts.Close()

	mr := NewMutationResolver(client, "Products")
	_, err := mr.ResolveDelete(gql.ResolveParams{
		Context: context.Background(),
		Args:    map[string]interface{}{"key": "1"},
	})
	if err == nil {
		t.Fatal("ResolveDelete() expected error, got nil")
	}
}

// --- getSelectedFields ---

func TestGetSelectedFields_NoFieldASTs(t *testing.T) {
	got := getSelectedFields(gql.ResolveParams{})
	if got != nil {
		t.Fatalf("getSelectedFields() = %v, want nil", got)
	}
}
