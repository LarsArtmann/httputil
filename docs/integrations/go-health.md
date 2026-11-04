# Using httputil with go-health

[**go-health**](https://github.com/larsartmann/go-health) is the fleet's Kubernetes health-probe SDK: liveness, readiness, and startup probes with critical/non-critical classification, background caching, and shutdown awareness. httputil ships a minimal `health.go` — always-up stubs plus `ReadyHandlerWithProbe(func() bool)` — as the zero-dependency floor. When a service has real dependencies, go-health is the upgrade; httputil remains the per-request plumbing around it.

## Why they fit

- **No overlap:** go-health owns probe semantics — roll-up classification (`pass`/`warn`/`fail`, check-level `off`), background caching, shutdown overlay, aggregate and federation. httputil owns per-request HTTP plumbing and server lifecycle. Neither duplicates the other.
- **Clean boundary:** every go-health handler is a plain `http.HandlerFunc`, so httputil's `Chain`/`Compose`/`MiddlewareStack` wrap it directly — the same seam as any other handler.
- **Neither imports the other, by policy:** httputil's `depguard` forbids third-party dependencies, and go-health keeps a three-dependency footprint. The integration is composition plus documentation, never a dependency edge.

## Vocabulary: `up`/`down` vs `pass`/`warn`/`fail`

httputil's stub payload speaks `up`/`down` (200/503). go-health roll-ups are `pass`/`warn`/`fail` with check-level `off`, mapped to 200/200/503 — a non-critical failure degrades the body to `warn` while the status code stays 200. Kubernetes probes and load balancers read only the status code, so the two vocabularies coexist on one mux without confusing them.

## Migrating from the floor to go-health

Before — the stub floor (`RegisterHealth` / `ReadyHandlerWithProbe`):

```go
httputil.RegisterHealth(mux) // /health, /health/live, /health/ready — always up
```

After — real probes behind the same routes:

```go
probe := health.NewChecks(map[string]health.CheckFunc{
	"db": checks.Database(db, 2*time.Second),
}, health.WithCriticalServices("db"))

mux.Handle("GET /health/live", probe.LivenessHandler())  // open, always 200
mux.Handle("GET /health/ready", probe.ReadinessHandler()) // 200 pass/warn, 503 fail
```

### The endpoint split

Wrap readiness (and startup) with httputil middleware when exposure demands it; leave liveness open and unwrapped — the kubelet cannot send credentials, and an auth or rate-limit rejection on liveness reads as a container failure (restart cascade):

```go
wrapped := httputil.Chain(
	probe.ReadinessHandler(),
	httputil.Recovery(slog.Default()),
	httputil.RequestID(httputil.DefaultRequestIDConfig()),
	httputil.Timeout(5*time.Second),
	httputil.KeyedRateLimiterMiddleware(httputil.DefaultKeyedRateLimiterConfig()),
)
```

The chain preserves go-health's code semantics exactly: only `fail` is 503. Composition order matters — middleware runs before go-health's method guard (`WithAllowedMethods`), so a non-allowed method gets the guard's `405` + `Allow` header after traversing the chain.

## Drain ordering

`probe.Shutdown()` turns readiness 503 immediately (with `shutting_down: true`), so the load balancer stops routing while `httputil.Server.Shutdown` drains in-flight requests. Flip the probe **before** the server drains — with [samber/do](./samber-do.md), `Shutdowner` services dispatch in reverse invocation order, so do not rely on `AsShutdowner` ordering for this; call `probe.Shutdown()` first, then hand over to `inj.ShutdownWithContext(ctx)` (which discovers `*httputil.Server` automatically).

## Server-Timing per-check durations

`server_timing` emits the W3C `Server-Timing` header; go-health's detailed checks report per-check durations (`duration_ns` on the wire). A ~10-line consumer-side bridge publishes each check as a Server-Timing metric — record **before** delegating, since the middleware commits the header at the response's first write. The proved version of the bridge lives in go-health's [`testdata/httputil-interop`](https://github.com/larsartmann/go-health/tree/master/testdata/httputil-interop) fixture and is re-verified on every go-health CI run (`nix run .#interop`).

## Where this is proved

This document describes the composition; the machine proof lives on the go-health side: the `httputil-interop` fixture module compiles the recipes (liveness through middleware, code semantics for all three roll-ups, method-guard ordering, drain flip, Server-Timing bridge) against the published httputil release. Full recipes with the boundary rationale: [docs/httputil-integration.md](https://github.com/larsartmann/go-health/blob/master/docs/httputil-integration.md) in the go-health repo.

## See also

- [go-health integration doc](https://github.com/larsartmann/go-health/blob/master/docs/httputil-integration.md) — the same recipes from the probe side
- [Using httputil with samber/do](./samber-do.md) — the composition root and drain story
- [Composing Middleware Bundles](./compose-bundles.md) — a reusable security baseline around probe handlers
- [Prometheus MetricsRecorder example](./prometheus-metrics.md) — go-health's evaluation hook pairs with it
