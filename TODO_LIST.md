# TODO List — httputil

Short- and mid-term improvement tasks. Each item verified against the actual code. Completed work lives in [CHANGELOG.md](CHANGELOG.md) (`[1.0.0]` and below); rejected ideas live in [ROADMAP.md](ROADMAP.md) Non-goals; process decisions live in [docs/DECISION_LOG.md](docs/DECISION_LOG.md).

_Updated: 2026-09-11 (docs-health pass: post-tag drift reconciled, completed items moved to CHANGELOG)._

---

## High Priority

- [ ] **Reconcile the local `v1.0.0` tag with its own changelog, then push** — the tag (`7a6d11b`) predates the panic-free finalization (`45026c1`) that the frozen `[1.0.0]` section already documents (`MiddlewareFunc.Then(nil)` 500-stub, dead `crypto/rand` guards deleted). Since the tag was never pushed: either `git tag -d v1.0.0` and re-cut on a commit containing that code, or ship the delta in v1.0.1 and note it. Then the normal release sequence: the full-code-review below, local govulncheck on the release commit (both modules), the erraudit `--type-aware` advisory re-pass, `git push origin master && git push origin v1.0.0`, and per [docs/RELEASE.md](docs/RELEASE.md) verify pkg.go.dev + `go get`. Sources: docs-health VERIFY 2026-09-11 (git show v1.0.0:compose.go still panics on `Then(nil)`), `09-26:f1–f10`.
- [ ] **Run the scheduled full-code-review before the push** — the 2026-09-10 sweep landed substantial code (Server listener tracking, writerPool probe, ID-generator ring, CSRF Validate purity, composition API, ~30 tests/fuzz/bench targets). Scope: everything in the `[1.0.0]` changelog section. Sources: `09-26:b2/c1`, TODO_LIST (carried).

## Medium Priority

- [ ] **v1.1.0 stabilization batch (first post-v1.0 release)** — remove the deprecated `TokenBucketLimiter`/`RateLimit()` per [docs/migrating-to-keyed-rate-limiter.md](docs/migrating-to-keyed-rate-limiter.md) (plan T18), delete the `BenchmarkTokenBucketLimiter*` rows and the redis integration-doc deprecation scaffolding, check whether the `httputil.ETag()` adapter's removal window matured ([docs/v1-stability.md](docs/v1-stability.md) first), and refresh the go-error-family `require` to the latest patch. Sources: `09-26:f26–f29`, `04-03:f48`.
- [ ] **Extract response compression into `go-compression`** — full Pareto plan: [docs/planning/2026-08-16_08-03_extract-compression-into-go-compression.md](docs/planning/2026-08-16_08-03_extract-compression-into-go-compression.md). Trigger: go-datastar needs SSE-safe compression. **Deferred post-v1.0 (decision 2026-09-10):** the v1.0 cut takes precedence, and the `writerPool` resettability-probe refactor invalidated the plan's line-by-line inventory — the plan's own anti-Verschlimmbesserung guardrail requires a fresh inventory pass before execution. Preconditions: (1) refresh the plan inventory, (2) new-repo/remote setup by the owner, (3) execute per the plan's phased-commit discipline. Sources: `09-26:a25/c3/f30–f32`.
- [ ] **Finish test-helper consolidation** — `waitForTLS` moved + upgraded in the sweep, but `waitForServerStart` (timeout-heuristic) is now redundant with `waitForListenerAddr`; migrate its callers or delete the helper. Sources: `09-26:b4/f12`.
- [ ] **Re-inventory `docs/architecture-reference.md` tables against HEAD** — the 2026-09-10 sweep added 7+ files (per-middleware bench files, new fuzz targets, compose API details) that are not in the file-by-file tables; only the three rows known to change were refreshed. Sources: `09-26:b5/f13`.
- [ ] **Fuzz the CSRF attestation-conflict path with unsafe methods** — `FuzzCSRFMiddleware_OriginHeaders` seeds the contradictory combo as GET only, but `ErrCSRFAttestationConflict` applies to unsafe methods; add a method parameter + corpus seeds for the contradictory POST combos. Sources: `04-03:b5/f5`.
- [ ] **Composition examples + benchmarks** — `ExampleCompose` (empty-list identity as executable documentation), `ExampleMiddlewareFunc_Then`, `BenchmarkCompose`, `BenchmarkMiddlewareStack_Middleware`. The sweep added `ExampleMiddlewareStack` only. Sources: `03-41:b3/c4–c5/f11–f12`, `09-26:a12`.
- [ ] **Re-measure httpspec + server_timing benchmark baselines** under the documented 3s×5 protocol — never re-measured; `docs/benchmarks.md` explicitly scopes them out today. Sources: `09-26:c5/f15`.

## Low Priority

- [ ] **httpspec discovery push** — the 2026-09-10 importer scan (pkg.go.dev + Sourcegraph) found ZERO importers of `httpspec` anywhere; add a README section, a runnable example, and a docs-site page for the spec runner (owner decision: push discovery, not retire). Owner also confirms PRIVATE consumer repos exist beyond the 4 public ones. Sources: `06-09:c2/g1/g3`, `09-26` TODO carry.
- [ ] **File the verified go-error-family issue** — draft verified and saved: [docs/planning/2026-09-10_go-error-family-conditional-request-classification-issue-draft.md](docs/planning/2026-09-10_go-error-family-conditional-request-classification-issue-draft.md). Owner action (own repo). Sources: `09-26:a24/b3/f11`.
- [ ] **Add direct tests for the three 0% `Code` constructors** — `Code.Conflict`, `Code.Orchestration`, `Code.WrapRejection` remain 0% covered (fresh 2026-09-11 race profile); `WrapConflict`/`WrapOrchestration` got cause+family tests in the sweep, these three did not. Sources: docs-health VERIFY 2026-09-11 (`go tool cover -func`), FEATURES sub-100% list.
- [ ] **Add tests for `scripts/coverage-threshold`** — the only Go in the repo with 0% coverage (excluded from the gate for that reason). Sources: `09-26:f18`.
- [ ] **benchstat in the flake + KeyedRateLimiter regression sanity-check** — replace best-of-five eyeballing with benchstat, and confirm whether `KeyedRateLimiterMiddleware` at 220 ns/op (vs the historical ~191) is protocol noise or a real regression. Sources: `09-26:e6/f20/f22`.
- [ ] **CI hardening batch** — (1) gate `.golangci.yml`-touching commits on a clean `golangci-lint run` (the 13-finding regression proved the invariant decays); (2) upload bench artifacts in CI (benchmarks.md claims "full raw data in CI artifacts"; the job uploads nothing); (3) run the nightly-fuzz workflow manually once (`workflow_dispatch`) and verify the `bug` label exists; (4) add the `GOWORK=off` server_timing build to the flake checks; (5) `--skip-flake` flag for `prerelease-check.sh` (CI parity). Sources: `03-41:d1/f7`, `09-26:f7/f21/f35/f37/f43`.
- [ ] **Compile the integration-doc snippets** — markdown code examples in docs/integrations are never build-tested; the `Recovery(nil)` incident proves the risk. Harness: parse fences → generated test file, or duplicated minimal examples in `example_test.go`. Sources: `09-26:e3/f14`.
- [ ] **Resolve the gopls `stdversion` warnings** — 5 standing diagnostics claim `json.MarshalWrite` "requires go1.27" while go.mod pins 1.26.7; decide whether go.mod moves to 1.27 when json/v2 stabilizes. Sources: `03-41:d3`, `09-26:f19`.
- [ ] **CSRF docs follow-ups** — migration note for the new `ErrCSRFAttestationConflict` 403 (behavior change for consumers relying on forged attestations); mention the httpspec no-injection-header-reflection spec in the README httpspec block; surface `csrf.origin_attestation_conflict` in a consumer-facing classification example. Sources: `04-03:e7/f1/f42/f43`.
- [ ] **Decide `CSRFConfig.withParsedTrustedProxies` export post-v1.0** — revisit only if external consumers need to construct the CIDR form themselves. Sources: `09-26:f34`.

---

_Long-term vision and raw ideas live in [ROADMAP.md](ROADMAP.md). Completed work is recorded in [CHANGELOG.md](CHANGELOG.md). Historical open-item evidence lives in the inline `done at` markers of `docs/status/2026-*.md`; this list is the live backlog._
