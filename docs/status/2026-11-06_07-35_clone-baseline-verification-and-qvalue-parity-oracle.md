# Status Report: Clone-Baseline Verification, Q-Value Parity Oracle, and Tool-Pin Single-Sourcing

- **Timestamp:** 2026-11-06 07:35 CET
- **Session scope:** The owner pasted a combined-mode `art-dupl` run (`-t 1 --type-aware --suggest-generics --rich-text`) and asked for READ → UNDERSTAND → RESEARCH → REFLECT, then execution and verification. Work done: re-derived the clone ledger, judged every reported pair, closed the architecture-review F1 gap (the two deliberately duplicated q-value parsers had no parity oracle), single-sourced the art-dupl flake pin to the commit the ledger stamps, corrected two false doc claims, and fixed a CI blind spot discovered while wiring the new gate.
- **Format note:** The status-report skill is HTML-canonical; the owner has previously demanded `.md` under `docs/status/`, so the override is honored and flagged here (same precedent as the 2026-10-24 art-dupl report).

---

## Headline

**The clone baseline is exactly as documented — the run reproduces to the group — and the session's real yield was three defects the run exposed in the _tooling and docs around_ the ledger, not in the ledger itself.** Re-measured twice (the go-install binary and the freshly pinned nix build): `-t 2` shows 5 groups (the q-value parser five plus the two `trim`/`trimOWS` helper fragments), `-t 1` adds the `Middleware` alias pair and the `scripts/doc-snippet-refs` if-err pair, and the generics lane is empty. Everything reported is an accepted clone family; **nothing new is a finding**.

The defects found and fixed:

1. **The flake pinned a tool that could not reproduce the documented ledger** — `art-dupl` was pinned at 0.7.0-81ce00b while every living doc stamps v0.9.1-0.20261017225421-93432fd910fb; combined mode (ADR-0026) does not exist in 0.7.0. A "single-source the stamp" open item from 2026-10-24 had never been executed.
2. **The q-value clone's sync duty was prose only** — a semantic change on one side could ship silently in the other. It is now mechanically enforced by a committed corpus asserted by both modules, plus a mirror-sync guard in four gates.
3. **CI never ran the sub-modules' tests** — `go test ./...` at the root cannot reach `server_timing/` or `endpoints/`, so ~100 tests (and the new parity oracle) only ran in the release workflow. Found only while auditing my own wiring claim; fixed with a dedicated step.

---

## Evidence trail (this session's runs, verbatim numbers)

1. Owner's run (go-install `art-dupl`, combined mode): `-t 1` → 7 shown (886 detected, 619 non-actionable, 260 filtered); `type-load 11.1s`.
2. My replication, same binary: `-t 2` → 5 shown (411 detected, 222 non-actionable, 184 filtered); `-t 1` → 7 shown (886 / 619 / 260). Same seven pairs.
3. Flake-pin rebuild (`nix build .#art-dupl`, commit 93432fd910fb2ccf25a7080cc78c3af7a40e2e56): `-t 2` → 5 shown (412 detected — one more detected group, from the two new corpus test files); `-t 1` → 7 shown (887 / 620 / 260). **Shown groups identical across binaries**; `--version` prints `dev` because buildGoModule disables VCS stamping.
4. Test-file handling measured, not assumed: default run discovers 153 files; `--ignore-tests` discovers 57 and reports 92 detected, **with the same 7 shown groups**. Therefore test files are analyzed and routed into the non-actionable/threshold-suppressed buckets — they are _not_ auto-excluded (the inherited doc claim was false).
5. Pin archaeology: `git -C ~/projects/art-dupl merge-base --is-ancestor 81ce00b 93432fd` → true (purely forward); `--type-aware` exists at 81ce00b (landed 2026-07-24) but `cmd/combined_analysis.go` does not (combined mode is 0.9.x+).
6. Upstream build reality: at 93432fd the `config` package's own tests fail under Go 1.27 json/v2 — `json: cannot marshal from Go time.Duration within "/timeout": no default representation` (TestSaveConfig, TestDetectionMode/SaveLoadRoundTrip, TestSaveConfig_FileExists). The fix exists only on the local unpushed fork HEAD (2f8ff5d6, not on GitHub — `gh api` 422).
7. Corpus mutation probes: editing one expectation (`0.555 → 0.556`) fails the root test at that line; deleting the trailing-character check fails the root test _and_ the endpoints test at line 73. Files restored byte-identically (`cmp` + `git status` clean).
8. Guard negative probe: appending one line to the mirror makes `check-qvalue-corpus-sync.sh` exit 1 with a unified diff and a fix hint; restore → OK.
9. Verification battery (all green): root + httpspec + endpoints + server_timing `go test -race`; `golangci-lint run` 0 issues (root and endpoints, full-package runs); `go vet` in all three modules; `GOWORK=off` build+vet+corpus test in `endpoints/`; `nix fmt` 0 changed; `dprint check` clean; `check-changelog-links.sh`; `doc-snippet-refs`; `pre-sweep-check.sh` 4/4; `actionlint` + `cue vet -d '#CIWorkflow'` on the edited workflow; `nix eval` of `packages.art-dupl.version` (0.9.1-93432fd) and the devShell drvPath.

---

## a) FULLY DONE

| #   | Item                                                                                                                                                                                                                                                                                                      | Evidence                                                                                                            |
| --- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------- |
| a1  | Both canonical art-dupl commands re-run independently; ledger reproduced to the group                                                                                                                                                                                                                     | session transcripts; `/tmp/dupl-t1.txt`, `/tmp/dupl-t2.txt`, `/tmp/dupl-nix-t1.txt`                                 |
| a2  | Every reported pair mapped to the five documented accepted families; **no NEW class**                                                                                                                                                                                                                     | `docs/architecture-reference.md` Accepted Code Duplication vs run output                                            |
| a3  | Generics lane ruled empty (only the combined-mode banner mentions "generics")                                                                                                                                                                                                                             | `grep -ci generic` = 1 (the banner)                                                                                 |
| a4  | Shared q-value corpus shipped: 43 cases, 77 lines, canonical root copy + byte-identical endpoints mirror                                                                                                                                                                                                  | `testdata/qvalue-corpus.txt`, `endpoints/testdata/qvalue-corpus.txt`                                                |
| a5  | Both modules ship a corpus test over that file                                                                                                                                                                                                                                                            | `qvalue_corpus_test.go` (144 lines), `endpoints/qvalue_corpus_test.go` (144 lines)                                  |
| a6  | Oracle proven non-vacuous in three directions (corpus edit, root parser edit, endpoints parser edit)                                                                                                                                                                                                      | probe transcripts; files restored byte-identically                                                                  |
| a7  | `scripts/check-qvalue-corpus-sync.sh` (29 lines) + negative probe                                                                                                                                                                                                                                         | exit 1 with diff on drift, exit 0 on sync                                                                           |
| a8  | Guard wired into FOUR gates: `ci.yml`, `release.yml`, `prerelease-check.sh` 8c, `pre-sweep-check.sh` 4/4                                                                                                                                                                                                  | diffs; pre-sweep output shows the new step passing                                                                  |
| a9  | art-dupl flake pin single-sourced to the stamped commit, rebuilt, and verified to reproduce the ledger                                                                                                                                                                                                    | `nix build .#art-dupl`; store binary run == documented 5/7 groups                                                   |
| a10 | `doCheck = false` rationale documented inline (upstream json/v2 test failure, unpushed fix)                                                                                                                                                                                                               | `flake.nix` comment block                                                                                           |
| a11 | CI blind spot fixed: sub-module tests now run on every push                                                                                                                                                                                                                                               | new `Sub-module tests (server_timing + endpoints)` step; both commands verified locally; actionlint + cue vet clean |
| a12 | Docs corrected: AGENTS (commands block, endpoints bullet, ledger paragraph, testing-convention exception, pin gotcha), architecture-reference (ledger + both file tables), CHANGELOG (`[Unreleased]` Added ×1, Changed ×1, Fixed ×1), TODO row closed, 2026-10-24 report's test-exclusion claim corrected | diffs in commits `db6d8a8..3242913` plus pending edits                                                              |
| a13 | Deliberate convention deviation recorded (data-driven corpus test is the one exception to "no table-driven tests")                                                                                                                                                                                        | AGENTS Testing Conventions bullet                                                                                   |
| a14 | R15 dependency-direction question explicitly _not_ foreclosed: the corpus is an oracle, not a direction ruling                                                                                                                                                                                            | architecture-reference bullet; question (g.1) below                                                                 |
| a15 | Tree state audited; no unowned file touched; no revert of foreign work                                                                                                                                                                                                                                    | `git status`/`git diff` reviewed before and after each mutation probe                                               |

---

## b) PARTIALLY DONE

| #   | Item                                                                                                     | What is missing                                                                                                                                   |
| --- | -------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------- |
| b1  | Consumer-view proof for the corpus (`endpoints/` excluded from the root module zip → mirror is required) | Asserted from Go's documented behaviour and encoded in comments/docs, but **not empirically verified** by building a module zip and inspecting it |
| b2  | Coverage gate (95%)                                                                                      | Not re-run locally; reasoned only (test-only additions cannot lower production coverage). CI will prove it                                        |
| b3  | `nix flake check` (full)                                                                                 | Only flake eval, package build, devShell eval, and `nix fmt` were run — the two standalone-build checks were not executed this session            |
| b4  | BuildFlow dev-mode pass                                                                                  | Not run, so markdownlint / lychee / jscpd / erraudit / go-structure-linter classes were not re-measured after these edits                         |
| b5  | Markdown lint beyond dprint                                                                              | `markdownlint-cli2` and `lychee` are not installed locally; only dprint (formatting) and the changelog-link script ran                            |
| b6  | art-dupl `--ignore-tests` as a possible canonical lane                                                   | Measured and documented, but the lane decision is owner-gated (question g.1) — the stamps still describe the all-files invocation                 |
| b7  | `doCheck = false` retirement path                                                                        | Documented, not executed: the upstream fix is unpushed/unfiled (items f.9, f.10)                                                                  |
| b8  | Corpus-loader duplication (test-only, cross-module)                                                      | Accepted implicitly; not yet listed as its own entry in the accepted-clone ledger (item f.12)                                                     |
| b9  | `docs/RELEASE.md` gate list                                                                              | The script gained step 8c; the runbook's demanded-gate list was not updated (item f.6)                                                            |
| b10 | Push                                                                                                     | 5 commits are ahead of `origin/master` (6 once the pending two files are swept); no push without instruction (question g.2)                       |

---

## c) NOT STARTED

| #  | Item                                                                               | Why it is open                                                           |
| -- | ---------------------------------------------------------------------------------- | ------------------------------------------------------------------------ |
| c1 | R15 ruling (shared parser module vs `language→go-compression`)                     | Owner decision, tied to the go-compression extraction plan               |
| c2 | Oracle for the `trim`/`trimOWS` whitespace helper clone                            | That family also has no parity lock; only the parser does (f.13)         |
| c3 | Cross-module differential fuzz (root parser vs endpoints parser over random input) | Blocked by module isolation: neither module may import the other         |
| c4 | Decision-log entry for "corpus over shared module"                                 | `docs/DECISION_LOG.md` was not consulted/written this session            |
| c5 | Updating the architecture-understanding review (F1 closed)                         | The HTML review still lists F1 as open (f.7)                             |
| c6 | art-dupl in CI as a gate                                                           | Owner REJECTED an enforcement gate on 2026-10-24; only the pin was fixed |
| c7 | Upstream art-dupl: json/v2 `time.Duration` test fix, VCS-stamped `--version`       | Out of repo, needs an owner-sanctioned push/PR (f.9, f.10)               |

---

## d) TOTALLY FUCKED UP (own errors, honestly)

| #  | What happened                                                                                                                                                                   | Damage                                                                               | Fixed?                                                                           |
| -- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------ | -------------------------------------------------------------------------------- |
| d1 | First nix hash discovery round: I hand-typed the `fetchFromGitHub` hash wrong (`…PL0M0…` instead of `…PL0M6…`) and burned a build to learn a one-character typo                 | ~1 build round                                                                       | Yes — second round produced src + vendor hashes                                  |
| d2 | Second round used a fake `vendorHash` knowingly, so two rounds were spent where one careful expression could have done it                                                       | ~1 build round                                                                       | Yes, but the ritual was wasteful                                                 |
| d3 | I **inherited** the doc claim "test files are auto-excluded by the tool" and built a derived numeric story on top of it (`Detected 887 rather than 886`) before testing it      | A false mechanism reached a living doc for ~10 minutes                               | Yes — measured with `--ignore-tests`, corrected in the ledger and the old report |
| d4 | I described the corpus tests as wired into CI **without checking that CI runs the endpoints suite at all** — it did not. The parity oracle would have protected nothing on push | The deliverable was briefly hollow; found only during the post-hoc audit             | Yes — new CI step, verified locally + actionlint + cue vet                       |
| d5 | I first wrote the correction note with a strikethrough (`~~Corrected~~`), which reads as "resolved/removed" in this repo's convention                                           | One misleading line                                                                  | Yes — rewritten as a plain `Corrected 2026-11-06:` note                          |
| d6 | Mutation probes edited tracked files with no pre-snapshot record (AGENTS.md "snapshot before `--fix`" discipline). The auto-commit daemon could have swept a mutated parser     | No commit was taken, but restoration was proven afterwards (`cmp`) instead of before | Partially — outcome verified, discipline violated                                |
| d7 | The ledger paragraph is now very long single-line prose (architecture-reference and AGENTS), and I added to it rather than restructuring it                                     | Readability debt in the highest-traffic duplication doc                              | No — item f.11                                                                   |

---

## e) WHAT WE SHOULD IMPROVE

1. **Verify the delivering layer before claiming it.** d4 is the whole lesson: "wired into CI" means "a CI job executes it on push", proven by reading the workflow, not by having edited one file in it.
2. **Measure the mechanism, not just the number.** d3: `--ignore-tests` was one command away and falsified a two-year-old assumption inherited through three docs.
3. **Hash discovery in one shot.** Prefer a single impure expression with `lib.fakeHash` for both `hash` and `vendorHash`, then re-run once; never re-type a 44-character SRI string by hand.
4. **Snapshot before mutation probes.** `git status --short > /tmp/pre-mutation.txt` plus `cp` originals into `/tmp` _before_ editing tracked files, and prove restoration with `cmp` in the same command block (d6).
5. **Keep oracle claims executable.** Every rationale in the clone ledger should be verifiable by one named command; prose that cannot be re-checked rots into d3-class errors.
6. **Split the ledger paragraph.** It is the highest-traffic doc and now four dense claims in one line (d7).
7. **Widen the parity pattern.** The `trim`/`trimOWS` clone and the two `qvalue` loaders are still unguarded; the corpus pattern is cheap to repeat.
8. **Track "tool pin == doc stamp" as a checked invariant**, not a note (f.3).
9. **Prefer pinned, VCS-stamped dev tools**; a binary that prints `dev` invites the next reader to re-litigate the pin.
10. **Probe test-file handling whenever a ledger claim mentions exclusions** — the tool has flags (`--ignore-tests`, `--exclude-pattern`) that make the distinction testable in seconds.

---

## f) WHAT WE SHOULD GET DONE NEXT (prioritized)

**P0 — close the loop on this session**

| #  | Task                                                                                                                                                        | Impact |
| -- | ----------------------------------------------------------------------------------------------------------------------------------------------------------- | ------ |
| f1 | Push the 5–6 commits (owner-gated) and confirm CI green on that exact head, then verify the new sub-module step actually executed in the run log            | High   |
| f2 | Run the full `nix flake check` so the two standalone-build checks re-validate the flake edit                                                                | High   |
| f3 | Add a tiny guard that the flake's `art-dupl` rev contains the commit hash in the AGENTS/architecture stamp (grep-based), so "pin == stamp" cannot rot again | High   |
| f4 | Re-run the coverage gate locally (root-module package list minus `scripts/`, then `coverage-threshold 95`)                                                  | Med    |
| f5 | Run a BuildFlow dev-mode pass and diff the findings against the documented residual baseline                                                                | Med    |

**P1 — finish the oracle story**

| #   | Task                                                                                                                                                                           | Impact |
| --- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ------ |
| f6  | Add prerelease step 8c and the new CI sub-module step to `docs/RELEASE.md`'s demanded-gate list                                                                                | High   |
| f7  | Mark the architecture-review F1 finding closed in `docs/architecture-understanding/2026-11-06_02-14_…` (and its `.md` twin if present)                                         | Med    |
| f8  | Empirically verify the module-zip claim (build the root module zip with `golang.org/x/mod/zip` and assert `endpoints/` absent; same for `endpoints` including its `testdata/`) | High   |
| f9  | Push the fork's json/v2 `time.Duration` fix upstream (or file it) so `doCheck = false` can be retired                                                                          | High   |
| f10 | Ask art-dupl for a VCS-stamped build (`-ldflags` version) or bake one into our derivation so `--version` stops printing `dev`                                                  | Med    |
| f11 | Restructure the Accepted Code Duplication section into per-family bullets under a short header (kills d7)                                                                      | Med    |
| f12 | Add an accepted-clone entry for the corpus-loaders pair (test-only, cross-module) so it is documented, not incidental                                                          | Low    |
| f13 | Build a corpus/oracle for the `trim`/`trimOWS` helper clone (`endpoints/negotiate.go` ↔ `compression_qvalue.go`)                                                               | Med    |
| f14 | Record the oracle decision in `docs/DECISION_LOG.md` (corpus + mirror + guard over a shared parser module, R15 open)                                                           | Med    |

**P2 — harden the new guard**

| #   | Task                                                                                                                                                     | Impact |
| --- | -------------------------------------------------------------------------------------------------------------------------------------------------------- | ------ |
| f15 | Add the corpus sync to a flake `checks` entry so `nix flake check` covers it, not only CI                                                                | Med    |
| f16 | Extend the guard to assert the mirror is a _regular file_ (not a symlink) — a symlink would satisfy `cmp` but vanish from a module zip                   | Low    |
| f17 | Add a corpus case-count floor assertion (e.g. ≥ 40) so a truncated file fails loudly instead of silently shrinking the oracle                            | Low    |
| f18 | Add `shellcheck` to the devShell and wire it for `scripts/*.sh` (not installed today; `bash -n` was the only syntax check)                               | Low    |
| f19 | Consider `.gitattributes` hardening for the corpus (`*.txt text eol=lf` already effective via `* text=auto eol=lf`) and document it in the corpus header | Low    |
| f20 | Add the guard to the nightly workflow if one runs a broader matrix                                                                                       | Low    |

**P3 — measurement and documentation polish**

| #   | Task                                                                                                                        | Impact |
| --- | --------------------------------------------------------------------------------------------------------------------------- | ------ |
| f21 | Decide g.1 (`--ignore-tests` canonical lane); if adopted, re-stamp both docs and record the cleaner `Detected` figure       | Med    |
| f22 | Add a short "how to re-measure the ledger" snippet that names the store binary path and the exact flags                     | Med    |
| f23 | Record the `Detected`/shown/`--ignore-tests` triple as a small table in the ledger rather than prose                        | Low    |
| f24 | Cross-check the ledger against `docs/review/lint/000-index.md` (does a clone-review row belong there?)                      | Low    |
| f25 | Sweep `FEATURES.md`'s testing/guard section for a mention of the parity oracle                                              | Low    |
| f26 | Update `endpoints/README.md` (module test layout, `testdata` corpus) if the module page should advertise the hand-run guard | Low    |
| f27 | Verify `docs/status/` archived-completeness gate still passes after the correction annotation (`grep -rLn '~~'`)            | Low    |
| f28 | Re-read the 2026-10-24 report's other still-open items (7, 8, 12, 13, 19) now that pin provenance is resolved               | Med    |
| f29 | Add the new files to any repo file-map/reference lists not yet updated (root table done; check `docs/modularization/*`)     | Low    |
| f30 | Confirm the daemon does not resurrect anything deleted this session (nothing was deleted — verify anyway)                   | Low    |

**P4 — findings noticed but out of scope**

| #   | Task                                                                                                                                                                                 | Impact |
| --- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ------ |
| f31 | `golangci_lint_ls` (editor LSP) reports stale golines/gosec warnings for the new test files that `golangci-lint run` does not — restart or reconfigure the LSP so editors stop lying | Med    |
| f32 | The repo has no `markdownlint`/`lychee` binaries in the devShell, so those two documented gates can only run in BuildFlow/CI                                                         | Low    |
| f33 | The auto-commit daemon keeps `auto_push` false; the session-tail discipline ("last commit pushed") cannot be satisfied by an agent                                                   | Med    |
| f34 | `art-dupl`'s `Detected` counter varies with threshold (411/412 at `-t 2`, 886/887 at `-t 1`) — worth a one-line note wherever the ledger quotes it                                   | Low    |
| f35 | Consider a repo-local `Makefile`-free `nix run .#dupl` app wrapping the two canonical commands with the pinned binary                                                                | Low    |
| f36 | Publish the parity-corpus idea upstream to the fleet (the same clone class exists in other LarsArtmann repos with duplicated parsers)                                                | Low    |
| f37 | Audit other accepted clones for "sync duty" sentences that are still prose-only (grep for "SYNC DUTY")                                                                               | Low    |
| f38 | If a shared q-value module is ever ruled in (R15), the corpus becomes its test data — note that migration path now                                                                   | Low    |
| f39 | Add the corpus to the nightly fuzz rotation only if a fuzz target is introduced (none today)                                                                                         | Low    |
| f40 | Time-box future ledger audits: this session's re-measurement itself was ~25 s; the cost was all in the neighbouring defects                                                          | Low    |

---

## g) THREE QUESTIONS I CANNOT ANSWER MYSELF

1. **Should `--ignore-tests` become the canonical clone lane?** It reports 92 detected groups instead of 887 with the **identical 7 shown**, so it removes ~795 test-file groups from every run at zero cost to the ledger's signal. Adopting it means re-stamping both docs (and the art-dupl pin note) with a new invocation; keeping the current lane preserves continuity with every prior measurement. Owner call — it changes what "the baseline" means.
2. **May I push?** `master` is 5 commits ahead of `origin/master` (6 once the daemon sweeps the pending `ci.yml` + `CHANGELOG.md`), all daemon heuristic commits plus my verified work, and CI has never seen any of it. Session-tail discipline wants a pushed, green head; the standing rule forbids push without your word.
3. **How should `doCheck = false` be retired?** Options: (a) push the fork's json/v2 fix to `LarsArtmann/art-dupl` and pin a new tag (re-stamp the ledger, re-measure), (b) keep the skip indefinitely with the inline rationale, (c) file the bug upstream and leave the pin until a release fixes it. This is an owner policy call about pinning a tool whose own tests fail on our toolchain.

---

_Waiting for instructions._
