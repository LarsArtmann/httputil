# httputil

[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/httputil.svg)](https://pkg.go.dev/github.com/larsartmann/httputil)
[![Go Version](https://img.shields.io/badge/Go-1.26+-00ADD8)](https://go.dev)
[![Coverage](https://img.shields.io/badge/coverage-97.2%25-brightgreen)](#)
[![govulncheck](https://img.shields.io/badge/govulncheck-clean-brightgreen)](#)
[![License](https://img.shields.io/badge/license-Proprietary-red)](LICENSE)

Composable HTTP middleware, utility primitives, and server lifecycle helpers for Go — CORS, client IP extraction, response recording, middleware chaining, security headers, request ID, panic recovery, timeout enforcement, structured logging, response compression, ETag generation, configurable HTTP server, and standard health checks.

Minimal footprint — two dependencies (`go-error-family` same-author + `golang.org/x/time`). Pure stdlib `net/http`. Go 1.26+.

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

`ResponseRecorder` transparently supports `http.Flusher` and `http.Hijacker` when the underlying writer implements them. Write errors carry classified error codes via [go-error-family](https://github.com/larsartmann/go-error-family) for retry decisions.

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

### Behavioral Spec Suite

Validate any `http.Handler` against standard HTTP conventions with a single call. The `httpspec` subpackage runs 18 behavioral specs as parallel subtests.

```go
import "github.com/larsartmann/httputil/httpspec"

func TestHTTPBehavior(t *testing.T) {
    t.Parallel()
    httpspec.Run(t, handler)
}
```

Specs checked: index page reachability, unknown path 404s, long URL handling, POST safety, Content-Type on bodies and errors, HEAD/OPTIONS/TRACE/CONNECT handling, redirect Location correctness, no duplicate headers, Accept header handling, no leaked internals, no Server version fingerprinting, no X-Powered-By header, X-Content-Type-Options: nosniff presence. Use `RunSerial` for handlers with shared state. Skip inapplicable specs or add custom ones:

```go
httpspec.Run(t, handler,
    httpspec.SkipSpec(httpspec.SpecNameUnknownPathReturns404), // SPA fallback
    httpspec.WithExtraSpecs(httpspec.Spec{
        Name:     "GET /health returns 200",
        Category: httpspec.CategoryRouting,
        Check:    httpspec.ExpectStatus(http.MethodGet, "/health", http.StatusOK),
    }),
)
```

### Security Headers

Sets common security response headers with sensible defaults.

```go
handler := httputil.SecurityHeaders(httputil.DefaultSecurityHeadersConfig())(mux)
```

Headers set by default: `X-Content-Type-Options: nosniff`, `X-Frame-Options: DENY`, `Referrer-Policy: strict-origin-when-cross-origin`.

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

### Response Compression

Transparent response compression negotiated from the client's `Accept-Encoding` header. Supports gzip and deflate out of the box, respects RFC 7231 q-values, and leaves small responses, non-2xx statuses, and already-encoded responses uncompressed.

```go
handler := httputil.Compression(httputil.DefaultCompressionConfig())(mux)
```

Use `cfg.Validate()` to catch invalid configurations at startup (e.g., invalid compression levels or negative minimum sizes).

To add encodings not bundled with this package (brotli, zstd, lz4), register a `WriterFactory`:

```go
cfg := httputil.DefaultCompressionConfig()
cfg.WriterFactories["br"] = func(dst io.Writer) (io.WriteCloser, error) {
    return brotli.NewWriterLevel(dst, brotli.DefaultCompression), nil
}
handler := httputil.Compression(cfg)(mux)
```

`DefaultWriterFactories()` returns a fresh map with gzip, deflate, and identity entries so you can extend rather than replace the built-ins.

### ETag Generation

Generates ETag headers from response body content and handles `If-None-Match` conditional requests with `304 Not Modified`. Only applies to `GET` and `HEAD` requests.

```go
handler := httputil.ETag(httputil.DefaultETagConfig())(mux)
```

Set `Weak: true` for weak ETags (`W/"..."`) if your content may change semantically but not byte-for-byte.

### HTTP Server

A configurable `http.Server` wrapper with sensible timeout defaults and lifecycle helpers.

```go
cfg := httputil.DefaultServerConfig()
cfg.Addr = ":8080"

srv, err := httputil.NewServer(cfg, handler)
if err != nil {
    log.Fatal(err)
}

errChan := srv.Start()

// Wait for shutdown signal, then call srv.Shutdown(ctx).
```

Default timeouts match production recommendations:

- `ReadTimeout`: 10s
- `ReadHeaderTimeout`: 5s
- `WriteTimeout`: 30s
- `IdleTimeout`: 60s

### Health Checks

Standard Kubernetes-compatible health handlers.

```go
mux := http.NewServeMux()
httputil.RegisterHealth(mux)

// Registers:
//   GET /health
//   GET /health/live
//   GET /health/ready
```

Or use individual handlers:

```go
mux.HandleFunc("GET /health", httputil.HealthHandler())
```

For dependency-based readiness (e.g., database connectivity):

```go
mux.HandleFunc("GET /health/ready", httputil.ReadyHandlerWithProbe(db.Ping))
```

### Error Classification

`ResponseRecorder` errors are classified with behavioral families via [go-error-family](https://github.com/larsartmann/go-error-family):

| Method   | Error Code                | Family         | Retryable | When                                         |
| -------- | ------------------------- | -------------- | --------- | -------------------------------------------- |
| `Write`  | `http.write_failed`       | Transient      | Yes       | Underlying ResponseWriter.Write fails        |
| `Hijack` | `http.hijack_unsupported` | Infrastructure | No        | Underlying writer doesn't implement Hijacker |
| `Hijack` | `http.hijack_failed`      | Transient      | Yes       | Underlying Hijack call fails                 |

Call `RegisterErrorClassifications()` at startup to enable classification of stdlib HTTP errors and register error message templates.

## API

| Function                         | Signature                                                             | Purpose                                         |
| -------------------------------- | --------------------------------------------------------------------- | ----------------------------------------------- |
| `CORS`                           | `func(CORSConfig) func(http.Handler) http.Handler`                    | CORS middleware factory                         |
| `DefaultCORSConfig`              | `func() CORSConfig`                                                   | Permissive dev config (allows all origins)      |
| `ClientIP`                       | `func(*http.Request) string`                                          | Extract client IP from proxied request          |
| `ClientIPMiddleware`             | `func(http.Handler) http.Handler`                                     | Store client IP in request context              |
| `ClientIPFromContext`            | `func(context.Context) string`                                        | Retrieve stored client IP                       |
| `WithClientIP`                   | `func(context.Context, string) context.Context`                       | Store client IP in context                      |
| `NewResponseRecorder`            | `func(http.ResponseWriter) *ResponseRecorder`                         | Wrap writer to capture status                   |
| `Chain`                          | `func(http.Handler, ...func(http.Handler) http.Handler) http.Handler` | Compose middleware                              |
| `SecurityHeaders`                | `func(SecurityHeadersConfig) func(http.Handler) http.Handler`         | Security response headers                       |
| `DefaultSecurityHeadersConfig`   | `func() SecurityHeadersConfig`                                        | Sensible security defaults                      |
| `RequestID`                      | `func(RequestIDConfig) func(http.Handler) http.Handler`               | Request ID propagation/generation               |
| `DefaultRequestIDConfig`         | `func() RequestIDConfig`                                              | Default X-Request-ID config                     |
| `RequestIDFromContext`           | `func(context.Context) string`                                        | Retrieve stored request ID                      |
| `Recovery`                       | `func(*slog.Logger) func(http.Handler) http.Handler`                  | Panic recovery                                  |
| `Timeout`                        | `func(time.Duration) func(http.Handler) http.Handler`                 | Request deadline enforcement                    |
| `Logging`                        | `func(*slog.Logger) func(http.Handler) http.Handler`                  | Structured request logging                      |
| `MaxBodySize`                    | `func(int64) func(http.Handler) http.Handler`                         | Request body size limit                         |
| `Compression`                    | `func(CompressionConfig) func(http.Handler) http.Handler`             | Negotiated response compression                 |
| `DefaultCompressionConfig`       | `func() CompressionConfig`                                            | gzip/deflate defaults                           |
| `DefaultWriterFactories`         | `func() map[string]WriterFactory`                                     | Built-in gzip/deflate/identity factories        |
| `DefaultWriterFactoriesForLevel` | `func(int) map[string]WriterFactory`                                  | Built-in factories at a given compression level |
| `GzipWriterFactory`              | `func(int) WriterFactory`                                             | Stdlib gzip factory at a given level            |
| `DeflateWriterFactory`           | `func(int) WriterFactory`                                             | Stdlib flate/raw-deflate factory                |
| `DefaultIncompressibleTypes`     | `func() []string`                                                     | Default content-type deny-list for compression  |
| `ETag`                           | `func(ETagConfig) func(http.Handler) http.Handler`                    | ETag generation + 304 handling                  |
| `DefaultETagConfig`              | `func() ETagConfig`                                                   | Strong ETag defaults                            |
| `RateLimit`                      | `func(RateLimitConfig) func(http.Handler) http.Handler`               | Token bucket rate limiting                      |
| `DefaultRateLimitConfig`         | `func() RateLimitConfig`                                              | Default rate limit config                       |
| `NewTokenBucketLimiter`          | `func(float64, int) (*TokenBucketLimiter, error)`                     | Token bucket limiter constructor                |
| `ParseUintQuery`                 | `func(*http.Request, string) uint`                                    | Parse uint from query param                     |
| `Metrics`                        | `func(MetricsConfig) func(http.Handler) http.Handler`                 | Request metrics recording                       |
| `DefaultMetricsConfig`           | `func() MetricsConfig`                                                | Default metrics config                          |
| `NewMiddlewareStack`             | `func() *MiddlewareStack`                                             | Named middleware stack builder                  |
| `DetectCapabilities`             | `func(http.ResponseWriter) Capabilities`                              | Report Hijacker/Flusher support                 |
| `RegisterErrorClassifications`   | `func()`                                                              | Register stdlib error sentinels + templates     |
| `NewServer`                      | `func(ServerConfig, http.Handler) (*Server, error)`                   | Configurable HTTP server with timeouts          |
| `DefaultServerConfig`            | `func() ServerConfig`                                                 | Sensible server timeout defaults                |
| `RegisterHealth`                 | `func(*http.ServeMux)`                                                | Register /health + /live + /ready               |
| `HealthHandler`                  | `func() http.HandlerFunc`                                             | Simple `{"status":"up"}` handler                |
| `LiveHandler`                    | `func() http.HandlerFunc`                                             | Kubernetes liveness probe handler               |
| `ReadyHandler`                   | `func() http.HandlerFunc`                                             | Kubernetes readiness probe handler              |
| `ReadyHandlerWithProbe`          | `func(func() bool) http.HandlerFunc`                                  | Readiness with dependency probe (200/503)       |

### `CORSConfig` fields

| Field                | Type       | Default                                                | Description                                                                     |
| -------------------- | ---------- | ------------------------------------------------------ | ------------------------------------------------------------------------------- |
| `AllowedOrigins`     | `[]string` | `["*"]`                                                | Origins permitted in CORS responses (supports `*.example.com`)                  |
| `AllowedMethods`     | `[]string` | `["GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"]` | Allowed HTTP methods                                                            |
| `AllowedHeaders`     | `[]string` | `["Content-Type", "Authorization", "X-Request-ID"]`    | Accepted request headers                                                        |
| `ExposedHeaders`     | `[]string` | `[]`                                                   | Headers the browser may access                                                  |
| `AllowCredentials`   | `bool`     | `false`                                                | Whether to send credentials                                                     |
| `MaxAge`             | `int`      | `86400`                                                | Preflight cache duration in seconds                                             |
| `AllowAllOrigins`    | `bool`     | `true`                                                 | Respond with `*` for any origin                                                 |
| `OptionsPassthrough` | `bool`     | `false`                                                | Forward OPTIONS to the next handler                                             |
| `DenyUnmatched`      | `bool`     | `true`                                                 | Withhold `Allow-Origin` for origins not in `AllowedOrigins` (secure by default) |

### `ResponseRecorder` methods

| Method             | Returns                                | Description                                            |
| ------------------ | -------------------------------------- | ------------------------------------------------------ |
| `Status()`         | `int`                                  | Captured status code (0 if `WriteHeader` not called)   |
| `WroteHeader()`    | `bool`                                 | Whether `WriteHeader` was called                       |
| `HeaderSnapshot()` | `http.Header`                          | Isolated copy of response headers                      |
| `WriteHeader(int)` | —                                      | Capture status and delegate                            |
| `Write([]byte)`    | `(int, error)`                         | Write body, implicitly set 200                         |
| `Flush()`          | —                                      | Delegate if underlying writer supports `http.Flusher`  |
| `Hijack()`         | `(net.Conn, *bufio.ReadWriter, error)` | Delegate if underlying writer supports `http.Hijacker` |

### `CompressionConfig` fields

| Field                 | Type                       | Default                        | Description                                                                                     |
| --------------------- | -------------------------- | ------------------------------ | ----------------------------------------------------------------------------------------------- |
| `MinSize`             | `int`                      | `512`                          | Minimum response body size before compression is attempted                                      |
| `Level`               | `int`                      | `gzip.DefaultCompression`      | Compression level used when `WriterFactories` is not supplied; applies to both gzip and deflate |
| `WriterFactories`     | `map[string]WriterFactory` | gzip, deflate, identity        | Encoding-name → factory map; replace or extend                                                  |
| `IncompressibleTypes` | `[]string`                 | `DefaultIncompressibleTypes()` | Content-types to skip (nil = defaults, empty = compress all)                                    |

### `ETagConfig` fields

| Field           | Type                  | Default   | Description                                                                      |
| --------------- | --------------------- | --------- | -------------------------------------------------------------------------------- |
| `Weak`          | `bool`                | `false`   | Emit weak ETags (`W/"..."`) for semantically-volatile content                    |
| `MaxBufferSize` | `int`                 | `1048576` | Max bytes buffered for ETag computation before abandoning and streaming (1 MB)   |
| `HashFunc`      | `func([]byte) uint64` | FNV-64a   | Body hash function for ETag generation; replace for application-specific hashing |

### `RateLimitConfig` fields

| Field      | Type                         | Default            | Description                                                       |
| ---------- | ---------------------------- | ------------------ | ----------------------------------------------------------------- |
| `Limiter`  | `RateLimiter`                | `nil`              | Decides whether to allow each request (required)                  |
| `KeyFunc`  | `func(*http.Request) string` | `nil` (RemoteAddr) | Extracts the rate-limiting key from the request (e.g., client IP) |
| `Status`   | `int`                        | `429`              | HTTP status when rate limited (ignored when `OnDenied` is set)    |
| `OnDenied` | `http.HandlerFunc`           | `nil`              | Custom handler for rejected requests; overrides default response  |

### `MetricsConfig` fields

| Field      | Type                         | Default              | Description                                     |
| ---------- | ---------------------------- | -------------------- | ----------------------------------------------- |
| `Recorder` | `MetricsRecorder`            | `nil`                | Receives one observation per request (required) |
| `PathFunc` | `func(*http.Request) string` | `nil` (`r.URL.Path`) | Extracts the path to record from the request    |

### `SecurityHeadersConfig` fields

| Field                     | Type     | Default                             | Description                                         |
| ------------------------- | -------- | ----------------------------------- | --------------------------------------------------- |
| `ContentTypeNosniff`      | `bool`   | `true`                              | Set `X-Content-Type-Options: nosniff`               |
| `FrameOptions`            | `string` | `"DENY"`                            | `X-Frame-Options` value (empty = header omitted)    |
| `StrictTransportSecurity` | `string` | `""`                                | `Strict-Transport-Security` value (empty = omitted) |
| `ReferrerPolicy`          | `string` | `"strict-origin-when-cross-origin"` | `Referrer-Policy` value (empty = omitted)           |
| `ContentSecurityPolicy`   | `string` | `""`                                | `Content-Security-Policy` value (empty = omitted)   |

### `RequestIDConfig` fields

| Field            | Type            | Default          | Description                                             |
| ---------------- | --------------- | ---------------- | ------------------------------------------------------- |
| `ResponseHeader` | `string`        | `"X-Request-ID"` | Response header to set with the resolved request ID     |
| `GenerateID`     | `func() string` | Time-ordered hex | ID generator invoked when no incoming header is present |
| `IncomingHeader` | `string`        | `"X-Request-ID"` | Incoming request header to read the upstream ID from    |

### `ServerConfig` fields

| Field               | Type            | Default   | Description                                               |
| ------------------- | --------------- | --------- | --------------------------------------------------------- |
| `Addr`              | `string`        | `":8080"` | Listen address                                            |
| `ReadTimeout`       | `time.Duration` | `10s`     | Maximum duration for reading the entire request           |
| `ReadHeaderTimeout` | `time.Duration` | `5s`      | Maximum duration for reading request headers              |
| `WriteTimeout`      | `time.Duration` | `30s`     | Maximum duration before timing out writes                 |
| `IdleTimeout`       | `time.Duration` | `60s`     | Maximum time to wait for the next request on a connection |

## Design

- **Stdlib-first** — all middleware uses `func(http.Handler) http.Handler`, compatible with any Go HTTP framework
- **Classified errors** — `ResponseRecorder` errors carry behavioral families (Transient, Infrastructure) and structured context via [go-error-family](https://github.com/larsartmann/go-error-family) for observability and retry logic
- **Minimal dependencies** — `go-error-family` (same author, zero transitive deps) and `golang.org/x/time` (canonical Go rate-limit extension)

### Middleware Ordering

`Chain` applies middleware in **reverse declaration order** so the first argument is outermost:

```go
handler := Chain(mux,
    Logging(logger),          // outermost — logs first, sees final status
    Recovery(logger),         // catches panics before they reach Logging
    Timeout(30*time.Second),  // enforces deadline
    RequestID(DefaultRequestIDConfig()),
    SecurityHeaders(DefaultSecurityHeadersConfig()),
    CORS(DefaultCORSConfig()),
)
```

When using **Compression** and **ETag** together, order matters:

```go
// CORRECT: ETag inner, Compression outer.
// ETag sees the uncompressed body, producing a stable ETag.
handler := Chain(mux, Compression(cfg), ETag(cfg))

// WRONG: Compression inner, ETag outer.
// ETag sees gzip-compressed bytes (includes metadata),
// producing a different ETag on every request.
handler := Chain(mux, ETag(cfg), Compression(cfg)) // don't do this
```

### Compression Extensibility

The default configuration supports **gzip and deflate**. Brotli, zstd, and other modern encodings are supported via the `WriterFactory` plugin interface without adding dependencies to the core library.

```go
cfg := httputil.DefaultCompressionConfig()
cfg.WriterFactories["zstd"] = func(dst io.Writer) (io.WriteCloser, error) {
    return zstd.NewWriter(dst)
}
handler := httputil.Compression(cfg)(mux)
```

This keeps the core package dependency-free while allowing you to plug in any encoder that implements `io.WriteCloser`.

For a complete guide including pool reuse and brotli/zstd patterns, see the [compression extensibility example](docs/integrations/brotli-zstd.md).

### Framework Integration

httputil composes cleanly with declarative API frameworks like [Huma](https://huma.rocks/), which generates OpenAPI 3.1 + JSON Schema from Go types but deliberately ships no middleware. Both target the same Go 1.22+ `http.ServeMux` via the [`humago`](https://pkg.go.dev/github.com/danielgtaylor/huma/v2/adapters/humago) adapter — no third-party router required:

```go
router := http.NewServeMux()
api := humago.New(router, huma.DefaultConfig("My API", "1.0.0"))
huma.Get(api, "/things/{id}", getThing) // huma: types, validation, OpenAPI

handler := httputil.Chain(router, // httputil: compression, security, logging
    httputil.Logging(slog.Default()),
    httputil.Recovery(slog.Default()),
    httputil.Compression(httputil.DefaultCompressionConfig()),
    httputil.SecurityHeaders(httputil.DefaultSecurityHeadersConfig()),
    httputil.CORS(httputil.DefaultCORSConfig()),
)
```

See the [full integration example](docs/integrations/huma.md) and the [detailed comparison](docs/research/2026-07-05_httputil-vs-huma.md).

For dependency injection and graceful lifecycle management, pair httputil with [samber/do](https://do.samber.dev/) — `httputil.Server.Shutdown(context.Context) error` satisfies `do.ShutdownerWithContextAndError` structurally, so the container discovers and shuts down the HTTP server automatically. See the [composition-root example](docs/integrations/samber-do.md).

For distributed rate limiting, see the [Redis-backed RateLimiter example](docs/integrations/redis-ratelimiter.md). For observability, see the [Prometheus MetricsRecorder example](docs/integrations/prometheus-metrics.md).

## Development

```bash
go test ./...             # Run tests
go test -race ./...       # Race detection
go vet ./...              # Vet
go test -bench=. ./...    # Benchmarks
golangci-lint run         # Lint (~70 linters)
```

## License

Proprietary — see [LICENSE](LICENSE). Contact `git@lars.software` for licensing inquiries.
