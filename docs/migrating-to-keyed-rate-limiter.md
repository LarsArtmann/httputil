# Migrating from TokenBucketLimiter to KeyedRateLimiter

The legacy `TokenBucketLimiter` / `RateLimit()` / `RateLimitConfig` API is
deprecated as of v0.8.0 and will be removed in a future release. Migrate to
`KeyedRateLimiter` / `KeyedRateLimiterMiddleware` / `KeyedRateLimiterConfig`,
which adds O(log n) min-heap eviction, a MaxKeys cap, Retry-After headers,
and a monitoring API.

## Symbol Mapping

| Deprecated                    | Replacement                       |
| ----------------------------- | --------------------------------- |
| `RateLimitConfig`             | `KeyedRateLimiterConfig`          |
| `DefaultRateLimitConfig()`    | `DefaultKeyedRateLimiterConfig()` |
| `RateLimit(cfg)`              | `KeyedRateLimiterMiddleware(cfg)` |
| `RateLimiter` interface       | `KeyExtractor` function type      |
| `TokenBucketLimiter`          | `KeyedRateLimiter`                |
| `NewTokenBucketLimiter(r, b)` | `NewKeyedRateLimiter(cfg)`        |

## Before (deprecated)

```go
limiter, err := httputil.NewTokenBucketLimiter(10, 20) // 10 rps, burst 20
if err != nil {
    log.Fatal(err)
}

cfg := httputil.RateLimitConfig{
    Limiter:  limiter,
    KeyFunc:  func(r *http.Request) string { return r.RemoteAddr },
}
handler := httputil.RateLimit(cfg)(mux)
```

## After (current)

```go
cfg := httputil.KeyedRateLimiterConfig{
    Limit:        10,
    Window:       time.Second,   // 10 requests per second
    Burst:        20,
    KeyExtractor: httputil.KeyExtractorFromRemoteAddr(),
    TTL:          10 * time.Minute,
}
handler := httputil.KeyedRateLimiterMiddleware(cfg)(mux)
```

Or use defaults (100 req/min per client IP):

```go
handler := httputil.KeyedRateLimiterMiddleware(
    httputil.DefaultKeyedRateLimiterConfig(),
)(mux)
```

## Behavioral Differences

| Concern            | TokenBucketLimiter                | KeyedRateLimiter                          |
| ------------------ | --------------------------------- | ----------------------------------------- |
| Rate units         | Tokens per second (`float64`)     | Limit per Window (`uint` / `Duration`)    |
| Eviction           | Linear scan (`O(n)`) when enabled | Min-heap (`O(log n)`)                     |
| MaxKeys cap        | Not available                     | `MaxKeys` caps tracked keys               |
| Retry-After header | Not sent                          | Sent on 429 responses                     |
| Monitoring         | No API                            | `ActiveKeys()`, `Check(r)`                |
| Callbacks          | Not available                     | `OnAllowed`, `OnRejected`                 |
| Custom rejection   | `OnDenied http.HandlerFunc`       | `RejectionHandler func(w, r, retryAfter)` |

## Monitoring

`KeyedRateLimiter` exposes active key count for dashboards and autoscaling:

```go
rl := httputil.NewKeyedRateLimiter(cfg)
handler := rl.Middleware()(mux)

// Elsewhere:
slog.Info("active rate-limit keys", "count", rl.ActiveKeys())
```

## Custom Key Extractors

The `KeyExtractor` function type replaces the `KeyFunc` field and the
`RateLimiter` interface. Common extractors are built in:

```go
// By RemoteAddr (IP:port).
KeyExtractor: httputil.KeyExtractorFromRemoteAddr()

// By ClientIP (respects X-Forwarded-For / X-Real-IP behind a proxy).
KeyExtractor: httputil.KeyExtractorFromClientIP()

// Custom: by user ID from context.
KeyExtractor: func(r *http.Request) string {
    return userIDFromContext(r.Context())
}
```

Return `""` from a `KeyExtractor` to exempt a request from rate limiting.
