package graphql

import (
	"fmt"
	"strings"

	gql "github.com/graphql-go/graphql"

	"github.com/jhonsferg/traverse"
)

// SchemaBuilder builds a GraphQL schema from OData metadata.
type SchemaBuilder struct {
	metadata    *traverse.Metadata
	client      *traverse.Client
	typeCache   map[string]gql.Type
	typeResolve map[string]*gql.Object
}

// NewSchemaBuilder creates a new schema builder.
func NewSchemaBuilder(metadata *traverse.Metadata, client *traverse.Client) *SchemaBuilder {
	return &SchemaBuilder{
		metadata:    metadata,
		client:      client,
		typeCache:   make(map[string]gql.Type),
		typeResolve: make(map[string]*gql.Object),
	}
}

// Build generates a GraphQL schema from the OData metadata.
func (sb *SchemaBuilder) Build() (*gql.Schema, error) {
	// Build entity types first (for potential circular references)
	for _, et := range sb.metadata.EntityTypes {
		if err := sb.buildEntityType(et); err != nil {
			return nil, fmt.Errorf("failed to build entity type %s: %w", et.Name, err)
		}
	}

	// Build root query type
	queryType := sb.buildQueryType()

	// Build root mutation type
	mutationType := sb.buildMutationType()

	// Create schema
	config := gql.SchemaConfig{
		Query: queryType,
	}

	if mutationType != nil {
		config.Mutation = mutationType
	}

	schema, err := gql.NewSchema(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create schema: %w", err)
	}

	return &schema, nil
}

// buildEntityType builds a GraphQL object type from an OData entity type.
func (sb *SchemaBuilder) buildEntityType(et traverse.EntityType) error {
	if _, exists := sb.typeResolve[et.Name]; exists {
		return nil // Already built
	}

	fields := gql.Fields{}

	// Add properties
	for _, prop := range et.Properties {
		gqlType := sb.odataTypeToGraphQL(prop.Type, prop.Nullable)
		if gqlType == nil {
			continue
		}

		fields[prop.Name] = &gql.Field{
			Type:        gqlType,
			Description: fmt.Sprintf("%s (%s)", prop.Name, prop.Type),
		}
	}

	// Add navigation properties
	for _, navProp := range et.NavigationProperties {
		navType := sb.findEntityType(navProp.ToEntityType)
		if navType == nil {
			continue
		}

		if strings.Contains(navProp.ToMultiplicity, "*") {
			// Collection (many)
			fields[navProp.Name] = &gql.Field{
				Type:        gql.NewList(navType),
				Description: fmt.Sprintf("Navigation to %s (collection)", navProp.ToEntityType),
				Resolve: func(p gql.ResolveParams) (interface{}, error) {
					return nil, fmt.Errorf("not implemented")
				},
			}
		} else {
			// Single
			fields[navProp.Name] = &gql.Field{
				Type:        navType,
				Description: fmt.Sprintf("Navigation to %s", navProp.ToEntityType),
				Resolve: func(p gql.ResolveParams) (interface{}, error) {
					return nil, fmt.Errorf("not implemented")
				},
			}
		}
	}

	objType := gql.NewObject(gql.ObjectConfig{
		Name:        et.Name,
		Fields:      fields,
		Description: fmt.Sprintf("OData entity type %s", et.Name),
	})

	sb.typeResolve[et.Name] = objType
	sb.typeCache[et.Name] = objType

	return nil
}

// buildQueryType builds the root Query type with fields for each entity set.
func (sb *SchemaBuilder) buildQueryType() *gql.Object {
	fields := gql.Fields{}

	for _, es := range sb.metadata.EntitySets {
		// Find the corresponding entity type
		entityType := sb.findEntityType(es.EntityTypeName)
		if entityType == nil {
			continue
		}

		// Query field for single item
		fields[es.Name] = &gql.Field{
			Type: gql.NewList(entityType),
			Args: gql.FieldConfigArgument{
				"filter": &gql.ArgumentConfig{
					Type:        gql.String,
					Description: "OData filter expression",
				},
				"orderBy": &gql.ArgumentConfig{
					Type:        gql.String,
					Description: "OData order by expression",
				},
				"top": &gql.ArgumentConfig{
					Type:        gql.Int,
					Description: "Number of items to return",
				},
				"skip": &gql.ArgumentConfig{
					Type:        gql.Int,
					Description: "Number of items to skip",
				},
			},
			Description: fmt.Sprintf("Query %s entity set", es.Name),
			Resolve:     NewQueryResolver(sb.client, es.Name, entityType).Resolve,
		}

		// Query field for single item by key (if entity type has a key)
		etDef := findEntityTypeByName(sb.metadata, es.EntityTypeName)
		if etDef != nil && len(etDef.Key) > 0 {
			keyType := gql.String
			fields[es.Name+"_key"] = &gql.Field{
				Type: entityType,
				Args: gql.FieldConfigArgument{
					"key": &gql.ArgumentConfig{
						Type:        gql.NewNonNull(keyType),
						Description: "Entity key",
					},
				},
				Description: fmt.Sprintf("Query %s by key", es.Name),
				Resolve:     NewQueryResolver(sb.client, es.Name, entityType).ResolveByKey,
			}
		}
	}

	return gql.NewObject(gql.ObjectConfig{
		Name:        "Query",
		Fields:      fields,
		Description: "Root query type",
	})
}

// buildMutationType builds the root Mutation type.
func (sb *SchemaBuilder) buildMutationType() *gql.Object {
	fields := gql.Fields{}

	for _, es := range sb.metadata.EntitySets {
		entityType := sb.findEntityType(es.EntityTypeName)
		if entityType == nil {
			continue
		}

		// Create mutation
		fields["create"+es.Name] = &gql.Field{
			Type: entityType,
			Args: gql.FieldConfigArgument{
				"input": &gql.ArgumentConfig{
					Type: gql.NewNonNull(gql.NewInputObject(
						gql.InputObjectConfig{
							Name:        es.Name + "Input",
							Fields:      sb.buildInputFields(entityType),
							Description: fmt.Sprintf("Input for creating %s", es.Name),
						},
					)),
					Description: "Input data",
				},
			},
			Description: fmt.Sprintf("Create %s", es.Name),
			Resolve:     NewMutationResolver(sb.client, es.Name).ResolveCreate,
		}

		// Update mutation
		fields["update"+es.Name] = &gql.Field{
			Type: entityType,
			Args: gql.FieldConfigArgument{
				"key": &gql.ArgumentConfig{
					Type:        gql.NewNonNull(gql.String),
					Description: "Entity key",
				},
				"input": &gql.ArgumentConfig{
					Type: gql.NewNonNull(gql.NewInputObject(
						gql.InputObjectConfig{
							Name:        es.Name + "UpdateInput",
							Fields:      sb.buildInputFields(entityType),
							Description: fmt.Sprintf("Update input for %s", es.Name),
						},
					)),
					Description: "Update data",
				},
			},
			Description: fmt.Sprintf("Update %s", es.Name),
			Resolve:     NewMutationResolver(sb.client, es.Name).ResolveUpdate,
		}

		// Delete mutation
		fields["delete"+es.Name] = &gql.Field{
			Type: gql.Boolean,
			Args: gql.FieldConfigArgument{
				"key": &gql.ArgumentConfig{
					Type:        gql.NewNonNull(gql.String),
					Description: "Entity key",
				},
			},
			Description: fmt.Sprintf("Delete %s", es.Name),
			Resolve:     NewMutationResolver(sb.client, es.Name).ResolveDelete,
		}
	}

	if len(fields) == 0 {
		return nil
	}

	return gql.NewObject(gql.ObjectConfig{
		Name:        "Mutation",
		Fields:      fields,
		Description: "Root mutation type",
	})
}

// buildInputFields builds input fields for a GraphQL input type.
func (sb *SchemaBuilder) buildInputFields(objType *gql.Object) gql.InputObjectConfigFieldMap {
	fields := gql.InputObjectConfigFieldMap{}

	if objType == nil {
		return fields
	}

	// Extract fields from the object type
	objFields := objType.Fields()
	for name, field := range objFields {
		if field == nil {
			continue
		}

		// Convert output type to input type
		inputType := sb.outputTypeToInputType(field.Type)
		if inputType != nil {
			fields[name] = &gql.InputObjectFieldConfig{
				Type:        inputType,
				Description: fmt.Sprintf("Input for %s", name),
			}
		}
	}

	return fields
}

// outputTypeToInputType converts an output type to an input type.
func (sb *SchemaBuilder) outputTypeToInputType(t gql.Type) gql.Input {
	if t == nil {
		return nil
	}

	switch v := t.(type) {
	case *gql.NonNull:
		inner := sb.outputTypeToInputType(v.OfType)
		if inner != nil {
			return gql.NewNonNull(inner)
		}
	case *gql.List:
		inner := sb.outputTypeToInputType(v.OfType)
		if inner != nil {
			return gql.NewList(inner)
		}
	case *gql.Scalar:
		return v
	case *gql.Enum:
		return v
	}

	return nil
}

// odataTypeToGraphQL maps OData types to GraphQL types.
func (sb *SchemaBuilder) odataTypeToGraphQL(odataType string, nullable bool) gql.Type {
	var baseType gql.Type

	switch {
	case strings.HasPrefix(odataType, "Edm.String"):
		baseType = gql.String
	case strings.HasPrefix(odataType, "Edm.Int32"):
		baseType = gql.Int
	case strings.HasPrefix(odataType, "Edm.Int64"):
		baseType = gql.String // JSON safe
	case strings.HasPrefix(odataType, "Edm.Double"):
		baseType = gql.Float
	case strings.HasPrefix(odataType, "Edm.Boolean"):
		baseType = gql.Boolean
	case strings.HasPrefix(odataType, "Edm.DateTime"):
		baseType = gql.String
	case strings.HasPrefix(odataType, "Edm.DateTimeOffset"):
		baseType = gql.String
	case strings.HasPrefix(odataType, "Edm.Guid"):
		baseType = gql.String
	case strings.HasPrefix(odataType, "Edm.Decimal"):
		baseType = gql.String // JSON safe
	default:
		return nil
	}

	if nullable {
		return baseType
	}

	return gql.NewNonNull(baseType)
}

// findEntityType finds an entity type object in the cache.
func (sb *SchemaBuilder) findEntityType(name string) *gql.Object {
	if t, exists := sb.typeResolve[name]; exists {
		return t
	}
	return nil
}

// findEntityTypeByName finds an entity type definition by name.
func findEntityTypeByName(metadata *traverse.Metadata, name string) *traverse.EntityType {
	for i := range metadata.EntityTypes {
		if metadata.EntityTypes[i].Name == name {
			return &metadata.EntityTypes[i]
		}
	}
	return nil
}
