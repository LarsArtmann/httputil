# LINT-001: CORS wildcard origin echo combined with `AllowCredentials: true`

- **Date:** 2026-10-08
- **Severity:** High (GmbH), Medium (blog)
- **Status:** Open
- **Consumers:** GmbH, blog
- **Ground truth:** `cors.go` `resolveOrigin()` (httputil v1.4.2) and `CORSConfig.Validate` (cors.go:81): `AllowCredentials=true with AllowAllOrigins=true is not permitted by the CORS spec`

## What the consumers do

### GmbH (`GmbH/server/app/server.go:184`)

```go
httputil.CORS(httputil.CORSConfig{
	AllowedOrigins:   []string{"*"},
	AllowedMethods:   []string{http.MethodGet, http.MethodPost, ...},
	AllowedHeaders:   []string{"Origin", "Content-Type", "Authorization", "X-Requested-With"},
	AllowCredentials: true,
	MaxAge:           86400,
}),
```

This chain fronts `POST /api/v1/auth/login|register|refresh|logout` and
`GET /api/v1/auth/profile` (same file, below the chain) — cookie/session
endpoints.

### blog (`blog/internal/server/server.go:122`)

```go
httputil.CORS(httputil.CORSConfig{
	AllowedOrigins:   cfg.Security.CORSAllowedOrigins,
	AllowAllOrigins:  len(cfg.Security.CORSAllowedOrigins) == 1 && cfg.Security.CORSAllowedOrigins[0] == "*",
	AllowCredentials: true, // hardcoded, never derived from config
	...
})
```

`blog/internal/config/config.go:115` sets the viper default
`security.cors_allowed_origins = ["*"]`, so any environment without an
explicit `CORS_ORIGINS` value runs the wildcard configuration. An empty
`CORS_ORIGINS` env var produces the list `[""]`, which is also not `"*"`,
and lands in the same resolveOrigin path described below.

## Why it is wrong

httputil's `resolveOrigin` treats a literal `"*"` entry in `AllowedOrigins`
as match-everything **and echoes the request's Origin header back** as
`Access-Control-Allow-Origin` (it returns `origin`, not `"*"`). Combined
with `Access-Control-Allow-Credentials: true`, the effective policy is
"any origin may make credentialed requests and read the responses".

This is exactly the combination the CORS spec forbids and that
`CORSConfig.Validate` rejects — but both consumers set `AllowedOrigins:
["*"]` (the literal) instead of `AllowAllOrigins: true`, which sidesteps
the Validate warning while achieving the same or worse behavior (echo
instead of `*`). Validate-and-log does not abort construction by design,
so nothing at runtime stops it.

Impact differs by consumer:

- **GmbH:** any web page can issue credentialed `fetch()` calls against
  the API and read authenticated JSON (profile, token refresh). Whether
  session cookies attach depends on the cookie's SameSite attribute;
  GmbH sets no SameSite anywhere in `GmbH/server` (verified by grep), so
  browsers without Lax-by-default (notably Safari-family) attach them.
  This defeats the read-side protection CORS exists for, on top of the
  missing CSRF middleware (see LINT-003).
- **blog:** wildcard + credentials produces the spec-violating header
  pair (`*` + `Allow-Credentials: true` via `AllowAllOrigins`), which
  browsers reject outright, so credentialed cross-origin requests break
  silently — an availability bug that will surface the day the frontend
  is served from a different origin. With explicit origins the config is
  correct; the hardcoded `AllowCredentials: true` makes correctness
  config-dependent and the default unsafe.

## Fix

1. GmbH: replace `AllowedOrigins: ["*"]` with the actual deployment
   origins (they are known: the GmbH site), or set `AllowAllOrigins` and
   drop credentials. `Validate` then guards the invariant.
2. blog: derive `AllowCredentials` from configuration instead of
   hardcoding `true`, and reject the wildcard value at config-load time
   (fail fast in `config.Load`, not at request time).
3. Both: add a test asserting `cfg.Validate() == nil` on the production
   config; httputil's Rejection for this combination is the intended
   tripwire.
