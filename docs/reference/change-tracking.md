# Change Tracking Reference

The `ChangeSet` type tracks mutations to entities and serialises them into an OData `$batch` changeset for atomic submission.

## Types

```go
// ChangeSet tracks entity mutations.
type ChangeSet[T any] struct { /* unexported */ }

// NewChangeSet creates a tracker for entities of type T.
func NewChangeSet[T any]() *ChangeSet[T]

// Track begins tracking an entity loaded from the server.
// Returns a mutable proxy; mutations are recorded automatically.
func (cs *ChangeSet[T]) Track(entity T) *T

// Dirty returns all entities with pending changes.
func (cs *ChangeSet[T]) Dirty() []ChangeEntry[T]

// Submit sends all pending changes as a single $batch request.
func (cs *ChangeSet[T]) Submit(ctx context.Context, client *Client) error

// Reset clears all recorded changes.
func (cs *ChangeSet[T]) Reset()

// ChangeEntry describes a single pending mutation.
type ChangeEntry[T any] struct {
    Entity    T
    Operation ChangeOperation // Create, Update, Delete
    Fields    []string        // which fields changed (for PATCH)
}

type ChangeOperation int

const (
    ChangeCreate ChangeOperation = iota
    ChangeUpdate
    ChangeDelete
)
```

## Basic usage

```go
cs := traverse.NewChangeSet[Product]()

// Load and track
var raw Product
client.Collection("Products").Get(ctx, 1, &raw)
product := cs.Track(raw)

// Mutate - changes are recorded
product.Price = 29.99
product.Discontinued = false

// Submit all changes as a single batch
err := cs.Submit(ctx, client)
```

## Inspecting pending changes

```go
for _, entry := range cs.Dirty() {
    fmt.Printf("%s %v - fields: %v\n",
        entry.Operation, entry.Entity, entry.Fields)
}
```

## Discarding changes

```go
cs.Reset() // clear all pending changes without submitting
```

## See also

- [Entity Change Tracking guide](../guides/change-tracking.md)
- [Batch Requests guide](../guides/batch.md)
- [Client Reference](client.md)
