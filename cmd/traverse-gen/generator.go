package main

import (
	"fmt"
	"strings"
)

// odataTypeMap maps OData primitive types to Go types (base, without pointer/slice decoration).
// Nullable handling is applied separately by mapODataType.
var odataTypeMap = map[string]string{
	"Edm.String":         "string",
	"Edm.Int16":          "int16",
	"Edm.Int32":          "int32",
	"Edm.Int64":          "int64",
	"Edm.Boolean":        "bool",
	"Edm.Decimal":        "traverse.Decimal",
	"Edm.Single":         "float32",
	"Edm.Double":         "float64",
	"Edm.DateTime":       "traverse.DateTime",
	"Edm.DateTimeOffset": "traverse.DateTimeOffset",
	"Edm.Time":           "time.Duration",
	"Edm.Date":           "time.Time",
	"Edm.TimeOfDay":      "time.Time",
	"Edm.Duration":       "time.Duration",
	"Edm.Guid":           "traverse.Guid",
	"Edm.Byte":           "byte",
	"Edm.SByte":          "int8",
	"Edm.Binary":         "[]byte",
	"Edm.Stream":         "[]byte",
}

// noPointerTypes holds Go types that must not be pointer-wrapped even when nullable,
// because they are already reference types or slices.
var noPointerTypes = map[string]bool{
	"[]byte": true,
}

// mapODataType converts an OData type string to a Go type string.
// For nullable non-collection non-scalar types, a pointer prefix is added.
func mapODataType(odataType string, nullable bool) string {
	if strings.HasPrefix(odataType, "Collection(") {
		inner := strings.TrimSuffix(strings.TrimPrefix(odataType, "Collection("), ")")
		return "[]" + mapODataType(inner, false)
	}

	if goType, ok := odataTypeMap[odataType]; ok {
		if nullable && !noPointerTypes[goType] && !strings.HasPrefix(goType, "[]") {
			return "*" + goType
		}
		return goType
	}

	// Unknown / complex type - treat as a generated struct type.
	// Strip namespace prefix if present.
	name := unqualifiedName(odataType)
	if nullable {
		return "*" + name
	}
	return name
}

// needsTimeImport reports whether any of the schemas use a time.Time or time.Duration field.
func needsTimeImport(schemas []Schema) bool {
	check := func(t string) bool {
		gt := mapODataType(t, false)
		return strings.Contains(gt, "time.")
	}
	for _, s := range schemas {
		for _, et := range s.EntityTypes {
			for _, p := range et.Properties {
				if check(p.Type) {
					return true
				}
			}
		}
		for _, ct := range s.ComplexTypes {
			for _, p := range ct.Properties {
				if check(p.Type) {
					return true
				}
			}
		}
	}
	return false
}

// needsTraverseImport reports whether any of the schemas use a traverse.* field type.
func needsTraverseImport(schemas []Schema) bool {
	check := func(t string) bool {
		gt := mapODataType(t, false)
		return strings.Contains(gt, "traverse.")
	}
	for _, s := range schemas {
		for _, et := range s.EntityTypes {
			for _, p := range et.Properties {
				if check(p.Type) {
					return true
				}
			}
		}
		for _, ct := range s.ComplexTypes {
			for _, p := range ct.Properties {
				if check(p.Type) {
					return true
				}
			}
		}
	}
	return false
}

// buildStructTag builds the backtick-enclosed struct tag for a property field.
// Rules:
//   - Key property:              `json:"Name" odata:"key"`
//   - Nav collection:            `json:"Name,omitempty" odata:"nav"`
//   - Nav single (pointer):      `json:"Name,omitempty" odata:"nav"`
//   - Nullable regular property: `json:"Name,omitempty"`
//   - Non-nullable regular:      `json:"Name"`
func buildStructTag(propName string, isKey, isNav, isNullable bool) string {
	jsonOptions := ""
	if isNullable && !isKey {
		jsonOptions = ",omitempty"
	}
	jsonPart := fmt.Sprintf(`json:"%s%s"`, propName, jsonOptions)

	var odataParts []string
	if isKey {
		odataParts = append(odataParts, "key")
	}
	if isNav {
		odataParts = append(odataParts, "nav")
	}

	if len(odataParts) > 0 {
		return fmt.Sprintf("`%s odata:\"%s\"`", jsonPart, strings.Join(odataParts, ","))
	}
	return fmt.Sprintf("`%s`", jsonPart)
}

// isKeyProp reports whether propName is in the entity's key list.
func isKeyProp(keys []string, propName string) bool {
	for _, k := range keys {
		if k == propName {
			return true
		}
	}
	return false
}

// navGoType returns the Go type for a navigation property.
// Collections become slices, single nav props become pointers.
func navGoType(nav SchemaNavProp) string {
	if nav.IsCollection {
		return "[]" + nav.TargetType
	}
	return "*" + nav.TargetType
}
