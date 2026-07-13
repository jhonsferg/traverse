package sap

import (
	"testing"

	traverse "github.com/jhonsferg/traverse"
)

func TestEntitySetAnnotation_NilMetadata(t *testing.T) {
	ann := EntitySetAnnotation(nil, "MaterialSet")
	if !ann.Creatable || !ann.Updatable || !ann.Deletable || !ann.Pageable || !ann.Addressable {
		t.Errorf("expected all-true defaults for nil metadata, got %+v", ann)
	}
	if ann.RequiresFilter || ann.ChangeTracking {
		t.Errorf("expected RequiresFilter/ChangeTracking to default false, got %+v", ann)
	}
}

func TestEntitySetAnnotation_NotFound(t *testing.T) {
	meta := &traverse.Metadata{
		EntitySets: []traverse.EntitySetInfo{
			{Name: "OtherSet"},
		},
	}
	ann := EntitySetAnnotation(meta, "MaterialSet")
	if !ann.Creatable || !ann.Updatable {
		t.Errorf("expected defaults when entity set not found, got %+v", ann)
	}
}

func TestEntitySetAnnotation_Found(t *testing.T) {
	meta := &traverse.Metadata{
		EntitySets: []traverse.EntitySetInfo{
			{
				Name: "MaterialSet",
				SAP: traverse.SAPAnnotations{
					Label:          "Materials",
					Creatable:      false,
					Updatable:      true,
					Deletable:      false,
					Pageable:       true,
					Addressable:    true,
					RequiresFilter: true,
					ChangeTracking: true,
				},
			},
		},
	}
	ann := EntitySetAnnotation(meta, "MaterialSet")
	if ann.Label != "Materials" {
		t.Errorf("Label = %q, want %q", ann.Label, "Materials")
	}
	if ann.Creatable {
		t.Error("expected Creatable = false")
	}
	if !ann.RequiresFilter || !ann.ChangeTracking {
		t.Errorf("expected RequiresFilter and ChangeTracking = true, got %+v", ann)
	}
}

func TestAllEntitySetAnnotations_NilMetadata(t *testing.T) {
	if got := AllEntitySetAnnotations(nil); got != nil {
		t.Errorf("AllEntitySetAnnotations(nil) = %v, want nil", got)
	}
}

func TestAllEntitySetAnnotations(t *testing.T) {
	meta := &traverse.Metadata{
		EntitySets: []traverse.EntitySetInfo{
			{Name: "MaterialSet", SAP: traverse.SAPAnnotations{Label: "Materials", Creatable: true}},
			{Name: "OrderSet", SAP: traverse.SAPAnnotations{Label: "Orders", Deletable: false}},
		},
	}
	got := AllEntitySetAnnotations(meta)
	if len(got) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(got))
	}
	if got["MaterialSet"].Label != "Materials" {
		t.Errorf("MaterialSet.Label = %q, want %q", got["MaterialSet"].Label, "Materials")
	}
	if got["OrderSet"].Deletable {
		t.Error("expected OrderSet.Deletable = false")
	}
}
