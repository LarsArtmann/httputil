# Roadmap

> Long-term direction and raw ideas. Items here are NOT actionable tasks.
> When an idea is refined into bounded work, it moves to TODO_LIST.md.

_Updated: 2026-07-30._

## Current Position

The library is past v0.7.1 (released 2026-07-29) with unreleased feature work: CSRF protection, W3C Server-Timing, and keyed rate limiting. These three middleware features are coded and tested but not yet released and not yet at full coverage (91.0% overall, down from 98.7%). The next release will be v0.8.0.

## Themes

### 1. v1.0 — API honesty and stability commitment

Resolved in v0.7.0:

- ~~Rename `RequestIDConfig.ForwardHeader` to `IncomingHeader`~~ — done in v0.7.0.
- ~~Rename `RequestIDConfig.HeaderName` to `ResponseHeader`~~ — done in v0.7.0.
- ~~Evaluate flipping `DenyUnmatched` default to `true`~~ — done in v0.7.0
  (see `docs/research/deny-unmatched-default-evaluation.md`).
- ~~Define which APIs are frozen at v1.0~~ — done: `docs/v1-stability.md`.

Before v1.0:

- Close coverage on the three new middleware features (CSRF, Server-Timing,
  KeyedRateLimit) and classify them in `docs/v1-stability.md` as
  Frozen/Additive/Evolving. The current 91.0% must recover toward 98%+.
- Decide whether the deprecated `TokenBucketLimiter` / `RateLimiter` interface
  is removed at v1.0 or carried as deprecated through v1.x.
- One stabilization cycle (v0.8.0) before the v1.0 commitment.

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

Remaining raw ideas:

- Request body decompression middleware as a counterpart to `Compression`.
- Rate limiter interface refinement: the `KeyedRateLimiter` supersedes the
  deprecated `RateLimiter` interface, but `AllowN` (burst > 1 per request) and
  `context.Context` cancellation support are not yet evaluated.

### 3. Depth and confidence

The suite is broad (16 middlewares); v1.0 should make it deep enough to trust
without audit.

Raw ideas:

- Fuzz tests and `Example*` functions for the newer surface (CSRF,
  Server-Timing, KeyedRateLimit) — matching the pattern established by CORS,
  ETag, Compression, and RequestID.
- Close the remaining coverage gaps in compression error branches, CORS, ETag,
  and the new middleware. Several gaps are genuinely hard (crypto/rand error
  injection, real server shutdown) and may need test-infrastructure design, not
  just more tests.
- An `httpspec` spec covering common CORS header behavior.
- An `httpspec` spec covering rate-limit headers (`Retry-After`,
  `X-RateLimit-*`).
- Property-based tests for token bucket behavior.

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
