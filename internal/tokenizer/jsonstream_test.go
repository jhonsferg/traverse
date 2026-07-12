package tokenizer

import (
	"errors"
	"strings"
	"testing"
)

func TestStreamArrayRecords_ODataV4(t *testing.T) {
	body := `{"@odata.context":"ctx","value":[{"ID":1,"Name":"a"},{"ID":2,"Name":"b"}],"@odata.nextLink":"next"}`
	var records []map[string]interface{}
	err := StreamArrayRecords(strings.NewReader(body), ODataV4, func(r map[string]interface{}) bool {
		records = append(records, r)
		return true
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(records))
	}
	if records[0]["Name"] != "a" || records[1]["Name"] != "b" {
		t.Fatalf("unexpected record contents: %v", records)
	}
}

func TestStreamArrayRecords_ODataV4_StopEarly(t *testing.T) {
	body := `{"value":[{"ID":1},{"ID":2},{"ID":3}]}`
	var count int
	err := StreamArrayRecords(strings.NewReader(body), ODataV4, func(r map[string]interface{}) bool {
		count++
		return count < 2
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected handler to stop after 2 calls, got %d", count)
	}
}

func TestStreamArrayRecords_ODataV4_NoValueField(t *testing.T) {
	body := `{"@odata.context":"ctx","other":"field"}`
	err := StreamArrayRecords(strings.NewReader(body), ODataV4, func(r map[string]interface{}) bool {
		return true
	})
	if err == nil {
		t.Fatal("expected error when no 'value' array is present")
	}
	if !strings.Contains(err.Error(), "no 'value' array") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestStreamArrayRecords_ODataV4_EmptyArray(t *testing.T) {
	body := `{"value":[]}`
	var count int
	err := StreamArrayRecords(strings.NewReader(body), ODataV4, func(r map[string]interface{}) bool {
		count++
		return true
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0 records, got %d", count)
	}
}

func TestStreamArrayRecords_ODataV4_MalformedRecord(t *testing.T) {
	body := `{"value":[{"ID":1}, "not-an-object"]}`
	err := StreamArrayRecords(strings.NewReader(body), ODataV4, func(r map[string]interface{}) bool {
		return true
	})
	if err == nil {
		t.Fatal("expected error decoding malformed record")
	}
}

func TestStreamArrayRecords_ODataV2(t *testing.T) {
	body := `{"d":{"results":[{"ID":1,"Name":"a"},{"ID":2,"Name":"b"}],"__count":"2"}}`
	var records []map[string]interface{}
	err := StreamArrayRecords(strings.NewReader(body), ODataV2, func(r map[string]interface{}) bool {
		records = append(records, r)
		return true
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(records))
	}
}

func TestStreamArrayRecords_ODataV2_StopEarly(t *testing.T) {
	body := `{"d":{"results":[{"ID":1},{"ID":2},{"ID":3}]}}`
	var count int
	err := StreamArrayRecords(strings.NewReader(body), ODataV2, func(r map[string]interface{}) bool {
		count++
		return false
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected handler to stop after 1 call, got %d", count)
	}
}

func TestStreamArrayRecords_ODataV2_NoDWrapper(t *testing.T) {
	body := `{"other":"field"}`
	err := StreamArrayRecords(strings.NewReader(body), ODataV2, func(r map[string]interface{}) bool {
		return true
	})
	if err == nil {
		t.Fatal("expected error when no 'd' object is present")
	}
	if !strings.Contains(err.Error(), "no 'd' object") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestStreamArrayRecords_ODataV2_NoResultsField(t *testing.T) {
	body := `{"d":{"__count":"0"}}`
	err := StreamArrayRecords(strings.NewReader(body), ODataV2, func(r map[string]interface{}) bool {
		return true
	})
	if err != nil {
		t.Fatalf("expected nil error when 'results' is absent (loop exits normally), got: %v", err)
	}
}

func TestStreamArrayRecords_InvalidRootToken(t *testing.T) {
	body := `[1,2,3]`
	err := StreamArrayRecords(strings.NewReader(body), ODataV4, func(r map[string]interface{}) bool {
		return true
	})
	if err == nil {
		t.Fatal("expected error for non-object root")
	}
}

func TestStreamArrayRecords_EmptyBody(t *testing.T) {
	err := StreamArrayRecords(strings.NewReader(``), ODataV4, func(r map[string]interface{}) bool {
		return true
	})
	if err == nil {
		t.Fatal("expected error for empty body")
	}
}

func TestArrayStreamer_LifecycleV4Style(t *testing.T) {
	body := `{"ID":1,"Name":"a"}
{"ID":2,"Name":"b"}`
	s := NewArrayStreamer(strings.NewReader(body), ODataV4)
	defer func() { _ = s.Close() }()

	var got []map[string]interface{}
	for s.Next() {
		var rec map[string]interface{}
		if err := s.Decode(&rec); err != nil {
			break
		}
		got = append(got, rec)
		if len(got) == 2 {
			break
		}
	}
	if err := s.Err(); err != nil {
		t.Fatalf("unexpected streamer error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 decoded records, got %d: %v", len(got), got)
	}
	if got[0]["Name"] != "a" || got[1]["Name"] != "b" {
		t.Fatalf("unexpected decoded contents: %v", got)
	}
}

func TestArrayStreamer_ClosedStopsNext(t *testing.T) {
	s := NewArrayStreamer(strings.NewReader(`{}`), ODataV2)
	if err := s.Close(); err != nil {
		t.Fatalf("unexpected error closing: %v", err)
	}
	if s.Next() {
		t.Fatal("Next() should return false after Close()")
	}
	// Close should be idempotent.
	if err := s.Close(); err != nil {
		t.Fatalf("second close should be nil, got: %v", err)
	}
}

func TestArrayStreamer_DecodeError(t *testing.T) {
	s := NewArrayStreamer(strings.NewReader(`not-json`), ODataV4)
	if !s.Next() {
		t.Fatal("expected Next() to return true before any error")
	}
	var v map[string]interface{}
	err := s.Decode(&v)
	if err == nil {
		t.Fatal("expected decode error for malformed JSON")
	}
	if s.Err() == nil {
		t.Fatal("expected Err() to return the decode error")
	}
	// Once an error occurred, subsequent Decode calls should short-circuit
	// and return the stored error without touching the decoder again.
	err2 := s.Decode(&v)
	if !errors.Is(err2, s.Err()) && err2.Error() != s.Err().Error() {
		t.Fatalf("expected subsequent Decode to return stored error, got: %v", err2)
	}
	if s.Next() {
		t.Fatal("Next() should return false once an error has occurred")
	}
}

func TestArrayStreamer_MetaDefaultsEmpty(t *testing.T) {
	s := NewArrayStreamer(strings.NewReader(`{}`), ODataV4)
	meta := s.Meta()
	if meta.NextLink != "" || meta.DeltaLink != "" || meta.Context != "" || meta.Count != nil {
		t.Fatalf("expected zero-value metadata, got: %+v", meta)
	}
}

func TestODataVersion_Constants(t *testing.T) {
	if ODataV2 != 2 {
		t.Fatalf("expected ODataV2 == 2, got %d", ODataV2)
	}
	if ODataV4 != 4 {
		t.Fatalf("expected ODataV4 == 4, got %d", ODataV4)
	}
}
