package traverse

import (
	"encoding/json"
	"testing"
)

type annotationProduct struct {
	ID    int     `json:"ID"`
	Name  string  `json:"Name"`
	Price float64 `json:"Price"`
}

func TestDecodeAnnotated_BasicEntity(t *testing.T) {
	data := []byte(`{
		"ID": 1,
		"Name": "Widget",
		"Price": 9.99,
		"@odata.etag": "W/\"abc123\"",
		"@odata.count": 42
	}`)

	result, err := DecodeAnnotated[annotationProduct](data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Entity.ID != 1 {
		t.Errorf("expected ID=1, got %d", result.Entity.ID)
	}
	if result.Entity.Name != "Widget" {
		t.Errorf("expected Name=Widget, got %q", result.Entity.Name)
	}
	if result.Entity.Price != 9.99 {
		t.Errorf("expected Price=9.99, got %f", result.Entity.Price)
	}

	if len(result.Annotations) != 2 {
		t.Errorf("expected 2 annotations, got %d", len(result.Annotations))
	}
}

func TestDecodeAnnotated_AnnotationsNotInEntity(t *testing.T) {
	data := []byte(`{
		"ID": 2,
		"Name": "Gadget",
		"Price": 19.99,
		"@odata.etag": "W/\"xyz\"",
		"@Custom.Score": 99.5
	}`)

	result, err := DecodeAnnotated[annotationProduct](data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, ok := result.Annotations["@odata.etag"]
	if !ok {
		t.Error("expected @odata.etag annotation")
	}

	_, ok = result.Annotations["@Custom.Score"]
	if !ok {
		t.Error("expected @Custom.Score annotation")
	}

	// Entity fields should not bleed into annotations
	_, ok = result.Annotations["ID"]
	if ok {
		t.Error("ID should not be in annotations")
	}
}

func TestDecodeAnnotated_NoAnnotations(t *testing.T) {
	data := []byte(`{"ID": 3, "Name": "Plain", "Price": 1.0}`)

	result, err := DecodeAnnotated[annotationProduct](data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Annotations) != 0 {
		t.Errorf("expected 0 annotations, got %d", len(result.Annotations))
	}
	if result.Entity.Name != "Plain" {
		t.Errorf("expected Name=Plain, got %q", result.Entity.Name)
	}
}

func TestGetAnnotation_String(t *testing.T) {
	data := []byte(`{
		"ID": 1,
		"Name": "Alice",
		"Price": 5.0,
		"@odata.etag": "W/\"abc123\""
	}`)

	result, err := DecodeAnnotated[annotationProduct](data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var etag string
	if err := result.GetAnnotation("@odata.etag", &etag); err != nil {
		t.Fatalf("GetAnnotation failed: %v", err)
	}
	if etag != `W/"abc123"` {
		t.Errorf("expected etag W/\"abc123\", got %q", etag)
	}
}

func TestGetAnnotation_Float(t *testing.T) {
	data := []byte(`{"ID": 1, "Name": "X", "Price": 1.0, "@Custom.Score": 99.5}`)

	result, err := DecodeAnnotated[annotationProduct](data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var score float64
	if err := result.GetAnnotation("@Custom.Score", &score); err != nil {
		t.Fatalf("GetAnnotation failed: %v", err)
	}
	if score != 99.5 {
		t.Errorf("expected score 99.5, got %f", score)
	}
}

func TestGetAnnotation_Int(t *testing.T) {
	data := []byte(`{"ID": 1, "Name": "X", "Price": 1.0, "@odata.count": 42}`)

	result, err := DecodeAnnotated[annotationProduct](data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var count int
	if err := result.GetAnnotation("@odata.count", &count); err != nil {
		t.Fatalf("GetAnnotation failed: %v", err)
	}
	if count != 42 {
		t.Errorf("expected count 42, got %d", count)
	}
}

func TestGetAnnotation_Missing(t *testing.T) {
	data := []byte(`{"ID": 1, "Name": "X", "Price": 1.0}`)

	result, err := DecodeAnnotated[annotationProduct](data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var val string
	err = result.GetAnnotation("@odata.etag", &val)
	if err == nil {
		t.Error("expected error for missing annotation, got nil")
	}
}

func TestGetAnnotation_RawJSON(t *testing.T) {
	data := []byte(`{"ID": 1, "Name": "X", "Price": 1.0, "@meta.tags": ["a","b"]}`)

	result, err := DecodeAnnotated[annotationProduct](data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var tags []string
	if err := result.GetAnnotation("@meta.tags", &tags); err != nil {
		t.Fatalf("GetAnnotation failed: %v", err)
	}
	if len(tags) != 2 || tags[0] != "a" || tags[1] != "b" {
		t.Errorf("unexpected tags: %v", tags)
	}
}

func TestDecodeAnnotated_InvalidJSON(t *testing.T) {
	_, err := DecodeAnnotated[annotationProduct]([]byte(`{not valid json`))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestDecodeAnnotated_AnnotationKeys(t *testing.T) {
	data := []byte(`{
		"ID": 5,
		"Name": "Test",
		"Price": 3.14,
		"@odata.context": "https://example.com/$metadata#Products",
		"@odata.etag": "W/\"v1\"",
		"@ns.term": true
	}`)

	result, err := DecodeAnnotated[annotationProduct](data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []string{"@odata.context", "@odata.etag", "@ns.term"}
	for _, key := range expected {
		if _, ok := result.Annotations[key]; !ok {
			t.Errorf("missing annotation %q", key)
		}
	}

	var raw json.RawMessage
	if err := result.GetAnnotation("@ns.term", &raw); err != nil {
		t.Fatalf("GetAnnotation @ns.term failed: %v", err)
	}
}
