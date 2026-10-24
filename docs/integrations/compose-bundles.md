# Composing Middleware Bundles

httputil gives you three composition primitives, and they are designed to be used together:

| Primitive         | Signature                       | What it is for                                                                |
| ----------------- | ------------------------------- | ----------------------------------------------------------------------------- |
| `Chain(h, mw…)`   | handler + middlewares → handler | Apply a middleware list to one handler (first = outermost)                    |
| `Compose(mw…)`    | middlewares → middleware        | Bundle a middleware list into a single reusable `Middleware`                  |
| `MiddlewareStack` | named, validated collection     | Collect middleware with duplicate prevention and optional ordering validation |

All middleware is `func(http.Handler) http.Handler`, so anything you write composes with anything the library provides.

## The bundle pattern: build once, apply to many

Define your security baseline once as a package-level value, then apply it wherever you need it:

```go
package middleware

import (
	"log/slog"

	"github.com/larsartmann/httputil"
)

// SecureStack is the security baseline for every public endpoint in the
// service. First entry = outermost, matching Chain's ordering contract.
var SecureStack = httputil.Compose(
	httputil.Recovery(slog.Default()), // recovers panics from everything below (logger is required)
	httputil.SecurityHeaders(httputil.DefaultSecurityHeadersConfig()),
	httputil.Compression(httputil.DefaultCompressionConfig()),
	httputil.RequestID(httputil.DefaultRequestIDConfig()),
)

// Applies identically to many handlers:
var healthHandler = SecureStack(httputil.HealthHandler())
var metricsHandler = SecureStack(metricsMux)
```

Because `Compose` returns an ordinary `Middleware`, a bundle is just another middleware: it nests inside chains, other bundles, and stacks without special wiring.

```go
// A bundle used inside a larger chain:
handler := httputil.Chain(mux,
	httputil.Recovery(slog.Default()), // outermost: catch panics from everything below
	SecureStack,                       // the reusable bundle
	authMiddleware,                    // your own middleware composes freely
)
```

Composing an empty list yields the identity middleware, which lets optional bundles default to a no-op cleanly:

```go
var extraStack httputil.Middleware // nil-safe: Compose() with no args is identity
```

## Nesting MiddlewareStack as a bundle

`MiddlewareStack` collects middleware under names, prevents duplicate registration, and (opt-in via `Validate()`) enforces ordering rules such as Recovery-must-be-outermost. `MiddlewareStack.Middleware()` exposes the whole stack as a single nestable `Middleware`:

```go
stack := httputil.NewMiddlewareStack()
_ = stack.Add(httputil.MiddlewareRecovery, httputil.Recovery(slog.Default()))
_ = stack.Add(httputil.MiddlewareRequestID, httputil.RequestID(httputil.DefaultRequestIDConfig()))
_ = stack.Add(httputil.MiddlewareCompression, httputil.Compression(httputil.DefaultCompressionConfig()))

if err := stack.Validate(); err != nil {
	// ordering rule violated (e.g. Recovery not outermost); handle at wiring time
	return err
}

// Use the stack directly:
handler := stack.Build(mux)

// …or nest it inside a bigger composition:
outer := httputil.Chain(mux, tenantMiddleware, stack.Middleware())
```

`stack.Middleware()` reads the current stack entries per application, so compositions built before later `Add` calls still pick them up, with the same first-is-outermost ordering as `Build`.

## A tracing-first bundle

Tracing is the one bundle member whose position is not negotiable: the span must cover the whole request, and Recovery must sit inside it. See [OpenTelemetry tracing](./otel.md) for the two lanes and the reasoning.

```go
// Lane 2 shape: you own the TracerProvider, otelhttp supplies the span.
bundle := httputil.Compose(
	otelhttp.NewMiddleware("orders-api"), // span outermost
	httputil.RecoveryWithConfig(httputil.RecoveryConfig{
		Logger:  logger,
		OnPanic: recordPanicOnSpan, // exception events, inside the span
	}),
	httputil.RequestID(httputil.DefaultRequestIDConfig()),
	httputil.CORS(httputil.DefaultCORSConfig()),
	httputil.SecurityHeaders(httputil.DefaultSecurityHeadersConfig()),
	httputil.Compression(httputil.DefaultCompressionConfig()),
)

handler := bundle(mux)
```

Inside a `MiddlewareStack`, add tracing first and let `Validate()` enforce the Recovery rule:

```go
stack := httputil.NewMiddlewareStack()
_ = stack.Add("Tracing", otelhttp.NewMiddleware("orders-api"))
_ = stack.Add(httputil.MiddlewareRecovery, httputil.Recovery(slog.Default()))
```

Two bundle-level rules:

- **Recovery goes inside the span, never outside.** Outer Recovery still produces a span, but an empty successful one: `otelhttp` ends the span in a `defer`, so a panic caught above it never sets the 500 status or records the exception.
- **`ServerTimingMiddleware` is innermost.** It measures handler time; keeping it inside the span makes Server-Timing and trace durations comparable.

## Ordering rules that always apply

- **First = outermost.** `Chain`, `Compose`, and `MiddlewareStack` all share this ordering contract.
- **Recovery outermost when present.** A panic in any middleware to its right is caught; a `Recovery` that is not outermost silently stops protecting everything to its left. The one exception is tracing, which belongs outside Recovery (see above).
- **Nonce inside SecurityHeaders.** The nonce-bearing CSP must overwrite any static CSP; see `Nonce`'s doc comment.
- **`MiddlewareFunc.Then(nil)` does not panic.** A nil handler wires a fallback that answers 500 with a diagnostic body, and a nil receiver with a non-nil handler applies as the identity, so a wiring mistake surfaces as a response rather than a crash. The library ships no `Must*` helpers for the same reason: construction errors are returned or logged, never fatal.

## See also

- [`compose.go`](../../compose.go) — `Compose` and `MiddlewareFunc.Then`
- [`stack.go`](../../stack.go) — `MiddlewareStack`, name constants, `Validate`
- [Using httputil with samber/do](./samber-do.md) — bundles in a DI composition root
