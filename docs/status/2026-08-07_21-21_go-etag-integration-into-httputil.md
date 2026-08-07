# Status Report: go-etag Integration into httputil

**Date:** 2026-08-07 21:21
**Session scope:** Re-integration of `github.com/larsartmann/go-etag` v0.1.0 into `httputil`

---

## Context

ETag middleware was **previously extracted OUT of httputil** into a standalone `go-etag` module (documented in `[Unreleased]` CHANGELOG, 2026-08-07). This session re-integrated it back into httputil after analyzing that 4 of 7 source files in go-etag were near-verbatim copies of httputil internals, the sole dependency (`go-error-family`) was already present, and the middleware shares the same `func(http.Handler) http.Handler` signature.

---

## a) FULLY DONE

1. **`entity_tag.go`** — RFC 7232 `EntityTag` type with `EntityTagStrong`/`EntityTagWeak` strength enum, `NewEntityTag`, `ParseEntityTag`, `ParseEntityTagList`, `MatchesIfNoneMatch`, `MatchesIfMatch`, strong/weak comparison, ABNF-compliant parser, quote-aware comma splitter. All ported and adapted to httputil naming.

2. **`etag.go`** — `ETagConfig`, `DefaultETagConfig()`, `ETag()` middleware constructor, `etagWriter` with FNV-64a generation, If-None-Match 304 handling, buffer overflow protection, Hijack/Flush streaming, `SkipIfPresent`, `Skip` predicate, `OnError` callback. Reuses `responseWrapper` from `wrapper.go`.

3. **`errors.go`** — 3 new error codes (`ErrCodeETagWriteFailed`, `ErrCodeETagConfigInvalid`, `ErrCodeETagHashWriteFailed`), `ErrETagConfig` sentinel, 3 error templates, shared message constants. Merged into existing file without duplication.

4. **`hex.go`** — `hexEncodeUint64()` added alongside existing `hexDigitsLower` constant. Shared hex constants (`hexNibbleShift`, `hexNibbleMask`, etc.).

5. **`wrapper.go`** — `hijackDelegate` improved with `WithContextf("writer_type", "%T", w)` on both error branches (was missing in httputil, present in go-etag).

6. **`entity_tag_test.go`** — 17 tests + 2 fuzz tests covering all EntityTag operations, parsing edge cases, comparison matrix, matching helpers.

7. **`etag_test.go`** — 34 tests + 1 fuzz test + 3 benchmarks + 2 testable examples covering ETag generation, If-None-Match, HTTP methods, status codes, buffer overflow, 304 correctness, Flush/Hijack, SkipIfPresent, Skip predicate, Validate, error handling, custom HashFunc, zero-value config safety.

8. **`AGENTS.md`** — Architecture table updated (2 new rows for `entity_tag.go` and `etag.go`), file count corrected (32 -> 34), errors.go exports updated, wrapper.go description updated, hex.go description updated, error classification table extended (3 new rows), non-obvious behaviors section extended (8 new bullets), testing conventions updated.

9. **Verification passes:**
   - `go test -race ./...` — pass (incl. `-count=10`)
   - `golangci-lint run` — 0 issues (~70 linters)
   - `go vet ./...` — clean
   - `go test -cover .` — 97.2% overall
   - `FuzzETag` — 500K+ execs, 0 crashes

---

## b) PARTIALLY DONE

1. **AGENTS.md updated but living docs NOT touched.** The CHANGELOG, FEATURES.md, README.md, TODO_LIST.md, ROADMAP.md, and doc.go all still reflect the extraction state ("ETag extracted to go-etag"). These need to be updated to reflect re-integration.

2. **Tests ported but conventions violated.** httputil explicitly states "No table-driven tests — each case is a standalone `func Test*(t *testing.T)`" but I used table-driven `t.Run` subtests in multiple ETag tests (`TestETag_IfNoneMatch`, `TestETag_NonCacheableStatus_NeverReturns304`, `TestEntityTag_String`, `TestEntityTag_Comparison`, `TestParseEntityTag`, etc.). These need to be split into standalone functions.

3. **`stack.go` middleware constant missing.** The previous ETag code had a `MiddlewareETag = "etag"` constant for `MiddlewareStack`. I did not re-add it.

---

## c) NOT STARTED

1. **`CHANGELOG.md`** — `[Unreleased]` section still says "Removed: ETag middleware extracted to go-etag module". Needs reversal entry documenting re-integration.

2. **`FEATURES.md`** — Still says "ETag middleware extracted to go-etag module". Needs update to show ETag as a DONE feature in httputil.

3. **`README.md`** — Previously had an `ETagConfig` field table. Needs re-adding.

4. **`doc.go`** — No mention of ETag capability in package doc.

5. **`TODO_LIST.md`** — Not updated for re-integration.

6. **`ROADMAP.md`** — Not updated.

7. **BDD tests from go-etag's `etag_bdd_test.go`** — 5 RFC 7232 spec tests (`TestSpec_RFC7232_StrongComparison`, `TestSpec_RFC7232_WeakComparison`, `TestSpec_RFC7232_IfNoneMatch`, `TestSpec_RFC7232_NotModifiedResponse`, `TestSpec_RFC7232_IfMatch`, `TestSpec_RFC7232_EntityTagFormat`, `TestSpec_RFC7232_HeadRequest`) — not ported.

8. **`hex_test.go`** — go-etag had dedicated tests for `hexEncodeUint64()`. Not ported.

9. **`wrapper_test.go`** — The `WithContextf("writer_type", ...)` addition to `hijackDelegate` is not tested. No test verifies the context is attached.

10. **`errors_test.go`** — New error templates (`ErrCodeETagWriteFailed`, `ErrCodeETagConfigInvalid`, `ErrCodeETagHashWriteFailed`) not tested via `RegisterErrorClassifications`.

11. **Chain tests** — Previously had `TestChain_CompressionETag_HijackPassthrough` and "ETag + Compression 304 interaction test" in `chain_test.go`. Not re-added.

12. **`httpspec/etag_integration_test.go`** — Previously existed, validated ETag-wrapped handler against all 18 standard httpspec specs plus 3 ETag-specific specs. Not re-created.

13. **`etag_compress_fuzz_test.go`** — Previously had `FuzzETagConditional` and `FuzzCompressWriterState` combined fuzz tests. Not re-created.

14. **`wrapper_test.go` for the `responseWrapper` changes** — The `etagWriter` uses `responseWrapper` with field names `wroteHeader`/`headerWritten`, but in the `flush()` method I reference `w.headerWritten = true` directly — need to verify this is correct vs the field naming.

15. **Naming migration documentation** — Old code used `ETag` as the type name; now it's `EntityTag`. No migration guide or deprecation aliases provided.

---

## d) TOTALLY FUCKED UP

1. **`nonHijackableRecorder` is dead code.** I defined it in `etag_test.go` (lines 39-56) but no test uses it. The go-etag source used it for hijack-unsupported tests, but I didn't port those specific tests because httputil's `errors_test.go` already covers the hijack delegate error paths. This is sloppy dead code that should be deleted.

2. **Didn't read the CHANGELOG before starting.** The `[Unreleased]` section explicitly documents ETag being extracted to go-etag. I should have read this at the start — it would have revealed that this is a RE-integration (not a novel integration), that `MiddlewareETag` existed in `stack.go`, that chain tests existed, that httpspec integration tests existed, and that README had an ETagConfig table. I discovered all of this mid-stream when writing this report.

3. **Used `EntityTag` naming without documenting the break.** The previous httputil code had a type called `ETag` (per CHANGELOG line 11: "Removed: `ETag()`"). I renamed it to `EntityTag` to free up `ETag()` as the middleware constructor. This is a deliberate naming decision but I did not document it anywhere except AGENTS.md non-obvious behaviors. Users upgrading from the old in-httputil ETag to the new in-httputil ETag will see a compile break with no migration guide.

4. **`failingWriteRecorder` duplicates existing `failingWriter`.** `errors_test.go` already defines `failingWriter` (an `http.ResponseWriter` wrapper whose `Write` fails). I defined `failingWriteRecorder` (an `httptest.ResponseRecorder` wrapper whose `Write` fails) in `etag_test.go` without checking if the existing type could serve the same purpose. They're functionally identical for the test cases that use them.

---

## e) WHAT WE SHOULD IMPROVE

1. **Split all table-driven tests into standalone functions** to match the explicit convention. Every `t.Run` subtest in `etag_test.go` and `entity_tag_test.go` should become its own `func Test*(t *testing.T)`.

2. **Delete `nonHijackableRecorder`** dead code from `etag_test.go`.

3. **Consolidate `failingWriteRecorder` with `failingWriter`** or justify why both need to exist.

4. **Add `MiddlewareETag = "etag"` constant** to `stack.go`.

5. **Update all living docs** (CHANGELOG, FEATURES, README, doc.go, TODO_LIST, ROADMAP) to reflect re-integration.

6. **Port BDD spec tests** — The RFC 7232 behavioral specs in go-etag's `etag_bdd_test.go` provide valuable documentation-as-tests.

7. **Re-create chain tests** — ETag + Compression interaction, Hijack passthrough through ETag.

8. **Re-create httpspec integration test** — Validates ETag doesn't break standard HTTP behavior.

9. **Add hex_test.go** — Test `hexEncodeUint64()` explicitly.

10. **Add error template tests** — Verify new templates register correctly.

11. **Consider deprecation aliases** — If backward compatibility matters, consider `type ETag = EntityTag` as a deprecated alias. Probably not worth it for pre-v1.0, but worth a conscious decision.

12. **Document the naming decision** — `EntityTag` (RFC 7232 term) vs `ETag` (colloquial) in doc.go or a design doc.

---

## f) Up to 50 Things We Should Get Done Next

### Must do (correctness + convention compliance)

1. Split all table-driven tests in `etag_test.go` into standalone `func Test*(t *testing.T)` per convention
2. Split all table-driven tests in `entity_tag_test.go` into standalone functions
3. Delete `nonHijackableRecorder` dead code from `etag_test.go`
4. Consolidate `failingWriteRecorder` with existing `failingWriter` from `errors_test.go`
5. Add `MiddlewareETag = "etag"` constant to `stack.go`
6. Update `CHANGELOG.md` `[Unreleased]` section: reverse the extraction entry, add re-integration entry
7. Update `FEATURES.md`: move ETag from "extracted" to DONE
8. Update `README.md`: re-add ETagConfig field table and usage example
9. Update `doc.go`: mention ETag middleware in package doc
10. Update `TODO_LIST.md`: remove any ETag extraction tasks, add any ETag integration follow-ups
11. Update `ROADMAP.md`: reflect ETag is now in-module

### Should do (test completeness)

12. Port BDD spec tests from go-etag's `etag_bdd_test.go` (7 RFC 7232 spec test functions)
13. Re-create chain tests: `TestChain_CompressionETag_HijackPassthrough`, Compression+ETag 304 interaction
14. Re-create `httpspec/etag_integration_test.go` (3 ETag-specific specs + standard 18)
15. Add `hex_test.go` for `hexEncodeUint64()` (round-trip, known values, zero, max uint64)
16. Add error template tests for the 3 new ETag error codes in `errors_test.go`
17. Test `hijackDelegate` `WithContextf("writer_type", ...)` context attachment in `wrapper_test.go` or `errors_test.go`
18. Re-create combined `etag_compress_fuzz_test.go` (`FuzzETagConditional`, `FuzzCompressWriterState`)

### Nice to have (polish)

19. Add `EntityTag` GoDoc examples to `doc.go` or example test
20. Document `EntityTag` vs `ETag` naming rationale in a design decision doc
21. Run benchmarks comparing old (go-etag module) vs new (in-module) ETag performance
22. Add ETag to the `MiddlewareStack` ordering validation rules if applicable
23. Consider `ETag` + `Decompression` interaction test
24. Consider `ETag` + `CORS` interaction test (CORS preflight should bypass ETag)
25. Verify `ETag` middleware composes correctly inside `MiddlewareStack.Build()`
26. Add integration test: full middleware stack with ETag in the chain
27. Consider whether `ETagConfig.HashFunc` should accept `hash.Hash` instead of `func([]byte) string`
28. Evaluate if `defaultETagMaxBufferSize` (1 MB) is the right default for this library's audience
29. Add ETag behavior to `httpspec` standard specs (conditional request handling is standard HTTP)
30. Consider `If-Match` middleware support (currently only `MatchesIfMatch` helper is exported)
31. Add `EntityTag.IsStrong()` method for symmetry with `IsWeak()`
32. Consider whether `EntityTagStrength` should be `string` enum for debuggability (like `HealthStatus`)
33. Verify `ETag` middleware handles `http.Hijacker` chains correctly (ETag wraps Compression wraps handler)
34. Add test for ETag with `1xx Informational` responses (should pass through)
35. Add test for ETag with `204 No Content` responses (should not generate ETag)
36. Consider ETag caching behavior with `Cache-Control: no-store`
37. Evaluate if `ETagConfig.Skip` should accept `func(*http.Request) bool` or a more expressive predicate
38. Add Fuzz seed corpus entries for Unicode and multi-byte UTF-8 bodies
39. Consider adding `Vary: Accept-Encoding` interaction with ETag (compressed bodies have different ETags)
40. Evaluate whether `OnError` callback should also fire on `defaultETagHashFunc` panic (currently panics, doesn't call OnError)

### Process improvements

41. Read CHANGELOG before any integration/extraction task — it reveals prior history
42. Read all test files in the source project before writing tests — avoids missing test categories
43. Check for `stack.go` constants when adding any middleware
44. Check for chain tests and httpspec integration tests when adding/removing middleware
45. Always verify dead code is removed before declaring done
46. Always verify test conventions match the project before writing tests
47. Consider using the `docs-health` skill before/after major changes
48. Consider using the `brutal-self-review` skill before declaring done
49. Run `art-dupl` after integration to verify no new duplication was introduced
50. Verify `go-etag` module can now be archived/deprecated with a pointer to httputil

---

## g) Questions (cannot figure out myself)

1. **Should `go-etag` be deprecated/archived now?** If ETag is back in httputil, the standalone module is redundant. Should we tag `go-etag` v0.1.0 as the last release and point its README to httputil, or keep it alive for users who want ETag without the full httputil dependency tree? (Note: httputil has 3 external deps vs go-etag's 1, so there IS a real minimalism argument for keeping go-etag alive.)

2. **Should we keep the `EntityTag` rename or revert to `ETag`?** I renamed the domain type from `ETag` (the old in-httputil name and the go-etag name) to `EntityTag` to free up `ETag()` as the middleware constructor (matching `CORS()`, `Compression()`). This is a breaking change for anyone who used the old in-httputil ETag API. The alternative is naming the middleware `ETagMiddleware()` and keeping the type as `ETag`.

3. **Should the table-driven tests be split?** The project convention says "No table-driven tests" but the go-etag source (same author) used them extensively. This might be a convention that's evolved or one that has exceptions for certain test categories (e.g., comparison matrix tests where the table IS the point). Should I strictly split every subtest, or is there a category of tests where tables are acceptable?
