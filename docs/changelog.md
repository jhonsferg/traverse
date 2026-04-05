# Changelog

All notable changes to traverse are documented in this file.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).
traverse uses [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [0.4.0] - Phase C: Production Readiness

### Added

- **Lambda filter DSL** (`lambda_filter.go`) - type-safe filter builder with `query.Field().Gt()`, `query.And()`, `query.Any()`, `query.All()` and full OData function support
- **Deep insert** (`deep_insert.go`) - create an entity and its related entities in a single POST, following the OData deep insert specification
- **Conditional headers** (`conditional_headers.go`) - typed `IfMatch`, `IfNoneMatch`, `IfModifiedSince`, `IfUnmodifiedSince` request modifiers with `ETag` type
- **Stream property support** (`stream_property.go`) - read, write, and delete OData stream properties; multipart chunked upload with progress callback

---

## [0.3.0] - Phase B2: Code Generation

### Added

- **traverse-gen CLI** (`cmd/traverse-gen/`) - reads EDMX metadata and generates typed Go client, entity structs, query builders, and enum types
- EDMX parser supporting OData v2 and v4 metadata documents
- PascalCase normalisation for entity and property names
- `--include` / `--exclude` flags for selective generation
- `--nullable` flag for pointer vs value types on nullable properties

---

## [0.2.0] - Phase B: Advanced OData

### Added

- **Entity change tracking** (`change_tracking.go`) - `ChangeSet[T]` records mutations to tracked entities and submits them as an atomic `$batch` request
- **Deep expand** (`deep_expand.go`) - nested `$expand` with per-level `$select`, `$filter`, `$orderby`, `$top` options (OData v4)
- **Async operation poller** (`async_op.go`) - `AsyncPoller[T]` polls `202 Accepted` / `Location` responses with configurable interval, progress callbacks, and timeout

---

## [0.1.0] - Phase A: Core OData Features

### Added

- **ETag upsert** (`etag_upsert.go`) - upsert entity with `If-Match: *` or `If-None-Match: *` semantics for atomic create-or-replace
- **Typed paginator** (`paginator.go`) - `Paginator[T]` supports `@odata.nextLink`, `$skiptoken`, and `$top/$skip` pagination strategies
- **`$apply` aggregation** - `CollectionBuilder.Apply()` for server-side group-by and aggregate expressions

---

## [0.0.1] - Initial release

### Added

- Declarative OData query builder (`$filter`, `$select`, `$expand`, `$orderby`, `$top`, `$skip`, `$count`, `$format`)
- CRUD operations: List, Get, Create, Update, Replace, Delete
- OData v2 and v4 support with auto-detection
- JSON and ATOM response parsing
- Built on relay for all transport-level concerns (retries, timeouts, circuit breaker)
- Typed error types with OData error code and details
