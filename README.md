# httputil

Composable HTTP middleware and utility primitives for Go — CORS, client IP extraction, response recording, and middleware chaining.

Minimal footprint — single same-author dependency. Pure stdlib `net/http`. Go 1.26+.

## Install

```bash
go get github.com/larsartmann/httputil
```

## Quick Start

```go
package main

import (
    "fmt"
    "net/http"

    "github.com/larsartmann/httputil"
)

func main() {
    mux := http.NewServeMux()
    mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        fmt.Fprintf(w, "hello from %s", httputil.ClientIP(r))
    })

    handler := httputil.Chain(
        mux,
        httputil.CORS(httputil.DefaultCORSConfig()),
        loggingMiddleware,
    )

    http.ListenAndServe(":8080", handler)
}

func loggingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        rec := httputil.NewResponseRecorder(w)
        next.ServeHTTP(rec, r)
        fmt.Printf("%s %s %d\n", r.Method, r.URL.Path, rec.Status())
    })
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
    AllowedOrigins:   []string{"https://myapp.com"},
    AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE"},
    AllowedHeaders:   []string{"Content-Type", "Authorization"},
    AllowCredentials: true,
    MaxAge:           3600,
}
handler := httputil.CORS(cfg)(mux)
```

Preflight `OPTIONS` requests receive `204 No Content` automatically. Set `OptionsPassthrough` to forward them to your handler instead.

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

`ResponseRecorder` transparently supports `http.Flusher`, `http.Hijacker`, and `http.Pusher` when the underlying writer implements them. Write errors carry classified error codes (`http.write_failed`, `http.hijack_failed`, etc.) via [go-error-family](https://github.com/larsartmann/go-error-family) for retry decisions.

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

## API

| Function              | Signature                                                             | Purpose                                    |
| --------------------- | --------------------------------------------------------------------- | ------------------------------------------ |
| `CORS`                | `func(CORSConfig) func(http.Handler) http.Handler`                    | CORS middleware factory                    |
| `DefaultCORSConfig`   | `func() CORSConfig`                                                   | Permissive dev config (allows all origins) |
| `ClientIP`            | `func(*http.Request) string`                                          | Extract client IP from proxied request     |
| `NewResponseRecorder` | `func(http.ResponseWriter) *ResponseRecorder`                         | Wrap writer to capture status              |
| `Chain`               | `func(http.Handler, ...func(http.Handler) http.Handler) http.Handler` | Compose middleware                         |

### `CORSConfig` fields

| Field                | Type       | Default                                                | Description                         |
| -------------------- | ---------- | ------------------------------------------------------ | ----------------------------------- |
| `AllowedOrigins`     | `[]string` | `["*"]`                                                | Origins permitted in CORS responses |
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
| `WriteHeader(int)`                | —                                      | Capture status and delegate                            |
| `Write([]byte)`                   | `(int, error)`                         | Write body, implicitly set 200                         |
| `Flush()`                         | —                                      | Delegate if underlying writer supports `http.Flusher`  |
| `Hijack()`                        | `(net.Conn, *bufio.ReadWriter, error)` | Delegate if underlying writer supports `http.Hijacker` |
| `Push(string, *http.PushOptions)` | `error`                                | Delegate if underlying writer supports `http.Pusher`   |

## Design

- **Zero-cost abstractions** — internal helpers avoid `fmt` and `strconv` allocations on the hot path
- **Stdlib-first** — all middleware uses `func(http.Handler) http.Handler`, compatible with any Go HTTP framework
- **Classified errors** — `ResponseRecorder` errors carry behavioral families (Transient, Infrastructure) and structured context via [go-error-family](https://github.com/larsartmann/go-error-family) for observability and retry logic

## Development

```bash
go test ./...              # Run tests
golangci-lint run          # Lint (~70 linters)
```

## License

Proprietary — see [LICENSE](LICENSE). Contact `git@lars.software` for licensing inquiries.
