# Status Report: WebSocket Upgrade Test + Open-Item Triage

**Date:** 2026-07-10 15:18
**Session Scope:** Resolve 4 open TODO items — WebSocket upgrade test (deliverable), Validate() duplication (decision), compress/ split (decision), RequestIDConfig naming (defer)
**Reporter:** Crush (glm-5.2)

---

## Executive Summary

This session took 4 open TODO items and resolved all of them: **1 implemented, 2 rejected with documented technical rationale, 1 correctly deferred.** The headline deliverable is `TestCompressionETag_WebSocketUpgrade_Passthrough` — a real TCP integration test that drives a WebSocket-style Upgrade handshake through the Compression + ETag middleware chain and verifies the 101 Switching Protocols response and post-hijack byte stream are uncorrupted.

All tests pass (283, +1 new), 0 lint issues across ~70 linters, `go vet` clean, race detector clean.

**Honest assessment:** The session was efficient and the decisions are well-reasoned, but the test has a coverage gap (see section E) and I violated a project convention (new file vs. `chain_test.go`). Details below.

---

## a) FULLY DONE

### WebSocket Upgrade Integration Test (the deliverable)

| Aspect          | Detail                                                                                                            |
| --------------- | ----------------------------------------------------------------------------------------------------------------- |
| File            | `websocket_upgrade_test.go` (new, 186 lines)                                                                      |
| Test            | `TestCompressionETag_WebSocketUpgrade_Passthrough`                                                                |
| Approach        | Real `httptest.NewServer` + real `net.Dial` TCP connection                                                        |
| Assertions      | 101 status line, Upgrade/Connection/Accept headers, no `Content-Encoding`, no `ETag`, post-hijack echo round-trip |
| RFC reference   | Uses RFC 6455 section 4.2.2 worked-example key/accept values                                                      |
| Mutation tested | Injected premature `WriteHeader(200)` in `compression.go` — test caught it instantly                              |
| Lint clean      | 0 issues after fixing 4 `noinlineerr` + 1 `wrapcheck` violations                                                  |

**What the test proves:**

1. The `compressWriter.Hijack()` → `beginPlainResponse()` path correctly drains buffered state before yielding the raw connection.
2. The `etagWriter.Hijack()` → `flushed = true` path correctly marks the writer as flushed so no deferred ETag stamp corrupts the handshake.
3. Compression negotiation (triggered by `Accept-Encoding: gzip`) does not inject `Content-Encoding` into a hijacked response.
4. Post-hijack raw byte exchange (the echo) works bidirectionally through the full middleware stack.

### Documentation Updates

| File           | Change                                                                |
| -------------- | --------------------------------------------------------------------- |
| `TODO_LIST.md` | Moved 3 items from Open → Done (Session 3). Added per-item rationale. |
| `FEATURES.md`  | Cleared "Near-term" planned items (both now implemented/verified).    |

### Decision Rationale (documented in TODO_LIST.md)

| Item                     | Decision     | Technical Justification                                                                                                                                                                           |
| ------------------------ | ------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `Validate()` duplication | **Accepted** | Go generics cannot express nil-comparable constraint. An `any`-typed helper silently misses nil `func` fields (typed-nil-in-interface footgun). Only 2 instances (below rule-of-three threshold). |
| `compress/` subfolder    | **Rejected** | Compression files depend on root symbols (`Middleware`, `responseWrapper`, `ErrCode*`), root must re-export compression types → circular import. The flat layout is structural.                   |
| `RequestIDConfig` naming | **Deferred** | Breaking API rename correctly belongs in a major version. Left open under "Deferred to major version."                                                                                            |

---

## b) PARTIALLY DONE

### WebSocket test — coverage gap

The test is **functionally correct and passes**, but it does not exercise one important scenario:

- **Body-then-hijack path not tested:** The handler in the test hijacks _immediately_ without writing any body first. This means the `compressWriter` buffer (`w.buf`) is always empty when `Hijack()` is called. A more thorough test would write some response body bytes _before_ hijacking, to verify that `beginPlainResponse()` correctly drains the buffer to the underlying writer without corrupting the upgrade. The existing `TestChain_CompressionETag_HijackPassthrough` (in `chain_test.go`) also doesn't cover this — it hijacks immediately too.

- **ETag path not mutation-tested:** My mutation test only injected a fault into `compression.go`. I did not run a corresponding mutation on `etag.go` (e.g., forcing `flushed = false` after hijack) to prove the test would catch ETag-specific corruption. The assertions exist (`ETag` header check), but I didn't verify they have teeth against an ETag mutation.

---

## c) NOT STARTED

Nothing. All 4 TODO items were addressed this session.

---

## d) TOTALLY FUCKED UP

Nothing catastrophic. But two things I should own:

1. **Convention violation — new file instead of `chain_test.go`:** The existing pattern in this codebase is that Compression + ETag integration tests live in `chain_test.go` (e.g., `TestChain_CompressionETag_HijackPassthrough`). I created a brand-new `websocket_upgrade_test.go` file instead. This is inconsistent with the established pattern. It's not wrong (the test is self-contained and has its own helper), but a new contributor would expect to find upgrade tests alongside the other chain tests. The AGENTS.md file table also now needs an entry for this new file (which I didn't add — see section E).

2. **First compilation failure — `bufio.Writer.WriteString` return value:** I wrote `_ = bufrw.WriteString(...)` assuming `WriteString` returns one value. It returns `(int, error)`. This is a basic API knowledge failure that cost one compilation cycle. Should have checked the signature.

---

## e) WHAT WE SHOULD IMPROVE

### Direct improvements to this session's work

1. **AGENTS.md file table not updated** — The architecture table lists every `.go` file with its exports and purpose. I added `websocket_upgrade_test.go` but did not add a row to the table. This is a doc-drift I introduced.

2. **RFC 6455 reference unverified** — I cited "RFC 6455 (section 4.2.2)" for the worked-example handshake values. The key/accept pair is widely known and correct, but I did not independently verify the section number. If wrong, it's a misleading citation.

3. **No body-before-hijack test** — As noted in section b), the buffer-drain path on hijack is untested. This is the most dangerous real-world scenario (a handler that writes partial response, then upgrades).

4. **ETag mutation test missing** — Only the compression path was mutation-tested. The ETag assertions are present but unverified for teeth.

5. **`readUpgradeHeaders` helper placement** — This is a test-only utility living in a standalone test file. It could be useful for future HTTP-level integration tests. Consider whether it belongs in `testutil_test.go` alongside the other shared helpers.

### Process improvements

6. **No mutation testing plan upfront** — I mutation-tested reactively (after the test passed) rather than planning which mutations to test against. A disciplined approach would define the mutation set before writing the test.

7. **Decision documentation could be stronger** — The "Accepted" and "Rejected" decisions are documented in TODO_LIST.md session notes, but the detailed technical reasoning (especially the typed-nil-in-interface footgun for the Validate helper) is only in this status report, not in a permanent location. Consider an ADR or a note in AGENTS.md.

---

## f) Up to 50 Things We Should Get Done Next

### High priority (this session's gaps)

1. ~~Add `websocket_upgrade_test.go` row to AGENTS.md file table~~ **Won't implement — websocket test removed 2026-08-07 (fragile); Hijack tiers live in chain_hijack_test.go; AGENTS table current.**
2. ~~Add a body-before-hijack test variant (write N bytes, then upgrade)~~ **Won't implement — websocket framing removed; Hijack byte-integrity test covers the write-then-upgrade path.**
3. ~~Mutation-test the ETag path (force `flushed = false` post-hijack)~~ done (mutation-verified Hijack tests shipped 2026-08-30)
4. ~~Verify the RFC 6455 section 4.2.2 citation or remove the section number~~ **Won't implement — moot — websocket test removed 2026-08-07.**
5. ~~Consider moving the test (or the shared helper) to `chain_test.go` / `testutil_test.go`~~ done (helpers live in testutil_test.go / chain_hijack_test.go)

### RequestIDConfig naming (deferred to major version)

6. ~~Plan the `HeaderName` → `ResponseHeader` rename~~ done (shipped: RequestIDConfig.IncomingHeader/ResponseHeader)
7. ~~Plan the `ForwardHeader` → `IncomingHeader` rename~~ done (shipped with the same rename)
8. ~~Audit all downstream callers of `RequestIDConfig` fields~~ done (renames landed cleanly (v0.7.0))
9. ~~Update all doc comments referencing the old names~~ done (doc comments current)
10. ~~Add a migration note to CHANGELOG.md when the major version lands~~ done (documented at the rename release)

### Test coverage gaps (from FEATURES.md "Not 100%")

11. ~~Error branch in `compression.go` `startCompression` — type mismatch from `factory.NewWriter()`~~ done (compression.pool_type_unexpected code + tests (v0.12.0))
12. ~~Error branch in `compressWriter.Close()` — compression writer close failure~~ done (Close idempotency + error tests (2026-08-30))
13. Edge cases in CORS wildcard matching with unusual patterns (e.g., `*.example.com:` with port)
14. ~~`ResponseRecorder` hijack failure paths — more thorough error classification tests~~ done (wrapper_test.go error-path tests (2026-08-30))

### Compression

15. ~~Brotli encoder example (via `WriterFactory` plugin, no core dependency)~~ done (docs/integrations/brotli-zstd.md)
16. ~~Zstd encoder example (via `WriterFactory` plugin)~~ done (same guide covers zstd)
17. ~~LZ4 encoder example (via `WriterFactory` plugin)~~ done (same WriterFactory pattern documented)
18. ~~Test compression with `Accept-Encoding: br` when only gzip is configured~~ done (negotiator property tests: unavailable encodings never selected)
19. ~~Test compression writer pool reuse under concurrent load (pool stress test)~~ done (pool behavior race-tested (-race -count=10 green))
20. ~~Test `CompressionConfig.IncompressibleTypes` with empty slice (compress everything)~~ done (empty-slice compress-all documented + tested (AGENTS Non-Obvious))
21. ~~Test `CompressionConfig.IncompressibleTypes` with custom MIME type overrides~~ done (custom-type override tests in compression_test.go)

### ETag

22. ~~Test ETag with weak indicator (`W/`) on conditional requests~~ done (v0.9.1 weak comparison + fuzz seeds)
23. Test ETag hash collision behavior with a custom `HashFunc` that always returns 0
24. ~~Test ETag buffer overflow (body > `MaxBufferSize`) streaming path~~ done (go-etag module suite owns buffer-limit behavior)
25. ~~Test ETag + Compression interaction with If-None-Match on compressed responses~~ done (ETag-inside-Compression chain tests + ExpectNotModifiedWithETag)

### Rate Limiting

26. ~~Test `TokenBucketLimiter` with `EvictionTTL` under concurrent access (race)~~ done (FuzzEvictionTTL + race sweeps)
27. ~~Test custom `RateLimiter` implementation (e.g., Redis-backed mock)~~ **Won't implement — deprecated API; migration guide + Redis integration doc cover the pattern.**
28. ~~Test rate limiting with `KeyFunc` that returns empty string~~ done (ticketed as the KeyExtractor-empty-key TODO item (2026-08-30))
29. ~~Test rate limit `OnDenied` callback execution~~ done (OnDenied tested; KRL OnRejected contract documented (T11))
30. ~~Benchmark `TokenBucketLimiter` under high QPS~~ done (BenchmarkTokenBucketLimiter)

### CORS

31. ~~Test CORS with credentials + specific origin (not wildcard)~~ done (CORS test suite covers credentials + specific origins)
32. ~~Test CORS preflight with `Access-Control-Request-Method`~~ done (preflight Access-Control-Request-Method tests)
33. ~~Test CORS with multiple `AllowedMethods`~~ done (multi-method configs tested)
34. ~~Test CORS `DenyUnmatched` with origin that partially matches a pattern~~ done (FuzzCORSOriginMatching covers lookalike/partial matches)

### Server / Lifecycle

35. ~~Test `NewServer` with invalid `ServerConfig` (Validate error propagation)~~ done (server config-validation tests)
36. ~~Test `Shutdown` with active connections (graceful drain timeout)~~ done (graceful-shutdown drain tests)
37. ~~Test `Server.Start` when port is already in use~~ done (listen-error channel tests)
38. ~~Test `Server.Addr()` before and after `Start()`~~ done (Addr() behavior tested; resolved-port variant ticketed (11-30:f26))

### Metrics

39. Test `Metrics` middleware with `PathFunc` that returns empty string
40. Test `Metrics` with custom `MetricsRecorder` implementation
41. Test `Metrics` status code 0 (implicit 200) recording

### httpspec

42. ~~Test `WithExtraSpecs` with custom specs that fail~~ done (httpspec tests cover every option (AGENTS))
43. ~~Test `SkipSpec` option behavior~~ done (SkipSpec tests)
44. ~~Test `RunSerial` vs `Run` execution order~~ done (RunSerial ordering tests)
45. ~~Add specs for common CORS headers (if desired)~~ done (CORSSpecs: 5 specs)

### Infrastructure

46. ~~Set up `govulncheck` in CI (referenced in release notes but verify it's wired)~~ done (ci.yml runs govulncheck)
47. ~~Add Go 1.26 to CI matrix if not already~~ done (setup-go 1.26.x)
48. ~~Consider adding `go test -race -count=10` for flaky-test detection~~ done (established practice: -race -count=10 documented + run green 2026-08-30)
49. ~~Review `go.mod` — is `go-error-family` at latest version?~~ done (go-error-family v0.10.0; go mod verify green)
50. ~~Consider a `just` → `flake.nix` migration audit (justfile marked deprecated in AGENTS.md)~~ done (no justfile exists; flake.nix owns tasks)

---

## g) Top 2 Questions I Cannot Answer Myself

### Q1: Should the WebSocket upgrade test live in `chain_test.go` or its own file?

The existing pattern puts all Compression + ETag integration tests in `chain_test.go`. I broke this convention by creating `websocket_upgrade_test.go`. The test is larger and more self-contained than the chain tests (it has its own helper function and RFC constants), which could justify a separate file. But consistency matters. **I need your call:** move it to `chain_test.go` for consistency, or keep it standalone for readability?

### Q2: Is the body-before-hijack scenario worth testing, or is it a theoretical concern?

The current test hijacks immediately (buffer is always empty). A handler that writes partial body _before_ upgrading would exercise the `beginPlainResponse()` buffer-drain path — which is where a subtle bug would hide. But WebSocket handlers almost never write body before upgrading (the 101 response must be the first thing on the wire). **I need your call:** is this a real-world scenario you want covered, or is it pure theory that doesn't justify the test complexity?

---

## Metrics Snapshot

| Metric            | Value                 |
| ----------------- | --------------------- |
| Go source files   | 66                    |
| Total LOC         | 9,766                 |
| Tests passing     | 283 (+1 this session) |
| Benchmarks        | 14                    |
| Runnable examples | 18                    |
| Lint issues       | 0 (~70 linters)       |
| Race detector     | Clean                 |
| Dependencies      | 1 (`go-error-family`) |
| Go version        | 1.26+                 |

---

## Session Changes (uncommitted)

```
Modified:
  FEATURES.md       — cleared near-term planned items
  TODO_LIST.md      — moved 3 items to Done, added Session 3 block

New:
  websocket_upgrade_test.go — 186 lines, 1 test, 1 helper
```

**Not committed** — awaiting user instruction per project rules.

---

## Resolution (2026-07-22)

The WebSocket upgrade test and all doc changes were committed as `f6c4860` ("Add WebSocket upgrade integration test + open-item triage") and are on `origin/master`. The test lives in `websocket_upgrade_test.go` as planned. However, the `AGENTS.md` file table was **not** updated with a row for `websocket_upgrade_test.go` (section E item 1 remains open). The body-before-hijack coverage gap (section B) and the ETag mutation test (section E item 4) are still open.

> **Final Resolution (2026-08-05, v0.8.0):** WebSocket upgrade integration test committed at v0.7.0. The "open items" listed in this report are resolved: ForwardHeader/HeaderName renames shipped in v0.7.0, DenyUnmatched default flip shipped in v0.7.0, q-value parsing coverage closed in v0.7.1, Go 1.26.5 is the current toolchain. v0.8.0 added CSRF, Server-Timing, and KeyedRateLimit. The WebSocket upgrade test remains in `websocket_upgrade_test.go` as the canonical passthrough validation.
