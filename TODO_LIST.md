# TODO List — httputil

Short- and mid-term improvement tasks. Each item verified against the actual code.

_Updated: 2026-08-07 — ETag middleware re-integrated from `go-etag` module. All ETag types and tests are back in the root package. See [CHANGELOG.md](CHANGELOG.md) `[Unreleased]` for shipped work._

---

## High Priority

- [ ] **Add `nix run .#vulncheck` to RELEASE.md** — document the new vulncheck app in the release runbook step 4. `docs/RELEASE.md`. Effort: 10min.

## Medium Priority

- [ ] **Add `ServerConfig.TLSConfig` validation** — `TLSConfig` is always nil in `NewServer()` but there is no validation for it if added. `server.go`. Effort: 30min.
- [ ] **Audit all `Validate()` methods for completeness** — ensure every config struct has one. Effort: 30min.

## Low Priority

- [ ] **Benchmark decompression** — gzip + deflate + passthrough. `decompression_test.go`. Effort: 15min.
- [ ] **Fuzz test for decompression** — random compressed bodies. `decompression_test.go`. Effort: 30min.

## Won't Implement

These items were considered and rejected, with reasoning:

- **Remove `nopCloserWriter` / `nopFlushCloser`** — defensive scaffolding for the `WriterFactory` contract; only reachable via direct unit construction but kept for API safety. Documented in AGENTS.md.
- **Add `MustNewTokenBucketLimiter`** — `TokenBucketLimiter` is deprecated; new code uses `KeyedRateLimiter`. Dead code.
- **Property-based tests for token bucket** — existing benchmarks + integration tests cover the contract; rapid/quickcheck adds dependencies.
- **Add `AllowN` on rate limiter interface** — `KeyedRateLimiter` uses `MaxKeys`; `AllowN` is not the right primitive.
- **Make `delegatingWriter` exported** — internal; not part of the public API.
- **Wrap post-header-commit body-write errors** — these are fundamentally unreportable in Go's Handler model. `compressWriter` now documents this with honest `_, _ =` + explanatory comment instead of wrapping ceremony.

---

_Long-term vision and raw ideas live in [ROADMAP.md](ROADMAP.md). Completed work is recorded in [CHANGELOG.md](CHANGELOG.md)._
