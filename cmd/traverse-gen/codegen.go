package main

import (
	"bytes"
	"fmt"
	"go/format"
	"strings"
	"text/template"
)

// CodeGenerator renders Go source files from parsed OData schemas using templates.
type CodeGenerator struct {
	schemas []Schema
	pkgName string
	reg     *TemplateRegistry
}

// NewCodeGenerator creates a new CodeGenerator for the given schemas.
func NewCodeGenerator(schemas []Schema, pkgName string) *CodeGenerator {
	return &CodeGenerator{
		schemas: schemas,
		pkgName: pkgName,
		reg:     NewTemplateRegistry(),
	}
}

// RenderTypes renders the types.go file content.
func (cg *CodeGenerator) RenderTypes() (string, error) {
	return cg.render(cg.reg.typesTemplate, cg.buildTypesData())
}

// RenderClient renders the client.go file content.
func (cg *CodeGenerator) RenderClient() (string, error) {
	return cg.render(cg.reg.clientTemplate, cg.buildClientData())
}

// RenderQueries renders the queries.go file content.
func (cg *CodeGenerator) RenderQueries() (string, error) {
	return cg.render(cg.reg.queriesTemplate, cg.buildQueriesData())
}

func (cg *CodeGenerator) render(tmpl *template.Template, data interface{}) (string, error) {
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("template render error: %w", err)
	}
	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return buf.String(), fmt.Errorf("gofmt error: %w\n--- source ---\n%s", err, buf.String())
	}
	return string(formatted), nil
}

// buildTypesData collects template data for types.go across all schemas.
func (cg *CodeGenerator) buildTypesData() TypesTemplateData {
	data := TypesTemplateData{
		PackageName:         cg.pkgName,
		NeedsTimeImport:     needsTimeImport(cg.schemas),
		NeedsTraverseImport: needsTraverseImport(cg.schemas),
	}

	for _, s := range cg.schemas {
		for _, et := range s.EntityTypes {
			ed := EntityData{
				Name:   et.Name,
				GoName: PascalCase(et.Name),
			}
			for _, prop := range et.Properties {
				isKey := isKeyProp(et.Keys, prop.Name)
				goType := mapODataType(prop.Type, prop.Nullable)
				ed.Fields = append(ed.Fields, FieldData{
					Name:       prop.Name,
					GoName:     PascalCase(prop.Name),
					GoType:     goType,
					IsKey:      isKey,
					IsNullable: prop.Nullable,
					Tag:        buildStructTag(prop.Name, isKey, false, prop.Nullable),
				})
			}
			for _, nav := range et.NavigationProperties {
				ed.NavProps = append(ed.NavProps, NavPropData{
					Name:   nav.Name,
					GoName: PascalCase(nav.Name),
					GoType: navGoType(nav),
					Tag:    buildStructTag(nav.Name, false, true, true),
				})
			}
			data.Entities = append(data.Entities, ed)
		}

		for _, ct := range s.ComplexTypes {
			cd := ComplexTypeData{
				Name:   ct.Name,
				GoName: PascalCase(ct.Name),
			}
			for _, prop := range ct.Properties {
				cd.Fields = append(cd.Fields, FieldData{
					Name:       prop.Name,
					GoName:     PascalCase(prop.Name),
					GoType:     mapODataType(prop.Type, prop.Nullable),
					IsNullable: prop.Nullable,
					Tag:        buildStructTag(prop.Name, false, false, prop.Nullable),
				})
			}
			data.ComplexTypes = append(data.ComplexTypes, cd)
		}

		for _, et := range s.EnumTypes {
			ed := EnumTypeData{
				Name:   et.Name,
				GoName: PascalCase(et.Name),
			}
			for _, m := range et.Members {
				ed.Members = append(ed.Members, EnumMemberData{
					Name:   m.Name,
					GoName: PascalCase(m.Name),
					Value:  m.Value,
				})
			}
			data.EnumTypes = append(data.EnumTypes, ed)
		}
	}

	return data
}

// buildClientData collects template data for client.go.
func (cg *CodeGenerator) buildClientData() ClientTemplateData {
	data := ClientTemplateData{PackageName: cg.pkgName}
	for _, s := range cg.schemas {
		for _, es := range s.EntitySets {
			data.EntitySets = append(data.EntitySets, EntitySetData{
				Name:       es.Name,
				GoName:     PascalCase(es.Name),
				EntityType: es.EntityType,
				QueryType:  PascalCase(es.Name) + "Query",
			})
		}
	}
	return data
}

// buildQueriesData collects template data for queries.go.
func (cg *CodeGenerator) buildQueriesData() QueriesTemplateData {
	data := QueriesTemplateData{PackageName: cg.pkgName}
	for _, s := range cg.schemas {
		for _, es := range s.EntitySets {
			data.QueryBuilders = append(data.QueryBuilders, QueryBuilderData{
				Name:       es.Name,
				GoName:     PascalCase(es.Name),
				QueryType:  PascalCase(es.Name) + "Query",
				EntityType: es.EntityType,
			})
		}
	}
	return data
}

// PascalCase converts a string to PascalCase.
// Strings already in PascalCase (no separators, starts with uppercase) are returned as-is.
func PascalCase(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	hasSep := strings.ContainsAny(s, "_ .-")
	if !hasSep {
		// Already PascalCase or a single word - just ensure first letter is upper.
		return strings.ToUpper(s[:1]) + s[1:]
	}
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
