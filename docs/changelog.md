# Changelog

All notable changes to traverse are documented in this file.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).
traverse uses [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [0.20.0] - 2026-04-27

### Added

- **CreateRawAs Method** (`entity.go`) - New generic function `CreateRawAs[T]()` returns raw response bytes from Create operations, complementing `CreateJsonAs()` and `CreateXmlAs()`. Useful for debugging, testing, handling non-standard OData formats, and transparently capturing both JSON and XML responses from SAP and other backends that may ignore Accept header preferences. Includes comprehensive test coverage.

### Notes

- Fully backward compatible - all existing APIs unchanged
- Complements the Phase 10 CSRF architecture fixes in traverse v0.19.0

---

## [0.19.0] - 2026-04-27 - Phase 10: Critical SAP CSRF Architecture Fix

### Added

- **Complete CSRF Token + Cookie Management** - Traverse now handles SAP CSRF tokens completely transparently, including session cookie management. Fixed 7 interrelated architectural bugs that broke OData integration with SAP backends:

  **BUG-001 (CRITICAL)** - CookieJar support in relay v0.4.0
  - `http.Client` now automatically captures and reuses session cookies
  - Fixes CSRF token validation failures
  - Root cause of all `403 FORBIDDEN` errors in enterprise SAP integrations
  - Fully transparent—no code changes needed

  **BUG-002 (CRITICAL)** - Automatic cookie management via http.CookieJar
  - No manual cookie handling required
  - Cookies are captured from `Set-Cookie` headers
  - Automatically included in subsequent requests
  - Managed transparently by Go's standard library

  **BUG-003 (CRITICAL)** - Token lifecycle fix
  - CSRF tokens are now reused for their full 30-minute validity window
  - Previously tokens were invalidated before every request, causing unnecessary latency
  - Automatic 403 recovery when tokens actually expire

  **BUG-004 (HIGH)** - Response hooks properly connected
  - `HandleResponse()` now correctly subscribes to `OnAfterResponse` hook
  - Enables automatic 403 recovery without retry logic in application code
  - Invalid tokens are detected and refreshed transparently

  **BUG-005 (HIGH)** - CSRF middleware API consistency
  - `Hook()` method signature aligns with relay's `OnBeforeRequest` hook system
  - Clean, documented API contract

  **BUG-006 (MEDIUM)** - URL normalization
  - Fixed malformed URLs with multiple consecutive slashes (`///sap/opu/...`)
  - Uses `url.URL` struct instead of `fmt.Sprintf` for proper path joining
  - Handles edge cases: trailing slashes, no scheme, special characters, URL encoding

  **BUG-007 (MEDIUM)** - Error diagnostics
  - Implemented `ErrorDiagnostic` with 9 error categories:
    - `csrf_expired` - token validation window expired
    - `csrf_invalid` - token format invalid
    - `auth_failed` - authentication failure
    - `network_error` - transport/connection error
    - `service_unavailable` - backend overloaded
    - `config_error` - metadata or configuration issue
    - And 3 others for observability
  - Includes suggested fixes and `IsRetryable()` flags for resilience

### Changed

- **ext/sap** - Improved CSRF middleware initialization and token management:
  - Token fetch now correctly captures and stores session cookies
  - All subsequent requests automatically include both token header (`X-CSRF-Token`) and cookie header
  - Fixes atomic CSRF+cookie validation required by SAP Gateway

- **XML Support** - XML struct mapping now correctly identifies response content type from `Content-Type` header:
  - Distinguishes JSON vs XML responses from SAP backends that may ignore Accept header
  - Auto-detects format without explicit type hints

- **URL Construction** - `ext/sap/client.go` uses `url.URL` struct for proper path joining:
  - Eliminates edge cases that produced double-slash patterns
  - Handles special characters and encoding correctly

### Notes for Upgrade

- **100% backward compatible** - All existing code works without changes
- **Transparent CSRF handling** - No application code changes needed to fix CSRF failures
- **Automatic error recovery** - 403 errors are automatically recovered without retry logic
- **Enhanced diagnostics** - Better error messages distinguish between config, auth, CSRF, and network failures

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
