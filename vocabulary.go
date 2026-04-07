package traverse

import (
	"strconv"
	"strings"
)

// CoreVocabulary defines commonly used Org.OData.Core.V1 annotation terms.
// These match the official Org.OData.Core.V1 namespace.
type CoreVocabulary struct {
	// Description corresponds to Org.OData.Core.V1.Description.
	Description string
	// LongDescription corresponds to Org.OData.Core.V1.LongDescription.
	LongDescription string
	// IsLanguageDependent corresponds to Org.OData.Core.V1.IsLanguageDependent.
	IsLanguageDependent bool
	// Immutable corresponds to Org.OData.Core.V1.Immutable.
	Immutable bool
	// Computed corresponds to Org.OData.Core.V1.Computed.
	Computed bool
	// Permissions corresponds to Org.OData.Core.V1.Permissions (e.g., Read, Write).
	Permissions []string
	// Example corresponds to Org.OData.Core.V1.Example (for documentation purposes).
	Example string
}

// ValidationVocabulary defines Org.OData.Validation.V1 annotation terms.
type ValidationVocabulary struct {
	// Minimum corresponds to Org.OData.Validation.V1.Minimum.
	Minimum *float64
	// Maximum corresponds to Org.OData.Validation.V1.Maximum.
	Maximum *float64
	// Pattern corresponds to Org.OData.Validation.V1.Pattern.
	Pattern string
	// AllowedValues corresponds to Org.OData.Validation.V1.AllowedValues.
	AllowedValues []string
	// Required corresponds to Org.OData.Validation.V1.Required.
	Required bool
}

const (
	corePrefix       = "Org.OData.Core.V1."
	validationPrefix = "Org.OData.Validation.V1."
)

// ParseCoreVocabulary extracts Org.OData.Core.V1 annotation terms from a raw annotation map.
// The map keys are fully-qualified term names (e.g. "Org.OData.Core.V1.Description").
func ParseCoreVocabulary(annotations map[string]string) CoreVocabulary {
	var v CoreVocabulary
	for k, val := range annotations {
		if !strings.HasPrefix(k, corePrefix) {
			continue
		}
		term := k[len(corePrefix):]
		switch term {
		case "Description":
			v.Description = val
		case "LongDescription":
			v.LongDescription = val
		case "IsLanguageDependent":
			v.IsLanguageDependent = val == "true"
		case "Immutable":
			v.Immutable = val == "true"
		case "Computed":
			v.Computed = val == "true"
		case "Permissions":
			if val != "" {
				v.Permissions = strings.Split(val, ",")
			}
		case "Example":
			v.Example = val
		}
	}
	return v
}

// ParseValidationVocabulary extracts Org.OData.Validation.V1 annotation terms from a raw annotation map.
// The map keys are fully-qualified term names (e.g. "Org.OData.Validation.V1.Minimum").
func ParseValidationVocabulary(annotations map[string]string) ValidationVocabulary {
	var v ValidationVocabulary
	for k, val := range annotations {
		if !strings.HasPrefix(k, validationPrefix) {
			continue
		}
		term := k[len(validationPrefix):]
		switch term {
		case "Minimum":
			if f, err := strconv.ParseFloat(val, 64); err == nil {
				v.Minimum = &f
			}
		case "Maximum":
			if f, err := strconv.ParseFloat(val, 64); err == nil {
				v.Maximum = &f
			}
		case "Pattern":
			v.Pattern = val
		case "AllowedValues":
			if val != "" {
				v.AllowedValues = strings.Split(val, ",")
			}
		case "Required":
			v.Required = val == "true"
		}
	}
	return v
}
