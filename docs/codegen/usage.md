# traverse-gen Usage

## Synopsis

```
traverse-gen [flags]
```

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--metadata` | required | Path or URL to EDMX metadata |
| `--out` | `./gen` | Output directory |
| `--package` | `gen` | Go package name for generated code |
| `--service` | (all) | Generate only for a specific EntityContainer |
| `--include` | (all) | Comma-separated entity set names to include |
| `--exclude` | | Comma-separated entity set names to exclude |
| `--prefix` | | Prefix added to all generated type names |
| `--nullable` | `true` | Use pointer types for nullable OData properties |
| `--timestamps` | `time.Time` | Go type to use for Edm.DateTimeOffset |
| `--v2` | auto | Force OData v2 mode |
| `--v4` | auto | Force OData v4 mode |

## Examples

### From a local EDMX file

```bash
traverse-gen --metadata api/metadata.xml --out gen/ --package myapi
```

### From a live service endpoint

```bash
traverse-gen \
  --metadata https://services.odata.org/V4/Northwind/Northwind.svc/$metadata \
  --out gen/ \
  --package northwind
```

### Generating only selected entity sets

```bash
traverse-gen \
  --metadata metadata.xml \
  --include Products,Categories,Orders \
  --out gen/
```

### Custom timestamp type

```bash
traverse-gen \
  --metadata metadata.xml \
  --timestamps "pgtype.Timestamptz" \
  --out gen/
```

### Adding a type prefix

```bash
traverse-gen --metadata metadata.xml --prefix SAP --out gen/
# generates: SAPProduct, SAPOrder, etc.
```

## Output files

| File | Contents |
|------|----------|
| `entities.go` | Entity structs and enum types |
| `client.go` | Service client and per-entity-set methods |
| `queries.go` | Typed query builders and filter helpers |

## Regenerating

Re-run `traverse-gen` with the same flags whenever the metadata changes. Generated files are safe to overwrite - do not edit them by hand.

## See also

- [Type Mapping](type-mapping.md)
- [Generated Client](generated-client.md)
