# Status Report — 2026-09-22 21:46 CEST — art-dupl -t 1: `Middleware` Alias Clone Acceptance

**Session scope (per owner instruction):** this report covers only the current session's run — the `art-dupl --sort total-tokens -t 1 --type-aware` finding and its disposition. No repo-wide research was done; "noted" items below are things directly observed while reading session files.

**Session task:** `art-dupl -t 1` reported 1 clone group — `type Middleware = func(http.Handler) http.Handler` in `recorder.go:18` (root module) and `server_timing/middleware.go:6` (sub-module). Owner asked: "deduplicate?"

**Format note:** status-report skill defaults to HTML dashboards; owner explicitly requested `.md` — override honored per skill's own escape clause.

| Verdict           | Count                         |
| ----------------- | ----------------------------- |
| Fully done        | 5                             |
| Partially done    | 3                             |
| Not started       | 5                             |
| Totally fucked up | 1 (process miss, zero damage) |

---

## a) FULLY DONE

| #  | What                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                     | Evidence                                                                                                                                              |
| -- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ----------------------------------------------------------------------------------------------------------------------------------------------------- |
| a1 | `deduplicate-code` skill loaded and followed (judgment pass: extract vs accept, not blind extraction)                                                                                                                                                                                                                                                                                                                                                                                                                                                                    | Skill mandates iteration to zero _harmful_ clones; every clone dispositioned                                                                          |
| a2 | Both clone sites + module wiring read: `recorder.go:17-18`, `server_timing/middleware.go:1-6`, `go.work`, `server_timing/go.mod`, root `go.mod:11,15`                                                                                                                                                                                                                                                                                                                                                                                                                    | Verified `server_timing/go.mod` has **zero requires** (stdlib-only holds) and root `require`+`replace => ./server_timing` (strict one-way dependency) |
| a3 | Judgment rendered: **ACCEPT, do not deduplicate** — key insight: both lines are `type` _aliases_ to the identical underlying type, so they are already the same type (zero drift risk); the signature is the stdlib middleware idiom (frozen by definition); all three extraction paths are strictly worse: (1) sub-module imports root → module cycle, breaks documented zero-dep guarantee; (2) third shared module → release/tag/pin machinery for one line; (3) root aliases `servertiming.Middleware` → foundational library type coupled to an optional sub-module |                                                                                                                                                       |
| a4 | Rationale recorded in `AGENTS.md` § Accepted Code Duplication: new `Middleware` bullet + "`-t 1` appears" note added to the intro sentence                                                                                                                                                                                                                                                                                                                                                                                                                               | Daemon commit `0679e81` (verified: exactly this 3-line AGENTS.md change)                                                                              |
| a5 | Detector re-verified after documentation: `-t 1` → exactly 1 group (the accepted pair, nothing new); `-t 2` → 0 groups (documented "0 clone groups at `-t 2..25`" floor intact)                                                                                                                                                                                                                                                                                                                                                                                          | Both runs executed this session, 123 files                                                                                                            |

Also verified accurate along the way: AGENTS.md's claim that `Middleware` is "aliased as `Middleware` in `recorder.go`" matches reality. No Go code changed → no test/lint impact possible (gates deliberately not run — see e8).

## b) PARTIALLY DONE

| #  | What works                                                                                                                                                                                                        | What remains                                                                                                                                                                                                    | Effort |
| -- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------ |
| ~~b1~~ | ~~AGENTS.md change is written and committed (`0679e81`)~~ done (docs-health pass 2026-09-23, treefmt+markdownlint run tree-wide at pass end) | ~~Doc quality gates **not run** on the change: `nix fmt`/treefmt and markdownlint (buildflow `markdown-lint` step) unverified. Low risk (plain list bullet matching existing style), but unverified is unverified~~ | ~~S~~ |
| b2 | Session lesson _identified_: the `edit` tool requires an in-session `view` even when the section text is already in conversation context (first edit attempt failed with "you must read the file before editing") | Lesson **not recorded anywhere** — next session can repeat the same wasted round-trip                                                                                                                           | S      |
| ~~b3~~ | ~~Acceptance rationale documented in AGENTS.md~~ done — accepted as deliberate, three independent dismissals beat one-line skill guidance | ~~`deduplicate-code` skill asks for a _one-line_ rationale; the bullet is 3 lines. Deliberate (three independent dismissals), but the divergence from skill guidance was not consciously flagged until now~~ | ~~S~~ |

## c) NOT STARTED

| #  | What                                                                                                                                                | Why not started                                                                                              | Still wanted?    |
| -- | --------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------ | ---------------- |
| c1 | CHANGELOG `[Unreleased]` entry for the doc-only change                                                                                              | Convention unclear — repo docs don't state whether doc-only changes warrant changelog entries                | Yes, pending Q3  |
| c2 | Compile-time alias-identity test pinning the "same type" claim (e.g. `var _ httputil.Middleware = servertiming.Middleware(nil)` in `chain_test.go`) | Identified as the right way to make the acceptance rationale executable; session ended before implementation | Yes — Medium     |
| c3 | Documenting the `-t 1` baseline (expected residue: exactly 1 accepted group) in AGENTS.md **Commands** block                                        | The `-t 1` observation lives only in the Accepted section; the Commands block still only implies `-t 2..25`  | Yes — pending Q2 |
| ~~c4~~ | ~~`docs-health` HARVEST of section (f) into `TODO_LIST.md` / `ROADMAP.md`~~ done (docs-health pass 2026-09-23, harvest executed into TODO_LIST) | ~~Report was being written when session paused~~ | ~~Yes~~ |
| ~~c5~~ | ~~Cross-check that `docs/architecture-reference.md` export tables document `servertiming.Middleware` and root `Middleware`~~ done — architecture-reference.md documents both aliases, recorder.go + server_timing/middleware.go rows | ~~AGENTS.md points the "code map" at that page; I did not open it this session (scope)~~ | ~~Yes — Medium~~ |

## d) TOTALLY FUCKED UP

| #  | What                                                                                                                                                                                                         | Severity                             | Root cause                                               | Mitigation                                                                                  |
| -- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ------------------------------------ | -------------------------------------------------------- | ------------------------------------------------------------------------------------------- |
| d1 | One wasted edit round-trip: first `edit` on AGENTS.md failed ("you must read the file before editing") because I treated the section text from conversation context as a substitute for an in-session `view` | None (no damage; one tool call lost) | Edit-tool protocol: in-context content ≠ in-session read | Applied immediately (viewed, then edited successfully). Systemic fix pending as b2/c-lesson |

That is the complete list. Honest accounting: no Go code was touched, working tree is clean, no tests/lint were invalidated (nothing they cover changed), no data loss, no reverts. The session produced zero functional breakage — if you expected a longer list here, the scope (one clone pair, one doc edit) is the reason, not gentleness.

## e) WHAT WE SHOULD IMPROVE

1. **Read-before-edit, always in-session.** Even when file content is present in context (project_context, earlier reads), `view` the exact section first. Cost observed this session: one failed edit. Fix: hard rule, no exceptions.
2. **Run cheap doc gates immediately after doc edits.** markdownlint/treefmt on the touched file takes seconds; deferring them leaves "unverified" state (b1) for no benefit.
3. **Check git state right after every edit.** The daemon pickup (`0679e81`) was verified this session but late; do it immediately so evidence is fresh.
4. **Match skill guidance by default, deviate consciously.** The 3-line rationale vs 1-line skill guidance is defensible but should be a flagged decision, not an accident (b3).
5. **Codify the art-dupl threshold policy.** Today's run proved `-t 1` finds residue the documented `-t 2..25` floor hides. Without a documented baseline, the next `-t 1` run will re-litigate this exact pair from scratch. Fix: record expected residue per threshold (pending Q2).
6. **Reconcile the Accepted-section wording.** Noticed while editing: the intro says "(test files auto-excluded)" and then lists test-file clones (`mw1`/`mw2`, `newTypedBodyHandler`) — the parenthetical applies only to the 0-groups claim, but the juxtaposition invites misreading. Small wording fix.

## f) TOP 25 NEXT TASKS (session-scoped, ranked by impact)

| #  | Task                                                                                                                                                    | Impact | Effort | Category      |
| -- | ------------------------------------------------------------------------------------------------------------------------------------------------------- | ------ | ------ | ------------- |
| ~~1~~  | ~~HARVEST this section into `TODO_LIST.md` (c4)~~ done (docs-health pass 2026-09-23, harvest executed) | ~~High~~ | ~~S~~ | ~~Documentation~~ |
| 2  | Add compile-time alias-identity test `var _ httputil.Middleware = servertiming.Middleware(nil)` (c2)                                                    | High   | S      | Quality       |
| 3  | Resolve Q2 and document the `-t 1` baseline + expected residue in AGENTS.md Commands (c3)                                                               | High   | S      | Documentation |
| 4  | Resolve Q1: explicit-commit policy for deliberate doc changes vs daemon heuristic commits                                                               | Medium | S      | Process       |
| 5  | Resolve Q3: CHANGELOG policy for doc-only changes; add `[Unreleased]` entry if wanted (c1)                                                              | Low    | S      | Documentation |
| ~~6~~  | ~~Run markdownlint (buildflow `markdown-lint`) over the AGENTS.md change and this report (b1)~~ done (docs-health pass 2026-09-23, tree-wide gate at pass end) | ~~Medium~~ | ~~S~~ | ~~Quality~~ |
| ~~7~~  | ~~Run `nix fmt`/treefmt to confirm md formatting gates pass (b1)~~ done (docs-health pass 2026-09-23, tree-wide gate at pass end) | ~~Low~~ | ~~S~~ | ~~Quality~~ |
| ~~8~~  | ~~Verify `docs/architecture-reference.md` documents both `Middleware` exports (c5); fix if stale~~ done — verified — architecture-reference.md documents both Middleware exports | ~~Medium~~ | ~~S~~ | ~~Documentation~~ |
| 9  | Record the view-before-edit lesson (b2) — project lesson if it generalizes, else crush-config `references/lessons.md` by commit                         | Low    | S      | Process       |
| ~~10~~ | ~~Check `.buildflow.yml` art-dupl step threshold; align with the Q2 decision~~ **Won't implement — .buildflow.yml has no art-dupl step; gate adoption is Q2-blocked.** | ~~Medium~~ | ~~S~~ | ~~Quality~~ |
| 11 | If Q2 adopts a `-t 1` baseline: consider an expected-residue gate (fail only on NEW clone groups) in buildflow                                          | Medium | M      | Quality       |
| ~~12~~ | ~~Fix the Accepted-section parenthetical wording ("test files auto-excluded" vs listed test clones) (e6)~~ done — AGENTS.md intro reworded 2026-09-23 | ~~Low~~ | ~~S~~ | ~~Documentation~~ |
| 13 | Decide policy for the two identical one-line doc comments above the aliases (art-dupl never flags comments — sync manually or accept silently)          | Low    | S      | Documentation |
| 14 | Decide whether `server_timing/middleware.go`'s doc comment should cross-reference `httputil.Middleware` (or stay deliberately self-contained)           | Low    | S      | Documentation |
| ~~15~~ | ~~Add the exact `-t 1` invocation to AGENTS.md Commands so future runs are reproducible~~ done — exact invocation + baseline added to AGENTS.md Commands 2026-09-23 | ~~Low~~ | ~~S~~ | ~~Documentation~~ |
| 16 | Next `docs-health` VERIFY pass: assert the new bullet's claim ("at `-t 1` the pair appears") against a live run                                         | Medium | S      | Documentation |
| 17 | Pre-tag cadence: re-run the full art-dupl sweep (`-t 2..25` and `-t 1`) before the next version tag                                                     | Medium | S      | Quality       |
| ~~18~~ | ~~Re-confirm `golangci-lint run` still reports 0 warnings (no code changed — expected clean, cheap to prove)~~ done (docs-health pass 2026-09-23, golangci-lint gate run at pass end) | ~~Low~~ | ~~S~~ | ~~Quality~~ |
| ~~19~~ | ~~Keep `mw1`/`mw2` + `newTypedBodyHandler` entries accurate at the next docs-health pass~~ done — entries re-verified accurate 2026-09-23 | ~~Low~~ | ~~S~~ | ~~Documentation~~ |
| ~~20~~ | ~~Older `docs/status/` reports were deliberately not read this session (scope); fold their pending ANNOTATE items into the next docs-health pass~~ done (docs-health pass 2026-09-23, this pass annotated the remaining reports) | ~~Low~~ | ~~S~~ | ~~Documentation~~ |
| 21 | Cross-link `docs/architecture-reference.md` duplication/lint-profile section to the AGENTS.md accepted list, if a natural anchor exists                 | Low    | S      | Documentation |
| 22 | If Q2/Q3 decisions produce durable policy, capture them as short planning notes under `docs/planning/`                                                  | Low    | S      | Documentation |
| ~~23~~ | ~~Note in the next status report that test/lint gates were _deliberately_ skipped (doc-only change) so the omission reads as a decision, not an oversight~~ done — superseded — the 2026-09-23 reports run full verification gates | ~~Low~~ | ~~S~~ | ~~Process~~ |
| ~~24~~ | ~~Confirm this report file itself passes the md gates (same run as task 6)~~ done (docs-health pass 2026-09-23, tree-wide gate at pass end) | ~~Low~~ | ~~S~~ | ~~Quality~~ |
| 25 | At next AGENTS.md touch: consider one Non-Obvious-Behaviors line stating the cross-module `Middleware` alias identity (pairs with task 3)               | Low    | S      | Documentation |

Items 4, 5, 3 are decision-blocked (section g). Everything else is executable without input. This section is the HARVEST input for `TODO_LIST.md`; items marked Low/brainstormy (13, 14, 21, 22, 25) may route to ROADMAP instead.

## g) QUESTIONS I CANNOT ANSWER MYSELF

What I tried first: AGENTS.md (freeze policy documented, entry-granularity not), `CHANGELOG.md` conventions, `.buildflow.yml`, and the skill docs — none state these three; they are owner conventions.

1. **Commit policy:** The daemon already committed the AGENTS.md rationale as `0679e81` with a generic heuristic message. Do you want deliberate one-file changes explicitly re-committed with real messages (amend or follow-up), or is the daemon heuristic acceptable for doc-only changes?
2. **art-dupl baseline:** Should `-t 1` become the documented sweep baseline (gate = "no NEW groups vs the accepted list"), with `-t 2..25` keeping the zero-claim floor — or should `-t 1` stay out of the gate entirely?
3. **Changelog granularity:** Do doc-only changes (AGENTS.md rationale entries, status reports) get a `CHANGELOG.md [Unreleased]` entry, or is the changelog reserved for code/API behavior?

---

_Point-in-time snapshot. Stale by design — annotate via `docs-health` ANNOTATE, never rewrite._
