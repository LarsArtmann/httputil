# Comprehensive Project Status

**Project:** `github.com/larsartmann/httputil`  
**Branch:** `master`  
**Date/Time:** 2026-06-14 15:57:45 CEST  
**Go Version:** 1.26.3  
**Reporter:** Crush (AI Engineering Partner)

---

## Executive Summary

The codebase is in a **strong, shippable state**. The latest sprint delivered two major feature pillars:

1. **Modern compression middleware** — `Accept-Encoding` negotiation with RFC 7231 q-values, built-in `gzip` + `deflate`, and a `WriterFactory` plugin interface for brotli/zstd/lz4 without adding core dependencies.
2. **Time-ordered request ID generator** — sortable, monotonic, amortized `crypto/rand` 16-byte IDs.

All quality gates pass: `go test`, race tests, `go vet`, and `golangci-lint run` report zero issues. However, there are **latent design questions and documentation/formatter inconsistencies** that should be resolved before cutting v0.2.0.

### Current Metrics

| Metric | Value | Status |
| --- | --- | --- |
| Tests | 193 | ✅ passing |
| Coverage | 90.4% | ✅ above 90% target |
| Lint | 0 issues | ✅ clean |
| Race | 0 races (`-count=3`) | ✅ clean |
| Production `.go` files | 19 | — |
| Test files | 21 | — |
| Total Go LOC | ~5,949 | — |

---

## a) FULLY DONE

### Compression System

- ✅ RFC 7231 `Accept-Encoding` negotiation with q-value parsing.
- ✅ Built-in `gzip` and raw `deflate` support via `GzipWriterFactory()` / `DeflateWriterFactory()`.
- ✅ `WriterFactory` plugin interface for extensible encodings (brotli, zstd, lz4).
- ✅ `DefaultWriterFactories()` helper to extend without replacing defaults.
- ✅ Per-factory `sync.Pool` keyed by `WriterFactory` pointer; `Reset(io.Writer)` reuse for gzip/deflate.
- ✅ Pre-allocation of `compressWriter.buf` to `max(minSize, 512)` capacity.
- ✅ Backward compatibility: empty `WriterFactories` falls back to defaults.
- ✅ `Vary: Accept-Encoding` header added on every response.
- ✅ Content-type deny-list still active for incompressible formats.
- ✅ Comprehensive negotiator tests in `compression_negotiator_test.go`.

### Request ID System

- ✅ New `id_generator.go` module.
- ✅ 16-byte time-ordered ID layout: Unix seconds (4 B) + atomic counter (4 B) + random tail (8 B).
- ✅ 32-character lowercase hex output.
- ✅ Lexicographically sortable and monotonic within a second.
- ✅ Amortized `crypto/rand` via a 2048-byte process-wide buffer (one syscall every ~256 IDs).
- ✅ Thread-safe refill with mutex + double-checked atomic slot allocation.
- ✅ Tests for format, uniqueness, sortability, default middleware behavior, and concurrency.

### Documentation

- ✅ Performance review HTML updated with new benchmark data, artifact callouts, and implemented optimizations.
- ✅ `README.md` rewritten compression section, added `WriterFactory` examples and `CompressionConfig` table.
- ✅ `FEATURES.md` updated inventory and stats.
- ✅ `TODO_LIST.md` marked completed items.
- ✅ `AGENTS.md` updated architecture table and non-obvious behaviors.
- ✅ `CHANGELOG.md` added Unreleased entry.

### Quality / Cleanup

- ✅ Removed unused `requestIDBytes` constant.
- ✅ Suppressed G705 false positive in `etag.go` `Flush()` with `//nolint:gosec`.
- ✅ Converted table-driven q-value tests to standalone test functions per project convention.

---

## b) PARTIALLY DONE

- ⚠️ **`CompressionConfig.QValues` field is documented but not wired into negotiation.**
  - It is declared, has doc comments, defaults to `nil`, and is listed in the README.
  - No code reads `cfg.QValues` during `buildNegotiator` or `negotiateEncoding`.
  - This is either a placeholder that should be removed or a feature that needs implementation.

- ⚠️ **Content-type filtering is still hardcoded.**
  - `incompressiblePrefixes` slice exists in `compress_writer.go`.
  - `FEATURES.md` / `TODO_LIST.md` list configurable content-type filtering as not yet done.

- ⚠️ **Compression benchmark reports misleading memory numbers.**
  - `BenchmarkCompression` shows ~825 KB/op because `httptest.ResponseRecorder.Body` grows across iterations.
  - Documented in the performance review but not fixed in the benchmark code.

- ⚠️ **`README.md` API table formatting is inconsistent.**
  - Content is correct, but column alignment is visually uneven in several rows.

- ⚠️ **Test coverage is 90.4%.**
  - Meets the 90% target, but not 100%.
  - Gaps remain in compression error branches, CORS wildcard edge cases, `ResponseRecorder` hijack failures, and some `util.go` branches.

- ⚠️ **`id_generator_test.go` has gopls-only style warnings.**
  - `intrange`, `wsl_v5`, and `varnamelen` warnings appear in the editor even though `golangci-lint run` passes.

---

## c) NOT STARTED

- ❌ Configurable content-type deny-list in `CompressionConfig`.
- ❌ Fast path for single-encoding `Accept-Encoding` headers.
- ❌ `MiddlewareStack` type with ordering validation.
- ❌ `ResponseWriter` capability interface to unify Hijack/Flush detection.
- ❌ Streaming / rolling-hash ETag option.
- ❌ Request/response metrics middleware.
- ❌ Rate-limiting middleware.
- ❌ Request body size limit middleware.
- ❌ Example functions for custom `WriterFactory` usage in `example_test.go`.
- ❌ Dedicated deflate negotiation benchmark.
- ❌ Request ID generator benchmark.
- ❌ Fuzz tests for `parseQValue`.

---

## d) TOTALLY FUCKED UP

Nothing is **totally fucked up** in the sense of broken builds, failing tests, or data loss. However, the following items are **architecturally misleading** and could bite users:

- 🚨 **`CompressionConfig.QValues` is a zombie field.**
  - It appears in public API and README but has no effect.
  - Users who set it will believe they are influencing negotiation when they are not.

- 🚨 **`BenchmarkCompression` memory metric is a trap.**
  - Anyone reading the benchmark could conclude the middleware leaks ~825 KB per request.
  - This could block adoption or trigger false optimization work.

- 🚨 **Editor diagnostics disagree with CI.**
  - `golangci-lint run` reports 0 issues, but gopls surfaces cyclop, exhaustruct, varnamelen, err113, mnd, and modernize warnings in `compression.go`, `compress_writer.go`, and `id_generator_test.go`.
  - This creates confusion about the real quality gate and may cause contributors to chase phantom warnings.

---

## e) WHAT WE SHOULD IMPROVE

1. **Resolve `QValues`:** Either implement server-side q-value hints or delete the field before it becomes public API debt.
2. **Fix the compression benchmark:** Reset the recorder body each iteration or write to `io.Discard` to get a real allocation profile.
3. **Make content-type filtering configurable:** Move `incompressiblePrefixes` into `CompressionConfig` with a sensible default and a validation rule.
4. **Add a fast path for simple `Accept-Encoding` headers:** Exact match for `gzip` / `deflate` avoids full negotiation overhead.
5. **Reconcile gopls vs. CLI linter output:** Determine whether the gopls warnings are stale cache, missing config, or real issues suppressed in CLI.
6. **Refactor negotiation complexity:** `negotiateEncoding` and `parseQValue` have high cyclomatic complexity; split them or add explicit rationale/inline docs.
7. **Make the ID generator injectable:** Encapsulate global random buffer and counter behind a type so tests and multi-tenant users can avoid shared mutable state.
8. **Improve README formatting:** Align API table columns and proofread new sections.
9. **Add more error-path tests:** Compression writer pool type mismatch, `Close` errors, `startCompression` failures.
10. **Add examples:** `WriterFactory` extension in `example_test.go` and a RequestID benchmark.

---

## f) Top #25 Things to Get Done Next

1. Decide and implement the fate of `CompressionConfig.QValues`.
2. Fix `BenchmarkCompression` memory artifact.
3. Add configurable content-type filtering to `CompressionConfig`.
4. Implement fast path for single-encoding `Accept-Encoding`.
5. Split/refactor `negotiateEncoding` to reduce cyclomatic complexity.
6. Add tests covering `QValues` behavior once designed.
7. Add tests for compression error branches (`Close`, pool type mismatch, custom factory without `Reset`).
8. Add deflate-specific benchmark.
9. Add `parseQValue` fuzz tests.
10. Add `WriterFactory` examples to `example_test.go`.
11. Add RequestID generator benchmark.
12. Refactor `id_generator.go` into an injectable generator type.
13. Resolve gopls vs. `golangci-lint` diagnostic discrepancy.
14. Format `README.md` API tables.
15. Improve overall test coverage to 95%+.
16. Add CORS origin map/trie optimization.
17. Pre-join CORS config slices at validation time.
18. Implement `MiddlewareStack` with ordering validation.
19. Add `ResponseWriter` capability interface.
20. Evaluate streaming ETag with rolling hash.
21. Add request/response metrics middleware.
22. Add rate-limiting middleware.
23. Add request body size limit middleware.
24. Add WebSocket upgrade integration test through Compression + ETag.
25. Automate benchmark regression checks in CI.

---

## g) My Top #1 Question I Cannot Figure Out Myself

> **`CompressionConfig.QValues` is declared, documented, and exposed in the README, but it is never read by the negotiation logic. Was this field intended to be a server-side quality hint that influences encoding selection when the client omits q-values, or is it dead API surface that should be removed?**

The README says: *"Server-side quality hints for clients without q-values."* That implies it should affect negotiation, yet `buildNegotiator` and `negotiateEncoding` ignore it entirely. I need a product decision:

- **If it should work:** I will implement it (merge client q-values with server q-values, validate ranges in `Validate()`).
- **If it should not exist:** I will remove the field, its docs, and the README table entry before the next release to avoid lying API.

I cannot make this call without knowing the original intent.

---

## Verification Snapshot

```text
go test ./...                 OK (193 tests)
go test -race -count=3 ./...  OK
go vet ./...                  OK
golangci-lint run             0 issues
golangci-lint fmt --diff      no changes
```

## Risk Assessment

| Risk | Level | Notes |
| --- | --- | --- |
| Broken builds | 🟢 Low | All gates pass. |
| Misleading public API | 🟡 Medium | `QValues` field is documented but unused. |
| Misleading benchmarks | 🟡 Medium | `BenchmarkCompression` memory metric is artifact. |
| Editor/CI linter drift | 🟡 Medium | Confuses contributors. |
| Coverage gaps | 🟢 Low | 90.4% meets target. |
| Dependency policy | 🟢 Low | Still only `go-error-family`. |

## Recommendation

Ship the current feature set after **resolving `QValues`** and **fixing the compression benchmark**. Everything else is polish or future work.
