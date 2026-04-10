# Traverse OData — VS Code Extension

Syntax highlighting, autocompletion, hover documentation, and validation for OData queries in [traverse](https://github.com/jhonsferg/traverse) Go projects.

## Features

### Syntax Highlighting
- Dedicated `.odata` file type with full OData grammar
- Inline highlighting inside `.Filter(...)`, `.Select(...)`, `.Expand(...)` calls in Go source files
- Highlights system query options (`$filter`, `$select`, …), operators (`eq`, `ne`, `and`, …), functions (`contains`, `year`, `geo.distance`, …), string literals, GUIDs, and numeric constants

### Autocompletion
Triggered inside OData string arguments of traverse builder calls:

| Category | Examples |
|---|---|
| System query options | `$filter`, `$select`, `$expand`, `$orderby`, `$top`, `$count`, … |
| Comparison operators | `eq`, `ne`, `lt`, `le`, `gt`, `ge`, `has`, `in` |
| Logical operators | `and`, `or`, `not` |
| Arithmetic operators | `add`, `sub`, `mul`, `divby`, `mod` |
| String functions | `contains`, `startswith`, `endswith`, `tolower`, `toupper`, `trim`, … |
| Math functions | `round`, `floor`, `ceiling` |
| Date/time functions | `year`, `month`, `day`, `now`, `date`, `time`, … |
| Type functions | `isof`, `cast` |
| Lambda operators | `any`, `all` |
| Geospatial functions | `geo.distance`, `geo.intersects`, `geo.length` |
| EDM types | `Edm.String`, `Edm.Int32`, `Edm.Decimal`, `Edm.GeographyPoint`, … |

### Hover Documentation
Hover over any OData function or operator inside a Go string or `.odata` file to see its signature and usage example.

### Diagnostics
- Detects unmatched parentheses inside `.Filter(...)` and `.Where(...)` arguments
- Detects unterminated string literals (`'unclosed`)
- Warnings are shown inline as you type

### Snippets
Over 15 ready-to-use snippets for common traverse patterns:

| Prefix | Description |
|---|---|
| `trav.from` | Full QueryBuilder chain |
| `trav.stream` | Streaming loop with error handling |
| `trav.findbykey` | FindByKey with error handling |
| `trav.create` | Create entity |
| `trav.update` | Update entity (PATCH) |
| `trav.delete` | Delete entity |
| `trav.batch` | Batch request skeleton |
| `trav.singleton` | Singleton access |
| `odata.contains` | `contains()` filter |
| `odata.any` | Lambda `any()` filter |
| `odata.in` | `in` operator filter |

## Requirements

- Visual Studio Code 1.85+
- [traverse](https://github.com/jhonsferg/traverse) Go library

## Extension Settings

| Setting | Default | Description |
|---|---|---|
| `traverse.odata.validateOnType` | `true` | Validate OData strings as you type |
| `traverse.odata.completionMode` | `full` | `full`, `keywords-only`, or `off` |
| `traverse.odata.hoverDocs` | `true` | Show hover documentation |

## Usage

### `.odata` files
Create a file with the `.odata` extension for standalone OData queries. Full syntax highlighting and completion apply automatically.

```
$filter=contains(Name, 'SAP') and Price gt 100
$select=ID,Name,Price
$orderby=Name asc
$top=50
```

### Go files
The extension activates automatically for Go files. Completions and hover docs are triggered inside string arguments of these traverse methods:

```go
client.From("Products").
    Filter("contains(Name, 'SAP') and Price gt 100").
    Select("ID,Name,Price").
    OrderBy("Name asc").
    Top(50).
    Collect(ctx)
```

## Building from source

```bash
cd traverse/cmd/vscode-ext
npm install
npm run compile
npm run package   # produces traverse-odata-0.1.0.vsix
```

Install the `.vsix` file via **Extensions → Install from VSIX…** in VS Code.

## License

MIT — same as the traverse library.
