# httputil vs. huma — Comparison

_Date: 2026-07-05_
_Sources: [huma README](https://github.com/danielgtaylor/huma/), [huma.rocks](https://huma.rocks/), pkg.go.dev/huma/v2, httputil repo._

## TL;DR

They solve **different problems at different layers**, and are **complementary, not competing**. Huma is an _API contract framework_ (types → validation → OpenAPI). httputil is an _infrastructure middleware toolkit_ (compression, security, observability, resilience). The telling fact: **huma v2 deliberately deleted its own middleware package** (v1 shipped `Logger`, `Recovery`, `ContentEncoding`, `OpenTracing`) to become "bring your own middleware" — and httputil _is_ one of those.

## What each one is

|                   | **httputil**                                                           | **huma**                                                              |
| ----------------- | ---------------------------------------------------------------------- | --------------------------------------------------------------------- |
| Layer             | Infrastructure / plumbing                                              | API contract / productivity                                           |
| Shape             | Bag of `func(http.Handler) http.Handler` middleware + server lifecycle | Declarative framework: `huma.Get(api, path, handler)` over any router |
| Killer feature    | Compression + ETag + security + observability middleware               | OpenAPI 3.1 + JSON Schema generated from Go types                     |
| Dependency policy | 1 dep (same author)                                                    | Minimal (famously reduced over time)                                  |
| Router            | Agnostic (pure stdlib signature)                                       | Bring-your-own (chi, stdlib mux, …)                                   |
| Maturity          | v0.4.0, single author, personal                                        | v2, large community, sponsors, many companies                         |

## Feature matrix (accurate to huma v2)

| Concern                                              |        httputil         | huma v2                                                           |
| ---------------------------------------------------- | :---------------------: | ----------------------------------------------------------------- |
| Response compression (gzip/deflate, q-values, pools) |           ✅            | ❌ (was in v1, dropped)                                           |
| ETag _generation_ from body + 304                    |           ✅            | ❌ (only `conditional` _evaluation_ helpers; you supply the ETag) |
| CORS middleware                                      |           ✅            | ❌ bring-your-own                                                 |
| Panic recovery                                       |           ✅            | ❌ (was in v1, dropped)                                           |
| Structured request logging                           |           ✅            | ❌ bring-your-own                                                 |
| Request ID generation                                |           ✅            | ❌ bring-your-own                                                 |
| Security headers (nosniff, frame, referrer)          |           ✅            | ❌ bring-your-own                                                 |
| Rate limiting / metrics / body-size                  |           ✅            | ❌ bring-your-own (huma has per-op size limits only)              |
| Server lifecycle + graceful shutdown                 |           ✅            | ✅ (via `humacli`)                                                |
| Health endpoints                                     |           ✅            | ❌                                                                |
| Behavioral spec test suite (validate any handler)    |     ✅ (`httpspec`)     | ❌                                                                |
| Classified/retryable errors                          | ✅ (`go-error-family`)  | — (uses RFC 9457 instead)                                         |
| **Typed request/response structs**                   |           ❌            | ✅                                                                |
| **Automatic input validation**                       |           ❌            | ✅ (JSON Schema from tags)                                        |
| **OpenAPI 3.1 + docs UI generation**                 |           ❌            | ✅ (the headline feature)                                         |
| **Content negotiation (JSON/CBOR bodies)**           |           ❌            | ✅                                                                |
| RFC 9457 problem+json errors                         |           ❌            | ✅                                                                |
| JSON Merge Patch / JSON Patch auto-PATCH             |           ❌            | ✅                                                                |
| Conditional-request _evaluation_ (If-Match…)         | partial (If-None-Match) | ✅                                                                |

## httputil's genuine niche (what huma doesn't do)

1. **Compression** — RFC 7231 negotiation, writer pooling, content-type deny-list. Huma has _nothing_ here.
2. **ETag from body bytes** — huma only evaluates headers _you_ supply; httputil generates them.
3. **The full security/observability/resilience middleware wall** — exactly the layer huma explicitly offloaded.
4. **`httpspec`** — a router-agnostic "does any handler behave like HTTP?" harness. Unique; neither huma nor chi ships this.
5. **Retry-classified errors** — `go-error-family` families are a different (and complementary) axis to huma's RFC 9457.

## What huma does that httputil doesn't (roadmap fuel)

The big gaps, if you ever wanted to move "up the stack": **typed handlers, schema-driven validation, OpenAPI generation, problem+json errors, body content negotiation.** These are large undertakings (huma is a whole framework) and would change httputil's identity — but they're the high-leverage features that make huma popular.

## Shared foundation: Go 1.22+ `http.ServeMux`

A detail that tightens the fit: both libraries are **stdlib-native to the same router**.

- **httputil** targets pure `net/http` and already uses Go 1.22+ method-pattern syntax (e.g. `mux.HandleFunc("GET /health", ...)` in the README).
- **huma** ships a first-party adapter, [`humago`](https://pkg.go.dev/github.com/danielgtaylor/huma/v2/adapters/humago), that wraps the same stdlib `http.ServeMux` (requires `go 1.22` in `go.mod`; httputil requires `go 1.26`).

So there is **no third-party router in the critical path** — no chi, no gorilla, no fiber. huma's broader "[Bring Your Own Router](https://huma.rocks/features/bring-your-own-router/)" strategy (chi, gin, echo, fiber, gorilla/mux, bunrouter, httprouter, and stdlib mux) means it can meet you wherever you are, but the `humago` path is the one where the two libraries align exactly: same mux, same `http.Handler` signature, zero router glue.

## How they combine

They stack cleanly — httputil _underneath_, huma _on top_, both over the same stdlib mux:

```go
router := http.NewServeMux()
api := humago.New(router, huma.DefaultConfig("My API", "1.0.0"))
huma.Get(api, "/things/{id}", getThing) // huma: types, validation, OpenAPI

handler := httputil.Chain(router, // httputil: plumbing
    httputil.Logging(slog.Default()),
    httputil.Recovery(slog.Default()),
    httputil.Compression(httputil.DefaultCompressionConfig()),
    httputil.SecurityHeaders(httputil.DefaultSecurityHeadersConfig()),
    httputil.CORS(httputil.DefaultCORSConfig()),
)
```

## Honest positioning

- **huma** = "FastAPI for Go": contract-first API development with living docs. Broad appeal, large ecosystem.
- **httputil** = the cross-cutting-concerns layer that huma (and chi, and stdlib mux) intentionally leaves to you. It's a focused, well-engineered **complement** to any of them, not an alternative.

**Bottom line:** comparing them head-to-head is a category error. The interesting question isn't "httputil _or_ huma" — it's "httputil _underneath_ huma/chi/mux." httputil's real peers are libraries like `go-chi/chi/v5/middleware`, `rs/cors`, and `tommy351/gin-compress` — not huma.
