# LINT-012: Hand-rolled middleware duplicating (or mis-duplicating) httputil surface

- **Date:** 2026-10-08
- **Severity:** Low (consolidation), with one correctness bug (PapDashboard CORS)
- **Status:** Open
- **Consumers:** PapDashboard, ksef-sandbox/e-invoicing, german-business-contract-automation, dynamic-markdown-site, RedditParse (partial)
- **Ground truth:** the httputil middleware set (CORS, SecurityHeaders, Recovery, Metrics) and the `MetricsRecorder` extension point blog uses

## What the consumers do

### PapDashboard — hand-rolled CORS without preflight handling (`cmd/server/main.go:639`)

The custom `addCORSMiddleware` sets only
`Access-Control-Allow-Origin` (echoing the request origin for
allowlisted origins or `*`). It never sets `Access-Control-Allow-Methods`
or `-Allow-Headers` and never answers preflights — so any browser
cross-origin caller gets a preflight response that the browser must
reject (no `Access-Control-Allow-Methods` on an OPTIONS response). The
httputil `CORS` middleware it duplicates handles preflight 204s,
method/header joins, MaxAge, and `OptionsPassthrough`. PapDashboard
also reimplements metrics via its own `NewStatusRecorder` +
in-flight/latency plumbing (main.go:606) where `httputil.Metrics` with
a `MetricsRecorder` is the extension point blog already uses
correctly (blog/internal/server/metrics.go:71, including `PathFunc`
cardinality control), and its `OriginGuard`
(internal/middleware/origin.go) reimplements the cross-origin
state-change check that `CSRFMiddleware` provides with token binding.

### ksef-sandbox / e-invoicing (`internal/http/middleware.go`)

Hand-rolled `SecurityHeaders` (static header list) plus a hand-rolled
panic `Recovery` (`recover()` + `http.Error(w, "Internal Server
Error", 500)`), directly adjacent to their use of `httputil.CORS`.
Their Recovery, like the Rolls one in LINT-009, does not re-panic
`http.ErrAbortHandler` and writes after arbitrary handler output.

### german-business-contract-automation (`internal/server/server.go:342`) and dynamic-markdown-site (`internal/server/handlers.go:83`)

Static `X-Content-Type-Options`/`X-Frame-Options`/`X-XSS-Protection`
header writers next to real httputil usage elsewhere in the same
chain. `X-XSS-Protection` is deprecated and the hand-rolled set lacks
HSTS/Permissions-Policy that `DefaultSecurityHeadersConfig` and
`RecommendedHSTS` provide.

## Why it is wrong

Every parallel implementation is a place where httputil's fixes stop
propagating: the v1.4.2 `Metrics` nil-Recorder fix, the pattern
propagation fixes, CSP/HSTS defaults, and preflight correctness all
bypass hand-rolled twins. PapDashboard's CORS twin is not just
duplication — it is functionally broken for its stated purpose
(preflighted cross-origin calls cannot succeed), which is the usual
outcome of reimplementing this surface.

## Fix

- PapDashboard: swap `addCORSMiddleware` for `httputil.CORS` with the
  config it already loads; move metrics onto `httputil.Metrics` +
  `MetricsRecorder`; keep `OriginGuard` only if API-key clients (no
  cookies/tokens in ambient storage) are the sole state-changing
  callers — otherwise adopt `CSRFMiddleware`.
- ksef/e-invoicing, GermanBA, dynamic-markdown-site: replace the
  hand-rolled header/panic middleware with `SecurityHeaders` +
  `Recovery` in the existing chains (both repos already chain httputil
  middleware, so the marginal change is deletion).
