# Features

Honest feature inventory for `httputil`.

_Updated: 2026-07-26_

---

## FULLY FUNCTIONAL

### Core Middleware Suite (13 middlewares)

| Middleware       | File                                   | Config Type                                                 | Tests | Examples                     | Benchmarks                  | Fuzz              |
| ---------------- | -------------------------------------- | ----------------------------------------------------------- | ----- | ---------------------------- | --------------------------- | ----------------- |
| CORS             | `cors.go`                              | `CORSConfig` + `Validate()`                                 | Yes   | `ExampleCORS`                | `BenchmarkCORS`             | `FuzzCORS`        |
| ClientIP         | `clientip.go`, `context.go`            | —                                                           | Yes   | `ExampleClientIP`            | `BenchmarkClientIP`         | `FuzzClientIP`    |
| RequestID        | `requestid.go`, `id_generator.go`      | `RequestIDConfig` + `Validate()`, time-ordered ID generator | Yes   | `ExampleRequestID`           | `BenchmarkRequestID`        | `FuzzRequestID`   |
| SecurityHeaders  | `security.go`                          | `SecurityHeadersConfig` + `Validate()`                      | Yes   | `ExampleSecurityHeaders`     | `BenchmarkSecurityHeaders`  | —                 |
| Recovery         | `recovery.go`                          | `*slog.Logger`                                              | Yes   | `ExampleRecovery`            | `BenchmarkRecovery`         | —                 |
| Timeout          | `timeout.go`                           | `time.Duration`                                             | Yes   | `ExampleTimeout`             | `BenchmarkTimeout`          | —                 |
| Logging          | `logging.go`                           | `*slog.Logger`                                              | Yes   | `ExampleLogging`             | `BenchmarkLogging`          | —                 |
| ResponseRecorder | `recorder.go`                          | —                                                           | Yes   | `ExampleNewResponseRecorder` | `BenchmarkResponseRecorder` | —                 |
| Compression      | `compression.go`, `compress_writer.go` | `CompressionConfig` + `Validate()`, `WriterFactory` plugin  | Yes   | `ExampleCompression`         | `BenchmarkCompression`      | `FuzzCompression` |
| ETag             | `etag.go`                              | `ETagConfig` + `Validate()`                                 | Yes   | `ExampleETag`                | `BenchmarkETag`             | `FuzzETag`        |
| MaxBodySize      | `maxbodysize.go`                       | `int64`                                                     | Yes   | —                            | —                           | —                 |
| RateLimit        | `ratelimit.go`                         | `RateLimitConfig` + `Validate()`, `RateLimiter` interface   | Yes   | —                            | —                           | —                 |
| Metrics          | `metrics.go`                           | `MetricsConfig` + `Validate()`, `MetricsRecorder` interface | Yes   | —                            | —                           | —                 |

Plus `Chain()` in `recorder.go` for middleware composition.

### Error Classification System

- 5 error codes registered via `go-error-family`: `ErrCodeWriteFailed`, `ErrCodeHijackUnsupported`, `ErrCodeHijackFailed`, `ErrCodeCompressWriteFailed`, `ErrCodeETagWriteFailed`.
- `RegisterErrorClassifications()` maps stdlib HTTP errors to behavioral families (Transient vs Infrastructure).
- Message templates with `what/why/fix/wayOut` for all classified errors.
- Test coverage in `errors_test.go`.

### Shared ResponseWriter Wrapper

- `wrapper.go` extracts common `WriteHeader` buffering, `Hijack`, and `Flush` delegation.
- Embedded by `compressWriter` and `etagWriter`, eliminating ~80 lines of duplication.

### Infrastructure Types

- `MiddlewareStack` collects named middleware with duplicate prevention and ordering validation (Recovery must be outermost when present).
- `DetectCapabilities()` inspects a ResponseWriter for Hijacker/Flusher support.
- `DefaultIncompressibleTypes()` returns the default content-type deny-list for Compression.

### CORS Security

- `DenyUnmatched` option on `CORSConfig` — when true, withholds `Access-Control-Allow-Origin` for origins not in `AllowedOrigins`, preventing allowlist bypass via wildcard fallback.
- Wildcard origin matching (e.g., `*.example.com`) rejects lookalike domains (`*.example.com.evil.com`).
- `AllowCredentials: true` + `AllowAllOrigins: true` rejected at `Validate()` time (browsers reject this combination).

### Compression Performance

- `DefaultWriterFactoriesForLevel(level int)` returns a fresh default factory map (gzip + deflate + identity) at any compression level.
- `Compression()` uses `cfg.Level` to build default factories when `WriterFactories` is empty — `Level` is no longer ignored.
- Per-encoding `sync.Pool` (owned by the negotiator, one pool per encoding per `Compression` instance) reuses `gzip.Writer` and `flate.Writer` instances.
- Content-type deny-list skips incompressible formats (`image/`, `video/`, `audio/`, `application/gzip`, `application/zip`, `application/pdf`, etc.).
- Bounded buffering: only buffers up to `minSize`, then streams tail bytes directly.
- Buffer pre-allocated to `max(minSize, 512)` capacity to avoid intermediate reallocations.
- RFC 7231 `Accept-Encoding` negotiation with q-value parsing; server priority order is brotli > zstd > gzip > deflate > identity.

### ETag Correctness

- FNV-64a hash (64-bit, birthday bound ~4 billion) via configurable `HashFunc` field — replaced CRC32 to eliminate collision risk.
- RFC 7232 compliant `If-None-Match` list parsing (`etagInList`).
- All 2xx statuses cacheable (`isCacheableStatus()`).
- 1MB memory safety limit (`MaxBufferSize`).
- Zero-allocation hex encoding via stack arrays and lookup table.

### Rate Limiting

- Token bucket algorithm via `TokenBucketLimiter` (backed by `golang.org/x/time/rate`) — per-key limiters with fixed-rate refill up to burst capacity.
- `NewTokenBucketLimiter(rate, burst)` validates inputs — returns error if rate or burst is not positive.
- `EvictionTTL` field enables opt-in lazy eviction of idle buckets (amortized sweep at most once per TTL interval). Zero (default) preserves unbounded-growth behavior.
- Pluggable `RateLimiter` interface for custom backends (Redis, etc.).
- Configurable key extraction and custom denial handlers via `RateLimitConfig`.

### Query Parameter Helpers

- `ParseUintQuery(r *http.Request, key string) uint` — extracts a base-10 unsigned integer from a named query parameter. Returns 0 if missing, empty, or invalid.

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

### Server Lifecycle

- `ServerConfig` with `Validate()` — read, header, write, and idle timeout validation.
- `DefaultServerConfig()` — production defaults (`:8080`, 10s/5s/30s/60s timeouts).
- `NewServer()` wraps `http.Server` with lifecycle helpers.
- `Start()` is non-blocking and returns a `<-chan error` for listen errors.
- `Shutdown()` performs graceful shutdown respecting a context deadline.
- `Addr()` returns the configured listen address.

### Health Checks

- `HealthHandler()`, `LiveHandler()`, `ReadyHandler()` — Kubernetes-compatible endpoints.
- `ReadyHandlerWithProbe(ready func() bool)` — dependency-based readiness: returns 200 when ready, 503 when not.
- `RegisterHealth(mux)` registers `/health`, `/health/live`, and `/health/ready`.
- `HealthStatus` enum (`"up"` / `"down"`) and `HealthResponse` JSON type.

### Documentation

- `README.md` — feature overview, API table, usage examples, middleware ordering guidance.
- `doc.go` — package-level godoc.
- `AGENTS.md` — architecture reference, testing conventions, lint rules.
- `CHANGELOG.md` — version history.
- `ROADMAP.md` — long-term direction and v1.0 vision.
- `docs/DOMAIN_LANGUAGE.md` — domain glossary.
- Status reports in `docs/status/`.
- Execution plans in `docs/planning/`.

### Tooling & Quality Gates

- `golangci-lint` with ~70 linters, 0 issues.
- `go test ./...` passes across the full suite with 98.7% statement coverage (`httputil`), 98.3% (`httpspec`) — measured 2026-07-29 with race detection enabled.
- Fuzz tests for CORS origin matching, `ParseUintQuery`, `EvictionTTL`, and `HealthResponse` JSON encoding — all verified with `-fuzztime`.
- `go vet` clean.
- `.editorconfig` enforces consistent indentation and formatting across editors.
- Nix flake for reproducible development environment.
- GitHub Actions CI for tests and lint.
- Release workflow with `govulncheck`.

### Behavioral Spec Suite (`httpspec` subpackage)

- `httpspec.Run(t, handler)` validates any `http.Handler` against 18 standard HTTP behavior specs.
- `httpspec.RunSerial(t, handler)` variant for handlers with shared mutable state.
- Specs cover routing (index reachability, unknown paths, long URLs), method handling (HEAD, OPTIONS, TRACE, POST, CONNECT), response headers (Content-Type, Location on redirects, no duplicate headers, Accept header handling), and security (no leaked internals, no version fingerprints, no X-Powered-By, X-Content-Type-Options: nosniff).
- Extensible via `SkipSpec`, `WithExtraSpecs`, `WithIndexPath`.
- Helper builders: `ExpectStatus`, `ExpectNotStatus`, `ExpectHeader`, `ExpectHeaderAbsent`, `ExpectBodyContains`.
- Pure stdlib, no third-party dependencies.

---

## PARTIALLY DONE

### Test Coverage

Measured 2026-07-29 with `go test -race -coverprofile`: **98.7%** (`httputil`), **98.3%** (`httpspec`). All compression writer/pool, CORS, SecurityHeaders, Logging, and RateLimit functions at 100%. Remaining gaps:

- `computeETag` empty-body branch (94.4%).
- `scanAcceptEncoding` ordering tie-break (95.5%).
- `Compression` middleware Vary-header edge (95.5%).
- `Server.Shutdown` context-cancellation path (75%).
- `drawRandomBytes`/`refillRandomBuffer` crypto/rand error paths (67-88%).
- `httpspec.runSpecs`/`mustRequest` internal helpers (75-88%).

---

## PLANNED

### Near-term

- _(none — all near-term items are implemented)_

---

## WORTH CONSIDERING

- **Brotli / zstd / lz4 support** — now possible via the `WriterFactory` plugin interface without adding core dependencies. Provide documented examples rather than built-in encoders to keep the dependency policy intact.
- **Streaming ETag option** — evaluated and rejected. HTTP requires headers before body, so buffering is mandatory. The current FNV-64a + 1MB buffer approach is correct and optimal.
- **HTTP/2 Server Push integration test** — removed, HTTP/2 push is deprecated.
