# OData Vocabularies

traverse supports OData Core and Validation vocabularies. Vocabulary annotations are parsed from `$metadata` and surfaced on `Property.Core` and `Property.Validation` fields.

## Quick Start

```go
import "github.com/jhonsferg/traverse"

meta, _ := traverse.ParseMetadataFromURL(ctx, "https://api.example.com/odata/$metadata")

for _, et := range meta.EntityTypes {
    for _, prop := range et.Properties {
        core := traverse.ParseCoreVocabulary(prop.Annotations)
        val  := traverse.ParseValidationVocabulary(prop.Annotations)

        if core.Computed {
            fmt.Printf("%s.%s is computed (read-only)\n", et.Name, prop.Name)
        }
        if val.Required {
            fmt.Printf("%s.%s is required\n", et.Name, prop.Name)
        }
        if val.Pattern != "" {
            fmt.Printf("%s.%s must match /%s/\n", et.Name, prop.Name, val.Pattern)
        }
    }
}
```

## API Reference

### `ParseCoreVocabulary`

```go
func ParseCoreVocabulary(annotations map[string]string) CoreVocabulary
```

Extracts `Org.OData.Core.V1.*` terms from a raw annotation map.

### `CoreVocabulary`

```go
type CoreVocabulary struct {
    Description         string
    LongDescription     string
    IsLanguageDependent bool
    Immutable           bool
    Computed            bool
    // ... additional Core V1 terms
}
```

| Field | OData term | Meaning |
|-------|-----------|---------|
| `Description` | `Org.OData.Core.V1.Description` | Short human-readable description |
| `LongDescription` | `Org.OData.Core.V1.LongDescription` | Detailed description |
| `IsLanguageDependent` | `Org.OData.Core.V1.IsLanguageDependent` | Value varies by language |
| `Immutable` | `Org.OData.Core.V1.Immutable` | Value cannot be changed after creation |
| `Computed` | `Org.OData.Core.V1.Computed` | Value is server-computed; omit from writes |

### `ParseValidationVocabulary`

```go
func ParseValidationVocabulary(annotations map[string]string) ValidationVocabulary
```

Extracts `Org.OData.Validation.V1.*` terms from a raw annotation map.

### `ValidationVocabulary`

```go
type ValidationVocabulary struct {
    Minimum       *float64
    Maximum       *float64
    Pattern       string
    AllowedValues []string
    Required      bool
}
```

| Field | OData term | Meaning |
|-------|-----------|---------|
| `Minimum` | `Org.OData.Validation.V1.Minimum` | Minimum numeric value (inclusive) |
| `Maximum` | `Org.OData.Validation.V1.Maximum` | Maximum numeric value (inclusive) |
| `Pattern` | `Org.OData.Validation.V1.Pattern` | Regex the value must match |
| `AllowedValues` | `Org.OData.Validation.V1.AllowedValues` | Enumeration of valid values |
| `Required` | `Org.OData.Validation.V1.Required` | Property must be supplied on create |

## Vocabulary annotations in EDMX

Vocabulary annotations appear as `Annotation` elements inside `EntityType/Property` in the EDMX document:

```xml
<Property Name="Email" Type="Edm.String">
  <Annotation Term="Org.OData.Core.V1.Description" String="Primary email address"/>
  <Annotation Term="Org.OData.Validation.V1.Pattern" String="^[^@]+@[^@]+$"/>
  <Annotation Term="Org.OData.Validation.V1.Required" Bool="true"/>
</Property>
```

These are stored in `Property.Annotations` as `map[string]string` and can be passed to either parse function.

## Notes / Limitations

- Only `Org.OData.Core.V1` and `Org.OData.Validation.V1` namespaces are parsed; custom vocabulary terms are available raw in `Property.Annotations`.
- Boolean annotations are stored as the string `"true"` or `"false"` in the raw map; the parse functions convert them to Go `bool`.
- `Minimum` and `Maximum` are `*float64`; a `nil` pointer means the annotation is absent.
