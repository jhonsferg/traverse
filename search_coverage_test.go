package traverse

import (
	"testing"
)

func TestSearchWord_Basic(t *testing.T) {
	expr := SearchWord("mountain")
	got := expr.searchString()
	if got != "mountain" {
		t.Errorf("SearchWord('mountain').searchString() = %q, want %q", got, "mountain")
	}
}

func TestSearchPhrase_Basic(t *testing.T) {
	expr := SearchPhrase("mountain bike")
	got := expr.searchString()
	if got != `"mountain bike"` {
		t.Errorf("SearchPhrase('mountain bike').searchString() = %q, want %q", got, `"mountain bike"`)
	}
}

func TestSearchPhrase_EscapesInternalQuotes(t *testing.T) {
	expr := SearchPhrase(`say "hi"`)
	got := expr.searchString()
	expected := `"say \"hi\""`
	if got != expected {
		t.Errorf("SearchPhrase escaped = %q, want %q", got, expected)
	}
}

func TestSearch_AutoDetectWord(t *testing.T) {
	expr := Search("mountain")
	got := expr.searchString()
	if got != "mountain" {
		t.Errorf("Search('mountain').searchString() = %q, want %q", got, "mountain")
	}
}

func TestSearch_AutoDetectPhrase(t *testing.T) {
	expr := Search("mountain bike")
	got := expr.searchString()
	if got != `"mountain bike"` {
		t.Errorf("Search('mountain bike').searchString() = %q, want %q", got, `"mountain bike"`)
	}
}

func TestSearchAnd_TwoExpressions(t *testing.T) {
	expr := SearchAnd(SearchWord("mountain"), SearchWord("view"))
	got := expr.searchString()
	if got != "(mountain AND view)" {
		t.Errorf("SearchAnd = %q, want %q", got, "(mountain AND view)")
	}
}

func TestSearchAnd_ThreeExpressions(t *testing.T) {
	expr := SearchAnd(SearchWord("a"), SearchWord("b"), SearchWord("c"))
	got := expr.searchString()
	if got != "(a AND b AND c)" {
		t.Errorf("SearchAnd = %q, want %q", got, "(a AND b AND c)")
	}
}

func TestSearchOr_TwoExpressions(t *testing.T) {
	expr := SearchOr(SearchWord("mountain"), SearchWord("hill"))
	got := expr.searchString()
	if got != "(mountain OR hill)" {
		t.Errorf("SearchOr = %q, want %q", got, "(mountain OR hill)")
	}
}

func TestSearchOr_ThreeExpressions(t *testing.T) {
	expr := SearchOr(SearchWord("a"), SearchWord("b"), SearchWord("c"))
	got := expr.searchString()
	if got != "(a OR b OR c)" {
		t.Errorf("SearchOr = %q, want %q", got, "(a OR b OR c)")
	}
}

func TestSearchNot_Basic(t *testing.T) {
	expr := SearchNot(SearchWord("bike"))
	got := expr.searchString()
	if got != "NOT bike" {
		t.Errorf("SearchNot = %q, want %q", got, "NOT bike")
	}
}

func TestSearchNot_WithPhrase(t *testing.T) {
	expr := SearchNot(SearchPhrase("mountain bike"))
	got := expr.searchString()
	if got != `NOT "mountain bike"` {
		t.Errorf("SearchNot phrase = %q, want %q", got, `NOT "mountain bike"`)
	}
}

func TestSearchAnd_Complex(t *testing.T) {
	expr := SearchAnd(
		SearchWord("mountain"),
		SearchNot(SearchWord("bike")),
	)
	got := expr.searchString()
	if got != "(mountain AND NOT bike)" {
		t.Errorf("SearchAnd complex = %q, want %q", got, "(mountain AND NOT bike)")
	}
}

func TestSearchOr_Complex(t *testing.T) {
	expr := SearchOr(
		SearchWord("mountain"),
		SearchWord("hill"),
		SearchWord("valley"),
	)
	got := expr.searchString()
	if got != "(mountain OR hill OR valley)" {
		t.Errorf("SearchOr complex = %q, want %q", got, "(mountain OR hill OR valley)")
	}
}

func TestCopyMapDeep_Basic(t *testing.T) {
	src := map[string]interface{}{
		"key1": "value1",
		"key2": 42,
		"key3": true,
	}

	dst := copyMapDeep(src)

	if len(dst) != len(src) {
		t.Errorf("copyMapDeep length = %d, want %d", len(dst), len(src))
	}

	for k, v := range src {
		if dst[k] != v {
			t.Errorf("copyMapDeep[%s] = %v, want %v", k, dst[k], v)
		}
	}

	// Modify src and verify dst is independent
	src["key1"] = "modified"
	if dst["key1"] == "modified" {
		t.Error("copyMapDeep should create independent copy")
	}
}

func TestCopyMapDeep_Nil(t *testing.T) {
	dst := copyMapDeep(nil)
	if dst != nil {
		t.Errorf("copyMapDeep(nil) = %v, want nil", dst)
	}
}

func TestEstimateBufferSize_Zero(t *testing.T) {
	got := estimateBufferSize(0)
	if got != 256 {
		t.Errorf("estimateBufferSize(0) = %d, want 256", got)
	}
}

func TestEstimateBufferSize_Negative(t *testing.T) {
	got := estimateBufferSize(-1)
	if got != 256 {
		t.Errorf("estimateBufferSize(-1) = %d, want 256", got)
	}
}

func TestEstimateBufferSize_Small(t *testing.T) {
	got := estimateBufferSize(100)
	if got != 1024 {
		t.Errorf("estimateBufferSize(100) = %d, want 1024", got)
	}
}

func TestEstimateBufferSize_Medium(t *testing.T) {
	got := estimateBufferSize(1024)
	if got != 1024 {
		t.Errorf("estimateBufferSize(1024) = %d, want 1024", got)
	}
}

func TestResetMapForPool_Small(t *testing.T) {
	m := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
	}
	result := resetMapForPool(m)
	if len(result) != 0 {
		t.Errorf("resetMapForPool should clear map, got %d entries", len(result))
	}
}

func TestResetMapForPool_Large(t *testing.T) {
	m := make(map[string]interface{}, 600)
	for i := 0; i < 600; i++ {
		m[string(rune(i))] = i
	}
	result := resetMapForPool(m)
	if len(result) != 0 {
		t.Errorf("resetMapForPool should return empty map, got %d entries", len(result))
	}
}

func TestReturnPageToPool(t *testing.T) {
	page := &Page{
		Value: []map[string]interface{}{
			{"key1": "value1"},
			{"key2": "value2"},
			{"key3": "value3"},
		},
	}
	returnPageToPool(page, 2)
	if len(page.Value) != 3 {
		t.Errorf("returnPageToPool should not modify page, got %d entries", len(page.Value))
	}
}
