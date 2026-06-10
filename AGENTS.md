# httputil — AGENTS.md

**Updated:** 2026-06-08

## Hard Constraints (Will Break Your Code)

These are the non-obvious rules that cause immediate lint failures. Read these before writing any code.

### Allowed Dependencies

`depguard` allows `$gostd`, `$module`, and `github.com/larsartmann/go-error-family` (same author, zero transitive deps). No other third-party libraries.

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

Extract numeric literals into named constants. The `86400` in `DefaultCORSConfig` is a pre-existing violation.

### `paralleltest` — Every Test Must Call `t.Parallel()`

If you write a test function, it must call `t.Parallel()` as its first line.

## Commands

```bash
go test ./...              # Run tests
go test -race ./...        # Race detection
go vet ./...               # Vet
go test -bench=. ./...     # Benchmarks
golangci-lint run          # Lint (~70 linters, 0 issues)
golangci-lint run --fix    # Auto-fix what's possible
golangci-lint fmt          # Format (gofumpt + golines@120 + gci)
```

`golangci-lint run` is the authoritative quality gate — it's configured with ~70 linters (see `.golangci.yml`). `go vet` alone is insufficient.

## Architecture

Single flat `httputil` package. One external dependency: `github.com/larsartmann/go-error-family`. Go 1.26+.

| File                 | Exports                                                                                                                                                                                   | Purpose                                                         |
| -------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------- |
| `cors.go`            | `CORSConfig`, `DefaultCORSConfig()`, `CORS()`, `Validate()`                                                                                                                               | CORS middleware + wildcard origin matching                      |
| `clientip.go`        | `ClientIP()`                                                                                                                                                                              | Client IP extraction (X-Forwarded-For → X-Real-IP → RemoteAddr) |
| `context.go`         | `WithClientIP()`, `ClientIPFromContext()`, `ClientIPMiddleware()`                                                                                                                         | Request context helpers for client IP                           |
| `recorder.go`        | `ResponseRecorder`, `NewResponseRecorder()`, `Chain()`, `HeaderSnapshot()`                                                                                                                | Response capture + middleware chaining                          |
| `errors.go`          | `ErrCodeWriteFailed`, `ErrCodeHijackUnsupported`, `ErrCodeHijackFailed`, `ErrCodeCompressWriteFailed`, `ErrCodeETagWriteFailed`, `RegisterErrorClassifications()`                         | Error codes + stdlib sentinel registration + message templates  |
| `security.go`        | `SecurityHeadersConfig`, `DefaultSecurityHeadersConfig()`, `SecurityHeaders()`, `Validate()`                                                                                              | Security response headers middleware                            |
| `requestid.go`       | `RequestIDConfig`, `DefaultRequestIDConfig()`, `RequestID()`, `RequestIDFromContext()`, `Validate()`                                                                                      | Request ID propagation/generation middleware                    |
| `recovery.go`        | `Recovery()`                                                                                                                                                                              | Panic recovery middleware                                       |
| `timeout.go`         | `Timeout()`                                                                                                                                                                               | Request deadline enforcement middleware                         |
| `logging.go`         | `Logging()`                                                                                                                                                                               | Structured request logging middleware                           |
| `compression.go`     | `CompressionConfig`, `DefaultCompressionConfig()`, `Compression()`, `Validate()`                                                                                                          | Gzip response compression middleware                            |
| `compress_writer.go` | (unexported `compressWriter`)                                                                                                                                                             | Buffered compress-or-pass-through response writer               |
| `etag.go`            | `ETagConfig`, `DefaultETagConfig()`, `ETag()`, `Validate()`                                                                                                                               | ETag generation + 304 conditional request middleware            |
| `util.go`            | (unexported `join`, `itoa`)                                                                                                                                                               | Internal helpers avoiding strconv import                        |
| `wrapper.go`         | (unexported `responseWrapper`)                                                                                                                                                            | Shared ResponseWriter wrapper for compress/etag writers         |
| `testutil_test.go`   | (unexported `newNoOpHandler`, `newCountingHandler`, `newWriteStatusHandler`, `newWriteBodyHandler`, `newTestRequest`, `newRecorder`, `newFlushHandler`, `assertItoa`, `assertSliceEqual`) | Shared test helpers for consistent test patterns                |
| `doc.go`             | (package doc only)                                                                                                                                                                        | Package-level GoDoc documentation                               |

**Middleware pattern:** All middleware is `func(http.Handler) http.Handler`. `Chain()` applies them in declaration order (first = outermost) via `slices.Backward`.

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
- **`util.go` has custom `itoa()` and `join()`** — these exist to avoid importing `strconv`/`strings.Join`. Use them for internal needs; don't add new stdlib imports for functionality already covered here.
- **`CORS` closure captures `allowOrigin`** before the per-request handler — if no `Origin` header is present and `AllowAllOrigins` is false, the default `"*"` is used. This is a known limitation, not a bug to fix.

## Testing Conventions

- **Same package** (`package httputil`, not `package httputil_test`) — tests can access unexported symbols
- **Plain `testing`** — no assertion libraries
- **No table-driven tests** — each case is a standalone `func Test*(t *testing.T)`
- **`t.Errorf`** for non-fatal, `t.Fatalf` for fatal assertions
- **`httptest.NewRecorder()`** + `httptest.NewRequest()` for HTTP doubles
- **Shared test helpers** in `testutil_test.go`: `newNoOpHandler()`, `newCountingHandler()`, `newTestRequest()`, `newRecorder()`
- **Test files split by middleware** — each middleware has its own `*_test.go` (e.g., `security_test.go`, `requestid_test.go`, `recovery_test.go`, `timeout_test.go`, `logging_test.go`). Chain integration tests in `chain_test.go`.

### Test File Lint Relaxations

In `_test.go` files: `exhaustruct`, `testpackage`, `gochecknoglobals`, `funlen`, `cyclop`, `goconst`, `unused` are suppressed.

## Pre-Existing Lint Warnings

There are **0 active warnings** across ~70 linters. `varnamelen` ignores `w`, `r`, `n`, `rw` for `http.ResponseWriter` and `bufio.ReadWriter` patterns. `noctx` warnings in test files are suppressed via `.golangci.yml` exclusions.

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
