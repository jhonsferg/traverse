package main

import (
	"bytes"
	"fmt"
	"go/format"
	"strings"

	"github.com/jhonsferg/traverse"
)

// CodeGenerator handles template rendering and Go code formatting.
type CodeGenerator struct {
	metadata  *traverse.Metadata
	pkgName   string
	templates *TemplateRegistry
}

// NewCodeGenerator creates a new code generator with templates.
func NewCodeGenerator(metadata *traverse.Metadata, pkgName string) *CodeGenerator {
	return &CodeGenerator{
		metadata:  metadata,
		pkgName:   pkgName,
		templates: NewTemplateRegistry(),
	}
}

// RenderTypes renders the types.go file using templates.
func (cg *CodeGenerator) RenderTypes() (string, error) {
	data := cg.buildTypesData()

	var buf bytes.Buffer
	if err := cg.templates.typesTemplate.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("template render error: %w", err)
	}

	return cg.formatGo(buf.String())
}

// RenderClient renders the client.go file using templates.
func (cg *CodeGenerator) RenderClient() (string, error) {
	data := cg.buildClientData()

	var buf bytes.Buffer
	if err := cg.templates.clientTemplate.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("template render error: %w", err)
	}

	return cg.formatGo(buf.String())
}

// RenderQueries renders the queries.go file using templates.
func (cg *CodeGenerator) RenderQueries() (string, error) {
	data := cg.buildQueryData()

	var buf bytes.Buffer
	if err := cg.templates.queriesTemplate.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("template render error: %w", err)
	}

	return cg.formatGo(buf.String())
}

// buildTypesData constructs template data for types.go.
func (cg *CodeGenerator) buildTypesData() TypesTemplateData {
	entities := make([]EntityData, 0, len(cg.metadata.EntityTypes))

	for _, et := range cg.metadata.EntityTypes {
		ed := EntityData{
			Name:     et.Name,
			GoName:   PascalCase(et.Name),
			Fields:   make([]FieldData, 0),
			NavProps: make([]NavPropData, 0),
			Keys:     make([]string, 0),
		}

		// Add fields
		for _, prop := range et.Properties {
			goType, _ := MapODataType(prop.Type, prop.Nullable)
			isKey := isKeyField(et, prop.Name)

			fd := FieldData{
				Name:       prop.Name,
				GoName:     PascalCase(prop.Name),
				Type:       goType,
				IsKey:      isKey,
				IsNullable: prop.Nullable,
				JSONTag:    FormatJSONTag(prop.Name, prop.Nullable),
			}

			ed.Fields = append(ed.Fields, fd)

			if isKey {
				ed.Keys = append(ed.Keys, prop.Name)
			}
		}

		// Add navigation properties
		for _, navProp := range et.NavigationProperties {
			nd := NavPropData{
				Name:       navProp.Name,
				GoName:     PascalCase(navProp.Name),
				JSONTag:    FormatJSONTag(navProp.Name, true),
				TargetType: PascalCase(navProp.Name),
				IsSingle:   false, // Default to collection
			}

			ed.NavProps = append(ed.NavProps, nd)
		}

		entities = append(entities, ed)
	}

	return TypesTemplateData{
		PackageName: cg.pkgName,
		Entities:    entities,
	}
}

// buildClientData constructs template data for client.go.
func (cg *CodeGenerator) buildClientData() ClientTemplateData {
	entitySets := make([]EntitySetData, 0, len(cg.metadata.EntitySets))

	for _, es := range cg.metadata.EntitySets {
		esd := EntitySetData{
			Name:       es.Name,
			GoName:     PascalCase(es.Name),
			EntityName: PascalCase(es.EntityTypeName),
		}
		entitySets = append(entitySets, esd)
	}

	return ClientTemplateData{
		PackageName: cg.pkgName,
		EntitySets:  entitySets,
	}
}

// buildQueryData constructs template data for queries.go.
func (cg *CodeGenerator) buildQueryData() QueryTemplateData {
	builders := make([]QueryBuilderData, 0, len(cg.metadata.EntityTypes))

	for _, et := range cg.metadata.EntityTypes {
		keys := make([]string, 0)
		for _, k := range et.Key {
			keys = append(keys, k.Name)
		}

		qbd := QueryBuilderData{
			EntityName:   et.Name,
			EntityGoName: PascalCase(et.Name),
			QueryName:    PascalCase(et.Name) + "Query",
			Keys:         keys,
			HasKeyAccess: len(keys) > 0,
		}

		builders = append(builders, qbd)
	}

	return QueryTemplateData{
		PackageName:   cg.pkgName,
		QueryBuilders: builders,
	}
}

// formatGo formats Go source code using gofmt.
func (cg *CodeGenerator) formatGo(src string) (string, error) {
	formatted, err := format.Source([]byte(src))
	if err != nil {
		// Return unformatted code with error info for debugging
		return src, fmt.Errorf("gofmt error: %w", err)
	}
	return string(formatted), nil
}

// PascalCase converts a string to PascalCase.
func PascalCase(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}

	// Handle special characters and spaces
	parts := strings.FieldsFunc(s, func(r rune) bool {
		return r == '_' || r == ' ' || r == '-' || r == '.'
	})

	for i, part := range parts {
		if len(part) > 0 {
			parts[i] = strings.ToUpper(string(part[0])) + strings.ToLower(part[1:])
		}
	}

	return strings.Join(parts, "")
}

// FormatJSONTag generates a JSON struct tag for a property.
func FormatJSONTag(propName string, nullable bool) string {
	tag := fmt.Sprintf("`json:\"%s\"", propName)
	if nullable {
		tag += ",omitempty"
	}
	tag += "`"
	return tag
}

// NamingConvention defines how to convert property names.
type NamingConvention int

const (
	// CamelCase converts to camelCase field names
	CamelCase NamingConvention = iota
	// SnakeCase converts to snake_case field names
	SnakeCase
	// PascalCase converts to PascalCase field names (default for Go)
	PascalCaseNaming
)

// ToCamelCase converts a PascalCase string to camelCase.
func ToCamelCase(s string) string {
	if len(s) == 0 {
		return s
	}
	s = PascalCase(s)
	return strings.ToLower(string(s[0])) + s[1:]
}

// ToSnakeCase converts a PascalCase string to snake_case.
func ToSnakeCase(s string) string {
	s = PascalCase(s)
	var buf bytes.Buffer

	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			buf.WriteRune('_')
		}
		buf.WriteRune(r)
	}

	return strings.ToLower(buf.String())
}

// JSONTagGenerator handles JSON tag generation with customizable options.
type JSONTagGenerator struct {
	NamingConvention NamingConvention
	OmitEmpty        bool
	OmitZero         bool
}

// Generate creates a JSON tag for a property.
func (jtg *JSONTagGenerator) Generate(propName string, nullable bool) string {
	var fieldName string

	switch jtg.NamingConvention {
	case CamelCase:
		fieldName = ToCamelCase(propName)
	case SnakeCase:
		fieldName = ToSnakeCase(propName)
	default:
		fieldName = propName
	}

	tag := fmt.Sprintf("`json:\"%s\"", fieldName)

	if (nullable && jtg.OmitEmpty) || jtg.OmitZero {
		tag += ",omitempty"
	}

	tag += "`"
	return tag
}
