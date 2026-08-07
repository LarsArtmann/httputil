# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## [Unreleased]

### Added

- **ETag adapter over `go-etag`** (`etag.go`, `etag_test.go`): thin adapter wrapping `github.com/larsartmann/go-etag` v0.1.0 so consumers can use `httputil.ETag(cfg etag.ETagConfig) Middleware` without a separate import for middleware composition. `MiddlewareETag = "etag"` constant added to `stack.go`. ETag error templates (`etag.ErrCodeETagWriteFailed`, `etag.ErrCodeInvalidConfig`, `etag.ErrCodeHashWriteFailed`) registered via `RegisterErrorClassifications()`. 7 adapter integration tests + `ExampleETag` runnable example. RFC 7232 compliance tests live in the go-etag module's own test suite.
- **`ServerConfig.TLSConfig` support** (`server.go`): `TLSConfig *tls.Config` field wired through `NewServer()`. `Validate()` enforces `MinVersion >= TLS 1.2` per RFC 8996; zero `MinVersion` is allowed (Go defaults to TLS 1.2 since Go 1.18). 7 new tests covering TLS 1.0/1.1 rejection, TLS 1.2/1.3 acceptance, zero-MinVersion acceptance, and wiring.
- **`KeyedRateLimiterConfig.TTL` validation** (`ratelimit_keyed.go`): rejects negative TTL values (was silently coerced to default).
- **`RateLimitConfig.Status` validation** (`ratelimit.go`): rejects HTTP status codes outside 100-599 (except 0 = default 429).
- **Decompression benchmarks** (`decompression_bench_test.go`): gzip, deflate, and passthrough throughput with `b.ReportAllocs()` and `b.SetBytes()`.
- **`FuzzDecompression`** (`decompression_fuzz_test.go`): 11 seed corpus entries covering valid gzip/deflate, truncated headers, garbage bytes, empty body, identity, and unsupported encodings. Verifies status is always 200 or 400.
- **Decompression bomb-protection tests** (`decompression_test.go`): `limitedReadCloser.Close()` delegation (was 0%), `limitedReadCloser.Read()` bomb-limit-exceeded path (was 58.3%), and full integration test verifying `errDecompressionSizeExceeded` on over-limit decompressed body.
- **`ExampleDecompression`** (`example_test.go`): testable example with `// Output:` directive.
- **govulncheck in devShell** (`flake.nix`): `nix run .#vulncheck` app prevents the release-gate skip that occurred in v0.9.1.
- **Decompression documentation** (`README.md`, `docs/v1-stability.md`, `docs/DOMAIN_LANGUAGE.md`): README feature section, API table entries, config reference, and middleware ordering guidance. `v1-stability.md` classifies `DecompressionConfig` and `DefaultDecompressionConfig`. `DOMAIN_LANGUAGE.md` gains Decompression bounded context, entity, value objects, commands, events, and rules.

### Changed

- **Server-Timing extracted into `server_timing` sub-module** (`server_timing/`): W3C Server-Timing instrumentation moved to `github.com/larsartmann/httputil/server_timing` (package `servertiming`). Stdlib-only, zero external deps. Root module references it via `replace` directive + `go.work`.
- **Hijack error context** (`wrapper.go`): both `Hijack` error branches now attach `writer_type` via `WithContextf` for better diagnostics.
- **Coverage now 97.0%** (`httputil`) / **99.3%** (`httpspec`), measured 2026-08-07 with race detection enabled (was 96.9% at v0.9.1).

### Fixed

- **depguard `$module` workaround replaced with explicit module paths** (`.golangci.yml`): the global `_test.go` exclusion for depguard (which allowed any third-party import in test files) is replaced with explicit `github.com/larsartmann/httputil` and `/**` entries. Restores the dependency boundary for all files while permitting same-module cross-package imports.

### Removed

- **ETag middleware extracted to `go-etag` module**: in-package `etag.go`, `etag_test.go`, `etag_compress_fuzz_test.go`, and `httpspec/etag_integration_test.go` removed. ETag generation + RFC 7232 conditional-request logic now lives in `github.com/larsartmann/go-etag`. The `httputil.ETag()` adapter wraps it so consumers compose it like any other httputil middleware.

## [0.9.1] - 2026-08-06

### Fixed

- **ETag `If-None-Match` now uses RFC 7232 weak comparison** (`etag.go`): the comparison previously used literal string equality (the strong comparison function), so a client sending `W/"abc"` would miss a server-generated strong ETag `"abc"` and vice versa. Per [RFC 7232 §2.3.2](https://www.rfc-editor.org/rfc/rfc7232#section-2.3.2), `If-None-Match` must use the weak comparison function, which ignores the `W/` weakness indicator. Added `stripWeakPrefix` and rewrote `etagInList` to strip both sides before comparing opaque-tags.
- **ETag list parsing now respects commas inside quoted opaque-tags** (`etag.go`): the previous `strings.Index(list, ",")` splitter broke on commas inside quoted opaque-tags (permitted by the RFC 7232 §2.3 `etagc` grammar). Replaced with `parseETagList`, a quote-state-aware splitter that only splits on commas outside quoted strings. A client sending `"a,b"` is now parsed as a single tag rather than two malformed fragments.

### Added

- **5 ETag compliance tests** (`etag_test.go`): weak-client-vs-strong-server (304), strong-client-vs-weak-server (304), weak validator in a multi-element list (304), weak no-match negative case, and `parseETagList` comma-in-quotes correctness.
- **Weak-comparison fuzz seeds** (`etag_test.go`): `FuzzETag` corpus now includes `W/"..."` inputs documenting the RFC 7232 §2.3.2 fix.
- **ETag list-comparison benchmarks** (`etag_test.go`): `BenchmarkETagInList` (single/multi/weak variants) and `BenchmarkETag_IfNoneMatch` quantifying the conditional-request path. The `parseETagList` slice allocation is 1 alloc / 16-64 B per conditional request, adding <5% to the full middleware path — acceptable, not optimized.

## [0.9.0] - 2026-08-05

### Added

- **`MaxBodySizeConfig` + `Validate()`** (`maxbodysize.go`): new config struct with `MaxBytes int64` field, `DefaultMaxBodySizeConfig()` (1 MiB default), `Validate()` (rejects negative), and `MaxBodySizeMiddleware(cfg)`. The existing `MaxBodySize(maxBytes)` convenience function is preserved for backward compatibility.
- **`ServerConfig.ShutdownTimeout`** (`server.go`): new field with `Validate()` (rejects negative) and default of 30 seconds. `Server.Shutdown(ctx)` now auto-derives a timeout context when the provided context has no deadline, preventing indefinite hangs.
- **`KeyExtractor` footgun warning** (`ratelimit_keyed.go`): type doc comment now warns that returning `""` from a custom `KeyExtractor` silently disables per-client rate limiting.
- **Health and Metrics benchmarks** (`health_metrics_bench_test.go`): 6 benchmarks for HealthHandler, LiveHandler, ReadyHandler, and MetricsMiddleware (default, with body, with custom path).
- **ETag conditional and compressWriter fuzz tests** (`etag_compress_fuzz_test.go`): `FuzzETagConditional` tests If-Match/If-None-Match handling; `FuzzCompressWriterState` tests compression with varied encodings, bodies, and content types.
- **Updated D2 architecture diagram** (`docs/architecture-understanding/2026-08-05_httputil-current.d2` + `.svg`): reflects the current 16-middleware architecture including CSRF, Server-Timing, KeyedRateLimit, and the three external dependencies.
- **Request body decompression middleware** (`decompression.go`): `Decompression(cfg)` decompresses gzip/deflate request bodies based on Content-Encoding. Includes `DecompressionConfig` with `Validate()`, configurable encoding filter, and decompression bomb protection (`MaxDecompressionSize`, default 16 MiB).

- **`SecurityHeadersConfig` enriched** (`security.go`): gained `ContentTypeOptions string`, `PermissionsPolicy string`, and `Custom map[string]string` fields. `ContentTypeOptions` takes precedence over the legacy `ContentTypeNosniff bool` when set. Added `SecurityHeaderSkip = "-"` sentinel const, `RecommendedHSTS`, and `RecommendedCSP` consts. The `SecurityHeaders()` middleware now supports the `SecurityHeaderSkip` sentinel on `FrameOptions`/`ReferrerPolicy`/`ContentTypeOptions` (omits the header), sets `Permissions-Policy`, and applies `Custom` headers. `Validate()` accepts `SecurityHeaderSkip` as a valid `FrameOptions` value. All changes are **additive and backward-compatible** — existing consumers using `ContentTypeNosniff: true` are unaffected. Enables cqrs-htmx's `security.go` to alias this type as the single source of truth.

- **`httpspec` CORS and rate-limit behavior specs** (`cors_ratelimit_specs.go`): 4 CORS specs (`SpecNameCORSAllowOrigin`, `SpecNameCORSAllowCredentials`, `SpecNameCORSVaryOrigin`, `SpecNameCORSWildcardNoCredentials`) and 3 rate-limit specs (`SpecNameRateLimitRetryAfter`, `SpecNameRateLimitHeaderOnReject`, `SpecNameRateLimitHintHeadersOnAllow`). All return `Pass()` for handlers that don't set CORS or rate-limit headers (opt-in).
- **`KeyedRateLimiterConfig.Validate()`** (`ratelimit_keyed.go`): was the only config type missing validation. Validates rate, window, and burst.
- **`SecurityHeadersConfig.Validate()`** hardened (`security.go`): replaced the prior no-op with real `FrameOptions` value validation per RFC 7034 §2.1 (rejects `ALLOW-FROM` and lowercase variants).
- **`ServerConfig.Validate()`** hardened (`server.go`): rejects empty `Addr` and `ReadHeaderTimeout > ReadTimeout`.
- **CSRF fuzz tests** (`csrf_fuzz_test.go`, 257 lines): 6 fuzz functions covering TrustedProxies CIDR parsing, TrustedOrigins parsing, `isTrustedProxy`, token validation, `remoteHostAndIP`, and origin-header validation.
- **`BenchmarkKeyedRateLimiter`** (`ratelimit_keyed_bench_test.go`): 6 variants (Allow, Reject, HighCardinality, EmptyKey, EvictionOverhead, ClientIPExtractor).
- **`BenchmarkCSRFMiddleware`** (`csrf_bench_test.go`): 6 variants (GET, POSTWithToken, POSTRejection, PostForm, ConfigValidate, TokenFromContext).
- **Full-stack integration test** (`stack_integration_test.go`): chains all 12 `Middleware*` constants + ClientIP + ServerTiming across 5 parallel subtests (GET headers, POST CSRF rejection, OPTIONS preflight, panic recovery, rate-limit headers).
- **Dynamic coverage badge** (`scripts/update-coverage-badge.sh` + CI step): computes coverage via `go tool cover -func`, picks color by threshold, rewrites the badge line in `README.md` in-place.
- **`TestMustRequestPanicsOnInvalidMethod`** (`httpspec`): closes `mustRequest` from 75% to 100%.
- **`docs/DOMAIN_LANGUAGE.md`** updated with CSRF Protection, Server-Timing, and KeyedRateLimiting bounded contexts (was stale since v0.7.x).

### Fixed

- **Data race in `httpspec` rate-limit spec test** (`cors_ratelimit_specs_test.go`): a shared `map[string]int` closure was accessed from 3 parallel subtests' goroutines. Each subtest now owns a private handler instance. Detected by `go test -race`; invisible to `go test -count=1`.
- **Data race in `stack_integration_test.go`**: a shared `atomic.Bool` across 5 parallel subtests. Each subtest now owns its own `called` flag and handler.
- **`AGENTS.md` race-detection documentation**: `go test -race` is now labeled as REQUIRED for tests with `t.Parallel()` or shared state, with an explicit warning that `go test -count=1` does NOT detect data races.

### Changed

- `server_timing_bench_test.go` migrated from `b.N` loops to `b.Loop()` (Go 1.24+ pattern), clearing 6 gopls warnings.
- `httpspec/benchmark_test.go` migrated from `b.N` to `b.Loop()` for consistency.
- Historical status reports (`docs/status/2026-07-*` through `2026-08-05`) annotated inline with per-item resolution markers.
- Living docs (`TODO_LIST.md`, `ROADMAP.md`, `FEATURES.md`, `CHANGELOG.md`, `AGENTS.md`) rebuilt for post-v0.8.0 accuracy: coverage figures corrected to 97.6% httputil / 99.3% httpspec, WORTH CONSIDERING split brains resolved, done items removed from TODO_LIST, CHANGELOG freeze policy documented.
- CI workflow hardened: added `go test -race -count=10` stress test step and coverage threshold gate (fail if < 95%).
- `README.md` Quality Gates section added with full verification command table; duplicate-bracket badge artifact fixed; coverage badge updated.
- `scripts/pre-commit.sh` added: runs `golangci-lint run` on staged Go files before commit.
- `httpspec` coverage improved from 96.0% to 99.3% by closing all 5 `cors_ratelimit_specs.go` gaps to 100%.

## [0.8.0] - 2026-07-31

### Added

- **CSRF protection** (`csrf.go`): double-submit cookie CSRF middleware backed by `justinas/nosurf` v1.2.0, with HTMX-aware helpers (`CSRFTokenHXHeaders`, `CSRFTokenHTMLMeta`, `CSRFTokenFormField`), trusted-proxy CIDR support, domain-level `TrustedOrigins`, per-handler `ValidateCSRF`, and `ConfigureNosurfHandler` for fine-grained control over the underlying nosurf handler.
- **W3C Server-Timing** (`server_timing.go`): `ServerTimingMiddleware`, `ServerTimingMiddlewareWhen`, `MeasureServerTiming`, `WrapServerTiming`, `RecordServerTiming`, `WithServerTiming`, and `ServerTimingFromContext` for response instrumentation with CRLF-injection-safe header values. Includes `delegatingWriter` for full Hijacker/Flusher/Pusher support.
- **Keyed rate limiting** (`ratelimit_keyed.go`): `KeyedRateLimiter` with O(log n) min-heap eviction, a `MaxKeys` cap, lazy `EvictionTTL` eviction, `Retry-After` headers, and a monitoring API (`ActiveKeys`). Replaces the deprecated `TokenBucketLimiter` (slated for removal at v1.0).
- `MiddlewareStack` name constants for the new middleware: `MiddlewareCSRF`, `MiddlewareServerTiming`, `MiddlewareKeyedRateLimit` (`stack.go`).
- Example functions: `ExampleCSRFMiddleware`, `ExampleServerTimingMiddleware`, `ExampleKeyedRateLimiterMiddleware` (required by `testableexamples` linter).
- `docs/migrating-to-keyed-rate-limiter.md` — deprecation migration guide with symbol mapping, before/after examples, behavioral differences table, and monitoring guidance.
- CI workflow: `scripts/check-changelog-links.sh` validates every `[version]` heading has a matching link definition and vice versa.
- CHANGELOG link reference at file bottom: `[0.8.0]`.

### Changed

- `docs/v1-stability.md` — all new types classified as Frozen/Additive. New sections: CSRF Protection (17 rows), Server-Timing (10 rows), expanded Rate Limiting (12 rows). Middleware constants count updated from 9 to 12.
- `docs/RELEASE.md` — added pre-release self-review step.
- `coverage` improved from 91.0% to 97.8% (`httputil`) / 98.3% (`httpspec`). New middleware (CSRF, Server-Timing, KeyedRateLimit) and pre-existing functions (`Server.Shutdown`, `id_generator.go`) closed to 100% or documented as defensive code paths. `httpspec.mustRequest` was at 75% at release (defensive path — `httptest.NewRequest` panics rather than returning the error branch); closed to 100% post-release (see [Unreleased]).
- `writeClassified` doc comment corrected from "single error-handling choke point" to "Write-path error-handling choke point" — documents that buffer-drain writes in `Close` and `flushPlainAndStream` call `compressWriteError` directly.
- `AGENTS.md` — error classification table expanded with CSRF error family (Rejection + Infrastructure).
- `CONTRIBUTING.md` — allowed dependencies updated to include `github.com/justinas/nosurf`.
- `README.md` — feature sections added for CSRF, Server-Timing, Rate Limiting; `CSRFConfig` and `KeyedRateLimiterConfig` field tables added; middleware ordering section updated.
- `go-error-family` upgraded from v0.9.0 to v0.10.0.
- `github.com/justinas/nosurf` v1.2.0 added as a dependency (CSRF protection).
- `nix run .#coverage` available in the devShell.
- GitHub Actions pinned to commit SHAs (supply-chain hardening): `actions/checkout`, `actions/setup-go`, `actions/upload-artifact`, `golangci/golangci-lint-action`, `softprops/action-gh-release`.

### Fixed

- `Server.Shutdown` 75% → 100% via `TestServerShutdownReturnsErrorOnContextExpiry` (uses manual `net.Listen` + blocking handler + expired context). Added `ReadHeaderTimeout` to suppress gosec G112.

### Deprecated

- `TokenBucketLimiter`, the `RateLimiter` interface, `RateLimitConfig`, `DefaultRateLimitConfig`, and `RateLimit()` middleware — superseded by `KeyedRateLimiter` and `KeyedRateLimiterMiddleware`. Will be removed at v1.0. Migration guide: `docs/migrating-to-keyed-rate-limiter.md`.

## [0.7.1] - 2026-07-29

### Fixed

- `FuzzParseUintQuery` panicked on query values containing characters invalid in URLs (e.g. spaces). Now uses `url.QueryEscape` so any string input is exercised safely.
- `FuzzHealthResponse_Encoding` (formerly `FuzzHealthHandler`) failed on invalid UTF-8 status values because the strict round-trip assertion did not account for JSON encoder normalization. Rewritten to verify no-panic and valid parseable JSON without asserting exact round-trip.

### Changed

- `FuzzHealthHandler` renamed to `FuzzHealthResponse_Encoding` and rewritten to fuzz `HealthStatus` JSON encoding instead of request paths (which the health handler ignores).
- Stale CORS test name `TestCORS_AllowlistFallsBackToWildcardForUnmatchedOriginByDefault` renamed to `TestCORS_BareLiteralFallsBackToWildcardForUnmatchedOrigin` (the "ByDefault" was misleading after the `DenyUnmatched` default flip in v0.7.0).

### Added

- Compression writer error-branch tests: all `compress_writer.go` and `compress_pool.go` functions now at 100% coverage (overall 95.2% → 97.2%).
- `TestCompression_FlushWhileCompressing`, `TestCompression_FlushNonFlushableCustomWriter`, `TestCompressWriter_PassthroughWriterRoundTrip`, and 9 more unit tests covering Flush, Close, streaming, pool, and factory error paths.
- v1-stability.md now lists all 8 `Default*` config constructors and 9 `Middleware*` name constants (previously missing).
- Integration docs (brotli-zstd, redis-ratelimiter, prometheus-metrics) fixed: undefined `mux` variables resolved, external-dependency notes added.

## [0.7.0] - 2026-07-29

### Changed

- **Breaking:** `RequestIDConfig.HeaderName` renamed to `ResponseHeader` — the field sets the outgoing response header, not a vague "header name."
- **Breaking:** `RequestIDConfig.ForwardHeader` renamed to `IncomingHeader` — the field reads an incoming request header, not forwards one.
- **Breaking:** `DefaultCORSConfig()` now sets `DenyUnmatched: true` — unmatched origins are denied by default instead of falling back to wildcard `*`. Set `DenyUnmatched: false` to preserve old behavior.
- Corresponding sentinel errors renamed: `errEmptyHeaderName` → `errEmptyResponseHeader`, `errEmptyForwardHdr` → `errEmptyIncomingHeader` (unexported, but test-visible).
- Test coverage improved from 93.9% to 95.2% total (94.4% `httputil`, 98.3% `httpspec`).
- Four historical status reports annotated with jsonv2 resolution notes (v0.6.1 fixed the build permanently).

### Added

- `docs/RELEASE.md` — release runbook with pre-release, release-time, and post-release checklists.
- `SECURITY.md` — vulnerability reporting policy, supported versions, and security posture documentation.
- Six config field tables in README: `ETagConfig`, `RateLimitConfig`, `MetricsConfig`, `SecurityHeadersConfig`, `RequestIDConfig`, `ServerConfig`.
- `docs/v1-stability.md` — v1.0 frozen API surface definition.
- `docs/research/deny-unmatched-default-evaluation.md` — CORS secure-by-default analysis.
- Extensibility examples: `docs/integrations/brotli-zstd.md`, `docs/integrations/redis-ratelimiter.md`, `docs/integrations/prometheus-metrics.md`.
- Fuzz tests: `FuzzParseUintQuery`, `FuzzCORSOriginMatching`, `FuzzEvictionTTL`, `FuzzHealthHandler`.
- Benchmark: `BenchmarkTokenBucketLimiter` and `BenchmarkTokenBucketLimiterWithEviction`.
- Example functions: `ExampleParseUintQuery`, `ExampleReadyHandlerWithProbe`.
- Test for compression custom factory without Reset support (covers `startCompression` fresh-writer path).
- Validate success-path tests for `MetricsConfig` and `RateLimitConfig`.
- CORS edge-case tests: wildcard with port, empty allowlist with DenyUnmatched.
- Health handler exact-byte test guarding JSON encoding stability.
- CONTRIBUTING.md expanded: govulncheck in quality gate, versioning policy, CHANGELOG contribution rules, nix flake app inventory, minimum Go version policy.
- README badges: coverage, govulncheck, Go version, license.
- `health.go` doc comment documenting `json.Encoder` trailing newline behavior.

## [0.6.1] - 2026-07-26

### Fixed

- **Build no longer requires `GOEXPERIMENT=jsonv2`.** `health.go` used `encoding/json/v2` and `json.MarshalWrite` (a Go 1.27 API), which broke `go build ./...` for anyone consuming the library via `go get` without the experiment flag. Reverted to `encoding/json` v1 (`json.NewEncoder`); response bytes are identical (`{"status":"up"}\n`). The `GOEXPERIMENT=jsonv2` workaround was removed from `flake.nix` (7 insertion points), `.github/workflows/ci.yml`, `README.md`, `CONTRIBUTING.md`, and `AGENTS.md`.

### Changed

- Compression writer error paths unified through a shared `compressWriteError` helper — every `ErrCodeCompressWriteFailed` now carries the negotiated `encoding` context uniformly (two buffered-write paths previously omitted it).
- `go-error-family` upgraded from v0.7.0 to v0.9.0.
- Go toolchain directive bumped from `1.26.4` to `1.26.5`.
- Nix flake inputs refreshed (nixpkgs, treefmt-nix) for reproducible builds.
- `docs/DOMAIN_LANGUAGE.md` corrected: compression descriptions updated from gzip-only to multi-encoding.
- Inline correction banners added to 3 historical HTML reports (`modularity.html`, `full-code-review.html`, `code-quality-scan.html`) — stale metric cards corrected in the first screenful.

### Added

- `.editorconfig` — enforces consistent tab indentation, trailing-whitespace trimming, UTF-8, and final-newline policy across editors and IDEs.
- `golang.org/x/time/rate` node added to both D2 architecture diagrams; both SVGs regenerated.

## [0.6.0] - 2026-07-22

### Added

- New `ParseUintQuery(r *http.Request, key string) uint` function: extracts a base-10 unsigned integer from a named query parameter. Returns 0 if missing, empty, or invalid.
- WebSocket upgrade integration test (`websocket_upgrade_test.go`): drives a real TCP connection through Compression + ETag, performs a full 101 Switching Protocols handshake, and asserts no Content-Encoding/ETag injection plus intact post-hijack byte exchange.

### Changed

- **Breaking:** `TokenBucketLimiter` now uses `golang.org/x/time/rate` internally. `NewTokenBucketLimiter(rate float64, burst int)` — the `burst` parameter changed from `float64` to `int` to match `rate.NewLimiter`. Token-refill math is now the library's responsibility; idle-bucket eviction and clock injection for tests are preserved.
- New dependency: `golang.org/x/time v0.15.0` (canonical Go extension for rate limiting). `.golangci.yml` depguard updated to allow it.
- Upgraded `go-error-family` from v0.6.1 to v0.7.0.
- `GOEXPERIMENT=jsonv2` is now required to build — `health.go` imports `encoding/json/v2` and uses `json.MarshalWrite`. Contributors must set this env var or pin Go 1.27+.
- Health handlers now write a trailing newline after the JSON response body.
- Replaced deprecated `mkShell` with `mkShellNoCC` in `flake.nix`.
- CI: replaced `govulncheck-action@v1` with a `go run` approach.

## [0.5.0] - 2026-07-06

### Added

- New `ReadyHandlerWithProbe(ready func() bool)` health handler: returns 200 `{"status":"up"}` when the probe function returns true, or 503 `{"status":"down"}` when it returns false. Enables Kubernetes to route traffic away from instances that are alive but not yet ready to serve (e.g., warming caches, waiting on dependencies).
- New `TokenBucketLimiter.EvictionTTL` field: opt-in lazy eviction of idle buckets. When non-zero, buckets not accessed within the TTL are swept on the next `Allow` call (amortized, at most one sweep per TTL interval). Zero (default) preserves the original unbounded-growth behavior.
- New `ETagConfig.HashFunc` field: pluggable hash algorithm for ETag computation. Defaults to FNV-64a (`defaultETagHash`). Provide a custom `func([]byte) uint64` for application-specific hashing.
- New `DefaultWriterFactoriesForLevel(level int)` function: returns a fresh default factory map at any compression level. `DefaultWriterFactories()` delegates to this at `gzip.DefaultCompression`.
- New `CORSConfig.DenyUnmatched` field: when true, withholds the `Access-Control-Allow-Origin` header for origins that match no `AllowedOrigins` entry (when `AllowAllOrigins` is false). Causes browsers to deny the cross-origin request, preventing the configured allowlist from being bypassed via the wildcard fallback.

### Changed

- **Breaking:** `NewTokenBucketLimiter` now returns `(*TokenBucketLimiter, error)` and validates that rate and burst are positive. Previously silently created broken limiters (rate <= 0 never refills tokens; burst <= 0 rejects every request). Acceptable at pre-1.0.
- `Compression()` now builds default factories from `cfg.Level` when `WriterFactories` is empty, instead of ignoring `Level` entirely. Previously `Level` had no effect unless custom factories were supplied.
- Upgraded `go-error-family` from v0.5.1 to v0.6.1 (module metadata correction, no API change).
- Excluded gosec G705 (XSS via taint analysis) globally in `.golangci.yml`. G705 is structurally a false positive for a response-writing library — every `ResponseWriter.Write` is intentional output.

### Removed

- **Breaking:** `CompressionConfig.QValues` field removed. It was documented and exposed but never read by the negotiator — a lying public API.

### Fixed

- **Security:** CORS allowlist bypass — when `AllowAllOrigins` is false and a request origin matched no `AllowedOrigins` entry, the middleware fell back to `Access-Control-Allow-Origin: *`, making the configured allowlist decorative. Set `DenyUnmatched: true` to suppress the header for unmatched origins.
- **Correctness:** ETag collision risk — replaced CRC32 (32-bit checksum, 50% collision probability at ~65K distinct bodies) with FNV-64a (64-bit, birthday bound ~4 billion). ETag values change from 8 to 16 hex characters, requiring one-time cache invalidation.

## [0.4.0] - 2026-07-02

### Added

- New `httputil/httpspec` subpackage: reusable behavioral HTTP spec suite with 18 standard specs that validate any `http.Handler` against common HTTP conventions. Specs cover routing (index reachability, unknown path 404s, long URL handling), method safety (HEAD, OPTIONS, TRACE, POST, CONNECT rejection), response headers (Content-Type on bodies and errors, Location on redirects, no duplicate headers, Accept header handling), and security (no leaked internals, no Server version fingerprinting, no X-Powered-By header, X-Content-Type-Options: nosniff presence). Includes helper builders (`ExpectStatus`, `ExpectNotStatus`, `ExpectHeader`, `ExpectHeaderAbsent`, `ExpectBodyContains`) for custom specs and options (`SkipSpec`, `WithExtraSpecs`, `WithIndexPath`) for configuration. `RunSerial` variant for handlers with shared mutable state.
- New `MaxBodySize(maxBytes)` middleware: limits request body size via `http.MaxBytesReader`.
- New `RateLimit(cfg)` middleware: token bucket rate limiting with pluggable `RateLimiter` interface. Includes `TokenBucketLimiter` built-in implementation with per-key buckets, configurable key extraction, and custom denial handlers.
- New `Metrics(cfg)` middleware: records per-request metrics (method, path, status, duration) via pluggable `MetricsRecorder` interface.
- New `MiddlewareStack` type: collects named middleware with duplicate prevention, ordering validation (Recovery must be outermost), and `Build()` method. Well-known name constants for all built-in middleware.
- New `DetectCapabilities(w)` function and `Capabilities` type: reports which optional `http.ResponseWriter` interfaces (Hijacker, Flusher) a writer supports.
- New `DefaultIncompressibleTypes()` function: returns the default content-type deny-list for compression, enabling users to extend rather than replace the list.
- `CompressionConfig.IncompressibleTypes` field: configurable content-type filtering for compression middleware. Nil uses defaults, empty slice compresses everything.

## [0.3.0] - 2026-06-18

### Added

- `Compression()` now negotiates encodings from `Accept-Encoding` using RFC 7231 q-values and a server priority order (brotli > zstd > gzip > deflate > identity).
- `deflate` encoding support via `DeflateWriterFactory()` and `compress/flate`.
- `WriterFactory` plugin interface and `DefaultWriterFactories()` for adding custom encodings (brotli, zstd, lz4) without core dependencies.
- Per-encoding `sync.Pool` for writer reuse (owned by the negotiator for each `Compression` instance), plus buffer pre-allocation to `max(minSize, 512)` in `compressWriter`.
- New `id_generator.go`: time-ordered 16-byte request IDs (Unix seconds + atomic counter + random tail) with amortized `crypto/rand` buffering.
- New tests: `compression_negotiator_test.go` and `id_generator_test.go`.
- New exports: `DefaultWriterFactories()`, `GzipWriterFactory()`, `DeflateWriterFactory()`.

### Changed

- `RequestID` default `GenerateID` now produces sortable, monotonic 32-character hex IDs instead of fully random 16-byte hex IDs.
- `Compression` benchmark memory profile now reports higher bytes/op due to `httptest.ResponseRecorder.Body` growth; this is a measurement artifact, not a production regression.
- `CORS()` pre-computes the joined `Access-Control-Allow-Methods/Headers/Expose-Headers` and `Max-Age` strings once at construction instead of on every request (removes 2-3 allocations/response, ~36% faster).
- `Compression` negotiator fast-paths single-token `Accept-Encoding` headers (e.g. `gzip`), skipping the q-value scanner (~7x faster for that common case).
- Documentation updated: `README.md`, `FEATURES.md`, `TODO_LIST.md`, `AGENTS.md`, and `docs/research/performance-review.html`.
- Test suite passes with >90% statement coverage; adds `FuzzCORSWildcardPattern`, compression writer error-branch tests, and a `ResponseRecorder` hijack-failure test.

### Removed

- `util.go` deleted: the unexported `itoa()` and `join()` helpers are replaced by stdlib `strconv` and `strings`. No public API change. The prior benchmark portraying `strconv.Itoa` as allocating was a dead-code-elision measurement artifact — in real usage `strconv.Itoa` is allocation-free.

- `http.Pusher` support removed: HTTP/2 Server Push was removed from Chrome in 2023 and is not part of HTTP/3. All Pusher-related code, error codes (`ErrCodePushUnsupported`, `ErrCodePushFailed`), and tests have been removed. This is a **breaking change**.

### Fixed

- `Compression` writer pool leak: pools were keyed by the address of a function parameter, which is unique per call, so a new pool entry was created on every request and writers were never reused (an unbounded memory leak that also defeated the documented pooling). Pools are now owned by the negotiator and keyed by encoding name, bounded to one pool per encoding per `Compression` instance.

## [0.1.1] - 2026-06-08

### Fixed

- `CORS()`: Eliminated data race where `allowOrigin` was a shared mutable closure variable across concurrent requests
- `Compression()`: Added nil guard in gzip writer pool — `gzip.NewWriterLevel` errors now panic at construction time instead of producing nil writers
- Removed unreachable `errPoolTypeMismatch` dead code, replaced with `panic()` for impossible states

### Changed

- Updated CHANGELOG, AGENTS.md, and status docs to reflect accurate metrics (112 tests, 91.2% coverage)

## [0.1.0] - 2026-06-08

### Added

- CORS middleware with configurable origins, methods, headers, credentials, and preflight handling
- CORS wildcard origin matching (e.g., `*.example.com`)
- `CORSConfig.Validate()`, `CompressionConfig.Validate()`, `RequestIDConfig.Validate()`, `ETagConfig.Validate()`, `SecurityHeadersConfig.Validate()` — all config types have startup validation
- `DefaultCORSConfig()` with permissive development defaults (allows all origins)
- Client IP extraction (`ClientIP`) with `X-Forwarded-For` → `X-Real-IP` → `RemoteAddr` precedence
- `WithClientIP()`, `ClientIPFromContext()`, `ClientIPMiddleware()` context helpers
- `ResponseRecorder` with `WriteHeader`, `Write`, `Flush`, `Hijack` support
- `HeaderSnapshot()` for capturing response headers
- `Chain()` for composing middleware in declaration order (first = outermost)
- Security headers middleware (`SecurityHeaders`) with sensible defaults
- Request ID middleware (`RequestID`) with propagation and generation
- `RequestIDFromContext()` context helper
- Panic recovery middleware (`Recovery`) with stack trace logging
- Request timeout middleware (`Timeout`) with context deadline enforcement
- Structured request logging middleware (`Logging`)
- Response compression middleware (`Compression`) with gzip, `sync.Pool`, content-type filtering, and bounded buffering
- `CompressionConfig.Validate()` for startup configuration validation (gzip levels, min size)
- `RequestIDConfig.Validate()` for startup validation (nil GenerateID, empty headers)
- `ETagConfig.Validate()` for startup validation (non-positive MaxBufferSize)
- `SecurityHeadersConfig.Validate()` for startup validation (all fields optional, consistent API)
- Classified errors via `go-error-family` integration for `ResponseRecorder` (`Write`, `Hijack`), `compressWriter`, and `etagWriter`
- 5 error code constants: `ErrCodeWriteFailed`, `ErrCodeHijackUnsupported`, `ErrCodeHijackFailed`, `ErrCodeCompressWriteFailed`, `ErrCodeETagWriteFailed`
- `RegisterErrorClassifications()` for stdlib HTTP error mapping
- Error message templates (`what/why/fix/wayOut`) for all classified errors
- `wrapper.go` shared `ResponseWriter` wrapper eliminating ~80 lines of duplication from compress/etag writers
- 112 tests, 11 example functions, 15 benchmarks, 5 fuzz tests
- 91.2% test coverage
- `golangci-lint` with ~70 linters, 0 issues
- GitHub Actions CI workflow (test, lint, build, vet)
- Release workflow with `govulncheck`
- Nix flake for reproducible development environment
- `CHANGELOG.md`, `FEATURES.md`, `TODO_LIST.md`, `AGENTS.md`
- `docs/DOMAIN_LANGUAGE.md` with domain glossary
- `doc.go` package-level godoc
- `CONTRIBUTING.md` and `CODE_OF_CONDUCT.md`

### Changed

- `util.go`: Fixed `itoa` MinInt overflow bug with per-digit absolute value
- `ResponseRecorder`: `Write`, `Hijack` return classified errors instead of bare `fmt.Errorf`
- `ResponseRecorder`: Fixed nil-wrapping bug where successful operations returned non-nil errors
- `CORS()`: Fixed data race where `allowOrigin` was a shared mutable closure variable across concurrent requests

---

[Unreleased]: https://github.com/larsartmann/httputil/compare/v0.9.1...HEAD
[0.9.1]: https://github.com/larsartmann/httputil/compare/v0.9.0...v0.9.1
[0.9.0]: https://github.com/larsartmann/httputil/compare/v0.8.0...v0.9.0
[0.8.0]: https://github.com/larsartmann/httputil/compare/v0.7.1...v0.8.0
[0.7.1]: https://github.com/larsartmann/httputil/compare/v0.7.0...v0.7.1
[0.7.0]: https://github.com/larsartmann/httputil/compare/v0.6.1...v0.7.0
[0.6.1]: https://github.com/larsartmann/httputil/compare/v0.6.0...v0.6.1
[0.6.0]: https://github.com/larsartmann/httputil/compare/v0.5.0...v0.6.0
[0.5.0]: https://github.com/larsartmann/httputil/compare/v0.4.0...v0.5.0
[0.4.0]: https://github.com/larsartmann/httputil/compare/v0.3.0...v0.4.0
[0.3.0]: https://github.com/larsartmann/httputil/compare/v0.2.0...v0.3.0
[0.1.1]: https://github.com/larsartmann/httputil/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/larsartmann/httputil/releases/tag/v0.1.0
