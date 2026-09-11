# TODO List — httputil

Short- and mid-term improvement tasks. Each item verified against the actual code. Completed work lives in [CHANGELOG.md](CHANGELOG.md) (`[Unreleased]`, `[1.0.1]`, `[1.0.0]`); rejected ideas live in [ROADMAP.md](ROADMAP.md) Non-goals; process decisions live in [docs/DECISION_LOG.md](docs/DECISION_LOG.md).

_Updated: 2026-09-11 (post-session reconciliation: TODO executed end-to-end, full-code-review v1.0.x scope landed — 12 findings fixed, 9 recorded below; completed items moved to CHANGELOG)._

---

## High Priority

- [ ] **Push `master`** — the v1.1.0-destined work (deprecated-API removals, race fixes, CSRF trust model, composition API, ~40 test/fuzz/bench targets) is committed locally; `v1.0.0`/`v1.0.1` tags are already pushed. Then per [docs/RELEASE.md](docs/RELEASE.md): verify pkg.go.dev + `go get` after the next tag.
- [ ] **Re-dispatch `nightly-fuzz.yml` and verify step 1 passes** — all 25 patterns are now anchored (`^Name$`) and smoke-verified to match exactly one target, but the in-session dispatch ran the OLD committed workflow and failed as predicted. Issues #7/#8 (false-positive auto-filings from the old pattern collision) are closed. After the push, `gh workflow run nightly-fuzz.yml --repo LarsArtmann/httputil`, confirm the anchor fix, and close any new false-positive issue it files.
- [ ] **Cut `v1.1.0`** — BLOCKED on owner answers (asked in docs/status/2026-09-11_09-05_todo-execution-v1-code-review-session.md): ① push master first vs tag in the same motion; ② re-run the two LLM-rate-limit-lost review passes or accept direct-authorship coverage; ③ CSRF security-degrading config: log-only final or remediate-to-secure-defaults (see Medium #2 below). Tags are permanent — never cut without an explicit owner go.

## Medium Priority

- [ ] **CSRF security-degrading config: log-only vs remediate-to-secure-defaults** — `Validate()`-and-log continues on dangerous combos (e.g. `SameSite=None` + `Secure=false`) instead of remediating to secure defaults; changing the validate-and-log contract overturns a documented decision (DECISION_LOG 2026-08-08) and needs its own design pass. Owner decision pending (question ③ above). Source: full-code-review finding 2.
- [ ] **Unify the two `TrustedOrigins` parsers** — nosurf's `StaticOrigins` vs `parseTrustedOriginURLs` can disagree on validity (entries one parser rejects); divergence window is narrow. Unify with tests in a focused change. Source: full-code-review finding 3.
- [ ] **Reclassify stdlib error registrations** — `http.ErrNoCookie`/`ErrNoLocation` are registered Transient; deterministic failures belong in Rejection by the package's own taxonomy. Also fix `ErrCodeHijackFailed` (doc says Transient, WayOut template reads Infrastructure). Reclassification changes consumer retry decisions — ships with its own changelog entry, not ride-along. Source: full-code-review finding 4.
- [ ] **Document `ValidateCSRF` caller-request mutation** — it mutates the request (header translation, form parse) without documenting it; "already validated" is inferred from token presence. Doc pass + consider a context flag. Source: full-code-review finding 5.
- [ ] **`nosurf.StaticOrigins` failure is fail-open (origin function silently unset)** — configs that fail to parse get different validation semantics than configured; only slog distinguishes. Needs the same fail-closed trust-model treatment as the TrustedProxies work. Source: full-code-review finding 6.
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

---

_Long-term vision and raw ideas live in [ROADMAP.md](ROADMAP.md). Completed work is recorded in [CHANGELOG.md](CHANGELOG.md). Historical open-item evidence lives in the inline `done at` markers of `docs/status/2026-*.md`; this list is the live backlog._
