# Status Report — MIT Release v1.5.0 / server_timing v1.0.2 + CI Repair

**Date:** 2026-10-09 15:39
**Session scope:** resume of the 2026-10-08 MIT/README session → §f follow-through → owner decisions (root MIT, v1.0.2, holder) → coordinated release cut → three hidden CI breakages healed → nightly-fuzz false-crash triage → #31 harness fix. A parallel writer session (docs/review/lint audit series) was active the whole time; its files were left untouched. (Format: user-mandated `.md`, overriding the HTML-canonical status-report default — one-off, not a new default.)

---

## a) FULLY DONE

1. **Resume verification** — all prior-session artifacts intact (`ede6a36`); 4 newer daemon commits attributed to the parallel lint-audit session before any edit.
2. **Stale-license-claim sweep (§f.3, §f.9, §f.10)** — grep-evidenced: only README License section + CONTRIBUTING.md assumed single-license (both fixed); FEATURES/ROADMAP/TODO_LIST/SECURITY/.github/flake.nix clean; zero license/badge assertions in CI.
3. **Mechanical MIT verification (§f.6)** — `server_timing/LICENSE` byte-identical (1069 B) to fleet-canonical (go-etag), holder line included; root LICENSE later swapped from the same canonical bytes.
4. **buildflow markdown-lint + lychee (§f.4)** — both steps green, all 165 md files covered; re-run after later edits.
5. **docs-health VERIFY pass (§f.5)** — zero-dep claim, go 1.27 directive, tag-date predates LICENSE/README, badge↔LICENSE cross-checks all confirmed.
6. **Owner decisions collected** — root goes MIT, `v1.0.2` patch, holder stays "Lars Artmann" (matches all 88 fleet MIT LICENSEs).
7. **CI repair trio** — (a) missing `[1.4.2]` link def + stale `[Unreleased]` fixed (`e70820f`); (b) nightly-fuzz go 1.26.x → 1.27.x + toolchain pin; (c) exact-Go-patch setup-go pins in ci.yml Lint job (`8a592ab`) and release.yml (`93f67d8`) after discovering golangci-lint forces `GOTOOLCHAIN=local` internally.
8. **Nightly-fuzz false-crash triage** — 9 of 10 open bug issues were the infra failure (nightly died pre-fuzz on the go.work floor since 10-01); #32–#40 closed with terse voice-checked comments; root cause quoted from run logs.
9. **Real bug #31 fixed** — `FuzzServerTimingMiddleware` harness panicked on 3 input classes (non-token methods like `"`, non-path-absolute targets like `0`, invalid escapes like `/%`); fixed with `http.ReadRequest`-exact predicates, all pinned as seeds, 19.4M-exec local replay clean; issue closed with the diagnosis.
10. **Coordinated release cut** — `v1.5.0` + `server_timing/v1.0.2`, SSH-signed annotated tags at `19f6a91`; master + both tags pushed in ONE push (the v1.4.2-incident mechanism, now RELEASE.md step 12); CI green on the release commit; prerelease-check.sh ALL gates passed (incl. `nix flake check`).
11. **Release content** — root LICENSE→MIT, README badge+License section, CONTRIBUTING, AGENTS.md decision bullet rewritten (both decisions recorded), CHANGELOG `[1.5.0]` (incl. v1.4.1 correction-of-record), go.mod require bumped to server_timing v1.0.2 via `go mod edit` (replace intact, no go.sum churn).
12. **GitHub Releases** — v1.5.0 created + marked Latest; v1.4.2's missing release healed (its Release workflow had died on the link check); release.yml's frozen-at-tag Lint failure documented and fixed forward.
13. **Consumer verification** — proxy indexed both tags at `19f6a91`; clean-dir `go get` + cross-module compile + `go mod verify` all green.
14. **Docs closeout** — TODO_LIST: stale v1.4.0-release item deleted, owner-gated items added then resolved by the release, header updated; status report 2026-10-08: §f items 1–15 ALL resolved inline (10 earlier + 5 release), §g 1–3 answered; 2026-09-15 report item 29 (the original pkg.go.dev license question) annotated with the answer.

## b) PARTIALLY DONE

1. **pkg.go.dev render verification** — proxy + consumer resolution verified; the actual page render (MIT badge, README, docs) still 404 at session end (index sync lag, ~30 min typical). The CHANGELOG claim is mechanism-verified (MIT + tag exists) but render-unconfirmed.
2. **`a0503a1` (annotations + TODO_LIST closeout)** — committed locally, **unpushed** (origin 1 behind); no CI run on it. Caught by this self-review, not by the release flow.
3. **RELEASE.md gate 6 (benchmark baseline doc)** — gates ran green, but `docs/benchmarks.md` not refreshed (header still "Measured 2026-09-11"); defensible for a zero-production-code-change release, yet stale against the runbook cadence since v1.4.0.
4. **RELEASE.md step 9 (historical-report sweep)** — license-themed open claims annotated; not an exhaustive open-claims sweep.

## c) NOT STARTED

1. pkg.go.dev post-sync visual confirmation (+ `/fetch` trigger if needed).
2. golangci-lint version bump (first release built with go1.27.2) to retire the exact-Go-pin workaround.
3. Nightly-fuzz issue dedupe (9 duplicate issues over 9 days is the failure signature).
4. FEATURES.md freshness pass — `_Updated: 2026-09-27` header predates v1.4.x/v1.5.0; server_timing README/LICENSE not inventoried.
5. Coverage-badge freshness check (CI computes it but `contents: read` prevents committing; may need a local `update-coverage-badge.sh` run).
6. Coordination with the parallel lint-audit session — `docs/review/lint/011-stale-httputil-versions.md` is likely affected by v1.5.0's existence.
7. /tmp scratch cleanup (release-verify module, nightly log zip+dir, close-comment drafts).

## d) TOTALLY FUCKED UP!

1. **`gh release create --generate-release-notes` — wrong flag, twice, errors piped away.** The flag is `--generate-notes`; both attempts (v1.4.2, v1.5.0) failed with the error hidden by `| tail -2`, and I proceeded as if the releases existed. Caught only because I verified `gh release list`. The github-voice skill documents this exact silent-failure class for `gh` posting — I walked into a variant of it anyway.
2. **Shipped a fix I didn't understand mechanically.** The first lint "fix" (job-env `GOTOOLCHAIN=go1.27.1`) was a no-op: golangci-lint forces `GOTOOLCHAIN=local` internally, so the env never reached it. Cost: one red CI cycle + a cleanup edit removing my own misleading comment. I copied the test job's pattern without tracing how the value travels into the linter binary.
3. **Pushed master while the fix sat uncommitted.** First unblock push shipped without the CHANGELOG/nightly edits (daemon hadn't committed them yet). Caught by post-push `git status`; recovered with a deliberate commit.
4. **Predicted-not-prevented lychee failure.** The `[1.5.0]:` compare links were knowably-unresolvable at edit time (tag didn't exist yet); I derived the push-together requirement only AFTER a red CI run paid for the lesson.
5. **Declared done before verifying the tail.** The session-end summary said "everything committed" while `a0503a1` was unpushed and CI-unverified. The self-review caught it; the release flow should have.

## e) WHAT WE SHOULD IMPROVE!

1. **Verify artifacts, not command exits** — every externally-visible artifact (release, issue, tag) gets an immediate existence check; never pipe release-critical commands through filters that eat errors.
2. **Mechanism-before-fix** — when transplanting a config pattern to a new context, trace HOW the value reaches the consumer before pushing it.
3. **Pre-push invariant** — `git status` + `git log origin/master..master` immediately before every push; daemon commit timing is not a guarantee.
4. **Chicken-and-egg anticipation** — any link/ref pointing at a not-yet-existing git ref will fail CI; derive push-order requirements at edit time, not from red runs.
5. **Session-tail discipline** — before declaring done: confirm last commit pushed, CI green on it, tree clean. The tail is exactly where daemon/parallel-writer races bite.
6. **Kept (good, preserve):** master+tags single-push release mechanics; coordinated same-commit tags; tag immutability respected (manual GH release instead of re-tag); harness root-cause fix instead of seed deletion; question-tool batch for owner decisions; deliberate commits for externally-cited hashes; parallel-writer file boundaries respected.

## f) Things we should get done next (priority order)

1. **Push `a0503a1`** and confirm CI green on it.
2. **Verify pkg.go.dev renders both pages post-sync** (MIT badge, README, docs visible; hit `/fetch` if the page stalls).
3. **Coordinate with the parallel lint-audit session** — `docs/review/lint/011-stale-httputil-versions.md` needs v1.5.0 context (consumers can now bump to a MIT-licensed, fully-documented version).
4. **Coverage-badge freshness** — compare CI's coverage artifact vs README badge; run `update-coverage-badge.sh` locally if drifted.
5. **golangci-lint bump evaluation** — first release built with go1.27.2 retires the exact-Go-pin workaround in both workflows (fleet-wide: BuildFlow pins lint versions too).
6. **Nightly-fuzz issue dedupe** — search-open-before-create (or a rolling issue + auto-close-on-green); 9 duplicates in 9 days is the signature.
7. **FEATURES.md freshness pass** — v1.4.x/v1.5.0 rows (server_timing README + LICENSE/MIT facts), refresh `_Updated_` header.
8. **docs/benchmarks.md refresh** — per gate-6 cadence (`nix run .#bench`, 3s×5 protocol).
9. **CI guard for the nightly-vs-floor skew** — a fail-fast check comparing go.work's floor with the nightly's pinned Go would have caught the 9-day breakage on day one.
10. **AGENTS.md size reduction** — doctor warns 297 > 220 lines; move CI-workflow detail into RELEASE.md, keep AGENTS pointers.
11. **Watch the first green nightly post-fix** — all 26 targets re-exercise for the first time since 2026-09-30; triage anything new.
12. **Add the Go-pin bump ritual to RELEASE.md pre-tag checklist** — the exact-pin must be right BEFORE tagging (workflow frozen at tag; the v1.5.0 Release run died on exactly this).
13. **docs-health HARVEST this report** — f-items 1–11 are the actionable subset.
14. **/tmp scratch cleanup** via `trash`.
15. **Optional: GitHub Release object for `server_timing/v1.0.2`** (currently tags-only by precedent).
16. **Optional: ROADMAP note** that the relicensing shipped (only if ROADMAP carries license-vision items — none found in the sweep).

## g) Questions I can NOT figure out myself

1. **golangci-lint pin strategy:** bump to the first release built with go1.27.2 now (removes the brittle exact-Go-patch pins from both workflows), or keep v2.13.2 + exact pins until a feature-relevant golangci release? This ripples fleet-wide (BuildFlow's own lint pins).
2. **Nightly-fuzz issue policy:** switch to deduplicated/rolling issue reporting (one open issue per underlying failure, auto-closed on a green run), or keep one-issue-per-night with a triage label?
3. **`server_timing` release surface:** keep tags-only (current precedent — the sub-module has never had a GitHub Release object), or start cutting release objects for sub-module tags too?

---

_Point-in-time snapshot. Annotate, don't rewrite, when this goes stale._
