package sap

import (
	"testing"
)

func TestParseSAPUIAnnotation_BasicLabel(t *testing.T) {
	attrs := map[string]string{
		"sap:label": "Customer Name",
	}
	a := ParseSAPUIAnnotation(attrs)
	if a.Label != "Customer Name" {
		t.Errorf("expected Label %q, got %q", "Customer Name", a.Label)
	}
}

func TestParseSAPUIAnnotation_Sortable_Filterable_Visible(t *testing.T) {
	attrs := map[string]string{
		"sap:sortable":   "true",
		"sap:filterable": "false",
		"sap:visible":    "true",
	}
	a := ParseSAPUIAnnotation(attrs)
	if a.Sortable == nil || !*a.Sortable {
		t.Error("expected Sortable to be true")
	}
	if a.Filterable == nil || *a.Filterable {
		t.Error("expected Filterable to be false")
	}
	if a.Visible == nil || !*a.Visible {
		t.Error("expected Visible to be true")
	}
}

func TestParseSAPUIAnnotation_SemanticEmail(t *testing.T) {
	attrs := map[string]string{
		"sap:semantics": "email",
		"sap:label":     "Email Address",
	}
	a := ParseSAPUIAnnotation(attrs)
	if a.Semantics != "email" {
		t.Errorf("expected Semantics %q, got %q", "email", a.Semantics)
	}
	if a.Label != "Email Address" {
		t.Errorf("expected Label %q, got %q", "Email Address", a.Label)
	}
}

func TestParseSAPUIAnnotation_NilOnEmpty(t *testing.T) {
	a := ParseSAPUIAnnotation(map[string]string{})
	if a.Label != "" || a.Semantics != "" || a.Sortable != nil || a.Filterable != nil || a.Visible != nil {
		t.Error("empty attrs should produce zero-value SAPUIAnnotation")
	}
}

func TestParseSAPUIAnnotation_MaxLengthScale(t *testing.T) {
	attrs := map[string]string{
		"sap:maxlength": "50",
		"sap:scale":     "2",
	}
	a := ParseSAPUIAnnotation(attrs)
	if a.MaxLength != 50 {
		t.Errorf("expected MaxLength 50, got %d", a.MaxLength)
	}
	if a.Scale != 2 {
		t.Errorf("expected Scale 2, got %d", a.Scale)
	}
}

func TestParseSAPUIAnnotation_UnitText(t *testing.T) {
	attrs := map[string]string{
		"sap:unit": "Currency",
		"sap:text": "ProductName",
	}
	a := ParseSAPUIAnnotation(attrs)
	if a.Unit != "Currency" {
		t.Errorf("expected Unit %q, got %q", "Currency", a.Unit)
	}
	if a.Text != "ProductName" {
		t.Errorf("expected Text %q, got %q", "ProductName", a.Text)
	}
}

func TestParseSAPUIAnnotation_FieldControl(t *testing.T) {
	attrs := map[string]string{
		"sap:field-control":  "Mandatory",
		"sap:display-format": "UpperCase",
	}
	a := ParseSAPUIAnnotation(attrs)
	if a.FieldControl != "Mandatory" {
		t.Errorf("expected FieldControl %q, got %q", "Mandatory", a.FieldControl)
	}
	if a.DisplayFormat != "UpperCase" {
		t.Errorf("expected DisplayFormat %q, got %q", "UpperCase", a.DisplayFormat)
	}
}

func TestParseSAPUIAnnotation_IsKey(t *testing.T) {
	attrs := map[string]string{
		"sap:key": "true",
	}
	a := ParseSAPUIAnnotation(attrs)
	if !a.IsKey {
		t.Error("expected IsKey to be true")
	}
}

func TestParseSAPUIAnnotation_ValueList(t *testing.T) {
	attrs := map[string]string{
		"sap:value-list": "fixed-values",
	}
	a := ParseSAPUIAnnotation(attrs)
	if a.ValueList != "fixed-values" {
		t.Errorf("expected ValueList %q, got %q", "fixed-values", a.ValueList)
	}
}
