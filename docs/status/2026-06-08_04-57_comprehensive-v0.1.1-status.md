# Comprehensive Status Report — httputil

**Date:** 2026-06-08 04:57 CEST
**Branch:** `master` (up to date with origin)
**Last commit:** `6584b26` docs(changelog): prepare v0.1.1 release
**Go version:** 1.26.3
**Lines of Go code:** 4,481 across 26 files
**Test coverage:** 91.2% of statements
**Lint status:** 0 issues across ~70 linters
**Test status:** 112 tests passing, race detection clean
**Benchmarks:** 15 covering all middlewares + Chain + Itoa + Join
**Fuzz tests:** 5 (ClientIP, Compression, ETag, CORS, RequestID)
**Example functions:** 11 covering all public API
**Nix flake:** `nix flake check` passes (format check only)

---

## a) FULLY DONE

### 1. Core Middleware Suite (10 middlewares) — Complete Coverage

| Middleware       | File             | Config + Validate                      | Tests | Examples                     | Benchmarks                                    | Fuzz              |
| ---------------- | ---------------- | -------------------------------------- | ----- | ---------------------------- | --------------------------------------------- | ----------------- |
| CORS             | `cors.go`        | `CORSConfig` + `Validate()`            | Yes   | `ExampleCORS`                | `BenchmarkCORS` (444 ns)                      | `FuzzCORS`        |
| ClientIP         | `clientip.go`    | —                                      | Yes   | `ExampleClientIP`            | `BenchmarkClientIP` (44 ns)                   | `FuzzClientIP`    |
| RequestID        | `requestid.go`   | `RequestIDConfig` + `Validate()`       | Yes   | `ExampleRequestID`           | `BenchmarkRequestID` (381 ns)                 | `FuzzRequestID`   |
| SecurityHeaders  | `security.go`    | `SecurityHeadersConfig` + `Validate()` | Yes   | `ExampleSecurityHeaders`     | `BenchmarkSecurityHeaders` (209 ns)           | —                 |
| Recovery         | `recovery.go`    | `*slog.Logger`                         | Yes   | `ExampleRecovery`            | `BenchmarkRecovery` (54 ns)                   | —                 |
| Timeout          | `timeout.go`     | `time.Duration`                        | Yes   | `ExampleTimeout`             | `BenchmarkTimeout` (386 ns)                   | —                 |
| Logging          | `logging.go`     | `*slog.Logger`                         | Yes   | `ExampleLogging`             | `BenchmarkLogging` (1,014 ns)                 | —                 |
| ResponseRecorder | `recorder.go`    | —                                      | Yes   | `ExampleNewResponseRecorder` | `BenchmarkResponseRecorder` (31 ns, 0 allocs) | —                 |
| Compression      | `compression.go` | `CompressionConfig` + `Validate()`     | Yes   | `ExampleCompression`         | `BenchmarkCompression` (6.5 µs)               | `FuzzCompression` |
| ETag             | `etag.go`        | `ETagConfig` + `Validate()`            | Yes   | `ExampleETag`                | `BenchmarkETag` (407 ns)                      | `FuzzETag`        |
| Chain            | `recorder.go`    | —                                      | Yes   | `ExampleChain`               | `BenchmarkChain` (2.9 µs, 37 allocs)          | —                 |

### 2. Error Classification System

7 error codes registered via `go-error-family`:

- `ErrCodeWriteFailed`, `ErrCodeHijackUnsupported`, `ErrCodeHijackFailed`
- `ErrCodePushUnsupported`, `ErrCodePushFailed`
- `ErrCodeCompressWriteFailed`, `ErrCodeETagWriteFailed`

`RegisterErrorClassifications()` maps stdlib HTTP errors to behavioral families (Transient vs Infrastructure) with `what/why/fix/wayOut` message templates.

### 3. Shared ResponseWriter Wrapper

`wrapper.go` extracts common `WriteHeader` buffering, `Hijack`, `Push`, and `Flush` delegation. Embedded by `compressWriter` and `etagWriter`, eliminating ~80 lines of duplication.

### 4. Configuration Validation

All config types have `Validate()` methods:

- `CORSConfig.Validate()` — catches credentials+allow-all, negative MaxAge
- `CompressionConfig.Validate()` — catches invalid levels, negative MinSize
- `RequestIDConfig.Validate()` — catches nil GenerateID, empty header names
- `ETagConfig.Validate()` — catches MaxBufferSize < 0
- `SecurityHeadersConfig.Validate()` — all fields optional, validates individual values

### 5. Documentation

| File                      | Purpose                                                                                   | Status  |
| ------------------------- | ----------------------------------------------------------------------------------------- | ------- |
| `README.md`               | Feature overview, API table, usage examples, middleware ordering, compression limitations | Current |
| `doc.go`                  | Package-level godoc                                                                       | Current |
| `AGENTS.md`               | Architecture reference, testing conventions, lint rules                                   | Current |
| `CHANGELOG.md`            | v0.1.0 and v0.1.1 entries                                                                 | Current |
| `FEATURES.md`             | Honest feature inventory with status indicators                                           | Current |
| `TODO_LIST.md`            | Centralized task list with priority tiers                                                 | Current |
| `docs/DOMAIN_LANGUAGE.md` | Complete domain glossary with 10 bounded contexts                                         | Current |
| `docs/status/`            | 10 status reports                                                                         | Current |
| `docs/planning/`          | 1 execution plan                                                                          | Current |

### 6. Tooling & Quality Gates

| Gate                  | Status                                     |
| --------------------- | ------------------------------------------ |
| `golangci-lint run`   | 0 issues (~70 linters)                     |
| `go test ./... -race` | 112 tests passing, race-free               |
| `go vet ./...`        | Clean                                      |
| `go test -bench`      | 15 benchmarks passing                      |
| Coverage              | 91.2% of statements                        |
| `nix flake check`     | Format check passes                        |
| GitHub Actions CI     | Build + vet + test + benchmark + lint      |
| Release workflow      | Test + lint + govulncheck + GitHub Release |

### 7. Race Fixes (v0.1.1)

- **CORS race** — `allowOrigin` was captured outside per-request closure; moved inside to eliminate data race between concurrent requests.
- **Compression pool race** — `getGzipPool()` had concurrent map read/write; added `sync.RWMutex` with double-checked locking.
- **Pool constructor panic** — `gzip.NewWriterLevel` could fail in pool `New` func; added error check with fallback to `gzip.DefaultCompression`.

---

## b) PARTIALLY DONE

### 1. Coverage Gaps (91.2% — 8.8% uncovered)

Functions below 90% coverage:

| Function                                   | Coverage | Gap                                                    |
| ------------------------------------------ | -------- | ------------------------------------------------------ |
| `compressWriter.Flush`                     | 61.5%    | Streaming flush with gzip active                       |
| `compressWriter.startCompressAndStream`    | 66.7%    | Error branch when `startCompression` returns false     |
| `compressWriter.writePlain`                | 75.0%    | Error branch on plain write                            |
| `compressWriter.writeCompressed`           | 75.0%    | Error branch on compressed write                       |
| `compressWriter.flushPlainAndStream`       | 76.9%    | Error branches during flush transition                 |
| `etagWriter.Write`                         | 80.0%    | Streaming write error branch                           |
| `etagWriter.Flush`                         | 77.8%    | Flush after already flushed                            |
| `compressWriter.isCompressibleContentType` | 83.3%    | One deny-list entry untested                           |
| `compressWriter.Write`                     | 83.3%    | Error branch after compression started                 |
| `compressWriter.Close`                     | 86.7%    | GzipWriter.Close error path                            |
| `getGzipPool`                              | 88.2%    | Double-checked locking slow path                       |
| `responseWrapper.Hijack`                   | 71.4%    | Hijack error path                                      |
| `responseWrapper.Push`                     | 71.4%    | Push error path                                        |
| `ResponseRecorder.Hijack`                  | 42.9%    | Successful Hijack path (untested with real connection) |

### 2. flake.nix — No Build Check

No `go build` nix check because the sandbox can't download `go-error-family`. Only the format check runs. CI handles the real build.

### 3. Performance Optimizations Not Yet Applied

- `getGzipPool` uses map+mutex — could use slice+atomic for 12 possible levels
- `generateRequestID` does `crypto/rand.Read` per request — could batch
- `etagInList` uses `strings.Split` — could be zero-allocation inline parse
- `incompressiblePrefixes` is linear scan — could use a prefix map

---

## c) NOT STARTED

### Near-term (Configurable & Architectural)

1. ~~**`CompressionConfig.SkipContentTypes`** — custom deny-list for content types~~ done (shipped as IncompressibleTypes)
2. ~~**`MiddlewareStack` type with ordering validation** — type-safe middleware composition~~ done (shipped (stack.go))
3. ~~**`ResponseWriter` capability interface** — unify Hijack/Push/Flush detection~~ done (shipped (DetectCapabilities, capabilities.go))

### Medium-term (Features)

4. ~~**Deflate support** — `compress/flate` writer, stdlib-only~~ done (shipped (DefaultWriterFactories))
5. ~~**Accept-Encoding quality value parsing** — RFC 7231 compliant negotiation~~ done (shipped (compression_qvalue.go + property tests))
6. ~~**Streaming ETag** — rolling hash without buffering~~ done (Won't implement — ROADMAP Non-goals: headers precede body, buffering is mandatory)

### Worth Considering

7. ~~**Request/response metrics middleware** — `expvar` or custom histograms~~ done (shipped (metrics.go))
8. ~~**Rate-limiting middleware** — token bucket or sliding window~~ done (shipped (ratelimit.go, deprecated; KeyedRateLimiter succeeded it))
9. ~~**Request body size limit middleware** — OOM prevention~~ done (shipped (maxbodysize.go))
10. ~~**Brotli support** — plugin interface or dependency relaxation~~ done (shipped as WriterFactory plugin docs (docs/integrations/brotli-zstd.md))
11. ~~**HTTP/2 Server Push integration test**~~ done (moot (http.Pusher code removed in v0.3.0))
12. ~~**WebSocket upgrade test through Compression + ETag**~~ done (Won't implement — removed 2026-08-07 as fragile; Hijack tiers restored 2026-08-30)

---

## d) TOTALLY FUCKED UP!

### Nothing is totally fucked up.

The codebase is in excellent shape:

- 112 tests passing with race detection
- 91.2% coverage
- 0 lint issues
- 15 benchmarks covering all middlewares
- Race-free (verified with `-race`)
- Two production-ready releases (v0.1.0, v0.1.1)

### But these will hurt if the project grows:

1. **No middleware ordering guardrails** — `Chain()` silently accepts any order. Wrong order (ETag outside Compression) produces subtly broken ETags. Tests catch this for Compression+ETag but not for all combinations.
2. **Compression `Flush` is undertested** (61.5% coverage) — streaming flush with gzip active is a complex path that could break in production.
3. **No `build` nix check** — the nix sandbox can't build the library. CI fills this gap, but it's not a nix-level guarantee.
4. **ETag always buffers** — even with 1MB limit, large JSON APIs or file serving will hit this. A streaming option would be architecturally significant.
5. **`generateRequestID` syscall per request** — at 10k+ req/s, `crypto/rand.Read` dominates latency. Not a problem today, but will be.

---

## e) WHAT WE SHOULD IMPROVE

### Critical (Next release)

1. **Fill compression `Flush` coverage gap** — 61.5% → 90%+. The streaming flush path is the most complex code in the library and the most likely to break.
2. **Fill `ResponseRecorder.Hijack` coverage gap** — 42.9% is the lowest single-function coverage. The successful Hijack path is untested.
3. **Update flake.lock** — nixpkgs is stale (June 2 vs June 8). `nix flake update` is a 30-second task.

### High Impact / Low Effort

4. **Replace `getGzipPool` map+mutex with slice+atomic** — 12 possible gzip levels; a `[13]*sync.Pool` + `sync.Once` per level is lock-free.
5. **Zero-allocation `etagInList`** — inline parse instead of `strings.Split`. One allocation per conditional request removed.
6. **Batch `generateRequestID` reads** — read 256 bytes, slice into 16-byte chunks. ~16x fewer syscalls.
7. **Add `CompressionConfig.SkipContentTypes`** — custom deny-list. Users need this for `application/wasm`, `font/woff2`, etc.

### Medium Impact / Medium Effort

8. **Add `MiddlewareStack` with ordering validation** — `Stack{}.Add(CORS).Add(Recovery).Build()`. Validates correct ordering at construction time.
9. **Implement deflate support** — `compress/flate` writer. Many clients accept deflate.
10. **Add `Accept-Encoding` quality parsing** — RFC 7231 compliant. Currently naive `strings.Contains`.
11. **Add `ResponseWriter` capability interface** — `type FlushHijackPusher interface { ... }` for unified detection.

### Long-term / Architectural

12. **Streaming ETag** — rolling hash, no buffering. Would require API change or new option.
13. **Rate-limiting middleware** — token bucket using `golang.org/x/time/rate`.
14. **Request body size limit middleware** — simple `io.LimitReader` wrapper.
15. **Brotli plugin interface** — `WriterFactory func(io.Writer) io.WriteCloser` for extensible compression.

---

## f) Top #25 Things We Should Get Done Next

| #  | Task                                                      | Impact   | Effort | Category        |
| -- | --------------------------------------------------------- | -------- | ------ | --------------- |
| 1  | Fill compression `Flush` coverage (61.5% → 90%+)          | Critical | 20 min | Quality         |
| 2  | Fill `ResponseRecorder.Hijack` coverage (42.9% → 90%+)    | Critical | 15 min | Quality         |
| 3  | Update flake.lock (stale nixpkgs)                         | Low      | 1 min  | Infrastructure  |
| 4  | Replace `getGzipPool` map+mutex with slice+atomic         | High     | 20 min | Performance     |
| 5  | Zero-allocation `etagInList`                              | Medium   | 15 min | Performance     |
| 6  | Batch `generateRequestID` reads                           | High     | 15 min | Performance     |
| ~~7~~  | ~~Add `CompressionConfig.SkipContentTypes`~~ done — shipped as IncompressibleTypes | ~~Medium~~ | ~~15 min~~ | ~~Configurability~~ |
| 8  | Fill `compressWriter.startCompressAndStream` coverage     | Medium   | 10 min | Quality         |
| ~~9~~  | ~~Fill `compressWriter.writePlain/writeCompressed` coverage~~ done — shipped (compress_writer_test.go error-branch tests) | ~~Medium~~ | ~~10 min~~ | ~~Quality~~ |
| ~~10~~ | ~~Fill `responseWrapper.Hijack/Push` error paths~~ done — (wrapper_test.go error-path tests, 2026-08-30) | ~~Medium~~ | ~~10 min~~ | ~~Quality~~ |
| ~~11~~ | ~~Fill `etagWriter.Write` streaming error branch~~ done — shipped (compress_writer_test.go error-branch tests) | ~~Medium~~ | ~~5 min~~ | ~~Quality~~ |
| ~~12~~ | ~~Fill `compressWriter.Close` error path~~ done — (Close idempotency + error tests, 2026-08-30) | ~~Medium~~ | ~~5 min~~ | ~~Quality~~ |
| ~~13~~ | ~~Fill `getGzipPool` slow path coverage~~ done — (pool coverage via newWriterPool tests) | ~~Low~~ | ~~5 min~~ | ~~Quality~~ |
| ~~14~~ | ~~Add `MiddlewareStack` with ordering validation~~ done — shipped (stack.go) | ~~High~~ | ~~45 min~~ | ~~Architecture~~ |
| ~~15~~ | ~~Add `ResponseWriter` capability interface~~ done — shipped (DetectCapabilities, capabilities.go) | ~~Medium~~ | ~~30 min~~ | ~~Architecture~~ |
| ~~16~~ | ~~Implement deflate support~~ done — shipped (DefaultWriterFactories) | ~~Medium~~ | ~~30 min~~ | ~~Feature~~ |
| ~~17~~ | ~~Add Accept-Encoding quality value parsing~~ done — shipped (compression_qvalue.go + property tests) | ~~Low~~ | ~~20 min~~ | ~~Correctness~~ |
| ~~18~~ | ~~Fill `etagWriter.Flush` after-flush path~~ done — shipped (compress_writer_test.go error-branch tests) | ~~Low~~ | ~~5 min~~ | ~~Quality~~ |
| ~~19~~ | ~~Streaming ETag (rolling hash)~~ done — Won't implement — ROADMAP Non-goals: headers precede body, buffering is mandatory | ~~High~~ | ~~60 min~~ | ~~Performance~~ |
| ~~20~~ | ~~Rate-limiting middleware~~ done — shipped (ratelimit.go, deprecated; KeyedRateLimiter succeeded it) | ~~Medium~~ | ~~60 min~~ | ~~Feature~~ |
| ~~21~~ | ~~Request body size limit middleware~~ done — shipped (maxbodysize.go) | ~~Low~~ | ~~20 min~~ | ~~Safety~~ |
| ~~22~~ | ~~Brotli plugin interface~~ done — shipped as WriterFactory plugin docs (docs/integrations/brotli-zstd.md) | ~~Medium~~ | ~~45 min~~ | ~~Extensibility~~ |
| ~~23~~ | ~~HTTP/2 Server Push integration test~~ done — moot (http.Pusher code removed in v0.3.0) | ~~Low~~ | ~~15 min~~ | ~~Coverage~~ |
| ~~24~~ | ~~WebSocket upgrade test through Compression + ETag~~ done — Won't implement — removed 2026-08-07 as fragile; Hijack tiers restored 2026-08-30 | ~~Low~~ | ~~15 min~~ | ~~Coverage~~ |
| 25 | Add nix build check that works offline                    | Medium   | 20 min | Infrastructure  |

---

## g) Top #1 Question I Cannot Figure Out Myself

**Should httputil add middleware that doesn't exist in the stdlib (rate-limiting, metrics, body-size-limit) or stay focused on "stdlib patterns made composable"?**

The library's identity is: stdlib `net/http` patterns composed into reusable middleware with zero surprises. Every middleware so far wraps stdlib behavior:

- CORS = HTTP headers
- Recovery = `defer/recover`
- Timeout = `context.WithTimeout`
- Compression = `compress/gzip`
- ETag = `crypto/hash`

Rate-limiting and metrics are **different beasts** — they require state (counters, windows, registries) and introduce opinions (what rate? what histogram buckets? what registry API?).

**Options:**

1. ~~**Stay focused** — httputil is "stdlib patterns made composable." Rate-limiting and metrics belong in a separate package. Document the boundary clearly.~~ done (not chosen — rate limiting & metrics shipped in scope (x/time allowed))
2. ~~**Expand scope** — add rate-limiting (using `golang.org/x/time/rate`) and metrics (using `expvar`). Accept that the library becomes opinionated.~~ done (chosen: rate-limiting + metrics shipped (golang.org/x/time allowed))
3. ~~**Plugin interface** — add `MiddlewareStack` with hooks for pre/post processing, letting users inject their own rate-limiting/metrics without httputil owning the implementations.~~ done (superseded — WriterFactory + MiddlewareStack shipped instead of generic hooks)

**My recommendation:** Option 1. The library's strength is its focused scope. Rate-limiting and metrics are genuinely different domains. A separate `httplimit` or `httpmetrics` package would be more honest.

**But:** The user might want a batteries-included package. I need a product decision.
