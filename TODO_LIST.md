# TODO List — httputil

Short- and mid-term improvement tasks. Each item verified against the actual code.

_Updated: 2026-08-29 — rebuilt via HARVEST over the 23 August `## Resolution` appendices. Execution order and subtask breakdown: [docs/planning/2026-08-29_20-24_superb-annotation-legacy-and-v1-hardening-plan.md](docs/planning/2026-08-29_20-24_superb-annotation-legacy-and-v1-hardening-plan.md) (T-numbers below reference its Level-1 table)._

---

## High Priority

- [ ] **Run the `brutal-self-review` skill and fix the top findings** (plan T3) — deferred 10+ consecutive sessions; once _claimed done but never run_. Sources: `docs/status/2026-08-05_10-32_…:f41`, `07-45:f7`, `05-45:f37`, `05-10:f38`, `02-40:f38`, `06-50:f36`, `00-51:f37`.
- [ ] **Run the `full-code-review` skill** — never executed (once claimed done without running, the exact integrity failure the corpus keeps relapsing into). Sources: `00-51:f45`, `05-45:f37`, `10-32:f42`, `22-43:f39`, `06-50:f37`.
- [ ] **v1.0 stabilization cycle → cut v1.0** (plan T18/T19 gate) — remove deprecated `TokenBucketLimiter`/`RateLimit()` per [docs/migrating-to-keyed-rate-limiter.md](docs/migrating-to-keyed-rate-limiter.md); decide `context.Context` cancellation on the keyed limiter; one clean stabilization pass; tag v1.0. Sources: `23-08:f7`, `07-10:f26`, ROADMAP "v1.0 — API stability commitment".

## Medium Priority

- [ ] **Extract response compression into `go-compression`** — full Pareto plan with 27 medium / 110 fine tasks: [docs/planning/2026-08-16_08-03_extract-compression-into-go-compression.md](docs/planning/2026-08-16_08-03_extract-compression-into-go-compression.md). Trigger: go-datastar needs SSE-safe compression without dragging codec deps into its root module. Decompression stays in httputil.
- [ ] **July–May report annotation upgrade pass** (plan T5) — same per-item treatment the 23 August reports received; sized first by the May–July banner inventory (T4). Sources: 20-09 status report g-Q1, plan gate Q1.
- [ ] **Upgrade ~200 highest-traffic `v`-markers to hash citations** (plan T14) — hash-only evidence policy for historical markers. Sources: 20-09 status report g-Q3, plan gate Q3.
- [ ] **CI release workflow** — automated release pipeline (tag → build → GitHub Release) so releases stop depending on in-shell ceremony. Sources: `23-33:f24`, `00-51:f24`.
- [ ] **go-error-family upstream: conditional-request classification** — propose `Corruption`/`Rejection` classification guidance for ETag/conditional-request error families upstream. Sources: `23-33:f40`, `22-43:f48`, `05-45:f47`.
- [ ] **`architecture-review` re-run** — last full structural pass predates ETag extraction + adapter + keyed limiter. Sources: `05-45:f38`, `06-50:f38`.

## Low Priority

- [ ] **Condense the 5 worst historical resolution tables** (plan T9; open-set preserved). Sources: `07-15:f20`, `10-32:f30`, `05-10:f24`.
- [ ] **httpspec small spec additions**: CORS wildcard-origin spec (`07-45:f28`), `RunSerial` state-sharing design note (`07-45:f47`), duplicate-header-KEY check + `Vary: *` + optional ETag spec (plan T24).
- [ ] **responseWrapper direct tests** + failure-recording behavior (`07-45:f41`, `06-50:f47`); negotiation property test (`07-10:f42`); `b.Run` sub-benchmark refactor (`08-52:f49`); adapter overhead benchmark (`22-43:f26`).
- [ ] **Hijack test design decision**: restore a real-connection Hijack test (deferred design call) + lighter upgrade-response test + mutation-test the Hijacker-preservation test. Sources: `06-12:c.3/f.2/f.4/f.7/f.9`.
- [ ] **Test-lint convention**: codify test-name/assertion style so new tests don't drift. Source: `06-12:f.6`.
- [ ] **`newTestRequest` noctx audit** — decide whether the shared helper can adopt `http.NewRequestWithContext` cleanly. Source: `07-45:f21`.
- [ ] **`httptest.NewRequest` profiling** — measure allocation cost behind the noctx warnings. Sources: `05-45:f27`, `00-51:f36`.
- [ ] **error-swallow classification sweep** — grep for discarded errors lacking the "honest silence" comment convention. Source: `06-50:f39`.
- [ ] **Upstream go-etag coordination** (plan T25): compliance-suite ownership doc, scratch-module consumer verification, v0.1.1 items, ordering-doc handoff. Sources: `23-33`, `22-43:f37`, `00-51:c1`.
- [ ] **Module hygiene extras**: module-boundary test (`06-44:f18`), workspace checker (`06-50:f50`), integration-docs import-path audit (`06-44:f16`), package-structure analysis refresh (`06-44:f22`), server-timing migration guide (`06-44:f25`), migrating-from-httputil-etag guide (`06-44:f32`), RELEASE.md decompression + module-release steps (`00-51:f22`, `06-50:f29`).
- [ ] **`.gitignore` fuzz testdata entry** + pre-commit-hook failure test. Sources: `00-51:c4/f18`, `c5/f13`.
- [ ] **AGENTS.md sibling-ecosystem note** (go-etag, server_timing, go-error-family cross-references). Source: `23-08:f9`.
- [ ] **nginx bomb-protection study** — research doc comparing nginx `client_max_body_size`/gzip behavior with `Decompression` bomb protection. Source: `05-45:f48`.

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
- **Treat the httpspec `cfg.indexPath` pattern as a data race** — verified NOT a bug (2026-08-29): options are applied before spec construction and the config is read-only afterwards; `go test -race -count=3 ./httpspec/...` is clean. The 07-45 appendix claim was stale and has been corrected in place.

## Open Questions (routed to user, not tasks)

- Daemon commit granularity preference (`07-45:g2`) — should the auto-git daemon batch docs-only changes per phase instead of per file?

---

_Long-term vision and raw ideas live in [ROADMAP.md](ROADMAP.md). Completed work is recorded in [CHANGELOG.md](CHANGELOG.md). Historical open-item evidence lives in the `## Resolution` appendices of `docs/status/2026-08-*.md` (those files remain the point-in-time record; this list is the live backlog)._
