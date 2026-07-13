package sap

import (
	"testing"

	traverse "github.com/jhonsferg/traverse"
)

func TestAnnotatedProperties_NilEntityType(t *testing.T) {
	if got := AnnotatedProperties(nil, nil); got != nil {
		t.Errorf("AnnotatedProperties(nil, nil) = %v, want nil", got)
	}
}

func TestAnnotatedProperties_NoAnnotations(t *testing.T) {
	et := &traverse.EntityType{
		Name: "Product",
		Properties: []traverse.Property{
			{Name: "ID"},
			{Name: "Name"},
		},
	}
	got := AnnotatedProperties(et, nil)
	if got != nil {
		t.Errorf("expected nil for properties with no SAP annotations, got %v", got)
	}
}

func TestAnnotatedProperties_WithAnnotations(t *testing.T) {
	et := &traverse.EntityType{
		Name: "Product",
		Properties: []traverse.Property{
			{Name: "ID"},
			{
				Name: "Price",
				SAP: traverse.SAPAnnotations{
					Label:      "Price",
					Filterable: true,
					Sortable:   true,
					Searchable: false,
				},
			},
			{
				Name: "Description",
				SAP: traverse.SAPAnnotations{
					Text: "descriptive text",
				},
			},
		},
	}
	got := AnnotatedProperties(et, nil)
	if len(got) != 2 {
		t.Fatalf("expected 2 annotated properties, got %d", len(got))
	}
	if got[0].Property.Name != "Price" {
		t.Errorf("got[0].Property.Name = %q, want %q", got[0].Property.Name, "Price")
	}
	if got[0].Annotation.Label != "Price" {
		t.Errorf("got[0].Annotation.Label = %q, want %q", got[0].Annotation.Label, "Price")
	}
	if got[0].Annotation.Filterable == nil || !*got[0].Annotation.Filterable {
		t.Error("expected Filterable = true")
	}
	if got[0].Annotation.Searchable == nil || *got[0].Annotation.Searchable {
		t.Error("expected Searchable = false (non-nil pointer)")
	}
	if got[1].Property.Name != "Description" {
		t.Errorf("got[1].Property.Name = %q, want %q", got[1].Property.Name, "Description")
	}
}
