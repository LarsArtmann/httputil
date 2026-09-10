# Status Report — CSRF Trust-Model Hardening + TODO Execution Session

**Date:** 2026-09-10 04:03 CEST
**Session window:** ~02:55–04:03 CEST
**Baseline:** `4bb191c` (toolchain bump) → tip `46a8524` (17 auto-daemon commits; work spans this session and one concurrent session)
**Gates at report time:** `go test -race` 3× ok (root + httpspec + server_timing) · `golangci-lint run` 0 issues (both modules) · both erraudit real gates exit 0 · `go vet` ok · `nix flake check` all checks passed
**Report format note:** HTML is the status-report skill's canonical output; the user explicitly requested `.md`, so this file is Markdown by instruction.

---

## a) FULLY DONE

All items verified against running code, not just claims. Each was test-covered and gate-checked in the same step it was implemented.

| # | Item | Evidence |
|---|------|----------|
| a1 | **CSRF `Sec-Fetch-Site` trust model verified at source** — fetched and read justinas/nosurf **v1.2.0** `handler.go`/`utils.go` (the pinned version): `ensureSameOrigin` returns nil on ANY literal `Sec-Fetch-Site: same-origin` before consulting Origin/Referer. No assumptions encoded. | This report; doc comments in `csrf.go` |
| a2 | **Attestation-conflict defense implemented** — new sentinel `ErrCSRFAttestationConflict` (`csrf.origin_attestation_conflict`, Rejection) + `contradictedAttestationOrigin` check wired into `CSRFMiddleware` and `ValidateCSRF`; rejects unsafe-method requests whose forged attestation contradicts an Origin nosurf would reject (cross-origin, not in TrustedOrigins, or unparseable). `Origin: null` exempt (mirrors nosurf). Browsers cannot produce the combination (forbidden header name). | 6 new middleware tests + 1 `ValidateCSRF` test; run 5× `-race` clean |
| a3 | **Trust boundary documented** — `CSRFMiddleware` doc (new "Origin attestation trust model" section), `SetPlaintextHTTPOrigin` doc ("fills a blank only"), README CSRF section (full trust-model block), AGENTS.md Non-Obvious Behaviors bullet, error-classification rows (AGENTS + README) | grep-able; links from TODO_LIST |
| a4 | **`CompressionConfig.Level = 0` semantics decided and aligned** — one meaning: "unset → `gzip.DefaultCompression`". Field doc, `Validate` doc, constructor comment, and `DefaultWriterFactoriesForLevel` doc (raw stdlib level where `0` = `NoCompression`) all agree. Level range-check applies only when `WriterFactories` is empty; constructor checks the level at its use site. | `TestCompression_ZeroLevelMeansDefaultCompression` (round-trip + size probe — the AGENTS "0 means X" execution-probe rule), `TestCompressionConfig_Validate_ZeroLevelIsValid` |
| a5 | **Rate-limiter admission contract confirmed + documented** (v1.0 pre-gate from the design note) — "tokens consumed at admission; no refund on abort" now on `KeyedRateLimiterMiddleware` doc, `Check` doc (pre-existing), and a new **Admission Contract** section in `docs/migrating-to-keyed-rate-limiter.md`; ROADMAP already records the `Wait(ctx)` post-v1.0 path | Design note's "what must happen before v1.0" checklist fully closed |
| a6 | **7 legacy table-driven tests converted** to standalone `TestX_Case` funcs (decision made once: split, matching AGENTS.md): `TestClientIP` → 4, `TestParseUintQuery` → 12, `TestVaryContainsToken` → 8, `TestValidateNonNegativeInt` → 7, `TestHasVersionLeakDetectsVersionPattern` → 5, `TestFormatMillis` → 5, `TestServerTiming_NameSanitization` → 5 | Full suite green; each new func has `t.Parallel()` |
| a7 | **`varyContainsToken` → `varyContainsOrigin`** — `unparam` finding (token always `"Origin"`; verified pre-existing at HEAD via throwaway git worktree before touching it). Renamed, constant param dropped; caller + 8 tests updated | `golangci-lint run` 0 issues |
| a8 | **`exhaustruct` → `exhaustruct_v5` migration** in both `.golangci.yml` files on golangci-lint 2.13.2 — probed the embedded JSON schema to discover the v5 keys (`enforce-patterns`/`ignore-patterns`, `exclude` rejected; struct tags gone → `//nolint:exhaustruct_v5` renames); `config verify` + full run 0 issues, no new findings | Both modules lint-clean |
| a9 | **T13 line-by-line test review finished** (`csrf_test.go` 810 lines, `nonce_test.go` 713, `security_test.go` 402, `requestid_test.go` 174, `id_generator_test.go` 137 — all read end-to-end) — found and fixed 2 real test bugs (see a10/a11), found 1 formal race (b2), rest clean | This report; fixes below |
| a10 | **`TestNonce_UniquePerRequest` assertion bug fixed** — it only compared every nonce against `nonces[0]`; a repeat at indexes 3/5 would have passed. Now pairwise via a set | Name now matches claim |
| a11 | **`TestSecurityHeaders_CustomHeadersEmptyMap` stale names fixed** — asserted `X-Custom` instead of the configured `X-Custom-Header` | Fixed |
| a12 | **Chain-level exact-fill regression** — `TestCompression_ExactMinSizeWrite_IsNotDuplicated` runs the 512-byte exact-fill case through the full `Compression()` middleware (was unit-only) | Passes; would catch the 2026-08-30 duplication bug at chain level |
| a13 | **`TestChain_RecoveryErrAbortHandler_ThroughStack`** — `http.ErrAbortHandler` sentinel re-panics through `Chain` (Nonce + Recovery), not just bare `Recovery` | Passes |
| a14 | **`compressWriter.Hijack` dropped-bytes semantics documented** — doc comment states buffered bytes below `minSize` are dropped on hijack | Code comment |
| a15 | **Config-validation hardening batch** — 5 validators tightened, all validate-and-log (zero runtime behavior change): `cors.methods_empty`, `decompression.encoding_unrecognized` + `decompression.encoding_duplicate` (case/whitespace-normalized), `compression.incompressible_prefix_invalid` (empty/no-slash/untrimmed entry would block ALL compression), `csrf.max_age_negative`. Each: code + package sentinel + template in `errorTemplates` + `allHTTputilErrorCodes` entry + tests (9 new), incl. acceptance tests proving defaults/variants still pass | Full suite green; template-completeness test passes |
| a16 | **httpspec no-injection-header-reflection spec** — `SpecNameNoInjectionHeaderReflection` (19 standard specs): 11 injection-prone request headers (forwarding metadata, Fetch Metadata, method/URL overrides); fails if any header NAME is reflected in the response | 2 new tests + expected-names map updated |
| a17 | **Docs synced** — CHANGELOG [Unreleased] (Added ×5, Changed ×2, Fixed ×2 entries), TODO_LIST (9 items `[x]` with DONE annotations, 1 new item, header date), README (trust model + 5 error-table rows + spec count), AGENTS.md (exhaustruct_v5 constraint, csrf exports row, 19 specs, 5 classification rows, Level bullet, attestation bullet, randBuf tradeoff) | grep-able |
| a18 | **Concurrent-session conflicts handled without damage** — a second session landed `compose.go`/`compose_test.go`/`stack.go` `Middleware()` and the `X-CSRF-Token` → `X-Csrf-Token` canonicalization mid-flight; my edits were re-based on their state, nothing of theirs reverted, my CSRF work adapted (doc bullets, const value) | `git log` 4bb191c..HEAD; working tree clean |

## b) PARTIALLY DONE

| # | Item | Done | Missing |
|---|------|------|---------|
| b1 | **v1.0 release decision** | All pre-v1.0 gates from the rate-limiter design note closed; admission contract documented everywhere the note required; deprecation plan (T18) referenced | The cut itself: no tag, no `TokenBucketLimiter`/`RateLimit` removal, no release workflow. Correctly left as a user decision (breaking change + push) |
| b2 | **ID-generator `randBuf` race** | Found, root-caused, documented (AGENTS tradeoff bullet + new TODO_LIST item) | NOT fixed. A pointer-swap "quick fix" would reintroduce cross-generation slot duplication; needs a generation-stamped ring + stress test as a focused change |
| b3 | **My own CHANGELOG accuracy** | Written and committed via daemon | First draft said "Six new tests" — actually **seven**. Self-caught during this report's fact-check and fixed (2026-09-10 ~04:05). Lesson: I published a numeric claim without recounting |
| b4 | **README spec-count freshness** | AGENTS.md updated 18→19 | README line 140 still said "18 behavioral specs" — missed during the CSRF pass; caught during this report's fact-check and fixed |
| b5 | **Fuzz coverage of the new CSRF path** | Existing `FuzzCSRFMiddleware_OriginHeaders` seeds the contradictory combo — but only as a **GET**, and the new rejection applies to unsafe methods only | No fuzz target exercises the POST rejection path; no new corpus seeds added |
| b6 | **Coverage numbers** | All new code is test-covered | FEATURES.md's 96.9% / 98.8% claims are now stale (new branches in csrf.go, compression.go, decompression.go, cors.go) — not re-measured per the coverage methodology |
| b7 | **Test-convention consistency** | All 7 named legacy tables converted | Two `t.Run` subtest clusters in `security_test.go` (`ContentTypeOptionsPrecedence`, `SecurityHeaderSkip` ×3 each) and several elsewhere remain — arguably property-style groups, but the "decide once, apply consistently" decision didn't rule on them |
| b8 | **erraudit full pass** | Both real gates (legacy_as, stdlib_constructor) exit 0 | The full `--type-aware` advisory baseline (~30 known-correct `errors.Is` advisories) not re-run to confirm no drift |

## c) NOT STARTED

Untouched from TODO_LIST (verified against the file at session end — these checkboxes are still open):

1. Extract response compression into `go-compression` (Pareto plan exists; go-datastar trigger)
2. CI release workflow (tag → build → GitHub Release, both modules)
3. go-error-family upstream: conditional-request classification guidance (needs `verify-before-filing`)
4. `architecture-review` re-run (stale: predates ETag extraction, keyed limiter, compose API, attestation defense)
5. Refresh `docs/benchmarks.md` rows for the 2026-08-30 bench changes (3s×5 protocol)
6. `CSRFConfig.Validate` side-effect cleanup (pure Validate + parse step; post-v1.0 candidate)
7. KeyedRateLimiter property test (heap/map consistency under churn) + MaxKeys-pressure benchmark
8. CORS fuzz invariant for exact-origin allowlists
9. `WrapConflict`/`WrapOrchestration` symmetry decision
10. Skip pool `Get` for non-resettable factories (wasted alloc/request)
11. `Server.Addr()` resolved-port variant API decision
12. Negotiator wire-format fuzz target + gzip multistream doc note
13. Test-helper hygiene trio (dissolve `bench_batch_test.go`, consolidate `waitForTLS`/`reserveFreePort`, Ed25519 test cert)
14. flake.nix benchmark-protocol app (one-command 3s×5 baseline)
15. Slim AGENTS.md below the 30 KB docs-health budget (~53 KB before today; I **added** ~2 KB — see e4)
16. Test-coverage gaps: MaxBodySize bench/fuzz + `ExampleMetrics`, `ExampleRateLimit`, `ExampleHealthHandler`, `ExampleServer`, `ExampleMiddlewareStack`
17. Nonce design decisions: `NonceConfig.Generator` override + public `GenerateNonce` (implement or formally decline) + nonce composition-test cluster
18. ID-generator refill-path benchmark
19. dprint availability in devShell (markdown formatter verification skipped 3+ sessions)
20. Nightly fuzz crash issue-template step
21. govulncheck for `server_timing` in CI
22. Commit-lint CI step (reject non-conventional prefixes)
23. Go-based coverage threshold check (replace awk in ci.yml) + pre-release checklist script
24. `go.work` `go 1.26.5` vs CI vs local 1.26.7 pinning
25. `TestChain_DecompressionThenMaxBodySize` 417-as-signal assertion → read limiter error directly
26. Post-v1.0: `KeyExtractor` returning `""` semantics doc check
27. Integration-docs content refresh (samber/do, HTMX-ideas, Redis, Prometheus)
28. Schedule next `full-code-review`
29. Re-mark deprecated `TokenBucketLimiter`/`RateLimit` for removal in the v1.0 CHANGELOG (part of the cut)

## d) TOTALLY FUCKED UP

Nothing shipped broken — every change landed behind green gates, and the working tree is clean. But honesty requires listing what went wrong **in the process**, including two factual errors I published and had to retract:

1. **I published a wrong numeric claim in the CHANGELOG** — wrote "Six new tests" when the CSRF work added **seven**. Caught it during this report's own fact-check (grep + recount), fixed within minutes. Severity: low (doc-only), but it is exactly the "unverified claim encoded into docs" failure mode this repo's skills warn about, and I did it while writing the docs that claim accuracy.
2. **I updated AGENTS.md's spec count (18→19) but missed the same claim in README** — stale "18 behavioral specs" sat in README line 140 until this report's fact-check caught it. Same root cause: I enumerated doc locations from memory instead of grepping for the number everywhere.
3. **Pipeline masking bit me twice, in a repo whose AGENTS.md literally warns about it** — `grep … | head -1 || sed …` never ran the `sed` because `head` exits 0. Cost: two failed builds (`bytes`, `errors` imports) and two wasted round trips. My own memory notes say "verify the raw exit codes, not the filtered tail."
4. **First CSRF test failure was my own test's fault, and I briefly suspected the middleware** — the test's custom `ErrorHandler` captured the error without writing a status, so the recorder stayed 200. The middleware log line (visible in `-v`) proved the rejection fired. Debugging order was backwards: I should have read the log line first.
5. **My first `multiedit` to `csrf.go` was rejected** (file modified mid-edit by the concurrent session/daemon) — unavoidable given the environment, but my recovery required a full re-read; a locked-file or retry-first strategy would have saved a round trip.
6. **Sloppy compound shell commands** — one `sed -i … /dev/null` noise error and one `cat >> file << EOF` used instead of proper edit tooling for appending tests (fast but bypasses read-before-write discipline; both landed correctly, verified by gates).
7. **Process, not product:** 17 generic auto-daemon commit messages ("chore: auto-commit N changed file(s) (heuristic)") bury this session's coherent units of work in history — the known commit-lint gap, made worse by a session this large.

## e) WHAT WE SHOULD IMPROVE

1. **Fact-check numeric claims by grep, not memory** — any count written into docs (tests added, spec counts, code totals) should be produced by a command, not recalled. Both d1/d2 trace to this.
2. **Fuzz what you harden** — the new POST-only rejection path has unit tests but zero fuzz exposure; security-critical request-classification logic is exactly where fuzz targets pay off (the exact-fill bug was found by fuzz, not by tests).
3. **Re-measure coverage the same session you add branches** — the coverage methodology treats a stale number in FEATURES.md as a docs bug.
4. **AGENTS.md is growing in the wrong direction** — the file is over budget (~53 KB, target <30 KB) and today's additions made it bigger. Next docs pass should move the file-export tables to `docs/` and keep only pointers.
5. **Stop using `||`-after-pipes for fallback commands** — pipelines mask exit codes; use explicit `if ! grep -q …; then` or just the edit tools.
6. **Run `golangci-lint fmt` after each file, not at session end** — one golines violation survived until the final pass.
7. **Consider a behavior-change migration note for `ErrCSRFAttestationConflict`** — any consumer who (deliberately or not) relied on forged attestations gets a new 403 after upgrading; that deserves a line in `docs/migrating-*` or RELEASE.md before v1.0, not just the CHANGELOG entry.
8. **Benchmark the new hot-path header lookups** — `CSRFMiddleware` now reads up to 3 extra headers per unsafe request; a csrf bench row update would prove the cost is noise (it should be, but the repo's culture is to measure).
9. **Review the concurrent session's `compose.go` API under the same lens** — `Compose`, `MiddlewareStack.Middleware()`, `MiddlewareFunc.Then` landed mid-flight with tests; I deliberately did not review them, and v1.0 API-freeze means someone should.
10. **Two-session concurrency protocol** — when two agents share a working tree, agree on file ownership up front; today it worked (no lost writes) but cost round trips.

## f) UP TO 50 THINGS TO GET DONE NEXT

Sorted by impact; first ~10 are the real queue, the rest are the committed backlog + new discoveries (ROADMAP fuel — do not treat 50 as a sprint list).

1. **Write the behavior-change migration note for the new CSRF 403** (`ErrCSRFAttestationConflict`) in `docs/migrating-*`/RELEASE.md before any release
2. **Decide + cut v1.0** (go-release skill: CHANGELOG cut, tag, release, pkg.go.dev check)
3. **Remove deprecated `TokenBucketLimiter`/`RateLimit`** (plan T18) if v1.0 is to ship without them — decide order with #2
4. **Re-measure coverage** (both modules, `-race -coverprofile`) and refresh FEATURES.md's 96.9%/98.8% claims
5. **Fuzz the CSRF attestation path**: extend `FuzzCSRFMiddleware_OriginHeaders` with a method parameter (POST included) + corpus seeds for the contradictory combos
6. **Re-run the full erraudit `--type-aware` advisory pass** and confirm the ~30-advisory baseline hasn't drifted
7. **Review `compose.go`/`MiddlewareFunc` API** (concurrent session's addition) before v1.0 API freeze — naming, docs, edge cases (empty bundle, nil handler)
8. **Convert the remaining `t.Run` subtest clusters** (`security_test.go` ×2 groups; sweep for others) or formally amend the convention to allow behavior-group subtests
9. **Fix `go.work`/CI Go version pinning** (1.26.5 vs 1.26.x vs 1.26.7)
10. **Refresh `docs/benchmarks.md`** per the 3s×5 protocol + add a CSRF middleware bench row (covers e8)
11. Extract response compression into `go-compression` (Pareto plan ready)
12. CI release workflow (tag → build → GitHub Release, `go vet` both modules)
13. go-error-family upstream: conditional-request classification proposal (`verify-before-filing` first)
14. `architecture-review` re-run (post-ETag/keyed-limiter/compose/attestation)
15. ID-generator: generation-stamped ring rework for the `randBuf` race + `-race -count=50` stress test
16. `CSRFConfig.Validate` side-effect cleanup (pure Validate + separate parse step)
17. KeyedRateLimiter property test (heap/map consistency above `MaxKeys`) + churn benchmark
18. CORS fuzz invariant for exact-origin allowlists
19. `WrapConflict`/`WrapOrchestration` symmetry decision
20. Skip pool `Get` for non-resettable factories
21. `Server.Addr()` resolved-port variant
22. Negotiator wire-format fuzz target + gzip multistream doc note
23. Test-helper hygiene trio (dissolve `bench_batch_test.go`; consolidate `waitForTLS`/`reserveFreePort`; Ed25519 test cert)
24. flake.nix benchmark-protocol app
25. MaxBodySize benchmark + fuzz target (only middleware without either)
26. Five missing examples: `ExampleMetrics`, `ExampleRateLimit`, `ExampleHealthHandler`, `ExampleServer`, `ExampleMiddlewareStack`
27. Nonce decisions: `Generator` override + public `GenerateNonce` (implement or decline in DECISION_LOG) + nonce×Compression/CORS/ServerTiming composition tests
28. ID-generator refill-path benchmark (pairs with #15)
29. dprint into the flake devShell (end the 3-session markdown-verification skips)
30. Nightly fuzz workflow: crash → issue-template step
31. govulncheck for `server_timing` in CI
32. Commit-lint CI step (conventional prefixes; directly attacks d7)
33. Go-based coverage threshold check replacing the awk in ci.yml
34. Pre-release checklist script automating RELEASE.md gates
35. `TestChain_DecompressionThenMaxBodySize`: read the limiter error instead of treating 417 as the signal
36. `KeyExtractor` returning `""` — exempt vs shared-bucket doc disambiguation
37. Integration-docs content refresh (samber/do, HTMX-ideas, Redis, Prometheus)
38. Schedule the next `full-code-review` (before v1.0 if substantial code lands)
39. Slim AGENTS.md below 30 KB (move file-export tables to `docs/`; NOTE today's session grew it — pay it down before it drifts again)
40. Consider Referer-based attestation contradiction (documented why only Origin today; needs browser-behavior evidence first)
41. Decide whether client-sent `Sec-Fetch-Site` should be normalized/stripped on trusted-proxy plain HTTP (policy, currently documented-only)
42. Verify `ErrCSRFAttestationConflict`/`csrf.max_age_negative` surface correctly through `RegisterErrorClassifications` docs/examples (template test covers it; consumer-facing example does not)
43. Add the httpspec injection-reflection spec to the README's httpspec example block
44. Update `FEATURES.md` error inventory with the 5 new codes (docs-health VERIFY)
45. Add `docs/status/2026-09-10_03-41_composability-session-status.md` items to the harvest queue if its section (f) wasn't harvested
46. Consider `http.NoBody`/nil-body fuzz coverage for the CSRF middleware (nosurf reads `PostFormValue` — multipart path is fuzz-touched only indirectly)
47. `Compose` empty-list identity: add a doc-test/example (`ExampleCompose`) so the identity contract is executable documentation
48. Decide deprecation timeline for `ETag()` adapter (marked deprecated; removal target unpinned)
49. Housekeeping: `/tmp/httputil-head` worktree from this session was removed — audit for other stale worktrees (`git worktree list`)
50. Post-v1.0: revisit `SetIsTLSFunc` interplay with `X-Forwarded-Proto` TLS-terminating proxies (nosurf's own doc suggests header-based detection; ours is `r.TLS != nil` — document the constraint for proxy deployments)

## g) QUESTIONS I CANNOT FIGURE OUT MYSELF

1. **v1.0 behavior-change policy:** the attestation-conflict rejection turns previously-passing (forged) requests into 403s. Should v1.0 ship it always-on (current), or behind a `CSRFConfig` opt-out with secure default? This is a risk-appetite call about breaking consumers, not a technical one.
2. **v1.0 sequencing:** cut v1.0 first (deprecated `TokenBucketLimiter`/`RateLimit` still present, removed in v1.1), or remove them first and let the removal BE the v1.0? The migration doc supports either.
3. **The concurrent session's `compose.go` / `MiddlewareFunc.Then` / `MiddlewareStack.Middleware()` API** landed mid-flight with 294 lines of tests. Was that yours/intended for v1.0's frozen surface — and do you want it reviewed under the same full-code-review lens before the freeze?

---

### Self-review addendum (brutal-self-review questions, answered honestly)

- **Forgot:** README's "18 specs" claim (b4); coverage re-measurement (b6); fuzz coverage for the new path (b5); the migration note for the new 403 (f1).
- **Lied?** No — but one imprecise claim ("Six" vs seven tests) was published and corrected; caught by my own fact-check, not by you.
- **Ghost systems?** None created; every new symbol is wired (middleware, validator, tests, docs). The attestation check is reachable in production (`CSRFMiddleware`/`ValidateCSRF`), not decorative.
- **Split brains?** Two near-misses fixed in-session: `Level` semantics now single-meaning everywhere, and `varyContainsOrigin` removed a helper whose signature implied genericity it didn't have.
- **Removed something useful?** No removals this session; only renames with call-site updates.
- **Scope creep?** The hardening batch (a15) and the unparam rename (a7) were adjacent-scope captures beyond the strict TODO items — both were already listed in TODO_LIST, so captured, not invented.
