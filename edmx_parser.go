package traverse

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"strconv"
)

// ParseEDMX parses an OData $metadata XML response in EDMX format.
//
// ParseEDMX reads an OData service's metadata document (typically from $metadata endpoint)
// and extracts schema information including entity types, entity sets, properties,
// navigation properties, and associations.
//
// The EDMX (Entity Data Model XML) format is the standard way OData services
// describe their data model. This parser supports both OData v2 and v4 metadata formats
// with special handling for SAP annotations (sap:label, sap:sortable, sap:filterable,
// sap:searchable, sap:required-in-filter, sap:text, sap:unit, sap:value-list,
// sap:display-format, sap:field-control, sap:semantics, sap:key, sap:updatable-path,
// and entity-set-level sap:creatable, sap:updatable, sap:deletable, sap:pageable,
// sap:addressable, sap:requires-filter, sap:change-tracking).
//
// The parser extracts:
//   - Entity types with their properties (name, type, nullable, length constraints)
//   - Entity sets (the exposed data sources)
//   - Navigation properties (relationships between entities)
//   - Associations (cardinality and relationship definitions)
//   - SAP-specific annotations for UI rendering hints
//
// Returns a [Metadata] struct containing the parsed schema information, or an error
// if XML parsing fails.
//
// Example:
//
//	resp, _ := http.Get("https://odata.service/$metadata")
//	metadata, err := ParseEDMX(resp.Body)
//	for _, et := range metadata.EntityTypes {
//		fmt.Println("Entity:", et.Name, "Key:", et.Key)
//	}
func ParseEDMX(reader io.Reader) (*Metadata, error) {
	// Read all bytes to allow SAP namespace normalization.
	// SAP ABAP Gateway services may declare xmlns:sap="http://www.sap.com/Protocols/SAPData"
	// which causes Go's xml decoder to expand sap:* attributes to full namespace URIs.
	// By stripping the namespace declaration before parsing, we ensure sap:label etc.
	// are matched as unprefixed namespace "sap" which the struct tags handle uniformly.
	raw, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("traverse: failed to read EDMX: %w", err)
	}
	// Normalize SAP namespace declaration so Go xml treats "sap:" as a literal prefix.
	raw = bytes.ReplaceAll(raw,
		[]byte(`xmlns:sap="http://www.sap.com/Protocols/SAPData"`), []byte{})

	var edmx struct {
		XMLName      xml.Name `xml:"Edmx"`
		Version      string   `xml:"Version,attr"`
		DataServices struct {
			Schema []struct {
				XMLName     xml.Name `xml:"Schema"`
				Namespace   string   `xml:"Namespace,attr"`
				EntityTypes []struct {
					XMLName  xml.Name `xml:"EntityType"`
					Name     string   `xml:"Name,attr"`
					Abstract string   `xml:"Abstract,attr"`
					BaseType string   `xml:"BaseType,attr"`
					Key      []struct {
						XMLName      xml.Name `xml:"Key"`
						PropertyRefs []struct {
							XMLName xml.Name `xml:"PropertyRef"`
							Name    string   `xml:"Name,attr"`
						} `xml:"PropertyRef"`
					} `xml:"Key"`
					Properties []struct {
						XMLName           xml.Name `xml:"Property"`
						Name              string   `xml:"Name,attr"`
						Type              string   `xml:"Type,attr"`
						Nullable          *bool    `xml:"Nullable,attr"`
						MaxLength         *string  `xml:"MaxLength,attr"`
						Precision         *string  `xml:"Precision,attr"`
						Scale             *string  `xml:"Scale,attr"`
						SAPPID            *string  `xml:"sap parameter-type,attr"`
						SAPLabel          *string  `xml:"sap label,attr"`
						SAPSortable       *string  `xml:"sap sortable,attr"`
						SAPFilterable     *string  `xml:"sap filterable,attr"`
						SAPSearchable     *string  `xml:"sap searchable,attr"`
						SAPRequiredFilter *string  `xml:"sap required-in-filter,attr"`
						SAPText           *string  `xml:"sap text,attr"`
						SAPUnit           *string  `xml:"sap unit,attr"`
						SAPValueList      *string  `xml:"sap value-list,attr"`
						SAPDisplayFormat  *string  `xml:"sap display-format,attr"`
						SAPFieldControl   *string  `xml:"sap field-control,attr"`
						SAPSemantics      *string  `xml:"sap semantics,attr"`
						SAPKey            *string  `xml:"sap key,attr"`
						SAPUpdatablePath  *string  `xml:"sap updatable-path,attr"`
					} `xml:"Property"`
					NavigationProperties []struct {
						XMLName      xml.Name `xml:"NavigationProperty"`
						Name         string   `xml:"Name,attr"`
						Relationship string   `xml:"Relationship,attr"`
						FromRole     string   `xml:"FromRole,attr"`
						ToRole       string   `xml:"ToRole,attr"`
					} `xml:"NavigationProperty"`
				} `xml:"EntityType"`
				ComplexTypes []struct {
					XMLName    xml.Name `xml:"ComplexType"`
					Name       string   `xml:"Name,attr"`
					BaseType   string   `xml:"BaseType,attr"`
					Properties []struct {
						XMLName           xml.Name `xml:"Property"`
						Name              string   `xml:"Name,attr"`
						Type              string   `xml:"Type,attr"`
						Nullable          *bool    `xml:"Nullable,attr"`
						MaxLength         *string  `xml:"MaxLength,attr"`
						Precision         *string  `xml:"Precision,attr"`
						Scale             *string  `xml:"Scale,attr"`
						SAPLabel          *string  `xml:"sap label,attr"`
						SAPSortable       *string  `xml:"sap sortable,attr"`
						SAPFilterable     *string  `xml:"sap filterable,attr"`
						SAPSearchable     *string  `xml:"sap searchable,attr"`
						SAPRequiredFilter *string  `xml:"sap required-in-filter,attr"`
						SAPText           *string  `xml:"sap text,attr"`
						SAPUnit           *string  `xml:"sap unit,attr"`
						SAPValueList      *string  `xml:"sap value-list,attr"`
						SAPDisplayFormat  *string  `xml:"sap display-format,attr"`
						SAPFieldControl   *string  `xml:"sap field-control,attr"`
						SAPSemantics      *string  `xml:"sap semantics,attr"`
						SAPKey            *string  `xml:"sap key,attr"`
						SAPUpdatablePath  *string  `xml:"sap updatable-path,attr"`
					} `xml:"Property"`
				} `xml:"ComplexType"`
				EnumTypes []struct {
					XMLName        xml.Name `xml:"EnumType"`
					Name           string   `xml:"Name,attr"`
					IsFlags        string   `xml:"IsFlags,attr"`
					UnderlyingType string   `xml:"UnderlyingType,attr"`
					Members        []struct {
						XMLName xml.Name `xml:"Member"`
						Name    string   `xml:"Name,attr"`
						Value   string   `xml:"Value,attr"`
					} `xml:"Member"`
				} `xml:"EnumType"`
				EntityContainers []struct {
					XMLName    xml.Name `xml:"EntityContainer"`
					Name       string   `xml:"Name,attr"`
					IsDefault  string   `xml:"m:IsDefaultEntityContainer,attr"`
					EntitySets []struct {
						XMLName                    xml.Name `xml:"EntitySet"`
						Name                       string   `xml:"Name,attr"`
						EntityType                 string   `xml:"EntityType,attr"`
						SAPLabel                   string   `xml:"sap label,attr"`
						SAPCreatable               *string  `xml:"sap creatable,attr"`
						SAPUpdatable               *string  `xml:"sap updatable,attr"`
						SAPDeletable               *string  `xml:"sap deletable,attr"`
						SAPPageable                *string  `xml:"sap pageable,attr"`
						SAPAddressable             *string  `xml:"sap addressable,attr"`
						SAPRequiresFilter          *string  `xml:"sap requires-filter,attr"`
						SAPChangeTracking          *string  `xml:"sap change-tracking,attr"`
						NavigationPropertyBindings []struct {
							XMLName xml.Name `xml:"NavigationPropertyBinding"`
							Path    string   `xml:"Path,attr"`
							Target  string   `xml:"Target,attr"`
						} `xml:"NavigationPropertyBinding"`
					} `xml:"EntitySet"`
					FunctionImports []struct {
						XMLName   xml.Name `xml:"FunctionImport"`
						Name      string   `xml:"Name,attr"`
						IsBinding *bool    `xml:"m:IsBindingParameter,attr"`
						Parameter []struct {
							XMLName        xml.Name `xml:"Parameter"`
							Name           string   `xml:"Name,attr"`
							Type           string   `xml:"Type,attr"`
							IsBindingParam *bool    `xml:"m:IsBindingParameter,attr"`
						} `xml:"Parameter"`
					} `xml:"FunctionImport"`
				} `xml:"EntityContainer"`
				Associations []struct {
					XMLName xml.Name `xml:"Association"`
					Name    string   `xml:"Name,attr"`
					Ends    []struct {
						XMLName      xml.Name `xml:"End"`
						Role         string   `xml:"Role,attr"`
						Type         string   `xml:"Type,attr"`
						Multiplicity string   `xml:"Multiplicity,attr"`
					} `xml:"End"`
				} `xml:"Association"`
				// Annotations holds external OData annotation groups (v4 style).
				// Each group targets a schema element (e.g. "Namespace.EntityType/Property").
				Annotations []struct {
					XMLName    xml.Name `xml:"Annotations"`
					Target     string   `xml:"Target,attr"`
					Annotation []struct {
						XMLName xml.Name `xml:"Annotation"`
						Term    string   `xml:"Term,attr"`
						String  string   `xml:"String,attr"`
						Bool    string   `xml:"Bool,attr"`
						Decimal string   `xml:"Decimal,attr"`
						Int     string   `xml:"Int,attr"`
						Float   string   `xml:"Float,attr"`
					} `xml:"Annotation"`
				} `xml:"Annotations"`
			} `xml:"Schema"`
		} `xml:"DataServices"`
	}

	decoder := xml.NewDecoder(bytes.NewReader(raw))
	if err := decoder.Decode(&edmx); err != nil {
		return nil, fmt.Errorf("failed to decode EDMX XML: %w", err)
	}

	// Build Metadata from parsed EDMX
	metadata := &Metadata{
		Version:      edmx.Version,
		EntityTypes:  make([]EntityType, 0),
		EntitySets:   make([]EntitySetInfo, 0),
		Associations: make([]Association, 0),
	}

	if len(edmx.DataServices.Schema) == 0 {
		return metadata, nil
	}

	// Process first schema (most common case)
	schema := edmx.DataServices.Schema[0]
	metadata.Namespace = schema.Namespace

	// Parse entity types
	for _, et := range schema.EntityTypes {
		entityType := EntityType{
			Name:                 et.Name,
			Abstract:             et.Abstract == "true",
			BaseType:             et.BaseType,
			Key:                  make([]PropertyRef, 0),
			Properties:           make([]Property, 0),
			NavigationProperties: make([]NavigationProperty, 0),
		}

		// Extract keys
		if len(et.Key) > 0 && len(et.Key[0].PropertyRefs) > 0 {
			for _, pr := range et.Key[0].PropertyRefs {
				entityType.Key = append(entityType.Key, PropertyRef{Name: pr.Name})
			}
		}

		// Extract properties
		for _, prop := range et.Properties {
			p := Property{
				Name:     prop.Name,
				Type:     prop.Type,
				Nullable: true, // Default to true per OData spec
			}
			if prop.Nullable != nil {
				p.Nullable = *prop.Nullable
			}
			if prop.MaxLength != nil {
				if ml, err := strconv.Atoi(*prop.MaxLength); err == nil {
					p.MaxLength = &ml
				}
			}
			if prop.Precision != nil {
				if pr, err := strconv.Atoi(*prop.Precision); err == nil {
					p.Precision = &pr
				}
			}
			if prop.Scale != nil {
				if sc, err := strconv.Atoi(*prop.Scale); err == nil {
					p.Scale = &sc
				}
			}

			// SAP annotations - all sap:* property-level attributes
			if prop.SAPLabel != nil || prop.SAPPID != nil || prop.SAPSortable != nil || prop.SAPFilterable != nil ||
				prop.SAPSearchable != nil || prop.SAPRequiredFilter != nil || prop.SAPText != nil ||
				prop.SAPUnit != nil || prop.SAPValueList != nil || prop.SAPDisplayFormat != nil ||
				prop.SAPFieldControl != nil || prop.SAPSemantics != nil || prop.SAPKey != nil ||
				prop.SAPUpdatablePath != nil {
				p.SAP = SAPAnnotations{
					Label:         derefStr(prop.SAPLabel),
					Filterable:    derefBool(prop.SAPFilterable),
					Sortable:      derefBool(prop.SAPSortable),
					Searchable:    derefBool(prop.SAPSearchable),
					Required:      derefBool(prop.SAPRequiredFilter),
					Text:          derefStr(prop.SAPText),
					Unit:          derefStr(prop.SAPUnit),
					ValueList:     derefStr(prop.SAPValueList),
					DisplayFormat: derefStr(prop.SAPDisplayFormat),
					FieldControl:  derefStr(prop.SAPFieldControl),
					Semantics:     derefStr(prop.SAPSemantics),
					IsKey:         derefStr(prop.SAPKey) == "true",
					UpdatablePath: derefStr(prop.SAPUpdatablePath),
				}
			}

			entityType.Properties = append(entityType.Properties, p)
		}

		// Extract navigation properties
		for _, navProp := range et.NavigationProperties {
			entityType.NavigationProperties = append(entityType.NavigationProperties, NavigationProperty{
				Name:           navProp.Name,
				FromEntityType: navProp.FromRole,
				ToEntityType:   navProp.ToRole,
			})
		}

		metadata.EntityTypes = append(metadata.EntityTypes, entityType)
	}

	// Parse entity containers and entity sets
	for _, et := range schema.EntityTypes {
		for _, container := range schema.EntityContainers {
			for _, es := range container.EntitySets {
				// EntityType attribute includes namespace, extract just the type name
				typeName := et.Name
				if es.EntityType == schema.Namespace+"."+et.Name {
					esInfo := EntitySetInfo{
						Name:           es.Name,
						EntityTypeName: typeName,
					}
					// Entity-set-level SAP annotations
					if es.SAPCreatable != nil || es.SAPUpdatable != nil || es.SAPDeletable != nil ||
						es.SAPPageable != nil || es.SAPAddressable != nil || es.SAPRequiresFilter != nil ||
						es.SAPChangeTracking != nil || es.SAPLabel != "" {
						esInfo.SAP = SAPAnnotations{
							Label:          es.SAPLabel,
							Creatable:      derefBoolStr(es.SAPCreatable, true),
							Updatable:      derefBoolStr(es.SAPUpdatable, true),
							Deletable:      derefBoolStr(es.SAPDeletable, true),
							Pageable:       derefBoolStr(es.SAPPageable, true),
							Addressable:    derefBoolStr(es.SAPAddressable, true),
							RequiresFilter: derefBoolStr(es.SAPRequiresFilter, false),
							ChangeTracking: derefBoolStr(es.SAPChangeTracking, false),
						}
					}
					for _, nb := range es.NavigationPropertyBindings {
						esInfo.NavigationBindings = append(esInfo.NavigationBindings, NavigationBinding{
							Path:   nb.Path,
							Target: nb.Target,
						})
					}
					metadata.EntitySets = append(metadata.EntitySets, esInfo)
				}
			}
		}
	}

	// Parse complex types
	for _, ct := range schema.ComplexTypes {
		complexType := ComplexType{
			Name:       ct.Name,
			Properties: make([]Property, 0),
		}
		for _, prop := range ct.Properties {
			p := Property{
				Name:     prop.Name,
				Type:     prop.Type,
				Nullable: true, // Default to true per OData spec
			}
			if prop.Nullable != nil {
				p.Nullable = *prop.Nullable
			}
			if prop.MaxLength != nil {
				if ml, err := strconv.Atoi(*prop.MaxLength); err == nil {
					p.MaxLength = &ml
				}
			}
			if prop.Precision != nil {
				if pr, err := strconv.Atoi(*prop.Precision); err == nil {
					p.Precision = &pr
				}
			}
			if prop.Scale != nil {
				if sc, err := strconv.Atoi(*prop.Scale); err == nil {
					p.Scale = &sc
				}
			}
			if prop.SAPLabel != nil || prop.SAPSortable != nil || prop.SAPFilterable != nil ||
				prop.SAPSearchable != nil || prop.SAPRequiredFilter != nil || prop.SAPText != nil ||
				prop.SAPUnit != nil || prop.SAPValueList != nil || prop.SAPDisplayFormat != nil ||
				prop.SAPFieldControl != nil || prop.SAPSemantics != nil || prop.SAPKey != nil ||
				prop.SAPUpdatablePath != nil {
				p.SAP = SAPAnnotations{
					Label:         derefStr(prop.SAPLabel),
					Filterable:    derefBool(prop.SAPFilterable),
					Sortable:      derefBool(prop.SAPSortable),
					Searchable:    derefBool(prop.SAPSearchable),
					Required:      derefBool(prop.SAPRequiredFilter),
					Text:          derefStr(prop.SAPText),
					Unit:          derefStr(prop.SAPUnit),
					ValueList:     derefStr(prop.SAPValueList),
					DisplayFormat: derefStr(prop.SAPDisplayFormat),
					FieldControl:  derefStr(prop.SAPFieldControl),
					Semantics:     derefStr(prop.SAPSemantics),
					IsKey:         derefStr(prop.SAPKey) == "true",
					UpdatablePath: derefStr(prop.SAPUpdatablePath),
				}
			}
			complexType.Properties = append(complexType.Properties, p)
		}
		metadata.ComplexTypes = append(metadata.ComplexTypes, complexType)
	}

	// Parse enum types
	for _, et := range schema.EnumTypes {
		enumType := EnumType{
			Name:           et.Name,
			IsFlags:        et.IsFlags == "true",
			UnderlyingType: et.UnderlyingType,
			Members:        make([]EnumMember, 0),
		}
		for _, m := range et.Members {
			member := EnumMember{Name: m.Name}
			if v, err := strconv.ParseInt(m.Value, 10, 64); err == nil {
				member.Value = int(v)
			}
			enumType.Members = append(enumType.Members, member)
		}
		metadata.EnumTypes = append(metadata.EnumTypes, enumType)
	}

	// Parse associations
	for _, assoc := range schema.Associations {
		if len(assoc.Ends) >= 2 {
			metadata.Associations = append(metadata.Associations, Association{
				Name: assoc.Name,
				From: AssociationEnd{
					EntityType:   assoc.Ends[0].Type,
					Multiplicity: assoc.Ends[0].Multiplicity,
				},
				To: AssociationEnd{
					EntityType:   assoc.Ends[1].Type,
					Multiplicity: assoc.Ends[1].Multiplicity,
				},
			})
		}
	}

	// Apply OData Core/Validation vocabulary annotations from <Annotations Target="..."> elements.
	// Target format: "Namespace.EntityTypeName/PropertyName" or "Namespace.EntityTypeName".
	applyEDMXVocabularyAnnotations(metadata, schema.Namespace, schema.Annotations)

	return metadata, nil
}

// derefStr dereferences a pointer to string, returning empty string if nil.
//
// derefStr is a helper for safely extracting string values from optional
// XML attributes. Used when processing SAP annotations and other optional metadata.
func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// derefBool dereferences a pointer to string and converts to boolean.
//
// derefBool interprets a string value as a boolean (true if value is "true"),
// returning false for nil or any other value. Used when processing SAP annotation
// booleans that are encoded as XML string attributes.
func derefBool(s *string) bool {
	if s == nil {
		return false
	}
	return *s == "true"
}

// derefBoolStr dereferences a pointer to string and converts to boolean with a default.
//
// derefBoolStr returns defaultVal when the pointer is nil (attribute absent),
// true when the string value is "true", and false otherwise.
// SAP entity-set attributes like sap:creatable default to true when absent.
func derefBoolStr(s *string, defaultVal bool) bool {
	if s == nil {
		return defaultVal
	}
	return *s == "true"
}

// applyEDMXVocabularyAnnotations populates Core and Validation vocabulary fields on
// properties by processing <Annotations Target="..."> groups found in the schema.
func applyEDMXVocabularyAnnotations(metadata *Metadata, namespace string, rawGroups interface{}) {
	// Build a property-keyed annotation map.
	// key: "EntityTypeName/PropertyName" → flat map of term → value.
	annotMap := make(map[string]map[string]string)

	switch groups := rawGroups.(type) {
	case []struct {
		XMLName    xml.Name `xml:"Annotations"`
		Target     string   `xml:"Target,attr"`
		Annotation []struct {
			XMLName xml.Name `xml:"Annotation"`
			Term    string   `xml:"Term,attr"`
			String  string   `xml:"String,attr"`
			Bool    string   `xml:"Bool,attr"`
			Decimal string   `xml:"Decimal,attr"`
			Int     string   `xml:"Int,attr"`
			Float   string   `xml:"Float,attr"`
		} `xml:"Annotation"`
	}:
		for _, grp := range groups {
			// Strip namespace prefix from target so we get "EntityTypeName/PropertyName".
			target := grp.Target
			prefix := namespace + "."
			if len(target) > len(prefix) && target[:len(prefix)] == prefix {
				target = target[len(prefix):]
			}
			for _, ann := range grp.Annotation {
				if ann.Term == "" {
					continue
				}
				// Pick the first non-empty value attribute.
				val := ann.String
				if val == "" {
					val = ann.Bool
				}
				if val == "" {
					val = ann.Decimal
				}
				if val == "" {
					val = ann.Int
				}
				if val == "" {
					val = ann.Float
				}
				if _, ok := annotMap[target]; !ok {
					annotMap[target] = make(map[string]string)
				}
				annotMap[target][ann.Term] = val
			}
		}
	}

	// Apply to each entity type / property.
	for etIdx := range metadata.EntityTypes {
		et := &metadata.EntityTypes[etIdx]
		for propIdx := range et.Properties {
			prop := &et.Properties[propIdx]
			key := et.Name + "/" + prop.Name
			if annots, ok := annotMap[key]; ok {
				if core := ParseCoreVocabulary(annots); hasAnyCoreField(core) {
					c := core
					prop.Core = &c
				}
				if val := ParseValidationVocabulary(annots); hasAnyValidationField(val) {
					v := val
					prop.Validation = &v
				}
				if meas := ParseMeasuresVocabulary(annots); hasAnyMeasuresField(meas) {
					m := meas
					prop.Measures = &m
				}
				if ana := ParseAnalyticsVocabulary(annots); hasAnyAnalyticsField(ana) {
					a := ana
					prop.Analytics = &a
				}
			}
		}
		// Also handle entity-level annotations (no "/" in target).
		if annots, ok := annotMap[et.Name]; ok {
			for propIdx := range et.Properties {
				prop := &et.Properties[propIdx]
				if prop.Core == nil {
					if core := ParseCoreVocabulary(annots); hasAnyCoreField(core) {
						c := core
						prop.Core = &c
					}
				}
			}
		}
	}
}

// hasAnyCoreField reports whether any field in v is non-zero.
func hasAnyCoreField(v CoreVocabulary) bool {
	return v.Description != "" || v.LongDescription != "" || v.IsLanguageDependent ||
		v.Immutable || v.Computed || len(v.Permissions) > 0 || v.Example != ""
}

// hasAnyValidationField reports whether any field in v is non-zero.
func hasAnyValidationField(v ValidationVocabulary) bool {
	return v.Minimum != nil || v.Maximum != nil || v.Pattern != "" ||
		len(v.AllowedValues) > 0 || v.Required
}

// hasAnyMeasuresField reports whether any field in v is non-zero.
func hasAnyMeasuresField(v MeasuresVocabulary) bool {
	return v.ISOCurrency != "" || v.Scale != nil || v.Unit != "" ||
		v.SIPrefix != "" || v.DurationGranularity != ""
}

// hasAnyAnalyticsField reports whether any field in v is non-zero.
func hasAnyAnalyticsField(v AnalyticsVocabulary) bool {
	return v.AggregationMethod != "" || v.IsDimension || v.IsMeasure ||
		v.RollupLevels != 0 || len(v.ReferencedProperties) > 0 ||
		len(v.GroupableProperties) > 0 || len(v.AggregatableProperties) > 0
}
