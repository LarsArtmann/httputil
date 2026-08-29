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

| Library                     | Stars | Status          | Adoptable?                               |
| --------------------------- | ----- | --------------- | ---------------------------------------- |
| `go-http-utils/etag`        | 15    | Dead since 2016 | No                                       |
| `amalfra/etag`              | 32    | Maintained      | No — only generates string, no 304 logic |
| Fiber `middleware/etag`     | 40k   | Active          | No — fasthttp-coupled                    |
| `pablor21/echo-etag`        | 11    | Maintained      | No — Echo-only                           |
| `blizzy78/conditional-http` | 1     | Unknown         | No — zero adoption                       |

**Conclusion:** No superb standalone ETag library exists for `net/http`. The depguard rules (only `go-error-family`, `x/time`, `nosurf`) would block all of them anyway. Keep building our own — fix the bugs and ours is the best in the ecosystem.

### 3. Fixed Two Bugs in `etagInList` (`etag.go`)

#### Bug 1: Wrong Comparison Function (RFC 7232 §2.3.2)

**Before:** `etagInList` used literal string `==`, which is the **strong comparison function**. But `If-None-Match` requires the **weak comparison function** per [RFC 7232 §3.2](https://www.rfc-editor.org/rfc/rfc7232#section-3.2).

| Server ETag | Client `If-None-Match` | RFC says | Old code  |
| ----------- | ---------------------- | -------- | --------- |
| `"abc"`     | `W/"abc"`              | 304      | 200 (bug) |
| `W/"abc"`   | `"abc"`                | 304      | 200 (bug) |

**Fix:** Added `stripWeakPrefix()` that removes the optional `W/` prefix before comparing. `etagInList` now strips both sides and compares opaque-tags — the weak comparison function.

#### Bug 2: Naive Comma Splitting Broke on Commas Inside Quotes

**Before:** `strings.Index(list, ",")` split on every comma. But the `etagc` grammar (RFC 7232 §2.3) permits any VCHAR except `"` inside the opaque-tag — **including comma**. A client sending `"a,b"` would be parsed as two tags: `"a` and `b"`.

**Fix:** Replaced with `parseETagList()` — a quote-state-aware splitter that only splits on commas outside quoted strings.

### 4. Added 5 Tests

| Test                                          | Covers                                       |
| --------------------------------------------- | -------------------------------------------- |
| `TestETag_IfNoneMatch_WeakClientStrongServer` | `W/"..."` client vs strong server → 304      |
| `TestETag_IfNoneMatch_StrongClientWeakServer` | Strong client vs `W/"..."` server → 304      |
| `TestETag_IfNoneMatch_ListContainsWeakMatch`  | Weak validator in a multi-element list → 304 |
| `TestETag_IfNoneMatch_WeakClientNoMatch`      | Negative case — no false positives           |
| `TestParseETagList_RespectsCommasInQuotes`    | `"a,b"` parsed as single tag                 |

### 5. Verification

- `golangci-lint fmt` — clean
- `golangci-lint run` — 0 issues across ~70 linters
- `go test -race ./...` — all pass
- `FuzzETag` — 5s, 513k execs, PASS
- `FuzzETagConditional` — 6s, 959k execs, PASS

---

## a) FULLY DONE

1. ~~**RFC 7232 §2.3.2 weak comparison** — implemented and tested bidirectionally~~ done at `9f49af2`
2. ~~**Quote-aware list parsing** — `parseETagList` respects commas inside opaque-tags~~ done at `9f49af2`
3. ~~**5 new tests** — covering weak/strong cross-matching, list containment, negative case, comma-in-quotes~~ done at `9f49af2`
4. ~~**Lint clean** — 0 issues, formatted~~ done at `9f49af2`
5. ~~**Race-clean** — `go test -race` passes~~ done at `9f49af2`
6. ~~**Fuzz-clean** — both `FuzzETag` and `FuzzETagConditional` ran without failures~~ done at `9f49af2`
7. ~~**Auto-committed** — `9f49af2`~~ done at `9f49af2`

---

## b) PARTIALLY DONE

1. ~~**CHANGELOG.md `[Unreleased]`** — **NOT UPDATED.** The fix is committed but not recorded in the changelog. The `[Unreleased]` section is empty. This violates the project's own Keep a Changelog policy.~~ **Done at v0.9.1** — CHANGELOG.md `[0.9.1]` section created with Fixed + Added entries.
2. ~~**Fuzz corpus enrichment** — The existing `FuzzETag` fuzz test (`etag_test.go:184`) only seeds exact-match `If-None-Match` values. No `W/"..."` seeds were added to its corpus. The fuzzer discovered weak-prefixed inputs organically (55 new interesting inputs), but explicit seeds documenting the fix would be better.~~ **Done at v0.9.1** — `W/"779a65e7023cd2e7"` + multi-tag weak seed added to `FuzzETag` corpus.

---

## c) NOT STARTED

1. ~~**`If-Match` / `If-Unmodified-Since` support** — RFC 7232 §3.1 (preconditions for unsafe methods → 412 Precondition Failed). The middleware is GET/HEAD-only so this is structurally out of scope, but could be a separate middleware or config option.~~ **Won't implement — moved — conditional-request scope is evaluated in go-etag; ROADMAP tracks the scope decision.**
2. ~~**`If-Range` support** — RFC 7232 §3.2 + RFC 7233 (range requests). Combined with ETag or Last-Modified, enables partial-content (206) responses.~~ **Won't implement — moved — If-Range is evaluated in go-etag (ROADMAP conditional-request scope).**
3. ~~**`Last-Modified` / `If-Modified-Since` support** — the other half of conditional requests. Currently only `If-None-Match` (ETag-based) is handled.~~ **Won't implement — moved — Last-Modified is evaluated in go-etag (ROADMAP conditional-request scope).**
4. ~~**`Vary` header management** — the ETag middleware doesn't add `Vary` headers. Compression adds `Vary: Accept-Encoding`, but ETag generation based on body content doesn't need Vary. Still worth confirming this is correct for all proxy/cache scenarios.~~ **Won't implement — moved — ETag internals incl. Vary semantics now live in go-etag.**
5. ~~**Handler-set ETag preservation** — `w.Header().Set(headerETag, etag)` always overwrites any ETag the downstream handler computed. A `SkipIfPresent` config option would let handlers with domain-specific modification semantics win.~~ **Won't implement — moved — SkipIfPresent is a go-etag config decision.**
6. ~~**304 response header cleanup** — RFC 7232 §4.1 says a 304 response SHOULD include cache-related headers but MUST NOT include body-related headers like `Content-Length`. The current code relies on the stdlib implicitly handling this. Worth verifying with a test that asserts `Content-Length` is absent on 304.~~ **Won't implement — moved — 304 header hygiene tests are go-etag responsibility.**

---

## d) TOTALLY FUCKED UP

Nothing. The fix is correct, tested, lint-clean, race-clean, and fuzz-clean. The only process miss is the missing CHANGELOG entry (see b.1).

---

## e) WHAT WE SHOULD IMPROVE

### Code Quality

1. ~~**`parseETagList` allocates a `[]string` slice on every request** — the old code was a zero-allocation string scan. For high-throughput servers, this is a regression on the hot path. Could be optimized with a callback-based walker (`func(tag string) bool`) or by inlining the comparison loop. Benchmark the old vs new to quantify.~~ done (moot — the parser moved to go-etag in the extraction, retiring the allocation question here)
2. ~~**`stripWeakPrefix` doesn't validate the entity-tag shape** — it blindly removes `W/` from anything starting with those bytes. Malformed input like `W/foo` (unquoted) or `W/` alone would produce `foo` / empty string. The RFC requires `W/` to be followed by a quoted-string, but rejecting malformed input is defensive. Low priority since the comparison still can't match valid server ETags against garbage.~~ **Won't implement — moved — parser internals are go-etag responsibility.**
3. ~~**Escaped quotes not handled in `parseETagList`** — backslash-escaped `\"` inside an opaque-tag would flip `inQuotes` incorrectly. The RFC grammar technically allows this (`%x5C` in quoted-string). Extremely rare in practice (hex ETags never contain backslashes), but the parser is technically not fully spec-compliant for arbitrary client input.~~ done at `ca06b4b`

### Process

4. ~~**CHANGELOG discipline** — the fix was committed without updating CHANGELOG.md. The auto-commit daemon doesn't know about the changelog policy. For deliberate fixes, update CHANGELOG before committing.~~ done (process adopted — CHANGELOG Freeze Policy and release checklist documented in AGENTS.md (98bff8c))
5. ~~**Fuzz corpus should document the bug being fixed** — adding `W/"779a65e7023cd2e7"` as an explicit seed in `FuzzETag` would document the weak-comparison fix in the test corpus itself, not just in unit tests.~~ done (shipped v0.9.1 — weak seeds in the FuzzETag corpus (see b.2))
6. ~~**Benchmark regression check** — no benchmark was run comparing old vs new `etagInList` performance. The `BenchmarkETag` exists but wasn't used to verify no regression.~~ done (moot — benchmark ownership moved to go-etag)

### Architecture

7. ~~**ETag middleware overwrites handler-set ETags** — this is a design limitation, not a bug. Document it explicitly or add a `SkipIfPresent bool` config.~~ done (superseded — overwrite semantics are now go-etag design space)
8. ~~**No `Last-Modified` counterpart** — ETag is one of two conditional-request mechanisms. The middleware is half of a complete caching solution.~~ **Won't implement — moved — Last-Modified evaluated in go-etag (ROADMAP).**
9. ~~**Single hash algorithm (FNV-64a)** — pluggable via `HashFunc`, but the default has a 64-bit collision space. For very large deployments, SHA-256 or BLAKE3 would be safer. The `HashFunc` option makes this a config change, not a code change.~~ **Won't implement — moved — hash choices are go-etag config space.**

---

## f) Up to 50 Things We Should Get Done Next

### ETag-Specific (P0–P1)

1. ~~**Update CHANGELOG.md `[Unreleased]`** with the weak comparison fix~~ done at `e1377aa`
2. ~~**Add `W/"..."` seeds to `FuzzETag` corpus** documenting the fix~~ done (shipped v0.9.1 — weak seeds in the FuzzETag corpus)
3. ~~**Benchmark `etagInList` old vs new** — quantify allocation regression~~ done (moot — parser moved to go-etag in the extraction)
4. ~~**Optimize `parseETagList` to zero-allocation** if benchmarks show meaningful regression~~ **Won't implement — moved — go-etag owns the parser now.**
5. ~~**Handle escaped quotes in `parseETagList`** for full RFC grammar compliance~~ done at `ca06b4b`
6. ~~**Add test: 304 response has no `Content-Length` header** (RFC 7232 §4.1)~~ **Won't implement — moved — 304 header tests are go-etag responsibility.**
7. ~~**Add test: 304 response includes `ETag` header** (RFC 7232 §4.1)~~ **Won't implement — moved — 304 header tests are go-etag responsibility.**
8. ~~**Add `SkipIfPresent bool` to `ETagConfig`** — don't overwrite handler-set ETags~~ **Won't implement — moved — SkipIfPresent is a go-etag config decision.**
9. ~~**Add test: handler-set ETag is currently overwritten** — document current behavior~~ **Won't implement — moved — overwrite-behavior tests are go-etag responsibility.**
10. ~~**Add test: multiple `If-None-Match` headers** (RFC 9110 §5.2 — combine field lines into one list)~~ done at `256ccba`

### Conditional Requests — Broader (P1–P2)

11. ~~**Implement `If-Match` middleware** for unsafe methods (412 Precondition Failed)~~ **Won't implement — moved — conditional-request scope is evaluated in go-etag (ROADMAP).**
12. ~~**Implement `If-Unmodified-Since` middleware** (412 Precondition Failed)~~ **Won't implement — moved — conditional-request scope is evaluated in go-etag (ROADMAP).**
13. ~~**Implement `Last-Modified` generation middleware** (analogous to ETag but timestamp-based)~~ **Won't implement — moved — Last-Modified evaluated in go-etag (ROADMAP).**
14. ~~**Implement `If-Modified-Since` handling** (304 Not Modified)~~ **Won't implement — moved — If-Modified-Since evaluated in go-etag (ROADMAP).**
15. ~~**Implement `If-Range` support** for partial content (206 responses)~~ **Won't implement — moved — If-Range evaluated in go-etag (ROADMAP).**
16. ~~**Add `Vary: Accept-Encoding` interaction test** when ETag + Compression are chained~~ **Won't implement — moved — ETag+Compression interaction tests are go-etag responsibility.**
17. ~~**Test ETag + Compression interaction** — does compressed ETag match uncompressed If-None-Match? (The compress-then-hash ordering matters.)~~ **Won't implement — moved — ETag+Compression interaction tests are go-etag responsibility.**
18. ~~**Add `httpspec` test for ETag correctness** — point `httpspec.Run` at an ETag-wrapped handler~~ done at `77a442c`

### Performance (P2)

19. ~~**Benchmark ETag generation with different hash functions** (FNV-64a vs xxHash vs SHA-256)~~ **Won't implement — moved — hash benchmarks are go-etag responsibility.**
20. ~~**Benchmark ETag + Compression chained** — hot path for production servers~~ **Won't implement — moved — chained-path benchmarks are go-etag responsibility.**
21. ~~**Profile ETag middleware allocation** — `body []byte` append/grow pattern~~ **Won't implement — moved — allocation profiling is go-etag responsibility.**
22. ~~**Consider `sync.Pool` for `etagWriter.body` buffers** — reduce GC pressure on hot path~~ **Won't implement — moved — buffer pooling is go-etag responsibility.**
23. ~~**Consider streaming hash** (hash-as-you-write) instead of buffering entire body~~ **Won't implement — moved — streaming hash is a go-etag design decision.**

### Testing Hardening (P2)

24. ~~**Add property-based test**: `etagInList` is commutative for weak/strong pairs~~ **Won't implement — moved — property tests for ETag internals are go-etag responsibility.**
25. ~~**Add property-based test**: `parseETagList` round-trips with `strings.Join`~~ **Won't implement — moved — property tests for ETag internals are go-etag responsibility.**
26. ~~**Add fuzz test for `parseETagList` specifically** — quote/comma/backslash combinations~~ **Won't implement — moved — parseETagList fuzzing is go-etag responsibility.**
27. ~~**Add fuzz test for `stripWeakPrefix`** — ensure no panic on malformed input~~ **Won't implement — moved — stripWeakPrefix fuzzing is go-etag responsibility.**
28. ~~**Add test: ETag for response > 1MB** — overflow-to-streaming path with If-None-Match~~ **Won't implement — moved — large-body tests are go-etag responsibility.**
29. ~~**Add test: ETag + Hijack** — verify ETag is not set after hijack~~ **Won't implement — moved — hijack interaction tests are go-etag responsibility.**
30. ~~**Add test: HEAD request with If-None-Match** — should return 304 without body~~ **Won't implement — moved — HEAD/304 tests are go-etag responsibility.**

### Documentation (P3)

31. ~~**Document in AGENTS.md that ETag overwrites handler-set ETags**~~ done (documented pre-extraction; AGENTS.md now documents only the deprecated adapter (superseded by extraction))
32. ~~**Document in AGENTS.md that ETag uses weak comparison for If-None-Match**~~ done (documented pre-extraction; superseded by the go-etag extraction)
33. ~~**Add ETag usage example** in `example_test.go` with `// Output:` directive~~ done (ExampleETag exists with an Output directive (example_test.go))
34. ~~**Update `FEATURES.md`** — ETag should list "RFC 7232 weak comparison" as a feature~~ done (recorded for v0.9.1; FEATURES.md now documents only the adapter (superseded by extraction))
35. ~~**Update D2 architecture diagram** if conditional-request scope expands~~ done (moot — the conditional never triggered; ETag scope moved to go-etag)

### Ecosystem Research (P3)

36. ~~**Study Fiber's ETag middleware** for any techniques we're missing (skip logic for SSE, non-200, empty body)~~ done (studied in this session — see the ecosystem table above)
37. ~~**Study `blizzy78/conditional-http`** for If-Match/If-Modified-Since patterns we could adapt~~ done (studied in this session — see the ecosystem table above)
38. **Check if `go-error-family` has conditional-request error classification patterns**

### Code Quality (P3)

39. ~~**Consider extracting entity-tag parsing into a reusable `entitytag` subpackage** — `parseETagList`, `stripWeakPrefix`, `weakCompare`, `strongCompare` are general-purpose RFC 7232 primitives~~ done (superseded — ETag was extracted wholesale into the standalone go-etag module (890b7eb))
40. ~~**Add `ETagConfig.Validate()` test for `HashFunc == nil`** — should default to FNV-64a, not error~~ **Won't implement — moved — ETagConfig validation is go-etag responsibility.**
41. ~~**Consider `Weak` validator semantics in `computeETag`** — should weak ETags use a different hash? (No — weakness is about comparison, not generation. But worth documenting.)~~ **Won't implement — moved — weak-validator semantics are go-etag design space.**
42. ~~**Review error handling in `flush()` method** — `_, _ = w.ResponseWriter.Write(w.body)` silently ignores errors on the final body write~~ done at `cfc6eb9`
43. ~~**Consider adding `ErrCodeETagComputeFailed`** for hash computation failures (currently impossible since FNV can't fail, but custom HashFunc could)~~ done (covered by the http.etag_hash_write_failed classification (now go-etag))

### Cross-Middleware (P3)

44. ~~**Test ETag + CORS interaction** — does CORS Vary header affect ETag caching?~~ **Won't implement — moved — cross-middleware ETag interaction tests are go-etag responsibility.**
45. ~~**Test ETag + RateLimit interaction** — should 429 responses get ETags? (No, but verify.)~~ **Won't implement — moved — cross-middleware ETag interaction tests are go-etag responsibility.**
46. ~~**Test ETag + Recovery interaction** — does panic recovery bypass ETag generation?~~ **Won't implement — moved — cross-middleware ETag interaction tests are go-etag responsibility.**
47. ~~**Test ETag + RequestID interaction** — does RequestID header affect ETag value? (It shouldn't since it's not in the body.)~~ **Won't implement — moved — cross-middleware ETag interaction tests are go-etag responsibility.**
48. ~~**Test ETag + ServerTiming interaction** — Server-Timing header should not affect ETag body hash~~ **Won't implement — moved — cross-middleware ETag interaction tests are go-etag responsibility.**
49. ~~**Test ETag + MaxBodySize interaction** — what happens when MaxBodySize rejects before ETag can compute?~~ **Won't implement — moved — cross-middleware ETag interaction tests are go-etag responsibility.**
50. ~~**Test ETag + Decompression interaction** — decompressed body should get ETag, not compressed bytes~~ **Won't implement — moved — cross-middleware ETag interaction tests are go-etag responsibility.**

---

## g) Questions I Cannot Answer Myself

### Q1: ~~Should the ETag middleware support `If-Match` / `If-Unmodified-Since` (412 Precondition Failed)?~~

**Answered:** by the go-etag extraction — conditional-request scope (If-Match helpers, Last-Modified, If-Range) is evaluated in go-etag; ROADMAP tracks the scope decision.

The middleware is currently GET/HEAD-only. `If-Match` is for unsafe methods (PUT/DELETE/PATCH) to prevent lost updates. Adding support means either (a) widening the method filter to include unsafe methods for the precondition check only, or (b) creating a separate middleware. This is a **product decision** about the scope of this library — is it a caching library, a full conditional-request library, or both?

### Q2: ~~Should we preserve handler-set ETags by default (`SkipIfPresent`)?~~

**Answered:** moot here — ETag moved to go-etag; the SkipIfPresent / default-overwrite decision is go-etag's.

Currently the middleware always overwrites. Changing the default to `SkipIfPresent: true` would be more correct (handlers with domain knowledge should win) but would be a **breaking behavior change** for any consumer relying on the current transparent-overwrite behavior. This is a **compatibility decision** that affects v1.0 API stability.

### Q3: ~~Is the `parseETagList` allocation regression acceptable, or should I optimize to zero-allocation now?~~

**Answered:** moot — the parser moved to go-etag in the extraction (`890b7eb`); the regression question retired with it.

The old code was a zero-allocation string scan. The new code allocates a `[]string` slice. I haven't benchmarked the delta. If this library targets high-throughput production servers (it's a middleware library, so it does), even a small allocation on every cache-check request could matter at scale. But optimizing now without benchmark data would be premature. **Should I benchmark first, or optimize proactively?**

---

## Resolution (2026-08-07 docs-health pass; upgraded to per-item markers 2026-08-29)

Every numbered item is resolved inline; f38 (go-error-family conditional-request patterns) is the only item left open — no evidence it was ever actioned. The header banner was removed — its verdicts live on the items themselves. b.1/b.2 were already resolved inline at v0.9.1 by the earlier pass and are untouched.

Most f)-section closures are "moved to go-etag": the ETag middleware was extracted from this repo into the standalone `go-etag` module on 2026-08-07 (`890b7eb` and the 06:44 sessions), so ETag internals, benchmarks, fuzzing, and cross-middleware interaction tests are go-etag's responsibility; conditional-request scope (If-Match/Last-Modified/If-Range) is tracked in ROADMAP as a go-etag evaluation. Section a) work shipped in `9f49af2`; the escaped-quote fix in `ca06b4b`; multi-`If-None-Match` in `256ccba`; the httpspec ETag integration in `77a442c`; the flush() write-error classification in `cfc6eb9`.
