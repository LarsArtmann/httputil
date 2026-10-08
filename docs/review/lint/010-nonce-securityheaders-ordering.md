# LINT-010: `Nonce` composed outside `SecurityHeaders` (documented ordering inverted)

- **Date:** 2026-10-08
- **Severity:** Low (latent)
- **Status:** Open
- **Consumers:** crush-daily, bank-sync
- **Ground truth:** AGENTS.md, `Nonce` bullet: "When chaining with `SecurityHeaders`, place `Nonce` **after** (inner to) `SecurityHeaders` so the nonce-bearing CSP overwrites any static CSP — the default `SecurityHeadersConfig` sets no CSP, so there is no conflict unless you explicitly set `ContentSecurityPolicy` in both." Pinned by httputil's `TestChain_NonceInnerToSecurityHeaders_OverwritesStaticCSP`.

## What the consumers do

### crush-daily (`internal/server/server.go:377-392`)

```go
return cqrshtmx.Chain(
	s.app.RecoverHandler(),
	cqrshtmx.RequestLoggingSlog(slog.Default()),
	cspNonceMiddleware,        // wraps httputil.Nonce with a CSP builder (csp.go:40)
	httputil.SecurityHeaders(securityCfg),
	...
```

### bank-sync (`internal/server/server.go:903-913`)

```go
httputil.Nonce(httputil.NonceConfig{CSPBuilder: dashboardCSP}),
httputil.SecurityHeaders(httputil.DefaultSecurityHeadersConfig()),
```

Chain semantics put the first entry outermost, so in both repos the
nonce-bearing CSP is written *before* `SecurityHeaders` runs inner.

## Why it is wrong (and why only "latent")

`DefaultSecurityHeadersConfig` sets no CSP today, so the inversion is
harmless at these exact configs — which is precisely why it survived.
The moment either repo sets a static `ContentSecurityPolicy` in
`SecurityHeadersConfig` (hardening pass, scanner finding, copy from
`RecommendedCSP`), the static header silently overwrites the
per-request nonce policy and every inline script on the site breaks —
or worse, someone "fixes" the breakage by weakening the CSP to
`unsafe-inline`, which is the failure mode the ordering rule exists to
prevent. The rule and its test exist because this exact composition bug
was caught once already; the correct in-fleet shapes are
file-and-image-renamer `healthd` (Recovery → SecurityHeaders → Nonce,
healthd/middleware.go:52-54) and cqrs-htmx's
`RecommendedSecurityMiddleware` (SecurityHeaders → Nonce → Recovery).

## Fix

Swap the two entries in both chains and add a chain-order assertion
test (render two responses, one with a static CSP in SecurityHeaders,
and pin that the nonce policy wins) so the ordering cannot regress
under refactor.
