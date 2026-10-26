# Status: BuildFlow erraudit sweeper identified (×5), sentinel restore, pipeline green again

**Session window:** 2026-10-25 ~20:10 → 2026-10-26 ~08:00 CET (spanned midnight; includes two overnight interruptions)
**Trigger:** `BUILDFLOW_NO_RESULT_CACHE=1 buildflow --fix --build-mode=full …` exited 69 with 4 failed steps (dprint-format, govalid-generate ×2, test-race), 21 steps skipped as blocked.
**Tree state at writing:** HEAD `26fb0a7` (= `origin/master`, all commits pushed). **51 modified files uncommitted** — the auto-commit daemon is LOCKED OUT by a stale `.git/index.lock` (see d-2). CI has only ever seen the pre-session head.
**Scope discipline:** everything below is from this session's run and observations only; no unrelated research.

---

## TL;DR

The 4 failed steps had ONE root cause: `example_test.go:927: ErrCSRFInvalid.WithContext undefined`. Seventeen package-level error sentinels had been re-annotated with an explicit `error` interface type — wiping their `WithContext`/`WithCause` methods. The actor is **finally identified**: it is not an "agent applying fleet rules" — it is **buildflow's own `erraudit [root]` fix step**, which applies every `legacyerrors.Analyzer` SuggestedFix in-process with no filter, flag, or env knob. It re-broke the tree MID-RUN on 2026-10-25 20:38 (caught red-handed: `Auto-Fixes Applied: erraudit [root] 17 fixed`, then `test-coverage` failed on the freshly broken tree). This was recurrence ×5 of the class documented in AGENTS.md.

The durable repo-owned stopgap is landed and verified: `.buildflow.yml` now has `skip_steps: [erraudit]` (same pattern as the existing `go-auto-upgrade` policy skip); the REAL erraudit gates are unaffected (`scripts/prerelease-check.sh` step 6 runs `legacy_as` + `stdlib_constructor`, both re-verified exit 0). The AGENTS.md canonical erraudit block is corrected (its 2026-10-23 claim "erraudit has no --fix" was factually wrong) and now records the ×5 mechanism.

Final full run (07:23): **`BuildFlow passed with warnings 86/98` (12m15s, 374 findings) — zero step failures.** The exit-69 is the findings gate only, composed of the documented accepted advisory classes. Locally: build/vet/tests/`-race`/golangci-lint (0 issues)/sentinel guards/`pre-sweep-check.sh`/erraudit CLI gates/`nix build .#`/`dprint check` all green. All commits pushed.

---

## a) FULLY DONE (verifiably complete, evidence cited)

| # | What | Evidence |
|---|---|---|
| a-1 | **17 sentinel declarations restored** to concrete `*errorfamily.Error` (no explicit `error` annotation): `ErrCSRFInvalid`, `ErrCSRFConfig` (csrf.go:60/66), `errDecompressionSizeExceeded` (decompression.go:41), `errNilMetricsRecorder` (metrics.go:11), plus 13 var-block sentinels in compression.go (×2), cors.go, language.go (×2), ratelimit_keyed.go, requestid.go (×3), server.go (×4) | `go build`/`go vet` clean; `go test -race -count=1 ./...` ok (root 4.1s); `golangci-lint run` 0 issues both modules; `TestSentinelsKeepConcreteType` green; `scripts/pre-sweep-check.sh` "OK: the tree compiles, vets clean, and keeps its sentinels concrete" |
| a-2 | **Sweeper identified**: buildflow's erraudit fix node (`erraudit_sdk.go`, in-process `legacyerrors.Analyzer`, applies ALL SuggestedFixes; no type filter/flag/env knob — verified against BuildFlow source at `~/projects/BuildFlow`) | Run-2 log (20:38): `Auto-Fixes Applied: erraudit [root] 17 fixed` immediately followed by `test-coverage` failing with the exact `WithContext undefined` error; sentinel re-annotation present in tree after that run |
| a-3 | **Stopgap landed**: `.buildflow.yml` `skip_steps: [erraudit]` with full rationale comment (mechanism, recurrence class, real-gates pointer, re-enable condition) | Run-3 skip list shows `⊘ erraudit (skipped via skip_steps config)`; sentinels STILL CLEAN after a complete `--fix --build-mode=full` run |
| a-4 | **Real gates proven unaffected**: `erraudit lint ./... --type legacy_as` exit 0; `erraudit lint ./... --type stdlib_constructor --enforce-go-error-family` exit 0; both also live in `scripts/prerelease-check.sh` step 6/9 (independent of the skipped pipeline step) | CLI exit codes captured in session |
| a-5 | **AGENTS.md canonical erraudit block corrected + extended**: ACTOR verdict corrected (2026-10-23's "erraudit has no --fix" was wrong — there are TWO actors: agents AND buildflow's fix phase), RECURRENCE ×5 paragraph added (mechanism + stopgap + re-enable condition), hint count 15→17 reconciled with the 2026-10-25 re-measurement | AGENTS.md lines ~78–118; the block remains the declared CANONICAL HOME |
| a-6 | **dprint drift fixed**: 49 markdown files formatted (markdown table re-alignment class — docs were authored across sessions without `dprint fmt`) | `nix develop -c dprint fmt` → "Formatted 49 files", exit 0; `dprint check` exit 0; buildflow `dprint-format` step passes. Prettier verified a scoped NO-OP (`buildflow -s prettier-format` touched zero files) — **no formatter war exists**; the fight hypothesis was tested and rejected |
| a-7 | **Nix flake health proven**: all three vendorHash pins valid (benchstat, art-dupl, root module); `nix build .#` exit 0 (also refreshed stale `./result`, clearing vulnix's stale-target advisory); `nix-build-verify` and `nix-hash-fix` steps pass | `nix build` log in session; step exit codes 0 |
| a-8 | **All originally failed steps green**: `test-race` ✔ (13.8s), `golangci-lint` ✔, `govalid-generate` ✔ (its only failure was the compile break), `test-coverage` ✔ (passed in run-3) | buildflow step exit codes 0 |
| a-9 | **Final full verification**: `BUILDFLOW_NO_RESULT_CACHE=1 buildflow --fix --build-mode=full …` → `passed with warnings 86/98 12m15s (374 findings)` — 0 failed steps | `/tmp/buildflow-full3.log` (ephemeral; decisive excerpt quoted in this report) |
| a-10 | **Session tail**: `git log origin/master..master` empty (everything committed is pushed); race + lint green at tail | `git status -sb` → `## master...origin/master` (in sync) |

## b) PARTIALLY DONE (works, with named gaps)

| # | Item | Works | Remaining gap | Effort |
|---|---|---|---|---|
| b-1 | Sentinel sweep-casualty repair (a-1) | Guards + full test suite + lint prove the tree is healthy | The three sweep commits (`a85ebb8` 15:32, `970e6e6` 16:30, `bc764b9` 19:47 on 10-25) were never deliberately diff-reviewed BEYOND the guard+test+lint scope — if the sweep touched anything the guards don't watch (comments, docs, config), it is still in master | S |
| b-2 | AGENTS.md erraudit block update (a-5) | Corrected and canonical | File is now 316 lines vs the 220-line budget (was already 308 pre-session; my +12 lines − trim = +8). The size gate (`go-structure-linter`) errors on it every run. Restructuring = owner-level content decisions | M |
| b-3 | erraudit pipeline step | skip_steps stopgap verified across one full `--fix` run | Upstream erraudit still EMITS the wrong SuggestedFixes (tracked: LarsArtmann/erraudit#10, #11; GOTCHAS #125 wording) — until fixed upstream, the step stays skipped and its DETECT value (advisory residuals) is lost from pipeline runs; re-enable + re-measure is an open task | M (upstream owner-gated) |
| b-4 | Session work persistence | All work verified locally; pushed state clean | The 51-file session tree is UNSWEPT (daemon locked out, d-2) — no commit carries it yet, so nothing is CI-verified and a careless `git reset --hard`-class accident anywhere would destroy it | S |
| b-5 | CHANGELOG | n/a — not started this session | `[Unreleased]` entry warranted: `ErrCSRFInvalid`/`ErrCSRFConfig` are EXPORTED — their declared type semantics (concrete `*errorfamily.Error`, `.WithContext` available) are consumer-visible and briefly regressed on master during the ×5 window | S |

## c) NOT STARTED (planned/observed, no action taken this session)

| # | Item | Why not started | Still wanted? |
|---|---|---|---|
| c-1 | Stale `.git/index.lock` removal | Owner-gated in practice: AGENTS.md documents removal + expected catch-up burst, but a parallel writer could theoretically be mid-operation; user instruction was "report, then wait" | Yes — first action after instructions |
| c-2 | Upstream erraudit fix (suppress SuggestedFixes for load-bearing sentinels; load-bearing analysis should read test files) | Owner-gated cross-repo item per AGENTS.md ("not silent edits") | Yes |
| c-3 | Go toolchain bump for stdlib CVEs (govulncheck: 44 root + 14 server_timing findings, e.g. GO-2026-6809 crypto/x509, GO-2026-6613 net/http) | Version-bump decision is owner's; detect-only advisory | Yes — real exposure, not noise |
| c-4 | nix-checker residual-class verdict + AGENTS.md accepted-residual entry | Not triaged this session; its 10 findings (pinned-hash info + vendorHash warnings) fail the findings gate every run but appear NOWHERE in the AGENTS.md accepted-residuals list | Yes |
| c-5 | AGENTS.md 220-line restructure | Owner-level content routing (see b-2) | Yes |
| c-6 | docs-health HARVEST of this report's section (f) into TODO_LIST.md | Report must exist first; harvest is the next step after the user reads it | Yes — skill mandates closing the loop |
| c-7 | dprint/CI markdown format gate (prevent the 49-file drift class from recurring) | Not designed; needs a decision where the gate lives (CI vs pre-sweep vs treefmt) | Yes |
| c-8 | Historical report annotation: `docs/status/2026-10-23_15-56_otel-tracing-exploration-and-red-master.md` documents the ×4 saga — its root-cause section is now superseded by the ×5 actor identification | Doc-Freshness ANNOTATE mode; read during triage but not annotated (my miss, see e-6) | Yes |

## d) TOTALLY FUCKED UP

| # | What is broken | Severity | Root cause | Mitigation |
|---|---|---|---|---|
| d-1 | **BuildFlow's erraudit fix step damages any repo with load-bearing sentinels, on every `--fix` run.** It applied 17 wrong rewrites mid-run on 10-25 20:38, instantly compile-breaking the test package. It did this at 15:32, 16:30, 19:47, and 20:38 on 10-25 alone (×5 class total). | HIGH — master-red class; CI inherited broken trees historically (the 072e6db precedent: red AND pushed ≈3¾ h) | `legacyerrors.Analyzer` SuggestedFixes applied unconditionally in-process; no type filter/flag/env knob exists (verified in BuildFlow source); the tool's own load-bearing analysis mislabels these sentinels because it reads non-test files only | **Landed**: `skip_steps: [erraudit]`. NOT fixed upstream — every other fleet repo with go-error-family sentinels is still exposed |
| d-2 | **`.git/index.lock` is present RIGHT NOW; the auto-commit daemon has been locked out since ≥20:28 (10-25).** 51 modified files unswept for ~11.5 h overnight. Exactly the AGENTS.md stale-lock corollary: "a multi-hour heuristic-commit gap means the daemon is locked out, not idle." | MEDIUM-HIGH — uncommitted work at risk; git-log co-change evidence frozen | Daemon killed mid-sweep or lock left by the SQLITE_BUSY-errored buildflow run (both seen this session); `conflict_strategy: abort` then silently skips every cycle | Documented removal procedure exists (AGENTS.md); NOT executed — awaiting go-ahead (c-1) |
| d-3 | **AGENTS.md carried a factually wrong root-cause claim for 3 days** ("the sweep is not a mechanical fix — erraudit has no --fix", P05 verdict 2026-10-23). The fleet hunted an "agent applying GOTCHAS #125" while buildflow's own pipeline was the offender — which is why the guards (test-time) kept losing the race. | MEDIUM — corrected now (a-5), but the wrong claim shaped two days of mitigation design | The P05 verdict was plausible-when-written; the buildflow erraudit step's fix phase (in-process SDK) postdates or was unobserved at verdict time | Corrected in the canonical block this session |
| d-4 | **Findings-gate alert fatigue: 374 findings per full run, exit 69 every time.** Six accepted classes (file-size 32, govulncheck 44+14, vulnix 23, nix-checker 10, go-structure-linter 4, flake-meta 1, + more) bury any NEW signal — including, until this session, the erraudit step's own damage. nix-checker isn't even in the AGENTS.md accepted-residuals list, yet fails the gate every run. | MEDIUM — process hazard, not a code bug | Detect-only advisories + findings gate semantics | Needs a residuals census + verdict per class (c-4) and possibly a gate-policy owner decision |
| d-5 | **buildflow binary staleness**: binary built at `58115f7`, repo HEAD `24dff81` at session start (preflight advisory). This session's mechanism findings were verified against SOURCE, but the RUNNING pipeline is the older binary. | LOW-MEDIUM | Store build predates repo (known delivery pattern) | `nix build . && nix run .#reinstall` — not run this session |

## e) WHAT WE SHOULD IMPROVE (process + my own performance)

**What I forgot / got wrong this session (honest list):**

- **e-1. I hand-rolled a grep before running the repo's own guard.** My `rg 'var Err… error'` scan missed all 13 var-block sentinels (block entries don't start with `var`); `TestSentinelsKeepConcreteType` found them instantly. The guards existed precisely for this — I should have run them as the FIRST discovery step, not as a later verification.
- **e-2. I attributed the re-annotation to "the daemon" for too long.** The sweep commits (15:32/16:30/19:47) were visible in `git log` early; identifying the ACTOR required the run-2 accident (erraudit re-breaking the tree while I watched). A deliberate `git log -p` review of one sweep commit would have shown the annotation pattern earlier; the run-2 log made it undeniable.
- **e-3. I lost a background shell across a turn boundary** (the first full verification run's result vanished — shell `02B`), wasting a full ~12 min run. Background jobs are not durable across turns here; long verifications should run foreground or be re-runnable cheaply.
- **e-4. My artificial `timeout 900` killed a healthy run** (exit 124; the run needed 12m15s under SQLITE_BUSY contention). Using buildflow's own `--max-time 30m` was correct from the start; I layered a redundant, tighter cap.
- **e-5. `dprint` PATH miss cost a cycle** — the binary lives in the project devShell (`nix develop -c dprint …`), not the raw shell. A repo-AGENTS.md line stating "dprint = `nix develop -c dprint fmt`" would have saved it.
- **e-6. I read a historical status report during triage and did not annotate it** (Doc-Freshness cadence requires inline annotation on read) — c-8.
- **e-7. I grew AGENTS.md while its size gate was already erroring** (308→316). The knowledge belongs in the canonical block, but I should have trimmed harder or flagged the restructure explicitly instead of deepening an over-budget file.
- **e-8. I didn't probe whether erraudit has an inline suppression mechanism before choosing skip_steps** (the BuildFlow workspace gate passes `--no-suppress`, implying a suppress mechanism EXISTS). A finer-grained repo-local fix (17 suppressions) might have kept the step's detect value. Chosen stopgap is still correct-and-verified, but this alternative was unexplored.

**Systemic improvements (not bugs — those are in d):**

- **e-9. Discovery order for "tree suddenly broken": guards first, grep second, blame third.** This session proves the repo's mechanical guards are the fastest truth source. Worth an AGENTS.md line in the incident playbook.
- **e-10. The AGENTS.md accepted-residuals list has no census owner.** Classes accumulate (nix-checker missing entirely); each full run's 374 findings re-litigate them. A single "residuals census" doc + AGENTS.md pointer would keep the gate readable.
- **e-11. Two-formatter surface (dprint + treefmt/golangci-lint fmt) has no combined "format everything" entry point** that sessions reliably know. `nix fmt` covers Go only; dprint needs the devShell. One documented command (or treefmt wiring for dprint) ends the drift class.
- **e-12. Background long-runs need a documented pattern** in this fleet (foreground with generous budget, or idempotent re-runs), after e-3.

## f) Next tasks (up to 50 — brainstorm ranked by impact; HARVEST fuel)

Impact: Critical/High/Medium/Low · Effort: S <30min, M 30min–2h, L >2h

| # | Task | Impact | Effort | Category |
|---|---|---|---|---|
| 1 | Remove stale `.git/index.lock` (verify no live daemon first) and confirm the daemon's catch-up sweep | Critical | S | Process |
| 2 | Commit the 51-file session tree deliberately (AGENTS.md soft-reset consolidation once all unpushed commits verifiably mine) so work is CI-visible | Critical | S | Process |
| 3 | Verify CI green on the post-sweep head (session-tail discipline: green on that exact head) | High | S | Quality |
| 4 | Diff-review sweep commits `a85ebb8`/`970e6e6`/`bc764b9` against guard scope (b-1) — confirm zero non-sentinel damage on master | High | S | Quality |
| 5 | Upstream erraudit: stop emitting SuggestedFixes for load-bearing sentinels (owner-gated; supersedes #10/#11 partial fixes) | High | L | Bug (upstream) |
| 6 | Upstream erraudit: load-bearing analysis should read test files (proof: `ErrCSRFInvalid.WithContext` at example_test.go:927/859) | High | M | Bug (upstream) |
| 7 | Correct BuildFlow docs/GOTCHAS #125 blanket wording ("sentinels MUST be error interface type") | High | M | Documentation (upstream) |
| 8 | Re-enable erraudit step after #5/#6 land; remove skip_steps; re-measure ledger | Medium | S | Cleanup |
| 9 | Go toolchain bump for stdlib CVEs (GO-2026-6809, GO-2026-6613, +44/14 findings); then re-run govulncheck to zero | High | M | Bug/Security |
| 10 | nix-checker: residual-class verdict + AGENTS.md accepted-residuals entry (c-4) | Medium | S | Documentation |
| 11 | Probe erraudit's suppression mechanism (`--no-suppress` implies one exists); if per-line suppressions work, evaluate 17 sentinels + re-enable detect-only (e-8) | Medium | M | Quality |
| 12 | Parser-based sentinel scan that does NOT need package compilation (standalone `go run` script in pre-sweep-check.sh) — closes the guard's chicken-and-egg blind spot when example_test.go is broken | High | M | Quality |
| 13 | Add `.git/index.lock`-presence + daemon-gap check to `pre-sweep-check.sh` (d-2 early warning) | Medium | S | Tooling |
| 14 | Add `dprint check` to pre-sweep/CI so the 49-file markdown drift class cannot silently accumulate again | Medium | S | Quality |
| 15 | Document `dprint = nix develop -c dprint fmt` in AGENTS.md Commands (e-5) | Low | S | Documentation |
| 16 | Decide treefmt-vs-dprint unification: wire markdown into `nix fmt` OR bless `nix develop -c dprint fmt` as the one command (e-11) | Medium | M | Quality |
| 17 | AGENTS.md restructure to ≤220 lines: move erraudit history (×4/×5 narrative) to `docs/review/lint/erraudit-policy.md`, keep ≤20-line summary + pointer (b-2) | Medium | M | Documentation |
| 18 | CHANGELOG `[Unreleased]`: sentinel concrete-type restoration (exported `ErrCSRFInvalid`/`ErrCSRFConfig` semantics) + erraudit skip note (b-5) | Medium | S | Documentation |
| 19 | HARVEST this report's (f) into TODO_LIST.md; route L/upstream items to ROADMAP (c-6) | Medium | S | Documentation |
| 20 | Annotate `docs/status/2026-10-23_15-56…` report: ×4 root cause superseded by ×5 actor identification (c-8) | Low | S | Documentation |
| 21 | Rebuild + reinstall buildflow binary (58115f7 → HEAD) per preflight advisory (d-5) | Medium | S | Process |
| 22 | Investigate buildflow resume-checkpoint SQLITE_BUSY retries under concurrent sessions; consider busy-timeout upstream (seen ~every 3s for minutes in run-2's overnight log) | Medium | M | Bug (upstream) |
| 23 | Explain/fix `nix-hash-fix` 56/63 (89%) historical failure rate — one `--verbose` diagnostic; is the fixer broken or are the stats stale? | Medium | S | Bug |
| 24 | Same for `nix-build-verify` 11/11 (100%) stats that now pass — stale retry stats mislead triage (seen live this session) | Low | S | Cleanup |
| 25 | Findings-gate policy decision: should `--build-mode=full` with only accepted-residual classes exit 0/"passed with warnings" instead of 69? (374 findings ≈ permanent red; owner call) | Medium | S | Process (owner) |
| 26 | Audit the other fleet repos with go-error-family sentinels (go-etag, go-appkit/otel, …) for the same re-landing damage this week | High | M | Bug |
| 27 | vulnix: decide keep/exclude; if kept, refresh `./result` on cadence (stale-target warning recurs after every flake.lock change) and triage its CVE noise into the residuals doc (nixpkgs-closure CVEs ≠ shipped code) | Low | S | Cleanup |
| 28 | Document that `BUILDFLOW_NO_RESULT_CACHE=1` runs produce false timing-regression verdicts (cold caches); or re-baseline timings cacheless (this session: golangci-lint "+788%") | Low | S | Documentation |
| 29 | Timing-regression verdict baselines: re-baseline after the erraudit step removal changes run composition (lint steps no longer raced against broken-tree runs) | Low | S | Cleanup |
| 30 | Verify CI does not invoke buildflow `--fix` on this repo (would re-land sentinels on CI infra pre-upstream-fix) | Medium | S | Quality |
| 31 | Record the durable evidence trail: copy decisive run-2 log excerpt (`erraudit [root] 17 fixed` → `test-coverage` break) from /tmp into `docs/review/lint/` (/tmp is ephemeral) | Medium | S | Documentation |
| 32 | Extend `sentinels_type_test.go` self-test to also cover the var-block form explicitly (13 of 17 casualties were block-style; the detector handled them, but the self-test fixture only pins the plain `var errX error` form) | Low | S | Quality |
| 33 | Cross-check AGENTS.md "Commands" erraudit block wording against the new skip_steps reality (no stale "run it in buildflow" implications) | Low | S | Documentation |
| 34 | Pre-existing flag check: FEATURES.md / docs/v1-stability.md error-code tables unaffected by type-annotation restore (behavior-identical; verify no doc claimed `error` interface type) | Low | S | Documentation |
| 35 | Add a regression note to the ×5 paragraph's re-enable condition: "re-enable ONLY after #5 AND #30 both hold" | Low | S | Documentation |
| 36 | Consider pinning dprint version in flake inputs (nixpkgs 0.64.1 today; formatter-version drift = future reformat waves) | Low | S | Quality |
| 37 | Run `buildflow -s erraudit` (single-step) once deliberately on a THROWAWAY branch to document its exact detect-vs-fix behavior for the AGENTS.md playbook (safe-invocation table) | Medium | S | Process |
| 38 | Sweep AGENTS.md "Pre-Existing Lint Warnings" claim ("0 active warnings") re-verification after this session's edits (still true: golangci 0 issues — re-cite at next docs-health pass) | Low | S | Documentation |
| 39 | Daemon health runbook: one script/command that answers "is the daemon alive, locked out, or config-disabled?" (config-controlled `auto_push` + stale-lock + index.lock checks) — AGENTS.md documents the parts; unify them | Medium | M | Tooling |
| 40 | Postmortem-cross-link: add this report's URL to the AGENTS.md ×5 paragraph (evidence pointer) once committed | Low | S | Documentation |
| 41 | `go.mod`/CI Go patch pin alignment check after any toolchain bump (#9) — RELEASE.md gate 6.6 pins `GO_VERSION` at tag time | Medium | S | Process |
| 42 | Verify `scripts/coverage-threshold` unaffected (test-coverage passed in run-3 — cite as evidence, then close) | Low | S | Quality |
| 43 | Re-run the clone baseline (`art-dupl` COMBINED mode) after this session's 9-file edit to re-confirm "exactly the two accepted pairs" prose baseline | Low | S | Quality |
| 44 | `docs-health` VERIFY pass on the updated AGENTS.md erraudit block (fresh-eyes check that every claim traces to this session's evidence) | Medium | S | Documentation |
| 45 | Check whether the 2026-10-25 19:44/19:56 planning docs (written by the same session window that produced the sweep) recorded decisions that contradict the landed skip_steps | Low | S | Documentation |
| 46 | Evaluate `buildflow doctor` output post-reinstall (#21) for the env/binary-freshness advisory clearing | Low | S | Process |
| 47 | Fleet question: should BuildFlow's findings gate treat `skip_steps` entries as a reported "policy-skipped" class in summaries (visible without --verbose)? | Low | S | Process (upstream) |
| 48 | Add the "guards first" discovery rule to AGENTS.md incident playbook (e-9) | Low | S | Documentation |
| 49 | Backfill: confirm no OTHER exported API surface changed in the sweep commits (export-diff `git diff a85ebb8^..bc764b9 -- '*.go'` scoped to exported identifiers) | Medium | S | Quality |
| 50 | Schedule the monthly docs-health cadence pass (last blocking session predates this one; TODO_LIST state unverified this session) | Medium | S | Documentation |

## g) Top 3 questions I cannot answer myself

1. **Erraudit endgame:** Do you want the upstream erraudit/BuildFlow fixes (#5/#6/#7 above) to be the path to re-enabling the erraudit step — and until then, is `skip_steps: [erraudit]` acceptable for THIS repo and should the same skip be applied to every other fleet repo with go-error-family sentinels (they are still exposed to the ×5 damage class)? I cannot judge the fleet-wide rollout from this repo.

2. **The locked-out daemon + 51 unswept files:** Shall I remove the stale `.git/index.lock` and let the daemon catch up, or do you prefer a deliberate consolidated commit now? (I verified all unpushed commits' file lists per the AGENTS.md consolidation protocol — but "who else wrote into this window" is only fully answerable by you, and removal triggers a documented catch-up burst.)

3. **Go toolchain bump for the stdlib CVEs:** govulncheck reports 44 root + 14 server_timing findings (GO-2026-6809 crypto/x509, GO-2026-6613 net/http, …). Bump `go.mod`/CI to the patched 1.27.x now as a patch release, or fold into the next planned version? This decides whether the largest remaining findings class is real exposure or accepted noise.

---

## HARVEST note

Section (f) is the primary input for `docs-health` → **HARVEST**: pull items 1–4, 9–14, 17–21 into `TODO_LIST.md` (actionable, bounded), route upstream/owner-gated items (5–8, 25, 26, 47) to `ROADMAP.md`, and keep the rest as candidates. Do not let them die in this timestamped file.
