# TODO List — httputil

Short- and mid-term improvement tasks. Each item verified against the actual code.

_Updated: 2026-07-31._

---

## High Priority

_All high-priority items completed in the 2026-07-31 session._

- [x] **Close coverage gaps for new middleware** — coverage improved from 91.0% to 97.8%. All three new middleware (CSRF, Server-Timing, KeyedRateLimiter) now have comprehensive tests covering ValidateCSRF, TranslateCSRFHeaders, CSRFTokenHXHeaders, isTrustedProxy, Validate, delegatingWriter delegation, eviction heap, and callback paths. Remaining sub-100% functions are unreachable defensive code (json.Marshal error on map[string]string, crypto/rand panic paths, stale-heap mismatch branches).
- [x] **Add `MiddlewareStack` name constants for new middleware** — `MiddlewareCSRF`, `MiddlewareServerTiming`, `MiddlewareKeyedRateLimit` added to `stack.go`.

## Medium Priority

_All medium-priority items completed in the 2026-07-31 session._

- [x] **Add config field tables for `CSRFConfig` and `KeyedRateLimiterConfig` in README** — both tables added with all fields, types, defaults, and descriptions.
- [x] **Add `Example*` functions for new middleware** — `ExampleCSRFMiddleware`, `ExampleServerTimingMiddleware`, `ExampleKeyedRateLimiterMiddleware` added with `// Output:` directives.
- [x] **Add `v1-stability.md` entries for new types and functions** — all new types classified as Frozen/Additive with "New in v0.8.0" notes. Added CSRF Protection, Server-Timing, and expanded Rate Limiting sections. Updated Middleware* constants count from 9 to 12.
- [x] **Fix `writeClassified` doc comment overclaim** — corrected to "Write-path error-handling choke point" and documents that buffer-drain writes in Close and flushPlainAndStream call `compressWriteError` directly.
- [x] **Write deprecation migration guide** for `TokenBucketLimiter` to `KeyedRateLimiter` — `docs/migrating-to-keyed-rate-limiter.md` created with symbol mapping table, before/after code, behavioral differences table, and monitoring guide.

## Low Priority

_All low-priority items completed in the 2026-07-31 session._

- [x] **Close remaining pre-v0.7.1 coverage gaps** — `Server.Shutdown` (75% → 100%) via context-expiry test with active connections. Remaining gaps (`computeETag` 94.4%, `scanAcceptEncoding` 95.5%, `Compression` 95.5%, `drawRandomBytes` 67%, `refillRandomBuffer` 87.5%, `httpspec.runSpecs` 88.2%, `httpspec.mustRequest` 75%) are error-injection or defensive code paths.
- [x] **Pin GitHub Actions to commit SHAs** — all 5 actions pinned: `actions/checkout`, `actions/setup-go`, `actions/upload-artifact`, `golangci/golangci-lint-action`, `softprops/action-gh-release`.
- [x] **Update v0.7.1 GitHub Release notes** to match the corrected CHANGELOG — release notes updated via `gh release edit` to use Keep a Changelog format matching CHANGELOG.md exactly.
- [x] **Add CHANGELOG comparison-link CI check** — `scripts/check-changelog-links.sh` created and added to CI workflow. Validates every heading has a matching link definition and vice versa.

---

_Long-term vision and raw ideas live in [ROADMAP.md](ROADMAP.md). Completed work is recorded in [CHANGELOG.md](CHANGELOG.md)._
