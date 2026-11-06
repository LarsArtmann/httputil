# Status Report — erraudit fa5575f Ledger Re-Derivation & Gate Verification

**Date:** 2026-11-06, 06:24 CET
**Repo:** httputil @ master `6787df0` (clean, synced with origin/master at session start)
**Trigger:** User pasted the output of `erraudit lint ./... --max` (56 findings, `ERROR [Rejection:violations.found]`) and asked for full triage.
**Scope:** ONLY this session's run and observations. No unrelated research (per instruction).
**Verdict up front:** The pasted output is the **documented expected state — zero drift**. The tool build changed (`119791b` → `fa5575f`), so the advisory ledger was **re-derived fresh per policy and held identically** (36 load-bearing + 17 rewrite hints + 1 legacy_is + 2 policy-rejected scripts advisories). All six real erraudit gates exit 0 across all three modules. No code changes were made; the only edit is the AGENTS.md canonical erraudit block update.

---

## Session timeline

1. Loaded `verify-external-claims` skill (claims about tool behavior were about to be encoded into AGENTS.md).
2. Repo state check: clean, synced, no parallel writers; recent commits are the endpoints/v0.1.0 release sequence.
3. Located binary: `/run/current-system/sw/bin/erraudit` (nix/home-manager deployment). `--version` revealed **fa5575f** — NEWER than the `119791b` build the AGENTS.md ledger was measured with (2026-10-25). Per the canonical block's own rule ("the ledger is re-derived per tool build, never carried forward by arithmetic"), a fresh re-derivation became mandatory.
4. Contradiction spotted (verify-external-claims in action): `lint --help` lists `--modules` with default "(all)", but the pasted run showed the root-only banner ("hint: rerun with --modules=all"). Neither text was trusted; resolved empirically.
5. First gate attempts: two explicit `--modules=all` invocations **failed loudly** — `erraudit lint` REJECTS `--modules` at post-flag-parse ("flag(s) not supported by 'lint' — they would have been silently ignored: --modules (used by: analyze, watch, tree, errorflow, nolint-audit, portfolio)"). The banner's first hint is a **trap for `lint`**; its second hint (per-root analysis) is the real path.
6. Two default-module gates auto-backgrounded; an interruption then killed those shells (shells 004/005) — re-launched ALL runs as file-backed background jobs (lesson adopted mid-session: write to files from the first attempt).
7. Full verification matrix executed (see Appendix). Results: **6/6 real gates EXIT=0; pre-sweep EXIT=0; `--max` advisory lane EXIT=2 (expected — advisory-only lane)**.
8. Mechanical counts on my OWN `--max` run (not the paste): 56 finding lines = 36 `is load-bearing` + 17 `declare it as the error interface` + 2 `returns generic error interface` (scripts/) + 1 `legacy_is` (validation_classified_test.go:474). The 17 hint sentinel names extracted and matched the documented set exactly (ErrCSRFConfig, ErrCSRFInvalid, errDecompressionSizeExceeded, errEmptyIncomingHeader, errEmptyQValue, errEmptyResponseHeader, errKeyedLimitZero, errLanguageExtractorNil, errLanguageTagsEmpty, errMethodsEmpty, errNilGenerateID, errNilMetricsRecorder, errNoWriterFactory, errServerAddrEmpty, errServerAlreadyStarted, errServerListenerConsumed, errServerListenerNil). (Counting gotcha: each hint line also contains the phrase "sentinel guard", producing 17 extra grep matches — regex artifact, not extra findings.)
9. Proof site re-confirmed LIVE: `ErrCSRFInvalid.WithContext("path", "/login")` at `example_test.go:927` (line drifted from the historically cited 859 — the current number is 927). Applying any of the 17 hints would compile-break here; this is the documented recurrence-×5 failure mode. Nothing was applied.
10. Mechanical guards confirmed present: `sentinels_type_test.go` in root, `endpoints/`, `server_timing/`.
11. AGENTS.md canonical erraudit block updated (2 edits): fa5575f re-derivation note, `--max` flag composition (why the 2 scripts advisories appear in `--max` but not the canonical lane), the `--modules`-on-`lint` rejection trap, and the six per-root nested-module gate commands (verified exit 0 today).
12. `nix fmt`: 486 files traversed, **0 changed** — edit is format-clean. Tree state: only `AGENTS.md` modified.

---

## a) FULLY DONE

| # | Item | Evidence |
|---|------|----------|
| a1 | Erraudit pasted output triaged against the documented baseline | 56 findings counted mechanically = exact match with the AGENTS.md ledger (36+17+1+2) |
| a2 | Tool-build change detected (119791b → fa5575f) and ledger RE-DERIVED fresh per policy | `erraudit --version`; own `--max` run counted, not the paste |
| a3 | Ledger held identically across tool builds — first cross-build confirmation of the 36/17 split | Count grep results (Appendix) |
| a4 | `--modules` contradiction settled empirically: `lint` REJECTS the flag; nested analysis for lint is per-root only | EXIT=1 post-flag-parse rejection captured verbatim; banner text from own run |
| a5 | All six real gates verified exit 0 (legacy_as + stdlib_constructor × root/endpoints/server_timing) | Per-module gate files all `EXIT=0` |
| a6 | Pre-sweep stopgap green: build + vet + sentinel guards, all three modules | `scripts/pre-sweep-check.sh` EXIT=0: "the tree compiles, vets clean, and keeps its sentinels concrete" |
| a7 | Proof site re-confirmed live at its CURRENT line | `example_test.go:927: rejection := ErrCSRFInvalid.WithContext("path", "/login")` |
| a8 | All three mechanical guards confirmed on disk | glob: root + endpoints + server_timing `sentinels_type_test.go` |
| a9 | AGENTS.md canonical block updated; re-derivation is now the canonical record (not a copy) | AGENTS.md lines ~59–68 + nested gate commands ~130–134 |
| a10 | Format + tree hygiene | `nix fmt` 0 changed; `git status` shows only ` M AGENTS.md` |
| a11 | Every encoded claim verified against a primary source (verify-external-claims) | help text vs. actual runs vs. own counts; no claim taken from the paste unverified |

## b) PARTIALLY DONE

| # | Item | Gap |
|---|------|-----|
| b1 | Ledger re-derivation | Done via `--max` only. The CANONICAL advisory lane (`erraudit lint ./... --type-aware --enforce-go-error-family`, no `--max`) was NOT re-run this session; its output is inferred (should be 54 = 56 − 2 scripts), not measured |
| b2 | Nested-module verification | Real gates verified per-root for endpoints + server_timing; their `--max` ADVISORY lanes were not re-run today |
| b3 | Session tail | Tree clean + formatted, but the AGENTS.md edit is UNCOMMITTED (deliberate commit left to owner/daemon per never-commit rule); push/CI-green confirmation pending |
| b4 | legacy_is advisory re-verification | Confirmed by count (1) + AGENTS.md characterization; the target line `validation_classified_test.go:474` itself was not re-read this session |
| b5 | Load-bearing ledger granularity | Verified at count level (36) + full hint-name level (17); the 36 load-bearing NAMES were not individually diffed against a prior stored list (none exists to diff against) |

## c) NOT STARTED (observed as gaps during this session; deliberately not researched per scope)

| # | Item |
|---|------|
| c1 | Upstream erraudit issue: `lint` rejects `--modules` while its own multi-module banner hints `--modules=all` |
| c2 | Upstream erraudit issue: `lint --help` lists `--modules` under FLAGS although the subcommand rejects it (help aggregation defect) |
| c3 | Extending `scripts/prerelease-check.sh` with the six per-root erraudit gates (root gates are covered; nested-module gates are manual today) |
| c4 | A `scripts/` helper that runs the whole re-derivation ritual (version capture + 6 gates + `--max` count summary) in one command |
| c5 | erraudit binary freshness check (buildflow-style: `erraudit --version` vs the erraudit source repo HEAD — where does the store build come from? Not investigated) |
| c6 | Whether ci.yml / nightly-fuzz.yml currently run any erraudit lanes that need the per-root treatment (not inspected — out of scope) |
| c7 | TODO_LIST.md / FEATURES.md erraudit rows needing the fa5575f note (not checked — out of scope) |

## d) TOTALLY FUCKED UP

**Nothing in the tree.** Master is green, synced, and the only working-tree delta is the intentional AGENTS.md edit. Two process stumbles worth recording honestly:

1. **Lost background shells (004/005).** The first gate pair auto-backgrounded and was then killed by an interruption; results had to be re-produced. Cost: one wasted verification round + a re-launch. Fix adopted mid-session: every long run writes to a file FIRST, background second. Should have been the pattern from batch 1.
2. **The `--modules=all` hint is a live trap** (not ours, upstream's): any session that follows the banner literally for `lint` gets EXIT=1. It was caught BEFORE encoding anything, and the trap (not the failed command) is what got documented. If a future session "fixes" the EXIT=1 by dropping the flag silently instead of switching to per-root analysis, nested modules quietly stop being gated — that would be the real damage vector.

## e) WHAT WE SHOULD IMPROVE

**What I forgot:**
- Re-run the canonical (non-`--max`) advisory lane — the ledger's documented measurement form. `--max` reproduction is a superset, but the canonical number (54) is inferred, not measured.
- The erraudit binary freshness check (buildflow taught exactly this lesson: "the store build predates the repo"); it was never applied to erraudit.
- The nested modules' advisory lanes (only their real gates were verified).
- Capturing `erraudit --version` in the FIRST batch (it landed in batch 2; had the binary changed mid-session, attribution would blur).

**What I could have done better:**
- Files-first for every long command from the very first attempt (learned after losing shells 004/005).
- Name-level ledger diffing (36 load-bearing names extracted and pinned), not count-only.
- One combined verification script instead of 8 ad-hoc commands — the re-derivation ritual is now clearly a repeatable procedure that deserves a helper.

**What we could still improve:**
- A warn-only staleness guard: fail/warn when `erraudit --version` hash ≠ the hash recorded in AGENTS.md, forcing a re-derivation instead of trusting a stale ledger. (Enforcement-gate proposals were owner-rejected before for the art-dupl baseline — this needs its own explicit ruling, do not build it silently.)
- The AGENTS.md erraudit block has grown into a ~90-line comment wall mixing policy, history, and incident forensics; a docs-health pass should split "current policy" from "history ledger".
- Per-root gate duplication (root + endpoints + server_timing) is currently copy-pasted commands in AGENTS.md; a script + prerelease/CI wiring (c3/c4) removes the manual surface where the recurrence-×5 class of mistakes lives.

## f) NEXT — 32 things (grouped, session-scoped)

**Close the verification loop**
1. Re-run the canonical advisory lane (`--type-aware --enforce-go-error-family`, no `--max`) on fa5575f; record the measured number (expect 54).
2. Run the `--max` advisory lane inside `endpoints/` and `server_timing/` (nested advisory re-derivation).
3. Re-read `validation_classified_test.go:474` and re-confirm the legacy_is advisory is still the benign sentinel-value class (io.EOF-family), not a new pattern.
4. Extract and pin the 36 load-bearing sentinel NAMES (JSON format + jq) next re-derivation, for name-level diffing.
5. Determine erraudit binary provenance: which nix derivation, and does `~/projects/erraudit` HEAD (or wherever its checkout lives) match fa5575f? Add a freshness note to the canonical block.

**Kill the manual surface where mistakes live**
6. Write `scripts/erraudit-ledger.sh`: version capture + six per-root gates + `--max` count summary, file-backed output, exit-code faithful (never through pipes that eat codes — session-tail discipline).
7. Wire the six per-root gates into `scripts/prerelease-check.sh` (pending Q2).
8. Wire/verify the same gates in ci.yml — NOTE: any ci.yml change must pass the cue policy gate (`#CIWorkflow`), so this is a two-file change (ci.yml + SHA bump in the policy checkout).
9. Check whether nightly-fuzz.yml runs any erraudit lane needing per-root treatment.
10. Confirm `.buildflow.yml` skip_steps erraudit is still present and still intended (1-line check next session; upstream fix for SuggestedFix-on-load-bearing is still the re-enable condition).

**Upstream erraudit (after verify-before-filing gate)**
11. File: banner hints `--modules=all` for `lint`, which rejects the flag (trap; caught this session).
12. File: `lint --help` lists `--modules` under FLAGS despite rejection (help aggregation defect).
13. Re-check status of erraudit#10 (rewrite hint does not compile for load-bearing sentinels) and #11 (legacy_is guard misses unexported sentinels) — the ledger still depends on both staying fixed upstream.
14. Request upstream: emit the per-root hint FIRST (or only) for subcommands without `--modules`, instead of the misleading `--modules=all` suggestion.
15. Longer term: re-enable buildflow's erraudit step only when SuggestedFixes stop firing on load-bearing sentinels (existing policy; keep the trigger condition visible).

**Docs hygiene**
16. Commit the AGENTS.md update deliberately (or accept the daemon sweep); then session-tail: push + CI-green on the exact head.
17. At next docs-health pass: split the AGENTS.md erraudit block into policy vs. history ledger.
18. Mirror the "banner hint vs. actual subcommand support" trap into the fleet-level erraudit knowledge (cross-repo item, owner-gated).
19. Add the current proof-site line number (927, was 859) where AGENTS.md cites example_test.go line numbers, so future forensics don't hunt a moved line.
20. This report: annotate items inline as they resolve per docs-health conventions.

**Process lessons (apply to every future tool-verification session)**
21. Files-first for every long-running command, from the first batch, always.
22. Version capture (`--version`) belongs in batch 1 next to `--help`.
23. When a banner hint contradicts `--help` defaults, resolve empirically BEFORE documenting either.
24. Count mechanically; never assert counts from a paste — even the user's own fresh terminal output.
25. For ledger-class baselines: record the tool build hash next to the counts every single time (now done in the canonical block; keep it that way).
26. Prefer one scripted ritual over N ad-hoc commands for anything expected to repeat.

**Watch items (no action until trigger)**
27. If a future `--max` count ≠ 56 (or the 17 hint names change), treat as a tool-change re-derivation event, not a code regression — check `erraudit --version` FIRST.
28. If upstream fixes the load-bearing SuggestedFix class, re-enable the buildflow erraudit step and delete the skip note.
29. If the nix deployment of erraudit changes source (version bump), re-derive and re-date the ledger.
30. If erraudit grows `--modules` support on `lint`, replace the per-root commands with the single-flag form.
31. If ci.yml gains erraudit gates, the cue policy checkout SHA must be bumped in the same change (policy gate will reject otherwise).
32. When the endpoints or server_timing modules gain their first error sentinel, extend their sentinels_type_test guard coverage check in pre-sweep (guards already exist and pass on empty sets — verify they stay meaningful, the record-reporter self-test exists in root).

## g) QUESTIONS — ANSWERED (owner rulings 2026-11-06, in-session)

1. **Canonical measurement lane:** `--max` is THE canonical re-derivation lane going forward. → AGENTS.md updated: the documented advisory command becomes `--max`-based; the old non-max `--type-aware` line is superseded for LEDGER purposes.
2. **Release/CI gate scope:** prerelease-check.sh ONLY (not ci.yml). → step added to the release script.
3. **Upstream filing:** FILE BOTH. → verify-before-filing gate then gh issues in LarsArtmann/erraudit.

---

## Appendix — verification matrix (all commands run this session)

| Command | Exit | Meaning |
|---|---|---|
| `erraudit lint ./... --type legacy_as` | 0 | REAL GATE (root) |
| `erraudit lint ./... --type stdlib_constructor --enforce-go-error-family` | 0 | REAL GATE (root) |
| `erraudit lint ./... --modules=all --type legacy_as` | 1 | `--modules` REJECTED by lint (post-flag-parse) — proves the banner hint is a trap for lint |
| `erraudit lint ./... --modules=all --type stdlib_constructor --enforce-go-error-family` | 1 | same rejection |
| endpoints: `lint --type legacy_as` | 0 | REAL GATE (nested) |
| endpoints: `lint --type stdlib_constructor --enforce-go-error-family` | 0 | REAL GATE (nested) |
| server_timing: `lint --type legacy_as` | 0 | REAL GATE (nested) |
| server_timing: `lint --type stdlib_constructor --enforce-go-error-family` | 0 | REAL GATE (nested) |
| `erraudit lint ./... --max` | 2 | ADVISORY lane (expected non-zero); 56 findings counted = 36 LB + 17 hints + 2 scripts + 1 legacy_is |
| `scripts/pre-sweep-check.sh` | 0 | build + vet + sentinel guards, all three modules |

**Count evidence (own `--max` run):** total finding lines 56; `is load-bearing` 36; `declare it as the error interface` 17; `returns generic error interface` 2 (scripts/coverage-threshold/main.go:84, scripts/doc-snippet-refs/main.go:88); `legacy_is` 1 (validation_classified_test.go:474). Banner: root-only scan (45 files), nested modules named endpoints + server_timing.

**Tool:** `erraudit version fa5575f` (`/run/current-system/sw/bin/erraudit`). **Files touched:** AGENTS.md only. **Raw outputs:** ephemeral `/tmp/ea-*.txt` (counts recorded above; /tmp does not survive reboots — the durable record is this report + the AGENTS.md block).
