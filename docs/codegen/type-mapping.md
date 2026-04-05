# OData to Go Type Mapping

`traverse-gen` maps every standard OData EDM type to an idiomatic Go type.

## Primitive types

| OData EDM Type | Go type (non-nullable) | Go type (nullable) |
|----------------|------------------------|---------------------|
| `Edm.Boolean` | `bool` | `*bool` |
| `Edm.Byte` | `uint8` | `*uint8` |
| `Edm.SByte` | `int8` | `*int8` |
| `Edm.Int16` | `int16` | `*int16` |
| `Edm.Int32` | `int32` | `*int32` |
| `Edm.Int64` | `int64` | `*int64` |
| `Edm.Single` | `float32` | `*float32` |
| `Edm.Double` | `float64` | `*float64` |
| `Edm.Decimal` | `string` | `*string` |
| `Edm.String` | `string` | `*string` |
| `Edm.Guid` | `string` | `*string` |
| `Edm.Binary` | `[]byte` | `[]byte` |
| `Edm.DateTime` | `time.Time` | `*time.Time` |
| `Edm.DateTimeOffset` | `time.Time` | `*time.Time` |
| `Edm.Date` | `time.Time` | `*time.Time` |
| `Edm.TimeOfDay` | `time.Duration` | `*time.Duration` |
| `Edm.Duration` | `time.Duration` | `*time.Duration` |
| `Edm.Stream` | `string` (URL) | `*string` |

## Enum types

OData enum types become Go `const` blocks with a typed string:

```xml
<EnumType Name="ProductStatus">
  <Member Name="Available" Value="0"/>
  <Member Name="Discontinued" Value="1"/>
</EnumType>
```

Generated:

```go
type ProductStatus string

const (
    ProductStatusAvailable    ProductStatus = "Available"
    ProductStatusDiscontinued ProductStatus = "Discontinued"
)
```

## Navigation properties

Single navigation properties become pointer fields; collections become slices:

```go
type Order struct {
    OrderID    int32    `json:"OrderID"`
    CustomerID string   `json:"CustomerID"`

    // single navigation property
    Customer *Customer `json:"Customer,omitempty"`

    // collection navigation property
    OrderItems []OrderItem `json:"OrderItems,omitempty"`
}
```

## Nullable fields

When `--nullable=true` (the default), every property marked `Nullable="true"` in the EDMX gets a pointer type. Disable to use value types everywhere:

```bash
traverse-gen --metadata metadata.xml --nullable=false --out gen/
```

## Custom timestamp type

Replace `time.Time` with any compatible type via `--timestamps`:

```bash
traverse-gen --metadata metadata.xml --timestamps "pgtype.Timestamptz" --out gen/
```

The generated file will include the correct import path if the type is provided as a fully qualified name.

## See also

- [Usage](usage.md)
- [Generated Client](generated-client.md)
