package dataverse

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOrderBy_Ascending(t *testing.T) {
	v := make(map[string][]string)
	OrderBy("name", false)(v)
	if v["$orderby"][0] != "name" {
		t.Errorf("$orderby = %q, want name", v["$orderby"][0])
	}
}

func TestOrderBy_Descending(t *testing.T) {
	v := make(map[string][]string)
	OrderBy("createdon", true)(v)
	if v["$orderby"][0] != "createdon desc" {
		t.Errorf("$orderby = %q, want createdon desc", v["$orderby"][0])
	}
}

func TestTop(t *testing.T) {
	v := make(map[string][]string)
	Top(25)(v)
	if v["$top"][0] != "25" {
		t.Errorf("$top = %q, want 25", v["$top"][0])
	}
}

func TestExpand(t *testing.T) {
	v := make(map[string][]string)
	Expand("primarycontactid")(v)
	if v["$expand"][0] != "primarycontactid" {
		t.Errorf("$expand = %q, want primarycontactid", v["$expand"][0])
	}
}

func TestClient_List_AppliesOrderByTopExpand(t *testing.T) {
	var capturedReq *http.Request
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedReq = r
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"value":[]}`))
	}))
	t.Cleanup(srv.Close)

	c, err := New(Config{OrgURL: srv.URL})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	_, err = c.List(context.Background(), "accounts",
		OrderBy("name", true),
		Top(10),
		Expand("primarycontactid"),
	)
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	q := capturedReq.URL.Query()
	if got := q.Get("$orderby"); got != "name desc" {
		t.Errorf("$orderby = %q, want name desc", got)
	}
	if got := q.Get("$top"); got != "10" {
		t.Errorf("$top = %q, want 10", got)
	}
	if got := q.Get("$expand"); got != "primarycontactid" {
		t.Errorf("$expand = %q, want primarycontactid", got)
	}
}

func TestClient_Get_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "(some-id)") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"accountid":"some-id","name":"Acme"}`))
	}))
	t.Cleanup(srv.Close)

	c, err := New(Config{OrgURL: srv.URL})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	body, err := c.Get(context.Background(), "accounts", "some-id", Select("name"))
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !strings.Contains(string(body), "Acme") {
		t.Errorf("body = %q, want to contain Acme", string(body))
	}
}

func TestClient_Get_ErrorStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":{"message":"not found"}}`))
	}))
	t.Cleanup(srv.Close)

	c, err := New(Config{OrgURL: srv.URL})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	_, err = c.Get(context.Background(), "accounts", "missing-id")
	if err == nil {
		t.Fatal("expected error for 404 status")
	}
	if !strings.Contains(err.Error(), "404") {
		t.Errorf("error = %v, want to mention status 404", err)
	}
}

func TestClient_List_ErrorStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(strings.Repeat("x", 5000)))
	}))
	t.Cleanup(srv.Close)

	c, err := New(Config{OrgURL: srv.URL})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	_, err = c.List(context.Background(), "accounts")
	if err == nil {
		t.Fatal("expected error for 500 status")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("error = %v, want to mention status 500", err)
	}
}

func TestTruncateBody(t *testing.T) {
	short := []byte("hello")
	if got := truncateBody(short); string(got) != "hello" {
		t.Errorf("truncateBody(short) = %q, want unchanged", got)
	}

	long := []byte(strings.Repeat("a", dataverseMaxErrorBodySize+500))
	got := truncateBody(long)
	if len(got) != dataverseMaxErrorBodySize {
		t.Errorf("truncateBody(long) length = %d, want %d", len(got), dataverseMaxErrorBodySize)
	}
}
