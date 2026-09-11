# Status: v1.1.0 Release Execution + Nightly-Fuzz Crash Discovery

**Session:** 2026-09-11, ~08:09–13:49 UTC (release session, owner instruction: "Release a new version!")
**Branch:** `master` @ `ba85626` (CI **green**, run 34579215622) · **Tag:** `v1.1.0` @ `ae0a46d` (SSH-signed, signature verified) · **GitHub release:** published 08:21:46Z

---

## a) FULLY DONE

1. **v1.1.0 released end-to-end** — CHANGELOG finalized (`[1.1.0] - 2026-09-11`), tag `-a -s` created ("Good git signature", ED25519), master + tag pushed, GitHub release published with the CHANGELOG section as notes and verified non-draft.
2. **CHANGELOG reorganization** — the duplicated `### Added`/`### Changed` headers merged into single canonical sections (Added → Changed → Removed → Fixed), 23 bullets preserved (count-verified), standing empty `[Unreleased]` section added (required by prerelease gate 8).
3. **All 9 prerelease gates green** end-to-end (build, vet, race tests both modules, lint 0 issues, erraudit, coverage 97.3% total / httputil 97.0% / httpspec 98.6%, changelog freshness, flake check).
4. **Sub-module release correctly skipped** — `git log server_timing/v1.0.1..HEAD -- server_timing/` is empty; zero drift; root go.mod's `require server_timing v1.0.1` stays accurate. No `server_timing/v1.0.2` needed.
5. **Consumer verification beyond the runbook** — not just `go get` + `go mod verify` (runbook step 15) but a real **build + run** of a program importing both `httputil v1.1.0` and `server_timing v1.0.1` through the module proxy: `CONSUMER_BUILD_OK`. `go get` alone can pass while a broken zip only fails at build time, so the probe was strengthened deliberately.
6. **Runbook steps 7–9 executed with verification** — FEATURES PLANNED section retitled "v1.1.0 stabilization (shipped 2026-09-11)" with the `[Unreleased]`→`[1.1.0]` reference update; `MiddlewareETag` documented as the intentional composition seam surviving the adapter removal (report item 19 from 09-35); step 9's historical-annotation sweep verified to have **zero** release-resolvable open claims (DECISION_LOG hits are standing decisions; RELEASE.md matches itself) — verified, not assumed.
7. **Benchmark baseline confirmed fresh** — `docs/benchmarks.md` was measured today, post-v1.1.0-removal, and already notes the ETag-adapter row death; runbook gate 6's refresh requirement satisfied without a redundant 3s×5 re-run.
8. **Granted on-sight fixes landed** — `server_test.go` double `t.TempDir()` collapsed (item 42; test verified green); FEATURES coverage figure corrected 97.4% → 97.0% to match the measured gate; `docs/RELEASE.md` gate-3 command widened to `grep -v /scripts/` (the documented manual command would now _fail its own gate_ since `scripts/doc-snippet-refs` is untested; noticed during this session's report prep).
9. **TODO_LIST High Priority closed with evidence** — push-master, fuzz re-dispatch, and cut-v1.1.0 items marked done with run IDs and verification notes.
10. **CI red-on-tag incident diagnosed and fixed on master** — link definitions added, `[Unreleased]` compare retargeted to `v1.1.0...HEAD`, freeze-policy `Fixed` bullet recorded; `scripts/check-changelog-links.sh` verified green locally before pushing; master CI green again.
11. **Nightly-fuzz crash triaged to root cause** (see d3/b2 — the crash is real, the bug is in the test oracle).
12. **`coverage.out` cleaned** via `trash-put` (runbook step 16).

## b) PARTIALLY DONE

1. **Nightly-fuzz cycle 34574182326: 24/25 targets green, 1 crasher** — `FuzzHealthResponse_Encoding` failed in seconds (see d3). All other targets (including the previously problematic `^FuzzDecompression$`) passed.
2. **Issue #9** (`nightly fuzz: crash detected 2026-09-11`) auto-filed and triaged in-session, **fix not yet landed** — the failing input (`ac6733e1726c6863`, 34 bytes, status `"\x82"`) lives in the job checkout, not in this repo's corpus yet.
3. **The v1.1.0 GitHub release page lacks the honesty note** about the tag commit's red link-check (fix landed in `ba85626`). The CHANGELOG `[Unreleased]` records it; the release page itself does not yet.
4. **Local unpushed docs edits** — RELEASE.md gate-3 fix and this report are committed-by-daemon but **not pushed** (per "THEN WAIT FOR INSTRUCTIONS").
5. **pkg.go.dev** — resolution verified through the module proxy (`go get`/`go build`); the public package page rendering itself was not visually confirmed.

## c) NOT STARTED

1. Issue #9 fix (oracle correction; corpus seed commit).
2. TODO_LIST Medium batch: CSRF remediation ruling (standing question ②), TrustedOrigins parser unification, stdlib error reclassification, ValidateCSRF mutation docs, StaticOrigins fail-open, go-compression extraction (preconditions 1–3), post-v1.1 architecture-review.
3. Post-v1.1/v2.0 items (review findings 1/7/8/9).
4. docs-health HARVEST of the two 2026-09-11 reports' next-item lists into TODO_LIST/ROADMAP.
5. v1.1.1 scope decision (see g1/g2).

## d) TOTALLY FUCKED UP

1. **CI was RED on the exact commit the v1.1.0 tag points at** (`ae0a46d`, CI run 34578794785, step "CHANGELOG link check"). Root cause, two stacked process failures of mine:
   - **I never read the bottom of CHANGELOG.md.** I viewed lines 1–49 and grepped the rest; the file's reference-link-definition block (lines 549+) requires a `[1.1.0]:` compare link per version heading, and `[Unreleased]:` must retarget to the just-cut tag. Restructuring a file without viewing it end-to-end is exactly the failure mode the global rules warn about ("READ the relevant context before editing").
   - **I pushed the tag before the tag commit's CI finished** — violating this session's own discipline (verify the _delivering_ gate; the summary's lesson after the unread-published-file incident was READ-before-PUBLISH, and I re-committed a variant of the same sin in the time dimension: PUBLISH-before-verify).
   - Blast radius: **docs-only.** The Release workflow (build + vet + race tests, both modules) passed on the same commit; all code gates were green; consumers are unaffected (proxy zip build+run verified). Per the immutable-tags policy the tag is **not** re-cut; master is green at `ba85626`; the fix is recorded in `[Unreleased]` per the freeze policy (v1.0.1 precedent).
2. **Wasted a full prerelease gate cycle** by not reading `scripts/prerelease-check.sh`'s gate list before reorganizing CHANGELOG — gate 8 hard-requires a standing `[Unreleased]` section, which Keep-a-Changelog practice would have suggested anyway. First run: 8/9 with a self-inflicted failure.
3. **The nightly fuzz found a real oracle bug the moment it ran clean** — `FuzzHealthResponse_Encoding` (health_test.go:138) asserts `json.MarshalWrite` returns nil for **any** `HealthStatus` string, but jsonv2 legitimately rejects invalid UTF-8 (`jsontext: invalid UTF-8 within "/status" after offset 10`). The fuzzer fed `"\x82"` and the oracle called a correct encoder rejection a crash. Secondary observation: production `writeHealthBody` (health.go:83) discards the `MarshalWrite` error (`_ =`) — harmless today (only `"up"`/`"down"` literals ship) but it would silently mask this entire class if health statuses ever become dynamic. The crasher is issue **#9**.
4. **Minor:** consumer probe written against guessed API signatures (`Chain()`, `RequestID()` with no args) — two avoidable compile round-trips; the signatures were one grep away. Also lost the commit race to the auto-commit daemon twice (attempted explicit commits on already-committed trees).

## e) WHAT WE SHOULD IMPROVE

1. **Release gate: CI-green-on-tag-commit.** `docs/RELEASE.md` must make "wait for CI green on the exact tag commit before `gh release create`" an explicit numbered step. Today it is implicit, and implicit steps get skipped — proven today.
2. **Fold `check-changelog-links.sh` into `prerelease-check.sh`** as gate 8b. The release script validated changelog _existence_ but missed the link-definition contract that CI enforces; the local gate must be a superset of CI, not a subset.
3. **Whole-file reads before restructures.** Any file being reorganized (not just edited) gets viewed end-to-end first — header, body, and trailing metadata blocks.
4. **Gate `release.yml` on CI** (workflow_run or a single-publisher design) so a tag whose CI is red cannot produce a published release silently.
5. **Document the CHANGELOG link-definition convention in AGENTS.md** (`[X.Y.Z]:` compare links, `[Unreleased]:` retarget at tag time) — it is a hard release constraint currently discoverable only by CI failure.
6. **Fuzz-oracle discipline for jsonv2:** encoder _refusals_ (invalid UTF-8, unsupported values) are legitimate outcomes; oracles must assert "error ⟺ contract-violating input," not "never errors." `FuzzHealthResponse_Encoding` is the second oracle-contrast finding this month; a review pass over string-input oracles for the same pattern is cheap insurance.
7. **Probe discipline:** grep the target repo's signatures before writing consumer verification programs.
8. **Codify the server_timing drift check** (`git log <last-submodule-tag>..HEAD -- server_timing/`) as an explicit prerelease step — I ran it ad hoc; it should be in the runbook.

## f) NEXT (prioritized; ≤50, from this session only)

1. Fix issue #9: correct the `FuzzHealthResponse_Encoding` oracle (invalid UTF-8 → assert the jsontext error; valid UTF-8 → nil + round-trip invariant), commit the minimized corpus seed (`ac6733e1726c6863`).
2. Decide + record the `writeHealthBody` `_ =` discard: documented honest-silence vs propagation (pairs with #1; health.go:83).
3. Add the CI-green-on-tag-commit gate to `docs/RELEASE.md` (step 12.5).
4. Add `check-changelog-links.sh` to `prerelease-check.sh` as gate 8b.
5. Add the honesty note to the published v1.1.0 GitHub release (tag predates the link-definition fix; fixed in `ba85626`).
6. Document the CHANGELOG link-definition convention in AGENTS.md hard constraints.
7. Gate `release.yml` on CI success.
8. Confirm the next nightly-fuzz cycle is 25/25 green.
9. Push the pending local docs (RELEASE.md fix + this report) with the next instruction batch.
10. pkg.go.dev page visual check for v1.1.0 and server_timing v1.0.1.
11. Decide v1.1.1 scope (see g1/g2): oracle fix alone would be a clean patch.
12. Owner question ② (standing): CSRF security-degrading config — log-only final vs remediate-to-secure-defaults.
13. Owner question ③ (standing): gzip-on-absent-Accept-Encoding ruling (issue #4).
14. TrustedOrigins parser unification (TODO_LIST Medium).
15. Stdlib error reclassification (`http.ErrNoCookie`/`ErrNoLocation` Transient→Rejection; `ErrCodeHijackFailed` doc/WayOut mismatch) — with its own changelog entry.
16. ValidateCSRF caller-request-mutation documentation.
17. `nosurf.StaticOrigins` fail-open → fail-closed treatment.
18. go-compression extraction: precondition 1 (refresh plan inventory post-`writerPool`), 2 (owner repo setup), 3 (phased execution).
19. Re-run architecture-review post-v1.1 (last run predates removals + XFP trust model).
20. v2.0 material: ValidateCSRF `*httptest.ResponseRecorder` return type (finding 1).
21. v2.0 material: `MiddlewareFunc` vs `Middleware` alias canonicalization (finding 7).
22. Route remaining review findings 8/9 into TODO_LIST/v2.0 buckets.
23. Refresh FEATURES PARTIALLY DONE line numbers (csrf.go refs likely drifted after the XFP changes).
24. Refresh `docs/architecture-reference.md` export tables for the v1.1.0 removals (doc-snippet-refs CI does not cover this file).
25. Adopt benchstat comparison records in the next `docs/benchmarks.md` baseline refresh (tool now pinned).
26. HARVEST both 2026-09-11 status reports' next-items into TODO_LIST/ROADMAP (docs-health).
27. Re-run the erraudit `--type-aware` sweep on v1.1.0 code; confirm the ~30 known-good advisories are unchanged.
28. Verify Dependabot's first grouped PR wave against v1.1.0.
29. nightly-fuzz: upload crasher corpus files as workflow artifacts (issue bodies currently only carry run-log pointers; the failing input lives in the job checkout).
30. Give the auto-filed fuzz issue a template (repro command, target, corpus path fields).
31. Add the `[Unreleased]` Fixed entry for the issue #9 fix when it lands.
32. Consider printable-alphabet seeds for string-status fuzz targets so strict oracles become assertable.
33. Codify the server_timing drift check into RELEASE.md pre-release verification.
34. Codify the build+run consumer probe (not just `go get`) into RELEASE.md step 15.
35. Check migration-guide link rot post-removals (doc-snippet-refs covers code fences, not markdown links).
36. Re-verify the AGENTS.md "0 active warnings" claim against the post-release tree (gates say green; keep the claim honest).
37. gopls `stdversion` × 6: revisit at Go 1.27 (DECISION_LOG row stands — no action until then).
38. Record today's red-tag-CI incident in DECISION_LOG with the new gate decision.
39. benchstat-verify the "195.6–220 ns/op is machine-state noise" claim at the next baseline refresh.
40. httpspec docs-site (TODO_LIST Low).
41. File the verified go-error-family upstream issue draft (drafted v1.0.0-era, never filed).
42. `withParsedTrustedProxies` doc gap (TODO_LIST Low).
43. Consider moving "CHANGELOG link check" out of the CI `Test` job into a docs job for clearer signal.
44. nightly-fuzz runtime: ~2h for 25 targets; consider a rotation subset for shorter cycles.
45. Bump actions/checkout + actions/setup-go past the Node 20 deprecation warnings seen in tonight's run.
46. Decide the release-commit mechanics (daemon won the race twice; tag identity may be enough — make it policy either way).
47. Remember the `[Unreleased]` Fixed bullet (link-def fix) retitles into the _next_ version section at that release.
48. After issue #9 lands, close it referencing the new corpus seed and the green nightly run.
49. Consider a `docs/status/` archival pass for fully-resolved reports (docs-health ANNOTATE cadence, monthly).
50. Keep `MiddlewareETag` seam documentation in sync if go-etag's API surface changes (Dependabot bumps will touch it).

## g) QUESTIONS (cannot answer myself)

1. **Tag integrity:** v1.1.0 permanently points at `ae0a46d`, whose CI failed on the docs-only link check (all code gates + the Release workflow passed; consumers verified by build+run). Options: (a) leave as-is and add a note to the GitHub release page, or (b) cut `v1.1.1` promptly to re-point the release at a fully green commit. Which do you want?
2. **Issue #9 fix shape:** oracle-only correction, or also harden the `writeHealthBody` discarded `MarshalWrite` error (health.go:83)? Both are non-API changes; oracle-only is the minimal correct fix, the hardening closes the silent-mask class permanently.
3. **Future nightly-fuzz crashers:** tonight's instruction cadence was report-then-wait, but nightly crashes would sit ~16h before the next session. Should future crashers be autonomously triaged-and-fixed (master-only, patch release only with your explicit go), or always report-and-wait?

---

_Report written per session instructions; awaiting owner decisions on g1–g3._
