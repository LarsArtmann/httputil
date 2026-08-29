# Comprehensive Status Report — httputil

**Date:** 2026-06-07 23:48 CEST
**Branch:** `master`
**Commits ahead of origin:** 0 (working tree clean, all pushed)
**Last commit:** `277fffa` docs(status): add final status report for compression-and-etag superb pass
**Go version:** 1.26.3
**Lines of code:** 3,646 across 26 Go files
**Test coverage:** 86.9% of statements
**Lint status:** 0 issues across ~70 linters

---

## a) FULLY DONE

### 1. Core Middleware Suite (10 middlewares)

| Middleware       | File(s)                     | Config                             | Tests | Examples             | Benchmarks             | Fuzz              |
| ---------------- | --------------------------- | ---------------------------------- | ----- | -------------------- | ---------------------- | ----------------- |
| CORS             | `cors.go`                   | `CORSConfig` + `Validate()`        | Yes   | Yes                  | `BenchmarkCORS`        | No                |
| ClientIP         | `clientip.go`, `context.go` | —                                  | Yes   | Yes                  | `BenchmarkClientIP`    | `FuzzClientIP`    |
| RequestID        | `requestid.go`              | `RequestIDConfig`                  | Yes   | No                   | No                     | No                |
| SecurityHeaders  | `security.go`               | `SecurityHeadersConfig`            | Yes   | No                   | No                     | No                |
| Recovery         | `recovery.go`               | —                                  | Yes   | No                   | No                     | No                |
| Timeout          | `timeout.go`                | `time.Duration`                    | Yes   | No                   | No                     | No                |
| Logging          | `logging.go`                | `*slog.Logger`                     | Yes   | No                   | No                     | No                |
| ResponseRecorder | `recorder.go`               | —                                  | Yes   | No                   | No                     | No                |
| Compression      | `compression.go`            | `CompressionConfig` + `Validate()` | Yes   | `ExampleCompression` | `BenchmarkCompression` | `FuzzCompression` |
| ETag             | `etag.go`                   | `ETagConfig`                       | Yes   | `ExampleETag`        | `BenchmarkETag`        | `FuzzETag`        |

Plus `Chain()` in `recorder.go` for middleware composition.

### 2. Error Classification System

- 7 error codes registered via `go-error-family`: `ErrCodeWriteFailed`, `ErrCodeHijackUnsupported`, `ErrCodeHijackFailed`, `ErrCodePushUnsupported`, `ErrCodePushFailed`, `ErrCodeCompressWriteFailed`, `ErrCodeETagWriteFailed`.
- `RegisterErrorClassifications()` maps stdlib HTTP errors to behavioral families (Transient vs Infrastructure).
- Message templates with `what/why/fix/wayOut` for all classified errors.
- Test coverage in `errors_test.go`.

### 3. Shared ResponseWriter Wrapper

- `wrapper.go` extracts common `WriteHeader` buffering, `Hijack`, `Push`, and `Flush` delegation.
- Embedded by `compressWriter` and `etagWriter`, eliminating ~80 lines of duplication.

### 4. Compression Performance

- `sync.Pool` keyed by gzip level reuses `gzip.Writer` instances.
- Content-type deny-list skips incompressible formats (`image/`, `video/`, `audio/`, `application/gzip`, `application/zip`, `application/pdf`, etc.).
- Bounded buffering: only buffers up to `minSize`, then streams tail bytes directly.

### 5. ETag Correctness

- RFC 7232 compliant `If-None-Match` list parsing (`etagInList`).
- All 2xx statuses cacheable (`isCacheableStatus()`).
- 1MB memory safety limit (`MaxBufferSize`).
- Zero-allocation hex encoding via stack arrays and lookup table.

### 6. Documentation

- `README.md` — feature overview, API table, usage examples, middleware ordering guidance.
- `doc.go` — package-level godoc.
- `AGENTS.md` — architecture reference, testing conventions, lint rules.
- `CHANGELOG.md` — version history.
- `docs/DOMAIN_LANGUAGE.md` — domain glossary.
- 7 status reports in `docs/status/`.
- 1 execution plan in `docs/planning/`.

### 7. Tooling & Quality Gates

- `golangci-lint` with ~70 linters, 0 issues.
- `go test ./...` — 114 tests passing.
- `go vet` clean.
- Nix flake for reproducible development environment.

---

## b) PARTIALLY DONE

### 1. Test Coverage (86.9%)

Not 100%. Gaps exist in:

- Error branches in `compression.go` (`startCompression` type mismatch, `Close` errors).
- Edge cases in `CORS` wildcard matching with unusual patterns.
- `ResponseRecorder` hijack/push failure paths.
- Some `util.go` branches (`itoa` negative numbers, `join` empty slices).

### 2. Benchmark Coverage

- Missing: `BenchmarkRequestID`, `BenchmarkSecurityHeaders`, `BenchmarkRecovery`, `BenchmarkTimeout`, `BenchmarkLogging`, `BenchmarkResponseRecorder`, `BenchmarkChain`.
- Only Compression, ETag, CORS, ClientIP, Itoa, and Join have benchmarks.

### 3. Fuzz Test Coverage

- Only `FuzzClientIP`, `FuzzCompression`, `FuzzETag` exist.
- Missing: `FuzzCORS`, `FuzzRequestID`, `FuzzChain`.

### 4. Example Functions

- Only 6 examples exist: `ExampleCORS`, `ExampleClientIP`, `ExampleChain`, `ExampleCompression`, `ExampleETag`, `ExampleLogging`.
- Missing: `ExampleRequestID`, `ExampleSecurityHeaders`, `ExampleRecovery`, `ExampleTimeout`, `ExampleResponseRecorder`.

### 5. Middleware Integration Tests

- Only `Chain(Compression, ETag)` is tested.
- No integration tests for other combinations (e.g., `Chain(Recovery, Logging, CORS)`).

---

## c) NOT STARTED

### 1. FEATURES.md

No centralized feature inventory file exists. The README serves this role partially, but there's no honest status tracking (DONE / PARTIALLY DONE / PLANNED / WORTH CONSIDERING) per feature.

### 2. TODO_LIST.md

No project-wide TODO list exists. Outstanding work is scattered across status reports, AGENTS.md, and memory.

### 3. HTTP/2 Server Push Tests

`wrapper.go` implements `Push()`, and both writers delegate it, but no dedicated test exercises HTTP/2 Push in an end-to-end middleware context.

### 4. WebSocket Upgrade Path Testing

`Hijack()` is tested for both writers, but no test verifies that a WebSocket upgrade (the most common `Hijack` use case) works correctly through Compression or ETag.

### 5. Content-Length Preservation for Small Responses

When a handler sets `Content-Length` and the response is below `minSize`, the original `Content-Length` is not explicitly preserved (it may be written through, but there's no test verifying this).

### 6. Deflate Support

`compress/flate` is in the stdlib, but only gzip is supported. No `Accept-Encoding: deflate` handling exists.

### 7. Accept-Encoding Quality Value Parsing

Current implementation uses `strings.Contains(req.Header.Get("Accept-Encoding"), "gzip")`. No RFC-compliant `q=0.8` parsing.

### 8. Brotli Support

Blocked by single-dependency policy. No decision made on whether to relax constraint, add plugin interface, or document limitation.

### 9. Streaming ETag (No Buffering)

No option to compute ETag on a rolling hash and stream body without buffering. Would require breaking 304 short-circuit semantics or significant complexity.

### 10. Configurable Content-Type Filtering

`incompressiblePrefixes` is hardcoded. No `CompressionConfig` field for custom allow/deny lists.

### 11. Middleware Ordering Validation

No compile-time or runtime check that middleware is applied in the correct order. A `MiddlewareStack` type with ordering rules is not implemented.

### 12. ResponseWriter Interface Hierarchy

Each middleware still detects `Hijacker`/`Pusher`/`Flusher` individually via type assertions. No unified interface or helper exists.

### 13. Performance Benchmarks for All Middlewares

Only 6 of 10 middlewares have benchmarks.

### 14. CI/CD Pipeline

No GitHub Actions workflow exists for running tests, lint, and benchmarks on PR.

---

## d) TOTALLY FUCKED UP!

### Nothing is totally fucked up.

The codebase compiles, all 114 tests pass, lint is clean, coverage is at 86.9%, benchmarks exist for the performance-critical paths, and the architecture is consistent.

However, the following are **architectural debts** that will cause pain if the project grows:

1. **No FEATURES.md or TODO_LIST.md** — makes it hard for new contributors (or future AI sessions) to understand what's done and what's next.
2. **Missing CI** — quality gates are manual. A contributor could break tests or lint without knowing until someone runs it locally.
3. **No middleware ordering guardrails** — `Chain()` silently accepts any order. Wrong ordering (e.g., ETag outside Compression) produces subtly broken behavior that tests won't catch.
4. **Brotli decision paralysis** — the single-dependency policy blocks a genuinely useful feature. Indecision is worse than either keeping the constraint or relaxing it.
5. **ETag always buffers** — even with the 1MB limit, this is a footgun for large-response APIs.

None of these are "fucked up" in the sense of breaking the build. They are **strategic gaps**.

---

## e) WHAT WE SHOULD IMPROVE

### Critical (Do before next release)

1. **Add CI pipeline** — GitHub Actions running `go test`, `golangci-lint run`, and `go test -bench` on every PR.
2. **Create FEATURES.md** — honest inventory of every feature with status.
3. **Create TODO_LIST.md** — centralized, actionable task list.
4. **Add benchmarks for remaining middlewares** — at minimum: `BenchmarkRequestID`, `BenchmarkSecurityHeaders`, `BenchmarkRecovery`, `BenchmarkResponseRecorder`.

### High Impact / Low Effort

5. **Add example functions for missing middlewares** — `ExampleRequestID`, `ExampleSecurityHeaders`, `ExampleRecovery`, `ExampleTimeout`.
6. **Add fuzz tests for CORS and RequestID**.
7. **Add integration tests for common middleware chains** — e.g., `Chain(Recovery, Logging, CORS, handler)`.
8. **Document brotli policy decision** — either relax constraint, add `WriterFactory` plugin, or document "gzip only" in README.
9. **Add WebSocket upgrade test** — verify `Hijack()` works correctly through Compression + ETag.
10. **Add `Content-Length` preservation test** for small responses that don't trigger compression.

### Medium Impact / Medium Effort

11. **Implement deflate support** using `compress/flate`.
12. **Add `Accept-Encoding` quality value parsing**.
13. **Make content-type filtering configurable** via `CompressionConfig`.
14. **Add `MiddlewareStack` type** with ordering validation.
15. **Add a `ResponseWriter` capability interface** to unify Hijack/Push/Flush detection.

### Long-term / Architectural

16. **Consider streaming ETag option** using a rolling hash with chunked writes.
17. **Evaluate brotli dependency relaxation** if performance gains justify it.
18. **Add request/response metrics middleware** (optional, using `expvar` or custom).
19. **Add rate-limiting middleware** (sliding window, token bucket).
20. **Add request body size limit middleware**.

---

## f) Top #25 Things We Should Get Done Next

Sorted by **impact / effort ratio** (highest impact per unit of work first):

| #  | Task                                          | Impact   | Effort | Category        |
| -- | --------------------------------------------- | -------- | ------ | --------------- |
| 1  | Add GitHub Actions CI (test + lint + bench)   | Critical | 20 min | Infrastructure  |
| 2  | Create FEATURES.md                            | High     | 15 min | Documentation   |
| 3  | Create TODO_LIST.md                           | High     | 15 min | Documentation   |
| 4  | Add benchmarks for remaining middlewares      | Medium   | 30 min | Observability   |
| 5  | Add example functions for missing middlewares | Medium   | 20 min | DX              |
| 6  | Add fuzz tests for CORS and RequestID         | Medium   | 20 min | Quality         |
| 7  | Add integration tests for common chains       | Medium   | 30 min | Correctness     |
| 8  | Document brotli policy decision               | Medium   | 10 min | Documentation   |
| 9  | Add WebSocket upgrade test                    | Low      | 15 min | Coverage        |
| 10 | Add Content-Length preservation test          | Low      | 10 min | Correctness     |
| 11 | Implement deflate support                     | Medium   | 30 min | Feature         |
| 12 | Add Accept-Encoding quality parsing           | Low      | 20 min | Correctness     |
| 13 | Make content-type filtering configurable      | Low      | 15 min | Configurability |
| 14 | Add MiddlewareStack with ordering validation  | Medium   | 45 min | Architecture    |
| 15 | Add ResponseWriter capability interface       | Low      | 30 min | Architecture    |
| 16 | Add streaming ETag option                     | High     | 60 min | Performance     |
| 17 | Evaluate brotli dependency relaxation         | Medium   | 30 min | Decision        |
| 18 | Add request/response metrics middleware       | Medium   | 45 min | Feature         |
| 19 | Add rate-limiting middleware                  | Medium   | 60 min | Feature         |
| 20 | Add request body size limit middleware        | Low      | 20 min | Safety          |
| 21 | Add HTTP/2 Server Push integration test       | Low      | 15 min | Coverage        |
| 22 | Add `ExampleResponseRecorder`                 | Low      | 10 min | DX              |
| 23 | Add `BenchmarkChain`                          | Low      | 10 min | Observability   |
| 24 | Improve test coverage to 90%+                 | Medium   | 60 min | Quality         |
| 25 | Add `go test -race` to CI                     | High     | 5 min  | Safety          |

---

## g) Top #1 Question I Cannot Figure Out Myself

**How do we decide whether to keep the single-dependency policy (`depguard` only allows stdlib + `go-error-family`) or relax it for brotli?**

Brotli achieves 15–25% better compression than gzip. It's table stakes for modern web servers. But our current policy is strict: only stdlib + one external dep.

Options:

1. **Keep the constraint** — document "gzip only" as an intentional limitation. Users who need brotli use a different middleware.
2. **Relax the constraint** for a zero-transitive-dep brotli library like `github.com/andybalholm/brotli`. This adds one more dependency but delivers real performance gains.
3. **Add a `WriterFactory` plugin interface** — `CompressionConfig.WriterFactory func(io.Writer) io.WriteCloser`. Default uses stdlib gzip. Users inject brotli if they want it. Keeps core dependency-free but adds API complexity.

**My recommendation:** Option 3 — the plugin interface. It preserves the purity of the core package while allowing users to opt into brotli (or zstd, or lz4) without us maintaining those dependencies.

**But:** This adds API surface area and a non-obvious configuration point. Is the complexity worth it for a utility package? Should we instead keep compression simple (gzip + deflate only) and document that users should use a dedicated compression library (like `chi/middleware` or `klauspost/compress`) if they need more?

**I need a decision on dependency policy before proceeding with any compression enhancements.**
