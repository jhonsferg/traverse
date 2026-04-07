package azure

import "context"

// ChangePublisher is a traverse middleware that publishes entity change events
// to Event Grid after successful Create/Update/Delete operations.
type ChangePublisher struct {
	client    *EventGridClient
	entitySet string
}

// NewChangePublisher creates a ChangePublisher for the given entity set.
func NewChangePublisher(client *EventGridClient, entitySet string) *ChangePublisher {
	return &ChangePublisher{
		client:    client,
		entitySet: entitySet,
	}
}

// AfterCreate publishes a "created" event. Call this after a successful POST.
func (p *ChangePublisher) AfterCreate(ctx context.Context, entityID string, newValue interface{}) error {
	return p.client.PublishEntityChange(ctx, p.entitySet, entityID, "created", nil, newValue)
}

// AfterUpdate publishes an "updated" event. Call this after a successful PATCH/PUT.
func (p *ChangePublisher) AfterUpdate(ctx context.Context, entityID string, oldValue, newValue interface{}) error {
	return p.client.PublishEntityChange(ctx, p.entitySet, entityID, "updated", oldValue, newValue)
}

// AfterDelete publishes a "deleted" event. Call this after a successful DELETE.
func (p *ChangePublisher) AfterDelete(ctx context.Context, entityID string, oldValue interface{}) error {
	return p.client.PublishEntityChange(ctx, p.entitySet, entityID, "deleted", oldValue, nil)
}
