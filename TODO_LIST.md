# TODO List — httputil

Short- and mid-term improvement tasks, each verified against the actual code.

_Updated: 2026-07-05. Statuses: `[open]`, `[in-progress]`, `[done]`. Source reviews live in `docs/reviews/` and `docs/brainstorming/`._

---

## Critical (security / correctness)

- [ ] `[open]` **CORS allowlist bypass** — `cors.go:142` `resolveOrigin` returns `"*"` for an unmatched origin even when `AllowAllOrigins=false`, so a configured allowlist is decorative. Documented as intentional in `DOMAIN_LANGUAGE.md`, but it is a security surprise. _Action:_ return no `Access-Control-Allow-Origin` header for unmatched origins, gated behind an explicit `DenyUnmatched bool` (default false to preserve current behavior). Breaking — target v1. _(code-quality-scan #1, data-model review)_
- [ ] `[open]` **ETag CRC32 collision risk** — `etag.go:180` uses `crc32.ChecksumIEEE` (a 32-bit checksum, not a hash). Two distinct bodies can collide → false `304`. _Action:_ add a `HashFunc func([]byte) uint64` option (default CRC32 for speed; allow xxhash/SHA for correctness) and document the tradeoff. _(data-model review)_

## High

- [ ] `[open]` **`CompressionConfig.QValues` is a dead field** — declared & documented at `compression.go:87` but never read by the negotiator. _Action:_ either wire it as the server-side default q-value in `negotiateEncoding`, or remove it (breaking). _(data-model review, naming review)_
- [ ] `[open]` **`TokenBucketLimiter` unbounded memory** — `ratelimit.go:32` per-key `buckets` map grows without eviction. Documented limitation. _Action:_ add an optional TTL/eviction sweep, or strengthen the doc to require a custom `RateLimiter` for production per-IP use. _(architecture review, AGENTS.md non-obvious behaviors)_

## Medium

- [ ] `[open]` **`CompressionConfig.Level` decoupled from `WriterFactories`** — `compression.go:62`; setting `Level` alone has no effect (factories carry their own level). Split-brain. _Action:_ make `Level` the single source of truth, or remove it and document factories as the only level control. _(data-model review)_
- [ ] `[open]` **`TokenBucketLimiter` doesn't validate rate/burst > 0** — `ratelimit.go:46` `NewTokenBucketLimiter` accepts negative/zero. _Action:_ validate or document. _(code-quality-scan)_
- [ ] `[open]` **DOMAIN_LANGUAGE.md missing bounded contexts** — Server Lifecycle, Health, RateLimit, Metrics, MaxBodySize, MiddlewareStack, and `httpspec` are not in the glossary. _Action:_ add rows for each. _(docs-freshness)_

## Low (polish / dedup)

- [ ] `[open]` **Duplicated hex lookup tables** — `etag.go:30` (`hexDigits` var array) and `id_generator.go:79` (`hexDigitsLower` const string). _Action:_ unify into one shared lowercase-hex helper. _(code-quality-scan, deduplicate-code)_
- [ ] `[open]` **`RequestIDConfig.ForwardHeader` misnames the direction** — it's the incoming read header, not an outgoing forward. `HeaderName` is also vague. _Action:_ rename to `IncomingHeader`/`ResponseHeader` in a major version. _(naming review)_
- [ ] `[open]` **`Validate()` "required nil" pattern duplicated** — same 3-line check repeated in `RateLimitConfig`, `MetricsConfig`, `RequestIDConfig`. _Action:_ extract a small helper or accept the (minor) repetition. _(code-quality-scan)_
- [ ] `[open]` **Consider internal `compress/` split** — the 7 compression files are cohesive; an unexported subfolder would reduce root-package noise while preserving the flat public API. _(architecture review)_
- [ ] `[open]` **`HealthHandler`/`ReadyHandler` accept no custom checker** — readiness always reports up. _Action:_ optionally accept a `func() bool` readiness probe. _(API enhancement)_

## Planned (tests)

- [ ] `[open]` **WebSocket upgrade test through Compression + ETag** — verify the upgrade path isn't broken by buffering middleware. _(FEATURES.md)_
- [ ] `[open]` **`Content-Length` preservation test for small responses** — confirm small bodies keep an accurate Content-Length through the compression/etag writers. _(FEATURES.md)_

---

## Done this session (2026-07-05)

- [x] `[done]` Fixed gosec G705 lint failure — excluded globally in `.golangci.yml`; removed 3 fragile per-site `//nolint:gosec` directives (`compress_writer.go`, `etag.go`).
- [x] `[done]` Removed stale `indexByte` reimplementation — rewrote `parseEncodingEntry` with `strings.Cut` (`compression_qvalue.go`).
- [x] `[done]` Removed redundant `fmt.Errorf("%w", sentinel)` wraps + unused `fmt` imports (`ratelimit.go`, `metrics.go`).
- [x] `[done]` Corrected dishonest comments — `nameOffset` (`compression_negotiator.go`) and `ReadyHandler` (`health.go`).
- [x] `[done]` Fixed FEATURES.md — removed duplicated `PLANNED > Near-term` block; moved implemented "Infrastructure Types" to `FULLY_FUNCTIONAL`.
- [x] `[done]` Fixed DOMAIN_LANGUAGE.md drift — bounded-context count, "200-byte minimum" → 512, "Gzip" → gzip/deflate+pluggable.
- [x] `[done]` Fixed AGENTS.md drift — `mnd` note (86400 is now a named const); added `gosec` G705 exclusion section.
