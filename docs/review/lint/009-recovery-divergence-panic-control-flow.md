# LINT-009: Rolls-Royce help-center diverges from `httputil.Recovery` — swallowable sentinel, leaked panic strings, panic-driven control flow

- **Date:** 2026-10-08
- **Severity:** Medium
- **Status:** Open
- **Consumers:** Rolls-Royce-mtuGoHelpCenter-golang
- **Ground truth:** httputil `recovery.go` — `Recovery()` re-panics the stdlib `http.ErrAbortHandler` sentinel (the net/http contract for silent connection aborts), writes a fixed generic 500 body, and never reflects panic values to clients; AGENTS.md owner directive: "API functions never panic by design"

## What the consumer does

`internal/api/middleware/error.go` ships a parallel recovery middleware
instead of `httputil.Recovery`, and pairs it with panic-based error
signaling:

1. **`http.ErrAbortHandler` is swallowed** (error.go:55-77): the
   deferred recover converts every panic into a JSON 500, including the
   sentinel `net/http`'s server uses to abort a connection silently.
   The stdlib contract (and httputil's deliberate re-panic) exists so
   aborted hijacked/streamed connections do not get a second,
   corrupting write. The custom recovery writes a response body onto
   connections the server contract says are abandoned.
2. **Panic strings are echoed to clients** (error.go:74-77): a panic
   carrying a `string` becomes `{"error": <that string>}` with status
   500 — internal error text, file paths, or user input embedded in
   `panic(...)` calls flow straight into the response body.
3. **Panic-as-control-flow for expected errors** (`handlers/jira_handler.go:24`):
   `panic(&httputil.HTTPError{Code: http.StatusServiceUnavailable, ...})`
   for a deterministic configuration-missing condition — "HTTPError
   panics are the must*-helper control flow" per the middleware's own
   comment. The owner directive that panics are never an API design
   tool is part of the reason httputil's error model returns classified
   errors (`errorfamily` Rejection/Transient) instead.
4. **Local package named `httputil`** (`internal/pkg/httputil`): a
   wrapper package with the same name as the real dependency
   (`github.com/larsartmann/httputil`, which the same repo also imports
   directly in `internal/api/middleware/common.go`). Two different
   `httputil.` qualifiers in one repo — the recovery code type-asserts
   the *local* `HTTPError`, and future readers will conflate the two.

## Why it is wrong

- The `ErrAbortHandler` divergence produces real corruption: mid-stream
  aborts get a JSON body appended after whatever was written, breaking
  SSE/streaming handlers that legitimately use the abort path.
- Echoing panic strings is an information-disclosure smell and breaks
  the "errors that help without leaking internals" contract the fleet's
  error responses follow.
- Panic-driven control flow loses the error family metadata (family,
  retryability, context) that `errorfamily.Classify` and
  `RegisterErrorClassifications` exist to carry — the rest of the fleet
  routes on codes; this repo routes on Go type assertions.

## Fix

- Replace the custom `Recovery` with `httputil.Recovery(slogger)`;
  keep any JSON-shape requirement in an `ErrorHandler`-style hook or
  wrap Recovery rather than reimplementing the recover logic.
- Return `errorfamily` errors from `jira_handler` and render them at
  the edge (httputil's `HTTPError`/`WriteJSON` helpers or the repo's
  own JSON writer), removing the panic sites.
- Rename `internal/pkg/httputil` (e.g. `internal/pkg/httpjson`) or make
  it a thin re-export with a distinct name.
