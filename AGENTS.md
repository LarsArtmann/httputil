# httputil — AGENTS.md

## Hard Constraints (Will Break Your Code)

These are the non-obvious rules that cause immediate lint failures. Read these before writing any code.

### Allowed Dependencies

`depguard` allows `$gostd`, `$module` root and subpackages (via explicit `github.com/larsartmann/httputil` + `/**` entries because `$module` does not expand correctly in depguard v2.12.2), `github.com/larsartmann/go-error-family` (same author, zero transitive deps), `golang.org/x/time` (canonical Go extension for rate limiting), and `github.com/justinas/nosurf` (CSRF protection, used by `csrf.go`). No other third-party libraries.

### `exhaustruct` — Every Struct Field Must Be Set

When creating any struct literal, you must populate **every field**. This applies to `CORSConfig`, `ResponseRecorder`, and all stdlib structs except `os/exec.Cmd`. In test files this is relaxed.

### `err113` — No Inline `errors.New()`

Package-level sentinel errors only. Do not call `errors.New()` or `fmt.Errorf()` inside functions to create error values that could be package-level sentinels.

### `wsl_v5` — Strict Whitespace Rules

Enforces blank lines before `return`, after declarations, and around control flow. Run `golangci-lint fmt` after editing — manual whitespace will likely be wrong.

### `nonamedreturns` — No Named Return Values

Do not use named returns in function signatures.

### `noctx` — Always Use Context

`http.NewRequest` is banned. Use `http.NewRequestWithContext`. In tests, `httptest.NewRequest` is also flagged by this linter (pre-existing warnings — don't fix unless asked).

### `godot` — Comments End With Periods

All doc comments and regular comments must end with a period.

### `mnd` — No Magic Numbers

Extract numeric literals into named constants (e.g. `defaultMaxAge`, `defaultCompressionMinSize`, `defaultETagMaxBufferSize`).

### `gosec` — G705 Excluded Globally

G705 ("XSS via taint analysis") is excluded in `.golangci.yml` gosec settings. This library's purpose is writing HTTP response bodies, so every `ResponseWriter.Write` is intentional output — G705 is structurally a false positive here. Do **not** re-add per-site `//nolint:gosec` directives for response writes; they are fragile under `nolintlint` (flagged as "unused" because gosec taint analysis is non-deterministic across cache states).

### `paralleltest` — Every Test Must Call `t.Parallel()`

If you write a test function, it must call `t.Parallel()` as its first line.

### `noinlineerr` — No Inline Error Checks

Forbidden: `if err := foo(); err != nil`. Use a separate assignment followed by the check.

### `canonicalheader` — Canonical Header Keys

Header keys must match Go's canonical MIME header form (`textproto.CanonicalMIMEHeaderKey`). Use `X-Api-Key`, not `X-API-Key` — all letters after the first hyphen segment are lowercased.

### `testableexamples` — Examples Need Output

Every `Example*` function must include a `// Output:` comment directive. Untestable examples fail the lint.

### `thelper` — Test Helpers Must Call `t.Helper()`

Any function taking `*testing.T` that calls `t.Fatal`/`t.Error` must start with `t.Helper()`.

## Commands

```bash
go test ./...              # Run tests
go test -race ./...        # Race detection (REQUIRED for tests with t.Parallel() or shared state)
go test -race -count=N ./... # Surface timing-dependent races — repeat N times
go vet ./...               # Vet
go test -bench=. ./...     # Benchmarks
golangci-lint run          # Lint (~70 linters, 0 issues)
golangci-lint run --fix    # Auto-fix what's possible
golangci-lint fmt          # Format (gofumpt + golines@120 + gci)
```

**`go test -count=1` does NOT detect data races.** Only `go test -race` catches shared-state access between goroutines. After writing or modifying ANY test that uses `t.Parallel()`, shared fixtures, or closures over mutable state, run `go test -race -count=10 ./...` to surface timing-dependent races before declaring done. (See 2026-08-05 fix in `cors_ratelimit_specs_test.go:138` for an example of a race that passed `go test -count=1 ./...` clean but failed 60% of `-race` runs.)

`golangci-lint run` is the authoritative quality gate — it's configured with ~70 linters (see `.golangci.yml`). `go vet` alone is insufficient.

### Auto-Git-Commit Daemon

An auto-git-commit daemon runs continuously and commits changes automatically. This is expected behavior — do not be surprised by commits you did not make. The daemon infers commit messages from diffs, so messages may be generic. For deliberate commits with meaningful messages, use `git commit` explicitly with `--no-verify` if the pre-commit hook is unavailable (e.g., `dprint` missing in Nix shell).

### Doc-Freshness Cadence

Living docs (`TODO_LIST.md`, `FEATURES.md`, `ROADMAP.md`, `CHANGELOG.md`) should be verified via the `docs-health` skill before each version tag and at least monthly. Historical status reports (`docs/status/`) should be annotated (inline `~~item~~ done at <hash>` markers) when read — reading a stale report without annotating it is a missed obligation.

## Architecture

Two Go modules in a workspace (`go.work`): the root `httputil` module (flat package with middleware + server lifecycle, and the `httputil/httpspec` subpackage for reusable HTTP behavior specs) and the `httputil/server_timing` sub-module (W3C Server-Timing instrumentation, stdlib-only, zero external deps). The root module has three external dependencies: `github.com/larsartmann/go-error-family`, `golang.org/x/time`, and `github.com/justinas/nosurf`. Go 1.26+.

### CHANGELOG Freeze Policy

Once a version tag (e.g., `v0.8.0`) is created, the corresponding `[version]` section in `CHANGELOG.md` is **frozen** — it is immutable history. Corrections, additions, or clarifications for already-released work go in `[Unreleased]`. This prevents retroactive edits that make release history unreliable.

### Why the Root Package Is Flat (Deliberate, Not Debt)

The root `httputil` package has 33 non-test files in one directory. **Decision confirmed by the user (2026-08-05) based on ergonomics**: for a middleware library where everything shares one signature (`func(http.Handler) http.Handler`), a single import path (`httputil.CORS()`) beats fragmented sub-package namespaces (`httputil/cors.CORS()`). Public sub-packages are also structurally impossible: compression depends on root symbols (`Middleware`, `responseWrapper`, `ErrCode*`), creating circular imports if extracted. An `internal/` extraction is technically viable (root → internal is one direction, no cycles) but deferred until post-v1.0 or if root exceeds ~50 non-test files. See `docs/architecture-understanding/2026-08-05_06-56_package-structure-analysis.html` and `docs/modularization/2026-08-05_DECISION.html` for the full analysis.

| File                          | Exports                                                                                                                                                                                                                                                                                                                                                                                                                                  | Purpose                                                                                      |
| ----------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------- |
| `cors.go`                     | `CORSConfig`, `DefaultCORSConfig()`, `CORS()`, `Validate()`                                                                                                                                                                                                                                                                                                                                                                              | CORS middleware + wildcard origin matching                                                   |
| `clientip.go`                 | `ClientIP()`                                                                                                                                                                                                                                                                                                                                                                                                                             | Client IP extraction (X-Forwarded-For → X-Real-IP → RemoteAddr)                              |
| `context.go`                  | `WithClientIP()`, `ClientIPFromContext()`, `ClientIPMiddleware()`                                                                                                                                                                                                                                                                                                                                                                        | Request context helpers for client IP                                                        |
| `recorder.go`                 | `ResponseRecorder`, `NewResponseRecorder()`, `Chain()`, `HeaderSnapshot()`                                                                                                                                                                                                                                                                                                                                                               | Response capture + middleware chaining                                                       |
| `errors.go`                   | `ErrCodeWriteFailed`, `ErrCodeHijackUnsupported`, `ErrCodeHijackFailed`, `ErrCodeCompressWriteFailed`, `ErrCodeETagWriteFailed`, `RegisterErrorClassifications()`                                                                                                                                                                                                                                                                        | Error codes + stdlib sentinel registration + message templates                               |
| `security.go`                 | `SecurityHeadersConfig`, `DefaultSecurityHeadersConfig()`, `SecurityHeaders()`, `Validate()`                                                                                                                                                                                                                                                                                                                                             | Security response headers middleware                                                         |
| `requestid.go`                | `RequestIDConfig`, `DefaultRequestIDConfig()`, `RequestID()`, `RequestIDFromContext()`, `Validate()`                                                                                                                                                                                                                                                                                                                                     | Request ID propagation/generation middleware                                                 |
| `id_generator.go`             | `generateTimeOrderedID()` (unexported)                                                                                                                                                                                                                                                                                                                                                                                                   | Time-ordered, amortized-random request ID generator                                          |
| `recovery.go`                 | `Recovery()`                                                                                                                                                                                                                                                                                                                                                                                                                             | Panic recovery middleware                                                                    |
| `timeout.go`                  | `Timeout()`                                                                                                                                                                                                                                                                                                                                                                                                                              | Request deadline enforcement middleware                                                      |
| `logging.go`                  | `Logging()`                                                                                                                                                                                                                                                                                                                                                                                                                              | Structured request logging middleware                                                        |
| `maxbodysize.go`              | `MaxBodySize()`, `MaxBodySizeConfig`, `DefaultMaxBodySizeConfig()`, `MaxBodySizeMiddleware()`, `Validate()`                                                                                                                                                                                                                                                                                                                              | Request body size limit middleware (wraps `http.MaxBytesReader`)                             |
| `ratelimit.go` _(deprecated)_ | `RateLimit()`, `RateLimiter`, `TokenBucketLimiter`, `NewTokenBucketLimiter()`, `RateLimitConfig`, `DefaultRateLimitConfig()`, `Validate()`                                                                                                                                                                                                                                                                                               | Token bucket rate limiting via `golang.org/x/time/rate` (superseded by `ratelimit_keyed.go`) |
| `ratelimit_keyed.go`          | `KeyedRateLimiter`, `NewKeyedRateLimiter()`, `KeyedRateLimiterMiddleware()`, `KeyedRateLimiterConfig`, `DefaultKeyedRateLimiterConfig()`, `KeyExtractor`, `KeyExtractorFromRemoteAddr()`, `KeyExtractorFromClientIP()`, `Validate()`                                                                                                                                                                                                     | Keyed rate limiting with O(log n) min-heap eviction + `MaxKeys` cap                          |
| `metrics.go`                  | `Metrics()`, `MetricsRecorder`, `MetricsConfig`, `DefaultMetricsConfig()`, `Validate()`                                                                                                                                                                                                                                                                                                                                                  | Request metrics recording with pluggable recorder interface                                  |
| `compression.go`              | `CompressionConfig`, `DefaultCompressionConfig()`, `DefaultWriterFactories()`, `DefaultWriterFactoriesForLevel()`, `GzipWriterFactory()`, `DeflateWriterFactory()`, `Compression()`, `Validate()`                                                                                                                                                                                                                                        | Response compression middleware with Accept-Encoding negotiation                             |
| `compression_negotiator.go`   | (unexported `negotiator`, `buildNegotiator`)                                                                                                                                                                                                                                                                                                                                                                                             | Accept-Encoding negotiation and encoding priority ordering                                   |
| `compression_qvalue.go`       | (unexported q-value parsers)                                                                                                                                                                                                                                                                                                                                                                                                             | RFC 7231 q-value parsing helpers                                                             |
| `compress_writer.go`          | (unexported `compressWriter`)                                                                                                                                                                                                                                                                                                                                                                                                            | Buffered compress-or-pass-through response writer state machine                              |
| `compress_writer_compress.go` | (unexported `compressWriter.startCompression`)                                                                                                                                                                                                                                                                                                                                                                                           | Compression writer setup and pool integration                                                |
| `compress_pool.go`            | (unexported `newWriterPool`)                                                                                                                                                                                                                                                                                                                                                                                                             | Per-encoding writer pools owned by the negotiator                                            |
| `compress_content_type.go`    | `DefaultIncompressibleTypes()`                                                                                                                                                                                                                                                                                                                                                                                                           | Default content-type deny-list + compressibility filtering                                   |
| `etag.go`                     | `ETagConfig`, `DefaultETagConfig()`, `ETag()`, `Validate()`                                                                                                                                                                                                                                                                                                                                                                              | ETag generation (FNV-64a) + RFC 7232 weak-comparison `If-None-Match` 304 middleware          |
| `health.go`                   | `HealthStatus`, `HealthResponse`, `HealthHandler()`, `LiveHandler()`, `ReadyHandler()`, `ReadyHandlerWithProbe()`, `RegisterHealth()`                                                                                                                                                                                                                                                                                                    | Kubernetes-compatible health endpoints                                                       |
| `server.go`                   | `ServerConfig`, `DefaultServerConfig()`, `NewServer()`, `Server`, `Start()`, `Shutdown()`, `Addr()`                                                                                                                                                                                                                                                                                                                                      | Server lifecycle: config, start, graceful shutdown                                           |
| `wrapper.go`                  | (unexported `responseWrapper`)                                                                                                                                                                                                                                                                                                                                                                                                           | Shared ResponseWriter wrapper for compress/etag writers                                      |
| `capabilities.go`             | `DetectCapabilities()`, `Capabilities`                                                                                                                                                                                                                                                                                                                                                                                                   | Reports Hijacker/Flusher support on a ResponseWriter                                         |
| `stack.go`                    | `MiddlewareStack`, `NewMiddlewareStack()`, `Middleware*` name constants                                                                                                                                                                                                                                                                                                                                                                  | Named middleware stack: duplicate prevention + ordering validation                           |
| `hex.go`                      | (unexported `hexDigitsLower`)                                                                                                                                                                                                                                                                                                                                                                                                            | Shared lowercase hex lookup table for ETag + RequestID encoding                              |
| `queryparam.go`               | `ParseUintQuery()`                                                                                                                                                                                                                                                                                                                                                                                                                       | Parse uint values from HTTP query parameters                                                 |
| `decompression.go`            | `DecompressionConfig`, `DefaultDecompressionConfig()`, `Decompression()`, `Validate()`                                                                                                                                                                                                                                                                                                                                                   | Request body decompression middleware (gzip/deflate) with bomb protection                    |
| `csrf.go`                     | `CSRFConfig`, `Validate()`, `CSRFMiddleware()`, `ConfigureNosurfHandler()`, `CSRFResponseHeaderMiddleware()`, `ValidateCSRF()`, `WithCSRFToken()`, `CSRFTokenFromContext()`, `CSRFTokenFromRequest()`, `InvalidateCSRFCookie()`, `CSRFTokenHXHeaders()`, `CSRFTokenHTMLMeta()`, `CSRFTokenFormField()`, `CSRFTestToken()`, `SetPlaintextHTTPOrigin()`, `TranslateCSRFHeaders()`, `ForbiddenHandler()`, `ErrCSRFInvalid`, `ErrCSRFConfig` | CSRF protection middleware via `justinas/nosurf` (double-submit cookie)                      |
| `testutil_test.go`            | (unexported `newNoOpHandler`, `newCountingHandler`, `newWriteStatusHandler`, `newWriteBodyHandler`, `newStatusOnlyHandler`, `newTypedBodyHandler`, `newTestRequest`, `newRecorder`, `newFlushHandler`, `assertSliceEqual`)                                                                                                                                                                                                               | Shared test helpers for consistent test patterns                                             |
| `doc.go`                      | (package doc only)                                                                                                                                                                                                                                                                                                                                                                                                                       | Package-level GoDoc documentation                                                            |

### `httpspec` subpackage

Reusable BDD-style HTTP behavior specifications. Point `httpspec.Run(t, handler)` at any `http.Handler` to validate standard HTTP conventions via parallel subtests with human-readable names.

| File                         | Exports                                                                                                                                                                                                                                                                          | Purpose                                                                                                                                                                                                                                                                                                                                          |
| ---------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `httpspec/doc.go`            | (package doc only)                                                                                                                                                                                                                                                               | Package-level GoDoc with usage examples                                                                                                                                                                                                                                                                                                          |
| `httpspec/httpspec.go`       | `Run()`, `RunSerial()`, `Spec`, `Check`, `Result`, `Category`, `Option`, `WithIndexPath()`, `SkipSpec()`, `WithExtraSpecs()`, `Pass()`, `Fail()`, `ExpectStatus()`, `ExpectNotStatus()`, `ExpectHeader()`, `ExpectHeaderAbsent()`, `ExpectBodyContains()`, `SpecName*` constants | Public API: types, options, spec runner, spec-name constants, check builders                                                                                                                                                                                                                                                                     |
| `httpspec/specs.go`          | (unexported `standardSpecs`, check functions)                                                                                                                                                                                                                                    | 18 standard specs: routing (index, unknown path 404, long URL), method safety (HEAD, OPTIONS, TRACE, POST, CONNECT rejection), response headers (Content-Type on bodies/errors, Location on redirects, no duplicates, Accept handling), security (no leaked internals, no Server version leak, no X-Powered-By, X-Content-Type-Options: nosniff) |
| `httpspec/httpspec_test.go`  | (unexported test handlers, check tests)                                                                                                                                                                                                                                          | Tests for every spec, option, and helper                                                                                                                                                                                                                                                                                                         |
| `httpspec/handlers_test.go`  | (unexported `newStatusOnlyHandler`, `newBareServerNameHandler`, `newHeaderNotFoundHandler`, `newTypedMux`, `newTypedHelloMux`, `newTypedBodyHandler`)                                                                                                                            | Shared test handlers used by httpspec tests, examples, and benchmarks                                                                                                                                                                                                                                                                            |
| `httpspec/example_test.go`   | (runnable examples)                                                                                                                                                                                                                                                              | 7 testable examples with `// Output:` directives for check builders                                                                                                                                                                                                                                                                              |
| `httpspec/benchmark_test.go` | (benchmarks)                                                                                                                                                                                                                                                                     | Benchmarks for `Check` and request-serving baseline                                                                                                                                                                                                                                                                                              |

**Middleware pattern:** All middleware is `func(http.Handler) http.Handler` (aliased as the `Middleware` type in `recorder.go`). `Chain()` applies them in declaration order (first = outermost) via `slices.Backward`. `MiddlewareStack` collects named middleware with duplicate prevention and ordering validation (Recovery must be outermost when present).

### `server_timing` sub-module

Separate Go module (`github.com/larsartmann/httputil/server_timing`, package `servertiming`) for W3C Server-Timing header instrumentation. Stdlib-only — zero external dependencies. Imported by the root module via a `replace` directive (`=> ./server_timing`) and coordinated through `go.work`.

| File                                   | Exports                                                                                                                                                                                      | Purpose                                                              |
| -------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------- |
| `server_timing/server_timing.go`       | `HeaderServerTiming`, `ServerTiming`, `NewServerTiming()`, `Record()`, `Measure()`, `MeasureWithDesc()`, `HeaderValue()`, `String()`, `WithServerTiming()`, `ServerTimingFromContext()`, `RecordServerTiming()`, `MeasureServerTiming()`, `WrapServerTiming()`, `ServerTimingMiddleware()`, `ServerTimingMiddlewareWhen()` | W3C Server-Timing header instrumentation with CRLF-safe values       |
| `server_timing/server_timing_test.go`  | (tests)                                                                                                                                                                                      | Unit tests for collector, context helpers, middleware, wrapper       |
| `server_timing/server_timing_bench_test.go` | (benchmarks)                                                                                                                                                                            | Benchmarks for disabled overhead, measure, record, header rendering  |
| `server_timing/server_timing_fuzz_test.go`  | (fuzz tests)                                                                                                                                                                            | Fuzz tests for HeaderValue CRLF injection and middleware robustness  |

## Error Classification

Errors from `ResponseRecorder`, `compressWriter`, and `CSRFMiddleware` are classified using `go-error-family`:

| Source     | Error Code                   | Family         | Retryable | When                                                           |
| ---------- | ---------------------------- | -------------- | --------- | -------------------------------------------------------------- |
| `Write`    | `http.write_failed`          | Transient      | Yes       | Underlying ResponseWriter.Write fails                          |
| `Hijack`   | `http.hijack_unsupported`    | Infrastructure | No        | Underlying writer doesn't implement Hijacker                   |
| `Hijack`   | `http.hijack_failed`         | Transient      | Yes       | Underlying Hijack call fails                                   |
| `Compress` | `http.compress_write_failed` | Transient      | Yes       | Compression writer Write/Close fails                           |
| `CSRF`     | `csrf_invalid`               | Rejection      | No        | CSRF token missing, malformed, or mismatched                   |
| `CSRF`     | `csrf_config`                | Infrastructure | No        | CSRF configuration invalid (e.g. SameSite=None without Secure) |

All classified errors implement `Coded`, `Classified`, `Contextual`, and `Retryable` from `go-error-family`. Consumers can use `errorfamily.Classify(err)` for retry/exit-code decisions.

Context is attached where relevant (e.g., `status` on write errors).

## Non-Obvious Behaviors

- **`ResponseRecorder.Status()` returns `0`** (not `200`) when `WriteHeader` hasn't been called. Check `WroteHeader()` to distinguish "no status set" from "status was actually 0".
- **`ClientIP` trusts proxy headers blindly** — it does not validate X-Forwarded-For or X-Real-IP. Only safe behind a reverse proxy that strips/overwrites these headers.
- **`Compression` negotiates encodings** per request from `Accept-Encoding` using RFC 7231 q-values and a server priority order (brotli > zstd > gzip > deflate > identity). If no header is present, the highest-priority configured encoding is chosen.
- **`Compression` pools writers per encoding**, so gzip and deflate each have their own `sync.Pool` owned by the negotiator (one pool per encoding per `Compression` instance). Custom factories can opt into pooling by implementing `Reset(io.Writer)`.
- **`Compression` short-circuits identity encoding** — when the client requests `identity` (or no encoding is negotiated), the middleware passes the raw `ResponseWriter` through without wrapping. This means `nopCloserWriter`, `nopFlushCloser`, and `passthroughFactory` are only reachable via direct `compressWriter` construction (unit tests), not through the `Compression()` middleware. They are defensive code for the `WriterFactory` contract.
- **`RequestID` default generator** produces a 16-byte time-ordered ID (Unix seconds + atomic counter + random tail) and amortizes `crypto/rand` syscalls across ~256 IDs via a process-wide buffer.
- **`CORS` denies unmatched origins by default** — `DefaultCORSConfig()` sets `DenyUnmatched: true`. When `AllowAllOrigins` is false and the origin matches no `AllowedOrigins` entry, no `Access-Control-Allow-Origin` header is sent. Bare `CORSConfig{...}` literals get the zero value (`false`), which falls back to `"*"` — set `DenyUnmatched: true` explicitly or start from `DefaultCORSConfig()`.
- **`TokenBucketLimiter` uses `golang.org/x/time/rate`** _(deprecated — see `KeyedRateLimiter` above)_ per key with opt-in eviction — set `EvictionTTL` to a non-zero duration to enable lazy eviction of idle buckets. Zero (default) means unbounded growth; for large-scale per-IP rate limiting, either set `EvictionTTL` or provide a custom `RateLimiter` implementation (e.g., Redis-backed).
- **`NewTokenBucketLimiter` validates inputs** _(deprecated)_ — `rate` is a positive `float64` (tokens per second), `burst` is a positive `int`; returns an error if either is not positive.
- **`ETag` uses FNV-64a by default** — the `HashFunc func([]byte) uint64` field on `ETagConfig` allows replacing the hash algorithm. Default is FNV-64a (fast, 64-bit, collision-resistant).
- **`ETag` `If-None-Match` uses the RFC 7232 §2.3.2 weak comparison function** — `W/"abc"` and `"abc"` are treated as equivalent (the `W/` weakness indicator is ignored). `parseETagList` is quote-state-aware so commas inside quoted opaque-tags do not split the list; backslash-escaped `\"` inside quotes is honored per RFC 7232 §2.3. Multiple `If-None-Match` header lines are combined per RFC 9110 §5.2 via `Header.Values` + `strings.Join` (not `Header.Get`, which returns only the first value). The `parseETagList` allocation (1 alloc / 16-64 B per conditional request) was benchmarked and is acceptable (<5% of the full middleware path); see `BenchmarkETagInList`.
- **`ETag` always overwrites handler-set ETags** — there is no `SkipIfPresent` config. If a downstream handler sets the `ETag` header, the middleware recomputes and overwrites it. Handlers needing domain-specific modification semantics cannot currently win; see `TODO_LIST.md`.
- **`Compression` uses `Level` when `WriterFactories` is empty** — `Compression()` builds factories from `cfg.Level` (defaulting to `gzip.DefaultCompression` when Level is 0). When `WriterFactories` is supplied, it takes precedence and `Level` is ignored.
- **`CompressionConfig.IncompressibleTypes`** — nil uses `DefaultIncompressibleTypes()` (backward compatible); an empty slice compresses everything (including images/video). Use `DefaultIncompressibleTypes()` to extend rather than replace the list.
- **`TokenBucketLimiter` is deprecated** — superseded by `KeyedRateLimiter` (`ratelimit_keyed.go`). New code should use `KeyedRateLimiterMiddleware` instead of `RateLimit()`. The old `RateLimiter` interface, `RateLimitConfig`, and `TokenBucketLimiter` remain for backward compatibility but will be removed.
- **`KeyedRateLimiter` uses O(log n) min-heap eviction** — when `MaxKeys` is set, the oldest accessed key is evicted when capacity is reached. Without `MaxKeys`, growth is unbounded. `EvictionTTL` provides lazy time-based eviction. Both can be combined.
- **`CSRFMiddleware` wraps `justinas/nosurf`** — the CSRF middleware requires the `github.com/justinas/nosurf` dependency (the third external dep). The middleware uses a double-submit cookie pattern. `CSRFConfig.Validate()` enforces secure defaults. HTMX-aware helpers (`CSRFTokenHXHeaders`, `CSRFTokenHTMLMeta`, `CSRFTokenFormField`) expose the token in formats templ/HTMX can consume.
- **`ServerTimingMiddleware` lives in the `server_timing` sub-module** — import `github.com/larsartmann/httputil/server_timing` (package `servertiming`). It injects `*servertiming.ServerTiming` via context — use `servertiming.ServerTimingFromContext(ctx)` or `servertiming.MeasureServerTiming(ctx, name)` to record metrics inside handlers. `servertiming.WrapServerTiming(w, r)` provides manual wrapping without the middleware. Header values are sanitized against CRLF injection.
- **`MiddlewareStack.Validate()` is opt-in** — `Build()` does NOT call `Validate()`. The caller decides whether to check ordering. `Validate()` enforces that Recovery is outermost when present.

## Testing Conventions

- **Same package** (`package httputil`, not `package httputil_test`) — tests can access unexported symbols
- **Plain `testing`** — no assertion libraries
- **No table-driven tests** — each case is a standalone `func Test*(t *testing.T)`
- **`t.Errorf`** for non-fatal, `t.Fatalf` for fatal assertions
- **`httptest.NewRecorder()`** + `httptest.NewRequest()` for HTTP doubles
- **Shared test helpers** in `testutil_test.go`: `newNoOpHandler()`, `newCountingHandler()`, `newTestRequest()`, `newRecorder()`
- **Test files split by middleware** — each middleware has its own `*_test.go` (e.g., `security_test.go`, `requestid_test.go`, `recovery_test.go`, `timeout_test.go`, `logging_test.go`, `csrf_test.go`, `ratelimit_keyed_test.go`). Compression middleware tests are in `compression_test.go` and `compression_negotiator_test.go`; q-value parsing tests are in `compression_qvalue_test.go`; factory tests are in `compression_factory_test.go`; the ID generator has `id_generator_test.go`. Chain integration tests in `chain_test.go`. Server-Timing tests, benchmarks, and fuzz tests are in the `server_timing` sub-module (`server_timing/server_timing_test.go`, `server_timing/server_timing_bench_test.go`, `server_timing/server_timing_fuzz_test.go`).

### Test File Lint Relaxations

In `_test.go` files: `exhaustruct`, `testpackage`, `gochecknoglobals`, `funlen`, `cyclop`, `goconst`, `unused` are suppressed.

## Pre-Existing Lint Warnings

There are **0 active warnings** across ~70 linters. `makezero` false positives (`make([]T, n)` for direct-index writes) are suppressed with `//nolint:makezero` directives above the statements in `recorder.go` and `stack.go`. `varnamelen` ignores `w`, `r`, `n`, `rw` for `http.ResponseWriter` and `bufio.ReadWriter` patterns. `noctx` warnings in test files are suppressed via `.golangci.yml` exclusions.

## Accepted Code Duplication

`art-dupl --type-aware` reports **0 clone groups at every threshold from `-t 2` up to `-t 25`**. `art-dupl` auto-excludes test files by default, so the two structural clones below do not appear in reports; they remain in the code intentionally:

- **`mw1` / `mw2` middleware definitions in `stack_test.go`** — these record `mw1-before` / `mw2-before` / `mw1-after` / `mw2-after` into a shared `order` slice; the integer label is intrinsic to the test (it asserts the order of middleware execution). Extracting to a name-parameterized helper would obscure test intent.
- **`newTypedBodyHandler` defined in both `testutil_test.go` (httputil) and `httpspec/handlers_test.go` (httpspec)** — the same helper shape exists in two test packages; the `httputil` package cannot import from `httpspec` (wrong dependency direction), so the duplication is structural.

Non-test duplication that **was** extracted:

- `errorfamily.WrapTransient(...).WithContext("encoding", w.encoding)` write/close error wrapping (formerly repeated across `writePlain`, `writeCompressed`, `startCompressAndStream`, `flushPlainAndStream`, and both `Close` branches) → consolidated behind `compressWriteError`, with every fallible write routed through the `writeClassified` / `streamClassified` choke points so the wrapping lives in one place.
- `if !w.wroteHeader { w.WriteHeader(http.StatusOK) }` (compress_writer.go, etag.go) → `responseWrapper.writeDefaultOK()` in `wrapper.go`.
- `w.plain = true; w.writeHeaderToUnderlying()` in `compressWriter.Hijack` → reuses `beginPlainResponse()`.

## Additional Active Linters Worth Knowing

These won't surprise you on every edit, but may trigger on specific patterns:

- `wrapcheck` — errors returned from interface methods must be wrapped with `fmt.Errorf("...: %w", err)`
- `godox` — no TODO/FIXME/HACK/BUG comments
- `forbidigo` — forbids certain stdlib patterns (e.g., `fmt.Print*` for logging)
- `gosec` — security linting
- `cyclop` max-complexity 12 — functions must stay under 12 cyclomatic complexity
- `gocritic` — `ifElseChain` check disabled, everything else enabled
- `revive` — `exported` and `package-comments` rules disabled (no doc comments required on exported types)
- `ireturn` — allows returning `error`, `empty`, `anon`, `stdlib`, `generic` interfaces
- `canonicalheader` — header keys must match Go's canonical form (`X-Api-Key` not `X-API-Key`)
- `testableexamples` — `Example*` functions must have `// Output:` directives
- `thelper` — test helpers taking `*testing.T` must start with `t.Helper()`
- `noinlineerr` — forbids `if err := ...; err != nil` inline style; requires separate assignment
- `tparallel` — enforces `t.Parallel()` and prefers `t.Cleanup` over `defer`

```

```
