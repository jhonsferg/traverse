# Query Builder

The `QueryBuilder` provides a fluent API for constructing OData queries. Every method returns a new builder (immutable), so you can compose queries safely across goroutines or branch from a base query.

```go
base := client.From("Products").
    Select("ProductID", "ProductName", "UnitPrice").
    OrderBy("UnitPrice desc")

// cheap items
cheap, _ := base.Filter("UnitPrice lt 10").Top(20).Build()

// expensive items - base is unchanged
expensive, _ := base.Filter("UnitPrice gt 100").Top(20).Build()
```

## Method Reference

### Filter

```go
Filter(expression string) *QueryBuilder
```

Appends a `$filter` expression. Multiple calls are ANDed together.

```go
client.From("Orders").
    Filter("Freight gt 100").
    Filter("ShipCountry eq 'Germany'")
// $filter=Freight gt 100 and ShipCountry eq 'Germany'
```

### Select

```go
Select(fields ...string) *QueryBuilder
```

Limits returned fields with `$select`.

```go
client.From("Products").Select("ProductID", "ProductName", "UnitPrice")
// $select=ProductID,ProductName,UnitPrice
```

### Expand

```go
Expand(navProps ...string) *QueryBuilder
```

Includes related entities via `$expand`.

```go
client.From("Orders").Expand("Customer", "OrderDetails")
// $expand=Customer,OrderDetails
```

For nested expand with options (OData v4):

```go
client.From("Orders").Expand("OrderDetails($select=ProductID,Quantity;$expand=Product)")
// $expand=OrderDetails($select=ProductID,Quantity;$expand=Product)
```

See [ExpandBuilder](../reference/query-builder.md) for the typed nested expand API.

### OrderBy

```go
OrderBy(fields ...string) *QueryBuilder
```

Sets `$orderby`. Each field can have an optional `asc` or `desc` suffix.

```go
client.From("Products").OrderBy("CategoryID asc", "UnitPrice desc")
// $orderby=CategoryID asc,UnitPrice desc
```

### Top

```go
Top(n int) *QueryBuilder
```

Limits results with `$top`.

```go
client.From("Products").Top(50)
// $top=50
```

### Skip

```go
Skip(n int) *QueryBuilder
```

Skips the first N results with `$skip`.

```go
client.From("Products").Skip(100).Top(20)
// $skip=100&$top=20
```

!!! warning "Avoid $skip for large offsets"
    Many OData services have poor `$skip` performance for large values. Prefer the [Paginator](pagination.md) which follows server-provided nextLink tokens.

### Count

```go
Count() *QueryBuilder
```

Requests the total count with `$count=true` (OData v4) or `$inlinecount=allpages` (OData v2).

```go
client.From("Products").Filter("UnitPrice gt 50").Count()
// $filter=UnitPrice gt 50&$count=true
```

### Search

```go
Search(term string) *QueryBuilder
```

Adds `$search` for full-text search. Support is service-dependent.

```go
client.From("Products").Search("chai")
// $search=chai
```

### Apply

```go
Apply(expression string) *QueryBuilder
```

Adds `$apply` for server-side aggregation (OData v4 aggregation extension).

```go
client.From("Sales").
    Apply("groupby((Region),aggregate(Amount with sum as TotalSales))")
// $apply=groupby((Region),aggregate(Amount with sum as TotalSales))
```

### Format

```go
Format(f string) *QueryBuilder
```

Sets the `$format` query option (e.g., `json`, `atom`).

```go
client.From("Products").Format("json")
// $format=json
```

### Key

```go
Key(k any) *QueryBuilder
```

Appends a key segment to the entity set path, targeting a single entity.

```go
client.From("Products").Key(1)           // Products(1)
client.From("Products").Key("ALFKI")     // Products('ALFKI')
client.From("OrderDetails").Key(map[string]any{
    "OrderID": 10248, "ProductID": 11,
}) // OrderDetails(OrderID=10248,ProductID=11)
```

## Executing Queries

### Into

```go
Into(ctx context.Context, dst any) error
```

Executes the query and decodes the OData value into `dst`. `dst` should be a pointer to a slice for collections, or a pointer to a struct for single entities.

```go
var products []Product
err := client.From("Products").Filter("UnitPrice gt 10").Into(ctx, &products)

var single Product
err := client.From("Products").Key(1).Into(ctx, &single)
```

### Build

```go
Build() (string, error)
```

Returns the constructed OData URL without executing it - useful for debugging or logging.

```go
url, err := client.From("Products").
    Filter("UnitPrice gt 20").
    Select("ProductID", "ProductName").
    Top(10).
    Build()
fmt.Println(url)
// /Products?$filter=UnitPrice gt 20&$select=ProductID,ProductName&$top=10
```

## $filter Expression Cheat Sheet

```
-- comparison
Name eq 'Chai'
Price ne 0
Price gt 10
Price ge 10
Price lt 100
Price le 100

-- logical
Price gt 10 and Price lt 100
Status eq 'A' or Status eq 'B'
not (Status eq 'Deleted')

-- arithmetic
(Price mul Quantity) gt 1000

-- string functions
startswith(Name, 'A')
endswith(Name, 'Ltd')
contains(Name, 'coffee')
indexof(Name, 'tea') ge 0
length(Name) gt 5
tolower(Name) eq 'acme'
toupper(Code) eq 'ABC'
trim(Name) eq 'Chai'
concat(FirstName, ' ', LastName) eq 'John Smith'
substring(Name, 0, 3) eq 'Pro'

-- date/time
year(OrderDate) eq 2024
month(OrderDate) eq 6
day(OrderDate) gt 15
hour(CreatedAt) lt 12

-- null / type checks
Description eq null
Description ne null
isof(Product, 'NS.SpecialProduct')

-- collection (OData v4)
Products/any(p: p/Price gt 100)
Tags/all(t: t ne 'archived')

-- geo (if supported)
geo.distance(Location, geography'POINT(-122.1 37.4)') lt 10
```

## Related Pages

- [Lambda Filters](lambda-filters.md) - Type-safe `any()`/`all()` expressions
- [Query Builder API Reference](../reference/query-builder.md) - Complete method signatures
- [OData Primer](../odata-primer.md) - OData filter syntax background
