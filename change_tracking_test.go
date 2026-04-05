package traverse

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/jhonsferg/traverse/testutil"
)

func TestTrackEntity_InitialNotDirty(t *testing.T) {
	entity := map[string]interface{}{"ID": 1, "Name": "Widget"}
	te := TrackEntity(entity)
	if te.IsDirty() {
		t.Error("new TrackedEntity should not be dirty")
	}
}

func TestTrackEntity_SetMakesDirty(t *testing.T) {
	entity := map[string]interface{}{"ID": 1, "Name": "Widget"}
	te := TrackEntity(entity)
	te.Set("Name", "Updated")
	if !te.IsDirty() {
		t.Error("IsDirty() should be true after Set()")
	}
}

func TestTrackEntity_Changes_OnlyModified(t *testing.T) {
	entity := map[string]interface{}{"ID": 1, "Name": "Widget", "Price": 9.99}
	te := TrackEntity(entity)
	te.Set("Name", "Updated")
	changes := te.Changes()
	if len(changes) != 1 {
		t.Errorf("expected 1 change, got %d", len(changes))
	}
	if _, ok := changes["Name"]; !ok {
		t.Error("Changes() should contain 'Name'")
	}
	if _, ok := changes["Price"]; ok {
		t.Error("Changes() should not contain unmodified 'Price'")
	}
}

func TestTrackEntity_DirtyFields(t *testing.T) {
	entity := map[string]interface{}{"ID": 1, "Name": "Widget", "Price": 9.99}
	te := TrackEntity(entity)
	te.Set("Name", "Updated")
	te.Set("Price", 19.99)

	fields := te.DirtyFields()
	if len(fields) != 2 {
		t.Errorf("expected 2 dirty fields, got %d: %v", len(fields), fields)
	}
	fieldSet := make(map[string]bool)
	for _, f := range fields {
		fieldSet[f] = true
	}
	if !fieldSet["Name"] || !fieldSet["Price"] {
		t.Errorf("DirtyFields() missing expected fields, got: %v", fields)
	}
}

func TestTrackEntity_Get_ReturnsCurrentValue(t *testing.T) {
	entity := map[string]interface{}{"ID": 1, "Name": "Widget"}
	te := TrackEntity(entity)
	te.Set("Name", "Updated")
	v, ok := te.Get("Name")
	if !ok {
		t.Error("Get() should return true for existing field")
	}
	if v != "Updated" {
		t.Errorf("Get() should return updated value, got %v", v)
	}
}

func TestTrackEntity_Get_ReturnsFalseForMissing(t *testing.T) {
	entity := map[string]interface{}{"ID": 1}
	te := TrackEntity(entity)
	_, ok := te.Get("NonExistent")
	if ok {
		t.Error("Get() should return false for missing field")
	}
}

func TestTrackEntity_Reset_ClearsDirty(t *testing.T) {
	entity := map[string]interface{}{"ID": 1, "Name": "Widget"}
	te := TrackEntity(entity)
	te.Set("Name", "Updated")
	te.Reset()
	if te.IsDirty() {
		t.Error("IsDirty() should be false after Reset()")
	}
	// Current value should be preserved as new baseline.
	v, _ := te.Get("Name")
	if v != "Updated" {
		t.Errorf("Get() after Reset() should return current value, got %v", v)
	}
}

func TestTrackEntity_Discard_RevertsChanges(t *testing.T) {
	entity := map[string]interface{}{"ID": 1, "Name": "Widget"}
	te := TrackEntity(entity)
	te.Set("Name", "Updated")
	te.Discard()
	if te.IsDirty() {
		t.Error("IsDirty() should be false after Discard()")
	}
	v, _ := te.Get("Name")
	if v != "Widget" {
		t.Errorf("Discard() should revert to original value, got %v", v)
	}
}

func TestTrackEntity_Original_Preserved(t *testing.T) {
	entity := map[string]interface{}{"ID": 1, "Name": "Widget"}
	te := TrackEntity(entity)
	te.Set("Name", "Updated")
	orig := te.Original()
	if orig["Name"] != "Widget" {
		t.Errorf("Original() should preserve original value, got %v", orig["Name"])
	}
}

func TestTrackEntity_MarshalJSON_OnlyChanges(t *testing.T) {
	entity := map[string]interface{}{"ID": 1, "Name": "Widget", "Price": 9.99}
	te := TrackEntity(entity)
	te.Set("Price", 19.99)

	b, err := json.Marshal(te)
	if err != nil {
		t.Fatalf("MarshalJSON() error: %v", err)
	}

	var out map[string]interface{}
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}
	if len(out) != 1 {
		t.Errorf("expected 1 field in JSON, got %d: %v", len(out), out)
	}
	if _, ok := out["Price"]; !ok {
		t.Error("MarshalJSON() should include 'Price'")
	}
	if _, ok := out["Name"]; ok {
		t.Error("MarshalJSON() should not include unmodified 'Name'")
	}
}

func TestTrackEntity_SaveChanges_NoDirty_IsNoOp(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	c, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	entity := map[string]interface{}{"ID": 1, "Name": "Widget"}
	te := TrackEntity(entity)

	// No changes - SaveChanges should be a no-op.
	err = te.SaveChanges(context.Background(), c, "Products", 1)
	if err != nil {
		t.Errorf("SaveChanges() on clean entity should return nil, got: %v", err)
	}
	if server.RequestCount() != 0 {
		t.Errorf("SaveChanges() on clean entity should make no requests, got %d", server.RequestCount())
	}
}

func TestTrackEntity_SaveChanges_PatchesDirtyFields(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{Status: 204, Body: ""})

	c, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	entity := map[string]interface{}{"ID": 1, "Name": "Widget", "Price": 9.99}
	te := TrackEntity(entity)
	te.Set("Name", "Updated Widget")
	te.Set("Price", 19.99)

	err = te.SaveChanges(context.Background(), c, "Products", 1)
	if err != nil {
		t.Fatalf("SaveChanges() error: %v", err)
	}

	reqs := server.RecordedRequests()
	if len(reqs) != 1 {
		t.Fatalf("expected 1 request, got %d", len(reqs))
	}
	req := reqs[0]
	if req.Method != "PATCH" {
		t.Errorf("expected PATCH, got %s", req.Method)
	}

	var body map[string]interface{}
	if err := json.Unmarshal(req.Body, &body); err != nil {
		t.Fatalf("failed to parse request body: %v", err)
	}
	if len(body) != 2 {
		t.Errorf("expected 2 fields in PATCH body, got %d: %v", len(body), body)
	}
	if _, ok := body["Name"]; !ok {
		t.Error("PATCH body should contain 'Name'")
	}
	if _, ok := body["Price"]; !ok {
		t.Error("PATCH body should contain 'Price'")
	}
	if _, ok := body["ID"]; ok {
		t.Error("PATCH body should not contain unmodified 'ID'")
	}
}

func TestTrackEntity_SaveChanges_ResetsAfterSave(t *testing.T) {
	server := testutil.NewMockServer()
	defer server.Close()

	server.Enqueue(testutil.MockResponse{Status: 204, Body: ""})

	c, err := New(WithBaseURL(server.URL()))
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	entity := map[string]interface{}{"ID": 1, "Name": "Widget"}
	te := TrackEntity(entity)
	te.Set("Name", "Updated")

	err = te.SaveChanges(context.Background(), c, "Products", 1)
	if err != nil {
		t.Fatalf("SaveChanges() error: %v", err)
	}

	if te.IsDirty() {
		t.Error("TrackedEntity should not be dirty after SaveChanges()")
	}
}
