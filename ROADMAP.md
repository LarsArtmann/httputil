# Roadmap

> Long-term direction and raw ideas. Items here are NOT actionable tasks.
> When an idea is refined into bounded work, it moves to TODO_LIST.md.

_Updated: 2026-08-05 — sourced from v0.8.0 release (commit `8a77900`)._

## Current Position

The library is at v0.8.0 (released 2026-07-31) with a complete 16-middleware suite, server lifecycle, health checks, error classification, an `httpspec` BDD subpackage, ~70 linters at 0 issues, and 97.8% httputil / 98.3% httpspec coverage. v0.8.0 ships CSRF protection, W3C Server-Timing, and keyed rate limiting. The deprecated `TokenBucketLimiter` is slated for removal at v1.0.

The next release will be **v0.9.0**, focused on feature additions (request body decompression, additional `httpspec` specs). The **v1.0** release will follow, marking the stability commitment.

## Themes

### 1. v1.0 — API stability commitment

Resolved in v0.7.0:

- ~~Rename `RequestIDConfig.ForwardHeader` to `IncomingHeader`~~ — done in v0.7.0.
- ~~Rename `RequestIDConfig.HeaderName` to `ResponseHeader`~~ — done in v0.7.0.
- ~~Evaluate flipping `DenyUnmatched` default to `true`~~ — done in v0.7.0
  (see `docs/research/deny-unmatched-default-evaluation.md`).
- ~~Define which APIs are frozen at v1.0~~ — done: `docs/v1-stability.md`.

Resolved in v0.8.0:

- ~~Close coverage on the three new middleware features~~ — done; 97.8% httputil coverage.
- ~~Classify new middleware in `docs/v1-stability.md`~~ — done; CSRF, Server-Timing, and KeyedRateLimit classified as Frozen/Additive.

Before v1.0:

- Decide whether the deprecated `TokenBucketLimiter` / `RateLimiter` interface is removed at v1.0 or carried as deprecated through v1.x. **Current plan: remove at v1.0.** Migration guide at `docs/migrating-to-keyed-rate-limiter.md`.
- One stabilization cycle (v0.9.0) before the v1.0 commitment.
- v1.0 release represents the "API is frozen" promise. After v1.0, breaking changes require a v2.0 major bump.

### 2. Extensibility without new dependencies

The dependency policy has expanded: stdlib + `go-error-family` +
`golang.org/x/time` + `justinas/nosurf`. The first three are zero-friction
(same-author or canonical Go extension); nosurf is a deliberate choice for CSRF
(double-submit cookie is complex and security-critical to hand-roll).

Resolved (documented examples exist):

- ~~Brotli / zstd / lz4 encoder examples via the `WriterFactory` plugin
  pattern~~ — `docs/integrations/brotli-zstd.md`.
- ~~A distributed (Redis-backed) `RateLimiter` implementation~~ —
  `docs/integrations/redis-ratelimiter.md`.
- ~~A Prometheus-compatible `MetricsRecorder` implementation~~ —
  `docs/integrations/prometheus-metrics.md`.

Remaining raw ideas (rough priority order):

- **Request body decompression middleware** as a counterpart to `Compression`. This is the highest-impact remaining raw idea — it rounds out the compression feature for request bodies (e.g., gzip-encoded `application/json` POSTs). Targeted for v0.9.0.
- **Rate limiter interface refinement**: the `KeyedRateLimiter` supersedes the deprecated `RateLimiter` interface, but `AllowN` (burst > 1 per request) and `context.Context` cancellation support are not yet evaluated. Deferred to v1.0.
- **Property-based tests for token bucket behavior**. Currently covered by integration tests + benchmarks. Deferred indefinitely.

### 3. Depth and confidence

The suite is broad (16 middlewares); v1.0 should make it deep enough to trust without audit.

Resolved in v0.8.0:

- ~~Fuzz tests and `Example*` functions for the newer surface (CSRF,
  Server-Timing, KeyedRateLimit)~~ — partial. Example functions exist for all three new middleware (via `ExampleCSRFMiddleware`, `ExampleServerTimingMiddleware`, `ExampleKeyedRateLimiterMiddleware`). Fuzz tests for Server-Timing CRLF injection exist; CSRF and KeyedRateLimiter fuzz tests remain in TODO_LIST.

Raw ideas:

- `httpspec` spec covering common CORS header behavior.
- `httpspec` spec covering rate-limit headers (`Retry-After`, `X-RateLimit-*`).
- Property-based tests for token bucket behavior (also listed in Theme 2).
- An integration test that chains all 16 middlewares in recommended order.

## Non-goals

Things we are deliberately NOT pursuing and why:

- **HTTP/2 Server Push:** removed in Chrome 2023, absent from HTTP/3. All
  `http.Pusher` code was deleted in v0.3.0.
- **Streaming ETag with a rolling hash:** HTTP requires headers before the body,
  so buffering is mandatory. The FNV-64a + 1MB buffer is correct and optimal.
- **Internal `compress/` subpackage:** evaluated and rejected — compression
  files depend on root symbols (`Middleware`, `responseWrapper`, `ErrCode*`),
  so extracting them creates a circular import. The flat layout is structural.
- **Built-in brotli/zstd encoders:** kept as `WriterFactory` plugin examples to
  preserve the dependency policy. Adding a compression codec as a core
  dependency would break depguard.
- **Functional options (`With*`) pattern:** the struct-config + `Validate()`
  pattern is established and consistent. Introducing functional options would
  create two parallel configuration styles.
- **Hand-rolled CSRF implementation:** `justinas/nosurf` was added as a
  dependency because double-submit cookie CSRF is security-critical and
  complex. Re-implementing it would be a liability, not a feature.
- **Removing `nopCloserWriter` / `nopFlushCloser`:** defensive scaffolding for
  the `WriterFactory` contract. Only reachable via direct unit construction
  but kept for API safety.
- **Removing `TokenBucketLimiter` before v1.0:** the API is deprecated but
  removal must wait for v1.0 stability freeze to avoid breaking consumers
  mid-development.
