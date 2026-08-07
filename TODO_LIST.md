# TODO List — httputil

Short- and mid-term improvement tasks. Each item verified against the actual code.

_Updated: 2026-08-07 — full docs-health audit. All prior items shipped (TLSConfig validation, Validate() audit, decompression benchmarks/fuzz, vulncheck in RELEASE.md). See [CHANGELOG.md](CHANGELOG.md) `[Unreleased]` for shipped work._

---

## High Priority

- [ ] **Update `stack_integration_test.go` for ETag** — `buildFullStack` does not include ETag and the comment still says "chains all 16 middlewares" (should be 17). ETag (`MiddlewareETag`) was re-added as an adapter but never integrated into the canonical full-stack composition test. `stack_integration_test.go:14`. Effort: 15min.
- [ ] **Add ETag positioning guidance to README** — ETag must be placed **inside** (after) Compression so it hashes the uncompressed body, not the compressed bytes. This ordering constraint is non-obvious and undocumented in the README middleware ordering section. `README.md`. Effort: 10min.

## Medium Priority

- [ ] **Remove unused `assertBodyEmpty` from `testutil_test.go`** — dead code flagged by gopls (`unusedfunc`). The function exists at `testutil_test.go:182` but no test calls it (it was used by the old in-package ETag tests, now extracted to go-etag). Effort: 2min.
- [ ] **Add `MiddlewareDecompression` constant to `stack.go`** — Decompression is the only middleware missing a `Middleware*` stack name constant (all 12 others have one). Adding it enables `MiddlewareStack.Add(MiddlewareDecompression, ...)`. `stack.go`. Effort: 5min.
- [ ] **Add `BenchmarkCompressionNegotiator`** — the `Accept-Encoding` negotiation logic runs on every request but has no dedicated benchmark. `compression_negotiator.go`. Effort: 15min.

## Won't Implement

These items were considered and rejected, with reasoning:

- **Remove `nopCloserWriter` / `nopFlushCloser`** — defensive scaffolding for the `WriterFactory` contract; only reachable via direct unit construction but kept for API safety. Documented in AGENTS.md.
- **Add `MustNewTokenBucketLimiter`** — `TokenBucketLimiter` is deprecated; new code uses `KeyedRateLimiter`. Dead code.
- **Property-based tests for token bucket** — existing benchmarks + integration tests cover the contract; rapid/quickcheck adds dependencies.
- **Add `AllowN` on rate limiter interface** — `KeyedRateLimiter` uses `MaxKeys`; `AllowN` is not the right primitive.
- **Make `delegatingWriter` exported** — internal; not part of the public API.
- **Wrap post-header-commit body-write errors** — these are fundamentally unreportable in Go's Handler model. `compressWriter` now documents this with honest `_, _ =` + explanatory comment instead of wrapping ceremony.
- **Re-export go-etag domain types** (type aliases like `type ETagConfig = etag.ETagConfig`) — decided against; the adapter exists for middleware composition, not to duplicate go-etag's API surface. Consumers import go-etag directly for config and domain types.
- **Retry middleware** (`go-retry`) — application-layer concern (retrying outbound calls); no natural integration point in a server-side middleware chain. See `docs/status/2026-08-07_08-39_dependency-review-go-retry-go-idempotency.md`.
- **Idempotency-key middleware** (`go-idempotency`) — legitimate httputil-shaped concern but deferred to post-v1.0. Would need a native `IdempotencyStore` interface, not a hard dependency. See [ROADMAP.md](ROADMAP.md).

---

_Long-term vision and raw ideas live in [ROADMAP.md](ROADMAP.md). Completed work is recorded in [CHANGELOG.md](CHANGELOG.md)._
