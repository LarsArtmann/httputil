# Session Status — Regression Test, Binary Rebuild, Vendor Decisions, and the Tail End of the BuildFlow Recovery

**Date:** 2026-09-11 06:16 (Friday)
**Session start:** ~06:00 (resumed from `2026-09-11_05-37_buildflow-recovery-link-debt-and-vendor-parser-fix.md`)
**Scope:** The one genuinely incomplete piece of the prior session — BuildFlow regression test + binary rebuild — plus per-step/full pipeline verification, documentation debt, and the three open decisions. Asked to self-review brutally at the end.

---

## a) FULLY DONE

1. **BuildFlow regression tests for the workspace-vendor parser fix** — two subtests added inside `TestCheckVendorConsistency` (`modules/gomod-checker/checker_godebug_test.go`):
   - `negative_workspace_explicit_marker_with_toolchain_suffix` — `## workspace` header + `## explicit; go 1.26` marker (the exact incident shape).
   - `negative_workspace_replace_suffixed_header` — `# mod v1.0.1 => ./sub` replace-suffixed header (the server_timing shape).
   - **Mutation-verified, not just written**: parser temporarily reverted → both subtests fail with the _exact_ original false-positive message ("is explicitly required in go.mod, but not marked as explicit in vendor/modules.txt") → fix restored → full `TestCheckVendorConsistency` green. The test provably catches the bug class it exists for.
   - Committed by the daemon as `742f89610`.
2. **Full BuildFlow suite** — `go build ./...` + `go test ./...` across the workspace: green (only `[no test files]` notices).
3. **Binary rebuilt and the global copy actually refreshed** — `nix build .` + discovered that `nix run .#reinstall` **only refreshes `./result`**; the global `~/.local/bin/buildflow` is a plain copy (not a symlink) and needed `cp result/bin/buildflow ~/.local/bin/buildflow`. Verified: `buildflow --version` = `984dac6` = BuildFlow HEAD at build time.
4. **Per-step re-runs of every previously-failed step** — `gomod-check`, `golangci-lint`, `govalid-generate`, `test-coverage`, `markdown-lint`, `shellcheck`: all **exit 0**, one `-s` per invocation. The "not marked as explicit" false positives are gone; gomod-check reports exactly one info finding (see d/3).
5. **Full pipeline green** — `buildflow --build-mode dev` → **exit 0, 97 success / 0 failed**, +3 skipped via config. (Original incident: exit 1, 45/64.)
6. **The three open questions from the prior report — decided, executed, documented**:
   - _Rebuild before or after the test?_ Test first — executed in that order.
   - _Silence opinion tools?_ Split decision: `go-auto-upgrade` stays skipped (advice unconditionally forbidden by depguard); branching-flow/erraudit/go-structure-linter stay **visible** (detect-only, never fail, blanket-skipping would hide new genuine findings mixed into the noise).
   - ~~_Keep committed `vendor/`?_ **Yes — deliberate.** Load-bearing for the `GOWORK=off` CI gates; mkPreparedSource solves a problem this repo doesn't have (public deps, flake runs checks not hermetic Go packages).~~ **Superseded same day (owner decision): `vendor/` deleted.** It was never committed (gitignored on-disk artifact), and workspace + `GOWORK=off` builds verify green from the module cache — see ROADMAP.md Non-goals and AGENTS.md BuildFlow Pipeline.
7. **Vendor-format facts established empirically** (scratch-copy, `/tmp`, cleaned up after):
   - go1.26.7 writes `## explicit; go <ver>` annotations under **both** `go mod vendor` and `go work vendor` — the parser fix was needed for both, not just workspace mode.
   - `go work vendor` output omits the trailing versionless replace-marking header (`# mod => ./path`), so forced `GOWORK=off go build -mod=vendor` fails with "not marked as replaced" — **pre-existing, latent, no gate uses `-mod=vendor`** (A/B tested: fails identically with and without any `ignore` directive — the failure is format-inherent, not caused by anything recent).
   - ~~`go mod vendor` format builds green in BOTH modes but is churn-reverted by the next `go work vendor` → workspace format stays canonical~~ mooted same day with the vendor/ removal — no vendor artifact exists to have a format; the module cache is canonical (ROADMAP Non-goals).
   - `ignore ./vendor`: A/B-tested functionally neutral, **deliberately not added** — real top-level `vendor/` is special-cased by the toolchain; convention in this ecosystem ignores only non-Go dirs (`ignore ./node_modules`); the author's own repos never ignore their real vendor dir.
8. **AGENTS.md** — new "BuildFlow Pipeline" section: invocation quirks (`--build-mode dev`, single `-s` per run), stale-global-binary trap with the exact `cp` command, workspace-vendor marker format + incident history, forced `-mod=vendor` quirk, accepted `vendor` info finding, "residual detect-only findings are policy-rejected, not debt" with pointers to the decision docs.
9. **CHANGELOG `[Unreleased]`** — Added (lint configs + AGENTS section), Changed (`apps.test-race` duplicate removal), Fixed (shellcheck SC2164/SC2046, badge-generator MD042 fixed generator+output together, README/SECURITY markdown defects, 35 archived-doc link rewrites, lychee 113→0).
10. **Prior status report annotated** — the three questions in section g) struck through with resolutions + a resolution addendum (house convention).
11. **Final gates** — `nix fmt` (0 changed), `nix flake check` (all checks passed), `buildflow -s markdown-lint` re-run **after** all doc edits (exit 0). Repo diff at end: exactly the 3 doc files (AGENTS.md, CHANGELOG.md, status report) — daemon will commit.

## b) PARTIALLY DONE

1. **Binary freshness** — the binary matches the tree _as of my rebuild_ (`984dac6`), but concurrent in-flight work landed in BuildFlow afterwards (`bcc60a687`, `29f9c20cc` + uncommitted `fail_on` feature files). By my own documented "stale-binary trap", the global binary is already one feature behind. One `nix build . && cp` refresh pending the moment that work settles (see g/1).
2. **`buildflow dev` full-mode link checking** — the dev-mode log shows the lychee step falling back to `nix develop -c` and then failing with `exec: lychee: not found` (plus a markdownlint eval-cache "busy" warning, marked ignored). Steps still counted success. So the individual `markdown-lint`/lychee verifications are solid, but inside a full dev run the link check may be silently no-opping. Root cause not yet chased (is lychee missing from the httputil devShell, or is BuildFlow's fallback broken?). See e/3, g/2.
3. **Detect-only noise ledger** — the decision to keep opinion findings visible is documented in AGENTS.md, but there is no machine-readable sign-off (config/ledger) that marks the ~180 residual findings as reviewed. Every future run summary will re-print them and every future session must re-read AGENTS.md to know they're intentional. Half-solved by documentation only.

## c) NOT STARTED

1. **`skip_steps` single-step-mode false warning fix** (BuildFlow) — `-s <step>` warns skip entries "match no registered tool" because `detectUnknownSkipTools` runs against the step-filtered registry, not the full one. Prior session flagged it as a candidate upstream fix; this session did not touch it.
2. **`nix run .#reinstall` gap** (BuildFlow) — it refreshes only `./result` while the doctor's own stale-binary warning tells users to run it. The app should either install to the real global location or the doctor's advice should match reality. I hit this trap personally; fixing the tool was out of this session's scope.
3. **buildflow DB VACUUM** (1.09 GB, prior session's note) — not touched, not re-measured.
4. **`go-licenses` missing from PATH** (doctor info) — lives in BuildFlow's devShell; not surfaced into the httputil flow.
5. **BuildFlow `execution` module tidy drift** (`dave/dst` v0.27.4 → v0.28.0 indirect) — pre-existing, noticed, deliberately not fixed: an indirect bump cascades into vendor + vendorHash + rebuild in the user's tool repo and wasn't mine to churn. Flagged only.
6. **server_timing sub-module gates this session** — not re-run (prior session verified; nothing this session touched Go code anywhere). The full dev run's `test-coverage` exercises root + httpspec + scripts/coverage-threshold, not the separate server_timing module.

## d) TOTALLY FUCKED UP

Nothing destructive. Honest missteps, ranked:

1. **`buildflow dev` invocation** — exit 69 "Unknown command dev" on the first full run. I trusted the prior session summary's phrasing instead of checking `--help` first (where `dev` is clearly listed as a build MODE). One wasted full-run attempt; also means the summary's own "Run a full `buildflow dev`" instruction was a landmine for any future reader — now defused in AGENTS.md.
2. **Mutation-check sed failures ×2** — delimiter collisions (`|`, then `,`, both appear in the Go expression) before `%` worked. Should have picked a collision-free delimiter or used the edit tool with a backup from the start. No damage (grep proved the file was untouched each time), just two wasted cycles.
3. **Under-reported live findings in the dev log** — I saw `lychee: not found` and the markdownlint eval-cache warning scroll past in the success run and moved on because exits were 0. "Exit 0 with a silently skipped tool" is exactly the green-banner-masking class the global AGENTS warns about. It belonged in the in-flight report, not just the retrospective. (Caught here; tracked in b/2.)
4. **erraudit count drift dismissed as "variance"** — 55 → 56 findings between the dry-run and the dev run. I hand-waved it. Probably tool-version or code-path differences, but one line of diffing would have turned "probably" into "certainly". Minor, still sloppy.

## e) WHAT WE SHOULD IMPROVE

1. **Verify CLI syntax before trusting session summaries.** The prior summary said "buildflow dev"; the binary says otherwise. Session handoff notes are documentation, not APIs — `--help` is cheap.
2. **A green exit is not a green tool.** Success-with-skipped-tool (lychee in dev mode) should be treated as a finding the moment it appears in a log we're reading, not discovered in a retrospective. BuildFlow could help by failing or loudly warning when a detect step's binary is missing instead of counting it as success.
3. **Make `#reinstall` reinstall.** The trap I documented for humans is better fixed once in BuildFlow: `nix run .#reinstall` should refresh the global binary (or doctor's advice should stop recommending a command that doesn't do what it says).
4. **Number drift deserves a diff, not a shrug.** Any count that changes between two runs of the "same" gate gets a one-command explanation, always.
5. **The `ignore ./vendor` investigation was the right kind of tangent** (it converted question 3 from opinion to evidence and surfaced the pre-existing `-mod=vendor` failure) — but it should have been time-boxed explicitly. It grew to three scratch experiments; two would have answered it.
6. **Mutation-verification of regression tests should be the default**, not the exception. It cost ~3 minutes and converted "I wrote a test" into "the test catches the bug". Keep doing this for every regression test written in either repo.

## f) NEXT (ranked, not padded to 50)

**httputil — pipeline & docs**

1. Chase the dev-mode lychee no-op: is lychee in the httputil devShell? Fix flake or BuildFlow fallback (see g/2).
2. Re-run `buildflow -s lychee` after any devShell change; pin the 301-link/0-error baseline somewhere checkable.
3. Consider a findings sign-off ledger (or buildflow config) marking the ~180 reviewed detect-only findings so run summaries stop re-printing them (see g/3).
4. Add `buildflow --build-mode dev` (exit 0) as an explicit item in `scripts/prerelease-check.sh` or docs/RELEASE.md gates, if desired.
5. server_timing: run its own gates (`cd server_timing && go test -race ./... && golangci-lint run`) at least once before the next tag; not exercised this session.
6. TODO_LIST/FEATURES pass via `docs-health` before the next version tag (monthly cadence is due).
7. Decide whether `[Unreleased]` is ready to cut as v1.0.2 (CHANGELOG now has real Added/Changed/Fixed content).
8. Consider a CI workflow that runs `buildflow --fail-on-findings` on the _signed-off_ baseline only (depends on f/3).
9. The `nix flake show` JSON parse warning inside buildflow's nix-steps ("no decodable JSON object found in 0 bytes") — one-time investigation; likely env-specific, currently benign.

**BuildFlow — follow-ups from what this session touched**
10. Rebuild + re-copy the global binary once the in-flight `fail_on` work lands (see g/1).
11. Fix `detectUnknownSkipTools` to compute against the full tool registry, killing the single-step-mode false warning (prior session's finding, still open).
12. Make `#reinstall` actually reinstall the global binary (or rename it + fix doctor's advice).
13. Make missing-tool detect steps loud (warn/fail) instead of silent success — the lychee class.
14. Add the `execution` module tidy (`dave/dst` indirect bump) to BuildFlow's own housekeeping (needs vendor + vendorHash refresh + rebuild in one go).
15. VACUUM the buildflow DB (1.09 GB) during a quiet window.
16. Consider a buildflow regression test for `#reinstall`'s contract (result vs global install) so the trap can't regress.

**Process**
17. Adopt "mutation-verify every regression test" as a written convention in BuildFlow's AGENTS.md (it's already practice in httputil's testing conventions).
18. Session handoff summaries should include exact CLI invocations that were verified, not paraphrased command names.

## g) QUESTIONS (cannot resolve myself)

1. **The uncommitted `fail_on` feature in `~/projects/BuildFlow`** (`domain/config/fail_on.go`, `internal/cli/fail_on_gate.go`, `config/materialize.go`, `.github/dependabot.yml` modified) — that's not my work and I left it strictly untouched. Is it yours in-flight? Once it lands: want me to run its gates, rebuild, and re-copy the global binary (one command chain), or is another session owning that?
2. **Dev-mode lychee fallback policy:** the httputil flake's devShell doesn't provide `lychee` (BuildFlow's `nix develop -c` fallback then fails silently-ish). Should lychee be added to the httputil devShell, or is this BuildFlow's provider fallback to fix (or a BuildFlow devShell dependency to add)? Both are one-line fixes; they're just in different repos with different owners.
3. **Standing-warnings endstate:** keep the ~180 policy-rejected detect-only findings visible in every run forever (current decision), or freeze them once into a reviewed-baseline/ledger so future summaries show only _new_ findings? The second needs a small config/ledger mechanism in BuildFlow (or a docs/ ledger + convention here).

---

_Session verdict: the original incident is closed end-to-end — regression-tested at the source, binary genuinely refreshed, every previously-failed step and the full dev-mode pipeline green, decisions documented in AGENTS.md/CHANGELOG, report annotated. The honest residue: the binary is one in-flight feature behind, the dev-mode link check may be silently skipping, and the standing-noise endstate is a real open design question, not a settled one._
