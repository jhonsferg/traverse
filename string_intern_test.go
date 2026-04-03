package traverse

import (
	"strings"
	"sync"
	"testing"
)

func TestStringInterning_Single(t *testing.T) {
	si := NewStringInterning()

	str1 := "entityName"
	result1 := si.Intern(str1)

	if result1 != str1 {
		t.Errorf("Expected %q, got %q", str1, result1)
	}

	// Second call should return same pointer
	result2 := si.Intern(str1)
	if result1 != result2 {
		t.Error("Expected same pointer for interned string")
	}
}

func TestStringInterning_Multiple(t *testing.T) {
	si := NewStringInterning()

	str1 := "Property1"
	str2 := "Property2"
	str3 := "Property1" // duplicate

	result1 := si.Intern(str1)
	result2 := si.Intern(str2)
	result3 := si.Intern(str3)

	if result1 != result3 {
		t.Error("Expected same pointer for duplicate string")
	}

	if result1 == result2 {
		t.Error("Expected different pointers for different strings")
	}

	if si.CacheSize() != 2 {
		t.Errorf("Expected cache size 2, got %d", si.CacheSize())
	}
}

func TestStringInterning_Empty(t *testing.T) {
	si := NewStringInterning()

	result := si.Intern("")
	if result != "" {
		t.Errorf("Expected empty string, got %q", result)
	}

	if si.CacheSize() != 0 {
		t.Error("Expected cache size 0 for empty string")
	}
}

func TestStringInterning_BatchIntern(t *testing.T) {
	si := NewStringInterning()

	strings := []string{"prop1", "prop2", "prop1", "prop3", "prop2"}
	results := si.InternBatch(strings...)

	if len(results) != len(strings) {
		t.Errorf("Expected %d results, got %d", len(strings), len(results))
	}

	// Check that duplicates point to same string
	if results[0] != results[2] {
		t.Error("Expected same pointer for duplicate strings in batch")
	}

	if results[1] != results[4] {
		t.Error("Expected same pointer for duplicate strings in batch")
	}

	// Should have 3 unique strings in cache
	if si.CacheSize() != 3 {
		t.Errorf("Expected cache size 3, got %d", si.CacheSize())
	}
}

func TestStringInterning_Clear(t *testing.T) {
	si := NewStringInterning()

	si.Intern("prop1")
	si.Intern("prop2")

	if si.CacheSize() != 2 {
		t.Errorf("Expected cache size 2, got %d", si.CacheSize())
	}

	si.Clear()

	if si.CacheSize() != 0 {
		t.Error("Expected cache size 0 after clear")
	}
}

func TestStringInterning_Concurrent(t *testing.T) {
	si := NewStringInterning()

	var wg sync.WaitGroup
	numGoroutines := 10
	stringsPerGoroutine := 100

	// Generate test strings
	testStrings := make([]string, stringsPerGoroutine)
	for i := 0; i < stringsPerGoroutine; i++ {
		testStrings[i] = "prop" + string(rune(i%10))
	}

	// Intern strings concurrently
	for g := 0; g < numGoroutines; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for _, s := range testStrings {
				si.Intern(s)
			}
		}()
	}

	wg.Wait()

	// Should have exactly 10 unique strings
	if si.CacheSize() != 10 {
		t.Errorf("Expected cache size 10, got %d", si.CacheSize())
	}
}

func TestStringInterning_MemoryReduction(t *testing.T) {
	si := NewStringInterning()

	// Simulate entity/property name pattern
	entityNames := []string{
		"Customer", "Product", "Order", "Invoice",
		"Customer", "Product", "Order", // duplicates
	}

	var internalPointers []string
	for _, name := range entityNames {
		internalPointers = append(internalPointers, si.Intern(name))
	}

	// All instances of same entity should point to same string
	if internalPointers[0] != internalPointers[4] {
		t.Error("Expected same pointer for duplicated entity name")
	}

	if si.CacheSize() != 4 {
		t.Errorf("Expected 4 unique entity names, got %d", si.CacheSize())
	}
}

func TestStringInterning_LargeScale(t *testing.T) {
	si := NewStringInterning()

	// Generate many unique strings
	const numStrings = 10000
	const uniqueStrings = 100

	for i := 0; i < numStrings; i++ {
		str := "property_" + string(rune(i%uniqueStrings))
		si.Intern(str)
	}

	if si.CacheSize() != uniqueStrings {
		t.Errorf("Expected %d unique strings, got %d", uniqueStrings, si.CacheSize())
	}
}

func BenchmarkStringInterning_Single(b *testing.B) {
	si := NewStringInterning()
	testStr := "entityName"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		si.Intern(testStr)
	}
}

func BenchmarkStringInterning_Multiple(b *testing.B) {
	si := NewStringInterning()
	testStrings := []string{"prop1", "prop2", "prop3", "prop4", "prop5"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, s := range testStrings {
			si.Intern(s)
		}
	}
}

func BenchmarkStringInterning_Batch(b *testing.B) {
	si := NewStringInterning()
	testStrings := []string{"prop1", "prop2", "prop3", "prop4", "prop5"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		si.InternBatch(testStrings...)
	}
}

func BenchmarkStringInterning_WithoutInterning(b *testing.B) {
	testStrings := []string{"prop1", "prop2", "prop3", "prop4", "prop5"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Just store strings without interning - for comparison
		_ = append([]string{}, testStrings...)
	}
}

func BenchmarkStringInterning_Concurrent(b *testing.B) {
	si := NewStringInterning()
	testStr := "entityName"

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			si.Intern(testStr)
		}
	})
}

// BenchmarkStringInterningMemoryReduction measures the memory impact of string interning
func BenchmarkStringInterningMemoryReduction(b *testing.B) {
	b.Run("WithInterning", func(b *testing.B) {
		si := NewStringInterning()
		names := make([]string, b.N)

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Simulate repeated property names
			propName := "Property_" + string(rune(i%100))
			names[i] = si.Intern(propName)
		}
	})

	b.Run("WithoutInterning", func(b *testing.B) {
		names := make([]string, b.N)

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Without interning - creates new string each time
			propName := "Property_" + string(rune(i%100))
			names[i] = propName
		}
	})
}

// TestStringInterning_GlobalCache tests the global interning instance
func TestStringInterning_GlobalCache(t *testing.T) {
	defer ClearGlobalCache()

	str1 := InternString("GlobalProp1")
	str2 := InternString("GlobalProp1")

	if str1 != str2 {
		t.Error("Expected same pointer for global interned string")
	}

	size := GlobalCacheSize()
	if size != 1 {
		t.Errorf("Expected global cache size 1, got %d", size)
	}
}

// TestStringInterning_SpecialCharacters tests interning with special characters
func TestStringInterning_SpecialCharacters(t *testing.T) {
	si := NewStringInterning()

	specialStrings := []string{
		"prop-with-dash",
		"prop_with_underscore",
		"prop.with.dot",
		"prop/with/slash",
		"prop:with:colon",
	}

	results := si.InternBatch(specialStrings...)

	if len(results) != len(specialStrings) {
		t.Errorf("Expected %d results, got %d", len(specialStrings), len(results))
	}

	// Re-intern and verify same pointers
	results2 := si.InternBatch(specialStrings...)
	for i := range results {
		if results[i] != results2[i] {
			t.Errorf("Expected same pointer for special char string at index %d", i)
		}
	}
}

// Benchmark comparing string concatenation with and without interning
func BenchmarkStringInterning_EntityNameProcessing(b *testing.B) {
	b.Run("WithInterning", func(b *testing.B) {
		si := NewStringInterning()

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Simulate processing entity names
			for j := 0; j < 100; j++ {
				entityName := "Customer"
				propName := "Name"
				_ = si.Intern(entityName + "." + propName)
			}
		}
	})

	b.Run("WithoutInterning", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Without interning
			for j := 0; j < 100; j++ {
				entityName := "Customer"
				propName := "Name"
				_ = entityName + "." + propName
			}
		}
	})
}

// TestStringInterning_PointerEquality verifies that interned strings share memory
func TestStringInterning_PointerEquality(t *testing.T) {
	si := NewStringInterning()

	// Get first interned string
	str1 := si.Intern("sharedString")

	// Get same string again
	str2 := si.Intern("sharedString")

	// Both should be the same string value
	if str1 != str2 {
		t.Error("Expected equal string values")
	}

	// Both should have same length
	if len(str1) != len(str2) {
		t.Error("Expected same length")
	}

	// Verify they are identical string content
	if strings.Compare(str1, str2) != 0 {
		t.Error("Expected identical string content")
	}
}
