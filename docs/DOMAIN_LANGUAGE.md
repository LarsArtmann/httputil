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

| Context          | Description                                                                                        | Key Type(s)                              |
| ---------------- | -------------------------------------------------------------------------------------------------- | ---------------------------------------- |
| Client IP        | Extracting the true client IP from proxied requests                                                | `ClientIP`                               |
| CORS             | Configuring and enforcing Cross-Origin Resource Sharing policy                                     | `CORSConfig`, `CORS`                     |
| Response Capture | Recording response state for inspection (status, headers, body)                                    | `ResponseRecorder`, `Chain`              |
| Error Protocol   | Classified errors with behavioral families for retry decisions                                     | Error codes, `go-error-family`           |
| Security Headers | Setting common browser security headers on responses                                               | `SecurityHeadersConfig`                  |
| Request ID       | Propagating or generating unique request identifiers                                               | `RequestIDConfig`                        |
| Recovery         | Catching panics and returning 500 responses                                                        | `Recovery`                               |
| Timeout          | Enforcing request deadlines via context cancellation                                               | `Timeout`                                |
| Compression      | Response compression (gzip/deflate/brotli/zstd + pluggable encodings) with pool-based writer reuse | `CompressionConfig`                      |
| ETag             | Entity tag generation and conditional 304 responses                                                | `ETagConfig`                             |
| Logging          | Structured request/response logging                                                                | `Logging`                                |
| Server Lifecycle | HTTP server start, graceful shutdown, and configuration                                            | `ServerConfig`, `Server`                 |
| Health           | Kubernetes-compatible health, liveness, and readiness endpoints                                    | `HealthHandler`, `ReadyHandlerWithProbe` |
| Rate Limiting    | Token bucket rate limiting with pluggable limiter and optional TTL eviction                        | `RateLimitConfig`, `TokenBucketLimiter`  |
| Metrics          | Request metrics recording with pluggable recorder interface                                        | `MetricsConfig`, `MetricsRecorder`       |
| Body Size Limit  | Enforcing maximum request body size                                                                | `MaxBodySize`                            |
| Query Parameters | Parsing typed values from URL query parameters                                                     | `ParseUintQuery`                         |
| Middleware Stack | Named middleware ordering with duplicate prevention                                                | `MiddlewareStack`                        |
| HTTP Spec        | Reusable BDD-style HTTP behavior specifications                                                    | `httpspec.Run`, `httpspec.Spec`          |

---

## Entities

Objects with identity and lifecycle within the library.

| Term                  | Definition                                                                                    | Context          |
| --------------------- | --------------------------------------------------------------------------------------------- | ---------------- |
| ResponseRecorder      | A wrapping `http.ResponseWriter` that captures the status code and write state                | Response Capture |
| CORSConfig            | A configuration value object defining CORS policy (origins, methods, headers, etc.)           | CORS             |
| SecurityHeadersConfig | A configuration value object defining which security headers to set                           | Security Headers |
| RequestIDConfig       | A configuration value object defining request ID header name and generation logic             | Request ID       |
| CompressionConfig     | A configuration value object defining compression parameters (encodings, level, min size)     | Compression      |
| ETagConfig            | A configuration value object defining ETag generation parameters (weak vs strong, max buffer) | ETag             |
| RateLimitConfig       | A configuration value object defining rate limiting policy (limiter, key func, denial)        | Rate Limiting    |
| TokenBucketLimiter    | An in-memory token bucket rate limiter with per-key buckets and optional TTL eviction         | Rate Limiting    |
| MetricsConfig         | A configuration value object defining metrics recording behavior                              | Metrics          |
| ServerConfig          | A configuration value object defining server address, timeouts, and TLS settings              | Server Lifecycle |
| MiddlewareStack       | A named middleware collection with duplicate prevention and ordering validation               | Middleware Stack |

---

## Value Objects

Immutable objects defined by their attributes.

| Term              | Definition                                                                                    | Context          |
| ----------------- | --------------------------------------------------------------------------------------------- | ---------------- |
| Client IP         | The extracted IP address string identifying the originating client                            | Client IP        |
| Origin            | The value of the `Origin` request header; identifies the requesting site's scheme+host+port   | CORS             |
| Allowed Origin    | An origin string permitted by the CORS policy; `*` means any origin is allowed                | CORS             |
| Preflight Request | An `OPTIONS` request sent by the browser before the actual cross-origin request               | CORS             |
| Actual Request    | The real request (GET, POST, etc.) following a successful preflight                           | CORS             |
| Status Code       | The HTTP status code captured by the ResponseRecorder (e.g., 200, 404)                        | Response Capture |
| Write State       | Whether `WriteHeader` has been called on the ResponseRecorder                                 | Response Capture |
| Request ID        | A unique string identifying a request, propagated via header or generated                     | Request ID       |
| ETag Value        | An opaque string identifying a specific version of a response body                            | ETag             |
| Weak ETag         | An ETag prefixed with `W/` indicating semantic equivalence rather than byte-for-byte identity | ETag             |
| Compression Level | An integer controlling the compression tradeoff (speed vs ratio)                              | Compression      |
| Min Size          | The minimum response body size (bytes) before compression is applied                          | Compression      |
| Max Buffer Size   | The maximum bytes buffered for ETag computation before abandoning                             | ETag             |
| Token Bucket      | A per-key container holding token count and last-refill timestamp                             | Rate Limiting    |
| Eviction TTL      | Duration after which idle token buckets are lazily removed; zero disables eviction            | Rate Limiting    |
| Health Status     | The operational state reported by health endpoints: `up` or `down`                            | Health           |
| Ready Probe       | A function that returns true when the service is ready to accept traffic                      | Health           |

---

## Commands

Actions the library performs.

| Term                                | Definition                                                                                             | Context          |
| ----------------------------------- | ------------------------------------------------------------------------------------------------------ | ---------------- |
| `ClientIP(r)`                       | Extract the client IP from a request using header precedence: X-Forwarded-For → X-Real-IP → RemoteAddr | Client IP        |
| `ClientIPMiddleware(next)`          | Create middleware that injects the client IP into the request context                                  | Client IP        |
| `ClientIPFromContext(ctx)`          | Retrieve the stored client IP from a context                                                           | Client IP        |
| `WithClientIP(ctx, ip)`             | Store a client IP string in a context                                                                  | Client IP        |
| `CORS(cfg)`                         | Create middleware that sets CORS response headers and handles preflight requests                       | CORS             |
| `DefaultCORSConfig()`               | Return a permissive CORS config suitable for local development (allows all origins)                    | CORS             |
| `NewResponseRecorder(w)`            | Create a ResponseRecorder wrapping the given ResponseWriter, defaulting to unwritten state             | Response Capture |
| `Chain(handler, mw...)`             | Compose multiple middleware around a handler; first middleware in list is outermost                    | Response Capture |
| `HeaderSnapshot(rec)`               | Return an isolated copy of the response headers from a ResponseRecorder                                | Response Capture |
| `SecurityHeaders(cfg)`              | Create middleware that sets security response headers (nosniff, frame-options, etc.)                   | Security Headers |
| `DefaultSecurityHeadersConfig()`    | Return a SecurityHeadersConfig with production defaults                                                | Security Headers |
| `RequestID(cfg)`                    | Create middleware that propagates or generates a request ID                                            | Request ID       |
| `DefaultRequestIDConfig()`          | Return a RequestIDConfig that reads/generates X-Request-ID                                             | Request ID       |
| `RequestIDFromContext(ctx)`         | Retrieve the stored request ID from a context                                                          | Request ID       |
| `Recovery(logger)`                  | Create middleware that catches panics, logs the stack trace, and returns 500                           | Recovery         |
| `Timeout(duration)`                 | Create middleware that sets a deadline on the request context                                          | Timeout          |
| `Logging(logger)`                   | Create middleware that logs each request with method, path, status, duration, and client IP            | Logging          |
| `Compression(cfg)`                  | Create middleware that compresses responses based on Accept-Encoding negotiation                       | Compression      |
| `DefaultCompressionConfig()`        | Return a CompressionConfig with sensible defaults (default level, 512-byte minimum)                    | Compression      |
| `ETag(cfg)`                         | Create middleware that generates ETags and handles If-None-Match conditional requests                  | ETag             |
| `DefaultETagConfig()`               | Return an ETagConfig with strong ETags and 1MB max buffer                                              | ETag             |
| `HealthHandler()`                   | Return a handler that responds with `{"status":"up"}`                                                  | Health           |
| `LiveHandler()`                     | Alias for `HealthHandler()` for Kubernetes liveness probes                                             | Health           |
| `ReadyHandler()`                    | Return a handler for Kubernetes readiness probes (always up by default)                                | Health           |
| `ReadyHandlerWithProbe(ready)`      | Return a handler that calls `ready()` and responds 200 up or 503 down                                  | Health           |
| `RegisterHealth(mux)`               | Register `/health`, `/health/live`, `/health/ready` on a ServeMux                                      | Health           |
| `NewTokenBucketLimiter(rate,burst)` | Create an in-memory token bucket rate limiter (returns error if rate/burst <= 0)                       | Rate Limiting    |
| `RateLimit(cfg)`                    | Create middleware that enforces rate limiting using the configured limiter                             | Rate Limiting    |
| `DefaultRateLimitConfig()`          | Return a RateLimitConfig with 429 status and RemoteAddr key func                                       | Rate Limiting    |
| `Metrics(cfg)`                      | Create middleware that records request metrics via a pluggable recorder                                | Metrics          |
| `MaxBodySize(limit)`                | Create middleware that rejects request bodies exceeding the limit                                      | Body Size Limit  |
| `NewServer(cfg)`                    | Create an HTTP server with configurable timeouts and graceful shutdown                                 | Server Lifecycle |
| `NewMiddlewareStack()`              | Create a named middleware stack with duplicate prevention and ordering validation                      | Middleware Stack |
| `RegisterErrorClassifications()`    | Register stdlib HTTP error sentinels and message templates with go-error-family                        | Error Protocol   |
| `Validate()`                        | Check a config for invalid values at startup; all config types implement this                          | Universal        |
| `ParseUintQuery(r, key)`            | Parse a uint value from a query parameter; returns 0 on missing, empty, or invalid values              | Query Parameters |

---

## Events

State transitions within the library.

| Term                  | Definition                                                                        | Context          |
| --------------------- | --------------------------------------------------------------------------------- | ---------------- |
| Header Written        | `WriteHeader` called on ResponseRecorder; status is now captured and immutable    | Response Capture |
| Body Written          | `Write` called on ResponseRecorder; implicitly sets status 200 if not yet written | Response Capture |
| Preflight Handled     | CORS middleware intercepts an OPTIONS request and returns 204 No Content          | CORS             |
| Request Passed        | CORS middleware delegates to the next handler (non-OPTIONS or passthrough mode)   | CORS             |
| Request ID Generated  | RequestID middleware generates a new random ID because no header was present      | Request ID       |
| Request ID Propagated | RequestID middleware forwards an existing ID from the request header              | Request ID       |
| Panic Recovered       | Recovery middleware catches a panic, logs it, and writes 500                      | Recovery         |
| Deadline Exceeded     | Timeout middleware's context deadline is reached; handler should stop work        | Timeout          |
| Request Logged        | Logging middleware records the request method, path, status, and duration         | Logging          |
| Security Headers Set  | SecurityHeaders middleware writes security headers before delegating              | Security Headers |
| Compression Applied   | Compression middleware encodes the response body using the negotiated encoding    | Compression      |
| Compression Skipped   | Compression middleware passes through (no gzip accept, below min size, non-2xx)   | Compression      |
| ETag Computed         | ETag middleware generates an ETag value from the response body                    | ETag             |
| Not Modified Returned | ETag middleware returns 304 because If-None-Match matched the computed ETag       | ETag             |
| ETag Skipped          | ETag middleware passes through (non-GET/HEAD, non-2xx, body too large)            | ETag             |
| Rate Limited          | RateLimit middleware rejects a request because the token bucket was empty         | Rate Limiting    |
| Bucket Evicted        | An idle token bucket is removed during lazy sweep (EvictionTTL > 0)               | Rate Limiting    |
| Health Checked        | Health endpoint responds with current status                                      | Health           |
| Readiness Failed      | ReadyHandlerWithProbe calls ready() and it returns false; responds 503            | Health           |
| Metrics Recorded      | Metrics middleware records request duration, status, and method                   | Metrics          |
| Body Rejected         | MaxBodySize middleware rejects a request body exceeding the configured limit      | Body Size Limit  |
| Server Starting       | Server begins listening on the configured address                                 | Server Lifecycle |
| Server Shutting Down  | Server enters graceful shutdown, draining in-flight requests                      | Server Lifecycle |

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

- If the `ForwardHeader` is present on the request, its value is used as the request ID
- If absent, `GenerateID` is called to produce a new ID
- The ID is stored in the request context and set as a response header named `HeaderName`
- `Validate()` rejects nil `GenerateID`, empty `HeaderName`, and empty `ForwardHeader`

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

### ETag Rules

- Only applies to `GET` and `HEAD` requests
- Only generates ETags for cacheable (2xx) responses
- Reads `If-None-Match` header for conditional request evaluation
- Returns `304 Not Modified` if the computed ETag matches
- Uses FNV-64a for hash computation (64-bit, collision-resistant) with zero-allocation hex encoding; `HashFunc` field allows customization
- Buffers response body up to `MaxBufferSize` (default: 1MB); larger responses skip ETag
- `Validate()` rejects non-positive `MaxBufferSize`
- Strong ETags by default; set `Weak: true` for `W/"..."` prefix

### Middleware Chaining

- `Chain` applies middleware in **reverse order** so the first middleware in the variadic list is the outermost handler
- Execution order: first middleware → ... → last middleware → final handler
- When using Compression and ETag together: ETag must be **inner** (closer to handler) so it sees uncompressed bytes

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

- `TokenBucketLimiter` creates per-key token buckets; tokens refill at the configured rate up to burst capacity
- `NewTokenBucketLimiter` rejects rate <= 0 or burst <= 0
- `EvictionTTL` (zero by default) controls lazy eviction of idle buckets — non-zero enables sweeping
- Each `Allow(key)` consumes one token; returns false when the bucket is empty
- Custom `RateLimiter` implementations can replace the in-memory limiter (e.g., Redis-backed)

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
- `ETagConfig.Validate()` — rejects non-positive MaxBufferSize
- `SecurityHeadersConfig.Validate()` — all fields optional, currently always valid

### Error Classification

| Error Code                   | Family         | Retryable | When                                         |
| ---------------------------- | -------------- | --------- | -------------------------------------------- |
| `http.write_failed`          | Transient      | Yes       | Underlying ResponseWriter.Write fails        |
| `http.hijack_unsupported`    | Infrastructure | No        | Underlying writer doesn't implement Hijacker |
| `http.hijack_failed`         | Transient      | Yes       | Underlying Hijack call fails                 |
| `http.compress_write_failed` | Transient      | Yes       | Compression writer Write fails               |
| `http.etag_write_failed`     | Transient      | Yes       | ETag writer Write fails                      |

All classified errors implement `Coded`, `Classified`, `Contextual`, and `Retryable` from `go-error-family`.

---

## Conventions

Patterns consumers and contributors should follow.

| Convention             | Description                                                                                         |
| ---------------------- | --------------------------------------------------------------------------------------------------- |
| Middleware signature   | Always `func(http.Handler) http.Handler` — the Go standard library convention                       |
| Middleware type alias  | `type Middleware func(http.Handler) http.Handler` in `recorder.go`                                  |
| Classified errors      | Errors from ResponseRecorder use `go-error-family` for behavioral classification                    |
| Config validation      | All config types implement `Validate() error` for startup checks                                    |
| `httputil` import name | Consumers import as `httputil`; no aliases needed                                                   |
| Allowed dependencies   | `go-error-family` and `golang.org/x/time` are the only external dependencies (enforced by depguard) |

---

> **How to use this file:**
>
> - Keep terms concise — one clear sentence per definition
> - Update when new domain concepts emerge
> - Use these terms consistently in code, docs, and conversations
> - When in doubt about a word's meaning, check here first
