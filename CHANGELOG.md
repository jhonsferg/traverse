# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.1] - 2026-04-07

### Added

- **CSDL JSON v4.01 parser** (`feat/csdl-json-impl`)  -  parses OData CSDL JSON alongside the existing EDMX parser; automatic format detection based on content-type.
- **OpenAPI 3.1 export** (`ext/openapi`)  -  generates an OpenAPI 3.1 spec from OData metadata; `ExportOpenAPI(metadata)` returns a serialisable `*openapi3.T`.
- **OData vocabulary support** (`vocabulary` package)  -  Core, Validation, and SAP Fiori UI annotation types; `ParseVocabularyAnnotations` extracts typed annotation values from EDMX.
- **Microsoft Dataverse adapter** (`ext/dataverse`)  -  typed `DataverseClient` with entity CRUD, batch operations, change tracking, and Dataverse-specific error unwrapping.
- **Offline / persistent store** (`ext/offline`)  -  SQLite-backed cache layer with background sync; `OfflineClient` wraps any traverse client and falls back to the local store when offline.
- **GraphQL bridge** (`ext/graphql`)  -  maps GraphQL queries to OData `$filter`/`$select`/`$expand`; `GraphQLClient.Query` translates and executes against any OData endpoint.
- **Azure Event Grid integration** (`ext/azure`)  -  `EventGridClient` publishes OData change events; `ChangePublisher` hooks into the traverse change-tracking pipeline.
- **Interactive TUI** (`cmd/traverse-tui`)  -  terminal UI (Bubble Tea) for exploring OData endpoints, building queries interactively, and inspecting results; run with `go run ./cmd/traverse-tui`.
- **SAP Fiori annotation support**  -  `UI.LineItem`, `UI.SelectionField`, `UI.HeaderInfo`, and `Common.Label` annotation types in the vocabulary package.
- Improved EDMX parser: handles abstract types, nullable properties, enum members, and complex type inheritance edge cases.

### Fixed

- `traverse.New` is the correct constructor name throughout; removed erroneous `traverse.NewClient` references from README and all documentation pages.
- `cmd/traverse-tui`: resolved golangci-lint errors  -  `errcheck` on `fmt.Sscan` and `resp.Body.Close`, `noctx` using `http.NewRequest`, `staticcheck SA9003` empty branch.

### Documentation

- Complete documentation overhaul: 8 new pages covering `ext/azure`, `ext/offline`, `ext/dataverse`, `ext/openapi`, `ext/audit`, CSDL JSON guide, vocabulary guide, and TUI CLI guide.
- Updated extension index and root index with all new modules.

## [0.1.0] - 2024-01-15

### Added

- Initial release of traverse OData client
- Full OData v2 and v4 support with automatic version detection
- Streaming-first architecture for processing millions of records with constant memory
- Fluent QueryBuilder API for composable OData queries
  - `Select`, `Filter`, `Where`, `Expand`, `OrderBy`, `Top`, `Skip`, `WithCount`
- CRUD operations: Create, Read, Update, Replace, Delete
- Batch operations support via `$batch` endpoint
- OData Functions and Actions support
- Delta sync support for incremental updates
- Comprehensive OData type support:
  - DateTime (SAP /Date(milliseconds)/ format)
  - DateTimeOffset (ISO 8601)
  - Guid
  - Decimal (arbitrary precision via big.Float)
  - Binary (base64)
- SAP-specific adapter (sap/ package)
  - Automatic CSRF token management
  - Basic authentication
  - OAuth2 support (placeholder)
  - Language header support
  - Client parameter routing
- Internal utilities:
  - JSON streaming parser (internal/tokenizer)
  - OData filter value serialization (internal/parser)
  - URL encoding helpers (internal/encoder)
- Test utilities (testutil/)
  - Fixture generators
  - Mock OData server support
- Configuration options via functional options pattern
- Thread-safe Client suitable for concurrent use
- Context-aware operations with cancellation support
- Comprehensive error types and sentinel errors
- Extensive test coverage (≥85%)

### Known Limitations

- CSRF middleware integration pending relay API clarification
- EDMX metadata parsing not fully implemented
- Batch multipart/mixed request building placeholder only
- Bearer token authentication via relay needs documentation
- SAP client parameter in all requests needs refinement

### Documentation

- Comprehensive README with quick start guide
- Full API documentation with examples
- Architecture overview
- Type support documentation
- Memory efficiency guarantees

## [Unreleased]

### Planned

- [ ] Complete EDMX metadata parsing
- [ ] Full batch multipart/mixed support
- [ ] Caching layer for metadata
- [ ] Query performance optimization
- [ ] Support for OData v4 complex types
- [ ] Built-in rate limiting
- [ ] Request logging and tracing
- [ ] GraphQL federation support (future)

---

## Version Format

- **Major**: Breaking API changes
- **Minor**: New features, backwards compatible
- **Patch**: Bug fixes and improvements

## Migration Guide

### Coming Soon (v0.2.0)

When updating between versions, please refer to this section for any breaking changes.
