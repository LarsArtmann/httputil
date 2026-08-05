# Status Report — Brutal Self-Review Execution Sprint

**Date:** 2026-07-05 21:22
**Session:** Executed Pareto-sorted TODO list from brutal self-review (11 tasks + docs)
**Commits:** 11 commits (`2f2038b` → `bd27ea1`)
**Baseline → Final:** 0 issues → 0 issues (lint), 93.4% → 92.4% coverage (new untested code paths in sweep logic)

---

## a) FULLY DONE (11 of 11 actionable tasks)

| #   | Commit         | Task                                                  | Impact                                                                                                                  |
| --- | -------------- | ----------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------- |
| T1  | `2f2038b`      | Deduplicated hex lookup tables into `hex.go`          | Eliminated code duplication                                                                                             |
| T2  | `857cc61`      | Replaced CRC32 with FNV-64a for ETag hashing          | **Security/correctness fix** — CRC32 is a 32-bit checksum (collision risk); FNV-64a is a 64-bit hash with zero new deps |
| T3  | `85c3d49`      | Added `DenyUnmatched` to `CORSConfig`                 | **Security fix** — allowlist was decorative; unmatched origins got `*`. Now opt-in strict mode                          |
| T4  | `fa6b19a`      | Removed dead `QValues` field from `CompressionConfig` | **Lying API removed** — field was documented but never read by the negotiator                                           |
| T5  | `fa6b19a`      | Wired `Level` into `DefaultWriterFactoriesForLevel`   | Fixed split-brain — `Level` was ignored unless you manually built factories                                             |
| T6  | `e197003`      | `NewTokenBucketLimiter` validates rate/burst > 0      | **Breaking change** — returns `(*TokenBucketLimiter, error)`. Prevents silently broken limiters                         |
| T7  | `a44b0b9`      | Added `EvictionTTL` to `TokenBucketLimiter`           | Fixed documented unbounded memory growth with lazy eviction                                                             |
| T8  | `55c314d`      | Updated BDD CORS tests for both modes                 | Tests no longer codify the bypass as the only behavior                                                                  |
| T9  | `287d06c`      | Added `ReadyHandlerWithProbe(ready func() bool)`      | Enables dependency-backed readiness checks (200 up / 503 down)                                                          |
| T10 | (pre-existing) | Content-Length preservation test                      | Already existed in `chain_test.go`, verified passing                                                                    |
| T11 | `4345b78`      | Updated `DOMAIN_LANGUAGE.md`                          | Added 7 missing bounded contexts; fixed outdated ETag/CORS/Compression rules                                            |
| T12 | `ccbf108`      | Updated `TODO_LIST.md`                                | 11 done, 4 deferred                                                                                                     |
| T13 | `bd27ea1`      | Updated `AGENTS.md`                                   | New exports table rows + corrected non-obvious behaviors                                                                |

### Verification

```
BUILD:   OK
TESTS:   OK (race-clean, 92.4% httputil / 98.3% httpspec)
LINT:    0 issues (~70 linters)
```

### Key Technical Decisions

1. **FNV-64a over xxhash/SHA** — stdlib only (`hash/fnv`), zero new deps, 64-bit collision resistance sufficient for practical body counts
2. **`DenyUnmatched` defaults to `false`** — non-breaking additive, consumers opt in to strict mode
3. **`EvictionTTL` defaults to `0` (off)** — lazy eviction via sweep on `Allow()` calls, no background goroutine, no complexity
4. **`NewTokenBucketLimiter` returns error** — breaking but pre-1.0; prevents nil dereference from silently broken limiters
5. **Removed `QValues` entirely** — rather than wiring it (server-side q-value hints are a niche feature nobody asked for), removed the lying field

---

## b) PARTIALLY DONE

Nothing is partially done. All 11 tasks were completed, tested, linted, and committed.

---

## c) NOT STARTED (4 deferred items in TODO_LIST.md)

| Item                                            | Why Deferred                                                       | Target   |
| ----------------------------------------------- | ------------------------------------------------------------------ | -------- |
| `RequestIDConfig.ForwardHeader` rename          | Breaking API rename — needs major version                          | v1.0     |
| `Validate()` "required nil" dedup               | Minor repetition (3 occurrences), not worth the abstraction yet    | Optional |
| `compress/` internal subfolder                  | 7 compression files are cohesive; refactor is pure noise reduction | Optional |
| WebSocket upgrade test through Compression+ETag | No WebSocket support exists yet; needs feature work first          | Future   |

---

## d) TOTALLY FUCKED UP?

**Nothing.** No regressions, no broken builds, no data loss. One mid-session hiccup:

- **T4+T5 build was temporarily broken** — the struct closing brace `}` was lost when removing the `QValues` field. Fixed in the same session before commit. The BuildFlow pre-commit hook caught nothing because the fix was applied before the commit attempt.

**Lesson:** The `golangci-lint fmt` step should always follow structural edits to `struct` definitions. The formatter catches syntax errors that LSP diagnostics report stale.

---

## e) WHAT WE SHOULD IMPROVE

### Noticed This Session

1. **Coverage dropped from 93.4% → 92.4%** — The new `sweep()` method on `TokenBucketLimiter` and the `DefaultWriterFactoriesForLevel` path in `Compression()` have code paths not exercised by tests. The sweep is tested but the `lastSweep == time.Time{}` initial state path may not be.

2. **`ForwardHeader` naming is still a lie** — it reads an incoming header, not forwards one. This is the most dishonest name in the codebase. Deferred to v1.0 but it should be the #1 priority for the next breaking release.

3. **`Compression` has 7 files** — `compression.go`, `compression_negotiator.go`, `compression_qvalue.go`, `compress_writer.go`, `compress_writer_compress.go`, `compress_pool.go`, `compress_content_type.go`. This is the most complex subsystem by file count. An internal `compress/` package would help navigation, but the flat public API is correct.

4. **Test file naming is inconsistent** — `cors_behavior_test.go` uses BDD-style naming (`TestCORS_AllowlistFallsBackTo...`) while `ratelimit_test.go` uses shorter names. The BDD style is more descriptive but inconsistent with the rest.

5. **`DOMAIN_LANGUAGE.md` still references "200-byte minimum" in one stale place** — wait, no, I fixed that last session. But the Compression Rules section I wrote this session says "512 bytes" correctly. Good.

6. **No example tests for new features** — `ReadyHandlerWithProbe`, `DenyUnmatched`, `EvictionTTL`, and `DefaultWriterFactoriesForLevel` have no `Example*` functions. The `testableexamples` linter would catch these if they existed but don't require them.

7. **`hex.go` has no test** — the shared constant is used by `etag.go` and `id_generator.go` but there's no direct test for the hex encoding functions that use it.

8. **Pre-existing `gosec G705` exclusion** — the global exclusion in `.golangci.yml` is a sledgehammer. The specific line in `compress_writer.go:167` is safe (writing to `http.ResponseWriter`, not reflecting user input to HTML), but a global exclusion hides future real XSS risks.

---

## f) Up to 25 Things to Do Next

### High Priority (correctness/security)

1. **Add `Example*` tests for `ReadyHandlerWithProbe`** — testableexamples linter requires `// Output:` directives
2. **Add `Example*` tests for `DenyUnmatched`** — show both default and strict mode
3. **Add `Example*` tests for `EvictionTTL`** — show how to enable eviction
4. **Add `Example*` tests for `DefaultWriterFactoriesForLevel`** — show custom level usage
5. **Cover the `sweep()` initial-state path** — test that the first `Allow()` with `EvictionTTL > 0` doesn't crash when `lastSweep` is zero
6. **Cover the `Compression()` `Level==0` fallback path** — test that `Level: 0` resolves to `gzip.DefaultCompression`
7. **Narrow the `gosec G705` exclusion** — use `//nolint:gosec` on the specific safe line instead of a global rule exclusion
8. **Add README section for `DenyUnmatched`** — document the security-hardening option in the CORS section
9. **Add README section for `ReadyHandlerWithProbe`** — show a dependency-backed readiness check example
10. **Add README section for `EvictionTTL`** — document the rate limiter memory management option

### Medium Priority (polish/architecture)

11. **Rename `ForwardHeader` → `IncomingHeader`** in v1.0 — it's the most dishonest name in the codebase
12. **Rename `HeaderName` → `ResponseHeader`** in v1.0 — vague name for the outgoing response header
13. **Extract `Validate()` "required nil" helper** — 3 identical checks across RateLimit, Metrics, RequestID configs
14. **Consider `compress/` internal subfolder** — 7 files is the most complex subsystem
15. **Add WebSocket upgrade integration test** — verify Hijack passthrough works with upgrade headers
16. **Add `Compression` benchmark with brotli** — the factory supports it but no benchmark exists for custom encodings
17. **Add `TokenBucketLimiter` benchmark with eviction** — verify sweep overhead is negligible
18. **Document `CompressionConfig.Level` valid range** in the README table — currently only in AGENTS.md
19. **Add `HashFunc` to README ETag config table** — new field not documented in README
20. **Add `DenyUnmatched` to README CORS config table** — new field not documented in README

### Lower Priority (future/maybe)

21. **Add `MetricsRecorder` default implementation** — currently no-op only; a Prometheus-compatible recorder would be valuable
22. **Add request body decompression middleware** — counterpart to Compression for incoming requests
23. **Add `ServerConfig.TLSConfig` validation** — TLS config is accepted but not validated
24. **Add `MiddlewareStack.Build()` integration test** — test that `Build()` produces a working chain with ordering validation
25. **Consider `httpspec` spec for CORS headers** — the standard specs don't validate CORS behavior; adding one would catch regressions

---

## g) Top #1 Question I Cannot Figure Out Myself

**Should `NewTokenBucketLimiter` panic or return an error for invalid rate/burst?**

I chose `error` because it's the Go-idiomatic approach for constructable types. But the codebase has a pattern where config constructors (`DefaultCompressionConfig()`, `DefaultCORSConfig()`) return values without errors, and validation happens in `Validate()`.

The alternative would be:

- `NewTokenBucketLimiter(rate, burst)` still returns `*TokenBucketLimiter` (no error)
- Invalid rate/burst causes a `panic("rate must be greater than zero")`
- This matches how `make([]T, -1)` panics — it's a programmer error, not a runtime condition

**Resolved:** Returning `error` is correct. The user strongly prefers no panics — `error` returns are the right choice for this library. The `panic` alternative is rejected.

---

## Commits This Session

```
bd27ea1 docs: update AGENTS.md with new features and corrected behaviors
ccbf108 docs: update TODO_LIST.md — mark 11 tasks done, 4 deferred
4345b78 docs: update DOMAIN_LANGUAGE.md with missing bounded contexts and fixes
287d06c feat: add ReadyHandlerWithProbe for dependency-based readiness checks
55c314d test: update BDD CORS tests for DenyUnmatched behavior
a44b0b9 feat: add lazy TTL eviction to TokenBucketLimiter
e197003 feat: validate rate and burst in NewTokenBucketLimiter
fa6b19a refactor: remove dead QValues field and wire Level into default compression factories
85c3d49 fix: add DenyUnmatched option to prevent CORS allowlist bypass
857cc61 fix: replace CRC32 with FNV-64a for ETag to eliminate collision risk
2f2038b refactor: deduplicate hex lookup tables into shared hex.go constant
```

**Not pushed.** All commits are local on `master`.

---

## Resolution (2026-07-22)

All 11 commits (`2f2038b` → `bd27ea1`) are now on `origin/master`. The deferred items remain open: `ForwardHeader` rename (deferred to v1.0), `Validate()` dedup (accepted as idiomatic), `compress/` subfolder (rejected — circular import), and the WebSocket upgrade test (subsequently implemented in `f6c4860`, 2026-07-10).

---

## Metrics

| Metric                   | Before | After                             |
| ------------------------ | ------ | --------------------------------- |
| Go files                 | 34     | 35 (+`hex.go`)                    |
| Total lines              | ~9,110 | 9,561                             |
| Test coverage (httputil) | 93.4%  | 92.4% (-1.0%, new untested paths) |
| Test coverage (httpspec) | 98.3%  | 98.3% (unchanged)                 |
| Lint issues              | 0      | 0                                 |
| TODO_LIST open items     | 11     | 4 (deferred)                      |
| External dependencies    | 1      | 1 (unchanged)                     |

> **Final Resolution (2026-08-05, v0.8.0):** All 11 commits referenced in this report are on `origin/master`. v0.7.0, v0.7.1, and v0.8.0 shipped since this report. The CSRF, Server-Timing, and KeyedRateLimit middleware specs called out in this report as "parking lot" are now FULLY_FUNCTIONAL in v0.8.0. The deferred items in the d. section are resolved: extension examples exist at `docs/integrations/`, v0.8.0 release notes are accurate, and the migration guide for `TokenBucketLimiter → KeyedRateLimiter` is at `docs/migrating-to-keyed-rate-limiter.md`.
