// Package openapi converts OData EDMX/CSDL metadata into an OpenAPI 3.x document.
package openapi

import (
	"fmt"
	"strings"

	"github.com/jhonsferg/traverse"
)

// OpenAPI represents an OpenAPI 3.1 document.
type OpenAPI struct {
	OpenAPI    string               `json:"openapi"`
	Info       Info                 `json:"info"`
	Servers    []Server             `json:"servers,omitempty"`
	Paths      map[string]*PathItem `json:"paths"`
	Components *Components          `json:"components,omitempty"`
}

// Info holds API metadata.
type Info struct {
	Title   string `json:"title"`
	Version string `json:"version"`
}

// Server represents an API server entry.
type Server struct {
	URL string `json:"url"`
}

// PathItem holds operations for a single path.
type PathItem struct {
	Get    *Operation `json:"get,omitempty"`
	Post   *Operation `json:"post,omitempty"`
	Patch  *Operation `json:"patch,omitempty"`
	Delete *Operation `json:"delete,omitempty"`
}

// Operation describes a single API operation.
type Operation struct {
	Summary     string              `json:"summary,omitempty"`
	Tags        []string            `json:"tags,omitempty"`
	Parameters  []Parameter         `json:"parameters,omitempty"`
	RequestBody *RequestBody        `json:"requestBody,omitempty"`
	Responses   map[string]Response `json:"responses"`
}

// Parameter describes a single operation parameter.
type Parameter struct {
	Name     string `json:"name"`
	In       string `json:"in"`
	Required bool   `json:"required"`
	Schema   Schema `json:"schema"`
}

// RequestBody describes the body of a request.
type RequestBody struct {
	Required bool                 `json:"required"`
	Content  map[string]MediaType `json:"content"`
}

// MediaType holds the schema for a media type.
type MediaType struct {
	Schema *Schema `json:"schema"`
}

// Response describes a single HTTP response.
type Response struct {
	Description string               `json:"description"`
	Content     map[string]MediaType `json:"content,omitempty"`
}

// Components holds reusable OpenAPI objects.
type Components struct {
	Schemas map[string]*Schema `json:"schemas,omitempty"`
}

// Schema represents a JSON Schema / OpenAPI Schema Object.
// The Ref field serialises as "$ref" and, when set, is the only field emitted.
type Schema struct {
	Ref         string             `json:"$ref,omitempty"`
	Type        string             `json:"type,omitempty"`
	Format      string             `json:"format,omitempty"`
	Properties  map[string]*Schema `json:"properties,omitempty"`
	Required    []string           `json:"required,omitempty"`
	Items       *Schema            `json:"items,omitempty"`
	Description string             `json:"description,omitempty"`
}

// options holds Export configuration.
type options struct {
	title             string
	version           string
	serverURLs        []string
	tagsFromNamespace bool
}

// Option configures the Export function.
type Option func(*options)

// WithTitle sets the OpenAPI info.title.
func WithTitle(title string) Option {
	return func(o *options) { o.title = title }
}

// WithVersion sets the OpenAPI info.version.
func WithVersion(version string) Option {
	return func(o *options) { o.version = version }
}

// WithServerURL adds a server entry to the OpenAPI document.
func WithServerURL(url string) Option {
	return func(o *options) { o.serverURLs = append(o.serverURLs, url) }
}

// WithTagsFromNamespace groups operations by OData namespace.
func WithTagsFromNamespace() Option {
	return func(o *options) { o.tagsFromNamespace = true }
}

// Export converts OData Metadata into an OpenAPI 3.1 document.
func Export(meta *traverse.Metadata, opts ...Option) (*OpenAPI, error) {
	if meta == nil {
		return nil, fmt.Errorf("openapi: metadata must not be nil")
	}

	o := &options{
		title:   "OData Service",
		version: "1.0.0",
	}
	for _, opt := range opts {
		opt(o)
	}

	doc := &OpenAPI{
		OpenAPI: "3.1.0",
		Info: Info{
			Title:   o.title,
			Version: o.version,
		},
		Paths: make(map[string]*PathItem),
		Components: &Components{
			Schemas: make(map[string]*Schema),
		},
	}

	for _, url := range o.serverURLs {
		doc.Servers = append(doc.Servers, Server{URL: url})
	}

	// Build a lookup map: unqualified name → EntityType.
	typeByName := make(map[string]*traverse.EntityType, len(meta.EntityTypes))
	for i := range meta.EntityTypes {
		et := &meta.EntityTypes[i]
		typeByName[et.Name] = et
		if meta.Namespace != "" {
			typeByName[meta.Namespace+"."+et.Name] = et
		}
	}

	// Build schemas for all EntityTypes.
	for i := range meta.EntityTypes {
		et := &meta.EntityTypes[i]
		doc.Components.Schemas[et.Name] = buildEntitySchema(et)
	}

	// Build paths for all EntitySets.
	for _, es := range meta.EntitySets {
		et := resolveEntityType(es.EntityTypeName, typeByName)

		var tags []string
		if o.tagsFromNamespace && meta.Namespace != "" {
			tags = []string{meta.Namespace}
		}

		refSchema := &Schema{Ref: "#/components/schemas/" + unqualifiedName(es.EntityTypeName)}

		// Collection path: /{EntitySetName}
		doc.Paths["/"+es.Name] = &PathItem{
			Get:  buildListOp(es.Name, refSchema, tags),
			Post: buildCreateOp(es.Name, refSchema, tags),
		}

		// Item path: /{EntitySetName}({KeyParam})
		if et != nil && len(et.Key) > 0 {
			keyParams, keySegment := buildKeySegment(et)
			itemPath := "/" + es.Name + keySegment
			doc.Paths[itemPath] = &PathItem{
				Get:    buildGetOp(es.Name, refSchema, keyParams, tags),
				Patch:  buildUpdateOp(es.Name, refSchema, keyParams, tags),
				Delete: buildDeleteOp(es.Name, keyParams, tags),
			}
		}
	}

	return doc, nil
}

// unqualifiedName returns the local part of a namespace-qualified name.
func unqualifiedName(qualifiedName string) string {
	if idx := strings.LastIndex(qualifiedName, "."); idx >= 0 {
		return qualifiedName[idx+1:]
	}
	return qualifiedName
}

// resolveEntityType looks up an EntityType by qualified or unqualified name.
func resolveEntityType(name string, byName map[string]*traverse.EntityType) *traverse.EntityType {
	if et, ok := byName[name]; ok {
		return et
	}
	return byName[unqualifiedName(name)]
}

// buildEntitySchema converts an EntityType into a JSON Schema object.
func buildEntitySchema(et *traverse.EntityType) *Schema {
	s := &Schema{
		Type:       "object",
		Properties: make(map[string]*Schema),
	}

	keySet := make(map[string]bool, len(et.Key))
	for _, kr := range et.Key {
		keySet[kr.Name] = true
	}

	for i := range et.Properties {
		p := &et.Properties[i]
		s.Properties[p.Name] = odataTypeToSchema(p.Type)
		if !p.Nullable && !keySet[p.Name] {
			s.Required = append(s.Required, p.Name)
		}
	}

	return s
}

// odataTypeToSchema maps an OData primitive type to a JSON Schema snippet.
func odataTypeToSchema(odataType string) *Schema {
	switch odataType {
	case "Edm.String":
		return &Schema{Type: "string"}
	case "Edm.Int16", "Edm.Int32", "Edm.Int64", "Edm.Byte", "Edm.SByte":
		return &Schema{Type: "integer"}
	case "Edm.Decimal", "Edm.Double", "Edm.Single":
		return &Schema{Type: "number"}
	case "Edm.Boolean":
		return &Schema{Type: "boolean"}
	case "Edm.DateTimeOffset", "Edm.DateTime":
		return &Schema{Type: "string", Format: "date-time"}
	case "Edm.Date":
		return &Schema{Type: "string", Format: "date"}
	case "Edm.TimeOfDay", "Edm.Time":
		return &Schema{Type: "string", Format: "time"}
	case "Edm.Guid":
		return &Schema{Type: "string", Format: "uuid"}
	case "Edm.Binary":
		return &Schema{Type: "string", Format: "byte"}
	case "Edm.Stream":
		return &Schema{Type: "string", Format: "binary"}
	default:
		return &Schema{Type: "string"}
	}
}

// buildKeySegment constructs the OData key path segment and corresponding parameters.
func buildKeySegment(et *traverse.EntityType) ([]Parameter, string) {
	params := make([]Parameter, 0, len(et.Key))

	if len(et.Key) == 1 {
		keyName := et.Key[0].Name
		keyType := findPropertyType(et, keyName)
		params = append(params, Parameter{
			Name:     keyName,
			In:       "path",
			Required: true,
			Schema:   *odataTypeToSchema(keyType),
		})
		return params, "({" + keyName + "})"
	}

	// Composite key: key1={key1},key2={key2}
	parts := make([]string, 0, len(et.Key))
	for _, kr := range et.Key {
		keyType := findPropertyType(et, kr.Name)
		params = append(params, Parameter{
			Name:     kr.Name,
			In:       "path",
			Required: true,
			Schema:   *odataTypeToSchema(keyType),
		})
		parts = append(parts, kr.Name+"={"+kr.Name+"}")
	}
	return params, "(" + strings.Join(parts, ",") + ")"
}

// findPropertyType returns the OData type of the named property, defaulting to Edm.String.
func findPropertyType(et *traverse.EntityType, name string) string {
	for _, p := range et.Properties {
		if p.Name == name {
			return p.Type
		}
	}
	return "Edm.String"
}

func jsonContent(s *Schema) map[string]MediaType {
	return map[string]MediaType{"application/json": {Schema: s}}
}

func buildListOp(name string, itemRef *Schema, tags []string) *Operation {
	return &Operation{
		Summary: "List " + name,
		Tags:    tags,
		Responses: map[string]Response{
			"200": {
				Description: "List of " + name,
				Content: jsonContent(&Schema{
					Type: "object",
					Properties: map[string]*Schema{
						"value": {Type: "array", Items: itemRef},
					},
				}),
			},
		},
	}
}

func buildCreateOp(name string, schema *Schema, tags []string) *Operation {
	return &Operation{
		Summary: "Create " + name,
		Tags:    tags,
		RequestBody: &RequestBody{
			Required: true,
			Content:  jsonContent(schema),
		},
		Responses: map[string]Response{
			"201": {Description: "Created", Content: jsonContent(schema)},
		},
	}
}

func buildGetOp(name string, schema *Schema, keyParams []Parameter, tags []string) *Operation {
	return &Operation{
		Summary:    "Get " + name + " by key",
		Tags:       tags,
		Parameters: keyParams,
		Responses: map[string]Response{
			"200": {Description: "OK", Content: jsonContent(schema)},
			"404": {Description: "Not Found"},
		},
	}
}

func buildUpdateOp(name string, schema *Schema, keyParams []Parameter, tags []string) *Operation {
	return &Operation{
		Summary:    "Update " + name,
		Tags:       tags,
		Parameters: keyParams,
		RequestBody: &RequestBody{
			Required: true,
			Content:  jsonContent(schema),
		},
		Responses: map[string]Response{
			"200": {Description: "Updated", Content: jsonContent(schema)},
			"404": {Description: "Not Found"},
		},
	}
}

func buildDeleteOp(name string, keyParams []Parameter, tags []string) *Operation {
	return &Operation{
		Summary:    "Delete " + name,
		Tags:       tags,
		Parameters: keyParams,
		Responses: map[string]Response{
			"204": {Description: "No Content"},
			"404": {Description: "Not Found"},
		},
	}
}
