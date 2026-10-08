# LINT-014: Per-request CSP nonces served without explicit `Cache-Control: no-store`

- **Date:** 2026-10-08
- **Severity:** Low (conditional on a shared cache being placed in front)
- **Status:** Open
- **Consumers:** cqrs-htmx adminui/dashboardui, ChastityAPI dashboard, crush-daily, timesheets
- **Ground truth:** AGENTS.md, `Nonce` bullet: "Responses with per-request nonces must not be cached (set `Cache-Control: no-store`)."

## What the consumers do

All four render HTML through `httputil.Nonce` (per-request nonce baked
into the `Content-Security-Policy` header and inline `nonce="..."`
attributes):

- cqrs-htmx `adminui`/`dashboardui` (via `RecommendedSecurityMiddleware`)
  set no `Cache-Control` on their HTML responses (grep across both
  packages finds no Cache-Control outside tests).
- ChastityAPI's dashboard (`internal/csp/csp.go` + templ handlers)
  similarly relies on default caching semantics.
- crush-daily and timesheets serve nonce'd templ pages with no
  no-store on the HTML routes (timesheets deliberately scopes CSP to
  dashboard paths; crush-daily's csp.go wrapper likewise).

Positive contrast: DiscordSync chains a `defaultCacheControl`
middleware that stamps explicit caching directives on every response
exactly because "HTML pages and API JSON are never heuristically
cached" (internal/api/middleware.go:13).

## Why it is wrong

A nonce is single-request by construction. If any shared cache sits in
front (Cloudflare in front of a dashboard, an nginx proxy_cache during
a migration, a corporate cache), one response's CSP/nonce pair is
served to other users: their inline scripts fail the nonce check
(availability bug), or — with `Builtins`-style static nonce fallbacks —
the nonce effectively becomes shared, which defeats the injection
defense. The failure only manifests after an infra change nobody
associates with these pages, which is why the library documents the
requirement rather than enforcing it (caching is invisible to
middleware).

## Fix

Stamp `Cache-Control: no-store` (or `no-cache` + `private`) on all
nonce'd HTML routes, ideally once at the chain level the way
DiscordSync does, with static-asset handlers overriding to `immutable`.
For cqrs-htmx the natural home is `RecommendedSecurityMiddleware`
(next to the nonce it already manages), which fixes every consumer at
once.
