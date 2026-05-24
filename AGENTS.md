# httputil — AGENTS.md

**Updated:** 2026-05-24

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
golangci-lint run          # Lint
golangci-lint run --fix    # Auto-fix what's possible
golangci-lint fmt          # Format (gofumpt + golines@120 + gci)
```

`golangci-lint run` is the authoritative quality gate — it's configured with ~70 linters (see `.golangci.yml`). `go vet` alone is insufficient.

## Architecture

Single flat `httputil` package. One external dependency: `github.com/larsartmann/go-error-family`. Go 1.26+.

| File          | Exports                                                                                                                | Purpose                                                         |
| ------------- | ---------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------- |
| `cors.go`     | `CORSConfig`, `DefaultCORSConfig()`, `CORS()`                                                                          | CORS middleware                                                 |
| `clientip.go` | `ClientIP()`                                                                                                           | Client IP extraction (X-Forwarded-For → X-Real-IP → RemoteAddr) |
| `recorder.go` | `ResponseRecorder`, `NewResponseRecorder()`, `Chain()`                                                                 | Response capture + middleware chaining                          |
| `errors.go`   | `ErrCodeWriteFailed`, `ErrCodeHijackUnsupported`, `ErrCodeHijackFailed`, `ErrCodePushUnsupported`, `ErrCodePushFailed` | Error codes for classified errors                               |
| `util.go`     | (unexported `join`, `itoa`)                                                                                            | Internal helpers avoiding strconv import                        |

**Middleware pattern:** All middleware is `func(http.Handler) http.Handler`. `Chain()` applies them in declaration order (first = outermost) via `slices.Backward`.

## Error Classification

Errors from `ResponseRecorder` are classified using `go-error-family`:

| Method   | Error Code                | Family         | Retryable | When                                         |
| -------- | ------------------------- | -------------- | --------- | -------------------------------------------- |
| `Write`  | `http.write_failed`       | Transient      | Yes       | Underlying ResponseWriter.Write fails        |
| `Hijack` | `http.hijack_unsupported` | Infrastructure | No        | Underlying writer doesn't implement Hijacker |
| `Hijack` | `http.hijack_failed`      | Transient      | Yes       | Underlying Hijack call fails                 |
| `Push`   | `http.push_unsupported`   | Infrastructure | No        | Underlying writer doesn't implement Pusher   |
| `Push`   | `http.push_failed`        | Transient      | Yes       | Underlying Push call fails                   |

All classified errors implement `Coded`, `Classified`, `Contextual`, and `Retryable` from `go-error-family`. Consumers can use `errorfamily.Classify(err)` for retry/exit-code decisions.

Context is attached where relevant (e.g., `status` on write errors, `target` on push errors).

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

### Test File Lint Relaxations

In `_test.go` files: `exhaustruct`, `testpackage`, `gochecknoglobals`, `funlen`, `cyclop`, `goconst`, `unused` are suppressed.

## Pre-Existing Lint Warnings

There are ~22 active warnings (mostly `varnamelen` for short parameter names and `noctx` for test requests). **Do not fix these unless explicitly asked** — they are pre-existing and not your responsibility.

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
