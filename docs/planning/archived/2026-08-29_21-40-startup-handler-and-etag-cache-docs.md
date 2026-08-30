# Design Note — StartupHandler (Readiness Warmup) and ETag Cache-Control Handoff

**Date:** 2026-08-29
**Status:** DESIGN — not scheduled for implementation; recorded so the idea survives and the shape is thought through.

## StartupHandler

### Problem

Kubernetes `readinessProbe` semantics (503 until ready) plus a "first request warms caches" pattern require handlers to coordinate startup state. Libraries typically want a tiny primitive, not a framework.

### Proposed shape

```go
// StartupHandler gates requests on a readiness signal. Before Ready, it
// answers 503 with Retry-After; after, it delegates for the lifetime of the
// process.
func StartupHandler(ready func()) http.HandlerFunc
```

- Constructor takes a `func()` "done" callback (or a `<-chan struct{}`); the zero-value handler reports not-ready.
- Wire into `RegisterHealth` as `/health/startup` (K8s 1.16+ startupProbe consumes plain 200/503).
- Composability rule: startup gating belongs OUTSIDE `Nonce`/`Compression` (nothing should be cached or instrumented as successful before readiness).

### Rejected

- A global singleton readiness registry (hidden global state; breaks the library's explicit-config ethos).
- Integrated cache-warming (out of scope for an HTTP utility library).

### Verdict

Viable post-v1.0 additive API. The `ReadyHandlerWithProbe(ready func() bool)` already covers the 80% case (external readiness signal); StartupHandler only earns its keep if startup-probe users ask for the auto-200-after-ready behavior in-library.

## ETag Cache-Control / Vary documentation handoff

The 07-45/22-43 reports flagged that ETag correctness depends on cache directives (`Cache-Control`, `Vary`) that the ETag middleware itself does not manage:

- **Ownership decision (2026-08-29):** conditional-request _headers_ (ETag / If-None-Match / 304 semantics) belong to go-etag; _cache policy_ (Cache-Control, Vary) belongs to the application or a `SecurityHeaders`-style header middleware. httputil will not add a Cache-Control type pre-v1.0.
- Handoff action recorded for go-etag: document in its README that a 304 response must preserve ETag and should be paired by the caller with appropriate Cache-Control/Vary; see the item list in T25 of the 2026-08-29 plan.
- In-repo guidance: the README ETag section should carry one sentence — "pair ETags with explicit Cache-Control; without Vary on negotiated representations, caches may serve mismatched bodies."
