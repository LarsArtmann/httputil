# servertiming

[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/httputil/server_timing.svg)](https://pkg.go.dev/github.com/larsartmann/httputil/server_timing)
[![Go Version](https://img.shields.io/badge/Go-1.26+-00ADD8)](https://go.dev)

W3C [Server-Timing](https://w3c.github.io/server-timing/) header instrumentation for Go `net/http`. **Zero dependencies — stdlib only.**

Records named timing metrics per request and injects them as a standards-compliant `Server-Timing` response header at first byte. Designed to compose with any middleware stack or router.

## Install

```bash
go get github.com/larsartmann/httputil/server_timing
```

## Quick Start

```go
package main

import (
    "net/http"

    "github.com/larsartmann/httputil/server_timing"
)

func main() {
    mux := http.NewServeMux()
    mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        stop := servertiming.MeasureServerTiming(r.Context(), "db")
        // ... do DB work ...
        stop()
        w.WriteHeader(http.StatusOK)
    })

    // Wraps every response with a Server-Timing header.
    handler := servertiming.ServerTimingMiddleware()(mux)
    http.ListenAndServe(":8080", handler)
}
```

Resulting header (total is measured automatically as time-to-first-byte):

```
Server-Timing: total;desc="Total request";dur=12.3, db;dur=8.1
```

## Conditional (debug/admin only)

Server-Timing leaks internal performance details. Gate it behind a predicate and the disabled path has **zero overhead** — no writer wrapping, no collector in context, and nil-receiver methods make all handler calls natural no-ops:

```go
handler := servertiming.ServerTimingMiddlewareWhen(func(r *http.Request) bool {
    return r.URL.Query().Has("debug")
})(mux)
```

## API

| Symbol                                     | Purpose                                                                        |
| ------------------------------------------ | ------------------------------------------------------------------------------ |
| `ServerTimingMiddleware()`                 | Always-on middleware; injects collector via context.                           |
| `ServerTimingMiddlewareWhen(pred)`         | Conditional middleware; zero-overhead passthrough when `pred` is false.        |
| `WrapServerTiming(w, r)`                   | Manual wrapping (returns wrapped writer + request with collector).             |
| `ServerTimingFromContext(ctx)`             | Retrieve the collector (nil-safe — no-op when absent).                         |
| `MeasureServerTiming(ctx, name)`           | Context-aware deferred-stop timer.                                             |
| `RecordServerTiming(ctx, name, desc, dur)` | Context-aware one-shot record.                                                 |
| `ServerTiming`                             | Direct collector type (`NewServerTiming`, `Record`, `Measure`, `HeaderValue`). |

All `*ServerTiming` methods are **nil-safe** — a nil collector (no middleware active) makes every call a no-op, so handlers never need per-request nil checks.

## Features

- **CRLF-safe** — metric names sanitized to RFC 7230 tokens; descriptions escaped against header injection.
- **Concurrency-safe** — collector uses a mutex; safe for concurrent handler goroutines.
- **Preserves capabilities** — writer delegates `Flush`, `Hijack`, and `Push` so SSE, WebSocket, and HTTP/2 work transparently.
- **Fractional milliseconds** — sub-millisecond timings are preserved (`dur=0.5`), not truncated.
- **stdlib only** — no third-party dependencies in the module graph.

## Relationship to httputil

This is a sub-module of [`github.com/larsartmann/httputil`](https://github.com/larsartmann/httputil), a composable HTTP middleware toolkit. `servertiming` is self-contained and can be used independently — but if you want CORS, compression, rate limiting, CSRF, health checks, and more in one coherent stack, use the parent module.
