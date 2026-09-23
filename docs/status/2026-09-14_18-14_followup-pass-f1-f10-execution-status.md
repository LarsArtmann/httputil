# Status Report — Follow-Up Pass (f1–f10) Execution Session

_Generated: 2026-09-14 18:14 CEST · Scope: this continuation session only (execution of the 10 direct follow-ups from the 2026-09-14 17:14 report, plus the small polish items 19/30/31/32/48 that were executed alongside)._
_Tree state at writing: all work committed by the auto-commit daemon (`39a8516`, `73aa190`, `83cb712`, `f4e3185`); only this report's annotations are uncommitted (daemon will pick them up). No manual commit made (harness contract)._

---

## Session in one paragraph

Executed all ten direct follow-ups from the 17:14 report end-to-end, verified each with gates, and closed five small polish items (doc-side checklist rule, template-render verification, stale `errors.go` header comment, message/template wording unification, migration-note routing). The TrustedOrigins fail-closed treatment now has its missing end-to-end pins (negative nosurf-path test + positive contrast test), the fuzz target pins the new shape contract (not just seeds), the two parse helpers share a single documented parse point, and the canonical gates (`nix flake check`, `buildflow --build-mode dev`, `scripts/coverage-threshold`) were run for the first time this session — closing the B5 gap. All quality gates green. Two self-created process warts were found and fixed within the session (d1, d2); neither is a code defect.

Quick counts: **10 follow-ups fully done · 5 polish items done alongside · 0 not-applicable surprises · 0 fucked-up code defects · 2 self-created process warts (both self-caught and fixed same session).**

---

## a) FULLY DONE

| #   | Item                                                                                                                                                                                                                                                                                                                                                                                                                                                                   | Evidence                                                                                                    |
| --- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------- |
| A1  | **f1 — erraudit advisory note fixed, class-based** (AGENTS.md): the rotting "43" replaced with measured class counts: 44 `sentinel_concrete_type` (load-bearing sentinels, +1 per added sentinel) and 40 test-side `errors.Is` advisories, measured 2026-09-14 via the full `--type-aware` run                                                                                                                                                                         | AGENTS.md Commands block; measurement transcript in session                                                 |
| A2  | **f2 — end-to-end fail-closed test added**: `TestCSRFMiddleware_UnparseableTrustedOriginFallsBackToSameOriginOnly` — broken list + cross-origin POST with _valid token_, no attestation header ⇒ 403 with `ErrCSRFInvalid`, proving the _nosurf token path_ (not the attestation check) enforces the same-origin-only fallback                                                                                                                                         | `csrf_test.go` (after `UnparseableTrustedOriginFailsClosed`); fails on pre-unification code by construction |
| A3  | **f2+ — positive contrast test added** (beyond the ask): `TestCSRFMiddleware_AllowsTrustedOriginWithoutAttestationHeader` — clean list, same request shape ⇒ 200. Without it, the negative test cannot distinguish "broken list rejects" from "Origin without attestation always rejects"                                                                                                                                                                              | `csrf_test.go`; both green under `-race -count=10`                                                          |
| A4  | **f3 — fuzz seeds + oracle strengthening**: `"example.com"` and `"https://"` seeds added, AND the `FuzzCSRFConfig_TrustedOrigins` oracle extended to pin the shape contract itself (any entry that fails `url.Parse` or lacks scheme/host ⇒ `Validate` must reject). 10-second smoke run: >1.6M execs, PASS                                                                                                                                                            | `csrf_fuzz_test.go`; fuzz transcript                                                                        |
| A5  | **f4 — `WithContext` consistency restored**: both `WithContextAny("parse_error", …)` _string_ passes now use the string-typed `WithContext` (verified against go-error-family v0.10.0 signatures). The third site (a real `error` value at the CIDR gate) legitimately stays `WithContextAny` — documented in the report annotation                                                                                                                                    | `csrf.go` `validateTrustedOriginEntry`                                                                      |
| A6  | **f5 — single parse point**: new `parseTrustedOrigin(origin)` — the one place a TrustedOrigins string becomes a URL, mirroring `nosurf.StaticOrigins` exactly; both `validateTrustedOriginEntry` (Validate gate) and `parseTrustedOriginURLs` (runtime allowlist) call it. No boolean control flag; the "runtime mirrors nosurf, Validate may be stricter" constraint is now structural, not comment-documented                                                        | `csrf.go:287`; wrapcheck passthrough directive with explanation (3 existing precedents in server_timing)    |
| A7  | **f6 — semantic stragglers updated**: FEATURES.md feature bullet and DOMAIN_LANGUAGE.md CSRF rule now state the `scheme://host` shape rule and all-or-nothing/fail-closed semantics                                                                                                                                                                                                                                                                                    | FEATURES.md (Middleware/CSRF section), DOMAIN_LANGUAGE.md (CSRF Protection Rules)                           |
| A8  | **f7 — v1-stability verified, obligation routed**: confirmed the doc has NO behavioral-delta section — only version-scoped Versioning-Policy bullets. Adding an "unreleased" note would rot, so the obligation (migration callout + Versioning-Policy bullet at next release) was routed to a new TODO_LIST Low-priority item covering both `[Unreleased]` behavior changes                                                                                            | docs/v1-stability.md (read, unchanged), TODO_LIST.md new item                                               |
| A9  | **f8 — `nix flake check` green**: all checks passed (treefmt verification + server-timing GOWORK=off build); only the standard incompatible-systems warning (aarch64/x86_64-darwin omitted)                                                                                                                                                                                                                                                                            | Gate transcript                                                                                             |
| A10 | **f9 — `buildflow --build-mode dev` run**: all fixable steps green; markdown format step reflowed three doc files (committed by daemon). Findings gate trips at 141 error-severity findings — verified these are exactly the documented policy-rejected residual classes (see B3) and that this session's delta is precisely +1 sentinel / −1 ignored discard (my fuzz-oracle edit removed the `_ = err` the tool had been counting)                                   | Buildflow summary + JSON finding dump; erraudit class counts 44+10=54 confirmed                             |
| A11 | **f10 — coverage gate via the real binary**: `go tool cover -func \| go run ./scripts/coverage-threshold 95` ⇒ total 97.5% ≥ 95%, green (library packages only, scripts excluded — matching prerelease step 7)                                                                                                                                                                                                                                                         | Gate transcript                                                                                             |
| A12 | **Polish items 19/30/31/32 closed**: doc-side checklist rule appended to AGENTS.md Message-templates bullet (sentinel/code ⇒ sweep erraudit note, FEATURES, v1-stability, classification tables); `{parse_error}` template rendering verified readable for all three problem kinds; `errors.go` header comment now names all four families (was the stale "Transient vs Infrastructure" dichotomy); template What unified on "scheme://host" with the sentinel message | AGENTS.md, errors.go                                                                                        |
| A13 | **Full gate re-run after every change**: `go build`, `go test -race -count=1` (all 3 modules), `go test -race -count=10` (all CSRF/parse/classification tests), `golangci-lint fmt` + `run` (0 issues), both erraudit real gates (`legacy_as`, `stdlib_constructor --enforce-go-error-family`) exit 0, server_timing sub-module race tests + lint 0 issues, `nix fmt` clean                                                                                            | Gate transcripts; final tree `nix fmt`: 0 changed                                                           |
| A14 | **Previous report annotated**: 15 items struck with `done 2026-09-14 (...)` evidence per the ANNOTATE convention (items 1–10, 19, 30, 31, 32, 48 of the 17:14 report)                                                                                                                                                                                                                                                                                                  | docs/status/2026-09-14_17-14_*.md                                                                           |

## b) PARTIALLY DONE

| #      | Item                                                                                                                                      | Done                                                                                                                                                                                                                                                                                                      | Missing                                                                                                                                                                                                                                                                                                                      |
| ------ | ----------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| B1     | **buildflow findings-gate baseline**                                                                                                      | Verified the 141 findings decompose into the documented policy-rejected classes (erraudit 44 sentinel + 10 ignored; branching-flow `PHANTOM_TYPE` etc.; go-structure-linter flat-root-package) and my delta is fully accounted for (+1 sentinel from the new code, −1 ignored from the removed `_ = err`) | Never established that the gate tripped identically _before_ this session — I inferred steady state from class composition, not from a baseline run (e.g., worktree at the pre-session commit). Note: buildflow also reported 9 tools unavailable (health check) — not drilled into per the no-unrelated-research constraint |
| ~~B2~~ | ~~**Coverage numbers**~~ done (docs-health pass 2026-09-23, 97.5% httputil / 99.1% httpspec re-measured; FEATURES refreshed)              | ~~The gate binary confirms 97.5% total (threshold gate, scripts excluded)~~                                                                                                                                                                                                                               | ~~Per-module race-profile numbers (97.2/98.6 in FEATURES) were NOT re-measured after the follow-up pass; the new statements (~2, all covered) make a tick unlikely, but the documented methodology is per-module race profiles, and I didn't re-run them~~                                                                   |
| ~~B3~~ | ~~**buildflow run hygiene**~~ done — AGENTS.md BuildFlow section documents the expected-non-zero findings-gate stance                     | ~~All fixable steps green; format verified~~                                                                                                                                                                                                                                                              | ~~The findings-gate exit-1 IS the documented steady state (AGENTS.md: residuals are policy-rejected), but AGENTS.md does not explicitly say "buildflow dev is expected to exit non-zero on the findings gate" — a future reader could misread a tripped gate as breakage. Candidate one-line doc note (f-list item)~~        |
| ~~B4~~ | ~~**HARVEST of section f**~~ done (docs-health pass 2026-09-23, harvest executed — the 2026-09-15 and 2026-09-23 passes routed the opens) | ~~Specific routing done (item 48/49 ⇒ TODO_LIST item; item 19 ⇒ AGENTS.md)~~                                                                                                                                                                                                                              | ~~The full docs-health HARVEST pass over the remaining f-items into TODO_LIST/ROADMAP was not run (skill-sized effort, owner-gated by standing instructions)~~                                                                                                                                                               |

## c) NOT STARTED

All deliberately untouched — owner-gated by their own recorded decisions or separate skill-sized efforts:

~~1. **CSRF security-degrading config: log-only vs remediate design pass** — standing owner question ③; would overturn DECISION_LOG 2026-08-08.~~ done (resolved 2026-09-15 — owner ruling; design pass + B1 fallback shipped in v1.2.0)
~~2. **Re-run the two LLM-rate-limit-lost review passes** (chain_test, server_test, ratelimit_keyed_test, id_generator tests; scripts/examples scope) — owner question ②.~~ done 2026-09-15: both passes re-run for real; 3 findings found and fixed (docs/status/2026-09-15_06-14 a2–a6, commit `0cb25ea`).
2. **Export `csrf.trusted_origin_invalid` as an exported `Code` constant** — owner question ③-adjacent (new this session's standing set; consumers can currently match only via the string).
3. **go-compression extraction** — deferred post-v1.1; owner owns new-repo/remote setup; my precondition (plan inventory refresh) still open.
4. **Re-run `architecture-review` post-v1.1** — HTML deliverable, separate session.
5. **httpspec docs-site page** — blocked on website-launch effort.
~~6. **File the verified go-error-family issue** — owner action (own repo).~~ done (filed 2026-09-15 as go-error-family#5; verified open; TODO carries the follow-through)
7. **`ValidateCSRF` result type / `MiddlewareFunc` split / typed stack names / pool-contract hardening** — post-v1.1/v2.0 material, recorded.
~~8. **Previous report's docs-health items 16–18, 20** (full HARVEST, annotate the 2026-09-11 report, README-prose TrustedOrigins sweep, FEATURES header rename) and testing-depth items 21–29 — not attempted this session (follow-up pass scope only).~~ done (items 16–18 resolved 2026-09-15; item 20 FEATURES header rename done 2026-09-23; 21–29 routed to the TODO CSRF test-depth batch)

## d) TOTALLY FUCKED UP

**No code-level damage.** Gates green at every step; no races; lint 0; nothing deleted; no reverts of others' work. The honest list:

1. **I wrote an unmeasured number into AGENTS.md — again, in the same file, same day.** My first fix of the erraudit note (f1) claimed "one advisory class in tests … 44" — conflating the 44 _sentinel-declaration_ advisories (non-test code) with the test-side `errors.Is` advisories (actually 40). I only caught it because I later measured to reconcile the buildflow erraudit counts, then rewrote the note with both classes properly separated. The previous session's sin was a stale number; mine was an _invented-adjacent_ number. Same root cause: writing documentation numbers from memory instead of from a measurement. The final state is correct and measured — but it took two writes.
2. **The wrapcheck/nolint fix cost two lint cycles because I forgot a project-specific mechanical fact I already knew:** `golangci-lint fmt` runs golines@120, which reflows any >120-char line — including splitting `return url.Parse(origin) //nolint:…` so the directive lands on the orphaned `)` line while wrapcheck reports at the `return`. Result: nolintlint "unused directive" + wrapcheck finding, both self-inflicted. Fixed by shortening the explanation to keep the line one-line. The project AGENTS.md literally documents a same-shaped trap (the gosec `//nolint` fragility note) and I walked into the line-length variant anyway.
3. **One of 15 batch edits to the status report failed on a mis-transcription** (item 48): I copied the wording from the conversation summary instead of from the file (the file's item 48 has different text than the summary's paraphrase). Caught by the tool's per-edit accounting, verified with grep, fixed with the actual file text. Small, but the pattern — trusting a summary over the source — is the same failure class the previous session flagged in the review-finding verification.
4. **Process friction, handled correctly but worth recording:** my first annotation attempt bounced twice on the stale-read guard because buildflow's markdown format step had rewritten the report mid-session (daemon committed it as `83cb712`). I initially could not tell what changed; I verified via `git show`/`git diff` that the f-section content was untouched (pure table reflow) before re-applying. Correct outcome; the cost was two wasted round trips because my first re-read used bash `sed` instead of the View tool (which is what the edit guard tracks).

## e) WHAT WE SHOULD IMPROVE

**What did I forget?**

- Nothing from the assigned list — all ten follow-ups landed. What I forgot _mid-flight_: (1) that doc numbers must come from measurements, not memory (d1); (2) that golines reflow will break long nolint lines (d2); (3) that the edit guard tracks View reads, not bash reads (d4).

**What could I have done better?**

- **Measure first, write once.** The f1 fix should have been: run the type-aware pass → read the counts → write the note. That order costs 30 seconds and makes the correction cycle impossible.
- **Add the positive contrast test as part of writing the negative one.** I did add it, but only after asking "what would make this test lie?"; the instinct should fire during test authoring, not during review.
- **Copy edit targets from the file, never from a summary** (d3). The stale-read guard and my own grep check caught it, but the failure was avoidable at transcription time.
- **Establish the buildflow baseline before, not after.** Running `buildflow --build-mode dev` on the clean tree _before_ editing would have turned B1's inference into a measurement. Cheap, and it converts "documented steady state (inferred)" into "verified identical".

**What could I still improve?**

- Re-measure per-module race coverage so FEATURES.md's 97.2/98.6 stay methodology-pure (B2).
- Add the AGENTS.md line documenting that a tripped buildflow findings gate is expected steady state for this repo (B3).
- The single-parse-point concept (`parseTrustedOrigin`) deserves a mention in the AGENTS.md "all-or-nothing" bullet so future sessions don't re-derive why three helpers exist (two callers + one core). One line.
- Institutionalize the nolint-line-length rule: "nolint explanations must keep the line under 120 or the directive must go on its own line above" — candidate for AGENTS.md lint-notes.

**Self-review answers (brutal pass):**

- _Did I lie?_ No. One imprecision caught and fixed in-session (d1), disclosed above.
- _Ghost systems?_ None. Everything written is wired: parse point → both call sites → tests → fuzz oracle → docs; checklist rule → AGENTS.md; routing → TODO_LIST.
- _Scope creep?_ Two conscious extensions beyond the literal asks, both defended by the codebase's own standards: the positive contrast test (A3 — a lone negative test proves less than its name) and the fuzz oracle strengthening (A4 — seeds without an assertion only explore, they don't pin). Both are the kind of thing "every change raises the bar" demands; flagging them as extensions rather than hiding them.
- _Useful thing removed?_ Yes, one: the fuzz target's `_ = err` discard — replaced by the real shape-contract assertion. Side effect: the documented erraudit "ignored" count dropped 11 → 10, which I verified against buildflow's measured 54 (44+10) rather than hand-waving.
- _Testing state?_ Suite green including `-race -count=10` on all CSRF/parse/classification tests; fuzz smoke run green; nightly workflow picks up the strengthened oracle automatically (all targets run there).

---

## f) Up to 50 things to get done next (brainstorm, impact-sorted within groups — NOT a commitment list)

**Direct session follow-ups (small, this week):**

1. ~~Re-measure per-module race-profile coverage (97.2/98.6) after this pass; refresh FEATURES.md numbers if they ticked (B2).~~ done (docs-health pass 2026-09-23, 97.5% httputil / 99.1% httpspec re-measured; FEATURES refreshed)
2. ~~Add AGENTS.md line: buildflow `--build-mode dev` findings gate is expected to exit non-zero on the documented policy-rejected residuals (B3).~~ done (AGENTS.md BuildFlow section documents the expected-non-zero findings-gate stance)
3. ~~Mention `parseTrustedOrigin` as the single parse point in the AGENTS.md all-or-nothing bullet (one line).~~ done (present — the AGENTS.md all-or-nothing bullet names parseTrustedOrigin as the single parse point)
4. AGENTS.md lint-note: nolint explanations must keep the line <120 cols (golines will orphan longer directives onto the wrong line).
5. Unit test pinning opaque-URL TrustedOrigins entries (`mailto:addr`, `data:text/plain,x`) — shape gate rejects via `Host == ""`; currently only fuzz-explored, not seed- or test-pinned.
6. Test asserting the fallback log fires: broken TrustedOrigins ⇒ the "falling back to same-origin-only validation" `slog.Error` record (behavior is pinned; the loud-log half of the contract is not asserted).
   ~~7. Owner-question ③ pending: close or plan the CSRF security-degrading-config design pass.~~ done 2026-09-15: design pass executed per the owner ruling; fallback shipped (docs/planning/2026-09-15_csrf-security-degrading-config-design-note.md).
7. ~~Owner-question ② pending: scope ruling on the two lost review passes.~~ done (resolved — question ② answered 2026-09-15; both review passes re-run, 3 findings fixed)
8. Owner question: export policy for `csrf.trusted_origin_invalid` (exported `Code` constant vs internal-until-v2).
9. ~~Full docs-health HARVEST of both 2026-09-14 reports' open f-items into TODO_LIST/ROADMAP (B4).~~ done (docs-health pass 2026-09-23, harvest executed)

**TODO_LIST execution (existing items, in order):**
11. Refresh the go-compression extraction plan inventory (my precondition; owner owns repo/remote setup).
12. Re-run `architecture-review` post-v1.1 (HTML deliverable, docs/architecture-understanding).
13. httpspec docs-site page (when website-launch resumes).
~~14. Annotate `docs/status/2026-09-11_13-49_*.md` items 14–17 with done-at hashes (carried item 17).~~ done 2026-09-15 (docs-health pass: items 14–17 struck in that report).
~~15. Sweep README prose (beyond the field table) for TrustedOrigins semantic stragglers (carried item 18).~~ done 2026-09-15 (docs-health VERIFY: README lines 382/441/577 verified carrying the all-or-nothing/`scheme://host` semantics; no stragglers).

**Docs hygiene:**
16. ~~Consider renaming/rewording the FEATURES "New middleware" coverage header (carried item 20).~~ done (done 2026-09-23 — FEATURES coverage section renamed to Middleware and server internals)
17. Add the buildflow × daemon interplay note: buildflow's markdown format step rewrites committed docs mid-session; the daemon then commits the reflow as heuristic churn (observed twice today: `83cb712`, and the stale-read guard episode).
18. Consider a `git worktree`-based baseline harness note for advisory gates ("run the gate on the pre-change tree before editing") — candidate AGENTS.md cross-cutting lesson.
19. ~~Next docs-health VERIFY pass: re-check the 97.5% gate total and the 44/40 advisory counts against fresh runs (carried item 50, extended to the new counts).~~ done (re-measured 2026-09-15 — 45 sentinels / 41 test-side; AGENTS.md updated)
20. ~~features/architecture-reference: confirm no table drifted after the follow-up edits (both were reflowed by the formatter — verify no semantic mangling).~~ done (verified — the 2026-09-15 docs-health VERIFY swept FEATURES; no semantic mangling from the reflow)

**Testing depth (carried 21–29, restated):**
21. Property test: `parseTrustedOriginURLs` output set ≡ effective accept-set of `nosurf.StaticOrigins` for the same input (oracle-style unification pin — now trivially true by construction via the shared parse point; the pin is still worth having at the allowlist level).
22. Test: scheme-less TrustedOrigins entry end-to-end (Validate logs, construction proceeds, entry never grants trust) — pins the log-only contract for the shape gate specifically.
23. Test: `ValidateCSRF` re-invocation idempotency (doc claim at the `ValidateCSRF` doc block, no pin).
24. Test: `needsTranslation` branch exercised through `ValidateCSRF` itself (lifts the 94.4% row's reason).
25. Test: `forwardedProtoFromTrustedProxy` untrusted-remote / empty-header / non-http(s)-proto branches (75% row) with a configured trusted proxy.
26. Test: `requestScheme` TLS branch via TLS fixture (80% row).
27. Seed-based fuzz: HX-headers token-less branch (71.4% row).
28. `errors.Is(err, ErrCSRFConfig)` chain assertions for the remaining new CSRF Validate branches (complete the set — 4 of 6 assert it).
29. CSRF middleware benchmark row (attestation check is hot-path for unsafe methods; carried item 40).

**API/doc polish (carried):**
30. Decide `csrf.trusted_origin_invalid` export (dup of 9 — listed here for the API-polish track; resolve once).
31. Wildcard-host `https://*` passes shape validation — dead config; Validate note or rejection in a future design pass.
32. `parseTrustedOriginURLs` nil-vs-empty semantics (nil = fail-closed, empty = no entries) — document or type-state.
33. Upstream candidate: `nosurf.StaticOrigins` accepts scheme-less URLs silently (per verify-before-filing, their docs say scheme://host) — carried item 38.
34. canonicalheader doc: add `Sec-Fetch-Site` as a canonical-form example (carried item 39).
35. Benchmarks doc: CSRF section absent from docs/benchmarks.md — pair with item 29.

**Pre-existing small debts (carried, not mine):**
36. `Validate()` complexity near the cyclop ceiling — extract the TrustedProxies loop before the next gate addition.
37. Manual coverage percentages rot (observed: `requestScheme` 66.7→80 note drifted) — prefer regenerating the FEATURES coverage table from `go tool cover`.
38. The 9 buildflow tools unavailable (health check) — worth one `buildflow doctor` pass someday to name them.
39. ~~Session-summary commit convention: 4 more heuristic daemon commits today; `git log` co-location analysis remains unreliable (documented; no action taken).~~ done (documented in AGENTS.md (co-change analysis unreliable); no action by design)
40. ~~`docs/benchmarks.md` and the benchmark protocol: unchanged by this pass, but the new tests add no benchmarks — intentional, noted to prevent "why is CSRF missing" archaeology.~~ done (noted to prevent archaeology; no action intended)

**Bigger arcs (carried for completeness):**
41. Post-v1.1: `ValidateCSRF` result-type redesign (v2.0).
42. Post-v1.1: `MiddlewareFunc`-canonical deprecation discussion (v2.0).
43. Post-v1.1: typed `MiddlewareStack` names (additive candidate).
44. Post-v1.1: pool-contract hardening (`(nil, nil)` probe, acquire provenance).
45. ~~Owner: file the verified go-error-family conditional-request classification issue.~~ done (filed 2026-09-15 as go-error-family#5)
46. Owner: go-compression new-repo/remote setup (precondition 2).
47. Website-launch effort → unblocks httpspec docs page.
48. ~~Next release: execute the routed migration-note item (TODO_LIST Low priority) — covers both `[Unreleased]` behavior changes + v1-stability Versioning-Policy bullet.~~ done at `9b9e032`
49. After next tag: freeze CHANGELOG section; annotate the 2026-09-14 reports fully-resolved items and archive per convention.
50. Consider whether `internal/` extraction trigger (~50 non-test files, post-v1.0) has been approached — count non-test files at the next review.

---

## g) Questions I cannot figure out myself

1. **Question ③ (standing): CSRF security-degrading configs — keep validate-and-log, or do you want a remediate-to-secure-defaults design pass?** This session deliberately preserved log-only (construction proceeds, fail-closed trust); the SameSite/Secure class stays open and decides whether I plan a behavior-changing pass or close the TODO item. ~~Answered 2026-09-15: remediate (owner ruling; B1 + opt-out shipped, DECISION_LOG 2026-09-15).~~
2. **Question ② (standing): should I re-run the two LLM-rate-limit-lost review passes now** (chain_test.go, server_test.go, ratelimit_keyed_test.go, id_generator tests; scripts/examples scope), or does that stay parked? ~~Answered 2026-09-15: re-run executed; 3 findings found and fixed (docs/status/2026-09-15_06-14 a2–a6).~~
3. **Export policy: should `csrf.trusted_origin_invalid` become an exported `Code` constant** (additive, frozen-API-safe; lets consumers route on it without string matching), or is internal-only the policy for new CSRF codes until v2? Today's `WithContext` cleanup touched all its construction sites, so exporting later would be a one-line change — but the decision is yours.

---

_Report ends. Per the status-report contract: written to `docs/status/` as the point-in-time snapshot; the auto-commit daemon will pick it up (no manual commit — harness rule). Section f is the HARVEST input for TODO_LIST/ROADMAP — awaiting instructions._
