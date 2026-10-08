# LINT-003: Zero-value `CSRFConfig` ships non-Secure CSRF cookies in production paths

- **Date:** 2026-10-08
- **Severity:** High (CV logic bug), Medium (cqrs-htmx bundle default, games/SEC, timesheets, Zlota44)
- **Status:** Open
- **Consumers:** CV, cqrs-htmx `setup`, games/SEC, timesheets, Zlota44, GmbH (absent entirely)
- **Ground truth:** `csrf.go` CSRFConfig field docs — `Secure bool` "Default: false"; `DefaultCSRFCookieName`, `Path` default "/". The zero value is a deliberate plaintext-cookie configuration, not a secure default.

## What the consumers do

### CV — inverted production logic (`CV/internal/server/middleware_chain.go:139`)

```go
func csrfConfig(cfg *config.Config) httputil.CSRFConfig {
	secCfg := httputil.CSRFConfig{}
	if !cfg.IsProduction() {
		secCfg.Secure = false // already the zero value; production never sets it true
	}
	secCfg.TrustedOrigins = cfg.GetAllowedOrigins()
	return secCfg
}
```

The conditional can only ever write the value the zero value already
has. Production serves the CSRF cookie without the `Secure` flag. A
separate zero-value duplicate exists at `CV/internal/di/handlers.go:381`
(`csrfCookie := httputil.CSRFConfig{}` for `InvalidateCSRFCookie`), so
the cookie name/path/Secure of issuance and invalidation can drift.

### cqrs-htmx setup bundle default (`cqrs-htmx/setup/bundle.go:135`)

```go
// The configuration comes from [Config.CSRF]; the default (nil) matches the
// zero httputil.CSRFConfig (no Secure flag — see that field for production).
if csrfCfg == nil {
	csrfCfg = &httputil.CSRFConfig{}
}
```

The bundle is the "production wiring" entry point, and its documented
default is the plaintext cookie. `crm`/`crm-exec-stage` pass
`&httputil.CSRFConfig{Secure: true}` explicitly (identity.go:150) —
proving the zero-value default is something consumers must know to
override, not a posture the bundle enforces.

### games/SEC (`games/SEC/server/middleware.go:14`) and timesheets (`timesheets/internal/server/server.go:195`)

Both construct `CSRFConfig` with `ErrorHandler`/`SameSite` only and rely
on zero-value `Secure: false` in deployed web apps.

### Zlota44 (`Zlota44/internal/server/server.go:86`)

`CSRFConfig{TrustedProxies: []string{"127.0.0.1", "::1"}}` — proxy trust
is wired (good) but `Secure` is left false.

### GmbH — no CSRF middleware at all

`GmbH/server/app/server.go:182` chains CORS + security headers + logging
only. Login/register/refresh/profile have no CSRF token flow and no
SameSite attribute on session cookies (grep across `GmbH/server` finds
no `SameSite` in non-test Go files). Defense rests entirely on the
browser SameSite default; combined with LINT-001's credentialed
wildcard-echo CORS, the API has neither read-side nor write-side
cross-origin protection that does not depend on browser defaults.

## Why it is wrong

A CSRF cookie without `Secure` travels over plaintext connections; on
HTTPS deployments it leaks the token on any downgrade/mitM opportunity
and signals mixed-precision security posture. httputil deliberately
defaults `Secure` to false (plaintext dev support), so the library will
never fix this silently — the consumer must opt in. The contrast inside
the fleet (crm, InboxClean with `csrfSecure`, SwettySwipperWeb with
`Secure: !isDev`) shows the intended pattern exists and is being
applied inconsistently.

CV's variant is worse than a missing flag: the `if !cfg.IsProduction()`
block documents an intent ("disable Secure in dev") that the code does
not and cannot implement — the dead branch will mislead the next editor
into believing production sets `Secure: true`.

## Fix

1. CV: `secCfg.Secure = cfg.IsProduction()` (or thread a config flag);
   delete the dead branch. Build the invalidation config from the same
   source (`csrfConfig(cfg)`) instead of a second zero value.
2. cqrs-htmx `setup`: make the nil default secure (Secure: true or an
   `Env`-derived value) and keep an explicit opt-out for plaintext dev.
   The doc comment currently advertises the insecure default.
3. games/SEC, timesheets, Zlota44: set `Secure` from an environment
   flag; assert `cfg.Validate() == nil` at startup so misconfigurations
   surface in logs at boot.
4. GmbH: adopt `httputil.CSRFMiddleware` (it is already a dependency)
   and set `SameSite` + `Secure` on the session cookie.
