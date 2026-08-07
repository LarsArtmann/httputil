# Features

Honest feature inventory for `httputil`.

_Updated: 2026-08-07 — full docs-health audit. Ghost CHANGELOG entries removed, coverage corrected to 97.0%, benchmark (44) / example (24) / fuzz (19) counts verified against source, ETag adapter + Decompression rows updated. All claims checked against current source with `go test -race -coverprofile`._

---

## FULLY FUNCTIONAL

### Core Middleware Suite (17 middlewares)

| Middleware               | File                                   | Config Type                                                   | Tests | Examples                            | Benchmarks                    | Fuzz                |
| ------------------------ | -------------------------------------- | ------------------------------------------------------------- | ----- | ----------------------------------- | ----------------------------- | ------------------- |
| CORS                     | `cors.go`                              | `CORSConfig` + `Validate()`                                   | Yes   | `ExampleCORS`                       | `BenchmarkCORS`               | `FuzzCORS`          |
| ClientIP                 | `clientip.go`, `context.go`            | —                                                             | Yes   | `ExampleClientIP`                   | `BenchmarkClientIP`           | `FuzzClientIP`      |
| RequestID                | `requestid.go`, `id_generator.go`      | `RequestIDConfig` + `Validate()`, time-ordered ID generator   | Yes   | `ExampleRequestID`                  | `BenchmarkRequestID`          | `FuzzRequestID`     |
| SecurityHeaders          | `security.go`                          | `SecurityHeadersConfig` + `Validate()`                        | Yes   | `ExampleSecurityHeaders`            | `BenchmarkSecurityHeaders`    | —                   |
| Recovery                 | `recovery.go`                          | `*slog.Logger`                                                | Yes   | `ExampleRecovery`                   | `BenchmarkRecovery`           | —                   |
| Timeout                  | `timeout.go`                           | `time.Duration`                                               | Yes   | `ExampleTimeout`                    | `BenchmarkTimeout`            | —                   |
| Logging                  | `logging.go`                           | `*slog.Logger`                                                | Yes   | `ExampleLogging`                    | `BenchmarkLogging`            | —                   |
| ResponseRecorder         | `recorder.go`                          | —                                                             | Yes   | `ExampleNewResponseRecorder`        | `BenchmarkResponseRecorder`   | —                   |
| Compression              | `compression.go`, `compress_writer.go` | `CompressionConfig` + `Validate()`, `WriterFactory` plugin    | Yes   | `ExampleCompression`                | `BenchmarkCompression*`        | `FuzzCompression`   |
| MaxBodySize              | `maxbodysize.go`                       | `MaxBodySizeConfig` + `Validate()`, `MaxBodySizeMiddleware()` | Yes   | —                                   | —                             | —                   |
| RateLimit _(deprecated)_ | `ratelimit.go`                         | `RateLimitConfig` + `Validate()`, `RateLimiter` interface     | Yes   | —                                   | `BenchmarkTokenBucketLimiter` | —                   |
| Metrics                  | `metrics.go`                           | `MetricsConfig` + `Validate()`, `MetricsRecorder` interface   | Yes   | —                                   | —                             | —                   |
| Server-Timing            | `server_timing.go`                     | —                                                             | Yes   | `ExampleServerTimingMiddleware`     | `BenchmarkServerTiming*`      | `FuzzServerTiming*` |
| CSRF                     | `csrf.go`                              | `CSRFConfig` + `Validate()`                                   | Yes   | `ExampleCSRFMiddleware`             | `BenchmarkCSRFMiddleware*`    | `FuzzCSRF*` (6)     |
| KeyedRateLimit           | `ratelimit_keyed.go`                   | `KeyedRateLimiterConfig` + `Validate()`                       | Yes   | `ExampleKeyedRateLimiterMiddleware` | `BenchmarkKeyedRateLimiter*`  | —                   |
| Decompression            | `decompression.go`                     | `DecompressionConfig` + `Validate()`, bomb protection         | Yes   | —                                   | `BenchmarkDecompression*`     | `FuzzDecompression` |
| ETag                     | `etag.go` (adapter)                    | `etag.ETagConfig` (from go-etag)                              | Yes   | `ExampleETag`                       | —                             | —                   |

Plus `Chain()` in `recorder.go` for middleware composition.

### Error Classification System

- 4 error codes registered via `go-error-family`: `ErrCodeWriteFailed`, `ErrCodeHijackUnsupported`, `ErrCodeHijackFailed`, `ErrCodeCompressWriteFailed`. Plus 3 ETag error codes from `go-etag` (`etag.ErrCodeETagWriteFailed`, `etag.ErrCodeInvalidConfig`, `etag.ErrCodeHashWriteFailed`) registered via `RegisterErrorClassifications()`.
- `RegisterErrorClassifications()` maps stdlib HTTP errors to behavioral families (Transient vs Infrastructure).
- CSRF middleware uses `go-error-family` directly: `ErrCSRFInvalid` (Rejection family) and `ErrCSRFConfig` (Infrastructure family), plus inline `NewInfrastructure` errors for config validation failures.
- Message templates with `what/why/fix/wayOut` for all classified errors.
- Test coverage in `errors_test.go`.

### Shared ResponseWriter Wrapper

- `wrapper.go` extracts common `WriteHeader` buffering, `Hijack`, and `Flush` delegation.
- Embedded by `compressWriter`, eliminating ~80 lines of duplication.

### Infrastructure Types

- `MiddlewareStack` collects named middleware with duplicate prevention and ordering validation (Recovery must be outermost when present). 12 well-known `Middleware*` constants (Recovery, Logging, RequestID, CORS, SecurityHeaders, Compression, Timeout, ClientIP, CSRF, ServerTiming, KeyedRateLimit, ETag).
- `DetectCapabilities()` inspects a ResponseWriter for Hijacker/Flusher support.
- `DefaultIncompressibleTypes()` returns the default content-type deny-list for Compression.

### CORS Security

- `DenyUnmatched` option on `CORSConfig` — when true, withholds `Access-Control-Allow-Origin` for origins not in `AllowedOrigins`, preventing allowlist bypass via wildcard fallback. Default is `true` since v0.7.0.
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
- Single error-classification choke point: compress write failures funnel through `compressWriteError` with `encoding` context.

### Rate Limiting

- **Token bucket algorithm** via `TokenBucketLimiter` (deprecated, backed by `golang.org/x/time/rate`) — per-key limiters with fixed-rate refill up to burst capacity. Slated for removal at v1.0.
- **Keyed rate limiting** via `KeyedRateLimiter` (new in v0.8.0) — O(log n) min-heap eviction, `MaxKeys` cap, lazy TTL eviction, `Retry-After` headers, and a monitoring API (`ActiveKeys()`). This is the recommended API going forward.
- `NewTokenBucketLimiter(rate, burst)` validates inputs — returns error if rate or burst is not positive.
- `EvictionTTL` field on `KeyedRateLimiterConfig` enables opt-in lazy eviction of idle buckets. Zero (default) preserves unbounded-growth behavior.
- Pluggable `KeyExtractor` interface (`KeyExtractorFromRemoteAddr`, `KeyExtractorFromClientIP`).
- Pluggable `RejectionHandler` for custom 429 responses.
- Migration guide: `docs/migrating-to-keyed-rate-limiter.md`.

### CSRF Protection

- **Double-submit cookie** middleware via `justinas/nosurf` (new in v0.8.0).
- `CSRFMiddleware` and `CSRFResponseHeaderMiddleware` for simple and header-based CSRF defense.
- `ValidateCSRF` for per-handler validation of `*http.Request`.
- `CSRFTokenHXHeaders`, `CSRFTokenHTMLMeta`, `CSRFTokenFormField` for HTMX/templ integration.
- `ConfigureNosurfHandler` for fine-grained control over the underlying nosurf handler.
- `WithCSRFToken`, `CSRFTokenFromContext`, `CSRFTokenFromRequest` for token retrieval.
- `InvalidateCSRFCookie` for explicit token rotation.
- `TranslateCSRFHeaders` for HTMX-style header forwarding.
- `isTrustedProxy` for secure `X-Forwarded-Proto` handling.
- `CSRFConfig.Validate()` enforces secure defaults (`SameSite=None` requires `Secure`).
- Domain-level `TrustedOrigins` and trusted-proxy CIDR allowlists.

### Server-Timing

- W3C Server-Timing header implementation (new in v0.8.0).
- `ServerTimingMiddleware` and `ServerTimingMiddlewareWhen` for conditional instrumentation.
- `MeasureServerTiming` for context-aware measurement.
- `WrapServerTiming` for manual wrapping without middleware.
- `RecordServerTiming`, `WithServerTiming`, `ServerTimingFromContext` for handler-internal recording.
- CRLF-injection-safe header values (sanitized via `escapeQuotedString` and CRLF replacement).
- Hijacker, Flusher, Pusher delegation via `delegatingWriter`.

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
- Exact-byte JSON output (`{"status":"up"}\n`) enforced by tests.

### Documentation

- `README.md` — feature overview, API table, usage examples, middleware ordering guidance.
- `doc.go` — package-level godoc.
- `AGENTS.md` — architecture reference, testing conventions, lint rules.
- `CHANGELOG.md` — version history.
- `ROADMAP.md` — long-term direction and v1.0 vision.
- `docs/v1-stability.md` — v1.0 frozen API surface.
- `docs/DOMAIN_LANGUAGE.md` — domain glossary.
- `docs/migrating-to-keyed-rate-limiter.md` — deprecation migration guide.
- `docs/integrations/` — extensibility examples (brotli, redis, prometheus).
- Status reports in `docs/status/`.
- Execution plans in `docs/planning/`.

### Tooling & Quality Gates

- `golangci-lint` with ~70 linters, 0 issues.
- `go test -race ./...` passes across the full suite with **97.0% statement coverage** (`httputil`), **99.3%** (`httpspec`) — measured 2026-08-07 with race detection enabled.
- 19 fuzz tests covering CORS (origin matching, wildcard patterns), Compression (compression writer state), RequestID, ClientIP, `ParseUintQuery`, `EvictionTTL`, `HealthResponse` encoding, Server-Timing (header value + middleware), Decompression (malformed compressed bodies), and CSRF (6 functions: TrustedProxies CIDR, TrustedOrigins, `isTrustedProxy`, token validation, `remoteHostAndIP`, origin headers). CORS, query params, eviction, health, compression, decompression, and CSRF fuzz tests verified with `-fuzztime`.
- 44 benchmarks and 24 example functions across both packages.
- `go vet` clean.
- `.editorconfig` enforces consistent indentation and formatting across editors.
- Nix flake for reproducible development environment.
- GitHub Actions CI for tests, lint, and `govulncheck`.
- Release workflow with `govulncheck`, CHANGELOG link validation, and pre-release self-review step (`docs/RELEASE.md`).
- GitHub Actions pinned to commit SHAs (supply-chain hardening).

### Behavioral Spec Suite (`httpspec` subpackage)

- `httpspec.Run(t, handler)` validates any `http.Handler` against 18 standard HTTP behavior specs.
- `httpspec.RunSerial(t, handler)` variant for handlers with shared mutable state.
- 7 pre-built extra specs available via `WithExtraSpecs`: `CORSSpecs()` (4 specs: allow-origin, allow-credentials, Vary: Origin, wildcard-no-credentials) and `RateLimitSpecs()` (3 specs: Retry-After on reject, X-RateLimit-* headers on reject, hint headers on allow). Total: 25 specs when all are included.
- Specs cover routing (index reachability, unknown paths, long URLs), method handling (HEAD, OPTIONS, TRACE, POST, CONNECT), response headers (Content-Type, Location on redirects, no duplicate headers, Accept header handling), and security (no leaked internals, no version fingerprints, no X-Powered-By, X-Content-Type-Options: nosniff).
- Extensible via `SkipSpec`, `WithExtraSpecs`, `WithIndexPath`.
- Helper builders: `ExpectStatus`, `ExpectNotStatus`, `ExpectHeader`, `ExpectHeaderAbsent`, `ExpectBodyContains`.
- Pure stdlib, no third-party dependencies.

---

## PARTIALLY DONE

### Test Coverage — sub-100% functions (defensive code paths)

Measured 2026-08-07 with `go test -race -coverprofile`: **97.0%** (`httputil`), **99.3%** (`httpspec`). The remaining sub-100% functions are documented defensive code paths:

**New middleware (CSRF, Server-Timing, KeyedRateLimit):**

- `csrf.go:209 ConfigureNosurfHandler` — 81.8%. TrustedOrigins parse error branch (internal to nosurf).
- `csrf.go:506 CSRFTokenHXHeaders` — 71.4%. `json.Marshal` error on `map[string]string` (practically unreachable).
- `csrf.go:542 CSRFTestToken` — 92.9%. Internal nosurf error branches.
- `csrf.go:577 ValidateCSRF` — 92.9%. Nosurf TrustedOrigins parse failure paths.
- `compression.go:171 Compression` — 95.5%. Vary-header identity-append edge (reachable only via direct unit construction).
- `compression_negotiator.go:148 scanAcceptEncoding` — 95.5%. q-value tie-break with identical values (low priority).
- `ratelimit_keyed.go:165 buildKeyedRateLimiter` — 92.9%. Defensive config validation edge.
- `ratelimit_keyed.go:282 limiter` — 78.3%. RLock-hit-but-TTL-expired path (race condition).
- `ratelimit_keyed.go:346 evictOldestIfAtCapacity` — 88.9%. Stale-heap-mismatch continue branch.
- `security.go:92 SecurityHeaders` — 84.2%. Custom header application edge cases.

**Decompression middleware:**

- `decompression.go:62 Decompression` — 78.1%. Encoding-filter reject path (unreachable `default:` switch case when allowed list contains only gzip/deflate).

**Pre-existing (error-injection / internal paths):**

- `id_generator.go:100 drawRandomBytes` — 66.7%. `crypto/rand` error path (requires kernel-level fault injection).
- `id_generator.go:139 refillRandomBuffer` — 87.5%. `crypto/rand` partial-read error path.
- `httpspec.go:232 runSpecs` — 88.2%. Internal option error paths.

**Honest assessment:** The remaining sub-100% functions are documented as defensive code paths. Closing them would require either (a) kernel-level fault injection for `crypto/rand`, (b) direct unit-only construction of internal types, or (c) test infrastructure that doesn't exist in this project.

---

## PLANNED

### Near-term

- _(none — all near-term items shipped in v0.9.0/v0.9.1)_

---

## WORTH CONSIDERING

- **Brotli / zstd / lz4 support** — now possible via the `WriterFactory` plugin interface without adding core dependencies. Documentation examples at `docs/integrations/brotli-zstd.md`; built-in encoders are deliberately not added to preserve the dependency policy.
- **Rate limiter `context.Context` cancellation** — add `context.Context` support to the rate limiter interface. Deferred to v1.0 (API design decision). See [ROADMAP.md](ROADMAP.md).
