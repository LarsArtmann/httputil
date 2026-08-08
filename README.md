# httputil

[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/httputil.svg)](https://pkg.go.dev/github.com/larsartmann/httputil)
[![Go Version](https://img.shields.io/badge/Go-1.26+-00ADD8)](https://go.dev)
[![Coverage](https://img.shields.io/badge/coverage-97.5%25-green)](#)
[![govulncheck](https://img.shields.io/badge/govulncheck-clean-brightgreen)](#)
[![License](https://img.shields.io/badge/license-Proprietary-red)](LICENSE)

Composable HTTP middleware, utility primitives, and server lifecycle helpers for Go — CORS, client IP extraction, response recording, middleware chaining, security headers, CSP nonce support, request ID, panic recovery, timeout enforcement, structured logging, response compression, request body decompression with bomb protection, ETag conditional requests (via go-etag adapter), W3C Server-Timing, CSRF protection (nosurf), keyed rate limiting, configurable HTTP server, and standard health checks.

Minimal footprint — four dependencies (`go-error-family` + `go-etag` same-author, `golang.org/x/time`, `justinas/nosurf`). Pure stdlib `net/http`. Go 1.26+.

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

### CSP Nonce

Generates a per-request cryptographic nonce for Content Security Policy, allowing specific inline `<script>` and `<style>` elements to execute while blocking all others — no `'unsafe-inline'` needed.

```go
handler := httputil.Nonce(httputil.DefaultNonceConfig())(mux)
```

Retrieve the nonce in handlers or templates:

```go
// In a handler:
nonce := httputil.NonceFromRequest(r)

// In a Go template:
//   <script {{ NonceAttr }}>...</script>
//   <style {{ NonceAttr }}>...</style>
attr := httputil.NonceAttr(r) // returns nonce="abc123"
```

Two built-in CSP builders:

- `RecommendedCSPWithNonce` (default): `script-src` and `style-src` with nonce.
- `ProductionCSPWithNonce`: adds `object-src 'none'`, `base-uri 'self'`, `frame-ancestors 'none'`.

```go
cfg := httputil.DefaultNonceConfig()
cfg.CSPBuilder = httputil.ProductionCSPWithNonce // stricter policy
handler := httputil.Nonce(cfg)(mux)
```

Use a custom CSP by setting `CSPBuilder`, or set it to `nil` to disable the header entirely (context-only mode for when you set CSP elsewhere).

> **Ordering:** Place `Nonce` **after** `SecurityHeaders` in the chain so the nonce-bearing CSP overwrites any static CSP. The default `SecurityHeadersConfig` does not set a CSP, so there is no conflict unless you explicitly set `ContentSecurityPolicy` in both.

> **Caching:** Responses with per-request nonces **must not be cached** — a cached page would serve a stale nonce that no longer matches the CSP header. Set `Cache-Control: no-store` in your handler or caching middleware when using nonce-based CSP.

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

### Request Body Decompression

Transparently decompresses request bodies based on the `Content-Encoding` header. Supports gzip and deflate. The middleware replaces `r.Body` with a decompressing reader and removes `Content-Encoding` and `Content-Length` headers so downstream handlers see the decompressed body transparently.

```go
handler := httputil.Decompression(httputil.DefaultDecompressionConfig())(mux)
```

**Bomb protection:** The decompressed body is limited to `MaxDecompressionSize` bytes (default: 16 MiB) to prevent decompression bomb attacks. When the limit is exceeded, reads return an error and the underlying reader is closed immediately. Set to `0` to disable the limit (not recommended).

Restrict which encodings are accepted by setting `Encodings`:

```go
cfg := httputil.DecompressionConfig{
    Encodings:            []string{"gzip"}, // deflate-only servers skip this
    MaxDecompressionSize: 1 << 20,           // 1 MiB limit
}
handler := httputil.Decompression(cfg)(mux)
```

Use `cfg.Validate()` to catch invalid configurations at startup (e.g., negative `MaxDecompressionSize`).

### ETag / Conditional Requests

ETag generation and `If-None-Match` handling via the [go-etag](https://github.com/larsartmann/go-etag) module. The middleware buffers GET/HEAD response bodies, computes an FNV-64a hash, and returns 304 Not Modified when the client's `If-None-Match` matches.

```go
import "github.com/larsartmann/go-etag"

handler := httputil.ETag(etag.DefaultETagConfig())(mux)
```

For domain types (`etag.ETag`, `etag.ParseETag`, `etag.MatchesIfNoneMatch`, etc.), import go-etag directly. The adapter only bridges the middleware to httputil's `Middleware` type so it composes with `Chain` and `MiddlewareStack`.

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

### CSRF Protection

Double-submit cookie CSRF protection via [justinas/nosurf](https://github.com/justinas/nosurf) with HTMX-aware helpers.

```go
handler := httputil.CSRFMiddleware(httputil.CSRFConfig{})(mux)
```

For HTMX, pair with `CSRFResponseHeaderMiddleware` to automatically set the token in response headers:

```go
handler := httputil.Chain(mux,
    httputil.CSRFMiddleware(httputil.CSRFConfig{}),
    httputil.CSRFResponseHeaderMiddleware,
)
```

Token helpers for templates: `CSRFTokenFormField` (hidden input), `CSRFTokenHTMLMeta` (meta tag), `CSRFTokenHXHeaders` (hx-headers attribute).

Call `InvalidateCSRFCookie(w, cfg)` on login/logout to rotate the token.

### Server-Timing

W3C Server-Timing header with per-request sub-metrics. Lives in the `server_timing` sub-module (`github.com/larsartmann/httputil/server_timing`, package `servertiming`).

```go
import "github.com/larsartmann/httputil/server_timing"

handler := servertiming.ServerTimingMiddleware()(mux)

// In a handler:
stop := servertiming.MeasureServerTiming(r.Context(), "db")
result, err := db.Query(ctx)
stop()
```

Gate behind a predicate to expose only for admin/debug requests:

```go
handler := servertiming.ServerTimingMiddlewareWhen(func(r *http.Request) bool {
    return r.URL.Query().Has("debug")
})(mux)
```

### Rate Limiting

Token-bucket rate limiting per client key with O(log n) min-heap eviction and monitoring.

```go
handler := httputil.KeyedRateLimiterMiddleware(httputil.DefaultKeyedRateLimiterConfig())(mux)
```

For monitoring active keys:

```go
rl := httputil.NewKeyedRateLimiter(httputil.DefaultKeyedRateLimiterConfig())
fmt.Println(rl.ActiveKeys())
handler := rl.Middleware()(mux)
```

Rejected requests receive `429 Too Many Requests` with a `Retry-After` header.

### Error Classification

`ResponseRecorder`, `compressWriter`, and `CSRFMiddleware` errors are classified with behavioral families via [go-error-family](https://github.com/larsartmann/go-error-family):

| Source     | Error Code                   | Family         | Retryable | When                                         |
| ---------- | ---------------------------- | -------------- | --------- | -------------------------------------------- |
| `Write`    | `http.write_failed`          | Transient      | Yes       | Underlying ResponseWriter.Write fails        |
| `Hijack`   | `http.hijack_unsupported`    | Infrastructure | No        | Underlying writer doesn't implement Hijacker |
| `Hijack`   | `http.hijack_failed`         | Transient      | Yes       | Underlying Hijack call fails                 |
| `Compress` | `http.compress_write_failed` | Transient      | Yes       | Compression writer Write/Close fails         |
| `CSRF`     | `csrf_invalid`               | Rejection      | No        | CSRF token missing, malformed, or mismatched |
| `CSRF`     | `csrf_config`                | Infrastructure | No        | CSRF configuration invalid                   |

Call `RegisterErrorClassifications()` at startup to enable classification of stdlib HTTP errors and register error message templates.

## API

| Function                         | Signature                                                             | Purpose                                               |
| -------------------------------- | --------------------------------------------------------------------- | ----------------------------------------------------- |
| `CORS`                           | `func(CORSConfig) func(http.Handler) http.Handler`                    | CORS middleware factory                               |
| `DefaultCORSConfig`              | `func() CORSConfig`                                                   | Permissive dev config (allows all origins)            |
| `ClientIP`                       | `func(*http.Request) string`                                          | Extract client IP from proxied request                |
| `ClientIPMiddleware`             | `func(http.Handler) http.Handler`                                     | Store client IP in request context                    |
| `ClientIPFromContext`            | `func(context.Context) string`                                        | Retrieve stored client IP                             |
| `WithClientIP`                   | `func(context.Context, string) context.Context`                       | Store client IP in context                            |
| `NewResponseRecorder`            | `func(http.ResponseWriter) *ResponseRecorder`                         | Wrap writer to capture status                         |
| `Chain`                          | `func(http.Handler, ...func(http.Handler) http.Handler) http.Handler` | Compose middleware                                    |
| `SecurityHeaders`                | `func(SecurityHeadersConfig) func(http.Handler) http.Handler`         | Security response headers                             |
| `DefaultSecurityHeadersConfig`   | `func() SecurityHeadersConfig`                                        | Sensible security defaults                            |
| `RequestID`                      | `func(RequestIDConfig) func(http.Handler) http.Handler`               | Request ID propagation/generation                     |
| `DefaultRequestIDConfig`         | `func() RequestIDConfig`                                              | Default X-Request-ID config                           |
| `RequestIDFromContext`           | `func(context.Context) string`                                        | Retrieve stored request ID                            |
| `Recovery`                       | `func(*slog.Logger) func(http.Handler) http.Handler`                  | Panic recovery                                        |
| `Timeout`                        | `func(time.Duration) func(http.Handler) http.Handler`                 | Request deadline enforcement                          |
| `Logging`                        | `func(*slog.Logger) func(http.Handler) http.Handler`                  | Structured request logging                            |
| `MaxBodySize`                    | `func(int64) func(http.Handler) http.Handler`                         | Request body size limit                               |
| `Compression`                    | `func(CompressionConfig) func(http.Handler) http.Handler`             | Negotiated response compression                       |
| `DefaultCompressionConfig`       | `func() CompressionConfig`                                            | gzip/deflate defaults                                 |
| `DefaultWriterFactories`         | `func() map[string]WriterFactory`                                     | Built-in gzip/deflate/identity factories              |
| `DefaultWriterFactoriesForLevel` | `func(int) map[string]WriterFactory`                                  | Built-in factories at a given compression level       |
| `GzipWriterFactory`              | `func(int) WriterFactory`                                             | Stdlib gzip factory at a given level                  |
| `DeflateWriterFactory`           | `func(int) WriterFactory`                                             | Stdlib flate/raw-deflate factory                      |
| `DefaultIncompressibleTypes`     | `func() []string`                                                     | Default content-type deny-list for compression        |
| `Decompression`                  | `func(DecompressionConfig) func(http.Handler) http.Handler`           | Request body decompression + bomb protection          |
| `DefaultDecompressionConfig`     | `func() DecompressionConfig`                                          | gzip/deflate defaults                                 |
| `ETag`                           | `func(etag.ETagConfig) Middleware`                                    | ETag adapter over go-etag                             |
| `RateLimit`                      | `func(RateLimitConfig) func(http.Handler) http.Handler`               | Token bucket rate limiting _(deprecated)_             |
| `DefaultRateLimitConfig`         | `func() RateLimitConfig`                                              | Default rate limit config _(deprecated)_              |
| `NewTokenBucketLimiter`          | `func(float64, int) (*TokenBucketLimiter, error)`                     | Token bucket limiter constructor _(deprecated)_       |
| `KeyedRateLimiterMiddleware`     | `func(KeyedRateLimiterConfig) func(http.Handler) http.Handler`        | Per-key rate limiting with eviction                   |
| `NewKeyedRateLimiter`            | `func(KeyedRateLimiterConfig) *KeyedRateLimiter`                      | Rate limiter with monitoring API                      |
| `DefaultKeyedRateLimiterConfig`  | `func() KeyedRateLimiterConfig`                                       | Default per-key rate limit config                     |
| `CSRFMiddleware`                 | `func(CSRFConfig) func(http.Handler) http.Handler`                    | CSRF protection middleware                            |
| `CSRFResponseHeaderMiddleware`   | `func(http.Handler) http.Handler`                                     | Auto-set CSRF token in response header                |
| `ValidateCSRF`                   | `func(*http.Request, CSRFConfig) (bool, *httptest.ResponseRecorder)`  | Standalone CSRF validation                            |
| `ServerTimingMiddleware`         | `func() func(http.Handler) http.Handler`                              | W3C Server-Timing middleware (`server_timing` module) |
| `ServerTimingMiddlewareWhen`     | `func(func(*http.Request) bool) func(http.Handler) http.Handler`      | Conditional Server-Timing (`server_timing` module)    |
| `ParseUintQuery`                 | `func(*http.Request, string) uint`                                    | Parse uint from query param                           |
| `Metrics`                        | `func(MetricsConfig) func(http.Handler) http.Handler`                 | Request metrics recording                             |
| `DefaultMetricsConfig`           | `func() MetricsConfig`                                                | Default metrics config                                |
| `NewMiddlewareStack`             | `func() *MiddlewareStack`                                             | Named middleware stack builder                        |
| `DetectCapabilities`             | `func(http.ResponseWriter) Capabilities`                              | Report Hijacker/Flusher support                       |
| `RegisterErrorClassifications`   | `func()`                                                              | Register stdlib error sentinels + templates           |
| `NewServer`                      | `func(ServerConfig, http.Handler) (*Server, error)`                   | Configurable HTTP server with timeouts                |
| `DefaultServerConfig`            | `func() ServerConfig`                                                 | Sensible server timeout defaults                      |
| `RegisterHealth`                 | `func(*http.ServeMux)`                                                | Register /health + /live + /ready                     |
| `HealthHandler`                  | `func() http.HandlerFunc`                                             | Simple `{"status":"up"}` handler                      |
| `LiveHandler`                    | `func() http.HandlerFunc`                                             | Kubernetes liveness probe handler                     |
| `ReadyHandler`                   | `func() http.HandlerFunc`                                             | Kubernetes readiness probe handler                    |
| `ReadyHandlerWithProbe`          | `func(func() bool) http.HandlerFunc`                                  | Readiness with dependency probe (200/503)             |

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

### `DecompressionConfig` fields

| Field                  | Type       | Default       | Description                                                                     |
| ---------------------- | ---------- | ------------- | ------------------------------------------------------------------------------- |
| `Encodings`            | `[]string` | gzip, deflate | Request body encodings to decompress; empty = both defaults                     |
| `MaxDecompressionSize` | `int64`    | `16777216`    | Max decompressed body size in bytes to prevent zip bombs (16 MiB); 0 = no limit |

### `RateLimitConfig` fields _(deprecated — use `KeyedRateLimiterConfig`)_

| Field      | Type                         | Default            | Description                                                       |
| ---------- | ---------------------------- | ------------------ | ----------------------------------------------------------------- |
| `Limiter`  | `RateLimiter`                | `nil`              | Decides whether to allow each request (required)                  |
| `KeyFunc`  | `func(*http.Request) string` | `nil` (RemoteAddr) | Extracts the rate-limiting key from the request (e.g., client IP) |
| `Status`   | `int`                        | `429`              | HTTP status when rate limited (ignored when `OnDenied` is set)    |
| `OnDenied` | `http.HandlerFunc`           | `nil`              | Custom handler for rejected requests; overrides default response  |

### `KeyedRateLimiterConfig` fields

| Field              | Type                          | Default                      | Description                                    |
| ------------------ | ----------------------------- | ---------------------------- | ---------------------------------------------- |
| `Limit`            | `uint`                        | `100`                        | Maximum requests per `Window` per key          |
| `Window`           | `time.Duration`               | `1m`                         | Time window for the limit                      |
| `Burst`            | `uint`                        | `0` (= Limit)                | Maximum burst size (can exceed Limit)          |
| `KeyExtractor`     | `KeyExtractor`                | `KeyExtractorFromClientIP()` | Extracts the rate-limit key from the request   |
| `TTL`              | `time.Duration`               | `10m`                        | How long idle entries are kept before eviction |
| `MaxKeys`          | `uint`                        | `0` (unbounded)              | Caps tracked keys; oldest evicted at capacity  |
| `OnAllowed`        | `func(*http.Request)`         | `nil`                        | Callback when a request passes                 |
| `OnRejected`       | `func(*http.Request, string)` | `nil`                        | Callback when rejected (receives retryAfter)   |
| `RejectionHandler` | `func(w, r, retryAfter)`      | `nil` (429 + Retry-After)    | Custom handler for rejected requests           |

### `CSRFConfig` fields

| Field                  | Type            | Default            | Description                                                            |
| ---------------------- | --------------- | ------------------ | ---------------------------------------------------------------------- |
| `CookieName`           | `string`        | `"csrf_token"`     | Name of the CSRF cookie                                                |
| `HeaderName`           | `string`        | `"X-CSRF-Token"`   | Request header containing the CSRF token                               |
| `FieldName`            | `string`        | `"csrf_token"`     | Form field name for the CSRF token                                     |
| `MaxAge`               | `time.Duration` | `24h`              | Cookie max age                                                         |
| `Secure`               | `bool`          | `false`            | Sets the Secure flag on the cookie (set `true` in production)          |
| `SameSite`             | `http.SameSite` | `SameSiteLaxMode`  | SameSite attribute on the cookie                                       |
| `Domain`               | `string`        | `""` (host-only)   | Cookie domain                                                          |
| `Path`                 | `string`        | `"/"`              | Cookie path                                                            |
| `TrustedOrigins`       | `[]string`      | `nil`              | Origins allowed for cross-domain CSRF                                  |
| `TrustedProxies`       | `[]string`      | `nil`              | IP/CIDR of reverse proxies that may strip origin headers               |
| `AllowPlaintextBypass` | `bool`          | `false`            | Allow plaintext-HTTP origin bypass for all non-TLS requests (insecure) |
| `ErrorHandler`         | `ErrorHandler`  | `nil` (403 + body) | Custom handler for CSRF validation failures                            |

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
- **Minimal dependencies** — `go-error-family` (same author, zero transitive deps), `go-etag` (same author, ETag conditional requests), `golang.org/x/time` (canonical Go rate-limit extension), and `justinas/nosurf` (CSRF protection).

### Middleware Ordering

`Chain` applies middleware in **reverse declaration order** so the first argument is outermost:

```go
handler := Chain(mux,
    Logging(logger),          // outermost — logs first, sees final status
    Recovery(logger),         // catches panics before they reach Logging
    Timeout(30*time.Second),  // enforces deadline
    RequestID(DefaultRequestIDConfig()),
    SecurityHeaders(DefaultSecurityHeadersConfig()),
    Nonce(DefaultNonceConfig()), // after SecurityHeaders so nonce CSP wins
    CORS(DefaultCORSConfig()),
)
```

**Decompression** should be placed outer so downstream middleware (e.g., `MaxBodySize`) sees the decompressed body size:

```go
// Decompression outer: handlers see decompressed bytes.
// MaxBodySize limits the decompressed size, not the compressed size.
handler := Chain(mux, Decompression(cfg), MaxBodySize(1<<20))
```

**ETag** must be placed inside (after) `Compression` so it hashes the uncompressed body. If placed before `Compression`, the ETag would be computed over the compressed bytes — different content encodings on the wire would yield different ETags for the same logical resource, defeating conditional caching:

```go
// Compression outer, ETag inner: ETag hashes the body the handler produced,
// not the bytes that left over the wire. All clients see the same ETag value
// regardless of which Accept-Encoding they negotiated.
handler := Chain(mux, Compression(cfg), ETag(etag.DefaultETagConfig()))
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

## Quality Gates

This project maintains strict quality standards enforced in CI:

| Gate             | Command                                    | Status                                           |
| ---------------- | ------------------------------------------ | ------------------------------------------------ |
| Tests            | `go test -race -count=1 ./...`             | Passing                                          |
| Race stress      | `go test -race -count=10 ./...`            | Passing                                          |
| Coverage         | `go test -coverprofile=coverage.out ./...` | 97.0% httputil / 99.3% httpspec (threshold: 95%) |
| Lint             | `golangci-lint run` (~70 linters)          | 0 issues                                         |
| Vet              | `go vet ./...`                             | Clean                                            |
| Vulnerabilities  | `govulncheck ./...`                        | None found                                       |
| Nix flake        | `nix flake check`                          | All checks passed                                |
| Module integrity | `go mod verify`                            | All modules verified                             |

## Development

```bash
go test ./...               # Run tests
go test -race ./...         # Race detection (REQUIRED for t.Parallel())
go test -race -count=10 ./...  # Stress race detection
go vet ./...                # Vet
go test -bench=. ./...      # Benchmarks
golangci-lint run           # Lint (~70 linters)
golangci-lint fmt           # Format (gofumpt + golines@120 + gci)
govulncheck ./...           # Vulnerability scan
nix flake check             # Nix flake validation
```

## License

Proprietary — see [LICENSE](LICENSE). Contact `git@lars.software` for licensing inquiries.
