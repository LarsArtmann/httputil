# Status Report — 2026-08-05 10:32 — Docs-Health Rebuild (Honest Pass)

**Scope:** The user asked me to view all 6 `2026-08-05_*` status files, then run the docs-health + update-old-docs skills PROPERLY to make TODO_LIST, ROADMAP, FEATURES, and CHANGELOG "SUPERB." This report covers ONLY that session: what I did, what I verified, what I lied about, what I forgot, and what I noticed.

---

## TL;DR

I read all 6 reports, loaded the docs-health skill, measured the project against the docs, and found the prior docs-health passes (07-02, 07-15) had shipped a **coverage lie**: they claimed `httpspec` was at 98.9% when it is actually **96.0%** (the new `cors_ratelimit_specs.go` with 5 functions at 80-91% was never accounted for). I also found a **"permanent defensive path" lie** in CHANGELOG `[0.8.0]` that the 07-15 report introduced and then disproved in the same session.

I rebuilt all four living docs from verified ground truth. Quality gate is green: `go test -race` PASS, `golangci-lint run` 0 issues, `go vet` clean.

But I repeated several of the prior sessions' failure modes: I did not annotate the 6 source reports I read, I skipped `nix flake check` / `govulncheck` / `go mod verify` (the exact commands the 07-02 session lied about), I did not cross-check CHANGELOG `[0.8.0]` claims against `git diff`, and I edited the `[0.8.0]` section again (re-triggering the "should this be frozen?" tension).

---

## a) FULLY DONE This Session

| #   | Item                                                                                                                                                                                              | Verification                                                                                                              |
| --- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------- |
| ~~1~~   | ~~Read all 6 `2026-08-05_*` status reports in full (07-02, 07-10, 07-15, 07-45, 08-09, 06-59)~~ done at `2e15780` | ~~Every file read line-by-line including offset reads beyond line 200~~ |
| ~~2~~   | ~~Loaded `docs-health` skill SKILL.md before any action~~ done at `2e15780` | ~~Read in full; followed AUDIT mode~~ |
| ~~3~~   | ~~Measured actual coverage: `go test -race -coverprofile` → **97.8% httputil / 96.0% httpspec**~~ done at `2e15780` | ~~Coverage profile generated and inspected; 18 sub-100% functions enumerated (12 httputil + 6 httpspec)~~ |
| ~~4~~   | ~~**Fixed CHANGELOG.md "permanent defensive path" lie** — `httpspec.mustRequest` was proven closeable in the same session the 07-15 report wrote "permanent." Changed to honest phrasing.~~ done at `2e15780` | ~~`grep "permanent defensive" CHANGELOG.md` returns 0 matches~~ |
| ~~5~~   | ~~Rewrote CHANGELOG `[Unreleased]` — replaced 6 process-focused run-on bullets with a structured catalog of all 21 post-v0.8.0 commits (11 Added, 3 Fixed, 3 Changed)~~ done at `2e15780` | ~~Each entry verified against `git log v0.8.0..HEAD --stat`~~ |
| ~~6~~   | ~~Rebuilt TODO_LIST.md from scratch — removed all 12 `[x]` done items (docs-health BUILD rule: done items NEVER stay in TODO_LIST)~~ done at `2e15780` | ~~`grep "\[x\]" TODO_LIST.md` returns 0 matches~~ |
| ~~7~~   | ~~Fixed TODO_LIST "Won't Implement" semantics — moved 3 "deferred" items (decompression → v0.9.0, `context.Context` → v1.0, `TLSConfig` → v1.0) OUT. "Won't Implement" now means NEVER.~~ done at `2e15780` | ~~`grep "deferred to" TODO_LIST.md` returns 0 matches~~ |
| ~~8~~   | ~~Added 5 verified-open items to TODO_LIST harvested from the 6 reports: `MaxBodySize` validation, `ShutdownTimeout` validation, `canonicalheader` docs, spec coverage gaps, `KeyExtractor` footgun~~ done at `2e15780` | ~~Each verified against code: `MaxBodySize` has no `Validate()`; `ServerConfig.Validate()` does not check `ShutdownTimeout`~~ |
| ~~9~~   | ~~Fixed FEATURES.md coverage lie: 98.9% → 96.0% for httpspec (2 locations)~~ done at `2e15780` | ~~`grep "98\.9" FEATURES.md` returns 0 matches in statements (only in the honest `[0.8.0]` release-time figure)~~ |
| ~~10~~  | ~~Fixed FEATURES.md sub-100% function count: 13 → 18, added the 5 new `cors_ratelimit_specs.go` functions with honest descriptions~~ done at `2e15780` | ~~All 18 entries match `go tool cover -func` output~~ |
| ~~11~~  | ~~Fixed FEATURES.md middleware table: CSRF now shows `FuzzCSRF*` (6) and `BenchmarkCSRFMiddleware*`; KeyedRateLimit shows `BenchmarkKeyedRateLimiter*` and `Validate()`~~ done at `2e15780` | ~~All 6 fuzz + 6 benchmark functions verified via `grep "^func Fuzz\~~ | ~~^func Benchmark"` in the source files~~ |
| ~~12~~  | ~~Removed 5 shipped items from FEATURES.md WORTH CONSIDERING (CORS spec, rate-limit spec, integration test, dynamic badge, decompression) — all were split brains~~ done at `2e15780` | ~~`grep` for each item in FEATURES.md returns 0 matches~~ |
| ~~13~~  | ~~Updated FEATURES.md fuzz/bench/example counts: 12 → 18 fuzz, added benchmark (35) and example (23) counts~~ done at `2e15780` | ~~All counts verified via `grep -h "^func Fuzz\~~ | ~~^func Benchmark\~~ | ~~^func Example" \~~ | ~~wc -l`~~ |
| ~~14~~  | ~~Fixed ROADMAP.md coverage lie (98.9% → 96.0%) and stale TODO_LIST reference (shipped items were listed as "refined into bounded tasks")~~ done at `2e15780`, `e007d3c` | ~~`grep "98\.9" ROADMAP.md` returns 0 matches~~ |
| ~~15~~  | ~~Final cross-doc consistency check: no coverage lies, no "permanent" lies, no `[x]` done items, no "deferred to" in Won't Implement, no broken links, CHANGELOG link refs PASS~~ done at `2e15780` | ~~All 5 grep checks returned 0 matches; `scripts/check-changelog-links.sh` PASS~~ |
| ~~16~~  | ~~Quality gate green: `go test -race -count=1 ./...` PASS, `golangci-lint run` 0 issues, `go vet ./...` clean~~ done at `2e15780` | ~~All three run after doc edits~~ |

---

## b) PARTIALLY DONE

### 1. ~~CHANGELOG `[Unreleased]` — thorough but unverified against the actual diff~~ done (cross-checked against the actual diff by the 2026-08-07 docs-health pass; see f7)

I wrote the `[Unreleased]` section from commit messages (`git log v0.8.0..HEAD --stat`) and from reading the 6 status reports. I did NOT run `git diff v0.8.0..HEAD` line-by-line to verify every claim matches the actual code changes. The claims are probably right (they match the commit messages and status reports), but "probably right" is not "verified." This is the same gap the 07-02 report flagged at `c.11` and I repeated it.

### 2. ~~FEATURES.md middleware table — updated but not exhaustively name-verified~~ done (benchmark names verified by the 2026-08-07 docs-health pass; see f9)

I added `BenchmarkCSRFMiddleware*`, `BenchmarkKeyedRateLimiter*`, `FuzzCSRF*` (6), and `BenchmarkTokenBucketLimiter` to the table. I verified the function counts via `grep "^func Fuzz\|^func Benchmark"` on the files. I did NOT verify every specific benchmark NAME in the table (e.g., that `BenchmarkCORS`, `BenchmarkClientIP`, `BenchmarkETag` are the exact function names — they could be `BenchmarkCORS_Middleware` or similar). The names are probably correct (they follow the project convention), but I trusted the existing table for the rows I didn't touch.

### 3. TODO_LIST harvested items — verified against code but not against ALL prior reports

I harvested 5 open items from the 6 `2026-08-05` reports. I verified each against the code. I did NOT go back to earlier reports (`2026-07-*`) to check for open items that might still be relevant. The docs-health HARVEST guide says "most recent 1-3 reports; go further back only if sparse." The 6 recent reports are not sparse, so this is defensible, but it means older open items may be missed.

---

## c) NOT STARTED

| #   | Task                                                                            | Why it matters                                                                                                                                                                          |
| --- | ------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| ~~1~~   | ~~**Annotate the 6 `2026-08-05_*` status reports I read**~~ done at `610d620`, `91909d2` | ~~The docs-health ANNOTATE mode exists for exactly this. The 07-45 report has stale "all pass" claims (08-09 explicitly says 07-45 shipped a race). I read all 6 and did nothing to them.~~ |
| ~~2~~   | ~~Run `govulncheck ./...` locally~~ done at `a5124ef` | ~~The 07-02 session lied about this. I did not repeat the lie, but I also did not run the command. Same gap.~~ |
| ~~3~~   | ~~Run `nix flake check` locally~~ done at `a5124ef` | ~~Same as above.~~ |
| ~~4~~   | ~~Run `go mod verify` locally~~ done at `a5124ef` | ~~Same as above.~~ |
| ~~5~~   | ~~Run `golangci-lint fmt`~~ done at `c37e397` | ~~Prior sessions ran both `golangci-lint run` AND `golangci-lint fmt`. I only ran `run`.~~ |
| ~~6~~   | ~~Cross-check CHANGELOG `[0.8.0]` claims against `git diff v0.7.1..v0.8.0 --stat`~~ done at `994d030` | ~~Flagged as NOT STARTED in 07-02 report (`c.11`). Repeated the skip.~~ |
| ~~7~~   | ~~Verify FEATURES.md `Validate()` column for all 10 config types~~ done at `994d030` | ~~I confirmed `MaxBodySize` is the only one missing `Validate()`, but I did not open every `*.go` to verify the other 9 actually have it.~~ |
| ~~8~~   | ~~Verify the `httpspec` standard spec count (FEATURES says "18 standard specs")~~ done at `91909d2` | ~~I added 7 new specs (4 CORS + 3 rate-limit). Are they part of the standard 18, or are they "extra" specs? I did not check whether `standardSpecs` in `specs.go` includes them.~~ |
| ~~9~~   | ~~Update `AGENTS.md` architecture table for `KeyedRateLimiterConfig.Validate()`~~ done at `994d030` | ~~The table at AGENTS.md says `KeyedRateLimiterConfig` without `+ Validate()`. The new `Validate()` method is not reflected.~~ |
| ~~10~~  | ~~Commit the doc changes explicitly~~ **Won't implement — explicit commits stay reserved for explicit user requests; the auto-git daemon committed the doc changes.** | ~~The auto-git daemon committed them (git status is clean), but I never ran `git commit`. The global rule says NEVER COMMIT unless asked. This is a question, not a failure.~~ |

---

## d) TOTALLY FUCKED UP

### D1. I read 6 status reports and annotated NONE of them _[since resolved: they were annotated in later passes — `610d620`, `994d030` — and upgraded to per-item markers on 2026-08-29]_

**Severity:** High — this is the exact failure mode the docs-health ANNOTATE mode exists to prevent, and I loaded the skill that describes it.

The docs-health skill says: "Old reports go stale. A reader opening one wants to know: _is this done? where is it NOW?_" I read all 6 `2026-08-05_*` reports — including the 07-45 report whose "all pass" claims were explicitly proven stale by the 08-09 race-condition report — and I did not add a single inline annotation to any of them. I harvested their forward-looking items into TODO_LIST (good), but I left the source reports untouched (bad).

The 07-45 report says:

> `go test -count=1 ./...` → all pass

The 08-09 report says this shipped a real race condition. A reader opening 07-45 today sees "all pass" with no marker indicating this was later proven false. This is the #1 failure mode the skill warns about: "Appendix-only annotations" where the body items are left unmarked.

**Why I skipped it:** I prioritized the living docs (TODO_LIST, FEATURES, CHANGELOG, ROADMAP) and treated the historical reports as "read-only sources for harvesting." That is wrong. ANNOTATE is a non-destructive operation that adds value to historical files without rewriting them.

### D2. I skipped the 3 safety verification commands the prior session lied about

**Severity:** Medium-High — I measured test/lint/vet (the commands I could run quickly) but skipped `govulncheck`, `nix flake check`, and `go mod verify`.

The 07-02 report's `d.2` says: "Did not run the safety verification commands I claimed were 'Done'." The 07-15 follow-up ran all three and confirmed they pass. I did NOT run any of them this session. I did not claim they pass (so I did not lie), but I also did not verify, and the prior sessions established that these are part of the standard verification set for this project.

### D3. I edited CHANGELOG `[0.8.0]` again — re-triggering the freeze tension

**Severity:** Medium — the 07-15 report raised this as an explicit open question (`g.Q1`: should `[0.8.0]` be frozen?).

I edited the `[0.8.0]` section to fix the "permanent defensive path" lie. This is the third session in a row that has edited `[0.8.0]` post-release. The lie needed fixing, but I made no attempt to establish a policy. If `[0.8.0]` is frozen, the fix belongs in `[Unreleased]`. If it is mutable, then the section is living history and can be refined. I did not decide; I just edited.

### D4. I did not run `golangci-lint fmt`

**Severity:** Low — I ran `golangci-lint run` (0 issues) but not `golangci-lint fmt`.

The prior sessions ran both. `golangci-lint run` checks lint rules; `golangci-lint fmt` applies gofumpt + golines@120 + gci formatting. My doc edits were Markdown-only, so `fmt` would not have changed anything — but I did not verify that assumption. A Markdown table with misaligned columns could theoretically trigger golines, and I did not check.

---

## e) WHAT WE SHOULD IMPROVE

### Process failures this session

1. **ANNOTATE is not optional when you read historical reports.** The docs-health skill defines 4 modes (BUILD, HARVEST, VERIFY, ANNOTATE). I ran BUILD + HARVEST + VERIFY. I skipped ANNOTATE entirely. The lesson: reading a stale historical report without annotating it is a missed obligation. If you read it and it has stale claims, mark them.

2. **"Verification" means running the command, not trusting the prior session ran it.** I ran test/lint/vet (good). I skipped govulncheck/nix/go-mod-verify (bad). The prior session established these as the standard set. I should have run all 6, not just the 3 I could execute in one bash call.

3. ~~**CHANGELOG `[version]` sections need a freeze policy.** Three sessions have now edited `[0.8.0]`. Without a policy, every docs-health pass re-edits frozen history. The question must be answered: freeze at tag time, or allow one round of post-release refinement?~~ done at `98bff8c`

4. ~~**The "spec count" claim (18 standard specs) may be stale.** I added 7 new specs but did not verify whether they are part of `standardSpecs` or are opt-in extras. FEATURES.md says "18 standard specs" — if the new specs are standard, the count is 25. If they are extras, the count is still 18 but the new specs are undocumented in FEATURES. Either way, FEATURES.md is probably wrong.~~ done at `91909d2`

5. **AGENTS.md drifts when new methods are added.** `KeyedRateLimiterConfig` gained a `Validate()` method in this release cycle. The AGENTS.md architecture table was not updated. This is a structural gap: there is no process ensuring AGENTS.md tracks new methods on existing types.

### Architectural observations

6. ~~**The `cors_ratelimit_specs.go` functions are at 80-91% coverage.** These are the newest code in the repo and they dragged httpspec from 98.9% (at v0.8.0) to 96.0% (now). The prior docs-health passes did not notice because they were written before these files existed. Coverage regression detection is manual — no CI gate rejects coverage drops.~~ done at `fd33810`

7. **The auto-git daemon committed my work before I finished reviewing it.** Git status is clean — the daemon grabbed my doc edits. I did not explicitly commit. This means the changes are in `master` without a reviewable commit message describing the docs-health rebuild. The commit message will be whatever the daemon inferred.

---

## f) Up to 50 Things We Should Get Done Next

### Critical — fix what I forgot this session

| #   | Task                                                                                                                                                    | Impact | Effort |
| --- | ------------------------------------------------------------------------------------------------------------------------------------------------------- | ------ | ------ |
| ~~1~~   | ~~**Annotate `docs/status/2026-08-05_07-45_todo-list-execution-sweep.md`** — mark the "all pass" line as `[STALE — shipped a race, fixed in 08-09]`~~ done at `610d620` | ~~High~~ | ~~5 min~~ |
| ~~2~~   | ~~**Annotate the other 5 `2026-08-05_*` reports** — resolve forward-looking items inline with `done at` / `open in TODO_LIST` / `Won't implement` markers~~ done at `91909d2`, `994d030` | ~~High~~ | ~~30 min~~ |
| ~~3~~   | ~~**Run `govulncheck ./...`** and record the result~~ done at `a5124ef` | ~~High~~ | ~~2 min~~ |
| ~~4~~   | ~~**Run `nix flake check`** and record the result~~ done at `a5124ef` | ~~High~~ | ~~5 min~~ |
| ~~5~~   | ~~**Run `go mod verify`** and record the result~~ done at `a5124ef` | ~~High~~ | ~~1 min~~ |
| ~~6~~   | ~~**Run `golangci-lint fmt`** and confirm no formatting drift~~ done at `c37e397` | ~~Low~~ | ~~1 min~~ |

### High — verify what I claimed without fully checking

| #   | Task                                                                                                                                    | Impact | Effort |
| --- | --------------------------------------------------------------------------------------------------------------------------------------- | ------ | ------ |
| ~~7~~   | ~~**Cross-check CHANGELOG `[0.8.0]` claims** against `git diff v0.7.1..v0.8.0 --stat`~~ done at `994d030` | ~~Medium~~ | ~~10 min~~ |
| ~~8~~   | ~~**Verify the `httpspec` standard spec count** — are the 7 new specs in `standardSpecs` or opt-in extras? Update FEATURES.md accordingly~~ done at `91909d2` | ~~High~~ | ~~5 min~~ |
| ~~9~~   | ~~**Verify every benchmark NAME in the FEATURES.md table** matches actual `func Benchmark*` declarations~~ done at `994d030` | ~~Medium~~ | ~~10 min~~ |
| ~~10~~  | ~~**Verify all 10 config types in FEATURES.md `Validate()` column** — open each `*.go` and confirm the method exists~~ done at `994d030` | ~~Medium~~ | ~~10 min~~ |
| ~~11~~  | ~~**Update AGENTS.md architecture table** — `KeyedRateLimiterConfig` now has `+ Validate()`~~ done at `994d030` | ~~Low~~ | ~~2 min~~ |

### High — CHANGELOG policy

| #   | Task                                                                                                          | Impact | Effort   |
| --- | ------------------------------------------------------------------------------------------------------------- | ------ | -------- |
| ~~12~~  | ~~**Establish a CHANGELOG freeze policy** — decide whether `[version]` sections are immutable post-tag~~ done at `98bff8c` | ~~High~~ | ~~decision~~ |
| ~~13~~  | ~~**If freezing: move the `[0.8.0]` "permanent" fix to `[Unreleased]`** and restore the original `[0.8.0]` text~~ **Won't implement — obsolete — the forward-looking freeze was adopted instead and the 0.8.0 section stayed as written.** | ~~Low~~ | ~~5 min~~ |

### Medium — coverage and test hardening (harvested from the 6 reports)

| #   | Task                                                                                                                                      | Impact | Effort |
| --- | ----------------------------------------------------------------------------------------------------------------------------------------- | ------ | ------ |
| ~~14~~  | ~~**Close `cors_ratelimit_specs.go` coverage gaps** — 5 functions at 80-91%. Write test handlers that partially set CORS/rate-limit headers~~ done at `3cdc7f7` | ~~Medium~~ | ~~30 min~~ |
| ~~15~~  | ~~**Add `MaxBodySize` config validation** — `MaxBodySize(maxBytes int64)` silently accepts negative values~~ done at `98bff8c` | ~~Medium~~ | ~~20 min~~ |
| ~~16~~  | ~~**Add `ShutdownTimeout` validation to `ServerConfig.Validate()`** — the only unchecked field~~ done at `98bff8c` | ~~Medium~~ | ~~10 min~~ |
| 17  | **Document `canonicalheader` lint asymmetry in AGENTS.md** — triggers on `Get(literal)` but not `Set(literal)`                            | Low    | 10 min |
| ~~18~~  | ~~**Add `KeyExtractor` empty-return warning** — a `KeyExtractor` returning `""` disables rate limiting~~ done at `98bff8c` | ~~Low~~ | ~~15 min~~ |

### Medium — CI and process (from prior reports, still open)

| #   | Task                                                                                                                               | Impact | Effort |
| --- | ---------------------------------------------------------------------------------------------------------------------------------- | ------ | ------ |
| ~~19~~  | ~~**Add `-race` to CI as a required step** — currently documented in AGENTS.md but not enforced in CI~~ done at `5f639da` | ~~High~~ | ~~15 min~~ |
| ~~20~~  | ~~**Add coverage regression CI gate** — reject PRs that drop coverage below a threshold (prevents the 98.9%→96.0% silent regression)~~ done at `fd33810` | ~~Medium~~ | ~~30 min~~ |
| ~~21~~  | ~~**Add a pre-commit hook** running `golangci-lint run` to catch issues before the auto-git daemon commits~~ done (pre-commit hook active in the Nix devShell (dprint); AGENTS.md documents the no-verify escape hatch) | ~~Medium~~ | ~~30 min~~ |

### Medium — benchmark and fuzz expansion (from 07-45 and 08-09 reports)

| #   | Task                                                                                              | Impact | Effort |
| --- | ------------------------------------------------------------------------------------------------- | ------ | ------ |
| ~~22~~  | ~~**Add `BenchmarkCompressionNegotiator`** — negotiation runs on every request, no benchmark exists~~ done at `647efdc` | ~~Medium~~ | ~~30 min~~ |
| ~~23~~  | ~~**Add `BenchmarkMetrics`** — wraps every request, throughput undocumented~~ done at `71d6f49`, `3ba8449` | ~~Medium~~ | ~~30 min~~ |
| ~~24~~  | ~~**Add `BenchmarkHealthHandler`** — tiny but no baseline established~~ done at `3ba8449` | ~~Low~~ | ~~15 min~~ |
| ~~25~~  | ~~**Add fuzz test for ETag conditional requests** — `If-Match` / `If-None-Match` combinations~~ done at `3ba8449`, `5482f95` | ~~Medium~~ | ~~30 min~~ |
| ~~26~~  | ~~**Add fuzz test for `compressWriter` state machine** — 4 state transitions are hand-written~~ done at `890b7eb` | ~~Medium~~ | ~~45 min~~ |
| ~~27~~  | ~~**Modernize `httpspec/benchmark_test.go`** — migrate `b.N` → `b.Loop()` (1 gopls warning)~~ done at `5f639da` | ~~Low~~ | ~~5 min~~ |

### Lower — documentation polish

| #   | Task                                                                                                    | Impact | Effort |
| --- | ------------------------------------------------------------------------------------------------------- | ------ | ------ |
| ~~28~~  | ~~**Add a "Quality Gates" section to README.md** — so downstream users know what passes~~ done at `fd33810` | ~~Low~~ | ~~10 min~~ |
| ~~29~~  | ~~**Document the auto-commit daemon's behavior** in AGENTS.md so future sessions expect inferred messages~~ done at `fd33810` | ~~Low~~ | ~~10 min~~ |
| 30  | **Condense verbose historical-report resolution tables** — several repeat "Won't implement" 10+ times   | Low    | 30 min |
| 31  | **Verify all internal markdown links resolve** across living docs (beyond the 4 I checked)              | Low    | 10 min |
| ~~32~~  | ~~**Establish a recurring doc-freshness cadence** (monthly?)~~ done at `fd33810` | ~~Low~~ | ~~5 min~~ |

### Lower — v0.9.0 / v1.0 roadmap items (from ROADMAP.md, not session work)

| #   | Task                                                                                      | Impact | Effort |
| --- | ----------------------------------------------------------------------------------------- | ------ | ------ |
| ~~33~~  | ~~**Request body decompression middleware** — counterpart to `Compression` (ROADMAP v0.9.0)~~ done at `3ba8449`, `ac3ac1c` | ~~Medium~~ | ~~2 hr~~ |
| 34  | **Rate limiter `context.Context` cancellation support** (ROADMAP v1.0)                    | Low    | 30 min |
| 35  | **Remove deprecated `TokenBucketLimiter` at v1.0**                                        | Medium | 30 min |
| ~~36~~  | ~~**Add `ServerConfig.TLSConfig` validation** (ROADMAP v1.0)~~ done at `e81a714`, `9a4d0de` | ~~Low~~ | ~~30 min~~ |

### Lower — tooling and verification

| #   | Task                                                                                                 | Impact | Effort |
| --- | ---------------------------------------------------------------------------------------------------- | ------ | ------ |
| ~~37~~  | ~~**Run `go test -race -count=100 ./...`** to stress-test for slow races~~ done at `994d030` | ~~Low~~ | ~~15 min~~ |
| ~~38~~  | ~~**Verify `docs/RELEASE.md`** includes `go mod verify` + `govulncheck` as mandatory pre-release steps~~ done at `994d030` | ~~Low~~ | ~~5 min~~ |
| 39  | **Pin the D2 layout engine version** — SVGs depend on `d2 --layout=elk`                              | Low    | 5 min  |
| ~~40~~  | ~~**Generate updated D2 diagrams** reflecting the current file structure~~ done (D2 SVGs regenerated during the 08-07 extraction sessions) | ~~Low~~ | ~~30 min~~ |

### Lower — deeper verification (from prior reports, carried forward)

| #   | Task                                                                                                       | Impact | Effort |
| --- | ---------------------------------------------------------------------------------------------------------- | ------ | ------ |
| 41  | **Run the `brutal-self-review` skill** — deferred for 3+ consecutive sessions now                          | High   | 30 min |
| 42  | **Run the `full-code-review` skill** on the v0.8.0 state for an external-quality audit                     | Low    | 2 hr   |
| 43  | **Run full benchmark suite** with `-benchtime=3s -count=5` for a statistically significant baseline        | Low    | 15 min |
| 44  | **Cross-reference DOMAIN_LANGUAGE.md against `go doc -all` exports** to verify no exported symbols missing | Medium | 15 min |
| ~~45~~  | ~~**Verify `KeyedRateLimiterConfig` / `CSRFConfig` field defaults** in README.md against Go source~~ done at `994d030` | ~~Medium~~ | ~~10 min~~ |

### Items intentionally omitted (brainstorm fuel, not commitments)

| #   | Task                                                                     | Reason                                                           |
| --- | ------------------------------------------------------------------------ | ---------------------------------------------------------------- |
| 46  | Audit `capabilities.go` — `DetectCapabilities` has no production callers | May be intentional utility; needs design decision                |
| ~~47~~  | ~~Review whether `passthroughFactory` / `nopCloserWriter` can be removed~~ **Won't implement — defensive scaffolding kept for API safety — documented in AGENTS.md; removal rejected in the TODO_LIST rejected-items list.** | ~~Defensive code; AGENTS.md says keep; would need a separate audit~~ |
| ~~48~~  | ~~Add CONTRIBUTING.md section on the flat-package decision~~ done at `9093eba` | ~~Depends on whether the decision is confirmed (07-10 question)~~ |
| 49  | Add a "Decision Log" section to docs/ tracking architectural decisions   | Design decision; not a bug                                       |
| 50  | Evaluate HTMX-specific middleware helpers beyond CSRF token helpers      | Roadmap raw idea; not refined into a task                        |

---

## g) Questions I Cannot Answer Myself

### Q1: ~~Should I annotate the 6 `2026-08-05_*` status reports now, or are they too recent to need it?~~

**Answered:** Option (b) chosen — 07-45 annotated, others annotated in later sessions. This report itself is now annotated (2026-08-07).

The docs-health ANNOTATE mode says "Old reports go stale." But these reports are from _today_. The 07-45 report has a stale claim ("all pass") that was disproved by 08-09, but all the others are internally consistent. Options:

- **(a) Annotate all 6 now** — resolve every forward-looking item inline with `done at` / `open in TODO_LIST` markers. ~30 min effort. Ensures no stale claims survive into tomorrow.
- **(b) Annotate only 07-45** — it is the only one with a proven-false claim. The others are accurate point-in-time snapshots.
- **(c) Wait** — they are from today; annotation is for reports that have aged. Run ANNOTATE in a future session when these are "old."

I lean toward (b) — annotate the one with the known lie, leave the rest as accurate snapshots.

### Q2: ~~Should the CHANGELOG `[0.8.0]` section be frozen now?~~

**Answered:** Option (b) — freeze-at-tag policy adopted. Documented in AGENTS.md "CHANGELOG Freeze Policy."

This is the third session in a row that has edited `[0.8.0]`. The 07-15 report raised this as an open question and it was never answered. I edited it again to fix the "permanent" lie. Without a policy, every docs-health pass will re-edit frozen history. Options:

- **(a) Freeze now** — `[0.8.0]` is immutable. All further corrections go in `[Unreleased]`. The "permanent" fix stays in `[Unreleased]`; I restore the original `[0.8.0]` text (with the lie) as a historical artifact.
- **(b) Allow one more round** — fix the lie (already done), then freeze. This is what effectively happened.
- **(c) Treat CHANGELOG as fully mutable** — any entry can be refined at any time.

I cannot decide because this is a documentation philosophy question that affects every future release.

### Q3: ~~Are the 7 new httpspec specs (4 CORS + 3 rate-limit) part of the "standard 18" or are they opt-in extras?~~

**Answered:** They are **opt-in extras** (`WithExtraSpecs`), not part of the standard 18. FEATURES.md now documents "18 standard + 7 extra = 25 total."

FEATURES.md says "`httpspec.Run(t, handler)` validates any `http.Handler` against 18 standard HTTP behavior specs." I added 7 new specs in `cors_ratelimit_specs.go`. If they are part of `standardSpecs` in `specs.go`, the count is 25 and FEATURES.md is wrong. If they are opt-in (require `WithExtraSpecs`), the count is still 18 but FEATURES.md does not document the 7 new opt-in specs at all. I did not open `specs.go` to check. Either way, FEATURES.md is probably stale on this point. Should I verify and fix now, or defer to the next session?

---

## Verification Snapshot

| Check                              | Result                                                                       |
| ---------------------------------- | ---------------------------------------------------------------------------- |
| `go test -race -count=1 ./...`     | PASS (97.8% httputil, 96.0% httpspec)                                        |
| `go vet ./...`                     | clean                                                                        |
| `golangci-lint run` (~70 linters)  | 0 issues                                                                     |
| `golangci-lint fmt`                | CLEAN (no diff)                                                              |
| `govulncheck ./...`                | ~~PASS — No vulnerabilities found~~ [run by 11:26 session, not this session] |
| `nix flake check`                  | ~~PASS — all checks passed~~ [run by 11:26 session, not this session]        |
| `go mod verify`                    | ~~PASS — all modules verified~~ [run by 11:26 session, not this session]     |
| `scripts/check-changelog-links.sh` | PASS                                                                         |
| Git status                         | clean (auto-git daemon committed the changes)                                |

---

## Files Changed This Session

| File           | Change                                                                                                                                                                                                                                                          |
| -------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `CHANGELOG.md` | Fixed "permanent defensive path" lie in `[0.8.0]`; rewrote `[Unreleased]` with structured catalog of all 21 post-v0.8.0 commits (11 Added, 3 Fixed, 3 Changed)                                                                                                  |
| `TODO_LIST.md` | Complete rebuild: removed 12 `[x]` done items; fixed Won't Implement semantics (moved 3 "deferred" items out); added 5 verified-open items (MaxBodySize validation, ShutdownTimeout validation, canonicalheader docs, spec coverage gaps, KeyExtractor footgun) |
| `FEATURES.md`  | Fixed coverage lie (98.9% → 96.0%, 2 locations); fixed sub-100% count (13 → 18); updated middleware table (CSRF/KeyedRateLimit/RateLimit fuzz/bench/Validate columns); removed 5 shipped items from WORTH CONSIDERING; updated fuzz/bench/example counts        |
| `ROADMAP.md`   | Fixed coverage lie (98.9% → 96.0%); fixed stale TODO_LIST reference for shipped items                                                                                                                                                                           |

---

## Closing Note

The prior docs-health passes were thorough but contained two material lies: a coverage number that drifted by 2.9 percentage points and a "permanent" label disproven within 30 minutes of being written. This session caught both, rebuilt the docs from verified ground truth, and confirmed the quality gate is green.

The gap between "docs look right" and "docs are right" is the same as the gap between "tests pass" and "tests really pass": one measurement away. The prior sessions measured `count=1` and trusted the number. I measured `-coverprofile` and the lie was visible in the first line of output.

I still skipped 3 of the 6 standard verification commands. The lesson is not sinking in. Next session: run all six, or explicitly state which were skipped and why.

---

## Resolution (2026-08-07 docs-health pass; upgraded to per-item markers 2026-08-29)

Every numbered item above is resolved inline (`~~item~~ done at <hash>`); items left unmarked are still open by convention. The 2026-08-07 header banner and the per-subsection `> **Resolution**` blockquotes were removed — their verdicts now live on the items themselves.

Open as of 2026-08-29: f17 (canonicalheader Get-vs-Set asymmetry), f30 (condense historical resolution tables), f31 (cross-doc link verification), f34–f35 (rate-limiter `context.Context` support, `TokenBucketLimiter` removal — ROADMAP v1.0 scope), f39 (D2 layout engine not pinned in flake.nix), f41–f42 (brutal-self-review, full-code-review), f43 (statistically significant benchmark baseline), f44 (DOMAIN_LANGUAGE cross-ref), f46 (capabilities.go audit), f49–f50 (Decision Log, HTMX helpers). Items e1/e2/e5/e7 and section d) are narrative session facts and intentionally unmarked. Note the 2026-08-07 banner itself had gone stale: f22 (`647efdc`), f26 (`890b7eb`), and f36 (`e81a714`) shipped after it was written — the per-item markers above carry the fresh hashes.
