# TODO List — httputil

Short- and mid-term improvement tasks. Each item verified against the actual code. Completed work lives in [CHANGELOG.md](CHANGELOG.md) (`[Unreleased]`, `[1.0.1]`, `[1.0.0]`); rejected ideas live in [ROADMAP.md](ROADMAP.md) Non-goals; process decisions live in [docs/DECISION_LOG.md](docs/DECISION_LOG.md).

_Updated: 2026-09-15 (harvest from the dnsblockd adoption analysis: CORS LNA gap recorded under Medium Priority; no completed items)._

---

## High Priority

- [x] ~~**Push `master`**~~ done 2026-09-11 (`20f6917..ae0a46d` pushed; pkg.go.dev proxy verified via consumer `go get` + build of both modules — see v1.1.0 item below).
- [x] ~~**Re-dispatch `nightly-fuzz.yml` and verify step 1 passes**~~ done 2026-09-11 (run 34574182326: anchored `^FuzzDecompression$` step PASS; no new false-positive issues, #7/#8 stay closed).
- [x] ~~**Cut `v1.1.0`**~~ done 2026-09-11 (owner ruled "Release a new version!"): all 9 prerelease gates green, tag SSH-signed and verified, GitHub release published with the CHANGELOG section as notes, consumer `go get`/`go build` verified for `httputil v1.1.0` + `server_timing v1.0.1` (sub-module had zero drift — no new sub-module tag).

## Medium Priority

- [ ] **CORS: `Access-Control-Allow-Private-Network` (Chrome LNA) preflight support** — the one hard blocker an external consumer audit found to full CORS adoption: dnsblockd's 82-line local CORS middleware exists solely to answer Chrome Local Network Access preflights, and its analysis rules migration "only after upstream grows an LNA option". Verified in-tree: zero `Private-Network` matches. Sketch: opt-in config field (secure default off), respond `Access-Control-Allow-Private-Network: true` on preflight only when the client requested it; verify semantics against current Chrome LNA docs at design time (the header contract has churned across Chrome versions). Source: `~/projects/dnsblockd/docs/research/2026-09-15_httputil-adoption-analysis.md`.
- [ ] **CSRF security-degrading config: log-only vs remediate-to-secure-defaults** — `Validate()`-and-log continues on dangerous combos (e.g. `SameSite=None` + `Secure=false`) instead of remediating to secure defaults; changing the validate-and-log contract overturns a documented decision (DECISION_LOG 2026-08-08) and needs its own design pass. Owner decision pending (question ③ above). Source: full-code-review finding 2.
- [x] ~~**Unify the two `TrustedOrigins` parsers**~~ done 2026-09-14 (`parseTrustedOriginURLs` is now all-or-nothing, mirroring `nosurf.StaticOrigins` wholesale-failure semantics, so the attestation check can never trust a different set than the nosurf handler; `Validate` additionally rejects entries that are not usable `scheme://host` origins via the new `csrf.trusted_origin_invalid` Rejection code — template, registry list, and classification tables updated; 6 new tests including a fail-closed middleware pin). Source: full-code-review finding 3.
- [x] ~~**Reclassify stdlib error registrations**~~ done 2026-09-14 (`http.ErrNoCookie`/`ErrNoLocation` registered Rejection — deterministic absence, retry cannot succeed; `ErrAbortHandler`/unsupported sentinels pinned by the extended classification test; `ErrCodeHijackFailed` doc + template Fix wording aligned while the family stays Transient per the documented taxonomy; own CHANGELOG entry under `[Unreleased]`). Source: full-code-review finding 4.
- [x] ~~**Document `ValidateCSRF` caller-request mutation**~~ done 2026-09-14 (doc comment now lists every in-place mutation — `Sec-Fetch-Site` bypass fill, `X-Csrf-Token` translation + form parse — and the nosurf isolation; the context flag was considered and declined: it requires either a signature change (v2.0 material, review finding 1) or `*r` reassignment (racy) — see DECISION_LOG 2026-09-14). Source: full-code-review finding 5.
- [x] ~~**`nosurf.StaticOrigins` failure is fail-open (origin function silently unset)**~~ done 2026-09-14 together with the parser unification above: the fail-closed treatment now mirrors `withParsedTrustedProxies` — a config with any unparseable entry gets no trusted origins on either side (nosurf origin function unset, attestation list nil) and construction logs the fallback loudly; behavior pinned by `TestCSRFMiddleware_UnparseableTrustedOriginFailsClosed`. Source: full-code-review finding 6.
- [ ] **Extract response compression into `go-compression`** — full Pareto plan: [docs/planning/2026-08-16_08-03_extract-compression-into-go-compression.md](docs/planning/2026-08-16_08-03_extract-compression-into-go-compression.md). Trigger: go-datastar needs SSE-safe compression. **Deferred post-v1.1 (decision 2026-09-10):** the `writerPool` resettability-probe refactor invalidated the plan's line-by-line inventory — the plan's own anti-Verschlimmbesserung guardrail requires a fresh inventory pass before execution. Preconditions: (1) refresh the plan inventory, (2) new-repo/remote setup by the owner, (3) execute per the plan's phased-commit discipline.
- [ ] **Re-run `architecture-review` post-v1.1** — the last full run predates the ETag extraction, keyed limiter, compose API, attestation defense + XFP trust model, and the v1.1.0 removals. Source: `04-03:c4/f14`, `03-41:a1`.

## Post-v1.1 (frozen-API or v2.0 material — recorded, not fixed, in the 2026-09-11 full-code review)

- [ ] **`ValidateCSRF` returns `*httptest.ResponseRecorder`** from a production API (test-support type, hidden behaviors). Frozen v1.0 symbol; a result-type change is a signature change → v2.0 material. (Review finding 1.)
- [ ] **`MiddlewareFunc` vs `Middleware` alias split** — inline literals silently lack `.Then`; making `MiddlewareFunc` canonical deprecates the core alias → v2.0 discussion. (Review finding 7.)
- [ ] **Typed `MiddlewareStack` names** — `Add` takes untyped `string` names despite the `Middleware*` constant set; typed names are an additive candidate (new type + overload). `Build`-without-validate is documented and relied upon. (Review finding 8.)
- [ ] **Pool-contract hardening** — `compress_pool.go` probe doesn't check a `(nil, nil)` factory return, `acquire`'s factory parameter is ignored on the pooled path, `release` has no provenance check. Internal-contract hardening for the pool's next touch. (Review finding 9.)

## Low Priority

- [ ] **httpspec discovery push (docs-site page remains)** — README section + runnable example landed 2026-09-11; the importer scan still shows zero public importers, and private consumer repos exist. Remaining: a docs-site page for the spec runner (blocked on the website-launch effort). Owner decision: push discovery, not retire.
- [ ] **File the verified go-error-family issue** — draft verified and saved: [docs/planning/2026-09-10_go-error-family-conditional-request-classification-issue-draft.md](docs/planning/2026-09-10_go-error-family-conditional-request-classification-issue-draft.md). Owner action (own repo).
- [ ] **Decide `CSRFConfig.withParsedTrustedProxies` export post-v1.1** — revisit only if external consumers need to construct the CIDR form themselves. Sources: `09-26:f34`.
- [ ] **Re-run the two review passes lost to LLM rate limits** — sweep test targets: chain_test.go, server_test.go, ratelimit_keyed_test.go, id_generator tests; scripts/examples scope. Owner question ② pending; everything else was covered by 4 completed passes or direct authorship.
- [ ] **Next release: migration-note the two `[Unreleased]` behavior changes** — (1) TrustedOrigins parsing is all-or-nothing and `Validate` now rejects non-`scheme://host` entries (consumers relying on partial parsing or scheme-less entries see behavior changes); (2) `http.ErrNoCookie`/`ErrNoLocation` classifications changed. Add a release-notes callout (or `docs/migrating-*.md`) and the `docs/v1-stability.md` Versioning Policy bullet — that doc has no behavioral-delta section beyond version-scoped bullets (verified 2026-09-14).

---

_Long-term vision and raw ideas live in [ROADMAP.md](ROADMAP.md). Completed work is recorded in [CHANGELOG.md](CHANGELOG.md). Historical open-item evidence lives in the inline `done at` markers of `docs/status/2026-*.md`; this list is the live backlog._
