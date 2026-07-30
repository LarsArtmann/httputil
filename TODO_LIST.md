# TODO List — httputil

Short- and mid-term improvement tasks. Each item verified against the actual code.

_Updated: 2026-07-30._

---

## High Priority

- [ ] **Close coverage gaps for new middleware** — three middleware features were added post-v0.7.1 without full coverage, dropping overall from 98.7% to 91.0%. `csrf.go` has `ValidateCSRF` (0%), `TranslateCSRFHeaders` (0%), `CSRFTokenHXHeaders` (0%), `isTrustedProxy` (20%), `Validate` (47%). `server_timing.go` and `ratelimit_keyed.go` also have sub-100% functions. _(Measured 2026-07-30 via `go test -race -coverprofile`.)_
- [ ] **Update AGENTS.md for new files and dependencies** — `csrf.go`, `server_timing.go`, `ratelimit_keyed.go`, and their test files are missing from the architecture file table. The allowed-dependencies claim ("`go-error-family` and `golang.org/x/time`") omits `justinas/nosurf` (now in depguard). New `Middleware*` constants for CSRF, Server-Timing, and KeyedRateLimit are not listed in `stack.go`. _(AGENTS.md is loaded as project context for every AI session — staleness here misleads every session.)_
- [ ] **Add `go-error-family` v0.9.0 → v0.10.0 entry to CHANGELOG `[Unreleased]`** — the dependency was bumped but the changelog only records the v0.7.0 → v0.9.0 bump from `[0.6.1]`. _(CHANGELOG.md `[Unreleased]`.)_

## Medium Priority

- [ ] **Add config field tables for `CSRFConfig` and `KeyedRateLimiterConfig` in README** — the README has field tables for all v0.7.x config types but not the new middleware configs. _(README.md config tables section.)_
- [ ] **Add `Example*` functions for new middleware** — `testableexamples` linter will flag missing examples for `CSRFMiddleware`, `ServerTimingMiddleware`, `KeyedRateLimiterMiddleware` once they are exported in example contexts. Each needs a `// Output:` directive. _(Pattern: `ExampleCORS`, `ExampleETag`, etc.)_
- [ ] **Add `v1-stability.md` entries for new types and functions** — `CSRFConfig`, `CSRFMiddleware`, `ServerTiming`, `ServerTimingMiddleware`, `KeyedRateLimiter`, `KeyedRateLimiterConfig`, `KeyedRateLimiterMiddleware`, and their helpers are not classified as Frozen/Additive/Evolving. _(docs/v1-stability.md.)_
- [ ] **Add `MiddlewareStack` name constants for new middleware** — `MiddlewareCSRF`, `MiddlewareServerTiming`, `MiddlewareKeyedRateLimit` do not exist in `stack.go`. The existing pattern requires named constants for duplicate prevention and ordering validation. _(stack.go.)_
- [ ] **Fix `writeClassified` doc comment overclaim** — `compress_writer.go` says "single error-handling choke point for compressWriter output" but `flushPlainAndStream` buffer-drain bypasses it. Either route the drain through the helper or correct the comment to "Write-path choke point". _(compress_writer.go, from `docs/status/2026-07-26_17-49_*` item f.1.)_
- [ ] **Write deprecation migration guide** for `TokenBucketLimiter` → `KeyedRateLimiter` — the old `RateLimit()`, `RateLimiter` interface, `RateLimitConfig`, and `TokenBucketLimiter` are deprecated but no migration documentation exists. _(A doc in `docs/` or a section in README.)_

## Low Priority

- [ ] **Close remaining pre-v0.7.1 coverage gaps** — `computeETag` (94.4%), `scanAcceptEncoding` (95.5%), `Compression` (95.5%), `Server.Shutdown` (75%), `drawRandomBytes`/`refillRandomBuffer` (67–88%), `httpspec.runSpecs`/`mustRequest` (75–88%). All are error-injection or internal paths. _(FEATURES.md "PARTIALLY DONE".)_
- [ ] **Pin GitHub Actions to commit SHAs** — BuildFlow flagged 9 tag-pinned actions. _(`.github/workflows/`.)_
- [ ] **Update v0.7.1 GitHub Release notes** to match the corrected CHANGELOG (CORS test rename moved from "Removed" to "Changed"). _(github.com/LarsArtmann/httputil/releases/tag/v0.7.1.)_
- [ ] **Add CHANGELOG comparison-link CI check** — automated format enforcement so links stay in sync. _(From v0.7.0 self-review, recurring item.)_

---

_Long-term vision and raw ideas live in [ROADMAP.md](ROADMAP.md). Completed work is recorded in [CHANGELOG.md](CHANGELOG.md)._
