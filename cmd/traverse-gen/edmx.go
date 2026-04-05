package main

import (
	"encoding/xml"
	"fmt"
	"io"
	"strings"
)

// Schema holds the parsed OData schema from an EDMX document.
type Schema struct {
	Namespace    string
	EntityTypes  []SchemaEntityType
	ComplexTypes []SchemaComplexType
	EnumTypes    []SchemaEnumType
	EntitySets   []SchemaEntitySet
	Functions    []SchemaFunction
	Actions      []SchemaAction
}

// SchemaEntityType represents a parsed OData EntityType.
type SchemaEntityType struct {
	Name                 string
	Keys                 []string
	Properties           []SchemaProperty
	NavigationProperties []SchemaNavProp
}

// SchemaProperty represents a parsed OData Property.
type SchemaProperty struct {
	Name     string
	Type     string
	Nullable bool
}

// SchemaNavProp represents a parsed OData NavigationProperty (v2 and v4).
type SchemaNavProp struct {
	Name         string
	Type         string // raw Type attr, e.g. "Collection(NorthWind.Order)" or "NorthWind.Customer"
	Partner      string
	IsCollection bool
	TargetType   string // unqualified type name, e.g. "Order"
}

// SchemaComplexType represents a parsed OData ComplexType.
type SchemaComplexType struct {
	Name       string
	Properties []SchemaProperty
}

// SchemaEnumType represents a parsed OData EnumType.
type SchemaEnumType struct {
	Name    string
	Members []SchemaEnumMember
}

// SchemaEnumMember is a single value in an EnumType.
type SchemaEnumMember struct {
	Name  string
	Value string
}

// SchemaEntitySet represents a parsed OData EntitySet.
type SchemaEntitySet struct {
	Name       string
	EntityType string // unqualified entity type name
	Bindings   []SchemaNavBinding
}

// SchemaNavBinding is a NavigationPropertyBinding within an EntitySet.
type SchemaNavBinding struct {
	Path   string
	Target string
}

// SchemaFunction represents a parsed OData FunctionImport.
type SchemaFunction struct {
	Name       string
	ReturnType string
	Parameters []SchemaParameter
}

// SchemaAction represents a parsed OData ActionImport.
type SchemaAction struct {
	Name       string
	Parameters []SchemaParameter
}

// SchemaParameter is a parameter in a function or action.
type SchemaParameter struct {
	Name string
	Type string
}

// parseEDMX parses an OData EDMX XML document and returns all matching schemas.
// If filterNamespace is non-empty, only schemas with that namespace are returned.
func parseEDMX(reader io.Reader, filterNamespace string) ([]Schema, error) {
	// Local XML binding types - kept inside the function to avoid polluting package namespace.
	type xmlPropertyRef struct {
		XMLName xml.Name `xml:"PropertyRef"`
		Name    string   `xml:"Name,attr"`
	}
	type xmlProperty struct {
		XMLName  xml.Name `xml:"Property"`
		Name     string   `xml:"Name,attr"`
		Type     string   `xml:"Type,attr"`
		Nullable *bool    `xml:"Nullable,attr"`
	}
	type xmlNavProp struct {
		XMLName xml.Name `xml:"NavigationProperty"`
		Name    string   `xml:"Name,attr"`
		// OData v4 fields
		Type    string `xml:"Type,attr"`
		Partner string `xml:"Partner,attr"`
		// OData v2 fields (kept for compatibility)
		Relationship string `xml:"Relationship,attr"`
		FromRole     string `xml:"FromRole,attr"`
		ToRole       string `xml:"ToRole,attr"`
	}
	type xmlKey struct {
		PropertyRefs []xmlPropertyRef `xml:"PropertyRef"`
	}
	type xmlEntityType struct {
		XMLName              xml.Name      `xml:"EntityType"`
		Name                 string        `xml:"Name,attr"`
		Key                  xmlKey        `xml:"Key"`
		Properties           []xmlProperty `xml:"Property"`
		NavigationProperties []xmlNavProp  `xml:"NavigationProperty"`
	}
	type xmlComplexType struct {
		XMLName    xml.Name      `xml:"ComplexType"`
		Name       string        `xml:"Name,attr"`
		Properties []xmlProperty `xml:"Property"`
	}
	type xmlEnumMember struct {
		XMLName xml.Name `xml:"Member"`
		Name    string   `xml:"Name,attr"`
		Value   string   `xml:"Value,attr"`
	}
	type xmlEnumType struct {
		XMLName xml.Name        `xml:"EnumType"`
		Name    string          `xml:"Name,attr"`
		Members []xmlEnumMember `xml:"Member"`
	}
	type xmlNavBinding struct {
		XMLName xml.Name `xml:"NavigationPropertyBinding"`
		Path    string   `xml:"Path,attr"`
		Target  string   `xml:"Target,attr"`
	}
	type xmlEntitySet struct {
		XMLName    xml.Name        `xml:"EntitySet"`
		Name       string          `xml:"Name,attr"`
		EntityType string          `xml:"EntityType,attr"`
		Bindings   []xmlNavBinding `xml:"NavigationPropertyBinding"`
	}
	type xmlParameter struct {
		XMLName xml.Name `xml:"Parameter"`
		Name    string   `xml:"Name,attr"`
		Type    string   `xml:"Type,attr"`
	}
	type xmlFunctionImport struct {
		XMLName    xml.Name       `xml:"FunctionImport"`
		Name       string         `xml:"Name,attr"`
		ReturnType string         `xml:"ReturnType,attr"`
		Parameters []xmlParameter `xml:"Parameter"`
	}
	type xmlActionImport struct {
		XMLName    xml.Name       `xml:"ActionImport"`
		Name       string         `xml:"Name,attr"`
		Parameters []xmlParameter `xml:"Parameter"`
	}
	type xmlEntityContainer struct {
		XMLName    xml.Name            `xml:"EntityContainer"`
		Name       string              `xml:"Name,attr"`
		EntitySets []xmlEntitySet      `xml:"EntitySet"`
		Functions  []xmlFunctionImport `xml:"FunctionImport"`
		Actions    []xmlActionImport   `xml:"ActionImport"`
	}
	type xmlSchema struct {
		XMLName          xml.Name             `xml:"Schema"`
		Namespace        string               `xml:"Namespace,attr"`
		EntityTypes      []xmlEntityType      `xml:"EntityType"`
		ComplexTypes     []xmlComplexType     `xml:"ComplexType"`
		EnumTypes        []xmlEnumType        `xml:"EnumType"`
		EntityContainers []xmlEntityContainer `xml:"EntityContainer"`
	}
	type xmlEdmx struct {
		XMLName      xml.Name `xml:"Edmx"`
		DataServices struct {
			Schemas []xmlSchema `xml:"Schema"`
		} `xml:"DataServices"`
	}

	var raw xmlEdmx
	if err := xml.NewDecoder(reader).Decode(&raw); err != nil {
		return nil, fmt.Errorf("failed to decode EDMX: %w", err)
	}

	var schemas []Schema
	for _, rs := range raw.DataServices.Schemas {
		if filterNamespace != "" && rs.Namespace != filterNamespace {
			continue
		}

		s := Schema{Namespace: rs.Namespace}

		for _, et := range rs.EntityTypes {
			set := SchemaEntityType{Name: et.Name}
			for _, pr := range et.Key.PropertyRefs {
				set.Keys = append(set.Keys, pr.Name)
			}
			for _, p := range et.Properties {
				set.Properties = append(set.Properties, parseProperty(p.Name, p.Type, p.Nullable))
			}
			for _, np := range et.NavigationProperties {
				set.NavigationProperties = append(set.NavigationProperties, parseNavProp(np.Name, np.Type, np.Partner))
			}
			s.EntityTypes = append(s.EntityTypes, set)
		}

		for _, ct := range rs.ComplexTypes {
			sct := SchemaComplexType{Name: ct.Name}
			for _, p := range ct.Properties {
				sct.Properties = append(sct.Properties, parseProperty(p.Name, p.Type, p.Nullable))
			}
			s.ComplexTypes = append(s.ComplexTypes, sct)
		}

		for _, et := range rs.EnumTypes {
			set := SchemaEnumType{Name: et.Name}
			for _, m := range et.Members {
				set.Members = append(set.Members, SchemaEnumMember{Name: m.Name, Value: m.Value})
			}
			s.EnumTypes = append(s.EnumTypes, set)
		}

		for _, container := range rs.EntityContainers {
			for _, es := range container.EntitySets {
				ses := SchemaEntitySet{
					Name:       es.Name,
					EntityType: unqualifiedName(es.EntityType),
				}
				for _, b := range es.Bindings {
					ses.Bindings = append(ses.Bindings, SchemaNavBinding{Path: b.Path, Target: b.Target})
				}
				s.EntitySets = append(s.EntitySets, ses)
			}
			for _, fi := range container.Functions {
				sf := SchemaFunction{Name: fi.Name, ReturnType: fi.ReturnType}
				for _, p := range fi.Parameters {
					sf.Parameters = append(sf.Parameters, SchemaParameter{Name: p.Name, Type: p.Type})
				}
				s.Functions = append(s.Functions, sf)
			}
			for _, ai := range container.Actions {
				sa := SchemaAction{Name: ai.Name}
				for _, p := range ai.Parameters {
					sa.Parameters = append(sa.Parameters, SchemaParameter{Name: p.Name, Type: p.Type})
				}
				s.Actions = append(s.Actions, sa)
			}
		}

		schemas = append(schemas, s)
	}

	return schemas, nil
}

func parseProperty(name, typ string, nullable *bool) SchemaProperty {
	isNullable := true // OData default is nullable
	if nullable != nil {
		isNullable = *nullable
	}
	return SchemaProperty{Name: name, Type: typ, Nullable: isNullable}
}

func parseNavProp(name, typ, partner string) SchemaNavProp {
	isCollection := strings.HasPrefix(typ, "Collection(")
	targetType := typ
	if isCollection {
		targetType = strings.TrimSuffix(strings.TrimPrefix(typ, "Collection("), ")")
	}
	return SchemaNavProp{
		Name:         name,
		Type:         typ,
		Partner:      partner,
		IsCollection: isCollection,
		TargetType:   unqualifiedName(targetType),
	}
}

// unqualifiedName strips the namespace prefix from a qualified OData type name.
// e.g. "NorthWind.Order" -> "Order", "Edm.String" -> "String", "Order" -> "Order".
func unqualifiedName(qualifiedName string) string {
	if idx := strings.LastIndex(qualifiedName, "."); idx >= 0 {
		return qualifiedName[idx+1:]
	}
	return qualifiedName
}
