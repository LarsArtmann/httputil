# Status Report — Full-Code-Review Findings 3–6 Execution Session

_Generated: 2026-09-14 17:14 CEST · Scope: this session only (TODO_LIST Medium-priority execution: findings 3, 4, 5, 6 from the 2026-09-11 full-code review)._
_Tree state at writing: clean; all changes picked up by the auto-commit daemon (`d778631..2eccd3d`). No manual commit made (harness contract: no commits without explicit user request)._

---

## Session in one paragraph

Executed the four executable Medium-priority TODO items end-to-end: (1) unified the two `TrustedOrigins` parsers with a fail-closed trust-model treatment, (2) reclassified the stdlib error registrations, (3) documented `ValidateCSRF`'s caller-request mutations, (4) hardened the `nosurf.StaticOrigins` fail-open path. All quality gates green. Docs updated (CHANGELOG, TODO_LIST, AGENTS.md, DECISION_LOG, FEATURES, README). Two self-created warts found during the post-session self-review (documented below in d/e) — neither is a code defect.

Quick counts: **4 work items fully done · 5 partially done · 9 not started (owner-gated by their own recorded decisions) · 0 fucked-up code defects · 2 self-created doc/wart items.**

---

## a) FULLY DONE

| #   | Item                                                                                                                                                                                                                                                                                                                                                                                                                                                                                          | Evidence                                                                                                                                 |
| --- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------- |
| A1  | **Research phase** — read `csrf.go` (897 lines), `errors.go`, `code.go`, `errors_templates_test.go`, nosurf v1.2.0 internals (`StaticOrigins`, `sameOrigin`, `checkOrigin/checkReferer`, `addNosurfContext`), go-error-family v0.10.0 API (`WithContextAny` nil→`""`, template/context semantics), and the TrustedProxies precedent (`withParsedTrustedProxies`)                                                                                                                              | Session log; every design decision below traces to a verified source                                                                     |
| A2  | **TrustedOrigins parsers unified + fail-closed** (review finding 3): `parseTrustedOriginURLs` is all-or-nothing (any `url.Parse` failure → nil), structurally mirroring `nosurf.StaticOrigins` wholesale failure; the attestation check can never trust a different set than the nosurf handler enforces                                                                                                                                                                                      | `csrf.go:660` (parser), `csrf.go:349` (ConfigureNosurfHandler fail-closed + louder fallback log)                                         |
| A3  | **New `Validate` gate for origin shape** (also finding 3): entries that are not usable `scheme://host` origins (unparseable / missing scheme / missing host) are rejected loudly with the new `csrf.trusted_origin_invalid` code (Rejection, cause-chained to `ErrCSRFConfig`, `{origin}` + `{parse_error}` context)                                                                                                                                                                          | `csrf.go:288` `validateTrustedOriginEntry` (100% cov), `csrf.go:113` sentinel, `errors.go` template, `errors_templates_test.go` registry |
| A4  | **Fail-closed StaticOrigins treatment** (review finding 6): mirrors `withParsedTrustedProxies` — a config with any unparseable entry gets no trusted origins on either side; construction stays log-only per the validate-and-log contract (2026-08-08 decision untouched)                                                                                                                                                                                                                    | `csrf.go:349`; pinned by `TestCSRFMiddleware_UnparseableTrustedOriginFailsClosed` (fails on the old code — error identity flips)         |
| A5  | **Stdlib error reclassification** (review finding 4): `http.ErrNoCookie`/`http.ErrNoLocation` registered **Rejection** (deterministic absence, retry cannot succeed); `ErrAbortHandler` stays Transient; unsupported/skip sentinels stay Infrastructure — all five now pinned by the extended test                                                                                                                                                                                            | `errors.go:420-432`, `errors_test.go:219` (1 assertion → 5)                                                                              |
| A6  | **`ErrCodeHijackFailed` doc/template alignment** (finding 4, part 2): family **stays Transient** per the documented taxonomy (AGENTS.md Error Model + 3 doc tables all say Transient; the review's "WayOut reads Infrastructure" claim did not verify against the tree). The real mismatch — the Fix sentence implying deterministic failure under a retryable WayOut — was reworded; doc comment expanded with the retry rationale                                                           | `errors.go:22-26`, `errors.go:73-78`                                                                                                     |
| A7  | **ValidateCSRF caller-request mutation documented** (review finding 5): doc block lists every in-place mutation (Sec-Fetch-Site bypass fill, X-Csrf-Token translation incl. form parse), nosurf isolation (verified: `addNosurfContext` copies), "already validated" inference, idempotency                                                                                                                                                                                                   | `csrf.go:853` doc block                                                                                                                  |
| A8  | **Tests**: 6 new (NotParseable / MissingScheme / MissingHost / WellFormedAccepted / ParseTrustedOriginURLs_AllOrNothing / Middleware fail-closed) + extended classification test; no table-driven tests (project convention), all `t.Parallel()`, name=claim convention held                                                                                                                                                                                                                  | `csrf_test.go:452-551`, `errors_test.go:219`                                                                                             |
| A9  | **Quality gates**: `go build`, `go test -race -count=1 ./...` (all 3 modules), `go test -race -count=10` on all CSRF/classification tests (no timing races), `golangci-lint run` 0 issues (one `varnamelen` hit on `u` fixed by rename), erraudit real gates green (`legacy_as`, `stdlib_constructor --enforce-go-error-family`), `nix fmt`, `golangci-lint fmt`                                                                                                                              | Gate transcript in session; advisory set = documented baseline + 1 same-class sentinel (see B3)                                          |
| A10 | **Docs updated**: CHANGELOG `[Unreleased]` (3 entries: reclassification+template alignment, TrustedOrigins unification+fail-closed, ValidateCSRF doc-only); TODO_LIST 4 items struck with evidence + header re-dated; AGENTS.md new "all-or-nothing on both sides" bullet; DECISION_LOG 2 rows (all-or-nothing parsing rationale; context-flag decline); README TrustedOrigins field-table note + `csrf.trusted_origin_invalid` classification row; architecture-reference classification row | File diffs `d778631..2eccd3d`                                                                                                            |
| A11 | **Coverage re-measured per the documented methodology** (race profiles, per-module): httputil 97.2%, httpspec 98.6%, combined 97.5% (was 97.0/98.6/97.3 at release). `ConfigureNosurfHandler` 93.8→100%, `parseTrustedOriginURLs` 85.7→100%, `contradictedAttestationOrigin` 93.8→100%, new helper 100%                                                                                                                                                                                       | FEATURES.md coverage section rewritten incl. corrected line numbers                                                                      |
| A12 | **Docs-honesty fix found during coverage pass**: `forwardedProtoFromTrustedProxy` (75.0%) had no FEATURES row — "a gap without a documented reason is a bug in the docs" — added with precise uncovered-branch reasons                                                                                                                                                                                                                                                                        | FEATURES.md CSRF gap section                                                                                                             |

## b) PARTIALLY DONE

| #  | Item                                      | Done                                                                                                                                                                | Missing                                                                                                                                                                                                                                                              |
| -- | ----------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| B1 | **Fail-closed verification, nosurf side** | Attestation-check side pinned end-to-end by test; `ConfigureNosurfHandler` error branch at 100% coverage                                                            | No explicit end-to-end test that a **cross-origin request with a valid token and no attestation header** is rejected by nosurf when the list contains an unparseable entry (the "broken list ⇒ same-origin only" contract is asserted only via the attestation path) |
| B2 | **CSRF trust-model docs**                 | AGENTS.md bullet, README field table + classification row, DECISION_LOG rows                                                                                        | FEATURES.md feature-inventory bullet (line ~98 "Domain-level TrustedOrigins…") and DOMAIN_LANGUAGE.md line 368 still describe the old semantics without the all-or-nothing/shape rule                                                                                |
| B3 | **erraudit advisory baseline**            | Verified the new `errCSRFInvalidOrigin` sentinel lands in the same documented load-bearing class (`sentinel_concrete_type`), advisory count 43 → 44, no new classes | AGENTS.md erraudit note still says "43 `sentinel_concrete_type`" — stale number I created minutes after editing that same file                                                                                                                                       |
| B4 | **Frozen-API diligence**                  | Confirmed no new exported identifiers (code + sentinel are unexported; only behavior of exported `Validate()` changed)                                              | Did not cross-check `docs/v1-stability.md` for a behavioral-compatibility section that should note the `Validate()` tightening                                                                                                                                       |
| B5 | **Canonical gate coverage**               | All documented Go gates run manually and green; `nix fmt` ran (treefmt clean)                                                                                       | `nix flake check` (treefmt verification, server-timing standalone build) and a full `buildflow --build-mode dev` run were **not** executed — manual gates only                                                                                                       |

## c) NOT STARTED

All deliberately untouched — each is blocked by its own recorded owner decision, precondition, or is a separate skill-sized effort:

1. **CSRF security-degrading config: log-only vs remediate** (standing owner question ③; would overturn DECISION_LOG 2026-08-08).
2. **Extract response compression into `go-compression`** — deferred post-v1.1; preconditions: plan-inventory refresh (mine to do eventually), owner new-repo/remote setup, phased execution.
3. **Re-run `architecture-review` post-v1.1** — separate large analysis with HTML deliverable; not attempted to avoid a rushed artifact in this session.
~~4. **Re-run the two review passes lost to LLM rate limits** — parked on owner question ② (scope).~~ done 2026-09-15: both passes re-run for real; 3 findings found and fixed (docs/status/2026-09-15_06-14 a2–a6, commit `0cb25ea`).
5. **httpspec docs-site page** — blocked on the website-launch effort.
6. **File the verified go-error-family issue** — TODO marks it "Owner action (own repo)"; draft is saved and verified.
7. **Decide `CSRFConfig.withParsedTrustedProxies` export** — revisit only on external consumer need.
8. **`ValidateCSRF` returns `*httptest.ResponseRecorder`** — frozen v1.0 symbol, v2.0 material.
9. **`MiddlewareFunc` vs `Middleware` alias split / typed `MiddlewareStack` names / pool-contract hardening** — recorded post-v1.1/v2.0 items.

## d) TOTALLY FUCKED UP

**No code-level damage.** Gates were green at every step; no races (count=10 verified); lint 0; no revertible mishaps; nothing deleted; the auto-daemon batching (5 heuristic commits) is pre-existing behavior, not session damage. What _is_ genuinely bad, said without varnish:

1. **I created doc drift in the very file I was editing.** AGENTS.md erraudit note says 43 advisories; my new sentinel made it 44 and I left the number stale (B3). A freshness rule violated within the same session that re-dated three other docs.
2. **API inconsistency in my own new code:** `WithContextAny("parse_error", problem)` passes plain strings through the `any`-typed setter while the sibling line correctly uses `WithContext("origin", …)`. Harmless (nil renders as `""`, both fill the template) but it is exactly the kind of wart the codebase's own standards say not to ship.
3. **The review's premise for finding 4 (part 2) was wrong and I initially couldn't tell.** Everything in the tree said Transient (doc comment, `WrapTransient` construction, WayOut, three doc tables). I resolved it by fixing the actual inconsistency (Fix wording) and documented the divergence from the finding — but I burned verification effort because the finding text was trusted before it was checked.

## e) WHAT WE SHOULD IMPROVE

Honest answers to the three questions asked, plus the self-review:

**What did I forget?**

- The AGENTS.md erraudit count update (B3).
- Fuzz seeds for the new rejection classes: `FuzzCSRFConfig_TrustedOrigins` still seeds only `""`, `"*"`, wildcard shapes, and a newline pair — scheme-less (`example.com`) and host-less (`https://`) entries are not in the corpus, so the new Validate branches are fuzz-explored but not seed-pinned.
- The nosurf-side end-to-end fail-closed test (B1).
- FEATURES/DOMAIN_LANGUAGE semantics lines (B2).

**What could I have done better?**

- **Two-parsers-by-design should be one parser with an explicit mode.** `validateTrustedOriginEntry` (strict: parse + shape, for Validate) and `parseTrustedOriginURLs` (parse-only, mirroring nosurf) are deliberate, but a reader will ask why two exist. A single `parseTrustedOrigin(orig string, requireShape bool) (*url.URL, error)` (or a shared core returning `(url.URL, error)` with shape checked by the caller) would make the "runtime must mirror nosurf exactly, Validate may be stricter" constraint structural instead of comment-documented. Small split-brain-by-intent; documented in DECISION_LOG but consolidatable.
- **Run the canonical gates, not just their manual equivalents.** `buildflow --build-mode dev` and `nix flake check` exist precisely so I don't re-derive the gate list by hand; I did the work twice and still skipped two gates (B5).
- **Check the doc-freshness blast radius of my own change.** Adding an advisory class changes a number documented in AGENTS.md; grep for the old number before declaring docs done.
- **Push back on the finding earlier.** Finding 4's "WayOut template reads Infrastructure" was checkable in one grep; I should have flagged the discrepancy to the owner instead of silently reinterpreting it (I did document the reinterpretation, but a one-line note in the report/TODO would have made the review trail cleaner).

**What could I still improve?**

- Replace `WithContextAny` string passes with `WithContext` (A-fix).
- Add the missing end-to-end broken-list test + fuzz seeds.
- Consider surfacing the parse failure programmatically (see question 3 below).
- Verify `v1-stability.md` mentions the `Validate()` behavioral tightening (additive, log-only, but behavior).
- Institutionalize: when a session adds a sentinel/code, grep the docs for the advisory count and the code-registry lists in the same change (the AGENTS.md "add to map, list, domain test" rule caught the Go side; the doc side has no equivalent checklist).

**Other self-review answers (brutal pass):**

- _Did I lie?_ No; one imprecision fixed in passing: I first wrote "6 new tests" for the unification — correct — but the fail-closed test's nosurf-side reach is narrower than the name suggests (B1). The report above states it precisely.
- _Ghost systems?_ None. Everything written is wired: new code → Validate → middleware behavior → tests → template → registry → docs.
- _Scope creep?_ Borderline calls made consciously: the scheme/host shape rule goes slightly beyond "unify the parsers" (it is the loud half of the fail-closed treatment); the case-sensitivity divergence between the attestation check (EqualFold, documented) and nosurf's exact Host compare was left alone — matching, not validity, and pre-documented.
- _Useful thing removed?_ The skip-invalid-keep-rest parsing — yes, removed deliberately; its only "useful" behavior (partial trust for broken configs) was the bug.
- _Testing state?_ Suite green incl. race-repeat; the gap class to watch remains nosurf-internal branches reachable only via fixtures we deliberately don't build (TLS, real TCP upgrades) — documented, not chased.

---

## f) Up to 50 things to get done next (brainstorm, impact-sorted within groups — NOT a commitment list)

**Direct session follow-ups (small, this week):**

1. ~~Update AGENTS.md erraudit note 43 → 44 (or reword to the class-based phrasing so the number stops rotting).~~ done 2026-09-14 (class-based rewording; measured: 44 sentinel_concrete_type + 40 test-side errors.Is advisories, both documented).
2. ~~Add end-to-end test: broken `TrustedOrigins` list ⇒ cross-origin request with valid token rejected by nosurf (same-origin-only fallback).~~ done 2026-09-14 (`TestCSRFMiddleware_UnparseableTrustedOriginFallsBackToSameOriginOnly` pinning `ErrCSRFInvalid`, plus positive contrast `TestCSRFMiddleware_AllowsTrustedOriginWithoutAttestationHeader`).
3. ~~Add fuzz seeds `"example.com"` and `"https://"` to `FuzzCSRFConfig_TrustedOrigins`.~~ done 2026-09-14 (both seeds added and the fuzz oracle extended to pin the scheme://host shape contract; 10s run green).
4. ~~Replace the two `WithContextAny("parse_error", …)` string passes with `WithContext`.~~ done 2026-09-14 (the error-value site legitimately stays `WithContextAny`).
5. ~~Consolidate the two TrustedOrigins parse helpers behind one core (`requireShape` flag or caller-side shape check).~~ done 2026-09-14 (single parse point `parseTrustedOrigin`, mirroring nosurf, used by both helpers; no boolean flag).
6. ~~Update FEATURES.md feature bullet + DOMAIN_LANGUAGE.md line 368 to the all-or-nothing/shape semantics.~~ done 2026-09-14.
7. ~~Check `docs/v1-stability.md` for a behavioral-delta note on `Validate()` tightening; add one line if the section exists.~~ done 2026-09-14 (verified: no behavioral-delta section exists — the Versioning Policy is version-scoped; obligation routed to the new TODO_LIST next-release item).
8. ~~Run `nix flake check` once on this tree to close the B5 gap.~~ done 2026-09-14 (all checks passed).
9. ~~Run `buildflow --build-mode dev` to confirm the pipeline agrees with the manual gates.~~ done 2026-09-14 (all fixable steps green; findings gate trips only on the documented policy-rejected residuals — erraudit delta exactly +1 sentinel/−1 ignored from this session's edits, zero findings outside documented classes).
10. ~~Re-run `scripts/coverage-threshold` (the actual gate binary) instead of the manual `go tool cover` total.~~ done 2026-09-14 (gate binary green: total 97.5% ≥ 95%).

**TODO_LIST execution (existing Medium items, in order):**
~~11. Owner-ruling pass on CSRF security-degrading config (question ③) → then either close or run the remediation design pass.~~ done 2026-09-15: design pass + owner ruling + implementation shipped (docs/planning/2026-09-15_csrf-security-degrading-config-design-note.md; DECISION_LOG 2026-09-15).
12. Refresh the go-compression extraction plan inventory (precondition 1 of the deferred item; owner owns preconditions 2–3).
13. Re-run `architecture-review` post-v1.1 (HTML deliverable, docs/architecture-understanding).
~~14. Rule on question ② and re-run the two lost review passes (chain_test, server_test, ratelimit_keyed_test, id_generator; scripts/examples scope).~~ done 2026-09-15 (docs/status/2026-09-15_06-14 a2–a6).
15. httpspec docs-site page (when website-launch effort resumes).

**Docs-health / hygiene:**
16. HARVEST this report's section f into TODO_LIST/ROADMAP with routing rigor (status-report↔docs-health loop is open).
17. Annotate `docs/status/2026-09-11_13-49_*.md` items 14–17 (four items this session executed) with done-at hashes per the ANNOTATE convention.
18. Sweep all living docs for the phrase "TrustedOrigins" to catch the last semantic stragglers (README prose section beyond the field table).
19. ~~Add the doc-side checklist analog of the "map, list, domain test" rule: "when adding a sentinel/code → grep AGENTS.md advisory count + FEATURES inventory + v1-stability" (candidate AGENTS.md line).~~ done 2026-09-14 (appended to the AGENTS.md Message-templates bullet).
20. Consider renaming/rewording the FEATURES "New middleware" coverage header (it predates the taxonomy rename and mixes sections).

**Testing depth (from self-review):**
21. Property test: `parseTrustedOriginURLs` output set ≡ effective accept-set of `nosurf.StaticOrigins` for the same input (oracle-style unification pin).
22. Test: scheme-less TrustedOrigins entry through `CSRFMiddleware` end-to-end (Validate logs, construction proceeds, entry never grants trust) — pins the log-only contract.
23. Test: `ValidateCSRF` re-invocation on the same request is idempotent (doc claim at `csrf.go:853` currently has no pin).
24. Test: `ValidateCSRF` translation branch (`needsTranslation`) exercised through `ValidateCSRF` itself (lifts the 94.4% row's reason).
25. Test: `forwardedProtoFromTrustedProxy` untrusted-remote / empty-header / non-http(s)-proto branches (75% → higher) with a configured trusted proxy.
26. Test: `requestScheme` TLS branch via a TLS test fixture (80% row).
27. Seed-based fuzz: HX-headers token-less branch (71.4% row) — trivial seed addition.
28. Consider `errors.Is(err, ErrCSRFConfig)` chain assertions for every new CSRF Validate branch (two of six tests assert it; complete the set).

**API/API-doc polish:**
29. Decide whether `csrf.trusted_origin_invalid` should be exposed as an exported `Code` constant for consumer error routing (additive; see question 3).
30. ~~Error-template pass on the new entry: confirm `{parse_error}` renders readably for all three problem kinds (parse error text / "missing scheme" / "missing host").~~ done 2026-09-14 (verified by template inspection: all three render readably inline after the colon).
31. ~~`errors.go:10-12` header comment still says "(Transient vs Infrastructure)" — stale family dichotomy; one-line refresh.~~ done 2026-09-14 (now lists all four families).
32. ~~`errCSRFInvalidOrigin` message says "not a scheme://host origin" while the template What says "not a usable origin" — unify wording.~~ done 2026-09-14 (template What unified on "scheme://host").

**Pre-existing small debts noticed this session (not mine, but real):**
33. `Validate()` complexity is near the cyclop ceiling after the new gate — the helper helped; next addition should extract the TrustedProxies loop too.
34. `requestScheme`'s 66.7→80% note in old FEATURES had drifted from measurement — signal that manual percentage edits rot; prefer regenerating the table.
35. `wildcard-host` entries (`https://*`) pass shape validation — dead config, not a bypass, but worth a Validate note or rejection in a future design pass.
36. `parseTrustedOriginURLs` returning `nil` vs empty slice is semantically loaded (nil = fail-closed, empty = no entries) — document or type-state it.
37. The 5 heuristic daemon commits from this session make `git log` co-location analysis useless for these changes — consider a session-summary commit convention (already noted in AGENTS.md as unreliable; no action taken).
38. `nosurf.StaticOrigins` accepts scheme-less URLs silently (upstream behavior) — candidate upstream issue per `verify-before-filing` (their doc says "expects scheme://host" but the code doesn't enforce it).
39. `Get-vs-Set` canonicalheader footgun doc could gain the new `Sec-Fetch-Site` constants as canonical examples.
40. Benchmarks: no CSRF middleware benchmark row exists (attestation check is on the hot path for unsafe methods) — candidate `BenchmarkCSRFMiddleware*`.

**Bigger arcs (pre-existing backlog, restated for completeness):**
41. Post-v1.1: `ValidateCSRF` result-type redesign (v2.0).
42. Post-v1.1: `MiddlewareFunc`-canonical deprecation discussion (v2.0).
43. Post-v1.1: typed `MiddlewareStack` names (additive candidate).
44. Post-v1.1: pool-contract hardening (`(nil, nil)` probe, acquire provenance).
45. Owner: file the verified go-error-family conditional-request classification issue.
46. Owner: go-compression new-repo/remote setup (precondition 2).
47. Website-launch effort → unblocks httpspec docs page.
48. ~~Next release: the `[Unreleased]` section now carries two behavior changes (reclassification, TrustedOrigins) — both need the migration-note treatment in the next version's notes.~~ done 2026-09-14 (routed: TODO_LIST now carries the next-release item covering both behavior changes, a migration callout, and the v1-stability Versioning-Policy bullet).
49. Consider a `docs/migrating-*.md` note for consumers who relied on partial TrustedOrigins parsing (narrow but real behavior change).
50. Next docs-health VERIFY pass should re-check today's coverage numbers (97.2/98.6/97.5) against a fresh race-profile run.

---

## g) Questions I cannot figure out myself

1. **Question ③ (standing): CSRF security-degrading config — should `Validate()`-and-log stay, or do you want a remediate-to-secure-defaults design pass?** My session touched the adjacent code (TrustedOrigins) and preserved log-only; the SameSite/Secure class is still open and it decides whether I plan a behavior-changing pass or close the TODO item. ~~Answered 2026-09-15: remediate — owner ruled "good defaults + config options + power to the applications"; B1 fallback (`Secure=true`) + `AllowInsecureSameSiteNone` opt-out shipped (DECISION_LOG 2026-09-15).~~
2. **Question ② (standing): should I re-run the two LLM-rate-limit-lost review passes now** (chain_test.go, server_test.go, ratelimit_keyed_test.go, id_generator tests; scripts/examples scope), or does that stay parked?
3. **Export policy for the new validity rule:** `csrf.trusted_origin_invalid` is currently internal-only (unexported `Code` + sentinel). Consumers who call `Validate()` themselves can match it only via the string. Do you want it exported (additive, frozen-API-safe), or is internal-only the policy for new CSRF codes until v2?

---

_Report ends. Per the status-report contract: written to `docs/status/` as the point-in-time snapshot; the auto-commit daemon will pick it up (no manual commit — harness rule). Section f is the HARVEST input for TODO_LIST/ROADMAP — awaiting instructions._
