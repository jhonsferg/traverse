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

// MeasuresVocabulary defines Org.OData.Measures.V1 annotation terms.
//
// The Measures vocabulary annotates properties with unit-of-measure semantics.
// These annotations are defined in the OASIS OData Measures vocabulary
// (namespace: Org.OData.Measures.V1).
type MeasuresVocabulary struct {
	// ISOCurrency is the ISO 4217 currency code for a monetary amount property.
	// Corresponds to Org.OData.Measures.V1.ISOCurrency.
	// The value may be a literal currency code (e.g. "USD") or a path to a property
	// that holds the currency code.
	ISOCurrency string
	// Scale is the number of significant decimal places for a currency or measure value.
	// Corresponds to Org.OData.Measures.V1.Scale.
	Scale *int
	// Unit is the unit of measure for the property value.
	// Corresponds to Org.OData.Measures.V1.Unit.
	// The value may be a literal unit symbol or a path to a property that holds it.
	Unit string
	// SIPrefix is the SI prefix that multiplies the unit (e.g. "Kilo", "Mega", "Milli").
	// Corresponds to Org.OData.Measures.V1.SIPrefix.
	SIPrefix string
	// DurationGranularity is the minimum granularity of duration values (e.g. "days", "hours").
	// Corresponds to Org.OData.Measures.V1.DurationGranularity.
	DurationGranularity string
}

// AuthorizationVocabulary defines Org.OData.Authorization.V1 annotation terms.
//
// The Authorization vocabulary annotates services and operations with the security
// schemes required to access them. These annotations are defined in the OASIS OData
// Authorization vocabulary (namespace: Org.OData.Authorization.V1).
type AuthorizationVocabulary struct {
	// Authorizations is the list of named authorization schemes that apply.
	// Corresponds to Org.OData.Authorization.V1.Authorizations.
	// Each entry is the Name of a SecurityScheme defined on the service.
	Authorizations []string
	// RequiredScopes is the list of OAuth2/OpenID scopes required.
	// Corresponds to Org.OData.Authorization.V1.RequiredScopes.
	RequiredScopes []string
	// SecuritySchemeType identifies the type of scheme: "ApiKey", "Http", "OAuth2", "OpenIDConnect".
	// Set on service-level annotations that define the security scheme itself.
	SecuritySchemeType string
	// KeyName is the name of the API key parameter (for ApiKey schemes).
	// Corresponds to Org.OData.Authorization.V1.KeyName.
	KeyName string
	// KeyLocation is where the API key is sent: "header", "query", "cookie" (for ApiKey schemes).
	// Corresponds to Org.OData.Authorization.V1.Location.
	KeyLocation string
	// Scheme is the HTTP authentication scheme (for Http type), e.g. "bearer", "basic".
	// Corresponds to Org.OData.Authorization.V1.Scheme.
	Scheme string
	// BearerFormat describes the bearer token format (for Http+bearer), e.g. "JWT".
	// Corresponds to Org.OData.Authorization.V1.BearerFormat.
	BearerFormat string
	// AuthorizationURL is the OAuth2 authorization endpoint URL.
	// Corresponds to Org.OData.Authorization.V1.AuthorizationURL.
	AuthorizationURL string
	// TokenURL is the OAuth2 token endpoint URL.
	// Corresponds to Org.OData.Authorization.V1.TokenURL.
	TokenURL string
	// OpenIDConnectURL is the OpenID Connect discovery document URL.
	// Corresponds to Org.OData.Authorization.V1.OpenIDConnectUrl.
	OpenIDConnectURL string
}

// AnalyticsVocabulary defines Org.OData.Aggregation.V1 annotation terms for analytics.
//
// The Aggregation (Analytics) vocabulary annotates entity types and properties with
// analytical semantics, indicating which properties are dimensions, measures, or
// aggregation methods. These annotations are defined in the OASIS OData Aggregation
// vocabulary (namespace: Org.OData.Aggregation.V1), and also appear in the
// SAP Analytics vocabulary (namespace: com.sap.vocabularies.Analytics.v1).
type AnalyticsVocabulary struct {
	// AggregationMethod is the aggregation function to apply to this property.
	// Common values: "sum", "min", "max", "average", "count", "countdistinct".
	// Corresponds to Org.OData.Aggregation.V1.default.
	AggregationMethod string
	// IsDimension is true when the property is an analytical dimension.
	// Corresponds to Org.OData.Aggregation.V1.Dimensionality="Dimension".
	IsDimension bool
	// IsMeasure is true when the property is an analytical measure.
	// Corresponds to Org.OData.Aggregation.V1.Dimensionality="Measure".
	IsMeasure bool
	// RollupLevels specifies the hierarchy levels for dimension rollup.
	// Corresponds to Org.OData.Aggregation.V1.RollupLevels.
	RollupLevels int
	// ReferencedProperties lists properties that this aggregation references.
	// Corresponds to Org.OData.Aggregation.V1.ReferencedProperties.
	ReferencedProperties []string
	// GroupableProperties lists the properties that can be used in $apply groupby().
	// Corresponds to Org.OData.Aggregation.V1.GroupableProperties.
	GroupableProperties []string
	// AggregatableProperties lists properties that can be aggregated via $apply aggregate().
	// Corresponds to Org.OData.Aggregation.V1.AggregatableProperties.
	AggregatableProperties []string
}

const (
	corePrefix          = "Org.OData.Core.V1."
	validationPrefix    = "Org.OData.Validation.V1."
	measuresPrefix      = "Org.OData.Measures.V1."
	authorizationPrefix = "Org.OData.Authorization.V1."
	aggregationPrefix   = "Org.OData.Aggregation.V1."
	sapAnalyticsPrefix  = "com.sap.vocabularies.Analytics.v1."
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

// ParseMeasuresVocabulary extracts Org.OData.Measures.V1 annotation terms from a raw annotation map.
// The map keys are fully-qualified term names (e.g. "Org.OData.Measures.V1.ISOCurrency").
//
// Example annotations map:
//
//	map[string]string{
//	    "Org.OData.Measures.V1.ISOCurrency": "USD",
//	    "Org.OData.Measures.V1.Scale":       "2",
//	    "Org.OData.Measures.V1.Unit":        "kg",
//	}
func ParseMeasuresVocabulary(annotations map[string]string) MeasuresVocabulary {
	var v MeasuresVocabulary
	for k, val := range annotations {
		if !strings.HasPrefix(k, measuresPrefix) {
			continue
		}
		term := k[len(measuresPrefix):]
		switch term {
		case "ISOCurrency":
			v.ISOCurrency = val
		case "Scale":
			if n, err := strconv.Atoi(val); err == nil {
				v.Scale = &n
			}
		case "Unit":
			v.Unit = val
		case "SIPrefix":
			v.SIPrefix = val
		case "DurationGranularity":
			v.DurationGranularity = val
		}
	}
	return v
}

// ParseAuthorizationVocabulary extracts Org.OData.Authorization.V1 annotation terms
// from a raw annotation map.
// The map keys are fully-qualified term names (e.g. "Org.OData.Authorization.V1.Authorizations").
//
// Example annotations map:
//
//	map[string]string{
//	    "Org.OData.Authorization.V1.Authorizations":  "OAuth2Implicit",
//	    "Org.OData.Authorization.V1.RequiredScopes":  "read:data,write:data",
//	    "Org.OData.Authorization.V1.Scheme":          "bearer",
//	    "Org.OData.Authorization.V1.BearerFormat":    "JWT",
//	}
func ParseAuthorizationVocabulary(annotations map[string]string) AuthorizationVocabulary {
	var v AuthorizationVocabulary
	for k, val := range annotations {
		if !strings.HasPrefix(k, authorizationPrefix) {
			continue
		}
		term := k[len(authorizationPrefix):]
		switch term {
		case "Authorizations":
			if val != "" {
				v.Authorizations = strings.Split(val, ",")
			}
		case "RequiredScopes":
			if val != "" {
				v.RequiredScopes = strings.Split(val, ",")
			}
		case "SecuritySchemeType":
			v.SecuritySchemeType = val
		case "KeyName":
			v.KeyName = val
		case "Location":
			v.KeyLocation = val
		case "Scheme":
			v.Scheme = val
		case "BearerFormat":
			v.BearerFormat = val
		case "AuthorizationURL":
			v.AuthorizationURL = val
		case "TokenURL":
			v.TokenURL = val
		case "OpenIDConnectUrl":
			v.OpenIDConnectURL = val
		}
	}
	return v
}

// ParseAnalyticsVocabulary extracts Org.OData.Aggregation.V1 (and SAP Analytics v1)
// annotation terms from a raw annotation map.
//
// Both the OASIS aggregation namespace and the SAP-specific analytics namespace are
// recognised, so annotations from either source populate the same struct.
//
// Example annotations map:
//
//	map[string]string{
//	    "Org.OData.Aggregation.V1.default":               "sum",
//	    "Org.OData.Aggregation.V1.RollupLevels":          "3",
//	    "com.sap.vocabularies.Analytics.v1.Dimension":    "true",
//	    "com.sap.vocabularies.Analytics.v1.Measure":      "false",
//	}
func ParseAnalyticsVocabulary(annotations map[string]string) AnalyticsVocabulary {
	var v AnalyticsVocabulary
	for k, val := range annotations {
		switch {
		case strings.HasPrefix(k, aggregationPrefix):
			term := k[len(aggregationPrefix):]
			switch term {
			case "default":
				v.AggregationMethod = val
			case "Dimensionality":
				switch val {
				case "Dimension":
					v.IsDimension = true
				case "Measure":
					v.IsMeasure = true
				}
			case "RollupLevels":
				if n, err := strconv.Atoi(val); err == nil {
					v.RollupLevels = n
				}
			case "ReferencedProperties":
				if val != "" {
					v.ReferencedProperties = strings.Split(val, ",")
				}
			case "GroupableProperties":
				if val != "" {
					v.GroupableProperties = strings.Split(val, ",")
				}
			case "AggregatableProperties":
				if val != "" {
					v.AggregatableProperties = strings.Split(val, ",")
				}
			}
		case strings.HasPrefix(k, sapAnalyticsPrefix):
			term := k[len(sapAnalyticsPrefix):]
			switch term {
			case "Dimension":
				v.IsDimension = val == "true"
			case "Measure":
				v.IsMeasure = val == "true"
			case "AggregationMethod":
				v.AggregationMethod = val
			case "RollupLevels":
				if n, err := strconv.Atoi(val); err == nil {
					v.RollupLevels = n
				}
			}
		}
	}
	return v
}
