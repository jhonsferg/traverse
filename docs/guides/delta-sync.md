# Delta Sync

Delta sync lets you fetch only the records that changed since your last synchronization. Instead of re-fetching the entire dataset on every run, you receive a delta token after each sync and use it next time to get only new, modified, and deleted records.

## How Delta Works

1. First run: request the entity set normally with `$deltatoken=latest` or no token
2. Server responds with all current data plus a `@odata.deltaLink`
3. Store the delta link (or token) between runs
4. Next run: request the delta link - server returns only changes since the token

```
First run:
GET /Orders?$deltatoken=latest
< 200 OK
< {"value": [...all orders...], "@odata.deltaLink": "/Orders?$deltatoken=abc123"}

Second run:
GET /Orders?$deltatoken=abc123
< 200 OK
< {"value": [...only changed orders...], "@odata.deltaLink": "/Orders?$deltatoken=def456"}
```

## Delta API

```go
// First run: pass empty string for initial full load
stream, deltaToken, err := client.From("Orders").Delta(ctx, "")
if err != nil {
    log.Fatal(err)
}
defer stream.Close()

for item := range stream.Items() {
    change := item.(traverse.DeltaItem[Order])
    switch change.Type {
    case traverse.DeltaAdded, traverse.DeltaModified:
        upsert(change.Entity)
    case traverse.DeltaDeleted:
        delete(change.ID)
    }
}

if err := stream.Err(); err != nil {
    log.Fatal(err)
}

// Save deltaToken for next run
saveDeltaToken("Orders", deltaToken)
```

Next run:

```go
token := loadDeltaToken("Orders")
stream, newToken, err := client.From("Orders").Delta(ctx, token)
// process only changes...
saveDeltaToken("Orders", newToken)
```

## DeltaItem

```go
type DeltaItem[T any] struct {
    Type   DeltaType
    Entity T      // populated for added/modified items
    ID     any    // populated for deleted items
}

type DeltaType int

const (
    DeltaAdded    DeltaType = iota // new or modified entity
    DeltaModified                  // explicitly marked as modified
    DeltaDeleted                   // entity was deleted
)
```

Deleted items carry the entity key in `ID` and have `@odata.removed` annotation in the raw JSON.

## Persistent Token Store Example

```go
package main

import (
    "context"
    "database/sql"
    "fmt"
    "log"

    "github.com/jhonsferg/traverse"
    _ "github.com/mattn/go-sqlite3"
)

type Order struct {
    ID         int    `json:"OrderID"`
    CustomerID string `json:"CustomerID"`
    Status     string `json:"Status"`
}

type TokenStore struct {
    db *sql.DB
}

func (s *TokenStore) Load(entitySet string) string {
    var token string
    s.db.QueryRow("SELECT token FROM delta_tokens WHERE entity_set = ?", entitySet).Scan(&token)
    return token // returns "" if not found
}

func (s *TokenStore) Save(entitySet, token string) {
    s.db.Exec(
        "INSERT OR REPLACE INTO delta_tokens (entity_set, token) VALUES (?, ?)",
        entitySet, token,
    )
}

func syncOrders(ctx context.Context, client *traverse.Client, store *TokenStore) error {
    token := store.Load("Orders")

    stream, newToken, err := client.From("Orders").
        Select("OrderID", "CustomerID", "Status").
        Delta(ctx, token)
    if err != nil {
        return fmt.Errorf("delta request: %w", err)
    }
    defer stream.Close()

    added, modified, deleted := 0, 0, 0

    for item := range stream.Items() {
        change := item.(traverse.DeltaItem[Order])
        switch change.Type {
        case traverse.DeltaAdded:
            upsertOrder(change.Entity)
            added++
        case traverse.DeltaModified:
            upsertOrder(change.Entity)
            modified++
        case traverse.DeltaDeleted:
            deleteOrder(change.ID.(int))
            deleted++
        }
    }

    if err := stream.Err(); err != nil {
        return fmt.Errorf("stream: %w", err)
    }

    store.Save("Orders", newToken)
    log.Printf("Delta sync: +%d ~%d -%d\n", added, modified, deleted)
    return nil
}

func upsertOrder(o Order)  { /* write to your DB */ }
func deleteOrder(id int)   { /* delete from your DB */ }
```

## $deltatoken vs $skiptoken

These are different mechanisms:

| | $deltatoken | $skiptoken |
|--|-------------|-----------|
| Purpose | Changes since last sync | Next page of results |
| Persisted between runs | Yes | No |
| Content | Added/modified/deleted | Full entities |
| Server support | OData v4 (optional) | Both v2 and v4 |

!!! note "Service support"
    Delta links require server-side support. Check the service `$metadata` for `DeltaLink` capability. SAP S/4HANA OData v4 services support delta links; SAP ABAP Gateway OData v2 services generally do not.

!!! tip "Initial sync"
    On the very first run (empty token), the server returns the complete current dataset. This can be large - consider using [Streaming](streaming.md) for the initial load.

## Related Pages

- [Streaming](streaming.md) - Process large initial loads
- [Typed Pagination](pagination.md) - Page through results without delta
- [Query Builder](query-builder.md) - Filter the initial delta request
