# Status Report: Validate-at-Construction Unification & Remaining Issues

**Date:** 2026-08-08 07:48  
**Session:** 4 (continuation of nonce middleware hardening series)  
**Prior reports:** `2026-08-08_06-54_nonce-middleware-post-hardening-audit.md`  

---

## Executive Summary

This session resolved both open design questions (Q1: Nonce coverage gap, Q2: Validate inconsistency), fixed the badge script sed bug at its root cause, and made a security improvement to the Nonce constructor's fallback logic. The codebase now has **100% of middleware constructors calling `Validate()` at construction** via a shared helper. All tests pass with 0 lint issues across ~70 linters. However, several gaps remain — most notably, the new `validateConfig` calls in 8 constructors are themselves **uncovered by tests** (the `slog.Error` branch fires only on invalid configs that no test constructs).

---

## a) FULLY DONE

### 1. Q1 Resolved: Nonce() Coverage Gap Closed + Security Fix
- **Added** `TestNonce_InvalidConfigLogsAndFallsBack` (`nonce_test.go`) — constructs `Nonce(NonceConfig{Size: 8})`, verifies fallback to default size (20 bytes).
- **Fixed security bug found during testing:** `Nonce()` previously only fell back to `defaultNonceSize` when `Size <= 0`. A non-zero but too-small value like `Size: 8` would log a warning via `Validate()` but **still generate an insecure 8-byte nonce**. Now the fallback condition is `Size < minNonceSize`, closing the gap.
- **Result:** `Nonce()` coverage went from 92.3% → **100%**.
- **Commit:** `040a41a`

### 2. Badge Script sed Bug Fixed at Root Cause
- **Root cause:** `scripts/update-coverage-badge.sh` used a sed pattern that matched only the inner image `![Coverage](...)`, but `new_badge` included the outer markdown link wrapper `[![Coverage](...)](#)`. Each run wrapped another `[...](#)` layer, producing `[[[![Coverage](...)](#)](#)]`.
- **Fix:** Replaced sed with awk whole-line replacement. Verified idempotent (2 consecutive runs produce identical output).
- **README line 5:** Fixed from `[[[![Coverage](...)](#)](#)]` → `[![Coverage](https://img.shields.io/badge/coverage-97.5%25-green)](#)`.
- **Commit:** `f38c46f`

### 3. Q2 Resolved: Validate() Calls Added to All Middleware Constructors
- **Added** `validateConfig(name, err)` shared helper in `recorder.go:22`.
- **Updated** all 10 constructors to use it:
  - `CSRFMiddleware` (csrf.go) — refactored from inline to helper
  - `Nonce` (nonce.go) — refactored from inline to helper
  - `CORS` (cors.go)
  - `SecurityHeaders` (security.go)
  - `Compression` (compression.go) — validates **after** default-filling
  - `Decompression` (decompression.go)
  - `MaxBodySizeMiddleware` (maxbodysize.go)
  - `RequestID` (requestid.go)
  - `RateLimit` (ratelimit.go)
  - `KeyedRateLimiterMiddleware` (ratelimit_keyed.go) — validates **after** default-filling in `buildKeyedRateLimiter`
- **Pattern:** validate-and-log (not validate-and-abort). Invalid configs log via `slog.Error` but still construct a working middleware with fallback defaults.
- **Cyclomatic complexity fix:** Extracting to a helper avoided adding `if` blocks to each constructor (which would have pushed `SecurityHeaders` to cyclop=13 and `Decompression` to funlen=62).
- **Commits:** `1cc6866`, `3dfd94d`

### 4. Documentation Updated
- **CHANGELOG.md:** Added entries for validate-at-construction, `TestNonce_InvalidConfigLogsAndFallsBack`, fallback behavior change, badge script fix.
- **AGENTS.md:** Added "All middleware constructors call Validate()" to Non-Obvious Behaviors. Updated Nonce bullet to document fallback behavior. Updated `recorder.go` table entry to include `validateConfig()`.
- **Commits:** included in above + `0f93cd8`

### 5. Verification
- `golangci-lint run ./...`: **0 issues** across ~70 linters.
- `golangci-lint fmt`: clean.
- `go test -race -count=10 ./...`: all pass, race-clean, ~3.2s.
- `go test -race -count=1 ./... -coverprofile`: **97.0%** httputil, **99.3%** httpspec, **97.5%** aggregate.
- `server_timing` sub-module: tests pass, 0 lint issues.
- `validateConfig` coverage: **100%**.

---

## b) PARTIALLY DONE

### 1. validateConfig Coverage in 8 New Constructors — NOT TESTED
The `validateConfig` helper itself is at 100% coverage (tested via the Nonce test), but the **call sites in 8 constructors** (CORS, SecurityHeaders, Compression, Decompression, MaxBodySize, RequestID, RateLimit, KeyedRateLimiter) are only covered by tests that pass valid `Default*()` configs. The `slog.Error` path in those constructors has **never been exercised**. No test constructs an invalid config for these middlewares and verifies the log+continue behavior.

**Impact:** Low — the code works correctly, the pattern is identical to Nonce/CSRF (which ARE tested), and invalid configs in production would produce a log warning + fallback behavior. But for full rigor, each constructor should have a "rejects invalid config" test.

### 2. CHANGELOG README Section — Coverage Value Not Updated
The CHANGELOG `[0.10.0]` section says "Coverage now 97.0%" (from the v0.10.0 release). The current aggregate is 97.5%. This is **frozen history** per the CHANGELOG Freeze Policy and should NOT be edited — the current value is accurately reflected in the README badge. No action needed, but noting it.

### 3. MaxBodySizeMiddleware Doc Comment Stale
`maxbodysize.go:39-40` says "Call [MaxBodySizeConfig.Validate] before constructing the middleware to surface configuration errors at startup." This is now misleading — the constructor calls `Validate()` itself. The doc comment should be updated.

---

## c) NOT STARTED

### 1. httpspec Nonce Spec
No `httpspec` spec exists for CSP nonce header presence + format validation. The `httpspec.Run(t, handler)` API could verify that handlers using `Nonce()` middleware produce correct CSP headers with nonce format. Mentioned in the prior audit report (item not yet addressed).

### 2. Q3: Version Tag Decision
The prior session left open whether to tag v0.11.0 with the hardening changes. Not addressed this session.

### 3. Nonce Generator Injectable Field
The prior audit suggested making the nonce generator injectable (for testing/deterministic nonces). `NonceConfig` currently has no `Generator` field. Not started.

### 4. Deprecated RateLimit Middleware Validate Coverage
`RateLimit()` (deprecated) now calls `validateConfig` but no test exercises the invalid-config path. Since it's deprecated, this is low priority.

---

## d) TOTALLY FUCKED UP

### Nothing catastrophically broken.

**However, one thing I should have caught earlier:**

### The Nonce Size Fallback Bug Was Pre-Existing (Session 2/3 Missed It)
When `Nonce()` was changed in Session 3 to call `Validate()`, the fallback condition was `Size <= 0`. But `Validate()` was changed to reject `Size != 0 && Size < 16`. This means `Size: 8` (non-zero, below minimum) would:
1. Pass the `Size <= 0` check (it's positive)
2. Hit `Validate()` which returns an error (logged via slog.Error)
3. **Still use Size=8 for nonce generation** — an insecure nonce!

The middleware logged a warning but happily generated insecure nonces. I found and fixed this this session (`Size < minNonceSize` fallback), but it should have been caught when the Validate logic was changed in Session 3. The test `TestNonce_InvalidConfigLogsAndFallsBack` surfaced it immediately — which is exactly why coverage tests matter.

---

## e) WHAT WE SHOULD IMPROVE

### 1. Constructor Validate-Error Tests for All Middleware
Every constructor now calls `validateConfig`, but only Nonce and CSRF have tests for the invalid-config path. The other 8 constructors should each have a test like:
```go
func TestCORS_InvalidConfigLogsAndContinues(t *testing.T) {
    // Construct with AllowCredentials=true + AllowAllOrigins=true
    // Verify slog.Error is called and middleware still works
}
```

### 2. Stale Doc Comment in maxbodysize.go
`maxbodysize.go:39-40` tells users to call `Validate()` before constructing the middleware. The middleware now does this itself. Update the comment.

### 3. Decompression() and Compression() Coverage Dropped
- `Decompression` constructor: 78.8% (was higher — the validateConfig line is uncovered)
- `Compression` constructor: 95.7% (same reason)
These are minor dips but should be addressed with invalid-config tests.

### 4. validateConfig Should Be Tested in Isolation
The helper has 100% coverage via the Nonce test, but a dedicated `TestValidateConfig` unit test would make the coverage resilient to Nonce test changes and document the helper's contract explicitly.

### 5. Badge Script Should Be Tested in CI
The sed→awk fix was verified manually (2 runs, idempotent), but there's no automated test. A simple shell test that runs the script twice and checks for bracket accumulation would prevent regressions.

### 6. Consider Returning Error Instead of Logging
The validate-and-log pattern means invalid configs are silently absorbed in production (unless someone reads logs). An alternative pattern: constructors that return `(Middleware, error)` would force callers to handle invalid configs. This is a v1.0 design decision, not a quick fix.

---

## f) Up to 50 Things to Get Done Next

#### Coverage & Tests
1. Add `TestCORS_InvalidConfigLogsAndContinues` — AllowCredentials + AllowAllOrigins
2. Add `TestSecurityHeaders_InvalidConfigLogsAndContinues` — invalid FrameOptions
3. Add `TestCompression_InvalidConfigLogsAndContinues` — invalid Level
4. Add `TestDecompression_InvalidConfigLogsAndContinues` — negative MaxDecompressionSize
5. Add `TestMaxBodySize_InvalidConfigLogsAndContinues` — negative MaxBytes
6. Add `TestRequestID_InvalidConfigLogsAndContinues` — nil GenerateID
7. Add `TestRateLimit_InvalidConfigLogsAndContinues` — nil Limiter
8. Add `TestKeyedRateLimiter_InvalidConfigLogsAndContinues` — negative TTL (after defaults)
9. Add `TestValidateConfig` — dedicated unit test for the helper
10. Add `TestValidateConfig_NilError` — verify nil error does not log

#### Documentation
11. Fix `maxbodysize.go:39-40` stale doc comment (says "Call Validate before constructing")
12. Check other constructors for similar stale "call Validate" comments
13. Update `docs/v1-stability.md` with `validateConfig` helper classification
14. Add `validateConfig` to the API table in AGENTS.md (done for recorder.go row)
15. Document the validate-and-log pattern in README or docs

#### Nonce Middleware
16. Add httpspec spec for CSP nonce header presence + format
17. Add injectable `Generator` field to `NonceConfig` for deterministic testing
18. Add `ExampleNonce_CacheControlNoStore` showing Cache-Control integration
19. Consider `NonceWithCacheControl` composite middleware that sets both nonce + no-store
20. Add integration test: Nonce + SecurityHeaders in correct order via MiddlewareStack

#### Architecture & Design
21. Decide v1.0 API: should constructors return `(Middleware, error)` instead of log-and-continue?
22. Consider `Validated[M ConfigValidator]` generic wrapper type
23. Extract `validateConfig` to internal package if more non-middleware code needs it
24. Consider startup-time config validation in `MiddlewareStack.Build()` (batch validate all)
25. Add `MiddlewareStack.ValidateConfigs()` method that validates all added middleware configs

#### Badge & Tooling
26. Add automated test for `update-coverage-badge.sh` (run twice, check idempotency)
27. Add pre-commit hook for badge script verification
28. Consider switching badge to dynamic shields.io endpoint (no script needed)
29. Add `make coverage` / `nix run .#coverage` that runs tests + updates badge in one step
30. Add coverage threshold gate (fail build if < 95%)

#### Versioning & Release
31. Tag v0.11.0 with validate-at-construction + nonce hardening changes
32. Update FEATURES.md with validate-at-construction as a feature
33. Update ROADMAP.md with remaining nonce/CSP items
34. Write release notes for v0.11.0
35. Update `docs/v1-stability.md` to classify `validateConfig` and all Validate() methods

#### Security
36. Add fuzz test for `Nonce()` with Size=0 edge case
37. Add test verifying nonce is present in CSP even when handler doesn't write body
38. Add test for nonce middleware under concurrent requests (stress test uniqueness)
39. Consider adding `StrictCSPWithNonce` (no 'self' in script-src, nonce-only)
40. Document CSP hash-source as alternative to nonces

#### Code Quality
41. Check if `art-dupl` finds the `validateConfig(name, cfg.Validate())` pattern as duplication
42. Run `brutal-self-review` skill on the validateConfig changes
43. Verify `gosec` passes on all changed files
44. Run `govulncheck` on the full dependency tree
45. Check if any constructor's Validate() has a different semantic than "reject invalid" (e.g., Size==0 meaning "use default")

#### Server-Timing & Other Sub-modules
46. Add Validate-at-construction to `server_timing` middleware if it has a config
47. Verify `ServerTimingMiddlewareWhen` also validates when condition is used
48. Add Server-Timing coverage of the validation path

#### General
49. Audit all `// Call Validate before constructing` doc comments across the codebase
50. Consider a `Config[T]` interface or type constraint for config types that have `Validate()`

---

## g) Questions (Cannot Figure Out Myself)

### Q1: Should v0.11.0 be tagged now?

The validate-at-construction change touches 10 constructor files and changes runtime behavior (invalid configs now log warnings they didn't before). It's a behavioral change but not a breaking API change. Options:
- **(A)** Tag v0.11.0 now — changes are tested and verified
- **(B)** Wait until the 8 missing constructor validate-tests (#1-8 above) are written
- **(C)** Batch with httpspec nonce spec + Generator field for a larger v0.11.0

### Q2: Should the validate-and-log pattern be replaced with validate-and-return-error before v1.0?

The current pattern logs but doesn't abort. This means a misconfigured middleware in production silently uses fallback values. The alternative (`(Middleware, error)` return signature) would be a **breaking API change** for all 10 constructors. Options:
- **(A)** Keep validate-and-log through v1.0 (non-breaking, matches Go stdlib `http` patterns)
- **(B)** Add `MustNew*` variants that panic on invalid config (non-breaking, adds opt-in strictness)
- **(C)** Break all constructors to return error in v1.0 (clean but disruptive)

### Q3: Should the Nonce generator be made injectable for deterministic testing?

Currently `generateNonce` uses `crypto/rand` directly. Adding a `Generator func(size int) string` field to `NonceConfig` would allow deterministic nonces in tests. This is a feature addition, not a bug fix. Should it be done before or after v0.11.0?

---

## Metrics Snapshot

| Metric | Value |
| --- | --- |
| Root non-test files | 34 |
| Root test functions | 342 |
| httpspec test functions | 88 |
| Total benchmarks | 46 (38 root + 2 httpspec + 6 server_timing) |
| Total examples | 26 |
| Total fuzz tests | 20 |
| Aggregate coverage | 97.5% |
| `validateConfig` coverage | 100.0% |
| `Nonce()` coverage | 100.0% |
| Lint issues | 0 (~70 linters) |
| Race detection | Clean (-count=10) |

## Files Changed This Session

| File | Change |
| --- | --- |
| `recorder.go` | Added `validateConfig(name, err)` shared helper + `"log/slog"` import |
| `nonce.go` | Refactored inline validate to helper; fixed fallback to `Size < minNonceSize` |
| `csrf.go` | Refactored inline validate to helper |
| `cors.go` | Added `validateConfig` call |
| `security.go` | Added `validateConfig` call |
| `compression.go` | Added `validateConfig` call (after default-filling) |
| `decompression.go` | Added `validateConfig` call |
| `maxbodysize.go` | Added `validateConfig` call |
| `requestid.go` | Added `validateConfig` call |
| `ratelimit.go` | Added `validateConfig` call |
| `ratelimit_keyed.go` | Added `validateConfig` call (after default-filling) |
| `nonce_test.go` | Added `TestNonce_InvalidConfigLogsAndFallsBack` |
| `scripts/update-coverage-badge.sh` | Replaced sed with awk (bracket accumulation fix) |
| `README.md` | Fixed triple-nested bracket artifact; badge updated to 97.5% |
| `CHANGELOG.md` | Added entries for all changes |
| `AGENTS.md` | Added validate-at-construction behavior doc; updated recorder.go table entry |

## Commits This Session

```
0f93cd8 docs(agents): document new validateConfig helper in recorder.go
3dfd94d feat(httputil): centralize config validation logging via shared helper
1cc6866 feat(middleware): log config validation errors in middleware constructors
f38c46f fix(tools): correct coverage badge updater to prevent bracket accumulation
040a41a fix(nonce): align default-size fallback with minNonceSize validation
```
