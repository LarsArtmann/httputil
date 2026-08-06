# TODO List — httputil

Short- and mid-term improvement tasks. Each item verified against the actual code.

_Updated: 2026-08-06 — ETag weak-comparison fix shipped (v0.9.1). Remaining items harvested from the 2026-08-06 gap analysis. See [CHANGELOG.md](CHANGELOG.md) for shipped work._

---

## Medium Priority

- [ ] **Verify 304 response excludes `Content-Length` and includes `ETag`** — RFC 7232 §4.1 requires a 304 to carry cache-related headers (including `ETag`) but MUST NOT include body-related headers like `Content-Length`. The ETag middleware relies on stdlib implicit cleanup. Add two tests asserting both invariants. `etag.go`, `etag_test.go`. Estimated effort: 30min.
- [ ] **Handle escaped quotes in `parseETagList`** — backslash-escaped `\"` inside an opaque-tag currently flips `inQuotes` incorrectly. The RFC 7232 §2.3 grammar permits `%x5C` in quoted-strings. Extremely rare in practice (hex ETags never contain backslashes), but not fully spec-compliant for arbitrary client input. `etag.go:234`. Estimated effort: 20min.
- [ ] **ETag + Compression interaction test** — verify whether a compressed ETag matches an uncompressed `If-None-Match`. The compress-then-hash ordering matters; document the actual behavior with a test. `etag_test.go` / `compression_test.go`. Estimated effort: 45min.
- [ ] **`httpspec` test for ETag correctness** — point `httpspec.Run(t, handler)` at an ETag-wrapped handler to validate standard HTTP conventions. `httpspec/`. Estimated effort: 30min.

## Low Priority

- [ ] **Add `ServerConfig.TLSConfig` validation** — `TLSConfig` is always nil in `NewServer()` but there is no validation for it if added. Deferred to v1.0. `server.go`. Estimated effort: 30min.
- [ ] **ETag for response > 1MB with `If-None-Match`** — the overflow-to-streaming path disables ETag generation, but the interaction with a conditional request header is not tested. Add a test documenting the behavior. `etag_test.go`. Estimated effort: 20min.

## Won't Implement

These items were considered and rejected, with reasoning:

- **Remove `nopCloserWriter` / `nopFlushCloser`** — defensive scaffolding for the `WriterFactory` contract; only reachable via direct unit construction but kept for API safety. Documented in AGENTS.md.
- **Add `MustNewTokenBucketLimiter`** — `TokenBucketLimiter` is deprecated; new code uses `KeyedRateLimiter`. Dead code.
- **Property-based tests for token bucket** — existing benchmarks + integration tests cover the contract; rapid/quickcheck adds dependencies.
- **Add `AllowN` on rate limiter interface** — `KeyedRateLimiter` uses `MaxKeys`; `AllowN` is not the right primitive.
- **Make `delegatingWriter` exported** — internal; not part of the public API.

---

_Long-term vision and raw ideas live in [ROADMAP.md](ROADMAP.md). Completed work is recorded in [CHANGELOG.md](CHANGELOG.md)._
