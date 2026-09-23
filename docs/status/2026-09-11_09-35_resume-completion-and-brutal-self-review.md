# Status Report — Session Resume Completion + Brutal Self-Review

**Date:** 2026-09-11 09:35 CEST
**Session scope:** Resume of the interrupted TODO-execution + v1.0.x full-code-review session (see [2026-09-11_09-05](2026-09-11_09-05_todo-execution-v1-code-review-session.md)). This session finished every resumable item, found and fixed a real CI-breaking bug, and pushed master.
**Format note:** written as `.md` per the owner's explicit instruction (the status-report skill's canonical format is HTML — flagged override, one-off, not propagated into the skill).

---

## Executive Summary

The resumed session completed 8 of 9 tracked tasks; the ninth (v1.1.0 tag cut) is owner-blocked by design. Master is **pushed and CI-green** (run 34574170086). Along the way the session caught a genuine integration bug — the new `scripts/doc-snippet-refs` tool silently dragged the 95% coverage gate to 93.1%, which had _already failed CI_ at 07:06 — and fixed it in both gate call sites. A brutal self-review found 4 honest mistakes, including one stale reference (`etag_test.go`) missed _inside the very doc-freshness pass being performed_ (fixed on sight, per granted 2026-09-06 permission).

**Gates at time of writing:** build/vet/race/lint/erraudit/coverage-97.3%/changelog/flake all green; nightly-fuzz dispatched, step 1 verified passing, full run in progress (~2 h for all 25 targets).

---

## Self-Review (the brutal part, answered directly)

**What did you forget?**

1. `AGENTS.md:232` still listed the deleted `etag_test.go` in the test-file split inventory — missed during a _dedicated_ AGENTS.md freshness pass. Caught in post-completion self-review, fixed on sight (`csrf_test.go`, `ratelimit_keyed_test.go` now terminate the list).
2. Initially did not check the push-triggered CI run. It turned out green — but the preceding 07:06 run on the pre-fix tree was a **failure**, so this miss had real stakes: had the fix been wrong, broken master would have sat unnoticed.
3. Did not verify FEATURES.md/ROADMAP.md/README.md claims myself this session (spot-checks during self-review found them correct: `MiddlewareETag` survives in stack.go:23 as a composition seam, FEATURES:231 is struck through, README's ETag section documents `etag.New` not the removed adapter — but that was luck-adjacent, not verified-by-process).
4. The two `t.TempDir()` calls in `TestServerStartTLSServesHTTPSWithSelfSignedCert` (two separate temp dirs) — found, judged harmless, not fixed. Defensible; inconsistent with the fix-on-sight bar.

**What is something that's stupid that we do anyway?**

- `MiddlewareETag` name-constant survives with no in-repo implementation — _intentional_ (composition seam for `etag.New`), but nothing says so; a consumer reading FEATURES:50 cannot tell the constant is still meaningful.
- The `[Unreleased]` CHANGELOG section has **duplicated `### Added` and `### Changed` headers** (two of each) — keep-a-changelog wants one of each; a pre-tag reorganization is needed.
- `doc-snippet-refs` runs in CI but **not in `prerelease-check.sh`** — local/CI gate parity gap of exactly the kind `--skip-flake` was added to fix.
- The gopls `stdversion` warnings (6 standing diagnostics) are decided-and-recorded but still render in every session; noise we chose.

**What could you have done better?**

- Read the parallel session's `2026-09-11_09-01_starttls_listener_race_fix.md` **before** pushing master — it went to the public remote unread (daemon-committed, push authorized, but READ-before-WRITE applies to what I publish, not just what I edit).
- Run the canonical coverage-gate invocation _first_ (as prerelease-check.sh does) instead of inventing the flag-form invocation twice before reading the script.
- Front-load the grep for stale symbols (`etag`, `ratelimit`, `RateLimit(`) across ALL living docs at session start — it would have caught the `etag_test.go` miss in one query instead of by post-hoc self-review.

**What could you still improve?**

- A mechanical "deleted symbol → grep living docs" step belongs in every removal task; TODO it as process, not memory.
- Verify-by-running beats verify-by-reading: the two real finds this session (coverage gate, stale bullet) both came from execution.
- The nightly-fuzz verification bar ("step 1 passes") was met, but 24 of 25 steps remain unobserved; identical discipline, zero independent confirmation until the run completes.

**Did you lie to you/the owner?** No. Every claimed completion was verified by execution (tests, gates, CI run IDs, `gh` output). Two near-lies avoided by checking first: the "unused imports" diagnostic was stale (build proved it), and the RequestID bullet was stale in the _other_ direction (ring landed, bullet said race open).

**Ghost systems?** One candidate examined: `MiddlewareETag` constant without implementation — ruled a deliberate seam, not a ghost, but undocumented as such (improvement item).

**Split brains?** One created-and-fixed by the _previous_ session's docs (RequestID ring vs old race bullet), one left by both sessions (`etag_test.go`), one pre-existing (`[Unreleased]` duplicate headers).

**Scope creep?** The AGENTS.md stale-ref sweep grew from 3 planned bullets to 9 edits. Net positive (each verified), but it consumed budget the TODO reconciliation needed — the reconciliation survived, barely.

**Tests?** Untouched Go code; `-race` green via three separate runs. The lost-review-pass re-run over ~3.5k lines found **zero findings** — coverage discipline is holding.

---

## a) FULLY DONE

| #  | Item                                                                                                                                                                                                                                                                                                                               | Evidence                                                                                                                                 |
| -- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------- |
| 1  | AGENTS.md Non-Obvious Behaviors update (CSRF XFP trust model + EqualFold, Server double-start guard, MiddlewareStack atomic snapshots)                                                                                                                                                                                             | AGENTS.md:195-197; contracts cross-checked against csrf.go:680-727, server.go:203, CHANGELOG                                             |
| 2  | AGENTS.md stale-ref sweep: TokenBucketLimiter "deprecated"→removed, `RateLimit` constructor-list drop, `writeCommittedBody` callers, `etag.go` adapter ×2, RequestID race→ring, `etag_test.go`                                                                                                                                     | 7 edits, each verified against code (`ls`, `grep`) before/after                                                                          |
| 3  | TODO_LIST.md rewritten: 14 items closed (each execution-verified), 9 recorded review findings added as post-v1.1/v2.0 candidates, owner-blocked v1.1.0 marked                                                                                                                                                                      | TODO_LIST.md; verification trail in session log (README:151/382/438, code_test.go:67-110, go.mod v0.10.0 = latest)                       |
| 4  | Coverage-gate bug found + fixed: `scripts/` tree-wide exclusion in `prerelease-check.sh:63` + `ci.yml:56`; documented in CHANGELOG [Unreleased]                                                                                                                                                                                    | Gate 97.3% locally (was 93.1%); **CI green on master** (run 34574170086) after the pre-fix failure (run 34572894391) proved the bug live |
| 5  | `nix fmt` clean (0 changed) + `nix flake check` all checks passed                                                                                                                                                                                                                                                                  | Including the new `server-timing-standalone` check                                                                                       |
| 6  | `go mod tidy` both modules — no drift                                                                                                                                                                                                                                                                                              | Clean tree after                                                                                                                         |
| 7  | End-to-end `prerelease-check.sh`: **all 9 gates passed** — first full green run on this tree                                                                                                                                                                                                                                       | "All pre-release gates passed" + flake tail                                                                                              |
| 8  | master pushed: `4f304c8..ffda15e` (4 commits)                                                                                                                                                                                                                                                                                      | `git push` output; branch in sync                                                                                                        |
| 9  | nightly-fuzz re-dispatched; **step 1 `^FuzzDecompression$` passed** (300 s); no new false-positive issues (#7/#8 stay closed)                                                                                                                                                                                                      | Run 34574182326; step transition visible via `gh run view`                                                                               |
| ~~10~~ | ~~Both lost review passes re-run by direct authorship (~3.5k lines: server_test, chain_test, ratelimit_keyed_test, id_generator_test, both scripts, examples/compose): **zero findings**~~ superseded 2026-09-15: the question-② split-brain re-run found 3 real findings (fixed in `0cb25ea`; docs/status/2026-09-15_06-14 a2–a6) | ~~Owner question ② resolved — moot~~ resolved again 2026-09-15, this time with findings                                                  |
| 11 | Stale-diagnostic triage: `doc-snippet-refs` "unused imports" warning disproven by build+vet+lint (LSP cache lie)                                                                                                                                                                                                                   | `go build ./...` + `go vet` + prerelease lint all green                                                                                  |

## b) PARTIALLY DONE

~~1. **nightly-fuzz full run** — in progress (13m31s at last check); step 1/25 verified; remaining 24 steps run the same anchored discipline but are unobserved until the ~2 h run completes.~~ done (observed green — no crashers; #7/#8 stayed closed)
~~2. **Docs freshness** — AGENTS.md fully reconciled, but no systematic docs-health VERIFY pass ran this session; the `etag_test.go` find proves hand-checked-only docs still hide drift. README/FEATURES/ROADMAP spot-checks passed.~~ done (docs-health pass 2026-09-15 ran the full VERIFY; renewed 2026-09-23)
~~3. **`[Unreleased]` CHANGELOG hygiene** — content is complete and accurate, but has duplicated `### Added`/`### Changed` headers; needs a pre-tag reorganization into single sections.~~ done (fixed at the v1.1.0 cut — single Added/Changed/Fixed/Removed blocks)
~~4. **server_timing sub-module** — build verified via flake standalone check; its `go test -race` not explicitly re-run this session (code untouched since last green).~~ done (re-run since — release batteries and the 2026-09-23 pass cover the sub-module)
~~5. **erraudit `--type-aware` advisory pass** — the two enforcing gates ran green in prerelease; the advisory-only full pass (~30 known-correct sentinel matches) not re-run.~~ done (re-measured 2026-09-15 — 45 sentinels / 41 test-side; AGENTS.md updated)
   ~~6. **Owner question ②** — resolved by execution (review passes re-run clean), but the 09-05 status report is not yet annotated with that answer (docs-health ANNOTATE territory).~~ annotated 2026-09-15 (docs-health pass); the underlying 're-run clean' claim was itself superseded — the 2026-09-15 re-run found and fixed 3 findings (docs/status/2026-09-15_06-14 a2–a6).

## c) NOT STARTED

1. ~~**v1.1.0 tag cut** — owner-blocked (question ① + ③); master is now in a tag-ready state: all gates green, pushed, CHANGELOG current.~~ done at `ae0a46d`
2. **go-compression extraction** — deferred post-v1.1 by decision; plan inventory refresh is a precondition.
3. **docs-site / website-launch** — httpspec discovery page et al.
4. **architecture-review re-run** — post-v1.1 scope.
5. ~~**go-error-family upstream issue filing** — verified draft waiting; owner action.~~ done (filed 2026-09-15 as go-error-family#5)
6. ~~**`withParsedTrustedProxies` export decision** — conditional, trigger not fired.~~ done (decided 2026-09-15 — stays unexported; zero public importers, trigger not fired (DECISION_LOG))
7. ~~**Full docs-health VERIFY/ANNOTATE cycle** for the 09-05 report + monthly cadence.~~ done (done — the 2026-09-15 docs-health pass annotated 09-05; this 2026-09-23 pass renews the cycle)

## d) TOTALLY FUCKED UP

Nothing rose to "fucked up" — no data loss, no broken master, no re-tagging. The honest worst-of:

1. **The coverage-gate bug shipped to CI in the first place** (prior session added `doc-snippet-refs` without re-running the gate; CI failed at 07:06 on master). Found and fixed this session, but it sat red on the public master for ~17 minutes of a session that claimed "individual gates pass." The lesson is the AGENTS.md rule already written: the _delivering layer_ (the gate) must be run, not just the changed code.
2. **Published unread content**: the parallel session's listener-race-fix report rode my push to the public remote without me reading it first. Process violation, zero known damage.
3. **`etag_test.go` stale ref survived a dedicated freshness pass** — the meta-failure: doc-freshness done by memory and narrative instead of by mechanical grep over deleted symbols.
4. **Two wasted tool calls** on the coverage checker (wrong flag form, wrong input format) before reading the canonical invocation in prerelease-check.sh — the script I was about to run anyway.

## e) WHAT WE SHOULD IMPROVE

1. **Mechanical stale-symbol sweep on every removal**: `git grep -e '<symbol>' -- '*.md'` across living docs is a 10-second step that beats narrative freshness passes. Make it a checklist row in docs-health VERIFY.
2. **Run the delivering gate first**: when a session changes anything near CI/scripts, the _first_ action is the gate invocation from `prerelease-check.sh`, not a reconstructed one.
3. **Local/CI gate parity**: `doc-snippet-refs` belongs in prerelease-check.sh so local preflight == CI. (`--skip-flake` set the precedent.)
4. **READ-before-PUBLISH**: push-bound commits get skimmed, even daemon-authored ones.
5. **Post-push CI check is part of "push"**: pushing without watching the triggered run is an unverified claim waiting to age badly.
6. **CHANGELOG section discipline**: enforce single `### Added/Changed/Fixed` per release section (pre-tag fix).
7. **Document the `MiddlewareETag` seam** (one FEATURES line: constant retained for naming go-etag stacks) so the constant doesn't read as a ghost.
8. **Keep fix-on-sight for two-line doc fixes** — it worked exactly as granted (etag_test.go), extend it to one-line code nits (t.TempDir collapse) where lint-safe.

## f) NEXT 50 (brainstorm sorted by impact; most beyond top ~15 are ROADMAP fuel — docs-health HARVEST should route, not commit)

**Release-critical (this week)**

1. ~~Owner ruling: cut **v1.1.0** now vs after nightly-fuzz full green (question ①).~~ done (resolved — v1.1.0 cut 2026-09-11 after the gates went green (tag ae0a46d))
2. ~~Watch nightly-fuzz run 34574182326 to completion; triage any crasher; close any auto-filed false positive.~~ done (observed green — no crashers; auto-filed issues stayed closed)
3. ~~Reorganize `[Unreleased]` CHANGELOG to single Added/Changed/Fixed sections; retitle to `[1.1.0]` at tag time.~~ done (done at the v1.1.0 cut — single sections; retitled [1.1.0])
4. ~~At tag: run RELEASE.md steps 7-16 (GitHub release, pkg.go.dev propagation, `go get` verification, badge).~~ done (done — RELEASE.md steps executed for v1.1.0)
5. Add `doc-snippet-refs` step to prerelease-check.sh (CI parity).
   ~~6. Annotate the 09-05 status report: question ② answered (passes re-run clean); ① resolution once ruled.~~ done 2026-09-15 (docs-health pass annotated the 09-05 report's question-② and ③ items; the ② answer was revised — the 09-15 re-run found 3 findings).
6. Triage incoming Dependabot PRs (weekly cadence is live).

**Owner decisions pending**
8. ~~CSRF security-degrading config: log-only vs remediate (question ③) → then implement the ruling.~~ done (resolved 2026-09-15 — owner ruling; B1 fallback + AllowInsecureSameSiteNone shipped in v1.2.0)
9. ~~Issue #4 ruling: negotiate gzip for absent `Accept-Encoding` (RFC 7231) or keep documented current behavior; either retire the issue or convert to spec'd work.~~ done (resolved 2026-09-15 — owner instructed the identity default; AbsentEncoding shipped in v1.2.0; issue closed)

**Post-v1.1 recorded review findings (from the HTML report, all in TODO_LIST)**
10. ~~Unify the two `TrustedOrigins` parsers with tests.~~ done (done 2026-09-14 — parseTrustedOrigin is the single parse point (17-14/18-14 reports))
11. ~~Reclassify `http.ErrNoCookie`/`ErrNoLocation` (Transient→Rejection) + fix `ErrCodeHijackFailed` doc/WayOut mismatch; own changelog entry.~~ done at `9b9e032`
12. ~~Document `ValidateCSRF` caller-request mutation; consider context flag.~~ done (done — shipped as the v1.2.0 documentation-only Fixed entry (ValidateCSRF mutation contract documented))
13. ~~`nosurf.StaticOrigins` failure: fail-closed design pass (TrustedProxies-style).~~ done (done — shipped in v1.2.0 as the all-or-nothing fail-closed parsing)
14. ~~CSRF remediation implementation (depends on #8).~~ done (done — shipped in v1.2.0 (withSecureFallback + opt-out))
15. v2.0 ledger: `ValidateCSRF` result type, `MiddlewareFunc` canonicalization.
16. Additive: typed `MiddlewareStack` names (type + overload).
17. Pool-contract hardening: `(nil,nil)` probe, factory-param divergence, release provenance (next pool touch).

**Docs & docs-health**
18. ~~Full docs-health VERIFY pass over living docs with the mechanical stale-symbol grep (the etag_test.go class-hunt).~~ done (docs-health pass 2026-09-23, full VERIFY with mechanical sweeps (etagmetrics/vendor/etag.go greps) executed)
19. ~~Document `MiddlewareETag` as composition seam (FEATURES one-liner).~~ done (present — FEATURES.md documents MiddlewareETag as the go-etag composition seam)
20. docs-site launch (website-launch skill): httpspec page, spec-runner example, discovery push.
21. ~~File the verified go-error-family conditional-request issue (owner repo).~~ done (filed 2026-09-15 as go-error-family#5)
22. architecture-review re-run post-v1.1 (compose API, XFP trust model, removals in scope).
23. ~~Monthly docs-health cadence: schedule next VERIFY before v1.2 planning.~~ done (cadence held — full passes ran 2026-09-15 and 2026-09-23)
24. AGENTS.md size watch (docs-health budget) — next pass likely crosses the line again.
25. DOMAIN_LANGUAGE: add "generation-swapped ring", "attestation conflict", "trusted proxy" entries if missing.

**Extraction & architecture**
26. go-compression extraction: refresh the plan's line-by-line inventory (writerPool probe invalidated it), then phase per plan discipline.
27. Server TLS-config mutation: evaluate clone-on-write or doc-only hardening (the `Clone()` footgun is documented; is it enough?).
28. Evaluate `internal/` extraction trigger (post-v1.0 or ~50 non-test files — track the count).
29. Consider `go-workflow-auditlog`-style sibling for Server-Timing if third consumer appears (ROADMAP fuel only).

**Testing & tooling**
30. ~~Unit tests for `doc-snippet-refs` (fixture fences + golden findings) — currently 0%; decide if it stays intentionally untested or earns a suite.~~ done (decided — intentionally untested per v1.1.0 (the CI drift check is its verification; CHANGELOG [1.1.0]))
31. benchstat regression gate in CI (pin the 3s×5 protocol output; fail on >N% regression).
32. Fuzz corpus expansion: real-world odd `Accept-Encoding`/`Origin` seeds from issue trackers.
33. Weekly (not per-push) `-race -count=10` stress workflow to cut CI minutes without losing the timing-race net.
34. ~~server_timing `-race` into prerelease-check.sh explicitly (parity with root module).~~ done (present — prerelease-check.sh runs server_timing vet + race + lint explicitly)
35. Property test for the fence parser (weird markdown: nested fences, CRLF, no-trailing-newline).
36. ~~Coverage-gap audit: confirm every sub-100% function still has its FEATURES reason (docs-health owns).~~ done (docs-health pass 2026-09-23, FEATURES gap section re-verified against a fresh -race -coverprofile (code.go trio now 100%))

**Ecosystem & consumers**
37. go-datastar SSE trigger check for go-compression timing.
38. httpspec zero-public-importers: outreach/discovery plan decision (README + example landed; page pending).
39. ~~Post-tag: monitor pkg.go.dev + first external `go get`.~~ done (done for v1.1.0 — pkg.go.dev verified in the 2026-09-15 docs-health pass (a12))
40. ~~GitHub Release notes for v1.1.0 drafted from CHANGELOG (freeze policy: no retro-edits after).~~ done (done — v1.1.0 GitHub release published from the frozen section)
41. ~~Public README claim sweep post-removal: "four external dependencies" wording vs go-etag-stays reality (AGENTS fixed; README unverified for this phrasing).~~ done (verified 2026-09-23 — README says four dependencies; accurate)

**Hygiene & process**
42. Fix the two-`t.TempDir()` nit in the TLS test (one dir, two Join calls).
43. `TestServerShutdownReturnsErrorOnContextExpiry`: replace raw `&Server{httpServer:...}` literal with a helper that survives future invariant fields.
44. gopls `stdversion` warnings: revisit when Go 1.27 + stable json/v2 land (DECISION_LOG row exists).
45. ~~`.gitignore` vendor/ entry: keep (guard against stray `go mod vendor`); verify in next hygiene pass.~~ done (verified 2026-09-23 — .gitignore keeps the vendor/ guard line, documented in AGENTS.md)
46. Auto-daemon commit messages: consider enforcing conventional prefixes for daemon commits too (commit-lint only gates PRs today).
47. ~~Status-report archive policy: move fully-resolved `docs/status/` reports to `archived/` once their questions are answered.~~ done (adopted — AGENTS.md Doc-Freshness documents the archive policy; executed in the 2026-09-15 and 2026-09-23 passes)
48. nightly-fuzz: add a run-summary step (single job-summary line: targets × crashers) so "is it green" is one glance.
49. Consider `workflow_dispatch` inputs (target-subset, fuzztime) on nightly-fuzz for cheaper verification dispatches.
50. Retire or finish the two `//nolint:makezero` sites if a modernize release makes append idioms equivalent (low priority, lint-hygiene).

## g) QUESTIONS I CANNOT FIGURE OUT MYSELF (3)

1. ~~**v1.1.0 timing:** cut the tag **now** (master pushed, all nine gates green, review passes clean) or **after** the full nightly-fuzz run goes green (~2 h)? Tags are permanent, so this is purely your risk call: ship on green gates, or ship on green gates _plus_ a full 2-hour fuzz soak.~~ done (resolved — v1.1.0 cut 2026-09-11 (tag ae0a46d))
2. **CSRF security-degrading config** (carried, still blocks TODO Medium #1): keep validate-and-log as final (documented 2026-08-08 decision), or spend the design pass to remediate dangerous combos (`SameSite=None` + `Secure=false` etc.) to secure defaults in v1.1.x/v1.2? ~~Answered 2026-09-15: remediate — B1 fallback + `AllowInsecureSameSiteNone` opt-out shipped (DECISION_LOG 2026-09-15).~~
3. ~~**Issue #4** (open since 2026-08-29): "Compression negotiates gzip for an absent `Accept-Encoding` header (RFC 7231 says identity)" — keep the current documented server-priority behavior, or align with the RFC reading and return identity when no header is present? Your ruling either retires the issue or converts it into a spec'd (behavior-changing) task.~~ done (resolved 2026-09-15 — identity default instructed, AbsentEncoding shipped in v1.2.0, issue closed)

---

_Branch: master @ ffda15e (+ this report). Tags pushed: v1.0.0, v1.0.1. Nightly-fuzz run 34574182326 in progress. WAITING FOR INSTRUCTIONS._
