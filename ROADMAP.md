# Roadmap

> Long-term direction and raw ideas. Items here are NOT actionable tasks.
> When an idea is refined into bounded work, it moves to [TODO_LIST.md](TODO_LIST.md).
> Completed work is recorded in [CHANGELOG.md](CHANGELOG.md).

_Updated: 2026-08-05._

## Current Position

v0.8.0 (released 2026-07-31) ships a complete 16-middleware suite, server lifecycle, health checks, error classification, an `httpspec` BDD subpackage, ~70 linters at 0 issues, and 97.6% httputil / 99.3% httpspec coverage (measured 2026-08-05 with race detection enabled). New in v0.8.0: CSRF protection, W3C Server-Timing, and keyed rate limiting.

The next release is **v0.9.0** (feature additions), followed by **v1.0** (API stability commitment).

## v0.9.0 — Feature additions

- **Request body decompression middleware** — counterpart to `Compression` for gzip-encoded request bodies (e.g. `Content-Encoding: gzip` POSTs). Rounds out compression symmetry. Highest-impact remaining raw idea.

Additional v0.8.0 follow-up work (CSRF fuzz tests, `httpspec` CORS and rate-limit specs, full-stack integration test, benchmarks, `Validate()` audit) shipped post-release — see [CHANGELOG.md](CHANGELOG.md) under `[Unreleased]`.

## v1.0 — API stability commitment

v1.0 marks the "API is frozen" promise. After v1.0, breaking changes require a v2.0 major bump. The frozen surface is defined in [`docs/v1-stability.md`](docs/v1-stability.md).

Before v1.0:

- **Remove the deprecated `TokenBucketLimiter` / `RateLimiter` interface** — superseded by `KeyedRateLimiter`. Migration guide: [`docs/migrating-to-keyed-rate-limiter.md`](docs/migrating-to-keyed-rate-limiter.md).
- **Rate limiter interface refinement** — evaluate `AllowN` (burst > 1 per request) and `context.Context` cancellation support on `KeyedRateLimiter`. Deferred to v1.0 because either could shape the final interface.
- One stabilization cycle (v0.9.0) before the commitment.

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
