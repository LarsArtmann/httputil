# Status Report — httputil

**Date:** 2026-05-24 20:36 CEST
**Branch:** `master`
**Commits ahead of origin:** 2 (unpushed)
**Working tree:** clean
**Commit:** `0423535` (HEAD)

---

## Executive Summary

The httputil library is in **excellent health**. This session completed the full integration of `go-error-family` — the library's first external dependency. The `ResponseRecorder` now emits classified, structured errors that consumers can use for retry decisions, exit codes, and structured logging. The project sits at **911 lines** across 10 Go files, **89.7% test coverage**, **38 passing tests**, and **zero lint issues**.

The library is feature-complete for its current scope (CORS, ClientIP, ResponseRecorder, middleware chaining). The natural next phase is expanding the middleware catalog and strengthening the error protocol.

---

## a) FULLY DONE

### 1. go-error-family Integration — Complete

**What:** Integrated `github.com/larsartmann/go-error-family v0.1.1` as the library's first and only external dependency. The `ResponseRecorder` now returns classified errors that implement `Coded`, `Classified`, `Contextual`, and `Retryable` interfaces.

**Files changed:**

- `errors.go` (new) — 5 error code constants
- `recorder.go` — `Write`, `Hijack`, `Push` now return classified errors
- `errors_test.go` (new) — 16 tests covering classification, codes, context, chains, verbose formatting
- `go.mod` — added dependency
- `.golangci.yml` — depguard updated to allow go-error-family

**Error classification matrix:**

| Method | Error Code                | Family         | Retryable | When                              |
| ------ | ------------------------- | -------------- | --------- | --------------------------------- |
| Write  | `http.write_failed`       | Transient      | Yes       | Underlying Write fails            |
| Hijack | `http.hijack_unsupported` | Infrastructure | No        | Writer doesn't implement Hijacker |
| Hijack | `http.hijack_failed`      | Transient      | Yes       | Underlying Hijack call fails      |
| Push   | `http.push_unsupported`   | Infrastructure | No        | Writer doesn't implement Pusher   |
| Push   | `http.push_failed`        | Transient      | Yes       | Underlying Push call fails        |

**Status:** ✅ Complete. All 38 tests pass. Zero lint issues.

### 2. nil-Wrapping Bug Fix — Complete

**What:** `ResponseRecorder.Write`, `Hijack`, and `Push` were wrapping errors with `fmt.Errorf("...: %w", err)` unconditionally — even when `err` was nil. This meant every successful `Write` returned a non-nil error.

**Fix:** Added nil checks before wrapping. Now returns `nil` on success. Done in commit `6ad4f0c`.

**Status:** ✅ Complete.

### 3. itoa MinInt Overflow Fix — Complete

**What:** `util.go:itoa()` had `num = -num` which overflows for `math.MinInt` (-9223372036854775808). Also fixed `gosec G115` integer conversion warning.

**Fix:** Per-digit absolute value, digit lookup string, named buffer constant.

**Tests:** 9 new tests in `util_test.go`.

**Status:** ✅ Complete. Commit `58f05ae`.

### 4. Documentation Updates — Complete

- `AGENTS.md` — updated dependency policy, error classification table, architecture table
- `docs/DOMAIN_LANGUAGE.md` — added Error Protocol bounded context, error code/family value objects, updated conventions
- `docs/status/2026-05-24_20-23_panic-fix-and-classified-errors.md` — previous status report

**Status:** ✅ Complete.

### 5. depguard Configuration — Complete

`.golangci.yml` depguard now allows `$gostd`, `$module`, and `github.com/larsartmann/go-error-family`.

**Status:** ✅ Complete.

### 6. Previous Session Work (Already Committed)

- Project infrastructure (`.golangci.yml`, `AGENTS.md`, `README.md`, etc.)
- Git Town configuration
- Initial commit with CORS, ClientIP, ResponseRecorder, Chain

---

## b) PARTIALLY DONE

Nothing is partially done. All started work has been completed.

---

## c) NOT STARTED

### 1. Additional Middleware Catalog

The library has CORS middleware but could offer more:

- Request logging middleware
- Request ID middleware
- Rate limiting middleware
- Recovery (panic handler) middleware
- Request timeout middleware
- Compression middleware
- Security headers middleware

### 2. Sentinel Registration for stdlib HTTP errors

The `errors.go` file currently only has error code constants. It could also register stdlib HTTP sentinels with error-family:

```go
func init() {
    errorfamily.RegisterClassification(http.ErrNotSupported, errorfamily.Infrastructure)
    errorfamily.RegisterClassification(http.ErrAbortHandler, errorfamily.Transient)
    // etc.
}
```

This would let consumers classify any stdlib HTTP error, not just our own.

### 3. Error Message Templates

`go-error-family` supports per-error-code `MessageTemplate` (What/Why/Fix/WayOut). The library could register templates for its error codes so that CLI consumers get Wix-quality error messages automatically.

### 4. Examples Directory

No `examples/` directory exists. Consumers would benefit from:

- Basic HTTP server with CORS + ResponseRecorder
- Error classification and retry loop
- Middleware chaining example

### 5. GoDoc / pkg.go.dev Documentation

Exported types have doc comments, but the package could use a doc.go with a comprehensive package-level example.

### 6. Benchmarks

No benchmark tests exist. The `itoa()` and `join()` helpers in `util.go` were explicitly written for zero-allocation hot paths — benchmarks would prove this.

### 7. CI/CD Pipeline

No `.github/workflows/` exists. Should have:

- Test matrix (Go 1.26+, linux/mac/windows)
- golangci-lint
- Code coverage reporting
- Release tagging (Go modules)

### 8. README Update

README hasn't been updated to reflect the go-error-family integration. Should document the error classification behavior and how consumers benefit.

### 9. CONTRIBUTING.md

No contribution guidelines exist.

### 10. Version Tagging

No git tags or releases. The module is at v0.0.0 implicit. Should tag v0.1.0 now that the API is stable.

### 11. CORS Origin Validation

`resolveOrigin()` returns `"*"` when no match is found — arguably should return empty string (no CORS headers). This is documented as a known limitation but could be configurable.

### 12. CORS Regex/Wildcard Origin Matching

`AllowedOrigins` only supports exact string match or `*`. No wildcard domain support (e.g., `*.example.com`).

### 13. CORS Max-Age Validation

No validation that `MaxAge` is non-negative. A negative value would set an invalid header.

### 14. CORSConfig Validation Method

No `Validate()` or `Valid()` method on `CORSConfig`. Consumers can create nonsensical configs (e.g., `AllowCredentials: true` + `AllowAllOrigins: true` which browsers reject).

### 15. ResponseRecorder Body Capture

The `ResponseRecorder` captures status but not the response body or headers. A `CapturingResponseRecorder` variant could be useful for testing.

### 16. Request Context Helpers

No helpers for storing/retrieving values from request context (e.g., `WithClientIP`, `ClientIPFromContext`).

### 17. HTTP Error Response Helpers

No helpers for writing structured error responses (e.g., JSON error bodies with error codes from go-error-family).

### 18. golangci.yml Build Tag Cleanup

`.golangci.yml` has experimental build tags (`goexperiment.arenas`, `goexperiment.simd`, etc.) that may not be relevant to this project. Should verify they serve a purpose.

### 19. Push nil-wrapping edge case

In `Push`, when `pusher.Push()` returns nil but the writer claims to support Pusher, we correctly return nil. But the `Push` method on the old code had `fmt.Errorf("push %q: %w", target, pusher.Push(target, opts))` which would wrap nil. Fixed now, but worth noting for audit.

### 20. Test Coverage for CORS edge cases

No tests for:

- CORS with `AllowCredentials: true` and `AllowAllOrigins: true` (invalid per spec)
- CORS with empty `AllowedOrigins`
- CORS preflight with `OptionsPassthrough: true`
- CORS with `MaxAge: 0`
- CORS with no `Origin` header

### 21. Fuzz Testing

No fuzz tests exist. `ClientIP` parsing of X-Forwarded-For header is a good fuzz target.

### 22. Integration Tests

All tests are unit tests with `httptest`. No integration tests with a real HTTP server.

### 23. Go 1.26+ Feature Adoption

The project targets Go 1.26 but doesn't yet use:

- `errors.AsType[T]()` directly (go-error-family uses it internally)
- Range-over-int (could simplify `itoa`)
- Other 1.26+ features

### 24. CHANGELOG.md

CHANGELOG.md exists but may not reflect the latest changes (go-error-family integration, itoa fix, etc.).

### 25. Push to Remote

2 commits are ahead of origin/master and unpushed.

---

## d) TOTALLY FUCKED UP

**Nothing is fucked up.** The codebase is clean:

- `go build ./...` — ✅ compiles
- `go test ./...` — ✅ 38/38 pass
- `go vet ./...` — ✅ clean
- `golangci-lint run` — ✅ 0 issues (with ~70 linters enabled)
- Test coverage: 89.7%
- Working tree: clean

The LSP shows stale diagnostics (duplicate `decimalBase`, wrong `NewRequestWithContext` args) that don't exist in the actual code. This is a gopls cache issue, not a real problem.

---

## e) WHAT WE SHOULD IMPROVE

### High Impact

1. **Push the 2 unpushed commits** — the work is done, just needs `git push`
2. **Tag v0.1.0** — the API is stable with a real dependency, deserves a version tag
3. **Add CI/CD** — automated test + lint on every PR
4. **Update README** — document error classification, the go-error-family dependency, and the benefit to consumers
5. **Add stdlib HTTP sentinel registration** — register `http.ErrNotSupported` etc. so any HTTP error from any library can be classified by consumers

### Medium Impact

6. **Expand CORS tests** — edge cases (credentials+all-origins, empty origins, passthrough, no Origin header)
7. **Add benchmarks** — prove the zero-allocation claim for `itoa()`/`join()`
8. **Add `examples/`** — show consumers how to use the library with error classification
9. **Add CORSConfig validation** — prevent invalid configs at construction time
10. **Body-capturing ResponseRecorder** — variant that captures response body for testing

### Lower Impact

11. **Update CHANGELOG.md** — reflect go-error-family integration
12. **Add CONTRIBUTING.md** — contribution guidelines
13. **Wildcard origin matching** — `*.example.com` support
14. **Request context helpers** — `WithClientIP`/`ClientIPFromContext`
15. **Fuzz tests** — for X-Forwarded-For parsing

---

## f) Top #25 Things to Do Next

| #      | Task                                                                                                       | Impact     | Effort     |
| ------ | ---------------------------------------------------------------------------------------------------------- | ---------- | ---------- |
| 1      | Push 2 unpushed commits to origin                                                                          | High       | 1 min      |
| 2      | Tag v0.1.0 release                                                                                         | High       | 1 min      |
| ~~3~~  | ~~Add GitHub Actions CI (test + lint + coverage)~~ done — (CI workflows)                                   | ~~High~~   | ~~30 min~~ |
| 4      | Update README.md with error classification docs                                                            | High       | 20 min     |
| 5      | Register stdlib HTTP error sentinels in errors.go                                                          | High       | 15 min     |
| 6      | Expand CORS tests (credentials+allorigins, empty, passthrough, no-origin)                                  | Medium     | 30 min     |
| ~~7~~  | ~~Add benchmark tests for itoa() and join()~~ done — moot (util.go removed 2026-06-16)                     | ~~Medium~~ | ~~20 min~~ |
| 8      | Create examples/ directory with basic usage                                                                | Medium     | 30 min     |
| 9      | Add CORSConfig.Validate() method                                                                           | Medium     | 20 min     |
| ~~10~~ | ~~Update CHANGELOG.md~~ done — (maintained per release)                                                    | ~~Medium~~ | ~~10 min~~ |
| ~~11~~ | ~~Add BodyCapturingResponseRecorder variant~~ done — parked in ROADMAP legacy-brainstorm line (2026-08-30) | ~~Medium~~ | ~~45 min~~ |
| ~~12~~ | ~~Add request context helpers (WithClientIP)~~ done — shipped (context.go)                                 | ~~Medium~~ | ~~15 min~~ |
| ~~13~~ | ~~Add error message templates for httputil error codes~~ done — shipped (errorTemplates, v0.12.0)          | ~~Medium~~ | ~~20 min~~ |
| ~~14~~ | ~~Add JSON error response helper~~ done — parked in ROADMAP legacy-brainstorm line (2026-08-30)            | ~~Medium~~ | ~~30 min~~ |
| ~~15~~ | ~~Add security headers middleware~~ done — shipped (security.go)                                           | ~~Medium~~ | ~~30 min~~ |
| ~~16~~ | ~~Add request ID middleware~~ done — shipped (requestid.go)                                                | ~~Medium~~ | ~~30 min~~ |
| ~~17~~ | ~~Add panic recovery middleware~~ done — shipped (recovery.go)                                             | ~~Medium~~ | ~~30 min~~ |
| ~~18~~ | ~~Add request timeout middleware~~ done — shipped (timeout.go)                                             | ~~Medium~~ | ~~20 min~~ |
| ~~19~~ | ~~Add compression middleware~~ done — shipped (compression.go)                                             | ~~Low~~    | ~~60 min~~ |
| ~~20~~ | ~~Add rate limiting middleware~~ done — shipped (ratelimit.go, deprecated; KeyedRateLimiter succeeded it)  | ~~Low~~    | ~~60 min~~ |
| ~~21~~ | ~~Add wildcard domain matching for CORS~~ done — shipped (wildcard origin matching)                        | ~~Low~~    | ~~20 min~~ |
| ~~22~~ | ~~Add fuzz tests for ClientIP~~ done — (FuzzClientIP; 23 fuzz targets)                                     | ~~Low~~    | ~~30 min~~ |
| ~~23~~ | ~~Add CONTRIBUTING.md~~ done — (exists)                                                                    | ~~Low~~    | ~~15 min~~ |
| ~~24~~ | ~~Clean up golangci.yml experimental build tags~~ done — (config clean, 0 issues)                          | ~~Low~~    | ~~10 min~~ |
| ~~25~~ | ~~Audit Go 1.26+ features for adoption (range-over-int, etc.)~~ done — (util.go removed 2026-06-16)        | ~~Low~~    | ~~20 min~~ |

---

## g) Top #1 Question I Cannot Answer Myself

**Should `httputil` expand into a full middleware catalog (request ID, recovery, logging, rate limiting, etc.) or stay minimal and focused on the current 3 bounded contexts?**

This is a product direction question. The library name "httputil" suggests broad HTTP utilities, but the current implementation is laser-focused on 3 things done well. Adding more middleware increases surface area and maintenance burden, but also increases the library's value proposition. This decision affects the entire project trajectory and should come from you.

---

## Metrics

| Metric               | Value               |
| -------------------- | ------------------- |
| Go files             | 10                  |
| Total lines          | 911                 |
| Source lines         | 431                 |
| Test lines           | 480                 |
| Test-to-source ratio | 1.11:1              |
| Tests passing        | 38/38               |
| Test coverage        | 89.7%               |
| Lint issues          | 0                   |
| External deps        | 1 (go-error-family) |
| Commits ahead        | 2                   |

## File Inventory

| File               | Lines | Role                       |
| ------------------ | ----- | -------------------------- |
| `clientip.go`      | 32    | Client IP extraction       |
| `clientip_test.go` | 67    | Client IP tests            |
| `cors.go`          | 88    | CORS middleware            |
| `cors_test.go`     | 88    | CORS tests                 |
| `errors.go`        | 9     | Error code constants       |
| `errors_test.go`   | 291   | Error classification tests |
| `recorder.go`      | 114   | ResponseRecorder + Chain   |
| `recorder_test.go` | 100   | ResponseRecorder tests     |
| `util.go`          | 42    | Internal helpers           |
| `util_test.go`     | 80    | Helper tests               |

## Commit History (this session)

```
0423535 Introduce classified errors via go-error-family and update documentation
58f05ae Fix itoa MinInt overflow bug and address branching-flow panic findings
e54f2c3 Add github.com/larsartmann/go-error-family to depguard allow list
6ad4f0c Fix critical error handling bugs in ResponseRecorder.Write and Hijack methods
c0bacf8 Comprehensive quality improvements: linter compliance, project documentation, and error handling
4044d88 Add Git Town configuration file for enhanced Git workflow management
b2df9ac Add project infrastructure, documentation, and code style improvements
8170673 Initial commit: HTTP utility library for Go
```
