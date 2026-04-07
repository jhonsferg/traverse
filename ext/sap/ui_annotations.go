package sap

import (
	"strconv"

	traverse "github.com/jhonsferg/traverse"
)

// SAPUIAnnotation holds SAP-specific Fiori UI annotations.
// These correspond to sap.ui, com.sap.vocabularies.UI.v1, and SAP.Semantics namespaces.
type SAPUIAnnotation struct {
	// Label is a human-readable label (sap:label or UI.v1.Label).
	Label string
	// Tooltip is a UI tooltip (sap:tooltip).
	Tooltip string
	// Visible controls field visibility; nil means unset (sap:visible or inverted UI.v1.Hidden).
	Visible *bool
	// Sortable indicates if the field is sortable (sap:sortable).
	Sortable *bool
	// Filterable indicates if the field can be used in filters (sap:filterable).
	Filterable *bool
	// Searchable indicates if the field participates in search (sap:searchable).
	Searchable *bool
	// DisplayFormat specifies the display format, e.g. UpperCase, NonNegative, Date (sap:display-format).
	DisplayFormat string
	// FieldControl specifies the field control mode: Mandatory, Optional, ReadOnly (sap:field-control).
	FieldControl string
	// IsKey is true when sap:key="true" is present.
	IsKey bool
	// Semantics is the semantic type of the value, e.g. email, phone, url (sap:semantics).
	Semantics string
	// MaxLength is the maximum character length (sap:maxlength).
	MaxLength int
	// Scale is the decimal precision (sap:scale).
	Scale int
	// Unit is the name of the unit-of-measure reference field (sap:unit).
	Unit string
	// Text is the name of the descriptive text reference field (sap:text).
	Text string
	// ValueList indicates how to fetch value help: standard or fixed-values (sap:value-list).
	ValueList string
}

// ParseSAPUIAnnotation extracts SAP UI annotations from an XML attribute map.
// The map keys are the XML attribute names (e.g. "sap:label", "sap:sortable").
func ParseSAPUIAnnotation(attrs map[string]string) SAPUIAnnotation {
	var a SAPUIAnnotation
	for k, v := range attrs {
		switch k {
		case "sap:label":
			a.Label = v
		case "sap:tooltip":
			a.Tooltip = v
		case "sap:visible":
			b := v == "true"
			a.Visible = &b
		case "sap:sortable":
			b := v == "true"
			a.Sortable = &b
		case "sap:filterable":
			b := v == "true"
			a.Filterable = &b
		case "sap:searchable":
			b := v == "true"
			a.Searchable = &b
		case "sap:display-format":
			a.DisplayFormat = v
		case "sap:field-control":
			a.FieldControl = v
		case "sap:key":
			a.IsKey = v == "true"
		case "sap:semantics":
			a.Semantics = v
		case "sap:maxlength":
			if n, err := strconv.Atoi(v); err == nil {
				a.MaxLength = n
			}
		case "sap:scale":
			if n, err := strconv.Atoi(v); err == nil {
				a.Scale = n
			}
		case "sap:unit":
			a.Unit = v
		case "sap:text":
			a.Text = v
		case "sap:value-list":
			a.ValueList = v
		}
	}
	return a
}

// AnnotatedProperty pairs a traverse.Property with its SAP UI annotation.
type AnnotatedProperty struct {
	Property   traverse.Property
	Annotation SAPUIAnnotation
}

// AnnotatedProperties returns all properties from an EntityType that have SAP UI annotations.
// A property is considered annotated when its SAP field carries at least a Label or any
// other non-zero SAP attribute already stored on the traverse.Property.SAP field.
// Callers can further filter by checking AnnotatedProperty.Annotation fields.
func AnnotatedProperties(entityType *traverse.EntityType, _ *traverse.Metadata) []AnnotatedProperty {
	if entityType == nil {
		return nil
	}
	var result []AnnotatedProperty
	for _, prop := range entityType.Properties {
		sap := prop.SAP
		if sap.Label == "" && !sap.Filterable && !sap.Sortable &&
			!sap.Searchable && !sap.Required && sap.Text == "" {
			continue
		}
		// Convert traverse.SAPAnnotations → SAPUIAnnotation so callers get a uniform type.
		ann := SAPUIAnnotation{
			Label:      sap.Label,
			Text:       sap.Text,
			Searchable: boolPtr(sap.Searchable),
			Sortable:   boolPtr(sap.Sortable),
			Filterable: boolPtr(sap.Filterable),
		}
		result = append(result, AnnotatedProperty{
			Property:   prop,
			Annotation: ann,
		})
	}
	return result
}

func boolPtr(b bool) *bool { return &b }
