# Status Report: art-dupl Clone-Baseline Reverification Session

- **Timestamp:** 2026-10-24 08:49 CEST
- **Session scope:** Triage of a user-run `art-dupl --suggest-generics --type-aware --html` report, canonical clone-baseline verification, combined-mode research, and baseline-doc freshness refresh. The owner explicitly scoped the session: "DO NOT RESEARCH OTHER STUFF UNRELATED TO WHAT YOU DID" — this report covers only this session's run and what it noticed.
- **Format note:** The status-report skill is HTML-canonical; the owner explicitly demanded a `.md` at `docs/status/`, so the override is honored and flagged here. The brutal-self-review reflection questions are merged into sections (d)/(e) instead of a separate `docs/reviews/*.html` episode (same owner scope: one report, this session only).

---

## Headline

**The clone baseline is intact and now verified with a newer tool than the docs recorded.** The user's combined-mode run and both canonical verification runs agree: at `-t 1` exactly the two documented accepted pairs show, at `-t 2` zero groups show, and the `--suggest-generics` lane contributes **no generics candidates**. Nothing new is actionable; no code changes were needed or made. Doc-only freshness refresh landed and is already pushed via the daemon (`e6fbd0f`).

| Threshold | Command (canonical, AGENTS.md) | Result | Expected | Verdict |
| --- | --- | --- | --- | --- |
| `-t 2` | `art-dupl --sort total-tokens -t 2 --type-aware .` | **0 shown** (370 detected, 204 non-actionable, 166 filtered) | 0 groups | ✅ match |
| `-t 1` | `art-dupl --sort total-tokens -t 1 --type-aware .` | **exactly 2 shown**: `Middleware` alias pair + `scripts/doc-snippet-refs` if-err pair | exactly 2 accepted pairs | ✅ match |
| `-t 1` combined (user's run) | `+ --suggest-generics` | same 2 shown; **0 generics candidates** | (new lane, no prior record) | ✅ clean |

## Evidence trail (this session)

1. User's combined run parsed: exit 0; 2 actionable groups; 798 suppressed; `type-load 59.111s` (cold go/packages cache), total 59.272s.
2. Tool identity: `art-dupl 0.9.0-93432fd` at `/run/current-system/sw/bin/art-dupl`; source repo `~/projects/art-dupl`. Docs previously recorded `0.7.0-81ce00b` — two minor versions of drift in the stamp, now corrected.
3. Research: ADR-0026 (combined mode = type-aware pass + type-erased pass over ONE shared type-check; a type-erased family survives only with ≥2 divergent type positions; same-type subgroups still reported by the type-aware pass). `docs/ACTIONABILITY_PATTERNS.md` confirms the suppression semantics: a group is NonActionable when every clone matches a boilerplate pattern (`single-call`, `assign-error-check`, `single-declaration`, …) — that is the 798.
4. Canonical runs executed verbatim with the required env (`GOCACHE`, `GOTOOLCHAIN=auto`), exit 0 both.
5. Judgment against `docs/architecture-reference.md:164-173`: both shown groups match their documented accept-rationale (module-cycle/zero-dep guarantee for the alias pair; standalone-stdlib-script idiom for the if-err pair). The other two accepted clones (`mw1`/`mw2`, `newTypedBodyHandler`) never show because the tool auto-excludes test files — consistent with the docs' wording.
6. Freshness refresh: `AGENTS.md:289` and `docs/architecture-reference.md:166` updated 2026-09-23/0.7.0 → 2026-10-24/0.9.0-93432fd, plus the combined-mode-clean clause and `0 groups` → `0 shown groups` (tool-vocabulary alignment).
7. Daemon sweep verified: `e6fbd0f` = exactly my 2 doc edits; tree clean; `git log origin/master..master` empty (all pushed).

---

## a) FULLY DONE

| # | Item | Evidence |
| --- | --- | --- |
| a1 | Combined-mode output triaged: both shown groups are the documented accepted pairs, nothing new actionable | user's run output; `docs/architecture-reference.md` §Accepted Code Duplication |
| a2 | Suppression semantics understood and documented (798 = actionability-pattern gate at threshold 1, expected behavior) | art-dupl `docs/ACTIONABILITY_PATTERNS.md` |
| a3 | Combined-mode semantics understood (ADR-0026: two passes, one type-check, ≥2-divergent-positions gate) | art-dupl `docs/adr/0026-*.md` |
| a4 | Canonical baseline re-verified at BOTH thresholds with the newer tool 0.9.0-93432fd, exit 0 | session shell transcripts |
| a5 | Generics lane ruled clean (0 candidates) — first recorded combined-mode verdict for this repo | user's run summary |
| a6 | Baseline stamps refreshed in both docs, single coherent sentence each | `AGENTS.md:289`, `docs/architecture-reference.md:166`, diff 2+/2- |
| a7 | Tail state confirmed: edits committed by daemon (`e6fbd0f`, exactly the 2 files), tree clean, everything pushed | `git status`, `git log origin/master..master` |
| a8 | Parallel-writer protocol respected on `072e6db` (noticed, stat-inspected, deliberately untouched) | session note |

## b) PARTIALLY DONE

| # | Item | Done | Missing |
| --- | --- | --- | --- |
| b1 | `072e6db` triage | `--stat` read (11 files: 9 middleware `.go` + `go.mod` + `flake.lock`, 19+/19-); author identified as the daemon (heuristic message) | The diff CONTENT was never read or judged — "heuristic = daemon" names the author, not the health of the change. `go.mod`/`flake.lock` deltas unverified against the known daemon-casualty classes (directive downgrade, stale lock catch-up) |
| b2 | Session-tail discipline | Push state ✅ and clean tree ✅ (checked at 08:49) | CI state on HEAD never checked; `scripts/pre-sweep-check.sh` never run even though `072e6db` touched `.go` files — tree health was assumed, not verified |
| b3 | Warm-run timing claim | Observed fast wall-clock on canonical re-runs | Never measured (`--timing` not passed); the 59s figure is the only measured number and it is the COLD cost |
| b4 | Self-review integration | Brutal-self-review questions answered inline (sections d/e) | Not written as its own `docs/reviews/` episode; prior review series not re-read (owner scope: this session only) |

## c) NOT STARTED

All of these are out of this session's scope (owner constraint or owner-gated decisions) — listed so nothing is silently dropped:

- c1. Harvest of section (f) into `TODO_LIST.md` / `ROADMAP.md` (docs-health HARVEST) — awaiting owner go-ahead; owner said "WAIT FOR INSTRUCTIONS".
- c2. Full audit of `072e6db`'s diff content (b1).
- c3. Pre-sweep build-gate run / CI verification on HEAD (b2).
- c4. Whether the combined-mode invocation becomes part of the canonical Commands-block baseline procedure (owner call, question 3).
- c5. Whether the clone baseline becomes an enforced gate instead of prose (owner call, question 1).
- c6. De-duplicating the baseline stamp into a single source of truth (currently maintained in two files by hand).

## d) TOTALLY FUCKED UP

**Nothing at that level from this session.** No code was touched, no red master was created, nothing was destroyed, no pushed-but-broken state is attributable to this session. Said honestly and completely.

Nearest misses (named so they don't grow — these are process stumbles, not damage):

1. **Two rejected edit calls.** I attempted the AGENTS.md/architecture-reference edits before `View`-ing the exact target sections in-session; both tool calls bounced, and my first `View` of AGENTS.md even landed on the wrong section (214 vs 289). Cost: two wasted round trips in a two-line edit task. The rule is "read the relevant context before editing" — I had the file contents via project context and grep, but not a View of the target lines, and the tool is right to enforce that.
2. **A pre-existing, still-open split brain that this session perpetuated rather than eliminated:** the baseline sentence lives in BOTH `AGENTS.md` and `docs/architecture-reference.md` and must be manually synchronized on every re-measurement. Today I synchronized both correctly — but nothing enforces the pair, and nothing enforces the count itself. The baseline is prose, so it can be silently relitigated (the deduplicate-code skill explicitly warns: "An enforced count cannot be relitigated; a prose note can"). go-etag already solved this with a pinned group-count gate.
3. **Doc vocabulary drift (now fixed):** AGENTS.md said "0 groups" where the tool and the reference doc say "0 shown groups" — a reader comparing doc to tool output could misread suppression accounting as a regression.

## e) WHAT WE SHOULD IMPROVE

1. **Verify tree health, don't assume it.** When a parallel writer's daemon commit touches `.go` files, run `scripts/pre-sweep-check.sh` before ending the session — even when my own change is docs-only. The daemon commits non-compiling intermediates; that is a recorded incident class here.
2. **Read unexpected diffs beyond `--stat`.** Authorship (daemon vs human) and correctness are different questions. The 30-second diff read was skipped and should not have been.
3. **Measure performance claims or don't make them.** "Warm re-runs were fast" should have been one `--timing` flag away from being evidence.
4. **Enforce the baseline, don't narrate it.** The skill doctrine is gate-over-prose (go-etag precedent: fail unless art-dupl shows exactly the pinned groups). This repo's baseline is the same shape and deserves the same treatment — see question 1.
5. **Single-source the baseline stamp.** One canonical location, the other doc pointing at it — removes the manual two-file sync every re-measurement.
6. **View-then-edit, always.** Two bounces for a 2-line change is avoidable friction.
7. **Record cold-vs-warm type-load expectations** (~59s cold for 137 files) near the art-dupl commands so a future session doesn't misread the stall as a hang and start "fixing" it.

## f) THINGS WE SHOULD GET DONE NEXT

The owner asked for up to 50. The status-report skill says a larger N is a brainstorm, not a commitment list — and honesty beats quota: this session observed exactly the items below. Padding to 50 with filler would manufacture work; items 15+ belong to the repo's broader backlog, which the owner explicitly fenced off this session ("DO NOT RESEARCH OTHER STUFF"). Sorted by impact; owner-gated items marked **[OWNER]**.

| # | Task | Type | Effort |
| --- | --- | --- | --- |
| 1 | ~~Run `scripts/pre-sweep-check.sh` to verify tree health post-`072e6db` (build + vet + sentinel guards, ~30s)~~ ✅ 2026-10-24, exit 0 after the revert (see addendum) | mechanical | ~1 min |
| 2 | ~~Read `072e6db`'s full diff; judge the `go.mod`/`flake.lock` deltas against the known daemon-casualty classes; escalate if either looks like a directive downgrade or stale-lock casualty~~ ✅ 2026-10-24 — verdict: forbidden sentinel sweep (recurrence ×4), reverted; go.mod/flake.lock legitimate, kept (see addendum) | mechanical | ~10 min |
| 3 | Confirm CI green on pushed HEAD `e6fbd0f` (tail discipline step that was skipped) | mechanical | ~2 min |
| 4 | ~~**[OWNER]** Decide: enforce the clone baseline as a gate (fail prerelease/CI unless `-t 1` shows exactly the 2 accepted pairs; `-t 2` exactly 0) — go-etag precedent, likely a `prerelease-check.sh` step rather than a unit test, since shelling to art-dupl from `go test` is the wrong shape~~ DECIDED 2026-10-24: owner REJECTED the gate — baseline stays prose; do not re-raise without new evidence | decision | — |
| 5 | ~~**[OWNER]** Decide: does `--suggest-generics` combined mode join the canonical Commands-block baseline procedure, or stay an ad-hoc audit lane?~~ DECIDED 2026-10-24: owner ruled CANONICAL — Commands block + baseline sentences updated, combined `-t 2` re-verified (0 shown) | decision | — |
| 6 | ~~If (4) approved: implement the gate as a `scripts/` step wired into `prerelease-check.sh` (mirroring the erraudit sentinel-guard pattern), and record the pinned counts (2 groups @ `-t 1`, 0 @ `-t 2`)~~ DEAD — gate rejected (item 4) | implementation | ~1 h |
| 7 | Single-source the baseline stamp: keep the dated sentence in `docs/architecture-reference.md` only; reduce `AGENTS.md` to a pointer — ends the two-file sync burden | docs | ~15 min |
| 8 | Re-run BOTH canonical thresholds at the next art-dupl or Go-toolchain bump and refresh stamps in the same change (the 0.7→0.9 jump happened inside one month; actionability tables evolve, e.g. the 2026-10-16 label additions, and can silently reclassify shown groups) | cadence | ~10 min per bump |
| 9 | ~~Record the generics-lane verdict per re-baseline ("combined mode clean as of 2026-10-24") so each future combined run has a comparison point~~ ✅ 2026-10-24 — verdict recorded in AGENTS.md + docs/architecture-reference.md baseline sentences | docs | ~2 min |
| 10 | Add a one-line cold-type-load expectation (~59s for 137 files) near the art-dupl commands in AGENTS.md — anti-panic note for future sessions | docs | ~2 min |
| 11 | Capture ONE warm `--timing` run to turn the "warm is fast" claim into a measured number next to the cold 59.1s | measurement | ~2 min |
| 12 | Document where the art-dupl binary/version comes from (nix profile → current-system/sw) so future freshness updates know the upgrade path instead of discovering drift by stamp mismatch **[needs owner confirmation of the pin location]** | docs | ~15 min |
| 13 | docs-health VERIFY pass on the two refreshed baseline paragraphs after the next daemon sweep, confirming nothing reformatted dates/tables | cadence | ~5 min |
| 14 | Annotate THIS report (docs-health ANNOTATE convention) as its items resolve; move to `docs/status/archived/` when fully done | cadence | ~5 min |
| 15 | Harvest the owner-approved subset of items 4-12 into `TODO_LIST.md` (bounded, actionable) and the rest into `ROADMAP.md` (brainstorm routing per docs-health HARVEST rigor) | docs | ~20 min |
| 16 | ~~When the gate (4/6) exists, extend the pre-existing-lint-warning doctrine: baseline failures name the NEW group explicitly in the failure message (group id + files), so triage is one glance~~ DEAD — gate rejected | implementation | ~30 min |
| 17 | ~~If `072e6db`'s `go.mod` delta turns out to be a directive-normalization sweep, verify it matches the documented minor-form policy (`go 1.27`, no patch floor) and close the question~~ RESOLVED 2026-10-24: the delta is a require-line bump `server_timing v1.0.2→v1.0.3`; the `server_timing/v1.0.3` tag exists (real release), not a directive change — kept | follow-on of #2 | ~5 min |
| 18 | Consider whether the two accepted `.go` clone pairs deserve `//art-dupl:accept` directives now that the tool supports them (the HTML showed the mechanism) — would make the shown-count self-documenting in code; **[OWNER]** since it changes the accepted-duplication bookkeeping convention | decision | ~15 min |
| 19 | Fleet cross-check (owner-gated, NOT this session's scope): whether sibling repos record combined-mode verdicts too, or this repo is the first — informs whether the AGENTS.md combined-mode clause should become a fleet convention | decision | — |
| 20 | ~~After any gate lands, add it to `docs/RELEASE.md` gates list so releases cannot skip it~~ DEAD — gate rejected | implementation | ~10 min |

Items 21-50: intentionally not fabricated. The remaining honest backlog of this repo (TODO_LIST, ROADMAP, docs-health cycles) was explicitly out of scope this session; inventing 30 more entries from zero research would be exactly the ghost-work the fleet rules prohibit. If the owner wants the full 50, the next instruction should be a docs-health/BUILD/audit pass, and the list will come from evidence rather than padding.

## g) QUESTIONS I CANNOT FIGURE OUT MYSELF

1. **Gate or prose?** Should the clone baseline become an enforced gate (fail prerelease/CI unless art-dupl shows exactly 2 groups at `-t 1` and 0 at `-t 2`, go-etag style), or stay documented prose refreshed per re-measurement? This is repo policy; I cannot rule it, and it determines items 4/6/16/20.
2. **Was `072e6db` expected?** 11 files swept (9 middleware `.go` + `go.mod` + `flake.lock`, 19+/19-) between the session snapshot and my work. I can read the diff myself, but I cannot know whether it was a deliberate formatter/normalize pass from your side (leave alone) or an unexplained sweep you want audited (item 2). Which is it?
3. **Canonical or ad-hoc?** Does combined mode (`--type-aware --suggest-generics`) become part of the documented baseline procedure in the AGENTS.md Commands block (replacing or augmenting the two `--type-aware` lines), or does it stay an occasional audit lane whose verdict I record per run?

## Addendum: owner rulings + 072e6db audit (2026-10-24, ~09:05 CEST)

**Rulings** (via structured questions): (1) clone baseline stays **prose** — gate rejected; (2) audit `072e6db`; (3) combined mode is the **canonical** baseline command.

**Audit verdict on `072e6db` — forbidden sentinel sweep, recurrence ×4, not benign formatter noise.** The commit applied all 15 erraudit "declare it as the error interface" rewrite hints at once (compression.go ×2, cors.go, csrf.go ×2 incl. exported `ErrCSRFInvalid`/`ErrCSRFConfig`, decompression.go, language.go ×2, metrics.go, ratelimit_keyed.go, requestid.go ×3, server.go ×2). Consequence, exactly as AGENTS.md predicted: `go vet` broke at the documented proof site (`example_test.go:859: ErrCSRFInvalid.WithContext undefined (type error has no field or method WithContext)`), and the state sat **red AND pushed from 08:19 until the revert content landed at 12:04 — ≈3¾ hours**. CI proof that every push inherited the break: the doc-only commits `e6fbd0f` and `2c9bd42` failed the same `Test (1.27.x) | Vet` and `Lint` jobs with the identical vet error. A daemon follow-up (`c2f0b8d`, 11:42) merely gofmt-realigned the annotated declarations — annotations persisted. The `sentinels_type_test.go` guards fire at test time and cannot stop a daemon that never runs tests — `scripts/pre-sweep-check.sh` remains the only pre-push stopgap, and the sweeping session did not run it.

**Series note (episode 2):** `docs/status/2026-10-24_11-34_clone-baseline-replication-and-session-review.md` is a parallel session that ran the SAME three gates blind ~2¾ h after this episode, replicating every count to the digit (370/204/166, 798 shown-2, combined clean). It made no tree changes and independently flagged the still-missing pre-sweep check; this episode's owner-ordered audit closed that gap.

**Fix:** the 9 `.go` files restored from `072e6db^` (every hunk in them was sentinel-only — verified hunk-by-hunk before restoring); `scripts/pre-sweep-check.sh` re-run green (build + vet + both module guards, exit 0). **Kept:** `go.mod` (`server_timing v1.0.2→v1.0.3` — the `server_timing/v1.0.3` tag exists, a real release) and `flake.lock` (routine nixpkgs bump). AGENTS.md's erraudit RECURRENCE line updated ×3→×4 with today's evidence.

**Canonical combined baseline** (now in the AGENTS.md Commands block): `art-dupl --sort total-tokens -t N --type-aware --suggest-generics .` — expected: `-t 2` → 0 shown (re-verified this session: 371 detected, 0 shown), `-t 1` → exactly the two accepted pairs and no generics candidates (per the owner's own run: 798 suppressed). Record the generics-lane verdict with every re-measurement.

**Still open from (f):** 3 (CI on HEAD — confirmed RED through `c2f0b8d`, all failures are the inherited vet break; the fix commit `80fbbde` is committed but **unpushed** — push needs owner go-ahead per harness rules), 7 (single-source the stamp), 8 (re-baseline cadence), 10 (cold type-load note), 11 (warm --timing measurement), 12 (art-dupl version pin provenance), 13 (VERIFY after next daemon sweep), 14 (archive this report when done), 15 (HARVEST remaining items), 18 (accept directives — owner call), 19 (fleet cross-check — owner call).

---

*Report by Crush (glm-5.3-flash) session, 2026-10-24 08:49 CEST. Inputs: user-run art-dupl combined-mode output; two canonical verification runs; art-dupl ADR-0026 + ACTIONABILITY_PATTERNS.md; AGENTS.md + docs/architecture-reference.md diffs. No unrelated research performed, per owner scope. Addendum appended same morning after owner rulings.*
