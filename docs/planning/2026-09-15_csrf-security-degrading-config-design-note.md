# Design note — CSRF security-degrading config: log-only vs remediate-to-secure-defaults

- **Date:** 2026-09-15
- **Status:** DESIGN COMPLETE — owner ruling pending (standing question ③, TODO_LIST Medium priority)
- **Origin:** full-code-review 2026-09-11 finding 2 (major); would overturn/refine DECISION_LOG 2026-08-08 ("Validate-and-log, not validate-and-abort, for middleware constructors")
- **Verification:** all external claims fetched from primary sources 2026-09-15 (sources inline); library claims cite `file:line` at master.

---

## 1. The finding

> Validate-and-log continues on security-degrading config (e.g. `SameSite=None` + `Secure=false`) instead of remediating to secure defaults.
> — [docs/reviews/2026-09-11_08-59_full-code-review.html](../reviews/2026-09-11_08-59_full-code-review.html), finding 2 (major)

The review recorded rather than fixed it: changing the validate-and-log contract touches a documented decision and needs this design pass.

## 2. Current behavior, exactly

`Validate()` is a pure reporter (`csrf.go:237`): it returns `csrf_samesite_insecure` (Infrastructure, cause-chained to `ErrCSRFConfig`) for `SameSite == http.SameSiteNoneMode && !Secure` (`csrf.go:242-246`). The constructor `CSRFMiddleware` then calls `validateConfig("CSRFConfig", cfg.Validate())` (`csrf.go:507`) — a `slog.LevelError` record with structured `code`/`family`/`domain` fields (`recorder.go:29-59`) — and **continues**. `ConfigureNosurfHandler` sets the base cookie with `cfg.Secure` and `cfg.SameSite` **verbatim** (`csrf.go:360-372`). Separately, any `Secure=false` gets a generic permissive warn (`csrf.go:509-514`).

### 2.1 Inventory: every `Validate()` rejection vs. what construction does

| Config problem | `Validate()` code | Construction behavior | Class |
| --- | --- | --- | --- |
| `MaxAge < 0` | `csrf.max_age_negative` | `maxAge()` falls back to the 24h default (`csrf.go:214-220`) | remediates to default |
| `SameSite=None` + `Secure=false` | `csrf_samesite_insecure` | **honored verbatim** — cookie sent `SameSite=None` with no `Secure` | **the gap** |
| `TrustedOrigins` `""`/`"*"`/non-origin | `csrf_unsafe_origin` / `csrf.trusted_origin_invalid` | all-or-nothing parse → same-origin-only fallback + loud log (2026-09-14 decision) | fail-closed |
| `TrustedProxies` `""`/bad CIDR | `csrf_unsafe_proxy` / `csrf_invalid_cidr` | `withParsedTrustedProxies` → nil CIDR list (`csrf.go:327-355`) | fail-closed |
| `AllowPlaintextBypass` w/o proxies | (none — deliberate) | honored + loud warn | opt-in insecure by design |

Conclusion: `SameSite=None` + `Secure=false` is the **only** security-degrading combo the constructor honors verbatim. Every other rejected config already remediates fail-closed or falls back to defaults. The 2026-08-08 decision text itself reads: *"Invalid config logs via slog **and falls back to defaults**"* — this combo is the one place where the fallback step is missing. A remediation here **completes** the documented contract rather than overturning it (the behavior change vs. status quo is still real and must ship as such).

## 3. What the misconfiguration does in reality (verified 2026-09-15)

1. **Spec (normative):** draft-ietf-httpbis-rfc6265bis, §5.7 (storage model): *"If the cookie's 'same-site-flag' is 'None', abort this algorithm and ignore the cookie entirely unless the cookie's secure-only-flag is true."* — a `SameSite=None` cookie without `Secure` is **never stored**. Source: `httpwg.org/http-extensions/draft-ietf-httpbis-rfc6265bis.html` (URL as cited by MDN browser-compat-data), raw text extracted 2026-09-15.
2. **MDN:** *"if `SameSite=None` is set then the `Secure` attribute must also be set — `SameSite=None` requires a secure context."* Source: raw `mdn/content` `files/en-us/web/http/guides/cookies/index.md` line 223, fetched 2026-09-15.
3. **Browser enforcement (mdn/browser-compat-data, `http/headers/Set-Cookie.json`):** Chromium-based browsers enforce the `SameSite` requirements (SameSite-by-default `Lax_default` row: Chrome 80 / Edge 86); older Chromium 51–67 rejected `SameSite=None` outright; Safari 12–13 treated `None` as `Strict` (WebKit bug 198181, fixed Catalina). Current evergreen browsers implement the rfc6265bis step.
4. **nosurf v1.2.0 mechanics (module source, `~/go/pkg/mod/github.com/justinas/nosurf@v1.2.0/handler.go`):** the token lives **in the CSRF cookie**; `ServeHTTP` regenerates whenever the cookie token is absent/malformed, and every unsafe method compares cookie token vs. sent token. `SetIsTLSFunc` is only a downstream hook — nosurf has no internal TLS gating.

**Consequence:** in every spec-compliant browser the combo is **self-announcing breakage**, not silent degradation: the cookie is never stored, so 100% of state-changing browser requests fail CSRF validation while safe methods pass. The misconfigured operator gets a very loud runtime signal. The residual exposure is (a) legacy clients predating enforcement, where the weaker `None`-without-`Secure` posture persists, and (b) operator confusion — the construction log names the code but no log line says what the middleware will actually do (answer today: honor the unenforceable setting).

## 4. Options

### A. Keep validate-and-log verbatim (status quo)

- **Pros:** zero behavior change; contract untouched; browser enforcement already makes the combo self-announcing; no v1.2 entry needed.
- **Cons:** the one `Validate()` rejection with no fallback, asymmetric with every other case; knowingly constructs a cookie modern browsers must drop; residual legacy-client exposure persists; log says "validation failed" while the middleware proceeds with the rejected value.

### B1. Remediate by forcing `Secure=true` (preserves the operator's intent)

Construction keeps `SameSite=None` but sets `Secure=true` with a structured remediation log (from/to). `Validate()` stays a pure reporter and unchanged.

- **Effect:** HTTPS deployments (direct or TLS-terminated behind a proxy) get exactly the cross-site behavior the config asked for — the cookie becomes storable. Plain-HTTP deployments behave as before at runtime (the cookie was already unstorable in enforcing browsers) but now get a precise log line naming the fix.
- **Pros:** never weakens anything (`Secure=true` is strictly the secure direction); preserves intent instead of downgrading it; asymmetry removed; log, behavior, and docs finally agree.
- **Cons:** runtime behavior change for misconfigured users → v1.2.0 minor + migration note per the established policy (`docs/v1-stability.md:293` precedent); `Secure` doc comment currently claims "auto-detected from request scheme" (`csrf.go:149`) which nothing implements — must be corrected in the same change.

### B2. Remediate by falling back to `SameSite=Lax`

Literal reading of the 2026-08-08 wording ("falls back to defaults"): drop `None` back to the documented default `Lax`.

- **Effect:** the cookie becomes storable everywhere; same-site flows work; **cross-site flows fail** (cookie not sent cross-site → token mismatch). The operator asked for cross-site and silently loses it.
- **Pros:** closest to the decision's literal text; documented default is the fallback.
- **Cons:** converts "loud total breakage" into "quieter partial breakage" of precisely the flows `None` was chosen for; downgrades expressed intent; same release mechanics as B1 with less upside.

### C. Request-time fail-closed (per-request scheme check)

Rejected in design: the config model is static, per-request cookie-attribute switching re-introduces per-request cost and surprise, and it makes the emitted cookie depend on the first request's transport — inconsistent with every other constructor in this package.

## 5. Recommendation

**B1.** It completes the documented contract's fallback step in the direction that preserves the operator's expressed intent where browsers can honor it, strictly strengthens the cookie, and keeps `Validate()` frozen. B2 is the conservative alternative if the owner prefers the literal "defaults" wording. A remains defensible on "self-announcing breakage" grounds but leaves the documented asymmetry in place.

## 6. Release mechanics (any B variant)

- Runtime behavior change **within unchanged signatures** → v1.2.0 minor, exactly like the two deltas already staged there (`docs/v1-stability.md:293`).
- `CHANGELOG.md` `[Unreleased]` Changed entry + a section in [docs/migrating-to-v1.2.md](../migrating-to-v1.2.md).
- Docs sweep in the same change: `AGENTS.md` CSRF bullets, `docs/DOMAIN_LANGUAGE.md:368` and `:424`, `FEATURES.md:98`, `SECURITY.md`, and the `errorTemplates` Fix wording for `csrf_samesite_insecure` (`errors.go`) — template text should say the constructor falls back and how to get the intended behavior.
- Adjacent doc-comment correction (same change): `CSRFConfig.Secure` "auto-detected from request scheme" claim (`csrf.go:149`) is false — nothing auto-detects (verified: `ConfigureNosurfHandler` uses `cfg.Secure` verbatim; nosurf's `IsTLS` hook is inert in v1.2.0).

## 7. Implementation sketch (B1)

```go
func CSRFMiddleware(cfg CSRFConfig) func(http.Handler) http.Handler {
    validateConfig("CSRFConfig", cfg.Validate())

    remediatingNone := cfg.SameSite == http.SameSiteNoneMode && !cfg.Secure
    if remediatingNone {
        cfg.Secure = true
        slog.Warn(
            "httputil: CSRFConfig: SameSite=None without Secure is not storable — fell back to Secure=true",
            slog.String("code", string(codeCSRFSameSiteInsecure)),
            slog.String("fix", "set Secure=true explicitly, or drop SameSite=None for same-site-only deployments"),
        )
    }

    if !cfg.Secure { // now unreachable for the remediated combo — no double warning
        slog.Warn("httputil: CSRFConfig: Secure is false — CSRF cookies will be sent over plain HTTP", ...)
    }
    // ... unchanged below; cfg now carries the remediated Secure flag
}
```

- Tests: constructor on `None+Secure=false` emits `Set-Cookie` with `Secure` (and keeps `SameSite=None`); `Lax`/`Strict`/zero-value configs unchanged; `Secure=false` non-`None` still warns and stays `false`; `TestCSRFMiddleware_InvalidConfigContinues` (csrf_test.go:298) keeps passing — add a cookie-attribute assertion next to it; validate-and-log contract tests (`validate_config_log_test.go`) unchanged since `Validate()` output is identical.
- Gates: `go build ./...`, `go test -race -count=10 ./...`, `golangci-lint run`, the three erraudit gates, `nix fmt`, `nix flake check`.

## 8. Decision ask (owner, question ③)

1. **A** — keep log-only verbatim; close the TODO as a documented decision.
2. **B1** (recommended) — constructor forces `Secure=true` for the combo; ship in v1.2.0 with the migration note.
3. **B2** — constructor falls back to `SameSite=Lax`; same release mechanics.
