# LINT-002: CORS allow-all configurations on authenticated or production APIs

- **Date:** 2026-10-08
- **Severity:** Medium (ksef-sandbox, e-invoicing, storbi, ChastityAPI), Info (sales-landing-page, middleware-showcase example)
- **Status:** Open
- **Consumers:** ksef-sandbox, e-invoicing, storbi, ChastityAPI, sales-landing-page/cloud-run, cqrs-htmx `examples/middleware-showcase`
- **Ground truth:** `cors.go` `resolveOrigin()` — origin matching is exact-string or `*.suffix` wildcard patterns (`matchWildcardOrigin`, cors.go:197); `DefaultCORSConfig()` (cors.go:48) is an allow-all dev default

## What the consumers do

### ksef-sandbox and e-invoicing (`internal/http/middleware.go:98`, byte-identical files)

```go
httputil.CORS(httputil.CORSConfig{
	AllowedOrigins:  []string{"*"},
	AllowAllOrigins: true,
	AllowedMethods:  []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
	...
})
```

### storbi (`storbi/internal/middleware/middleware.go:111`)

```go
AllowedOrigins: []string{"https://*", "http://*"},
...
AllowAllOrigins: true,
```

with a comment that production "should narrow AllowedOrigins" once the
cookie strategy is finalized.

### ChastityAPI (`ChastityAPI/cmd/server/main.go:490`)

`httputil.DefaultCORSConfig()` verbatim — which is `AllowAllOrigins: true`
(cors.go:51) — on an API with accounts and a dashboard.

### sales-landing-page/cloud-run (`main.go:101`)

Allow-all on a public PostHog analytics proxy. Intentional for a proxy
endpoint with no cookies; recorded for completeness.

### cqrs-htmx `examples/middleware-showcase/main.go:127`

The example teaches `httputil.DefaultCORSConfig()` with the comment
"DefaultCORSConfig allows all origins (dev-friendly). Restrict
AllowedOrigins in production." Examples propagate: template-arch-lint and
go-website-template are the seeds for new projects.

## Why it is wrong

- ksef-sandbox/e-invoicing are e-invoicing products (invoice submission,
  auth surfaces). Allow-all CORS on them means any origin can read
  non-credentialed API responses and drive preflighted state-changing
  requests; CSRF defense then rests entirely on cookie SameSite. There
  is no environment split — the wildcard is unconditional.
- storbi's two origin entries are **dead config**: `matchWildcardOrigin`
  only expands patterns of the form `*.example.com`; `https://*` matches
  no real origin and is silently ignored. The config is allow-all only
  because `AllowAllOrigins: true` is also set, so the reader who removes
  that flag "to lock it down" actually locks out every cross-origin
  client at once. The comment admits the end state but nothing tracks it.
- ChastityAPI adopting `DefaultCORSConfig()` wholesale is the exact
  footgun the httputil docs warn about: the default is a dev posture.
  The API serves public content, so impact is bounded, but any endpoint
  that later grows authenticated JSON becomes readable from any origin
  with no config change.

## Fix

- Make the origin list environment-driven with a fail-closed production
  default (`DenyUnmatched: true` + explicit list, like
  german-business-contract-automation and nsfw-classifier already do —
  those two are the correct in-fleet reference patterns).
- storbi: delete the dead `https://*`/`http://*` entries and add the
  tracked production list now (the comment is older than the code
  around it).
- ChastityAPI: start from `DefaultCORSConfig()` and override
  `AllowedOrigins`/`AllowAllOrigins` per environment, mirroring the
  nsfw-classifier `Handler()` pattern (server.go:723).
