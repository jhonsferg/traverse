# Traverse: JSON to XML Migration Guide

## Overview

Traverse now provides explicit XML support alongside JSON. This guide helps you update existing code to use the new format-explicit methods.

## Why Change?

Old method names like `CreateAs[T]()` and `CollectAs[T]()` were ambiguous about response format. They assumed JSON but failed silently if a backend returned XML. New explicit names make the format intent clear:

- `CreateJsonAs[T]()` - Expect JSON response
- `CreateXmlAs[T]()` - Expect XML response

## Backward Compatibility

All old method names still work as aliases to the JSON versions. No breaking changes. However, deprecation notices guide you toward explicit names.

## Migration Path

### Step 1: Add XML struct tags

If your type has only `json:` tags, add `xml:` tags for XML support:

```go
type Product struct {
    ID    int    `json:"ProductID" xml:"ProductID"`
    Name  string `json:"ProductName" xml:"ProductName"`
    Price float64 `json:"Price" xml:"Price"`
}
```

### Step 2: Update method calls

**Before (ambiguous):**
```go
results, err := traverse.CollectAs[Product](qb, ctx)
first, err := traverse.FirstAs[Product](qb, ctx)
err = traverse.CreateAs[Product](qb, ctx, newProduct)
```

**After (explicit):**
```go
results, err := traverse.CollectJsonAs[Product](qb, ctx)
first, err := traverse.FirstJsonAs[Product](qb, ctx)
err = traverse.CreateJsonAs[Product](qb, ctx, newProduct)
```

Or, if handling XML:

```go
results, err := traverse.CollectXmlAs[Product](qb, ctx)
first, err := traverse.FirstXmlAs[Product](qb, ctx)
err = traverse.CreateXmlAs[Product](qb, ctx, newProduct)
```

### Step 3: Handle both formats gracefully

For backends that return unpredictable formats:

```go
var results []Product
var err error

results, err = traverse.CollectJsonAs[Product](qb, ctx)
if err != nil {
    results, err = traverse.CollectXmlAs[Product](qb, ctx)
}
if err != nil {
    log.Fatalf("query failed: %v", err)
}
```

## Affected Methods

The following methods now have explicit JSON/XML variants:

| Category | Methods |
|----------|---------|
| CRUD Entity Operations | `CreateJsonAs/XmlAs`, `FindByKeyJsonAs/XmlAs`, `FirstJsonAs/XmlAs` |
| Collection Queries | `CollectJsonAs/XmlAs`, `StreamJsonAs/XmlAs` |
| OData Functions | `ExecuteFunctionJsonAs/XmlAs` |
| OData Actions | `ExecuteActionJsonAs/XmlAs` |
| Function Imports | `ExecuteFunctionImportJsonAs/XmlAs` |
| Delta Sync | `DeltaSyncJsonAs/XmlAs` (types) |

## Deprecated Methods (Still Work)

These methods are aliased to JSON versions for backward compatibility:

- `CreateAs[T]()`
- `CollectAs[T]()`
- `StreamAs[T]()`
- `FirstAs[T]()`
- `FindByKeyAs[T]()`
- `ExecuteFunctionAs[T]()`
- `ExecuteActionAs[T]()`
- `ExecuteFunctionImportAs[T]()`
- `DeltaSyncAs[T]` (type)

Usage:
```go
results, _ := traverse.CollectAs[Product](qb, ctx)
```

Equivalent to:
```go
results, _ := traverse.CollectJsonAs[Product](qb, ctx)
```

## SAP-Specific Notes

SAP OData backends often:
1. Accept any `Accept:` header but return XML regardless
2. Use namespace-qualified XML element names
3. Return `406 Not Acceptable` if JSON is strictly required

For maximum compatibility with SAP:

```go
type SAPMaterial struct {
    MatID  string `json:"MATERIAL_ID" xml:"MATERIAL_ID"`
    Name   string `json:"MATERIAL_NAME" xml:"MATERIAL_NAME"`
}

qb := client.From("Materials")

mats, err := traverse.CollectXmlAs[SAPMaterial](qb, ctx)
if err != nil {
    log.Printf("XML failed, trying JSON: %v", err)
    mats, err = traverse.CollectJsonAs[SAPMaterial](qb, ctx)
}
```

## Testing

Ensure your types have both `json:` and `xml:` tags for complete test coverage:

```go
type TestEntity struct {
    ID int `json:"id" xml:"id"`
    Name string `json:"name" xml:"name"`
}

func TestMyEndpoint(t *testing.T) {
    t.Run("JSON", func(t *testing.T) {
        results, err := traverse.CollectJsonAs[TestEntity](qb, ctx)
        if err != nil {
            t.Fatalf("JSON failed: %v", err)
        }
        if len(results) == 0 {
            t.Error("expected results")
        }
    })

    t.Run("XML", func(t *testing.T) {
        results, err := traverse.CollectXmlAs[TestEntity](qb, ctx)
        if err != nil {
            t.Fatalf("XML failed: %v", err)
        }
        if len(results) == 0 {
            t.Error("expected results")
        }
    })
}
```

## Troubleshooting

### Error: "cannot unmarshal into struct"

Usually means struct tags don't match response format:

```go
type Product struct {
    ID int `xml:"id"` // Wrong if response is JSON with "id"
}

results, _ := traverse.CollectJsonAs[Product](qb, ctx)
```

Fix: Add matching tags for both formats:

```go
type Product struct {
    ID int `json:"id" xml:"id"`
}
```

### Error: "406 Not Acceptable"

SAP backend rejected your `Accept:` header. Use XML variant:

```go
results, err := traverse.CollectJsonAs[Product](qb, ctx)
if err != nil && strings.Contains(err.Error(), "406") {
    results, err = traverse.CollectXmlAs[Product](qb, ctx)
}
```

## Performance Considerations

- JSON and XML marshaling have similar performance
- XML is slightly more verbose (larger payloads)
- Use `StreamJsonAs/XmlAs` for large result sets regardless of format
- Delta sync with `DeltaSyncJsonAs/XmlAs` is format-specific but equally efficient

## Next Steps

1. Run your test suite - old code still works
2. Update critical paths to use explicit format methods
3. Add XML tags to struct definitions
4. Test with actual backend responses
5. Deploy incrementally

For questions or issues, see the [traverse documentation](https://jhonsferg.github.io/traverse) or the [GitHub repository](https://github.com/jhonsferg/traverse).
