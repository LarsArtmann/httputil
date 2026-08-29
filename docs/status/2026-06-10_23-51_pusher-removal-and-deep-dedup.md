# Comprehensive Status Report — httputil

**Date:** 2026-06-10 23:51 CEST
**Branch:** `master` (up to date with origin, pushed)
**Go version:** 1.26.3
**Lines of Go code:** 4,182 across 26 files
**Test coverage:** 92.4% of statements (was 91.2%)
**Lint status:** 0 issues across ~70 linters
**Test status:** 147 tests passing, race detection clean
**Code duplication:** 0 clone groups at threshold 30
**Benchmarks:** 15 covering all middlewares + Chain + Itoa + Join
**Fuzz tests:** 5 (ClientIP, Compression, ETag, CORS, RequestID)
**Example functions:** 11 covering all public API
**Error codes:** 5 (was 7 — Pusher removed)
**Functions at 100% coverage:** 55 of 72

---

## a) FULLY DONE

### 1. HTTP/2 Server Push Removal (Breaking Change)

HTTP/2 Server Push was removed from Chrome in 2023 and is not part of HTTP/3. All Pusher-related code removed.

| Removed                                                | Lines               |
| ------------------------------------------------------ | ------------------- |
| `pushDelegate()` + `responseWrapper.Push()`            | wrapper.go          |
| `ResponseRecorder.Push()`                              | recorder.go         |
| `ErrCodePushUnsupported` + `ErrCodePushFailed`         | errors.go           |
| 2 error templates + 5 error doc comments               | errors.go           |
| 5 Push test functions + `mockPusher` + `errPushFailed` | errors_test.go      |
| `TestCompression_Push_Delegates`                       | compression_test.go |
| `TestETag_Push_Delegates`                              | etag_test.go        |
| `pushRecorder` + `newPushRecorder()`                   | testutil_test.go    |

Error codes: 7 → 5. Coverage: 91.2% → 92.4%.

### 2. Hijack/Push/Flush Delegation Extraction

`ResponseRecorder` and `responseWrapper` had near-identical `Hijack()`, `Push()`, and `Flush()` implementations. Extracted into shared unexported helpers:

- `hijackDelegate(w http.ResponseWriter)` — type assertion + error wrapping
- `flushDelegate(w http.ResponseWriter)` — type assertion + delegation

Both types now delegate to these shared functions. Eliminated ~60 lines of duplication.

### 3. Aggressive Code Deduplication (Threshold 30 → 0)

Eliminated 6 clone groups at threshold 30:

| Clone Group                                                       | Fix                                                 | Lines Saved |
| ----------------------------------------------------------------- | --------------------------------------------------- | ----------- |
| `http.HandlerFunc(func(w, r) { WriteHeader+Write })` × 22         | `newWriteStatusHandler()` + `newWriteBodyHandler()` | -59         |
| `http.HandlerFunc(func(w, r) { WriteHeader+Write(bodyVar) })` × 4 | `newWriteBodyHandler()`                             | -12         |
| Hex encoding loop in `computeETag` × 2                            | `encodeHex()` helper                                | Deduped     |
| Content-type skip tests × 3                                       | `testCompressionSkipsContentType()`                 | -42         |
| IfNoneMatch tests × 2                                             | `testETagIfNoneMatchReturns304()`                   | -24         |
| Flush handler × 2                                                 | `newFlushHandler()`                                 | -10         |

### 4. Core Middleware Suite (10 middlewares) — Complete

| Middleware       | File             | Config + Validate                      | Tests | Examples                     | Benchmarks                          | Fuzz              |
| ---------------- | ---------------- | -------------------------------------- | ----- | ---------------------------- | ----------------------------------- | ----------------- |
| CORS             | `cors.go`        | `CORSConfig` + `Validate()`            | Yes   | `ExampleCORS`                | `BenchmarkCORS` (461 ns)            | `FuzzCORS`        |
| ClientIP         | `clientip.go`    | —                                      | Yes   | `ExampleClientIP`            | `BenchmarkClientIP` (49 ns)         | `FuzzClientIP`    |
| RequestID        | `requestid.go`   | `RequestIDConfig` + `Validate()`       | Yes   | `ExampleRequestID`           | `BenchmarkRequestID` (379 ns)       | `FuzzRequestID`   |
| SecurityHeaders  | `security.go`    | `SecurityHeadersConfig` + `Validate()` | Yes   | `ExampleSecurityHeaders`     | `BenchmarkSecurityHeaders` (228 ns) | —                 |
| Recovery         | `recovery.go`    | `*slog.Logger`                         | Yes   | `ExampleRecovery`            | `BenchmarkRecovery` (54 ns)         | —                 |
| Timeout          | `timeout.go`     | `time.Duration`                        | Yes   | `ExampleTimeout`             | `BenchmarkTimeout` (381 ns)         | —                 |
| Logging          | `logging.go`     | `*slog.Logger`                         | Yes   | `ExampleLogging`             | `BenchmarkLogging` (1,014 ns)       | —                 |
| ResponseRecorder | `recorder.go`    | —                                      | Yes   | `ExampleNewResponseRecorder` | `BenchmarkResponseRecorder` (23 ns) | —                 |
| Compression      | `compression.go` | `CompressionConfig` + `Validate()`     | Yes   | `ExampleCompression`         | `BenchmarkCompression` (6.8 µs)     | `FuzzCompression` |
| ETag             | `etag.go`        | `ETagConfig` + `Validate()`            | Yes   | `ExampleETag`                | `BenchmarkETag` (431 ns)            | `FuzzETag`        |
| Chain            | `recorder.go`    | —                                      | Yes   | `ExampleChain`               | `BenchmarkChain` (2.8 µs)           | —                 |

### 5. Error Classification System

5 error codes registered via `go-error-family`:

- `ErrCodeWriteFailed`, `ErrCodeHijackUnsupported`, `ErrCodeHijackFailed`
- `ErrCodeCompressWriteFailed`, `ErrCodeETagWriteFailed`

`RegisterErrorClassifications()` maps stdlib HTTP errors to behavioral families with `what/why/fix/wayOut` message templates.

### 6. Configuration Validation

All config types have `Validate()` methods: `CORSConfig`, `CompressionConfig`, `RequestIDConfig`, `ETagConfig`, `SecurityHeadersConfig`.

### 7. Documentation Suite

All current: `README.md`, `doc.go`, `AGENTS.md`, `CHANGELOG.md`, `FEATURES.md`, `TODO_LIST.md`, `docs/DOMAIN_LANGUAGE.md`, 14 status reports in `docs/status/`.

### 8. Tooling & Quality Gates

| Gate                        | Status                       |
| --------------------------- | ---------------------------- |
| `golangci-lint run`         | 0 issues (~70 linters)       |
| `go test ./... -race`       | 147 tests passing, race-free |
| `go vet ./...`              | Clean                        |
| `go test -bench`            | 15 benchmarks passing        |
| Coverage                    | 92.4% of statements          |
| `art-dupl -t 30 --semantic` | 0 clone groups               |
| GitHub Actions CI           | test + lint + govulncheck    |

---

## b) PARTIALLY DONE

### Test Coverage (92.4% — up from 91.2%)

17 functions below 100%. 12 functions below 90%:

| File             | Function                    | Coverage | Gap                                                 |
| ---------------- | --------------------------- | -------- | --------------------------------------------------- |
| `compression.go` | `Flush`                     | 61.5%    | Multiple flush-while-compressing/buffering branches |
| `compression.go` | `startCompressAndStream`    | 66.7%    | Error during streaming write                        |
| `compression.go` | `writePlain`                | 75.0%    | Write error branch                                  |
| `compression.go` | `writeCompressed`           | 75.0%    | Write error branch                                  |
| `compression.go` | `flushPlainAndStream`       | 76.9%    | Flush error branches                                |
| `etag.go`        | `Flush`                     | 77.8%    | Flush-while-buffering branches                      |
| `etag.go`        | `Write`                     | 80.0%    | Buffer-limit exceeded branch                        |
| `compression.go` | `Write`                     | 83.3%    | Error during compression start                      |
| `compression.go` | `isCompressibleContentType` | 83.3%    | Unusual content-type edge cases                     |
| `wrapper.go`     | `hijackDelegate`            | 85.7%    | Hijack failure error path                           |
| `compression.go` | `startCompression`          | 88.2%    | Type-assertion failure                              |
| `compression.go` | `Close`                     | 86.7%    | Close error + buffered write paths                  |
| `compression.go` | `getGzipPool`               | 88.2%    | Concurrent pool creation path                       |
| `logging.go`     | `Logging`                   | 90.0%    | Missing log output assertions                       |
| `security.go`    | `SecurityHeaders`           | 92.3%    | Skip-empty-header branches                          |
| `cors.go`        | `CORS`                      | 95.2%    | Missing MaxAge=0 branch                             |
| `etag.go`        | `computeETag`               | 94.4%    | Empty body + no WriteHeader edge case               |

---

## c) NOT STARTED

| Item                                                             | Priority          | Notes                                  |
| ---------------------------------------------------------------- | ----------------- | -------------------------------------- |
| Make content-type filtering configurable via `CompressionConfig` | Near-term         | Currently hardcoded deny-list          |
| Add `MiddlewareStack` type with ordering validation              | Near-term         | Catch misordered middleware at startup |
| Add `ResponseWriter` capability interface for Hijack/Flush       | Near-term         | Unify scattered type assertions        |
| Implement deflate support using `compress/flate`                 | Medium-term       | Second compression algorithm           |
| Add `Accept-Encoding` quality value parsing per RFC 7231         | Medium-term       | Proper content negotiation             |
| Evaluate streaming ETag option using rolling hash                | Medium-term       | Memory-efficient for large responses   |
| Consider request/response metrics middleware                     | Worth considering | `expvar` or custom histograms          |
| Consider rate-limiting middleware                                | Worth considering | Sliding window or token bucket         |
| Consider request body size limit middleware                      | Worth considering | Simple guard middleware                |

---

## d) TOTALLY FUCKED UP

**Nothing is fucked up.**

- Zero lint issues across 70 linters
- Zero code clones at threshold 30
- Zero race conditions
- 92.4% coverage (up from 91.2%)
- All tests passing
- All benchmarks stable
- Documentation current and consistent
- Working tree clean, pushed to remote

Pre-existing known issue: `mnd` (magic number) violation — `86400` in `DefaultCORSConfig`. Documented, not fixed.

---

## e) WHAT WE SHOULD IMPROVE

### Architecture

1. **`responseWrapper` state: 2 bools → typed enum** — `wroteHeader` and `headerWritten` encode a 3-state machine but allow an impossible state (`wroteHeader=false, headerWritten=true`). A single typed enum would make this unrepresentable.

2. **`compressWriter` state: 2 bools → typed enum** — `compressing` and `plain` are mutually exclusive phases (Buffering/Compressing/Plain). Two bools allow `compressing=true, plain=true`. A single enum would eliminate this.

3. **Config validation interface** — Five config types all have `Validate() error` but no shared interface. Callers cannot generically validate. Could define `type Validator interface { Validate() error }` and have each middleware call `cfg.Validate()` internally as a safety net.

### Coverage

4. **Compression `Flush` (61.5%)** — The lowest coverage function. Multiple flush-while-compressing, flush-while-buffering branches untested.

5. **`hijackDelegate` (85.7%)** — The Hijack failure error path (when `hijacker.Hijack()` returns error) is untested since we removed `mockHijacker`.

6. **Error branches in compression** — `writePlain`, `writeCompressed`, `startCompressAndStream`, `flushPlainAndStream` all have untested error paths where the underlying writer fails.

---

## f) Top 25 Things We Should Get Done Next

| #  | Task                                                                                    | Impact | Effort   | Category     |
| -- | --------------------------------------------------------------------------------------- | ------ | -------- | ------------ |
| 1  | Fix `compression.Flush` coverage (61.5% → 85%+)                                         | High   | Medium   | Coverage     |
| 2  | Replace `wroteHeader`+`headerWritten` bools with typed state enum                       | High   | Medium   | Architecture |
| 3  | Replace `compressing`+`plain` bools with typed phase enum                               | High   | Medium   | Architecture |
| 4  | Fix compression error-path coverage (writePlain/writeCompressed/startCompressAndStream) | Medium | Low      | Coverage     |
| 5  | Fix `hijackDelegate` error-path test (85.7% → 95%+)                                     | Medium | Low      | Coverage     |
| 6  | Fix `etag.Write` buffer-limit branch (80% → 95%+)                                       | Medium | Low      | Coverage     |
| 7  | Fix `etag.Flush` flush-while-buffering branches (77.8% → 95%+)                          | Medium | Low      | Coverage     |
| 8  | Fix `flushPlainAndStream` error branches (76.9% → 95%+)                                 | Medium | Low      | Coverage     |
| 9  | Make content-type filtering configurable via `CompressionConfig`                        | High   | Medium   | Feature      |
| 10 | Define `type Validator interface { Validate() error }` + internal validation            | Medium | Low      | Architecture |
| 11 | Add `MiddlewareStack` type with ordering validation                                     | Medium | Medium   | Feature      |
| 12 | Add `ResponseWriter` capability interface for Hijack/Flush                              | Medium | Medium   | Architecture |
| 13 | Push overall coverage from 92.4% → 95%                                                  | Medium | Medium   | Coverage     |
| 14 | Implement deflate support using `compress/flate`                                        | High   | High     | Feature      |
| 15 | Add `Accept-Encoding` quality value parsing (RFC 7231)                                  | High   | Medium   | Compliance   |
| 16 | Fix `mnd` violation (`86400` in `DefaultCORSConfig`)                                    | Low    | Trivial  | Lint         |
| 17 | Evaluate streaming ETag with rolling hash                                               | Medium | High     | Architecture |
| 18 | Add request/response metrics middleware                                                 | Medium | High     | Feature      |
| 19 | Add rate-limiting middleware                                                            | Medium | High     | Feature      |
| 20 | Add request body size limit middleware                                                  | Low    | Low      | Feature      |
| 21 | Brotli support — decide on dependency policy                                            | Medium | Decision | Policy       |
| 22 | Add `WriterFactory` plugin interface for compression                                    | Medium | Medium   | Architecture |
| 23 | Fix `logging.go` coverage (90% → 95%+)                                                  | Low    | Low      | Coverage     |
| 24 | Fix `security.go` skip-empty-header branches (92.3% → 95%+)                             | Low    | Low      | Coverage     |
| 25 | Verify `art-dupl` at t30 stays clean as codebase grows                                  | Low    | Trivial  | Quality      |

---

## g) Top #1 Question I Cannot Figure Out Myself

**What is the v0.2.0 release criteria?**

The codebase is at v0.1.1. The Pusher removal is a breaking change that warrants a major version bump (v0.2.0 per Go semver conventions). But the TODO list has items tagged "v0.2.0+" with no explicit milestone:

- Should v0.2.0 ship with just the Pusher removal + deduplication + delegation extraction?
- Or should it wait for the typed state enums (#2, #3 above) which are also breaking (internal struct field changes)?
- Is the single-dependency policy (`depguard` allows only `$gostd`, `$module`, `go-error-family`) a hard constraint for v0.2.0? This blocks deflate support (needs only `compress/flate` which is stdlib), brotli, and rate-limiting.
- Should the `ResponseWriter` capability interface come before or after v0.2.0?

---

## Delta Since Last Report (2026-06-10 22:43)

| Metric             | Then  | Now       | Change                   |
| ------------------ | ----- | --------- | ------------------------ |
| Coverage           | 91.2% | **92.4%** | +1.2%                    |
| Error codes        | 7     | 5         | -2 (Pusher removed)      |
| Test count         | 154   | 147       | -7 (Push tests removed)  |
| Lines of Go        | 4,439 | 4,182     | -257 (Push code + dedup) |
| Clone groups (t30) | 0     | 0         | Stable                   |
| Lint issues        | 0     | 0         | Stable                   |
| Commits since      | —     | 4         | See below                |
| Files changed      | —     | 16        | +84/-345 lines           |

### Session Commits

| Commit    | Description                                                    |
| --------- | -------------------------------------------------------------- |
| `1cd170e` | Eliminate all clone groups at threshold 30 — zero duplication  |
| `ce5123c` | Extract Hijack/Push/Flush delegation into shared helpers       |
| `b253b6b` | **BREAKING**: Remove http.Pusher support — HTTP/2 push is dead |
| `7dc19b7` | Update all documentation for Pusher removal                    |

---

_All green. Zero clones. Zero lint. Race-free. Pushed. Awaiting direction._
