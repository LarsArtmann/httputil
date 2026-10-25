# Features

Honest feature inventory for `httputil`.

_Updated: 2026-10-23 (OTEL tracing support: `RecoveryConfig`/`RecoveryWithConfig` with the `OnPanic` hook, `httpspec.TracingSpecs()` + `httpspec.RunTracing()`, and `docs/integrations/otel.md` added; counts re-derived from the tree: 53 benchmarks / 63 result rows, 60 examples (45 root / 10 `httpspec` / 5 `server_timing`), 30 fuzz targets, 34 `httpspec` specs when all extras are included. Prior: 2026-10-15 (T18/T19 measurement batch: BenchmarkLanguageStaticRequest + ExampleKeyedRateLimiter_evictionTTL added; counts re-derived from the tree: 53 benchmarks / 63 result rows, 59 examples, 30 fuzz targets. Prior: 2026-10-14 docs-health AUDIT — the middleware-suite table's separator row normalized to the 8-column shape; the duplicated httpspec coverage rows merged; the Language row's Benchmarks cell corrected (BenchmarkLanguage/HeaderOnly/ParseAcceptLanguageCandidates exist in language_bench_test.go) and the TrustedClientIP row's Tests cell corrected (ratelimit_keyed_trusted_test.go, 100% covered).)

---

## FULLY FUNCTIONAL

### Core Middleware Suite (18 rows: 16 root constructors, Server-Timing from its own module, and the ResponseRecorder utility)

| Middleware | File | Config Type | Tests | Examples | Benchmarks | Fuzz | |
| ------------- | -------- | ------------- | ------- | ------------ | -------------- | -------- | ----- |
| CORS | `cors.go` | `CORSConfig` + `Validate()` | Yes | `ExampleCORS` | `BenchmarkCORS` | `FuzzCORS*` (4) | |
| ClientIP | `clientip.go`, `context.go` | — | Yes | `ExampleClientIP` | `BenchmarkClientIP` | `FuzzClientIP` | |
| RequestID | `requestid.go`, `id_generator.go` | `RequestIDConfig` + `Validate()`, time-ordered ID generator | Yes | `ExampleRequestID` | `BenchmarkRequestID` | `FuzzRequestID` | |
| SecurityHeaders | `security.go` | `SecurityHeadersConfig` + `Validate()` | Yes | `ExampleSecurityHeaders` | `BenchmarkSecurityHeaders` | — | |
| Recovery | `recovery.go` | `*slog.Logger` (via `Recovery`) or `RecoveryConfig` + `RecoveryWithConfig()` (`Logger`, `OnPanic` observability hook) | Yes | `ExampleRecovery`, `ExampleRecoveryWithConfig` | `BenchmarkRecovery` | — | |
| Timeout | `timeout.go` | `time.Duration` | Yes | `ExampleTimeout` | `BenchmarkTimeout` | — | |
| Logging | `logging.go` | `*slog.Logger`; `LoggingFromContext` (context-resolved logger + fallback) | Yes | `ExampleLogging` | `BenchmarkLogging` | — | |
| ResponseRecorder | `recorder.go` | — | Yes | `ExampleNewResponseRecorder` | `BenchmarkResponseRecorder` | — | |
| Compression | `compression.go`, `compress_writer.go` | `CompressionConfig` + `Validate()`, `WriterFactory` plugin; `Range` requests pass through uncompressed | Yes | `ExampleCompression` | `BenchmarkCompression*` | `FuzzCompression*` (3) | |
| MaxBodySize | `maxbodysize.go` | `MaxBodySizeConfig` + `Validate()`, `MaxBodySizeMiddleware()` | Yes | `ExampleMaxBodySize` | `BenchmarkMaxBodySize` | `FuzzMaxBodySize` | |
| Metrics | `metrics.go` | `MetricsConfig` + `Validate()`, `MetricsRecorder` interface | Yes | `ExampleMetrics` | `BenchmarkMetricsMiddleware*` | — | |
| Server-Timing | `server_timing/server_timing.go` | — | Yes | `ExampleServerTimingMiddleware` | `BenchmarkServerTiming*` | `FuzzServerTiming*` | |
| CSRF | `csrf.go` | `CSRFConfig` + `Validate()` (incl. `BootLogUntrustedForwardedProto` boot-warning opt-in) | Yes | `ExampleCSRFMiddleware`, `ExampleCSRFTokenFormField`, `ExampleCSRFTokenHXHeaders` | `BenchmarkCSRFMiddleware*` | `FuzzCSRF*` (6) | |
| KeyedRateLimit | `ratelimit_keyed.go` | `KeyedRateLimiterConfig` + `Validate()` | Yes | `ExampleKeyedRateLimiterMiddleware` | `BenchmarkKeyedRateLimiter*` | — | |
| TrustedClientIP | `ratelimit_keyed_trusted.go` | `TrustedProxySet` via `ParseTrustedProxies()`, `KeyExtractorFromTrustedClientIP()` (additive, v1.5-cycle) | Yes | — | — | `FuzzTrustedProxySetClientIP` | |
| Decompression | `decompression.go` | `DecompressionConfig` + `Validate()`, bomb protection | Yes | `ExampleDecompression` | `BenchmarkDecompression*` | `FuzzDecompression` | |
| CSP Nonce | `nonce.go` | `NonceConfig` + `Validate()`, `NonceAttr`, CSP builders | Yes | `ExampleNonce` | `BenchmarkNonce*` | `FuzzNonce` | |
| Language | `language.go` | `LanguageConfig` + `Validate()`, pluggable extractors + `TagMatcher` | Yes | `ExampleLanguage` | `BenchmarkLanguage*` | `FuzzParseAcceptLanguage` | |

ETag conditional requests are not a row above since v1.1.0: the adapter was removed, and the middleware composes directly from [`go-etag`](https://github.com/larsartmann/go-etag) (`etag.New`), named in stacks via the surviving `MiddlewareETag` constant.

Plus `Chain()` and `Compose()` (bundle middlewares into one reusable `Middleware`) in `recorder.go`/`compose.go`, `MiddlewareFunc.Then()` for value-level chaining, and `MiddlewareStack.Middleware()` to nest a stack as a single middleware.

### Error Classification System

- Typed error-code model (`Code`/`Domain`, v0.12.0): every code is `domain.failure`; constructor + Wrap pairs cover all six families; `DomainOf`/`InDomain` route by failing component. Runtime codes: `http.write_failed`, `http.hijack_unsupported`, `http.hijack_failed`, `http.compress_write_failed`, `http.etag_*`, `compression.pool_type_unexpected`, `compression.qvalue_*`, `compression.incompressible_prefix_invalid`, `compression.absent_encoding_invalid`, `decompression.*`, `server.shutdown_failed`, `ratelimit.*`, `maxbodysize.*`, `requestid.*`, `security.*`, `metrics.*`, `nonce.*`, `cors.*`, `csrf.*`, `language.*`, `stack.*`. Plus 3 ETag codes from `go-etag` registered via `RegisterErrorClassifications()`.
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
- Previously only `CSRFMiddleware` and `Nonce` validated at construction; `Compression`, `CORS`, `SecurityHeaders`, `Decompression`, `MaxBodySize`, `RequestID`, and `KeyedRateLimiterMiddleware` now do too.

### CORS Security

- `DenyUnmatched` option on `CORSConfig` — when true, withholds `Access-Control-Allow-Origin` for origins not in `AllowedOrigins`, preventing allowlist bypass via wildcard fallback. Default is `true` since v0.7.0.
- `AllowPrivateNetwork` option on `CORSConfig` — when true, the preflight response carries `Access-Control-Allow-Private-Network: true` for Chrome's Private Network Access / Local Network Access check (pages from a less-private address space fetching LAN/localhost subresources). Preflight-only and opt-in (default `false`); never set on actual requests or with `OptionsPassthrough`.
- Wildcard origin matching (e.g., `*.example.com`) rejects lookalike domains (`*.example.com.evil.com`).
- `AllowCredentials: true` + `AllowAllOrigins: true` rejected at `Validate()` time (browsers reject this combination).

### Compression Performance

- `DefaultWriterFactoriesForLevel(level int)` returns a fresh default factory map (gzip + deflate + identity) at any compression level.
- `Compression()` uses `cfg.Level` to build default factories when `WriterFactories` is empty — `Level` is no longer ignored.
- Per-encoding `writerPool` (owned by the negotiator, one pool per encoding per `Compression` instance) reuses `gzip.Writer` and `flate.Writer` instances; a construction-time probe makes non-resettable custom factories skip the pool entirely (no wasted pooled allocation per request). The pool owns exactly one factory, pooled elements carry owner provenance (`release` refuses foreign writers), and `(nil, nil)` factory returns are rejected — panics at construction/refill (factory-contract class), a classified `compression.pool_type_unexpected` error on the direct path.
- Content-type deny-list skips incompressible formats (`image/`, `video/`, `audio/`, `application/gzip`, `application/zip`, `application/pdf`, etc.).
- Bounded buffering: only buffers up to `minSize`, then streams tail bytes directly.
- Buffer pre-allocated to `max(minSize, 512)` capacity to avoid intermediate reallocations.
- RFC 7231 `Accept-Encoding` negotiation with q-value parsing; server priority order is brotli > zstd > gzip > deflate > identity.
- Requests without an `Accept-Encoding` header (or with an empty one) are served uncompressed by default (`CompressionConfig.AbsentEncoding` zero value `AbsentEncodingIdentity`); `AbsentEncodingFirstConfigured` restores the v1.1.x behavior of compressing with the highest-priority configured encoding (issue #4).
- Single error-classification choke point: compress write failures funnel through `compressWriteError` with `encoding` context.

### Rate Limiting

- **Keyed rate limiting** via `KeyedRateLimiter` (new in v0.8.0) — O(log n) min-heap eviction, `MaxKeys` cap, lazy TTL eviction, `Retry-After` headers, and a monitoring API (`ActiveKeys()`). This is the recommended API going forward.
- `EvictionTTL` field on `KeyedRateLimiterConfig` enables opt-in lazy eviction of idle buckets. Zero (default) preserves unbounded-growth behavior.
- Pluggable `KeyExtractor` interface (`KeyExtractorFromRemoteAddr`, `KeyExtractorFromClientIP`, and — new in the v1.5 cycle — `KeyExtractorFromTrustedClientIP(cidrs)`, which honors forwarded headers only from trusted-proxy CIDR peers; `ParseTrustedProxies` returns the reusable `TrustedProxySet` with `ClientIP`/`Contains` for guard and logging consumers).
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

### Observability Hooks (OpenTelemetry bridge)

- `RecoveryWithConfig(RecoveryConfig)` — panic recovery with a `RecoveryConfig{Logger, OnPanic}`; the `OnPanic func(*http.Request, any)` hook runs after the panic is logged and before the 500 is written, so a tracing SDK can record the panic on the active span (`otel.RecordError`) without adding a telemetry dependency here.
- `Recovery(logger)` is unchanged and delegates to `RecoveryWithConfig` with a nil hook; `RecoveryConfig` has zero-value semantics for every field, so there is no invalid configuration to validate.
- A panicking hook is logged (`"panic recovery hook panicked"`) and swallowed, so a buggy integration cannot replace the original panic handling; the 500 response is still written.
- Hook runs synchronously on the request goroutine; `http.ErrAbortHandler` panics are re-panicked unchanged and never reach the hook.
- Two-lane OTEL guidance in [`docs/integrations/otel.md`](docs/integrations/otel.md): `go-appkit/otel` owns SDK/provider setup, httputil owns zero-dependency hooks and ordering. An `httputil/otel` sub-module is a recorded non-goal.
- `RequestID` correlation ruling: `X-Correlation-ID` echo + fallback (opt-in), with `correlation_id` baggage as the cross-service complement; the W3C trace ID never seeds the request ID.

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
- `NewServerListener(ln, cfg, handler)` serves on a caller-bound listener (issue #41 composition seam): wrappers own the bind, `Start`/`StartTLS` never bind, `ListenerAddr()` reports the injected listener from construction, `Addr` is derived from `ln.Addr()`. Single-serve by ownership — `net/http` closes the listener when serving ends, and a restart after that returns the classified `server.listener_consumed` rejection. Works for any `net.Listener` (TCP, unix sockets, socket-activation fds).
- `Start()` / `StartTLS()` bind via `net.Listen` first (bind errors are delivered immediately on the returned channel) and track the listener. Error delivery implies full settlement: the listener is closed and `ListenerAddr()` reports `(nil, false)` before the error can be received, so restarting the same `Server` after any failure (or after `Shutdown()`) is safe; starting a listening server returns the classified `server.already_started` error and leaves it running. `Shutdown` on a never-started `Server` is a no-op returning nil (the embedded `http.Server` is not poisoned, so a later `Start` still serves; for `NewServerListener` servers the listener stays open and addressable).
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
- `docs/migrating-to-v1.2.md` — v1.2.0 behavioral-change migration guide.
- `docs/integrations/` — extensibility examples (brotli/zstd, redis, prometheus, samber/do, huma, compose bundles, `otel.md` for the OpenTelemetry tracing lanes).
- `docs/architecture-reference.md` — file-by-file export tables, error-classification table, lint profile.
- `docs/RELEASE.md` (runbook) + `scripts/prerelease-check.sh` — the documented, automated release gates.
- Status reports in `docs/status/`.
- Execution plans in `docs/planning/`.

### Tooling & Quality Gates

- `golangci-lint` with ~70 linters, 0 issues.
- `go test -race ./...` passes across the full suite with **98.7% statement coverage** (`httputil`, library packages), **98.4%** (`httpspec`), **100%** (`server_timing`) — measured 2026-10-23 with race detection enabled (the dev-tooling `scripts/coverage-threshold` package is excluded, matching the CI gate). The previous race measurement was 97.9% / 99.1% on 2026-09-27; the root gain comes from the `RecoveryWithConfig` tests, the `httpspec` change reflects the new `tracing_specs.go` (its checks are fully covered, only failure-forwarding lines are not).
- 30 fuzz targets (28 root + 2 `server_timing`): CORS (header building + origin matching + wildcard patterns + exact-allowlist origin echo + preflight Private-Network), Compression (round-trip gunzip-and-compare invariant + writer state machine + Accept-Encoding wire format), MaxBodySize (limit contract), RequestID, ClientIP, `ParseUintQuery`, `HealthResponse` encoding, ResponseRecorder, limited reader (bomb boundary), Decompression (malformed bodies + round-trip invariants), CSRF (8 targets: TrustedProxies CIDR, TrustedOrigins, `isTrustedProxy`, token validation, `remoteHostAndIP`, origin headers, the token HTML formatters, XFP attestation), trusted-proxy client-IP keying, Accept-Language wire format (`FuzzParseAcceptLanguage` — three real parser bugs in its first 45s), and Server-Timing (header value + middleware). The compression round-trip invariant caught the exact-fill duplication bug within seconds of first execution (2026-08-30). The nightly fuzz workflow runs all 30 at 300 seconds each — the 2026-10-15 rotation added the last four (`FuzzCORSPreflightPrivateNetwork`, `FuzzCSRFTokenHTMLFormatters`, `FuzzCSRFMiddleware_XFPAttestation`, `FuzzTrustedProxySetClientIP`) after the mandatory pre-wiring smoke run, which promptly crashed `FuzzCSRFTokenHTMLFormatters` in 0.08 seconds (Go 1.27's encoding/json rejects invalid UTF-8 where v1 coerced; oracle fixed, production code untouched) — the standing worked example for why smoke-before-rotate is the discipline.
- 53 top-level benchmark functions (63 result rows counting `b.Run` sub-benchmarks) and 60 example functions across the three modules — 45 root (incl. the health/queryparam, language, and `ExampleRecoveryWithConfig` examples), 10 `httpspec`, 5 `server_timing` (added 2026-09-23, all self-contained per the Example Conventions; every one executes with a verified `// Output:` block).
- `go vet` clean.
- AST sentinel guard (`sentinels_type_test.go`) — parses the package and fails any `err*`/`Err*` package variable whose declared type is the `error` interface, the exact shape that silently breaks `.WithContext`/`.WithCause` and left master uncompilable for 8 days (2026-10-15 → 2026-10-23).
- `.editorconfig` enforces consistent indentation and formatting across editors.
- Nix flake for reproducible development environment.
- GitHub Actions CI for tests, lint, and `govulncheck`.
- Release workflow with `govulncheck`, CHANGELOG link validation, and pre-release self-review step (`docs/RELEASE.md`).
- GitHub Actions pinned to commit SHAs (supply-chain hardening).

### Behavioral Spec Suite (`httpspec` subpackage)

- `httpspec.Run(t, handler)` validates any `http.Handler` against 19 standard HTTP behavior specs.
- `httpspec.RunSerial(t, handler)` variant for handlers with shared mutable state.
- 15 pre-built extra specs available via `WithExtraSpecs`: `CORSSpecs()` (5 specs: allow-origin, allow-credentials, Vary: Origin, wildcard-no-credentials, origin-matches-request), `RateLimitSpecs()` (3 specs: Retry-After on reject, X-RateLimit-* headers on reject, hint headers on allow), `PrivateNetworkSpecs()` (1 spec: denied-origin LNA preflight still answers 204 without the Private-Network header — added v1.2.0), `LanguageSpecs()` (2 specs: Content-Language present, Vary lists Accept-Language — new in v1.6.0), and `TracingSpecs()` (4 specs: valid traceparent and valid tracestate never produce a server error, malformed traceparent and malformed tracestate behave like absent headers — new in v1.6.0). Total: 34 specs when all are included.
- `httpspec.RunTracing(t, middleware)` — middleware-level tracing contract checks that handler-level `Spec`s cannot observe: `traceparent`, `tracestate`, and the route pattern reach the wrapped handler; an absent `traceparent` stays absent; method, path, and body arrive unmodified; status and body pass through unchanged; and `Flusher`, `Pusher`, and `Hijacker` survive the wrapper. Point it at `otelhttp.NewMiddleware(...)` (or any tracing middleware) to prove interop (new in v1.6.0).
- Specs cover routing (index reachability, unknown paths, long URLs), method handling (HEAD, OPTIONS, TRACE, POST, CONNECT), response headers (Content-Type, Location on redirects, no duplicate headers, Accept header handling), and security (no leaked internals, no version fingerprints, no X-Powered-By, X-Content-Type-Options: nosniff).
- Extensible via `SkipSpec`, `WithExtraSpecs`, `WithIndexPath`.
- Helper builders: `ExpectStatus`, `ExpectNotStatus`, `ExpectHeader`, `ExpectHeaderAbsent`, `ExpectBodyContains`, `ExpectJSON` (validates body parses as JSON + JSON Content-Type), `ExpectHTML` (HTML Content-Type incl. XHTML), `ExpectVaryContains` (cache-correctness, `Vary: *` aware), `ExpectNotModifiedWithETag` (opt-in 304 contract via `WithExtraSpecs`).
- Pure stdlib, no third-party dependencies.

---

## PARTIALLY DONE

### Test Coverage — sub-100% functions (defensive code paths)

Re-measured 2026-10-25 with `go test -race -coverprofile` per package: **98.7%** (`httputil`, library packages), **98.4%** (`httpspec`), **100%** (`server_timing`). The remaining sub-100% functions are documented defensive code paths, listed below with their current figures.

**Typed error model (`code.go`):** fully covered as of the 2026-09-23 measurement (all constructor and Wrap methods at 100% — the error-routing examples closed the last three).

**Middleware, server, language, and ID-generator internals (2026-10-23 race measurement):**

- `csrf.go:991 CSRFTestToken` — 96.4%. Internal nosurf error branches. (`CSRFTokenHXHeaders`, `requestScheme`, `forwardedProtoFromTrustedProxy`, and `ValidateCSRF` are now 100%.)
- `ratelimit_keyed.go:220 buildKeyedRateLimiter` — 95.5%. Defensive config validation edge.
- `ratelimit_keyed.go:340 limiter` — 91.5%. RLock-hit-but-TTL-expired path (race condition).
- `ratelimit_keyed.go:404 evictOldestIfAtCapacity` — 90.9%. Stale-heap-mismatch continue branch.
- `server.go:272 Start` — 98.5%. The non-shutdown Serve-error send — forcing an accept failure needs the unexported listener handle, not injectable from tests.
- `compression.go:269 Compression` — 97.9%. Constructor warning path for an invalid custom level.
- `decompression.go:107 Decompression` — 84.0%. Encoding-filter reject path (unreachable `default:` switch case when allowed list contains only gzip/deflate — documented as the custom-Encodings contract).
- `id_generator.go:129 drawRandomBytes` — 84.6%. Short-read loop branch (kernel-level fault injection).
- `language.go:168 supports` — 75.0%. Charset-reject and duplicate-entry branches of the built-in matcher.
- `language.go:181 validLanguageTag` — 83.3%. Charset-guard reject branches.
- `language.go:216 buildLanguageMatcher` — 95.5%. Invalid-tag skip branch.
- `language.go:383 LanguageExtractorFromQuery` — 80.0%. Missing query-parameter branch.
- `language.go:397 LanguageExtractorFromCookie` — 75.0%. Missing-cookie and malformed-cookie branches.

**`httpspec` subpackage (separate module, 98.4% race):**

- `httpspec/httpspec.go:235 runSpecs` — 91.7%. Internal option error paths.
- `httpspec/httpspec.go:317 parsedMediaType` — 75.0%. Malformed media-type defensive branches.
- `httpspec/httpspec.go:326 isJSONContentType` / `:335 isHTMLContentType` — 75.0% each. Malformed content-type and structured-syntax-suffix (`+json`/`+xml`) branches; the suffix matrix is exercised only via the dedicated test.
- `httpspec/tracing_specs.go:206 RunTracing` — 82.9%. The five `t.Error` forwarding lines and the nil-middleware `t.Fatal` line execute only when a check fails; the checks themselves (`checkRequestPassthrough`, `checkAbsentTraceparent`, `checkRequestInvariance`, `checkResponsePassthrough`, `checkWriterInterfaces`, `tracingMiddlewareProblem`) are 100% covered and each failure arm is mutation-probed directly.
- `httpspec/tracing_specs.go:419 readProbeBody` — 83.3%. Defensive branches: a nil `r.Body` (every probe request carries `http.NoBody`) and a body read error (requires fault injection).
- `httpspec/tracing_specs.go:441 Hijack` / `:447 Push` — 0.0% each. Deliberate: the probing writer exists so the checks can type-assert `http.Hijacker`/`http.Pusher`; the methods are never called (documented on each).

**Honest assessment:** The remaining sub-100% functions are documented as defensive code paths, error-injection-only branches, or failure-reporting lines. The deleted `crypto/rand` error guards (nonce, ID-generator refill) left the profile entirely in the v1.0.0 panic-free pass. Closing what remains would require either (a) kernel-level fault injection, (b) direct unit-only construction of internal types, or (c) test infrastructure that doesn't exist in this project.

---

## Distribution & Licensing

- **Both modules are MIT** — root `LICENSE` since v1.5.0, `server_timing/LICENSE` since v1.0.2 (owner decisions 2026-10-08/09; holder "Lars Artmann"). pkg.go.dev renders README and full documentation for both module pages (verified 2026-10-09; visibility is per-tag — it began with those tags).
- **`server_timing/README.md`** — the sub-module's own end-user page (install, quick start, the measure-before-write gotcha), rendered on pkg.go.dev's module page and the GitHub directory view (added in the v1.5.0 cycle, shipped at `server_timing/v1.0.2`).

## WORTH CONSIDERING

- **Locale-formatting helpers** — dates/numbers/currency are out of the middleware's scope by design (`language.go` doc comment); both integration guides now route readers to `golang.org/x/text/message` per-request `message.Printer` fed from `LanguageFromRequest`. A convenience wrapper (middleware-injected printer in context) would be additive — build only on a real consumer demand signal.
- **Brotli / zstd / lz4 support** — now possible via the `WriterFactory` plugin interface without adding core dependencies. Documentation examples at `docs/integrations/brotli-zstd.md`; built-in encoders are deliberately not added to preserve the dependency policy.
- **Rate limiter `context.Context` cancellation** — a cancel-aware `Wait(ctx, key)` remains the post-v1.0 additive path; v1.0 shipped the admission-only contract (evaluated in [docs/planning/archived/2026-08-29_21-30_rate-limiter-ctx-cancellation-design-note.md](docs/planning/archived/2026-08-29_21-30_rate-limiter-ctx-cancellation-design-note.md), decided in [ROADMAP.md](ROADMAP.md)).
