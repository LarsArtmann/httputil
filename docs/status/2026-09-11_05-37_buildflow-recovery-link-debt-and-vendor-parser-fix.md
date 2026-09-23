# Status Report: BuildFlow Recovery, Markdown/Link Debt Payoff, and BuildFlow Vendor-Parser Fix

> **Date:** 2026-09-11 05:37
> **Session scope:** Triage the failed `buildflow dev` run (4 failed checks, ~15k detect-only findings), repair everything actionable in httputil, and root-cause the recurring false positives into the owner's BuildFlow tool.

---

## Context (the failed run, deconstructed)

The pasted BuildFlow run failed exit 1 with **4 failed nix checks** — `golangci-lint` (×2), `govalid-generate`, `test-coverage` — ALL caused by one thing: `vendor/modules.txt` out of sync with `go.mod` after the `server_timing` bump to v1.0.1 ("inconsistent vendoring"). That was already repaired by commit `945ed0d` (workspace-vendor sync) before this session started; the pasted output came from a run against the pre-fix tree with a stale buildflow binary (preflight warned "binary predates HEAD by 21h").

The remaining ~15.2k findings were detect-only warnings: markdown-lint 13,374 (87% MD013), lychee 113, erraudit 55, branching-flow 80, go-structure-linter 41, go-mod-ignore-check 7, shellcheck 11, go-auto-upgrade 8.

---

## a) FULLY DONE

1. **Verified the vendor fix end-to-end** — `go build ./...` in workspace mode AND `GOWORK=off` module mode (root + `server_timing`), full `go test -race -count=1 ./...` suite: **all green**. `golangci-lint run`: **0 issues**. The 4 failed checks' root cause is confirmed dead.
2. **shellcheck (scripts/prerelease-check.sh)** — fixed SC2164 (`cd … || exit 1`) and SC2046 (coverage package list now `mapfile` + `"${coverage_packages[@]}"`, preserving the intentional word-splitting shellcheck-cleanly). Verified clean via nixpkgs shellcheck + `bash -n`.
3. **markdown-lint: 13,374 → 0 findings.**
   - Added `.markdownlint.json` disabling exactly the rules whose findings are repo-intentional style or frozen history (MD001/009/010/012/013/022/024/026/028/029/031/034/036/037/040/051); every disabled rule's findings were located and judged first (archived session logs, CHANGELOG section repetition, Makefile-tab snippets, long-line table style).
   - Added `.markdownlintignore` (`vendor/`).
   - Fixed the _real_ MD042: README badge links `](#)` → plain images — in README.md **and** in `scripts/update-coverage-badge.sh` (the generator would have reintroduced the pattern on every coverage run; its stale sed-history comment updated too).
   - Fixed MD034 in SECURITY.md: `**git@lars.software**` → `<git@lars.software>` autolink.
4. **lychee: 113 → 0 broken links** (301 links checked, 300 OK, 1 excluded).
   - Root cause: the docs-health archiving move (`docs/status/` → `docs/status/archived/`, same for planning) added a directory level without rewriting the `../../CHANGELOG.md`/`../../ROADMAP.md` header links; all 35 archived reports got `../../../` depth fixes.
   - Fixed 4 living-doc references to the relocated rate-limiter design note (`ROADMAP.md:22`, `FEATURES.md:241`, `docs/DECISION_LOG.md:13`, `docs/migrating-to-keyed-rate-limiter.md:73` → `docs/planning/archived/…`).
5. **go-auto-upgrade findings rejected on policy** — all 4 unique sites are `stdlib2lo` suggestions (use `lo.Reduce`/`lo.SliceToMap`), i.e. "add `github.com/samber/lo`". That dependency is forbidden by the depguard allowlist (AGENTS.md hard constraint); applying the tool's advice would fail golangci-lint instantly. Created `.buildflow.yml` with `skip_steps: [go-auto-upgrade]` + rationale comment (verified the skip takes effect via `--dry-run`: "skipped via skip_steps config").
6. **flake.nix deduplicated** — removed `apps.test-race`, byte-identical to `apps.test` (both ran `go test ./... -race -count=1`); pure maintenance trap.
7. **`nix fmt`** (0 changes needed) and **`nix flake check`: all checks passed.**
8. **BuildFlow vendor-parser bug root-caused and fixed at source** (`~/projects/BuildFlow`): `parseVendorModulesTxt` matched only the exact line `## explicit` but `go work vendor` writes `## explicit; go 1.26` — so EVERY module in a workspace-vendored repo was falsely reported "not marked as explicit". Fixed with prefix matching (`raw == "## explicit" || strings.HasPrefix(raw, "## explicit;")`) + doc comment. This kills the recurring `go-mod-ignore-check`/`gomod-check` error findings at their actual root.
9. All changes were auto-committed by the daemon (httputil `6a41ebb` et al.; BuildFlow `fa782b72e`).

---

## b) PARTIALLY DONE

1. **BuildFlow parser fix** — source edited and committed by the daemon, but (i) the regression test in `checker_godebug_test.go` is NOT yet written (I had just finished reading the test patterns when interrupted), and (ii) the **installed binary is still old** — until `nix build . && nix run .#reinstall` runs in `~/projects/BuildFlow`, local runs keep printing the false positives.
2. ~~**Full-pipeline re-verification** — `buildflow --dry-run` shows 61 success / 0 failed / pass-with-warnings, and I verified golangci/shellcheck/markdownlint/lychee/vendor via direct tool runs, but I did NOT re-run each previously-failed buildflow step individually through buildflow itself (`-s` does not accumulate; my one combined invocation silently ran only the last step — see d).~~ done (documented since — AGENTS.md BuildFlow section lists the residual detect-only findings as policy-rejected, not debt)
3. **BuildFlow test conventions check** — `gofmt`/lint/tests of the BuildFlow repo itself not run after my edit.

---

## c) NOT STARTED

1. **AGENTS.md memory updates** — nothing recorded yet about: `.buildflow.yml` skip rationale, `.markdownlint.json` rule rationale, the workspace-vendor false-positive history, the archived-link-depth trap, `prerelease-check.sh`'s new `mapfile` shape, `apps.test-race` removal.
2. **Remaining detect-only opinion findings** (188 in dry-run): `branching-flow` 80 (bool-fields-as-bitflags, single-implementer interface, httpspec `mustRequest` panic — all contradict documented repo decisions), `erraudit` 55 blank-identifier advisories (the documented "honest silence" discards), `go-structure-linter` 41 (root-package flat layout — explicitly decided; one claim "compiled binary tracked in git" was verified FALSE — `scripts/coverage-threshold/` is Go source, not a binary). None documented as accepted-rejections.
3. **`go-mod-ignore-check` info finding** "directory vendor exists but is not ignored in go.mod" — dismissed without understanding what the tool actually wants there.
4. **BuildFlow housekeeping items it self-reported**: 1.09 GB DB (VACUUM), `go-licenses` missing from PATH, 9 tools failing health check, stale binary.
5. **CHANGELOG [Unreleased] entry** for the doc/link/tooling fixes (shellcheck fixes, link repairs, lint configs are user-visible repo changes).

---

## d) TOTALLY FUCKED UP

Nothing destructive. Two honest process fumbles:

1. **`buildflow -s a -s b -s c …` does not accumulate steps** — only the LAST one ran. I treated the output as "all six steps pass" evidence before noticing the "single-step mode" line; the markdown-lint/shellcheck/golangci/govalid/test-coverage confirmations in that background job never happened there (they were covered by equivalent direct runs instead — but the mistake was real and the conclusion initially overstated).
2. **markdownlint JSON capture fumbled twice** (`> file` produced 0 bytes; the tool writes through a stream that needed `2>&1` merging) — two wasted round trips before asking why, instead of checking the file size first.
3. Minor: I read `recorder.go:149`'s stale finding against current code before realizing the pasted output predated HEAD — one extra tool call that a "check the run's age first" reflex would have saved.

---

## e) WHAT WE SHOULD IMPROVE

1. **The archiving move must rewrite link depth.** The docs-health `git mv` to `archived/` silently broke 70 links across 35 files — invisible until lychee ran. Archive procedure needs a "fix `../../` → `../../../` (or re-check with lychee)" step, or archived headers should use root-relative paths that survive moves.
2. **Config-gate the noise, document the rejections.** We now carry `.markdownlint.json` + `.buildflow.yml` whose every disabled rule/skipped step encodes a decision — those decisions live only in my head and the file comments until AGENTS.md is updated. A rejection without a written rationale rots into "why is this disabled?" in three months.
3. **Stale-tooling runs waste whole sessions.** The failed paste was 90% already-fixed-at-HEAD problems seen through a 21h-old binary. Rebuild-before-triage (or trusting `nix`-fresh step runs like the `-s` invocations) should be the reflex.
4. **`skip_steps` single-step mode lies** — it warns your skip entries "match no registered tool" because only one provider is registered in `-s` mode. BuildFlow should compute `detectUnknownSkipTools` against the full registry, not the step-filtered one (candidate upstream fix).
5. **Generators and their output drift silently.** `update-coverage-badge.sh` would have re-broken README's badge links on the next coverage run. Any lint/style fix needs a grep for the generator that produces the pattern.
6. **19k-finding walls hide the 5 real ones.** Before this session, 4 genuinely broken MD042/MD034 findings were buried under 13k style findings. Zero-finding baselines (now in place) keep future signal visible.

---

## f) Next (bounded, prioritized)

1. Write the BuildFlow regression test: `## explicit; go 1.26` marker + replace-suffixed header (`# … v1.0.1 => ./server_timing`) parse as explicit (pattern: `negative_consistent` subtest in `checker_godebug_test.go`).
2. Run BuildFlow's own gates on the parser fix: `go test ./modules/gomod-checker/...`, lint, then rebuild+reinstall the binary (`nix build . && nix run .#reinstall` in `~/projects/BuildFlow`) — see questions.
3. Update httputil AGENTS.md: `.buildflow.yml` skip rationale, `.markdownlint.json` disabled-rule rationale, workspace-vendor `## explicit; go` false-positive history, link-depth archive trap, `test-race` app removal.
4. Re-run the previously-failed buildflow steps individually (`buildflow -s golangci-lint`, `-s govalid-generate`, `-s test-coverage`, `-s markdown-lint`, `-s shellcheck`) with the rebuilt binary for authoritative greens.
5. Run full `buildflow dev` (not dry-run) once, confirm exit 0 and a sane findings residual.
6. Add CHANGELOG [Unreleased] entries for the script/shellcheck/link/lint-config fixes.
7. Decide + document the detect-only opinion tools: keep `branching-flow`/`go-structure-linter`/`erraudit` advisories visible, or skip/exclude the subsets that contradict decided policy (flat root package, honest-silence discards, defensive single-implementer interfaces).
8. Understand or fix the `go-mod-ignore-check` info finding about vendor + go.mod ignore semantics (upstream doc/read of that check).
9. Consider `--fail-on-findings` for CI once the residual warnings are curated — so regressions like the 35-file link break actually fail builds.
10. BuildFlow upstream candidates: (a) fix `detectUnknownSkipTools` scope in `-s` mode; (b) support `lo`-suggestion suppression per-repo without skipping the whole `go-auto-upgrade` step (keeps json v1→v2 detection alive); (c) teach `go-structure-linter` about "flat package is deliberate" via config.
11. VACUUM the 1.09 GB buildflow DB; add `go-licenses` to a devShell; investigate the 9 health-check-failed tools.
12. Chase the 5 gopls `stdversion` warnings (`json.Marshal` "requires go1.27", file is go1.26) in csrf.go/health.go/tests — gopls stricter than the toolchain; confirm whether GOEXPERIMENT=jsonv2 makes them spurious.
13. Add a CI/pre-commit lychee gate (links now at 0 — keep them there).
14. Sweep for other generators that emit lintable output (badge scripts, coverage-threshold tool) and pin their output to the new configs.
15. Docs-health VERIFY pass over this report's claims when convenient (items are fresh; nothing older than today).

---

## g) Questions (cannot resolve myself)

1. **Rebuild the global buildflow binary now?** The parser fix only takes effect after rebuilding/reinstalling from `~/projects/BuildFlow` (`nix build . && nix run .#reinstall`). It mutates your global tool; the regression test (f.1) is not yet written. Do it now, or after f.1–f.2?
   ~~Resolved 2026-09-11: test first, then rebuild — done in that order. Regression tests written (two subtests, mutation-verified: both fail with the exact original false-positive under the reverted parser), full BuildFlow suite green, then binary rebuilt. Gotcha: `nix run .#reinstall` only refreshes `./result`; the global `~/.local/bin/buildflow` is a plain copy and needs `cp result/bin/buildflow ~/.local/bin/buildflow`. `buildflow --version` now matches HEAD (`984dac6`).~~
2. **Opinion-tool noise policy:** should detect-only advisors whose suggestions contradict decided policy (flat root package, honest-silence discards, `samber/lo` ban) be silenced via `skip_steps`/config like `go-auto-upgrade` was, or left visible as standing warnings for review?
   ~~Resolved 2026-09-11: split decision. `go-auto-upgrade` stays skipped (its advice is unconditionally forbidden — samber/lo vs depguard). The rest stay visible: they are detect-only, never fail the run, and blanket-skipping would hide new genuine findings mixed into the noise. Rationale recorded in AGENTS.md "BuildFlow Pipeline" → "Residual detect-only findings are policy-rejected, not debt".~~
3. **Is committed `vendor/` still wanted at all?** The flake builds no Go packages (checks only); the workspace + `replace => ./server_timing` resolves everything. The `nix-private-go-repos` skill treats committed `vendor/` as legacy debt (mkPreparedSource/GOPRIVATE is the preferred pattern). Migrate off `vendor/`, or is it deliberate here (e.g. for hermetic `GOWORK=off` CI checks like govalid/test-coverage)?
   ~~Resolved 2026-09-11: keep committed `vendor/` — deliberate. The GOWORK=off CI gates are its load-bearing consumer; mkPreparedSource solves a problem this repo does not have (public deps, no hermetic nix Go builds — the flake runs checks, not packages). Documented quirks: workspace-vendor format is canonical (`go work vendor`), forced `-mod=vendor` in module mode fails on the missing replace-marking header (latent, no gate uses it), and the gomod-check "vendor not ignored" info finding is accepted (an `ignore ./vendor` directive was A/B-tested as neutral but deliberately not added). Full rationale in AGENTS.md "BuildFlow Pipeline".~~

---

_Session verdict: the pipeline is green again (dry-run 61/0 failed), zero markdown/link/shellcheck findings, and the biggest recurring false-positive generator is fixed at its source. The open debt is documentation (AGENTS.md/CHANGELOG), the BuildFlow test+rebuild, and the three decisions above._

_Resolution addendum (2026-09-11, later session): all three questions resolved (see strikethroughs in g); regression tests written and mutation-verified; binary rebuilt to `984dac6`; `buildflow --build-mode dev` exit 0 (97 success / 0 failed); AGENTS.md "BuildFlow Pipeline" section and CHANGELOG `[Unreleased]` entries added; `nix fmt` + `nix flake check` green. One new fact discovered: `buildflow dev` is not a command — the full run is `buildflow --build-mode dev`._
