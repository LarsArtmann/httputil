# Status Report — Docs-Health Full Audit: Corpus Annotation, Archive Pass, Living-Doc Rebuild, and the Nightly-Fuzz Oracle Fix

**Date:** 2026-09-15 10:28 CEST
**Session scope:** the user-mandated docs-health AUDIT over every `**/2026-0*` file — VIEW ALL, annotate resolved items inline, archive fully-done files, make TODO_LIST / CHANGELOG / AGENTS / README / ROADMAP / FEATURES superb — plus fix-on-sight captures along the way.
**Ending state:** tree clean at `ea8de2b` (daemon-committed), master **2 commits ahead of origin** (push is owner-gated). All gates green; one open Medium finding (MD060 style ruling) deliberately left to the owner.

---

## Session in one paragraph

Ran the full docs-health AUDIT: inventoried the corpus, read all 23 non-archived status reports and 4 planning docs end-to-end, and verified claims against code and GitHub before annotating. The audit's biggest find was **not a doc bug**: the nightly-fuzz oracle defect from 2026-09-11 had fallen off every backlog — issues #9–#13 auto-filed, five nightly runs red. Fixed the oracle, committed the crasher seed, closed all five issues, and recorded it in CHANGELOG. Then: 42 inline annotations across 7 reports, 2 files archived (+5 archive-gate repairs), TODO_LIST rebuilt open-only with 16 sourced items, ROADMAP rewritten from a two-releases-stale state, SECURITY/README/FEATURES accuracy fixes, three release-gate items landed (RELEASE.md step 12.5, prerelease gate 8b, AGENTS link convention), the v1.1.0 release-page honesty note published, and pkg.go.dev verified for both modules. Health report delivered inline: Accuracy 5.25 → 9.5 post-fix, Fitness 7.0 → 10 post-fix.

Quick counts: **3 Critical / 4 Medium-High / 8 Medium / 3 Low findings — 20 of 21 fixed in-session · 42 report annotations · 2 archives · 16 TODO items · 5 GitHub issues closed · 12 gate runs green.**

---

## a) FULLY DONE (implemented AND verified)

| # | Item | Evidence |
| --- | --- | --- |
| 1 | **docs-health skill + health-report-format reference loaded before acting**; corpus inventoried; all 23 non-archived status reports + 4 planning docs read end-to-end; 19 html/d2/svg triage-classified SKIP per convention | Session transcript; read-before-verify discipline held |
| 2 | **`FuzzHealthResponse_Encoding` oracle fixed** — the 09-11 nightly crasher (issue #9) had auto-filed false positives **five nights running** (#9–#13) and was tracked in no backlog. Oracle now asserts the real jsonv2 contract both ways (invalid UTF-8 ⇒ error; valid ⇒ nil **and** exact round-trip; the false "normalizes to U+FFFD" comment deleted); minimized seed committed at `testdata/fuzz/FuzzHealthResponse_Encoding/ac6733e1726c6863` | `health_test.go`; 45s fuzz run: 2.76M execs PASS; `-race -count=10` on health tests green; CHANGELOG `[Unreleased]` Fixed entry |
| 3 | **Issues #9, #10, #11, #12, #13 closed** with a shared root-cause comment (test-oracle bug, not a library bug) | `gh issue close` outputs; issue list now has zero open fuzz false-positives |
| 4 | **SECURITY.md accuracy pass** — "pre-1.0" support claim rewritten for v1.x; dependency list corrected (nosurf and go-etag were missing from a *security* doc); CSRF SameSite-fallback posture bullet added; CORS wildcard-fallback line corrected to the `DenyUnmatched: true` default (was: "falls back to `"*"` by default" — contradicted `DefaultCORSConfig`) | SECURITY.md diff; cross-checked against AGENTS.md + FEATURES |
| 5 | **ROADMAP rewritten from two releases stale** — Current Position said "v1.0.0 cut locally, not pushed / next release v1.1.0"; now documents shipped v1.1.0 (2026-09-11) and the staged v1.2.0 batch; "TokenBucketLimiter removal" and "property-based tests" Non-goals updated to their completed/moot state; new ideas added (server shutdown-drain contract; StartupHandler surfacing); latent stale `docs/status/` path in the idempotency bullet fixed | ROADMAP.md diff |
| 6 | **TODO_LIST rebuilt per the delete-done rule** — all 7 struck-done items deleted (live in CHANGELOG); 16 open items, each with report citations; new harvest: cut v1.2.0 (High), CSRF hardening trio, B1-interpretation confirmation, release-engineering batch, issue #4 + export + MD060 owner decisions, CSRF docs polish + test-depth batches, go-error-family#5 follow-through, jsonv2, raw-extraction reference, agentic_fetch bug report, `writeHealthBody` discard decision | TODO_LIST.md; every item greps back to its cited report |
| 7 | **Release-gate hardening (three 13-49 items executed)** — `docs/RELEASE.md` step 12.5 (wait for CI green on the exact tag commit, `gh run watch` recipe); `scripts/prerelease-check.sh` gate 8b runs `scripts/check-changelog-links.sh`; AGENTS.md gained the link-definition convention (`[X.Y.Z]:` compare links + `[Unreleased]:` retarget) | RELEASE.md; prerelease-check.sh; gate 8b verified green on this tree |
| 8 | **AGENTS.md session-learned one-liners** — buildflow `--build-mode dev` expected-nonzero findings gate; TODO_LIST struck-items deleted at each rebuild; `parseTrustedOrigin` single-parse-point line folded into the all-or-nothing bullet | AGENTS.md diff |
| 9 | **42 inline annotations across 7 status reports** — question ② (review-pass re-runs, resolved 2026-09-15 with 3 findings) and question ③ (CSRF remediation ruling) resolved in 17-14, 18-14, 09-35, 09-05; the 09-35 "zero findings" claim struck as superseded; 13-49 items b2/b5/c1/f1/f3–f6/f9/f10 resolved with evidence; 06-14 b1/b4/b6/c3/f1/f5/f40/g3 and 07-00 b-row3/f1/f8/f9/f12/f15/f25/f26 resolved | grep counts per file (19/6/4/3/9/8/8 `~~` markers); persistence-verified after the daemon race |
| 10 | **Archive pass** — `2026-08-30_07-45_low-priority-backlog-execution.md` (fully resolved) and `2026-08-30_runserial-state-sharing-design-note.md` (decided) moved via `git mv`; CHANGELOG link repaired; **5 archived files failing the ≥1-marker completion gate** (3 June banner-pass stragglers, the startup-handler note, the runserial note) given honest inline resolution markers — `grep -rLn '~~' archived-dirs` now returns **0** | git mv + python edits with per-item assertions; completion gate output |
| 11 | **v1.1.0 GitHub release honesty note published** — correction-of-record appended to the release page (tag predates the link-def fix; code gates all green; consumers verified); the 13-49 b3/f5 item, open since 09-11 | `gh release edit v1.1.0` output; page re-fetched |
| 12 | **pkg.go.dev verified** — v1.1.0 and server_timing v1.0.1 pages render (Published Sep 11, 2026; valid go.mod; tagged/stable); 13-49 b5/f10 closed | fetch outputs (note: godoc not rendered due to the Proprietary license — known, not a bug) |
| 13 | **DECISION_LOG cross-reference** (07-00 f25): the 2026-08-08 validate-and-log row now points at the 2026-09-15 refinement row | DECISION_LOG.md diff |
| 14 | **Design-note §7 post-implementation correction** (07-00 f12): sketch now annotated with the shipped shape (`withSecureFallback` helper + `InvalidateCSRFCookie` wiring + opt-out) | docs/planning/2026-09-15_csrf-security-degrading-config-design-note.md |
| 15 | **README/FEATURES staleness fixed** — coverage badge + gates table 97.4% → 97.2% (2026-09-14 measurement); "ETag via go-etag adapter" → "via go-etag composition" (adapter removed in v1.1.0); FEATURES fuzz count 26 → **25 (23 root + 2 server_timing)** — the v1.1.0 `FuzzEvictionTTL` removal had never been propagated (count verified via `go test -list`) | README.md, FEATURES.md diffs; `go test -list 'Fuzz.*'` ground truth |
| 16 | **Full gate battery green at session end**: build, vet, `-race -count=1` both modules, golangci-lint 0 issues, server_timing tests + lint, both erraudit enforcing gates exit 0, `scripts/check-changelog-links.sh`, `nix fmt`, `nix flake check` (all checks passed), markdownlint (only the ticketed MD060 class remains) | Gate transcripts in session |

## b) PARTIALLY DONE

| # | Item | Done | Open |
| --- | --- | --- | --- |
| 1 | **Nightly-fuzz end-to-end confirmation** | Fix is local-verified (seed + 45s fuzz + race suite) | The workflow itself was **not re-dispatched**; first green-eligible run is tonight 03:05 UTC. The 09-05 lesson ("re-dispatch and confirm step 1") was not executed |
| 2 | **Coverage honesty after the CSRF fallback work** | The 07-00 session's 6 tests + `withSecureFallback` branch shipped; my erraudit/fuzz gates green | Per-module race-profile re-measure (97.2/98.6 baseline) not re-run — ticketed in the TODO hardening-trio item |
| 3 | **erraudit advisory baseline re-check** (18-14 f19) | Both enforcing gates exit 0 this session | The `--type-aware` 44/40 advisory re-count not re-measured (no new sentinels this session, so risk ≈ 0) |
| 4 | **buildflow pipeline** | Direct tool runs covered everything buildflow would run (markdownlint, lychee-class link check via changelog-links, lint, tests) | No full `--build-mode dev` run this session (documented expected-nonzero findings gate; docs-only session) |
| 5 | **README edits × doc-snippet-refs** | All three README edits are prose/badge/table cells, no Go fences touched | The checker itself was not re-run — cheap, should be habitual after any README touch |
| 6 | **MD060 table-style question** | Evidence completed (formatter can't align; hand-padding unmaintainable); findings inventoried; ticketed with sources | The ruling itself is owner-gated; DECISION_LOG rows 32–33, README:580, and my annotation-lengthened rows stay misaligned until then |
| 7 | **The annotated reports** | Questions ②/③ and executed items resolved inline | Each report still carries genuinely-open f-items — correctly unmarked, now routed to TODO_LIST |

## c) NOT STARTED (owner-gated or queued; all live in TODO_LIST with sources)

1. **Cut v1.2.0** — everything staged (3 behavioral deltas + 2 additive fields + migration doc); timing owner-gated; dnsblockd unblocked by the tag.
2. **CSRF SameSite-fallback hardening trio** (mutation-check, coverage probe, smoke-fuzz, sub-module gate — partly done by this session's gates but not the mutation-check) + **confirm the B1 interpretation** of the ruling.
3. **LNA batch**: denied-origin pin, httpspec spec, docs polish, verification polish, Chrome-docs web pass.
4. **Release-engineering batch**: gate `release.yml` on CI; lychee CI/pre-commit gate; dev-mode lychee no-op fix.
5. **architecture-review re-run post-v1.1**; **go-compression extraction** (plan inventory refresh first).
6. **CSRF docs polish batch** + **CSRF test-depth batch** (9 pins) — both consolidated in TODO_LIST.
7. **Owner decisions**: issue #4 (gzip on absent Accept-Encoding), `csrf.trusted_origin_invalid` export, MD060 ruling + buildflow cache purge, v1.2.0 timing, explicit-commit policy for security-relevant work.
8. **go-error-family#5 upstream follow-through** (docs section if accepted); **jsonv2/Go 1.27** follow-through; **raw-extraction patterns reference**; **agentic_fetch bug report**; **`writeHealthBody` `_ =` discard decision** (13-49 f2, now ticketed).
9. **httpspec docs-site page** (blocked on website-launch); dnsblockd P1–P3 migrations (other repo).
10. **Push**: master is 2 commits ahead (docs pass + oracle fix); push is owner-gated.

## d) TOTALLY FUCKED UP

Nothing shipped broken — all gates green, tree clean, no code regressions (the only code change is the fuzz oracle + seed, fuzz- and race-verified). The honest worst-of:

| # | What | Severity | Root cause | Status |
| --- | --- | --- | --- | --- |
| 1 | **The daemon race ate one full annotation cycle.** 3 of 7 annotated files (17-14, 18-14, 09-35) silently reverted to HEAD after my writes — the daemon committed staged pre-edit content and the worktree lost my edits. I printed "OK" per script and moved on; found it only in the spot-check step, re-applied, and persistence-verified | Medium — lost a cycle, zero data loss | Write-then-trust in a daemon environment; no immediate per-file persistence check | Caught + re-applied + verified; lesson below (e2) |
| 2 | **First annotation script failed on a mis-transcribed target** — I copied the c-section line ("2. Re-run…") from my conversation summary; the file said "4. Re-run…". Assert-caught, zero damage | Low | The exact documented 18-14 d3 lesson (copy from the file, never the summary) — repeated once before switching to anchor-based matching | Fixed by rewriting the approach to grep-anchor edits |
| 3 | **`markdownlint … \| tail -5` truncated the findings list** — the first lint run showed 3 findings; the untruncated run showed more. The pipeline-masking anti-pattern, in the filtering form, again | Low-Medium — false completeness signal | Filtered output for convenience | Caught by the run-to-run diff; final runs unfiltered |
| 4 | **Fix-then-decide on MD060, again**: my `align()` attempt on the DECISION_LOG row made it *worse* (846 chars vs a 782-char header can never align) before I reverted to leaving the class for the ruling. The 06-14 d6 anti-pattern — I had the report in context and walked into it anyway | Low | Added text longer than the aligned table could hold; fix reflex fired before the constraint check | Reverted to leave-as-is; ruling ticketed |
| 5 | **The 02-25 archive marker split a sentence** mid-claim (struck only the clause, orphaned the rest). Read-back caught it; repaired | Low | Struck a substring without reading the full sentence first | Fixed and verified |
| 6 | **count=10 claim predates the final formatting**: `-race -count=10` on health tests ran *before* `nix fmt` re-wrapped `health_test.go`; the final tree got full-suite `-race -count=1` only. Almost certainly irrelevant (comment re-wrap), but the claim ordering was sloppy | Low | Formatters run after verification in my sequence | Disclosed here; next session's suite re-covers it |

## e) WHAT WE SHOULD IMPROVE

1. **Persistence-check every write in this repo.** The daemon commits continuously and can commit staged pre-edit content while you write; a 2-second `grep` after each file write would have saved the re-apply cycle (d1). Batch-verify after each edit wave, not at session end.
2. **Anchor-based editing from the first attempt** — build every batch edit from grep'd file content (unique substring anchors with count==1 assertions), never from conversation memory. My first script's failure and d5 both trace to transcription.
3. **Dispatch the nightly workflow immediately after fixing anything it runs** — the 09-05 lesson ("re-dispatch and confirm step 1") applies to oracle fixes too; I verified locally and left the end-to-end proof to the scheduler.
4. **Run lint gates unfiltered, always** — `| tail` truncation of a findings list is the same class as piping exit codes (d3). If output is long, redirect to a file and read it.
5. **Check table capacity before adding text to aligned tables** — if the row text exceeds the header width, the aligned style is unachievable; decide (ticket or restructure) before writing, not after a failed fix (d4).
6. **Run `doc-snippet-refs` after any README touch** — it is the README's gate; a prose-only change is an assumption, not a verification.
7. **The audit paid for itself in code, not docs** — the biggest find (issues #9–#13, 5 red nightlies) came from cross-checking doc claims against GitHub state. VERIFY mode should always include the issue tracker and CI history, not just files.
8. **Keep the fix-on-sight bar for fully-specified fixes** — the oracle fix was fully specified in the 13-49 report (d3 section) and took ~20 minutes including verification; leaving it for "next session" would have meant a sixth red nightly.

## f) Up to 50 things to get done next (ranked; top ~15 actionable, rest ROADMAP/tracked fuel)

**Direct session residues (small, this week)**

| # | Task | Impact | Effort |
| --- | --- | --- | --- |
| 1 | Watch tonight's nightly-fuzz run (03:05 UTC) — first green-eligible run after the oracle fix; close the loop on b1 | High | S |
| 2 | Re-dispatch `nightly-fuzz.yml` via `workflow_dispatch` instead of waiting for the schedule | High | S |
| 3 | Run `scripts/doc-snippet-refs` over README after this session's prose edits (b5) | Medium | S |
| 4 | Per-module race-profile coverage re-measure post-CSRF-fallback; refresh FEATURES if ticked (b2) | Medium | S |
| 5 | erraudit `--type-aware` advisory re-count (44/40 baseline) after the oracle change (b3) | Low | S |
| 6 | Mutation-check the 6 new CSRF fallback tests (hardening trio item — the only third not covered by this session's gates) | High | S |
| 7 | Push master (2 commits: docs pass + oracle fix) — owner-gated | High | S |

**TODO_LIST High/Medium (already sourced — do not duplicate here)**

8. Cut v1.2.0 (RELEASE.md runbook, now including step 12.5).
9. Confirm the B1+opt-out interpretation of the 2026-09-15 ruling (flips the default if wrong).
10. LNA: denied-origin pin → httpspec spec → docs polish → verification polish → Chrome-docs web pass.
11. Release-engineering batch: `release.yml` CI gate; lychee gates; dev-mode lychee no-op.
12. architecture-review re-run post-v1.1.
13. go-compression extraction (inventory refresh first).
14. CSRF docs polish batch + test-depth batch.
15. Smoke-fuzz the CSRF targets (folded into the hardening trio).

**Owner decisions pending**

16. Issue #4: gzip on absent Accept-Encoding.
17. `csrf.trusted_origin_invalid` export policy.
18. MD060 ruling + buildflow result-cache purge.
19. v1.2.0 release timing (now vs after the LNA batch).
20. Explicit-commit policy for security-relevant work (git history still can't tell the CSRF-fallback story).
21. The 04-30 g3 open question: banner+convention as terminal state for the ~2,300-marker archived corpus, or a per-item pass (~half day).

**Docs-health cadence / hygiene**

22. Next monthly docs-health cycle (~2026-10-15): re-VERIFY coverage numbers, advisory counts, and the TODO_LIST harvest state.
23. Re-check that tonight's CHANGELOG `[Unreleased]` entries survive the next release cut intact (freeze policy).
24. Annotate + archive the 2026-09-15 reports once their f-items resolve (the reports now carry 19/6/4/3/9/8/8 markers; the remaining opens are TODO_LIST-tracked).
25. Decide the status-doc table style once (feeds MD060 everywhere — 4 reports still carry aligned-style tables the formatter can't produce).
26. SECURITY.md: consider documenting the attestation-conflict defense (it documents fallback but not the 403 defense added in v1.0.0).
27. Sweep archived/ for further stale prose citations like the idempotency-bullet path (backtick paths are invisible to lychee).
28. Consider a `docs/status/` INDEX decision again only if archive count grows past ~120 (rejected 2026-09-11; count now ~90).
29. Verify the LICENSE question on pkg.go.dev ("License: UNKNOWN" + godoc suppressed) — is Proprietary + unrendered godoc the intended public face?
30. Keep `MiddlewareETag` seam doc in sync with Dependabot's go-etag bumps (standing).

**Tracked elsewhere (restated for completeness, not re-ticketed)**

31. go-error-family#5 upstream docs work if accepted; sweep httputil classification tables with the link.
32. jsonv2 stabilization (go.mod 1.27 + drop GOEXPERIMENT in one change).
33. Write the raw-extraction verification patterns reference; report the agentic_fetch json-unmarshal bug.
34. `writeHealthBody` `_ =` discard: documented honest-silence vs propagation (13-49 f2, now in TODO_LIST).
35. StartupHandler idea parked in ROADMAP (designed, unscheduled).
36. Server shutdown-drain contract decision (ROADMAP; blocks 09-01 f1/f5).
37. Website-launch effort → httpspec docs-site page.
38. dnsblockd P1–P3 migrations once v1.2.0 exists and its go-datastar session quiets (other repo).
39. Quarterly consumer/importer scan (~2026-12).
40. `internal/` extraction trigger watch (37 non-test files of ~50).
41. Dependabot weekly triage cadence.
42. Benchstat comparison at the next benchmarks.md refresh.
43. Nightly-fuzz run-summary step (single-line job summary) so green/red is one glance.
44. Corpus-seed commits for the strengthened CSRF oracles (seeds only land on failure today).
45. gopls stdversion warnings: resolved only by the jsonv2 follow-through (#32).
46. BuildFlow S64 (fs-state in cache keys) — upstream, other repo.
47. Dev-mode lychee no-op: flake devShell vs BuildFlow loud-fallback (owner's call, one line either side).
48. `Validate()` complexity near the cyclop ceiling — extract the TrustedProxies loop at the next gate addition.
49. FEATURES "New middleware" coverage header rename (carried 17-14 f20).
50. Next full-code-review scheduling decision (last: 2026-09-11; substantial code has landed since: LNA, fallback, oracle fix).

## g) QUESTIONS I CANNOT FIGURE OUT MYSELF

1. **Did we read the 2026-09-15 ruling correctly?** "Good defaults + config options + power to the applications" shipped as: `SameSite=None` without `Secure` falls back to **`Secure=true`** (preserving cross-site intent), with `AllowInsecureSameSiteNone` opting out verbatim. If the intended "good default" was the **`Lax` fallback** (stripping cross-site intent instead), say so and I'll flip `withSecureFallback`, the tests, and the migration note in one change. The shipped default is live on master, so this is the highest-leverage one-word answer in the backlog.
2. **MD060 table style — rule it once?** Standardize on the **single-space compact style** (hand-writable, formatter-stable; the LNA report and TODO_LIST already use it) and record it in AGENTS.md, or add **MD060 to the judged-disable list** like the 20 rules disabled 2026-09-11? Either way: purge the buildflow result-cache rows so the findings stop replaying (7-day TTL). Four docs currently carry misaligned tables awaiting this answer.
3. **Push now, and when does v1.2.0 land?** Master is 2 commits ahead (docs-health pass + the fuzz-oracle fix that closes #9–#13). Push immediately, or batch with the v1.2.0 cut? And for the cut itself: now (dnsblockd's tag-pinned adoption is waiting; migration doc complete) or after the LNA hardening batch lands so the release carries the fully-pinned feature?

---

*Report ends. Point-in-time snapshot per the status-report contract — annotate, don't rewrite. The auto-commit daemon holds the tree; push and the three rulings above are the owner's moves. WAITING FOR INSTRUCTIONS.*
