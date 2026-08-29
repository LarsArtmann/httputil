# TODO List — httputil

Short- and mid-term improvement tasks. Each item verified against the actual code.

_Updated: 2026-08-29 (evening) — execution pass complete for plan tasks T1–T17 minus gates; remaining open work below. Execution order and subtask breakdown: [docs/planning/2026-08-29_20-24_superb-annotation-legacy-and-v1-hardening-plan.md](docs/planning/2026-08-29_20-24_superb-annotation-legacy-and-v1-hardening-plan.md) (appendix records decisions)._

---

## High Priority

- [ ] **Run the `full-code-review` skill** — never executed; the last known-unknown surfacer before v1.0. Sources: `00-51:f45`, `05-45:f37`, `10-32:f42`, `22-43:f39`, `06-50:f37`. (brutal-self-review ran 2026-08-29: `docs/reviews/2026-08-29_20-42_brutal-self-review.html`.)
- [ ] **v1.0 release decision → cut v1.0** — one stabilization cycle, then per [docs/migrating-to-keyed-rate-limiter.md](docs/migrating-to-keyed-rate-limiter.md) remove deprecated `TokenBucketLimiter`/`RateLimit()` (plan T18) and confirm the rate-limiter admission contract per [docs/planning/2026-08-29_21-30_rate-limiter-ctx-cancellation-design-note.md](docs/planning/2026-08-29_21-30_rate-limiter-ctx-cancellation-design-note.md). Sources: `23-08:f7`, ROADMAP "v1.0".

## Medium Priority

- [ ] **Extract response compression into `go-compression`** — full Pareto plan: [docs/planning/2026-08-16_08-03_extract-compression-into-go-compression.md](docs/planning/2026-08-16_08-03_extract-compression-into-go-compression.md). Trigger: go-datastar needs SSE-safe compression. Decompression stays in httputil.
- [ ] **CI release workflow** — automated tag → build → GitHub Release pipeline. Sources: `23-33:f24`, `00-51:f24`.
- [ ] **go-error-family upstream: conditional-request classification** — propose classification guidance upstream. Sources: `23-33:f40`, `22-43:f48`, `05-45:f47`.
- [ ] **`architecture-review` re-run** — last full pass predates ETag extraction + adapter + keyed limiter. Sources: `05-45:f38`, `06-50:f38`.

## Low Priority

- [ ] **httpspec small spec additions**: CORS wildcard-origin spec (`07-45:f28`), `RunSerial` state-sharing design note (`07-45:f47`). (Vary/304/dup-KEY builders shipped 2026-08-29.)
- [ ] **responseWrapper direct tests** + negotiation property test (`07-10:f42`); adapter overhead benchmark (`22-43:f26`).
- [ ] **Hijack test design decision**: real-connection Hijack test + lighter upgrade-response test + mutation-test the Hijacker-preservation test. Sources: `06-12:c.3/f.2/f.4/f.7/f.9`.
- [ ] **Test-lint convention**: codify test-name/assertion style. Source: `06-12:f.6`.
- [ ] **`httptest.NewRequest` profiling** — measure allocation cost behind the noctx warnings. Sources: `05-45:f27`, `00-51:f36`.
- [ ] **error-swallow classification sweep** — grep for discarded errors lacking the "honest silence" comment. Source: `06-50:f39`.
- [ ] **Module hygiene extras**: integration-docs import-path audit (`06-44:f16`), package-structure analysis refresh (`06-44:f22`).
- [ ] **nginx bomb-protection study** — compare nginx behavior with `Decompression` bomb protection. Source: `05-45:f48`.
- [ ] **b.Run sub-benchmark refactor** (`08-52:f49`) — modernize multi-phase benchmarks to `b.Run` groups.

## Won't Implement

These items were considered and rejected, with reasoning:

- **Remove `nopCloserWriter` / `nopFlushCloser`** — defensive scaffolding for the `WriterFactory` contract; kept for API safety. Documented in AGENTS.md.
- **Add `MustNewTokenBucketLimiter`** — deprecated API; dead code.
- **Property-based tests for token bucket** — benchmarks + integration tests cover the contract; rapid/quickcheck adds dependencies.
- **Add `AllowN` on rate limiter interface** — `KeyedRateLimiter` uses `MaxKeys`; not the right primitive.
- **Make `delegatingWriter` exported** — internal; not part of the public API.
- **Wrap post-header-commit body-write errors** — unreportable in Go's Handler model; "honest silence" documented in AGENTS.md.
- **Re-export go-etag domain types** — the adapter exists for composition, not API duplication (see [docs/adr/0001](docs/adr/0001-adapter-pattern-for-external-middleware.md)).
- **Retry middleware** (`go-retry`) — application-layer concern; no natural integration point. See `docs/status/2026-08-07_08-39_dependency-review-go-retry-go-idempotency.md`.
- **Idempotency-key middleware** — deferred to post-v1.0. See [ROADMAP.md](ROADMAP.md).
- **Treat the httpspec `cfg.indexPath` pattern as a data race** — verified NOT a bug (2026-08-29): options apply before spec construction; `go test -race -count=3 ./httpspec/...` clean.
- **Condense the historical resolution tables** — after the per-item upgrade, the tables are the evidence appendix the inline markers cite; condensing would break the evidence chain (plan T9 skip decision, 2026-08-29).
- **Mandatory hash-only evidence for state-claim markers** — state-claims ("verified clean in every later session") have no single hash; policy is hash-for-changes, dated falsifiable evidence for state-claims (plan T14 appendix, 2026-08-29).

---

_Long-term vision and raw ideas live in [ROADMAP.md](ROADMAP.md). Completed work is recorded in [CHANGELOG.md](CHANGELOG.md). Historical open-item evidence lives in the `## Resolution` appendices of `docs/status/2026-*.md`; this list is the live backlog._
