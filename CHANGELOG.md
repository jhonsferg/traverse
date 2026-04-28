# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.20.0] - 2026-04-27

### Added

- `feat(entity)`: new `CreateRawAs()` generic function returns raw response bytes from Create operations, complementing `CreateJsonAs()` and `CreateXmlAs()`. Useful for debugging, testing, handling non-standard OData formats, and transparently capturing both JSON and XML responses. Includes comprehensive test coverage via `TestCreateRawAs()`.

## [0.19.0] - 2026-04-27

### Added

- `feat(phase-10)`: Critical SAP CSRF Architecture Fix addressing 7 interrelated bugs that broke OData integration with SAP backends:
  - **BUG-001** (CRITICAL): CookieJar support in relay v0.4.0 — http.Client now automatically captures and reuses session cookies, fixing CSRF token validation failures. This is the root cause of all `403 FORBIDDEN` errors in enterprise SAP integrations.
  - **BUG-002** (CRITICAL): Automatic cookie management via http.CookieJar — no manual cookie handling needed.
  - **BUG-003** (CRITICAL): Token lifecycle fix — tokens are now reused for their full 30-minute validity window instead of being invalidated before every request.
  - **BUG-004** (HIGH): Response hooks properly connected — `HandleResponse()` now correctly subscribes to `OnAfterResponse` to enable automatic 403 recovery.
  - **BUG-005** (HIGH): CSRF middleware API consistency — `Hook()` method signature aligns with relay's `OnBeforeRequest` hook system.
  - **BUG-006** (MEDIUM): URL normalization — fixed malformed URLs with multiple consecutive slashes (`///sap/opu/...`) using `url.URL` struct instead of `fmt.Sprintf`.
  - **BUG-007** (MEDIUM): Error diagnostics — implemented `ErrorDiagnostic` with 9 error categories (csrf_expired, csrf_invalid, auth_failed, network_error, etc.) to distinguish between configuration, security, and transport failures.

### Changed

- `fix(ext/sap)`: improved CSRF middleware initialization and token management. Token fetch now correctly captures and stores session cookies alongside tokens. All subsequent requests automatically include both token header (`X-CSRF-Token`) and cookie header, fixing atomic CSRF+cookie validation.
- `fix(xml)`: XML struct mapping now correctly identifies response content type from `Content-Type` header to distinguish JSON vs XML responses from SAP backends that may ignore Accept header preferences.
- `refactor(url)`: URL construction in `ext/sap/client.go` uses `url.URL` struct for proper path joining, eliminating edge cases that produced double-slash patterns.

### Fixed

- `fix(client)`: prevent double-encoding of OData string key literals; `ProductSet(Product='ABC')` was being re-encoded to `ProductSet(Product=%27ABC%27)` on subsequent URL rewrites.

## [0.14.0] - 2026-04-10

### Added

- `feat(cmd)`: `sap-mock-server` (`cmd/sap-mock`) — a standalone SAP OData v2 mock server for local integration testing; simulates CSRF token lifecycle, Basic Auth, `$metadata` responses, entity-set query and key-predicate endpoints, and property-path navigation; request/response logging with query-param and body inspection.

## [0.13.30] - 2026-04-09

### Fixed

- `fix(cache)`: use comma-ok pattern on `sync.Map` type assertion in `Get` to prevent panic when the stored value is an unexpected type (v0.13.30).
- `fix(stream)`: respect the per-client `maxPages` limit in all pagination loops — previously the limit was enforced only on the first page fetch (v0.13.29).
- `fix(deep)`: accept the full 2xx range (200–299) in deep insert and deep update responses; previously only 200 and 201 were accepted, causing `DeepUpdate` to return an error on 204 No Content from some SAP services (v0.13.28).
- `fix(types)`: correct `Binary` base64 encoding and decoding; OData `Edm.Binary` values were incorrectly using standard base64 instead of the URL-safe variant required by the spec (v0.13.27).
- `fix(query)`: place entity key predicate before embedded query params in the URL path; `EntitySet('key')?$select=...` was previously emitted as `EntitySet?$select=...('key')` (v0.13.26).
- `fix(ext/webhooks)`: `Delete()` hung indefinitely when auto-renew was disabled; the renew goroutine used a context that was never cancelled (v0.13.25).
- `fix(batch)`: goroutine leak when `ExecuteStream` was cancelled; missing `release()` call on early exit; part count limit not enforced for streaming batch requests (v0.13.24).
- `fix(stream)`: TOCTOU race in `pool.submit` between capacity check and enqueue; goroutine leak on context cancellation; SSRF and infinite-loop guard in prefetch worker (v0.13.23).
- `fix(tracking)`: `SaveChanges` now takes an atomic snapshot of dirty fields before sending the PATCH; on failure the dirty set is restored, preventing partial updates from silently discarding changes (v0.13.22).
- `fix(offline)`: propagate body read error from the offline store instead of returning partial data silently (v0.13.21).

## [0.13.20] - 2026-04-09

### Fixed

- `fix(ext/graphql)`: replace poisoned `sync.Once` with a retryable mutex for schema build to allow recovery after a transient schema fetch error (v0.13.20).
- `fix(ext/tracing)`: add per-span mutex to eliminate data race on `Span` fields when `Finish` and attribute setters are called concurrently (v0.13.19).
- `fix(ext/prometheus)`: unexport mutable counter fields to eliminate data race in `WithPrometheus` transport middleware (v0.13.18).
- `fix(ext/webhooks)`: surface auto-renewal errors via `OnRenewError` callback instead of silently swallowing them (v0.13.17).
- `fix(ext/oauth2)`: credential value leaked into error message; `singleflight` group was not propagating the error to all waiting callers on token refresh failure (v0.13.16).
- `fix(delta,batch)`: `sync.Map`-backed pool leaked map entries after eviction; EOF comparison used `==` on wrapped errors instead of `errors.Is` (v0.13.15).
- `fix(ext/graphql)`: implement `getSelectedFields` to enable OData `$select` optimisation when executing GraphQL queries (v0.13.14).
- `fix(graph,deep_update)`: wire relay client correctly in graph traversal and deep-update paths; sanitise `Prefer` header value to prevent CRLF injection (v0.13.12 / v0.13.13).
- `fix(ext)`: SAP `ext/sap` dropped Basic Auth header on redirect; `ext/oauth2` singleflight did not propagate errors; `ext/prometheus` counter labels were incorrect for retry attempts (v0.13.11).
- `fix(graphql,cache/redis)`: data race in GraphQL schema cache concurrent build; `cache/redis` scan silently discarded items with unmarshalling errors (v0.13.10).
- `fix(azure)`: drain EventGrid response body before close to allow connection reuse (v0.13.9).
- `fix(cache/memory)`: eliminate TOCTOU race in `Get()` between expiry check and value return (v0.13.8).
- `fix(ext/tracing,ext/webhooks)`: memory leak in tracing span pool; invalid W3C trace-ID generation; handler registration race in webhooks (v0.13.7).
- `fix(ext)`: multiple stability fixes — latency histogram OOM on unbounded label cardinality, stop-signal loss in background workers, non-atomic file writes in offline store, error visibility in batch changesets (v0.13.6).
- `fix(client)`: strip leading slashes from entity set name in `From()` to prevent double-slash in URLs (v0.13.5).

## [0.13.3] - 2026-04-08

### Fixed

- `fix(security)`: key injection via `Key()` parameter; `nextLink` SSRF allowing redirect to arbitrary hosts; unbounded metadata EDMX parse OOM; missing batch part count limit; ETag value CRLF injection (v0.13.3).
- `fix(security)`: CSRF token thundering herd — concurrent requests all fetching a token simultaneously; header injection via user-supplied SAP client number; batch changeset errors silently discarded; error response body leaked into log output (v0.13.2).
- `fix(client)`: `WithRelayClient` did not inherit the base URL; relay options passed to `New()` were not applied when a custom client was provided (v0.13.1).

## [0.13.0] - 2026-04-08

### Added

- `feat(batch)`: OData 4.01 JSON batch format — `WithJSONBatch()` option on the batch builder sends `multipart/mixed` or `application/json` batch requests per the 4.01 spec; `ExecuteStream` processes response parts as a channel.

## [0.12.0] - 2026-04-08

### Added

- CI: per-extension auto-tagging — each `ext/*` module is tagged independently when its `go.mod` changes; coverage gate raised to 85.6%.

## [0.11.0] - 2026-04-08

### Added

- `feat(geo)`: geospatial primitive types — `GeographyPoint`, `GeographyPolygon`, `GeometryPoint`, and the full OData 4.0 geospatial type hierarchy with GeoJSON marshal/unmarshal.
- `feat(geo)`: geospatial filter functions — `GeoDistance`, `GeoIntersects`, `GeoLength`, `GeoContains`, `GeoCrosses` in the filter expression builder.
- `feat(geo)`: GeoJSON serialisation for create/update request bodies.
- `perf`: zero-alloc `sync.Pool` for `GeographyPoint` and `GeometryPoint`; string interning extended to cover geo coordinate literals (F3/F4 phases).

## [0.9.0] - 2026-04-07

### Added

- `feat(vocabulary)`: Measures vocabulary (`@Measures.ISOCurrency`, `@Measures.Scale`, `@Measures.Unit`, `@Measures.DurationUnit`) for annotating numeric fields with currency and unit metadata.
- `feat(vocabulary)`: Authorization vocabulary (`@Authorization.Authorizations`, `@Authorization.SecuritySchemes`) for generating OpenAPI security definitions from OData metadata.
- `feat(vocabulary)`: Analytics vocabulary (`@Analytics.AggregationMethod`, `@Analytics.Dimension`, `@Analytics.Measure`) for SAP BW and Power BI connected services.
- `feat(stream)`: Atom/XML response body parser — streaming `xml.Decoder`-based parser for `application/atom+xml` OData v2 responses; format auto-detected from `Content-Type`; constant memory usage.

### Added (from v0.8.0 / Phase 2)

- `feat(odata)`: Deep Update — `PATCH` with nested entity bodies in a single round-trip via `From("Orders").Key(id).DeepUpdate(ctx, patch)`.
- `feat(ext/sap)`: SAP metadata attribute extensions — `sap:label`, `sap:filterable`, `sap:sortable`, `sap:required-in-filter`, `sap:updatable-path`, `sap:creatable`, `sap:deletable`, `sap:pageable` parsed from EDMX and exposed via `ext/sap/sap_attributes.go`.

### Added (from v0.7.0 / Phase 1)

- `feat(odata)`: Singletons as first-class entities — `client.Singleton("me").Page(ctx)` and `traverse.SingletonAs[T](client, "me").Find(ctx)` with zero-alloc URL construction.
- `feat(odata)`: Derived types and type casting — `AsType("Model.Manager")` path segment, `IsOf("Model.Manager")` and `Cast("Edm.Decimal")` filter builder helpers.
- `feat(query)`: `$expand` with `$levels` — `Expand("Children", traverse.WithExpandLevels(traverse.LevelsMax))` for recursive tree expansion.
- `feat(odata)`: BulkUpdate — `PATCH /EntitySet?$filter=...` via `From("Products").Filter(...).BulkUpdate(ctx, patch)`.
- `feat(odata)`: Advanced `Prefer` headers — `WithPrefer(traverse.PreferHandlingStrict)`, `ReturnMinimal`, `ReturnRepresentation`, `PreferTrackChanges`.
- `feat(odata)`: `$schemaversion` header via `WithSchemaVersion("2.0")` at client or per-query level.
- `fix(query)`: OData v2 `$inlinecount=allpages` emitted correctly when client is configured for v2; response key `d.__count` parsed alongside v4 `@odata.count`.

## [0.5.0] - 2026-04-06

### Added

- `feat(sap)`: `WithSAPTLSConfig(cfg)` option for configuring TLS settings on the SAP relay client, enabling custom CA bundles and insecure-TLS bypass for self-signed certificates.
- `feat(entity)`: `FetchPropertyAs[T](ctx, entitySet, key, property)` generic helper for fetching a single scalar or complex property from an entity by key and property path.
- `feat(stream)`: single-entity OData v2 response parsing via `FetchByKey` — handles the `d: { results: {...} }` envelope returned by SAP ABAP Gateway.

### Fixed

- `fix(query)`: handle pre-existing query string embedded in the `entitySet` path passed to `From()`.

### Changed

- `chore(deps)`: relay dependency bumped to v0.2.0 across core and all extension modules.
- `style`: replace em dash (U+2014) with plain hyphen throughout all source files for consistent rendering.

## [0.4.0] - 2026-04-05

### Added

- `feat(batch)`: OData 4.01 JSON batch format support alongside existing `multipart/mixed`.
- `feat(query)`: `Key()` builder method for fluent key-predicate construction; `BulkDelete` via `DELETE /EntitySet?$filter=...`.
- `feat(query)`: background page prefetching via `WithPrefetch(n)` — pre-loads the next N pages while the caller processes the current one.
- `feat(query)`: instance annotations (`@odata.type`, custom annotations) and `$compute` expression support.
- `feat(ext/audit)`: audit trail extension recording all OData write operations with timestamp, principal, entity set, key, and before/after delta.
- `feat(cache)`: HTTP response caching and ETag conditional requests integrated into the core client via `WithCacheStore`.
- `feat(query)`: `$search` expression DSL and `$crossjoin` builder for multi-entity-set cross products.
- CI: per-extension coverage warnings and OData conformance test suite.
- `feat(webhooks)`: OData v4 webhook subscription manager extension (`ext/webhooks`).
- `feat(capabilities)`: OData v4 capabilities vocabulary parsing and validation.

## [0.2.0] - 2026-04-04

### Added

- `feat(traverse)`: integrated relay v0.1.12 — all relay resilience and transport options (retry, circuit breaker, rate limiting, load balancing, TLS pinning) are now available via `New(traverse.WithRelayOptions(...))`.
- `feat(ext/azure)`: Azure Event Grid integration (`ext/azure`) — `EventGridClient` publishes OData change events; `ChangePublisher` hooks into the traverse change-tracking pipeline.
- `feat(ext/graphql)`: GraphQL bridge (`ext/graphql`) — maps GraphQL queries to OData `$filter`/`$select`/`$expand`; `GraphQLClient.Query` translates and executes against any OData endpoint.
- `feat(ext/dataverse)`: Microsoft Dataverse adapter (`ext/dataverse`) — typed `DataverseClient` with entity CRUD, batch operations, change tracking, and Dataverse-specific error unwrapping.
- `feat(offline)`: persistent offline store (`ext/offline`) — SQLite-backed cache layer with background sync; `OfflineClient` wraps any traverse client and falls back to the local store when offline.
- `feat(parser)`: improved EDMX parser — handles abstract types, nullable properties, enum members, and complex type inheritance edge cases.
- CI: autotag and autorelease workflows; single unified pipeline.

### Fixed

- `fix(delta)`: eliminate data race in `DeltaSync` token updates under concurrent page fetches (v0.2.1).

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

---

## Version Format

- **Major**: Breaking API changes
- **Minor**: New features, backwards compatible
- **Patch**: Bug fixes and improvements
