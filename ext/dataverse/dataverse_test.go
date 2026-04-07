package dataverse

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestClient_List_SendsCorrectHeaders(t *testing.T) {
	var capturedReq *http.Request
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedReq = r
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]interface{}{"value": []interface{}{}}); err != nil {
			t.Errorf("encode response: %v", err)
		}
	}))
	t.Cleanup(srv.Close)

	c, err := New(Config{OrgURL: srv.URL, MaxPageSize: 50})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	_, err = c.List(context.Background(), "accounts")
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	if got := capturedReq.Header.Get("OData-MaxVersion"); got != "4.0" {
		t.Errorf("OData-MaxVersion = %q, want 4.0", got)
	}
	if got := capturedReq.Header.Get("OData-Version"); got != "4.0" {
		t.Errorf("OData-Version = %q, want 4.0", got)
	}
	if got := capturedReq.Header.Get("Accept"); !strings.Contains(got, "application/json") {
		t.Errorf("Accept = %q, want to contain application/json", got)
	}
	if got := capturedReq.Header.Get("Prefer"); !strings.Contains(got, "odata.maxpagesize=50") {
		t.Errorf("Prefer = %q, want to contain odata.maxpagesize=50", got)
	}
}

func TestClient_List_AppliesSelectFilter(t *testing.T) {
	var capturedReq *http.Request
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedReq = r
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]interface{}{"value": []interface{}{}}); err != nil {
			t.Errorf("encode response: %v", err)
		}
	}))
	t.Cleanup(srv.Close)

	c, err := New(Config{OrgURL: srv.URL})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	_, err = c.List(context.Background(), "contacts",
		Select("name", "email"),
		Filter("statecode eq 0"),
	)
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	q := capturedReq.URL.Query()
	if got := q.Get("$select"); got != "name,email" {
		t.Errorf("$select = %q, want name,email", got)
	}
	if got := q.Get("$filter"); got != "statecode eq 0" {
		t.Errorf("$filter = %q, want statecode eq 0", got)
	}
}

func TestClient_Create_ReturnsID(t *testing.T) {
	const entityID = "https://org/api/data/v9.2/accounts(some-guid-1234)"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("OData-EntityId", entityID)
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(srv.Close)

	c, err := New(Config{OrgURL: srv.URL})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	id, err := c.Create(context.Background(), "accounts", []byte(`{"name":"Acme"}`))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if id != "some-guid-1234" {
		t.Errorf("Create returned id %q, want some-guid-1234", id)
	}
}

func TestClient_Update_UsesPatch(t *testing.T) {
	var capturedMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedMethod = r.Method
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(srv.Close)

	c, err := New(Config{OrgURL: srv.URL})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	err = c.Update(context.Background(), "accounts", "some-id", []byte(`{"name":"Updated"}`))
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if capturedMethod != http.MethodPatch {
		t.Errorf("HTTP method = %q, want PATCH", capturedMethod)
	}
}

func TestClient_Delete_UsesDelete(t *testing.T) {
	var capturedMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedMethod = r.Method
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(srv.Close)

	c, err := New(Config{OrgURL: srv.URL})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	err = c.Delete(context.Background(), "accounts", "some-id")
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if capturedMethod != http.MethodDelete {
		t.Errorf("HTTP method = %q, want DELETE", capturedMethod)
	}
}
