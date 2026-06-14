# Features

Honest feature inventory for `httputil`.

_Updated: 2026-06-14_

---

## FULLY FUNCTIONAL

### Core Middleware Suite (10 middlewares)

| Middleware       | File                        | Config Type                            | Tests | Examples                     | Benchmarks                  | Fuzz              |
| ---------------- | --------------------------- | -------------------------------------- | ----- | ---------------------------- | --------------------------- | ----------------- |
| CORS             | `cors.go`                   | `CORSConfig` + `Validate()`            | Yes   | `ExampleCORS`                | `BenchmarkCORS`             | `FuzzCORS`        |
| ClientIP         | `clientip.go`, `context.go` | —                                      | Yes   | `ExampleClientIP`            | `BenchmarkClientIP`         | `FuzzClientIP`    |
| RequestID        | `requestid.go`, `id_generator.go` | `RequestIDConfig` + `Validate()`, time-ordered ID generator | Yes   | `ExampleRequestID`           | `BenchmarkRequestID`        | `FuzzRequestID`   |
| SecurityHeaders  | `security.go`               | `SecurityHeadersConfig` + `Validate()` | Yes   | `ExampleSecurityHeaders`     | `BenchmarkSecurityHeaders`  | —                 |
| Recovery         | `recovery.go`               | `*slog.Logger`                         | Yes   | `ExampleRecovery`            | `BenchmarkRecovery`         | —                 |
| Timeout          | `timeout.go`                | `time.Duration`                        | Yes   | `ExampleTimeout`             | `BenchmarkTimeout`          | —                 |
| Logging          | `logging.go`                | `*slog.Logger`                         | Yes   | `ExampleLogging`             | `BenchmarkLogging`          | —                 |
| ResponseRecorder | `recorder.go`               | —                                      | Yes   | `ExampleNewResponseRecorder` | `BenchmarkResponseRecorder` | —                 |
| Compression      | `compression.go`, `compress_writer.go` | `CompressionConfig` + `Validate()`, `WriterFactory` plugin | Yes   | `ExampleCompression`         | `BenchmarkCompression`      | `FuzzCompression` |
| ETag             | `etag.go`                   | `ETagConfig` + `Validate()`            | Yes   | `ExampleETag`                | `BenchmarkETag`             | `FuzzETag`        |

Plus `Chain()` in `recorder.go` for middleware composition.

### Error Classification System

- 5 error codes registered via `go-error-family`: `ErrCodeWriteFailed`, `ErrCodeHijackUnsupported`, `ErrCodeHijackFailed`, `ErrCodeCompressWriteFailed`, `ErrCodeETagWriteFailed`.
- `RegisterErrorClassifications()` maps stdlib HTTP errors to behavioral families (Transient vs Infrastructure).
- Message templates with `what/why/fix/wayOut` for all classified errors.
- Test coverage in `errors_test.go`.

### Shared ResponseWriter Wrapper

- `wrapper.go` extracts common `WriteHeader` buffering, `Hijack`, and `Flush` delegation.
- Embedded by `compressWriter` and `etagWriter`, eliminating ~80 lines of duplication.

### Compression Performance

- Per-factory `sync.Pool` keyed by `WriterFactory` pointer reuses `gzip.Writer` and `flate.Writer` instances.
- Content-type deny-list skips incompressible formats (`image/`, `video/`, `audio/`, `application/gzip`, `application/zip`, `application/pdf`, etc.).
- Bounded buffering: only buffers up to `minSize`, then streams tail bytes directly.
- Buffer pre-allocated to `max(minSize, 512)` capacity to avoid intermediate reallocations.
- RFC 7231 `Accept-Encoding` negotiation with q-value parsing; server priority order is brotli > zstd > gzip > deflate > identity.

### ETag Correctness

- RFC 7232 compliant `If-None-Match` list parsing (`etagInList`).
- All 2xx statuses cacheable (`isCacheableStatus()`).
- 1MB memory safety limit (`MaxBufferSize`).
- Zero-allocation hex encoding via stack arrays and lookup table.

### Context Helpers

- `WithClientIP()` stores client IP in request context.
- `ClientIPFromContext()` retrieves it downstream.
- `ClientIPMiddleware()` wraps a handler to inject client IP into context.
- `RequestIDFromContext()` retrieves request ID from context.

### Request ID Generator

- 16-byte time-ordered ID: Unix seconds (4 B) + atomic counter (4 B) + random tail (8 B).
- 32-character lowercase hex output, lexicographically sortable and monotonic within a second.
- Amortized `crypto/rand` via a process-wide 2048-byte buffer (one syscall every ~256 IDs).
- Thread-safe refill with mutex and double-checked atomic slot allocation.

### Documentation

- `README.md` — feature overview, API table, usage examples, middleware ordering guidance.
- `doc.go` — package-level godoc.
- `AGENTS.md` — architecture reference, testing conventions, lint rules.
- `CHANGELOG.md` — version history.
- `docs/DOMAIN_LANGUAGE.md` — domain glossary.
- Status reports in `docs/status/`.
- Execution plans in `docs/planning/`.

### Tooling & Quality Gates

- `golangci-lint` with ~70 linters, 0 issues.
- `go test ./...` — 193 tests passing.
- 90.4% coverage.
- `go vet` clean.
- Nix flake for reproducible development environment.
- GitHub Actions CI for tests and lint.
- Release workflow with `govulncheck`.

---

## PARTIALLY DONE

### Test Coverage (91.2%)

Not 100% (target met at 90%+). Gaps exist in:

- Error branches in `compression.go` (`startCompression` type mismatch, `Close` errors).
- Edge cases in `CORS` wildcard matching with unusual patterns.
- `ResponseRecorder` hijack failure paths.
- Some `util.go` branches (`itoa` negative numbers, `join` empty slices).

---

## PLANNED

### Near-term

- Add WebSocket upgrade test through Compression + ETag.
- Add `Content-Length` preservation test for small responses.

### Medium-term

- Make content-type filtering configurable via `CompressionConfig`.
- Add `MiddlewareStack` type with ordering validation.
- Add a `ResponseWriter` capability interface to unify Hijack/Flush detection.

---

## WORTH CONSIDERING

- **Brotli / zstd / lz4 support** — now possible via the `WriterFactory` plugin interface without adding core dependencies. Provide documented examples rather than built-in encoders to keep the dependency policy intact.
- **Streaming ETag option** — compute ETag on a rolling hash and stream body without buffering. Would require breaking 304 short-circuit semantics or significant complexity.
- **Request/response metrics middleware** — optional, using `expvar` or custom histograms.
- **Rate-limiting middleware** — sliding window or token bucket.
- **Request body size limit middleware**.
- **HTTP/2 Server Push integration test** — removed, HTTP/2 push is deprecated.
