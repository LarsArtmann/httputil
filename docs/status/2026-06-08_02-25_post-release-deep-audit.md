# Status Report — Post-Release Deep Audit

_Date: 2026-06-08 02:25_
_Author: Crush (AI assistant)_
_Trigger: Post-release audit after v0.1.0 push — "what did we miss"_

---

## Executive Summary

v0.1.0 is pushed and the release CI is running. All quality gates pass (110 tests, 89.1% coverage, 0 lint, 0 races). However, this deep forensic audit uncovered a **critical data race** in `CORS()` that slipped through because `go test -race` only catches races that manifest during the test run — and no current test exercises concurrent requests with different origins.

Additionally, CHANGELOG.md has stale metrics and AGENTS.md is missing 2 error codes. These are doc fixes. The race condition is a real production bug.

---

## A. FULLY DONE ✅

| Item                                                       | Evidence                                                                                                             |
| ---------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------- |
| 10 middlewares — all return `Middleware` type              | Consistent across all factory functions                                                                              |
| 5 config `Validate()` methods — all with tests             | RequestIDConfig (3 checks), ETagConfig (1), SecurityHeadersConfig (0 — no-op), CORSConfig (2), CompressionConfig (2) |
| Error classification — 7 error codes, all with templates   | `RegisterErrorClassifications()` registers all                                                                       |
| `responseWrapper` — shared base for compress/etag          | Correct design, no panics                                                                                            |
| Context keys — correct pattern (unexported empty struct)   | `clientIPKey{}`, `requestIDKey{}`                                                                                    |
| All exported functions have doc comments                   | Verified every `func [A-Z]` and `type [A-Z]`                                                                         |
| Middleware type alias used consistently                    | `type Middleware func(http.Handler) http.Handler` in all signatures                                                  |
| 110 tests, 15 benchmarks, 5 fuzz tests, 11 examples        | All passing                                                                                                          |
| 89.1% coverage                                             | Up from 87.1% at start of sprint                                                                                     |
| 0 lint issues (~70 linters)                                | `golangci-lint run`                                                                                                  |
| 0 race conditions detected by test suite                   | `go test -race`                                                                                                      |
| DOMAIN_LANGUAGE.md — all 10 bounded contexts               | Up from 4 at start of sprint                                                                                         |
| doc.go — mentions middleware count, Chain(), Validate()    | Updated                                                                                                              |
| CI — govulncheck on every push, pinned golangci-lint v2.12 | ci.yml                                                                                                               |
| CHANGELOG.md — has [Unreleased] header                     | Fixed during sprint                                                                                                  |
| v0.1.0 tag pushed                                          | `5a67945`                                                                                                            |

---

## B. PARTIALLY DONE ⚠️

### B1. CHANGELOG.md [0.1.0] — Stale Metrics + Missing Validate()

The [0.1.0] section says "94 tests, 11 example functions, 15 benchmarks, 5 fuzz tests" and "87.1% test coverage". These numbers are from the start of the sprint. Actual: 110 tests, 89.1% coverage.

Also missing: `RequestIDConfig.Validate()`, `ETagConfig.Validate()`, `SecurityHeadersConfig.Validate()` are not mentioned anywhere in the [0.1.0] entry, even though they were added before the tag.

### B2. AGENTS.md — Missing 2 Error Codes

The `errors.go` row lists only 5 of 7 error codes: missing `ErrCodeCompressWriteFailed` and `ErrCodeETagWriteFailed`.

### B3. Coverage Gaps — `errors.go` at 0%

`RegisterErrorClassifications()`, `registerAllErrorTemplates()`, and `registerErrorTemplate()` have 0% coverage. These are startup registration functions that aren't called in tests (only `errors_test.go` tests error wrapping, not registration). Low risk but creates a coverage floor.

### B4. `ResponseRecorder.Hijack()` at 42.9%

The successful Hijack path (where underlying writer IS a Hijacker) is covered in integration tests, but the `ResponseRecorder.Hijack()` direct test only covers the unsupported case.

---

## C. NOT STARTED ❌

All items from TODO_LIST.md "Not Started (v0.2.0+)" section:

- Improve test coverage to 90%+
- Make content-type filtering configurable via `CompressionConfig`
- Add `MiddlewareStack` type with ordering validation
- Add `ResponseWriter` capability interface for Hijack/Push/Flush
- Implement deflate support
- Accept-Encoding quality value parsing
- Streaming ETag evaluation
- Rate-limiting, metrics, body-size-limit middlewares

---

## D. TOTALLY FUCKED UP 💥

### D1. 🔴 CRITICAL: Data Race in `CORS()` — `cors.go:75-86`

```go
func CORS(cfg CORSConfig) Middleware {
    allowOrigin := "*"          // ← shared mutable closure variable

    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
            origin := req.Header.Get("Origin")
            if origin != "" {
                allowOrigin = resolveOrigin(origin, cfg)  // ← WRITE from goroutine A
            }
            resp.Header().Set("Access-Control-Allow-Origin", allowOrigin)  // ← READ from goroutine B
```

**This is a data race.** Under concurrent requests:

1. Goroutine A calls `allowOrigin = "https://evil.com"`
2. Goroutine B reads `allowOrigin` and gets `"https://evil.com"` instead of the correct origin

**Impact:** A user could receive the wrong `Access-Control-Allow-Origin` header, potentially allowing cross-origin requests from unauthorized origins. This is a **security vulnerability**.

**Fix:** Move `allowOrigin` inside the per-request closure:

```go
func CORS(cfg CORSConfig) Middleware {
    allowCredentials := "false"
    if cfg.AllowCredentials {
        allowCredentials = "true"
    }

    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
            allowOrigin := "*"  // ← per-request, no sharing
            origin := req.Header.Get("Origin")
            if origin != "" {
                allowOrigin = resolveOrigin(origin, cfg)
            }
            ...
```

**Why wasn't it caught?** `go test -race` only detects races that manifest during execution. No test sends concurrent requests with different origins to the same CORS middleware instance.

**Note:** The AGENTS.md already mentions this was fixed: "Fix data race in `getGzipPool()` (added `sync.RWMutex`)" — but that was a different race. The CORS race was never caught.

### D2. 🟠 HIGH: Compression Pool Could Store Nil `*gzip.Writer`

`compression.go:54-57`:

```go
New: func() any {
    gz, _ := gzip.NewWriterLevel(io.Discard, level)  // ← error discarded
    return gz  // ← could be nil
},
```

If `level` is invalid (despite `Validate()`), `gzip.NewWriterLevel` returns `(nil, error)`. The nil gets stored in the pool. Later, `gz.Reset()` on line 322 would nil-pointer panic.

**Fix:** Panic in `New` if error is non-nil (this should never happen after Validate, so a panic is appropriate):

```go
New: func() any {
    gz, err := gzip.NewWriterLevel(io.Discard, level)
    if err != nil {
        panic(fmt.Sprintf("gzip.NewWriterLevel(%d): %v", level, err))
    }
    return gz
},
```

### D3. 🟡 MEDIUM: CHANGELOG.md Stale Metrics

"94 tests, 11 example functions, 15 benchmarks, 5 fuzz tests" and "87.1% test coverage" in the [0.1.0] entry are wrong. Actual: 110 tests, 89.1% coverage. This was tagged and pushed with wrong numbers.

---

## E. WHAT WE SHOULD IMPROVE

### E1. Critical Fixes

| #   | Fix                                                                | Severity    | Effort |
| --- | ------------------------------------------------------------------ | ----------- | ------ |
| 1   | Fix CORS data race — move `allowOrigin` inside per-request closure | 🔴 Critical | 5min   |
| 2   | Add concurrent CORS test to prove the fix                          | 🔴 Critical | 5min   |
| 3   | Fix compression pool nil guard                                     | 🟠 High     | 3min   |

### E2. Doc Accuracy

| #   | Fix                                                                    | Severity  | Effort |
| --- | ---------------------------------------------------------------------- | --------- | ------ |
| 4   | CHANGELOG.md — update to 110 tests, 89.1%, add 3 new Validate()        | 🟡 Medium | 3min   |
| 5   | AGENTS.md — add `ErrCodeCompressWriteFailed`, `ErrCodeETagWriteFailed` | 🟡 Medium | 2min   |

### E3. Coverage Improvements

| #   | Fix                                                  | Severity | Effort |
| --- | ---------------------------------------------------- | -------- | ------ |
| 6   | Test `RegisterErrorClassifications()` (0% → covered) | 🟢 Low   | 5min   |
| 7   | Test `ResponseRecorder.Hijack()` success path        | 🟢 Low   | 5min   |
| 8   | Remove unreachable `errPoolTypeMismatch` path        | 🟢 Low   | 5min   |

### E4. Architecture

| #   | Consideration                                                              | Priority                                                |
| --- | -------------------------------------------------------------------------- | ------------------------------------------------------- |
| 9   | `responseWrapper.Write()` — add defensive `writeHeaderToUnderlying()` call | Low — safe as-is because all consumers override Write() |
| 10  | `SecurityHeadersConfig.Validate()` — add FrameOptions enum validation      | Low — no-op is fine for v0.1.0                          |
| 11  | Add race condition test pattern to testutil_test.go                        | Medium — prevents future races                          |

---

## F. TOP 25 THINGS TO DO NEXT

Sorted by **impact × urgency / effort**.

| #   | Task                                                                | Category    | Impact      | Effort   | Rationale                                           |
| --- | ------------------------------------------------------------------- | ----------- | ----------- | -------- | --------------------------------------------------- |
| 1   | Fix CORS data race — move `allowOrigin` inside closure              | Bug         | 🔴 Critical | 5min     | Security vulnerability — wrong origin could be sent |
| 2   | Add concurrent CORS test (multiple goroutines, different origins)   | Test        | 🔴 Critical | 5min     | Proves the fix works                                |
| 3   | Fix compression pool nil guard — panic on gzip.NewWriterLevel error | Bug         | 🟠 High     | 3min     | Prevents nil pointer panic in edge case             |
| 4   | Update CHANGELOG.md [0.1.0] — 110 tests, 89.1%, add 3 Validate()    | Doc         | 🟡 Medium   | 3min     | Tagged with wrong numbers                           |
| 5   | Update AGENTS.md errors.go row — add 2 missing error codes          | Doc         | 🟡 Medium   | 2min     | Architecture reference incomplete                   |
| 6   | Add test for `RegisterErrorClassifications()`                       | Test        | 🟢 Low      | 5min     | errors.go at 0% coverage                            |
| 7   | Add `ResponseRecorder.Hijack()` success path test                   | Test        | 🟢 Low      | 5min     | 42.9% → higher                                      |
| 8   | Remove unreachable `errPoolTypeMismatch` sentinel and branch        | Refactor    | 🟢 Low      | 5min     | Dead code creates coverage noise                    |
| 9   | Add race-condition test helper to testutil_test.go                  | Infra       | 🟢 Low      | 5min     | Prevents future races like CORS                     |
| 10  | Improve test coverage to 90%+                                       | Test        | 🟢 Low      | 30min    | Currently 89.1%                                     |
| 11  | Make content-type filtering configurable via `CompressionConfig`    | Feature     | Low         | 20min    | TODO_LIST.md — v0.2.0                               |
| 12  | Add `MiddlewareStack` type with ordering validation                 | Feature     | Low         | 30min    | TODO_LIST.md — v0.2.0                               |
| 13  | Add `ResponseWriter` capability interface                           | Feature     | Low         | 20min    | TODO_LIST.md — v0.2.0                               |
| 14  | Implement deflate support                                           | Feature     | Low         | 30min+   | TODO_LIST.md — v0.2.0                               |
| 15  | Accept-Encoding quality value parsing                               | Feature     | Low         | 30min+   | TODO_LIST.md — v0.2.0                               |
| 16  | Evaluate streaming ETag with rolling hash                           | Research    | Low         | Research | TODO_LIST.md — v0.2.0                               |
| 17  | Consider rate-limiting middleware                                   | Feature     | Low         | Research | TODO_LIST.md — v0.2.0                               |
| 18  | Consider request/response metrics middleware                        | Feature     | Low         | Research | TODO_LIST.md — v0.2.0                               |
| 19  | Consider request body size limit middleware                         | Feature     | Low         | Research | TODO_LIST.md — v0.2.0                               |
| 20  | Add `SecurityHeadersConfig.Validate()` — FrameOptions enum          | Enhancement | Low         | 5min     | Currently no-op                                     |
| 21  | Add `responseWrapper.Write()` defensive fallback                    | Enhancement | Low         | 5min     | Safety net if consumers forget to override          |
| 22  | Add integration test for `Chain(Recovery, Timeout, Logging)`        | Test        | Low         | 5min     | Common stack                                        |
| 23  | Pin govulncheck version in CI (not @latest)                         | Infra       | Low         | 2min     | Reproducibility                                     |
| 24  | Add `ExampleChain` example function                                 | Doc         | Low         | 3min     | Shows Chain() usage in godoc                        |
| 25  | Consider adding a `ROADMAP.md`                                      | Doc         | Low         | 10min    | Long-term vision doc                                |

---

## G. TOP QUESTION I CANNOT FIGURE OUT MYSELF

**#1: The CORS race condition — was this intentional?**

The `allowOrigin` variable was placed OUTSIDE the per-request closure on line 75. `allowCredentials` is similarly placed but is never mutated (it's set once based on config). Was `allowOrigin` intentionally placed outside the closure as an optimization (avoid re-resolving on every request), or was this an oversight?

If intentional, the correct fix is a `sync.RWMutex` or per-request copy. If unintentional, the fix is simply moving it inside the closure. The AGENTS.md says "CORS closure captures `allowOrigin` before the per-request handler" — suggesting this was intentional, but the security implication (wrong origin sent to wrong client) seems like a clear bug regardless of intent.

**My recommendation:** Move inside. The performance cost of calling `resolveOrigin()` per request is negligible (string comparison against a small slice).

---

## Raw Metrics

| Metric                   | Value                                                                 |
| ------------------------ | --------------------------------------------------------------------- |
| Tests                    | 110                                                                   |
| Examples                 | 11                                                                    |
| Benchmarks               | 15                                                                    |
| Fuzz tests               | 5                                                                     |
| Coverage                 | 89.1%                                                                 |
| Lint issues              | 0                                                                     |
| Race conditions detected | 0 (but 1 real race exists — see D1)                                   |
| Data races in production | 1 (CORS)                                                              |
| Doc inaccuracies         | 3 (CHANGELOG metrics, CHANGELOG missing Validate, AGENTS error codes) |

---

_Generated by Crush on 2026-06-08_
