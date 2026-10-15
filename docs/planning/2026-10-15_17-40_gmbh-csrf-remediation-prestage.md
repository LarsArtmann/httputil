# GmbH CSRF Remediation — Pre-Staged Plan (read-only until the R10 ruling)

_Created: 2026-10-15 17:40 CEST · Source: pareto plan P100.22 · Gate: the consumer fix-PR operating-model ruling (decision brief R10; nothing here is executed against the GmbH repo until it lands)_

## What the verified evidence already dissolved

The consumer-audit verification log (`docs/review/lint/000-index.md` §GmbH, verified 2026-10-10) resolved the cookie half of LINT-003 **negative**: GmbH's Go server sets no cookies anywhere (`http.Cookie{`/`SetCookie`/`SameSite`: zero non-vendor hits); auth is header-based JWT (`server/app/server.go:302`, `server/infrastructure/auth/middleware.go:14`). A CSRF-cookie remediation (Secure/SameSite hardening) for GmbH is therefore **vacuous** — there is no cookie to harden.

The JWT-header auth model also structurally defeats classic CSRF for the token-bearing endpoints: a cross-site attacker cannot attach an `Authorization` header to a victim's browser request, so the write-side risk does not come from CSRF at all.

## What remains real (the actual exposure)

1. **LINT-001 — CORS wildcard origin echo with `AllowCredentials`** (GmbH is a named consumer). With no cookies this is weaker than the fleet's worst case, but echoing arbitrary origins while allowing credentials is still wrong-by-construction and becomes live the moment a cookie or a `credentials: 'include'` frontend call appears. This — not CSRF — is GmbH's real cross-origin finding.
2. **Defense-in-depth CSRF middleware is optional, not needed** — adding `httputil.CSRFMiddleware` would protect future cookie-based flows (session upgrade, cookie consent features) but has nothing to protect today.

## The conditional minimal plan (executes only after R10 approves fix-PRs)

| Step | Change | File | Risk |
| --- | --- | --- | --- |
| 1 | Replace wildcard origin echo with an explicit origin allowlist (env-configured), `AllowCredentials` only for allowlisted origins | wherever the CORS config is built (near `server/app/server.go:182` chain) | Low — frontend origins are known and few |
| 2 | (Optional, only if cookie flows are planned) add `httputil.CSRFMiddleware` starting from `DefaultCORSConfig()`-style secure values with `Secure: true` | server chain | Low — additive |
| 3 | Re-run the audit pattern grep (`scripts/consumer-audit/run-audit.sh`) against GmbH to confirm the 001/003 entries clear | audit corpus | None |

Step 2 is explicitly NOT recommended as part of a fix-PR: it adds a token endpoint + frontend changes for a class with no live exposure. It is listed so the R10 ruling sees the full option space.

## Verdict

Pre-staged and parked. The plan's center of mass moved from "CSRF cookies" to "CORS allowlist" on the strength of the 2026-10-10 verification; any R10 approval should treat LINT-001 (not 003) as the GmbH work item.
