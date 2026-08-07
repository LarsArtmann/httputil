# Roadmap

> Long-term direction and raw ideas. Items here are NOT actionable tasks.
> When an idea is refined into bounded work, it moves to [TODO_LIST.md](TODO_LIST.md).
> Completed work is recorded in [CHANGELOG.md](CHANGELOG.md).

_Updated: 2026-08-07._

## Current Position

v0.9.0 (released 2026-08-05) ships request body decompression, hardened `Validate()` methods across all config structs, `SecurityHeadersConfig` enrichment, `httpspec` CORS and rate-limit specs, CSRF fuzz tests, and a full-stack integration test. 96.7% httputil / 99.3% httpspec coverage with ~70 linters at 0 issues.

v0.9.1 (2026-08-06) is an RFC 7232 compliance patch: ETag `If-None-Match` now uses the weak comparison function, and list parsing respects commas inside quoted opaque-tags.

The next milestone is **v1.0** (API stability commitment).

## v0.9.0 — Feature additions

- **Request body decompression middleware** — counterpart to `Compression` for gzip/deflate-encoded request bodies. **Shipped in v0.9.0.** Includes decompression bomb protection (configurable `MaxDecompressionSize`, default 16 MiB).

Additional v0.8.0 follow-up work (CSRF fuzz tests, `httpspec` CORS and rate-limit specs, full-stack integration test, benchmarks, `Validate()` audit, MaxBodySize validation, ShutdownTimeout, coverage gap closure, CI hardening) shipped in v0.9.0.

## v1.0 — API stability commitment

v1.0 marks the "API is frozen" promise. After v1.0, breaking changes require a v2.0 major bump. The frozen surface is defined in [`docs/v1-stability.md`](docs/v1-stability.md).

Before v1.0:

- **Remove the deprecated `TokenBucketLimiter` / `RateLimiter` interface** — superseded by `KeyedRateLimiter`. Migration guide: [`docs/migrating-to-keyed-rate-limiter.md`](docs/migrating-to-keyed-rate-limiter.md).
- **Rate limiter interface refinement** — evaluate `AllowN` (burst > 1 per request) and `context.Context` cancellation support on `KeyedRateLimiter`. Deferred to v1.0 because either could shape the final interface.
- **Conditional-request scope decision** — the ETag middleware handles `If-None-Match` (GET/HEAD → 304) only. Open questions for v1.0 scope: (a) Should we add `If-Match` / `If-Unmodified-Since` (412 Precondition Failed for unsafe methods like PUT/DELETE/PATCH)? This widens the middleware from GET/HEAD-only or requires a separate precondition middleware. (b) Should we add `Last-Modified` generation + `If-Modified-Since` handling (the timestamp-based half of RFC 7232)? (c) Should we add `If-Range` support for partial-content (206) responses? These define whether httputil is a "caching" library or a "full conditional-request" library.
- **ETag `SkipIfPresent` decision** — the ETag middleware always overwrites handler-set ETags. Adding `SkipIfPresent bool` to `ETagConfig` would let handlers with domain-specific modification semantics win. Changing the default would be a breaking behavior change; adding the option is additive. Decide before v1.0 freeze.
- One stabilization cycle before the commitment.

## Dependency policy

Stdlib + `go-error-family` (same author, zero transitive deps) + `golang.org/x/time` (canonical Go rate-limit extension) + `github.com/justinas/nosurf` (CSRF double-submit cookie — security-critical, complex to hand-roll). Extensibility for encoders (brotli/zstd/lz4), distributed rate limiters (Redis), and metrics (Prometheus) is exposed via plugin interfaces with documentation examples in [`docs/integrations/`](docs/integrations/), not core dependencies.

## Non-goals

Things we are deliberately NOT pursuing and why:

- **HTTP/2 Server Push** — removed in Chrome 2023, absent from HTTP/3. All `http.Pusher` code deleted in v0.3.0.
- **Streaming ETag with a rolling hash** — HTTP requires headers before the body, so buffering is mandatory. FNV-64a + 1MB buffer is correct and optimal.
- **Internal `compress/` subpackage** — compression files depend on root symbols (`Middleware`, `responseWrapper`, `ErrCode*`), so extracting creates a circular import. The flat layout is structural (confirmed 2026-08-05).
- **Built-in brotli/zstd encoders** — kept as `WriterFactory` plugin examples to preserve the dependency policy.
- **Functional options (`With*`) pattern** — the struct-config + `Validate()` pattern is established and consistent. Functional options would create two parallel configuration styles.
- **Hand-rolled CSRF implementation** — `justinas/nosurf` was added because double-submit cookie CSRF is security-critical and complex. Re-implementing it would be a liability.
- **Removing `nopCloserWriter` / `nopFlushCloser`** — defensive scaffolding for the `WriterFactory` contract; kept for API safety.
- **Removing `TokenBucketLimiter` before v1.0** — deprecated, but removal waits for the v1.0 stability freeze to avoid breaking consumers.
- **Property-based tests for token bucket** — existing benchmarks and integration tests cover the contract; adding rapid/quickcheck would violate the dependency policy.
