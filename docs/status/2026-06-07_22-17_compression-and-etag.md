# Status Report — httputil

**Date:** 2026-06-07 22:17 CEST
**Branch:** `master`
**Commits ahead of origin:** 0 (working tree has uncommitted changes)
**Working tree:** 4 modified + 4 untracked files
**Last commit:** `64f7f3b` (HEAD) — docs: add zero clones deduplication status update

---

## Executive Summary

This session added **two major middleware features** to httputil: **response compression** (gzip) and **ETag generation** with conditional request handling. Both follow the established `func(http.Handler) http.Handler` pattern, include configuration structs with `Validate()` methods, and ship with comprehensive test coverage. The library now covers 11 public middleware/features (up from 9).

The project sits at **~2,500 lines** across 25 Go files, **91 passing tests**, **79.9% test coverage**, and **zero lint issues** with ~70 linters enabled. Coverage dropped from 90.1% because new code paths (error branches, edge cases in custom ResponseWriter wrappers) are not fully exercised.

---

## a) FULLY DONE

### 1. Compression Middleware — Complete

**What:** `Compression()` middleware transparently gzip-compresses responses when the client accepts `gzip` encoding and the response body exceeds a configurable minimum size (default: 512 bytes).

**Design decisions:**

- Buffers writes in memory until `minSize` is reached or the handler completes.
- Only compresses 2xx responses. Non-2xx and already-encoded responses are passed through uncompressed.
- Adds `Vary: Accept-Encoding` to all responses where compression is evaluated.
- Custom `compressWriter` wraps `http.ResponseWriter` and delegates `Flush`, `Hijack`, and `Push` when the underlying writer supports them.
- `CompressionConfig.Validate()` checks that `Level` is within `gzip.HuffmanOnly` to `gzip.BestCompression`, and that `MinSize` is non-negative.
- Error classification via `go-error-family` for `Hijack` and `Push` failures (reuses existing `ErrCode*` constants).

**Files:**

- `compression.go` — 267 lines
- `compression_test.go` — 209 lines, 9 tests

**Tests cover:**

- No `Accept-Encoding` header → passthrough
- `Accept-Encoding: gzip` with large body → compressed
- Small response below `minSize` → uncompressed passthrough
- Non-2xx status code → uncompressed
- Already-encoded response (`Content-Encoding: br`) → skipped
- `Flush()` during response → disables compression, writes buffered data
- Empty response → uncompressed
- `Vary` header correctness
- `Validate()` for valid config, invalid level, negative min size

**Status:** Complete and passing. Zero lint issues.

---

### 2. ETag Middleware — Complete

**What:** `ETag()` middleware generates `ETag` response headers from CRC-32 checksums of response bodies and handles `If-None-Match` conditional requests with `304 Not Modified`. Only applies to `GET` and `HEAD` requests.

**Design decisions:**

- Buffers the entire response body in memory to compute the ETag.
- Uses `crc32.ChecksumIEEE` for fast hashing. Produces 8-hex-digit strong ETags (`"0d4a1185"`) or weak ETags (`W/"0d4a1185"`) depending on config.
- Supports exact-match `If-None-Match` comparison and wildcard `*` match.
- Only returns 304 for cacheable 2xx statuses (200 OK and implicit 200).
- Custom `etagWriter` wraps `http.ResponseWriter` and delegates `Flush`, `Hijack`, and `Push`.
- `Flush()` disables ETag generation and writes buffered data immediately (streaming mode).

**Files:**

- `etag.go` — 215 lines
- `etag_test.go` — 203 lines, 9 tests

**Tests cover:**

- Strong ETag generation
- Weak ETag generation (`Weak: true`)
- `If-None-Match` exact match → 304 with empty body
- `If-None-Match` no match → 200 with full body
- `If-None-Match: *` → 304
- Non-GET/HEAD methods → passthrough with no ETag
- Empty body → ETag of `"00000000"`
- `Flush()` → disables ETag, writes full body
- `HEAD` request → ETag generated

**Status:** Complete and passing. Zero lint issues.

---

### 3. Documentation Updates — Complete

**What:** Updated all public-facing documentation to reflect the two new features.

**Files changed:**

- `doc.go` — Added compression and ETag to package description.
- `README.md` — Added "Response Compression" and "ETag Generation" feature sections with usage examples. Updated API table with 4 new rows (`Compression`, `DefaultCompressionConfig`, `ETag`, `DefaultETagConfig`).
- `AGENTS.md` — Updated architecture table with `compression.go` and `etag.go` entries. Updated date.

**Status:** Complete.

---

## b) PARTIALLY DONE

### 1. Error Classification for Compression — Partial

**What:** Compression writer reuses existing `ErrCodeHijackUnsupported`, `ErrCodeHijackFailed`, `ErrCodePushUnsupported`, and `ErrCodePushFailed` for `Hijack()` and `Push()` failures. However, `Write()` errors from the gzip writer and the underlying response writer are NOT classified with `go-error-family`.

**Gap:** `ResponseRecorder.Write()` wraps errors with `errorfamily.WrapTransient(..., ErrCodeWriteFailed, ...)`. `compressWriter.Write()` returns `fmt.Errorf("failed to write to gzip writer: %w", err)` and `fmt.Errorf("failed to write to response writer: %w", err)` — these are plain errors, not classified.

**Why it matters:** Consumers using `errorfamily.Classify()` to make retry decisions will see plain errors from compression, breaking the classified-error contract established by the rest of the library.

**Fix needed:** Add `ErrCodeCompressWriteFailed` and `ErrCodeCompressFlushFailed` (or similar) and wrap compression-specific write errors consistently.

---

### 2. Benchmarks — Missing

**What:** The codebase has existing benchmarks for `itoa`, `join`, `ClientIP`, and `CORS`. No benchmarks were added for `Compression` or `ETag`.

**Gap:** We have no baseline to measure compression throughput, allocation profile, or ETag overhead. This makes performance regressions undetectable.

**Fix needed:** Add `BenchmarkCompression`, `BenchmarkETag`, and ideally a benchmark for the combined `Compression(ETag(...))` chain.

---

### 3. Example Functions — Missing

**What:** The codebase has 4 `Example*` functions in `example_test.go` for godoc. No examples added for the new middleware.

**Gap:** godoc users won't see runnable examples for `Compression` or `ETag`.

**Fix needed:** Add `ExampleCompression` and `ExampleETag` to `example_test.go`.

---

### 4. Integration Tests — Missing

**What:** All tests exercise `Compression` and `ETag` in isolation.

**Gap:** No test verifies that `Compression` and `ETag` compose correctly when chained together. For example:

- Does `Chain(handler, ETag(cfg), Compression(cfg))` produce a correct gzipped response with an ETag?
- Does the ETag get computed on the uncompressed or compressed body?
- Does `If-None-Match` still work when compression is also active?

**Critical architectural concern:** The order of `ETag` and `Compression` in the middleware chain matters significantly. If `Compression` is outermost, the ETag is computed on the uncompressed body (good). If `ETag` is outermost, the ETag is computed on the compressed body (bad — same content could produce different ETags due to gzip metadata). This should be documented and tested.

---

## c) NOT STARTED

### 1. sync.Pool for gzip.Writer Reuse — Not Started

**What:** Every request that triggers compression creates a brand-new `gzip.Writer` via `gzip.NewWriterLevel(w.ResponseWriter, w.level)`.

**Why it matters:** `gzip.Writer` allocates significant internal state (hash tables, sliding window, Huffman tables). For high-throughput servers, this is a major allocation hot spot. The standard pattern (used by `NYTimes/gziphandler`, `chi/middleware`, etc.) is to maintain a `sync.Pool` of `gzip.Writer` instances and `Reset()` them for each request.

**Effort:** Medium — ~20 lines of code + benchmark to verify improvement.

**Impact:** High for throughput-sensitive deployments.

### 2. Content-Type Filtering for Compression — Not Started

**What:** The compression middleware compresses ALL 2xx responses regardless of content type. It does not skip already-compressed formats like `image/png`, `image/jpeg`, `video/mp4`, `application/gzip`, `application/pdf`, etc.

**Why it matters:** Compressing already-compressed data wastes CPU and can increase response size (gzip overhead on incompressible data).

**Effort:** Low — add a deny-list of MIME type prefixes.

**Impact:** Medium — saves CPU on image/video assets.

### 3. Brotli / Deflate Support — Not Started

**What:** Only `gzip` encoding is supported. Modern browsers prefer `br` (brotli) and some support `deflate`.

**Why it matters:** Brotli achieves 15–25% better compression than gzip at comparable speeds. Supporting it is table stakes for a modern compression middleware.

**Constraint:** `depguard` only allows stdlib + `go-error-family`. Brotli is not in the stdlib.

**Effort:** High — would require adding `github.com/andybalholm/brotli` or `github.com/klauspost/compress`, which violates the single-dependency policy.

**Impact:** High for modern web performance, but blocked by dependency policy.

### 4. Multiple ETags in If-None-Match — Not Started

**What:** The ETag middleware's `matchesIfNoneMatch()` only does exact string comparison:

```go
func (w *etagWriter) matchesIfNoneMatch(req *http.Request, etag string) bool {
    inm := req.Header.Get(headerIfNoneMatch)
    if inm == "*" { return true }
    return inm == etag
}
```

**Why it matters:** Per RFC 7232, `If-None-Match` can contain a comma-separated list of ETags, optionally with whitespace. A client might send:

```
If-None-Match: "abc123", "def456", "0d4a1185"
```

Our current implementation would fail to match the third ETag.

**Effort:** Low — parse comma-separated list, trim whitespace, compare each.

**Impact:** Medium — correctness issue for real-world HTTP clients.

### 5. Cacheable Status Code Expansion — Not Started

**What:** `isCacheableStatus()` only returns `true` for status `0` (implicit 200) and `http.StatusOK` (200).

```go
func (w *etagWriter) isCacheableStatus() bool {
    return w.status == 0 || w.status == http.StatusOK
}
```

**Why it matters:** Per RFC 7232, 201 Created, 204 No Content, and other 2xx responses can also carry ETags and be subject to conditional requests. Restricting to 200 breaks caching for valid use cases (e.g., POST-then-redirect with 201).

**Effort:** Low — change to `w.status >= 200 && w.status < 300`.

**Impact:** Medium — correctness for non-200 2xx responses.

### 6. Streaming ETag (No Buffering) — Not Started

**What:** The current ETag middleware buffers the ENTIRE response body in memory before computing the ETag. For large responses (e.g., file downloads, JSON APIs returning MBs of data), this is a memory problem.

**Why it matters:** A server streaming a 100MB file would buffer all 100MB in the `etagWriter.body` slice.

**Options:**

- **Option A:** Skip ETag generation for responses above a size threshold (pass through).
- **Option B:** Use a rolling hash (e.g., xxhash64, md5) and write chunks as they arrive, then compute the final ETag at close. This still requires buffering if `If-None-Match` needs to short-circuit to 304.
- **Option C:** Require the handler to pre-compute and set its own ETag, making the middleware a passthrough validator only.

**Effort:** Medium — requires careful design to maintain 304 semantics without full buffering.

**Impact:** High for large-response use cases.

### 7. Weak ETag Semantics — Not Started

**What:** The `Weak` config field changes the ETag prefix from `"..."` to `W/"..."`, but the hash is still computed on the exact byte sequence.

**Why it matters:** Per RFC 7232, weak ETags should change only when the response semantics change, not when the byte representation changes. A true weak ETag might ignore trailing whitespace, encoding differences, etc. Our implementation is "weak in name only."

**Effort:** Low (document current behavior) to High (implement semantic hashing).

**Impact:** Low — most consumers treat weak ETags as "may change at any time, just don't use for byte-range."

---

## d) TOTALLY FUCKED UP!

### Nothing is totally fucked up.

Both middlewares are functional, tested, lint-clean, and follow established patterns. However, the following are **architectural debts** that will bite us if not addressed:

1. **No error classification for compression write errors** — breaks the library's error-classification contract.
2. **ETag buffers entire response in memory** — unbounded memory growth for large responses.
3. **No gzip.Writer pooling** — unnecessary allocations on every compressed request.
4. **No content-type filtering** — wastes CPU compressing images/videos.
5. **If-None-Match parsing is RFC-noncompliant** — only handles exact match, not lists.
6. **isCacheableStatus is too restrictive** — only 200, not all 2xx.

None of these are "fucked up" in the sense of breaking the build or causing panics. They are **correctness and performance gaps** that should be fixed before calling these features "production-ready."

---

## e) WHAT WE SHOULD IMPROVE

### Critical (Fix before next release)

1. **Add error classification for compression write errors.** Compression-specific write failures should use `go-error-family` with new error codes (`http.compress_write_failed`, `http.compress_flush_failed`).
2. **Fix `If-None-Match` to support comma-separated ETag lists per RFC 7232.**
3. **Fix `isCacheableStatus` to accept all 2xx responses, not just 200.**
4. **Add memory safety to ETag.** Set a maximum buffer size (e.g., 1MB). If the response exceeds it, flush the buffer and disable ETag generation.

### High Impact / Low Effort

5. **Add `sync.Pool` for `gzip.Writer` reuse.** ~20 lines, significant allocation reduction.
6. **Add content-type filtering to skip compression for already-compressed formats.**
7. **Add benchmarks for Compression and ETag.**
8. **Add example functions for godoc.**
9. **Add integration test for `Chain(handler, ETag(cfg), Compression(cfg))`.**
10. **Document recommended middleware ordering** (ETag should wrap the handler, Compression should wrap ETag — so ETag sees uncompressed body).

### Medium Impact / Medium Effort

11. **Extract a shared `ResponseWriter` wrapper helper.** Both `compressWriter` and `etagWriter` duplicate `WriteHeader` buffering, `Hijack`, `Push`, and `Flush` delegation. A shared `wrappedWriter` type (or struct embedding) would eliminate ~80 lines of duplication and make future writers easier.
12. **Support deflate encoding** (`Accept-Encoding: deflate`). Stdlib has `compress/flate` — no new dependencies needed.
13. **Add `Content-Length` handling for small responses.** Currently `Content-Length` is always deleted when compression starts. For small responses that don't trigger compression, the original `Content-Length` (if set by the handler) should be preserved.
14. **Add `Accept-Encoding` quality value parsing.** Currently we do `strings.Contains("gzip")`. RFC-compliant parsing would handle `gzip;q=0.8, deflate;q=0.5` correctly.

### Long-term / Architectural

15. **Consider a `ResponseWriter` interface hierarchy.** Instead of every middleware re-implementing `Hijacker`, `Pusher`, `Flusher` detection, define an interface that advertises which optional interfaces are supported, and a helper that delegates method calls safely.
16. **Consider a `MiddlewareStack` type** that encapsulates ordering rules (e.g., "Compression must be outermost, Recovery must be innermost, ETag must be before Compression").

---

## f) Top #25 Things We Should Get Done Next

Sorted by **impact / effort ratio** (Pareto principle — highest impact per unit of work first):

| #  | Task                                                       | Impact | Effort | Category        |
| -- | ---------------------------------------------------------- | ------ | ------ | --------------- |
| 1  | Fix `If-None-Match` to parse comma-separated ETag lists    | High   | 5 min  | Correctness     |
| 2  | Fix `isCacheableStatus` to accept all 2xx                  | High   | 2 min  | Correctness     |
| 3  | Add ETag memory limit (skip ETag if body > 1MB)            | High   | 15 min | Safety          |
| 4  | Add error classification for compression write errors      | High   | 20 min | Architecture    |
| 5  | Add `sync.Pool` for gzip.Writer reuse                      | High   | 20 min | Performance     |
| ~~6~~  | ~~Add content-type filtering for compression~~ done — shipped (IncompressibleTypes) | ~~Medium~~ | ~~15 min~~ | ~~Performance~~ |
| 7  | Add benchmarks for Compression and ETag                    | Medium | 15 min | Observability   |
| ~~8~~  | ~~Add example functions for godoc~~ done | ~~Medium~~ | ~~10 min~~ | ~~DX~~ |
| 9  | Add integration test for ETag + Compression chain          | Medium | 15 min | Correctness     |
| ~~10~~ | ~~Document recommended middleware ordering in README~~ done — (ordering section + mermaid) | ~~Medium~~ | ~~5 min~~ | ~~DX~~ |
| ~~11~~ | ~~Extract shared ResponseWriter wrapper helper~~ done — shipped (wrapper.go responseWrapper) | ~~Medium~~ | ~~30 min~~ | ~~Architecture~~ |
| ~~12~~ | ~~Add deflate support~~ done — shipped (DefaultWriterFactories) | ~~Medium~~ | ~~30 min~~ | ~~Feature~~ |
| ~~13~~ | ~~Support `Accept-Encoding` quality value parsing~~ done — shipped (compression_qvalue.go + property tests) | ~~Low~~ | ~~20 min~~ | ~~Correctness~~ |
| 14 | Add `Content-Length` preservation for small responses      | Low    | 15 min | Correctness     |
| ~~15~~ | ~~Add `ETagConfig` max buffer size field~~ done — shipped (go-etag ETagConfig buffer limit) | ~~Low~~ | ~~10 min~~ | ~~Configurability~~ |
| ~~16~~ | ~~Add `CompressionConfig` content type allow/deny lists~~ done — shipped (IncompressibleTypes) | ~~Low~~ | ~~15 min~~ | ~~Configurability~~ |
| ~~17~~ | ~~Add weak ETag documentation clarifying "weak in name only"~~ done — (go-etag docs + v0.9.1 weak comparison) | ~~Low~~ | ~~5 min~~ | ~~DX~~ |
| ~~18~~ | ~~Add streaming ETag option (no buffering)~~ done — Won't implement — ROADMAP Non-goals: headers precede body, buffering is mandatory | ~~Medium~~ | ~~45 min~~ | ~~Performance~~ |
| ~~19~~ | ~~Add brotli support (blocked: requires external dep)~~ done — shipped as WriterFactory plugin docs (docs/integrations/brotli-zstd.md) | ~~High~~ | ~~60 min~~ | ~~Feature~~ |
| ~~20~~ | ~~Consider `MiddlewareStack` type with ordering rules~~ done — shipped (stack.go) | ~~Low~~ | ~~60 min~~ | ~~Architecture~~ |
| ~~21~~ | ~~Fuzz test compression with random bodies~~ done — (FuzzCompression round-trip invariant) | ~~Medium~~ | ~~20 min~~ | ~~Quality~~ |
| ~~22~~ | ~~Fuzz test ETag with random bodies and headers~~ done — (go-etag module fuzz suite) | ~~Medium~~ | ~~20 min~~ | ~~Quality~~ |
| ~~23~~ | ~~Add HTTP/2 Push test for compression writer~~ done — moot (http.Pusher code removed in v0.3.0) | ~~Low~~ | ~~15 min~~ | ~~Coverage~~ |
| ~~24~~ | ~~Add Hijack test for ETag writer~~ done — (chain_hijack_test + wrapper_test, 2026-08-30) | ~~Low~~ | ~~15 min~~ | ~~Coverage~~ |
| ~~25~~ | ~~Profile allocation hot spots under benchmark load~~ done — (benchmarks + research notes) | ~~Medium~~ | ~~30 min~~ | ~~Performance~~ |

---

## g) Top #1 Question I Cannot Figure Out Myself

**How do we reconcile the single-dependency constraint (`depguard` only allows stdlib + `go-error-family`) with the desire for brotli compression?**

Brotli is table stakes for modern web compression (15–25% smaller than gzip). The standard library does not include brotli. `compress/flate` exists for deflate, but brotli is not there.

Options:

1. ~~**Keep the constraint** and document that brotli is intentionally not supported. Users who need brotli should use a different middleware.~~ done (chosen & documented: constraint kept; brotli via WriterFactory docs)
2. ~~**Relax the constraint** for a zero-dependency brotli implementation. `github.com/andybalholm/brotli` is pure Go with zero transitive deps — similar profile to our existing `go-error-family` dependency.~~ done (Won't implement — dependency policy; WriterFactory plugin interface shipped instead)
3. ~~**Provide a plugin/extension interface** where users inject their own compression writer factory, keeping the core stdlib-only.~~ done (shipped (WriterFactory plugin interface))

**My recommendation:** Option 3 — add a `CompressionConfig.WriterFactory` field that accepts `func(io.Writer) io.WriteCloser`. The default uses stdlib gzip. Users who want brotli provide their own factory. This keeps the core dependency-free while allowing extensibility.

**But:** This adds API surface area and complexity. Is the tradeoff worth it? Should we instead keep compression simple (gzip only) and let users chain a third-party compression middleware if they need more?

**I need a decision on this before proceeding with any compression enhancements.**
