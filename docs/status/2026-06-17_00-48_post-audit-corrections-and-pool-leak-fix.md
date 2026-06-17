# Comprehensive Status Report

**Date:** 2026-06-17 00:48  
**Commit:** _work in progress_ (based on b88eaf5)  
**Trigger:** Follow-up execution on the 2026-06-16 post-`util.go`-removal audit; documentation sync, test-gap closure, performance wins, and a real production defect discovered in the compression writer pool.

---

## Project Metrics (Live)

| Metric                | Value                                | Notes                             |
| --------------------- | ------------------------------------ | --------------------------------- |
| Go version            | 1.26.3                               | `go.mod`                          |
| Source files          | 46 `.go` files total (source + test) | +1 new `compress_writer_test.go`  |
| Lines of code         | ~6,050 total (source + test)         | approx. +210 from previous report |
| External dependencies | 1 (`go-error-family` v0.3.0)         | zero transitive deps              |
| Tests                 | **148 passing**                      | +10 from previous count           |
| Test coverage         | **91.3%**                            | up from 90.2%                     |
| Fuzz tests            | 6                                    | +1 `FuzzCORSWildcardPattern`      |
| Benchmarks            | 12                                   | +1 `BenchmarkNegotiateEncoding`   |
| Examples              | 11                                   | unchanged                         |
| Lint                  | **0 issues** across ~70 linters      | `golangci-lint run`               |
| Race detector         | **Clean**                            | `go test -race ./...`             |
| CI                    | GitHub Actions (test + lint)         | `.github/workflows/ci.yml`        |
| Release CI            | `govulncheck` + tag-triggered        | `.github/workflows/release.yml`   |
| Dev env               | Nix flake                            | reproducible                      |

### Benchmark Snapshot (Live, 2026-06-17)

```
BenchmarkChain-32                              415878     2700 ns/op     3395 B/op    34 allocs/op
BenchmarkClientIP-32                         27120610    95.19 ns/op       32 B/op     1 allocs/op
BenchmarkCompression-32                         91304    11926 ns/op     2156 B/op    12 allocs/op
BenchmarkNegotiateEncoding/single_token-32  185206503    7.199 ns/op        0 B/op     0 allocs/op
BenchmarkNegotiateEncoding/multi_token-32    25973326    51.71 ns/op        0 B/op     0 allocs/op
BenchmarkNegotiateEncoding/qvalues-32        26697678    44.56 ns/op        0 B/op     0 allocs/op
BenchmarkCORS-32                              2919865    392.9 ns/op      592 B/op     9 allocs/op
BenchmarkETag-32                              2350726    495.5 ns/op     1136 B/op    12 allocs/op
BenchmarkLogging-32                            998708    1022 ns/op      716 B/op    10 allocs/op
BenchmarkResponseRecorder-32                100000000    26.12 ns/op       85 B/op     0 allocs/op
BenchmarkRecovery-32                         15577314    76.70 ns/op      160 B/op     3 allocs/op
BenchmarkRequestID-32                         2438485    487.5 ns/op      976 B/op    11 allocs/op
BenchmarkSecurityHeaders-32                   4649213    263.7 ns/op      560 B/op     7 allocs/op
BenchmarkTimeout-32                           2324130    512.5 ns/op      752 B/op     8 allocs/op
```

- `BenchmarkCORS` improved from ~593 ns/op to ~393 ns/op (-34%) and from 12 to 9 allocs/op.
- `BenchmarkNegotiateEncoding/single_token` now exists at ~7 ns/op, proving the fast path is ~7x faster than the q-value parser path.

---

## a) FULLY DONE

### Documentation Sync (Items 1-9 from 2026-06-16 Audit)

| Document       | Fix                                                                                                                                                                                   |
| -------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `FEATURES.md`  | Removed rotting "193 tests / 90.4% coverage" numbers; switched to qualitative coverage claims. Added **Server Lifecycle** and **Health Checks** sections. Updated date to 2026-06-17. |
| `TODO_LIST.md` | Removed stale "112 tests, 91.2% coverage". Removed deleted `Itoa`/`Join` benchmarks from the benchmark list. Marked coverage >90% as achieved. Updated verification date.             |
| `AGENTS.md`    | Added `health.go` and `server.go` to the architecture table. Updated the compression pooling note to reflect the new per-encoding design. Updated date.                               |
| `CHANGELOG.md` | Added `[Unreleased]` entries for `util.go` removal, pool leak fix, CORS pre-computation, and `Accept-Encoding` fast path. Removed stale "193 tests, 90.4% coverage" claim.            |

### Test-Gap Closure (Items 13-15 from 2026-06-16 Audit)

- **`compress_writer_test.go` (new file)**:
  - `TestCompressWriter_WriteCompressed_ReturnsClassifiedError` — covers the compressed-writer write failure path.
  - `TestCompressWriter_Close_ReturnsClassifiedError` — covers the compression-writer close failure path.
  - `TestCompressWriter_StartCompression_PoolTypeMismatch` — covers the pool type-mismatch panic guard.
  - `TestNegotiator_PoolsAreStablePerEncoding` — regression test for the pool leak.
- **`cors_test.go`**:
  - Added `FuzzCORSWildcardPattern` with seed corpus including `*.*`, `.*`, `a.*.b`, `*.`, empty, and unicode patterns.
- **`errors_test.go`**:
  - Added `TestHijack_Failure_ClassifiedAsTransient` to cover the underlying `Hijack()` returning an error (`ErrCodeHijackFailed`).
- **`testutil_test.go`**:
  - Added `failingHijacker` helper and `errHijackFailed` sentinel for the above test.

### Critical Bug Fix: Compression Writer Pool Leak

The previous global pool registry in `compress_pool.go` keyed pools by `&factory` — the address of the `WriterFactory` function parameter. Because Go function parameters are re-allocated on every call, **every request created a new `sync.Pool` entry** while writers were never reused. This was both an unbounded memory leak and a complete defeat of the documented pooling optimization.

**Fix:**

- Removed the global `writerPools` map.
- Added `negotiator.pools`, a `map[string]*sync.Pool` owned per `Compression` middleware instance and keyed by encoding name.
- `buildWriterPools()` constructs one pool per configured encoding at middleware creation time.
- `compressWriter` now carries a `*sync.Pool` field populated by `neg.poolFor(encoding)`.
- `compress_pool.go` reduced to a pure `newWriterPool(WriterFactory) *sync.Pool` constructor.

**Files changed:** `compress_pool.go`, `compress_writer.go`, `compress_writer_compress.go`, `compression.go`, `compression_negotiator.go`, `compression_test.go`, `compress_writer_test.go`.

### Performance Wins (Items 16-17 from 2026-06-16 Audit)

- **`CORS()` pre-computes header strings** (`AllowedMethods`, `AllowedHeaders`, `ExposedHeaders`, `MaxAge`) once at middleware construction instead of on every request. Saves 2-3 allocations per response and ~34% latency.
- **`negotiator.negotiateEncoding` fast path** for single-token `Accept-Encoding` headers (e.g. `gzip`) skips the RFC 7231 q-value scanner entirely. ~7x faster for the common non-browser/API client case.

---

## b) PARTIALLY DONE

### Test Coverage (91.3%)

Up from 90.2%. Remaining gaps are narrow:

- `compressWriter.Flush()` uncompressed path (forces plain mode and writes buffered body).
- `compressWriter.startCompression()` failure branch where the pooled writer's buffered write fails after `Reset()`.
- `ResponseRecorder.Flush()` with an underlying writer that does not implement `http.Flusher` (no-op path is currently tested; a non-flusher underlying type is not).
- Edge-case HTTP status codes in `compressWriter.shouldCompress()` boundary values.

### Compression Architecture

- Content-type deny-list is still **hardcoded** in `compress_content_type.go`.
- `CompressionConfig` does not yet expose a way to override or extend the deny-list.

### Middleware Ordering / Validation

- No `MiddlewareStack` type exists.
- Ordering rules (Recovery outermost, Compression before ETag, etc.) are documented in `README.md` but not enforced by code.

### ResponseWriter Capability Interface

- `wrapper.go` still uses inline type assertions (`http.Flusher`, `http.Hijacker`) rather than a unified capability interface.

---

## c) NOT STARTED

From `TODO_LIST.md` and `FEATURES.md`:

| Item                                                                     | Priority          |
| ------------------------------------------------------------------------ | ----------------- |
| Configurable content-type filtering via `CompressionConfig`              | Near-term         |
| `MiddlewareStack` type with ordering validation                          | Near-term         |
| `ResponseWriter` capability interface for unified Hijack/Flush detection | Near-term         |
| Streaming ETag option using rolling hash                                 | Medium-term       |
| Request/response metrics middleware                                      | Worth considering |
| Rate-limiting middleware                                                 | Worth considering |
| Request body size limit middleware                                       | Worth considering |
| Brotli/zstd/lz4 documented factory examples                              | Worth considering |
| Server graceful shutdown integration test                                | Worth considering |
| Configurable readiness checker for `HealthHandler`                       | Worth considering |

---

## d) TOTALLY FUCKED UP!

### 1. Compression Writer Pool Leak (FIXED)

The "per-factory `sync.Pool`" feature documented in `AGENTS.md` and `FEATURES.md` did not actually work. Keying pools by `&factory` (the address of a function parameter) meant every call to `getWriterPool(factory)` created a brand-new map entry. Writers were never reused, and the global `writerPools` map grew without bound for the lifetime of the process.

**Why it mattered:**

- Defeated the entire pooling optimization.
- Memory leak proportional to request volume.
- Misled users reading the docs ("per-factory pooling" was fiction).

**Status:** Fixed. Pools are now per-encoding and owned by each `Compression` middleware instance. Verified by `TestNegotiator_PoolsAreStablePerEncoding` and stress-tested under `-race`.

### 2. Documentation Numbers Were Lies (FIXED)

`FEATURES.md` claimed 193 tests / 90.4%, `TODO_LIST.md` claimed 112 tests / 91.2%, and the audit itself had to be sanity-checked because nobody trusts docs anymore. These are now corrected and hardcoded counts were replaced with qualitative statements where possible.

**Status:** Fixed in this session. Ongoing discipline required.

### 3. Ghost Systems in Architecture Table (FIXED)

`health.go` and `server.go` were missing from `AGENTS.md` and `FEATURES.md` despite being shipped, tested features.

**Status:** Fixed in this session.

---

## e) WHAT WE SHOULD IMPROVE

1. **Doc verification protocol** — Every time tests change, update the numbers. Better: stop hardcoding counts; describe behavior and quality gates instead.
2. **Single source of truth** — `AGENTS.md`, `FEATURES.md`, and `TODO_LIST.md` duplicate overlapping information. Consider generating the architecture table or feature inventory from code/doc comments.
3. **Stop benchmarking lies** — The `util.go` fraud and the pool fiction both happened because claims were not empirically verified against real runtime behavior.
4. **AGENTS.md completeness** — Every `.go` source file should be in the architecture table. It is now complete, but a generation step would prevent future drift.
5. **CHANGELOG hygiene** — The `[Unreleased]` section now reflects util.go removal and pool fix. Keep it current per PR.
6. **Reduce doc cholesterol** — 18 status reports now in `docs/status/`. Historical value is real, but a rolling "current" report plus archive pruning would reduce noise.
7. **Pool test determinism** — The initial pool-mismatch test used `sync.Pool.Put`, which is non-deterministic under GC. The final version uses a custom `New` function. Make this pattern the default for future pool-error tests.
8. **Fuzz test corpus coverage** — The CORS wildcard fuzz is good; consider adding fuzz targets for `negotiateEncoding`, `HeaderSnapshot`, and `matchWildcardOrigin` boundaries.
9. **Benchmark regression CI** — Add a GitHub Actions step that fails if a benchmark regresses by more than a configurable threshold on PRs.
10. **Configurable compression deny-list** — A frequent user request and the largest remaining near-term gap.

---

## f) Top 25 Things to Get Done Next

Sorted by impact / effort ratio (highest first).

### Documentation & Testing (Low Effort, High Impact)

1. **Configurable content-type filtering via `CompressionConfig`** — Replace hardcoded deny-list with a configurable list; default to current set for backward compatibility.
2. **Add `MiddlewareStack` type** — Validate ordering at construction (Recovery outermost, Compression before ETag, Logging after Recovery, etc.).
3. **Add `ResponseWriter` capability interface** — Unify Hijack/Flush detection across wrappers; eliminate repeated type assertions.
4. **Add compression `Flush()` plain-mode test** — Cover the remaining uncompressed flush path.
5. **Add `ResponseRecorder.Flush()` no-op test with non-Flusher underlying writer** — Tiny gap, easy closure.
6. **Add `compressWriter.startCompression` buffered-write failure test** — Pool writer returns an error while replaying the buffer.
7. **Add boundary status tests for `shouldCompress()`** — 199/200, 299/300, etc.
8. **Fuzz `negotiateEncoding`** — Adversarial Accept-Encoding strings.
9. **Document brotli/zstd factory examples** — Show how to wire custom `WriterFactory` implementations without core deps.

### Architecture & Middleware (Medium Effort, High Impact)

10. **Add request body size limit middleware** — Simple, high value for API protection.
11. **Add rate-limiting middleware** — Token bucket or sliding window; keep it dependency-free.
12. **Add request/response metrics middleware** — Optional, via `expvar` or custom histograms.
13. **Pre-compute more per-request strings** — `allowCredentials` is already pre-computed; audit other middlewares for similar wins.
14. **Incremental ETag hashing evaluation** — Compute CRC32 during `Write` calls instead of on flush; measure complexity/benefit.
15. **Streaming ETag option** — Rolling hash for responses > 1MB; requires 304-short-circuit design decision.

### Server & Lifecycle (Medium Effort)

16. **Add `Server` graceful shutdown integration test** — Verify `Start()` + `Shutdown()` lifecycle with a real port.
17. **Add `HealthHandler` configurable readiness checker** — Allow custom readiness function instead of always `"up"`.
18. **Document recommended middleware ordering** — Already in README; expand with concrete examples and rationale.

### Quality & Tooling (Medium Effort)

19. **Benchmark regression CI step** — Fail PRs on significant benchmark regressions.
20. **Generate architecture table from code** — Single source of truth for `AGENTS.md`.
21. **Prune/archive old status reports** — Keep last N + tagged milestones.
22. **Add `govulncheck` to local dev shell** — Already in CI; make it a one-locally command.
23. **Add integration test for `Chain(Compression, ETag, CORS)`** — Multi-middleware realistic stack.
24. **Add timeout middleware test for already-canceled context** — Edge case in deadline enforcement.
25. **Review all `defer` error ignores** — Several `_ = cw.Close()` patterns; document why errors are safe to ignore or surface them.

---

## g) Top #1 Question I Cannot Figure Out Myself

**Should `health.go` and `server.go` be in this package at all?**

`httputil` began as a middleware library (`func(http.Handler) http.Handler`). `Server` and `HealthHandler` are application-level primitives — consumers of middleware, not middleware themselves. Every other file in the architecture table is middleware or direct middleware support.

Including them blurs package identity: is `httputil` a focused middleware library or an HTTP application framework? The `README.md` subtitle says "Composable HTTP middleware, utility primitives, and server lifecycle helpers" — the "utility primitives" and "server lifecycle" clauses were added retroactively.

**Arguments for keeping:** Useful, zero-dependency, tested, shipped in README examples. Users get a batteries-included package.

**Arguments for removing:** Package cohesion drops. `Server` has nothing to do with `ResponseRecorder` or `Compression`. A user who only wants CORS middleware also imports a health-check framework. The package name `httputil` is broad enough to justify either decision.

**I cannot resolve this without your domain intent.** Is `httputil` meant to be a focused middleware library (extract `Server`/`HealthHandler`) or a general HTTP utility package (keep them)?
