# Status Report — httputil

**Generated:** Thursday, July 02, 2026 at 01:50 CEST
**Version:** v0.3.0 (1 commit ahead: `b137f50`)
**Branch:** master (up to date with origin)

---

## Executive Summary

httputil is a mature, production-ready Go HTTP middleware library at v0.3.0 with 10 middlewares, a full server lifecycle, health checks, and a new `httpspec` behavioral testing subpackage. The codebase is clean: 91.9% total test coverage, 179 test functions, 12 benchmarks, 6 fuzz tests, 11 examples, zero external dependencies beyond the same-author `go-error-family`. Three pre-existing `makezero` lint warnings remain in the root package. The `httpspec` subpackage was just added this session and is fully passing with 96.4% coverage.

---

## a) FULLY DONE

| #   | Item                                      | Evidence                                                                                                                                                        |
| --- | ----------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | **10 core middlewares**                   | CORS, ClientIP, RequestID, SecurityHeaders, Recovery, Timeout, Logging, ResponseRecorder, Compression, ETag — all with tests, examples, benchmarks              |
| 2   | **Chain() middleware composition**        | Reverse-order application via `slices.Backward`, integration tested                                                                                             |
| 3   | **Error classification system**           | 5 error codes via `go-error-family`, Transient vs Infrastructure families, message templates                                                                    |
| 4   | **Compression with RFC 7231 negotiation** | q-value parsing, server priority order, per-encoding `sync.Pool`, content-type deny-list, bounded buffering                                                     |
| 5   | **WriterFactory plugin interface**        | `GzipWriterFactory()`, `DeflateWriterFactory()`, `DefaultWriterFactories()` — extensible to brotli/zstd/lz4                                                     |
| 6   | **Time-ordered request ID generator**     | 16-byte sortable IDs, amortized `crypto/rand` buffer (~1 syscall/256 IDs), thread-safe                                                                          |
| 7   | **ETag with RFC 7232 compliance**         | `If-None-Match` list parsing, 1MB memory limit, zero-allocation hex encoding                                                                                    |
| 8   | **Server lifecycle**                      | `NewServer()`, non-blocking `Start()`, graceful `Shutdown()`, `Addr()`, all timeout validation                                                                  |
| 9   | **Health checks**                         | `/health`, `/health/live`, `/health/ready` — Kubernetes-compatible                                                                                              |
| 10  | **`httpspec` behavioral spec subpackage** | 7 standard specs (index reachability, 404 routing, Content-Type, HEAD/OPTIONS handling, no leaked internals), 31 tests, 96.4% coverage — **added this session** |
| 11  | **Shared `wrapper.go`**                   | Common ResponseWriter wrapper embedded by compress/etag writers, eliminates ~80 lines duplication                                                               |
| 12  | **Nix flake build system**                | Reproducible devShell, test/lint/build/vet/coverage apps, treefmt integration                                                                                   |
| 13  | **GitHub Actions CI**                     | Test + lint + build + vet + govulncheck, pinned golangci-lint v2.12                                                                                             |
| 14  | **Release workflow**                      | Tag-triggered, includes `govulncheck` security scan                                                                                                             |
| 15  | **Documentation suite**                   | README.md, AGENTS.md, FEATURES.md, TODO_LIST.md, CHANGELOG.md, DOMAIN_LANGUAGE.md, doc.go                                                                       |
| 16  | **Fuzz tests**                            | CORS, ClientIP, Compression, ETag, RequestID — 6 total fuzz targets                                                                                             |
| 17  | **Context helpers**                       | `WithClientIP()`, `ClientIPFromContext()`, `ClientIPMiddleware()`, `RequestIDFromContext()`                                                                     |

---

## b) PARTIALLY DONE

| #   | Item                                   | What's Done                                                                                     | What's Missing                                                                                                                                                                |
| --- | -------------------------------------- | ----------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | **Test coverage**                      | 91.9% total (root: 91.3%, httpspec: 96.4%)                                                      | Not 100%. Gaps in compression error branches (`startCompression` type mismatch, `Close` errors), CORS wildcard edge cases, `ResponseRecorder` hijack failure paths            |
| 2   | **Compression content-type filtering** | Hardcoded deny-list (`image/`, `video/`, `audio/`, `application/gzip`, etc.)                    | Not configurable via `CompressionConfig` — consumers can't add/remove entries                                                                                                 |
| 3   | **Lint compliance**                    | `httpspec` subpackage: 0 issues. ~67 of 70 linters fully clean across root.                     | 3 pre-existing `makezero` warnings in root: `id_generator.go:85`, `id_generator_test.go:126`, `recorder.go:44` — all `make([]T, n)` calls that linter wants `make([]T, 0, n)` |
| 4   | **Documentation freshness**            | AGENTS.md updated for httpspec this session                                                     | FEATURES.md and TODO_LIST.md not yet updated to mention `httpspec` subpackage                                                                                                 |
| 5   | **Uncommitted working tree**           | `.gitignore` (buildflow-managed), `flake.lock` (nixpkgs bump), formatting fixes in 3 test files | These changes are sitting unstaged — need commit                                                                                                                              |

---

## c) NOT STARTED

| #   | Item                                                               | Priority |
| --- | ------------------------------------------------------------------ | -------- |
| 1   | `MiddlewareStack` type with ordering validation                    | Medium   |
| 2   | `ResponseWriter` capability interface for Hijack/Flush detection   | Medium   |
| 3   | Streaming ETag option using rolling hash                           | Low      |
| 4   | Request/response metrics middleware                                | Low      |
| 5   | Rate-limiting middleware (sliding window / token bucket)           | Low      |
| 6   | Request body size limit middleware                                 | Low      |
| 7   | Configurable content-type filtering in `CompressionConfig`         | Medium   |
| 8   | Brotli/zstd built-in encoder examples (via WriterFactory)          | Low      |
| 9   | `httpspec` README section documenting the subpackage               | High     |
| 10  | `httpspec` integration test using `httptest.Server` (real network) | Low      |

---

## d) TOTALLY FUCKED UP

| #   | Issue                                                 | Severity | Details                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                            |
| --- | ----------------------------------------------------- | -------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| 1   | **3 `makezero` lint warnings**                        | Low      | `id_generator.go:85` — `out := make([]byte, len(src)*hexEncodedBytes)` — the linter wants `make([]byte, 0, len(src)*hexEncodedBytes)` followed by `append`. This is a **false positive** — the code writes to indices directly (`out[i] = ...`), so zero-length + append would be wrong. Same pattern in `recorder.go:44` and `id_generator_test.go:126`. These need either `//nolint:makezero` comments or a refactor the linter is happy with. The AGENTS.md claims "0 active warnings" which is now inaccurate. |
| 2   | **AGENTS.md claims 0 lint warnings — actually has 3** | Low      | Documentation drift. The `makezero` warnings appeared after the Go 1.26.4 upgrade likely changed linter behavior. AGENTS.md "Pre-Existing Lint Warnings" section says "0 active warnings" — needs updating.                                                                                                                                                                                                                                                                                                        |
| 3   | **Uncommitted formatter changes**                     | Low      | `cors_test.go`, `compression_negotiator_test.go`, `server_test.go` have unstaged formatting changes (multiline argument wrapping). These appear to be from a treefmt/golangci-lint fmt run that was never committed.                                                                                                                                                                                                                                                                                               |

---

## e) WHAT WE SHOULD IMPROVE

### High Impact

1. **Fix the 3 `makezero` warnings** — Either add targeted `//nolint:makezero` with explanatory comments (since the code is correct), or configure `.golangci.yml` to exclude these specific patterns. Restore "0 issues" status.

2. **Update FEATURES.md and TODO_LIST.md for httpspec** — The new subpackage is completely absent from both files. They still say "Last verified: 2026-06-17" and describe the project as a single package.

3. **Document httpspec in README.md** — The README is the sales page. The new behavioral testing subpackage is a significant feature that should be mentioned.

4. **Commit the uncommitted working tree** — `flake.lock` bump, `.gitignore` improvements, and formatting fixes are sitting unstaged. These should be committed.

### Medium Impact

5. **Add `httpspec` to the CI matrix** — Ensure `go test ./httpspec/...` runs in CI alongside the root package (it already runs via `./...` but should be explicit in docs).

6. **Consider a `Spec` interface instead of a struct** — The current `Spec` struct with a `Check` function field works, but an interface would allow more complex spec implementations (setup/teardown, multiple requests).

7. **Expand httpspec leak patterns** — The current leak detection covers `goroutine`, `/usr/local/go/`, `.go:`, `panic:`, `runtime error`. Could add: SQL errors, file system paths, environment variable names, internal hostnames.

8. **Add `httpspec.ExpectHeader` helper** — Natural complement to `ExpectStatus`. Would allow specs like "responses include X-Content-Type-Options: nosniff".

9. **Configurable content-type filtering** — The compression deny-list is hardcoded. Making it configurable via `CompressionConfig.ContentTypeDenyList` would let consumers override defaults.

### Lower Impact

10. **Add `httpspec` benchmarks** — Measure overhead of running the full spec suite (currently ~1s with race detection, but no per-spec timing).

11. **Consider `httpspec.RunConcurrent` variant** — For handlers that use shared state, running specs sequentially may be needed. Currently all specs run parallel.

12. **Streaming ETag** — Currently ETag buffers up to 1MB. A streaming option with rolling hash would handle large responses better.

---

## f) Top 25 Things to Do Next

| #   | Task                                                                          | Impact   | Effort  | Category    |
| --- | ----------------------------------------------------------------------------- | -------- | ------- | ----------- |
| 1   | Fix 3 `makezero` lint warnings (nolint or refactor)                           | Critical | 15 min  | Bug fix     |
| 2   | Update AGENTS.md "Pre-Existing Lint Warnings" section (it says 0, actually 3) | Critical | 5 min   | Docs        |
| 3   | Update FEATURES.md to include httpspec subpackage                             | High     | 15 min  | Docs        |
| 4   | Update TODO_LIST.md to include httpspec and mark completed items              | High     | 15 min  | Docs        |
| 5   | Add httpspec section to README.md                                             | High     | 20 min  | Docs        |
| 6   | Commit uncommitted working tree (flake.lock, .gitignore, formatting)          | High     | 10 min  | Maintenance |
| 7   | Add `ExpectHeader(method, path, header, value) Check` helper to httpspec      | High     | 15 min  | Feature     |
| 8   | Add `ExpectBodyContains(method, path, substring) Check` helper                | Medium   | 15 min  | Feature     |
| 9   | Add CHANGELOG entry for httpspec subpackage                                   | High     | 10 min  | Docs        |
| 10  | Expand httpspec leak pattern coverage                                         | Medium   | 10 min  | Improvement |
| 11  | Make content-type filtering configurable in CompressionConfig                 | Medium   | 30 min  | Feature     |
| 12  | Add `MiddlewareStack` type with ordering validation                           | Medium   | 45 min  | Feature     |
| 13  | Add `ResponseWriter` capability interface for Hijack/Flush                    | Medium   | 30 min  | Refactor    |
| 14  | Tag v0.4.0 release (includes httpspec)                                        | Medium   | 10 min  | Release     |
| 15  | Add httpspec example to doc.go or example_test.go                             | Medium   | 15 min  | Docs        |
| 16  | Improve compression error branch test coverage                                | Medium   | 30 min  | Testing     |
| 17  | Add CORS wildcard edge case tests                                             | Low      | 20 min  | Testing     |
| 18  | Add `httpspec.WithMaxBodySize` option to validate response sizes              | Low      | 20 min  | Feature     |
| 19  | Write brotli/zstd WriterFactory examples in a separate docs/examples dir      | Low      | 30 min  | Docs        |
| 20  | Evaluate streaming ETag with rolling hash                                     | Low      | 2 hours | Research    |
| 21  | Consider rate-limiting middleware                                             | Low      | 2 hours | Feature     |
| 22  | Consider request body size limit middleware                                   | Low      | 1 hour  | Feature     |
| 23  | Consider request/response metrics middleware                                  | Low      | 2 hours | Feature     |
| 24  | Add `httpspec` integration test against real `httptest.Server`                | Low      | 30 min  | Testing     |
| 25  | Update docs/status reports index (if one exists)                              | Low      | 5 min   | Docs        |

---

## g) Top Question I Cannot Figure Out Myself

**Should the 3 `makezero` warnings be suppressed with `//nolint` comments, or should the code be refactored to satisfy the linter?**

The three warnings are all in code where `make([]T, n)` is followed by direct index assignment (`out[i] = value`), not `append`. The linter (`makezero` with `always: true`) wants `make([]T, 0, n)` + `append`, which would be semantically wrong for these patterns — the code writes to specific indices. The options are:

1. **Add `//nolint:makezero` with explanation** — fastest, preserves correct code
2. **Change `.golangci.yml` to `makezero.always: false`** — allows `make([]T, n)` without append, but weakens the linter globally
3. **Refactor to use `slices.Concat` or `bytes.NewBuffer` patterns** — may not be possible for all three cases

This is a judgment call about lint policy that depends on the project owner's philosophy on `nolint` comments vs. configuration changes.

---

## Appendix: Project Metrics Snapshot

| Metric                | Value                                        |
| --------------------- | -------------------------------------------- |
| Go files              | 50                                           |
| Production LOC        | 2,790                                        |
| Test LOC              | 4,192                                        |
| Test functions        | 179                                          |
| Benchmarks            | 12                                           |
| Fuzz tests            | 6                                            |
| Example functions     | 11                                           |
| Total test coverage   | 91.9%                                        |
| Root package coverage | 91.3%                                        |
| httpspec coverage     | 96.4%                                        |
| External dependencies | 1 (`go-error-family` v0.5.1, same author)    |
| Go version            | 1.26.4                                       |
| Lint issues           | 3 (all `makezero`, all pre-existing in root) |
| Tags/releases         | 4 (v0.1.0 → v0.3.0)                          |
| Packages              | 2 (`httputil`, `httputil/httpspec`)          |

---

## Resolution (2026-07-22)

The "NOT STARTED" items in section c) are **all shipped**: `MiddlewareStack` (v0.4.0), `DetectCapabilities` (v0.4.0), rate limiting (v0.4.0), body size limit (v0.4.0), metrics middleware (v0.4.0), configurable content-type filtering (v0.4.0). The `makezero` warnings (section d) were suppressed with `//nolint:makezero` directives. The project is now at v0.5.0 with 13 middlewares, 18 specs, and 2 dependencies (`go-error-family` + `golang.org/x/time`).
