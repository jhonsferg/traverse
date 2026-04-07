package traverse

import (
	"testing"
)

func TestParseCoreVocabulary_Description(t *testing.T) {
	annotations := map[string]string{
		"Org.OData.Core.V1.Description": "Hello",
	}
	v := ParseCoreVocabulary(annotations)
	if v.Description != "Hello" {
		t.Errorf("expected Description %q, got %q", "Hello", v.Description)
	}
}

func TestParseCoreVocabulary_Computed(t *testing.T) {
	annotations := map[string]string{
		"Org.OData.Core.V1.Computed": "true",
	}
	v := ParseCoreVocabulary(annotations)
	if !v.Computed {
		t.Error("expected Computed to be true")
	}
}

func TestParseCoreVocabulary_AllFields(t *testing.T) {
	annotations := map[string]string{
		"Org.OData.Core.V1.Description":         "desc",
		"Org.OData.Core.V1.LongDescription":     "long desc",
		"Org.OData.Core.V1.IsLanguageDependent": "true",
		"Org.OData.Core.V1.Immutable":           "true",
		"Org.OData.Core.V1.Computed":            "true",
		"Org.OData.Core.V1.Permissions":         "Read,Write",
		"Org.OData.Core.V1.Example":             "example-value",
	}
	v := ParseCoreVocabulary(annotations)
	if v.Description != "desc" {
		t.Errorf("Description: got %q", v.Description)
	}
	if v.LongDescription != "long desc" {
		t.Errorf("LongDescription: got %q", v.LongDescription)
	}
	if !v.IsLanguageDependent {
		t.Error("IsLanguageDependent should be true")
	}
	if !v.Immutable {
		t.Error("Immutable should be true")
	}
	if !v.Computed {
		t.Error("Computed should be true")
	}
	if len(v.Permissions) != 2 || v.Permissions[0] != "Read" || v.Permissions[1] != "Write" {
		t.Errorf("Permissions: got %v", v.Permissions)
	}
	if v.Example != "example-value" {
		t.Errorf("Example: got %q", v.Example)
	}
}

func TestParseCoreVocabulary_Empty(t *testing.T) {
	v := ParseCoreVocabulary(map[string]string{})
	if v.Description != "" || v.Computed || v.Immutable {
		t.Error("empty map should produce zero-value CoreVocabulary")
	}
}

func TestParseValidationVocabulary_MinMax(t *testing.T) {
	annotations := map[string]string{
		"Org.OData.Validation.V1.Minimum": "0",
		"Org.OData.Validation.V1.Maximum": "100",
	}
	v := ParseValidationVocabulary(annotations)
	if v.Minimum == nil || *v.Minimum != 0 {
		t.Errorf("expected Minimum 0, got %v", v.Minimum)
	}
	if v.Maximum == nil || *v.Maximum != 100 {
		t.Errorf("expected Maximum 100, got %v", v.Maximum)
	}
}

func TestParseValidationVocabulary_Pattern(t *testing.T) {
	annotations := map[string]string{
		"Org.OData.Validation.V1.Pattern": `^[A-Z]+$`,
	}
	v := ParseValidationVocabulary(annotations)
	if v.Pattern != `^[A-Z]+$` {
		t.Errorf("expected Pattern %q, got %q", `^[A-Z]+$`, v.Pattern)
	}
}

func TestParseValidationVocabulary_AllFields(t *testing.T) {
	annotations := map[string]string{
		"Org.OData.Validation.V1.Minimum":       "1.5",
		"Org.OData.Validation.V1.Maximum":       "99.9",
		"Org.OData.Validation.V1.Pattern":       `\d+`,
		"Org.OData.Validation.V1.AllowedValues": "A,B,C",
		"Org.OData.Validation.V1.Required":      "true",
	}
	v := ParseValidationVocabulary(annotations)
	if v.Minimum == nil || *v.Minimum != 1.5 {
		t.Errorf("Minimum: got %v", v.Minimum)
	}
	if v.Maximum == nil || *v.Maximum != 99.9 {
		t.Errorf("Maximum: got %v", v.Maximum)
	}
	if v.Pattern != `\d+` {
		t.Errorf("Pattern: got %q", v.Pattern)
	}
	if len(v.AllowedValues) != 3 {
		t.Errorf("AllowedValues: got %v", v.AllowedValues)
	}
	if !v.Required {
		t.Error("Required should be true")
	}
}

func TestParseValidationVocabulary_Empty(t *testing.T) {
	v := ParseValidationVocabulary(map[string]string{})
	if v.Minimum != nil || v.Maximum != nil || v.Pattern != "" || v.Required {
		t.Error("empty map should produce zero-value ValidationVocabulary")
	}
}
