# LINT-007: `httputil.ClientIP` consumed for security decisions and security-relevant logging without a trust boundary

- **Date:** 2026-10-08
- **Severity:** Medium (KeyCountdown session binding), Low (logging)
- **Status:** Open
- **Consumers:** games/KeyCountdown, Rolls-Royce-mtuGoHelpCenter-golang, DiscordSync, cqrs-htmx `requestmeta`, dynamic-markdown-site
- **Ground truth:** AGENTS.md ClientIP trust caveat (see LINT-006); attacker-controlled log fields are a log-injection surface

## What the consumers do

- **games/KeyCountdown** `internal/validation/security.go:400, 478`:
  `generateSessionID` / `generateWebAuthnSessionID` bind CSRF-ish
  session identity to `httputil.ClientIP(r) + User-Agent`, with an
  in-code admission ("In production, you'd want proper session
  management"). The client fully controls both inputs, so the "session
  identifier" is attacker-chosen: rotating XFF rotates the identity and
  defeats any server-side correlation built on it. `logging/structured.go:723`
  also logs the spoofable value as `client_ip`.
- **Rolls-Royce-mtuGoHelpCenter-golang** `internal/api/middleware/common.go:15`:
  request logging via `httputil.ClientIP` — spoofable values reach the
  log pipeline unescaped (low: logging only).
- **DiscordSync** `internal/api/middleware.go:56`: same logging pattern.
- **cqrs-htmx** `requestmeta.go:98`: ClientIP parsed into event
  metadata — spoofed IPs propagate into persisted audit/event data, not
  just transient logs, so the falsification outlives the request.
- **dynamic-markdown-site** `internal/server/helpers.go:24`: `clientIP()`
  helper exported to the rest of the server (rate limiting per LINT-006
  plus logging).

## Why it is wrong

Every one of these treats a client-controlled string as an
infrastructure-attested fact. For logging the cost is polluted analytics
and injection-shaped log entries; for KeyCountdown the cost is that a
security classification ("session identity") inherits the spoofability.
cqrs-htmx's case is the most durable: event metadata becomes part of
the audit trail its consumers (dnsblockd, KeyHolderAI) rely on.

## Fix

- Route ClientIP reads through the fleet's trusted-proxy pattern
  (LINT-006 table) so unproxied XFF never wins.
- KeyCountdown: replace IP+UA hashing with a real server-issued session
  ID (its own comment already concedes this); the httputil nonce/CSRF
  machinery it partially reimplements is already a dependency of the
  cqrs-htmx stack it builds on.
- cqrs-htmx requestmeta: mark client_ip as client-claimed (field name
  or log annotation) until it is proxy-attested.
