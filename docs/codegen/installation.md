# Installing traverse-gen

## From source (recommended)

```bash
go install github.com/jhonsferg/traverse/cmd/traverse-gen@latest
```

Verify the installation:

```bash
traverse-gen --version
```

## Using `go run` (no install)

Run directly without installing a binary:

```bash
go run github.com/jhonsferg/traverse/cmd/traverse-gen@latest \
  --metadata ./api/metadata.xml \
  --out gen/
```

## Using `go generate`

Add a `go:generate` directive to your source:

```go
//go:generate go run github.com/jhonsferg/traverse/cmd/traverse-gen@latest --metadata ./metadata.xml --out ./gen --package myapi
```

Then run:

```bash
go generate ./...
```

## Requirements

| Tool | Version |
|------|---------|
| Go | 1.24+ |
| traverse | latest |

## See also

- [Usage](usage.md)
- [Code Generation Overview](index.md)
