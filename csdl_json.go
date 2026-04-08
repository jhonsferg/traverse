package traverse

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// ParseCSDLJSON parses an OData CSDL JSON v4.01 document and returns the service
// metadata. The input may be a full service document or just the $metadata
// endpoint response in JSON format.
//
// CSDL JSON is the JSON alternative to the XML EDMX format used by services such
// as Microsoft Graph. The top-level object contains $Version and $EntityContainer
// annotations alongside schema namespace objects whose entries describe entity
// types, complex types, enum types, actions, functions, and entity containers.
//
// Returns [ErrMetadataInvalid] wrapped in the error if the document cannot be
// decoded as valid JSON.
func ParseCSDLJSON(data []byte) (*Metadata, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("traverse: empty CSDL JSON input: %w", ErrMetadataInvalid)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("traverse: invalid CSDL JSON: %w", ErrMetadataInvalid)
	}

	return parseCSDLMap(raw)
}

// ParseCSDLJSONReader is like [ParseCSDLJSON] but reads from an [io.Reader].
//
// ParseCSDLJSONReader is convenient when the CSDL JSON document is available as
// an HTTP response body or other streaming source.
func ParseCSDLJSONReader(r io.Reader) (*Metadata, error) {
	if r == nil {
		return nil, fmt.Errorf("traverse: nil reader for CSDL JSON: %w", ErrMetadataInvalid)
	}

	var raw map[string]json.RawMessage
	if err := json.NewDecoder(r).Decode(&raw); err != nil {
		return nil, fmt.Errorf("traverse: invalid CSDL JSON: %w", ErrMetadataInvalid)
	}

	return parseCSDLMap(raw)
}

// parseCSDLMap converts the top-level decoded map into a [Metadata] value.
func parseCSDLMap(raw map[string]json.RawMessage) (*Metadata, error) {
	md := &Metadata{
		EntityTypes:  make([]EntityType, 0),
		EntitySets:   make([]EntitySetInfo, 0),
		Associations: make([]Association, 0),
		Actions:      make([]ActionInfo, 0),
		Functions:    make([]FunctionInfo, 0),
		ComplexTypes: make([]ComplexType, 0),
		EnumTypes:    make([]EnumType, 0),
	}

	// Extract $Version.
	if v, ok := raw["$Version"]; ok {
		_ = json.Unmarshal(v, &md.Version)
	}

	// Extract $EntityContainer to discover the default namespace.
	var defaultContainer string
	if ec, ok := raw["$EntityContainer"]; ok {
		_ = json.Unmarshal(ec, &defaultContainer)
	}

	// Process each top-level key that is a schema namespace (non-$ keys that
	// decode as objects with optional "$Kind": "Schema").
	for key, val := range raw {
		if strings.HasPrefix(key, "$") {
			continue
		}

		var obj map[string]json.RawMessage
		if err := json.Unmarshal(val, &obj); err != nil {
			// Not an object  -  skip.
			continue
		}

		// A namespace object may carry "$Kind": "Schema" but the key is also
		// absent in some documents. We treat any non-$ top-level object as a
		// potential schema.
		if kind := csdlKind(obj); kind != "" && kind != "Schema" {
			continue
		}

		// Track the first namespace encountered as the primary one.
		if md.Namespace == "" {
			md.Namespace = key
		}

		parseCSDLSchema(key, obj, md)
	}

	return md, nil
}

// parseCSDLSchema processes one schema namespace object and appends the parsed
// definitions to md.
func parseCSDLSchema(namespace string, obj map[string]json.RawMessage, md *Metadata) {
	for name, val := range obj {
		if strings.HasPrefix(name, "$") {
			continue
		}

		// OData CSDL JSON allows Function/Action overloads as JSON arrays.
		// Try array format first (OData spec 4.0 section 5.1.1.2).
		if len(val) > 0 && val[0] == '[' {
			var overloads []map[string]json.RawMessage
			if err := json.Unmarshal(val, &overloads); err == nil {
				for _, overload := range overloads {
					switch csdlKind(overload) {
					case "Function":
						fi := parseCSDLFunction(name, overload)
						md.Functions = append(md.Functions, fi)
					case "Action":
						ai := parseCSDLAction(name, overload)
						md.Actions = append(md.Actions, ai)
					}
				}
			}
			continue
		}

		var child map[string]json.RawMessage
		if err := json.Unmarshal(val, &child); err != nil {
			continue
		}

		switch csdlKind(child) {
		case "EntityType":
			et := parseCSDLEntityType(name, child)
			md.EntityTypes = append(md.EntityTypes, et)

		case "EntitySet":
			es := parseCSDLEntitySet(name, namespace, child)
			md.EntitySets = append(md.EntitySets, es)

		case "EntityContainer":
			parseCSDLEntityContainer(name, namespace, child, md)

		case "ComplexType":
			ct := parseCSDLComplexType(name, child)
			md.ComplexTypes = append(md.ComplexTypes, ct)

		case "EnumType":
			en := parseCSDLEnumType(name, child)
			md.EnumTypes = append(md.EnumTypes, en)

		case "Action":
			ai := parseCSDLAction(name, child)
			md.Actions = append(md.Actions, ai)

		case "Function":
			fi := parseCSDLFunction(name, child)
			md.Functions = append(md.Functions, fi)

		default:
			// Unknown or absent $Kind - silently skip for forward compatibility.
		}
	}
}

// parseCSDLEntityType parses a CSDL entity type object.
func parseCSDLEntityType(name string, obj map[string]json.RawMessage) EntityType {
	et := EntityType{
		Name:                 name,
		Key:                  make([]PropertyRef, 0),
		Properties:           make([]Property, 0),
		NavigationProperties: make([]NavigationProperty, 0),
	}

	// $Key is an array of property name strings.
	if raw, ok := obj["$Key"]; ok {
		var keys []json.RawMessage
		if err := json.Unmarshal(raw, &keys); err == nil {
			for _, k := range keys {
				var keyRef string
				if err2 := json.Unmarshal(k, &keyRef); err2 == nil {
					et.Key = append(et.Key, PropertyRef{Name: keyRef})
				} else {
					// Key alias object: {"alias": "PropertyPath"}
					var aliasMap map[string]string
					if err3 := json.Unmarshal(k, &aliasMap); err3 == nil {
						for _, path := range aliasMap {
							et.Key = append(et.Key, PropertyRef{Name: path})
						}
					}
				}
			}
		}
	}

	// Remaining non-$ keys are structural or navigation properties.
	for propName, propVal := range obj {
		if strings.HasPrefix(propName, "$") {
			continue
		}

		var propObj map[string]json.RawMessage
		if err := json.Unmarshal(propVal, &propObj); err != nil {
			continue
		}

		switch csdlKind(propObj) {
		case "NavigationProperty":
			np := parseCSDLNavProperty(propName, propObj)
			et.NavigationProperties = append(et.NavigationProperties, np)
		default:
			// Default (absent $Kind) is a structural property.
			p := parseCSDLProperty(propName, propObj)
			et.Properties = append(et.Properties, p)
		}
	}

	return et
}

// parseCSDLProperty parses a structural property definition.
func parseCSDLProperty(name string, obj map[string]json.RawMessage) Property {
	p := Property{
		Name:     name,
		Nullable: true, // OData default is nullable=true.
	}

	if raw, ok := obj["$Type"]; ok {
		_ = json.Unmarshal(raw, &p.Type)
	}
	// Absent $Type implies Edm.String per the spec.
	if p.Type == "" {
		p.Type = "Edm.String"
	}

	if raw, ok := obj["$Nullable"]; ok {
		_ = json.Unmarshal(raw, &p.Nullable)
	}

	if raw, ok := obj["$MaxLength"]; ok {
		var v int
		if err := json.Unmarshal(raw, &v); err == nil {
			p.MaxLength = &v
		}
	}

	if raw, ok := obj["$Precision"]; ok {
		var v int
		if err := json.Unmarshal(raw, &v); err == nil {
			p.Precision = &v
		}
	}

	if raw, ok := obj["$Scale"]; ok {
		var v int
		if err := json.Unmarshal(raw, &v); err == nil {
			p.Scale = &v
		}
	}

	return p
}

// parseCSDLNavProperty parses a navigation property definition.
func parseCSDLNavProperty(name string, obj map[string]json.RawMessage) NavigationProperty {
	np := NavigationProperty{Name: name}

	if raw, ok := obj["$Type"]; ok {
		_ = json.Unmarshal(raw, &np.ToEntityType)
	}

	if raw, ok := obj["$Partner"]; ok {
		_ = json.Unmarshal(raw, &np.Partner)
	}

	return np
}

// parseCSDLEntitySet parses a standalone EntitySet definition (outside a container).
func parseCSDLEntitySet(name, _ string, obj map[string]json.RawMessage) EntitySetInfo {
	es := EntitySetInfo{Name: name}

	if raw, ok := obj["$Type"]; ok {
		var typeName string
		if err := json.Unmarshal(raw, &typeName); err == nil {
			es.EntityTypeName = localName(typeName)
		}
	}

	return es
}

// parseCSDLEntityContainer processes an EntityContainer and appends entity sets to md.
func parseCSDLEntityContainer(_ string, namespace string, obj map[string]json.RawMessage, md *Metadata) {
	for esName, esVal := range obj {
		if strings.HasPrefix(esName, "$") {
			continue
		}

		var esObj map[string]json.RawMessage
		if err := json.Unmarshal(esVal, &esObj); err != nil {
			continue
		}

		// Container children without $Kind or with $Kind absent are EntitySets
		// when they reference an entity type. NavigationPropertyBindings and
		// Singletons are handled similarly but we emit them as EntitySets too.
		kind := csdlKind(esObj)
		if kind != "" && kind != "EntitySet" && kind != "Singleton" {
			continue
		}

		es := EntitySetInfo{Name: esName}

		if raw, ok := esObj["$Type"]; ok {
			var typeName string
			if err := json.Unmarshal(raw, &typeName); err == nil {
				es.EntityTypeName = localName(typeName)
			}
		} else if raw, ok := esObj["$EntityType"]; ok {
			// Older CSDL JSON drafts used $EntityType.
			var typeName string
			if err := json.Unmarshal(raw, &typeName); err == nil {
				es.EntityTypeName = localName(typeName)
			}
		}

		// Only append if we could determine the entity type; avoid duplicates.
		if es.EntityTypeName != "" && !entitySetExists(md.EntitySets, esName, namespace) {
			md.EntitySets = append(md.EntitySets, es)
		}
	}
}

// parseCSDLComplexType parses a complex type definition.
func parseCSDLComplexType(name string, obj map[string]json.RawMessage) ComplexType {
	ct := ComplexType{
		Name:       name,
		Properties: make([]Property, 0),
	}

	for propName, propVal := range obj {
		if strings.HasPrefix(propName, "$") {
			continue
		}

		var propObj map[string]json.RawMessage
		if err := json.Unmarshal(propVal, &propObj); err != nil {
			continue
		}

		if csdlKind(propObj) == "NavigationProperty" {
			continue
		}

		p := parseCSDLProperty(propName, propObj)
		ct.Properties = append(ct.Properties, p)
	}

	return ct
}

// parseCSDLEnumType parses an enum type definition.
func parseCSDLEnumType(name string, obj map[string]json.RawMessage) EnumType {
	en := EnumType{
		Name:    name,
		Members: make([]EnumMember, 0),
	}

	if raw, ok := obj["$UnderlyingType"]; ok {
		_ = json.Unmarshal(raw, &en.UnderlyingType)
	}

	if raw, ok := obj["$IsFlags"]; ok {
		_ = json.Unmarshal(raw, &en.IsFlags)
	}

	for memberName, memberVal := range obj {
		if strings.HasPrefix(memberName, "$") {
			continue
		}

		var v int
		if err := json.Unmarshal(memberVal, &v); err != nil {
			continue
		}

		en.Members = append(en.Members, EnumMember{Name: memberName, Value: v})
	}

	return en
}

// parseCSDLAction parses a bound or unbound action definition.
func parseCSDLAction(name string, obj map[string]json.RawMessage) ActionInfo {
	ai := ActionInfo{
		Name:       name,
		Parameters: parseCSDLParameters(obj),
	}

	if raw, ok := obj["$ReturnType"]; ok {
		var rt map[string]json.RawMessage
		if err := json.Unmarshal(raw, &rt); err == nil {
			if typeRaw, ok2 := rt["$Type"]; ok2 {
				_ = json.Unmarshal(typeRaw, &ai.ReturnType)
			}
		}
	}

	return ai
}

// parseCSDLFunction parses a bound or unbound function definition.
func parseCSDLFunction(name string, obj map[string]json.RawMessage) FunctionInfo {
	fi := FunctionInfo{
		Name:       name,
		Parameters: parseCSDLParameters(obj),
	}

	if raw, ok := obj["$IsComposable"]; ok {
		_ = json.Unmarshal(raw, &fi.IsComposable)
	}

	if raw, ok := obj["$ReturnType"]; ok {
		var rt map[string]json.RawMessage
		if err := json.Unmarshal(raw, &rt); err == nil {
			if typeRaw, ok2 := rt["$Type"]; ok2 {
				_ = json.Unmarshal(typeRaw, &fi.ReturnType)
			}
		}
	}

	return fi
}

// parseCSDLParameters extracts the $Parameter array from an action or function
// object and returns the parsed parameter list.
func parseCSDLParameters(obj map[string]json.RawMessage) []FunctionParameter {
	params := make([]FunctionParameter, 0)

	raw, ok := obj["$Parameter"]
	if !ok {
		return params
	}

	var paramList []map[string]json.RawMessage
	if err := json.Unmarshal(raw, &paramList); err != nil {
		return params
	}

	for _, pm := range paramList {
		p := FunctionParameter{Nullable: true}

		if n, ok2 := pm["$Name"]; ok2 {
			_ = json.Unmarshal(n, &p.Name)
		}

		if t, ok2 := pm["$Type"]; ok2 {
			_ = json.Unmarshal(t, &p.Type)
		}

		if n, ok2 := pm["$Nullable"]; ok2 {
			_ = json.Unmarshal(n, &p.Nullable)
		}

		params = append(params, p)
	}

	return params
}

// csdlKind extracts the $Kind value from a CSDL object, returning an empty
// string if the annotation is absent.
func csdlKind(obj map[string]json.RawMessage) string {
	raw, ok := obj["$Kind"]
	if !ok {
		return ""
	}

	var kind string
	if err := json.Unmarshal(raw, &kind); err != nil {
		return ""
	}

	return kind
}

// localName returns the local (unqualified) name from a fully qualified OData
// type name such as "MyService.Customer" → "Customer". If the name contains no
// dot the original value is returned unchanged.
func localName(qualifiedName string) string {
	if idx := strings.LastIndex(qualifiedName, "."); idx >= 0 {
		return qualifiedName[idx+1:]
	}

	return qualifiedName
}

// entitySetExists reports whether an EntitySetInfo with the given name already
// exists in the slice, preventing duplicates when the same entity set appears
// both in the container and as a top-level definition.
func entitySetExists(sets []EntitySetInfo, name, _ string) bool {
	for _, s := range sets {
		if s.Name == name {
			return true
		}
	}

	return false
}
