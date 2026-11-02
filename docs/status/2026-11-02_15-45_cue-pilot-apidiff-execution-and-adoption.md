# Status Report — CUE pilot + apidiff gate: full plan execution, adopted and shipped

- **Written:** 2026-11-02 15:45 CET (clock via CLI)
- **Session scope:** executed the committed Pareto plan (`docs/planning/2026-11-02_05-05_pareto-master-plan_cue-pilot-apidiff.md`, M01–M22) end-to-end on top of the same-day research/adoption session's status report (`docs/status/2026-11-02_04-56_cue-research-and-adoption-decision.md`).
- **Verified:** every commit hash below exists on master (pushed `48b664e..3a3bb64`, `git status -sb` shows 0 unpushed); pre-sweep gate exit 0 before the push; full-package `golangci-lint run` 0 issues; `go test -race ./...` green after every code change; the negative test executed (`GATE_EXIT=1`, `- CORS: removed`); GitHub CI run **41464061041 on the final head completed SUCCESS (4m0s)** — which means the new `apidiff` AND `policy-vet` jobs passed on real GitHub runners, not just locally; the wave-1 vet commands were run live against three repos with outputs captured.
- **NOT verified / out of scope:** BuildFlow's full test suite end-state (the parallel writer's in-flight formatter work left 3 tests failing in their area at session end — attributed, not mine to fix); whether `~/.nix-profile`'s cue survives the next home-manager activation (untested deployment durability); the store buildflow binary still predates both new steps (deployment is SystemNix's layer); anything in repos I only vetted read-only.

---

## a) FULLY DONE

**Tier 1 — the apidiff gate (frozen-API promise → mechanical, both layers)**

1. **Feasibility spike (M01)** — apidiff facts verified from SOURCE, not docs: `-w` writes committed export-data snapshots (zip, self-describing module entry), module mode with `GOWORK=off` per module, internal packages filtered, and **exit code stays 0 on incompatible findings** (verified `EXIT=0` with a live detection) — so the gate must fail on the REPORT, not the exit code. gorelease surveyed as the alternative (exit-code gate, but proxy-dependent = network flake in the gate); decided apidiff + committed baselines, pinned `golang.org/x/exp/cmd/apidiff@v0.0.0-20261014143641-f0251eb4aa3e`.
2. **Baselines at the released tags (M02, `4cbb0c9`)** — generated from tag worktrees (root module incl. httpspec at v1.7.0, server_timing at v1.0.3), verified master compatible with both (post-tag commits are comment-only churn — apidiff was the ground truth), stored under `apidiff/` with a README carrying the regeneration ritual; global `*.zip` gitignore re-included deliberately for this directory.
3. **Gate + negative test (M03, `1a869e0`)** — shared `scripts/check-apidiff.sh` wired into prerelease gate **3b** and a new ci.yml `apidiff` job; negative test in a scratch worktree proved the SHIPPED gate fails on a renamed export (`GATE_EXIT=1`, report `- CORS: removed`); the full prerelease run passed gates 1–4b locally (step-5 lint timeout was self-inflicted CPU contention, re-verified implicitly by the later green suite).
4. **BuildFlow central step (M04, external `c8eff161f`)** — real Go provider (ToolApidiff), convention-based: any repo carrying `apidiff/*.zip` gets the gate; baseline zips are self-describing so mapping to workspace modules needs zero per-repo config; stale/unmatched baselines warn instead of failing; DataFlowEdges ledger bumped 359 → **364** after an edge probe proved all 5 new edges point INTO apidiff; run green on httputil (`✓ BuildFlow passed 1/1`).
5. **The promise says "enforced" (M05, `f9ae3b5`)** — `docs/v1-stability.md` gained the Mechanical Enforcement section (three layers, baseline convention, "cannot be waived for v1") and the stale pre-1.0 header now states the truth (active since v1.0).

**Tier 2 — the CUE pilot (evidence → ADOPT)**

6. **Packaging claim verified (M06)** — `nix eval nixpkgs#cue.version` = **0.17.1** (the previous session's one unverified claim is now evidence); installed via `nix profile` (home-manager did not conflict); binary reports `version (devel)` — the derivation version is the recorded pin.
7. **Policy repo scaffolded + published (M07, cue-policy `081e3dc`, public MIT)** — `github.com/LarsArtmann/cue-policy`, module `github.com/larsartmann/cue-policy` (lowercase: CUE rejects uppercase paths), README decides canonical-doc-home (schemas live ONLY there; consumer repos carry no copies); no split brain with `library-policy`/`go-policy-dsl` (different domain: library governance vs workflow schemas).
8. **`#ReleaseWorkflow` authored, caught a REAL defect first run (M08, `48b664e`)** — `concurrency: field is required but not present`: httputil's release.yml had NO guard against a racing second tag push → fixed same-session (`release-${{ github.ref }}`, `cancel-in-progress: false`), then vet green.
9. **actionlint sweep caught the second real defect** — ci.yml's commit-lint job spliced `github.head_ref` (attacker-controlled on fork PRs) directly into an inline shell command → fixed via env indirection (actionlint's documented mitigation), script contract verified unchanged; actionlint then ALL GREEN.
10. **Delta matrix + adoption memo: ADOPT (M09/M10, cue-policy `b00ef1f`)** — the two tools are complementary with zero overlap (each real finding had exactly ONE detector); CUE pins policy classes actionlint has no concept of; `on:` YAML-key handling verified correct (no YAML 1.1 bool mangling); negative control proves the schema discriminates workflow types.
11. **`#CIWorkflow` (M12, cue-policy `90c4183`)** — vets ci.yml + nightly-fuzz.yml green; concurrency deliberately NOT required for CI (would be new fleet policy, and policy schemas only encode decided rules).

**Tier 3 — wiring + records**

12. **BuildFlow `cue-vet` step (M11, external `f263829b7` + path fix `2d6bbee17`)** — schemas resolved via the fleet sibling checkout (`../cue-policy` default, `tool_options: cue-vet: schemas_dir` override); cue/checkout absence = clean no-op (per-repo CI owns gating); best-effort detect-only; DataFlowEdges 364 → **366** (treefmt + github-actions-pinning point INTO cue-vet — the vet reads SHA-pinned files); live e2e test against the real policy checkout; first REAL run exposed the absolute-path bug (bogus findings) — fixed with `filepath.Rel`, cache-replay documented; final run `✓ BuildFlow passed 1/1`.
13. **Per-repo `policy-vet` CI job (M11, `290c0a2`)** — installs pinned cue v0.17.1 (`go install cuelang.org/go/cmd/cue@v0.17.1`, verified locally), checks out cue-policy at a pinned SHA, vets all three workflows; the exact command shape simulated green in a CI-layout sandbox BEFORE landing.
14. **HARVEST (M13, `52ece58`)** — TODO_LIST gained the genuinely-remaining rows (rollout completion, buildflow redeploy, SHA-pin bumping duty); CHANGELOG `[Unreleased]` documents the apidiff gate, policy-vet job, and both workflow fixes; changelog links check green.
15. **Decision records + durable lessons (M14, `5ac7bc8`)** — two DECISION_LOG rows (never a Go dependency; adopted as the external workflow-policy tool, both layers) + 8 research lessons distilled with fleet applications (`docs/research/2026-11-02_cue-engineering-lessons.md`).

**Tier 4 — code, debts, research, hygiene**

16. **CSP conflict detector (M16, `3534246`)** — request-time detection in `Nonce`: a CSP already written by an outer middleware now logs `security.csp_conflict` (served behavior UNCHANGED — the nonce CSP wins; limitation documented: only the outer-writes-first order is visible); full adding-a-code sweep done (template map, code registry, AGENTS Nonce bullet, FEATURES row, architecture-reference classification table); slog-swap tests with the `//nolint:paralleltest` convention; lint 0 issues; full race suite green.
17. **WithContext provenance audit (M17, `bc04305`)** — all 23 context call sites + 60 message templates audited; 12 of 13 heuristic-flagged entries audited CLEAN (request-data/runtime-lifecycle, correctly not config errors); one real gap fixed: `ratelimit.trusted_proxy_cidr_invalid` now names `KeyedRateLimiterConfig.TrustedProxiesCIDR`.
18. **Verification debts (M18, `5c50829`)** — fleet census stored (348 directories, point-in-time baseline file); a-team-openapi-spec inspected: Bun/TS HAR→OpenAPI capture pipeline, single consumer, spec is captured OUTPUT not authored schema → **CUE-OpenAPI lane NOT NEEDED**, closed with reasoning and a re-open trigger.
19. **CUE research round 2 (M19, `b24f693`)** — the-logic-of-cue read in full (backwards-compat = subsumption — the apidiff gate's theorem; order-independent defaults; optional-field logic; cycles-as-equations) + CUE's own practices (issue-first design gate, Unity canary program, action-shaped triage labels) → 8 more durable lessons.
20. **Watchdogs + integrations (M20, `6db3763` + cue-policy `1ea1c3d`)** — AGENTS Commands block gained the verified-live apidiff + cue-vet invocations (relative paths — cue rejects non-canonical absolutes, gotcha recorded); cue-policy README gained the two-pin watching protocol; cue-policy registered in the meta inventory (`lib, policy`, importance 45).
21. **Parked evals resolved in writing (M21, `a97d871`)** — httpspec strict-set stays DEMAND_GATED (explicit opt-in already covers it; a closed mode would touch frozen signatures with zero importer pull); website-launch provenance recorded for the next launch session behind its owner gate.
22. **Status report + plan annotated (M22, `2488424` + `8a147eb`)** — every (f) item carries its evidence hash in an execution appendix (appendix form chosen over in-table strikethrough: the `| N |`-table annotation has the known SKILLS#3 mangling defect); sections (a)–(d) re-baselined; the plan's honest deviations recorded (tag drift to v1.7.0/v1.0.3, the config-step assumption, no filings needed, profile install).

**Rollout wave 1 (M15, read-only portion)**

23. **Three repos vetted live (cue-policy `a3e9c79` + `7ecfcaf`)** — library-policy, go-etag, go-error-family: after generalizing the schemas (fleet-legal constructs the pilot schemas had never seen: manual dispatch + inputs, path filters, `pull_request:` null, extra env, `id-token`, glob-style tags — schema closedness, NOT repo violations; env pin triad became format-checks-when-present), **all fleet CI workflows PASS** and the single remaining finding class is real and consistent: **all three repos' release.yml lack the concurrency guard** — the identical defect CUE caught in httputil. Findings triaged in the wave report; per-repo remediation harvested (per-repo AGENTS constraints are not mine to bypass).
24. **Policy pin bumped deliberately (`3a3bb64`)** — httputil's policy-vet job follows the schema changes (`90c4183` → `7ecfcaf`), verified green against the new head before bumping.

**Session tail**

25. **Pushed and CI-green on the exact head** — pre-sweep exit 0 → push `48b664e..3a3bb64` → CI run 41464061041 **completed success (4m0s)**, covering both new gates on GitHub runners; tree clean except the pre-existing, never-mine `flake.lock` bump (untouched all session, stash round-tripped for the clean-tree prerelease gate).

## b) PARTIALLY DONE

1. **Rollout wave 1 (M15)** — vet + triage done read-only; the per-repo half (concurrency blocks, `policy-vet` CI jobs, 30-day review date) belongs to each repo's own session; TODO_LIST row updated to the remaining scope.
2. **BuildFlow deployment** — both steps exist in source and run green via a source-built binary, but the `~/.nix-profile/bin/buildflow` store binary predates them; the delivering layer is a SystemNix activation. TODO_LIST row tracks it.
3. **BuildFlow repository state** — 5 commits local (not pushed; its push policy is config-controlled and I did not have the mandate); a parallel writer's in-flight formatter/doctor refactor has 3 failing formatter tests + 2 pre-commit-audit failures in THEIR files at session end — correctly attributed and left untouched.
4. **The apidiff floor** — still equals v1.7.0/v1.0.3 by design; regeneration at the NEXT release commit is the ritual's first live exercise and has not happened yet (no release this session).
5. **The two new CI gates** — verified by one green full-workflow run on GitHub; a deliberately-breaking PR proving the CI-side negative case (the local negative test passed) has not been run.

## c) NOT STARTED

1. Per-repo remediation in the three wave repos (concurrency guard + policy-vet job + 30-day review date).
2. The 30-day signal/noise review for cue-vet + apidiff fleet adoption (re-decide triggers are in the adoption memo).
3. CUE v0.18 re-check + nixpkgs pin re-check (protocols written in cue-policy README; nothing to run today).
4. `#Changelog` schema (compare-link/[Unreleased] invariants) — named in a schema comment, not built.
5. website-launch provenance implementation — owner-gated, next launch session.
6. apidiff baseline adoption in OTHER fleet libraries (go-etag, go-error-family) — only httputil carries `apidiff/` today.
7. The `docs/reviews/*.html` self-review series entry (owner ruling Q3 made it a separate series; today's report is the `.md` status; the HTML series has no first entry yet).
8. Wave-2 repos (go-taskqueue, go-cqrs-lite, cmdguard, …) not yet vetted.
9. Building the cue pin into the fleet SystemNix config for activation-durable install (current install is a user-profile add).
10. Meta inventory entries for repos other than cue-policy (untouched this session).

## d) TOTALLY FUCKED UP

Nothing shipped is broken — but the process had real failures, named precisely:

1. **I filled the 48 GB /tmp tmpfs with my own caches.** An 826 MB GOCACHE plus Go's WORK directories caused cascading `no space left on device` failures (cmdguard sibling compile check, BuildFlow link step). Fixed by moving GOCACHE/TMPDIR to the home mount — but this was self-inflicted and cost ~15 minutes. The machine's tmpfs was already ~97% full from other sessions' artifacts (crush-daily-snapshot 4.8G, tq-battery 4.2G); I correctly did NOT touch those, but I should have checked `df` BEFORE choosing cache locations.
2. **The plan itself carried a wrong assumption and I executed into it.** M04 said "draft central step config" — BuildFlow HAS no config-defined external steps (the plugin system was deliberately deleted, documented). The plan's own d1 "verify the model first" caught it before code, so the cost was re-planning, not breakage — but the research session recommended a BuildFlow step twice without ever checking this (the previous report's own self-review had already flagged it; it was still not checked until execution).
3. **The cue-vet provider shipped a bug that the first REAL run caught:** the absolutized schema-dir argument made cue reject every invocation ("non-canonical import path"), turning conformant workflows into 3 bogus error findings — and buildflow's result cache replayed the bogus findings, masking the fix until purged. Fixed (`2d6bbee17`, `filepath.Rel`) and green-since, but the unit tests passed while the integration path was broken — the e2e-on-real-repo step is what saved it.
4. **The documented cache-purge command is stale and I burned time on it:** AGENTS' `DELETE FROM result_cache … buildflow.db` hits a nonexistent table — the real store is `cache.db` (my first purge even created a stray 0-byte `buildflow.db`). This is a pre-existing doc bug now confirmed by use; fixing the doc is queued (section f).
5. **Interrupted background commits left a stale `.git/index.lock` in BuildFlow** — I removed it only after verifying no live commit process existed. Additionally, my committed hash there was REBASED mid-session (`3526ed8ed` → `c8eff161f`) by fleet tooling — a live demonstration that hashes in parallel-writer repos are not stable identifiers.
6. **Compiler-caught sloppiness, twice:** invented non-existent helper functions in test files (edge-probe API mismatch; invented `execLookPathCueVet`/`fleetCuePolicyDir`/gofinding helpers), an import edit that applied late and confused the sequence, one edit that deleted a doc-comment fragment (caught and repaired), a python replace that silently no-op'd on escaping (caught by re-running the vet). Every one was caught by a gate before landing — but each cost a cycle that earlier compilation would have saved.
7. **Self-inflicted lint timeout:** the first full prerelease run failed at step 5 because I ran BuildFlow's 237-second test suite concurrently on the same machine. Sequencing heavy jobs is my responsibility.

Not fucked up, worth recording as verified-clean: no Go API changes beyond the additive detector; no red master at any point I controlled; parallel writers' staged work (formatter/doctor files, AGENTS/CHANGELOG/TODO in BuildFlow) untouched; the cmdguard sibling's broken mid-refactor state routed around (`GOWORK=off` + published versions) rather than "fixed".

## e) WHAT WE SHOULD IMPROVE

1. **Environment preflight before heavy builds:** `df -h /tmp` and a home-mount default for GOCACHE/TMPDIR on this host would have saved the tmpfs cascade; make it a habit step for any Go build outside a repo's standard flow.
2. **Fix the stale cache-purge doc:** AGENTS.md still documents `buildflow.db`; the real path is `~/.cache/buildflow/cache.db`, table `result_cache`. Confirmed stale by use — a two-line fix worth doing on sight (see section f).
3. **Compile early, compile often when writing new-API code:** every invented-helper incident would have died at a 2-second `go vet` instead of surfacing in a 4-minute test run. The BuildFlow suite is slow enough that per-file vetting pays.
4. **Draft policy schemas against ≥3 diverse repos, not one.** The closedness failures (dispatch, path filters, null triggers, extra env) were all present in wave-1 repos from the start; the pilot's single-repo schema encoded httputil specifics as fleet law. First-vet-against-3 is the cheap version of wave-1.
5. **Scope cross-repo work to read-only until each repo's constraints are read.** The wave protocol worked; formalize it: vet everywhere, edit nowhere without that repo's session.
6. **Never trust hashes across parallel-writer sessions** (BuildFlow rebased under me mid-session); cite messages + file lists, and re-check `git log` before amending.
7. **Run commits with hooks in the foreground or budget for lock cleanup** — two interrupted background commits cost a stale-lock round trip.
8. **Sequence machine-heavy jobs** (full lint + big test suites) — the prerelease lint timeout was pure contention.
9. **Result-cache keys should include provider/tool version** (BuildFlow-side improvement worth proposing upstream of the fleet): the cue-vet path-fix was masked by replayed findings keyed only on tracked files.
10. **Prefer `write` over heredoc/python replacement for whole-section rewrites** — both silent-replace failures this session were shell-quoting escapes; the write tool has no such failure mode.

## f) Up to 50 things we should get done next

*Brainstorm ranked by value/effort; harvested candidates should survive docs-health routing rigor. "owner" = gated on you.*

1. Add the concurrency guard + `policy-vet` job to **library-policy** (template: its own wave report) and set the 30-day review date.
2. Same for **go-etag**.
3. Same for **go-error-family** (release.yml only; its other workflows already pass).
4. Redeploy the **buildflow binary** (SystemNix activation) and verify both new steps via `buildflow -s apidiff` / `-s cue-vet` on httputil.
5. Cut **httputil v1.8.0**: freeze `[Unreleased]`, regenerate both apidiff baselines in the release commit (first live exercise of the ritual), one-push master+tags.
6. After the tag: confirm the regenerated baselines gate green and pkg.go.dev renders the enforcement docs.
7. Push **BuildFlow's 5 local commits** (check its auto_push config first — owner input wanted, see g2).
8. Adopt `apidiff/` baselines in **go-etag** (fleet lib with released tags).
9. Adopt baselines in **go-error-family**.
10. Decide + adopt baselines for **library-policy** (tool repo — semver discipline may differ).
11. Fix AGENTS.md's **stale buildflow cache-purge command** (`buildflow.db` → `cache.db`, `result_cache`) — two-line doc fix, confirmed stale this session.
12. Sweep OTHER fleet repos' docs for the same stale purge command (the bug is fleet-shipped).
13. Move the **cue pin into SystemNix** so home-manager activation can't drop the nix-profile install (durability unknown — see g-question).
14. Watch **CI run 41464061041's siblings**: nightly-fuzz next run (schedule unaffected, zero expected impact — cheap confirmation).
15. **Wave 2**: vet go-taskqueue, go-cqrs-lite, cmdguard workflows (read-only, wave protocol).
16. Run the **CI-side negative test**: a scratch PR renaming an exported symbol must fail the `apidiff` job on GitHub (local proof exists; CI proof pending).
17. Author the **`#Changelog` schema** (compare-link definitions + `[Unreleased]:` retarget — the v1.1.0 red-release class).
18. Consider **gorelease as a prerelease cross-check** (exit-code semantics complement; research first).
19. **BuildFlow upstream proposal**: include tool/provider version in result-cache keys (the replay-masked-fix pain).
20. **BuildFlow docs**: document the `cue-vet` step + `schemas_dir` option (their CHANGELOG is currently a parallel writer's — coordinate).
21. **BuildFlow docs**: add apidiff step to the tools list/README if one exists.
22. Resolve the **BuildFlow pre-commit audit failures** (sync.Once allowlist: `binary_hash_memo.go` missing entry, `result_cache_3.go` stale) — parallel writer's files; coordinate or wait for their landing.
23. **doctor_survey.go 372-line ceiling** violation — same owner (parallel writer).
24. Re-run BuildFlow's **formatter test trio** after the parallel writer lands their refactor (3 failures attributed, unresolved at session end).
25. Register **remaining first-party repos** in the meta inventory (only cue-policy was added this session).
26. Decide a **census refresh cadence** for the stored 348-dir baseline (quarterly? per-fleet-audit?).
27. Add a **server_timing/ README line** noting its apidiff baseline (module-local discoverability).
28. Cross-link the CSP-conflict warning from `NonceConfig.CSPBuilder`'s field doc (one-line doc patch, teaches at the config site).
29. Consider a **fuzz target for the apidiff zip module-path reader** (BuildFlow-side; malformed zips already take the warn path — low priority).
30. Evaluate **joining CUE's Unity program** (upstream compat canary) once ≥2 fleet repos vet their workflows in CI.
31. Verify **cue survives home-manager activation** (empirical: trigger an activation in a controlled moment; informs item 13).
32. Add **`GO_VERSION`/`GOLANGCI_LINT_VERSION` presence check to RELEASE.md's gate 6.6** ritual text (the schema now enforces shape; the doc should name it).
33. Record in RELEASE.md step 12 that **policy-vet/apidiff run on tag pushes too** (they do — both jobs trigger on push events).
34. Consider **tagging cue-policy** (v0.1.0) so non-SHA consumers have a floating ref (SHA pinning stays the fleet default).
35. **First wave-1 repo fix** doubles as the template-validation run for items 1–3 — do one repo end-to-end before batching.
36. Re-check **`~/go/bin/cue` vs `~/.nix-profile/bin/cue` divergence** (two installs, same version — document which is canonical for interactive use).
37. Extend the **AGENTS.md erraudit-block pattern** with a one-line pointer to the new apidiff/policy-vet commands (discoverability for fresh sessions).
38. Audit whether any **fleet repo's workflows use `pull_request: null` shape** beyond go-etag (schema now handles it; census of the pattern validates the generalization).
39. Propose **`cancel-in-progress` semantics review** for fleet CI workflows (currently only releases have concurrency; PR churn cancellation is uncoded fleet policy — new-policy decision, owner-gated).
40. Verify **docs/v1-stability.md's new links** resolve in the GitHub-rendered view (relative paths from docs/ — locally consistent, GitHub rendering unconfirmed).
41. Decide whether **README.md should mention the mechanical API gate** (user-facing honesty vs maintainer-scope noise — owner call).
42. Start the **`docs/reviews/*.html` self-review series** (owner ruling Q3 — series has no first entry; this session is a candidate subject).
43. After BuildFlow lands their formatter work, **re-run my two suites' cross-check** (apidiff + cue-vet providers) to confirm no regression from their rebase.
44. **Pin-watch drill**: simulate a nixpkgs cue bump against the schemas (the protocol exists; never executed).
45. Add **cue-policy CI** (self-check `cue vet -c .` + fmt on push — the policy repo itself has no CI yet).
46. **Nightly rotation**: consider adding a weekly fleet-wide `cue vet` sweep job to catch schema/repo drift without waiting for per-repo CI (BuildFlow step already covers local runs).
47. Sweep **README/architecture-reference for "promises" language** that should now say "enforced" (v1-stability was the canonical spot; stragglers possible).
48. Check **dependabot/gomod coverage for `cue-policy`** (not a Go module — should be excluded from dependabot gomod ecosystems to avoid module-inventory findings).
49. **meta inventory**: set importance for httputil's new sub-surfaces (`apidiff/` dir? no — dirs aren't projects; skip) — instead: verify `meta list` picks up cue-policy correctly (registration smoke).
50. **Archive-sweep candidates**: when the 30-day review lands, the wave-1 report and delta matrix become archive candidates per docs-health conventions (not now — they're referenced).

## g) Questions I cannot figure out myself

1. **Wave-1 remediation authorization & shape:** may I (or a dispatched per-repo session) apply the concurrency-guard + policy-vet changes directly in library-policy, go-etag, and go-error-family now, or do you want each repo handled under its own session/queue after its AGENTS constraints are read? Batching is faster; per-repo is safer — your call decides wave completion timing.
2. **BuildFlow push policy intent:** its config-controlled auto-push was off/uncertain this session and my 5 commits (apidiff step, cue-vet step, path fix) sit local while a parallel writer has in-flight work that currently fails the repo's own audit gates. Do you want those commits pushed now (accepting that the remote tree inherits their in-flight audit failures), or held until their formatter/doctor work lands?
3. **Release timing for v1.8.0:** the `[Unreleased]` section now carries the apidiff gate, policy-vet job, CSP detector, and both workflow fixes. Cut v1.8.0 once you're ready (baselines regenerate in the release commit per the ritual), or keep accumulating? If cut, should it be the coordinated single-push (RELEASE.md step 12) straight after CI green on `3a3bb64`, or wait for anything else you have queued?

---

*Point-in-time snapshot. When stale, annotate (docs-health ANNOTATE); harvest open items into TODO_LIST rather than re-planning from scratch.*
