# OpenTelemetry Tracing with httputil

httputil ships **no OpenTelemetry dependency**. That is deliberate: a reusable middleware library must not force a tracing SDK, exporter, or sampling policy on its consumers. What httputil guarantees is that tracing composes cleanly when you add it — and this page is the canonical wiring guide.

## Two lanes, one owner each (ruling 2026-10-23)

| Lane           | Owner                                                                                                    | Use when                                                                                                          |
| -------------- | -------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------- |
| SDK / provider | [`github.com/larsartmann/go-appkit/otel`](https://github.com/larsartmann/go-appkit) (alias `appkitotel`) | You want OTLP export, sampling, dashboards, exception events, and an instrumented client transport out of the box |
| Bare provider  | `go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp`                                          | You already own the OpenTelemetry provider and only need the middleware bridge                                    |

Do not build a third lane. An `httputil/otel` sub-module is a recorded non-goal: it would split the SDK lane in two and add the telemetry dependency this library refuses to carry.

## Why composition works

- `Middleware` is a **type alias** (`type Middleware = func(http.Handler) http.Handler`), not a named type. Any stdlib middleware — including `otelhttp.NewMiddleware` — composes with `Chain`, `Compose`, and `MiddlewareStack` without adapters.
- Every request-forking middleware copies the matched route pattern back to the outermost request (`r.Pattern`) before calling `next`. `otelhttp` names spans from that pattern, so you get `GET /users/{id}` instead of `GET` even when RequestID, CSRF, Nonce, Timeout, or Server-Timing wraps the handler.
- Response-writer wrappers preserve `http.Flusher` and `http.Hijacker`, so streaming and WebSocket handlers keep working underneath the span.

## Lane 1 — go-appkit/otel

This is the fleet default. `Setup` installs the provider, `Middleware` produces the span, and `Recovery` adds `exception.*` span events that backends such as SigNoz index into a dedicated exceptions view.

```go
// This example references github.com/larsartmann/go-appkit/otel, which is NOT
// a dependency of httputil — add it to your go.mod to compile.
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	appkitotel "github.com/larsartmann/go-appkit/otel"
	"github.com/larsartmann/httputil"
)

func main() {
	logger := slog.Default()

	provider, err := appkitotel.Setup(
		appkitotel.WithService("orders-api", "1.0.0", os.Getenv("POD_NAME")),
		appkitotel.WithOTLP(), // endpoint from OTEL_EXPORTER_OTLP_ENDPOINT
	)
	if err != nil {
		logger.Error("otel setup", "error", err)
		os.Exit(1)
	}

	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_ = provider.Shutdown(shutdownCtx)
	}()

	handler := httputil.Compose(
		appkitotel.Middleware(),     // span outermost
		appkitotel.Recovery(logger), // exception events, inside the span
		httputil.RequestID(httputil.DefaultRequestIDConfig()),
		httputil.CORS(httputil.DefaultCORSConfig()),
	)(mux())

	_ = http.ListenAndServe(":8080", handler)
}

func mux() http.Handler {
	m := http.NewServeMux()
	m.HandleFunc("GET /users/{id}", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	return m
}
```

`Setup()` with no options and the standard `OTEL_*` environment variables is enough for a fully configured service. Explicit options win per signal, so a partial migration is safe.

## Lane 2 — bare otelhttp

If you already have a `TracerProvider`, the bridge is one line. `otelhttp` is a `func(http.Handler) http.Handler`, which is exactly what `httputil.Compose` accepts. (`httputil.Chain` takes the handler first — `Chain(mux, mw1, mw2)` — so it is the other valid spelling.)

```go
// This example references
// go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp, which is NOT
// a dependency of httputil — add it to your go.mod to compile.
package main

import (
	"net/http"

	"github.com/larsartmann/httputil"
	otelhttp "go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

func newHandler(mux *http.ServeMux) http.Handler {
	return httputil.Compose(
		otelhttp.NewMiddleware("orders-api",
			// Only for internet-facing endpoints.
			otelhttp.WithPublicEndpointFn(func(*http.Request) bool { return true }),
		),
		httputil.RequestID(httputil.DefaultRequestIDConfig()),
	)(mux)
}
```

## Middleware ordering

Order is declaration order: the first middleware is outermost.

| Position      | Middleware                                                                    | Why                                                                                                                          |
| ------------- | ----------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------- |
| 1 (outermost) | tracing (`appkitotel.Middleware` or `otelhttp.NewMiddleware`)                 | The span must cover the full request lifetime, including recovery and compression                                            |
| 2             | `Recovery` (or `appkitotel.Recovery`)                                         | Must run **inside** the span: it turns a panic into a 500, and the tracing middleware records the status on the span it owns |
| 3             | `RequestID`, `CORS`, `SecurityHeaders`, `CSRFMiddleware`, `Nonce`, `Language` | Ordinary request policy; each propagates `r.Pattern` outward                                                                 |
| 4             | `Compression` / `Decompression`                                               | Writer-interception belongs near the handler; the span still sees status and byte counts                                     |
| 5 (innermost) | `ServerTimingMiddleware`                                                      | Measures handler time only; keep it inside the span so Server-Timing and trace durations are comparable                      |

In a `MiddlewareStack`, add tracing first and call `Validate()` if you want the Recovery-ordering check enforced.

### Why Recovery goes inside the span

`otelhttp` ends the span in a `defer`, so a panic does not lose the span — but it also does not record an exception or set the span status. `Recovery` (httputil's or appkit's) must be inner to the tracing middleware for the 500 status to reach the span and for exception events to attach. The reverse order (Recovery outer) still produces a span, but it is an empty successful one.

## Exception events without an SDK dependency

If you use plain `otelhttp` and `httputil.Recovery`, bridge panics to your span with the nil-default hook:

```go
import (
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

mw := httputil.RecoveryWithConfig(httputil.RecoveryConfig{
	Logger: logger,
	OnPanic: func(r *http.Request, recovered any) {
		span := trace.SpanFromContext(r.Context())
		if err, ok := recovered.(error); ok {
			span.RecordError(err)
		} else {
			span.RecordError(fmt.Errorf("%v", recovered))
		}

		span.SetStatus(codes.Error, "panic")
	},
})
```

The hook runs after the panic is logged and **before** the 500 is written, so the span is still recording. A panicking hook is logged and swallowed; it cannot replace the recovery path. `Recovery(logger)` itself is unchanged and needs no hook.

## Correlation: trace ID and request ID

Tracing answers "which trace"; `RequestID` answers "which request" in logs. They are complementary, not duplicates:

- Read the trace ID from the span context: `trace.SpanContextFromContext(r.Context()).TraceID()`.
- Log it next to `httputil.RequestIDFromRequest(r)` so log lines are joinable to traces.
- For cross-service correlation, use the fleet baggage convention `correlation_id` (go-appkit/otel's `WithCorrelationID` / `CorrelationID`). Do not invent a parallel header or seed request IDs from `traceparent` without a ruling — the dual-header `X-Correlation-ID` question is tracked in [ROADMAP.md](../../ROADMAP.md).

## Sampling

httputil makes no sampling decision and never mutates trace headers. Sampling belongs to the provider, which means there is exactly one lane-specific answer:

- **Lane 1 (`go-appkit/otel`).** With no option, sampling is env-native: `OTEL_TRACES_SAMPLER` and `OTEL_TRACES_SAMPLER_ARG` drive the SDK (`parentbased_traceidratio` with a ratio is the usual production setting). Pass `WithSampler` only when the decision needs code:

```go
provider, err := appkitotel.Setup(
	appkitotel.WithService("orders-api", "1.0.0", os.Getenv("POD_NAME")),
	appkitotel.WithOTLP(),
	// ParentBased keeps distributed traces continuous: the inner sampler only
	// decides for requests that arrive without a sampled parent.
	appkitotel.WithSampler(sdktrace.ParentBased(sdktrace.TraceIDRatioBased(0.1))),
)
```

- **Lane 2 (bare `otelhttp`).** You already own the `TracerProvider`, so you already own the sampler. `otelhttp` contributes nothing here.

Two rules that matter more than the ratio:

1. **Wrap any ratio sampler in `ParentBased`.** A bare `TraceIDRatioBased` re-decides on every hop and silently breaks traces into unrelated fragments, which looks like missing spans rather than a sampling bug.
2. **Sampling is not a security control.** A caller can send `traceparent` claiming to be sampled; that is why public endpoints need `WithPublicEndpoint()` (lane 1) or `WithPublicEndpointFn` (lane 2) so the header becomes a link instead of a parent.

## Gotchas

- **Compression and tracing both wrap the writer.** Keep tracing outermost (see the ordering table). `Compression` short-circuits `identity` and passes the raw writer through, so the span is unaffected either way; the order only decides whether compression time is inside the span.
- **Streaming and hijacking.** httputil wrappers preserve `http.Flusher` and `http.Hijacker`. If you add your own writer wrapper between tracing and the handler, preserve them too or SSE/WebSocket handlers will fail.
- **Public endpoints.** For endpoints reachable from the internet, unverified `traceparent` headers should become links, not parents — otherwise a caller can force trace continuity and skew sampling. `appkitotel` exposes this as `WithPublicEndpoint()`; bare `otelhttp` as `WithPublicEndpointFn(func(*http.Request) bool)`. Use it for public APIs, never for internal ones.
- **Health endpoints.** `appkitotel.Middleware` filters `/health`, `/health/live`, and `/health/ready` by default and offers `WithFilteredPaths`. `otelhttp` does not filter anything; exclude probes yourself.
- **Span names need patterns.** `r.Pattern` is populated by Go 1.22+ `ServeMux` pattern routing. With a custom router that does not set `Pattern`, spans fall back to the HTTP method.
- **`Server.MaxHeaderValueCount`** is not reachable through httputil's `Server` wrapper; if you need it, wire `http.Server` directly.

## Verification

1. **Span names carry the route.** Send a request to a parametrized route and confirm the span name is `GET /users/{id}`, not `GET`. If it is `GET`, check that your router sets `r.Pattern` and that a middleware is not replacing the request without copying it.
2. **Parent/child continuity.** Send a request with a valid `traceparent` and confirm the server span shares its trace ID. With `WithPublicEndpoint`, confirm the header becomes a link instead.
3. **Exception events.** Panic in a handler and confirm the backend shows an exception event (type, message, stacktrace) and an error span status, with the client still receiving `500 Internal Server Error`.
4. **Contract checks.** Two complementary surfaces:
   - `httpspec.TracingSpecs()` adds handler-level specs for both W3C trace-context headers (a valid `traceparent` or `tracestate` never causes a 5xx; a malformed one behaves like an absent header) to a normal spec run.
   - `httpspec.RunTracing(t, middleware)` verifies the middleware layer directly: `traceparent`, `tracestate`, and `r.Pattern` reach the wrapped handler, no header is invented, method/path/body arrive unmodified, status/body pass through, and `Flusher`/`Pusher`/`Hijacker` survive the writer wrapper.

```go
httpspec.Run(t, handler, httpspec.WithExtraSpecs(httpspec.TracingSpecs()...))
httpspec.RunTracing(t, appkitotel.Middleware())
```

5. **The snippets in this page are compiled, not merely prose.** `testdata/otelinterop` is a separate module that builds both lanes and the `OnPanic` bridge, and runs `RunTracing` against the real `otelhttp` and `appkitotel` middlewares. The CI job `otel-interop` runs it; reproduce locally with `cd testdata/otelinterop && GOWORK=off go test -race ./...`. The module lives outside the root `go.mod` so httputil keeps its four-dependency policy.

## Related

- [Architecture reference — Observability lane](../architecture-reference.md#observability-otel-lane)
- [Prometheus metrics](prometheus-metrics.md) — the same plugin pattern for metrics
- [Composition bundles](compose-bundles.md) — reusable middleware stacks
