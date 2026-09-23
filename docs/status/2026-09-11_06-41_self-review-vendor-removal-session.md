# Session Status — Vendor Removal: Self-Review and Brutal Retrospective

**Date:** 2026-09-11 06:41 (Friday)
**Session start:** ~06:20 (continued from `2026-09-11_06-30_vendor-removal.md`, same working session)
**Scope:** Owner said "I do not like vendor/" → skill applicability check → vendor/ deleted → stale-cache fallout fixed → decision documented in 6 places → self-review demanded. This report supersedes the 06-30 interim report's "Verdict" and adds the retrospective the owner asked for.

---

## a) FULLY DONE

1. **Skill applicability established with evidence, not vibes** — `nix-private-go-repos` prescribes mkPreparedSource + GOPRIVATE for _private_ deps; httputil has none (`go-error-family`, `go-etag` public via `git ls-remote`; nosurf + x/time public; flake builds no Go packages). Also **corrected the prior session's factual error**: `vendor/` was never committed — `git ls-files vendor/` = 0, gitignored at `.gitignore:63`; it was a 468K on-disk `go work vendor` artifact.
2. **`vendor/` deleted** via `trash` (recoverable). `.gitignore` keeps the `vendor/` line as a reintroduction guard.
3. **`.buildflow.yml` needed NO changes** (discovered, not assumed): `go-mod-vendor`/`go-work-vendor` auto-skip in full runs ("no files matching: vendor/modules.txt"; 54 vs 52 not-applicable before), `vendor/vendor-freshness` preflight silent. Forced single-step mode exits 69 ("no tools matched") — documented as expected.
4. **Stale result-cache rows purged** — the real hidden work of this session: deleting an untracked dir does NOT invalidate buildflow's result cache (key = SHA-256 over matched input files + binary path), so the phantom "directory vendor exists but is not ignored" finding replayed from cache. Verified `BUILDFLOW_NO_RESULT_CACHE=1` recomputes clean but does NOT overwrite the stale key (next cached run served the old row again). Purged 52 rows via surgical SQL; both steps then recompute clean and cache the clean result.
5. **Builds verified in both modes** — `go build ./...` and `GOWORK=off go build ./...` green from the module cache.
6. **Full gate green, twice** — `buildflow --build-mode dev`: exit 0, 92 success / 0 failed / +3 skipped via config; only vendor mentions in logs are the two benign auto-skips.
7. **Documentation across 6 files, both repos:**
   - httputil `AGENTS.md` — BuildFlow Pipeline section rewritten (no-vendor state, cache-purge pattern, incident history archived).
   - httputil `CHANGELOG.md` — `[Unreleased]` `### Removed` entry.
   - httputil `ROADMAP.md` — Non-goal: "Vendoring (`vendor/` directory)".
   - httputil 06-16 report — reversed "keep vendor" decision struck with supersession note.
   - BuildFlow `AGENTS.md` — **gotcha #155**: result-cache content-addressing blind spot (fs-state-dependent findings stale for full TTL; no-cache runs skip the write; purge SQL; 3 fix directions).
   - BuildFlow `TODO_LIST.md` — **S64**: the cache-key limitation as bounded work.
8. **All trees committed** (daemon): httputil and BuildFlow both clean at report time.

## b) PARTIALLY DONE

1. **Open-question triage from 06-16 §g** — Q1 (`fail_on`): the feature **landed** (gotcha #154; daemon commits `23b2849b4`/`802bbbcbb`), but the global binary was NOT rebuilt (binary `984dac6` < HEAD, drifting further). Rebuild was gated on the owner's earlier answer; documented in reports, not executed.
2. ~~**Stale-prose sweep** — grepped the tree for vendor references but **excluded `docs/`** in that grep; `docs/architecture-reference.md` (the canonical code-map/lint-profile page AGENTS.md points to) was never checked for vendor mentions. Living docs verified clean (TODO_LIST/FEATURES/ROADMAP: zero hits).~~ done (verified 2026-09-23 — architecture-reference.md has zero vendor mentions)
3. ~~**06-16 report annotation** — line 23 (the reversed decision) struck, but line 27 ("workspace format stays canonical") is now equally moot and was left unannotated. Minor.~~ done (06-16 line 27 struck as mooted 2026-09-23 (docs-health pass))
4. **Vestigial config references** — `.markdownlintignore` excludes `vendor/` and AGENTS.md:131 documents lychee's `--exclude-path vendor`; both now reference a nonexistent directory. Harmless (and arguably a guard), but neither the configs nor the doc note were touched.

## c) NOT STARTED (carried; nothing new dropped)

1. Dev-mode lychee silent no-op (`exec: lychee: not found` via `nix develop -c` fallback, step counts success) — 06-16 §g Q2, still unanswered.
2. ~~Detect-only findings baseline decision — 216 findings this run (was ~180, drifted) — 06-16 §g Q3, still unanswered.~~ done (resolved 2026-09-15 — the documented AGENTS.md stance: dev runs expected to exit non-zero on the findings gate, policy-rejected residuals listed; only NEW finding classes are breakage)
3. BuildFlow S64 implementation (fs-state in detector inputs / `cache purge` subcommand / no-cache upsert) — documented only.
4. The 18-item ranked list in the 06-16 report §f — carried wholesale.
5. buildflow DB VACUUM (preflight says 1.28 GB) — suggested by the tool itself, not run.
6. ~~`erraudit` findings drift 55→56 between runs — unexplained, carried.~~ done (superseded — erraudit counts re-measured and documented in AGENTS.md 2026-09-15 (45 sentinel_concrete_type + 41 test-side advisories, do-not-migrate))
7. gopls stdversion jsonv2 warnings (5 files) — pre-existing, untouched.

## d) TOTALLY FUCKED UP (nothing catastrophic; real missteps)

1. **Pipelined exit codes — again.** First per-step probe ran `buildflow -s X 2>&1 | tail -6; echo "exit: $?"` — measuring `tail`'s exit code, not buildflow's. Reported "exit: 0" for steps that actually exit 69. This is the EXACT pipeline-masking lesson written in the global AGENTS.md. Caught on the very next command by re-measuring unpiped, but it should not have happened at all — that rule exists because of me-classes of errors.
2. **Batch edit without read discipline.** Fired 4 parallel edits; 3 failed (two BuildFlow files never View-read; the 06-16 file daemon-touched after the prior session's read). Cost: one round trip and a user prompt to slow down. The mtime change itself I investigated properly before touching (git diff HEAD empty, benign daemon commit `8942c19`) — but I never root-caused WHAT rewrote the file (suspect: daemon formatting pass). "Benign by diff" is not "understood".
3. **First gotcha-#155 draft had `go.md` instead of `go.mod`** — caught during re-plan, never landed. A draft written faster than read.
4. **Under-scoped the vendor-reference sweep** — excluded `docs/` from the grep on the theory that "docs are historical", then immediately edited `docs/status/` files. Inconsistent scope reasoning; the architecture-reference page may now be stale and I don't know because I didn't look.
5. **S64 convention bend** — dropped into a "Session Follow-ups" table whose header says every row "cites the originating report (now under `docs/status/archived/`)"; my source cites a foreign repo's session. Doesn't resolve. Should have either put it in "Code Health" or adjusted the citation convention.
6. **Carried forward without challenge**: the prior session's summary said "keep committed vendor/" and I repeated it in my first reply this session before checking — the check (30 seconds, `git ls-files`) is what killed the myth. Trust-but-verify applies to my own handoffs too.

## e) WHAT WE SHOULD IMPROVE (process, from this session)

1. **Unpiped exit codes, always** — when a command's exit status is the claim, run it bare or capture `$?` on the command itself, never after a pipe. Consider adding this to the httputil AGENTS.md command section (it's in global already; this session proves local repetition is warranted).
2. **Cross-repo edits: View first, every file, no exceptions** — even for "one-line appends".
3. **Concept removal ⇒ full-tree grep INCLUDING docs/** — the sweep that proves "nothing references X anymore" must cover the directories you're about to edit.
4. **Cache-layer thinking** — when a tool caches, ask "what inputs does the key NOT cover?" before trusting freshness. Today: untracked-dir existence. BuildFlow should fold fs-state into detector inputs (S64) so users never need the sqlite purge.
5. **Status reports should state open questions' status explicitly** — Q1 resolved itself (fail_on landed) and I only noticed incidentally while reading BuildFlow's gotcha list for an unrelated reason. A deliberate check would have caught it an hour earlier.
6. **The daemon commits everything within minutes** — write-then-verify-then-let-go is the right rhythm here; no manual commits needed unless release-blocking.

## f) NEXT WORK (ranked, impact × effort)

1. **Decide `fail_on` policy BEFORE rebuilding the global binary** — gotcha #154 makes the findings gate default-ON at error severity; the new binary may make httputil's dev runs FAIL on erraudit errors that today exit 0. Pick: `fail_on: none` in `.buildflow.yml`, `strict: true` (warn), or accept-and-fix.
2. Rebuild + reinstall global binary (`cd ~/projects/BuildFlow && nix build . && cp result/bin/buildflow ~/.local/bin/buildflow && buildflow --version`), then re-run full dev gate under the new gate semantics.
3. ~~Sweep `docs/` (esp. `architecture-reference.md`) for stale vendor prose; refresh the code-map/lint-profile page if it mentions vendoring.~~ done (verified 2026-09-23 — architecture-reference.md has zero vendor prose)
4. Decide vestigial-config question: keep `.markdownlintignore`/`--exclude-path vendor` as guards (document as such in AGENTS.md:131) or strip them.
5. ~~Annotate 06-16 line 27 ("workspace format stays canonical") as mooted — 30 seconds.~~ done (06-16 line 27 struck as mooted 2026-09-23)
6. Fix dev-mode lychee no-op: add `lychee` to httputil devShell OR make BuildFlow's `nix develop -c` fallback loud on missing tools (owner decision pending since 06-16).
7. Re-run `buildflow -s lychee` after (6); pin the 301-links/0-errors baseline in AGENTS.md.
8. ~~Detect-only findings baseline: freeze reviewed ledger (216 today) or leave visible — owner decision pending.~~ done (resolved via the documented AGENTS.md stance (expected non-zero findings gate; policy-rejected residuals enumerated))
9. BuildFlow S64(a): fold directory-existence probes into gomod-check detector inputs.
10. BuildFlow S64(b): `buildflow cache purge --tool X --contains 'text'` subcommand.
11. BuildFlow S64(c): no-cache runs upsert fresh results over stale keys.
12. BuildFlow: make missing-tool detect steps loud (go-licenses silent no-op in dev mode).
13. BuildFlow: `#reinstall` app should actually refresh the global binary (the `cp` trap, documented twice now).
14. BuildFlow: `detectUnknownSkipTools` full-registry computation (single-step-mode false warning).
15. `sqlite3 ~/.cache/buildflow/buildflow.db VACUUM` (1.28 GB).
16. ~~httputil: `cd server_timing && go test -race ./... && golangci-lint run` before any tag (protocol from AGENTS.md).~~ done (protocol documented in AGENTS.md Commands (server_timing sub-module gates); enforced at each release via RELEASE.md)
17. ~~docs-health skill pass over TODO_LIST/FEATURES/ROADMAP/CHANGELOG before next tag.~~ done (docs-health pass 2026-09-23, this docs-health pass)
18. ~~Decide whether `[Unreleased]` is ready to cut as v1.0.2 (it now has Added/Changed/Fixed/Removed).~~ done (superseded — v1.1.0 shipped 2026-09-11 instead of a v1.0.2; the ladder ran v1.0.1 → v1.1.0)
19. ~~Investigate erraudit 55→56 findings drift (unexplained since 06-16 session).~~ done (superseded — counts re-measured and documented in AGENTS.md 2026-09-15)
20. Root-cause the 06-16 file mtime rewrite (suspect: daemon formatting pass) — low priority, curiosity plus audit hygiene.
21. Decide gopls stdversion jsonv2 warnings: bump Go to 1.27 or silence per-file (5 files, pre-existing).
22. markdownlint eval-cache SQLite "busy" warnings — carried noise, likely fixed by (15).
23. BuildFlow `execution` module tidy drift (`dave/dst` indirect bump) — carried from 06-16 §f.
24. Consider a `buildflow doctor` check for "finding depends on fs state outside cache key" — generic lint for the #155 bug class.
25. ~~If v1.0.2 is cut: run `scripts/prerelease-check.sh` + docs/RELEASE.md runbook.~~ done (superseded — v1.1.0 shipped 2026-09-11; its gates ran via scripts/prerelease-check.sh + RELEASE.md)

## g) QUESTIONS FOR THE OWNER (cannot be figured out from here)

1. ~~**`fail_on` policy for httputil:** once the global binary is rebuilt, the findings gate is ON at error severity by default (BuildFlow gotcha #154). httputil currently runs with 216 detect-only findings, erraudit among them. Do you want (a) `fail_on: none` in `.buildflow.yml` (status quo semantics), (b) `strict: true` (warn loudly, never fail), or (c) accept the gate and drive error-severity findings to zero? This decides whether I rebuild the binary now or configure first.~~ done (resolved de facto — AGENTS.md documents the stance: full dev runs expected to exit non-zero on the findings gate with enumerated policy-rejected residuals; only NEW finding classes are breakage)
2. **Dev-mode lychee no-op — where does the fix belong:** httputil's devShell (add `lychee` so the fallback path works) or BuildFlow (make the `nix develop -c` fallback LOUD when the tool is missing, since a silently-successful link check is a lying gate)? Both is also an answer.
3. ~~**The 216 detect-only findings:** freeze a reviewed baseline (ledger/config so genuinely-new findings stand out) or keep them all visible forever as today? (Carried from 06-16 §g Q3 — your call, both defensible.)~~ done (resolved — keep visible: AGENTS.md documents the expected-non-zero stance with the policy-rejected residual list (2026-09-15))

---

**Session verdict:** vendor/ removal itself was clean, verified, and documented on both sides of the repo boundary. The self-review found no damage — but four process failures (piped exits, unread-file edits, draft typo, under-scoped sweep) that the existing written rules already forbade. The rules were right; the compliance wasn't.
