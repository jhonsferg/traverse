package traverse

import (
	"context"
	"encoding/json"
	"fmt"
)

// Singleton returns a [QueryBuilder] targeting a named singleton resource at the
// service root.
//
// A singleton is a single-entity resource addressable directly by name, without
// a key predicate. It is defined in OData v4.0 spec section 11.2.4.
//
// Microsoft Graph uses singletons extensively (e.g., "/me", "/organization").
// SAP uses them for session-level resources.
//
// The returned [QueryBuilder] supports the full query API including $select,
// $expand, $filter, and navigation to related collections via [From].
//
// Example:
//
//	// Fetch the "me" singleton
//	me, err := client.Singleton("me").Page(ctx)
//
//	// Expand a navigation property from a singleton
//	result, err := client.Singleton("me").Expand("manager").Page(ctx)
//
//	// Navigate to a related collection on a singleton
//	for r := range client.Singleton("me").From("messages").Stream(ctx) {
//	    // process r.Value
//	}
func (c *Client) Singleton(name string) *QueryBuilder {
	return &QueryBuilder{
		client:    c,
		entitySet: name,
		urlDirty:  true,
	}
}

// SingletonAs fetches a singleton resource and decodes it into the provided type T.
//
// SingletonAs is a convenience wrapper around [Client.Singleton] that handles
// the common pattern of fetching a singleton and decoding it into a typed struct.
//
// Example:
//
//	type Me struct {
//	    ID          string `json:"id"`
//	    DisplayName string `json:"displayName"`
//	    Mail        string `json:"mail"`
//	}
//
//	me, err := traverse.SingletonAs[Me](client, "me")
func SingletonAs[T any](c *Client, name string) (*T, error) {
	ctx := context.Background()
	return SingletonAsCtx[T](c, ctx, name)
}

// SingletonAsCtx fetches a singleton resource and decodes it into the provided type T,
// using the provided context for cancellation and deadline control.
//
// Example:
//
//	me, err := traverse.SingletonAsCtx[Me](client, ctx, "me")
func SingletonAsCtx[T any](c *Client, ctx context.Context, name string) (*T, error) {
	req := c.http.Get("/" + name).WithContext(ctx)

	resp, err := c.http.Execute(req)
	if err != nil {
		return nil, fmt.Errorf("traverse: SingletonAs[%T] failed: %w", *new(T), err)
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("traverse: SingletonAs[%T]: server returned status %d", *new(T), resp.StatusCode)
	}

	body := resp.Body()
	var dest T

	if c.version == ODataV2 {
		// OData v2: response may be wrapped in {"d": {...}}
		var wrapped struct {
			D json.RawMessage `json:"d"`
		}
		if jsonErr := json.Unmarshal(body, &wrapped); jsonErr == nil && wrapped.D != nil {
			body = wrapped.D
		}
	}

	if jsonErr := json.Unmarshal(body, &dest); jsonErr != nil {
		return nil, fmt.Errorf("traverse: SingletonAs[%T]: decode failed: %w", *new(T), jsonErr)
	}
	return &dest, nil
}
