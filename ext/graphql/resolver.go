package graphql

import (
	"context"
	"fmt"

	gql "github.com/graphql-go/graphql"
	"github.com/graphql-go/graphql/language/ast"

	"github.com/jhonsferg/traverse"
)

// QueryResolver handles query field resolution by translating GraphQL to OData.
type QueryResolver struct {
	client    *traverse.Client
	entitySet string
	objType   *gql.Object
}

// NewQueryResolver creates a new query resolver.
func NewQueryResolver(client *traverse.Client, entitySet string, objType *gql.Object) *QueryResolver {
	return &QueryResolver{
		client:    client,
		entitySet: entitySet,
		objType:   objType,
	}
}

// Resolve handles list queries.
func (qr *QueryResolver) Resolve(p gql.ResolveParams) (interface{}, error) {
	ctx := p.Context
	if ctx == nil {
		ctx = context.Background()
	}

	qb := qr.client.From(qr.entitySet)

	// Apply filter
	if filter, ok := p.Args["filter"].(string); ok && filter != "" {
		qb = qb.Filter(filter)
	}

	// Apply order by
	if orderBy, ok := p.Args["orderBy"].(string); ok && orderBy != "" {
		qb = qb.OrderBy(orderBy)
	}

	// Apply top/limit
	if top, ok := p.Args["top"].(int); ok && top > 0 {
		qb = qb.Top(top)
	}

	// Apply skip
	if skip, ok := p.Args["skip"].(int); ok && skip > 0 {
		qb = qb.Skip(skip)
	}

	// Select only requested fields
	selectedFields := getSelectedFields(p)
	if len(selectedFields) > 0 {
		qb = qb.Select(selectedFields...)
	}

	// Execute query
	results, err := traverse.CollectAs[map[string]interface{}](qb, ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query %s: %w", qr.entitySet, err)
	}

	return results, nil
}

// ResolveByKey handles single entity queries by key.
func (qr *QueryResolver) ResolveByKey(p gql.ResolveParams) (interface{}, error) {
	ctx := p.Context
	if ctx == nil {
		ctx = context.Background()
	}

	key, ok := p.Args["key"].(string)
	if !ok || key == "" {
		return nil, fmt.Errorf("key argument is required")
	}

	qb := qr.client.From(qr.entitySet)

	// Select only requested fields
	selectedFields := getSelectedFields(p)
	if len(selectedFields) > 0 {
		qb = qb.Select(selectedFields...)
	}

	result, err := traverse.FindByKeyAs[map[string]interface{}](qb, ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to find %s by key: %w", qr.entitySet, err)
	}

	return result, nil
}

// MutationResolver handles create, update, and delete mutations.
type MutationResolver struct {
	client    *traverse.Client
	entitySet string
}

// NewMutationResolver creates a new mutation resolver.
func NewMutationResolver(client *traverse.Client, entitySet string) *MutationResolver {
	return &MutationResolver{
		client:    client,
		entitySet: entitySet,
	}
}

// ResolveCreate handles create mutations.
func (mr *MutationResolver) ResolveCreate(p gql.ResolveParams) (interface{}, error) {
	ctx := p.Context
	if ctx == nil {
		ctx = context.Background()
	}

	input, ok := p.Args["input"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("input argument is required")
	}

	result, err := traverse.CreateAs[map[string]interface{}](mr.client, ctx, mr.entitySet, input)
	if err != nil {
		return nil, fmt.Errorf("failed to create %s: %w", mr.entitySet, err)
	}

	return result, nil
}

// ResolveUpdate handles update mutations.
func (mr *MutationResolver) ResolveUpdate(p gql.ResolveParams) (interface{}, error) {
	ctx := p.Context
	if ctx == nil {
		ctx = context.Background()
	}

	key, ok := p.Args["key"].(string)
	if !ok || key == "" {
		return nil, fmt.Errorf("key argument is required")
	}

	input, ok := p.Args["input"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("input argument is required")
	}

	// Perform update
	if err := traverse.UpdateAs[map[string]interface{}](mr.client, ctx, mr.entitySet, key, input); err != nil {
		return nil, fmt.Errorf("failed to update %s: %w", mr.entitySet, err)
	}

	// Fetch updated entity
	qb := mr.client.From(mr.entitySet)
	result, err := traverse.FindByKeyAs[map[string]interface{}](qb, ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch updated %s: %w", mr.entitySet, err)
	}

	return result, nil
}

// ResolveDelete handles delete mutations.
func (mr *MutationResolver) ResolveDelete(p gql.ResolveParams) (interface{}, error) {
	ctx := p.Context
	if ctx == nil {
		ctx = context.Background()
	}

	key, ok := p.Args["key"].(string)
	if !ok || key == "" {
		return nil, fmt.Errorf("key argument is required")
	}

	if err := mr.client.Delete(ctx, mr.entitySet, key); err != nil {
		return nil, fmt.Errorf("failed to delete %s: %w", mr.entitySet, err)
	}

	return true, nil
}

// getSelectedFields extracts the top-level field names requested by the
// GraphQL client from the selection set of the first resolved field AST.
// The names are used to build an OData $select clause so that the server
// returns only the requested properties, reducing payload size.
//
// Only direct field selections are considered; fragments and inline fragments
// are ignored because they may span multiple entity types and cannot be
// reliably mapped to a flat $select list without schema introspection.
func getSelectedFields(p gql.ResolveParams) []string {
	if len(p.Info.FieldASTs) == 0 {
		return nil
	}
	ss := p.Info.FieldASTs[0].SelectionSet
	if ss == nil || len(ss.Selections) == 0 {
		return nil
	}
	fields := make([]string, 0, len(ss.Selections))
	for _, s := range ss.Selections {
		if f, ok := s.(*ast.Field); ok && f.Name != nil {
			fields = append(fields, f.Name.Value)
		}
	}
	return fields
}
