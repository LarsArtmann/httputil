# Roadmap

> Long-term direction and raw ideas. Items here are NOT actionable tasks.
> When an idea is refined into bounded work, it moves to [TODO_LIST.md](TODO_LIST.md).
> Completed work is recorded in [CHANGELOG.md](CHANGELOG.md).

_Updated: 2026-09-11._

## Current Position

**v1.0.0 is cut locally (2026-09-10, annotated tag on `7a6d11b`, not yet pushed).** It freezes the API surface defined in [`docs/v1-stability.md`](docs/v1-stability.md): 18 middlewares, the composition API (`Compose`, `MiddlewareFunc.Then`, `MiddlewareStack.Middleware`), `Server.ListenerAddr`/`StartTLS`, the typed error model, 19-spec httpspec suite, 26 nightly fuzz targets, ~70 linters at 0 issues, 97.4% (httputil) / 98.6% (httpspec) race-enabled coverage. Post-tag HEAD already carries the panic-free finalization and a go-etag v0.3.0 pin — see CHANGELOG `[Unreleased]`.

The next release (v1.1.0) is the first stabilization release: removal of the deprecated `TokenBucketLimiter`/`RateLimit()` per the migration guide, then the deferred `go-compression` extraction when the go-datastar trigger fires.

## v1.0 — shipped (2026-09-10, local)

The "API is frozen" promise: after v1.0, breaking changes require a v2.0 major bump. The frozen surface is defined in [`docs/v1-stability.md`](docs/v1-stability.md).

Decisions taken on the way to v1.0:

- **Deprecated APIs stayed in v1.0.0** — `TokenBucketLimiter`/`RateLimit()` and the `httputil.ETag()` adapter ship deprecated; removal is the first post-v1.0 stabilization release (v1.1.0), tracked in [TODO_LIST.md](TODO_LIST.md).
- **Rate limiter interface refinement closed** — `AllowN` (burst > 1 per request) was evaluated and rejected (`KeyedRateLimiter` uses `MaxKeys`, not per-request burst). `context.Context` cancellation was evaluated in [docs/planning/archived/2026-08-29_21-30_rate-limiter-ctx-cancellation-design-note.md](docs/planning/archived/2026-08-29_21-30_rate-limiter-ctx-cancellation-design-note.md): v1.0 ships the admission-only contract (tokens consumed at admission, no refund on abort); a `Wait(ctx, key)` primitive remains the post-v1.0 additive evolution path.
- **Conditional-request scope** — ETag middleware lives in the independent `go-etag` module (`etag.New()` composes directly with httputil via the `Middleware` type alias; pinned v0.3.0). Conditional-request scope decisions (If-Match helpers, Last-Modified, If-Range) are evaluated in go-etag.

## Post-v1.0 ideas

- **Ecosystem extensions (plugin-shaped, documented examples rather than core deps)** — brotli/zstd `WriterFactory` implementations (`docs/integrations/brotli-zstd.md` is the pattern); a Prometheus `MetricsRecorder`; a Redis-backed keyed rate-limiter store; a samber/do composition-root guide (live: `docs/integrations/samber-do.md`); HTMX helper ideas (per-request `Vary`/nonce-aware fragment headers). Each fits an existing plugin interface; none belong in core.
- **HSTS middleware** — `Strict-Transport-Security` with configurable max-age/includeSubDomains/preload. Deferred: HSTS is a policy decision that belongs to the deployer, but a validated config type would match the established `SecurityHeadersConfig` pattern.
- **HTTPS-redirect helper** — a `RedirectToHTTPS` middleware behind `X-Forwarded-Proto` awareness. Deferred: proxy-dependent semantics (the header is spoofable without a trusted proxy, the same trust model as `ClientIP`).
- **`MaxHeaderBytes` on ServerConfig** — Go's default 1 MiB is sane for this library's audience; a validated field would mirror `MaxBodySize`. Deferred until a real consumer need appears; note that `http.Server.MaxHeaderBytes` already exists and `NewServer` could pass it through with one line when needed.
- **Parked legacy brainstorm ideas (May–June 2026 sessions, never pursued)** — circuit-breaker middleware, request-header validation middleware, OpenTelemetry tracing integration, mutation testing, a fluent middleware-builder API, a JSON error-response helper, CORS regex origin matching, a `Version` constant, CODEOWNERS, a runnable `example/` directory, a body-capturing `ResponseRecorder` variant, `ResponseRecorder` context propagation, `http.ResponseWriter` wrapper-interface standardization, CORS origin map/trie + pre-joined config strings, per-request-string pre-computation audit, benchmark-regression CI gates, real-world-payload-size benchmarks, HTTP/2 integration tests, generating the AGENTS architecture table from code. None have consumer demand behind them; revisit individually only if one appears. Sources: the `2026-05-24`–`2026-06-17` status report f-lists.
- **CSP nonce extensions (post-v1.0 candidate batch)** — ideas floated across the 2026-08-08 nonce audits and still unimplemented: a `StrictCSP` preset (Mozilla template), CSP Report-Only / `ReportURI` support for staged rollouts, hash-source (`Nonce-SHA256`) allowlisting, per-route nonce injection (`NonceMiddlewareWhen`), automatic `Cache-Control: no-store` pairing, and a nonce httpspec spec. Each is additive config surface. Sources: `2026-08-08_03-20` f8/f12/f16, `2026-08-08_06-54` f3/f15/f16.
- **CSRF hardening follow-ups (post-v1.0 candidate batch)** — Referer-based attestation contradiction (today only `Origin` is cross-checked; needs browser-behavior evidence first); normalizing/stripping client-sent `Sec-Fetch-Site` on trusted-proxy plain HTTP (policy decision); `http.NoBody`/nil-body fuzz coverage for the CSRF multipart path; documenting the `SetIsTLSFunc` vs `X-Forwarded-Proto` constraint for TLS-terminating proxy deployments. Sources: `04-03:f40/f41/f46/f50`.
- **Ideas surfaced 2026-09-10, not yet decided** — property-test the middleware stack ordering rules (Recovery-outermost) like the limiter heap invariants; an execution-probe test pinning the `ServeTLS` ALPN h2-append mutation; committing representative fuzz corpus seeds for the CORS echo oracle; a CI check that committed `.d2` diagrams still render; `MiddlewareFunc.Append(...)` (gorilla parity) decide-or-decline; a quarterly consumer/importer scan re-feeding the boundary verdict; a coverage-badge refresh step in the release runbook; a multi-module tag-annotation convention decision (single-tag-with-replace is the documented pattern). Sources: `09-26:f42/f45–f49`, `06-09:c6–c7`, `03-41:f24/f33`.
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
- **Vendoring (`vendor/` directory)** — removed 2026-09-11 (owner decision). Every dependency is public; workspace and `GOWORK=off` builds resolve from the module cache. Vendoring bought offline builds we never needed, and its modules.txt staleness caused the 2026-09-11 pipeline incident. `.gitignore` still lists `vendor/` as a guard against accidental reintroduction.
- **Hand-rolled CSRF implementation** — `justinas/nosurf` was added because double-submit cookie CSRF is security-critical and complex. Re-implementing it would be a liability.
- **Removing `nopCloserWriter` / `nopFlushCloser`** — defensive scaffolding for the `WriterFactory` contract; kept for API safety.
- **Removing `TokenBucketLimiter` before the first post-v1.0 stabilization release** — deprecated, shipped in v1.0.0 for compatibility; removal is scheduled work now (TODO_LIST v1.1.0 batch), no longer a question.
- **Property-based tests for token bucket** — existing benchmarks and integration tests cover the contract; adding rapid/quickcheck would violate the dependency policy. Moot once the v1.1.0 removal lands.
- **`MustNewTokenBucketLimiter`** — would add code to a deprecated API.
- **All `Must*`-style APIs (`MustAdd` on `MiddlewareStack`, etc.)** — owner decision 2026-09-10: `Must*` functions panic and panics are rejected as an API design tool. Wiring errors return errors from `Add` (callers choose how to fail); no Must variants will be added to any type.
- **`AllowN` on the rate limiter interface** — evaluated and rejected: `KeyedRateLimiter` uses `MaxKeys` and per-key capacity, not per-request burst; `AllowN` is the wrong primitive.
- **Exporting `delegatingWriter`** — internal ResponseWriter plumbing; not part of the public API.
- **Wrapping post-header-commit body-write errors** — unreportable in Go's Handler model; the "honest silence" contract is documented in AGENTS.md.
- **Re-exporting go-etag domain types from httputil** — the adapter exists for composition, not API duplication (see [docs/adr/0001](docs/adr/0001-adapter-pattern-for-external-middleware.md)).
- **Retry middleware** — application-layer concern (retrying outbound calls with backoff). No natural integration point in a server-side `func(http.Handler) http.Handler` middleware chain; a "retry middleware" would semantically mean replaying inbound requests through the handler, which is unsafe for non-idempotent methods. See `docs/status/2026-08-07_08-39_dependency-review-go-retry-go-idempotency.md`.
