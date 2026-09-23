# TODO List — httputil

Short- and mid-term improvement tasks. Each item verified against the actual code. Completed work lives in [CHANGELOG.md](CHANGELOG.md) (`[Unreleased]` and the frozen version sections); rejected ideas live in [ROADMAP.md](ROADMAP.md) Non-goals; process decisions live in [docs/DECISION_LOG.md](docs/DECISION_LOG.md).

_Updated: 2026-09-23 (evening sweep: high/medium/low execution pass — CI-green verification, nightly-fuzz crash #24 fixed+closed, CSRF fallback hardening trio, example gap batch, httpspec LNA preflight spec, CSRF docs+test-depth batches, LNA docs+verification polish, release-engineering trio, go-etag ecosystem verification, fleet etagmetrics sweep clean, LNA spec churn confirmed and recorded; done items deleted per the delete-done rule, residuals below)._

---

## High Priority

- [ ] **Decide and cut the next release: `v1.3.1` patch vs `v1.4.0` minor** — `[Unreleased]` carries the `etagmetrics` removal (+ HitRatio correction-of-record), the Godoc examples overhaul, the `server.go` Go-1.27 exhaustruct fix, and the go-etag v0.5.0 / Go 1.27.1 floor. A removal plus a dependency/toolchain floor suggests a minor; the correction-of-record alone would be a patch. Then RELEASE.md steps (incl. step 12.5: CI green on the exact tag commit), plus the pre-tag art-dupl sweep (`-t 2..25` zero floor and `-t 1` exactly-one-accepted-group baseline, both documented in AGENTS.md Commands). Timing is owner-gated. Sources: docs/status/2026-09-23_00-04_etagmetrics c2/f14, docs/status/2026-09-22_21-46 f17, CHANGELOG `[Unreleased]`.

## Medium Priority

- [ ] **Denied-origin LNA preflight: keep the header or suppress it?** — the denied-origin case is now pinned (`TestCORS_Preflight_PrivateNetworkHeaderSent_OnDeniedOrigin` + field-doc sentence, 2026-09-23); the residual is the owner security-posture question: should `AllowPrivateNetwork` be suppressed when the origin is denied, or stay unconditional as pinned? See the AGENTS.md spec-churn note (PNA on hold, LNA permission prompt in Chrome 142+). Source: docs/status/2026-09-15_05-47 f2/g3.
- [ ] **Confirm the B1+opt-out interpretation of the 2026-09-15 owner ruling** — "good defaults" shipped as `SameSite=None` → `Secure=true` fallback (NOT the `Lax` default). If the intended default was the Lax fallback, flip `withSecureFallback`, the tests, and the migration note. One word from the owner decides. Source: docs/status/2026-09-15_07-00 g1/f11.
- [ ] **Extract response compression into `go-compression`** — full Pareto plan: [docs/planning/2026-08-16_08-03_extract-compression-into-go-compression.md](docs/planning/2026-08-16_08-03_extract-compression-into-go-compression.md). Trigger: go-datastar needs SSE-safe compression. **Deferred post-v1.1 (decision 2026-09-10):** the `writerPool` resettability-probe refactor invalidated the plan's line-by-line inventory — the plan's own anti-Verschlimmbesserung guardrail requires a fresh inventory pass before execution. Preconditions: (1) refresh the plan inventory, (2) new-repo/remote setup by the owner, (3) execute per the plan's phased-commit discipline.
- [ ] **Re-run `architecture-review` post-v1.1** — the last full run predates the ETag extraction, keyed limiter, compose API, attestation defense + XFP trust model, the v1.1.0 removals, LNA, the SameSite fallback, pattern propagation, and the etagmetrics add/remove cycle. Sources: docs/status/2026-09-15_06-14 f15, docs/status/2026-09-10_04-03 c4.

## Low Priority

- [ ] **httpspec discovery push (docs-site page remains)** — README section + runnable example landed 2026-09-11; the importer scan still shows zero public importers, and private consumer repos exist. Remaining: a docs-site page for the spec runner (blocked on the website-launch effort). Owner decision: push discovery, not retire.
- [ ] **Export `csrf.trusted_origin_invalid` as an exported `Code` constant?** — consumers currently match it via the string; exporting is additive and frozen-API-safe (verified unexported 2026-09-23). Owner decision, standing since 2026-09-14. Sources: docs/status/2026-09-15_07-00 f21, docs/status/2026-09-14_18-14 f9/g3.
- [ ] **MD060 table-style ruling + buildflow result-cache purge** — standardize status/living docs on the single-space compact style (record in AGENTS.md) or add MD060 to the judged-disable list (verified not disabled 2026-09-23); either way purge the buildflow result-cache rows (7-day TTL replay). Owner style ruling. Source: docs/status/2026-09-15_06-14 b5/f6/e3.
- [ ] **go-error-family#5 follow-through** — issue filed and verified (exists, open); `gh issue view` rendering sanity done 2026-09-15. Remaining: implement the upstream "Conditional requests" docs section if accepted; if accepted, sweep the httputil README + architecture-reference classification tables with a link to the new guidance. Sources: docs/status/2026-09-15_06-14 b6/f7/f38.
- [ ] **jsonv2 stabilization follow-through** — bump go.mod when stdlib lands `encoding/json/v2` non-experimental (Go ≥ 1.28 expected) and drop `GOEXPERIMENT=jsonv2` from the documented erraudit commands in one change (DECISION_LOG 2026-09-11; clears the standing gopls `stdversion` warnings). Source: docs/status/2026-09-15_06-14 c12.
- [ ] **Write the external-claim raw-extraction patterns into a reference** — `gh api repos/<o>/<r>/contents/<path>` for raw markdown, `download`+grep for large specs, `mdn/browser-compat-data` JSON for version claims; proven in the 2026-09-15 sessions so the next `agentic_fetch` outage costs nothing. Also: report/fix the `agentic_fetch` json-unmarshal failure (environment tooling). Sources: docs/status/2026-09-15_07-00 f32/f33/e3.
- [ ] **`writeHealthBody` `_ =` discard — documented honest-silence vs propagation** — the health handler's post-header-commit body write discards its error by design; owner call whether it joins the documented honest-silence set explicitly or gains a log line. Standing since the 2026-09-11 release session. Source: docs/status/2026-09-11_13-49 f2.
- [ ] **Consolidate the erraudit-residual documentation** — the topic lives in three places (Commands-block verdict, BuildFlow "Residual detect-only findings" bullet, Non-Obvious Behaviors honest-silence paragraph); one canonical home + cross-references. Source: docs/status/2026-09-11_10-03 c3/f6.

## Post-v1.1 (frozen-API or v2.0 material — recorded, not fixed, in the 2026-09-11 full-code review)

- [ ] **`ValidateCSRF` returns `*httptest.ResponseRecorder`** from a production API (test-support type, hidden behaviors). Frozen v1.0 symbol; a result-type change is a signature change → v2.0 material. (Review finding 1.)
- [ ] **`MiddlewareFunc` vs `Middleware` alias split** — inline literals silently lack `.Then`; making `MiddlewareFunc` canonical deprecates the core alias → v2.0 discussion. (Review finding 7.)
- [ ] **Typed `MiddlewareStack` names** — `Add` takes untyped `string` names despite the `Middleware*` constant set; typed names are an additive candidate (new type + overload). `Build`-without-validate is documented and relied upon. (Review finding 8.)
- [ ] **Pool-contract hardening** — `compress_pool.go` probe doesn't check a `(nil, nil)` factory return, `acquire`'s factory parameter is ignored on the pooled path, `release` has no provenance check. Internal-contract hardening for the pool's next touch. (Review finding 9.)

---

_Long-term vision and raw ideas live in [ROADMAP.md](ROADMAP.md). Completed work is recorded in [CHANGELOG.md](CHANGELOG.md). Historical open-item evidence lives in the inline `done at` markers of `docs/status/2026-*.md`; this list is the live backlog._
