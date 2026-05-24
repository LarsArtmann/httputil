# httputil

Composable HTTP middleware and utility primitives for Go — CORS, client IP extraction, response recording, middleware chaining, security headers, request ID, panic recovery, timeout enforcement, and structured logging.

Minimal footprint — single same-author dependency. Pure stdlib `net/http`. Go 1.26+.

## Install

```bash
go get github.com/larsartmann/httputil
```

## Quick Start

```go
package main

import (
    "log/slog"
    "net/http"
    "time"

    "github.com/larsartmann/httputil"
)

func main() {
    mux := http.NewServeMux()
    mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        requestID, _ := httputil.RequestIDFromContext(r.Context())
        slog.Info("handled", "path", r.URL.Path, "request_id", requestID)
        w.WriteHeader(http.StatusOK)
    })

    handler := httputil.Chain(
        mux,
        httputil.Logging(slog.Default()),
        httputil.Recovery(slog.Default()),
        httputil.Timeout(30 * time.Second),
        httputil.RequestID(httputil.DefaultRequestIDConfig()),
        httputil.SecurityHeaders(httputil.DefaultSecurityHeadersConfig()),
        httputil.CORS(httputil.DefaultCORSConfig()),
    )

    http.ListenAndServe(":8080", handler)
}
```

## Features

### CORS Middleware

Configurable Cross-Origin Resource Sharing with sensible defaults.

```go
// Permissive config for local development (allows all origins)
handler := httputil.CORS(httputil.DefaultCORSConfig())(mux)

// Restrict to specific origins in production
cfg := httputil.CORSConfig{
    AllowedOrigins:   []string{"https://myapp.com", "*.example.com"},
    AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE"},
    AllowedHeaders:   []string{"Content-Type", "Authorization"},
    AllowCredentials: true,
    MaxAge:           3600,
}
handler := httputil.CORS(cfg)(mux)
```

Preflight `OPTIONS` requests receive `204 No Content` automatically. Set `OptionsPassthrough` to forward them to your handler instead. Wildcard patterns like `*.example.com` match subdomains.

Use `cfg.Validate()` to catch invalid configurations at startup (e.g., `AllowCredentials: true` with `AllowAllOrigins: true` which browsers reject).

### Client IP Extraction

Extracts the real client IP behind reverse proxies using standard header precedence.

```go
func handler(w http.ResponseWriter, r *http.Request) {
    ip := httputil.ClientIP(r)
    fmt.Fprintf(w, "Your IP: %s", ip)
}
```

Resolution order: `X-Forwarded-For` (first entry) → `X-Real-IP` → `RemoteAddr`.

> **Security:** `ClientIP` trusts proxy headers without validation. Only use behind a reverse proxy that strips or overwrites these headers.

Context helpers are available for downstream access:

```go
handler := httputil.ClientIPMiddleware(next)
// In downstream handler:
ip := httputil.ClientIPFromContext(r.Context())
```

### Response Recording

Wraps `http.ResponseWriter` to capture the status code for logging, metrics, or auditing.

```go
func loggingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        rec := httputil.NewResponseRecorder(w)
        next.ServeHTTP(rec, r)
        log.Printf("%s %s → %d", r.Method, r.URL.Path, rec.Status())
    })
}
```

`ResponseRecorder` transparently supports `http.Flusher`, `http.Hijacker`, and `http.Pusher` when the underlying writer implements them. Write errors carry classified error codes via [go-error-family](https://github.com/larsartmann/go-error-family) for retry decisions.

`HeaderSnapshot()` returns an isolated copy of response headers for inspection:

```go
headers := rec.HeaderSnapshot()
```

### Middleware Chaining

Compose multiple middleware into a single handler. First middleware in the list is outermost.

```go
handler := httputil.Chain(
    mux,
    httputil.CORS(httputil.DefaultCORSConfig()),
    authMiddleware,
    loggingMiddleware,
)
// Execution: CORS → auth → logging → mux
```

### Security Headers

Sets common security response headers with sensible defaults.

```go
handler := httputil.SecurityHeaders(httputil.DefaultSecurityHeadersConfig())(mux)
```

Headers set by default: `X-Content-Type-Options: nosniff`, `X-Frame-Options: DENY`, `X-XSS-Protection: 0`, `Referrer-Policy: strict-origin-when-cross-origin`.

### Request ID

Propagates or generates a request ID per request.

```go
handler := httputil.RequestID(httputil.DefaultRequestIDConfig())(mux)
// Access in downstream handlers:
id := httputil.RequestIDFromContext(r.Context())
```

### Panic Recovery

Catches panics, logs the stack trace, and returns `500 Internal Server Error`.

```go
handler := httputil.Recovery(slog.Default())(mux)
```

### Request Timeout

Enforces a deadline on the request context. The handler must respect context cancellation.

```go
handler := httputil.Timeout(30 * time.Second)(mux)
```

### Structured Logging

Logs each request with method, path, status, duration, and client IP.

```go
handler := httputil.Logging(slog.Default())(mux)
```

### Error Classification

`ResponseRecorder` errors are classified with behavioral families via [go-error-family](https://github.com/larsartmann/go-error-family):

| Method   | Error Code                | Family         | Retryable | When                                         |
| -------- | ------------------------- | -------------- | --------- | -------------------------------------------- |
| `Write`  | `http.write_failed`       | Transient      | Yes       | Underlying ResponseWriter.Write fails        |
| `Hijack` | `http.hijack_unsupported` | Infrastructure | No        | Underlying writer doesn't implement Hijacker |
| `Hijack` | `http.hijack_failed`      | Transient      | Yes       | Underlying Hijack call fails                 |
| `Push`   | `http.push_unsupported`   | Infrastructure | No        | Underlying writer doesn't implement Pusher   |
| `Push`   | `http.push_failed`        | Transient      | Yes       | Underlying Push call fails                   |

Call `RegisterErrorClassifications()` at startup to enable classification of stdlib HTTP errors and register error message templates.

## API

| Function                        | Signature                                                             | Purpose                                  |
| ------------------------------- | --------------------------------------------------------------------- | ---------------------------------------- |
| `CORS`                          | `func(CORSConfig) func(http.Handler) http.Handler`                    | CORS middleware factory                   |
| `DefaultCORSConfig`             | `func() CORSConfig`                                                   | Permissive dev config (allows all origins)|
| `ClientIP`                      | `func(*http.Request) string`                                          | Extract client IP from proxied request    |
| `ClientIPMiddleware`            | `func(http.Handler) http.Handler`                                     | Store client IP in request context        |
| `ClientIPFromContext`           | `func(context.Context) string`                                        | Retrieve stored client IP                 |
| `NewResponseRecorder`           | `func(http.ResponseWriter) *ResponseRecorder`                         | Wrap writer to capture status             |
| `Chain`                         | `func(http.Handler, ...func(http.Handler) http.Handler) http.Handler` | Compose middleware                        |
| `SecurityHeaders`               | `func(SecurityHeadersConfig) func(http.Handler) http.Handler`         | Security response headers                 |
| `DefaultSecurityHeadersConfig`  | `func() SecurityHeadersConfig`                                        | Sensible security defaults                |
| `RequestID`                     | `func(RequestIDConfig) func(http.Handler) http.Handler`               | Request ID propagation/generation         |
| `DefaultRequestIDConfig`        | `func() RequestIDConfig`                                              | Default X-Request-ID config               |
| `RequestIDFromContext`          | `func(context.Context) string`                                        | Retrieve stored request ID                |
| `Recovery`                      | `func(*slog.Logger) func(http.Handler) http.Handler`                  | Panic recovery                            |
| `Timeout`                       | `func(time.Duration) func(http.Handler) http.Handler`                 | Request deadline enforcement              |
| `Logging`                       | `func(*slog.Logger) func(http.Handler) http.Handler`                  | Structured request logging                |
| `RegisterErrorClassifications`  | `func()`                                                              | Register stdlib error sentinels + templates|

### `CORSConfig` fields

| Field                | Type       | Default                                                | Description                         |
| -------------------- | ---------- | ------------------------------------------------------ | ----------------------------------- |
| `AllowedOrigins`     | `[]string` | `["*"]`                                                | Origins permitted in CORS responses (supports `*.example.com`) |
| `AllowedMethods`     | `[]string` | `["GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"]` | Allowed HTTP methods                |
| `AllowedHeaders`     | `[]string` | `["Content-Type", "Authorization", "X-Request-ID"]`    | Accepted request headers            |
| `ExposedHeaders`     | `[]string` | `[]`                                                   | Headers the browser may access      |
| `AllowCredentials`   | `bool`     | `false`                                                | Whether to send credentials         |
| `MaxAge`             | `int`      | `86400`                                                | Preflight cache duration in seconds |
| `AllowAllOrigins`    | `bool`     | `true`                                                 | Respond with `*` for any origin     |
| `OptionsPassthrough` | `bool`     | `false`                                                | Forward OPTIONS to the next handler |

### `ResponseRecorder` methods

| Method                            | Returns                                | Description                                            |
| --------------------------------- | -------------------------------------- | ------------------------------------------------------ |
| `Status()`                        | `int`                                  | Captured status code (0 if `WriteHeader` not called)   |
| `WroteHeader()`                   | `bool`                                 | Whether `WriteHeader` was called                       |
| `HeaderSnapshot()`                | `http.Header`                          | Isolated copy of response headers                      |
| `WriteHeader(int)`                | —                                      | Capture status and delegate                            |
| `Write([]byte)`                   | `(int, error)`                         | Write body, implicitly set 200                         |
| `Flush()`                         | —                                      | Delegate if underlying writer supports `http.Flusher`  |
| `Hijack()`                        | `(net.Conn, *bufio.ReadWriter, error)` | Delegate if underlying writer supports `http.Hijacker` |
| `Push(string, *http.PushOptions)` | `error`                                | Delegate if underlying writer supports `http.Pusher`   |

## Design

- **Zero-cost abstractions** — internal helpers avoid `fmt` and `strconv` allocations on the hot path
- **Stdlib-first** — all middleware uses `func(http.Handler) http.Handler`, compatible with any Go HTTP framework
- **Classified errors** — `ResponseRecorder` errors carry behavioral families (Transient, Infrastructure) and structured context via [go-error-family](https://github.com/larsartmann/go-error-family) for observability and retry logic
- **Single dependency** — only `go-error-family` (same author, zero transitive deps)

## Development

```bash
go test ./...              # Run tests
golangci-lint run          # Lint (~70 linters)
```

## License

Proprietary — see [LICENSE](LICENSE). Contact `git@lars.software` for licensing inquiries.
