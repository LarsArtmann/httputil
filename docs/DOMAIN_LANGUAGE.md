# Domain Language — httputil

A **Unified Language** for `httputil` — shared across Contributor, Consumer (developer using this library), Reviewer, and AI.
Inspired by Domain-Driven Design (DDD) Ubiquitous Language.

Every term below should mean the **same thing** to everyone who reads it.
If a word means something different to a contributor than to a consumer, define it here.

---

## Glossary

| Term       | Definition                                                               | Context                                                      |
| ---------- | ------------------------------------------------------------------------ | ------------------------------------------------------------ |
| httputil   | A Go library providing composable HTTP middleware and utility primitives | The project itself; module `github.com/larsartmann/httputil` |
| Consumer   | A developer who imports and uses `httputil` in their Go HTTP service     | Library user                                                 |
| Middleware | A function that wraps an `http.Handler` to intercept/modify request flow | Signature: `func(http.Handler) http.Handler`                 |
| Request    | An incoming `*http.Request` being processed by the HTTP server           | Go `net/http` standard library                               |
| Response   | An `http.ResponseWriter` used to construct the HTTP response             | Go `net/http` standard library                               |

---

## Bounded Contexts

The library has these bounded contexts, each with a distinct vocabulary and responsibility.

| Context          | Description                                                                                        | Key Type(s)                                  |
| ---------------- | -------------------------------------------------------------------------------------------------- | -------------------------------------------- |
| Client IP        | Extracting the true client IP from proxied requests                                                | `ClientIP`                                   |
| CORS             | Configuring and enforcing Cross-Origin Resource Sharing policy                                     | `CORSConfig`, `CORS`                         |
| Response Capture | Recording response state for inspection (status, headers, body)                                    | `ResponseRecorder`, `Chain`                  |
| Error Protocol   | Classified errors with behavioral families for retry decisions                                     | Error codes, `go-error-family`               |
| Security Headers | Setting common browser security headers on responses                                               | `SecurityHeadersConfig`                      |
| Request ID       | Propagating or generating unique request identifiers                                               | `RequestIDConfig`                            |
| Recovery         | Catching panics and returning 500 responses                                                        | `Recovery`                                   |
| Timeout          | Enforcing request deadlines via context cancellation                                               | `Timeout`                                    |
| Compression      | Response compression (gzip/deflate/brotli/zstd + pluggable encodings) with pool-based writer reuse | `CompressionConfig`                          |
| Decompression    | Request body decompression (gzip/deflate) with decompression bomb protection                       | `DecompressionConfig`                        |
| Logging          | Structured request/response logging                                                                | `Logging`                                    |
| Server Lifecycle | HTTP server start, graceful shutdown, and configuration                                            | `ServerConfig`, `Server`                     |
| Health           | Kubernetes-compatible health, liveness, and readiness endpoints                                    | `HealthHandler`, `ReadyHandlerWithProbe`     |
| Rate Limiting    | Per-key rate limiting with O(log n) eviction, MaxKeys cap, and lazy TTL sweep                      | `KeyedRateLimiterConfig`, `KeyedRateLimiter` |
| Metrics          | Request metrics recording with pluggable recorder interface                                        | `MetricsConfig`, `MetricsRecorder`           |
| CSRF Protection  | Double-submit cookie CSRF defense backed by justinas/nosurf with HTMX-aware helpers                | `CSRFConfig`, `CSRFMiddleware`               |
| Server-Timing    | W3C Server-Timing header instrumentation with CRLF-safe values and context-aware measurement       | `ServerTiming`, `ServerTimingMiddleware`     |
| Conditional Requests | ETag generation and If-None-Match conditional GET handling (thin adapter over the independent go-etag module) | `ETag`, `etag.ETagConfig`                    |
| Body Size Limit  | Enforcing maximum request body size                                                                | `MaxBodySize`                                |
| Query Parameters | Parsing typed values from URL query parameters                                                     | `ParseUintQuery`                             |
| Middleware Stack | Named middleware ordering with duplicate prevention                                                | `MiddlewareStack`                            |
| HTTP Spec        | Reusable BDD-style HTTP behavior specifications                                                    | `httpspec.Run`, `httpspec.Spec`              |

---

## Entities

Objects with identity and lifecycle within the library.

| Term                   | Definition                                                                                        | Context          |
| ---------------------- | ------------------------------------------------------------------------------------------------- | ---------------- |
| ResponseRecorder       | A wrapping `http.ResponseWriter` that captures the status code and write state                    | Response Capture |
| CORSConfig             | A configuration value object defining CORS policy (origins, methods, headers, etc.)               | CORS             |
| SecurityHeadersConfig  | A configuration value object defining which security headers to set                               | Security Headers |
| RequestIDConfig        | A configuration value object defining request ID header name and generation logic                 | Request ID       |
| CompressionConfig      | A configuration value object defining compression parameters (encodings, level, min size)         | Compression      |
| DecompressionConfig    | A configuration value object defining decompression parameters (encodings, bomb-protection limit) | Decompression    |
| RateLimitConfig        | A configuration value object defining deprecated token-bucket rate limiting policy                | Rate Limiting    |
| TokenBucketLimiter     | A deprecated in-memory token bucket rate limiter with per-key buckets (removal at v1.0)           | Rate Limiting    |
| KeyedRateLimiterConfig | A configuration value object defining keyed rate limiting policy (limit, window, burst, keys)     | Rate Limiting    |
| KeyedRateLimiter       | A per-key rate limiter with O(log n) min-heap eviction, MaxKeys cap, and monitoring API           | Rate Limiting    |
| CSRFConfig             | A configuration value object defining CSRF policy (cookie, headers, trusted origins/proxies)      | CSRF Protection  |
| ServerTiming           | A per-request timing collector injected via context for handler-internal sub-metrics              | Server-Timing    |
| ETagConfig             | A configuration value object (in go-etag) defining ETag strength, buffer size, hash function, and skip rules | Conditional Requests |
| MetricsConfig          | A configuration value object defining metrics recording behavior                                  | Metrics          |
| ServerConfig           | A configuration value object defining server address, timeouts, and TLS settings                  | Server Lifecycle |
| MiddlewareStack        | A named middleware collection with duplicate prevention and ordering validation                   | Middleware Stack |

---

## Value Objects

Immutable objects defined by their attributes.

| Term                   | Definition                                                                                             | Context          |
| ---------------------- | ------------------------------------------------------------------------------------------------------ | ---------------- |
| Client IP              | The extracted IP address string identifying the originating client                                     | Client IP        |
| Origin                 | The value of the `Origin` request header; identifies the requesting site's scheme+host+port            | CORS             |
| Allowed Origin         | An origin string permitted by the CORS policy; `*` means any origin is allowed                         | CORS             |
| Preflight Request      | An `OPTIONS` request sent by the browser before the actual cross-origin request                        | CORS             |
| Actual Request         | The real request (GET, POST, etc.) following a successful preflight                                    | CORS             |
| Status Code            | The HTTP status code captured by the ResponseRecorder (e.g., 200, 404)                                 | Response Capture |
| Write State            | Whether `WriteHeader` has been called on the ResponseRecorder                                          | Response Capture |
| Request ID             | A unique string identifying a request, propagated via header or generated                              | Request ID       |
| Compression Level      | An integer controlling the compression tradeoff (speed vs ratio)                                       | Compression      |
| Min Size               | The minimum response body size (bytes) before compression is applied                                   | Compression      |
| Max Decompression Size | The maximum decompressed body size (bytes) before the bomb-protection limit triggers (default: 16 MiB) | Decompression    |
| Decompression Bomb     | A small compressed payload that decompresses to an enormous size, designed to exhaust server memory    | Decompression    |
| Token Bucket           | A per-key container holding token count and last-refill timestamp                                      | Rate Limiting    |
| Eviction TTL           | Duration after which idle rate-limit entries are lazily removed; zero disables eviction                | Rate Limiting    |
| Max Keys               | Caps the number of tracked rate-limit keys; oldest is evicted at capacity (zero = unbounded)           | Rate Limiting    |
| Key Extractor          | A function type extracting the rate-limit key from a request (RemoteAddr or ClientIP)                  | Rate Limiting    |
| Retry-After            | Duration until a rejected rate-limited client may retry; sent as an HTTP response header               | Rate Limiting    |
| CSRF Token             | A cryptographically random nonce stored in a cookie and submitted with each state-changing request     | CSRF Protection  |
| Double-Submit Cookie   | CSRF defense pattern: token sent in both cookie and request header/body for comparison                 | CSRF Protection  |
| Trusted Origin         | An origin explicitly allowed for cross-domain CSRF validation                                          | CSRF Protection  |
| Trusted Proxy          | An IP/CIDR of a reverse proxy that may strip or overwrite origin/protocol headers                      | CSRF Protection  |
| Server-Timing Metric   | A named sub-measurement within a single request's Server-Timing header (name + duration)               | Server-Timing    |
| Entity Tag (ETag)       | An opaque string validator identifying a specific version of a representation, sent in the ETag response header | Conditional Requests |
| Validator Strength      | Whether an ETag is strong or weak per RFC 7232 §2.1; strong ETags change on any byte change             | Conditional Requests |
| Conditional Request     | A request carrying If-None-Match/If-Match headers that the server evaluates against the resource's ETag | Conditional Requests |
| If-None-Match           | A request header listing ETags the client already holds; a match yields 304 Not Modified               | Conditional Requests |
| Health Status          | The operational state reported by health endpoints: `up` or `down`                                     | Health           |
| Ready Probe            | A function that returns true when the service is ready to accept traffic                               | Health           |

---

## Commands

Actions the library performs.

| Term                                 | Definition                                                                                             | Context          |
| ------------------------------------ | ------------------------------------------------------------------------------------------------------ | ---------------- |
| `ClientIP(r)`                        | Extract the client IP from a request using header precedence: X-Forwarded-For → X-Real-IP → RemoteAddr | Client IP        |
| `ClientIPMiddleware(next)`           | Create middleware that injects the client IP into the request context                                  | Client IP        |
| `ClientIPFromContext(ctx)`           | Retrieve the stored client IP from a context                                                           | Client IP        |
| `WithClientIP(ctx, ip)`              | Store a client IP string in a context                                                                  | Client IP        |
| `CORS(cfg)`                          | Create middleware that sets CORS response headers and handles preflight requests                       | CORS             |
| `DefaultCORSConfig()`                | Return a permissive CORS config suitable for local development (allows all origins)                    | CORS             |
| `NewResponseRecorder(w)`             | Create a ResponseRecorder wrapping the given ResponseWriter, defaulting to unwritten state             | Response Capture |
| `Chain(handler, mw...)`              | Compose multiple middleware around a handler; first middleware in list is outermost                    | Response Capture |
| `HeaderSnapshot(rec)`                | Return an isolated copy of the response headers from a ResponseRecorder                                | Response Capture |
| `SecurityHeaders(cfg)`               | Create middleware that sets security response headers (nosniff, frame-options, etc.)                   | Security Headers |
| `DefaultSecurityHeadersConfig()`     | Return a SecurityHeadersConfig with production defaults                                                | Security Headers |
| `RequestID(cfg)`                     | Create middleware that propagates or generates a request ID                                            | Request ID       |
| `DefaultRequestIDConfig()`           | Return a RequestIDConfig that reads/generates X-Request-ID                                             | Request ID       |
| `RequestIDFromContext(ctx)`          | Retrieve the stored request ID from a context                                                          | Request ID       |
| `Recovery(logger)`                   | Create middleware that catches panics, logs the stack trace, and returns 500                           | Recovery         |
| `Timeout(duration)`                  | Create middleware that sets a deadline on the request context                                          | Timeout          |
| `Logging(logger)`                    | Create middleware that logs each request with method, path, status, duration, and client IP            | Logging          |
| `Compression(cfg)`                   | Create middleware that compresses responses based on Accept-Encoding negotiation                       | Compression      |
| `DefaultCompressionConfig()`         | Return a CompressionConfig with sensible defaults (default level, 512-byte minimum)                    | Compression      |
| `Decompression(cfg)`                 | Create middleware that decompresses request bodies based on Content-Encoding with bomb protection      | Decompression    |
| `DefaultDecompressionConfig()`       | Return a DecompressionConfig with gzip+deflate and 16 MiB bomb-protection limit                        | Decompression    |
| `HealthHandler()`                    | Return a handler that responds with `{"status":"up"}`                                                  | Health           |
| `LiveHandler()`                      | Alias for `HealthHandler()` for Kubernetes liveness probes                                             | Health           |
| `ReadyHandler()`                     | Return a handler for Kubernetes readiness probes (always up by default)                                | Health           |
| `ReadyHandlerWithProbe(ready)`       | Return a handler that calls `ready()` and responds 200 up or 503 down                                  | Health           |
| `RegisterHealth(mux)`                | Register `/health`, `/health/live`, `/health/ready` on a ServeMux                                      | Health           |
| `NewTokenBucketLimiter(rate,burst)`  | Create an in-memory token bucket rate limiter (returns error if rate/burst <= 0) _(deprecated)_        | Rate Limiting    |
| `RateLimit(cfg)`                     | Create middleware that enforces rate limiting using the configured limiter _(deprecated)_              | Rate Limiting    |
| `DefaultRateLimitConfig()`           | Return a RateLimitConfig with 429 status and RemoteAddr key func _(deprecated)_                        | Rate Limiting    |
| `NewKeyedRateLimiter(cfg)`           | Create a keyed rate limiter with O(log n) eviction, MaxKeys cap, and monitoring API                    | Rate Limiting    |
| `KeyedRateLimiterMiddleware(cfg)`    | Create middleware enforcing per-key rate limits with Retry-After on rejection                          | Rate Limiting    |
| `DefaultKeyedRateLimiterConfig()`    | Return a KeyedRateLimiterConfig with sensible defaults (100 req/min, ClientIP key)                     | Rate Limiting    |
| `KeyExtractorFromRemoteAddr()`       | Return a KeyExtractor that uses the request RemoteAddr for rate-limit keying                           | Rate Limiting    |
| `KeyExtractorFromClientIP()`         | Return a KeyExtractor that uses the extracted ClientIP for rate-limit keying                           | Rate Limiting    |
| `CSRFMiddleware(cfg)`                | Create double-submit cookie CSRF middleware backed by nosurf                                           | CSRF Protection  |
| `CSRFResponseHeaderMiddleware(next)` | Create middleware that auto-sets the CSRF token in response headers for HTMX consumption               | CSRF Protection  |
| `ValidateCSRF(r, cfg)`               | Validate a request's CSRF token; returns (ok, responseRecorder) for standalone use                     | CSRF Protection  |
| `ConfigureNosurfHandler(h, cfg)`     | Configure the underlying nosurf handler with cookie, header, and origin settings                       | CSRF Protection  |
| `WithCSRFToken(ctx, token)`          | Store a CSRF token in a context                                                                        | CSRF Protection  |
| `CSRFTokenFromContext(ctx)`          | Retrieve the stored CSRF token from a context                                                          | CSRF Protection  |
| `CSRFTokenFromRequest(r)`            | Extract the CSRF token from a request (header or cookie)                                               | CSRF Protection  |
| `CSRFTokenHXHeaders(token)`          | Generate an `hx-headers` attribute string containing the CSRF token for HTMX                           | CSRF Protection  |
| `CSRFTokenHTMLMeta(token)`           | Generate an HTML `<meta>` tag containing the CSRF token for templ rendering                            | CSRF Protection  |
| `CSRFTokenFormField(token)`          | Generate an HTML hidden `<input>` field containing the CSRF token                                      | CSRF Protection  |
| `InvalidateCSRFCookie(w, cfg)`       | Expire the CSRF cookie to force token rotation (e.g., on login/logout)                                 | CSRF Protection  |
| `TranslateCSRFHeaders(h)`            | Translate HTMX-style CSRF headers to the canonical header name expected by nosurf                      | CSRF Protection  |
| `SetPlaintextHTTPOrigin()`           | Configure the package to use plaintext HTTP origin for local development                               | CSRF Protection  |
| `NewServerTiming()`                  | Create a ServerTiming collector for manual wrapping (without middleware)                               | Server-Timing    |
| `ServerTimingMiddleware()`           | Create middleware that injects a ServerTiming collector via context and writes the header on response  | Server-Timing    |
| `ServerTimingMiddlewareWhen(pred)`   | Create conditional Server-Timing middleware that only activates when the predicate returns true        | Server-Timing    |
| `MeasureServerTiming(ctx, name)`     | Start a named sub-metric timer; returns a stop function that records the elapsed duration              | Server-Timing    |
| `WrapServerTiming(w, r)`             | Manually wrap a ResponseWriter with Server-Timing instrumentation (without middleware)                 | Server-Timing    |
| `RecordServerTiming(ctx, name, dur)` | Record a named sub-metric with an explicit duration (no timer needed)                                  | Server-Timing    |
| `WithServerTiming(ctx, st)`          | Store a ServerTiming collector in a context                                                            | Server-Timing    |
| `ServerTimingFromContext(ctx)`       | Retrieve the ServerTiming collector from a context                                                     | Server-Timing    |
| `ETag(cfg)`                          | Create ETag middleware (thin adapter over go-etag) that generates ETags and serves 304 on If-None-Match | Conditional Requests |
| `Metrics(cfg)`                       | Create middleware that records request metrics via a pluggable recorder                                | Metrics          |
| `MaxBodySize(limit)`                 | Create middleware that rejects request bodies exceeding the limit                                      | Body Size Limit  |
| `MaxBodySizeMiddleware(cfg)`         | Create body-size middleware from a validated `MaxBodySizeConfig`                                       | Body Size Limit  |
| `MaxBodySizeConfig`                  | Configuration struct for body-size middleware with `Validate()`                                        | Body Size Limit  |
| `ServerConfig.ShutdownTimeout`       | Duration the server waits for in-flight requests during graceful shutdown (default: 30s)               | Server Lifecycle |
| `NewServer(cfg)`                     | Create an HTTP server with configurable timeouts and graceful shutdown                                 | Server Lifecycle |
| `NewMiddlewareStack()`               | Create a named middleware stack with duplicate prevention and ordering validation                      | Middleware Stack |
| `RegisterErrorClassifications()`     | Register stdlib HTTP error sentinels and message templates with go-error-family                        | Error Protocol   |
| `Validate()`                         | Check a config for invalid values at startup; all config types implement this                          | Universal        |
| `ParseUintQuery(r, key)`             | Parse a uint value from a query parameter; returns 0 on missing, empty, or invalid values              | Query Parameters |

---

## Events

State transitions within the library.

| Term                          | Definition                                                                                                | Context          |
| ----------------------------- | --------------------------------------------------------------------------------------------------------- | ---------------- |
| Header Written                | `WriteHeader` called on ResponseRecorder; status is now captured and immutable                            | Response Capture |
| Body Written                  | `Write` called on ResponseRecorder; implicitly sets status 200 if not yet written                         | Response Capture |
| Preflight Handled             | CORS middleware intercepts an OPTIONS request and returns 204 No Content                                  | CORS             |
| Request Passed                | CORS middleware delegates to the next handler (non-OPTIONS or passthrough mode)                           | CORS             |
| Request ID Generated          | RequestID middleware generates a new random ID because no header was present                              | Request ID       |
| Request ID Propagated         | RequestID middleware forwards an existing ID from the request header                                      | Request ID       |
| Panic Recovered               | Recovery middleware catches a panic, logs it, and writes 500                                              | Recovery         |
| Deadline Exceeded             | Timeout middleware's context deadline is reached; handler should stop work                                | Timeout          |
| Request Logged                | Logging middleware records the request method, path, status, and duration                                 | Logging          |
| Security Headers Set          | SecurityHeaders middleware writes security headers before delegating                                      | Security Headers |
| Compression Applied           | Compression middleware encodes the response body using the negotiated encoding                            | Compression      |
| Compression Skipped           | Compression middleware passes through (no gzip accept, below min size, non-2xx)                           | Compression      |
| Body Decompressed             | Decompression middleware wraps r.Body with a decompressing reader and removes encoding headers            | Decompression    |
| Decompression Bomb Detected   | Decompressed body exceeds MaxDecompressionSize; reads return error and the underlying reader is closed    | Decompression    |
| Rate Limited                  | RateLimit middleware rejects a request because the token bucket was empty _(deprecated)_                  | Rate Limiting    |
| Bucket Evicted                | An idle token bucket is removed during lazy sweep (EvictionTTL > 0) _(deprecated)_                        | Rate Limiting    |
| Key Rate Limited              | KeyedRateLimiterMiddleware rejects a request with 429 + Retry-After because the key's budget is exhausted | Rate Limiting    |
| Key Evicted                   | The oldest rate-limit key is evicted from the min-heap when MaxKeys capacity is reached                   | Rate Limiting    |
| CSRF Token Validated          | nosurf validates the double-submit token (cookie matches header/form); request proceeds                   | CSRF Protection  |
| CSRF Rejected                 | nosurf detects a missing, malformed, or mismatched CSRF token; returns 403 Forbidden                      | CSRF Protection  |
| CSRF Token Rotated            | InvalidateCSRFCookie expires the cookie, forcing a new token on the next request                          | CSRF Protection  |
| Server-Timing Metric Recorded | A named sub-metric's duration is captured in the ServerTiming collector via stop() or RecordServerTiming  | Server-Timing    |
| Server-Timing Header Emitted  | The ServerTimingMiddleware writes the W3C Server-Timing response header with CRLF-safe sanitized values   | Server-Timing    |
| ETag Generated                | ETag middleware computes an ETag from the response body and writes the ETag response header              | Conditional Requests |
| Not Modified Served           | ETag middleware matches If-None-Match and responds 304 Not Modified with an empty body                  | Conditional Requests |
| Health Checked                | Health endpoint responds with current status                                                              | Health           |
| Readiness Failed              | ReadyHandlerWithProbe calls ready() and it returns false; responds 503                                    | Health           |
| Metrics Recorded              | Metrics middleware records request duration, status, and method                                           | Metrics          |
| Body Rejected                 | MaxBodySize middleware rejects a request body exceeding the configured limit                              | Body Size Limit  |
| Server Starting               | Server begins listening on the configured address                                                         | Server Lifecycle |
| Server Shutting Down          | Server enters graceful shutdown, draining in-flight requests                                              | Server Lifecycle |

---

## Rules

Invariants and policies that the library enforces.

### Client IP Extraction Order

1. `X-Forwarded-For` header — use the **first** entry in the comma-separated list
2. `X-Real-IP` header — use the trimmed value directly
3. `RemoteAddr` — strip the port via `net.SplitHostPort`; fall back to raw value on error

> **Security Warning:** `ClientIP` trusts proxy headers without validation. Only use behind a reverse proxy that strips/overwrites these headers.

### CORS Policy

- If `AllowAllOrigins` is true → always respond with `Access-Control-Allow-Origin: *`
- If the request `Origin` matches an entry in `AllowedOrigins` → echo that origin back
- Wildcard patterns like `*.example.com` match subdomains
- If no match → fall back to `*` (default), or suppress the header entirely when `DenyUnmatched=true`
- Preflight `OPTIONS` requests receive `204 No Content` (unless `OptionsPassthrough` is set)
- `MaxAge` is sent as `Access-Control-Max-Age` in seconds (default: 86400 = 24 hours)
- `Validate()` rejects `AllowCredentials=true` with `AllowAllOrigins=true` (browsers reject this)
- `Validate()` rejects negative `MaxAge`

### ResponseRecorder Invariants

- `WriteHeader` only captures on **first call**; subsequent calls are ignored for capture but still delegated
- `Write` implicitly sets status 200 if no `WriteHeader` was called yet
- `Flush` and `Hijack` are optional — they delegate only if the underlying ResponseWriter supports them
- `Hijack` returns a classified `Infrastructure` error if the underlying writer is not an `http.Hijacker`
- Write/Hijack failures return classified `Transient` errors wrapping the underlying cause
- All errors carry an error code (e.g., `http.write_failed`), family, and relevant context
- `errors.Is(err, http.ErrNotSupported)` still works for unsupported Hijack

### Security Headers Defaults

- `X-Content-Type-Options: nosniff` — enabled by default
- `X-Frame-Options: DENY` — enabled by default
- `Referrer-Policy: strict-origin-when-cross-origin` — enabled by default
- `Strict-Transport-Security` — not set by default (requires site-specific configuration)
- `Content-Security-Policy` — not set by default (requires site-specific configuration)
- All headers are opt-in/opt-out via `SecurityHeadersConfig` fields

### Request ID Rules

- If the `IncomingHeader` is present on the request, its value is used as the request ID
- If absent, `GenerateID` is called to produce a new ID
- The ID is stored in the request context and set as a response header named `ResponseHeader`
- `Validate()` rejects nil `GenerateID`, empty `ResponseHeader`, and empty `IncomingHeader`

### Recovery Rules

- Catches any panic in the downstream handler chain
- Logs the panic value and stack trace via the provided `*slog.Logger`
- Writes `500 Internal Server Error` to the response
- Does not recover `http.ErrAbortHandler` (propagates it instead)

### Timeout Rules

- Sets a deadline on the request context using `context.WithTimeout`
- The handler must respect context cancellation (check `ctx.Done()` or `ctx.Err()`)
- Does not write a response itself — the handler or downstream middleware is responsible

### Logging Rules

- Logs after the downstream handler completes (captures final status)
- Fields: method, path, status, duration, client IP
- Uses `slog.Logger.Info` for all requests

### Compression Rules

- Negotiates between configured encodings (gzip, deflate by default) based on client `Accept-Encoding` and RFC 7231 q-values
- Server priority order: brotli > zstd > gzip > deflate > identity (only gzip and deflate are bundled)
- Skips responses smaller than `MinSize` (default: 512 bytes)
- Skips known-incompressible content types (images, video, audio, pre-compressed types)
- Uses per-encoding `sync.Pool` to reuse writer instances; custom factories opt into pooling via `Reset(io.Writer)`
- `Level` field controls compression level when `WriterFactories` is not supplied (applies to both gzip and deflate)
- `Validate()` rejects compression levels outside `[gzip.HuffmanOnly, gzip.BestCompression]`, negative `MinSize`, and empty `WriterFactories`

### Decompression Rules

- Decompresses request bodies for gzip and deflate encodings based on the `Content-Encoding` header
- Replaces `r.Body` with a decompressing reader so downstream handlers see the decompressed body transparently
- Removes `Content-Encoding` and `Content-Length` headers from the request after decompression
- Enforces a `MaxDecompressionSize` limit (default: 16 MiB) to prevent decompression bomb attacks
- When the limit is exceeded, reads return `errDecompressionSizeExceeded` and the underlying reader is closed immediately
- `Encodings` controls which encodings are accepted; empty means both gzip and deflate
- `Validate()` rejects negative `MaxDecompressionSize`

### Middleware Chaining

- `Chain` applies middleware in **reverse order** so the first middleware in the variadic list is the outermost handler
- Execution order: first middleware → ... → last middleware → final handler

### Server Lifecycle Rules

- `NewServer(cfg)` creates an `http.Server` with configurable read/write/idle timeouts
- `Start()` begins listening; blocks until the server stops
- `Shutdown(ctx)` gracefully drains in-flight requests within the context deadline
- `Addr()` returns the actual listening address (useful when port 0 is assigned)

### Health Rules

- `/health` and `/health/live` always return `{"status":"up"}` with 200
- `/health/ready` returns `{"status":"up"}` with 200 by default
- `ReadyHandlerWithProbe(ready)` returns 503 `{"status":"down"}` when `ready()` returns false
- All health responses use `Content-Type: application/json`

### Rate Limiting Rules

- `TokenBucketLimiter` creates per-key token buckets; tokens refill at the configured rate up to burst capacity _(deprecated — use `KeyedRateLimiter`)_
- `NewTokenBucketLimiter` rejects rate <= 0 or burst <= 0
- `EvictionTTL` (zero by default) controls lazy eviction of idle buckets — non-zero enables sweeping
- Each `Allow(key)` consumes one token; returns false when the bucket is empty
- Custom `RateLimiter` implementations can replace the in-memory limiter (e.g., Redis-backed)

### Keyed Rate Limiting Rules

- `KeyedRateLimiter` enforces a `Limit` of requests per `Window` per key, with optional `Burst` capacity
- `KeyExtractor` determines the key: `KeyExtractorFromClientIP()` (default) or `KeyExtractorFromRemoteAddr()`
- `MaxKeys` caps the number of tracked keys; when capacity is reached, the oldest-accessed key is evicted via O(log n) min-heap pop
- `TTL` enables lazy time-based eviction: entries idle longer than TTL are swept on the next `Allow` check
- Rejected requests receive `429 Too Many Requests` with a `Retry-After` header indicating when to retry
- `ActiveKeys()` returns the current count of tracked keys for monitoring
- `OnAllowed` and `OnRejected` callbacks provide hooks for metrics/logging without custom middleware
- `RejectionHandler` allows a custom 429 response body (defaults to plain text)

### CSRF Protection Rules

- Double-submit cookie pattern: a random token is set in a cookie and must match the token in the request header or form field
- `CSRFMiddleware` wraps `justinas/nosurf` and applies to all non-safe methods (POST, PUT, DELETE, PATCH)
- GET, HEAD, OPTIONS, and TRACE requests are exempt from CSRF validation
- `CSRFConfig.Validate()` rejects insecure configurations (`SameSite=None` requires `Secure=true`)
- `TrustedOrigins` explicitly allows cross-domain origins for CSRF validation
- `TrustedProxies` defines CIDR ranges of reverse proxies that may set `X-Forwarded-Proto`
- `AllowPlaintextBypass` permits plaintext HTTP origin for local development (insecure; off by default)
- `TranslateCSRFHeaders` maps HTMX-style headers (`X-CSRF-Token` from `HX-Request`) to the canonical name nosurf expects
- `InvalidateCSRFCookie` expires the cookie to force token rotation on login/logout
- CSRF errors are classified: `ErrCSRFInvalid` (Rejection family) and `ErrCSRFConfig` (Infrastructure family)

### Server-Timing Rules

- `ServerTimingMiddleware` injects a `*ServerTiming` into the request context and writes the header on response
- Use `MeasureServerTiming(ctx, name)` to start a named timer; the returned `stop()` function records the duration
- Use `RecordServerTiming(ctx, name, duration)` for pre-measured durations (no timer needed)
- `ServerTimingMiddlewareWhen(pred)` gates the middleware behind a predicate (e.g., admin-only or debug-query)
- `WrapServerTiming(w, r)` provides manual wrapping without the middleware
- Header values are sanitized against CRLF injection: quoted strings are escaped, raw CR/LF replaced
- `delegatingWriter` transparently delegates Hijacker, Flusher, and Pusher to the underlying ResponseWriter

### Conditional Requests Rules

- `ETag(cfg)` is a thin adapter over the independent `go-etag` module; the config type (`etag.ETagConfig`) and domain types (`etag.ETag`, `etag.ParseETag`) live in go-etag, not httputil
- For GET/HEAD requests, the middleware buffers the response body (up to `MaxBufferSize`, default 1 MB), computes an ETag via `HashFunc` (default FNV-64a), and writes the `ETag` response header
- When `If-None-Match` matches the computed ETag, the middleware responds `304 Not Modified` with an empty body
- Responses exceeding `MaxBufferSize` are streamed without an ETag (ETag generation abandoned)
- `Skip` excludes requests from buffering (e.g., large file downloads, SSE)
- `SkipIfPresent` respects a handler-set ETag instead of overwriting it
- `RegisterErrorClassifications` registers a strict superset of go-etag's error templates, so consumers call only the httputil registration once

### Metrics Rules

- `MetricsRecorder` is a pluggable interface for recording request metrics
- Records method, path, status, and duration for each request
- Default implementation is a no-op; consumers provide their own recorder

### Body Size Limit Rules

- Wraps the request body in `http.MaxBytesReader`
- Returns 413 Request Entity Too Large when the limit is exceeded

### Config Validation

- All config types implement `Validate() error` for startup validation
- `CORSConfig.Validate()` — rejects credentials+allorigins, negative MaxAge
- `CompressionConfig.Validate()` — rejects invalid gzip levels, negative MinSize
- `RequestIDConfig.Validate()` — rejects nil GenerateID, empty HeaderName, empty ForwardHeader
- `SecurityHeadersConfig.Validate()` — all fields optional, currently always valid

### Error Classification

| Error Code                   | Family         | Retryable | When                                                            |
| ---------------------------- | -------------- | --------- | --------------------------------------------------------------- |
| `http.write_failed`          | Transient      | Yes       | Underlying ResponseWriter.Write fails                           |
| `http.hijack_unsupported`    | Infrastructure | No        | Underlying writer doesn't implement Hijacker                    |
| `http.hijack_failed`         | Transient      | Yes       | Underlying Hijack call fails                                    |
| `http.compress_write_failed` | Transient      | Yes       | Compression writer Write fails                                  |
| `csrf_invalid`               | Rejection      | No        | CSRF token missing, malformed, or mismatched                    |
| `csrf_config`                | Infrastructure | No        | CSRF configuration invalid (e.g., SameSite=None without Secure) |
| `http.etag_write_failed`     | Transient      | Yes       | ETag writer fails to write buffered/streamed data              |
| `http.etag_config_invalid`   | Rejection      | No        | ETagConfig has an invalid field value                          |
| `http.etag_hash_write_failed`| Orchestration  | No        | Hash.Write fails, violating the hash.Hash contract             |

All classified errors implement `Coded`, `Classified`, `Contextual`, and `Retryable` from `go-error-family`.

---

## Conventions

Patterns consumers and contributors should follow.

| Convention             | Description                                                                                                             |
| ---------------------- | ----------------------------------------------------------------------------------------------------------------------- |
| Middleware signature   | Always `func(http.Handler) http.Handler` — the Go standard library convention                                           |
| Middleware type alias  | `type Middleware func(http.Handler) http.Handler` in `recorder.go`                                                      |
| Classified errors      | Errors from ResponseRecorder and CSRF use `go-error-family` for behavioral classification                               |
| Config validation      | All config types implement `Validate() error` for startup checks                                                        |
| `httputil` import name | Consumers import as `httputil`; no aliases needed                                                                       |
| Allowed dependencies   | `go-error-family`, `golang.org/x/time`, `justinas/nosurf`, and `go-etag` are the only external dependencies (enforced by depguard) |

---

> **How to use this file:**
>
> - Keep terms concise — one clear sentence per definition
> - Update when new domain concepts emerge
> - Use these terms consistently in code, docs, and conversations
> - When in doubt about a word's meaning, check here first
