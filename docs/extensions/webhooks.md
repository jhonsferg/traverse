# Webhooks Extension

The `ext/webhooks` package adds OData v4 webhook subscription support. It handles the full subscription lifecycle -- creating, renewing, and deleting subscriptions -- and provides an `http.Handler` that verifies incoming notifications and dispatches them to registered callbacks.

## Installation

```bash
go get github.com/jhonsferg/traverse/ext/webhooks@latest
```

## Subscribing

```go
import (
    "context"
    "github.com/jhonsferg/traverse/ext/webhooks"
)

sub, err := webhooks.Subscribe(ctx, client, webhooks.Config{
    EntitySet:          "Orders",
    CallbackURL:        "https://myapp.example.com/webhooks/orders",
    Expiry:             24 * time.Hour,
    Secret:             os.Getenv("WEBHOOK_SECRET"), // for HMAC verification
    RenewAutomatically: true,
})
if err != nil {
    log.Fatal(err)
}
```

`Subscribe` sends a `POST /$subscriptions` request to the OData service and returns a `*Subscription` on success.

## Registering handlers

```go
sub.OnCreated(func(ctx context.Context, n webhooks.Notification) {
    fmt.Printf("Order created: %s\n", n.EntityID)
})

sub.OnUpdated(func(ctx context.Context, n webhooks.Notification) {
    fmt.Printf("Order updated: %s -- data: %s\n", n.EntityID, n.Entity)
})

sub.OnDeleted(func(ctx context.Context, n webhooks.Notification) {
    fmt.Printf("Order deleted: %s\n", n.EntityID)
})
```

Multiple handlers can be registered for the same change type; they are called in registration order.

## Mounting the HTTP handler

```go
mux := http.NewServeMux()
mux.Handle("/webhooks/orders", sub.Handler())
http.ListenAndServe(":8080", mux)
```

The handler also responds to validation `GET` requests (with a `validationToken` query parameter) that some OData services send during subscription creation.

## Configuration reference

```go
type Config struct {
    // EntitySet is the OData entity set to subscribe to.
    EntitySet string

    // CallbackURL is the HTTPS URL the service will POST notifications to.
    CallbackURL string

    // Expiry controls how long the subscription lives (default: 24 h).
    Expiry time.Duration

    // Secret is used for HMAC-SHA256 signature verification.
    // Optional but strongly recommended.
    Secret string

    // ClientState is an opaque string echoed back in every notification.
    ClientState string

    // RenewAutomatically renews the subscription 5 minutes before expiry.
    RenewAutomatically bool
}
```

## Notification fields

```go
type Notification struct {
    SubscriptionID string
    ClientState    string
    EntitySet      string
    ChangeType     ChangeType   // "created", "updated", or "deleted"
    EntityID       string       // key value of the changed entity
    Entity         []byte       // raw JSON of the entity (nil for deletes)
    ReceivedAt     time.Time
}
```

## Signature verification

When `Secret` is set, the handler checks the `X-Signature` request header against an HMAC-SHA256 digest of the raw body. Requests without a valid signature receive a `401 Unauthorized` response.

```go
// The service must set X-Signature: <hex(HMAC-SHA256(secret, body))>
sub, _ := webhooks.Subscribe(ctx, client, webhooks.Config{
    EntitySet:   "Products",
    CallbackURL: "https://myapp.example.com/webhooks/products",
    Secret:      "my-shared-secret",
})
```

## Manual renewal and deletion

```go
// Renew for another 48 hours
if err := sub.Renew(ctx, 48*time.Hour); err != nil {
    log.Printf("renew failed: %v", err)
}

// Cancel the subscription
if err := sub.Delete(ctx); err != nil {
    log.Printf("delete failed: %v", err)
}
```

Passing `0` to `Renew` reuses the original `Expiry` from the `Config`.

## Automatic renewal

When `RenewAutomatically: true`, a background goroutine renews the subscription 5 minutes before expiry (or at 25 % of the expiry duration for very short subscriptions). The goroutine is stopped cleanly when `Delete` is called.

## Subscription ID

```go
fmt.Println(sub.SubscriptionID()) // e.g. "a1b2c3d4-..."
```

## Full example

```go
package main

import (
    "context"
    "log"
    "net/http"
    "os"
    "time"

    "github.com/jhonsferg/traverse"
    "github.com/jhonsferg/traverse/ext/webhooks"
)

func main() {
    client := traverse.New(traverse.Config{
        BaseURL: "https://api.example.com/odata/",
    })

    sub, err := webhooks.Subscribe(context.Background(), client, webhooks.Config{
        EntitySet:          "Orders",
        CallbackURL:        "https://myapp.example.com/webhooks/orders",
        Expiry:             12 * time.Hour,
        Secret:             os.Getenv("WEBHOOK_SECRET"),
        RenewAutomatically: true,
    })
    if err != nil {
        log.Fatal(err)
    }
    defer sub.Delete(context.Background()) //nolint:errcheck

    sub.OnCreated(func(ctx context.Context, n webhooks.Notification) {
        log.Printf("new order: %s", n.EntityID)
    })

    http.Handle("/webhooks/orders", sub.Handler())
    log.Fatal(http.ListenAndServe(":8080", nil))
}
```

## See also

- [Extensions Overview](index.md)
- [Delta Sync](../guides/delta-sync.md)
