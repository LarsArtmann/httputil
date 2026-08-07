# v1.0 API Stability Definition

**Status:** Target for v1.0 release. Pre-1.0, all APIs are subject to change per SemVer 0.x convention.

This document enumerates every exported symbol and classifies its stability commitment at v1.0.

## Stability Tiers

| Tier     | Commitment                                                                         |
| -------- | ---------------------------------------------------------------------------------- |
| Frozen   | Signature locked. Breaking changes require a major version bump (v2.0).            |
| Additive | Existing fields/methods locked. New fields/methods may be added in minor versions. |
| Evolving | May change before v1.0. Stabilizes at v1.0 unless noted.                           |

## Core Types

### Middleware Type

| Symbol       | Tier   | Notes                                              |
| ------------ | ------ | -------------------------------------------------- |
| `Middleware` | Frozen | `func(http.Handler) http.Handler` — the core alias |

### Config Types (all Additive — existing fields locked, new fields may be added)

| Type                     | Tier     | Notes                                                     |
| ------------------------ | -------- | --------------------------------------------------------- |
| `CORSConfig`             | Additive | `DenyUnmatched` default flipped in v0.7.0; frozen at v1.0 |
| `CSRFConfig`             | Additive | New in v0.8.0                                             |
| `CompressionConfig`      | Additive |                                                           |
| `DecompressionConfig`    | Additive |                                                           |
| `KeyedRateLimiterConfig` | Additive | New in v0.8.0                                             |
| `MetricsConfig`          | Additive |                                                           |
| `RateLimitConfig`        | Additive | Deprecated v0.8.0; removal targeted for v1.0              |
| `RequestIDConfig`        | Additive | Fields renamed in v0.7.0; frozen at v1.0                  |
| `SecurityHeadersConfig`  | Additive |                                                           |
| `ServerConfig`           | Additive |                                                           |
| `ResponseRecorder`       | Additive | Existing methods frozen; new methods may be added         |

### Default Config Constructors (all Frozen at v1.0)

Each returns a config with sensible defaults. Frozen at v1.0.

| Constructor                     | Config                         |
| ------------------------------- | ------------------------------ |
| `DefaultCORSConfig`             | `CORSConfig`                   |
| `DefaultCompressionConfig`      | `CompressionConfig`            |
| `DefaultDecompressionConfig`    | `DecompressionConfig`          |
| `DefaultKeyedRateLimiterConfig` | `KeyedRateLimiterConfig`       |
| `DefaultMetricsConfig`          | `MetricsConfig`                |
| `DefaultRateLimitConfig`        | `RateLimitConfig` (deprecated) |
| `DefaultRequestIDConfig`        | `RequestIDConfig`              |
| `DefaultSecurityHeadersConfig`  | `SecurityHeadersConfig`        |
| `DefaultServerConfig`           | `ServerConfig`                 |

### Middleware Factory Functions (all Frozen at v1.0)

| Function                       | Signature                                       |
| ------------------------------ | ----------------------------------------------- |
| `CORS`                         | `func(CORSConfig) Middleware`                   |
| `CSRFMiddleware`               | `func(CSRFConfig) Middleware`                   |
| `CSRFResponseHeaderMiddleware` | `func(http.Handler) http.Handler`               |
| `Compression`                  | `func(CompressionConfig) Middleware`            |
| `Decompression`                | `func(DecompressionConfig) Middleware`          |
| `KeyedRateLimiterMiddleware`   | `func(KeyedRateLimiterConfig) Middleware`       |
| `Logging`                      | `func(*slog.Logger) Middleware`                 |
| `MaxBodySize`                  | `func(int64) Middleware`                        |
| `Metrics`                      | `func(MetricsConfig) Middleware`                |
| `RateLimit`                    | `func(RateLimitConfig) Middleware` (deprecated) |
| `Recovery`                     | `func(*slog.Logger) Middleware`                 |
| `RequestID`                    | `func(RequestIDConfig) Middleware`              |
| `SecurityHeaders`              | `func(SecurityHeadersConfig) Middleware`        |
| `ServerTimingMiddleware`       | `func() Middleware`                             |
| `ServerTimingMiddlewareWhen`   | `func(func(*http.Request) bool) Middleware`     |
| `Timeout`                      | `func(time.Duration) Middleware`                |
| `ClientIPMiddleware`           | `func(http.Handler) http.Handler`               |

### Server Lifecycle (Frozen at v1.0)

| Symbol            | Tier   | Notes                                               |
| ----------------- | ------ | --------------------------------------------------- |
| `Server`          | Frozen | Struct; constructor + methods locked                |
| `NewServer`       | Frozen | `func(ServerConfig, http.Handler) (*Server, error)` |
| `Server.Start`    | Frozen | `func() <-chan error`                               |
| `Server.Shutdown` | Frozen | `func(context.Context) error`                       |
| `Server.Addr`     | Frozen | `func() string`                                     |

### Client IP (Frozen at v1.0)

| Symbol                | Tier   |
| --------------------- | ------ |
| `ClientIP`            | Frozen |
| `ClientIPFromContext` | Frozen |
| `WithClientIP`        | Frozen |

### Request ID (Frozen at v1.0)

| Symbol                   | Tier   |
| ------------------------ | ------ |
| `RequestID`              | Frozen |
| `DefaultRequestIDConfig` | Frozen |
| `RequestIDFromContext`   | Frozen |

### Health Checks (Frozen at v1.0)

| Symbol                  | Tier   | Notes    |
| ----------------------- | ------ | -------- |
| `HealthStatus`          | Frozen | Type     |
| `HealthStatusUp`        | Frozen | Constant |
| `HealthStatusDown`      | Frozen | Constant |
| `HealthResponse`        | Frozen | Struct   |
| `HealthHandler`         | Frozen |          |
| `LiveHandler`           | Frozen |          |
| `ReadyHandler`          | Frozen |          |
| `ReadyHandlerWithProbe` | Frozen |          |
| `RegisterHealth`        | Frozen |          |

### Response Recording (Frozen at v1.0)

| Symbol                | Tier     |
| --------------------- | -------- |
| `NewResponseRecorder` | Frozen   |
| `Chain`               | Frozen   |
| `ResponseRecorder`    | Additive |

### Capabilities (Frozen at v1.0)

| Symbol               | Tier   |
| -------------------- | ------ |
| `DetectCapabilities` | Frozen |
| `Capabilities`       | Frozen |

### Rate Limiting (Frozen at v1.0)

| Symbol                          | Tier     | Notes                                                                   |
| ------------------------------- | -------- | ----------------------------------------------------------------------- |
| `RateLimiter`                   | Frozen   | Interface (deprecated; removal targeted for v1.0)                       |
| `RateLimitConfig`               | Additive | Deprecated v0.8.0                                                       |
| `TokenBucketLimiter`            | Additive | Deprecated v0.8.0; removal targeted for v1.0                            |
| `NewTokenBucketLimiter`         | Frozen   | Deprecated v0.8.0                                                       |
| `KeyExtractor`                  | Frozen   | Function type                                                           |
| `KeyExtractorFromRemoteAddr`    | Frozen   |                                                                         |
| `KeyExtractorFromClientIP`      | Frozen   |                                                                         |
| `KeyedRateLimiterConfig`        | Additive | New in v0.8.0                                                           |
| `KeyedRateLimiter`              | Additive | New in v0.8.0; `ActiveKeys`/`Check`/`Middleware` methods frozen at v1.0 |
| `NewKeyedRateLimiter`           | Frozen   | New in v0.8.0                                                           |
| `KeyedRateLimiterMiddleware`    | Frozen   | New in v0.8.0                                                           |
| `DefaultKeyedRateLimiterConfig` | Frozen   | New in v0.8.0                                                           |

### CSRF Protection (Frozen at v1.0)

| Symbol                         | Tier     | Notes          |
| ------------------------------ | -------- | -------------- |
| `CSRFConfig`                   | Additive | New in v0.8.0  |
| `CSRFMiddleware`               | Frozen   | New in v0.8.0  |
| `CSRFResponseHeaderMiddleware` | Frozen   | New in v0.8.0  |
| `ForbiddenHandler`             | Frozen   | New in v0.8.0  |
| `ValidateCSRF`                 | Frozen   | New in v0.8.0  |
| `TranslateCSRFHeaders`         | Frozen   | New in v0.8.0  |
| `SetPlaintextHTTPOrigin`       | Frozen   | New in v0.8.0  |
| `WithCSRFToken`                | Frozen   | New in v0.8.0  |
| `CSRFTokenFromContext`         | Frozen   | New in v0.8.0  |
| `CSRFTokenFromRequest`         | Frozen   | New in v0.8.0  |
| `CSRFTokenHXHeaders`           | Frozen   | New in v0.8.0  |
| `CSRFTokenHTMLMeta`            | Frozen   | New in v0.8.0  |
| `CSRFTokenFormField`           | Frozen   | New in v0.8.0  |
| `InvalidateCSRFCookie`         | Frozen   | New in v0.8.0  |
| `CSRFTestToken`                | Frozen   | New in v0.8.0  |
| `ErrorHandler`                 | Frozen   | Type alias     |
| `ErrCSRFInvalid`               | Frozen   | Sentinel error |
| `ErrCSRFConfig`                | Frozen   | Sentinel error |

### Server-Timing (Frozen at v1.0)

| Symbol                       | Tier     | Notes           |
| ---------------------------- | -------- | --------------- |
| `ServerTiming`               | Additive | New in v0.8.0   |
| `NewServerTiming`            | Frozen   | New in v0.8.0   |
| `ServerTimingMiddleware`     | Frozen   | New in v0.8.0   |
| `ServerTimingMiddlewareWhen` | Frozen   | New in v0.8.0   |
| `WrapServerTiming`           | Frozen   | New in v0.8.0   |
| `WithServerTiming`           | Frozen   | New in v0.8.0   |
| `ServerTimingFromContext`    | Frozen   | New in v0.8.0   |
| `RecordServerTiming`         | Frozen   | New in v0.8.0   |
| `MeasureServerTiming`        | Frozen   | New in v0.8.0   |
| `HeaderServerTiming`         | Frozen   | String constant |

### Compression (Frozen at v1.0)

| Symbol                           | Tier   |
| -------------------------------- | ------ |
| `WriterFactory`                  | Frozen |
| `GzipWriterFactory`              | Frozen |
| `DeflateWriterFactory`           | Frozen |
| `DefaultWriterFactories`         | Frozen |
| `DefaultWriterFactoriesForLevel` | Frozen |
| `DefaultIncompressibleTypes`     | Frozen |

### Middleware Stack (Frozen at v1.0)

| Symbol                       | Tier     |
| ---------------------------- | -------- |
| `NewMiddlewareStack`         | Frozen   |
| `MiddlewareStack`            | Additive |
| `Middleware*` constants (11) | Frozen   | Name constants for ordering validation (`MiddlewareRecovery`, `MiddlewareLogging`, `MiddlewareRequestID`, `MiddlewareCORS`, `MiddlewareSecurityHeaders`, `MiddlewareCompression`, `MiddlewareTimeout`, `MiddlewareClientIP`, `MiddlewareCSRF`, `MiddlewareServerTiming`, `MiddlewareKeyedRateLimit`) |

### Query Parsing (Frozen at v1.0)

| Symbol           | Tier   |
| ---------------- | ------ |
| `ParseUintQuery` | Frozen |

### Error Classification (Frozen at v1.0)

| Symbol                         | Tier   | Notes           |
| ------------------------------ | ------ | --------------- |
| `ErrCodeWriteFailed`           | Frozen | String constant |
| `ErrCodeHijackUnsupported`     | Frozen |                 |
| `ErrCodeHijackFailed`          | Frozen |                 |
| `ErrCodeCompressWriteFailed`   | Frozen |                 |
| `ErrCSRFInvalid`               | Frozen | CSRF sentinel   |
| `ErrCSRFConfig`                | Frozen | CSRF sentinel   |
| `RegisterErrorClassifications` | Frozen |                 |

### Metrics (Frozen at v1.0)

| Symbol                 | Tier     |
| ---------------------- | -------- |
| `MetricsRecorder`      | Frozen   |
| `MetricsConfig`        | Additive |
| `Metrics`              | Frozen   |
| `DefaultMetricsConfig` | Frozen   |

## `httpspec` Subpackage

### Public API (Frozen at v1.0)

| Symbol                     | Tier     | Notes                             |
| -------------------------- | -------- | --------------------------------- |
| `Run`                      | Frozen   |                                   |
| `RunSerial`                | Frozen   |                                   |
| `Spec`                     | Additive | New fields may be added           |
| `Check`                    | Frozen   |                                   |
| `Result`                   | Additive |                                   |
| `Category`                 | Frozen   |                                   |
| `Option`                   | Frozen   |                                   |
| `WithIndexPath`            | Frozen   |                                   |
| `SkipSpec`                 | Frozen   |                                   |
| `WithExtraSpecs`           | Frozen   |                                   |
| `Pass`                     | Frozen   |                                   |
| `Fail`                     | Frozen   |                                   |
| `ExpectStatus`             | Frozen   |                                   |
| `ExpectNotStatus`          | Frozen   |                                   |
| `ExpectHeader`             | Frozen   |                                   |
| `ExpectHeaderAbsent`       | Frozen   |                                   |
| `ExpectBodyContains`       | Frozen   |                                   |
| `SpecName*` constants (18) | Frozen   | String values are part of the API |

### Standard Specs (Additive)

New standard specs may be added in minor versions. Existing specs will not be removed or have their assertions loosened.

## Versioning Policy

- **v1.0:** This document takes effect. All "Frozen" symbols are locked.
- **Post-1.0 minor versions:** May add new fields to "Additive" types, new exported functions, and new standard specs. No existing symbol signature changes.
- **Post-1.0 major versions (v2.0+):** Breaking changes permitted with migration guide.
