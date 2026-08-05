# Comprehensive Status Report — httputil v0.4.0 Candidate

**Date:** 2026-07-02 03:50 CEST
**Author:** Crush session (continuation of multi-session effort)
**Branch:** master (uncommitted working tree)

---

## Executive Summary

The project has grown from 10 middleware + 13-spec httpspec to **13 middleware + 18-spec httpspec + 3 infrastructure types**, all passing build, lint (0 issues across ~70 linters), and tests (266 tests, race-clean). Coverage holds at 92.1% (main) / 98.3% (httpspec). The work is **ready to commit and tag as v0.4.0** once documentation is finalized.

### Quality Gates

| Gate                           | Status      | Detail                                                       |
| ------------------------------ | ----------- | ------------------------------------------------------------ |
| `go build ./...`               | ✅ PASS     | Both packages compile                                        |
| `golangci-lint run ./...`      | ✅ 0 issues | ~70 linters, includes race-safe fmt                          |
| `go test ./... -count=1 -race` | ✅ PASS     | 266 tests, 0 failures                                        |
| Coverage (main)                | 92.1%       | Down from 92.5% — new middleware has untested error branches |
| Coverage (httpspec)            | 98.3%       | Up from 97.9%                                                |
| `go vet ./...`                 | ✅ Clean    |                                                              |

---

## a) FULLY DONE — Completed and Verified

### A1. httpspec: 5 new standard specs (13 → 18)

| Spec                           | Category | File:Line           | Tests                            |
| ------------------------------ | -------- | ------------------- | -------------------------------- |
| `SpecNameXContentTypeOptions`  | Security | `httpspec/specs.go` | 2 (pass + fail)                  |
| `SpecNameNoDuplicateHeaders`   | Headers  | `httpspec/specs.go` | 2 (pass + fail)                  |
| `SpecNameConnectRejected`      | Security | `httpspec/specs.go` | 3 (404 pass, 405 pass, 200 fail) |
| `SpecNameRespectsAcceptHeader` | Headers  | `httpspec/specs.go` | 2 (pass + 500 fail)              |
| `SpecNameLongURLHandled`       | Routing  | `httpspec/specs.go` | 2 (pass + 500 fail)              |

All wired into `routingSpecs()`, `methodSpecs()`, `headerSpecs()`, `securitySpecs()`. `TestStandardSpecsHasAllExpectedNames` updated to validate all 18 are present.

### A2. httpspec: New builders, RunSerial, examples, benchmarks

| Item                               | File                         | Status                                                                        |
| ---------------------------------- | ---------------------------- | ----------------------------------------------------------------------------- |
| `ExpectNotStatus` builder          | `httpspec/httpspec.go:151`   | ✅ 2 tests                                                                    |
| `RunSerial` function               | `httpspec/httpspec.go:226`   | ✅ 2 tests                                                                    |
| `example_test.go` (7 examples)     | `httpspec/example_test.go`   | ✅ All have `// Output:`                                                      |
| `benchmark_test.go` (2 benchmarks) | `httpspec/benchmark_test.go` | ✅ `BenchmarkCheck` (sub-benchmarks per check), `BenchmarkCheckServesRequest` |

### A3. MaxBodySize middleware

| Aspect         | Detail                                           |
| -------------- | ------------------------------------------------ |
| File           | `maxbodysize.go`                                 |
| API            | `MaxBodySize(maxBytes int64) Middleware`         |
| Implementation | Wraps `r.Body` with `http.MaxBytesReader`        |
| Tests          | 3: normal request, oversized rejection, nil body |
| Coverage       | 100%                                             |

### A4. RateLimit middleware + TokenBucketLimiter

| Aspect          | Detail                                                                                                                   |
| --------------- | ------------------------------------------------------------------------------------------------------------------------ |
| File            | `ratelimit.go`                                                                                                           |
| API             | `RateLimit(cfg RateLimitConfig) Middleware`, `NewTokenBucketLimiter(rate, burst)`, `RateLimiter` interface               |
| Config validate | `RateLimitConfig.Validate()` returns error if `Limiter == nil`                                                           |
| Custom hooks    | `KeyFunc` (per-key extraction), `OnDenied` (custom 429 handler), `Status`                                                |
| Tests           | 7: burst allow/deny, independent keys, within-limit, exceed-denied, custom key func, custom on-denied JSON, validate nil |
| Coverage        | `Allow` 100%, `RateLimit` 94.7%, `Validate` 66.7%                                                                        |

### A5. Metrics middleware + MetricsRecorder interface

| Aspect          | Detail                                                                                         |
| --------------- | ---------------------------------------------------------------------------------------------- |
| File            | `metrics.go`                                                                                   |
| API             | `Metrics(cfg MetricsConfig) Middleware`, `MetricsRecorder` interface                           |
| Config validate | `MetricsConfig.Validate()` returns error if `Recorder == nil`                                  |
| Features        | Records method, path, status (normalizes 0→200), duration. Uses `ResponseRecorder` internally. |
| Tests           | 3: records request data, normalizes status zero, validate nil                                  |
| Coverage        | `Metrics` 100%, `Validate` 66.7%                                                               |

### A6. MiddlewareStack type

| Aspect               | Detail                                                                                                          |
| -------------------- | --------------------------------------------------------------------------------------------------------------- |
| File                 | `stack.go`                                                                                                      |
| API                  | `NewMiddlewareStack()`, `Add(name, mw)`, `Names()`, `Validate()`, `Build(handler)`                              |
| Ordering rule        | Recovery must be position 0 (outermost)                                                                         |
| Duplicate prevention | `Add` returns `errDuplicateMiddleware`                                                                          |
| Well-known names     | `MiddlewareRecovery`, `MiddlewareCORS`, `MiddlewareCompression`, `MiddlewareETag`, etc. (9 constants)           |
| Tests                | 6: add+build order, duplicate rejection, empty validate, recovery-first passes, recovery-not-first fails, names |
| Coverage             | 100% across all methods                                                                                         |

### A7. Capabilities detection

| Aspect   | Detail                                                                            |
| -------- | --------------------------------------------------------------------------------- |
| File     | `capabilities.go`                                                                 |
| API      | `DetectCapabilities(w) Capabilities`, `Capabilities{Hijacker bool, Flusher bool}` |
| Tests    | 3: recorder has Flusher only, bare writer has neither, through responseWrapper    |
| Coverage | 100%                                                                              |

### A8. Configurable content-type filtering for Compression

| Aspect              | Detail                                                                                                         |
| ------------------- | -------------------------------------------------------------------------------------------------------------- |
| Files               | `compress_content_type.go`, `compression.go`, `compress_writer.go`                                             |
| Change              | `CompressionConfig.IncompressibleTypes []string` field added; `DefaultIncompressibleTypes()` exported          |
| Behavior            | nil → defaults, empty → compress everything, custom → user list                                                |
| Wiring              | `skipTypes` passed through `newCompressWriter` → `shouldCompress` → `isCompressibleContentType(ct, skipTypes)` |
| Tests               | 3: custom list skips text/, empty compresses image/, nil uses defaults                                         |
| Coverage            | `isCompressibleContentType` 100%, `DefaultIncompressibleTypes` 100%                                            |
| Backward compatible | ✅ Existing callers using `DefaultCompressionConfig()` get identical behavior                                  |

### A9. Documentation updates (docs files)

| File              | Status                                                                           |
| ----------------- | -------------------------------------------------------------------------------- |
| `httpspec/doc.go` | ✅ Updated: 18 specs, RunSerial, ExpectNotStatus, quick-start                    |
| `CHANGELOG.md`    | ✅ `[Unreleased]` expanded with all new features                                 |
| `FEATURES.md`     | ✅ 13 middlewares table, 18 specs, infrastructure types, streaming ETag rejected |
| `TODO_LIST.md`    | ✅ All items marked done                                                         |
| `README.md`       | ✅ Behavioral Spec Suite section updated (18 specs, RunSerial)                   |

---

## b) PARTIALLY DONE — Needs More Work

### B1. AGENTS.md not yet updated

The AGENTS.md file table does not include the 6 new files (`maxbodysize.go`, `ratelimit.go`, `metrics.go`, `stack.go`, `capabilities.go`, `compress_content_type.go` changes). The httpspec table still says 13 specs. The lint section and architecture notes need updating for the new config validation patterns.

**Effort:** ~10 min.

### B2. Coverage gaps in new middleware Validate() methods

`RateLimitConfig.Validate()` and `MetricsConfig.Validate()` both at 66.7% — the success path (nil error) is untested. One test each needed.

**Effort:** ~5 min.

### B3. No example_test.go functions for new middleware

The main package has `example_test.go` with 11 examples for the original 10 middleware. The 3 new middleware (MaxBodySize, RateLimit, Metrics) have no `Example*` functions.

**Effort:** ~10 min.

---

## c) NOT STARTED

### C1. v0.4.0 release tag

Not yet created. The `[Unreleased]` section in CHANGELOG.md needs to become `[0.4.0]` with a date, and `git tag v0.4.0` needs to be run. Should be done after AGENTS.md update.

### C2. Streaming ETag (evaluated, rejected)

Investigated during this session. HTTP requires headers before body, so buffering is mandatory. The current CRC-32 + 1MB buffer is correct and optimal. **Marked as rejected in FEATURES.md and TODO_LIST.md.** No further action needed.

---

## d) TOTALLY FUCKED UP — Nothing

No regressions, no broken tests, no lint failures, no data loss. The `httptest.ResponseRecorder` not implementing `http.Hijacker` in Go 1.26 was caught and fixed during the capabilities test (initial assumption was wrong — corrected immediately).

One design observation: the `ratelimit.go` `TokenBucketLimiter` has an unbounded `buckets` map with no eviction. For production use with many unique keys (per-IP rate limiting behind a large proxy), this would leak memory. This is a known limitation worth documenting but not a blocker for a library — the consumer provides the limiter.

---

## e) WHAT WE SHOULD IMPROVE

### Architecture & Design

1. **TokenBucketLimiter has no eviction** — buckets map grows unbounded. Needs TTL or LRU eviction for production use with per-IP keys.
2. **RateLimitConfig and MetricsConfig don't follow the Options pattern** — unlike CORS/Compression/ETag which use `Default*Config()` + struct fields, these use the same pattern but lack `With*` functional options. Inconsistent but acceptable — the struct pattern is already established.
3. **MiddlewareStack only validates Recovery ordering** — could also warn about SecurityHeaders-before-COMPRESSION (security headers should be outermost to ensure they're always set) and Timeout-innermost (timeout should be close to the handler).
4. **No integration test composing all 3 new middleware** — each is tested in isolation but not chained together or with existing middleware.
5. **`httpspec.Run` creates a `testing.T` in benchmarks** — the `BenchmarkRun` was removed in favor of `BenchmarkCheck` which is more correct, but the benchmark approach is still indirect.

### Type Safety

6. **`MiddlewareStack.Add` takes a string name** — could use a typed `MiddlewareName` string type to prevent typos. Currently any string works, which means `MiddlewareRecovery` vs `"Recovery"` (wrong case) silently passes validation.
7. **RateLimitConfig.Status is `int` not `int64`** — minor but inconsistent with `MaxBodySize` which uses `int64`.

### Testing

8. **No fuzz tests for new middleware** — RateLimit's token bucket math and MaxBodySize's boundary behavior are fuzzable.
9. **No benchmarks for new middleware** — MaxBodySize, RateLimit, Metrics have no `Benchmark*` functions.
10. **`runSpecs` coverage at 88.2%** — the nil-handler fatal path is tested but the test doesn't cover the skip path properly.

### Documentation

11. **AGENTS.md is stale** — missing 6 new files, wrong spec count, no mention of config validation patterns for new middleware.
12. **No example functions for new middleware** — the existing `example_test.go` pattern should be extended.

---

## f) Top 25 Things to Do Next

Sorted by impact × 1/effort (highest value first):

| #   | Task                                                                                                                                                                | Impact | Effort | Category   |
| --- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------ | ------ | ---------- |
| 1   | **Update AGENTS.md** — add 6 new files to table, fix spec count 13→18, document config validation                                                                   | High   | 10 min | Docs       |
| 2   | **Add Validate() success-path tests** for RateLimitConfig + MetricsConfig                                                                                           | Med    | 5 min  | Tests      |
| 3   | **Commit all work** — this is a large uncommitted changeset (12 new + 13 modified files)                                                                            | High   | 5 min  | Process    |
| 4   | **Tag v0.4.0** — move `[Unreleased]` → `[0.4.0]`, tag, push                                                                                                         | High   | 5 min  | Release    |
| 5   | **Add example_test.go functions** for MaxBodySize, RateLimit, Metrics                                                                                               | Med    | 10 min | Docs/Tests |
| 6   | **Add benchmarks** for MaxBodySize, RateLimit, Metrics                                                                                                              | Low    | 10 min | Perf       |
| 7   | **Add integration test** chaining Recovery+RateLimit+MaxBodySize+Metrics+handler                                                                                    | Med    | 10 min | Tests      |
| 8   | **Add fuzz test for TokenBucketLimiter.Allow** — rate/burst edge cases                                                                                              | Low    | 10 min | Tests      |
| 9   | **Document TokenBucketLimiter memory limitation** in AGENTS.md or doc comment                                                                                       | Med    | 3 min  | Docs       |
| 10  | **Consider typed `MiddlewareName`** instead of bare string                                                                                                          | Med    | 15 min | Types      |
| 11  | **Add MiddlewareStack ordering rules** — SecurityHeaders should be early, Timeout should be late                                                                    | Med    | 15 min | Design     |
| 12  | **Add `Stack.AddFirst()` method** — for middleware that must be outermost (Recovery)                                                                                | Low    | 10 min | Feature    |
| 13  | **TokenBucketLimiter eviction** — TTL-based bucket cleanup for production per-IP use                                                                                | High   | 30 min | Feature    |
| 14  | **Add `WithRate(r)`, `WithBurst(b)` functional options** to TokenBucketLimiter                                                                                      | Low    | 10 min | API        |
| 15  | **Add X-Frame-Options presence spec** to httpspec                                                                                                                   | Low    | 10 min | Feature    |
| 16  | **Add Strict-Transport-Security presence spec** to httpspec                                                                                                         | Low    | 10 min | Feature    |
| 17  | **Add Content-Security-Policy presence spec** to httpspec                                                                                                           | Low    | 10 min | Feature    |
| 18  | **Add HTTPS redirect spec** to httpspec (verify redirect to HTTPS)                                                                                                  | Low    | 15 min | Feature    |
| 19  | **Add `ExpectJSON` / `ExpectHTML` builders** — verify Content-Type is JSON or HTML                                                                                  | Low    | 10 min | Feature    |
| 20  | **Add `MiddlewareStack.BuildWithValidation()`** — calls Validate() then Build()                                                                                     | Low    | 5 min  | Feature    |
| 21  | **Add per-route RateLimit middleware** — `RateLimitFor(pattern, cfg)` using http.ServeMux patterns                                                                  | Med    | 20 min | Feature    |
| 22  | **Add Prometheus MetricsRecorder implementation** — as a documented example, not a dependency                                                                       | Med    | 20 min | Feature    |
| 23  | **Add `DetectCapabilities` to ResponseRecorder and compressWriter** — verify they correctly report capabilities through wrapping                                    | Low    | 10 min | Tests      |
| 24  | **Document middleware ordering recommendations** in README — Recovery → RateLimit → MaxBodySize → CORS → SecurityHeaders → RequestID → Compression → ETag → handler | Med    | 10 min | Docs       |
| 25  | **Add `go doc` generation CI step** — verify all exported symbols have doc comments                                                                                 | Low    | 10 min | CI         |

---

## g) Top #1 Question I Cannot Figure Out Myself

**Should `TokenBucketLimiter` ship as-is (unbounded map, documented limitation) or should it get eviction before v0.4.0?**

The unbounded `buckets map[string]*tokenBucket` will leak memory in production per-IP rate limiting. But:

- Adding eviction (TTL sweep, background goroutine, or LRU) adds complexity and a `time.Ticker` that needs lifecycle management.
- The library's philosophy is "zero dependencies, stdlib only" — a proper LRU would need `container/list` which is stdlib but adds code.
- The consumer can always implement their own `RateLimiter` with eviction.

I lean toward **shipping as-is with a documented limitation** and adding eviction in v0.5.0, but this is a product judgment call about whether "production-ready" means "safe by default" or "flexible by design."

---

## File Inventory

### New files (12)

| File                         | Purpose                                                        | Lines |
| ---------------------------- | -------------------------------------------------------------- | ----- |
| `capabilities.go`            | `DetectCapabilities()` + `Capabilities` type                   | 22    |
| `capabilities_test.go`       | 3 tests                                                        | 65    |
| `maxbodysize.go`             | `MaxBodySize()` middleware                                     | 18    |
| `maxbodysize_test.go`        | 3 tests                                                        | 70    |
| `ratelimit.go`               | `RateLimit()` + `TokenBucketLimiter` + `RateLimiter` interface | 140   |
| `ratelimit_test.go`          | 7 tests                                                        | 170   |
| `metrics.go`                 | `Metrics()` + `MetricsRecorder` interface                      | 60    |
| `metrics_test.go`            | 3 tests                                                        | 110   |
| `stack.go`                   | `MiddlewareStack` type + 9 well-known name constants           | 97    |
| `stack_test.go`              | 6 tests                                                        | 150   |
| `httpspec/example_test.go`   | 7 runnable examples                                            | 70    |
| `httpspec/benchmark_test.go` | 2 benchmarks (6 sub-benchmarks)                                | 55    |

### Modified files (13)

| File                        | Changes                                                                                                                                                  |
| --------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `compress_content_type.go`  | Refactored: `incompressiblePrefixes` var → `DefaultIncompressibleTypes()` function; `isCompressibleContentType` now takes `skipTypes []string` parameter |
| `compress_writer.go`        | Added `skipTypes []string` field to struct, parameter to constructor, wired into `shouldCompress`                                                        |
| `compress_writer_test.go`   | Added `nil` arg to `newCompressWriter` call                                                                                                              |
| `compression.go`            | Added `IncompressibleTypes []string` field to `CompressionConfig`, wired skipTypes resolution into `Compression()`                                       |
| `compression_test.go`       | Added `nil` arg to `newCompressWriter` call; 3 new tests for configurable filtering                                                                      |
| `httpspec/doc.go`           | Rewritten: 18 specs, RunSerial, ExpectNotStatus, examples                                                                                                |
| `httpspec/httpspec.go`      | Added 5 SpecName constants, `ExpectNotStatus` builder, `RunSerial`, refactored `Run` to share `runSpecs`                                                 |
| `httpspec/httpspec_test.go` | Added X-Content-Type-Options to `newGoodHandler`, updated skip list, added RunSerial tests, added 12 new check tests + handlers                          |
| `httpspec/specs.go`         | Added 5 new check functions, wired into spec builders, `securitySpecs` now takes config param                                                            |
| `CHANGELOG.md`              | Expanded `[Unreleased]` with all new features                                                                                                            |
| `FEATURES.md`               | Updated middleware table (10→13), spec count (13→18), streaming ETag rejected, infrastructure types section                                              |
| `README.md`                 | Updated Behavioral Spec Suite section (13→18 specs, RunSerial)                                                                                           |
| `TODO_LIST.md`              | Marked all items done, streaming ETag rejected with explanation                                                                                          |

---

## Metrics Summary

| Metric                   | Before Session      | After Session       | Delta |
| ------------------------ | ------------------- | ------------------- | ----- |
| Middleware count         | 10                  | 13                  | +3    |
| httpspec standard specs  | 13                  | 18                  | +5    |
| httpspec helper builders | 4                   | 5                   | +1    |
| Test count               | ~230                | 266                 | +36   |
| Main package coverage    | 92.5%               | 92.1%               | -0.4% |
| httpspec coverage        | 97.9%               | 98.3%               | +0.4% |
| Go source files          | ~50                 | 62                  | +12   |
| Lint issues              | 0                   | 0                   | —     |
| Dependencies             | 1 (go-error-family) | 1 (go-error-family) | —     |

---

## Resolution (2026-07-22)

v0.4.0 was tagged (`4f1bb35`) and is on `origin`. Key items from section f) resolved: AGENTS.md was updated, `Validate()` success-path tests added, examples added, `TokenBucketLimiter` eviction shipped as `EvictionTTL` in v0.5.0 (`a44b0b9`), and the rate limiter was subsequently switched to `golang.org/x/time/rate` (`4ce4fdf`). The project is now at v0.5.0 (local tag) with 2 dependencies. The streaming ETag item (C2) remains correctly rejected.

> **Final Resolution (2026-08-05, v0.8.0):** v0.4.0 was tagged and released. Subsequent releases v0.5.0, v0.6.0, v0.6.1 (jsonv2 fix), v0.7.0 (breaking API renames + DenyUnmatched flip), v0.7.1 (coverage closure), and v0.8.0 (CSRF + Server-Timing + KeyedRateLimit) followed. The library is now at v0.8.0 with 16 middlewares, 3 dependencies, 97.8% httputil coverage, 0 lint issues. The `#1 risk` from this report (release discipline) is now codified in `docs/RELEASE.md` with a pre-release self-review step.
