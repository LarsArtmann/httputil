# Status Report — 2026-08-07 08:52

## TLSConfig Validation, Validate() Audit, Decompression Benchmarks & Fuzz Test

**Session scope:** Execute 4 TODO_LIST items (2 Medium, 2 Low priority).

---

## a) FULLY DONE

### 1. ServerConfig.TLSConfig validation (Medium Priority) ✅

- Added `TLSConfig *tls.Config` field to `ServerConfig` struct (`server.go`)
- Wired `cfg.TLSConfig` through `NewServer()` to the underlying `http.Server` (was hardcoded `nil`)
- Added validation: `MinVersion` (when explicitly set) must be >= TLS 1.2 per RFC 8996
- Zero `MinVersion` is allowed (Go defaults to TLS 1.2 since Go 1.18)
- 7 new tests in `server_test.go`: TLS 1.0 rejection, TLS 1.1 rejection, TLS 1.2 acceptance, TLS 1.3 acceptance, zero MinVersion acceptance, TLSConfig wiring test
- Commits: `e81a714`, `9a4d0de`

### 2. Validate() audit — completeness (Medium Priority) ✅

Audited all 11 config structs + `MiddlewareStack.Validate()`. Every config struct already had a `Validate()` method. Two gaps found and fixed:

- **`KeyedRateLimiterConfig`**: Added negative `TTL` validation — was silently coerced to default by `buildKeyedRateLimiter`. New sentinel `errKeyedTTLNegative`. 2 new tests.
- **`RateLimitConfig`**: Added invalid `Status` code validation (< 100 or > 599, except 0 = default). New sentinel `errInvalidStatus`. 3 new tests.
- Commit: `d990946`, `b6a50fb`, `bd4345f`

### 3. Decompression benchmarks (Low Priority) ✅

- New file `decompression_bench_test.go` with 3 benchmarks:
  - `BenchmarkDecompression_Gzip` — ~5 GB/s throughput, 3 allocs/op
  - `BenchmarkDecompression_Deflate` — ~5 GB/s throughput, 3 allocs/op
  - `BenchmarkDecompression_Passthrough` — measures no-op overhead when no Content-Encoding, ~46 MB/s (small body), 3 allocs/op
- All use `b.ReportAllocs()`, `b.SetBytes()`, `b.Loop()`
- Commit: `8c1cb47`

### 4. Decompression fuzz test (Low Priority) ✅

- New file `decompression_fuzz_test.go` with `FuzzDecompression`
- 11 seed corpus entries: valid gzip, valid deflate, truncated gzip header, garbage bytes, empty body, no encoding, identity, unsupported encoding
- Ran 63K+ executions, 0 crashes
- Verifies status is always 200 or 400 (never 500/panic)
- Has one small unstaged formatting change (makezero nolint)

---

## b) PARTIALLY DONE

### Nothing partially done — all 4 tasks reached completion.

---

## c) NOT STARTED (things I forgot)

### Critical omissions:

1. **TODO_LIST.md not updated** — All 4 items still show `[ ]` unchecked. They should be `[x]` or moved to a "Completed" section. The file explicitly says "Completed work is recorded in CHANGELOG.md" but the items haven't been removed or checked off.

2. **CHANGELOG.md `[Unreleased]` not updated** — None of the 4 tasks have entries. The `[Unreleased]` section has ETag and Server-Timing entries from prior sessions but nothing from this session:
   - Missing: TLSConfig field + validation
   - Missing: KeyedRateLimiterConfig.TTL validation
   - Missing: RateLimitConfig.Status validation
   - Missing: Decompression benchmarks
   - Missing: Decompression fuzz test

3. **AGENTS.md architecture table not updated** — `server.go` row does not mention `TLSConfig` field in `ServerConfig`. The exports column lists `ServerConfig` but anyone reading the table would not know TLSConfig exists.

4. **AGENTS.md "Non-Obvious Behaviors" not updated** — No mention of TLSConfig validation behavior (zero MinVersion = safe default, explicit < TLS 1.2 = error).

5. **FEATURES.md not updated** — No mention of TLS config support as a feature.

6. **One unstaged change** — `decompression_fuzz_test.go` has an unstaged `//nolint:makezero` edit that was lint-formatted but not committed by the auto-commit daemon yet.

### Less critical omissions:

7. **No integration test for TLSConfig through actual server startup** — Tests validate the config and wiring but never start a real TLS server (would require cert generation). This is a coverage gap but arguably out of scope.

8. **Fuzz test invariants are weak** — The fuzz test only checks status codes (200 or 400). It could also verify:
   - Content-Encoding and Content-Length headers are removed after successful decompression
   - No panic on nil body
   - Response body matches decompressed input for valid payloads

9. **Benchmark payload is text-only** — The benchmark uses repetitive English text which compresses very well. Real-world payloads (JSON, HTML, mixed binary) would show different ratios. Not a correctness issue but limits benchmark usefulness.

---

## d) TOTALLY FUCKED UP

### Nothing. All code compiles, all tests pass, 0 lint issues.

- `golangci-lint run`: 0 issues
- `golangci-lint fmt`: clean
- `go test -race -count=3 ./...`: all pass
- `server_timing` sub-module: all tests pass

### Pre-existing issue noticed (not our work):

- `testutil_test.go:182` — `assertBodyEmpty` is flagged as unused by gopls (`unusedfunc`). This is a pre-existing dead function. Not introduced this session. Not our responsibility to fix.

---

## e) WHAT WE SHOULD IMPROVE

### Process improvements:

1. **Update living docs as part of the task, not as an afterthought** — The AGENTS.md memory protocol says "Update at the moment of discovery, not end of session." I discovered TLSConfig was added but didn't update AGENTS.md's architecture table or non-obvious behaviors section.

2. **Check off TODO_LIST items when done** — This is basic hygiene. The items are still unchecked.

3. **CHANGELOG entries should be written when the code ships** — Not batched for later.

### Technical improvements:

4. **TLSConfig validation could go deeper** — Currently only validates `MinVersion`. Could also validate:
   - `CipherSuites` against a safe list (but Go 1.18+ already removes insecure suites)
   - `NextProtos` not empty when HTTP/2 is desired
   - `Certificates` or `GetCertificate` is set (required for actual TLS serving)
   - However, over-validation here could break legitimate use cases, so this is debatable.

5. **Validate() audit was thorough but not exhaustive on edge cases** — I validated the obvious numeric/range fields. I did not deeply analyze whether `CORSConfig.AllowedMethods` being empty is a validation gap (it would produce an empty `Access-Control-Allow-Methods` header, which is valid HTTP but probably a misconfiguration).

6. **Fuzz test should use `t.Parallel()`** — Wait, fuzz tests don't use `t.Parallel()`. Disregard.

7. **Benchmark file could use sub-benchmarks** — Instead of 3 top-level benchmarks, a single `BenchmarkDecompression` with sub-benchmarks (`b.Run("gzip", ...)`) would be more conventional and easier to compare.

---

## f) Up to 50 things to get done next

### Documentation (immediate — should have been done this session):

1. Update `TODO_LIST.md` — check off all 4 completed items
2. Add `[Unreleased]` CHANGELOG entries for all 4 tasks
3. Update `AGENTS.md` architecture table — add `TLSConfig` to `server.go` exports
4. Update `AGENTS.md` Non-Obvious Behaviors — document TLSConfig validation behavior
5. Update `FEATURES.md` — add TLS config support to the server section
6. Commit the unstaged `decompression_fuzz_test.go` formatting change

### Validation hardening:

7. Consider validating `CORSConfig.AllowedMethods` not empty when CORS is active
8. Consider validating `DecompressionConfig.Encodings` entries are recognized ("gzip", "deflate" only)
9. Consider validating `CompressionConfig.IncompressibleTypes` entries are valid content-type prefixes
10. Consider validating `CSRFConfig.MaxAge` not negative (currently zero-value defaults to 24h, but explicit negative would be confusing)
11. Consider adding `ServerConfig.Validate()` call documentation — note that `NewServer` calls it automatically

### Testing improvements:

12. Add integration test: TLS server startup with self-signed cert + `TLSConfig`
13. Strengthen fuzz test: verify header removal after decompression
14. Strengthen fuzz test: verify body roundtrip for valid compressed payloads
15. Add fuzz test for `limitedReadCloser` directly (bomb protection edge cases)
16. Add benchmark: decompression bomb protection (limit-hit path)
17. Add benchmark: decompression with varying payload sizes (sub-benchmarks)
18. Add benchmark: `KeyedRateLimiterConfig.Validate()` (existing benchmarks don't cover Validate)
19. Add benchmark: `ServerConfig.Validate()` with TLSConfig set
20. Add test: `NewServer` with `TLSConfig` containing `Certificates` — verify wired correctly
21. Add test: `RateLimitConfig.Validate()` with valid non-default Status (e.g., 503)
22. Add test: `KeyedRateLimiterConfig.Validate()` with all fields populated (happy path)

### Cleanup:

23. Remove unused `assertBodyEmpty` from `testutil_test.go:182` (pre-existing dead code)
24. Audit for other unused test helpers across `*_test.go` files
25. Run `art-dupl --type-aware` to verify no new duplication introduced by the benchmark/fuzz test files

### Feature work (from ROADMAP/TODO_LIST):

26. Add `nix run .#vulncheck` to RELEASE.md (High Priority TODO item)
27. Consider adding `ServerConfig.MaxHeaderBytes` field (currently hardcoded to 0)
28. Consider adding `ServerConfig.MaxHeaderBytes` validation
29. Add HTTPS redirect middleware (separate from TLSConfig)
30. Add HSTS preload list validation

### Monitoring/observability:

31. Add benchmark for `Decompression` middleware overhead (middleware wrapper cost without actual decompression)
32. Add allocation profiling for decompression hot path
33. Consider adding `Server.StartTLS()` method (parallel to `Start()` but uses `ListenAndServeTLS`)
34. Document TLS certificate management strategy (static files vs `autocert`)

### Architecture:

35. Consider whether `TLSConfig` should be cloned in `NewServer` to prevent caller mutation after server start
36. Consider whether `ServerConfig` should have a `Validate()` call in documentation showing it's automatically called by `NewServer`
37. Review whether `DecompressionConfig.Encodings` validation should reject duplicates
38. Consider whether `CompressionConfig.Level` validation should also check when `WriterFactories` is set (currently Level is validated even when ignored)

### Security:

39. Audit all error messages for information leakage (do any include user input in error responses?)
40. Verify decompression bomb protection works with chunked transfer encoding
41. Consider adding rate limiting to the decompression middleware itself (CPU-bound DoS via compression)
42. Review TLS cipher suite defaults for Go 1.26 — are the defaults still secure?

### Quality gates:

43. Add `go test -fuzz=FuzzDecompression -fuzztime=5m` to CI (longer fuzz runs)
44. Add `golangci-lint run` to a pre-push git hook
45. Verify `govulncheck` passes with all new code
46. Run `go test -race -count=20 ./...` to surface any timing-dependent races in new tests
47. Add code coverage report generation to CI
48. Verify `golangci-lint run` passes in `server_timing/` sub-module with no changes there

### Polish:

49. Refactor benchmarks to use `b.Run` sub-benchmarks for consistency with Go conventions
50. Add `// Output:` example test for `Decompression` if not already present (AGENTS.md says there is one — verify it's current)

---

## g) Questions (cannot figure out myself)

### 1. Should TLSConfig validation go beyond MinVersion?

I validate only `MinVersion >= TLS 1.2`. Should I also validate:

- That `Certificates` or `GetCertificate` is non-nil? (Required for actual TLS serving, but the user might set it later on the `*http.Server` directly.)
- That `CipherSuites` doesn't contain insecure suites? (Go 1.18+ already removes most insecure suites by default, so this is likely redundant.)

This is a tradeoff between strictness and flexibility — I need your preference.

### 2. Should `NewServer` clone the `*tls.Config` to prevent post-startup mutation?

Currently `NewServer` passes `cfg.TLSConfig` directly to `http.Server`. A caller could mutate the config after `NewServer` returns but before `Start()`, which is fine. But they could also mutate it _during_ `Start()`, which is a data race. Should I deep-clone it? (This adds complexity and a `crypto/tls.Config` clone method is available since Go 1.21.)

### 3. Should I update TODO_LIST items now, or do you want a separate docs-health pass?

All 4 TODO items are done but still unchecked. Should I check them off + write CHANGELOG entries right now as part of this session's cleanup? Or do you prefer to run the `docs-health` skill separately to handle all documentation updates in one pass?
