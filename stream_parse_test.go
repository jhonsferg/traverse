package traverse

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/jhonsferg/traverse/testutil"
)

// ---- parseODataResponse direct unit tests ----

func TestParseODataResponse_V4Basic(t *testing.T) {
	body := `{"value":[{"ID":1,"Name":"A"},{"ID":2,"Name":"B"}]}`
	dec := json.NewDecoder(strings.NewReader(body))
	page := &Page{}
	err := parseODataResponse(dec, page, ODataV4)
	if err != nil {
		t.Fatalf("parseODataResponse: %v", err)
	}
	if len(page.Value) != 2 {
		t.Fatalf("want 2 records, got %d", len(page.Value))
	}
}

func TestParseODataResponse_V4WithNextLink(t *testing.T) {
	body := `{"value":[{"ID":1}],"@odata.nextLink":"http://svc/Products?$skip=10"}`
	dec := json.NewDecoder(strings.NewReader(body))
	page := &Page{}
	err := parseODataResponse(dec, page, ODataV4)
	if err != nil {
		t.Fatalf("parseODataResponse: %v", err)
	}
	if page.NextLink == "" {
		t.Error("expected NextLink to be populated")
	}
}

func TestParseODataResponse_V4WithCount(t *testing.T) {
	body := `{"@odata.count":42,"value":[{"ID":1}]}`
	dec := json.NewDecoder(strings.NewReader(body))
	page := &Page{}
	err := parseODataResponse(dec, page, ODataV4)
	if err != nil {
		t.Fatalf("parseODataResponse: %v", err)
	}
	if page.Count == nil || *page.Count != 42 {
		t.Errorf("expected count 42, got %v", page.Count)
	}
}

func TestParseODataResponse_V4WithContext(t *testing.T) {
	body := `{"@odata.context":"$metadata#Products","value":[]}`
	dec := json.NewDecoder(strings.NewReader(body))
	page := &Page{}
	err := parseODataResponse(dec, page, ODataV4)
	if err != nil {
		t.Fatalf("parseODataResponse: %v", err)
	}
	if page.Context != "$metadata#Products" {
		t.Errorf("expected context, got %q", page.Context)
	}
}

func TestParseODataResponse_V4WithDeltaLink(t *testing.T) {
	body := `{"value":[],"@odata.deltaLink":"http://svc/Products?$deltatoken=abc"}`
	dec := json.NewDecoder(strings.NewReader(body))
	page := &Page{}
	err := parseODataResponse(dec, page, ODataV4)
	if err != nil {
		t.Fatalf("parseODataResponse: %v", err)
	}
	if page.DeltaLink == "" {
		t.Error("expected DeltaLink to be populated")
	}
}

func TestParseODataResponse_V4SkipsUnknownFields(t *testing.T) {
	body := `{"@odata.context":"ctx","unknownField":{"nested":true},"value":[{"ID":1}]}`
	dec := json.NewDecoder(strings.NewReader(body))
	page := &Page{}
	err := parseODataResponse(dec, page, ODataV4)
	if err != nil {
		t.Fatalf("parseODataResponse with unknown fields: %v", err)
	}
	if len(page.Value) != 1 {
		t.Fatalf("want 1 record, got %d", len(page.Value))
	}
}

func TestParseODataResponse_V4_TopLevelNextLink(t *testing.T) {
	body := `{"value":[{"ID":1}],"__next":"http://svc/next"}`
	dec := json.NewDecoder(strings.NewReader(body))
	page := &Page{}
	err := parseODataResponse(dec, page, ODataV4)
	if err != nil {
		t.Fatalf("parseODataResponse: %v", err)
	}
	// __next at top level should set NextLink
	if page.NextLink == "" {
		t.Error("expected NextLink from top-level __next")
	}
}

func TestParseODataResponse_InvalidJSON(t *testing.T) {
	body := `[1,2,3]` // not an object
	dec := json.NewDecoder(strings.NewReader(body))
	page := &Page{}
	err := parseODataResponse(dec, page, ODataV4)
	if err == nil {
		t.Fatal("expected error for non-object JSON, got nil")
	}
}

// ---- parseODataV2Wrapper direct unit tests ----

func TestParseODataV2Wrapper_Basic(t *testing.T) {
	// Simulate decoder positioned after the "d" key - decoder at the value position
	body := `{"results":[{"ID":1,"Name":"A"},{"ID":2,"Name":"B"}]}`
	dec := json.NewDecoder(strings.NewReader(body))
	page := &Page{}
	err := parseODataV2Wrapper(dec, page)
	if err != nil {
		t.Fatalf("parseODataV2Wrapper: %v", err)
	}
	if len(page.Value) != 2 {
		t.Fatalf("want 2 records, got %d", len(page.Value))
	}
}

func TestParseODataV2Wrapper_WithNextLink(t *testing.T) {
	body := `{"results":[{"ID":1}],"__next":"http://svc/Products?$skiptoken=100"}`
	dec := json.NewDecoder(strings.NewReader(body))
	page := &Page{}
	err := parseODataV2Wrapper(dec, page)
	if err != nil {
		t.Fatalf("parseODataV2Wrapper: %v", err)
	}
	if page.NextLink == "" {
		t.Error("expected NextLink from __next")
	}
}

func TestParseODataV2Wrapper_WithCount(t *testing.T) {
	body := `{"results":[{"ID":1}],"__count":99}`
	dec := json.NewDecoder(strings.NewReader(body))
	page := &Page{}
	err := parseODataV2Wrapper(dec, page)
	if err != nil {
		t.Fatalf("parseODataV2Wrapper: %v", err)
	}
	if page.Count == nil || *page.Count != 99 {
		t.Errorf("expected count 99, got %v", page.Count)
	}
}

func TestParseODataV2Wrapper_SkipsUnknownFields(t *testing.T) {
	body := `{"results":[],"metadata":{"type":"Products"}}`
	dec := json.NewDecoder(strings.NewReader(body))
	page := &Page{}
	err := parseODataV2Wrapper(dec, page)
	if err != nil {
		t.Fatalf("parseODataV2Wrapper with unknown fields: %v", err)
	}
}

func TestParseODataV2Wrapper_InvalidToken(t *testing.T) {
	body := `[1,2,3]` // not an object
	dec := json.NewDecoder(strings.NewReader(body))
	page := &Page{}
	err := parseODataV2Wrapper(dec, page)
	if err == nil {
		t.Fatal("expected error for non-object, got nil")
	}
}

// ---- parseValueArray tests ----

func TestParseValueArray_Empty(t *testing.T) {
	body := `[]`
	dec := json.NewDecoder(strings.NewReader(body))
	page := &Page{}
	err := parseValueArray(dec, page)
	if err != nil {
		t.Fatalf("parseValueArray empty: %v", err)
	}
	if len(page.Value) != 0 {
		t.Errorf("expected 0 records, got %d", len(page.Value))
	}
}

func TestParseValueArray_WithRecords(t *testing.T) {
	body := `[{"ID":1,"Name":"A"},{"ID":2,"Name":"B"},{"ID":3,"Name":"C"}]`
	dec := json.NewDecoder(strings.NewReader(body))
	page := &Page{}
	err := parseValueArray(dec, page)
	if err != nil {
		t.Fatalf("parseValueArray: %v", err)
	}
	if len(page.Value) != 3 {
		t.Fatalf("want 3 records, got %d", len(page.Value))
	}
}

func TestParseValueArray_NotArray(t *testing.T) {
	body := `{"key":"value"}` // not an array
	dec := json.NewDecoder(strings.NewReader(body))
	page := &Page{}
	err := parseValueArray(dec, page)
	if err == nil {
		t.Fatal("expected error for non-array, got nil")
	}
}

// ---- Integration test: OData V2 via HTTP mock ----

func TestCollect_ODataV2_WithDWrapper(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{
		Status: 200,
		Headers: map[string]string{
			"Content-Type":       "application/json",
			"DataServiceVersion": "2.0",
		},
		Body: `{"d":{"results":[{"ID":1,"Name":"A"},{"ID":2,"Name":"B"}]}}`,
	})

	c, err := New(WithBaseURL(server.URL()), WithODataVersion(ODataV2))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	records, err := c.From("Products").Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("want 2 records, got %d", len(records))
	}
}

func TestCollect_ODataV2_WithNextLink(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	// Page 1 - with __next link pointing to page 2
	server.Enqueue(testutil.MockResponse{
		Status:  200,
		Headers: map[string]string{"Content-Type": "application/json"},
		Body:    `{"d":{"results":[{"ID":1},{"ID":2}],"__next":"/Products?$skip=2"}}`,
	})
	// Page 2 - no next link
	server.Enqueue(testutil.MockResponse{
		Status:  200,
		Headers: map[string]string{"Content-Type": "application/json"},
		Body:    `{"d":{"results":[{"ID":3}]}}`,
	})

	c, err := New(WithBaseURL(server.URL()), WithODataVersion(ODataV2))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	records, err := c.From("Products").Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if len(records) != 3 {
		t.Fatalf("want 3 records, got %d", len(records))
	}
}

func TestStream_V4_MultiplePages(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	// Page 1
	server.Enqueue(testutil.MockResponse{
		Status: 200,
		Body: `{"value":[{"ID":1},{"ID":2}],"@odata.nextLink":"` +
			server.URL() + `/Products?$skip=2"}`,
	})
	// Page 2
	server.Enqueue(testutil.MockResponse{
		Status: 200,
		Body:   `{"value":[{"ID":3}]}`,
	})

	c, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ch := c.From("Products").Stream(context.Background())
	var count int
	for r := range ch {
		if r.Err != nil {
			t.Fatalf("stream error: %v", r.Err)
		}
		count++
	}
	if count != 3 {
		t.Fatalf("want 3 streamed records, got %d", count)
	}
}

func TestStream_V4_Error(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{
		Status: 503,
		Body:   `{"error":{"code":"503","message":"unavailable"}}`,
	})

	c, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ch := c.From("Products").Stream(context.Background())
	var errs int
	for r := range ch {
		if r.Err != nil {
			errs++
		}
	}
	if errs == 0 {
		t.Fatal("expected at least one stream error")
	}
}

// ---- copyMapDeep test ----

func TestCopyMapDeep(t *testing.T) {
	src := map[string]interface{}{
		"str":   "hello",
		"num":   42.0,
		"slice": []interface{}{1, 2, 3},
		"map":   map[string]interface{}{"nested": true},
	}

	dst := copyMapDeep(src)

	// Values should match
	if dst["str"] != "hello" {
		t.Errorf("str mismatch: %v", dst["str"])
	}

	// Modifying dst should not affect src
	dst["extra"] = "added"
	if _, ok := src["extra"]; ok {
		t.Error("modifying dst should not affect src")
	}
}

// ---- edmx_parser: derefStr and derefBool ----

func TestDerefStr(t *testing.T) {
	s := "hello"
	result := derefStr(&s)
	if result != "hello" {
		t.Errorf("derefStr: want %q, got %q", "hello", result)
	}
	result2 := derefStr(nil)
	if result2 != "" {
		t.Errorf("derefStr(nil): want empty, got %q", result2)
	}
}

func TestDerefBool(t *testing.T) {
	trueStr := "true"
	result := derefBool(&trueStr)
	if result != true {
		t.Errorf("derefBool('true'): want true, got %v", result)
	}
	falseStr := "false"
	result2 := derefBool(&falseStr)
	if result2 != false {
		t.Errorf("derefBool('false'): want false, got %v", result2)
	}
	result3 := derefBool(nil)
	if result3 != false {
		t.Errorf("derefBool(nil): want false, got %v", result3)
	}
}

// ---- service_parser: parseODataV2ServiceDocument ----

func TestParseODataV2ServiceDocument(t *testing.T) {
	body := `{"d":{"EntitySets":[{"Name":"Products","Url":"Products"},{"Name":"Orders","Url":"Orders"},{"Name":"Customers","Url":"Customers"}]}}`
	doc, err := parseODataV2ServiceDocument(bytes.NewReader([]byte(body)))
	if err != nil {
		t.Fatalf("parseODataV2ServiceDocument: %v", err)
	}
	if len(doc.EntitySets) != 3 {
		t.Fatalf("want 3 entity sets, got %d", len(doc.EntitySets))
	}
}

func TestParseODataV4ServiceDocument(t *testing.T) {
	body := `{"@odata.context":"$metadata","value":[{"name":"Products","url":"Products"},{"name":"Orders","url":"Orders"}]}`
	doc, err := parseODataV4ServiceDocument(bytes.NewReader([]byte(body)))
	if err != nil {
		t.Fatalf("parseODataV4ServiceDocument: %v", err)
	}
	if len(doc.EntitySets) != 2 {
		t.Fatalf("want 2 entity sets, got %d: %v", len(doc.EntitySets), doc.EntitySets)
	}
}

// ---- goroutinePool.submit coverage ----

func TestGoroutinePoolSubmitAndClose(t *testing.T) {
	pool := newGoroutinePool(2)

	done := make(chan struct{}, 5)
	for i := 0; i < 5; i++ {
		pool.submit(func() {
			done <- struct{}{}
		})
	}

	count := 0
	for range done {
		count++
		if count == 5 {
			break
		}
	}

	pool.close()
}

// ---- string_intern.go InternBatch ----

func TestInternBatch_WithDuplicates(t *testing.T) {
	si := NewStringInterning()
	interned := si.InternBatch("alpha", "beta", "alpha", "gamma", "beta")
	if len(interned) != 5 {
		t.Fatalf("InternBatch len mismatch: want 5, got %d", len(interned))
	}
	// Same string should be same pointer
	if interned[0] != interned[2] {
		t.Error("expected identical pointer for duplicate 'alpha'")
	}
	if interned[1] != interned[4] {
		t.Error("expected identical pointer for duplicate 'beta'")
	}
}

func TestInternBatch_Empty(t *testing.T) {
	si := NewStringInterning()
	result := si.InternBatch()
	if len(result) != 0 {
		t.Errorf("InternBatch() should return empty, got %v", result)
	}
}
