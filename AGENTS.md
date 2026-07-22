# httputil — AGENTS.md

## Hard Constraints (Will Break Your Code)

These are the non-obvious rules that cause immediate lint failures. Read these before writing any code.

### Allowed Dependencies

`depguard` allows `$gostd`, `$module`, `github.com/larsartmann/go-error-family` (same author, zero transitive deps), and `golang.org/x/time` (canonical Go extension for rate limiting). No other third-party libraries.

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
GOEXPERIMENT=jsonv2 go test ./...   # Run tests (jsonv2 required — see below)
GOEXPERIMENT=jsonv2 go test -race ./...   # Race detection
GOEXPERIMENT=jsonv2 go vet ./...    # Vet
GOEXPERIMENT=jsonv2 go test -bench=. ./...     # Benchmarks
golangci-lint run          # Lint (~70 linters, 0 issues)
golangci-lint run --fix    # Auto-fix what's possible
golangci-lint fmt          # Format (gofumpt + golines@120 + gci)
```

All Go commands require `GOEXPERIMENT=jsonv2` because `health.go` imports `encoding/json/v2`. The `flake.nix` devShell sets this automatically via `shellHook`, and all `nix run .#*` apps export it as well. Manual `go` invocations outside the devShell still need the env var. This is a known issue — see `TODO_LIST.md`.

`golangci-lint run` is the authoritative quality gate — it's configured with ~70 linters (see `.golangci.yml`). `go vet` alone is insufficient.

## Architecture

Two packages: the flat `httputil` package (middleware + server lifecycle) and the `httputil/httpspec` subpackage (reusable HTTP behavior specs). Two external dependencies: `github.com/larsartmann/go-error-family` and `golang.org/x/time`. Go 1.26+.

| File                          | Exports                                                                                                                                                                                                                    | Purpose                                                            |
| ----------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------ |
| `cors.go`                     | `CORSConfig`, `DefaultCORSConfig()`, `CORS()`, `Validate()`                                                                                                                                                                | CORS middleware + wildcard origin matching                         |
| `clientip.go`                 | `ClientIP()`                                                                                                                                                                                                               | Client IP extraction (X-Forwarded-For → X-Real-IP → RemoteAddr)    |
| `context.go`                  | `WithClientIP()`, `ClientIPFromContext()`, `ClientIPMiddleware()`                                                                                                                                                          | Request context helpers for client IP                              |
| `recorder.go`                 | `ResponseRecorder`, `NewResponseRecorder()`, `Chain()`, `HeaderSnapshot()`                                                                                                                                                 | Response capture + middleware chaining                             |
| `errors.go`                   | `ErrCodeWriteFailed`, `ErrCodeHijackUnsupported`, `ErrCodeHijackFailed`, `ErrCodeCompressWriteFailed`, `ErrCodeETagWriteFailed`, `RegisterErrorClassifications()`                                                          | Error codes + stdlib sentinel registration + message templates     |
| `security.go`                 | `SecurityHeadersConfig`, `DefaultSecurityHeadersConfig()`, `SecurityHeaders()`, `Validate()`                                                                                                                               | Security response headers middleware                               |
| `requestid.go`                | `RequestIDConfig`, `DefaultRequestIDConfig()`, `RequestID()`, `RequestIDFromContext()`, `Validate()`                                                                                                                       | Request ID propagation/generation middleware                       |
| `id_generator.go`             | `generateTimeOrderedID()` (unexported)                                                                                                                                                                                     | Time-ordered, amortized-random request ID generator                |
| `recovery.go`                 | `Recovery()`                                                                                                                                                                                                               | Panic recovery middleware                                          |
| `timeout.go`                  | `Timeout()`                                                                                                                                                                                                                | Request deadline enforcement middleware                            |
| `logging.go`                  | `Logging()`                                                                                                                                                                                                                | Structured request logging middleware                              |
| `maxbodysize.go`              | `MaxBodySize()`                                                                                                                                                                                                            | Request body size limit middleware (wraps `http.MaxBytesReader`)   |
| `ratelimit.go`                | `RateLimit()`, `RateLimiter`, `TokenBucketLimiter`, `NewTokenBucketLimiter()`, `RateLimitConfig`, `DefaultRateLimitConfig()`, `Validate()`                                                                                 | Token bucket rate limiting via `golang.org/x/time/rate`            |
| `metrics.go`                  | `Metrics()`, `MetricsRecorder`, `MetricsConfig`, `DefaultMetricsConfig()`, `Validate()`                                                                                                                                    | Request metrics recording with pluggable recorder interface        |
| `compression.go`              | `CompressionConfig`, `DefaultCompressionConfig()`, `DefaultWriterFactories()`, `DefaultWriterFactoriesForLevel()`, `GzipWriterFactory()`, `DeflateWriterFactory()`, `Compression()`, `Validate()`                          | Response compression middleware with Accept-Encoding negotiation   |
| `compression_negotiator.go`   | (unexported `negotiator`, `buildNegotiator`)                                                                                                                                                                               | Accept-Encoding negotiation and encoding priority ordering         |
| `compression_qvalue.go`       | (unexported q-value parsers)                                                                                                                                                                                               | RFC 7231 q-value parsing helpers                                   |
| `compress_writer.go`          | (unexported `compressWriter`)                                                                                                                                                                                              | Buffered compress-or-pass-through response writer state machine    |
| `compress_writer_compress.go` | (unexported `compressWriter.startCompression`)                                                                                                                                                                             | Compression writer setup and pool integration                      |
| `compress_pool.go`            | (unexported `newWriterPool`)                                                                                                                                                                                               | Per-encoding writer pools owned by the negotiator                  |
| `compress_content_type.go`    | `DefaultIncompressibleTypes()`                                                                                                                                                                                             | Default content-type deny-list + compressibility filtering         |
| `etag.go`                     | `ETagConfig`, `DefaultETagConfig()`, `ETag()`, `Validate()`                                                                                                                                                                | ETag generation (FNV-64a) + 304 conditional request middleware     |
| `health.go`                   | `HealthStatus`, `HealthResponse`, `HealthHandler()`, `LiveHandler()`, `ReadyHandler()`, `ReadyHandlerWithProbe()`, `RegisterHealth()`                                                                                      | Kubernetes-compatible health endpoints                             |
| `server.go`                   | `ServerConfig`, `DefaultServerConfig()`, `NewServer()`, `Server`, `Start()`, `Shutdown()`, `Addr()`                                                                                                                        | Server lifecycle: config, start, graceful shutdown                 |
| `wrapper.go`                  | (unexported `responseWrapper`)                                                                                                                                                                                             | Shared ResponseWriter wrapper for compress/etag writers            |
| `capabilities.go`             | `DetectCapabilities()`, `Capabilities`                                                                                                                                                                                     | Reports Hijacker/Flusher support on a ResponseWriter               |
| `stack.go`                    | `MiddlewareStack`, `NewMiddlewareStack()`, `Middleware*` name constants                                                                                                                                                    | Named middleware stack: duplicate prevention + ordering validation |
| `hex.go`                      | (unexported `hexDigitsLower`)                                                                                                                                                                                              | Shared lowercase hex lookup table for ETag + RequestID encoding    |
| `queryparam.go`               | `ParseUintQuery()`                                                                                                                                                                                                         | Parse uint values from HTTP query parameters                       |
| `testutil_test.go`            | (unexported `newNoOpHandler`, `newCountingHandler`, `newWriteStatusHandler`, `newWriteBodyHandler`, `newStatusOnlyHandler`, `newTypedBodyHandler`, `newTestRequest`, `newRecorder`, `newFlushHandler`, `assertSliceEqual`) | Shared test helpers for consistent test patterns                   |
| `websocket_upgrade_test.go`   | `TestCompressionETag_WebSocketUpgrade_Passthrough`                                                                                                                                                                         | WebSocket upgrade integration test through Compression + ETag      |
| `doc.go`                      | (package doc only)                                                                                                                                                                                                         | Package-level GoDoc documentation                                  |

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

## Error Classification

Errors from `ResponseRecorder` are classified using `go-error-family`:

| Method   | Error Code                | Family         | Retryable | When                                         |
| -------- | ------------------------- | -------------- | --------- | -------------------------------------------- |
| `Write`  | `http.write_failed`       | Transient      | Yes       | Underlying ResponseWriter.Write fails        |
| `Hijack` | `http.hijack_unsupported` | Infrastructure | No        | Underlying writer doesn't implement Hijacker |
| `Hijack` | `http.hijack_failed`      | Transient      | Yes       | Underlying Hijack call fails                 |

All classified errors implement `Coded`, `Classified`, `Contextual`, and `Retryable` from `go-error-family`. Consumers can use `errorfamily.Classify(err)` for retry/exit-code decisions.

Context is attached where relevant (e.g., `status` on write errors).

## Non-Obvious Behaviors

- **`ResponseRecorder.Status()` returns `0`** (not `200`) when `WriteHeader` hasn't been called. Check `WroteHeader()` to distinguish "no status set" from "status was actually 0".
- **`ClientIP` trusts proxy headers blindly** — it does not validate X-Forwarded-For or X-Real-IP. Only safe behind a reverse proxy that strips/overwrites these headers.
- **`Compression` negotiates encodings** per request from `Accept-Encoding` using RFC 7231 q-values and a server priority order (brotli > zstd > gzip > deflate > identity). If no header is present, the highest-priority configured encoding is chosen.
- **`Compression` pools writers per encoding**, so gzip and deflate each have their own `sync.Pool` owned by the negotiator (one pool per encoding per `Compression` instance). Custom factories can opt into pooling by implementing `Reset(io.Writer)`.
- **`RequestID` default generator** produces a 16-byte time-ordered ID (Unix seconds + atomic counter + random tail) and amortizes `crypto/rand` syscalls across ~256 IDs via a process-wide buffer.
- **`CORS` defaults to wildcard for unmatched origins** — when `AllowAllOrigins` is false and the origin matches no `AllowedOrigins` entry, the middleware falls back to `"*"`. Set `DenyUnmatched: true` on `CORSConfig` to suppress the `Access-Control-Allow-Origin` header entirely for unmatched origins (security-hardening, non-breaking).
- **`TokenBucketLimiter` uses `golang.org/x/time/rate`** per key with opt-in eviction — set `EvictionTTL` to a non-zero duration to enable lazy eviction of idle buckets. Zero (default) means unbounded growth; for large-scale per-IP rate limiting, either set `EvictionTTL` or provide a custom `RateLimiter` implementation (e.g., Redis-backed).
- **`NewTokenBucketLimiter` validates inputs** — `rate` is a positive `float64` (tokens per second), `burst` is a positive `int`; returns an error if either is not positive.
- **`ETag` uses FNV-64a by default** — the `HashFunc func([]byte) uint64` field on `ETagConfig` allows replacing the hash algorithm. Default is FNV-64a (fast, 64-bit, collision-resistant).
- **`Compression` uses `Level` when `WriterFactories` is empty** — `Compression()` builds factories from `cfg.Level` (defaulting to `gzip.DefaultCompression` when Level is 0). When `WriterFactories` is supplied, it takes precedence and `Level` is ignored.
- **`CompressionConfig.IncompressibleTypes`** — nil uses `DefaultIncompressibleTypes()` (backward compatible); an empty slice compresses everything (including images/video). Use `DefaultIncompressibleTypes()` to extend rather than replace the list.
- **`MiddlewareStack.Validate()` is opt-in** — `Build()` does NOT call `Validate()`. The caller decides whether to check ordering. `Validate()` enforces that Recovery is outermost when present.

## Testing Conventions

- **Same package** (`package httputil`, not `package httputil_test`) — tests can access unexported symbols
- **Plain `testing`** — no assertion libraries
- **No table-driven tests** — each case is a standalone `func Test*(t *testing.T)`
- **`t.Errorf`** for non-fatal, `t.Fatalf` for fatal assertions
- **`httptest.NewRecorder()`** + `httptest.NewRequest()` for HTTP doubles
- **Shared test helpers** in `testutil_test.go`: `newNoOpHandler()`, `newCountingHandler()`, `newTestRequest()`, `newRecorder()`
- **Test files split by middleware** — each middleware has its own `*_test.go` (e.g., `security_test.go`, `requestid_test.go`, `recovery_test.go`, `timeout_test.go`, `logging_test.go`). Compression middleware tests are in `compression_test.go` and `compression_negotiator_test.go`; q-value parsing tests are in `compression_qvalue_test.go`; factory tests are in `compression_factory_test.go`; the ID generator has `id_generator_test.go`. Chain integration tests in `chain_test.go`.

### Test File Lint Relaxations

In `_test.go` files: `exhaustruct`, `testpackage`, `gochecknoglobals`, `funlen`, `cyclop`, `goconst`, `unused` are suppressed.

## Pre-Existing Lint Warnings

There are **0 active warnings** across ~70 linters. `makezero` false positives (`make([]T, n)` for direct-index writes) are suppressed with `//nolint:makezero` directives above the statements in `recorder.go` and `stack.go`. `varnamelen` ignores `w`, `r`, `n`, `rw` for `http.ResponseWriter` and `bufio.ReadWriter` patterns. `noctx` warnings in test files are suppressed via `.golangci.yml` exclusions.

## Accepted Code Duplication

`art-dupl` at `-t 25` reports **2 clone groups** that are **intentional and not refactored**:

- **`mw1` / `mw2` middleware definitions in `stack_test.go:16-34`** — these record `mw1-before` / `mw2-before` / `mw1-after` / `mw2-after` into a shared `order` slice; the integer label is intrinsic to the test (it asserts the order of middleware execution). Extracting to a name-parameterized helper would obscure test intent.
- **`newTypedBodyHandler` defined in both `testutil_test.go:79` (httputil) and `httpspec/handlers_test.go:61` (httpspec)** — the same helper shape exists in two test packages; the `httputil` package cannot import from `httpspec` (wrong dependency direction), so the duplication is structural.

At `-t 5` there are 321 additional groups — all are single-line idioms (`t.Parallel()`, `return n, nil`, single-statement patterns) that the threshold of 25 skips.

The two real cross-file duplications that **were** extracted:

- `if !w.wroteHeader { w.WriteHeader(http.StatusOK) }` (compress_writer.go:73, etag.go:119) → `responseWrapper.writeDefaultOK()` in `wrapper.go`.
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
