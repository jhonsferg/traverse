# OData Primer

OData (Open Data Protocol) is a REST-based protocol for building and consuming data APIs. This page covers the concepts you need to use traverse effectively - it is not a full OData specification.

## Entity Sets

An **entity set** is a named collection of entities, analogous to a database table. In OData URLs, entity sets appear as path segments:

```
GET /Products               -- all products
GET /Products(1)            -- product with key 1
GET /Orders(10248)          -- order with key 10248
GET /OrderDetails(OrderID=10248,ProductID=11)  -- composite key
```

In traverse:

```go
client.From("Products")           // entity set
client.From("Products").Key(1)    // single entity
client.From("OrderDetails").Key(map[string]any{
    "OrderID": 10248, "ProductID": 11,
})
```

## $filter Syntax

`$filter` selects entities matching a boolean expression.

### Comparison operators

| Operator | Meaning | Example |
|----------|---------|---------|
| `eq` | equal | `Status eq 'Active'` |
| `ne` | not equal | `Status ne 'Deleted'` |
| `lt` | less than | `Price lt 100` |
| `le` | less than or equal | `Price le 100` |
| `gt` | greater than | `Price gt 10` |
| `ge` | greater than or equal | `Price ge 10` |

### Logical operators

```
$filter=Price gt 10 and Price lt 100
$filter=Status eq 'Active' or Status eq 'Pending'
$filter=not (Status eq 'Deleted')
```

### String functions

```
$filter=startswith(Name,'A')
$filter=endswith(Name,'Ltd')
$filter=contains(Name,'acme')
$filter=tolower(Name) eq 'acme corp'
```

### Arithmetic

```
$filter=(Price mul Quantity) gt 1000
```

### Null checks

```
$filter=Description ne null
$filter=ShippedDate eq null
```

## $select

Limit which fields are returned (reduces payload size):

```
GET /Products?$select=ProductID,ProductName,UnitPrice
```

## $expand

Include related entities inline (avoids N+1 round trips):

```
GET /Orders?$expand=OrderDetails
GET /Orders?$expand=OrderDetails($select=ProductID,Quantity)
GET /Orders?$expand=Customer,OrderDetails($expand=Product)
```

In OData v4, `$expand` supports nested query options. In OData v2, expand is a flat comma-separated list.

## $orderby

```
$orderby=Price desc
$orderby=LastName asc,FirstName asc
```

## $top and $skip

Server-side pagination:

```
$top=20&$skip=40    -- page 3 of 20-item pages
```

!!! note "Prefer Paginator over manual $skip"
    Manual `$skip` can miss or duplicate records if the dataset changes between requests. Use traverse's [Paginator](guides/pagination.md) or delta sync instead.

## $count

```
GET /Products?$count=true      -- include total count in response
GET /Products/$count           -- return only the count
```

## $search

Full-text search (service-dependent support):

```
$search=coffee
$search="dark roast"
```

## ETags

An **ETag** (entity tag) is an opaque string the server returns in the `ETag` response header representing the current version of an entity:

```
ETag: W/"datetime'2024-01-15T10%3A30%3A00'"
```

- `W/` prefix indicates a **weak** ETag (content-equivalent, not byte-identical)
- Strong ETags have no prefix

To update or delete safely, send the last-known ETag in the `If-Match` header:

```
PATCH /Products(1) HTTP/1.1
If-Match: W/"datetime'2024-01-15T10%3A30%3A00'"
```

If the entity was modified since you fetched it, the server returns `412 Precondition Failed`.

See [ETag & Concurrency](guides/etag-concurrency.md) for traverse's ETag API.

## Batch Requests

OData `$batch` lets you bundle multiple operations into a single HTTP request, reducing round trips:

```
POST /odata/$batch
Content-Type: multipart/mixed; boundary=batch_1
```

A **changeset** within a batch is atomic: all operations succeed or all are rolled back.

See [Batch Requests](guides/batch.md) for traverse's batch API.

## Delta Links

After requesting `?$deltatoken=...`, the server returns only entities that changed since that token was issued:

- Modified entities appear as full entity objects
- Deleted entities appear with `@odata.removed` annotation

Delta tokens let you sync incrementally without fetching the full dataset every time.

See [Delta Sync](guides/delta-sync.md) for traverse's delta API.

## Actions and Functions

**Functions** are read-only operations that return data:

```
GET /Products/MostExpensive()
GET /Products(1)/DefaultSupplier
```

**Actions** are operations with side effects:

```
POST /Orders(1)/Ship
POST /Customers(1)/SendReminder
```

Both can have parameters passed in the URL (functions) or request body (actions).

## OData v2 vs v4 Differences

| Feature | OData v2 | OData v4 |
|---------|----------|----------|
| Date format | `/Date(milliseconds)/` | ISO 8601 |
| Count | `$inlinecount=allpages` | `$count=true` |
| Skip token | `$skiptoken` | `$skiptoken` or `@odata.nextLink` |
| Batch content type | `multipart/mixed` | `multipart/mixed` or `application/json` |
| Deep insert | `__deferred` links | inline navigation properties |
| Delta | limited | full delta link support |

traverse detects the OData version automatically from the service metadata and response headers.
