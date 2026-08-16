# TODO List — httputil

Short- and mid-term improvement tasks. Each item verified against the actual code.

_Updated: 2026-08-07 — all prior items shipped including `ExampleMaxBodySize`. See [CHANGELOG.md](CHANGELOG.md) `[Unreleased]` for shipped work._

---

## High Priority

- [ ] **Extract response compression into `go-compression`** — full Pareto plan with 27 medium / 110 fine tasks: [docs/planning/2026-08-16_08-03_extract-compression-into-go-compression.md](docs/planning/2026-08-16_08-03_extract-compression-into-go-compression.md). Trigger: go-datastar needs SSE-safe compression without dragging codec deps into its root module. Epics: (1) mechanical move of 16 files to a new repo with green tests, (2) infra + tag v0.1.0 + httputil adapter migration, (3) SSE flush-guarantee tests + optional brotli/zstd subpackages + go-datastar README row flip, (4) benchmarks, pkg.go.dev, docs reconciliation. Decompression stays in httputil.

## Medium Priority

- [ ] **Upgrade historical report annotations to per-item** — the 22 `docs/status/2026-08-*.md` reports have header-level annotation banners, not inline `~~item~~` strikethrough on every numbered item. Strict docs-health ANNOTATE compliance requires per-item markers. Prioritize the 5 most-read reports. Effort: 1-2hr.

## Low Priority

_(none — `ExampleMaxBodySize` shipped)_

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
