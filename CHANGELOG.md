# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## [Unreleased]

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

[Unreleased]: https://github.com/larsartmann/httputil/compare/v0.6.1...HEAD
[0.6.1]: https://github.com/larsartmann/httputil/compare/v0.6.0...v0.6.1
[0.6.0]: https://github.com/larsartmann/httputil/compare/v0.5.0...v0.6.0
[0.5.0]: https://github.com/larsartmann/httputil/compare/v0.4.0...v0.5.0
[0.4.0]: https://github.com/larsartmann/httputil/compare/v0.3.0...v0.4.0
[0.3.0]: https://github.com/larsartmann/httputil/compare/v0.2.0...v0.3.0
[0.1.1]: https://github.com/larsartmann/httputil/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/larsartmann/httputil/releases/tag/v0.1.0
