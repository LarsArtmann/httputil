# Security Policy

## Supported Versions

httputil is v1.x (API frozen as of v1.0). Only the latest release receives security fixes.

| Version  | Supported          |
| -------- | ------------------ |
| latest   | :white_check_mark: |
| < latest | :x:                |

Within the current major version, the latest minor/patch release is supported.

## Reporting a Vulnerability

Email <git@lars.software> with a description of the issue, reproduction steps, and impact assessment.

- **Do not** open a public GitHub issue for security vulnerabilities.
- You will receive an acknowledgment within **48 hours**.
- A fix or mitigation will be prioritized based on severity:
  - **Critical** (RCE, auth bypass): patch within 24 hours of confirmation.
  - **High** (data leak, DoS): patch within 72 hours.
  - **Medium/Low**: next release cycle.

## Disclosure

- Once a fix is released, the vulnerability will be documented in the GitHub Release notes and CHANGELOG.
- Coordinated disclosure is preferred — please allow time for a fix before public disclosure.
- Credit will be given to reporters unless they prefer to remain anonymous.

## Scope

This policy covers the `httputil` package and its `httpspec` subpackage.

Out of scope:

- Vulnerabilities in dependencies (`go-error-family`, `golang.org/x/time`) — report to the respective upstream maintainers.
- Social engineering attacks against the maintainer.
- Issues requiring access to an already-compromised system.

## Security Posture

httputil handles untrusted HTTP input. Security-relevant behaviors:

- **CORS**: `ClientIP` trusts proxy headers without validation. Only safe behind a reverse proxy that strips or overwrites `X-Forwarded-For` and `X-Real-IP`.
- **Rate limiting**: `KeyedRateLimiter` is in-memory per-instance. For distributed deployments, front `KeyedRateLimiterMiddleware` with a proxy-level limiter (e.g., at your reverse proxy or gateway).
- **CORS wildcard fallback**: `DefaultCORSConfig()` sets `DenyUnmatched: true` (unmatched origins get no `Access-Control-Allow-Origin`). A bare `CORSConfig{}` literal carries the zero value, which falls back to `"*"` — start from `DefaultCORSConfig()` or set `DenyUnmatched` explicitly for hardened deployments.
- **CORS private-network access**: `AllowPrivateNetwork` is opt-in and off by default. Enabling it answers Chrome's Local Network Access preflights, granting cross-origin pages from a less-private address space permission to fetch this origin's LAN/localhost subresources — an explicit decision, never a default.
- **CSRF**: `SameSite=None` without `Secure` is remediated to `Secure=true` at construction (rfc6265bis-compliant browsers refuse to store the combination, so the cookie would silently never persist); `AllowInsecureSameSiteNone` opts out for legacy-client deployments only. The fallback logs with the `csrf_samesite_insecure` code — configs never weaken silently.
- **Dependencies**: `go-error-family` (same author, zero transitive deps), `golang.org/x/time` (canonical Go rate-limit extension), `github.com/justinas/nosurf` (CSRF double-submit cookie — security-critical and complex to hand-roll), and `go-etag` (same author; error-code registration). No other third-party attack surface.
