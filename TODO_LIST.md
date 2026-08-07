# TODO List — httputil

Short- and mid-term improvement tasks. Each item verified against the actual code.

_Updated: 2026-08-07 — ETag compliance follow-up: escaped-quote fix, 304 header tests, multiple If-None-Match support, httpspec validation, interaction tests. See [CHANGELOG.md](CHANGELOG.md) for shipped work._

---

## Medium Priority

- [ ] **Add `ServerConfig.TLSConfig` validation** — `TLSConfig` is always nil in `NewServer()` but there is no validation for it if added. Deferred to v1.0. `server.go`. Estimated effort: 30min.
- [ ] **ETag + CORS interaction test** — does CORS `Vary` header affect ETag caching? Add integration test. Estimated effort: 20min.
- [ ] **ETag + Recovery interaction test** — does panic recovery bypass ETag generation? Add integration test. Estimated effort: 20min.
- [ ] **Fuzz test for `parseETagList` specifically** — quote/comma/backslash/escape combinations beyond the existing `FuzzETag`. Estimated effort: 30min.
- [ ] **Fuzz test for `stripWeakPrefix`** — ensure no panic on malformed input. Estimated effort: 15min.

## Low Priority

- [ ] **Consider `ErrCodeETagComputeFailed`** — for hash computation failures (currently impossible since FNV can't fail, but custom `HashFunc` could). `errors.go`, `etag.go`. Estimated effort: 30min.
- [ ] **Review error handling in `etagWriter.Flush()`** — `_, _ = w.ResponseWriter.Write(w.body)` silently ignores errors on the final body write. `etag.go:175`. Estimated effort: 30min.
- [ ] **ETag + Decompression interaction** — decompressed body should get ETag, not compressed bytes. Verify ordering. Estimated effort: 45min.
- [ ] **Cross-middleware ETag tests** — ServerTiming, RequestID, MaxBodySize, SecurityHeaders interactions with ETag. Estimated effort: 1hr.

## Won't Implement

These items were considered and rejected, with reasoning:

- **Remove `nopCloserWriter` / `nopFlushCloser`** — defensive scaffolding for the `WriterFactory` contract; only reachable via direct unit construction but kept for API safety. Documented in AGENTS.md.
- **Add `MustNewTokenBucketLimiter`** — `TokenBucketLimiter` is deprecated; new code uses `KeyedRateLimiter`. Dead code.
- **Property-based tests for token bucket** — existing benchmarks + integration tests cover the contract; rapid/quickcheck adds dependencies.
- **Add `AllowN` on rate limiter interface** — `KeyedRateLimiter` uses `MaxKeys`; `AllowN` is not the right primitive.
- **Make `delegatingWriter` exported** — internal; not part of the public API.
- **Extract entity-tag parsing into `entitytag` subpackage** — `parseETagList`, `stripWeakPrefix` are only used by `etag.go`; extraction would be premature. Revisit if If-Match support is added.

---

_Long-term vision and raw ideas live in [ROADMAP.md](ROADMAP.md). Completed work is recorded in [CHANGELOG.md](CHANGELOG.md)._
