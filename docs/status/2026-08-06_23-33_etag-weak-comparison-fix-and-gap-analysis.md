# Status Report: ETag Weak Comparison Fix & Gap Analysis

**Date:** 2026-08-06 23:33
**Session scope:** ETag middleware (`etag.go`) — RFC 7232 compliance fix, review, and gap analysis
**Commit:** `9f49af2` — fix(etag): use RFC 7232 weak comparison for If-None-Match header

---

## What This Session Did

### 1. Reviewed the ETag Middleware

Read `etag.go` (274 lines), `etag_test.go` (392 lines), `etag_compress_fuzz_test.go` (106 lines), and searched for related conditional-request handling across the codebase.

**Verdict:** Strong "good," not superb. One real RFC 7232 compliance bug + a parsing edge case.

### 2. Researched the Go ETag Library Ecosystem

Searched Sourcegraph + web for all Go ETag libraries. Result:

| Library | Stars | Status | Adoptable? |
|---|---|---|---|
| `go-http-utils/etag` | 15 | Dead since 2016 | No |
| `amalfra/etag` | 32 | Maintained | No — only generates string, no 304 logic |
| Fiber `middleware/etag` | 40k | Active | No — fasthttp-coupled |
| `pablor21/echo-etag` | 11 | Maintained | No — Echo-only |
| `blizzy78/conditional-http` | 1 | Unknown | No — zero adoption |

**Conclusion:** No superb standalone ETag library exists for `net/http`. The depguard rules (only `go-error-family`, `x/time`, `nosurf`) would block all of them anyway. Keep building our own — fix the bugs and ours is the best in the ecosystem.

### 3. Fixed Two Bugs in `etagInList` (`etag.go`)

#### Bug 1: Wrong Comparison Function (RFC 7232 §2.3.2)

**Before:** `etagInList` used literal string `==`, which is the **strong comparison function**. But `If-None-Match` requires the **weak comparison function** per [RFC 7232 §3.2](https://www.rfc-editor.org/rfc/rfc7232#section-3.2).

| Server ETag | Client `If-None-Match` | RFC says | Old code |
|---|---|---|---|
| `"abc"` | `W/"abc"` | 304 | 200 (bug) |
| `W/"abc"` | `"abc"` | 304 | 200 (bug) |

**Fix:** Added `stripWeakPrefix()` that removes the optional `W/` prefix before comparing. `etagInList` now strips both sides and compares opaque-tags — the weak comparison function.

#### Bug 2: Naive Comma Splitting Broke on Commas Inside Quotes

**Before:** `strings.Index(list, ",")` split on every comma. But the `etagc` grammar (RFC 7232 §2.3) permits any VCHAR except `"` inside the opaque-tag — **including comma**. A client sending `"a,b"` would be parsed as two tags: `"a` and `b"`.

**Fix:** Replaced with `parseETagList()` — a quote-state-aware splitter that only splits on commas outside quoted strings.

### 4. Added 5 Tests

| Test | Covers |
|---|---|
| `TestETag_IfNoneMatch_WeakClientStrongServer` | `W/"..."` client vs strong server → 304 |
| `TestETag_IfNoneMatch_StrongClientWeakServer` | Strong client vs `W/"..."` server → 304 |
| `TestETag_IfNoneMatch_ListContainsWeakMatch` | Weak validator in a multi-element list → 304 |
| `TestETag_IfNoneMatch_WeakClientNoMatch` | Negative case — no false positives |
| `TestParseETagList_RespectsCommasInQuotes` | `"a,b"` parsed as single tag |

### 5. Verification

- `golangci-lint fmt` — clean
- `golangci-lint run` — 0 issues across ~70 linters
- `go test -race ./...` — all pass
- `FuzzETag` — 5s, 513k execs, PASS
- `FuzzETagConditional` — 6s, 959k execs, PASS

---

## a) FULLY DONE

1. **RFC 7232 §2.3.2 weak comparison** — implemented and tested bidirectionally
2. **Quote-aware list parsing** — `parseETagList` respects commas inside opaque-tags
3. **5 new tests** — covering weak/strong cross-matching, list containment, negative case, comma-in-quotes
4. **Lint clean** — 0 issues, formatted
5. **Race-clean** — `go test -race` passes
6. **Fuzz-clean** — both `FuzzETag` and `FuzzETagConditional` ran without failures
7. **Auto-committed** — `9f49af2`

---

## b) PARTIALLY DONE

1. **CHANGELOG.md `[Unreleased]`** — **NOT UPDATED.** The fix is committed but not recorded in the changelog. The `[Unreleased]` section is empty. This violates the project's own Keep a Changelog policy.
2. **Fuzz corpus enrichment** — The existing `FuzzETag` fuzz test (`etag_test.go:184`) only seeds exact-match `If-None-Match` values. No `W/"..."` seeds were added to its corpus. The fuzzer discovered weak-prefixed inputs organically (55 new interesting inputs), but explicit seeds documenting the fix would be better.

---

## c) NOT STARTED

1. **`If-Match` / `If-Unmodified-Since` support** — RFC 7232 §3.1 (preconditions for unsafe methods → 412 Precondition Failed). The middleware is GET/HEAD-only so this is structurally out of scope, but could be a separate middleware or config option.
2. **`If-Range` support** — RFC 7232 §3.2 + RFC 7233 (range requests). Combined with ETag or Last-Modified, enables partial-content (206) responses.
3. **`Last-Modified` / `If-Modified-Since` support** — the other half of conditional requests. Currently only `If-None-Match` (ETag-based) is handled.
4. **`Vary` header management** — the ETag middleware doesn't add `Vary` headers. Compression adds `Vary: Accept-Encoding`, but ETag generation based on body content doesn't need Vary. Still worth confirming this is correct for all proxy/cache scenarios.
5. **Handler-set ETag preservation** — `w.Header().Set(headerETag, etag)` always overwrites any ETag the downstream handler computed. A `SkipIfPresent` config option would let handlers with domain-specific modification semantics win.
6. **304 response header cleanup** — RFC 7232 §4.1 says a 304 response SHOULD include cache-related headers but MUST NOT include body-related headers like `Content-Length`. The current code relies on the stdlib implicitly handling this. Worth verifying with a test that asserts `Content-Length` is absent on 304.

---

## d) TOTALLY FUCKED UP

Nothing. The fix is correct, tested, lint-clean, race-clean, and fuzz-clean. The only process miss is the missing CHANGELOG entry (see b.1).

---

## e) WHAT WE SHOULD IMPROVE

### Code Quality

1. **`parseETagList` allocates a `[]string` slice on every request** — the old code was a zero-allocation string scan. For high-throughput servers, this is a regression on the hot path. Could be optimized with a callback-based walker (`func(tag string) bool`) or by inlining the comparison loop. Benchmark the old vs new to quantify.
2. **`stripWeakPrefix` doesn't validate the entity-tag shape** — it blindly removes `W/` from anything starting with those bytes. Malformed input like `W/foo` (unquoted) or `W/` alone would produce `foo` / empty string. The RFC requires `W/` to be followed by a quoted-string, but rejecting malformed input is defensive. Low priority since the comparison still can't match valid server ETags against garbage.
3. **Escaped quotes not handled in `parseETagList`** — backslash-escaped `\"` inside an opaque-tag would flip `inQuotes` incorrectly. The RFC grammar technically allows this (`%x5C` in quoted-string). Extremely rare in practice (hex ETags never contain backslashes), but the parser is technically not fully spec-compliant for arbitrary client input.

### Process

4. **CHANGELOG discipline** — the fix was committed without updating CHANGELOG.md. The auto-commit daemon doesn't know about the changelog policy. For deliberate fixes, update CHANGELOG before committing.
5. **Fuzz corpus should document the bug being fixed** — adding `W/"779a65e7023cd2e7"` as an explicit seed in `FuzzETag` would document the weak-comparison fix in the test corpus itself, not just in unit tests.
6. **Benchmark regression check** — no benchmark was run comparing old vs new `etagInList` performance. The `BenchmarkETag` exists but wasn't used to verify no regression.

### Architecture

7. **ETag middleware overwrites handler-set ETags** — this is a design limitation, not a bug. Document it explicitly or add a `SkipIfPresent bool` config.
8. **No `Last-Modified` counterpart** — ETag is one of two conditional-request mechanisms. The middleware is half of a complete caching solution.
9. **Single hash algorithm (FNV-64a)** — pluggable via `HashFunc`, but the default has a 64-bit collision space. For very large deployments, SHA-256 or BLAKE3 would be safer. The `HashFunc` option makes this a config change, not a code change.

---

## f) Up to 50 Things We Should Get Done Next

### ETag-Specific (P0–P1)

1. **Update CHANGELOG.md `[Unreleased]`** with the weak comparison fix
2. **Add `W/"..."` seeds to `FuzzETag` corpus** documenting the fix
3. **Benchmark `etagInList` old vs new** — quantify allocation regression
4. **Optimize `parseETagList` to zero-allocation** if benchmarks show meaningful regression
5. **Handle escaped quotes in `parseETagList`** for full RFC grammar compliance
6. **Add test: 304 response has no `Content-Length` header** (RFC 7232 §4.1)
7. **Add test: 304 response includes `ETag` header** (RFC 7232 §4.1)
8. **Add `SkipIfPresent bool` to `ETagConfig`** — don't overwrite handler-set ETags
9. **Add test: handler-set ETag is currently overwritten** — document current behavior
10. **Add test: multiple `If-None-Match` headers** (RFC 9110 §5.2 — combine field lines into one list)

### Conditional Requests — Broader (P1–P2)

11. **Implement `If-Match` middleware** for unsafe methods (412 Precondition Failed)
12. **Implement `If-Unmodified-Since` middleware** (412 Precondition Failed)
13. **Implement `Last-Modified` generation middleware** (analogous to ETag but timestamp-based)
14. **Implement `If-Modified-Since` handling** (304 Not Modified)
15. **Implement `If-Range` support** for partial content (206 responses)
16. **Add `Vary: Accept-Encoding` interaction test** when ETag + Compression are chained
17. **Test ETag + Compression interaction** — does compressed ETag match uncompressed If-None-Match? (The compress-then-hash ordering matters.)
18. **Add `httpspec` test for ETag correctness** — point `httpspec.Run` at an ETag-wrapped handler

### Performance (P2)

19. **Benchmark ETag generation with different hash functions** (FNV-64a vs xxHash vs SHA-256)
20. **Benchmark ETag + Compression chained** — hot path for production servers
21. **Profile ETag middleware allocation** — `body []byte` append/grow pattern
22. **Consider `sync.Pool` for `etagWriter.body` buffers** — reduce GC pressure on hot path
23. **Consider streaming hash** (hash-as-you-write) instead of buffering entire body

### Testing Hardening (P2)

24. **Add property-based test**: `etagInList` is commutative for weak/strong pairs
25. **Add property-based test**: `parseETagList` round-trips with `strings.Join`
26. **Add fuzz test for `parseETagList` specifically** — quote/comma/backslash combinations
27. **Add fuzz test for `stripWeakPrefix`** — ensure no panic on malformed input
28. **Add test: ETag for response > 1MB** — overflow-to-streaming path with If-None-Match
29. **Add test: ETag + Hijack** — verify ETag is not set after hijack
30. **Add test: HEAD request with If-None-Match** — should return 304 without body

### Documentation (P3)

31. **Document in AGENTS.md that ETag overwrites handler-set ETags**
32. **Document in AGENTS.md that ETag uses weak comparison for If-None-Match**
33. **Add ETag usage example** in `example_test.go` with `// Output:` directive
34. **Update `FEATURES.md`** — ETag should list "RFC 7232 weak comparison" as a feature
35. **Update D2 architecture diagram** if conditional-request scope expands

### Ecosystem Research (P3)

36. **Study Fiber's ETag middleware** for any techniques we're missing (skip logic for SSE, non-200, empty body)
37. **Study `blizzy78/conditional-http`** for If-Match/If-Modified-Since patterns we could adapt
38. **Check if `go-error-family` has conditional-request error classification patterns**

### Code Quality (P3)

39. **Consider extracting entity-tag parsing into a reusable `entitytag` subpackage** — `parseETagList`, `stripWeakPrefix`, `weakCompare`, `strongCompare` are general-purpose RFC 7232 primitives
40. **Add `ETagConfig.Validate()` test for `HashFunc == nil`** — should default to FNV-64a, not error
41. **Consider `Weak` validator semantics in `computeETag`** — should weak ETags use a different hash? (No — weakness is about comparison, not generation. But worth documenting.)
42. **Review error handling in `flush()` method** — `_, _ = w.ResponseWriter.Write(w.body)` silently ignores errors on the final body write
43. **Consider adding `ErrCodeETagComputeFailed`** for hash computation failures (currently impossible since FNV can't fail, but custom HashFunc could)

### Cross-Middleware (P3)

44. **Test ETag + CORS interaction** — does CORS Vary header affect ETag caching?
45. **Test ETag + RateLimit interaction** — should 429 responses get ETags? (No, but verify.)
46. **Test ETag + Recovery interaction** — does panic recovery bypass ETag generation?
47. **Test ETag + RequestID interaction** — does RequestID header affect ETag value? (It shouldn't since it's not in the body.)
48. **Test ETag + ServerTiming interaction** — Server-Timing header should not affect ETag body hash
49. **Test ETag + MaxBodySize interaction** — what happens when MaxBodySize rejects before ETag can compute?
50. **Test ETag + Decompression interaction** — decompressed body should get ETag, not compressed bytes

---

## g) Questions I Cannot Answer Myself

### Q1: Should the ETag middleware support `If-Match` / `If-Unmodified-Since` (412 Precondition Failed)?

The middleware is currently GET/HEAD-only. `If-Match` is for unsafe methods (PUT/DELETE/PATCH) to prevent lost updates. Adding support means either (a) widening the method filter to include unsafe methods for the precondition check only, or (b) creating a separate middleware. This is a **product decision** about the scope of this library — is it a caching library, a full conditional-request library, or both?

### Q2: Should we preserve handler-set ETags by default (`SkipIfPresent`)?

Currently the middleware always overwrites. Changing the default to `SkipIfPresent: true` would be more correct (handlers with domain knowledge should win) but would be a **breaking behavior change** for any consumer relying on the current transparent-overwrite behavior. This is a **compatibility decision** that affects v1.0 API stability.

### Q3: Is the `parseETagList` allocation regression acceptable, or should I optimize to zero-allocation now?

The old code was a zero-allocation string scan. The new code allocates a `[]string` slice. I haven't benchmarked the delta. If this library targets high-throughput production servers (it's a middleware library, so it does), even a small allocation on every cache-check request could matter at scale. But optimizing now without benchmark data would be premature. **Should I benchmark first, or optimize proactively?**
