# Status Report: Self-Critique Backlog + ETag Extraction Cascade

**Date:** 2026-08-07 06:50
**Session scope:** Execute the prior session's 10-item self-critique backlog (D2 SVG, honest-silence comments, FEATURES timestamp, AGENTS.md, LSP restart, art-dupl, coverage tests). Then got interrupted by the auto-commit daemon extracting ETag and Server-Timing into separate modules, which broke the build and required a cascade of cleanup work.
**Starting state:** `ada0c8d` — build broken (3 test files referencing deleted ETag symbols)
**Ending state:** All quality gates green. Build, vet, lint 0 issues, fmt, race -count=1, nix flake check, art-dupl 0 clones. Coverage 96.9% (`go test` reported) / 97.4% (`go tool cover` total).

---

## Session Timeline

1. Read prior session's status report (263 lines) in full
2. Researched current state: git log, git status, verified build/lint/test were green at start
3. Created 10-item todo list from the prior session's self-critique backlog
4. **D2 SVG regenerated** — `d2 --layout=elk` available on PATH, ran successfully, verified SVG shows "Middleware Chain (17)" with Decompression node
5. **6 honest-silence comments added** — `recovery.go`, `compression.go`, `ratelimit.go`, `csrf.go`, `ratelimit_keyed.go`, `decompression.go` — all error-swallow sites now documented
6. **FEATURES.md timestamp updated** to reflect this session's work
7. **AGENTS.md updated** — added honest-silence pattern to Non-Obvious Behaviors, corrected "structurally impossible" sub-package claim to acknowledge server_timing extraction
8. **LSP restarted** — phantom `etag_httpspec_test.go` warnings cleared
9. **art-dupl verified** — 0 clone groups at threshold 2
10. **Coverage tests written** — `TestLimitedReaderCloseError`, `TestLimitedReaderReadError` with `errorReadCloser` helper
11. **ETag extraction cascade hit** — discovered `etag.go` was deleted by auto-commit daemon (`890b7eb`), build broken with undefined symbols in `chain_test.go`, `testutil_test.go`, `example_test.go`, `stack_integration_test.go`
12. **Cascade cleanup** — removed ETag-dependent tests from `chain_test.go`, removed `assertETagEmpty` helper from `testutil_test.go`, fixed `newWriteStatusHandler` signature change across 4 files, removed unused `strings` import, cleaned ETag references in `doc.go`, `hex.go`, `wrapper.go`
13. **D2 diagram updated again** — removed ETag node, updated chain 17→16, regenerated SVG
14. **Living docs updated** — coverage 97.2%→96.9% across FEATURES.md, README.md, ROADMAP.md; FEATURES.md decompression coverage gaps updated (Read and Close now 100%)
15. Final quality gate: all green
16. This self-critique

---

## a) FULLY DONE

### Self-Critique Backlog (10/10 complete)

1. ~~**D2 SVG regenerated twice** — first for Decompression addition (17 middleware), then updated again after ETag extraction (16 middleware). Both `.d2` source and `.svg` artifact now consistent. The prior session's #1 "TOTALLY FUCKED UP" item is resolved.~~ done at `4043884`, `108b3bf`

2. ~~**Honest-silence comments on all 6 remaining error-swallow sites** — `recovery.go:27`, `compression.go:217`, `ratelimit.go:192`, `csrf.go:248`, `ratelimit_keyed.go:206`, `decompression.go:137`. Every `_, _ = w.Write(...)` and `_ = closer.Close()` in non-test source code now has an inline comment explaining why the error is intentionally discarded. The prior session's #2 "TOTALLY FUCKED UP" item is resolved.~~ done at `4043884`, `108b3bf`

3. ~~**FEATURES.md timestamp updated** — now reflects ETag extraction, not stale ETag compliance work. The prior session's #3 "TOTALLY FUCKED UP" item is resolved.~~ done at `4043884`, `108b3bf`

4. ~~**AGENTS.md updated** — honest-silence pattern documented in Non-Obvious Behaviors section with a complete list of all sites. "Structurally impossible" sub-package claim corrected to acknowledge `server_timing` extraction as proof that independent modules CAN be extracted.~~ done at `4043884`, `108b3bf`

5. ~~**LSP restarted** — phantom `etag_httpspec_test.go` diagnostics cleared. The 3 persistent warnings that polluted 3+ sessions of tool output are gone.~~ done at `4043884`, `108b3bf`

6. ~~**art-dupl verified** — 0 clone groups at threshold 2, confirmed twice (before and after ETag removal).~~ done at `4043884`, `108b3bf`

### Coverage Gap Tests

7. ~~**`limitedReader.Close()` error path** — now 100% (was 75%). `TestLimitedReaderCloseError` injects an `errorReadCloser` with `closeErr` and verifies the `fmt.Errorf("decompression close failed: %w", err)` wrapping.~~ done at `4043884`, `108b3bf`

8. ~~**`limitedReader.Read()` non-EOF error path** — now 100% (was 91.7%). `TestLimitedReaderReadError` injects an `errorReadCloser` with `readErr` and verifies the `fmt.Errorf("decompression read failed: %w", err)` wrapping.~~ done at `4043884`, `108b3bf`

### ETag Extraction Cascade Cleanup (unplanned, required)

9. ~~**All dangling ETag references cleaned up** across 10 files:~~ done at `4043884`, `108b3bf`
   ~~- `chain_test.go` — removed 2 ETag tests + unused `strings` import~~
   ~~- `testutil_test.go` — removed `assertETagEmpty` helper (only ETag tests used it)~~
   ~~- `compression_test.go`, `example_test.go`, `server_test.go` — fixed `newWriteStatusHandler` signature (unparam removed always-200 status param)~~
   ~~- `doc.go` — replaced "ETag generation" with "request body decompression with bomb protection"~~
   ~~- `hex.go` — removed "shared by the ETag encoder" from comment~~
   ~~- `wrapper.go` — removed "and etagWriter" from comment~~
   ~~- D2 `.d2` + `.svg` — removed ETag node, updated chain 17→16, regenerated SVG~~

### Living Docs

10. ~~**Coverage numbers updated** 97.2%→96.9% across FEATURES.md (2 locations), README.md (2 locations), ROADMAP.md (1 location). FEATURES.md decompression coverage gaps updated (Read 91.7%→100%, Close 75%→100%).~~ done at `4043884`, `108b3bf`

### Verification

11. ~~**All quality gates green:**~~ done at `4043884`, `108b3bf`
    ~~- `go build ./...` — clean~~
    ~~- `go vet ./...` — clean~~
    ~~- `golangci-lint run` — 0 issues (~70 linters)~~
    ~~- `golangci-lint fmt` — clean~~
    ~~- `go test -race -count=1 ./...` — all pass~~
    ~~- `nix flake check` — all checks passed~~
    ~~- `art-dupl --type-aware -t 2` — 0 clone groups~~
    ~~- D2 `.d2` and `.svg` consistent (both say "Middleware Chain (16)", no ETag node)~~

---

## b) PARTIALLY DONE

1. **`Decompression` function coverage is 78.1%, not higher** — The encoding-filter reject path is the `default:` switch case at `decompression.go:107-110`. This case is structurally unreachable when the allowed set only contains `gzip` and `deflate` — the `switch` on `strings.ToLower(encoding)` can only match those two cases or fall through to the early return at line 87-90 (which checks `!allowed[strings.ToLower(encoding)]` before reaching the switch). The `default:` is defensive code for a custom encoding set where a new encoding is added to `allowed` but not to the `switch`. I wrote the test to exercise the early-return path (which IS tested via `TestDecompressionRespectsEncodingFilter`), but the `default:` case itself cannot be reached without adding a third encoding to the switch. This is acceptable defensive code.

2. **Coverage number discrepancy** — `go test -coverprofile` reports 96.9% but `go tool cover -func` total reports 97.4%. The docs cite the `go test` number (96.9%) because that's what users see. The discrepancy arises from how the two tools aggregate per-package vs cross-package coverage. Not wrong, but worth noting.

---

## c) NOT STARTED

1. ~~**go-etag not wired into workspace** — The `go-etag` repo exists at `../go-etag` (`github.com/larsartmann/go-etag`, package `etag`) but is NOT in `go.work`, NOT in `go.mod`, and NOT in depguard's allowed list. The CHANGELOG says ETag was "moved to `github.com/larsartmann/go-etag`" but httputil cannot actually import it yet. This is a documentation claim that exceeds the implementation.~~ done (resolved — the 22:22 session added go-etag as a dependency and the thin adapter)

2. ~~**Prior session reports remain section-level annotated** — Q3 from the prior-prior session about upgrading 90 historical items to per-item strikethrough is deferred for the 4th time. I chose to work on code/tests/docs instead.~~ done (resolved — later passes annotated the reports; the 2026-08-29 pass upgraded them per-item)

3. ~~**Full benchmark suite not run** — No baselines captured after code changes (ETag removal, honest-silence comments, decompression tests).~~ done (done — benchmarks captured in later sessions (8c1cb47 and the 08-05 11:xx runs))

4. ~~**`Decompression` function 78.1% coverage** — encoding-filter reject `default:` switch case is structurally unreachable with default config (see b.1 above).~~ done (closed — limitedReader/Decompression coverage reached 100% in the 08-07 sessions (54afaa7, ac3ac1c))

---

## d) TOTALLY FUCKED UP

1. **I didn't check whether the auto-commit daemon had already committed changes before I started editing.** When I began this session, the daemon had already committed `890b7eb` (ETag removal) and `e5a1321`/`1a70dd1` (Server-Timing extraction). I started editing files based on the prior session's status report, which described a codebase state that no longer existed. My first edit to AGENTS.md (adding honest-silence pattern with `etagWriter` references) was immediately stale because `etagWriter` had been deleted. I had to re-do the edit without ETag references. I should have run `git log` BEFORE reading the status report to understand what had changed since it was written.

2. **I updated coverage to 96.9% when the tool actually reports 97.4%.** I saw `go test` output saying "coverage: 96.9% of statements" and mechanically sed-replaced all 97.2% references to 96.9%. I didn't notice that `go tool cover -func` reports 97.4% until the self-critique data-gathering phase. The docs now consistently say 96.9% (the `go test` number), but the more precise per-statement tool says 97.4%. I should have verified which number is authoritative before bulk-replacing.

3. **I didn't verify the go-etag claim.** The CHANGELOG says ETag was "moved to `github.com/larsartmann/go-etag`" and ROADMAP says "ETag middleware has been extracted to the `go-etag` module." I read these claims and didn't question them. Only during self-critique did I discover that `go-etag` is NOT wired into `go.work`, NOT in `go.mod`, and NOT in depguard. The module exists at `../go-etag` but httputil cannot import it. The daemon wrote the CHANGELOG entry before the wiring was complete, and I propagated the claim without verifying.

---

## e) WHAT WE SHOULD IMPROVE

### Process

1. **Run `git log` before reading status reports.** The auto-commit daemon changes the codebase between sessions. A status report describes a snapshot that may be hours old. Always check what the daemon committed before treating the report as current truth. This is the same lesson as the prior session's cross-cutting note: "Status reports are point-in-time, not living documents."

2. **Verify module-wiring claims before propagating them.** When a CHANGELOG says "moved to module X," check `go.mod`, `go.work`, and depguard before repeating the claim in other docs. The daemon writes aspirational CHANGELOG entries before wiring is complete.

3. **Use `go tool cover -func | grep total` as the authoritative coverage number.** The `go test` output (96.9%) and `go tool cover` total (97.4%) differ because of aggregation methods. Pick one, use it consistently, and verify it matches before bulk-replacing in docs.

4. **The auto-commit daemon is a write-write conflict source.** It commits while I'm editing. My edits to `chain_test.go` were overwritten by the daemon's `newWriteStatusHandler` signature change. I had to re-fix the file. This happened 3+ times this session. Consider whether the daemon should be paused during active editing sessions, or whether I should `git stash` before each tool call to avoid conflicts.

### Code

5. **`decompression.go:107` `default:` case is dead code** — structurally unreachable when the allowed set only contains gzip/deflate. The `allowed[strings.ToLower(encoding)]` check at line 87 ensures only known encodings reach the switch. Either remove the `default:` case or add a comment explaining it's defensive against future encodings.

6. **go-etag integration is incomplete** — the module exists but isn't wired. Consumers reading the CHANGELOG will expect to import `github.com/larsartmann/go-etag` and use it with httputil, but there's no `replace` directive, no `go.work` entry, and no depguard allowance.

### Documentation

7. **Coverage number inconsistency** — docs say 96.9%, `go tool cover` says 97.4%. Should pick one authoritative source and document the choice in AGENTS.md.

8. **Two status reports from the daemon's extractions are unannotated** — `docs/status/2026-08-07_06-44_etag-extraction-to-go-etag.md` and `docs/status/2026-08-07_06-44_server-timing-module-extraction.md` exist but haven't been read or annotated this session.

---

## f) Up to 50 Things We Should Get Done Next

### P0: Critical Wiring Gaps

1. ~~**Wire `go-etag` into `go.work`** — add `use ./../go-etag` (or move it to `./etag/` within the repo). Effort: 5min.~~ done at `cc6439e`, `aaaed73`
2. ~~**Add `go-etag` to depguard allowed list** in `.golangci.yml`. Effort: 2min.~~ done at `cc6439e`, `aaaed73`
3. ~~**Add `go-etag` dependency to `go.mod`** with `replace` directive if local. Effort: 5min.~~ done at `cc6439e`, `aaaed73`
4. ~~**Verify the CHANGELOG claim** — does `github.com/larsartmann/go-etag` actually build and pass its own tests? Run `cd ../go-etag && go test ./...`. Effort: 5min.~~ done (verified — go-etag builds and passes its own suite as a go.mod dependency)
5. ~~**Reconcile coverage number** — decide whether 96.9% or 97.4% is authoritative and update docs consistently. Effort: 10min.~~ done at `166c181`

### P1: Code Quality

6. **Remove or comment the dead `default:` case** in `decompression.go:107`. Effort: 5min.
7. ~~**Add `go-etag` integration test** — verify httputil + go-etag compose correctly via `Chain()`. Effort: 30min.~~ done at `242aac7`, `77a442c`
8. ~~**Run full benchmark suite** — capture baselines after ETag removal. Effort: 15min.~~ done (done — benchmark baselines captured in later sessions)
9. ~~**Fuzz test for decompression** — random compressed bodies, malformed data, edge-case sizes. Effort: 30min.~~ done (done — decompression_fuzz_test.go exists (08-07 sessions))
10. **Fuzz test for `limitedReadCloser`** — random data sizes around the bomb limit boundary. Effort: 20min.

### P1: Documentation

11. ~~**Read and annotate the 06-44 status reports** — understand what the daemon did during ETag/Server-Timing extraction. Effort: 15min.~~ done (done — the 06-44 reports were annotated by the 08-07 passes and upgraded 2026-08-29)
12. ~~**Update AGENTS.md file table** — remove ETag row (if daemon hasn't already), verify all file entries match current source. Effort: 10min.~~ done (done — the AGENTS.md file table reflects the adapter and the extraction)
13. ~~**Update FEATURES.md** — remove ETag from feature list if not already done. Verify all claims against current source. Effort: 15min.~~ done (done — FEATURES.md reflects the extraction (verified by later passes))
14. **Cross-reference DOMAIN_LANGUAGE.md** — verify no stale ETag entity/event references remain. Effort: 10min.
15. **Add coverage methodology note to AGENTS.md** — document that `go test` and `go tool cover -func` report different numbers and which is authoritative. Effort: 5min.

### P2: Testing Hardening

16. **Decompression + MaxBodySize interaction test** — verify body-size limit applies to decompressed size. Effort: 20min.
17. **Decompression + Compression chain test** — compressed request body, compressed response. Effort: 20min.
18. **CSRF + Server-Timing chain test** — verify Server-Timing header on CSRF-rejected responses. Effort: 15min.
19. **KeyedRateLimit bomb-protection test** — verify `MaxKeys` eviction under rapid key creation. Effort: 20min.
20. **Recovery + Logging chain test** — verify panic is logged AND recovered. Effort: 15min.
21. ~~**Fuzz `parseETagList`** (now in go-etag module) — focused quote/comma/backslash/escape combinations. Effort: 30min.~~ **Won't implement — moved — parseETagList fuzzing is go-etag responsibility.**

### P2: Ecosystem Verification

22. ~~**Verify `go-etag` module is self-contained** — only depends on `go-error-family`, not on httputil. Effort: 10min.~~ done (verified — go-etag depends only on go-error-family)
23. ~~**Verify `server_timing` module is self-contained** — stdlib-only, zero external deps. Effort: 5min.~~ done (verified — server_timing is stdlib-only (documented in AGENTS.md))
24. ~~**Run `cd ../go-etag && golangci-lint run`** — verify the extracted module passes lint. Effort: 5min.~~ done (clean — go-etag lint runs 0 issues in later sessions)
25. ~~**Run `cd server_timing && golangci-lint run`** — verify the sub-module passes lint. Effort: 5min.~~ done (clean — the sub-module lint runs 0 issues per the documented cadence)

### P2: Documentation Polish

26. ~~**Add `nix run .#vulncheck` to RELEASE.md** — document the vulncheck app in the release runbook. Effort: 10min.~~ done at `994d030`
27. **Verify all internal markdown links resolve** across living docs. Effort: 15min.
28. **Pin D2 layout engine version** in flake.nix — SVGs depend on `d2 --layout=elk`. Effort: 5min.
29. **Update RELEASE.md** with `go-etag` and `server_timing` module release steps. Effort: 15min.
30. ~~**Review `docs/v1-stability.md`** — classify every exported symbol, verify none missing. Effort: 30min.~~ done (done — every exported symbol classified (maintained since b90616e))

### P3: v1.0 Preparation

31. ~~**Audit all `Validate()` methods for completeness** — ensure every config struct has one. Effort: 30min.~~ done (done — all config types validated (08-08 sessions; AGENTS.md lists Validate for every config))
32. ~~**Add `ServerConfig.TLSConfig` validation** — `server.go`. Effort: 30min.~~ done at `e81a714`, `9a4d0de`
33. **Remove deprecated `TokenBucketLimiter`** — or confirm v1.0 timeline for removal. Effort: 1hr.
34. ~~**Evaluate `MiddlewareDecompression` constant** — add to `MiddlewareStack` name constants. Effort: 10min.~~ done (done — MiddlewareDecompression is in the stack constants (stack.go))
35. ~~**Consider `MiddlewareCSRF` and `MiddlewareServerTiming` constants** — add to stack name constants if not present. Effort: 10min.~~ done (done — MiddlewareCSRF and MiddlewareServerTiming are in the constants (stack.go:19-20))

### P3: Code Quality

36. **Run `brutal-self-review` skill** — deferred 6+ sessions. Effort: 30min.
37. **Run `full-code-review` skill** — never actually run. Effort: 2hr.
38. **Run `architecture-review` skill** — assess structural health after ETag + Server-Timing extraction. Effort: 30min.
39. **Review all error-swallow sites for classification opportunities** — should any get `errorfamily.Wrap`? Effort: 30min.
40. **Consider extracting `responseWrapper` into shared internal package** — if go-etag also needs it. Effort: 1hr.

### P3: Historical Report Annotation

41. ~~**Annotate the 06-44 etag-extraction report** — verify its claims against current source. Effort: 10min.~~ done (done — annotated and upgraded to per-item markers on 2026-08-29)
42. ~~**Annotate the 06-44 server-timing report** — verify its claims against current source. Effort: 10min.~~ done (done — annotated and upgraded to per-item markers on 2026-08-29)
43. ~~**Upgrade 10-32 and 11-26 to per-item strikethrough** — 90 items, deferred 4 sessions. Effort: 1hr.~~ done (done — both upgraded to per-item markers on 2026-08-29)
44. ~~**Annotate the 05-10 report** — its items are partially resolved across 2 sessions. Effort: 20min.~~ done (done — annotated and upgraded to per-item markers on 2026-08-29)

### P3: Product Direction (needs user input)

45. ~~**Decide: `If-Match` / `If-Unmodified-Since` support** — now lives in go-etag module. Effort: decision.~~ **Won't implement — moved — conditional-request scope is evaluated in go-etag (ROADMAP).**
46. ~~**Decide: `Last-Modified` + `If-Modified-Since` suite** — now lives in go-etag module. Effort: decision.~~ **Won't implement — moved — conditional-request scope is evaluated in go-etag (ROADMAP).**
47. ~~**Decide: `If-Range` support** — now lives in go-etag module. Effort: decision.~~ **Won't implement — moved — conditional-request scope is evaluated in go-etag (ROADMAP).**
48. ~~**Decide: Should httputil re-export go-etag's `ETag()` for backward compatibility?** Effort: decision.~~ done (done — the deprecated ETag() adapter re-exports; the Middleware alias makes etag.New() composable)
49. ~~**Decide: Should `server_timing` be published as independent versioned module?** Effort: decision.~~ done (answered by practice — lockstep tagging (the v0.9.1 sub-module bump at 7e964b7))

### P3: Architecture

50. **Consider a `go.mod` workspace checker** — script that verifies all `replace` directives point to directories that exist and have valid `go.mod` files. Prevents the "aspirational CHANGELOG" problem. Effort: 30min.

---

## g) Questions I Cannot Answer Myself

### Q1: ~~Should `go-etag` be wired into `go.work` now, or is the daemon going to do it as part of an ongoing extraction?~~

**Answered:** wired — go-etag is a go.mod dependency with a replace directive (`cc6439e`, `aaaed73`).

The CHANGELOG says ETag was "moved to `github.com/larsartmann/go-etag`" and the module exists at `../go-etag` with a valid `go.mod` and source files. But `go.work` only lists `.` and `./server_timing` — no `go-etag` entry. And `go.mod` has no `go-etag` dependency or `replace` directive. And depguard doesn't allow `go-etag` imports. I don't know if this is intentional (the daemon will wire it in a later commit) or an oversight I should fix. If I wire it now, I might conflict with a planned daemon commit. If I don't, the CHANGELOG claim is dishonest.

### Q2: ~~Is 96.9% or 97.4% the authoritative coverage number?~~

**Answered:** reconciled — 97.0% (`go test`) / 97.5% (`go tool cover`) per the docs-health pass; later sessions re-measured.

`go test -coverprofile=./...` prints "coverage: 96.9% of statements" in its output line. `go tool cover -func=cov.out` reports "total: 97.4%". The 0.5pp difference appears to stem from how each tool aggregates per-package coverage profiles. The docs currently say 96.9% (the `go test` number) because that's what a user running `go test` will see. But 97.4% is the more granular per-statement number. I don't know which you consider authoritative, and I don't want to pick wrong and have to re-update all docs again.

### Q3: ~~Should I pause or disable the auto-commit daemon during active editing sessions?~~

**Answered:** by practice — the daemon stayed active; AGENTS.md documents to expect inferred messages and to use explicit `--no-verify` commits when needed.

The daemon committed changes 3+ times during this session while I was editing, causing write-write conflicts. My edits to `chain_test.go` were overwritten by the daemon's `newWriteStatusHandler` signature change, requiring re-work. The daemon also extracted ETag and Server-Timing into separate modules mid-session, which broke the build and required a cascade of cleanup. The daemon's commits have empty messages (commits `108b3bf` and `a8ebe7b` have blank subjects), making history harder to read. I don't know if this daemon behavior is intentional and I should work around it, or if it should be configured to skip commits when files are actively being edited.

---

## Resolution (2026-08-07 docs-health pass; upgraded to per-item markers 2026-08-29)

Every actionable numbered item is resolved inline; unmarked items are still open by convention. The header banner was removed — its verdicts live on the items.

Open as of 2026-08-29: f6 (dead `default:` case in decompression.go), f10 (limitedReadCloser fuzz), f14 (DOMAIN_LANGUAGE stale-ETag sweep), f15 (coverage methodology note in AGENTS.md), f16–f20 (cross-middleware chain tests: Dec+MaxBodySize, Dec+Compression, CSRF+ServerTiming, KeyedRateLimit eviction, Recovery+Logging), f27 (cross-doc link verification), f28 (D2 layout pin), f29 (RELEASE.md module release steps), f33 (TokenBucketLimiter removal — ROADMAP v1.0), f36 (brutal-self-review), f37 (full-code-review), f38 (architecture-review re-run), f39 (error-swallow classification sweep), f40 (shared responseWrapper), f50 (workspace checker). Sections b), d), e) are narrative session facts and process lessons, intentionally unmarked.
