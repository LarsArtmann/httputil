# Comprehensive Status Report — httputil

**Date:** 2026-06-08 00:25 CEST
**Branch:** `master`
**Commits ahead of origin:** 0 (working tree has uncommitted changes)
**Last commit:** `1f79f73` docs(status): add comprehensive project status report
**Go version:** 1.26.3
**Lines of code:** ~3,700 across 26 Go files
**Test coverage:** 87.1% of statements
**Lint status:** 0 issues across ~70 linters
**Tests:** 114+ passing, race detection clean
**Benchmarks:** 15 benchmarks covering all middlewares + Chain + Itoa + Join

---

## a) FULLY DONE

### 1. Documentation Infrastructure (Inspired by `~/projects/overview`)

| File           | Status  | Notes                                                                                   |
| -------------- | ------- | --------------------------------------------------------------------------------------- |
| `FEATURES.md`  | Created | Honest feature inventory with FULLY DONE / PARTIALLY DONE / PLANNED / WORTH CONSIDERING |
| `TODO_LIST.md` | Created | Centralized task list with completed and not-started sections, verified against code    |
| `README.md`    | Updated | Added compression limitations section, expanded development commands                    |

### 2. flake.nix Improvements (Inspired by `~/projects/overview`)

| Improvement      | Before                               | After                                                              |
| ---------------- | ------------------------------------ | ------------------------------------------------------------------ |
| App scripts      | `writeShellScriptBin` (no `errexit`) | `writeShellApplication` (`set -euo pipefail`, `PATH` managed)      |
| Source filtering | Copies entire directory              | `lib.fileset.toSource` with explicit fileset unions                |
| Format check     | Missing                              | `checks.format = config.treefmt.build.check self`                  |
| DevShell         | Manual package list                  | Uses `inputsFrom` pattern (removed because library has no package) |
| Build check      | Broken (no vendor, sandboxed)        | Removed for library; kept format check only                        |

### 3. CI Workflow Improvements

| Step      | Before          | After                              |
| --------- | --------------- | ---------------------------------- |
| Build     | Missing         | `go build ./...` added             |
| Vet       | Missing         | `go vet ./...` added               |
| Benchmark | Missing         | `go test -bench=. -benchmem` added |
| Test      | `go test ./...` | `go test -race -count=1 ./...`     |

### 4. .golangci.yml Improvements

| Improvement               | Before                                                     | After                                                                               |
| ------------------------- | ---------------------------------------------------------- | ----------------------------------------------------------------------------------- |
| `gocognit` test exclusion | Missing                                                    | Added to test file exclusions                                                       |
| `noctx` test exclusion    | Missing                                                    | Already present, kept                                                               |
| Build tags                | `goexperiment.jsonv2` only                                 | Added `arenas`, `goroutineleakprofile`, `runtimesecret`, `simd`                     |
| `varnamelen` ignores      | `err`, `ok`, `tt`, `fn`, `t`, `i`, `m`, `g`, `a`, `b`, `v` | Added `w`, `r`, `n`, `rw` for `http.ResponseWriter` and `bufio.ReadWriter` patterns |

### 5. Benchmarks — All Middlewares Now Covered

| Benchmark                   | File                  | Body Size | ~ns/op | ~allocs/op |
| --------------------------- | --------------------- | --------- | ------ | ---------- |
| `BenchmarkClientIP`         | `clientip_test.go`    | N/A       | 52     | 1          |
| `BenchmarkCORS`             | `cors_test.go`        | N/A       | 511    | 12         |
| `BenchmarkCompression`      | `compression_test.go` | ~1KB      | 6,639  | 12         |
| `BenchmarkETag`             | `etag_test.go`        | ~31B      | 439    | 12         |
| `BenchmarkRequestID`        | `middleware_test.go`  | N/A       | 407    | 11         |
| `BenchmarkSecurityHeaders`  | `middleware_test.go`  | N/A       | 231    | 7          |
| `BenchmarkRecovery`         | `middleware_test.go`  | N/A       | 61     | 3          |
| `BenchmarkTimeout`          | `middleware_test.go`  | N/A       | 387    | 8          |
| `BenchmarkLogging`          | `middleware_test.go`  | N/A       | 1,035  | 10         |
| `BenchmarkResponseRecorder` | `recorder_test.go`    | ~31B      | 45     | 0          |
| `BenchmarkChain`            | `middleware_test.go`  | N/A       | 3,013  | 37         |
| `BenchmarkItoa`             | `util_test.go`        | N/A       | 69     | 0          |
| `BenchmarkItoa_Strconv`     | `util_test.go`        | N/A       | 101    | 4          |
| `BenchmarkJoin`             | `util_test.go`        | N/A       | 48     | 1          |
| `BenchmarkJoin_StringsJoin` | `util_test.go`        | N/A       | 41     | 1          |

### 6. Example Functions — All Middlewares Now Covered

| Example                      | File              | Output Verified  |
| ---------------------------- | ----------------- | ---------------- |
| `ExampleClientIP`            | `example_test.go` | `203.0.113.1`    |
| `ExampleCORS`                | `example_test.go` | `204`            |
| `ExampleChain`               | `example_test.go` | `[first second]` |
| `ExampleNewResponseRecorder` | `example_test.go` | `404`            |
| `ExampleCompression`         | `example_test.go` | `gzip`           |
| `ExampleETag`                | `example_test.go` | `true`           |
| `ExampleRequestID`           | `example_test.go` | `true`           |
| `ExampleSecurityHeaders`     | `example_test.go` | `nosniff`        |
| `ExampleRecovery`            | `example_test.go` | `500`            |
| `ExampleTimeout`             | `example_test.go` | `true`           |
| `ExampleLogging`             | `example_test.go` | `200`            |

### 7. Fuzz Tests

| Fuzz Test         | File                  | Seed Corpus | What It Tests                                      |
| ----------------- | --------------------- | ----------- | -------------------------------------------------- |
| `FuzzClientIP`    | `clientip_test.go`    | 5 seeds     | X-Forwarded-For parsing with arbitrary strings     |
| `FuzzCompression` | `compression_test.go` | 5 seeds     | gzip compression with arbitrary bodies             |
| `FuzzETag`        | `etag_test.go`        | 5 seeds     | ETag generation with arbitrary bodies              |
| `FuzzCORS`        | `cors_test.go`        | 7 seeds     | CORS origin matching with arbitrary origins        |
| `FuzzRequestID`   | `middleware_test.go`  | 4 seeds     | Request ID generation with arbitrary header values |

### 8. Integration Tests for Common Middleware Chains

| Test                                        | File                 | Chain Under Test              | Validates                                                     |
| ------------------------------------------- | -------------------- | ----------------------------- | ------------------------------------------------------------- |
| `TestChain_RecoveryLoggingCORS`             | `middleware_test.go` | `CORS → Recovery → Logging`   | All three middlewares compose correctly, CORS headers present |
| `TestChain_RecoveryCatchesPanicWithLogging` | `middleware_test.go` | `CORS → Recovery → Logging`   | Panic caught, 500 returned, no crash                          |
| `TestChain_RequestIDSecurityHeaders`        | `middleware_test.go` | `SecurityHeaders → RequestID` | Both headers set correctly                                    |
| `TestChain_TimeoutThenRecovery`             | `middleware_test.go` | `Recovery → Timeout`          | Deadline set, 200 returned                                    |

### 9. Data Race Fix

`compression.go:getGzipPool()` had a race condition: concurrent map read/write on `gzipWriterPools`.

**Fix:** Added `sync.RWMutex` around the map with double-checked locking pattern.

Before:

```go
var gzipWriterPools = make(map[int]*sync.Pool)
func getGzipPool(level int) *sync.Pool {
    pool, ok := gzipWriterPools[level] // race: concurrent read
    if !ok {
        pool = &sync.Pool{...}
        gzipWriterPools[level] = pool // race: concurrent write
    }
    return pool
}
```

After:

```go
var (
    gzipWriterPools   = make(map[int]*sync.Pool)
    gzipWriterPoolsMu sync.RWMutex
)
func getGzipPool(level int) *sync.Pool {
    // RLock → check → RUnlock → Lock → check again → create
}
```

### 10. Code Quality Improvements

| Issue                     | Fix                                                                                           |
| ------------------------- | --------------------------------------------------------------------------------------------- |
| `goconst` "Content-Type"  | Extracted to `headerContentType` constant, used in `compression.go`, `cors.go`, `recovery.go` |
| `errcheck` in benchmark   | `recorder.Write(body)` → `_, _ = recorder.Write(body)`                                        |
| `sloglint` example tests  | `slog.NewTextHandler(io.Discard, nil)` → `slog.DiscardHandler`                                |
| `wsl_v5` in `getGzipPool` | Added blank lines around RUnlock, fixed by `golangci-lint fmt`                                |

---

## b) PARTIALLY DONE

### 1. flake.nix — Still Has Warnings

`nix flake check` warns that apps lack `meta.description`. The apps work fine, but the warnings clutter output. Easy 5-minute fix.

### 2. Test Coverage (87.1%)

Still not 90%+. Gaps in:

- `compression.go` error branches (`startCompression` type mismatch, `Close` errors)
- `ResponseRecorder` hijack/push failure paths
- `wrapper.go` hijack/push failure paths (same code, same gap)
- `util.go` branches (`itoa` negative numbers, `join` empty slices)

### 3. AGENTS.md Out of Date

Still references "114 tests", "missing benchmarks", "no FEATURES.md", "no TODO_LIST.md". Needs update to reflect current state.

### 4. nix flake check — No Build Check

The `build` check was removed because `buildGoModule` with `vendorHash = null` fails in the sandbox (no network access to download `go-error-family`). For a library, this is acceptable — the format check ensures code compiles syntactically, and CI handles the full build. But having a `go build` check that works offline would be nice.

### 5. Status Reports

Previous reports still describe the old state. The new report (this one) captures the current state, but older reports are stale.

---

## c) NOT STARTED

### 1. Performance: `getGzipPool` Could Use a Slice Instead of Map + Mutex

Gzip levels are limited to -2 (HuffmanOnly) through 9 (BestCompression). That's 12 possible values. A `[12]*sync.Pool` array with `atomic.Pointer` would be lock-free and faster than a `map[int]*sync.Pool` + `RWMutex`.

### 2. Performance: `incompressiblePrefixes` Uses Linear Scan

The deny-list is a `[]string` checked with `strings.HasPrefix` in a loop. A `map[string]struct{}` of prefixes or a trie would be O(1) instead of O(n).

### 3. Performance: `etagInList` Allocates with `strings.Split`

RFC 7232 `If-None-Match` parsing splits on commas and trims spaces. `strings.Split` allocates a new slice. An inline parse loop with `strings.IndexByte` would be zero-allocation.

### 4. Performance: `generateRequestID` Does a Syscall Per Request

`crypto/rand.Read` calls into the kernel for 16 bytes. For high-throughput servers, this is expensive. Options:

- Batch reads into a larger buffer and slice from it
- Use `math/rand/v2` with a CSPRNG (Go 1.26 has `math/rand/v2.ChaCha8`)
- Use a `sync.Pool` of pre-filled buffers

### 5. Type Safety: `MiddlewareStack` with Ordering Validation

Currently `Chain()` silently accepts any order. A typed `MiddlewareStack` could validate that ETag is inner to Compression, Recovery is outermost, etc.

### 6. Type Safety: `StatusCode` Type Instead of Bare `int`

`ResponseRecorder.status` is `int`. A `type StatusCode int` with constants would make impossible states unrepresentable (e.g., negative status codes).

### 7. Type Safety: `Encoding` Enum for Compression

`encodingGzip = "gzip"` is a string constant. A typed enum would prevent mixing compression encoding values with arbitrary strings.

### 8. Configuration: `CompressionConfig.SkipContentTypes`

The deny-list is hardcoded. Users might want to skip `application/wasm`, `font/woff2`, or other formats we didn't think of.

### 9. Feature: Deflate Support

`compress/flate` is in the stdlib. Many clients accept `deflate`. This is a medium-effort feature with real compatibility value.

### 10. Feature: Accept-Encoding Quality Value Parsing

Current code: `strings.Contains(req.Header.Get("Accept-Encoding"), "gzip")`. RFC 7231 allows `gzip;q=0.8, deflate;q=0.5`. A proper parser would negotiate the best encoding.

### 11. Feature: Streaming ETag (No Buffering)

Current ETag buffers the entire response body (up to 1MB). For large JSON APIs or file serving, a rolling hash (e.g., `hash/fnv128`) would allow streaming computation without the memory cost.

### 12. Feature: Request/Response Metrics Middleware

A `Metrics()` middleware that counts requests by status code, tracks duration histograms, and exposes them via `expvar` or a custom registry. This is a common need that every HTTP service eventually wants.

### 13. Feature: Rate-Limiting Middleware

A `RateLimit()` middleware using a token bucket or sliding window. Currently not in scope, but genuinely useful.

### 14. Feature: Request Body Size Limit Middleware

A `MaxBodySize()` middleware that errors if the request body exceeds a limit. Prevents OOM from malicious uploads.

---

## d) TOTALLY FUCKED UP!

### Nothing is totally fucked up.

The codebase compiles, all tests pass (with race detection), lint is clean, benchmarks exist for all middlewares, examples exist for all middlewares, fuzz tests exist for CORS/RequestID/ClientIP/Compression/ETag, integration tests cover common chains, and documentation is comprehensive.

However, these are **architectural debts** that will cause pain as the project grows:

1. **`getGzipPool` map + mutex** — works, but a slice + atomic would be faster and simpler
2. **`generateRequestID` syscall per request** — will be the bottleneck at 10k+ req/s
3. **No `MiddlewareStack` ordering validation** — wrong order (ETag outside Compression) produces subtly broken behavior that tests won't catch in all cases
4. **ETag always buffers** — even with the 1MB limit, this is a footgun for large-response APIs
5. **No `build` nix check** — the sandbox can't build the library. Acceptable for a library, but not ideal.

---

## e) WHAT WE SHOULD IMPROVE

### Critical (Do Before Next Release)

1. **Fix `nix flake check` app warnings** — add `meta.description` to all apps. 5 min.
2. **Update AGENTS.md** — reflect new test count, benchmarks, examples, fuzz tests, and files. 10 min.
3. **Replace `getGzipPool` map+mutex with slice+atomic** — lock-free, faster, simpler. 20 min.

### High Impact / Low Effort

4. **Zero-allocation `etagInList`** — inline parse without `strings.Split`. 15 min.
5. **Add `CompressionConfig.SkipContentTypes`** — custom deny list. 15 min.
6. **Batch `generateRequestID` reads** — read 256 bytes at once, slice into 16-byte chunks. 15 min.
7. **Add `http.StatusCode` type** — typed status codes. 20 min.
8. **Improve test coverage to 90%+** — fill gaps in compression errors, recorder failures. 30 min.

### Medium Impact / Medium Effort

9. **Implement deflate support** — `compress/flate` writer. 30 min.
10. **Add `Accept-Encoding` quality value parsing** — RFC 7231 compliant. 20 min.
11. **Add `MiddlewareStack` type** — ordering validation. 45 min.
12. **Add `ResponseWriter` capability interface** — unify Hijack/Push/Flush detection. 30 min.
13. **Add metrics middleware** — request counts + duration histograms. 45 min.

### Long-term / Architectural

14. **Streaming ETag** — rolling hash, no buffering. 60 min.
15. **Rate-limiting middleware** — token bucket. 60 min.
16. **Request body size limit middleware** — OOM prevention. 20 min.
17. **Evaluate brotli dependency relaxation** — if we add a plugin interface. 30 min.
18. **Add HTTP/2 Server Push integration test** — end-to-end with real pusher. 15 min.
19. **WebSocket upgrade test** — verify Hijack works through Compression + ETag. 15 min.
20. **Content-Length preservation test** — small responses that don't trigger compression. 10 min.

---

## f) Top #25 Things We Should Get Done Next

Sorted by **impact / effort ratio** (highest impact per unit of work first):

| #   | Task                                              | Impact | Effort | Category        |
| --- | ------------------------------------------------- | ------ | ------ | --------------- |
| 1   | Fix nix flake app `meta.description` warnings     | Low    | 5 min  | Infrastructure  |
| 2   | Update AGENTS.md to current state                 | Medium | 10 min | Documentation   |
| 3   | Replace `getGzipPool` map+mutex with slice+atomic | High   | 20 min | Performance     |
| 4   | Zero-allocation `etagInList` parsing              | Medium | 15 min | Performance     |
| 5   | Batch `generateRequestID` reads                   | High   | 15 min | Performance     |
| 6   | Add `CompressionConfig.SkipContentTypes`          | Medium | 15 min | Configurability |
| 7   | Add typed `StatusCode`                            | Medium | 20 min | Type Safety     |
| 8   | Improve test coverage to 90%+                     | Medium | 30 min | Quality         |
| 9   | Add `MiddlewareStack` with ordering validation    | High   | 45 min | Architecture    |
| 10  | Add deflate support                               | Medium | 30 min | Feature         |
| 11  | Add `Accept-Encoding` quality value parsing       | Low    | 20 min | Correctness     |
| 12  | Add metrics middleware                            | Medium | 45 min | Feature         |
| 13  | Add `ResponseWriter` capability interface         | Low    | 30 min | Architecture    |
| 14  | Streaming ETag (rolling hash)                     | High   | 60 min | Performance     |
| 15  | Rate-limiting middleware                          | Medium | 60 min | Feature         |
| 16  | Request body size limit middleware                | Low    | 20 min | Safety          |
| 17  | WebSocket upgrade test                            | Low    | 15 min | Coverage        |
| 18  | Content-Length preservation test                  | Low    | 10 min | Correctness     |
| 19  | HTTP/2 Server Push integration test               | Low    | 15 min | Coverage        |
| 20  | Evaluate brotli plugin interface                  | Medium | 30 min | Decision        |
| 21  | Add `ExampleResponseRecorder`                     | Low    | 10 min | DX              |
| 22  | Add `BenchmarkChain`                              | Low    | 10 min | Observability   |
| 23  | Improve benchmark suite with varying body sizes   | Low    | 20 min | Observability   |
| 24  | Add `go test -race` to CI                         | High   | 5 min  | Safety          |
| 25  | Add nix `build` check that works offline          | Medium | 20 min | Infrastructure  |

---

## g) Top #1 Question I Cannot Figure Out Myself

**How do we make the nix `build` check work for a Go library with external dependencies in a pure sandbox?**

Options:

1. **Use `go mod vendor`** — run `go mod vendor` in the repo, commit `vendor/`, and set `vendorHash = null` (or compute a real hash). This makes the build fully offline. Downside: `vendor/` bloats the repo by ~100KB per dependency.

2. **Use `buildGoModule` with a real `vendorHash`** — compute the vendor hash from `go.sum` and let Nix prefetch it. This is the standard pattern. Downside: requires running `nix-prefetch-url` or `nix-prefetch-git` manually when dependencies change.

3. **Skip the build check for libraries** — libraries are never built as standalone binaries. The format check + CI build is sufficient. Downside: no nix-level guarantee that the code compiles.

4. **Use `nix build` on a test binary** — create a `main_test.go` or a `cmd/` entry that imports the library, then build that. This forces the compiler to type-check all exported APIs. Downside: adds noise to the repo.

**My preference:** Option 1 — vendor dependencies. It's explicit, reproducible, and makes the nix build check work. The `vendor/` directory is a one-time ~100KB addition for `go-error-family`.

**But:** Should a library repo really vendor its dependencies? Most Go libraries don't. The `vendor/` directory is typically for applications, not libraries.

**I need a decision:** Do we vendor for the sake of the nix build check, or do we accept that library nix builds are inherently different from application nix builds?

---

## Self-Reflection: What Did I Forget? What Could Be Better?

### What I Forgot

1. **I didn't update AGENTS.md.** It's now stale. The test count changed, benchmarks were added, examples were added. AGENTS.md should be the single source of truth for AI sessions.

2. **I didn't check if `overview` had patterns we should adopt for `httputil`.** `overview` has `docs/architecture-understanding/` with D2 diagrams. `httputil` has nothing visual. A simple D2 diagram of the middleware chain would be valuable.

3. **I didn't check if there were pre-existing issues I could have fixed.** The `goconst` warning for "Content-Type" in test files (excluded by `.golangci.yml`) still shows in LSP. I should verify `.golangci.yml` actually excludes test files from `goconst`.

4. **I committed formatting changes without verifying them.** `golangci-lint fmt` changed `wrapper.go`, `etag.go`, `recorder.go`, `clientip_test.go`, `compression_test.go`. These are purely cosmetic but touch many lines. I should have reviewed each diff.

5. **I didn't consider established libraries before implementing.** For example:
   - `rs/cors` has mature CORS handling with more edge cases
   - `gorilla/handlers` has battle-tested logging/recovery patterns
   - `chi/middleware` has compression with more features

   But our single-dependency policy is intentional. We should at least **document** what features we intentionally omit compared to these libraries.

### What Could Be Better

1. **The `getGzipPool` fix was a band-aid.** A map + mutex is correct but not optimal. A slice + atomic is the right solution for a bounded key space.

2. **The benchmarks measure ServeHTTP overhead, not just the middleware function.** This is realistic but makes it hard to isolate middleware cost. A `BenchmarkRequestIDRaw` that calls the middleware function directly would show the true overhead.

3. **The integration tests only verify composition, not interaction.** `TestChain_RecoveryLoggingCORS` checks that all three run, but doesn't verify that Recovery catches a panic before Logging logs it, or that CORS headers are set before Recovery writes its 500 response.

4. **The status reports are prose-heavy.** A machine-readable status format (YAML/JSON) would make it easier for AI sessions to parse and act on.

### What Could Still Improve

1. **Type models:** `Middleware` is just a function type. A `MiddlewareStack` struct with `Add()`, `Validate()`, and `Build()` methods would be a huge DX improvement.

2. **Error classification:** Currently only `ResponseRecorder` uses `go-error-family`. Compression and ETag errors are classified but the middleware constructors don't validate configs with classified errors.

3. **Configuration validation:** `CORSConfig.Validate()` and `CompressionConfig.Validate()` exist but return plain `error`. They could return classified errors with context.

4. **Documentation:** No architecture diagram exists. A D2 diagram showing `Request → CORS → Recovery → Logging → Timeout → RequestID → SecurityHeaders → Compression → ETag → Handler` would be invaluable.

5. **Performance:** `generateRequestID` is the only middleware that does a syscall per request. At high throughput, this dominates latency.

---

> **Next steps:** Fix the nix warnings, update AGENTS.md, then tackle the top 5 performance/type-safety improvements from the Pareto plan.
