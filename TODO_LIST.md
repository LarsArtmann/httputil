# TODO List — httputil

Short- and mid-term improvement tasks. Each item verified against the actual code.

_Updated: 2026-08-07 — post-self-critique execution: writeBufferedBody ceremony reverted to honest silence, compressWriter split brain resolved, decompression bomb-protection tests added (coverage 96.7% to 97.2%), Decompression documented across README/v1-stability/DOMAIN_LANGUAGE/D2, ETag edge-case tests added, ExampleDecompression added, `nix flake check` run. See [CHANGELOG.md](CHANGELOG.md) `[Unreleased]` for shipped work._

---

## High Priority

- [ ] **ETag + Decompression interaction** — decompressed body should get ETag, not compressed bytes. Verify ordering with `Chain(inner, Decompression(), ETag())`. `chain_test.go`. Effort: 45min.
- [ ] **Add `nix run .#vulncheck` to RELEASE.md** — document the new vulncheck app in the release runbook step 4. `docs/RELEASE.md`. Effort: 10min.

## Medium Priority

- [ ] **ETag + CORS interaction test** — does CORS `Vary` header affect ETag caching? Add integration test. `chain_test.go`. Effort: 20min.
- [ ] **ETag + Recovery interaction test** — does panic recovery bypass ETag generation? Add integration test. `chain_test.go`. Effort: 20min.
- [ ] **Fuzz test for `parseETagList` specifically** — quote/comma/backslash/escape combinations beyond the existing `FuzzETag`. `etag_test.go`. Effort: 30min.
- [ ] **Fuzz test for `stripWeakPrefix`** — ensure no panic on malformed input (`W/`, `W`, empty). `etag_test.go`. Effort: 15min.
- [ ] **Add `ServerConfig.TLSConfig` validation** — `TLSConfig` is always nil in `NewServer()` but there is no validation for it if added. `server.go`. Effort: 30min.
- [ ] **Audit all `Validate()` methods for completeness** — ensure every config struct has one. Effort: 30min.

## Low Priority

- [ ] **Cross-middleware ETag tests** — ServerTiming, RequestID, MaxBodySize, SecurityHeaders interactions with ETag. `chain_test.go`. Effort: 1hr.
- [ ] **Consider `ErrCodeETagComputeFailed`** — for hash computation failures (currently impossible since FNV can't fail, but custom `HashFunc` could). `errors.go`, `etag.go`. Effort: 30min.
- [ ] **Benchmark decompression** — gzip + deflate + passthrough. `decompression_test.go`. Effort: 15min.
- [ ] **Fuzz test for decompression** — random compressed bodies. `decompression_test.go`. Effort: 30min.

## Won't Implement

These items were considered and rejected, with reasoning:

- **Remove `nopCloserWriter` / `nopFlushCloser`** — defensive scaffolding for the `WriterFactory` contract; only reachable via direct unit construction but kept for API safety. Documented in AGENTS.md.
- **Add `MustNewTokenBucketLimiter`** — `TokenBucketLimiter` is deprecated; new code uses `KeyedRateLimiter`. Dead code.
- **Property-based tests for token bucket** — existing benchmarks + integration tests cover the contract; rapid/quickcheck adds dependencies.
- **Add `AllowN` on rate limiter interface** — `KeyedRateLimiter` uses `MaxKeys`; `AllowN` is not the right primitive.
- **Make `delegatingWriter` exported** — internal; not part of the public API.
- **Extract entity-tag parsing into `entitytag` subpackage** — `parseETagList`, `stripWeakPrefix` are only used by `etag.go`; extraction would be premature. Revisit if If-Match support is added.
- **Wrap post-header-commit body-write errors** — these are fundamentally unreportable in Go's Handler model. Both `etagWriter` and `compressWriter` now document this with honest `_, _ =` + explanatory comment instead of wrapping ceremony.

---

_Long-term vision and raw ideas live in [ROADMAP.md](ROADMAP.md). Completed work is recorded in [CHANGELOG.md](CHANGELOG.md)._
