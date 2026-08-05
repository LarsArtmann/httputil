# TODO List — httputil

Short- and mid-term improvement tasks. Each item verified against the actual code.

_Updated: 2026-08-05 — sourced from v0.8.0 release cycle and 2026-08-05 docs-health pass._

---

## High Priority

The v0.8.0 release bottleneck tasks are complete. No high-priority items remain.

- [x] **Close coverage gaps for new middleware** — coverage closed from 91.0% to 97.8% httputil / 98.9% httpspec. All three new middleware (CSRF, Server-Timing, KeyedRateLimit) have comprehensive tests covering ValidateCSRF, TranslateCSRFHeaders, CSRFTokenHXHeaders, isTrustedProxy, Validate, delegatingWriter delegation, eviction heap, and callback paths. Remaining 13 sub-100% functions are unreachable defensive code paths (json.Marshal error on map[string]string, crypto/rand panic paths, stale-heap mismatch branches).
- [x] **Add `MiddlewareStack` name constants for new middleware** — `MiddlewareCSRF`, `MiddlewareServerTiming`, `MiddlewareKeyedRateLimit` added to `stack.go` (commit `46dd59d`).

## Medium Priority

Five items from the v0.7.0 → v0.8.0 cycle remain open. Each is bounded and well-scoped.

- [ ] **Add CSRF fuzz tests** — fuzz origin matching, token validation, and TrustedCIDR parsing. CSRF processes untrusted input; existing coverage is deterministic via integration tests but a fuzz test would harden the security boundary. Estimated effort: 60min.
- [ ] **Add `httpspec` spec for CORS headers** — extend `httpspec` with a check that validates `Vary: Origin`, `Access-Control-Allow-Origin`, and `Access-Control-Allow-Credentials` semantics. Estimated effort: 30min.
- [ ] **Add `httpspec` spec for rate-limit headers** — extend `httpspec` with `Retry-After`, `X-RateLimit-Limit`, `X-RateLimit-Remaining`, `X-RateLimit-Reset` checks. Estimated effort: 30min.
- [ ] **Add integration test for full middleware stack** — chain all 16 middlewares in recommended order, verify composition, and assert no order-dependent breakage. Estimated effort: 30min.
- [ ] **Modernize `server_timing_bench_test.go`** — migrate `b.N` loops to `b.Loop()` (Go 1.24+ pattern) to clear 6 gopls warnings. Estimated effort: 10min.

## Low Priority

Polish and future work. Each is bounded and well-scoped.

- [ ] **Add `BenchmarkKeyedRateLimiter`** — measure allow/reject throughput with various `MaxKeys` and `EvictionTTL` settings. Estimated effort: 30min.
- [ ] **Add `BenchmarkCSRFMiddleware`** — measure per-request cost including nosurf integration. Estimated effort: 30min.
- [ ] **Add `Example*` function for `KeyedRateLimiterMiddleware`** — required by `testableexamples` linter for full coverage. Estimated effort: 10min.
- [ ] **Add `Example*` function for `ServerTimingMiddleware`** — required by `testableexamples` linter. Estimated effort: 10min.
- [ ] **Add `Example*` function for `CSRFMiddleware`** — required by `testableexamples` linter. Estimated effort: 10min.
- [ ] **Make README coverage badge dynamic** — wire coverage badge to CI output instead of hardcoded 97.8%. Estimated effort: 30min.
- [ ] **Audit all `Validate()` methods for completeness** — verify `ServerConfig`, `CORSConfig`, `CompressionConfig`, `ETagConfig`, `RequestIDConfig`, `SecurityHeadersConfig`, `MetricsConfig`, `RateLimitConfig`, `KeyedRateLimiterConfig`, `CSRFConfig` all surface actionable errors. Estimated effort: 60min.

## Won't Implement

These items were considered and rejected, with reasoning:

- **Remove `nopCloserWriter` / `nopFlushCloser`** — defensive scaffolding for the `WriterFactory` contract; only reachable via direct unit construction but kept for API safety. Documented in AGENTS.md.
- **Add `MustNewTokenBucketLimiter`** — `TokenBucketLimiter` is deprecated; new code uses `KeyedRateLimiter`. Dead code.
- **Property-based tests for token bucket** — existing benchmarks + integration tests cover the contract; rapid/quickcheck adds dependencies.
- **Add `ServerConfig.TLSConfig` validation** — deferred to v1.0 (breaking schema change).
- **Add request body decompression middleware** — deferred to v0.9.0 (ROADMAP).
- **Add `context.Context` support in rate limiter interface** — deferred to v1.0 (API design).
- **Add `AllowN` on rate limiter interface** — `KeyedRateLimiter` uses `MaxKeys`; `AllowN` is not the right primitive.
- **Make `delegatingWriter` exported** — internal; not part of the public API.

---

_Long-term vision and raw ideas live in [ROADMAP.md](ROADMAP.md). Completed work is recorded in [CHANGELOG.md](CHANGELOG.md)._
