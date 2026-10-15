# Status Report — Completion Tier T20/T21 Cross-Repo Batch, Sentinel-Revert Recovery, and Two Self-Inflicted Wounds

**Session start:** ~18:20 (resumed from `2026-10-15_18-10_completion-tier-t11-t27-lock-forensics-and-two-phantom-tasks.md` with standing instruction WAIT, overridden by the owner's execute instruction)
**Session end:** 2026-10-15 19:08 CEST
**Scope:** Finish pareto T20 (go-etag hygiene batch) and T21 (BuildFlow batch), fix the T15 harvest-ledger forward references, consolidate + push + CI-watch, refresh the bench baseline. Per docs-health convention, this report is a point-in-time snapshot; the living backlog is TODO_LIST.md.

**Tail state at writing:** httputil master pushed through `c3caa6c`, CI green (run 38773512473), tree clean, `nix flake check` all-checks-passed on `84fbdba`. go-etag master pushed through `ff2cf28`, CI green (run 38772001561), tree clean. BuildFlow tree clean (go.mod casualty restored, no commits made). SKILLS untouched (2 pre-existing unpushed draft commits remain).

---

## Session narrative (short)

The handoff's "5+ unpushed commits" had grown to 7, and two were NOT benign: `8302e66` had typed 12 error sentinels as the `error` interface, breaking `go vet`/test compilation (`example_test.go:859: ErrCSRFInvalid.WithContext undefined`) — the exact trap AGENTS.md's do-NOT-migrate policy names, chasing advisory-only `sentinel_concrete_type` findings. Reverted the 9 Go files (kept the flake.lock refresh), verified build/vet/race/lint/both-real-erraudit-gates green, committed deliberately (`28681f2` after consolidation).

T20 landed in go-etag: the `doc-snippet-refs` checker ported (multi-module source-importer resolution worked first try; failure path mutation-tested: fake symbol → exit 1), wired into CI as a new `doc-snippets` job; DOMAIN_LANGUAGE glossary rows ("Hit ratio", "Adopted tag"); the no-fuzz decision for `metrics/` (pure atomic counters, no oracle); a dated drift banner on the pre-split 2026-08-07 `review-and-roadmap.md` (OQ3 answered by the split, OQ1 evolved via the client module); the local `-count=6` metrics bench artifact (hooks add ~0.5–1.5 µs/op, zero extra allocs — both paths 1224 B/op / 14 allocs/op). KeyHolderAI's "stale require" finding was already superseded: go.mod:93–94 carry `entitytag`/`server v0.6.1 // indirect` (the v0.6 sweep dropped the root require).

T21 resolved as verdicts instead of an issue: (1) the 11 failed 2026-09-23 format steps are **unrecoverable** — BuildFlow's state DB `step_timings` starts 2026-10-06; (2) the downgrade issue is **obsolete** — the strip became fleet policy on 2026-10-03 (go-etag v0.6.1 de-poisoning cascade) and the `honor-the-pin` opt-out shipped 2026-10-14 (go-version-auto-configure `ed93855`); (3) "BuildFlow pins lint versions fleet-wide" is **false** — BuildFlow's CI installs `@latest`, its flake has no golangci input; the real stale fleet pin was **go-etag's `ci.yml` v2.13.2**, and go-etag's remote master was RED for exactly the predicted reason (`export data version 5 is greater than maximum supported version 4`, run 38658774560). Bumped to v2.14.0 (verified all five modules lint-clean locally under the v2.14.0 binary first) — go-etag master went green. Also restored BuildFlow's dirty `go.mod` (internal requires zeroed to `v0.0.0-00010101000000-000000000000` — the mid-cycle `go mod tidy` casualty class; zero pseudo-versions are never valid).

T15 ledger rows for T20/T21 rewritten from premature "executed" to Closed with real evidence. TODO rows 21/36/39 struck with verdicts; the unscoped row-36 residue kept actionable as a NEW open row; row 37 (server_timing + limiter examples) verified all-three-exist and struck. AGENTS.md gained two BuildFlow bullets (downgrade-is-policy + no-lint-pin fact + the zero-pseudo-version casualty pattern). Bench baseline refreshed (`nix run .#bench`, 967 s) and committed with the new `BenchmarkLanguageStaticRequest` rows (584 B/op, 8 allocs/op, stable ×5).

---

## a) FULLY DONE

| # | Item | Evidence |
| - | ---- | -------- |
| a1 | Build-breaking sentinel annotations reverted; all gates re-verified (build, vet, `-race`, golangci 0 issues, erraudit `legacy_as` + `stdlib_constructor` both exit 0) | httputil `28681f2`; pushed in `8302e66..84fbdba` |
| a2 | Broken remote master repaired — the annotations sat pushed on origin (CI run 38768572045 failure) until my revert landed; CI green on `84fbdba` (38771948814) and `c3caa6c` (38773512473) | gh run list, both success |
| a3 | doc-snippet-refs ported to go-etag (4-alias checkedPackages over the five-module surface, source importer resolves cross-module); green on README + docs/*.md; mutation test: fake symbol → exit 1 | go-etag `7f39f78`; CI job "Doc snippet symbols" ✓ 24 s in run 38772001561 |
| a4 | go-etag CI lint pin v2.13.2 → v2.14.0; root-caused the pre-existing red master to the exact predicted export-data break; post-push run green (Lint ✓ 1 m49 s) | go-etag `ff2cf28`, run 38772001561 |
| a5 | Glossary rows + no-fuzz decision + drift banner + CHANGELOG [Unreleased] entries in go-etag; metrics `-count=6` artifact saved locally per the untracked-bench-artifact convention (only `reports/dupl/` is negation-tracked) | go-etag `7f39f78`; `reports/bench/2026-10-15_metrics-hook-overhead-count6.txt` |
| a6 | KeyHolderAI f49 verdict: stale root require already gone (v0.6 sweep), `entitytag`/`server v0.6.1 // indirect` at go.mod:93–94 = matches latest; no action needed | read of KeyHolderAI go.mod |
| a7 | T15 harvest ledger: both forward-referencing rows now Closed with dated evidence; trailing count fixed ("four" → five tasks) | `docs/status/2026-10-15_18-05_...ledger...md`, committed in `2a84dc0`/`84fbdba` |
| a8 | TODO_LIST rows 21/36/39 struck with verdicts; row-36 residue (repair-dry-by-default, module-inventory, mutation-verify) preserved as a NEW open row; row 37 struck after verifying all three examples exist in-tree (`ExampleServerTiming_MeasureWithDesc`, `ExampleServerTimingFromContext`, `ExampleKeyedRateLimiter_evictionTTL`) | TODO_LIST, in `84fbdba` |
| a9 | AGENTS.md memory: BuildFlow downgrade-is-policy bullet (with the `honor-the-pin` pointer and the zero-pseudo-version casualty pattern) + no-lint-pin fact bullet | AGENTS.md BuildFlow Pipeline section, in `84fbdba` |
| a10 | BuildFlow go.mod casualty restored (zero pseudo-version requires → HEAD state); `go build ./...` green after restore | working-tree restore; BuildFlow tree clean at 19:08 |
| a11 | Bench baseline refreshed (full 3 s×5, 967 s, exit 0) and committed: 265 bench lines vs 260, the delta is `BenchmarkLanguageStaticRequest` ×5 (584 B/op, 8 allocs/op, ns/op 625–730) | httputil `c3caa6c`, pushed, CI green |
| a12 | `nix flake check` passed on the pushed head `84fbdba` (all checks) | session log |
| a13 | T21 enumeration attempt executed honestly: `buildflow format --dry-run --verbose` (60 success / 0 failed, 19.6 s, tree verified unchanged, snapshot diff'd), source inspection of format mode (= fast build-mode + AutoFix), state-DB retention check | session log; /tmp snapshot diff clean |

## b) PARTIALLY DONE

| # | Item | What's missing |
| - | ---- | -------------- |
| b1 | go-etag doc-snippet gate: wired into **ci.yml** only — httputil runs the checker in ci.yml AND release.yml; go-etag's release workflow and `scripts/pre-release-check.sh` do not run it | release-gate wiring |
| b2 | benchstat sanity check on the refreshed artifact: I invoked benchstat but piped stderr to /dev/null and got only the header — **no actual old-vs-new comparison was produced**; the artifact is committed unverified beyond eyeballing | real comparison (see d3) |
| b3 | BuildFlow binary freshness: `buildflow --version` = `518739c-dirty` vs HEAD `d114f28ce` — mismatch documented, **rebuild/redeploy NOT done** (I chose not to touch BuildFlow's flake while a parallel writer's dirty state was just repaired there) | `nix build` + redeploy + freshness verify |
| b4 | KeyHolderAI verdict is a survey (read go.mod), not an execution — `go mod tidy` was never run there to *prove* the file is tidy; the indirect requires at :93–94 could in principle still be unsorted/stale in other ways | run tidy + `git diff` empty check |
| b5 | Fleet lint-pin sweep: go-etag bumped; the T26 consumer scan counted 46→65 consumers and I did NOT sweep any other fleet repo's CI for v2.13.2 pins — each remaining one is a future red master of the go-etag kind | per-repo CI pin audit/bump |
| b6 | The 2026-10-15 16:18 buildflow dev run (22 failed steps, run 20261015-161830) was noticed in history but NOT decomposed row-by-row against the AGENTS policy-rejected residual classes — I assumed the documented composition without verifying | row-level triage per the branching-flow delta-check discipline |
| b7 | `honor-the-pin` recorded as an opt-out pointer, but the actual mechanism (flag? env? file?) was never verified end-to-end — if this repo ever needs a patch floor, the how-to is not in AGENTS | read go-version-auto-configure docs, record the concrete knob |

## c) NOT STARTED (in scope of the handoff's orbit, consciously untouched)

| # | Item | Why |
| - | ---- | --- |
| c1 | SKILLS repo: 2 unpushed draft commits | another session's drafts; pushing them publishes work I cannot vouch for (annotate-tool defects #3/#4 are filed, drafts live there) |
| c2 | TODO row 38 REMAINING: owner ruling on bulk-striking the 24 KEEP files | owner-gated (decision-brief row T6) |
| c3 | TODO row 29: MD060 table-style ruling + buildflow result-cache purge | owner style ruling |
| c4 | TODO rows 33–35 consumer-audit backlog | read-only pending R10; GmbH pre-staged 2026-10-15 (prior session) |
| c5 | T23/T24/T25 owner-ruling notes (shutdown-drain, diagram embed, v2.0 ledger proposals) — all written 2026-10-15 prior session, all awaiting owner | owner-gated |
| c6 | go-etag TODO_LIST rows 1–7 (consumer follow-ups incl. go-github-kit nix lane, fuzz expansion, erraudit-in-CI posture, release.yml dry-run mode, docs tier, process micro-policies, epics) | out of today's scope |
| c7 | Next httputil version tag / docs-health pass over the now-considerable [Unreleased] section | release = owner-called |

## d) TOTALLY FUCKED UP

| # | Incident | Damage | Root cause |
| - | -------- | ------ | ---------- |
| d1 | **Self-inflicted CHANGELOG damage, briefly pushed.** My two-part multiedit had a malformed `old_string` (an adjacency that never existed in the file); the tool applied the connector-strip edit and failed the insertion, and I then misread "Applied 1 of 2" as the insertion having landed. Result: the csrf correction-of-record bullet lost its `(`csrf.go`):` connector, the damage was swept by the daemon into `2a84dc0` **and pushed to origin** before I diagnosed it; repaired in `84fbdba` | transient corruption on remote master (~20 min), zero functional impact, history carries the bad intermediate | I constructed a too-clever two-edit batch instead of two verified single edits; I did not re-read the file after the ambiguous "1 of 2" report before moving on |
| d2 | **Lost the deliberate-commit race three times.** In go-etag (twice) and httputil (once) my explicit `git commit` found "nothing to commit" because the daemon had swept my verified edits into heuristic commits minutes earlier; each loss cost a soft-reset consolidation round | no lost content (all consolidated deliberately), but generic heuristic commits briefly held the work and one range (`f407195`/`f31371b`) was pushed by the parallel actor before consolidation | I batched multiple doc edits across minutes instead of committing each verified unit immediately — the go-etag AGENTS literally documents "commit immediately … so the auto-daemon doesn't capture the diff" |
| d3 | **Gates after push.** I pushed httputil `84fbdba` and only then ran `nix flake check` (passed — luck, not discipline). Same class: the benchstat "check" I cited was actually an eaten-stderr no-op | risk of a red master on doc-gate findings; a false "verified" claim in my own narration | sequencing error + `2>/dev/null` on a verification command — the exact "filters that eat errors" anti-pattern AGENTS bans |
| d4 | **Pushed two foreign daemon commits unreviewed.** go-etag `1968247` (go.mod/go.sum, 01:49 today) and `2d2ae74` (SECURITY.md, yesterday) sat unpushed from a prior session; pushing my work necessarily published them. I checked file lists only, never read the diffs | two heuristic commits with unreviewed content are now on origin master | consolidation protocol allowed pushing past them (not verifiably mine = cannot soft-reset over them), but the diff-read step I skipped is what "verify" should have meant |

## e) WHAT WE SHOULD IMPROVE

1. **Edit-tool hygiene:** no more multi-part edits on load-bearing docs; one edit, re-read, verify, next. The "Applied N of M" response must always trigger an immediate re-read.
2. **Commit-per-verified-unit in daemon repos:** the consolidation round-trip is more expensive than committing early; treat any >2-minute edit window in a daemon repo as sweep-bait.
3. **Verification commands must never swallow stderr** (`2>/dev/null` on benchstat produced a fake check). Redirect to a file and read `$?`, per the session-tail discipline already in AGENTS.
4. **Gate-before-push:** `nix flake check` and the markdown gates belong before `git push`, not after; luck is not a pipeline.
5. **When a checker is the deliverable, prove it can fail in the same session AND wire it into every gate surface the sibling repo uses** (ci + release + pre-release script), not just the first one.
6. **Binary-freshness checks should trigger remediation in the same session** (`buildflow doctor`-style), or be explicitly deferred with a reason — b3 sat unnoticed-in-plain-sight until this report.
7. **"Not mine" is not "don't read":** foreign unpushed commits under mine deserve a diff read before I make pushing them unavoidable.
8. **Assumed-residual counts (b6) should be decomposed the way branching-flow taught:** row-level census, then verdicts — "the number matches the documented class composition" is not evidence.

## f) NEXT (up to 50, ordered by impact)

**Direct fallout of this session (highest impact):**
1. Reconcile `docs/benchmarks.md:62` with the committed artifact — table says ~1210 ns/op for `BenchmarkLanguageStaticRequest`, the refreshed artifact says 625–730 ns/op (T18's harness revision postdates the table row; ~2× disagreement needs a verdict: update row or explain the delta).
2. Run the real benchstat old-vs-new over the bench artifact rotation (and fix d3's pattern: file + `$?`, no stderr-eating).
3. Rebuild + redeploy the BuildFlow binary (b3: `518739c-dirty` vs `d114f28ce`), then `buildflow --version` freshness verify.
4. Sweep fleet repos' CI for golangci-lint v2.13.2 pins and bump to v2.14.0 (go-etag proved the break is real and live) — start with the T26 consumer-scan list.
5. Wire the doc-snippet checker into go-etag's release gate (`release.yml` + `scripts/pre-release-check.sh`), mirroring httputil's two-surface wiring.
6. Decompose the 16:18 buildflow dev run's 22 failed steps against the documented residual classes (row-level, then verdicts).
7. Verify and document the concrete `honor-the-pin` mechanism (flag/env/file) in AGENTS so a future patch-floor need is one lookup, not archaeology.
8. Confirm KeyHolderAI is actually tidy: run `go mod tidy`, expect empty diff; close f49 with execution evidence.
9. Read go-etag `1968247` (go.mod/go.sum) and `2d2ae74` (SECURITY.md) diffs post-hoc — what did I publish blind?
10. Annotate this report's resolved items as they close (docs-health cadence).

**BuildFlow thread (from T21 verdicts):**
11. Repair-steps-dry-by-default proposal (row-36 residue; f46's dev-run mutation evidence).
12. Module-inventory consistency check (row-36 residue).
13. Mutation-verify convention into BuildFlow AGENTS (row-36 residue).
14. Consider a buildflow `history` retention bump or export (the 11-step enumeration died with the 30-day DB; post-mortems need longer memory).
15. Decide whether BuildFlow should pin a fleet lint version after all (the no-pin fact cuts both ways: `@latest` CI also means silent future breakage).

**httputil backlog items touched-but-open this session:**
16. Architecture-review re-run post-v1.x (TODO row 7; consciously unscheduled — schedule it).
17. MD060 ruling + result-cache purge (row 29).
18. Owner ruling: bulk-striking the 24 KEEP files (row 38 / T6).
19. AGENTS.md slim execution (T7 — outline shipped 2026-10-15, execution awaits variant pick).
20. Language-train leftovers (T8: go-i18n guard section, sperrmuell cookbook, survey HEAD re-verify).
21. RELEASE.md runbook gaps (T9: server_timing drift check, consumer probe, gate numbering).
22. go-compression plan inventory refresh (T10).
23. Nightly fuzz rotation verification for the 4 newer targets' first scheduled runs (T13 follow-through).
24. GmbH CSRF remediation pre-stage → actual remediation when the ruling lands (T12 pre-stage).
25. T23 shutdown-drain ruling → implementation if WaitGroup wins.
26. T24 diagram embeds → owner eyeball → README embed.
27. T25 v2.0 ledger proposals → owner rulings → Post-v1.1 rows resolve.
28. Docs-health pass + version tag: [Unreleased] now carries the measurement batch, revert entry, and verdict-doc entries — freeze-and-tag when owner calls it.

**go-etag backlog (repo I just touched):**
29. go-github-kit nix lane fix + six-consumer remote-CI verification (go-etag TODO row 1).
30. go-etag benchstat tables + post-split no-drift proof (row 2, M11) — same tooling as item 2.
31. go-etag fuzz expansion (row 2: `FuzzStoredValidatorWeaklyMatches`, `FuzzMergeHeader`, corpus).
32. go-etag erraudit-in-CI posture decision (row 3, M12).
33. go-etag coverage floors decision (row 3, M13).
34. go-etag release.yml dispatch-safe dry-run mode (row 4).
35. go-etag README "Middleware Chaining"/"Troubleshooting"/CDN-stripping docs tier (row 5).
36. go-etag process micro-policies (row 6, M18) incl. release.yml proof (OQ11).
37. go-etag epics (row 7: freshness design doc, client hooks spike, v1.0.0 criteria, daemon go-directive root-cause upstream M26 — M26 now half-answered by the honor-the-pin discovery, close the loop there).
38. go-etag: record in its AGENTS that `reports/bench/` artifacts are untracked-by-convention (only `dupl/` negation-tracked) — discovered the hard way today.

**Cross-repo hygiene:**
39. SKILLS: push or explicitly park the 2 draft commits (owner call, c1).
40. crush-config `references/lessons.md` candidates from this session: malformed-multi-edit self-wound (d1), gates-before-push (d3), foreign-commit-diff-read (d4) — cross-project, so they belong in the crush-config repo by commit.
41. KeyHolderAI: while there, check whether its go-etag bump cadence hook (dependabot/fleet sweep) covers nested-module requires post-split.
42. Verify `buildflow format --dry-run` really is tree-neutral for ALL step classes (today's run was, on this repo — one data point, not a proof; detect-only vs repair classification per step would make the snapshot protocol cheap).
43. go-etag: add the metrics bench numbers (2.3–4.2 µs overhead vs plain 1.8–2.7 µs, 0 extra allocs) to its FEATURES/README observability section if not already there — the artifact is local-only, the claim should live in tracked docs.
44. Consider upstreaming the "export-data version 5 vs 4" failure signature into a doctor check (BuildFlow `doctor` or golangci-lint issue) — it cost go-etag a red master for ~16 h before anyone looked.
45. Follow up the runner-images notice seen in CI logs (ubuntu-latest → Ubuntu 26 migration beginning 2026-10-19; both repos' workflows may need matrix pinning or acceptance).

## Postscript (19:20 CEST) — the sweep RECURRED, master re-broken and re-repaired

The tail state above was true for ~10 minutes. At 19:08:53 the daemon swept
`e6431ce` — the **identical 9-file sentinel annotation sweep** as `8302e66`,
re-applied 44 minutes after my first revert landed green. I pushed my status
report (`42aaa9d`) on top of it blind (the range `c3caa6c..42aaa9d` carried
`e6431ce`), and CI on remote went red with the exact original signature
(`example_test.go:859: ErrCSRFInvalid.WithContext undefined`, run 38775125327).
Re-reverted in `a6e0a12`, build+vet verified, pushed.

**New evidence gathered while root-causing:**
- `/etc/projects-management-automation/service.yaml` (read 19:12): `auto_push: false` — the daemon does NOT push, so the mid-session push of `8302e66` and earlier was a **second actor** (owner or parallel session); this sharpens g1.
- The daemon config shows only commit behavior (`mode: active`, on-change, 120 s min interval, auto_stage) — **no fixer/lint step**, so whatever reapplies the annotations is a separate periodic mechanism (system-unit timer or a parallel agent session running the erraudit fix whose rewrite hint is the known-noncompiling one, upstream LarsArtmann/erraudit#10). No live fixer process was visible in `ps` at 19:13; `systemctl` is blocked for me.
- The two sweeps are 49 minutes apart (18:19:35, 19:08:53) — roughly periodic; expect a third around ~20:00 if the mechanism is a timer.
- Consequence of the recurrence: a sentinel re-revert is now the expected recurring maintenance until the writer is identified and stopped. The build (vet) fails loudly each time, so detection is instant; the risk is blind pushes between sweep and revert (mine this time, d-list amendment: pushing a range without reading the intermediate heuristic commits).

**Amended g1 (supersedes the question as written):** who runs the periodic
sentinel-annotation fixer (and the mid-session push)? Candidates: a system-unit
timer visible only to you, a parallel agent session with erraudit-fix in its
loop, or the daemon's undocumented second mode. Stopping it at the source is
owner-level; everything I can do from here is detect-and-revert.

## g) Questions I can NOT figure out myself

1. **Who or what runs the periodic sentinel-annotation fixer — and who pushed mid-session?** The handoff recorded `auto_push` flipped **false** at 17:09 today, yet during my session commits through `8302e66` landed on origin/master (my unpushed set shrank from 9 to 3 while I worked). Was that you, another agent session, or a daemon config change — and should `auto_push` stay off, or was re-enabling it intentional? This determines whether I treat remote master as a shared live surface (consolidation urgency) or a quiescent one.
2. **Is a parallel session mid-work in BuildFlow right now?** I found and restored a dirty `go.mod` (zero-pseudo-version requires) 13 h after the last commit there, and the deployed binary is stale (`518739c-dirty` vs `d114f28ce`). Do you want me to rebuild + redeploy BuildFlow's flake package now (item 3), or is someone mid-migration such that I should stay out of that repo entirely?
3. **Fleet lint-pin sweep: now or never?** go-etag's red master proves every fleet repo pinning golangci-lint v2.13.2 is one Go-toolchain bump away from the same failure. Should I sweep all known fleet repos' CI pins to v2.14.0 proactively (item 4, ~an hour + CI watches), or do you want per-repo bumps to happen only when each repo's own toolchain moves?

_Arte in Aeternum_
