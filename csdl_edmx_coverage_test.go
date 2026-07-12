package traverse

import (
	"encoding/json"
	"testing"
)

func TestParseCSDLEntitySet(t *testing.T) {
	rawMsg := make(map[string]json.RawMessage)
	rawMsg["$Type"] = json.RawMessage(`"#MyNamespace.Product"`)

	es := parseCSDLEntitySet("Products", "MyNamespace", rawMsg)
	if es.Name != "Products" {
		t.Errorf("Name = %q, want Products", es.Name)
	}
	if es.EntityTypeName != "Product" {
		t.Errorf("EntityTypeName = %q, want Product", es.EntityTypeName)
	}
}

func TestParseCSDLEntitySet_NoType(t *testing.T) {
	rawMsg := make(map[string]json.RawMessage)
	es := parseCSDLEntitySet("Items", "NS", rawMsg)
	if es.Name != "Items" {
		t.Errorf("Name = %q, want Items", es.Name)
	}
	if es.EntityTypeName != "" {
		t.Errorf("EntityTypeName = %q, want empty", es.EntityTypeName)
	}
}

func TestParseCSDLEntitySet_InvalidType(t *testing.T) {
	rawMsg := make(map[string]json.RawMessage)
	rawMsg["$Type"] = json.RawMessage(`123`) // not a string
	es := parseCSDLEntitySet("Bad", "NS", rawMsg)
	if es.EntityTypeName != "" {
		t.Errorf("EntityTypeName = %q, want empty for invalid type", es.EntityTypeName)
	}
}

func TestHasAnyCoreField(t *testing.T) {
	if hasAnyCoreField(CoreVocabulary{}) {
		t.Error("empty CoreVocabulary should return false")
	}
	if !hasAnyCoreField(CoreVocabulary{Description: "desc"}) {
		t.Error("CoreVocabulary with Description should return true")
	}
	if !hasAnyCoreField(CoreVocabulary{LongDescription: "long"}) {
		t.Error("CoreVocabulary with LongDescription should return true")
	}
	if !hasAnyCoreField(CoreVocabulary{IsLanguageDependent: true}) {
		t.Error("CoreVocabulary with IsLanguageDependent should return true")
	}
	if !hasAnyCoreField(CoreVocabulary{Immutable: true}) {
		t.Error("CoreVocabulary with Immutable should return true")
	}
	if !hasAnyCoreField(CoreVocabulary{Computed: true}) {
		t.Error("CoreVocabulary with Computed should return true")
	}
	if !hasAnyCoreField(CoreVocabulary{Permissions: []string{"read"}}) {
		t.Error("CoreVocabulary with Permissions should return true")
	}
	if !hasAnyCoreField(CoreVocabulary{Example: "ex"}) {
		t.Error("CoreVocabulary with Example should return true")
	}
}

func TestHasAnyValidationField(t *testing.T) {
	if hasAnyValidationField(ValidationVocabulary{}) {
		t.Error("empty ValidationVocabulary should return false")
	}
	minVal := 1.0
	if !hasAnyValidationField(ValidationVocabulary{Minimum: &minVal}) {
		t.Error("ValidationVocabulary with Minimum should return true")
	}
	maxVal := 100.0
	if !hasAnyValidationField(ValidationVocabulary{Maximum: &maxVal}) {
		t.Error("ValidationVocabulary with Maximum should return true")
	}
	if !hasAnyValidationField(ValidationVocabulary{Pattern: "^[a-z]+$"}) {
		t.Error("ValidationVocabulary with Pattern should return true")
	}
	if !hasAnyValidationField(ValidationVocabulary{AllowedValues: []string{"a", "b"}}) {
		t.Error("ValidationVocabulary with AllowedValues should return true")
	}
	if !hasAnyValidationField(ValidationVocabulary{Required: true}) {
		t.Error("ValidationVocabulary with Required should return true")
	}
}

func TestHasAnyMeasuresField(t *testing.T) {
	if hasAnyMeasuresField(MeasuresVocabulary{}) {
		t.Error("empty MeasuresVocabulary should return false")
	}
	if !hasAnyMeasuresField(MeasuresVocabulary{ISOCurrency: "USD"}) {
		t.Error("MeasuresVocabulary with ISOCurrency should return true")
	}
	scaleVal := 2
	if !hasAnyMeasuresField(MeasuresVocabulary{Scale: &scaleVal}) {
		t.Error("MeasuresVocabulary with Scale should return true")
	}
	if !hasAnyMeasuresField(MeasuresVocabulary{Unit: "kg"}) {
		t.Error("MeasuresVocabulary with Unit should return true")
	}
	if !hasAnyMeasuresField(MeasuresVocabulary{SIPrefix: "kilo"}) {
		t.Error("MeasuresVocabulary with SIPrefix should return true")
	}
	if !hasAnyMeasuresField(MeasuresVocabulary{DurationGranularity: "seconds"}) {
		t.Error("MeasuresVocabulary with DurationGranularity should return true")
	}
}

func TestHasAnyAnalyticsField(t *testing.T) {
	if hasAnyAnalyticsField(AnalyticsVocabulary{}) {
		t.Error("empty AnalyticsVocabulary should return false")
	}
	if !hasAnyAnalyticsField(AnalyticsVocabulary{AggregationMethod: "sum"}) {
		t.Error("AnalyticsVocabulary with AggregationMethod should return true")
	}
	if !hasAnyAnalyticsField(AnalyticsVocabulary{IsDimension: true}) {
		t.Error("AnalyticsVocabulary with IsDimension should return true")
	}
	if !hasAnyAnalyticsField(AnalyticsVocabulary{IsMeasure: true}) {
		t.Error("AnalyticsVocabulary with IsMeasure should return true")
	}
	if !hasAnyAnalyticsField(AnalyticsVocabulary{RollupLevels: 3}) {
		t.Error("AnalyticsVocabulary with RollupLevels should return true")
	}
	if !hasAnyAnalyticsField(AnalyticsVocabulary{ReferencedProperties: []string{"a"}}) {
		t.Error("AnalyticsVocabulary with ReferencedProperties should return true")
	}
	if !hasAnyAnalyticsField(AnalyticsVocabulary{GroupableProperties: []string{"a"}}) {
		t.Error("AnalyticsVocabulary with GroupableProperties should return true")
	}
	if !hasAnyAnalyticsField(AnalyticsVocabulary{AggregatableProperties: []string{"a"}}) {
		t.Error("AnalyticsVocabulary with AggregatableProperties should return true")
	}
}
