# Consumer Reinvents httputil's Entire Nonce/CSP System — Introduces CSP Bug

**Discovered:** 2026-08-09
**Consumer:** `file-and-image-renamer` (pkg/healthd)
**Severity:** Medium (consumer bug already fixed, but the API gap that enabled it remains)

---

## What Happened

`file-and-image-renamer` imports `httputil v0.10.0` and uses it for `Recovery`, `SecurityHeaders`, `RequestID`, `Logging`, `Chain`, and the `Middleware` type. It then **reinvents the entire nonce/CSP system from scratch** in `pkg/healthd/nonce.go` — duplicating every function httputil already provides:

| httputil (already exists)                              | Consumer reinvented                        | Notes                                                            |
| ------------------------------------------------------ | ------------------------------------------ | ---------------------------------------------------------------- |
| `httputil.Nonce(cfg)` (nonce.go:131)                   | `healthd.Nonce` (nonce.go:66)              | Same logic, fewer features                                       |
| `httputil.ProductionCSPWithNonce(nonce)` (nonce.go:90) | `healthd.cspHeader(nonce)` (nonce.go:51)   | Near-identical output                                            |
| `httputil.nonceKey` + `WithNonce()` (nonce.go:126,155) | `healthd.nonceCtxKey` (nonce.go:14)        | Identical pattern                                                |
| `httputil.generateNonce(size)` (nonce.go:113)          | `healthd.generateNonce()` (nonce.go:19)    | Consumer uses 16 bytes (minimum); httputil uses 20 (recommended) |
| `httputil.NonceFromContext()` (nonce.go:161)           | `healthd.NonceFromContext()` (nonce.go:79) | Identical                                                        |
| Fuzz tested (nonce_fuzz_test.go)                       | No fuzzing                                 | CRLF injection not tested                                        |
| Ordering tests (Nonce must be innermost)               | No ordering tests                          |                                                                  |

The reinvented `dashboardCSPMiddleware` then **appended** a nonce to the end of the CSP string with `csp += " 'nonce-...'"`, which landed it in `frame-ancestors 'none'` (the last directive) instead of `script-src`. This caused 5 CSP console errors and silently disabled clickjacking protection on the `/health` route.

**The bug would have been structurally impossible** if the consumer had used `httputil.ProductionCSPWithNonce(HealthDashboardNonce)` — that function builds the CSP atomically via `fmt.Sprintf`, so the nonce can only ever land in the correct directives.

---

## Root Cause: Why the Consumer Rebuilt It

The consumer has two CSP requirements:

1. **Per-request nonce** for `/` (operations dashboard) — standard httputil pattern
2. **Fixed nonce** for `/health` (go-health-dashboard) — the dashboard library bakes the nonce at construction time via `WithNonce`, not per-request

httputil's `Nonce` middleware only generates **random** per-request nonces. The consumer needed to override the CSP with a **fixed** nonce for one route, and didn't realize `ProductionCSPWithNonce(nonce string)` is a public function that accepts ANY nonce — random or fixed. So they reimplemented everything.

---

## What httputil Could Improve

### 1. Make per-route CSP override a documented, tested pattern

The consumer needed "use this specific nonce's CSP for route X." httputil has all the pieces (`ProductionCSPWithNonce`, `NonceFromRequest`) but no explicit middleware or documentation for this pattern. Consider:

- A `RouteCSP(path, builder)` middleware that replaces the CSP for a specific route
- A documentation example showing how to combine `Nonce()` (per-request) with a manual `w.Header().Set("Content-Security-Policy", ProductionCSPWithNonce(fixedNonce))` for route-specific overrides

### 2. Structured CSP type instead of raw strings

`CSPBuilder` is `func(nonce string) string`. The consumer's bug was string concatenation on a free-form CSP header — `csp += " 'nonce-...'"`. A structured type would prevent this:

```go
type CSP struct {
    directives map[string][]string
}

func (c CSP) WithNonce(directive string, nonce string) CSP { ... }
func (c CSP) Render() string { ... }
```

This would make it a compile-time (or at least test-time) error to put a nonce in `frame-ancestors`.

### 3. Consider exposing a CSP validation helper

A function like `ValidateCSP(policy string) error` could catch common mistakes:

- `'none'` alongside other sources in any directive
- Nonces/sources in directives that don't support them (`frame-ancestors`, `report-uri`)
- Missing semicolons between directives

Consumers could call it in tests to catch CSP bugs before they reach production.

### 4. The consumer gap is discoverable — consider a lint or integration test

The consumer imported `httputil.Middleware` (the type) for their hand-rolled `Nonce` — `var _ httputil.Middleware = Nonce` — while ignoring that httputil already exports `Nonce(cfg NonceConfig) Middleware`. There's no way for httputil to prevent this, but a "common mistakes" doc section would help.

---

## Impact on httputil

- **No bug in httputil itself** — the library is correct and well-tested
- **API discoverability gap** — the consumer didn't find `ProductionCSPWithNonce` as a standalone function usable outside `NonceConfig`
- **Missing per-route override pattern** — the most common real-world CSP need (different policies for different routes) has no documented or tested pattern in httputil

---

## Recommendation

| Priority | Improvement                                                          | Effort    |
| -------- | -------------------------------------------------------------------- | --------- |
| High     | Add documentation example: "per-route CSP override with fixed nonce" | 15 min    |
| Medium   | Add `RouteCSP(path string, builder func(string) string) Middleware`  | 30 min    |
| Medium   | Add structured `CSP` type with `WithNonce()` / `Render()`            | 1-2 hours |
| Low      | Add `ValidateCSP(policy string) error` helper                        | 30 min    |
