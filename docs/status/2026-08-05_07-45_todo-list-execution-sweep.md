# Status: 2026-08-05 TODO-List Execution Sweep

_07:45 CEST — Session closed; all 10 TODO items marked complete; one honest self-critique and an open follow-up list below._

## TL;DR

Walked the entire 12-item v0.8.0 TODO list (5 medium-priority + 7 low-priority items) and shipped 10 of the 12. The 2 "skipped" items (CSRFMiddleware, ServerTimingMiddleware, KeyedRateLimiterMiddleware `Example*` functions) turned out to already exist in `example_test.go:173-236` — confirmed by `go test -v -run Example` and `golangci-lint run -E testableexamples`. No new work was needed; the docs were simply out of date.

All 9 deliberate code changes were:

- Lint-clean (`golangci-lint run` → 0 issues)
- ~~Test-clean (`go test -count=1 ./...` → all pass)~~ **[STALE — `go test -count=1` does NOT detect races. `RateLimitSpecs` shipped with a data race caught later by `-race` (fixed at `e291a19`).]**
- ~~Race-clean for the touched packages (`go test -race -count=3 -run TestStack_` → 5 runs, no flakes)~~ **[STALE — only `TestStack_*` was race-checked. `RateLimitSpecs` had a race that passed `count=1` clean but failed 60% of `-race` runs (fixed at `e291a19`).]**

**What I forgot:** the linter hit me on `paralleltest` once because I removed `t.Parallel()` from subtests that shared an atomic — I correctly diagnosed the race, but the fix to make each subtest own its own `called` atomic came on the second attempt after the auto-git-commit daemon reverted my edit. I should have written the file defensively from the start instead of trying the race-prone "shared atomic" pattern.

**What I fucked up:** I left a duplicate-bracket artifact (`[[[!Coverage...`](#)](#)`) in `README.md`after running the badge-update script twice. The daemon captured the bad state. I corrected it manually. Also, the first run of the coverage script's color logic flagged 97.4% as "red" because`bc`wasn't available; I fixed it with`awk` but the daemon had already captured the red state.

**What I could still improve:** see the follow-up list below — there are 30+ items the audit surfaced that I deliberately did NOT do (out of scope), plus 3 questions I cannot answer from the codebase.

---

## What I Did

### a) Fully Done

| #   | Item                           | Files                                                                                                                                  | Notes                                                                                                                                                                                                                                                                                                                                   |
| --- | ------------------------------ | -------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| ~~1~~   | ~~CSRF fuzz tests~~ done at `e31f144` | ~~`csrf_fuzz_test.go` (new, 257 lines)~~ | ~~6 fuzz functions: `FuzzCSRFConfig_TrustedProxiesCIDR`, `FuzzCSRFConfig_TrustedOrigins`, `FuzzCSRFIsTrustedProxy`, `FuzzCSRFMiddleware_TokenValidation`, `FuzzCSRFRemoteHostAndIP`, `FuzzCSRFMiddleware_OriginHeaders`. All found issues in 1-2s; one required adding a `isValidHTTPToken` filter (httptest.NewRequest panics on `" "`).~~ |
| ~~2~~   | ~~`httpspec` CORS spec~~ done at `538a575` | ~~`httpspec/cors_ratelimit_specs.go` (new, 308 lines), `httpspec/cors_ratelimit_specs_test.go` (new, 264 lines)~~ | ~~4 CORS specs: `SpecNameCORSAllowOrigin`, `SpecNameCORSAllowCredentials`, `SpecNameCORSVaryOrigin`, `SpecNameCORSWildcardNoCredentials`. All return `Pass()` for handlers that don't set CORS (opt-in).~~ |
| ~~3~~   | ~~`httpspec` rate-limit spec~~ done at `538a575` | ~~same file~~ | ~~3 specs: `SpecNameRateLimitRetryAfter`, `SpecNameRateLimitHeaderOnReject`, `SpecNameRateLimitHintHeadersOnAllow`. Probes up to 100 requests to detect 429 then asserts on the response.~~ |
| ~~4~~   | ~~Full-stack integration test~~ done at `6bae773`, `e062ef7` | ~~`stack_integration_test.go` (new, 280 lines)~~ | ~~`TestStack_FullMiddlewareComposition` chains all 12 `Middleware*` constants + ClientIP + ServerTiming, runs 5 subtests (GET headers, POST CSRF rejection, OPTIONS preflight, panic recovery, rate-limit headers). Plus `TestStack_RecoveryMustBeOuterMost` and `TestStack_DuplicateMiddlewareRejected`.~~ |
| ~~5~~   | ~~`b.Loop()` migration~~ done at `ae78e9a` | ~~`server_timing_bench_test.go`~~ | ~~6 benchmark loops converted. Cleared 6 gopls warnings.~~ |
| ~~6~~   | ~~`BenchmarkKeyedRateLimiter`~~ done at `314e37a` | ~~`ratelimit_keyed_bench_test.go` (new, 192 lines)~~ | ~~6 variants: Allow, Reject, HighCardinality, EmptyKey, EvictionOverhead, ClientIPExtractor.~~ |
| ~~7~~   | ~~`BenchmarkCSRFMiddleware`~~ done at `6bae773` | ~~`csrf_bench_test.go` (new, 156 lines)~~ | ~~6 variants: GET, POSTWithToken, POSTRejection, PostForm, ConfigValidate, TokenFromContext.~~ |
| ~~8~~   | ~~Dynamic coverage badge~~ done at `eb1ac6a` | ~~`scripts/update-coverage-badge.sh` (new, 56 lines), `.github/workflows/ci.yml` (3-line addition)~~ | ~~Computes coverage via `go tool cover -func`, picks color by threshold (≥90 green, ≥70 yellow, <70 red), rewrites the `[![Coverage](...)]` line in-place via `sed`.~~ |
| ~~9~~   | ~~`Validate()` audit — additions~~ done at `eb1ac6a` | ~~`ratelimit_keyed.go` (new `Validate` method, ~22 lines), `server.go` (2 new checks), `security.go` (real validation + 2 new constants)~~ | ~~Added `KeyedRateLimiterConfig.Validate()` (was the only config in the audit list missing it). Hardened `ServerConfig.Validate()` to reject empty `Addr` and `ReadHeaderTimeout > ReadTimeout`. Replaced `SecurityHeadersConfig.Validate()` no-op with real `FrameOptions` value validation per RFC 7034 §2.1.~~ |
| ~~10~~  | ~~`Validate()` audit — tests~~ done at `eb1ac6a` | ~~`server_test.go` (3 new tests), `security_test.go` (2 new tests), `ratelimit_keyed_test.go` (5 new tests)~~ | ~~Each new validation rule has positive + negative test coverage.~~ |

### b) Partially Done

- **README coverage badge**: now dynamic (97.4% as of the run), but the badge color was "red" for one commit because `bc` was unavailable and I had a fallback bug. Fixed to use `awk` for threshold logic. The final state shows 97.4% green. The script is correct; the script's first run was broken.
- **TODO_LIST.md**: marked all 10 items as `[x]`. Updated the date-stamp. But I did NOT update `FEATURES.md` to reflect the new test/benchmark/spec inventory.

### c) Not Started (because already done in prior sessions)

- `ExampleCSRFMiddleware` — `example_test.go:173`
- `ExampleServerTimingMiddleware` — `example_test.go:193`
- `ExampleKeyedRateLimiterMiddleware` — `example_test.go:213`

All three had `// Output:` directives and pass `golangci-lint run -E testableexamples`. The TODO list was simply out of date.

---

## What I Forgot / What I Fucked Up

1. **Duplicate bracket artifact in `README.md`**: the auto-git-commit daemon captured a `[[[![Coverage...`](#)](#)` line after I ran the badge-update script twice. I manually fixed it but did not catch the daemon's first commit in time.
2. **Red badge for valid 97.4%**: the first run of `update-coverage-badge.sh` evaluated color with `bc -l`, which isn't installed. The fallback to "red" was wrong for a 97.4% number. Fixed by rewriting color logic with `awk`, but the daemon captured the red state once.
3. ~~**Race in shared `called` atomic**: my first `TestStack_FullMiddlewareComposition` shared one `atomic.Bool` and one `handler` across 5 parallel subtests. `go test -race` correctly identified it. The fix — make each subtest own its `called` + `handler` — was obvious in hindsight; I should have written it that way from the start.~~ done at `e062ef7`
4. **`httptest.NewRequest` panics on invalid methods**: my first `FuzzCSRFMiddleware_TokenValidation` fuzzer hit the panic on input `" "` (space). I added a `isValidHTTPToken` filter, but this is brittle — the next fuzzer author may not know. A `t.Skip` with a clear comment would be safer than trusting future maintainers to read the helper.
5. **`canonicalheader` lint asymmetry**: the linter triggers on `Header.Get(literal)` and on literal constants in some positions, but NOT on `Header.Set(literal)` or on `Header.Get(constant)`. I went through 3 iterations before I understood this. The documentation gap is a real footgun.
6. ~~**No `FEATURES.md` update**: the inventory table is now stale — it doesn't list the new fuzz tests, the CORS/rate-limit specs, the integration test, or the new benchmarks. I should have updated it.~~ done at `2e15780`
7. ~~**No `CHANGELOG.md` entry**: the auto-commit daemon captured my work into 4 separate commits, but I never wrote a `[Unreleased]` CHANGELOG entry summarizing the sweep. The commits themselves are well-described, but a top-level entry would help.~~ done at `2e15780`
8. **`go.mod` not bumped**: this was a code+test sweep, no API changes to public types except `KeyedRateLimiterConfig.Validate()` (additive) and the FrameOptions constant types. Should still be v0.8.0 patch level, not new minor. Skipped intentionally.
9. ~~**Did not run `go test -race` on the whole repo, only the touched tests**: I verified race-clean on `TestStack_` 5x and saw a pre-existing race in `httpspec/RunSerialAllSpecsPassForGoodHandler` — but I did not confirm whether MY changes (the new httpspec files) introduced any. I should have isolated this more rigorously.~~ done at `e291a19`
10. **Coverage script's `--benchtime` and `--count=1` flags are not in the CI step**: the CI does `go test -bench=. -benchmem -count=1 -run=^$ ./...` but I never verified the benchmark suite still passes in CI mode. Local runs were fine.

---

## What I Could Still Improve (Follow-Up List)

1. ~~**Update `FEATURES.md` middleware test/benchmark/fuzz table** to include the new `csrf_fuzz_test.go`, `httpspec/cors_ratelimit_specs_test.go`, `stack_integration_test.go`, `ratelimit_keyed_bench_test.go`, `csrf_bench_test.go` rows.~~ done at `2e15780` (docs-health rebuild updated all middleware table rows)
2. ~~**Add `[Unreleased]` CHANGELOG entry** summarizing the v0.8.0 sweep (audit, fuzz tests, specs, integration test, dynamic badge).~~ done at `2e15780` (docs-health rebuild rewrote [Unreleased] with structured catalog)
3. ~~**Investigate the pre-existing race in `httpspec/RunSerialAllSpecsPassForGoodHandler`** — reproducible with `go test -race -count=3 ./httpspec/...`. Likely the `cfg.indexPath = defaultIndexPath` mutation across parallel specs.~~ resolved — passes `go test -race -count=3` as of `e291a19` (2026-08-05)
4. ~~**Make the `httptest.NewRequest` panic in fuzz tests** never panic — wrap the body of every fuzz test in `defer func() { recover(); t.Skip() }()` so future authors don't need to know.~~ done (addressed differently — the isValidHTTPToken filter keeps fuzz inputs valid instead of recover-and-skip (see a.1))
5. **Document the `canonicalheader` lint asymmetry** in `AGENTS.md` Hard Constraints so future authors don't hit the same 3-iteration debug cycle I did.
6. ~~**Add a `BenchmarkCompressionNegotiator` and `BenchmarkETagWriter`** — these were skipped from the original suite per the `compress_writer_test.go` boundary; the audit shows neither has a benchmark yet.~~ done at `647efdc`
7. ~~**`Compression` Level=0 is currently accepted** (`gzip.DefaultCompression`). Document this in `AGENTS.md` so callers don't pass `0` and silently get default behavior they didn't ask for.~~ done (documented — AGENTS.md Non-Obvious Behaviors covers the Level=0 default)
8. ~~**`MaxBodySize` has no `Validate()` method** — a negative `maxBytes` is silently accepted today. The audit list didn't include it, but it's an obvious gap.~~ done at `98bff8c`
9. ~~**`Health` config (`Server.ShutdownTimeout`)** has no `Validate()` — same as #8.~~ done at `98bff8c`
10. **The `KeyedRateLimiterMiddleware` `OnRejected` callback contract** isn't documented in the type comment — what does the callback return? What if the user writes to `w` from the callback and then also writes from `RejectionHandler`? Race potential.
11. ~~**The `httptest` Request panics in the rate-limit spec** when 100 requests all have `RemoteAddr=""` — wait, do they? I should verify. Multiple `httptest.NewRequest` calls in a loop with the same body.~~ done at `e291a19` (race in RateLimitSpecs fixed)
12. ~~**The coverage script does not fail CI** when the badge update fails (e.g., `coverage.out` missing). It should `exit 1` on parse failure but I made it `exit 0` on "no badge found". The latter is intentional but the former isn't.~~ done (the badge script now exits 1 on failure paths)
13. **`http.NoBody` vs `nil` body in fuzz tests** — `httptest.NewRequest` may behave differently. Should standardize.
14. ~~**The `Update coverage badge` CI step** runs after the `Test with coverage` step, but on cache-hit, `coverage.out` may not be regenerated. The script handles missing-file but the workflow order should `force` a fresh coverage run.~~ done (CI generates a fresh coverage.out in the same workflow before the badge step)
15. **Add `BenchmarkCSRFMiddleware_PlainHTTPNosurf`** — currently I benchmark loopback IP. Real-world CSRF performance with the plaintext-HTTP bypass path is undocumented.
16. ~~**The `FrameOptions` validation rejects lowercase `deny` and `sameorigin`** — strictly correct per RFC 7034, but many tools send lowercase. Either accept both or document the strictness.~~ done (strictness surfaced at runtime — the classified Rejection error names the accepted values (DENY, SAMEORIGIN, skip, empty))
17. ~~**The `errKeyedLimitZero` and `errKeyedWindowZero` sentinels** are in `ratelimit_keyed.go:24-27` but the existing pattern in other files uses `errors.New` inline in the `var (...)` block. Style is consistent, but I should verify with the existing convention.~~ done (superseded — the typed error system rebuilt the sentinels on the Code model)
18. ~~**The `httputil` package is a flat package of 33+ non-test files per AGENTS.md** — the new `csrf_buzz_test.go`, `csrf_bench_test.go`, `ratelimit_keyed_bench_test.go`, and `stack_integration_test.go` push the test file count up. The decision (2026-08-05) was "postpone until v1.0 or >50 non-test files." Currently 37 non-test files; we're moving toward that threshold faster than expected.~~ done (decision and threshold documented in AGENTS.md (flat package, post-v1.0))
19. ~~**The auto-git-commit daemon was helpful but also hostile** — it captured a buggy state (red badge, duplicate brackets) and I had to manually correct. Consider adding a pre-commit hook in `.git/hooks/pre-commit` that runs `golangci-lint run` and rejects on failure.~~ done (pre-commit hook active in the Nix devShell (dprint); AGENTS.md documents the no-verify escape hatch)
20. **The pre-existing race in `TestWithIndexPathChangesTestedPath` (and friends) in `httpspec/`** was not fixed. I noted it but did not act. Should I? My AGENTS.md says "don't fix unrelated bugs" — but the race detector flags it on every full-package run, and a CI run with `-race` would fail intermittently. This is a real CI flake.
21. **The `httptest` test helper `newTestRequest` in `testutil_test.go`** — I never read it. Should I verify it actually creates requests that pass the noctx linter (which is excluded for `_test.go` per `.golangci.yml`)?
22. **The CORS spec's `Vary: Origin` check** uses `varyContainsToken` which is case-insensitive — good. But it doesn't handle the rare case where `Vary: *` is sent. Spec says that's a no-cache directive and overrides individual Vary tokens.
23. ~~**The `KeyedRateLimiterConfig.Validate()` doesn't validate `KeyExtractor`** — `nil` is allowed (defaults to RemoteAddr). But what if a user passes a `KeyExtractor` that always returns `""`? That's a valid pattern for "exempt all" but accidentally enabling it could disable rate limiting entirely. Document the warning.~~ done at `98bff8c`
24. ~~**The CSRF fuzz test for `FuzzCSRFRemoteHostAndIP`** never actually exercises an IPv6 address with a port — the `remoteHostAndIP` function may have edge cases with bracketed IPv6 like `[2001:db8::1]:8080`. The seed corpus has `[::1]:8080` but fuzz may not generate bracketed cases.~~ done (bracketed IPv6 seeds present in the fuzz corpus (csrf_fuzz_test.go))
25. **The integration test asserts `assertStatus(t, rec, http.StatusForbidden)` for CSRF rejection** but doesn't assert the `Content-Type` header. Some browsers and CDN caching behaviors vary based on the rejection Content-Type.
26. ~~**The `httputil.SecurityHeaders` middleware** sets `X-Frame-Options` to the configured value verbatim — but `ALLOW-FROM` is deprecated (RFC 7034). I rejected it in Validate, good, but the documentation doesn't say so.~~ done (the rejection error documents that only DENY/SAMEORIGIN/skip/empty are accepted (ALLOW-FROM rejected per RFC 7034))
27. ~~**`httputil.Compression` accepts a negative `MinSize`** by virtue of `< 0` (rejects it) but the error message just says "got -1" without telling the caller "use 0 to disable the size check".~~ done (the error template Fix text says to set MinSize to zero or a positive byte count)
28. **The `httputil.CORS` `Wildcard` (`*`) origin matching** is not testable through `httpspec` because the spec asserts on `Access-Control-Allow-Origin` value, not on which origin was requested. A spec like "matches the request's Origin when dynamic" would be more thorough.
29. ~~**The `httputil.ETag` middleware** uses FNV-64a by default. The audit could include a benchmark showing the throughput difference between FNV-64a, SHA-256, and xxHash.~~ **Won't implement — moved — ETag benchmarks are go-etag responsibility since the extraction.**
30. ~~**The `httputil.HealthHandler` and `httputil.LiveHandler` and `httputil.ReadyHandler`** have no benchmarks. They're tiny but a benchmark would establish a baseline.~~ done at `3ba8449`
31. ~~**The `httputil.Metrics` middleware** is missing a benchmark — `Metrics` wraps every request and calls `Recorder.Record`. Throughput matters.~~ done at `71d6f49`, `3ba8449`
32. **The `httputil.Timeout` middleware** uses `http.TimeoutHandler` under the hood. Does it correctly propagate `context.DeadlineExceeded`? Audit didn't cover this.
33. ~~**The `httputil.Recovery` middleware** uses `slog.Default()` if logger is nil. The audit didn't cover this fallback.~~ done (covered by the 08-08 validate-at-construction work; the slog fallback is documented in AGENTS.md)
34. ~~**The `httputil.Logging` middleware** also uses `slog.Default()` if logger is nil. Same as #33.~~ done (covered by the 08-08 validate-at-construction work; the slog fallback is documented in AGENTS.md)
35. **The `httputil.MaxBodySize` middleware** wraps `http.MaxBytesReader` but doesn't update `r.ContentLength` after wrapping. The audit didn't verify this.
36. **The `httputil.ClientIPMiddleware`** doesn't strip X-Forwarded-For if the request comes from a non-trusted proxy. The behavior is documented but should be tested fuzz-style.
37. ~~**The `httputil.RateLimit` (deprecated)** `KeyFunc` field has no validation — a nil `KeyFunc` defaults to `RemoteAddr`, but if a user-supplied function panics, the middleware will propagate the panic to the response. Audit didn't cover.~~ **Won't implement — deprecated component superseded by KeyedRateLimiter — hardening there instead.**
38. ~~**The `httputil.TokenBucketLimiter` (deprecated) `Allow` method** has a race: if `EvictionTTL` is changed concurrently with `Allow`, the sweep behavior is undefined. The audit didn't cover.~~ **Won't implement — deprecated component superseded by KeyedRateLimiter — hardening there instead.**
39. ~~**The `httputil.compression_negotiator.go` `buildNegotiator` function** has no benchmarks. The negotiation logic runs on every request.~~ done at `647efdc`
40. ~~**The `httputil.compress_writer.go` `compressWriter` state machine** has no fuzz tests. The 4 state transitions (plain, compress, closed, hijacked) are all hand-written.~~ done at `890b7eb`
41. **The `httputil.wrapper.go` `responseWrapper`** is used by both compress and etag writers — no direct test of the wrapper itself. Indirect coverage only.
42. **The `httputil.recorder.go` `ResponseRecorder`** has 1 benchmark (`BenchmarkResponseRecorder`) but no fuzz tests. It's a security boundary (it captures status before WriteHeader).
43. **The `httputil.id_generator.go` `generateTimeOrderedID`** uses `crypto/rand` with a process-wide buffer of 256 entries. Under high load, the buffer drains and triggers a refill syscall. No benchmark for the refill path.
44. ~~**The `httputil.server_timing.go` `delegatingWriter`** is mentioned in CHANGELOG as supporting Hijacker/Flusher/Pusher but the Pusher support was removed in a prior commit (per AGENTS.md). The documentation is stale.~~ done (sub-module docs now claim only Flusher and Hijacker support (stale Pusher mention gone))
45. **The `httputil.csrf.go` `TranslateCSRFHeaders`** mutates `r.Header` in place to set the default header. If a user reads the original header before passing the request to nosurf, they see the un-translated value. Documented but a footgun.
46. **The `httpspec.SpecNameNoDuplicateHeaders`** only checks for duplicate VALUES in the same header, not duplicate KEYS. `Vary: A` + `vary: B` (different case) would be treated as duplicates by `http.Header` but the spec might not catch it.
47. **The `httpspec.RunSerial`** doesn't share the spec-execution state across specs. If two specs both need to inspect the same response (e.g., the 4 CORS specs), they each make their own request. This is a design choice but could be optimized.
48. ~~**The `coverage.out` file is generated in the repo root by the script** — should it be in a `build/` or `tmp/` directory to avoid accidentally committing it?~~ done (mitigated — coverage.out is git-ignored)
49. ~~**The `scripts/update-coverage-badge.sh` script is not idempotent across runs of `go test`** — the `coverage.out` path includes test-execution time, and the `total:` line in `go tool cover -func` output can change based on test ordering. The percentage is stable but the per-function output is not. The script only uses the total, so this is fine.~~ Won't fix — the original item states "this is fine."
50. ~~**The `httputil` package's test surface is now significantly larger** — the audit added ~10 new test files (5 fuzz + 5 bench + 3 validate-tests + 1 integration + 1 spec). Total test files in the repo: 33. I should have a count, but I don't. Worth verifying.~~ done (counted 2026-08-29 — 58 test files across root, httpspec, and server_timing)

---

## Three Questions I Cannot Answer From the Code

1. ~~**Should I have also added `KeyedRateLimiterConfig.Validate()` to the `KeyedRateLimiterMiddleware()` constructor itself** (so the error is surfaced at middleware build time, not at config-build time)? Other configs do the validation only at construction (e.g., `CSRFConfig.Validate()` is called by `CSRFMiddleware()`, not by users). I followed the explicit `Validate()` pattern from `CSRFConfig` but the auto-construction pattern from `ServerConfig` would have been more consistent. Which do you want?~~ done (answered — the 08-08 validate-at-construction unification made every constructor call Validate via the shared helper)

2. **The auto-git-commit daemon split my work into 9 commits** (4 unprompted by the daemon + 5 by my pre-commit + 1 final TODO_LIST). Should I have squashed them into a single "v0.8.0 TODO sweep" commit, or do you prefer the granular history the daemon produced? AGENTS.md doesn't specify.

3. ~~**The TODO_LIST.md update format** — I marked items as `[x]` and added a single line of context. But the project has rich status documents at `docs/status/`. Should each completed TODO item have its own dedicated status report, or is a single sweep report (this one) sufficient? The pattern in prior commits suggests one report per major change, but the 12 items here are all small.~~ done (answered by practice — one sweep report per session remains the pattern in docs/status/)

---

## Resolution (2026-08-05 11:00 annotation pass; upgraded to per-item markers 2026-08-29)

Every numbered item is resolved inline; unmarked items are still open by convention. The header banner was removed — its content was already inline (the stale "Test-clean"/"Race-clean" corrections carry `e291a19`; the coverage figure was corrected in FEATURES.md).

Open as of 2026-08-29: f5 (canonicalheader Get-vs-Set asymmetry docs), f10 (OnRejected callback write-race contract), f13 (http.NoBody vs nil in fuzz), f15 (BenchmarkCSRFMiddleware_PlainHTTPNosurf), f20 (httpspec indexPath race — `cfg.indexPath` mutation still present), f21 (newTestRequest noctx audit), f22 (Vary: * spec handling), f25 (CSRF rejection Content-Type assert), f28 (CORS wildcard-origin spec), f32 (Timeout DeadlineExceeded propagation test), f35 (MaxBodySize ContentLength update), f36 (ClientIP fuzz-style tests), f41 (responseWrapper direct test), f42 (ResponseRecorder fuzz), f43 (ID-generator refill benchmark), f46 (duplicate-header-KEY spec check), f47 (RunSerial state sharing design), g2 (daemon commit granularity preference). Forgotten-item #5 duplicates f5; #4's recover-and-skip idea was superseded by the input filter (see f4). Items #1/#2/#8 and e)-style observations are session facts, intentionally unmarked.
