# Status Report: ETag Re-Integration — Self-Critique and Gap Analysis

**Date:** 2026-08-07 21:59
**Session scope:** Completing the self-critique backlog from the go-etag → httputil re-integration
**Verdict:** Functional code is solid (0 lint, race-clean, 97.2% coverage), but **documentation and test depth have significant gaps** that were missed.

---

## ❗❗❗ CRITICAL FUNDAMENTAL MISUNDERSTANDING — READ THIS FIRST ❗❗❗

**The entire premise of this session's work was WRONG.**

The user extracted ETag into `go-etag` **yesterday** as an **intentional, independent module**.
The goal was to **keep go-etag independent** while **integrating it well** into httputil.

**What I did instead:** I copied ALL of go-etag's code back into httputil — `etag.go`,
`entity_tag.go`, error codes, `hexEncodeUint64`, tests, everything. This creates a **split
brain**: the same code now exists in both modules with no link between them. This is the
exact opposite of what the user wanted.

**The correct approach was:** Add `github.com/larsartmann/go-etag` as a dependency in
httputil's `go.mod` (like `justinas/nosurf` or `golang.org/x/time`), then write a thin
adapter in httputil that wraps go-etag's middleware to fit httputil's `Middleware` type
and error classification system. The go-etag module stays independent, self-contained,
and separately versioned. httputil gets ETag functionality without duplicating code.

**All work below this section was done on a WRONG premise. The code changes need to be
REVERTED and redone as a thin adapter. The documentation changes need to be REVERTED
and redone to reflect go-etag-as-dependency, not re-integrated code.**

### What needs to happen to fix this:

1. ~~Revert all code copied from go-etag: `etag.go`, `entity_tag.go`, `hexEncodeUint64` in~~ done (done — executed by the 22:22 session (the revert))
   ~~`hex.go`, ETag error codes/templates in `errors.go`, WithContextf in `wrapper.go`,~~
   ~~`MiddlewareETag` in `stack.go`~~
2. ~~Revert all test files I created/modified: `etag_test.go`, `entity_tag_test.go`,~~ done (done — executed by the 22:22 session (the revert))
   ~~`hex_test.go`, `chain_test.go` chain tests~~
3. ~~Revert all doc changes: `CHANGELOG.md`, `FEATURES.md`, `README.md`, `doc.go`,~~ done (done — executed by the 22:22 session (the revert))
   ~~`TODO_LIST.md`, `ROADMAP.md`~~
4. ~~Add `github.com/larsartmann/go-etag` as a dependency in `go.mod`~~ done (done — go-etag added as a dependency (e043a33, cc6439e))
5. ~~Write a thin adapter in httputil (e.g., `etag_adapter.go`) that:~~ done (done — the thin adapter landed (etag.go, e043a33))
   ~~- Wraps `etag.New()` to return httputil's `Middleware` type~~
   ~~- Maps go-etag's error codes to httputil's error classification system~~
   ~~- Adds `MiddlewareETag` constant for `MiddlewareStack`~~
6. ~~Write integration tests verifying the adapter works~~ done (done — adapter and chain tests landed (e043a33, 58ed5e1, dda97d1))
7. ~~Update docs to reflect go-etag as a dependency (like nosurf), not re-integrated code~~ done (done — docs updated to the dependency model (5c5a2a9, 3dd096d))

---

## A) FULLY DONE (verified this session)

### Code quality fixes — all verified

1. ~~**`MiddlewareETag = "etag"` added to `stack.go`** — was removed during extraction, re-added. (`stack.go:23`)~~ **Won't implement — reverted — the session was rolled back at 22:22; the work was redone via the thin-adapter approach (see the 22:22 report).**
2. ~~**Dead code deleted** — `nonHijackableRecorder` (22 lines, defined but never used) removed from `etag_test.go`.~~ **Won't implement — reverted — the session was rolled back at 22:22; the work was redone via the thin-adapter approach (see the 22:22 report).**
3. ~~**Duplicate test helper consolidated** — `failingWriteRecorder` (same purpose as `failingWriter` in `errors_test.go`) deleted; all 4 call sites updated to use the shared `failingWriter`.~~ **Won't implement — reverted — the session was rolled back at 22:22; the work was redone via the thin-adapter approach (see the 22:22 report).**
4. ~~**All table-driven tests split into standalone `func Test*` functions** — per httputil's "No table-driven tests" convention:~~ **Won't implement — reverted — the session was rolled back at 22:22; the work was redone via the thin-adapter approach (see the 22:22 report).**
   ~~- `entity_tag_test.go`: 6 table-driven functions → 39 standalone functions (full rewrite)~~
   ~~- `etag_test.go`: 2 table-driven functions → 14 standalone functions~~
5. ~~**Chain tests re-created** (`chain_test.go`) — 3 new tests:~~ **Won't implement — reverted — the session was rolled back at 22:22; the work was redone via the thin-adapter approach (see the 22:22 report).**
   ~~- `TestChain_CompressionETag_Matching304` — 304 through Compression+ETag excludes Content-Encoding, includes ETag~~
   ~~- `TestChain_CompressionETag_NoMatch200` — 200 through Compression+ETag includes both Content-Encoding and ETag~~
   ~~- `TestChain_CompressionETag_HijackPassthrough` — Hijack through both middleware delegates correctly~~
6. ~~**`hex_test.go` created** — 5 tests for `hexEncodeUint64()`: zero, max uint64, known FNV-64a value, small value, 16-char length invariant.~~ **Won't implement — reverted — the session was rolled back at 22:22; the work was redone via the thin-adapter approach (see the 22:22 report).**

### Living docs updated

7. ~~**`CHANGELOG.md` [Unreleased]** — reversed extraction entry to re-integration; added chain tests, hex_test.go, and all new ETag work.~~ **Won't implement — reverted — the session was rolled back at 22:22; the work was redone via the thin-adapter approach (see the 22:22 report).**
8. ~~**`FEATURES.md`** — middleware count 16→17, ETag row added to table, error codes 4→7, Middleware constants 11→12, new ETag feature section, fuzz tests 19→22, benchmarks 41→44, examples 23→25, wrapper.go description updated.~~ **Won't implement — reverted — the session was rolled back at 22:22; the work was redone via the thin-adapter approach (see the 22:22 report).**
9. ~~**`README.md`** — ETag in description, feature section with usage examples, `ETagConfig` fields table, 7 new API table entries, 3 new error classification table rows.~~ **Won't implement — reverted — the session was rolled back at 22:22; the work was redone via the thin-adapter approach (see the 22:22 report).**
10. ~~**`doc.go`** — ETag added to package doc.~~ **Won't implement — reverted — the session was rolled back at 22:22; the work was redone via the thin-adapter approach (see the 22:22 report).**
11. ~~**`TODO_LIST.md`** — timestamp/context updated.~~ **Won't implement — reverted — the session was rolled back at 22:22; the work was redone via the thin-adapter approach (see the 22:22 report).**
12. ~~**`ROADMAP.md`** — conditional-request scope and streaming ETag non-goal updated.~~ **Won't implement — reverted — the session was rolled back at 22:22; the work was redone via the thin-adapter approach (see the 22:22 report).**

### Verification (all passing)

- `golangci-lint run` — 0 issues (~70 linters)
- `go test -race -count=10 ./...` — pass
- `go test -race -cover` — 97.2% httputil / 99.3% httpspec
- `go vet ./...` — clean
- All 3 ETag fuzz tests (`FuzzETag`, `FuzzParseEntityTag`, `FuzzParseEntityTagList`) — 0 crashes, 500K+ execs each
- `server_timing` sub-module — tests + lint clean

---

## B) PARTIALLY DONE

### Nothing is partially done — everything I touched is either complete or untouched.

---

## C) NOT STARTED (identified but not executed this session)

### Missing test files

1. ~~**BDD spec tests NOT ported** — go-etag's `etag_bdd_test.go` (334 lines, 7 RFC 7232 spec test functions: strong comparison, weak comparison, If-None-Match, NotModified response, If-Match, EntityTag format, HEAD request) were listed as "Should do" in the context summary but I skipped them entirely. These test RFC compliance from a behavioral angle distinct from the unit tests.~~ **Won't implement — superseded — the revert made these moot; the adapter integration re-landed the equivalents (see the 22:22 report).**
2. ~~**`httpspec/etag_integration_test.go` NOT re-created** — Was deleted during extraction. Should validate an ETag-wrapped handler passes all 18 standard httpspec specs plus 3 ETag-specific specs.~~ **Won't implement — superseded — the revert made these moot; the adapter integration re-landed the equivalents (see the 22:22 report).**
3. ~~**`etag_compress_fuzz_test.go` NOT re-created** — `FuzzETagConditional` and `FuzzCompressWriterState` were deleted during extraction and not ported back.~~ **Won't implement — superseded — the revert made these moot; the adapter integration re-landed the equivalents (see the 22:22 report).**
4. ~~**Error template tests NOT added** — No tests verify the 3 new ETag error templates (`ErrCodeETagWriteFailed`, `ErrCodeETagConfigInvalid`, `ErrCodeETagHashWriteFailed`) render correctly via `errorfamily.RenderTemplate`.~~ **Won't implement — superseded — the revert made these moot; the adapter integration re-landed the equivalents (see the 22:22 report).**
5. ~~**`hijackDelegate` WithContextf test NOT added** — I added `.WithContextf("writer_type", "%T", w)` to both error branches in `wrapper.go` but did not write a test verifying the context is attached.~~ **Won't implement — superseded — the revert made these moot; the adapter integration re-landed the equivalents (see the 22:22 report).**

### Missing documentation updates

6. ~~**`docs/v1-stability.md` NOT updated** — This is the biggest miss. ETag types were removed from the v1.0 frozen API surface tables during extraction and NEVER re-added. Missing entries: `ETagConfig` (config types table), `DefaultETagConfig` (constructors table), `ETag` (middleware factories table), `EntityTag`/`EntityTagStrength`/`NewEntityTag`/`ParseEntityTag`/`ParseEntityTagList`/`MatchesIfNoneMatch`/`MatchesIfMatch` (domain types table), `MiddlewareETag` (constants count), 3 error codes, `ErrETagConfig` sentinel.~~ **Won't implement — superseded — the revert made these moot; the adapter integration re-landed the equivalents (see the 22:22 report).**
7. ~~**D2 architecture diagram NOT updated** — `docs/architecture-understanding/2026-08-05_httputil-current.d2` still says "Middleware Chain (16)" and has no ETag node. Should be 17 with an ETag node.~~ **Won't implement — superseded — the revert made these moot; the adapter integration re-landed the equivalents (see the 22:22 report).**
8. ~~**`docs/DOMAIN_LANGUAGE.md` NOT updated** — 0 ETag/EntityTag entries. Should have a Conditional Requests bounded context with entity-tag, strength, If-None-Match, If-Match terms.~~ **Won't implement — superseded — the revert made these moot; the adapter integration re-landed the equivalents (see the 22:22 report).**
9. ~~**`CONTRIBUTING.md` NOT checked** — May reference ETag extraction or stale allowed-dependency list.~~ **Won't implement — superseded — the revert made these moot; the adapter integration re-landed the equivalents (see the 22:22 report).**

---

## D) TOTALLY FUCKED UP

### Nothing is broken — all code compiles, tests pass, lint is clean.

However, two things I should be honest about:

1. **I almost broke the build by eating the `BenchmarkChain` function declaration** during the chain_test.go edit. The edit replaced the closing `}` of the last chain test with the benchmark body, leaving the benchmark function without a signature. I caught it on the next test run and fixed it, but it was a sloppy edit that should have been caught by reading the edit result more carefully before testing.

2. **The `entity_tag_test.go` rewrite had a duplicate entry in the CHANGELOG** — I wrote `hex_test.go` twice in the Added section. I caught and fixed it immediately, but it shows I wasn't proofreading my CHANGELOG entry carefully.

---

## E) WHAT WE SHOULD IMPROVE

### Process improvements

1. **Read `docs/v1-stability.md` FIRST** — This is the v1.0 API freeze document. Any time exported symbols are added or removed, this file MUST be updated. I updated CHANGELOG, FEATURES, README, doc.go, TODO_LIST, ROADMAP — but completely missed v1-stability.md. This is the file that actually defines the contractual API surface for consumers.
2. **Port BDD tests when porting functional code** — The BDD spec tests from go-etag test RFC 7232 compliance from a different angle than unit tests. Skipping them creates a test coverage gap that's invisible in coverage percentages (97.2%!) but real in behavioral coverage.
3. **Check for deleted files that should be re-created** — The extraction deleted `etag_compress_fuzz_test.go`, `httpspec/etag_integration_test.go`, and `websocket_upgrade_test.go`. I re-created chain tests but missed these. A `git log --diff-filter=D --name-only` would have surfaced them.
4. **Stale LSP diagnostics** — The LSP reports `entity_tag.go:33 EntityTagStrength.valid is unused` and `assertBodyEmpty is unused` but both are actively used (the former in `etag.go:107`, the latter in 11 test call sites). The LSP cache is stale from the session's file changes. A `lsp_restart` would clear this.

### Code improvements

5. **The 7 non-cacheable status tests are repetitive** — `TestETag_NonCacheable_301MovedPermanently` through `TestETag_NonCacheable_503ServiceUnavailable` are structurally identical (only the status code differs). The httputil convention says "no table-driven tests" but 7 copies of the same test body is arguably worse. A shared helper function that takes `(t, status)` would reduce the repetition while keeping each test standalone.
6. **`serveGetWithIfNoneMatch` helper uses `newWriteStatusHandler` hardcoded to "hello world"** — This is fine for the If-None-Match tests but couples the helper to a specific body. If the body changes, all ETag hash values change.

---

## F) Up to 50 things to get done next

### High priority — correctness and API surface

1. ~~**Update `docs/v1-stability.md`** — Add `ETagConfig`, `DefaultETagConfig`, `ETag`, `EntityTag`, `EntityTagStrength`, `EntityTagStrong`, `EntityTagWeak`, `NewEntityTag`, `ParseEntityTag`, `ParseEntityTagList`, `MatchesIfNoneMatch`, `MatchesIfMatch`, `MiddlewareETag`, `ErrETagConfig`, `ErrCodeETagWriteFailed`, `ErrCodeETagConfigInvalid`, `ErrCodeETagHashWriteFailed` to the frozen API surface tables. Effort: 30min.~~ **Won't implement — obsolete — describes work on code deleted by the 22:22 revert; see the 22:22/22:43 adapter sessions.**
2. ~~**Port BDD spec tests** — Translate `go-etag/etag_bdd_test.go` (7 functions, 334 lines) to httputil naming conventions (`NewETag`→`NewEntityTag`, `Strong`→`EntityTagStrong`, etc.). Effort: 45min.~~ **Won't implement — obsolete — describes work on code deleted by the 22:22 revert; see the 22:22/22:43 adapter sessions.**
3. ~~**Re-create `httpspec/etag_integration_test.go`** — 3 ETag-specific specs + standard 18. Effort: 30min.~~ **Won't implement — obsolete — describes work on code deleted by the 22:22 revert; see the 22:22/22:43 adapter sessions.**
4. ~~**Re-create `etag_compress_fuzz_test.go`** — `FuzzETagConditional` (If-Match/If-None-Match) and `FuzzCompressWriterState` (compression with varied encodings/bodies/content types). Effort: 30min.~~ **Won't implement — obsolete — describes work on code deleted by the 22:22 revert; see the 22:22/22:43 adapter sessions.**

### Medium priority — documentation completeness

5. ~~**Update D2 architecture diagram** — `docs/architecture-understanding/2026-08-05_httputil-current.d2` + `.svg`: middleware count 16→17, add ETag node. Effort: 15min.~~ **Won't implement — obsolete — describes work on code deleted by the 22:22 revert; see the 22:22/22:43 adapter sessions.**
6. ~~**Update `docs/DOMAIN_LANGUAGE.md`** — Add Conditional Requests bounded context: entity-tag, strength (strong/weak), opaque-tag, If-None-Match, If-Match, 304 Not Modified, cache validation. Effort: 20min.~~ **Won't implement — obsolete — describes work on code deleted by the 22:22 revert; see the 22:22/22:43 adapter sessions.**
7. ~~**Check `CONTRIBUTING.md`** for stale ETag references. Effort: 5min.~~ **Won't implement — obsolete — describes work on code deleted by the 22:22 revert; see the 22:22/22:43 adapter sessions.**
8. ~~**Add error template tests** for the 3 new ETag error codes — verify `errorfamily.RenderTemplate` produces expected what/why/fix/wayOut for each. Effort: 20min.~~ **Won't implement — obsolete — describes work on code deleted by the 22:22 revert; see the 22:22/22:43 adapter sessions.**
9. ~~**Add `hijackDelegate` WithContextf test** — Verify the `writer_type` context is attached on both unsupported and failed hijack paths. Effort: 10min.~~ **Won't implement — obsolete — describes work on code deleted by the 22:22 revert; see the 22:22/22:43 adapter sessions.**
10. ~~**Restart LSP** to clear stale `EntityTagStrength.valid unused` and `assertBodyEmpty unused` diagnostics. Effort: 1min.~~ **Won't implement — obsolete — describes work on code deleted by the 22:22 revert; see the 22:22/22:43 adapter sessions.**

### Lower priority — polish and robustness

11. ~~**Add `TestETagConfig_Validate_InvalidStrength_ChecksContext`** — Verify the error context includes the `strength` field value, not just the code.~~ **Won't implement — obsolete — describes work on code deleted by the 22:22 revert; see the 22:22/22:43 adapter sessions.**
12. ~~**Add `TestETagConfig_Validate_ZeroMaxBufferSize_ChecksContext`** — Same for `max_buffer_size`.~~ **Won't implement — obsolete — describes work on code deleted by the 22:22 revert; see the 22:22/22:43 adapter sessions.**
13. ~~**Add a test for `defaultETagHashFunc` panic path** — The `hash.Write` error is caught by a panic, but there's no test verifying the panic message carries the correct error code. Would require injecting a broken `hash.Hash`.~~ **Won't implement — obsolete — describes work on code deleted by the 22:22 revert; see the 22:22/22:43 adapter sessions.**
14. ~~**Consider reducing the 7 non-cacheable status test copies** to a helper-driven pattern if the convention allows.~~ **Won't implement — obsolete — describes work on code deleted by the 22:22 revert; see the 22:22/22:43 adapter sessions.**
15. ~~**Add `BenchmarkETag_LargeBody`** — Current `BenchmarkETag` uses a tiny body; large-body benchmark would surface buffer overflow overhead.~~ **Won't implement — obsolete — describes work on code deleted by the 22:22 revert; see the 22:22/22:43 adapter sessions.**
16. ~~**Add `BenchmarkETag_Overflow`** — Benchmark the streaming-overflow path to quantify the performance cliff at MaxBufferSize.~~ **Won't implement — obsolete — describes work on code deleted by the 22:22 revert; see the 22:22/22:43 adapter sessions.**
17. ~~**Add `BenchmarkETag_SkipIfPresent`** — Benchmark the SkipIfPresent path to quantify parse cost.~~ **Won't implement — obsolete — describes work on code deleted by the 22:22 revert; see the 22:22/22:43 adapter sessions.**
18. ~~**Verify `EntityTag.String()` does not allocate excessively** — Consider a benchmark with `allocs/op`.~~ **Won't implement — obsolete — describes work on code deleted by the 22:22 revert; see the 22:22/22:43 adapter sessions.**
19. ~~**Add ETag to the D2 diagram's middleware chain visualization** — not just the count, but the actual node with connections.~~ **Won't implement — obsolete — describes work on code deleted by the 22:22 revert; see the 22:22/22:43 adapter sessions.**
20. ~~**Update `docs/v1-stability.md` middleware constant count** from 11→12.~~ **Won't implement — obsolete — describes work on code deleted by the 22:22 revert; see the 22:22/22:43 adapter sessions.**
21. ~~**Consider whether `EntityTagStrength.valid()` should be exported** — It's used internally by `ETagConfig.Validate()` but consumers might want to validate strength values too.~~ **Won't implement — obsolete — describes work on code deleted by the 22:22 revert; see the 22:22/22:43 adapter sessions.**
22. ~~**Add `ExampleMatchesIfNoneMatch` and `ExampleMatchesIfMatch`** — Consistent with all other middleware having testable examples.~~ **Won't implement — obsolete — describes work on code deleted by the 22:22 revert; see the 22:22/22:43 adapter sessions.**
23. ~~**Add `ExampleParseEntityTag`** — Show parsing a weak tag from a header value.~~ **Won't implement — obsolete — describes work on code deleted by the 22:22 revert; see the 22:22/22:43 adapter sessions.**
24. ~~**Review whether `headerETag = "ETag"` should be canonical `"Etag"`** — The `canonicalheader` linter didn't flag it, but Go's `textproto.CanonicalMIMEHeaderKey("ETag")` returns `"Etag"`. This is a latent inconsistency.~~ **Won't implement — obsolete — describes work on code deleted by the 22:22 revert; see the 22:22/22:43 adapter sessions.**
25. ~~**Check if `newStatusBodyHandler` should move to `testutil_test.go`** — It's defined in `etag_test.go` but could be reused by other tests.~~ **Won't implement — obsolete — describes work on code deleted by the 22:22 revert; see the 22:22/22:43 adapter sessions.**
26. ~~**Verify the 304 Content-Length stripping works through the Compression+ETag chain** — The chain test checks Content-Encoding exclusion but not Content-Length stripping on 304.~~ **Won't implement — obsolete — describes work on code deleted by the 22:22 revert; see the 22:22/22:43 adapter sessions.**
27. ~~**Add a test for HEAD through Compression+ETag chain** — Verify HEAD body suppression works when compression is also applied.~~ **Won't implement — obsolete — describes work on code deleted by the 22:22 revert; see the 22:22/22:43 adapter sessions.**
28. ~~**Consider adding `Last-Modified` / `If-Modified-Since` support** — RFC 7232 §3.3 companion to ETag. Currently only If-None-Match is supported.~~ **Won't implement — obsolete — describes work on code deleted by the 22:22 revert; see the 22:22/22:43 adapter sessions.**
29. ~~**Consider adding `If-Range` support** — RFC 7233 §3.2 for range requests.~~ **Won't implement — obsolete — describes work on code deleted by the 22:22 revert; see the 22:22/22:43 adapter sessions.**
30. ~~**Add ETag to `docs/RELEASE.md` pre-release checklist** — Ensure ETag tests are in the release gate.~~ **Won't implement — obsolete — describes work on code deleted by the 22:22 revert; see the 22:22/22:43 adapter sessions.**
31. ~~**Verify `go-etag` module can be deprecated** — Add a deprecation notice to its README pointing to httputil.~~ **Won't implement — obsolete — describes work on code deleted by the 22:22 revert; see the 22:22/22:43 adapter sessions.**
32. ~~**Run `art-dupl` on the new test files** — Verify no harmful duplication was introduced by the table-driven test splitting.~~ **Won't implement — obsolete — describes work on code deleted by the 22:22 revert; see the 22:22/22:43 adapter sessions.**
33. ~~**Add ETag middleware to `stack_integration_test.go`** — The full-stack integration test chains all `Middleware*` constants; ETag should be included now that `MiddlewareETag` exists.~~ **Won't implement — obsolete — describes work on code deleted by the 22:22 revert; see the 22:22/22:43 adapter sessions.**
34. ~~**Update `stack_test.go`** — Verify `MiddlewareETag` is tested in the stack duplicate-prevention and ordering tests.~~ **Won't implement — obsolete — describes work on code deleted by the 22:22 revert; see the 22:22/22:43 adapter sessions.**
35. ~~**Consider `ETagConfig` immutability** — The config is passed by value, but `HashFunc`, `Skip`, and `OnError` are function pointers that could be mutated. Document expectations.~~ **Won't implement — obsolete — describes work on code deleted by the 22:22 revert; see the 22:22/22:43 adapter sessions.**
36. ~~**Add a fuzz test for `ETagConfig.Validate`** — Fuzz the config fields to ensure no panic on arbitrary input.~~ **Won't implement — obsolete — describes work on code deleted by the 22:22 revert; see the 22:22/22:43 adapter sessions.**
37. ~~**Review memory allocation in `ParseEntityTagList`** — It allocates `make([]EntityTag, 0)` per call. Consider a pooled allocation for hot paths.~~ **Won't implement — obsolete — describes work on code deleted by the 22:22 revert; see the 22:22/22:43 adapter sessions.**
38. ~~**Add `hexEncodeUint64` to the `hex_test.go` benchmark** — Quantify the zero-allocation claim.~~ **Won't implement — obsolete — describes work on code deleted by the 22:22 revert; see the 22:22/22:43 adapter sessions.**
39. ~~**Consider exporting `hexEncodeUint64`** — Consumers with custom hash functions might want efficient hex encoding.~~ **Won't implement — obsolete — describes work on code deleted by the 22:22 revert; see the 22:22/22:43 adapter sessions.**
40. ~~**Review whether `etagWriter.body` should be pooled** — Each request allocates a `[]byte` that grows to the response size. For high-throughput servers, a `sync.Pool` could reduce GC pressure.~~ **Won't implement — obsolete — describes work on code deleted by the 22:22 revert; see the 22:22/22:43 adapter sessions.**
41. ~~**Add a test for `etagWriter.Flush()` after `Hijack()`** — Verify double-flush doesn't panic.~~ **Won't implement — obsolete — describes work on code deleted by the 22:22 revert; see the 22:22/22:43 adapter sessions.**
42. ~~**Add a test for `etagWriter.Write()` after `Hijack()`** — Verify post-hijack writes stream correctly.~~ **Won't implement — obsolete — describes work on code deleted by the 22:22 revert; see the 22:22/22:43 adapter sessions.**
43. ~~**Consider `ETagConfig.SkipMethod`** — Currently only GET/HEAD are processed; a method predicate would be more flexible than hardcoding.~~ **Won't implement — obsolete — describes work on code deleted by the 22:22 revert; see the 22:22/22:43 adapter sessions.**
44. ~~**Document the FNV-64a collision probability** — The doc comment mentions "~4.3 billion distinct bodies (birthday bound)" but this should be in the README too for consumer visibility.~~ **Won't implement — obsolete — describes work on code deleted by the 22:22 revert; see the 22:22/22:43 adapter sessions.**
45. ~~**Add `ErrETagConfig` to the error classification table in AGENTS.md** — The 3 new error codes are in the table but `ErrETagConfig` sentinel is not mentioned.~~ **Won't implement — obsolete — describes work on code deleted by the 22:22 revert; see the 22:22/22:43 adapter sessions.**
46. ~~**Update `AGENTS.md` error classification table** — Add the ETag error family rows (Transient for write, Rejection for config, Orchestration for hash).~~ **Won't implement — obsolete — describes work on code deleted by the 22:22 revert; see the 22:22/22:43 adapter sessions.**
47. ~~**Review `wrapper.go` `hijackDelegate` changes** — Both branches now have `.WithContextf("writer_type", "%T", w)`. Verify this doesn't break existing tests that check error equality (it shouldn't, since context is additive).~~ **Won't implement — obsolete — describes work on code deleted by the 22:22 revert; see the 22:22/22:43 adapter sessions.**
48. ~~**Consider a `MiddlewareETag` ordering rule in `MiddlewareStack.Validate()`** — ETag should be innermost (closest to handler) so it sees the final response body. Document this.~~ **Won't implement — obsolete — describes work on code deleted by the 22:22 revert; see the 22:22/22:43 adapter sessions.**
49. ~~**Run `nix flake check`** — Verify the full Nix build passes with all changes.~~ **Won't implement — obsolete — describes work on code deleted by the 22:22 revert; see the 22:22/22:43 adapter sessions.**
50. ~~**Tag the next release** — After the above items are complete, the `[Unreleased]` section should be tagged as the next version (likely v0.10.0 given the breaking `EntityTag` rename).~~ **Won't implement — obsolete — describes work on code deleted by the 22:22 revert; see the 22:22/22:43 adapter sessions.**

---

## G) Questions I cannot figure out myself

1. ~~**Should the `go-etag` module be deprecated/archived now?** It still exists at `/home/lars/projects/go-etag/` with v0.1.0. If consumers depend on it, a deprecation notice pointing to httputil is appropriate. If no consumers exist, archiving is cleaner. I cannot determine external consumer status from this codebase.~~ done (answered by history: go-etag is the canonical module (v0.2.0); httputil keeps the deprecated thin adapter)

2. ~~**Should the `EntityTag` type name be kept, or reverted to `ETag`?** The rename frees `ETag()` as the middleware constructor (matching `CORS()`, `Compression()`, etc.), and `EntityTag` is the RFC 7232 term. But it's a **breaking change** from the old in-httputil API. If any consumer depended on `ETag` as a type name, this breaks them. I cannot assess external impact.~~ done (go-etag ships ParseETag/ETagConfig naming; httputil.ETag stays the deprecated adapter)

3. ~~**Is the `EntityTagStrength.valid()` method acceptable as unexported, or should it be exported for consumer-side validation?** It's used internally by `ETagConfig.Validate()`, but consumers constructing `EntityTag` values from external input might want to validate the strength before calling `NewEntityTag`. This is an API design question about the intended usage pattern.~~ **Won't implement — moot — this session was fully reverted; the types live in go-etag.**

---

## Resolution (2026-08-07 docs-health pass; upgraded to per-item markers 2026-08-29)

This session was **reverted in full** at 22:22 — the code-copy approach was wrong and the correct thin-adapter approach landed in `docs/status/2026-08-07_22-22_go-etag-adapter-integration.md` and `docs/status/2026-08-07_22-43_etag-adapter-self-critique.md`. Every item is resolved inline: the CRITICAL fix-steps were executed by the 22:22 session; section A) work was reverted; sections C)/F) are superseded or obsolete. The header banner was removed — its verdict lives on the items and in this appendix. Section E) process lessons are intentionally unmarked.
