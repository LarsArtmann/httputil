# LINT-004: CSRF deployments behind TLS termination without `TrustedProxies`

- **Date:** 2026-10-08
- **Severity:** Medium
- **Status:** Open
- **Consumers:** cqrs-htmx `setup` default, timesheets, games/SEC, Zlota44 (partial), CV (partial)
- **Ground truth:** `csrf.go` CSRFConfig.TrustedProxies doc — the plaintext-HTTP origin bypass and the scheme resolution (`requestScheme`) honor `X-Forwarded-Proto` only from `TrustedProxies`/`TrustedProxiesCIDR` addresses; AGENTS.md "CSRF origin-attestation trust boundary" section

## The failure mode (already proven in-fleet)

InboxClean documented it exactly (`InboxClean/internal/web/server.go:828`):

> Without it, a browser's truthful https Origin behind the TLS front
> contradicts the locally-seen http scheme and every browser POST is
> rejected as a forged attestation (403) before the token pair is even
> checked.

nosurf compares the `Origin` scheme/host against the server-local view
of the request. Behind a TLS-terminating proxy the local view is
`http://`, the browser's Origin is `https://`, and state-changing
requests 403 unless either (a) the proxy IP is in `TrustedProxies` so
`X-Forwarded-Proto` is honored, or (b) the public origin is listed in
`TrustedOrigins` so the origin is accepted as an explicitly trusted
cross-origin entry.

## What the consumers do

| Consumer | TrustedProxies | TrustedOrigins | TLS-terminated deploy |
|---|---|---|---|
| cqrs-htmx `setup` default | nil | nil | bundle consumers behind a proxy hit the 403 wall the moment they enable CSRF |
| timesheets (server.go:195) | none | none | yes (dashboard product) |
| games/SEC (middleware.go:14) | none | none | unknown, likely behind nginx |
| Zlota44 (server.go:86) | loopback only | none | covered only if the TLS front is on-host |
| CV (middleware_chain.go:144) | none | config origins | saved by TrustedOrigins exact-match, not by design |

Contrast with the consumers that solved it: InboxClean
(`forwarderTrustCIDRs` from the app-wide forwarder trust boundary),
webphone (`cfg.n.TrustedProxies` + `TrustedOrigins` both logged at
boot), crm/crm-exec-stage via the bundle with explicit `Secure: true`
configs.

## Why it is wrong

Each of these deployments either (a) intermittently 403s legitimate
browser traffic behind the proxy, or (b) has not enabled CSRF at all in
the protected environment because "it breaks behind nginx" — which is
the vulnerability LINT-003/GmbH records. The library provides the exact
mechanism (`TrustedProxies`); not wiring it converts a documented,
solvable misconfiguration into either breakage or abandonment of CSRF.

There is a secondary trap: `TrustedOrigins` exact-match rescues the
scheme mismatch (as with CV), but only for origins listed verbatim. A
`www.` vs apex mismatch or a staging origin silently loses coverage.

## Fix

- Every deployment that terminates TLS in a proxy: set
  `TrustedProxies` to the proxy addresses/CIDRs (the same list the app
  already trusts for `ClientIP`, e.g. CV's `httpx.SetTrustedProxies`
  config) and mirror the public origins into `TrustedOrigins`.
- cqrs-htmx `setup`: add `TrustedProxies` to the bundle's production
  story (config passthrough exists; the nil default is the trap).
- Add a boot-time log when CSRF is enabled and `TrustedProxies` is nil
  (httputil logs construction rejections only for invalid values; a nil
  list is valid, so the deployment-level gap is invisible today).
