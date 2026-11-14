# Status Report — Session 2: §f Verification Sweep, endpoints Accept-Line Join, Fleet Lint-Pin Ripple

**Date:** 2026-11-14 11:27 CET
**Session scope:** continuation of the 09:46 report (erraudit sweep ×6 restore + go1.27.4 pins). The user pasted TODO_LIST.md with a WORK-THE-LIST directive ("break down, execute, verify, repeat until done"). I executed the report's §f verification menu, the TODO_LIST's two actionable non-owner rows (endpoints Accept posture, fleet lint-pin ripple), and the §f.22 fleet toolchain audit. Interrupted mid-library-policy-audit by this report request; the dangling thread was closed by re-run before writing.
**Parallel writers:** the owner PUSHED mid-session — origin/master tip is now `952351a` (a 2-file heuristic commit that is NOT mine, sitting on top of the previously-unpushed `e735984`/`f2e8c93`/`c260426`); the three restore-session commits are therefore ON ORIGIN now. The go-etag and go-error-family daemons swept my fleet edits into `f8c9322` (3 files) and `98e9e93` (4 files) — file lists match my edits exactly, attribution clean. A BuildFlow binary rebuild (`buildflow/internal/cli` compile) was observed running early in the session; I stayed off that lane.

## Scope line — what this session verified vs. did NOT verify

**VERIFIED by command this session** (exit status noted):
- `erraudit lint ./... --max` (exit 2 = advisory findings, expected): ledger re-derived at **exactly 57** — 36 load-bearing + 17 hints + 3 `enforce_generic_return` (endpoints.go:111 + scripts/ ×2) + 1 `legacy_is`; matches the AGENTS "NEXT BUILD" line; stamped into AGENTS.md. (The local binary is newer than fa5575f — messages now say `//nolint:legacyerrors` — and the count held across the restore.)
- `GOTOOLCHAIN=go1.27.4`: root build, root + httpspec race tests, server_timing race tests, endpoints race tests — ALL green; `GOTOOLCHAIN=go1.27.4 go version` → **go1.27.4** confirmed (the toolchain actually resolved, not silently ignored). §f.3 closed: the shipped CI pins compile-and-pass locally.
- Fuzz smoke under go1.27.4: `FuzzParseAcceptLanguage`, 15s, ~1.3M execs, PASS (§f.31 by its own one-target definition).
- `govulncheck ./...` ×3 modules (exit 3 = findings): GO-2026-6859 found in `net/http@go1.27.3` in ALL THREE (sub-modules via stdlib traces only), "no other vulnerabilities" in each; fixed-in `net/http@go1.27.4`. Measured state recorded; CI is protected by the pins.
- `nix flake update nixpkgs` + `--refresh` (exit 0, **no lock change**): locked rev `d4f25a3` (2026-11-11) IS the current gated nixos-unstable HEAD; `nix eval` at BOTH the locked rev and the registry head shows **go_1_27 = 1.27.3** — go1.27.4 has NOT landed in nixpkgs. The §g-3 question from the prior report resolves itself: there is nothing to bump; the local toolchain is blocked upstream, not by lane coordination. AGENTS.md toolchain note rewritten with the evidence and the exact re-run steps.
- `nix fmt` (154 files, 0 changed after my Go edits; 3 formatter-passed) and **`nix flake check`: all 6 checks PASSED** (treefmt-check, gowork-off builds ×2, vendor-hash, dirty tree builds) — §f.7 closed, and the vendorHashes are re-proven on the current lock.
- `./result` symlink inspected (recon): → `httputil-f2e8c93` build — the vulnix staleness assumption from the prior report converted to a check (§f.8).
- v1.9.0 tag hygiene (§f.23): `git merge-base --is-ancestor 850b3e5 v1.9.0` → **NOT an ancestor**; the tagged tree is pre-sweep, tag clean, no action (immutability policy).
- endpoints multi-Accept fix: `go test -race -count=1` green, `golangci-lint run` 0 issues (after one golines auto-fix), `go vet` clean, `check-qvalue-corpus-sync.sh` OK + corpus test green; **mutation probe: all 3 new tests FAIL under the old `Header.Get` behavior and pass restored**.
- go-etag: `cue vet -d '#CIWorkflow' ../cue-policy` OK; `golangci-lint run` green on all FIVE modules (root, client, entitytag, metrics, server) with v2.15.0 + new sigs; `go build` ×5 green; **mutation probe done right: in `server/` (the module holding the 4 clone call sites) without the sigs → exit 1 with 6 `WithContextf` findings; restored → green**.
- go-error-family: `cue vet` ×2 (`#CIWorkflow` + `#ReleaseWorkflow`) OK after the pin bumps; lint green on root + agent + bridge + diagnose + examples (the examples module was RED until the two package-level func sigs landed — see b/d).
- library-policy audit (re-run to close the dangled thread): its CI is **Nix-based** (`nix flake check`/`nix build` steps only; zero golangci mentions in ci.yml; zero `GOTOOLCHAIN` anywhere) and it has **0 errorfamily clone call sites** — nothing to port, no workflow pin to bump.
- Git forensics: daemon commit `e735984` = the prior session's report + TODO row (verified stat); owner's push = origin tip `952351a`; my two fleet daemon commits verified by file list.

**NOT verified, and why:**
- **Full `buildflow --build-mode=full` re-run** (§f.2 — the loop-closer from the originally-failed command) — NOT STARTED; I sequenced verification + the two actionable TODO rows ahead of it and the report request landed first.
- **CI on the pushed head** (`952351a`) — the go1.27.4 pins' first live CI run; I cannot watch CI without push/CI access being exercised, and the run state is unknown to me.
- **Mutation probe for go-error-family's four METHOD sigs** — only the package-level func sigs (`WrapRejection`/`WrapTransient`) were live-proven (examples failed without them). Root is green WITH the sigs; green-without is unproven (error.go:234's `.WithContext` is the only non-test method call site).
- **go-error-family `git` and `postgres` submodule lint jobs** — submodules not initialized locally; not runnable.
- **httputil session tail** — 6 modified files await the daemon; `pre-sweep-check.sh` not re-run after the latest doc edits; the 2 newest daemon commits (`faed81e`, `9eff8df`) are unpushed.
- **meta inventory row** — NOT STARTED; discovered `meta` is the `project-meta` CLI with per-repo `.config/metadata.yaml`; how multi-module repos (go-cqrs-lite precedent) are represented was not researched.
- **Nightly-fuzz breadth** — one target smoked, not the full rotation.
- **BuildFlow binary freshness** — the rebuild I observed was never re-checked (`buildflow --version` reports "dev", unusable for the freshness comparison anyway).
- **httputil root-module tests after the doc-only edits** — not re-run (docs don't affect code; the flake check's dirty-tree build compiled everything, which mitigates but is not the test suite).

## a) FULLY DONE

| Work | Evidence |
| --- | --- |
| erraudit `--max` ledger re-derived post-restore: 57 held exactly (36+17+3+1) | /tmp/erraudit-max.txt; AGENTS.md stamped "RE-CONFIRMED 2026-11-14" |
| go1.27.4 CI-pin smoke: build + race tests across root/httpspec/server_timing/endpoints | ALL-GO1274-SMOKE-OK; `go version` → go1.27.4 |
| Fuzz smoke under go1.27.4 (FuzzParseAcceptLanguage, 15s) | FUZZ-SMOKE-OK, ~1.3M execs |
| govulncheck ×3: GO-2026-6859 measured present vs go1.27.3, nothing else | exit 3 ×3, "no other vulnerabilities" ×3, fixed-in 1.27.4 recorded |
| nixpkgs-bump question resolved as externally blocked (evidence, not assumption) | `--refresh` no-op; go_1_27 = 1.27.3 at locked rev AND registry head; AGENTS note rewritten |
| `nix flake check` — all 6 checks green on the current lock | flake-check-exit=0, "all checks passed" |
| `./result` freshness converted from assumption to check | → /nix/store/…httputil-f2e8c93… |
| v1.9.0 tag verified clean of the sweep commit | merge-base ancestry check |
| **endpoints: multiple `Accept` header lines now combined (RFC 9110 § 5.3)** | `combinedAccept` in endpoints.go; `negotiateRepresentation(combinedAccept(req))` call site |
| endpoints: 3 new standalone tests, each mutation-probed to FAIL under the old first-line-only behavior | `TestHandler_MultipleAcceptLines_{SecondLineQValueSelectsHTML,LineOrderDoesNotChangeSelection,TieBreaksTowardJSON}` |
| endpoints doc sweep: README negotiation paragraph, package doc, Handler godoc | all mention the § 5.3 combining |
| Root doc sweep: AGENTS.md endpoints paragraph, architecture-reference negotiate.go row + § 5.3 clause | grep-verified |
| **Stale architecture-reference fix (found on sight): "No released tag yet" → released v0.1.0 + apidiff baseline** | arch-ref line 88 (was contradicting line 288 since 2026-11-06) |
| CHANGELOG [Unreleased] `### Fixed` entry for the Accept-line join | written |
| **go-etag fleet ripple complete**: +4 method sigs, pin v2.14.0→v2.15.0, GOTOOLCHAIN ×4 → go1.27.4, rationale comment updated, CHANGELOG entry | cue vet OK; lint ×5 modules green; build ×5; server/-scope mutation probe (6 findings → green) |
| **go-error-family fleet ripple complete**: +6 sigs (4 methods + 2 package-level funcs), action pins v2.13.2→v2.15.0 ×10 sites, CHANGELOG entry | cue vet ×2 OK; lint green ×5 local modules; examples RED→GREEN proves the func sigs live |
| library-policy audit verdict: nix-based CI, 0 call sites → nothing to port | ci.yml inspected (nix flake check/build steps only) |
| FEATURES.md inline-code plural form → `Spec` checks fix-on-sight (§f.15, NOT T6-blocked) | FEATURES.md:214 |
| Fleet git states documented: GEF `98e9e93`, go-etag `f8c9322` (daemon commits of my edits); owner push discovered | git log ×4 repos |

## b) PARTIALLY DONE

| Work | Done | Missing |
| --- | --- | --- |
| Fleet lint-pin ripple (TODO row) | go-etag + go-error-family shipped and verified | library-policy = nix-managed lint (no workflow pin exists — row scope narrows to "no action"); "other errorfamily consumers" in ~/projects not yet enumerated |
| Fleet GOTOOLCHAIN audit (§f.22) | go-etag's 4 pins bumped to go1.27.4 | GEF and library-policy have NO GOTOOLCHAIN pins (nothing to bump — verdict recorded); non-wave-1 repos not audited |
| Session harvest | CHANGELOG/AGENTS/arch-ref/features updated in-tree | TODO_LIST rows not struck/updated (Accept row is done; ripple row narrowed); "Updated:" header not refreshed; report annotations not applied |
| httputil session tail | tree builds (flake check dirty build) | 6 files uncommitted (daemon pending); `faed81e`+`9eff8df` unpushed; pre-sweep re-run pending; CI unwatched |
| go-error-family sig verification | func sigs live-proven via examples RED→GREEN | method sigs not mutation-probed; git/postgres submodules not locally runnable |
| BuildFlow binary freshness | rebuild observed running | never re-checked; `--version` returns "dev" so the AGENTS freshness check can't apply as written |
| Nightly-fuzz readiness | 1 target green under go1.27.4 | full rotation untested locally |
| Owner push of the restore commits | 3 commits + owner's `952351a` on origin | CI state on that head unknown; my 2 new commits not yet pushed |

## c) NOT STARTED

- Full `buildflow --build-mode=full` re-run (§f.2 — the original failing command's end-to-end proof)
- meta inventory: record endpoints (needs project-meta multi-module research first)
- Enumeration + porting to "other errorfamily consumers" in ~/projects
- README MaxBytesReader consumer-security-note placement decision (§f.24)
- `buildflow timings --regressions` review (§f.10)
- T6-blocked docs campaign (annotate 229; cadence now 49 unarchived with this report)
- TODO_LIST "Updated:" header refresh for this session (part of harvest)
- Daemon-commit consolidation (optional; now interleaved with the owner's push — do NOT consolidate across the owner's `952351a`)
- Push of `faed81e`+`9eff8df` and the two fleet commits — owner-gated

## d) TOTALLY FUCKED UP

Nothing this session did is irreversible, and every claim above carries its command evidence — but the misses are real:

1. **The go-error-family workflow bump was a blind bulk edit.** I `sed`'d `version: v2.13.2 → v2.15.0` expecting 2+3 sites and got 7+3 — I audited all ten AFTER the fact (they were all golangci-lint-action inputs, so the outcome was correct), but the correct-ness was luck plus post-hoc verification, not intent. Count-then-edit, not edit-then-explain.
2. **The first go-etag mutation probe was vacuous** — run from the repo root while the clone call sites live in the `server/` sub-module, so it returned a meaningless green that could have been cited as evidence. Caught it because the "0 errorfamily mentions" felt wrong for a repo with call sites; redid it in the right scope (exit 1, 6 findings). A green probe should always be suspected until it has demonstrably been able to fail.
3. **Two edit-tool round-trips wasted on read-before-edit and same-string slips** (GEF `.golangci.yml`, go-etag CHANGELOG) — no damage, but sloppy sequencing under momentum.
4. **The dangled library-policy thread**: the audit was interrupted mid-grep and sat unresolved until this report forced the re-run. Small, but the pattern (start a thread, switch context, reconstruct later) is the exact discipline the session-tail rules warn about.
5. **Inherited and still open: the full buildflow re-run** — the user's original loop-closer from the failed `--fix --build-mode=full` run remains unexecuted after two sessions. I prioritized verification and TODO-row work; defensible, but the loop is still open and I own that.
6. **arch-ref:88's stale "No released tag yet" survived 8 days** (since the 2026-11-06 release) and was found by accident while sweeping for THIS change — the docs-gates audit-all red exists precisely for this class and stays blocked on T6.

## e) WHAT WE SHOULD IMPROVE

1. **Count-then-edit for bulk sed.** Enumerate and review every match site BEFORE the edit lands; the 7-site surprise cost nothing this time only because every site was the same action input.
2. **Self-test every mutation probe.** A probe pass in the wrong scope is worse than no probe — it manufactures false evidence. Assert the probe CAN fail (or run it where the claim lives) before recording the verdict.
3. **Collect background jobs before writing any summary.** The flake check had been sitting completed for ~15 minutes uncollected; had it failed, the whole nix section of this report would have been wrong. Check → then write.
4. **Close dangled threads in the same session** — or record them explicitly as open in the notes the next reader will see (this report's §c now does that for the buildflow run).
5. **Fleet edits deserve per-repo AGENTS.md notes where those repos keep such context** — I wrote CHANGELOGs only; go-etag/GEF AGENTS conventions weren't checked for pin-bump note expectations.
6. **The nixpkgs go1.27.4 block is fleet-wide** — every repo's LOCAL toolchain stalls at go1.27.3 until upstream packages it; a one-line standing note (fleet-level) prevents every repo re-discovering the no-op `nix flake update` independently.
7. **What worked, keep doing:** verify-before-claiming (ledger count, toolchain version, symlink target), mutation-probes in both directions, snapshot discipline, and leaving owner-gated rows strictly alone despite "keep going until done".

## f) Up to 50 things we should get done next

Impact-ordered; R = TODO_LIST candidate, O = owner-gated, U = upstream/other-repo, V = verification/cheap, D = deferred-by-design.

1. **[V] Full `buildflow --build-mode=full` re-run** on the current tree — close the original loop (§f.2, two sessions open).
2. **[O] Watch CI on origin head `952351a`** — the go1.27.4 pins' first live run; if red, check the golangci-lint/GOTOOLCHAIN interaction first.
3. **[V] Push `faed81e`+`9eff8df` after harvest** (this session's ledger stamps + endpoints fix) — owner-gated, but the tail discipline is open until pushed.
4. **[R] TODO_LIST harvest**: strike the Accept-headers row (done, in CHANGELOG), narrow the lint-ripple row (wave-1 complete except library-policy=no-action; remaining scope = other consumers), refresh docs-gates counts (49 unarchived now), refresh the "Updated:" header.
5. **[R] Annotate this report + the 09:46 report** per cadence (strike completed §f items with done-at markers).
6. **[U] meta inventory**: research how project-meta represents sub-modules (go-cqrs-lite precedent), then record `endpoints`.
7. **[U] Enumerate remaining errorfamily consumers** in ~/projects (grep go.mod files) and port sigs where call sites + lint pins co-exist.
8. **[V] Mutation-probe go-error-family's four method sigs** (error.go:234) for probe-parity with go-etag.
9. **[V] Re-run `scripts/pre-sweep-check.sh`** after the daemon sweeps this session's files.
10. **[R] README MaxBytesReader consumer note** — placement decision (§f.24): README security note vs CHANGELOG-only guidance.
11. **[D] nixpkgs re-run when upstream ships go1.27.4** — `nix flake update nixpkgs` (expect a real rev change) → `buildflow -s nix-hash-fix --fix`; AGENTS note carries the steps.
12. **[V] `buildflow timings --regressions`** on the next green run (§f.10 — was the +N% contention or corruption?).
13. **[U] Verify the BuildFlow binary rebuild** that was observed running (freshness verdict post-rebuild).
14. **[R] Standing fleet note: local toolchains stall at go1.27.3 until nixpkgs ships 1.27.4** — prevents every repo re-deriving the no-op bump.
15. **[V] Confirm `faed81e`/`9eff8df` file lists match exactly this session's edits** (attribution vs the owner's `952351a` — never cite heuristic commits externally either way).
16. **[R] endpoints fuzz corpus: add multi-line Accept seed entries** so the join behavior stays oracle-covered at handler level (currently only unit-probed).
17. **[O] Push authority for go-etag `f8c9322` and GEF `98e9e93`** — the fleet bumps sit unpushed; their CI (incl. the new v2.15 pins) is unwatched.
18. **[R] Check whether go-etag/GEF AGENTS.md conventions expect a pin-bump note** (CHANGELOG-only may be under-documented by their own rules).
19. **[V] `nix flake check --all-systems`** — today's run omitted aarch64-darwin/aarch64-linux; the fleet has mac members (AGENTS says safe, 2026-10-10 — re-confirm on the current lock).
20. **[O] The T6 ruling** — unchanged blocker for ~200 annotate findings + the archive ritual (now 49 reports).
21. **[R] file-size-check owner verdict** (standing TODO row — 36 findings, policy-reject triage awaiting confirmation).
22. **[V] Verify the OWNER's `952351a` doesn't collide with this session's files** (2 changed files — read-only curiosity, attribution hygiene).
23. **[D] dprint remote-plugin local wasm paths** (§f.27 carried) and **[D] shellcheck+vulnix in devShell** (§f.28) — unchanged.
24. **[R] AGENTS.md size budget ruling** (standing row T7 — unchanged).
25. **[V] Nightly-fuzz breadth under go1.27.4** — run the full local rotation once before tonight's scheduled run hits the new pin.

## g) Questions I cannot figure out myself

1. **Push + CI watch:** you pushed the restore commits (+`952351a`) mid-session — should I push the two new daemon commits (`faed81e`, `9eff8df`: the Accept-line join + ledger stamps) and watch CI on the go1.27.4-pinned head? If it goes red, my first suspect is the golangci-lint/GOTOOLCHAIN interaction; I'd want to be the one holding that page.
2. **Fleet push authority:** go-etag (`f8c9322`) and go-error-family (`98e9e93`) carry the verified v2.15.0 + go1.27.4 bumps, unpushed and CI-unwatched. Push them now, or do you review first? (Both verified locally: cue vet, lint green ×5 modules each, mutation probes.)
3. **Fleet scope for the remaining ripple:** shall I enumerate the other errorfamily consumers in ~/projects and port the sigs proactively this week, or let each repo hit the wave on its own next pin bump? The first is N repos touched today; the second is zero risk but guarantees a repeat of this triage per repo.

---

*Historical record — a)/g) sections and the timeline are not struck later; follow-ups live in TODO_LIST/ROADMAP via HARVEST.*
