# TODO List — httputil

Short- and mid-term improvement tasks. Each item verified against the actual code.

_Updated: 2026-08-05 — sourced from the 2026-08-05 status report harvest. All v0.8.0-cycle items shipped and recorded in [CHANGELOG.md](CHANGELOG.md) under `[Unreleased]`._

---

## Medium Priority

- [ ] **Add `MaxBodySize` config validation** — `MaxBodySize(maxBytes int64)` silently accepts negative values. Wrap in a config struct with `Validate()`, or add an explicit guard. `maxbodysize.go`. Estimated effort: 20min.
- [ ] **Add `ShutdownTimeout` validation to `ServerConfig.Validate()`** — `ShutdownTimeout` is the only `ServerConfig` field not checked. A zero or negative value means the server never shuts down. `server.go:61`. Estimated effort: 10min.

## Low Priority

- [ ] **Document `canonicalheader` lint asymmetry in `AGENTS.md`** — the linter triggers on `Header.Get(literal)` and literal constants in some positions, but NOT on `Header.Set(literal)` or `Header.Get(constant)`. Future authors hit a 3-iteration debug cycle before understanding this. Add a one-line note to the Hard Constraints section. Estimated effort: 10min.
- [ ] **Close coverage gaps in `cors_ratelimit_specs.go`** — 5 functions at 80-91%: `corsAllowCredentialsCheck` (80%), `corsVaryOriginCheck` (90.9%), `rateLimitRetryAfterCheck` (85.7%), `rateLimitHeaderOnRejectCheck` (84.6%), `rateLimitHintHeadersOnAllowCheck` (81.2%). These are the edge-case branches in the new httpspec specs (handlers that partially set CORS/rate-limit headers). Estimated effort: 30min.
- [ ] **Add `KeyExtractor` empty-return warning to `KeyedRateLimiterConfig`** — a `KeyExtractor` that always returns `""` silently disables rate limiting (all requests map to the same key). Document the footgun in the `KeyExtractor` type comment or validate at construction. `ratelimit_keyed.go`. Estimated effort: 15min.

## Won't Implement

These items were considered and rejected, with reasoning:

- **Remove `nopCloserWriter` / `nopFlushCloser`** — defensive scaffolding for the `WriterFactory` contract; only reachable via direct unit construction but kept for API safety. Documented in AGENTS.md.
- **Add `MustNewTokenBucketLimiter`** — `TokenBucketLimiter` is deprecated; new code uses `KeyedRateLimiter`. Dead code.
- **Property-based tests for token bucket** — existing benchmarks + integration tests cover the contract; rapid/quickcheck adds dependencies.
- **Add `AllowN` on rate limiter interface** — `KeyedRateLimiter` uses `MaxKeys`; `AllowN` is not the right primitive.
- **Make `delegatingWriter` exported** — internal; not part of the public API.

---

_Long-term vision and raw ideas live in [ROADMAP.md](ROADMAP.md). Completed work is recorded in [CHANGELOG.md](CHANGELOG.md)._
