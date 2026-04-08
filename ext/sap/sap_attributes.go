package sap

import traverse "github.com/jhonsferg/traverse"

// SAPEntitySetAnnotation holds entity-set-level SAP annotations from EDMX metadata.
//
// These attributes appear on <EntitySet> elements in SAP ABAP Gateway OData v2 EDMX
// and control client behaviour such as whether create, update, or delete operations
// are permitted, and whether server-side paging or change tracking are supported.
//
// All boolean fields reflect the SAP default when the attribute is absent:
//   - Creatable, Updatable, Deletable, Pageable, Addressable default to true
//   - RequiresFilter, ChangeTracking default to false
type SAPEntitySetAnnotation struct {
	// Label is a human-readable label for the entity set (sap:label).
	Label string
	// Creatable is true when the entity set supports POST create operations (sap:creatable).
	// Defaults to true when the attribute is absent.
	Creatable bool
	// Updatable is true when the entity set supports PATCH/PUT update operations (sap:updatable).
	// Defaults to true when the attribute is absent.
	Updatable bool
	// Deletable is true when the entity set supports DELETE operations (sap:deletable).
	// Defaults to true when the attribute is absent.
	Deletable bool
	// Pageable is true when the entity set supports server-side paging via $top/$skip (sap:pageable).
	// Defaults to true when the attribute is absent.
	Pageable bool
	// Addressable is true when individual entities can be addressed by key (sap:addressable).
	// Defaults to true when the attribute is absent.
	Addressable bool
	// RequiresFilter is true when a $filter clause must be present in all queries (sap:requires-filter).
	// Defaults to false when the attribute is absent.
	RequiresFilter bool
	// ChangeTracking is true when the entity set supports OData delta / change-tracking (sap:change-tracking).
	// Defaults to false when the attribute is absent.
	ChangeTracking bool
}

// EntitySetAnnotation returns the SAP entity-set-level annotations for a given
// entity set name. If the entity set is not found in the metadata, the returned
// annotation uses the SAP-specified defaults (all CRUD enabled, no filter required).
//
// Example:
//
//	meta, _ := client.Metadata(ctx)
//	ann := sap.EntitySetAnnotation(meta, "MaterialSet")
//	if !ann.Creatable {
//	    // POST /MaterialSet not allowed by SAP configuration
//	}
func EntitySetAnnotation(meta *traverse.Metadata, entitySetName string) SAPEntitySetAnnotation {
	if meta == nil {
		return defaultEntitySetAnnotation()
	}
	for _, es := range meta.EntitySets {
		if es.Name == entitySetName {
			s := es.SAP
			return SAPEntitySetAnnotation{
				Label:          s.Label,
				Creatable:      s.Creatable,
				Updatable:      s.Updatable,
				Deletable:      s.Deletable,
				Pageable:       s.Pageable,
				Addressable:    s.Addressable,
				RequiresFilter: s.RequiresFilter,
				ChangeTracking: s.ChangeTracking,
			}
		}
	}
	return defaultEntitySetAnnotation()
}

// AllEntitySetAnnotations returns a map of entity-set name to SAPEntitySetAnnotation
// for all entity sets in the metadata.
//
// Example:
//
//	anns := sap.AllEntitySetAnnotations(meta)
//	for name, ann := range anns {
//	    fmt.Printf("%s: creatable=%v deletable=%v\n", name, ann.Creatable, ann.Deletable)
//	}
func AllEntitySetAnnotations(meta *traverse.Metadata) map[string]SAPEntitySetAnnotation {
	if meta == nil {
		return nil
	}
	result := make(map[string]SAPEntitySetAnnotation, len(meta.EntitySets))
	for _, es := range meta.EntitySets {
		s := es.SAP
		result[es.Name] = SAPEntitySetAnnotation{
			Label:          s.Label,
			Creatable:      s.Creatable,
			Updatable:      s.Updatable,
			Deletable:      s.Deletable,
			Pageable:       s.Pageable,
			Addressable:    s.Addressable,
			RequiresFilter: s.RequiresFilter,
			ChangeTracking: s.ChangeTracking,
		}
	}
	return result
}

// defaultEntitySetAnnotation returns an annotation with SAP's implied defaults:
// all CRUD operations allowed, paging and addressing enabled, no filter required,
// no change-tracking.
func defaultEntitySetAnnotation() SAPEntitySetAnnotation {
	return SAPEntitySetAnnotation{
		Creatable:   true,
		Updatable:   true,
		Deletable:   true,
		Pageable:    true,
		Addressable: true,
	}
}
