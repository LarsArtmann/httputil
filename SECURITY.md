# Security Policy

## Supported Versions

httputil is pre-1.0. Only the latest release receives security fixes.

| Version  | Supported          |
| -------- | ------------------ |
| latest   | :white_check_mark: |
| < latest | :x:                |

Once v1.0 is released, the latest minor within the current major will be supported.

## Reporting a Vulnerability

Email **git@lars.software** with a description of the issue, reproduction steps, and impact assessment.

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
- **Rate limiting**: `TokenBucketLimiter` is in-memory per-instance. For distributed deployments, provide a custom `RateLimiter` implementation (e.g., Redis-backed).
- **CORS wildcard fallback**: unmatched origins fall back to `"*"` by default. Set `DenyUnmatched: true` for security-hardened deployments.
- **Dependencies**: only `go-error-family` (same author, zero transitive deps) and `golang.org/x/time` (canonical Go extension). No third-party attack surface.
