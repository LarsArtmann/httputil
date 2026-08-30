# Redis-Backed Rate Limiter

httputil's `RateLimiter` interface is designed for distributed backends. The built-in `TokenBucketLimiter` is in-memory; for multi-instance deployments, implement the interface with Redis.

> **Deprecated API notice (2026-08-30):** `RateLimit()` / `RateLimiter` / `RateLimitConfig` are deprecated and will be removed at v1.0 (see [migrating-to-keyed-rate-limiter.md](../migrating-to-keyed-rate-limiter.md)). This integration pattern applies to the deprecated interface until then. The successor [`KeyedRateLimiter`](https://pkg.go.dev/github.com/larsartmann/httputil#KeyedRateLimiterMiddleware) intentionally does not expose a pluggable limiter backend yet — if you need distributed rate limiting today, stay on the deprecated API or front `KeyedRateLimiterMiddleware` with a proxy-level limiter. The interface below is the last supported backend hook.

## Interface

```go
type RateLimiter interface {
    Allow(key string) bool
}
```

The interface is intentionally minimal: one method, one boolean. This makes it trivial to wrap any backend.

## Redis Implementation

Using a sliding-window counter with Lua for atomicity:

```go
// This example references github.com/redis/go-redis/v9, which is NOT a
// dependency of httputil — add it to your go.mod to compile.
package main

import (
    "context"
    "net/http"
    "time"

    "github.com/larsartmann/httputil"
    "github.com/redis/go-redis/v9"
)

type RedisRateLimiter struct {
    client *redis.Client
    rate   int           // requests per window
    window time.Duration // window size (e.g., 1s for rate-per-second)
}

func NewRedisRateLimiter(client *redis.Client, rate int, window time.Duration) *RedisRateLimiter {
    return &RedisRateLimiter{
        client: client,
        rate:   rate,
        window: window,
    }
}

// Lua script: atomically increment and check the counter for the window.
// Returns 1 if allowed, 0 if rate-limited.
const luaScript = `
local key = KEYS[1]
local limit = tonumber(ARGV[1])
local window = tonumber(ARGV[2])
local current = redis.call("INCR", key)
if current == 1 then
    redis.call("EXPIRE", key, window)
end
if current <= limit then
    return 1
else
    return 0
end
`

func (r *RedisRateLimiter) Allow(key string) bool {
    ctx := context.Background()
    redisKey := "ratelimit:" + key

    val, err := r.client.Eval(ctx, luaScript, []string{redisKey}, r.rate, int(r.window.Seconds())).Int()
    if err != nil {
        // Fail open: allow the request if Redis is unavailable.
        // Adjust to fail closed based on your security requirements.
        return true
    }

    return val == 1
}

// Usage:
func main() {
    client := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
    limiter := NewRedisRateLimiter(client, 100, time.Second)

    cfg := httputil.DefaultRateLimitConfig()
    cfg.Limiter = limiter

    mux := http.NewServeMux()
    handler := httputil.RateLimit(cfg)(mux)
    _ = handler
}
```

## Key Extraction

The default key function uses `r.RemoteAddr`. For accurate rate limiting behind a reverse proxy, use `httputil.ClientIP(r)`:

```go
cfg := httputil.DefaultRateLimitConfig()
cfg.Limiter = limiter
cfg.KeyFunc = func(r *http.Request) string {
    return httputil.ClientIP(r)
}
```

## Tradeoffs

| Aspect              | TokenBucketLimiter (in-memory) | RedisRateLimiter                |
| ------------------- | ------------------------------ | ------------------------------- |
| Scope               | Single process                 | All instances sharing Redis     |
| Status              | Deprecated (removal at v1.0)   | Built on the deprecated API     |
| Latency             | ~100ns per call                | ~0.5-1ms per call (network)     |
| Consistency         | Per-instance                   | Global                          |
| Eviction            | Built-in via `EvictionTTL`     | Redis TTL on keys               |
| Failure mode        | N/A (always available)         | Configurable (fail open/closed) |
| External dependency | None                           | Redis                           |

## Fail-Open vs Fail-Closed

When Redis is unavailable, you must decide:

- **Fail open** (allow): maximizes availability, risks over-serving during outages.
- **Fail closed** (deny): maximizes protection, risks blocking legitimate traffic during outages.

For most applications, fail-open is the pragmatic choice — the rate limiter is a protection layer, not a security boundary.
