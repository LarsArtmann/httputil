# Status Report — Backlog Completion + §g Decision Execution

**Date:** 2026-10-09 17:41
**Session scope:** resume of the `2026-10-09_16-36` handoff — f8 benchmarks refresh, fmt/push/CI, HARVEST, §g ask, /tmp cleanup, final table — plus execution of all three owner §g decisions answered mid-session (golangci bump+un-pin / rolling fuzz issues / server_timing Release objects, with the owner's added constraint "single commit for both"). Machine shared with parallel session(s) throughout; load spikes shaped half the session. (Format: `.md` per explicit instruction + the repo's standing override of the HTML-canonical default.)

---

## a) FULLY DONE

1. **Handoff state verified** — both 2026-10-09 reports read in full; git tree clean (daemon had committed everything); /tmp artifact inventory confirmed before any action.
2. **f8: docs/benchmarks.md rewritten from scratch** — discovered the handoff's 16:05 "clean" run was ITSELF load-inflated (bimodal split: alloc-heavy rows 1.5–2.6x slow, pure-CPU rows stable-or-faster); ran a second full 3-module protocol pass AND a targeted 3s×5 pass of the ~12 suspect rows; merged three passes into a best-pass table with ReadyHandler as the in-run contamination detector (689 ns quiet vs 1,547 ns loaded, same binary). Doc now states: Go 1.27.1 (flake-pinned), the flate encoder-output change (Compression 2.2x faster; deflate +24% — flagged ‡ as not cross-comparable with the 1.26 baseline), Go 1.27 small-alloc speedups, the flushHeader() merge effect (273→105 ns), the new CSRF-attestation row (4,220 ns; per-op logging noted), † markers on never-quiet rows, and the root-only flake-app note.
3. **§g asked EARLY, all three answered** — question-tool batch fired DURING the bench run (lesson e.5 from the 16-36 report applied; zero idle time). Owner: bump / rolling / objects, plus "unless it can be a single commit for both" for g3.
4. **§g1 executed: golangci-lint v2.14.0 + exact-Go pins retired** — ci.yml Lint job `go-version: "1.27.x"` + `version: v2.14.0`; release.yml `GO_VERSION: "1.27.x"` + `GOLANGCI_LINT_VERSION: v2.14.0`; both comments rewritten (golangci-version-vs-Go-patch is now the pin that matters); RELEASE.md gate 6.6 reworded accordingly.
5. **§g2 executed: rolling nightly-fuzz issue policy** — nightly-fuzz.yml's per-night filer replaced by a rollup step: `nightly-fuzz` label (idempotent create), green run auto-closes open issues, failed run comments-on-or-creates ONE issue, failing targets derived from new corpus files under testdata/fuzz/ (no corpus = infra signature).
6. **§g3 executed: server_timing Release objects** — release.yml tag trigger extended with `server_timing/v*` (both objects auto-cut from the single coordinated commit — the owner's constraint satisfied: v1.5.0 and server_timing/v1.0.2 both reference `19f6a91`); RELEASE.md step 13 documents the sub-module surface; **`server_timing/v1.0.2` Release object backfilled** with hand-written notes from the shipped CHANGELOG facts (created via `--notes`, verified with `gh release view`: published, non-draft).
7. **Records written** — CHANGELOG `[Unreleased]` three new Changed bullets (jsonv2 entry preserved); DECISION_LOG 2026-10-09 row covering all three decisions; TODO_LIST harvested (below).
8. **HARVEST (f13)** — TODO_LIST: 10 items added across Medium (nightly watch, CI-skew guard, BuildFlow ripple) and Low (011 note, slog silencing, FEATURES re-derive, gopls confirm, AGENTS audit, TestServer, MaxHeaderValueCount); jsonv2 item deleted (done in CHANGELOG); `_Updated` header and High-priority note refreshed. 26 open items.
9. **markdown-lint 42→0 findings repo-wide** — realigned docs/benchmarks.md tables and the DECISION_LOG row (including the previous session's already-noncompliant jsonv2 row — fix-on-sight).
10. **43 inline annotations across both reports** (23 in 15-39, 18+2 in 16-36) via the docs-health annotator (emit-keys → --verify → apply; CI-dependent verdicts deferred until after the green run — no claim-before-evidence repeats).
11. **fmt + push + CI** — `nix fmt` clean (0 files changed); deliberate commit + push `6a72091..7dcc0c6`; **CI green**: Test (1.27.x) + Lint — the first CI run with NO GOEXPERIMENT in any workflow and the first Lint job on v2.14.0 (52 s). Tail commit `eccd453` (post-CI annotations) pushed and its CI run also green (37953009072).
12. **/tmp cleanup** — 21 session artifacts trashed (~1.3 GB, incl. three raw bench files); foreign-session files identified and untouched.

## b) PARTIALLY DONE

1. **§g2 rolling step: shipped but never live-tested** — YAML-validated only; no `workflow_dispatch` dry-run (a full run is 26×300 s ≈ 2.5 h of Actions time). Tonight's 03:08 UTC scheduled run is its first exercise; a rollup bug would surface as a failed nightly plus possibly a wrong issue action (see g.1).
2. **§g3 trigger for FUTURE sub-module tags: untested** — no sub-module tag exists until the next release; the mechanism is the same softprops action the root tags already use, but the `server_timing/v*` glob itself has never fired.
3. **Benchmark † rows remain loose** — MaxKeysChurn (620), Negotiator/browserMulti (127.9), Decompression/deflate (11,867), IDGeneratorRefillRawRandRead (4,117) never caught a fully quiet window; the doc flags them honestly († + header load note), but they are estimates, not measurements. The httpspec table ran in ONE loaded window (±25% caveat written into its section header).

## c) NOT STARTED

1. Tonight's nightly-fuzz watch (~03:08 UTC 2026-10-10; first post-fix run + first rolling-issue exercise) — TODO_LIST Medium.
2. CI guard: nightly-vs-floor Go skew preflight — TODO_LIST Medium.
3. BuildFlow fleet ripple of golangci v2.14.0 — TODO_LIST Medium (different repo).
4. 011-stale-httputil-versions v1.5.0 context — parallel-session-owned; TODO_LIST note item.
5. slog WARN silencing in `BenchmarkCSRFMiddleware_UnsafeMethodAttestationCheck` — TODO_LIST Low.
6. gopls `stdversion` confirmation post-jsonv2-drop — TODO_LIST Low.
7. FEATURES.md coverage-gap percentage re-derivation — TODO_LIST Low.
8. Fresh-session fact-loss audit of the 217-line AGENTS.md — TODO_LIST Low.
9. `httptest.NewTestServer` evaluation for Timeout tests; `MaxHeaderValueCount` doc line — TODO_LIST Low.
10. MD060 standing style ruling — still open as an owner decision, though today's 42→0 sweep changes its cost basis (see g.3).

## d) TOTALLY FUCKED UP!

1. **The DECISION_LOG alignment saga: five Python attempts, three assertion crashes, and one intermediate write that made the file WORSE.** I kept DERIVING column widths arithmetically (wrong four times: [11,249,479,266], then [10,249,479,270], then a segment-join that produced pipes at 12/262/742/1013 and 4 findings) instead of MEASURING the passing template row's part widths — which solved it in one 10-line script. The atomic asserts prevented corrupt writes, but the third attempt DID write a worse line 35 that lived for ~4 minutes. ~8 tool calls for a trivial formatting fix.
2. **TODO_LIST multiedit failures from memory: two failed edits plus my own typo.** I assumed the jsonv2 item's neighbor from recollection (wrong twice — it precedes external-claim, not writeHealthBody) and separately typed "v1.0.0.2" into the High-priority note. Three avoidable repair edits because I didn't re-view a 40-line file I had read minutes earlier. Item ORDER is context; edit from the file, not the memory.
3. **Burned a 20-minute full bench re-run without a pre-run quietness probe.** Sequence was extract-poisoned-data → notice skew → probe → re-run; it should have been probe → gate the run. The load-53 spike from the parallel session sat right on the re-run's first half. (The later targeted run got this right: probe first, detector row inside the run.)
4. **Verification theater.** My post-verify grep (`grep -c 'lineNo\|matched'`) printed 0 for both specfiles — the pattern never matched the tool's output format — and I proceeded on the adjacent "VERIFY OK" tail lines. An empty verification result is a FAILED verification, not a neutral one.
5. **Pushed the tail commit `eccd453` without watching its CI run** — the exact session-tail-discipline item (15-39 e.5: "CI green on the last commit before declaring done"). It WAS green (37953009072), but I only confirmed that during this self-review, ~25 minutes after declaring the session done. The green outcome was luck-adjacent, not process.
6. **Wrong-mechanism diagnosis, one wasted cycle**: attributed a stale "3 findings" markdown-lint result to the buildflow result cache when the truth was my own script had crashed before writing (file unchanged). Same conclusion, wrong cause.

## e) WHAT WE SHOULD IMPROVE!

1. **Measure, don't derive** — any format-preserving edit (table alignment, code blocks with house style) starts by MEASURING a known-good template, never from arithmetic first principles.
2. **Probe-before-run gating for any benchmark longer than a minute** — a 30-second detector probe (ReadyHandler 1s×3) gates every multi-minute run; keep the in-run detector row pattern, it caught both contaminations.
3. **Never edit from memory** — re-view even small files immediately before multiedit; neighbor-order and exact spelling are exactly what memory gets wrong.
4. **Ask owner questions DURING long background work** (question-tool block overlapping the bench run cost zero wall time and unblocked execution mid-session) — codify this scheduling.
5. **Empty verification output = failed verification** — fix the pattern or verify differently; never lean on adjacent output lines.
6. **CI-green-the-LAST-commit before "done"** — re-learned today on the tail push; the release runbook's 12.5 pattern (watch the exact head) is the right habit for session ends too.
7. **Kept (good, preserve):** claim-after-evidence held all session (CI-dependent annotations deferred until green — the previous sessions' d.2/d.5 lessons did NOT repeat); atomic annotator tooling with emit-keys/--verify; trash over rm; foreign-writer boundaries respected (/tmp probes, docs/review/lint/*); evidence-first bench writing (†/‡ honesty markers instead of silently shipping estimates); g3 executed to the owner's exact constraint (both release objects from the one coordinated commit).

## f) Things we should get done next (priority order)

1. **Watch tonight's ~03:08 UTC nightly-fuzz** (first post-fix run, first rolling-issue exercise); triage anything new; verify the rollup step took the correct action for the run's outcome.
2. Optional pre-flight: `workflow_dispatch` the nightly NOW to live-test the rollup step ahead of schedule (see g.1).
3. CI guard: nightly-vs-floor Go skew preflight (fail-fast preflight comparing go.work's floor with the workflow's pinned toolchain).
4. BuildFlow: ripple golangci-lint v2.14.0 into the fleet lint pins; sweep BuildFlow-side exact-Go-pin workarounds; then `buildflow doctor` binary-freshness check on every covered repo (`nix run .#reinstall` + `cp` per the stale-binary trap).
5. Coordinate 011-stale-httputil-versions v1.5.0 context with the parallel lint-audit session.
6. Silence the per-op slog WARN in the attestation benchmark (stabilizes the doc row AND removes the mangled-line parsing hazard).
7. Quiet-window re-measure of the four † rows + the httpspec table; drop † notes where a clean value lands.
8. Confirm gopls `stdversion` warnings are gone post-jsonv2-drop (editor-side pass).
9. Re-derive FEATURES.md coverage-gap percentages against the current CI artifact.
10. Fresh-session fact-loss audit of the 217-line AGENTS.md.
11. Evaluate `httptest.NewTestServer` (Go 1.27 synctest) for Timeout-middleware tests.
12. One README/API-notes line: `http.Server.MaxHeaderValueCount` settable through the server wrapper.
13. MD060 ruling (g.3) + buildflow result-cache purge for the retired MD060 finding rows.
14. Consider mechanizing benchstat regression checks (CI already uploads bench.txt; a compare-vs-doc-baseline step would replace eyeballing).
15. Next tag: standard docs-health cadence — verify living docs, harvest THIS report, archive fully-resolved older reports.
16. Tomorrow morning: confirm the nightly outcome was handled correctly by the rolling policy (green → no-op or close; red → single issue with corpus-derived targets).
17. Watch for the pkg.go.dev "not in the latest version" banner clearing at the next re-index (cosmetic; noted 16-36 §a.4).
18. If the owner wants the attestation bench de-noised deeper: route its log through `t.Setenv`/slog-discard inside the benchmark only (no production change).

## g) Questions I can NOT figure out myself

1. **Dry-run the new nightly rollup step now, or let tonight's scheduled run be its first live test?** A `workflow_dispatch` run costs ~2.5 h of Actions time (26 targets × 300 s) but proves the rollup logic (label create, comment-or-create, corpus-derived targets) before it acts on a real failure; waiting costs nothing unless the step has a bug, in which case tomorrow's nightly both fails AND possibly files/closes the wrong thing.
2. **Re-measure the four † benchmark rows + httpspec on a guaranteed-quiet machine, or accept the flagged best-pass values?** The doc is honest either way; a quiet re-measure needs a window where the parallel sessions are idle (you would know when; I can only probe and guess).
3. **MD060 style ruling (standing TODO, cost basis changed):** the repo is markdown-lint-clean as of today — rule "aligned tables everywhere" into AGENTS.md and purge the buildflow cache rows, or keep the ruling pending since future status reports will keep introducing compact tables?

---

_Point-in-time snapshot. Annotate, don't rewrite, when this goes stale._
