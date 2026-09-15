# Features

Honest feature inventory for `httputil`.

_Updated: 2026-09-15 — CSRF `SameSite=None` fallback (+ `AllowInsecureSameSiteNone`) and CORS `AllowPrivateNetwork` swept in. Coverage re-measured with race detection 2026-09-14: 97.2% (`httputil`, library packages), 98.6% (`httpspec`)._

---

## FULLY FUNCTIONAL

### Core Middleware Suite (18 middlewares)

| Middleware          | File                                   | Config Type                                                   | Tests | Examples                            | Benchmarks                                             | Fuzz                   |
| ------------------- | -------------------------------------- | ------------------------------------------------------------- | ----- | ----------------------------------- | ------------------------------------------------------ | ---------------------- |
| CORS                | `cors.go`                              | `CORSConfig` + `Validate()`                                   | Yes   | `ExampleCORS`                       | `BenchmarkCORS`                                        | `FuzzCORS*` (4)        |
| ClientIP            | `clientip.go`, `context.go`            | —                                                             | Yes   | `ExampleClientIP`                   | `BenchmarkClientIP`                                    | `FuzzClientIP`         |
| RequestID           | `requestid.go`, `id_generator.go`      | `RequestIDConfig` + `Validate()`, time-ordered ID generator   | Yes   | `ExampleRequestID`                  | `BenchmarkRequestID`                                   | `FuzzRequestID`        |
| SecurityHeaders     | `security.go`                          | `SecurityHeadersConfig` + `Validate()`                        | Yes   | `ExampleSecurityHeaders`            | `BenchmarkSecurityHeaders`                             | —                      |
| Recovery            | `recovery.go`                          | `*slog.Logger`                                                | Yes   | `ExampleRecovery`                   | `BenchmarkRecovery`                                    | —                      |
| Timeout             | `timeout.go`                           | `time.Duration`                                               | Yes   | `ExampleTimeout`                    | `BenchmarkTimeout`                                     | —                      |
| Logging             | `logging.go`                           | `*slog.Logger`                                                | Yes   | `ExampleLogging`                    | `BenchmarkLogging`                                     | —                      |
| ResponseRecorder    | `recorder.go`                          | —                                                             | Yes   | `ExampleNewResponseRecorder`        | `BenchmarkResponseRecorder`                            | —                      |
| Compression         | `compression.go`, `compress_writer.go` | `CompressionConfig` + `Validate()`, `WriterFactory` plugin    | Yes   | `ExampleCompression`                | `BenchmarkCompression*`                                | `FuzzCompression*` (3) |
| MaxBodySize         | `maxbodysize.go`                       | `MaxBodySizeConfig` + `Validate()`, `MaxBodySizeMiddleware()` | Yes   | `ExampleMaxBodySize`                | `BenchmarkMaxBodySize`                                 | `FuzzMaxBodySize`      |
| Metrics             | `metrics.go`                           | `MetricsConfig` + `Validate()`, `MetricsRecorder` interface   | Yes   | `ExampleMetrics`                    | `BenchmarkMetricsMiddleware*`                          | —                      |
| Server-Timing       | `server_timing/server_timing.go`       | —                                                             | Yes   | `ExampleServerTimingMiddleware`     | `BenchmarkServerTiming*`                               | `FuzzServerTiming*`    |
| CSRF                | `csrf.go`                              | `CSRFConfig` + `Validate()`                                   | Yes   | `ExampleCSRFMiddleware`             | `BenchmarkCSRFMiddleware*`                             | `FuzzCSRF*` (6)        |
| KeyedRateLimit      | `ratelimit_keyed.go`                   | `KeyedRateLimiterConfig` + `Validate()`                       | Yes   | `ExampleKeyedRateLimiterMiddleware` | `BenchmarkKeyedRateLimiter*`                           | —                      |
| Decompression       | `decompression.go`                     | `DecompressionConfig` + `Validate()`, bomb protection         | Yes   | `ExampleDecompression`              | `BenchmarkDecompression*`                              | `FuzzDecompression`    |
| ETag _(deprecated)_ | `etag.go` (adapter)                    | `etag.ETagConfig` (from go-etag)                              | Yes   | `ExampleETag`                       | `BenchmarkETagAdapterOverhead` (zero-cost passthrough) | —                      |
| CSP Nonce           | `nonce.go`                             | `NonceConfig` + `Validate()`, `NonceAttr`, CSP builders       | Yes   | `ExampleNonce`                      | `BenchmarkNonce*`                                      | `FuzzNonce`            |

Plus `Chain()` and `Compose()` (bundle middlewares into one reusable `Middleware`) in `recorder.go`/`compose.go`, `MiddlewareFunc.Then()` for value-level chaining, and `MiddlewareStack.Middleware()` to nest a stack as a single middleware.

### Error Classification System

- Typed error-code model (`Code`/`Domain`, v0.12.0): every code is `domain.failure`; constructor + Wrap pairs cover all six families; `DomainOf`/`InDomain` route by failing component. Runtime codes: `http.write_failed`, `http.hijack_unsupported`, `http.hijack_failed`, `http.compress_write_failed`, `http.etag_*`, `compression.pool_type_unexpected`, `compression.qvalue_*`, `compression.incompressible_prefix_invalid`, `decompression.*`, `server.shutdown_failed`, `ratelimit.*`, `maxbodysize.*`, `requestid.*`, `security.*`, `metrics.*`, `nonce.*`, `cors.*`, `csrf.*`, `stack.*`. Plus 3 ETag codes from `go-etag` registered via `RegisterErrorClassifications()`.
- `RegisterErrorClassifications()` maps stdlib HTTP errors to behavioral families (Transient vs Infrastructure).
- CSRF middleware uses `go-error-family` directly: `ErrCSRFInvalid` (Rejection family) and `ErrCSRFConfig` (Infrastructure family), plus inline `NewInfrastructure` errors for config validation failures.
- Message templates with `what/why/fix/wayOut` for all classified errors.
- Test coverage in `errors_test.go`.

### Shared ResponseWriter Wrapper

- `wrapper.go` extracts common `WriteHeader` buffering, `Hijack`, and `Flush` delegation.
- Embedded by `compressWriter`, eliminating ~80 lines of duplication.

### Infrastructure Types

- `MiddlewareStack` collects named middleware with duplicate prevention and ordering validation (Recovery must be outermost when present). 14 well-known `Middleware*` constants (Recovery, Logging, RequestID, CORS, SecurityHeaders, Nonce, Compression, Decompression, Timeout, ClientIP, CSRF, ServerTiming, KeyedRateLimit, ETag).
- `MiddlewareETag` intentionally survives the v1.1.0 adapter removal: it is the stack name key for composing `etag.New` (from `github.com/larsartmann/go-etag/server`) directly via `MiddlewareStack.Add`.
- `DetectCapabilities()` inspects a ResponseWriter for Hijacker/Flusher support.
- `DefaultIncompressibleTypes()` returns the default content-type deny-list for Compression.

### Validate-at-Construction

- All middleware constructors call `cfg.Validate()` at startup via a shared `validateConfig(name, err)` helper in `recorder.go` (new in v0.11.0).
- Invalid configs are logged via `slog.Error` and fall back to default values — the validate-and-log pattern, not validate-and-abort.
- Previously only `CSRFMiddleware` and `Nonce` validated at construction; `Compression`, `CORS`, `SecurityHeaders`, `Decompression`, `MaxBodySize`, `RequestID`, `RateLimit`, and `KeyedRateLimiterMiddleware` now do too.

### CORS Security

- `DenyUnmatched` option on `CORSConfig` — when true, withholds `Access-Control-Allow-Origin` for origins not in `AllowedOrigins`, preventing allowlist bypass via wildcard fallback. Default is `true` since v0.7.0.
- `AllowPrivateNetwork` option on `CORSConfig` — when true, the preflight response carries `Access-Control-Allow-Private-Network: true` for Chrome's Private Network Access / Local Network Access check (pages from a less-private address space fetching LAN/localhost subresources). Preflight-only and opt-in (default `false`); never set on actual requests or with `OptionsPassthrough`.
- Wildcard origin matching (e.g., `*.example.com`) rejects lookalike domains (`*.example.com.evil.com`).
- `AllowCredentials: true` + `AllowAllOrigins: true` rejected at `Validate()` time (browsers reject this combination).

### Compression Performance

- `DefaultWriterFactoriesForLevel(level int)` returns a fresh default factory map (gzip + deflate + identity) at any compression level.
- `Compression()` uses `cfg.Level` to build default factories when `WriterFactories` is empty — `Level` is no longer ignored.
- Per-encoding `writerPool` (owned by the negotiator, one pool per encoding per `Compression` instance) reuses `gzip.Writer` and `flate.Writer` instances; a construction-time probe makes non-resettable custom factories skip the pool entirely (no wasted pooled allocation per request).
- Content-type deny-list skips incompressible formats (`image/`, `video/`, `audio/`, `application/gzip`, `application/zip`, `application/pdf`, etc.).
- Bounded buffering: only buffers up to `minSize`, then streams tail bytes directly.
- Buffer pre-allocated to `max(minSize, 512)` capacity to avoid intermediate reallocations.
- RFC 7231 `Accept-Encoding` negotiation with q-value parsing; server priority order is brotli > zstd > gzip > deflate > identity.
- Single error-classification choke point: compress write failures funnel through `compressWriteError` with `encoding` context.

### Rate Limiting

- **Keyed rate limiting** via `KeyedRateLimiter` (new in v0.8.0) — O(log n) min-heap eviction, `MaxKeys` cap, lazy TTL eviction, `Retry-After` headers, and a monitoring API (`ActiveKeys()`). This is the recommended API going forward.
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
- `SameSite=None` without `Secure` falls back to `Secure=true` at construction (browsers refuse to store the cookie); `AllowInsecureSameSiteNone` opts out for legacy-client deployments.
- Domain-level `TrustedOrigins` allowlist and trusted-proxy CIDR allowlists. `TrustedOrigins` entries must be well-formed `scheme://host` origins (`Validate` rejects others via `csrf.trusted_origin_invalid`); runtime parsing is all-or-nothing, mirroring `nosurf.StaticOrigins` — one unparseable entry falls back to same-origin-only validation (fail-closed).

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
- Generation-swapped immutable buffers: monotonic generation-stamped slot claims, atomic publication, GC reclamation of superseded generations (2026-09-10); the refill mutex serializes publishers only.

### Server Lifecycle

- `ServerConfig` with `Validate()` — read, header, write, and idle timeout validation.
- `DefaultServerConfig()` — production defaults (`:8080`, 10s/5s/30s/60s timeouts).
- `NewServer()` wraps `http.Server` with lifecycle helpers.
- `Start()` / `StartTLS()` bind via `net.Listen` first (bind errors are delivered immediately on the returned channel) and track the listener.
- `ListenerAddr()` returns the resolved address of the active listener (`"127.0.0.1:0"` → the real port), `(nil, false)` when not listening; cleared on successful `Shutdown()`.
- `StartTLS(certFile, keyFile)` serves HTTPS with the validated `TLSConfig` (TLS 1.2+ enforced); in-memory certs work via `GetCertificate` with empty paths.
- `Shutdown()` performs graceful shutdown respecting a context deadline.
- `Addr()` returns the configured listen address (never rewritten for ephemeral ports).

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
- `docs/integrations/` — extensibility examples (brotli/zstd, redis, prometheus, samber/do, huma, compose bundles).
- `docs/architecture-reference.md` — file-by-file export tables, error-classification table, lint profile.
- `docs/RELEASE.md` (runbook) + `scripts/prerelease-check.sh` — the documented, automated release gates.
- Status reports in `docs/status/`.
- Execution plans in `docs/planning/`.

### Tooling & Quality Gates

- `golangci-lint` with ~70 linters, 0 issues.
- `go test -race ./...` passes across the full suite with **97.2% statement coverage** (`httputil`, library packages), **98.6%** (`httpspec`) — measured 2026-09-14 with race detection enabled (the dev-tooling `scripts/coverage-threshold` package is excluded, matching the CI gate).
- 26 fuzz targets (24 root + 2 `server_timing`): CORS (header building + origin matching + wildcard patterns + exact-allowlist origin echo), Compression (round-trip gunzip-and-compare invariant + writer state machine + Accept-Encoding wire format), MaxBodySize (limit contract), RequestID, ClientIP, `ParseUintQuery`, `EvictionTTL`, `HealthResponse` encoding, Server-Timing (header value + middleware), ResponseRecorder, limited reader (bomb boundary), Decompression (malformed bodies + round-trip invariants), and CSRF (6 targets: TrustedProxies CIDR, TrustedOrigins, `isTrustedProxy`, token validation, `remoteHostAndIP`, origin headers). The compression round-trip invariant caught the exact-fill duplication bug within seconds of first execution (2026-08-30). All 26 targets run 5 minutes each in the nightly fuzz workflow.
- 49 top-level benchmark functions (58 result rows counting `b.Run` sub-benchmarks) and 30 example functions across `httputil` + `httpspec` (`server_timing` has none).
- `go vet` clean.
- `.editorconfig` enforces consistent indentation and formatting across editors.
- Nix flake for reproducible development environment.
- GitHub Actions CI for tests, lint, and `govulncheck`.
- Release workflow with `govulncheck`, CHANGELOG link validation, and pre-release self-review step (`docs/RELEASE.md`).
- GitHub Actions pinned to commit SHAs (supply-chain hardening).

### Behavioral Spec Suite (`httpspec` subpackage)

- `httpspec.Run(t, handler)` validates any `http.Handler` against 19 standard HTTP behavior specs.
- `httpspec.RunSerial(t, handler)` variant for handlers with shared mutable state.
- 8 pre-built extra specs available via `WithExtraSpecs`: `CORSSpecs()` (5 specs: allow-origin, allow-credentials, Vary: Origin, wildcard-no-credentials, origin-matches-request) and `RateLimitSpecs()` (3 specs: Retry-After on reject, X-RateLimit-* headers on reject, hint headers on allow). Total: 27 specs when all are included.
- Specs cover routing (index reachability, unknown paths, long URLs), method handling (HEAD, OPTIONS, TRACE, POST, CONNECT), response headers (Content-Type, Location on redirects, no duplicate headers, Accept header handling), and security (no leaked internals, no version fingerprints, no X-Powered-By, X-Content-Type-Options: nosniff).
- Extensible via `SkipSpec`, `WithExtraSpecs`, `WithIndexPath`.
- Helper builders: `ExpectStatus`, `ExpectNotStatus`, `ExpectHeader`, `ExpectHeaderAbsent`, `ExpectBodyContains`, `ExpectJSON` (validates body parses as JSON + JSON Content-Type), `ExpectHTML` (HTML Content-Type incl. XHTML), `ExpectVaryContains` (cache-correctness, `Vary: *` aware), `ExpectNotModifiedWithETag` (opt-in 304 contract via `WithExtraSpecs`).
- Pure stdlib, no third-party dependencies.

---

## PARTIALLY DONE

### Test Coverage — sub-100% functions (defensive code paths)

Measured 2026-09-14 with `go test -race -coverprofile`: **97.2%** (`httputil`, library packages; 97.5% combined across both modules), **98.6%** (`httpspec`). The remaining sub-100% functions are documented defensive code paths:

**Typed error model (`code.go`):**

- `code.go:45 Code.Conflict`, `code.go:65 Code.Orchestration`, `code.go:72 Code.WrapRejection` — 0%. Exported constructor methods kept for `Code` API completeness (every family has a constructor); not yet exercised by any test. Tracked in TODO_LIST (the v1.0 sweep added `WrapConflict`/`WrapOrchestration` tests, these three remain).

**New middleware (CSRF, Server-Timing, KeyedRateLimit, TLS):**

- `csrf.go:734 requestScheme` — 80.0%. The `r.TLS != nil` HTTPS branch needs a TLS request fixture.
- `csrf.go:755 forwardedProtoFromTrustedProxy` — 75.0%. The untrusted-remote, empty-header, and non-http(s)-proto early returns need XFP fixtures with a configured trusted proxy (current tests exercise the no-proxy and trusted-proxy happy paths).
- `csrf.go:835 CSRFTokenHXHeaders` — 71.4%. Token-less request branch and the `json.Marshal` error on `map[string]string` (practically unreachable).
- `csrf.go:876 CSRFTestToken` — 92.9%. Internal nosurf error branches.
- `csrf.go:928 ValidateCSRF` — 94.4%. The custom header/field-name translation branch (`needsTranslation`) is only exercised through `CSRFMiddleware`, not through `ValidateCSRF`.
- `ratelimit_keyed.go:200 buildKeyedRateLimiter` — 93.1%. Defensive config validation edge.
- `ratelimit_keyed.go:320 limiter` — 78.3%. RLock-hit-but-TTL-expired path (race condition).
- `ratelimit_keyed.go:384 evictOldestIfAtCapacity` — 88.9%. Stale-heap-mismatch continue branch.
- `server.go:193 Start` — 91.7%. Listener-close failure branch.
- `server.go:227 StartTLS` — 75.0%. Listen-failure error branch (requires port-conflict injection).
- `compress_pool.go:44 newWriterPool` — 88.9%. Factory-probe error branch.
- `compress_pool.go:73 acquire` — 90.9%. Pool-type-mismatch defensive branch.
- `compression.go:205 Compression` — 96.0%. Constructor warning path for an invalid custom level.
- `decompression.go:107 Decompression` — 84.8%. Encoding-filter reject path (unreachable `default:` switch case when allowed list contains only gzip/deflate — documented as the custom-Encodings contract).
- `id_generator.go:125 drawRandomBytes` — 80.0%. Short-read loop branch (kernel-level fault injection).

**`httpspec` subpackage (separate module, 98.6%):**

- `httpspec/httpspec.go:235 runSpecs` — 88.2%. Internal option error paths.
- `httpspec/httpspec.go:317 parsedMediaType` — 75.0%. Malformed Content-Type branches.
- `httpspec/httpspec.go:326 isJSONContentType` / `:335 isHTMLContentType` — 75.0% each. Structured-syntax-suffix branches (`+json`/`+xml`) exercised only via the suffix test matrix.

**Honest assessment:** The remaining sub-100% functions are documented as defensive code paths or error-injection-only branches. The deleted `crypto/rand` error guards (nonce, ID-generator refill) left the profile entirely in the v1.0.0 panic-free pass. Closing what remains would require either (a) kernel-level fault injection, (b) direct unit-only construction of internal types, or (c) test infrastructure that doesn't exist in this project. The three 0% `Code` constructors are API-completeness, not defensive paths — they are tracked in TODO_LIST for direct tests.

---

## PLANNED

### v1.1.0 stabilization (shipped 2026-09-11)

- ~~**v1.1.0** — remove the deprecated `TokenBucketLimiter`/`RateLimit()` and the `httputil.ETag()` adapter per the migration guide~~ done 2026-09-11 (removed on master, recorded in CHANGELOG [1.1.0]; the v1.0.0 drift shipped as v1.0.1 and both tags are pushed).

---

## WORTH CONSIDERING

- **Brotli / zstd / lz4 support** — now possible via the `WriterFactory` plugin interface without adding core dependencies. Documentation examples at `docs/integrations/brotli-zstd.md`; built-in encoders are deliberately not added to preserve the dependency policy.
- **Rate limiter `context.Context` cancellation** — a cancel-aware `Wait(ctx, key)` remains the post-v1.0 additive path; v1.0 shipped the admission-only contract (evaluated in [docs/planning/archived/2026-08-29_21-30_rate-limiter-ctx-cancellation-design-note.md](docs/planning/archived/2026-08-29_21-30_rate-limiter-ctx-cancellation-design-note.md), decided in [ROADMAP.md](ROADMAP.md)).
