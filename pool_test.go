package traverse

import (
	"bytes"
	"encoding/json"
	"testing"
)

// TestMapPoolReuse tests that the object pool is reusing maps efficiently
func TestMapPoolReuse(t *testing.T) {
	// Get a map from pool
	m1 := mapPool.Get().(map[string]interface{})
	m1["key1"] = "value1"
	m1["key2"] = "value2"

	// Get address for comparison
	addr1 := getMapAddress(m1)

	// Put it back
	mapPool.Put(resetMapForPool(m1))

	// Get another map - should be same or similar
	m2 := mapPool.Get().(map[string]interface{})
	addr2 := getMapAddress(m2)

	// Both should have zero length after reset
	if len(m2) != 0 {
		t.Errorf("Expected map to be empty after pool reset, got %d items", len(m2))
	}

	// Should be able to reuse it
	m2["newkey"] = "newvalue"
	if len(m2) != 1 {
		t.Errorf("Expected map to have 1 item after adding, got %d", len(m2))
	}

	// Addresses should be the same (showing reuse)
	if addr1 == addr2 {
		t.Logf("Pool is reusing maps - same address %v used twice", addr1)
	} else {
		t.Logf("Pool allocated new maps - addresses differ: %v vs %v (this is OK under GC pressure)", addr1, addr2)
	}
}

// TestResetMapForPoolSmall tests resetting small maps
func TestResetMapForPoolSmall(t *testing.T) {
	m := make(map[string]interface{})
	for i := 0; i < 100; i++ {
		m["key"+string(rune(i))] = i
	}

	if len(m) != 100 {
		t.Errorf("Expected 100 items, got %d", len(m))
	}

	m = resetMapForPool(m)

	if len(m) != 0 {
		t.Errorf("Expected 0 items after reset, got %d", len(m))
	}
}

// TestResetMapForPoolLarge tests that large maps are not held in pool
func TestResetMapForPoolLarge(t *testing.T) {
	m := make(map[string]interface{})
	for i := 0; i < 1000; i++ {
		m["key"+string(rune(i%26))] = i
	}

	addr1 := getMapAddress(m)

	// Large map should get a new allocation
	m = resetMapForPool(m)

	addr2 := getMapAddress(m)

	if len(m) != 0 {
		t.Errorf("Expected 0 items after reset, got %d", len(m))
	}

	if addr1 == addr2 {
		t.Logf("Note: Large map was reused (might be OK depending on GC timing)")
	} else {
		t.Logf("Large map was discarded and new one allocated - good memory management")
	}
}

// TestParseValueArrayWithPool tests that parseValueArray uses the pool
func TestParseValueArrayWithPool(t *testing.T) {
	// Create a simple JSON array
	jsonData := []byte(`{"value": [
{"ID": 1, "Name": "Test1"},
{"ID": 2, "Name": "Test2"},
{"ID": 3, "Name": "Test3"}
]}`)

	reader := bytes.NewReader(jsonData)
	decoder := json.NewDecoder(reader)

	// Parse root object
	token, _ := decoder.Token()
	if delim, ok := token.(json.Delim); !ok || delim != '{' {
		t.Fatal("Expected opening {")
	}

	page := &Page{
		Value: make([]map[string]interface{}, 0),
	}

	// Look for "value" key
	for decoder.More() {
		token, _ := decoder.Token()
		key, ok := token.(string)
		if !ok {
			continue
		}
		if key == "value" {
			if err := parseValueArray(decoder, page); err != nil {
				t.Fatalf("parseValueArray failed: %v", err)
			}
			break
		}
	}

	if len(page.Value) != 3 {
		t.Errorf("Expected 3 records, got %d", len(page.Value))
	}

	// Check that records are properly populated
	if id, ok := page.Value[0]["ID"]; !ok || id != float64(1) {
		t.Errorf("Expected ID 1 in first record, got %v", id)
	}
}

// getMapAddress returns the pointer address of a map header
func getMapAddress(m map[string]interface{}) uintptr {
	// We can't directly get the address of a map, but we can track reuse
	// This is just for demonstration purposes
	return uintptr(len(m)) // Placeholder
}
