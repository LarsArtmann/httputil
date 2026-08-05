# TODO List — httputil

Short- and mid-term improvement tasks. Each item verified against the actual code.

_Updated: 2026-08-05 — sourced from the 2026-08-05 status report harvest. All v0.8.0-cycle items shipped and recorded in [CHANGELOG.md](CHANGELOG.md) under `[Unreleased]`._

---

## Medium Priority

_(none — all medium-priority items shipped in this session)_

## Low Priority

- [ ] **Add `ServerConfig.TLSConfig` validation** — `TLSConfig` is always nil in `NewServer()` but there is no validation for it if added. Deferred to v1.0. `server.go`. Estimated effort: 30min.

## Won't Implement

These items were considered and rejected, with reasoning:

- **Remove `nopCloserWriter` / `nopFlushCloser`** — defensive scaffolding for the `WriterFactory` contract; only reachable via direct unit construction but kept for API safety. Documented in AGENTS.md.
- **Add `MustNewTokenBucketLimiter`** — `TokenBucketLimiter` is deprecated; new code uses `KeyedRateLimiter`. Dead code.
- **Property-based tests for token bucket** — existing benchmarks + integration tests cover the contract; rapid/quickcheck adds dependencies.
- **Add `AllowN` on rate limiter interface** — `KeyedRateLimiter` uses `MaxKeys`; `AllowN` is not the right primitive.
- **Make `delegatingWriter` exported** — internal; not part of the public API.

---

_Long-term vision and raw ideas live in [ROADMAP.md](ROADMAP.md). Completed work is recorded in [CHANGELOG.md](CHANGELOG.md)._
