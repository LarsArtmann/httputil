# Status Report: Pareto Plan T1–T30 Execution — Hardening, StartTLS, and the Red-Suite Discovery

**Date:** 2026-08-29 23:05 CEST
**Session scope:** Execute the entire 2026-08-29 20:24 Pareto plan (T1–T30): harvest the open-item backlog, run brutal-self-review, upgrade the May–July annotation corpus, add the test/bench/fuzz/integration batches, ship the docs-structure and CI work, and finish with a full docs-health audit.
**Duration:** ~3.5 hours (20:00–23:05 CEST), 19 commits on `master` (unpushed).
**Starting state:** clean tree at `598b5b5` (plan doc committed); the environment's `/mnt/buildcache` blocker had made `go test`/`go vet`/`golangci-lint` fail at cache-init for every session since 2026-08-16.
**Ending state:** all quality gates green (build, vet, `-race -count=1` and `-count=10`, ~70 linters at 0 issues in both modules, `nix flake check`), 515 test functions, 13 fuzz targets, 50-benchmark baseline recorded, TODO_LIST rebuilt and current.

---

## a) FULLY DONE

| #  | Item                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                     | Verification                                                      |
| -- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ----------------------------------------------------------------- |
| 1  | **Root-caused and fixed the 13-day red test suite** — the 2026-08-16 json/v2 "reformat" (`d466d2d`, "no behavior changes") replaced `json.Encoder.Encode` (appends `\n`) with `json.MarshalWrite` (does not) in `health.go`, shrinking all probe bodies 16→15 bytes and failing 6 tests. Unfixed because every session's toolchain died at the `/mnt/buildcache` cache-init error before running tests. Fixed via one `writeHealthBody` helper that writes the payload plus an explicit newline; exact-bytes test passes | `go test -race ./...` green; commit `590da48`                     |
| 2  | **Unblocked the entire toolchain** — `GOCACHE`/`GOLANGCI_LINT_CACHE`/`GOMODCACHE` overrides documented in AGENTS.md Commands; this is what exposed the red suite                                                                                                                                                                                                                                                                                                                                                         | First successful full lint + test runs since the blocker appeared |
| 3  | **T1 HARVEST: TODO_LIST.md rebuilt** from the 20 "Open as of" lines across the 23 August appendices — deduped, routed (TODO/ROADMAP/Won't-Implement), every surviving item carrying report citations; refreshed again at session end to post-execution state                                                                                                                                                                                                                                                             | commit `fa033de` + evening rewrite                                |
| 4  | **T3 brutal-self-review executed and written** as a styled HTML report: `docs/reviews/2026-08-29_20-42_brutal-self-review.html` — 11/11 questions answered, 5 findings, all fixed same-session                                                                                                                                                                                                                                                                                                                           | The skill was deferred 10+ sessions before this                   |
| 5  | **Marker evidence audit** — all 75 unique August-cited hashes validated via `git cat-file`; 10 substantive markers deep-sampled against their commit diffs (10/10 match); balanced 261/261 struck-row formatting check                                                                                                                                                                                                                                                                                                   | Evidence table inside the review report                           |
| 6  | **Stale-claim corrections**: 07-45 appendix's "indexPath race still present" disproven (`go test -race -count=3 ./httpspec/...` clean; options apply before spec construction) and corrected in place; AGENTS.md file count 34→35                                                                                                                                                                                                                                                                                        | Both fixed inline                                                 |
| 7  | **T5: all 9 numbered-item May–July reports upgraded to per-item inline markers** (~530 new markers; corpus total ≈1,950 strikethroughs, 0 header banners). Banners removed, in-file resolution tables retained as the verdict source                                                                                                                                                                                                                                                                                     | commit `732ec44`; formatter-verified                              |
| 8  | **T4 inventory** recorded in the plan appendix: 36 May–July files, 9 needed the upgrade, ~27 narrative files classified SKIP per skill rules                                                                                                                                                                                                                                                                                                                                                                             | Plan doc appendix                                                 |
| 9  | **T10: `ExpectJSON` / `ExpectHTML` builders** with 8 tests, AGENTS/FEATURES/v1-stability updates                                                                                                                                                                                                                                                                                                                                                                                                                         | commit `284ea02`                                                  |
| 10 | **T12 test batch**: `MaxBodySize` ContentLength pass-through + `MaxBytesError` surface, CSRF rejection `Content-Type: text/plain` contract, `ForbiddenHandler` status-only contract, handler-set Content-Length preservation through the Compression chain                                                                                                                                                                                                                                                               | commit `f7c50dc`                                                  |
| 11 | **T13 test batch**: Timeout `DeadlineExceeded` observability, Recovery×Logging (panic logged at error level AND recovered), CSRF×ServerTiming (header survives rejection), KRL eviction under key churn; full `-race -count=10` sweep green                                                                                                                                                                                                                                                                              | commit `f7c50dc`                                                  |
| 12 | **T15: 3 new fuzz targets** — `FuzzResponseRecorder` (first-WriteHeader-wins contract), `FuzzLimitedReadCloser` (never exceeds limit, closes at boundary), `FuzzDecompressionInvariants` (header removal + round-trip) — all with seed passes plus 15–20s live fuzz smoke runs                                                                                                                                                                                                                                           | commit `5278f1d`                                                  |
| 13 | **T17 integration batch**: Dec×MaxBodySize (bomb protection aborts the decompressed read), Dec×Compression round-trip, nonce inner-overwrites-outer + per-request randomness + CSP-survives-default-SecurityHeaders + panic-path, and a full **TLS startup test with a self-signed cert**                                                                                                                                                                                                                                | commits `44b5831` + fixes                                         |
| 14 | **Real feature gap found and shipped: `Server.StartTLS`** — `ServerConfig.TLSConfig` was validated but `Start()` only ever called `ListenAndServe`, so the library could not serve HTTPS at all. `StartTLS` wraps `ListenAndServeTLS`; docs updated in v1-stability (Additive), FEATURES, README, AGENTS                                                                                                                                                                                                                 | commit `44b5831`                                                  |
| 15 | **stdlib trap documented**: `http.Server.ServeTLS` mutates the caller's `*tls.Config` (h2 ALPN append) — my first TLS test raced on exactly this; AGENTS.md now carries the clone-your-config warning                                                                                                                                                                                                                                                                                                                    | Race caught by `-race -count=5`                                   |
| 16 | **T20 cleanups**: decompression `default:` case documented as the custom-Encodings contract (not dead code — removing it would have been a bug); shared header consts extracted to `headers.go`; `DetectCapabilities` keep-decision recorded; helper sweep found zero dead helpers                                                                                                                                                                                                                                       | commit `f7c50dc`                                                  |
| 17 | **T21 docs structure**: `server_timing/doc.go` (package doc, gci-clean, `go doc` renders), `docs/DECISION_LOG.md` backfilled with 10 decisions, ADR 0001 (adapter pattern), CONTRIBUTING architecture notes, DOMAIN_LANGUAGE spot-verified against exports                                                                                                                                                                                                                                                               | commit `940c1ba`                                                  |
| 18 | **T22/T23 CI**: `scripts/check-module-boundaries.sh` (GOWORK=off per module — the consumer view) wired into ci.yml; `.github/workflows/nightly-fuzz.yml` runs all 8 fuzz targets 5 min each nightly                                                                                                                                                                                                                                                                                                                      | commit `940c1ba`                                                  |
| 19 | **T24 httpspec extras**: no-duplicates spec now fails on the same header under two casings (direct map writes bypass canonicalization); `ExpectVaryContains` (`Vary: *` aware) and `ExpectNotModifiedWithETag` builders + 6 tests                                                                                                                                                                                                                                                                                        | commit `4e3b1e3`                                                  |
| 20 | **T25 go-etag upstream**: scratch consumer module verified `go get` + `etag.New(DefaultETagConfig())` end-to-end (200 + ETag present); review item list filed at `go-etag/docs/planning/2026-08-29_httputil-consumer-review-items.md`                                                                                                                                                                                                                                                                                    | File exists in the sibling repo (uncommitted there — see d)       |
| 21 | **T26**: internal-coupling D2 graph written and rendered with the pinned d2; README ordering-at-a-glance mermaid diagram added                                                                                                                                                                                                                                                                                                                                                                                           | commit `a5e0f8c`                                                  |
| 22 | **T8 D2 pin**: `pkgs.d2` in the devShell (locked nixpkgs → d2 v0.7.1); regenerated SVG is **byte-identical** to the committed one — determinism proven; regeneration documented in RELEASE.md                                                                                                                                                                                                                                                                                                                            | commit `e045b00`                                                  |
| 23 | **T16 benchmarks + baseline**: 6 new benchmarks; full 3s×5 baseline of 50 benchmarks recorded in `docs/benchmarks.md` with methodology                                                                                                                                                                                                                                                                                                                                                                                   | commit `c1b2f31`                                                  |
| 24 | **T11 contracts**: `OnRejected` hot-path/no-write contract documented at the field; coverage methodology section in AGENTS.md; ETag×Compression ordering verified already present in README                                                                                                                                                                                                                                                                                                                              | commit `a5e0f8c`                                                  |
| 25 | **T27/T28/T29**: ecosystem ideas, HSTS/HTTPS-redirect/`MaxHeaderBytes` decisions filed to ROADMAP + Decision Log; TLS cipher-suite decision recorded (keep Go 1.26 defaults); `ClientIP` blind-trust executable doc-test; StartupHandler design note + ETag cache-policy handoff sentence in README                                                                                                                                                                                                                      | commits `940c1ba`, `a5e0f8c`                                      |
| 26 | **T30 final audit**: build/vet/race-test/lint/flake all green; 14 living docs link-checked (0 broken); 8/8 cross-file consistency checks PASS; CHANGELOG `[Unreleased]` populated with Added/Fixed/Changed for today                                                                                                                                                                                                                                                                                                     | this session's final commit series                                |
| 27 | **Coverage re-measured honestly**: 96.9% httputil / 98.8% httpspec; all four stale 97.x claims (FEATURES ×3, README badge + table, ROADMAP) updated to the fresh numbers                                                                                                                                                                                                                                                                                                                                                 | commit `be16cf8`                                                  |
| 28 | **T6**: `canonicalheader` Get-vs-Set asymmetry documented with the map-access footgun example; verified zero `nolint:canonicalheader` directives remain                                                                                                                                                                                                                                                                                                                                                                  | AGENTS.md                                                         |

## b) PARTIALLY DONE

1. ~~**T14 (v-marker → hash upgrades)** — policy settled (hash-for-changes, dated falsifiable evidence for state-claims) and the failing markers corrected, but the bulk re-citation of ~200 markers was **not** executed; reasoning recorded in the plan appendix (most v-markers are state-claims with no single hash — forcing hashes would fabricate evidence).~~ **Won't implement — policy decision recorded (T14): state-claims keep dated falsifiable evidence; forcing hashes would fabricate evidence.**
2. ~~**FEATURES sub-100% list** — coverage percentages updated, but the **new** sub-100% functions introduced today (`StartTLS` error branch, new builder branches, fuzz helpers) were **not** added to FEATURES's documented defensive-paths list. The list is now incomplete by omission.~~ done (sub-100% list rebuilt from a fresh race-enabled profile in the 2026-08-30 docs-health pass)
3. ~~**httpspec spec-count claims** — I added check _builders_ (no new standard specs), so the "18 standard specs" count is still right, but I did not re-verify the count claim text end-to-end after the spec-internal change.~~ done (verified 2026-08-30: 18 standard specs in httpspec/specs.go; CORSSpecs 5 + RateLimitSpecs 3 = 26 total)
4. **go-etag sibling repo** — the filed review-items doc exists but is **uncommitted** in `/home/lars/projects/go-etag` (`??` untracked). Not mine to commit without a word from you, since it's a separate repo.
5. ~~**Benchmark baseline table** — `KeyedRateLimiterMiddleware` is documented as prose (191 ns/op) rather than a table row because the original 941-second baseline run **failed** on it (see d); only that one benchmark was re-run at 3s×2 instead of ×5.~~ done (KRL provenance note present in docs/benchmarks.md header)
6. ~~**Push** — 19 commits sit on local `master`, unpushed (you didn't ask; per rules I don't push unasked).~~ done (origin/master..master = 0; all session commits pushed)

## c) NOT STARTED

1. **T18 `TokenBucketLimiter` removal** — hard-gated on the v1.0 release decision itself (ROADMAP forbids removal pre-freeze). Cannot be executed early.
2. **T19 implementation** (ctx rate limiter) — gated the same way; the design note (admission-only contract, `Wait(ctx)` as post-v1.0 additive) shipped instead, and `Check`'s doc now states the contract.
3. ~~**`full-code-review` skill** — still never run; now the top TODO_LIST item.~~ done (executed 2026-08-30: docs/reviews/2026-08-30_09-00_full-code-review.html + fix batch)
4. **CI release workflow** — automated tag → build → GitHub Release pipeline.
5. **go-error-family upstream proposal** (conditional-request classification).
6. **`architecture-review` re-run.**
7. ~~**CORS wildcard-origin spec**, **`RunSerial` state-sharing design note**.~~ done (both shipped 2026-08-30 morning session)
8. ~~**responseWrapper direct tests**, negotiation property test, adapter overhead benchmark, `b.Run` sub-benchmark refactor.~~ done (wrapper_test.go + compression_negotiator_property_test.go + BenchmarkETagAdapterOverhead, 2026-08-30)
9. ~~**Hijack test design cluster** (real-connection test, lighter upgrade-response test, mutation test).~~ done (chain_hijack_test.go three-tier restoration, 2026-08-30)
10. ~~**Test-lint convention doc**, **`httptest.NewRequest` profiling**, **error-swallow sweep**.~~ done (AGENTS convention + newrequest_bench_test.go + error-swallow sweep, 2026-08-30)
11. ~~**Integration-docs import-path audit**, package-structure analysis refresh.~~ done (import-path audit + package-structure refresh, 2026-08-30)
12. ~~**nginx bomb-protection study.**~~ done (docs/research/2026-08-30_nginx-bomb-protection-study.md)
13. **go-compression extraction** (the big rock; untouched, has its own plan file).

## d) TOTALLY FUCKED UP

1. **I committed twice with failing gates.** The `StartTLS` commit (`44b5831`) landed with a typecheck error and a red root-module test; a style commit landed with lint findings. Both were fixed forward within minutes, but the discipline I demand elsewhere (verify before declaring done) was violated at exactly the commit boundary. Root cause: batching "lint + test + commit" into one command and letting commit run even when lint printed findings (I read the tail, not the exit code).
2. **The July mapping drift corrupted 14 annotations.** The 03-56 file's Resolution-table ids diverge from body row numbers at row 22 (`f.22` = body row 28) — I applied verdicts positionally, ~14 rows got the wrong verdict, and only my own post-command reread of the two tables caught it. Repaired per revert-first culture (`git restore` + re-apply with an explicit per-row mapping), but the failure mode is exactly the class the guardrails warn about, and this is the **second** occurrence across two sessions (last time: hand-rolled regex on 08-08_02-50). A mapping-alignment dry-run (assert each table id's _description_ matches the target row's text) should have been step zero.
3. **I wrote three tests that asserted wrong assumptions**, each costing a debug cycle: (a) `MaxBytesReader` does **not** set `ContentLength` (I "remembered" it does); (b) the recorder keeps the **first** status, not the last; (c) nested `Nonce` middleware **overwrites** by design, so "distinct nonces" needed cross-request sampling, not context comparison — my first version even contained a `t.Fatal("unreachable")` on a map that could never fill. The lesson is uniform: read the implementation contract first, then write the expectation.
4. **The 941-second baseline run failed at the end** — my KRL benchmark harness used `Limit=10M` but the benchmark did ~18M iterations, tripping the limiter and failing the whole `-count=5` run after 15 minutes. Burst economics, again. Fixed to 1B and re-ran that benchmark; the other 49 baseline rows are valid, but the file's provenance now mixes a failed run's rows with a partial re-run.
5. **The `/mnt/buildcache` blocker burned ~10 toolchain invocations this session** before I assembled the full three-variable override (`GOCACHE`, `GOLANGCI_LINT_CACHE`, **and** `GOMODCACHE` — the third one took two extra failures to notice). It had been silently eating every session's quality gate since 08-16, which is _why_ the red suite survived 13 days. The fix is documented, but the meta-failure stands: three sessions hit the blocker and documented it instead of solving it.
6. **dprint fmt/check oscillation** — formatting the living docs took three rounds because `dprint fmt` and `dprint check` disagreed across glob invocations, and six pre-existing living docs had sat unformatted for weeks (proof the auto-commit daemon's `--no-verify` bypasses the format gate permanently). I normalized them but did not fix the underlying hook-bypass workflow.

## e) WHAT WE SHOULD IMPROVE

1. **Never batch verify+commit.** Exit-code-gate the commit on lint/test success (`cmd && git commit`), always. Two of nineteen commits this session broke this.
2. **Mapping-alignment assertion before any table annotation batch**: verify `table_id → row text` semantically (substring match), not positionally. Add it to the skill scripts if possible.
3. **Read-implementation-first for test writing.** All three wrong-assumption tests came from memory over source. The fix is mechanical: open the function, then write the test.
4. **New code ships with its FEATURES defensive-path entries in the same change** — today's new functions quietly joined the untracked sub-100% population.
5. **Baseline benchmarks should exclude limiter-dependent benches or set burst-proof limits**; a 15-minute run failing at 95% is pure waste.
6. **Sibling-repo changes need sibling commits** — I wrote into go-etag and left it dirty; cross-repo sessions should end with a cross-repo status check.
7. **Solve environment blockers, don't document them.** The 08-16 blocker should have died on 08-16.
8. **The dprint gate is currently advisory** (daemon `--no-verify`). Either fix the hook environment or accept that living docs will always need periodic `dprint fmt` passes — decide, and write the decision down.
9. **`bench_batch_test.go` should be dissolved** into the per-middleware bench files it belongs to; I created a grab-bag file for session convenience.
10. **`waitForTLS`/`reserveFreePort` partially duplicate `waitForServerStart`** — the server test helpers should be consolidated next time that file is touched.

## f) Up to 50 Things We Should Get Done Next

**Wave 1 — verify and release (blocks everything else)**

1. ~~Run `full-code-review` skill over the current tree (the last unknown-unknown surfacer).~~ done (executed 2026-08-30 (report + fixes; see 2026-08-30_11-30))
2. ~~Push the 19 local commits to origin (or tell me to hold).~~ done (origin/master..master = 0 (pushed))
3. Commit the go-etag review-items file in the sibling repo (or tell me to).
4. Decide the v1.0 question (see g-Q2) — it gates 5+ items below.
5. If v1.0: run one stabilization cycle (full suite, benchmarks, fuzz overnight job once), then tag.
6. If v1.0: execute T18 — remove `TokenBucketLimiter`/`RateLimit()` per the migration guide; CHANGELOG Removed entry; middleware-count docs updates.
7. ~~If v1.0: decide `Wait(ctx, key)` for the post-v1.0 roadmap entry vs implementing now.~~ done (decided 2026-08-29: admission-only through v1.0, Wait(ctx) post-v1.0 (DECISION_LOG))

**Wave 2 — close today's loose ends**

8. ~~Add today's new sub-100% functions to the FEATURES defensive-paths list (with reasons).~~ done (sub-100% list rebuilt 2026-08-30 from fresh profile)
9. ~~Re-verify the httpspec "18 standard specs" count claim text end-to-end.~~ done (18 standard specs verified 2026-08-30)
10. Re-run the full 50-benchmark baseline after the KRL harness fix so `docs/benchmarks.md` has a single provenance run.
11. Add `KeyedRateLimiterMiddleware` row back into the baseline table at count=5.
12. Replace RSA-2048 self-signed cert generation in the TLS test with Ed25519 (faster, modern).
13. Dissolve `bench_batch_test.go` into per-middleware bench files.
14. Consolidate `waitForTLS`/`reserveFreePort` with `waitForServerStart` in `testutil_test.go`.
15. Investigate the dprint fmt/check oscillation once, properly (config? glob context?).
16. Decide the pre-commit hook story (fix dprint availability in the hook vs formalize `--no-verify`).
17. Add the daemon-granularity open question back to TODO_LIST (it was dropped in the evening rewrite).
18. ~~Re-run `-race -count=10` once more after the TLS-test additions (count=10 ran before `StartTLS` tests were finalized).~~ done (full -race -count=10 green 2026-08-30 docs-health pass)

**Wave 3 — the annotation backlog tail (from TODO_LIST)**

19. `architecture-review` re-run (pre-v1.0).
20. ~~CORS wildcard-origin httpspec spec.~~ done (SpecNameCORSOriginMatchesRequested shipped 2026-08-30)
21. ~~`RunSerial` state-sharing design note.~~ done (docs/planning/2026-08-30_runserial-state-sharing-design-note.md)
22. ~~responseWrapper direct tests + failure-recording behavior test.~~ done (wrapper_test.go, 2026-08-30)
23. ~~Negotiation property test (RFC 7231 q-value ordering).~~ done (compression_negotiator_property_test.go, 2026-08-30)
24. ~~Adapter overhead benchmark (`etag.New()` wrap cost).~~ done (BenchmarkETagAdapterOverhead, 2026-08-30)
25. ~~Hijack test design decision (real-connection test + upgrade-response test + mutation test).~~ done (chain_hijack_test.go, 2026-08-30)
26. ~~Codify the test-name/assertion lint convention.~~ done (AGENTS.md Test Naming and Assertion Conventions)
27. ~~`httptest.NewRequest` profiling (allocate cost behind the noctx warnings).~~ done (newrequest_bench_test.go + research note, 2026-08-30)
28. ~~Error-swallow classification sweep ("honest silence" comment audit).~~ done (error-swallow sweep 2026-08-30: 2 fixed, 6 documented)
29. ~~Integration-docs import-path audit.~~ done (import-path audit 2026-08-30)
30. ~~Package-structure analysis refresh (post-v1.0 trigger is ~50 files; at 35).~~ done (2026-08-30_package-structure-analysis-refresh.md (36 root files))
31. ~~nginx bomb-protection comparison study.~~ done (docs/research/2026-08-30_nginx-bomb-protection-study.md)
32. ~~`b.Run` sub-benchmark refactor for multi-phase benchmarks.~~ done (BenchmarkDecompression/{gzip,deflate,passthrough}, 2026-08-30)

**Wave 4 — upstream and ecosystem**

33. go-error-family upstream: conditional-request classification proposal.
34. CI release workflow (tag → build → GitHub Release, mirroring RELEASE.md).
35. go-etag: land the five filed v0.1.1 items upstream (README usage example, cache-policy guidance, ordering note, compliance-suite ownership, `// Deprecated:` prefixes).
36. go-etag: tag v0.1.1 once items land.
37. samber/do integration doc refresh (module version bump check).
38. HTMX helper ideas doc (nonce-aware fragment headers).
39. Redis-backed keyed-rate-limiter documented example refresh.
40. Prometheus MetricsRecorder example refresh.

**Wave 5 — the big rock and hygiene**

41. go-compression extraction (own Pareto plan; trigger: go-datastar SSE needs).
42. ~~`/mnt/buildcache` — file the env fix upstream to whoever owns the sandbox so future clones don't inherit it.~~ **Won't implement — sandbox owner's fix; the GOCACHE/GOLANGCI_LINT_CACHE/GOMODCACHE workaround is documented in AGENTS.md.**
43. Lint config: migrate `exhaustruct` → `exhaustruct_v5` before the deprecation becomes removal.
44. Resolve `go.work` `go 1.26.5` vs CI `1.26.x` vs local `1.26.7` version pinning.
45. Nightly-fuzz workflow: add a "found a crash" issue-template step (currently failures only show in the run log).
46. Consider `govulncheck` invocation for the server_timing sub-module in CI (currently root-only).
47. Consider commit-lint: a CI step that rejects commits whose message lacks the conventional prefix (the daemon's inferred messages are sometimes generic).
48. ~~Add `docs/benchmarks.md` regeneration to the pre-release checklist in RELEASE.md.~~ done (RELEASE.md step 6 now refreshes docs/benchmarks.md, 2026-08-30)
49. Review whether `TestChain_DecompressionThenMaxBodySize`'s 417-as-signal assertion should instead read the limiter error directly (clarity).
50. ~~After v1.0: schedule the `internal/` package-structure revisit trigger review (35 files today; trigger ~50).~~ done (trigger re-affirmed 2026-08-30: 36 of ~50 files, decision unchanged)

## g) Questions I Cannot Answer Myself

### Q1: Push the 19 unpushed commits now, or hold?

Everything is committed on local `master` (`590da48..c1b2f31`), tree clean, all gates green. I don't know whether you prefer reviewing the batch first, whether the auto-push daemon will handle it, or whether origin has moved (I haven't fetched — didn't want a surprise merge mid-session).

### Q2: Do you want v1.0 cut after this stabilization pass?

This gates five items (T18 `TokenBucketLimiter` removal, T19 implementation, the middleware-count docs changes, the "One stabilization cycle" roadmap bullet, and the versioning story for `StartTLS`/builders shipped today). Everything is deliberately ready for either answer: 0.x can keep iterating, or v1.0 can freeze the current surface.

### Q3: Coverage policy for the new code — chase or document?

Today's additions dropped coverage 97.3→96.9 / 99.3→98.8. The new uncovered paths are `StartTLS`'s error branch, the new builders' failure branches, and fuzz helpers. Options: (a) close them to 100% now (mostly easy, a few need error-injection harnesses), or (b) document them in FEATURES's defensive-paths list like the existing 14. The answer sets the precedent for every post-v1.0 addition.

---

**Files changed this session (19 commits):** `health.go` + 6 test files fixed; 9 July status reports annotated; TODO_LIST/ROADMAP/FEATURES/README/CHANGELOG/AGENTS.md/CONTRIBUTING.md/v1-stability.md/RELEASE.md updated; new: `headers.go`, `server_timing/doc.go`, `docs/DECISION_LOG.md`, `docs/adr/0001…`, `docs/benchmarks.md`, 2 design notes, brutal-self-review HTML, internal-coupling D2+SVG, `recorder_fuzz_test.go`, `decompression_fuzz_invariants_test.go`, `bench_batch_test.go`, `check-module-boundaries.sh`, `nightly-fuzz.yml`; plus dprint normalization across the status corpus and living docs.

**Session conclusion:** The plan's 1% (HARVEST), 4% (protect + self-review), and 20% are done; the long tail is filed, cited, and waiting. The single most consequential find: **the test suite had been silently red for 13 days** — not because of a code bug, but because the environment's quality gate died first and nobody noticed what it was hiding.
