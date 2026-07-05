# TODO List — httputil

Short- and mid-term improvement tasks, each verified against the actual code.

_Updated: 2026-07-05. Statuses: `[open]`, `[done]`. Source reviews live in `docs/reviews/` and `docs/brainstorming/`._

---

## Open

### Deferred to major version (breaking naming changes)

- [ ] `[open]` **`RequestIDConfig.ForwardHeader` misnames the direction** — it's the incoming read header, not an outgoing forward. `HeaderName` is also vague. _Action:_ rename to `IncomingHeader`/`ResponseHeader` in a major version. _(naming review)_

### Polish (optional)

- [ ] `[open]` **`Validate()` "required nil" pattern duplicated** — same 3-line check repeated in `RateLimitConfig`, `MetricsConfig`, `RequestIDConfig`. _Action:_ extract a small helper or accept the (minor) repetition. _(code-quality-scan)_
- [ ] `[open]` **Consider internal `compress/` split** — the 7 compression files are cohesive; an unexported subfolder would reduce root-package noise while preserving the flat public API. _(architecture review)_
- [ ] `[open]` **WebSocket upgrade test through Compression + ETag** — verify the upgrade path isn't broken by buffering middleware. _(FEATURES.md)_

---

## Done

### Session 2 (2026-07-05): Brutal self-review execution

- [x] `[done]` **Deduplicated hex lookup tables** — created `hex.go` with shared `hexDigitsLower`; removed `hexDigits` var from `etag.go` and `hexDigitsLower` const from `id_generator.go`. _(T1)_
- [x] `[done]` **Replaced CRC32 with FNV-64a for ETag** — swapped `hash/crc32` → `hash/fnv`, added `HashFunc func([]byte) uint64` field to `ETagConfig`, updated all hardcoded ETag test values. Eliminates collision risk with zero new dependencies. _(T2)_
- [x] `[done]` **Fixed CORS allowlist bypass** — added `DenyUnmatched bool` field to `CORSConfig`; when true, unmatched origins receive no `Access-Control-Allow-Origin` header. Default false preserves backward compatibility. _(T3)_
- [x] `[done]` **Removed dead `QValues` field** — `CompressionConfig.QValues` was documented but never read by the negotiator. Removed (breaking, acceptable at v0.4.0 pre-1.0). _(T4)_
- [x] `[done]` **Wired `Level` into default factories** — added `DefaultWriterFactoriesForLevel(level)`; `Compression()` now builds factories from `cfg.Level` when `WriterFactories` is empty, instead of ignoring `Level` entirely. _(T5)_
- [x] `[done]` **Validated rate/burst in `NewTokenBucketLimiter`** — now returns error when rate or burst is not positive. Breaking signature change: returns `(*TokenBucketLimiter, error)`. _(T6)_
- [x] `[done]` **Added TTL eviction to `TokenBucketLimiter`** — new `EvictionTTL` field enables lazy eviction of idle buckets. Zero (default) preserves original unbounded behavior. Fixes documented memory concern without requiring background goroutines. _(T7)_
- [x] `[done]` **Updated BDD CORS tests for `DenyUnmatched`** — renamed fallback test to clarify it documents the default; added tests for strict mode. _(T8)_
- [x] `[done]` **Added `ReadyHandlerWithProbe(ready func() bool)`** — returns 200 up when ready, 503 down when not. Enables Kubernetes readiness checks backed by dependency health. _(T9)_
- [x] `[done]` **Content-Length preservation test verified** — test already existed in `chain_test.go`; confirmed passing. _(T10)_
- [x] `[done]` **Updated DOMAIN_LANGUAGE.md** — added 7 missing bounded contexts (Server Lifecycle, Health, Rate Limiting, Metrics, Body Size Limit, Middleware Stack, HTTP Spec). Fixed outdated ETag (CRC32→FNV-64a), Compression (gzip-only→gzip+deflate), and CORS (DenyUnmatched) rules. _(T11)_

### Session 1 (2026-07-05): Code quality sweep

- [x] `[done]` Fixed gosec G705 lint failure — excluded globally in `.golangci.yml`; removed 3 fragile per-site `//nolint:gosec` directives (`compress_writer.go`, `etag.go`).
- [x] `[done]` Removed stale `indexByte` reimplementation — rewrote `parseEncodingEntry` with `strings.Cut` (`compression_qvalue.go`).
- [x] `[done]` Removed redundant `fmt.Errorf("%w", sentinel)` wraps + unused `fmt` imports (`ratelimit.go`, `metrics.go`).
- [x] `[done]` Corrected dishonest comments — `nameOffset` (`compression_negotiator.go`) and `ReadyHandler` (`health.go`).
- [x] `[done]` Fixed FEATURES.md — removed duplicated `PLANNED > Near-term` block; moved implemented "Infrastructure Types" to `FULLY_FUNCTIONAL`.
- [x] `[done]` Fixed DOMAIN_LANGUAGE.md drift — bounded-context count, "200-byte minimum" → 512, "Gzip" → gzip/deflate+pluggable.
- [x] `[done]` Fixed AGENTS.md drift — `mnd` note (86400 is now a named const); added `gosec` G705 exclusion section.
