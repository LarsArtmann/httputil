# server_timing

[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/httputil/server_timing.svg)](https://pkg.go.dev/github.com/larsartmann/httputil/server_timing)
[![Go Version](https://img.shields.io/badge/Go-1.27+-00ADD8)](https://go.dev)
[![License](https://img.shields.io/badge/license-MIT-blue)](LICENSE)

W3C [Server-Timing](https://w3c.github.io/server-timing/) header instrumentation for Go HTTP handlers. Record named timing metrics (database, cache, external calls, ...) from anywhere in the handler call chain and have them serialized into the `Server-Timing` response header — where browser DevTools, RUM agents, and load-testing tools can read them without any vendor-specific agent.

Zero dependencies. This module is stdlib-only, so you can add Server-Timing to a service without pulling in the full [`httputil`](https://github.com/larsartmann/httputil) dependency set.

## Install

```bash
go get github.com/larsartmann/httputil/server_timing
```

## Quick Start

```go
package main

import (
	"net/http"
	"time"

	"github.com/larsartmann/httputil/server_timing" // package servertiming
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		stop := servertiming.MeasureServerTiming(r.Context(), "work")
		time.Sleep(10 * time.Millisecond) // your actual work: db, cache, render, ...
		stop()

		w.WriteHeader(http.StatusOK)
	})

	http.ListenAndServe(":8080", servertiming.ServerTimingMiddleware()(mux))
}
```

Response header (also auto-records `total` = time to first byte):

```text
Server-Timing: total;desc="Total request";dur=12, work;dur=10
```

Open Chrome or Firefox DevTools → Network → Timing and the metrics appear per request.

## How It Works

1. `ServerTimingMiddleware()` wraps the `http.ResponseWriter` and stores a concurrency-safe `*ServerTiming` collector in the request context.
2. Any handler downstream records metrics via the context helpers — `MeasureServerTiming` (timed regions), `RecordServerTiming` (already-measured durations), or `ServerTimingFromContext` (direct collector access). All are nil-safe: without the middleware they are no-ops, so handlers need no per-request branching.
3. At the first `WriteHeader`/`Write`, the middleware finalizes the `total` metric (time to first byte) and sets the header — exactly once, before the response commits.

Manual control (frameworks that own the middleware chain): `WrapServerTiming(w, r)` wraps an individual writer/request pair.

## Features

- **Zero dependencies** — pure stdlib; embeddable where the root `httputil` module is too heavy.
- **Conditional instrumentation** — `ServerTimingMiddlewareWhen(pred)` enables collection only for matching requests (debug flag, admin role); non-matching requests pass through with no writer wrapping at all. Server-Timing can leak internal performance details — gate it in production.
- **Concurrency-safe collector** — record from goroutines your handler spawns; metrics serialize in completion order.
- **Injection-safe header values** — metric names are sanitized to RFC 7230 tokens, descriptions are quoted-string-escaped, CR/LF are neutralized (fuzz-tested).
- **Fractional milliseconds preserved** — `dur=0.5` is not rounded away.
- **Transparent capability delegation** — `Flush`, `Hijack`, `Push`, and `Unwrap` pass through the wrapper, so SSE, WebSockets, and HTTP/2 keep working; `http.ResponseController` sees through it.

## The One Gotcha: Measure Before You Write

The header is stamped when the response commits. A metric recorded after the first `Write`/`WriteHeader` misses the header — so in a regular handler, call `stop()` before writing the response, not via `defer`:

```go
stop := servertiming.MeasureServerTiming(r.Context(), "db")
rows, err := db.Query(r.Context())
stop() // BEFORE the write — a deferred stop() fires after the response commits
renderRows(w, rows, err)
```

The `defer` idiom is correct only for streaming responses (SSE), where the header is sent once at stream start and metrics are recorded during the stream.

## Composability

The `Middleware` type is the standard `func(http.Handler) http.Handler` shape — it composes with [`httputil.Chain`](https://pkg.go.dev/github.com/larsartmann/httputil#Chain) / `Compose` or any router's `Use` without adapters.

## License

[MIT](LICENSE)
