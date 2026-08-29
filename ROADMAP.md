# Roadmap

> Long-term direction and raw ideas. Items here are NOT actionable tasks.
> When an idea is refined into bounded work, it moves to [TODO_LIST.md](TODO_LIST.md).
> Completed work is recorded in [CHANGELOG.md](CHANGELOG.md).

_Updated: 2026-08-07._

## Current Position

v0.9.0 (released 2026-08-05) ships request body decompression, hardened `Validate()` methods across all config structs, `SecurityHeadersConfig` enrichment, `httpspec` CORS and rate-limit specs, CSRF fuzz tests, and a full-stack integration test.

v0.9.1 (2026-08-06) is an RFC 7232 compliance patch for the ETag middleware.

**In `[Unreleased]`:** Server-Timing extracted into a stdlib-only sub-module (`server_timing/`). ETag extracted to the independent `go-etag` module; `httputil.ETag()` adapter deprecated in favor of `etag.New()` (composes directly via the `Middleware` type alias). `ServerConfig.TLSConfig` validation, decompression benchmarks/fuzz, and govulncheck in the devShell. 96.9% httputil / 98.8% httpspec coverage with ~70 linters at 0 issues.

## v0.9.0 — Feature additions

- **Request body decompression middleware** — counterpart to `Compression` for gzip/deflate-encoded request bodies. **Shipped in v0.9.0.** Includes decompression bomb protection (configurable `MaxDecompressionSize`, default 16 MiB).

Additional v0.8.0 follow-up work (CSRF fuzz tests, `httpspec` CORS and rate-limit specs, full-stack integration test, benchmarks, `Validate()` audit, MaxBodySize validation, ShutdownTimeout, coverage gap closure, CI hardening) shipped in v0.9.0.

## v1.0 — API stability commitment

v1.0 marks the "API is frozen" promise. After v1.0, breaking changes require a v2.0 major bump. The frozen surface is defined in [`docs/v1-stability.md`](docs/v1-stability.md).

Before v1.0:

- **Remove the deprecated `TokenBucketLimiter` / `RateLimiter` interface** — superseded by `KeyedRateLimiter`. Migration guide: [`docs/migrating-to-keyed-rate-limiter.md`](docs/migrating-to-keyed-rate-limiter.md).
- **Rate limiter interface refinement** — `AllowN` (burst > 1 per request) was evaluated and rejected (`KeyedRateLimiter` uses `MaxKeys`, not per-request burst). `context.Context` cancellation was evaluated in [docs/planning/2026-08-29_21-30_rate-limiter-ctx-cancellation-design-note.md](docs/planning/2026-08-29_21-30_rate-limiter-ctx-cancellation-design-note.md): v1.0 keeps the admission-only contract (tokens consumed at admission, no refund on abort); a `Wait(ctx, key)` primitive stays available as a post-v1.0 additive evolution.
- **Conditional-request scope** — ETag middleware lives in the independent `go-etag` module (`etag.New()` composes directly with httputil via the `Middleware` type alias). Conditional-request scope decisions (If-Match helpers, Last-Modified, If-Range) are evaluated in go-etag.
- One stabilization cycle before the commitment.

## Post-v1.0 ideas

- **Ecosystem extensions (plugin-shaped, documented examples rather than core deps)** — brotli/zstd `WriterFactory` implementations (`docs/integrations/brotli-zstd.md` is the pattern); a Prometheus `MetricsRecorder`; a Redis-backed keyed rate-limiter store; a samber/do composition-root guide (live: `docs/integrations/samber-do.md`); HTMX helper ideas (per-request `Vary`/nonce-aware fragment headers). Each fits an existing plugin interface; none belong in core.
- **HSTS middleware** — `Strict-Transport-Security` with configurable max-age/includeSubDomains/preload. Deferred: HSTS is a policy decision that belongs to the deployer, but a validated config type would match the established `SecurityHeadersConfig` pattern.
- **HTTPS-redirect helper** — a `RedirectToHTTPS` middleware behind `X-Forwarded-Proto` awareness. Deferred: proxy-dependent semantics (the header is spoofable without a trusted proxy, the same trust model as `ClientIP`).
- **`MaxHeaderBytes` on ServerConfig** — Go's default 1 MiB is sane for this library's audience; a validated field would mirror `MaxBodySize`. Deferred until a real consumer need appears; note that `http.Server.MaxHeaderBytes` already exists and `NewServer` could pass it through with one line when needed.
- **Idempotency-key middleware** — Stripe-style `Idempotency-Key` middleware is a legitimate httputil-shaped concern, but deferred to post-v1.0 to avoid scope creep against the API freeze. If pursued, define a native `IdempotencyStore` interface (Get/Save with TTL) rather than importing `go-idempotency` — its Store only dedupes keys (seen/not-seen), not the response body needed to replay a prior result. The `ResponseRecorder` captures status/headers/body but is not designed as a replay primitive; a separate cache type would be needed. See `docs/status/2026-08-07_08-39_dependency-review-go-retry-go-idempotency.md`.

## Dependency policy

Stdlib + `go-error-family` (same author, zero transitive deps) + `go-etag` (same author, ETag conditional requests) + `golang.org/x/time` (canonical Go rate-limit extension) + `github.com/justinas/nosurf` (CSRF double-submit cookie — security-critical, complex to hand-roll). Extensibility for encoders (brotli/zstd/lz4), distributed rate limiters (Redis), and metrics (Prometheus) is exposed via plugin interfaces with documentation examples in [`docs/integrations/`](docs/integrations/), not core dependencies.

## Non-goals

Things we are deliberately NOT pursuing and why:

- **HTTP/2 Server Push** — removed in Chrome 2023, absent from HTTP/3. All `http.Pusher` code deleted in v0.3.0.
- **Streaming ETag with a rolling hash** — ETag has been extracted to the `go-etag` module. HTTP requires headers before the body, so buffering is mandatory regardless of where the middleware lives.
- **Internal `compress/` subpackage** — compression files depend on root symbols (`Middleware`, `responseWrapper`, `ErrCode*`), so extracting creates a circular import. The flat layout is structural (confirmed 2026-08-05).
- **Built-in brotli/zstd encoders** — kept as `WriterFactory` plugin examples to preserve the dependency policy.
- **Functional options (`With*`) pattern** — the struct-config + `Validate()` pattern is established and consistent. Functional options would create two parallel configuration styles.
- **Hand-rolled CSRF implementation** — `justinas/nosurf` was added because double-submit cookie CSRF is security-critical and complex. Re-implementing it would be a liability.
- **Removing `nopCloserWriter` / `nopFlushCloser`** — defensive scaffolding for the `WriterFactory` contract; kept for API safety.
- **Removing `TokenBucketLimiter` before v1.0** — deprecated, but removal waits for the v1.0 stability freeze to avoid breaking consumers.
- **Property-based tests for token bucket** — existing benchmarks and integration tests cover the contract; adding rapid/quickcheck would violate the dependency policy.
- **Retry middleware** — application-layer concern (retrying outbound calls with backoff). No natural integration point in a server-side `func(http.Handler) http.Handler` middleware chain; a "retry middleware" would semantically mean replaying inbound requests through the handler, which is unsafe for non-idempotent methods. See `docs/status/2026-08-07_08-39_dependency-review-go-retry-go-idempotency.md`.
