# CSP Nonce Management: Architecture Analysis

**Date:** 2026-08-09
**Scope:** How httputil's CSP nonce middleware works, what broke in go-health-dashboard, and whether nonce should be extracted into a dedicated project.

---

## 1. How CSP Nonce Management Works (httputil)

The `Nonce()` middleware in `nonce.go:131` implements a **per-request** nonce lifecycle:

| Stage | What happens | API |
|---|---|---|
| **Generate** | `crypto/rand` produces 20 random bytes (160 bits), base64-URL-encoded | `generateNonce(size)` |
| **Set header** | `CSPBuilder func(nonce string) string` renders the CSP string and sets `Content-Security-Policy` | `RecommendedCSPWithNonce` / `ProductionCSPWithNonce` |
| **Store** | Nonce injected into `context.Context` | `WithNonce(ctx, nonce)` |
| **Retrieve** | Handlers/templates read it back | `NonceFromRequest(r)`, `NonceAttr(r)` -> `nonce="<value>"` |

Every request gets a **fresh, unique nonce**. The browser matches the nonce in the CSP `script-src`/`style-src` directives against `nonce="..."` attributes on inline `<script>`/`<style>` tags -- mismatched inline content is blocked, eliminating the need for `'unsafe-inline'`.

Two prebuilt policies: `RecommendedCSPWithNonce` (script-src + style-src) and `ProductionCSPWithNonce` (adds `object-src 'none'`, `base-uri 'self'`, `frame-ancestors 'none'`). The `CSPBuilder` field makes the CSP fully pluggable.

---

## 2. What Broke in go-health-dashboard

The status report (`go-health-dashboard/docs/status/2026-08-09_00-41_csp-nonce-misplacement-fix.md`) documents a **fundamental model mismatch**:

- **httputil** generates a **new random nonce per request**.
- **go-health-dashboard** accepts a **static nonce at construction time** via `WithNonce(nonce string)` (`dashboard.go:63`) -- one fixed string for the dashboard's entire lifetime.

The consumer (`file-and-image-renamer`) bridged this with a hardcoded constant (`HealthDashboardNonce`) and a `dashboardCSPMiddleware` that **appended** the nonce to the end of the CSP string:

```go
csp += " 'nonce-" + HealthDashboardNonce + "'"
```

Since `frame-ancestors 'none'` was the **last directive**, the nonce landed there. This caused a cascade:

1. `frame-ancestors` does not support nonce source expressions, so the browser **ignored the entire directive** -- **clickjacking protection silently disabled** (because `'none'` must stand alone).
2. `script-src` never received the nonce -- **all inline scripts blocked** (theme toggle, Datastar SDK, FOUC prevention).

The fix replaced string-append with a full CSP rebuild via `cspHeader(HealthDashboardNonce)`. But this is a **band-aid over the real disease**: the static nonce itself defeats CSP's purpose (a known constant nonce lets any injected script pass).

---

## 3. Can and Should We Extract It Into a Dedicated Project?

### Can?

Yes -- `nonce.go` is fully self-contained (stdlib-only, 192 lines). The only coupling is the `Middleware` type alias from `recorder.go`, trivially replaced by `func(http.Handler) http.Handler`.

### Should? No.

| Factor | Assessment |
|---|---|
| **Flat-package decision** | AGENTS.md confirms (2026-08-05, user-approved): for a `func(http.Handler) http.Handler` library, one import path (`httputil.Nonce()`) beats fragmented namespaces. Extraction contradicts this. |
| **Code volume** | 192 lines -- a whole repo/module for that is overhead-heavy with no payoff. |
| **Pattern consistency** | Nonce follows the identical middleware signature as CORS, Compression, SecurityHeaders, etc. It belongs with its siblings. |
| **CSP cohesion** | `security.go` already owns `RecommendedCSP` + `SecurityHeaders`; nonce is the per-request extension of that. Splitting them across repos fragments the CSP story. |
| **Extensibility already exists** | `CSPBuilder func(nonce string) string` is already a plug point -- you can inject any policy without extraction. |

### The Actual Disease

The real problems are not "nonce lives in the wrong repo" -- they are two concrete issues:

1. **go-health-dashboard needs per-request nonce support.** It should accept `NonceProvider func(*http.Request) string` (or read from a well-known context key) instead of `Nonce string`. This lets consumers wire `httputil.NonceFromRequest(r)` directly -- no static constant, no fragile append middleware, no security hole. This is a 1-2 hour change in `go-health-dashboard` (status report item #6/#71).

2. **String-concatenated CSP is structurally fragile** (the bug class that caused this). A **structured CSP directive builder** -- where you compose directives as typed values and nonces are placed into the correct directive automatically -- would have made this bug **impossible**. That builder belongs in httputil (a `csp` sub-package or `cspbuilder.go` in the flat root), not a separate repo, because it is consumed alongside the nonce middleware.

---

## 4. Recommendation

- **Keep** the nonce middleware in httputil (do not extract).
- **Fix go-health-dashboard**: replace `WithNonce(string)` with `WithNonceProvider(func(*http.Request) string)`. This eliminates the static-nonce hack at its root.
- **Optionally** add a structured CSP builder to httputil that places nonces in the right directive by construction -- this prevents the entire append-concatenation bug class across all consumers, not just go-health-dashboard.

---

## References

- `httputil/nonce.go` -- nonce middleware implementation
- `httputil/nonce_test.go` / `nonce_fuzz_test.go` -- test coverage
- `httputil/security.go` -- `RecommendedCSP` constant and `SecurityHeaders` middleware
- `go-health-dashboard/dashboard.go:63` -- `WithNonce(string)` option (the construction-time limitation)
- `go-health-dashboard/docs/status/2026-08-09_00-41_csp-nonce-misplacement-fix.md` -- full incident report
- `httputil/docs/architecture-understanding/2026-08-05_06-56_package-structure-analysis.html` -- flat-package decision
- `httputil/docs/modularization/2026-08-05_DECISION.html` -- modularization decision record
