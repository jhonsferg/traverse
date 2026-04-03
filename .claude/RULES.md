# RULES.md - Claude Code Standing Instructions
# relay · E:\Projects\jhonsferg\relay
# Read this first in every session that touches performance work.

---

## 1. COMMIT RULES (ABSOLUTE)

- **NO Co-Authored-By**, no Signed-off-by, no author emails in commits - ever.
- Subject line ≤ 72 chars, imperative mood, no trailing period.
- Format: `type(scope): description`  
  Valid types: `feat` `fix` `perf` `refactor` `test` `bench` `docs` `chore` `ci` `security`
- Body only when the *why* is non-obvious.

## 2. BRANCH & PUSH RULES

- Performance work goes on `optimize/` branches, not directly to master.
- Before any push to master, temporarily disable `enforce_admins`:
  ```
  gh api -X DELETE repos/jhonsferg/relay/branches/master/protection/enforce_admins
  git push origin master
  gh api -X POST  repos/jhonsferg/relay/branches/master/protection/enforce_admins
  ```
- Never force-push master.

## 3. TEST RULES (ABSOLUTE)

- Every change must pass `go test ./...` locally before pushing.
- CI runs with `-race`; any DATA RACE is a blocker, not a warning.
- Known safe pattern: **never pool `httptrace.ClientTrace`** - the transport's
  `dialParallel` fires callbacks from background goroutines after `Do()` returns.
  Use `atomic.Int64` for any field written by trace callbacks (see `timing.go`).
- Coverage must stay ≥ 85%.

## 4. PERFORMANCE RULES

- Every optimisation must show a measurable improvement in benchmarks (> 5%).
- Measure with `go test -bench=. -benchmem -count=5` before AND after each change.
- Use `benchstat before.txt after.txt` to validate; attach results in commit body.
- Do not optimise what has not been measured. No speculative micro-optimisations.

## 5. API STABILITY RULES

- Zero breaking changes to exported types, functions, or interfaces.
- Public `Response` fields are stable: `StatusCode`, `Status`, `Headers`, `Timing`,
  `Truncated`, `RedirectCount`, `Body()`, `String()`, etc.
- `PutResponse(r)` is opt-in; callers who don't call it are safe (no UAF).

## 6. MEMORY SAFETY RULES

- Pooled buffers from `pool.GetSizedBuffer` must be returned via `pool.PutSizedBuffer`
  before the `Response` is handed back to the caller - the caller owns the body slice.
- `bytes.Reader` from `pool.GetBytesReader` must be released in `releasePooledReader()`,
  which is called after every `RoundTrip` attempt (success or failure).
- Never return a slice that aliases a pool buffer to the caller.

## 7. GOLANGCI-LINT V2 RULES

- Config file: `.golangci.yml` - version `"2"` required.
- Formatters (`gofmt`, `goimports`) go under `formatters:` section, NOT `linters:`.
- `gosimple` and `typecheck` do not exist in v2 - do not add them.
- `exclude-dirs` does not exist at the top level or under `issues:` in v2.
- `misspell` locale is `UK` (codebase uses British English).
- Run `gofmt -w <file>` before committing any file that touches formatting.

## 8. WHEN TO READ ADDITIONAL CONTEXT

If starting a new session, read in this order:
1. This file (`RULES.md`)
2. `WORK_PLAN.md` - current phase status and next tasks
3. `CLAUDE.md` - original commit rules and zero-alloc plan with checkboxes
4. `C:\Users\Jhon\.claude\projects\E--Projects-jhonsferg-relay\memory\MEMORY.md`

## 9. TYPOGRAPHY RULES

- Never use the em-dash symbol. Use a plain hyphen (-) instead.

---

*Last updated: 2026-04-03*
